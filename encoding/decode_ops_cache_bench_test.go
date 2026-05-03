package encoding

import (
	"reflect"
	"testing"
	"unsafe"
)

func BenchmarkDecodeOps_CacheHit(b *testing.B) {
	t := reflect.TypeOf(bigRec{})
	if _, err := decodeOpsFor(t, tagOpts{}); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := decodeOpsFor(t, tagOpts{}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecodeOps_NoCache(b *testing.B) {
	t := reflect.TypeOf(bigRec{})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := compileDecodeOps(t, tagOpts{}); err != nil {
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
		p, err := compileDecodeOps(t, tagOpts{})
		if err != nil {
			b.Fatal(err)
		}
		var out bigRec
		d := NewDecoder(data)
		if err := d.execDecodeOps(p, unsafe.Pointer(&out)); err != nil {
			b.Fatal(err)
		}
	}
}
