package solana

import (
	"testing"

	"github.com/MevYu/solana-go/encoding"
)

// These benches now measure the reflective baseline only — leaf-type
// override paths (RegisterDecoder + opCallFunc) were deleted after bench
// data showed they were net-negative on [N]byte-backed named types
// (1.5–6.5% slower under load due to BTB pressure on the indirect call
// vs the inlined opFixedBytes switch arm). Kept as regression catchers
// for the plan-cache / opFixedBytes path.

// Account-like struct: 4 PublicKey + 2 u64 + 1 u32, mimics the leaf shape
// of a token-program account or AMM state.
type pubkeyBenchAccount struct {
	A PublicKey
	B PublicKey
	C PublicKey
	D PublicKey
	E uint64
	F uint64
	G uint32
}

// Slice-of-Pubkey struct: mimics Message.AccountKeys (variable-length).
type pubkeyBenchSlice struct {
	Keys []PublicKey
}

func makePubkeyBenchAccountData() []byte {
	e := encoding.NewEncoder(4*PublicKeySize + 8 + 8 + 4)
	for i := 0; i < 4; i++ {
		var pk PublicKey
		pk[0] = byte(i + 1)
		e.WriteBytes(pk[:])
	}
	e.WriteUint64(1)
	e.WriteUint64(2)
	e.WriteUint32(3)
	return e.Bytes()
}

func makePubkeyBenchSliceData(n int) []byte {
	e := encoding.NewEncoder(8 + n*PublicKeySize)
	e.WriteUint64(uint64(n))
	for i := 0; i < n; i++ {
		var pk PublicKey
		pk[0] = byte(i + 1)
		e.WriteBytes(pk[:])
	}
	return e.Bytes()
}

func BenchmarkDecodePubkeyAccount(b *testing.B) {
	data := makePubkeyBenchAccountData()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var v pubkeyBenchAccount
		if err := encoding.BinDecodeTo(data, &v); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecodePubkeySingle(b *testing.B) {
	var pk PublicKey
	pk[0] = 0xAB
	data := pk[:]
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var v PublicKey
		if err := encoding.BinDecodeTo(data, &v); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecodePubkeySlice32(b *testing.B) {
	data := makePubkeyBenchSliceData(32)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var v pubkeyBenchSlice
		if err := encoding.BinDecodeTo(data, &v); err != nil {
			b.Fatal(err)
		}
	}
}

// Parallel variants exercise the read-only path (registry / plan cache are
// sync.Map; each goroutine has its own Decoder). If hidden contention
// existed in the registered fast path it would show here.

func BenchmarkDecodePubkeyAccountParallel(b *testing.B) {
	data := makePubkeyBenchAccountData()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var v pubkeyBenchAccount
			if err := encoding.BinDecodeTo(data, &v); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkDecodePubkeySlice32Parallel(b *testing.B) {
	data := makePubkeyBenchSliceData(32)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var v pubkeyBenchSlice
			if err := encoding.BinDecodeTo(data, &v); err != nil {
				b.Fatal(err)
			}
		}
	})
}
