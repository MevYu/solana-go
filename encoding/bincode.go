// Package encoding provides Solana wire-format serialization primitives.
//
// bincode.go implements the little-endian binary codec used for Solana
// transaction and message encoding, including Solana's compact-u16
// (shortvec) length-prefix format.
package encoding

import (
	"errors"
	"sync"
)

// ErrShortBuffer is returned when a Decoder runs out of bytes mid-value
// or when a caller asks for a negative read length.
var ErrShortBuffer = errors.New("solana/encoding: short buffer")

// ErrInvalidShortvec is returned when a compact-u16 encoding is
// structurally invalid: either the third byte sets the continuation bit
// (attempting a 4-byte encoding) or its value would overflow the
// destination uint16. The canonical Solana shortvec is 1-3 bytes and
// the third byte must be in the range 0x00-0x03.
var ErrInvalidShortvec = errors.New("solana/encoding: invalid shortvec")

// ----------------------------------------------------------------------------
// Encoder
// ----------------------------------------------------------------------------

// Encoder is an append-based writer for Solana's little-endian wire
// format. It holds a growable byte buffer and never reflects on its
// arguments.
type Encoder struct {
	buf []byte
}

// NewEncoder returns an Encoder whose internal buffer starts with the
// given initial capacity. Passing 0 is legal — the buffer grows on first
// write — but providing an estimate avoids the small re-alloc cost.
func NewEncoder(capacity int) *Encoder {
	return &Encoder{buf: make([]byte, 0, capacity)}
}

// New returns an Encoder with a 128-byte default capacity — large enough
// that essentially every Solana instruction encodes without a single
// buffer re-allocation. Use this when you don't care to compute the
// exact instruction-data size yourself; reach for NewEncoder(N) only
// when N is much larger than 128 (e.g. a variable-length address list).
func New() *Encoder {
	return &Encoder{buf: make([]byte, 0, 128)}
}

var encoderPool = sync.Pool{New: func() any { return &Encoder{} }}

// AcquireEncoder returns a pooled Encoder reset to zero length.
// initialCap is a capacity hint; the pooled buffer may already be larger.
// The caller must call ReleaseEncoder when done — do not use the Encoder
// after calling ReleaseEncoder.
func AcquireEncoder(initialCap int) *Encoder {
	e := encoderPool.Get().(*Encoder)
	if cap(e.buf) < initialCap {
		e.buf = make([]byte, 0, initialCap)
	} else {
		e.buf = e.buf[:0]
	}
	return e
}

// ReleaseEncoder returns e to the pool. The caller must not use e after this.
func ReleaseEncoder(e *Encoder) {
	e.buf = e.buf[:0]
	encoderPool.Put(e)
}

// Wrap returns an Encoder whose buffer is the slice b truncated to
// zero length. The underlying array is reused, so callers who want to
// preserve b's contents must not write through the returned Encoder.
func Wrap(b []byte) *Encoder {
	return &Encoder{buf: b[:0]}
}

// Bytes returns the accumulated bytes. The returned slice aliases the
// internal buffer and is valid only until the next write.
func (e *Encoder) Bytes() []byte { return e.buf }

// Len returns the number of bytes written so far.
func (e *Encoder) Len() int { return len(e.buf) }

// Reset truncates the buffer length to zero, keeping the allocated
// capacity so the Encoder can be reused for a new encoding run.
func (e *Encoder) Reset() { e.buf = e.buf[:0] }

// WriteUint8 appends a single byte.
func (e *Encoder) WriteUint8(v uint8) {
	e.buf = append(e.buf, v)
}

// WriteUint16 appends a little-endian uint16.
func (e *Encoder) WriteUint16(v uint16) {
	e.buf = append(e.buf, byte(v), byte(v>>8))
}

// WriteUint32 appends a little-endian uint32.
func (e *Encoder) WriteUint32(v uint32) {
	e.buf = append(e.buf,
		byte(v), byte(v>>8), byte(v>>16), byte(v>>24))
}

// WriteUint64 appends a little-endian uint64.
func (e *Encoder) WriteUint64(v uint64) {
	e.buf = append(e.buf,
		byte(v), byte(v>>8), byte(v>>16), byte(v>>24),
		byte(v>>32), byte(v>>40), byte(v>>48), byte(v>>56))
}

// WriteInt8 appends a signed int8.
func (e *Encoder) WriteInt8(v int8) { e.WriteUint8(uint8(v)) }

// WriteInt16 appends a little-endian int16.
func (e *Encoder) WriteInt16(v int16) { e.WriteUint16(uint16(v)) }

// WriteInt32 appends a little-endian int32.
func (e *Encoder) WriteInt32(v int32) { e.WriteUint32(uint32(v)) }

