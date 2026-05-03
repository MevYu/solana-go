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
	opU128
	opU256
	opFixedBytes
	opFixedArray
	opByteSlice
	opSlice
	opString
	opPointer
	opCallFunc
	opCallIface
)

type op struct {
	kind      opKind
	offset    uintptr
	count     uint32
	prefix    sizePrefix
	elemSize  uintptr
	sliceType reflect.Type
	elemType  reflect.Type
	sub       *decodeOps
	fn        decoderFunc
	ifaceType reflect.Type
}

type decodeOps struct {
	ops  []op
	typ  reflect.Type
	size uintptr
}

var decodeOpsCache sync.Map // map[reflect.Type]*decodeOps

var unmarshalerType = reflect.TypeOf((*Unmarshaler)(nil)).Elem()

func decodeOpsFor(t reflect.Type, opts tagOpts) (*decodeOps, error) {
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

func compileDecodeOps(t reflect.Type, opts tagOpts) (*decodeOps, error) {
	p := &decodeOps{typ: t, size: t.Size()}
	if err := emitValue(p, t, 0, opts); err != nil {
		return nil, err
	}
	return p, nil
}

func emitValue(p *decodeOps, t reflect.Type, off uintptr, opts tagOpts) error {
	if fn, ok := registry.Load(t); ok {
		p.ops = append(p.ops, op{kind: opCallFunc, offset: off, fn: fn.(decoderFunc)})
		return nil
	}
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
		if t.Elem().Kind() == reflect.Uint8 {
			p.ops = append(p.ops, op{kind: opFixedBytes, offset: off, count: uint32(t.Len())})
			return nil
		}
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

func (d *Decoder) execDecodeOps(p *decodeOps, base unsafe.Pointer) error {
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
	case opU128:
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
		// Use reflect so the GC sees the named slice type, not just []byte.
		rv := reflect.NewAt(o.sliceType, ptr).Elem()
		out := reflect.MakeSlice(o.sliceType, int(n), int(n))
		if n > 0 {
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
	p, err := decodeOpsFor(t, tagOpts{})
	if err != nil {
		return err
	}
	return d.execDecodeOps(p, unsafe.Pointer(rv.Pointer()))
}
