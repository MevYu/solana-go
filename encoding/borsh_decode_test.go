package encoding

import (
	"bytes"
	"testing"
)

func TestBorshDecodeTo_SliceUsesU32Prefix(t *testing.T) {
	// 3-element u32 slice: u32 length=3, then three u32 values.
	data := New().U32(3).U32(10).U32(20).U32(30).Bytes()

	type S struct{ Items []uint32 }
	var s S
	if err := BorshDecodeTo(data, &s); err != nil {
		t.Fatalf("BorshDecodeTo: %v", err)
	}
	if len(s.Items) != 3 || s.Items[0] != 10 || s.Items[1] != 20 || s.Items[2] != 30 {
		t.Errorf("Items = %v", s.Items)
	}
}

func TestBorshDecodeTo_StringUsesU32Prefix(t *testing.T) {
	data := New().U32(5).Raw([]byte("hello")).Bytes()

	type S struct{ Name string }
	var s S
	if err := BorshDecodeTo(data, &s); err != nil {
		t.Fatalf("BorshDecodeTo: %v", err)
	}
	if s.Name != "hello" {
		t.Errorf("Name = %q", s.Name)
	}
}

func TestBinDecodeTo_StillUsesU64Prefix(t *testing.T) {
	// Bincode mode: same payload but u64 length.
	data := New().U64(3).U32(10).U32(20).U32(30).Bytes()

	type S struct{ Items []uint32 }
	var s S
	if err := BinDecodeTo(data, &s); err != nil {
		t.Fatalf("BinDecodeTo: %v", err)
	}
	if len(s.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(s.Items))
	}
}

func TestBorshDecodeTo_OptionTagSameAsBincode(t *testing.T) {
	// Option<u32> in Borsh: 1-byte tag (0=None, 1=Some) + payload.
	// Same wire format as bincode, so DecodeTo and BorshDecodeTo agree.
	dataSome := New().U8(1).U32(42).Bytes()
	dataNone := []byte{0}

	type S struct{ Maybe *uint32 }
	var s1, s2 S
	if err := BorshDecodeTo(dataSome, &s1); err != nil {
		t.Fatal(err)
	}
	if s1.Maybe == nil || *s1.Maybe != 42 {
		t.Errorf("Some(42): got %v", s1.Maybe)
	}
	if err := BorshDecodeTo(dataNone, &s2); err != nil {
		t.Fatal(err)
	}
	if s2.Maybe != nil {
		t.Error("None should decode to nil")
	}
}

func TestBorshDecodeTo_FixedArrayUnchanged(t *testing.T) {
	// Fixed-size [N]byte arrays are not length-prefixed — same in both
	// Borsh and bincode.
	data := New().Raw(bytes.Repeat([]byte{0xAB}, 32)).Bytes()

	type Pubkey [32]byte
	type S struct{ K Pubkey }
	var s S
	if err := BorshDecodeTo(data, &s); err != nil {
		t.Fatalf("BorshDecodeTo: %v", err)
	}
	for i, b := range s.K {
		if b != 0xAB {
			t.Errorf("K[%d] = 0x%02x", i, b)
		}
	}
}

func TestUseBorsh_OnExistingDecoder(t *testing.T) {
	data := New().U32(2).U16(11).U16(22).Bytes()

	d := NewDecoder(data).UseBorsh()
	type S struct{ Items []uint16 }
	var s S
	if err := d.DecodeTo(&s); err != nil {
		t.Fatal(err)
	}
	if len(s.Items) != 2 || s.Items[0] != 11 || s.Items[1] != 22 {
		t.Errorf("Items = %v", s.Items)
	}
}
