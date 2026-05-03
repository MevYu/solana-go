package encoding

import "testing"

func writeSmallPayload(e *Encoder) {
	e.WriteUint8(1)
	e.WriteUint16(2)
	e.WriteUint32(3)
	e.WriteUint64(4)
	e.WriteBytes(make([]byte, 32))
}

func writeMediumPayload(e *Encoder) {
	for i := 0; i < 8; i++ {
		writeSmallPayload(e)
	}
}

func writeLargePayload(e *Encoder) {
	for i := 0; i < 64; i++ {
		writeSmallPayload(e)
	}
}

func BenchmarkEncoder_New_Default_Small(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		e := New()
		writeSmallPayload(e)
		_ = e.Bytes()
	}
}

func BenchmarkEncoder_NewEncoder_Sized_Small(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		e := NewEncoder(64)
		writeSmallPayload(e)
		_ = e.Bytes()
	}
}

func BenchmarkEncoder_AcquireRelease_Small(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		e := AcquireEncoder(64)
		writeSmallPayload(e)
		_ = e.Bytes()
		ReleaseEncoder(e)
	}
}

func BenchmarkEncoder_New_Default_Medium(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		e := New()
		writeMediumPayload(e)
		_ = e.Bytes()
	}
}

func BenchmarkEncoder_NewEncoder_Sized_Medium(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		e := NewEncoder(512)
		writeMediumPayload(e)
		_ = e.Bytes()
	}
}

func BenchmarkEncoder_AcquireRelease_Medium(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		e := AcquireEncoder(512)
		writeMediumPayload(e)
		_ = e.Bytes()
		ReleaseEncoder(e)
	}
}

func BenchmarkEncoder_New_Default_Large(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		e := New()
		writeLargePayload(e)
		_ = e.Bytes()
	}
}

func BenchmarkEncoder_NewEncoder_Sized_Large(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		e := NewEncoder(4096)
		writeLargePayload(e)
		_ = e.Bytes()
	}
}

func BenchmarkEncoder_AcquireRelease_Large(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		e := AcquireEncoder(4096)
		writeLargePayload(e)
		_ = e.Bytes()
		ReleaseEncoder(e)
	}
}

func BenchmarkEncoder_AcquireRelease_Parallel_Medium(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			e := AcquireEncoder(512)
			writeMediumPayload(e)
			_ = e.Bytes()
			ReleaseEncoder(e)
		}
	})
}

func BenchmarkEncoder_NewEncoder_Sized_Parallel_Medium(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			e := NewEncoder(512)
			writeMediumPayload(e)
			_ = e.Bytes()
		}
	})
}
