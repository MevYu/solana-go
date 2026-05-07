package encoding

import (
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
	// All reflective decoding goes through DecodeFast: compile the type's
	// decode plan once (memoised in decodePlanCache), then drive every
	// subsequent decode through pointer arithmetic without materialising
	// any per-value reflect.Value on the hot path.
	return d.DecodeFast(v)
}
