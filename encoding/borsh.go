package encoding

import (
	"fmt"
	"math"
)

// ----------------------------------------------------------------------------
// Borsh encoder
// ----------------------------------------------------------------------------

// BorshEncoder writes Borsh-encoded data into a growable byte buffer.
// Borsh is a deterministic binary serialization format used by Solana
// programs for on-chain account state.
type BorshEncoder struct {
	buf []byte
}

// NewBorshEncoder returns a BorshEncoder with the given initial capacity.
func NewBorshEncoder(capacity int) *BorshEncoder {
	return &BorshEncoder{buf: make([]byte, 0, capacity)}
}

// BorshDecodeTo decodes data into v using Borsh prefix conventions.
func BorshDecodeTo(data []byte, v any) error {
	return NewDecoder(data).UseBorsh().DecodeTo(v)
}

// UseBorsh switches the bincode Decoder into Borsh mode (u32 length prefix).
func (d *Decoder) UseBorsh() *Decoder { d.defaultPrefix = prefixU32; return d }

// Bytes returns the accumulated encoded bytes. The slice aliases the
// internal buffer and is valid only until the next write.
func (e *BorshEncoder) Bytes() []byte { return e.buf }

// Len returns the number of bytes written so far.
func (e *BorshEncoder) Len() int { return len(e.buf) }

// WriteBool writes a Borsh boolean (0x00 or 0x01).
func (e *BorshEncoder) WriteBool(v bool) {
	if v {
		e.buf = append(e.buf, 1)
	} else {
		e.buf = append(e.buf, 0)
	}
}

// WriteU8 writes a Borsh uint8.
func (e *BorshEncoder) WriteU8(v uint8) {
	e.buf = append(e.buf, v)
}

// WriteI8 writes a Borsh int8.
func (e *BorshEncoder) WriteI8(v int8) { e.WriteU8(uint8(v)) }

// WriteU16 writes a Borsh uint16 (little-endian).
func (e *BorshEncoder) WriteU16(v uint16) {
	e.buf = append(e.buf, byte(v), byte(v>>8))
}

// WriteI16 writes a Borsh int16 (little-endian).
func (e *BorshEncoder) WriteI16(v int16) { e.WriteU16(uint16(v)) }

// WriteU32 writes a Borsh uint32 (little-endian).
func (e *BorshEncoder) WriteU32(v uint32) {
	e.buf = append(e.buf,
		byte(v), byte(v>>8), byte(v>>16), byte(v>>24))
}

// WriteI32 writes a Borsh int32 (little-endian).
func (e *BorshEncoder) WriteI32(v int32) { e.WriteU32(uint32(v)) }

// WriteU64 writes a Borsh uint64 (little-endian).
func (e *BorshEncoder) WriteU64(v uint64) {
	e.buf = append(e.buf,
		byte(v), byte(v>>8), byte(v>>16), byte(v>>24),
		byte(v>>32), byte(v>>40), byte(v>>48), byte(v>>56))
}

// WriteI64 writes a Borsh int64 (little-endian).
func (e *BorshEncoder) WriteI64(v int64) { e.WriteU64(uint64(v)) }

// WriteU128 writes a Borsh uint128 as two little-endian uint64 words
// (lo then hi).
func (e *BorshEncoder) WriteU128(lo, hi uint64) {
	e.WriteU64(lo)
	e.WriteU64(hi)
}

// WriteF32 writes a Borsh f32 (IEEE 754, little-endian).
func (e *BorshEncoder) WriteF32(v float32) {
	e.WriteU32(math.Float32bits(v))
}

// WriteF64 writes a Borsh f64 (IEEE 754, little-endian).
func (e *BorshEncoder) WriteF64(v float64) {
	e.WriteU64(math.Float64bits(v))
}

// WriteBytes writes a Borsh byte array: a u32 length prefix followed
// by the raw bytes.
func (e *BorshEncoder) WriteBytes(b []byte) {
	e.WriteU32(uint32(len(b)))
	e.buf = append(e.buf, b...)
}

// WriteString writes a Borsh string (UTF-8 bytes with a u32 length prefix).
func (e *BorshEncoder) WriteString(s string) {
	e.WriteU32(uint32(len(s)))
	e.buf = append(e.buf, s...)
}

// WriteFixedBytes writes b verbatim with no length prefix (for
// fixed-size arrays like public keys and hashes).
func (e *BorshEncoder) WriteFixedBytes(b []byte) {
	e.buf = append(e.buf, b...)
}

// WriteOption writes a Borsh Option<T>: 0x00 if none is true, or 0x01
// followed by the caller-written payload otherwise. The caller is
// responsible for writing the inner value after WriteOption(false).
func (e *BorshEncoder) WriteOption(none bool) {
	if none {
		e.buf = append(e.buf, 0)
	} else {
		e.buf = append(e.buf, 1)
	}
}

// ----------------------------------------------------------------------------
// Borsh decoder
// ----------------------------------------------------------------------------

// BorshDecoder reads Borsh-encoded data from a caller-owned byte slice.
type BorshDecoder struct {
	buf []byte
	pos int
}

// NewBorshDecoder returns a BorshDecoder that reads from b.
func NewBorshDecoder(b []byte) *BorshDecoder {
	return &BorshDecoder{buf: b}
}