// WriteInt64 appends a little-endian int64.
func (e *Encoder) WriteInt64(v int64) { e.WriteUint64(uint64(v)) }

// WriteBytes appends b verbatim, without any length prefix.
func (e *Encoder) WriteBytes(b []byte) {
	e.buf = append(e.buf, b...)
}

// WriteShortvec appends a Solana compact-u16 length prefix (1-3 bytes),
// encoding v using EncodeShortvec.
func (e *Encoder) WriteShortvec(v uint16) {
	var tmp [3]byte
	n := EncodeShortvec(tmp[:], v)
	e.buf = append(e.buf, tmp[:n]...)
}

// ----------------------------------------------------------------------------
// Decoder
// ----------------------------------------------------------------------------

// Decoder is a zero-copy reader for Solana's little-endian wire format.
// It holds a cursor into a caller-owned byte slice; ReadBytes returns
// slices aliasing that buffer.
type Decoder struct {
	buf []byte
	pos int

	// defaultPrefix overrides the implied length prefix for Vec<T> /
	// string fields whose `bin:"..."` tag does not pick one explicitly.
	// Zero value (= prefixDefault) preserves the bincode u64 default;
	// setting it to prefixU32 switches the same codec into Borsh mode.
	defaultPrefix sizePrefix
}

// UseBorsh switches the decoder into Borsh mode: untagged Vec<T> /
// string fields use a u32 length prefix instead of the bincode u64
// default. Returns d for chaining.
//
// Per-field `bin:"sizePrefix=..."` tags still take precedence; UseBorsh
// only changes the implied default. Other Borsh / bincode differences
// (1-byte vs 4-byte enum discriminator) are not auto-translated — model
// enums as uint8 (Borsh) or uint32 (bincode) on the Go side.
func (d *Decoder) UseBorsh() *Decoder { d.defaultPrefix = prefixU32; return d }

// NewDecoder returns a Decoder that reads from b. The caller retains
// ownership of b and must not mutate it for the lifetime of the Decoder
// or of any slice returned by ReadBytes.
func NewDecoder(b []byte) *Decoder {
	return &Decoder{buf: b}
}

// DecodeTo is a one-shot reflection decoder: equivalent to
// NewDecoder(data).DecodeTo(v). Use this when you have a struct tagged
// with bin:"..." and just want to parse a buffer into it. For
// fixed-shape, performance-sensitive decoders prefer the Reader API.
func DecodeTo(data []byte, v any) error {
	return NewDecoder(data).DecodeTo(v)
}

// BorshDecodeTo is the Borsh-flavoured one-shot reflection decoder:
// equivalent to NewDecoder(data).UseBorsh().DecodeTo(v). It flips the
// default Vec/string length prefix from bincode's u64 to Borsh's u32.
// Use this for Anchor-generated account state and other Borsh-encoded
// structs.
//
// Anchor's 8-byte account discriminator must be stripped (or modeled as
// the first [8]byte field) before calling. Borsh enums are 1-byte —
// model them as a Go uint8 with one explicit case per variant.
func BorshDecodeTo(data []byte, v any) error {
	return NewDecoder(data).UseBorsh().DecodeTo(v)
}

var decoderPool = sync.Pool{New: func() any { return &Decoder{} }}

// AcquireDecoder returns a pooled Decoder reset to read from b. The
// caller must call ReleaseDecoder when done — do not use the Decoder
// after that call.
//
// Note: ReadBytes returns zero-copy slices into b. Those slices remain
// valid after ReleaseDecoder because the pool only reuses the Decoder
// struct, not the underlying data buffer.
func AcquireDecoder(b []byte) *Decoder {
	d := decoderPool.Get().(*Decoder)
	d.buf = b
	d.pos = 0
	return d
}

// ReleaseDecoder returns d to the pool. The caller must not use d after this.
func ReleaseDecoder(d *Decoder) {
	d.buf = nil
	decoderPool.Put(d)
}

// Pos returns the number of bytes consumed so far.
func (d *Decoder) Pos() int { return d.pos }

// Remaining returns the number of bytes left to read.
func (d *Decoder) Remaining() int { return len(d.buf) - d.pos }

// ReadUint8 reads a single byte.
func (d *Decoder) ReadUint8() (uint8, error) {
	if d.Remaining() < 1 {
		return 0, ErrShortBuffer
	}
	v := d.buf[d.pos]
	d.pos++
	return v, nil
}

// PeekUint8 returns the next byte without advancing the cursor.
func (d *Decoder) PeekUint8() (uint8, error) {
	if d.Remaining() < 1 {
		return 0, ErrShortBuffer
	}
	return d.buf[d.pos], nil
}

