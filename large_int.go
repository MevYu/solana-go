package solana

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/MevYu/solana-go/encoding"
)

// ----------------------------------------------------------------------------
// Uint128 / Int128
// ----------------------------------------------------------------------------

// Uint128 is a little-endian unsigned 128-bit integer.
// Lo holds bits [0:64); Hi holds bits [64:128).
type Uint128 struct {
	Lo uint64
	Hi uint64
}

func (u *Uint128) UnmarshalFromDecoder(d *encoding.Decoder) error {
	raw, err := d.ReadU128()
	if err != nil {
		return err
	}
	u.Lo = binary.LittleEndian.Uint64(raw[0:8])
	u.Hi = binary.LittleEndian.Uint64(raw[8:16])
	return nil
}

func (u Uint128) MarshalToEncoder(e *encoding.Encoder) {
	var raw encoding.U128
	binary.LittleEndian.PutUint64(raw[0:8], u.Lo)
	binary.LittleEndian.PutUint64(raw[8:16], u.Hi)
	e.WriteU128(raw)
}

func (u Uint128) BigInt() *big.Int {
	var b [16]byte
	binary.LittleEndian.PutUint64(b[0:8], u.Lo)
	binary.LittleEndian.PutUint64(b[8:16], u.Hi)
	reverse16(&b)
	return new(big.Int).SetBytes(b[:])
}

func (u Uint128) Bytes() [16]byte {
	var b [16]byte
	binary.LittleEndian.PutUint64(b[0:8], u.Lo)
	binary.LittleEndian.PutUint64(b[8:16], u.Hi)
	return b
}

func (u Uint128) IsZero() bool { return u.Lo == 0 && u.Hi == 0 }

func (u Uint128) String() string { return u.BigInt().String() }

func Uint128FromBigInt(x *big.Int) Uint128 {
	b := x.Bytes()
	var le [16]byte
	if len(b) > 16 {
		b = b[len(b)-16:]
	}
	copy(le[16-len(b):], b)
	reverse16(&le)
	return Uint128{
		Lo: binary.LittleEndian.Uint64(le[0:8]),
		Hi: binary.LittleEndian.Uint64(le[8:16]),
	}
}

// Int128 is a little-endian signed 128-bit integer.
type Int128 struct {
	Lo uint64
	Hi int64
}

func (i *Int128) UnmarshalFromDecoder(d *encoding.Decoder) error {
	raw, err := d.ReadU128()
	if err != nil {
		return err
	}
	i.Lo = binary.LittleEndian.Uint64(raw[0:8])
	i.Hi = int64(binary.LittleEndian.Uint64(raw[8:16]))
	return nil
}

func (i Int128) MarshalToEncoder(e *encoding.Encoder) {
	var raw encoding.U128
	binary.LittleEndian.PutUint64(raw[0:8], i.Lo)
	binary.LittleEndian.PutUint64(raw[8:16], uint64(i.Hi))
	e.WriteU128(raw)
}

func (i Int128) BigInt() *big.Int {
	var b [16]byte
	binary.LittleEndian.PutUint64(b[0:8], i.Lo)
	binary.LittleEndian.PutUint64(b[8:16], uint64(i.Hi))
	reverse16(&b)
	v := new(big.Int).SetBytes(b[:])
	if i.Hi < 0 {
		v.Sub(v, new(big.Int).Lsh(big.NewInt(1), 128))
	}
	return v
}

func (i Int128) IsZero() bool { return i.Lo == 0 && i.Hi == 0 }

func (i Int128) String() string { return i.BigInt().String() }

// ----------------------------------------------------------------------------
// Uint256 / Int256
// ----------------------------------------------------------------------------

// Uint256 is a little-endian unsigned 256-bit integer.
type Uint256 struct {
	Lo Uint128
	Hi Uint128
}

func (u *Uint256) UnmarshalFromDecoder(d *encoding.Decoder) error {
	loRaw, err := d.ReadU128()
	if err != nil {
		return fmt.Errorf("Uint256 lo: %w", err)
	}
	hiRaw, err := d.ReadU128()
	if err != nil {
		return fmt.Errorf("Uint256 hi: %w", err)
	}
	u.Lo = Uint128{
		Lo: binary.LittleEndian.Uint64(loRaw[0:8]),
		Hi: binary.LittleEndian.Uint64(loRaw[8:16]),
	}
	u.Hi = Uint128{
		Lo: binary.LittleEndian.Uint64(hiRaw[0:8]),
		Hi: binary.LittleEndian.Uint64(hiRaw[8:16]),
	}
	return nil
}

func (u Uint256) MarshalToEncoder(e *encoding.Encoder) {
	u.Lo.MarshalToEncoder(e)
	u.Hi.MarshalToEncoder(e)
}

func (u Uint256) Bytes() [32]byte {
	var b [32]byte
	lo := u.Lo.Bytes()
	hi := u.Hi.Bytes()
	copy(b[0:16], lo[:])
	copy(b[16:32], hi[:])
	return b
}

func (u Uint256) BigInt() *big.Int {
	b := u.Bytes()
	for i, j := 0, 31; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	return new(big.Int).SetBytes(b[:])
}

func (u Uint256) IsZero() bool { return u.Lo.IsZero() && u.Hi.IsZero() }

func (u Uint256) String() string { return u.BigInt().String() }

// Int256 is a little-endian signed 256-bit integer.
type Int256 struct {
	Lo Uint128
	Hi Int128
}

func (i *Int256) UnmarshalFromDecoder(d *encoding.Decoder) error {
	loRaw, err := d.ReadU128()
	if err != nil {
		return fmt.Errorf("Int256 lo: %w", err)
	}
	hiRaw, err := d.ReadU128()
	if err != nil {
		return fmt.Errorf("Int256 hi: %w", err)
	}
	i.Lo = Uint128{
		Lo: binary.LittleEndian.Uint64(loRaw[0:8]),
		Hi: binary.LittleEndian.Uint64(loRaw[8:16]),
	}
	i.Hi = Int128{
		Lo: binary.LittleEndian.Uint64(hiRaw[0:8]),
		Hi: int64(binary.LittleEndian.Uint64(hiRaw[8:16])),
	}
	return nil
}

func (i Int256) String() string {
	lo := i.Lo.Bytes()
	var hb [16]byte
	binary.LittleEndian.PutUint64(hb[0:8], i.Hi.Lo)
	binary.LittleEndian.PutUint64(hb[8:16], uint64(i.Hi.Hi))
	var b [32]byte
	copy(b[0:16], lo[:])
	copy(b[16:32], hb[:])
	for i, j := 0, 31; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	v := new(big.Int).SetBytes(b[:])
	if i.Hi.Hi < 0 {
		v.Sub(v, new(big.Int).Lsh(big.NewInt(1), 256))
	}
	return v.String()
}

// ----------------------------------------------------------------------------
// helpers
// ----------------------------------------------------------------------------

func reverse16(b *[16]byte) {
	for i, j := 0, 15; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
}
