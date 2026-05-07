package encoding

import (
	"reflect"
	"strings"
)

// Unmarshaler lets a type plug in hand-written decoding logic and bypass
// reflection entirely. Types whose addressable value satisfies this interface
// are dispatched directly by Decode; reflection never descends into them.
//
// When implementing helps vs hurts, by type shape:
//
//   - Leaf [N]byte-backed named types (PublicKey, Hash, Signature, U128,
//     U256, …): do NOT implement. The plan-cache emits opFixedBytes for
//     them — a single ReadBytes(N) + memmove the compiler inlines through
//     the op switch. An UnmarshalFromDecoder method dispatches through an
//     interface that cannot inline, so leaf overrides are 1.5–5% SLOWER
//     under concurrent load (BTB pressure widens the gap with goroutine
//     count).
//
//   - Large composite structs (10+ exported fields, account-shaped):
//     implementing CAN help. The reflective walker pays per-field op-dispatch
//     cost; a hand-written method that reads the whole struct in one shot
//     amortises that to zero. Bench shows 5–15% speedup on a ~30-field AMM
//     pool struct (see poolstate_decode_bench_test.go for methodology).
//     Whether the win materialises depends on the field-count vs decode-cost
//     ratio — bench before committing.
//
//   - Custom wire formats, layout transformations, post-decode validation:
//     implement without question — no other path expresses these.
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
// If the target type (or *T) implements Unmarshaler, the plan-cache emits
// a single dispatch op and reflection never descends into the type's fields.
//
// Otherwise the supported kinds are: uint8/16/32/64, int8/16/32/64, bool,
// [N]byte and [N]T fixed arrays, []byte and []T slices (length-prefixed),
// string (length-prefixed), pointer (bincode Option: 1-byte tag + optional
// payload), and struct (recursively, respecting `bin:"..."` field tags).
//
// The default length prefix is Rust's bincode u64. Override on a per-field
// basis with `bin:"sizePrefix=shortvec"` etc. Top-level slice decoding uses
// the default; to read a shortvec-prefixed slice at the top level, either
// wrap it in a struct or call ReadShortvec + a loop manually.
func (d *Decoder) DecodeTo(v any) error {
	// All reflective decoding goes through DecodeFast: compile the type's
	// decode plan once (memoised in decodePlanCache), then drive every
	// subsequent decode through pointer arithmetic without materialising
	// any per-value reflect.Value on the hot path.
	return d.DecodeFast(v)
}