// Pos returns the number of bytes consumed so far.
func (d *BorshDecoder) Pos() int { return d.pos }

// Remaining returns the number of bytes left to read.
func (d *BorshDecoder) Remaining() int { return len(d.buf) - d.pos }

func (d *BorshDecoder) need(n int) error {
	if d.Remaining() < n {
		return fmt.Errorf("solana/encoding: borsh: need %d bytes, have %d", n, d.Remaining())
	}
	return nil
}

// ReadBool reads a Borsh boolean.
func (d *BorshDecoder) ReadBool() (bool, error) {
	if err := d.need(1); err != nil {
		return false, err
	}
	b := d.buf[d.pos]
	d.pos++
	if b == 0 {
		return false, nil
	}
	if b == 1 {
		return true, nil
	}
	return false, fmt.Errorf("solana/encoding: borsh: invalid bool byte 0x%02x", b)
}

// ReadU8 reads a Borsh uint8.
func (d *BorshDecoder) ReadU8() (uint8, error) {
	if err := d.need(1); err != nil {
		return 0, err
	}
	v := d.buf[d.pos]
	d.pos++
	return v, nil
}

// ReadI8 reads a Borsh int8.
func (d *BorshDecoder) ReadI8() (int8, error) {
	v, err := d.ReadU8()
	return int8(v), err
}

// ReadU16 reads a Borsh uint16 (little-endian).
func (d *BorshDecoder) ReadU16() (uint16, error) {
	if err := d.need(2); err != nil {
		return 0, err
	}
	b := d.buf[d.pos:]
	v := uint16(b[0]) | uint16(b[1])<<8
	d.pos += 2
	return v, nil
}

// ReadI16 reads a Borsh int16 (little-endian).
func (d *BorshDecoder) ReadI16() (int16, error) {
	v, err := d.ReadU16()
	return int16(v), err
}

// ReadU32 reads a Borsh uint32 (little-endian).
func (d *BorshDecoder) ReadU32() (uint32, error) {
	if err := d.need(4); err != nil {
		return 0, err
	}
	b := d.buf[d.pos:]
	v := uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
	d.pos += 4
	return v, nil
}

// ReadI32 reads a Borsh int32 (little-endian).
func (d *BorshDecoder) ReadI32() (int32, error) {
	v, err := d.ReadU32()
	return int32(v), err
}

// ReadU64 reads a Borsh uint64 (little-endian).
func (d *BorshDecoder) ReadU64() (uint64, error) {
	if err := d.need(8); err != nil {
		return 0, err
	}
	b := d.buf[d.pos:]
	v := uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
		uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56
	d.pos += 8
	return v, nil
}

// ReadI64 reads a Borsh int64 (little-endian).
func (d *BorshDecoder) ReadI64() (int64, error) {
	v, err := d.ReadU64()
	return int64(v), err
}

// ReadU128 reads a Borsh uint128 as (lo uint64, hi uint64).
func (d *BorshDecoder) ReadU128() (lo, hi uint64, err error) {
	if lo, err = d.ReadU64(); err != nil {
		return
	}
	hi, err = d.ReadU64()
	return
}

// ReadF32 reads a Borsh f32.
func (d *BorshDecoder) ReadF32() (float32, error) {
	v, err := d.ReadU32()
	return math.Float32frombits(v), err
}

// ReadF64 reads a Borsh f64.
func (d *BorshDecoder) ReadF64() (float64, error) {
	v, err := d.ReadU64()
	return math.Float64frombits(v), err
}

// ReadBytes reads a Borsh byte array: a u32 length prefix then that
// many bytes, returned as a copy.
func (d *BorshDecoder) ReadBytes() ([]byte, error) {
	n, err := d.ReadU32()
	if err != nil {
		return nil, err
	}
	if err := d.need(int(n)); err != nil {
		return nil, err
	}
	out := make([]byte, n)
	copy(out, d.buf[d.pos:d.pos+int(n)])
	d.pos += int(n)
	return out, nil
}

// ReadString reads a Borsh string (u32 length prefix + UTF-8 bytes).
func (d *BorshDecoder) ReadString() (string, error) {
	b, err := d.ReadBytes()
	return string(b), err
}

// ReadFixedBytes reads exactly n bytes with no length prefix (for
// fixed-size arrays like public keys and hashes). Returns a copy.
func (d *BorshDecoder) ReadFixedBytes(n int) ([]byte, error) {
	if err := d.need(n); err != nil {
		return nil, err
	}
	out := make([]byte, n)
	copy(out, d.buf[d.pos:d.pos+n])
	d.pos += n
	return out, nil
}

// ReadOption reads a Borsh Option tag byte. Returns (true, nil) when
// the tag is 0x01 (Some), (false, nil) when it is 0x00 (None). The
// caller is responsible for reading the inner value when some == true.
func (d *BorshDecoder) ReadOption() (some bool, err error) {
	tag, err := d.ReadU8()
	if err != nil {
		return false, err
	}
	switch tag {
	case 0:
		return false, nil
	case 1:
		return true, nil
	default:
		return false, fmt.Errorf("solana/encoding: borsh: invalid option tag 0x%02x", tag)
	}
}

// ----------------------------------------------------------------------------
// Borsh mode for the shared Decoder
// ----------------------------------------------------------------------------

