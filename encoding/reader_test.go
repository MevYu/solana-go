package encoding

import (
	"bytes"
	"testing"
)

func TestReader_HappyPath(t *testing.T) {
	data := New().U8(0xAB).U16(0xCDEF).U32(0x12345678).U64(0xDEADBEEFCAFEBABE).Bytes()
	r := NewReader(data)
	if r.U8() != 0xAB {
		t.Error("U8 mismatch")
	}
	if r.U16() != 0xCDEF {
		t.Error("U16 mismatch")
	}
	if r.U32() != 0x12345678 {
		t.Error("U32 mismatch")
	}
	if r.U64() != 0xDEADBEEFCAFEBABE {
		t.Error("U64 mismatch")
	}
	if err := r.Done(); err != nil {
		t.Errorf("Done: %v", err)
	}
}

func TestReader_Bool(t *testing.T) {
	data := New().Bool(true).Bool(false).Bytes()
	r := NewReader(data)
	if !r.Bool() {
		t.Error("expected true")
	}
	if r.Bool() {
		t.Error("expected false")
	}
}

func TestReader_Pubkey(t *testing.T) {
	pk := [32]byte{0x01, 0x02, 0x03}
	data := New().Raw(pk[:]).Bytes()
	r := NewReader(data)
	var out [32]byte
	r.Read(out[:])
	if !bytes.Equal(out[:], pk[:]) {
		t.Errorf("Pubkey mismatch: %x", out)
	}
}

func TestReader_Str(t *testing.T) {
	data := New().Str("hello").Bytes()
	r := NewReader(data)
	if got := r.Str(); got != "hello" {
		t.Errorf("Str = %q", got)
	}
}

func TestReader_ShortBuffer_StickyError(t *testing.T) {
	data := []byte{0x01, 0x02} // only 2 bytes
	r := NewReader(data)
	r.U8() // ok, consumes 1
	r.U64() // requires 8, fails
	r.U64() // should be no-op (already errored)
	r.U32() // same
	if r.Err() == nil {
		t.Fatal("expected sticky error")
	}
}

func TestReader_ShortBuffer_ZeroValuesAfterErr(t *testing.T) {
	data := []byte{0x01}
	r := NewReader(data)
	r.U64() // error
	if r.U32() != 0 {
		t.Error("after error, reads must return zero")
	}
	if r.Bool() {
		t.Error("after error, Bool must return false")
	}
	var out [4]byte
	r.Read(out[:])
	if out != [4]byte{} {
		t.Errorf("after error, Read must leave dst untouched: %x", out)
	}
}

func TestReader_Done_TrailingBytes(t *testing.T) {
	data := []byte{0x01, 0xFF, 0xFF}
	r := NewReader(data)
	r.U8()
	if err := r.Done(); err == nil {
		t.Error("Done must report trailing bytes")
	}
}

func TestReader_Done_OK(t *testing.T) {
	data := []byte{0x01}
	r := NewReader(data)
	r.U8()
	if err := r.Done(); err != nil {
		t.Errorf("Done: %v", err)
	}
}

func TestReader_U128(t *testing.T) {
	var v U128
	v.SetUint64(0xDEADBEEF)
	data := New().U128c(v).Bytes()
	r := NewReader(data)
	got := r.U128()
	if got != v {
		t.Errorf("U128 mismatch: %v vs %v", got, v)
	}
}

func TestReader_Skip(t *testing.T) {
	data := New().U64(1).U64(2).U64(3).Bytes()
	r := NewReader(data)
	r.Skip(8) // skip first u64
	if r.U64() != 2 {
		t.Error("Skip didn't advance")
	}
}

func TestReader_FromDecoder(t *testing.T) {
	data := New().U8(0xAA).U64(42).Bytes()
	d := NewDecoder(data)
	if _, err := d.ReadUint8(); err != nil {
		t.Fatal(err)
	}
	r := FromDecoder(d)
	if r.U64() != 42 {
		t.Error("FromDecoder didn't share position")
	}
}
