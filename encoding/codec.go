package encoding

import (
	"reflect"
)

// Unmarshaler lets a type plug in hand-written decoding logic and bypass
// reflection entirely. Types whose addressable value satisfies this interface
// are dispatched directly by DecodeTo; reflection never descends into them.
type Unmarshaler interface {
	UnmarshalFromDecoder(d *Decoder) error
}

// unmarshalerReflectType returns the reflect.Type for the Unmarshaler
// interface, used by the plan compiler when checking whether *T satisfies
// it. Wrapped in a function so the literal `(*Unmarshaler)(nil)` magic
// only appears in one place.
func unmarshalerReflectType() reflect.Type {
	return reflect.TypeOf((*Unmarshaler)(nil)).Elem()
}

// readLen reads a slice / string length using the Decoder's configured prefix
// width. All widths are promoted to uint64 so call sites have one integer
// type to range over regardless of on-wire width.
func (d *Decoder) readLen() (uint64, error) {
	if d.borsh {
		v, err := d.ReadUint32()
		return uint64(v), err
	}
	return d.ReadUint64()
}

// DecodeTo decodes the next value from the stream into v, which must be a
// non-nil pointer. It is the entry point for reflection-based decoding.
//
// If the target type (or *T) implements Unmarshaler, the plan-cache emits
// a single dispatch op and reflection never descends into the type's fields.
//
// Otherwise the supported kinds are: uint8/16/32/64, int8/16/32/64, bool,
// [N]byte and [N]T fixed arrays, []byte and []T slices (length-prefixed),
// string (length-prefixed), pointer (bincode Option: 1-byte tag + optional
// payload), and struct (recursively, all exported fields).
//
// The length prefix is bincode's u64 by default; call UseBorsh on the
// decoder to switch to Borsh's u32. Bespoke widths (shortvec, u8, …) are
// not exposed via tags — hand-write the instruction with Reader / Encoder.
func (d *Decoder) DecodeTo(v any) error {
	// All reflective decoding goes through DecodeFast: compile the type's
	// decode plan once (memoised in decodePlanCache), then drive every
	// subsequent decode through pointer arithmetic without materialising
	// any per-value reflect.Value on the hot path.
	return d.DecodeFast(v)
}
