package encoding

import (
	"fmt"
	"math/big"
	"strconv"
	"unsafe"
)

// U128 is a 128-bit unsigned integer stored in Rust / Solana little-endian
// byte order: u[0] is the least-significant byte, u[15] the most-significant.
// This matches the wire representation, so ReadU128 / WriteU128 are a single
// 16-byte memcpy on little-endian hosts.
type U128 [16]byte

// U256 is a 256-bit unsigned integer in Rust / Solana little-endian byte order
// (u[0] least-significant, u[31] most-significant). Used by SPL programs that
// expose u256 fields (oracle prices, curve parameters, some concentrated-liquidity
// math) and by Ethereum-style bridged state.
type U256 [32]byte

// ----------------------------------------------------------------------------
// U128
// ----------------------------------------------------------------------------

// Lo returns the low 64 bits of u.
func (u U128) Lo() uint64 {
	return uint64(u[0]) | uint64(u[1])<<8 | uint64(u[2])<<16 | uint64(u[3])<<24 |
		uint64(u[4])<<32 | uint64(u[5])<<40 | uint64(u[6])<<48 | uint64(u[7])<<56
}

// Hi returns the high 64 bits of u.
func (u U128) Hi() uint64 {
	return uint64(u[8]) | uint64(u[9])<<8 | uint64(u[10])<<16 | uint64(u[11])<<24 |
		uint64(u[12])<<32 | uint64(u[13])<<40 | uint64(u[14])<<48 | uint64(u[15])<<56
}

// SetUint64 sets u to v (high 64 bits zero).
func (u *U128) SetUint64(v uint64) {
	*u = U128{}
	u[0] = byte(v)
	u[1] = byte(v >> 8)
	u[2] = byte(v >> 16)
	u[3] = byte(v >> 24)
	u[4] = byte(v >> 32)
	u[5] = byte(v >> 40)
	u[6] = byte(v >> 48)
	u[7] = byte(v >> 56)
}

// SetLoHi sets u from two 64-bit words.
func (u *U128) SetLoHi(lo, hi uint64) {
	u[0] = byte(lo)
	u[1] = byte(lo >> 8)
	u[2] = byte(lo >> 16)
	u[3] = byte(lo >> 24)
	u[4] = byte(lo >> 32)
	u[5] = byte(lo >> 40)
	u[6] = byte(lo >> 48)
	u[7] = byte(lo >> 56)
	u[8] = byte(hi)
	u[9] = byte(hi >> 8)
	u[10] = byte(hi >> 16)
	u[11] = byte(hi >> 24)
	u[12] = byte(hi >> 32)
	u[13] = byte(hi >> 40)
	u[14] = byte(hi >> 48)
	u[15] = byte(hi >> 56)
}

// BigInt returns u as a new non-negative *big.Int.
func (u U128) BigInt() *big.Int {
	var be [16]byte
	for i := 0; i < 16; i++ {
		be[i] = u[15-i]
	}
	return new(big.Int).SetBytes(be[:])
}

// SetBigInt sets u from x, which must be non-negative and fit in 128 bits.
// Returns an error otherwise; u is left unchanged on error.
func (u *U128) SetBigInt(x *big.Int) error {
	if x.Sign() < 0 {
		return fmt.Errorf("solana/encoding: U128.SetBigInt: negative value")
	}
	if x.BitLen() > 128 {
		return fmt.Errorf("solana/encoding: U128.SetBigInt: overflow (%d bits)", x.BitLen())
	}
	be := x.Bytes()
	var le U128
	for i, b := range be {
		le[len(be)-1-i] = b
	}
	*u = le
	return nil
}

// IsZero reports whether u == 0. Branch-free on amd64/arm64.
func (u U128) IsZero() bool {
	return u.Lo()|u.Hi() == 0
}

// String renders u as a decimal string, like %d for built-in uints.
func (u U128) String() string {
	if u.Hi() == 0 {
		return strconv.FormatUint(u.Lo(), 10)
	}
	return u.BigInt().String()
}

// MarshalText implements encoding.TextMarshaler (decimal string).
func (u U128) MarshalText() ([]byte, error) { return []byte(u.String()), nil }

// UnmarshalText parses a decimal U128 produced by MarshalText.
func (u *U128) UnmarshalText(b []byte) error {
	x, ok := new(big.Int).SetString(string(b), 10)
	if !ok {
		return fmt.Errorf("solana/encoding: U128.UnmarshalText: invalid decimal %q", b)
	}
	return u.SetBigInt(x)
}

// MarshalJSON emits u as a JSON *string* decimal, so JavaScript consumers
// that lose precision above 2^53 still receive the exact value.
func (u U128) MarshalJSON() ([]byte, error) {
	return strconv.AppendQuote(nil, u.String()), nil
}

// UnmarshalJSON accepts either a JSON string or a JSON number encoding a
// non-negative decimal U128.
func (u *U128) UnmarshalJSON(b []byte) error {
	if len(b) >= 2 && b[0] == '"' && b[len(b)-1] == '"' {
		b = b[1 : len(b)-1]
	}
	return u.UnmarshalText(b)
}

