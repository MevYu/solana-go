package benchmarks

import (
	"testing"

	"github.com/MevYu/solana-go/encoding"
)

// bigRec exercises the reflective decode plan: primitives of several widths,
// two U128/U256 leaves, a fixed [32]byte, an Option pointer, a byte slice,
// and a nested slice of sub-records.
type bigRec struct {
	A   uint8
	B   uint16
	C   uint32
	D   uint64
	E   int64
	F   bool
	G   encoding.U128
	H   encoding.U256
	Key [32]byte
	Opt *uint64
	Tag uint16
	Mem []byte
	Sub []subRec
}

type subRec struct {
	K uint32
	V uint64
	P [32]byte
}

func makeBigRecPayload(nsub int) []byte {
	enc := encoding.NewEncoder(32 + 56*nsub)
	enc.WriteUint8(0xab)
	enc.WriteUint16(0xcdef)
	enc.WriteUint32(0x12345678)
	enc.WriteUint64(0x1122334455667788)
	enc.WriteInt64(-42)
	enc.WriteUint8(1)
	var u128 encoding.U128
	u128.SetLoHi(0x1111111111111111, 0x2222222222222222)
	enc.WriteU128(u128)
	var u256 encoding.U256
	for i := range u256 {
		u256[i] = byte(i + 1)
	}
	enc.WriteU256(u256)
	for i := 0; i < 32; i++ {
		enc.WriteUint8(byte(i))
	}
	enc.WriteUint8(1)
	enc.WriteUint64(999)
	enc.WriteUint16(0xbeef)
	body := []byte("hello world")
	enc.WriteUint64(uint64(len(body)))
	enc.WriteBytes(body)
	enc.WriteUint64(uint64(nsub))
	for i := 0; i < nsub; i++ {
		enc.WriteUint32(uint32(i))
		enc.WriteUint64(uint64(i) * 1000)
		for j := 0; j < 32; j++ {
			enc.WriteUint8(byte(j + i))
		}
	}
	out := make([]byte, len(enc.Bytes()))
	copy(out, enc.Bytes())
	return out
}

func BenchmarkDecodeBigRec(b *testing.B) {
	data := makeBigRecPayload(50)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var v bigRec
		if err := encoding.NewDecoder(data).DecodeTo(&v); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecodeU128(b *testing.B) {
	enc := encoding.NewEncoder(16)
	var u encoding.U128
	u.SetLoHi(0x1234567890abcdef, 0x1122334455667788)
	enc.WriteU128(u)
	data := enc.Bytes()
	b.ReportAllocs()
	b.SetBytes(16)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var v encoding.U128
		if err := encoding.NewDecoder(data).DecodeTo(&v); err != nil {
			b.Fatal(err)
		}
	}
}
