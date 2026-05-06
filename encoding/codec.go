package encoding

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"unsafe"
)

// Unmarshaler lets a type plug in hand-written decoding logic and bypass
// reflection entirely. Types whose addressable value satisfies this interface
// are dispatched directly by Decode; reflection never descends into them.
type Unmarshaler interface {
	UnmarshalFromDecoder(d *Decoder) error
}

// decoderFunc is the type of hand-written fast-path decoders stored in the
// registry. The pointer is a typed *T that the registration wrapper has
// already converted from an unsafe.Pointer, so individual decoders do not
// juggle unsafe themselves.
type decoderFunc func(d *Decoder, ptr unsafe.Pointer) error

// registry maps reflect.Type to a hand-written decoder. Populated by
// RegisterDecoder; consulted by Decode before any reflective walking.
var registry sync.Map // map[reflect.Type]decoderFunc

// RegisterDecoder associates a hand-written decoder with type T. When Decode
// encounters a value of type T — directly, as a struct field, slice element,
// pointer target, or inside a larger containing type — fn is called instead
// of walking the type reflectively.
//
// Intended use is during package init. RegisterDecoder is safe for concurrent
// use, but registering a second decoder for the same type replaces the first.
//
// The wrapper handles the unsafe.Pointer → *T conversion so callers write
// normal typed Go.
//
// When registration helps vs hurts, by type shape:
//
//   - Leaf [N]byte-backed named types (PublicKey, Hash, Signature, U128,
//     U256, …): do NOT register. The reflective fast path emits opFixedBytes
//     for them — a single ReadBytes(N) + memmove the compiler inlines through
//     the op switch. opCallFunc dispatches through a function pointer that
//     cannot inline, so leaf registration is 1.5–5% SLOWER under concurrent
//     load (the gap widens with goroutine count due to BTB pressure).
//
//   - Large composite structs (10+ exported fields, account-shaped):
//     registration CAN help. The reflective walker pays per-field op-dispatch
//     cost; a hand-written closure that reads the whole struct in one shot
//     amortises that to zero. Bench shows 5–15% speedup on a ~30-field AMM
//     pool struct (see poolstate_decode_bench_test.go for methodology).
//     Whether the win materialises depends on the field-count vs decode-cost
//     ratio — bench before committing.
//
//   - Custom wire formats / layout transformations / third-party types
//     that can't implement Unmarshaler / post-decode validation:
//     register without question — no other path expresses these.
func RegisterDecoder[T any](fn func(*Decoder, *T) error) {
	var zero T
	t := reflect.TypeOf(zero)
	if t == nil {
		panic("encoding.RegisterDecoder: T must be a concrete non-interface type")
	}
	wrap := decoderFunc(func(d *Decoder, p unsafe.Pointer) error {
		return fn(d, (*T)(p))
	})
	registry.Store(t, wrap)
}

// sizePrefix identifies the length-prefix encoding for a slice or string.
// The default (prefixDefault, zero value) is Rust bincode's u64, which is
// what Solana emits for Vec<T> outside of transaction/message bodies.
type sizePrefix uint8

const (
	prefixDefault  sizePrefix = iota // u64 little-endian (Rust bincode default for Vec<T> / String)
	prefixShortvec                   // compact-u16 (Solana transaction & message length fields)
	prefixU32                        // 4-byte little-endian (Borsh convention)
	prefixU16                        // 2-byte little-endian
	prefixU8                         // 1-byte
)

// tagOpts are the per-field knobs parsed out of a `bin:"..."` struct tag.
type tagOpts struct {
	skip       bool
	sizePrefix sizePrefix
}

// parseTag parses a `bin:"..."` tag into tagOpts. Unknown keys are ignored so
// future tag keys can be added without breaking existing tagged structs.
func parseTag(tag string) tagOpts {
	var opts tagOpts
	if tag == "" {
		return opts
	}
	if tag == "-" {
		opts.skip = true
		return opts
	}
	for _, part := range strings.Split(tag, ",") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "sizePrefix=") {
			switch part[len("sizePrefix="):] {
			case "shortvec":
				opts.sizePrefix = prefixShortvec
			case "u32":
				opts.sizePrefix = prefixU32
			case "u16":
				opts.sizePrefix = prefixU16
			case "u8":
				opts.sizePrefix = prefixU8
			case "u64", "bincode":
				opts.sizePrefix = prefixDefault
			}
		}
	}
	return opts
}

