package encoding

import "testing"

// bigRec is a decently complex struct whose shape stresses the executor:
// primitives of several widths, two U128 leaves, a fixed [32]byte (Hash-like),
// a pointer (Option), a byte slice, and a nested slice of sub-records.
type bigRec struct {
	A   uint8
	B   uint16
	C   uint32
	D   uint64
	E   int64
	F   bool
	G   U128
	H   U256
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

// makeBigRec builds a serialised payload for a bigRec with `nsub` sub-records
// so benchmarks scale up the workload without hitting a decode fast path for
// zero-length slices.
func makeBigRecPayload(nsub int) []byte {
	enc := NewEncoder(32 + 56*nsub)
	enc.WriteUint8(0xab)
	enc.WriteUint16(0xcdef)
	enc.WriteUint32(0x12345678)
	enc.WriteUint64(0x1122334455667788)
	enc.WriteInt64(-42)
	enc.WriteUint8(1)
	var u128 U128
	u128.SetLoHi(0x1111111111111111, 0x2222222222222222)
	enc.WriteU128(u128)
	var u256 U256
	for i := range u256 {
		u256[i] = byte(i + 1)
	}
	enc.WriteU256(u256)
	for i := 0; i < 32; i++ {
		enc.WriteUint8(byte(i))
	}
	// Option<u64> = Some(999)
	enc.WriteUint8(1)
	enc.WriteUint64(999)
	enc.WriteUint16(0xbeef)
	// []byte("hello world")
	body := []byte("hello world")
	enc.WriteUint64(uint64(len(body)))
	enc.WriteBytes(body)
	// []subRec
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

func BenchmarkDecodePlanCache(b *testing.B) {
	data := makeBigRecPayload(50)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var v bigRec
		if err := NewDecoder(data).DecodeTo(&v); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecodeReflective(b *testing.B) {
	data := makeBigRecPayload(50)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var v bigRec
		if err := NewDecoder(data).decodeReflect(&v); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDecodeU128 stresses the registered-decoder short-circuit.
func BenchmarkDecodeU128(b *testing.B) {
	enc := NewEncoder(16)
	var u U128
	u.SetLoHi(0x1234567890abcdef, 0x1122334455667788)
	enc.WriteU128(u)
	data := enc.Bytes()
	b.ReportAllocs()
	b.SetBytes(16)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var v U128
		if err := NewDecoder(data).DecodeTo(&v); err != nil {
			b.Fatal(err)
		}
	}
}
