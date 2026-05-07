package encoding

import (
	"fmt"
	"reflect"
	"sync"
	"unsafe"
)

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
	opFixedBytes
	opFixedArray
	opByteSlice
	opSlice
	opString
	opPointer
	opCallIface
)

type op struct {
	kind      opKind
	offset    uintptr
	count     uint32
	elemSize  uintptr
	sliceType reflect.Type
	elemType  reflect.Type
	sub       *decodePlan
	ifaceType reflect.Type
}

type decodePlan struct {
	ops  []op
	typ  reflect.Type
	size uintptr
}

// decodePlanCache memoises compiled plans keyed by reflect.Type so each
// type pays the reflective compile cost (~2.7us / 4KB) only once.
var decodePlanCache sync.Map // map[reflect.Type]*decodePlan

// decodePlanFor returns a compiled plan for t, compiling on first use.
func decodePlanFor(t reflect.Type) (*decodePlan, error) {
	if p, ok := decodePlanCache.Load(t); ok {
		return p.(*decodePlan), nil
	}
	p, err := compileDecodePlan(t)
	if err != nil {
		return nil, err
	}
	if existing, loaded := decodePlanCache.LoadOrStore(t, p); loaded {
		return existing.(*decodePlan), nil
	}
	return p, nil
}

func compileDecodePlan(t reflect.Type) (*decodePlan, error) {
	p := &decodePlan{typ: t, size: t.Size()}
	if err := emitValue(p, t, 0); err != nil {
		return nil, err
	}
	return p, nil
}

func emitValue(p *decodePlan, t reflect.Type, off uintptr) error {
	if reflect.PointerTo(t).Implements(unmarshalerReflectType()) {
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
		if t.Elem().Kind() == reflect.Uint8 {
			p.ops = append(p.ops, op{kind: opFixedBytes, offset: off, count: uint32(t.Len())})
			return nil
		}
		sub, err := compileDecodePlan(t.Elem())
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
			p.ops = append(p.ops, op{
				kind:      opByteSlice,
				offset:    off,
				sliceType: t,
			})
			return nil
		}
		sub, err := compileDecodePlan(t.Elem())
		if err != nil {
			return err
		}
		p.ops = append(p.ops, op{
			kind:      opSlice,
			offset:    off,
			elemSize:  t.Elem().Size(),
			sliceType: t,
			elemType:  t.Elem(),
			sub:       sub,
		})

	case reflect.String:
		p.ops = append(p.ops, op{kind: opString, offset: off})

	case reflect.Pointer:
		sub, err := compileDecodePlan(t.Elem())
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
			// `bin:"-"` opts a field out of reflective decoding so domain
			// structs can carry non-wire state (caches, derived data,
			// unsupported kinds like maps) alongside on-wire fields.
			if f.Tag.Get("bin") == "-" {
				continue
			}
			if err := emitValue(p, f.Type, off+f.Offset); err != nil {
				return fmt.Errorf("%s.%s: %w", t.Name(), f.Name, err)
			}
		}

	default:
		return fmt.Errorf("encoding: cannot compile plan for kind %s (%s)", t.Kind(), t)
	}
	return nil
}

func (d *Decoder) execDecodePlan(p *decodePlan, base unsafe.Pointer) error {
	for i := range p.ops {
		if err := d.execDecodeOp(&p.ops[i], base); err != nil {
			return err
		}
	}
	return nil
}

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
	case opFixedBytes:
		b, err := d.ReadBytes(int(o.count))
		if err != nil {
			return err
		}
		dst := unsafe.Slice((*byte)(ptr), int(o.count))
		copy(dst, b)
	case opFixedArray:
		elemBase := ptr
		for i := uint32(0); i < o.count; i++ {
			if err := d.execDecodePlan(o.sub, unsafe.Add(elemBase, uintptr(i)*o.elemSize)); err != nil {
				return fmt.Errorf("[%d]: %w", i, err)
			}
		}
	case opByteSlice:
		n, err := d.readLen()
		if err != nil {
			return err
		}
		b, err := d.ReadBytes(int(n))
		if err != nil {
			return err
		}
		// Allocate-and-copy (not alias d.buf) so the decoded struct can
		// outlive the input buffer; reflect handles named slice types
		// like `type Bytes []byte`.
		rv := reflect.NewAt(o.sliceType, ptr).Elem()
		out := reflect.MakeSlice(o.sliceType, int(n), int(n))
		if n > 0 {
			reflect.Copy(out, reflect.ValueOf(b))
		}
		rv.Set(out)
	case opSlice:
		n, err := d.readLen()
		if err != nil {
			return err
		}
		rv := reflect.NewAt(o.sliceType, ptr).Elem()
		out := reflect.MakeSlice(o.sliceType, int(n), int(n))
		rv.Set(out)
		if n == 0 {
			return nil
		}
		elemBase := unsafe.Pointer(rv.Index(0).UnsafeAddr())
		for i := uint64(0); i < n; i++ {
			if err := d.execDecodePlan(o.sub, unsafe.Add(elemBase, uintptr(i)*o.elemSize)); err != nil {
				return fmt.Errorf("[%d]: %w", i, err)
			}
		}
	case opString:
		n, err := d.readLen()
		if err != nil {
			return err
		}
		b, err := d.ReadBytes(int(n))
		if err != nil {
			return err
		}
		*(*string)(ptr) = string(b)
	case opPointer:
		// Rust Option<T>: 1-byte tag, then payload only when Some.
		tag, err := d.ReadUint8()
		if err != nil {
			return err
		}
		rv := reflect.NewAt(reflect.PointerTo(o.elemType), ptr).Elem()
		switch tag {
		case 0:
			rv.SetZero()
		case 1:
			// reflect.New so the GC tracks the allocation as the right type.
			fresh := reflect.New(o.elemType)
			rv.Set(fresh)
			if err := d.execDecodePlan(o.sub, unsafe.Pointer(fresh.Pointer())); err != nil {
				return err
			}
		default:
			return fmt.Errorf("encoding: invalid Option tag 0x%02x", tag)
		}
	case opCallIface:
		rv := reflect.NewAt(o.ifaceType, ptr)
		if err := rv.Interface().(Unmarshaler).UnmarshalFromDecoder(d); err != nil {
			return err
		}
	default:
		return fmt.Errorf("encoding: unknown opcode %d", o.kind)
	}
	return nil
}

// DecodeFast is Decode using a cached, compiled per-type plan.
func (d *Decoder) DecodeFast(v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("encoding: DecodeFast requires a non-nil pointer, got %T", v)
	}
	t := rv.Type().Elem()
	p, err := decodePlanFor(t)
	if err != nil {
		return err
	}
	return d.execDecodePlan(p, unsafe.Pointer(rv.Pointer()))
}
