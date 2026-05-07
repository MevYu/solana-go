package encoding

// BorshDecodeTo decodes data into v using Borsh wire conventions.
//
// What's different from bincode (BinDecodeTo) — exactly two field shapes:
//
//   - []T / Vec<T> length: 4-byte u32 LE prefix      (bincode: 8-byte u64)
//   - string / String length: 4-byte u32 LE prefix   (bincode: 8-byte u64)
//
// Everything else (fixed [N]byte arrays, all primitives, *T / Option<T> as
// 1-byte tag + optional payload, structs as concatenated fields) is byte-
// for-byte identical between the two — so structs with no slice or string
// fields decode the same either way.
//
// Pick BorshDecodeTo for accounts produced by:
//
//   - Anchor programs (#[derive(BorshSerialize, BorshDeserialize)])
//   - SPL Token-2022 extension state
//   - Token Metadata
//   - Address Lookup Table state
//   - most modern third-party programs
//
// If unsure, check the Rust source: BorshSerialize/BorshDeserialize derives
// mean Borsh; the bincode crate or solana_program::serialization means
// BinDecodeTo.
func BorshDecodeTo(data []byte, v any) error {
	return NewDecoder(data).UseBorsh().DecodeTo(v)
}

// UseBorsh switches the Decoder's reflective length prefix from bincode u64
// to Borsh u32 for slice and string fields. Returns the receiver for chaining.
func (d *Decoder) UseBorsh() *Decoder { d.borsh = true; return d }
