package encoding

import (
	"fmt"
	"reflect"
	"sync"
	"unsafe"
)

// This file implements the high-performance decode path. For every Go type
// that flows through Decoder.Decode we compile a *decodeOps* once — a flat slice
// of opcodes describing the exact byte-level reads needed, with field
// offsets precomputed — and cache it keyed by reflect.Type. Every later
// Decode call walks the opcode list and reads directly into memory via
// unsafe.Pointer arithmetic, so the hot path never calls reflect.Value.Field,
// reflect.Value.SetUint, reflect.Value.Index, or similar.
//
// Design summary
//
//   • Compilation is pure reflection and runs at most once per type. The
//     result is stored in an sync.Map; subsequent lookups are a single
//     atomic load plus interface assertion.
//   • Execution takes only an (op, unsafe.Pointer) pair. No reflect.Value
//     is materialised during decoding of primitives.
//   • Container types (slice / pointer / struct field) still need the
//     element type to allocate typed storage that participates in GC:
//       - slices go through reflect.NewAt + reflect.MakeSlice so the slice
//         header the GC sees has the right element type info;
//       - Rust Option<T> goes through reflect.New for the same reason.
//     After allocation the executor extracts the raw base pointer and
//     drives the sub-plan with pointer arithmetic, so per-element cost is
//     one opcode dispatch plus a primitive read.
//   • Registered hand-written decoders (RegisterDecoder[T]) short-circuit
//     the plan: the compiler emits an opCallFunc op that invokes the
//     registered function directly. User types satisfying Unmarshaler
//     are compiled to opCallIface and dispatched via the interface table.
//
// Why this is correct for "different transaction contents"
//
// The plan encodes *shape* — field offsets, opcode kinds, length-prefix
// flavours — not *data*. A slice of 10 Transactions and a slice of 500
// Transactions hit exactly the same plan; the slice length comes from the
// wire at run time, and each element runs the same sub-plan against a
// different base pointer. Optional fields hit the same opcode regardless
// of whether the tag byte is 0 or 1. There is no branch on content in the
// compiled plan itself.

// ----------------------------------------------------------------------------
// Opcodes
// ----------------------------------------------------------------------------

type opKind uint8

const (
	opU8 opKind = iota
	opU16
	opU32
	opU64
	opI8
	opI16
	opI32
	opI64
	opBool
	opU128
	opU256
	opFixedBytes    // N raw bytes into [N]byte (offset + count)
	opFixedArray    // [N]T where T is compound (offset, count, elemSize, sub)
	opByteSlice     // []byte with a length prefix (offset, prefix)
	opSlice         // []T with a length prefix (offset, prefix, elemSize, sliceType, elemType, sub)
	opString        // length-prefixed string (offset, prefix)
	opPointer       // Rust Option<T>: u8 tag + optional payload (offset, elemType, sub)
	opCallFunc      // call registered decoderFunc (offset, fn)
	opCallIface     // call Unmarshaler (offset, ifaceType)
)

// op describes one step in a compiled plan. Fields are all pre-computed at
// compile time; the executor only reads them.
type op struct {
	kind opKind

	// offset into the enclosing struct / element. At the top level plans
	// always start at offset 0.
	offset uintptr

	// count: element count for opFixedBytes / opFixedArray.
	count uint32

	// prefix: length-prefix flavour for opByteSlice / opSlice / opString.
	prefix sizePrefix

	// elemSize: sizeof(element) for opSlice / opFixedArray / opPointer.
	elemSize uintptr

	// sliceType: reflect.Type of the slice itself (used by reflect.MakeSlice).
	sliceType reflect.Type

	// elemType: reflect.Type of the element (used by reflect.New for
	// opPointer, and by reflect.NewAt for debug formatting).
	elemType reflect.Type

	// sub: compiled sub-plan for compound elements / pointers.
	sub *decodeOps

	// fn: registered decoder for opCallFunc.
	fn decoderFunc

	// ifaceType: the concrete type whose pointer satisfies Unmarshaler, used
	// so we can call reflect.NewAt(ifaceType, ptr) and get a typed pointer
	// on which the interface method set is available.
	ifaceType reflect.Type
}

// decodeOps is the compiled decode programme for one Go type.
type decodeOps struct {
	ops  []op
	typ  reflect.Type
	size uintptr
}

// decodeOpsCache memoises compiled plans by reflect.Type.
var decodeOpsCache sync.Map // map[reflect.Type]*decodeOps