// ----------------------------------------------------------------------------
// U256
// ----------------------------------------------------------------------------

// Word returns the i-th 64-bit word of u (i = 0 is least significant, i = 3
// most significant). Panics if i is out of range.
func (u U256) Word(i int) uint64 {
	if i < 0 || i >= 4 {
		panic("U256.Word: index out of range")
	}
	off := i * 8
	return uint64(u[off]) | uint64(u[off+1])<<8 | uint64(u[off+2])<<16 | uint64(u[off+3])<<24 |
		uint64(u[off+4])<<32 | uint64(u[off+5])<<40 | uint64(u[off+6])<<48 | uint64(u[off+7])<<56
}

// SetWord sets the i-th 64-bit word of u. Panics if i is out of range.
func (u *U256) SetWord(i int, v uint64) {
	if i < 0 || i >= 4 {
		panic("U256.SetWord: index out of range")
	}
	off := i * 8
	u[off] = byte(v)
	u[off+1] = byte(v >> 8)
	u[off+2] = byte(v >> 16)
	u[off+3] = byte(v >> 24)
	u[off+4] = byte(v >> 32)
	u[off+5] = byte(v >> 40)
	u[off+6] = byte(v >> 48)
	u[off+7] = byte(v >> 56)
}

// BigInt returns u as a new non-negative *big.Int.
func (u U256) BigInt() *big.Int {
	var be [32]byte
	for i := 0; i < 32; i++ {
		be[i] = u[31-i]
	}
	return new(big.Int).SetBytes(be[:])
}

// SetBigInt sets u from x, which must be non-negative and fit in 256 bits.
// Returns an error otherwise; u is left unchanged on error.
func (u *U256) SetBigInt(x *big.Int) error {
	if x.Sign() < 0 {
		return fmt.Errorf("solana/encoding: U256.SetBigInt: negative value")
	}
	if x.BitLen() > 256 {
		return fmt.Errorf("solana/encoding: U256.SetBigInt: overflow (%d bits)", x.BitLen())
	}
	be := x.Bytes()
	var le U256
	for i, b := range be {
		le[len(be)-1-i] = b
	}
	*u = le
	return nil
}

// IsZero reports whether u == 0.
func (u U256) IsZero() bool {
	return u.Word(0)|u.Word(1)|u.Word(2)|u.Word(3) == 0
}

// String renders u as a decimal string.
func (u U256) String() string {
	if u.Word(1)|u.Word(2)|u.Word(3) == 0 {
		return strconv.FormatUint(u.Word(0), 10)
	}
	return u.BigInt().String()
}

// MarshalText implements encoding.TextMarshaler (decimal string).
func (u U256) MarshalText() ([]byte, error) { return []byte(u.String()), nil }

// UnmarshalText parses a decimal U256 produced by MarshalText.
func (u *U256) UnmarshalText(b []byte) error {
	x, ok := new(big.Int).SetString(string(b), 10)
	if !ok {
		return fmt.Errorf("solana/encoding: U256.UnmarshalText: invalid decimal %q", b)
	}
	return u.SetBigInt(x)
}

// MarshalJSON emits u as a JSON string decimal. See U128.MarshalJSON.
func (u U256) MarshalJSON() ([]byte, error) {
	return strconv.AppendQuote(nil, u.String()), nil
}

// UnmarshalJSON accepts either a JSON string or a JSON number encoding a
// non-negative decimal U256.
func (u *U256) UnmarshalJSON(b []byte) error {
	if len(b) >= 2 && b[0] == '"' && b[len(b)-1] == '"' {
		b = b[1 : len(b)-1]
	}
	return u.UnmarshalText(b)
}

// ----------------------------------------------------------------------------
// Encoder / Decoder methods
// ----------------------------------------------------------------------------

// WriteU128 appends u verbatim in 16 little-endian bytes.
func (e *Encoder) WriteU128(u U128) {
	e.buf = append(e.buf, u[:]...)
}

// WriteU256 appends u verbatim in 32 little-endian bytes.
func (e *Encoder) WriteU256(u U256) {
	e.buf = append(e.buf, u[:]...)
}

// ReadU128 reads a 16-byte little-endian U128. The bytes are copied, so the
// returned value is independent of the Decoder's underlying buffer.
func (d *Decoder) ReadU128() (U128, error) {
	b, err := d.ReadBytes(16)
	if err != nil {
		return U128{}, err
	}
	return *(*U128)(unsafe.Pointer(&b[0])), nil
}

// ReadU256 reads a 32-byte little-endian U256. See ReadU128 for ownership.
func (d *Decoder) ReadU256() (U256, error) {
	b, err := d.ReadBytes(32)
	if err != nil {
		return U256{}, err
	}
	return *(*U256)(unsafe.Pointer(&b[0])), nil
}