// ReadUint16 reads a little-endian uint16.
func (d *Decoder) ReadUint16() (uint16, error) {
	if d.Remaining() < 2 {
		return 0, ErrShortBuffer
	}
	b := d.buf[d.pos:]
	v := uint16(b[0]) | uint16(b[1])<<8
	d.pos += 2
	return v, nil
}

// ReadUint32 reads a little-endian uint32.
func (d *Decoder) ReadUint32() (uint32, error) {
	if d.Remaining() < 4 {
		return 0, ErrShortBuffer
	}
	b := d.buf[d.pos:]
	v := uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
	d.pos += 4
	return v, nil
}

// ReadUint64 reads a little-endian uint64.
func (d *Decoder) ReadUint64() (uint64, error) {
	if d.Remaining() < 8 {
		return 0, ErrShortBuffer
	}
	b := d.buf[d.pos:]
	v := uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
		uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56
	d.pos += 8
	return v, nil
}

// ReadInt8 reads a signed int8.
func (d *Decoder) ReadInt8() (int8, error) {
	v, err := d.ReadUint8()
	return int8(v), err
}

// ReadInt16 reads a little-endian int16.
func (d *Decoder) ReadInt16() (int16, error) {
	v, err := d.ReadUint16()
	return int16(v), err
}

// ReadInt32 reads a little-endian int32.
func (d *Decoder) ReadInt32() (int32, error) {
	v, err := d.ReadUint32()
	return int32(v), err
}

// ReadInt64 reads a little-endian int64.
func (d *Decoder) ReadInt64() (int64, error) {
	v, err := d.ReadUint64()
	return int64(v), err
}

// ReadBytes returns a zero-copy subslice of n bytes aliasing the
// underlying buffer. The returned slice is valid only as long as the
// buffer passed to NewDecoder is valid, and its capacity is clamped to
// n so that a rogue append cannot overwrite neighbouring bytes.
func (d *Decoder) ReadBytes(n int) ([]byte, error) {
	if n < 0 {
		return nil, ErrShortBuffer
	}
	if d.Remaining() < n {
		return nil, ErrShortBuffer
	}
	out := d.buf[d.pos : d.pos+n : d.pos+n]
	d.pos += n
	return out, nil
}

// ReadShortvec decodes a Solana compact-u16 length prefix and advances
// the cursor by the number of bytes consumed. On error the cursor is
// left unchanged.
func (d *Decoder) ReadShortvec() (uint16, error) {
	v, n, err := DecodeShortvec(d.buf[d.pos:])
	if err != nil {
		return 0, err
	}
	d.pos += n
	return v, nil
}

// ----------------------------------------------------------------------------
// Compact-u16 (shortvec)
// ----------------------------------------------------------------------------

// EncodeShortvec writes v into b using Solana's compact-u16 (shortvec)
// encoding and returns the number of bytes written. b must have length
// at least 3. The encoding is 1-3 bytes: each byte holds 7 data bits in
// the low bits and a continuation flag in the high bit.
func EncodeShortvec(b []byte, v uint16) int {
	if v < 0x80 {
		b[0] = byte(v)
		return 1
	}
	if v < 0x4000 {
		b[0] = byte(v) | 0x80
		b[1] = byte(v >> 7)
		return 2
	}
	b[0] = byte(v) | 0x80
	b[1] = byte(v>>7) | 0x80
	b[2] = byte(v >> 14)
	return 3
}

// DecodeShortvec reads a compact-u16 from b and returns the decoded
// value, the number of bytes consumed, and an error. The error is
// ErrShortBuffer when b is truncated mid-value, and ErrInvalidShortvec
// when the encoding is structurally invalid (fourth-byte continuation
// or uint16 overflow).
func DecodeShortvec(b []byte) (uint16, int, error) {
	if len(b) == 0 {
		return 0, 0, ErrShortBuffer
	}
	b0 := b[0]
	if b0 < 0x80 {
		return uint16(b0), 1, nil
	}
	if len(b) < 2 {
		return 0, 0, ErrShortBuffer
	}
	b1 := b[1]
	v := uint16(b0&0x7f) | uint16(b1&0x7f)<<7
	if b1 < 0x80 {
		return v, 2, nil
	}
	if len(b) < 3 {
		return 0, 0, ErrShortBuffer
	}
	b2 := b[2]
	// The third byte is the most significant. Valid values are 0x00-0x03:
	// any higher value either sets the continuation bit (attempting a
	// 4-byte encoding) or would overflow the destination uint16 when
	// shifted left by 14.
	if b2 > 0x03 {
		return 0, 0, ErrInvalidShortvec
	}
	v |= uint16(b2) << 14
	return v, 3, nil
}
