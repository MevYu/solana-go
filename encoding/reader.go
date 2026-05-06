package encoding

import "strconv"

// Reader is a sticky-error wrapper around Decoder for chained, fluent
// decoding of fixed-shape Solana wire data. Each accessor reads one
// field; the first short-buffer (or invalid-shortvec) error is stored on
// the Reader and silences subsequent reads — so callers can sequence the
// happy path without per-call err checks and check Err() once at the end.
//
//	r := encoding.NewReader(data)
//	flag := r.U32()
//	var pk [32]byte
//	r.Read(pk[:])
//	supply := r.U64()
//	decimals := r.U8()
//	if err := r.Err(); err != nil {
//	    return nil, err
//	}
//
// The underlying Decoder is exposed via Decoder() for callers who need
// shortvec, position, or the reflective Decode path mid-stream.
type Reader struct {
	d   *Decoder
	err error
}

// NewReader wraps b in a Reader. The bytes are not copied.
func NewReader(b []byte) *Reader { return &Reader{d: NewDecoder(b)} }

// Decoder returns the underlying Decoder. Mutations through it are
// visible to subsequent Reader calls.
func (r *Reader) Decoder() *Decoder { return r.d }

// Err returns the first error encountered, or nil. Once non-nil, all
// subsequent reads are no-ops and return zero values.
func (r *Reader) Err() error { return r.err }

// Pos reports the current read offset.
func (r *Reader) Pos() int { return r.d.Pos() }

// Remaining reports the number of unread bytes.
func (r *Reader) Remaining() int { return r.d.Remaining() }

// Done returns nil if all bytes were consumed exactly. If a sticky read
// error is pending it returns that. Otherwise, if any unread bytes remain,
// it returns a *TrailingBytesError reporting how many.
//
// Use this to enforce "no extra data" at the end of a decode. Callers that
// want to allow trailing data should check Err() instead.
func (r *Reader) Done() error {
	if r.err != nil {
		return r.err
	}
	if rem := r.d.Remaining(); rem != 0 {
		return &TrailingBytesError{Remaining: rem}
	}
	return nil
}

// TrailingBytesError is returned by Reader.Done when the buffer was not
// fully consumed. Match it via errors.As:
//
//	var te *encoding.TrailingBytesError
//	if errors.As(err, &te) {
//	    // log te.Remaining, recover, etc.
//	}
type TrailingBytesError struct {
	// Remaining is the number of unread bytes at the end of the buffer.
	Remaining int
}

func (e *TrailingBytesError) Error() string {
	return "solana/encoding: " + strconv.Itoa(e.Remaining) + " trailing byte(s) after decode"
}

// ─── unsigned ────────────────────────────────────────────────────────────────

// U8 reads a single byte. Returns 0 if the Reader is in an error state.
func (r *Reader) U8() uint8 {
	if r.err != nil {
		return 0
	}
	v, err := r.d.ReadUint8()
	if err != nil {
		r.err = err
	}
	return v
}

// U16 reads a little-endian uint16.
func (r *Reader) U16() uint16 {
	if r.err != nil {
		return 0
	}
	v, err := r.d.ReadUint16()
	if err != nil {
		r.err = err
	}
	return v
}

// U32 reads a little-endian uint32.
func (r *Reader) U32() uint32 {
	if r.err != nil {
		return 0
	}
	v, err := r.d.ReadUint32()
	if err != nil {
		r.err = err
	}
	return v
}

// U64 reads a little-endian uint64.
func (r *Reader) U64() uint64 {
	if r.err != nil {
		return 0
	}
	v, err := r.d.ReadUint64()
	if err != nil {
		r.err = err
	}
	return v
}

// ─── signed ──────────────────────────────────────────────────────────────────

func (r *Reader) I8() int8   { return int8(r.U8()) }
func (r *Reader) I16() int16 { return int16(r.U16()) }
func (r *Reader) I32() int32 { return int32(r.U32()) }
func (r *Reader) I64() int64 { return int64(r.U64()) }

// ─── 128 / 256-bit ───────────────────────────────────────────────────────────

// U128 reads a 16-byte little-endian unsigned 128-bit integer.
func (r *Reader) U128() U128 {
	if r.err != nil {
		return U128{}
	}
	v, err := r.d.ReadU128()
	if err != nil {
		r.err = err
	}
	return v
}

// U256 reads a 32-byte little-endian unsigned 256-bit integer.
func (r *Reader) U256() U256 {
	if r.err != nil {
		return U256{}
	}
	v, err := r.d.ReadU256()
	if err != nil {
		r.err = err
	}
	return v
}

// ─── misc ────────────────────────────────────────────────────────────────────

// Bool reads 1 byte and reports whether it is non-zero.
func (r *Reader) Bool() bool { return r.U8() != 0 }

// Read fills out from the stream. The slice length determines how many
// bytes are consumed; passing pubkey[:] reads exactly 32 bytes.
func (r *Reader) Read(out []byte) {
	if r.err != nil {
		return
	}
	b, err := r.d.ReadBytes(len(out))
	if err != nil {
		r.err = err
		return
	}
	copy(out, b)
}

// Bytes32 reads exactly 32 bytes and returns them by value. Convenient
// for fields backed by solana.PublicKey or solana.Hash, which are both
// [32]byte.
func (r *Reader) Bytes32() [32]byte {
	var out [32]byte
	r.Read(out[:])
	return out
}

// Bytes64 reads exactly 64 bytes and returns them by value. Convenient
// for fields backed by solana.Signature, which is [64]byte.
func (r *Reader) Bytes64() [64]byte {
	var out [64]byte
	r.Read(out[:])
	return out
}

// Bytes reads exactly n bytes and returns them as a fresh, independent
// slice (the data is copied out of the underlying buffer). Use Read
// when you already have a destination slice.
func (r *Reader) Bytes(n int) []byte {
	out := make([]byte, n)
	r.Read(out)
	return out
}


// Skip advances the position by n bytes without copying.
func (r *Reader) Skip(n int) {
	if r.err != nil {
		return
	}
	if _, err := r.d.ReadBytes(n); err != nil {
		r.err = err
	}
}

// StrU64 reads a bincode string: u64 little-endian length, then UTF-8 bytes.
// For Borsh strings (u32 length) compose r.U32() + r.Bytes(int(n)).
func (r *Reader) StrU64() string {
	n := r.U64()
	if r.err != nil {
		return ""
	}
	b, err := r.d.ReadBytes(int(n))
	if err != nil {
		r.err = err
		return ""
	}
	return string(b)
}