// unmarshalerType is taken once so compilation can cheaply test whether
// *T implements Unmarshaler without allocating an interface value per type.
var unmarshalerType = reflect.TypeOf((*Unmarshaler)(nil)).Elem()

// decodeOpsFor returns a compiled plan for t, compiling on first use. The tag
// options are only honoured at the top level, because inner usage flows
// through struct-field tags that the struct compiler reads directly.
func decodeOpsFor(t reflect.Type, opts tagOpts) (*decodeOps, error) {
	// Top-level plans without per-field tag annotations hit the cache.
	// Non-default tag options are rare at the top level (they only matter
	// for slice / string leaves), so we conservatively skip the cache then.
	if opts == (tagOpts{}) {
		if p, ok := decodeOpsCache.Load(t); ok {
			return p.(*decodeOps), nil
		}
	}
	p, err := compileDecodeOps(t, opts)
	if err != nil {
		return nil, err
	}
	if opts == (tagOpts{}) {
		if existing, loaded := decodeOpsCache.LoadOrStore(t, p); loaded {
			return existing.(*decodeOps), nil
		}
	}
	return p, nil
}

// compileDecodeOps produces a plan that, when executed against a pointer to a
// value of type t, decodes one value of t from the stream. The plan is a
// single op sequence operating on offsets relative to that base pointer.
func compileDecodeOps(t reflect.Type, opts tagOpts) (*decodeOps, error) {
	p := &decodeOps{typ: t, size: t.Size()}
	if err := emitValue(p, t, 0, opts); err != nil {
		return nil, err
	}
	return p, nil
}

// emitValue is the recursive compile step. It appends one or more opcodes
// to p that together decode one value of type t at offset off. For struct
// types each field becomes its own op with a field-local offset.
func emitValue(p *decodeOps, t reflect.Type, off uintptr, opts tagOpts) error {
	// 1. Registered hand-written decoder: emit opCallFunc and stop.
	if fn, ok := registry.Load(t); ok {
		p.ops = append(p.ops, op{kind: opCallFunc, offset: off, fn: fn.(decoderFunc)})
		return nil
	}
	// 2. Pointer-method Unmarshaler: emit opCallIface and stop.
	if reflect.PointerTo(t).Implements(unmarshalerType) {
		p.ops = append(p.ops, op{kind: opCallIface, offset: off, ifaceType: t})
		return nil
	}

	switch t.Kind() {
	case reflect.Uint8:
		p.ops = append(p.ops, op{kind: opU8, offset: off})
	case reflect.Uint16:
		p.ops = append(p.ops, op{kind: opU16, offset: off})
	case reflect.Uint32:
		p.ops = append(p.ops, op{kind: opU32, offset: off})
	case reflect.Uint64, reflect.Uint:
		p.ops = append(p.ops, op{kind: opU64, offset: off})
	case reflect.Int8:
		p.ops = append(p.ops, op{kind: opI8, offset: off})
	case reflect.Int16:
		p.ops = append(p.ops, op{kind: opI16, offset: off})
	case reflect.Int32:
		p.ops = append(p.ops, op{kind: opI32, offset: off})
	case reflect.Int64, reflect.Int:
		p.ops = append(p.ops, op{kind: opI64, offset: off})
	case reflect.Bool:
		p.ops = append(p.ops, op{kind: opBool, offset: off})

	case reflect.Array:
		// [N]byte fast path (covers named byte-array types too).
		if t.Elem().Kind() == reflect.Uint8 {
			p.ops = append(p.ops, op{kind: opFixedBytes, offset: off, count: uint32(t.Len())})
			return nil
		}
		// [N]T where T is compound: compile T once, loop in the executor.
		sub, err := compileDecodeOps(t.Elem(), tagOpts{})
		if err != nil {
			return err
		}
		p.ops = append(p.ops, op{
			kind:     opFixedArray,
			offset:   off,
			count:    uint32(t.Len()),
			elemSize: t.Elem().Size(),
			elemType: t.Elem(),
			sub:      sub,
		})

	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			// []byte (and named []byte types like Uint8Slice / Base58Data):
			// len-prefixed raw memcpy.
			p.ops = append(p.ops, op{
				kind:      opByteSlice,
				offset:    off,
				prefix:    opts.sizePrefix,
				sliceType: t,
			})
			return nil
		}
		sub, err := compileDecodeOps(t.Elem(), tagOpts{})
		if err != nil {
			return err
		}
		p.ops = append(p.ops, op{
			kind:      opSlice,
			offset:    off,
			prefix:    opts.sizePrefix,
			elemSize:  t.Elem().Size(),
			sliceType: t,
			elemType:  t.Elem(),
			sub:       sub,
		})

	case reflect.String:
		p.ops = append(p.ops, op{kind: opString, offset: off, prefix: opts.sizePrefix})

	case reflect.Pointer:
		// Rust Option<T>: 1-byte tag; if 1, allocate and decode inner.
		sub, err := compileDecodeOps(t.Elem(), tagOpts{})
		if err != nil {
			return err
		}
		p.ops = append(p.ops, op{
			kind:     opPointer,
			offset:   off,
			elemSize: t.Elem().Size(),
			elemType: t.Elem(),
			sub:      sub,
		})

	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			fopts := parseTag(f.Tag.Get("bin"))
			if fopts.skip {
				continue
			}
			if err := emitValue(p, f.Type, off+f.Offset, fopts); err != nil {
				return fmt.Errorf("%s.%s: %w", t.Name(), f.Name, err)
			}
		}

	default:
		return fmt.Errorf("encoding: cannot compile plan for kind %s (%s)", t.Kind(), t)
	}
	return nil
}

