package encoding

import "testing"

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