// readLen reads a slice / string length using the chosen prefix convention.
// All widths are promoted to uint64 so callers have one integer type to
// range over regardless of on-wire width.
//
// When the field-level prefix is prefixDefault, the decoder's per-instance
// default (defaultPrefix) is consulted; if that is also unset, falls back
// to bincode's u64. Setting d.defaultPrefix = prefixU32 (via UseBorsh)
// flips this fallback to Borsh's u32.
func (d *Decoder) readLen(p sizePrefix) (uint64, error) {
	if p == prefixDefault {
		p = d.defaultPrefix
	}
	switch p {
	case prefixShortvec:
		v, err := d.ReadShortvec()
		return uint64(v), err
	case prefixU8:
		v, err := d.ReadUint8()
		return uint64(v), err
	case prefixU16:
		v, err := d.ReadUint16()
		return uint64(v), err
	case prefixU32:
		v, err := d.ReadUint32()
		return uint64(v), err
	default:
		return d.ReadUint64()
	}
}

// DecodeTo decodes the next value from the stream into v, which must be a
// non-nil pointer. It is the entry point for reflection-based decoding and
// is a drop-in replacement for the gagliardetto/binary NewBinDecoder+Decode
// pattern.
//
// Before reflecting, DecodeTo consults two escape hatches in order:
//  1. A hand-written decoder registered via RegisterDecoder for the target
//     type — library-provided fast paths for PublicKey, Transaction, etc.
//  2. The Unmarshaler interface — types that implement it are dispatched
//     directly without reflecting into their fields.
//
// For reflected values, the supported kinds are: uint8/16/32/64, int8/16/32/64,
// bool, [N]byte and [N]T fixed arrays, []byte and []T slices (length-prefixed),
// string (length-prefixed), pointer (bincode Option: 1-byte tag + optional
// payload), and struct (recursively, respecting `bin:"..."` field tags).
//
// The default length prefix is Rust's bincode u64. Override on a per-field
// basis with `bin:"sizePrefix=shortvec"` etc. Top-level slice decoding uses
// the default; to read a shortvec-prefixed slice at the top level, either
// wrap it in a struct or call ReadShortvec + a loop manually.
func (d *Decoder) DecodeTo(v any) error {
	// Plan-cached executor is the default. DecodeFast compiles the type
	// once (memoised globally) and then drives all subsequent decodes
	// through pointer arithmetic — no per-value reflect.Value is
	// materialised on the hot path. The legacy reflective implementation
	// is preserved below as decodeValue and reachable via decodeReflect
	// for fallback / benchmarking.
	return d.DecodeFast(v)
}

// decodeReflect runs the legacy fully-reflective decoder (no plan cache).
// Kept for benchmarks that compare paths, and as an emergency fallback if
// a future container kind is added to the compiler's switch but its
// executor op is missing.
func (d *Decoder) decodeReflect(v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return errors.New("encoding: Decode requires a non-nil pointer")
	}
	elem := rv.Elem()
	if !elem.CanAddr() {
		return errors.New("encoding: Decode target must be addressable")
	}
	return d.decodeValue(elem, tagOpts{})
}