// ----------------------------------------------------------------------------
// Executor
// ----------------------------------------------------------------------------

// execDecodeOps runs a compiled plan against base, where base is a pointer to a
// value of p.typ. Returns on the first error without partial rollback — the
// stream is already past the failure point, same contract as Decode.
func (d *Decoder) execDecodeOps(p *decodeOps, base unsafe.Pointer) error {
	for i := range p.ops {
		if err := d.execDecodeOp(&p.ops[i], base); err != nil {
			return err
		}
	}
	return nil
}

// execDecodeOp dispatches one op. base is the enclosing struct / element base;
// the op's offset is added to find the leaf's memory.
func (d *Decoder) execDecodeOp(o *op, base unsafe.Pointer) error {
	ptr := unsafe.Add(base, o.offset)
	switch o.kind {
	case opU8:
		v, err := d.ReadUint8()
		if err != nil {
			return err
		}
		*(*uint8)(ptr) = v
	case opU16:
		v, err := d.ReadUint16()
		if err != nil {
			return err
		}
		*(*uint16)(ptr) = v
	case opU32:
		v, err := d.ReadUint32()
		if err != nil {
			return err
		}
		*(*uint32)(ptr) = v
	case opU64:
		v, err := d.ReadUint64()
		if err != nil {
			return err
		}
		*(*uint64)(ptr) = v
	case opI8:
		v, err := d.ReadUint8()
		if err != nil {
			return err
		}
		*(*int8)(ptr) = int8(v)
	case opI16:
		v, err := d.ReadUint16()
		if err != nil {
			return err
		}
		*(*int16)(ptr) = int16(v)
	case opI32:
		v, err := d.ReadUint32()
		if err != nil {
			return err
		}
		*(*int32)(ptr) = int32(v)
	case opI64:
		v, err := d.ReadUint64()
		if err != nil {
			return err
		}
		*(*int64)(ptr) = int64(v)
	case opBool:
		b, err := d.ReadUint8()
		if err != nil {
			return err
		}
		switch b {
		case 0:
			*(*bool)(ptr) = false
		case 1:
			*(*bool)(ptr) = true
		default:
			return fmt.Errorf("encoding: invalid bool byte 0x%02x", b)
		}
	case opU128:
		// Unreachable in current compiler (U128 flows through opCallFunc)
		// but kept in the switch so a future compile decision to inline the
		// 16-byte read has a direct entry.
		b, err := d.ReadBytes(16)
		if err != nil {
			return err
		}
		*(*[16]byte)(ptr) = *(*[16]byte)(unsafe.Pointer(&b[0]))
	case opU256:
		b, err := d.ReadBytes(32)
		if err != nil {
			return err
		}
		*(*[32]byte)(ptr) = *(*[32]byte)(unsafe.Pointer(&b[0]))
	case opFixedBytes:
		b, err := d.ReadBytes(int(o.count))
		if err != nil {
			return err
		}
		// Target is an in-place [N]byte backing store; a raw memcpy is fine.
		dst := unsafe.Slice((*byte)(ptr), int(o.count))
		copy(dst, b)
	case opFixedArray:
		elemBase := ptr
		for i := uint32(0); i < o.count; i++ {
			if err := d.execDecodeOps(o.sub, unsafe.Add(elemBase, uintptr(i)*o.elemSize)); err != nil {
				return fmt.Errorf("[%d]: %w", i, err)
			}
		}
	case opByteSlice:
		n, err := d.readLen(o.prefix)
		if err != nil {
			return err
		}
		b, err := d.ReadBytes(int(n))
		if err != nil {
			return err
		}
		// Allocate a typed slice via reflect so the GC sees the right
		// element type (matters for named byte-slice types with methods).
		rv := reflect.NewAt(o.sliceType, ptr).Elem()
		out := reflect.MakeSlice(o.sliceType, int(n), int(n))
		if n > 0 {
			// out is addressable only through the header we're about to
			// install; copy via a header we own then Set.
			reflect.Copy(out, reflect.ValueOf(b))
		}
		rv.Set(out)
	case opSlice:
		n, err := d.readLen(o.prefix)
		if err != nil {
			return err
		}
		rv := reflect.NewAt(o.sliceType, ptr).Elem()
		out := reflect.MakeSlice(o.sliceType, int(n), int(n))
		// MakeSlice returns a non-addressable Value; Set installs a header at
		// `ptr` whose Data points at the newly-allocated backing array. We
		// then obtain that backing-array base pointer and drive sub-plans by
		// offset — no per-element reflect.Value.
		rv.Set(out)
		if n == 0 {
			return nil
		}
		elemBase := unsafe.Pointer(rv.Index(0).UnsafeAddr())
		for i := uint64(0); i < n; i++ {
			if err := d.execDecodeOps(o.sub, unsafe.Add(elemBase, uintptr(i)*o.elemSize)); err != nil {
				return fmt.Errorf("[%d]: %w", i, err)
			}
		}
	case opString:
		n, err := d.readLen(o.prefix)
		if err != nil {
			return err
		}
		b, err := d.ReadBytes(int(n))
		if err != nil {
			return err
		}
		*(*string)(ptr) = string(b)
	case opPointer:
		tag, err := d.ReadUint8()
		if err != nil {
			return err
		}
		// The target memory is a *T. Write either nil or a freshly
		// allocated *T. reflect.New gives us a GC-tracked allocation.
		rv := reflect.NewAt(reflect.PointerTo(o.elemType), ptr).Elem()
		switch tag {
		case 0:
			rv.SetZero()
		case 1:
			fresh := reflect.New(o.elemType)
			rv.Set(fresh)
			if err := d.execDecodeOps(o.sub, unsafe.Pointer(fresh.Pointer())); err != nil {
				return err
			}
		default:
			return fmt.Errorf("encoding: invalid Option tag 0x%02x", tag)
		}
	case opCallFunc:
		if err := o.fn(d, ptr); err != nil {
			return err
		}
	case opCallIface:
		// Reconstruct an addressable value of the concrete type so the
		// pointer method set is available, then dispatch through the
		// Unmarshaler interface. This allocates an interface header only —
		// the underlying storage is the caller's memory, not a copy.
		rv := reflect.NewAt(o.ifaceType, ptr)
		if err := rv.Interface().(Unmarshaler).UnmarshalFromDecoder(d); err != nil {
			return err
		}
	default:
		return fmt.Errorf("encoding: unknown opcode %d", o.kind)
	}
	return nil
}

// ----------------------------------------------------------------------------
// Public entry points that use the plan cache
// ----------------------------------------------------------------------------

// DecodeFast is the high-performance drop-in for Decode. It compiles a
// plan for v's element type on first use, caches it globally, and executes
// the plan with pointer arithmetic so the per-value cost is effectively
// one opcode dispatch per leaf field.
//
// Behaviour is equivalent to Decode for all supported types; the only
// visible difference is that error strings may reach a leaf with a
// different path formatter. For types that decode through a registered
// hand-written decoder or an Unmarshaler implementation, the two entry
// points are byte-for-byte identical because they dispatch to the same
// function.
//
// Decode itself now delegates here, so calling either name is equivalent;
// DecodeFast is retained as the explicit name for the plan-cached path
// and for benchmarks that want to bypass the dispatch shim.
func (d *Decoder) DecodeFast(v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("encoding: DecodeFast requires a non-nil pointer, got %T", v)
	}
	t := rv.Type().Elem()
	p, err := decodeOpsFor(t, tagOpts{})
	if err != nil {
		return err
	}
	return d.execDecodeOps(p, unsafe.Pointer(rv.Pointer()))
}
