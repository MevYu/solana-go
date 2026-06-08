// Package encoding implements Solana's binary wire formats.
//
// Three formats live here:
//
//   - bincode: Rust bincode-crate output. Vec / String use u64 LE length
//     prefixes. Used by Solana built-ins (System nonce, Vote, Stake).
//   - Borsh: Vec / String use u32 LE length prefixes. Used by Anchor
//     programs, SPL Token-2022 extension state, Token Metadata, Address
//     Lookup Table state, and most modern third-party programs.
//   - shortvec: compact-u16 length prefix. Used inside transaction and
//     message bodies (signature count, account-keys count, etc.).
//
// Three layers of API, by use case:
//
//   - One-shot reflection: BinDecodeTo / BorshDecodeTo. Pick by program;
//     see those godocs for the full picker. Cheapest to write, slowest to
//     run. The two formats only differ in slice / string length prefix
//     width — structs without slice or string fields decode the same with
//     either function.
//   - Hand-written hot-path decoding: NewReader + r.U64() / r.Bytes32().
//     Canonical path for performance-sensitive code and for
//     wire layouts that reflection can't express (TLV, COption<Pubkey>,
//     OptionalNonZeroPubkey, custom enum tags).
//   - Hand-written instruction-data builders: NewEncoder + chained
//     New().U8(tag).U64(amount).Bytes(). The convention every program in
//     programs/* follows.
//
// The Decoder type underpins the reflective entry points and the Reader
// API; it never copies the input buffer.
package encoding