// decodeValue is the recursive workhorse. rv must be settable and, for the
// Unmarshaler / registry dispatches, addressable. Callers that descend into
// struct fields, array elements, or slice elements always pass addressable
// values because they come from Field/Index on an addressable parent.
func (d *Decoder) decodeValue(rv reflect.Value, opts tagOpts) error {
	t := rv.Type()

	// 1. Hand-written registry: short-circuit all reflection on known types.
	if fn, ok := registry.Load(t); ok {
		return fn.(decoderFunc)(d, unsafe.Pointer(rv.UnsafeAddr()))
	}

	// 2. Unmarshaler interface: a type can hook itself without RegisterDecoder.
	//    Taking the pointer method set is cheaper than a registry lookup and
	//    gives user types a zero-dependency extension point.
	if rv.CanAddr() {
		if u, ok := rv.Addr().Interface().(Unmarshaler); ok {
			return u.UnmarshalFromDecoder(d)
		}
	}

	switch rv.Kind() {
	case reflect.Uint8:
		v, err := d.ReadUint8()
		if err != nil {
			return err
		}
		rv.SetUint(uint64(v))
	case reflect.Uint16:
		v, err := d.ReadUint16()
		if err != nil {
			return err
		}
		rv.SetUint(uint64(v))
	case reflect.Uint32:
		v, err := d.ReadUint32()
		if err != nil {
			return err
		}
		rv.SetUint(uint64(v))
	case reflect.Uint64, reflect.Uint:
		v, err := d.ReadUint64()
		if err != nil {
			return err
		}
		rv.SetUint(v)
	case reflect.Int8:
		v, err := d.ReadUint8()
		if err != nil {
			return err
		}
		rv.SetInt(int64(int8(v)))
	case reflect.Int16:
		v, err := d.ReadUint16()
		if err != nil {
			return err
		}
		rv.SetInt(int64(int16(v)))
	case reflect.Int32:
		v, err := d.ReadUint32()
		if err != nil {
			return err
		}
		rv.SetInt(int64(int32(v)))
	case reflect.Int64, reflect.Int:
		v, err := d.ReadUint64()
		if err != nil {
			return err
		}
		rv.SetInt(int64(v))
	case reflect.Bool:
		b, err := d.ReadUint8()
		if err != nil {
			return err
		}
		switch b {
		case 0:
			rv.SetBool(false)
		case 1:
			rv.SetBool(true)
		default:
			return fmt.Errorf("encoding: invalid bool byte 0x%02x", b)
		}
	case reflect.Array:
		return d.decodeArray(rv)
	case reflect.Slice:
		return d.decodeSlice(rv, opts)
	case reflect.String:
		n, err := d.readLen(opts.sizePrefix)
		if err != nil {
			return err
		}
		b, err := d.ReadBytes(int(n))
		if err != nil {
			return err
		}
		rv.SetString(string(b))
	case reflect.Pointer:
		// Rust Option<T>: 1-byte tag, payload follows only when tag is 1.
		tag, err := d.ReadUint8()
		if err != nil {
			return err
		}
		switch tag {
		case 0:
			rv.Set(reflect.Zero(t))
			return nil
		case 1:
			rv.Set(reflect.New(t.Elem()))
			return d.decodeValue(rv.Elem(), tagOpts{})
		default:
			return fmt.Errorf("encoding: invalid Option tag 0x%02x", tag)
		}
	case reflect.Struct:
		return d.decodeStruct(rv)
	default:
		return fmt.Errorf("encoding: cannot decode kind %s (%s)", rv.Kind(), t)
	}
	return nil
}

// decodeArray handles [N]T. [N]byte (including named byte-array types like
// PublicKey before its hand-written decoder is registered) hits a single
// ReadBytes + reflect.Copy fast path; other element types recurse.
func (d *Decoder) decodeArray(rv reflect.Value) error {
	t := rv.Type()
	if t.Elem().Kind() == reflect.Uint8 {
		n := t.Len()
		b, err := d.ReadBytes(n)
		if err != nil {
			return err
		}
		reflect.Copy(rv, reflect.ValueOf(b))
		return nil
	}
	for i := 0; i < rv.Len(); i++ {
		if err := d.decodeValue(rv.Index(i), tagOpts{}); err != nil {
			return fmt.Errorf("array[%d]: %w", i, err)
		}
	}
	return nil
}

// decodeSlice handles []T. Length prefix follows opts.sizePrefix
// (default bincode u64). []byte-like element types (kind uint8) hit a single
// ReadBytes + reflect.Copy fast path that works for the bare []byte as well
// as named types like Uint8Slice and Base58Data.
func (d *Decoder) decodeSlice(rv reflect.Value, opts tagOpts) error {
	t := rv.Type()
	n, err := d.readLen(opts.sizePrefix)
	if err != nil {
		return err
	}
	if t.Elem().Kind() == reflect.Uint8 {
		b, err := d.ReadBytes(int(n))
		if err != nil {
			return err
		}
		out := reflect.MakeSlice(t, int(n), int(n))
		reflect.Copy(out, reflect.ValueOf(b))
		rv.Set(out)
		return nil
	}
	slc := reflect.MakeSlice(t, int(n), int(n))
	for i := 0; i < int(n); i++ {
		if err := d.decodeValue(slc.Index(i), tagOpts{}); err != nil {
			return fmt.Errorf("slice[%d]: %w", i, err)
		}
	}
	rv.Set(slc)
	return nil
}

// decodeStruct walks each exported field in declaration order. Unexported
// fields are skipped silently; `bin:"-"` tagged fields are skipped
// explicitly. The struct name and field name are embedded in any error so
// callers can locate decode failures without a debugger.
func (d *Decoder) decodeStruct(rv reflect.Value) error {
	t := rv.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		opts := parseTag(f.Tag.Get("bin"))
		if opts.skip {
			continue
		}
		if err := d.decodeValue(rv.Field(i), opts); err != nil {
			return fmt.Errorf("%s.%s: %w", t.Name(), f.Name, err)
		}
	}
	return nil
}
