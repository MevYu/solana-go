package solana

import (
	"unsafe"

	"github.com/MevYu/solana-go/encoding"
)

// Re-exports of the U128 / U256 primitive types so callers can write
// `solana.U128` / `solana.U256` in struct fields, matching the PublicKey /
// Hash / Signature discoverability. Both types keep all methods defined on
// their underlying encoding.* identities.
type (
	U128 = encoding.U128
	U256 = encoding.U256
)

// init wires hand-written decoders into the encoding registry for the core
// Solana primitive types. These registrations let encoding.Decoder.DecodeTo
// skip the reflective [N]byte array walk when it encounters a PublicKey,
// Hash, or Signature as a struct field or slice element — the decoder
// collapses to a single 32-byte (or 64-byte) memcpy per value.
//
// This is the first building block of the high-performance decode path:
// for any container type whose leaf fields are these primitives (plus u8 /
// u16 / u32 / u64 / U128 / U256), reflection walks only the *shape* of the
// struct, never the contents of a leaf. For streaming workloads — decoding
// Vec<Transaction>, Vec<Entry>, oracle snapshots, AMM state arrays — the
// register-backed dispatch eliminates per-byte reflection cost.
func init() {
	encoding.RegisterDecoder[PublicKey](func(d *encoding.Decoder, p *PublicKey) error {
		b, err := d.ReadBytes(PublicKeySize)
		if err != nil {
			return err
		}
		*p = *(*PublicKey)(unsafe.Pointer(&b[0]))
		return nil
	})
	encoding.RegisterDecoder[Hash](func(d *encoding.Decoder, p *Hash) error {
		b, err := d.ReadBytes(HashSize)
		if err != nil {
			return err
		}
		*p = *(*Hash)(unsafe.Pointer(&b[0]))
		return nil
	})
	encoding.RegisterDecoder[Signature](func(d *encoding.Decoder, p *Signature) error {
		b, err := d.ReadBytes(SignatureSize)
		if err != nil {
			return err
		}
		*p = *(*Signature)(unsafe.Pointer(&b[0]))
		return nil
	})
}
