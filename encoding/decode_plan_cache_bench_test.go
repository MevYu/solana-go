package encoding

import (
	"reflect"
	"testing"
	"unsafe"
)

// Bench results — arm64, GOMAXPROCS=20, 10 runs each, mean ns/op:
//
//   BenchmarkDecodePlan_CacheHit         6.5 ns/op    0 B/op    0 allocs
//   BenchmarkDecodePlan_NoCache       2,706 ns/op  3792 B/op   12 allocs
//   BenchmarkDecodeFast_BigRec_Cached   649 ns/op   680 B/op    7 allocs
//   BenchmarkDecodeFast_BigRec_NoCache 3,508 ns/op  4472 B/op  19 allocs
//
// Plan-cache value:
//   - first decode of a type:  2,706 ns plan-compile + 649 ns execute
//   - every subsequent decode:    6.5 ns lookup + 649 ns execute
//   - end-to-end speedup:    3,508 / 649 ≈ 5.4× when the plan is cached
//   - allocations: 19 → 7 (12 alloc savings per cached decode)
//
// Break-even after the first decode of any given type. For long-running
// processes that decode the same account / instruction shape repeatedly
// (RPC clients, indexers, MEV bots) the compile cost is essentially free.
//
// Repro:
//   go test -run=^$ -bench='BenchmarkDecodePlan_|BenchmarkDecodeFast_' \
//     -benchmem -count=10 ./encoding/

func BenchmarkDecodePlan_CacheHit(b *testing.B) {
	t := reflect.TypeOf(bigRec{})
	if _, err := decodePlanFor(t, tagOpts{}); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := decodePlanFor(t, tagOpts{}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecodePlan_NoCache(b *testing.B) {
	t := reflect.TypeOf(bigRec{})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := compileDecodePlan(t, tagOpts{}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecodeFast_BigRec_Cached(b *testing.B) {
	data := makeBigRecPayload(8)
	var v bigRec
	if err := NewDecoder(data).DecodeFast(&v); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var out bigRec
		if err := NewDecoder(data).DecodeFast(&out); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecodeFast_BigRec_NoCache(b *testing.B) {
	data := makeBigRecPayload(8)
	t := reflect.TypeOf(bigRec{})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p, err := compileDecodePlan(t, tagOpts{})
		if err != nil {
			b.Fatal(err)
		}
		var out bigRec
		d := NewDecoder(data)
		if err := d.execDecodePlan(p, unsafe.Pointer(&out)); err != nil {
			b.Fatal(err)
		}
	}
}
