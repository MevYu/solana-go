package encoding

import (
	"bytes"
	"errors"
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

func TestDecodeTo_ByteSlice_PlainAndNamed(t *testing.T) {
	// Bincode u64 length prefix + payload, decoded into both a plain []byte
	// field (reflect-free fast path) and a named-type field (reflect path).
	payload := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	data := New().U64(uint64(len(payload))).Raw(payload).Bytes()

	type Named []byte
	type S struct {
		Plain []byte
		Named Named
	}
	// Two back-to-back fields: prefix+payload twice.
	data2 := New().
		U64(uint64(len(payload))).Raw(payload).
		U64(uint64(len(payload))).Raw(payload).
		Bytes()

	var s S
	if err := BinDecodeTo(data2, &s); err != nil {
		t.Fatalf("BinDecodeTo: %v", err)
	}
	if !bytes.Equal(s.Plain, payload) {
		t.Errorf("Plain = %x, want %x", s.Plain, payload)
	}
	if !bytes.Equal(s.Named, payload) {
		t.Errorf("Named = %x, want %x", s.Named, payload)
	}
	// Decoded slice must not alias the input buffer.
	if len(s.Plain) > 0 && &s.Plain[0] == &data2[8] {
		t.Error("Plain aliases input buffer; must be a copy")
	}
	_ = data
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

// ######## slice length bound (DoS) ########

// A slice length prefix that exceeds the remaining buffer must error
// rather than make the decoder allocate billions of elements (the
// opSlice MakeSlice-before-read DoS).
func TestDecodeTo_SliceLengthExceedingBufferErrors(t *testing.T) {
	type S struct{ Items []uint32 }

	// Borsh: u32 length 0xFFFFFFFF, no element data follows.
	if err := BorshDecodeTo(New().U32(0xFFFFFFFF).Bytes(), &S{}); err == nil {
		t.Error("borsh: expected error for oversized slice length, got nil")
	}
	// Bincode: u64 length 0xFFFFFFFFFFFFFFFF, no element data follows.
	if err := BinDecodeTo(New().U64(0xFFFFFFFFFFFFFFFF).Bytes(), &S{}); err == nil {
		t.Error("bincode: expected error for oversized slice length, got nil")
	}
}

// ######## shortvec canonical encoding ########

// Solana's consensus shortvec decoder enforces minimal (canonical)
// encoding. DecodeShortvec must reject overlong forms a validator rejects.
func TestDecodeShortvec_RejectsNonCanonical(t *testing.T) {
	overlong := [][]byte{
		{0x80, 0x00},       // value 0 in 2 bytes (canonical: {0x00})
		{0x81, 0x00},       // value 1 in 2 bytes (canonical: {0x01})
		{0x80, 0x80, 0x00}, // value 0 in 3 bytes
		{0xff, 0xff, 0x00}, // 2-byte value in 3 bytes
	}
	for _, b := range overlong {
		if _, _, err := DecodeShortvec(b); !errors.Is(err, ErrInvalidShortvec) {
			t.Errorf("DecodeShortvec(%x) err = %v, want ErrInvalidShortvec", b, err)
		}
	}
}

func TestDecodeShortvec_AcceptsCanonical(t *testing.T) {
	cases := []struct {
		in   []byte
		want uint16
		n    int
	}{
		{[]byte{0x00}, 0, 1},
		{[]byte{0x7f}, 127, 1},
		{[]byte{0x80, 0x01}, 128, 2},
		{[]byte{0xff, 0x7f}, 16383, 2},
		{[]byte{0x80, 0x80, 0x01}, 16384, 3},
		{[]byte{0xff, 0xff, 0x03}, 65535, 3},
	}
	for _, c := range cases {
		v, n, err := DecodeShortvec(c.in)
		if err != nil || v != c.want || n != c.n {
			t.Errorf("DecodeShortvec(%x) = (%d,%d,%v), want (%d,%d,nil)", c.in, v, n, err, c.want, c.n)
		}
	}
}
