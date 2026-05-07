package encoding

// Borsh shares Decoder with bincode; the only on-wire difference for
// reflection-driven decoding is that Vec<T> and String use a u32 length
// prefix instead of bincode's u64. UseBorsh flips that default; BorshDecodeTo
// is the one-shot helper.

// BorshDecodeTo decodes data into v using Borsh prefix conventions.
// It is the Borsh counterpart to BinDecodeTo.
func BorshDecodeTo(data []byte, v any) error {
	return NewDecoder(data).UseBorsh().DecodeTo(v)
}

// UseBorsh switches the Decoder's reflective length prefix from bincode u64
// to Borsh u32 for slice and string fields. Returns the receiver for chaining.
func (d *Decoder) UseBorsh() *Decoder { d.borsh = true; return d }
