package solana

import (
	"crypto/ed25519"
	"database/sql/driver"
	"fmt"

	"github.com/MevYu/solana-go/encoding"
	"github.com/mr-tron/base58"
)

// Size constants for the core Solana primitives, in bytes.
const (
	// PublicKeySize is the byte length of an Ed25519 public key used to
	// identify accounts, programs, and program-derived addresses on
	// Solana.
	PublicKeySize = ed25519.PublicKeySize
	// HashSize is the byte length of a Solana blockhash or transaction
	// hash.
	HashSize = 32
	// SignatureSize is the byte length of an Ed25519 signature produced
	// by a Solana keypair.
	SignatureSize = ed25519.SignatureSize
)

// PublicKey is a 32-byte Solana account or program address. Its
// canonical textual form is base58; JSON, text and SQL marshalling all
// use that form.
type PublicKey [PublicKeySize]byte

// Hash is a 32-byte Solana blockhash or transaction hash. It uses the
// same base58 textual form as PublicKey.
type Hash [HashSize]byte

// Signature is a 64-byte Ed25519 signature produced by a Solana
// keypair. It uses the same base58 textual form as PublicKey.
type Signature [SignatureSize]byte

// ---------------------------------------------------------------------
// Constructors from raw bytes
// ---------------------------------------------------------------------

// PublicKeyFromBytes constructs a PublicKey from a 32-byte slice. It
// returns an error wrapping ErrInvalidLength when the input does not
// match PublicKeySize; it never silently truncates or pads.
func PublicKeyFromBytes(b []byte) (PublicKey, error) {
	var pk PublicKey
	if len(b) != PublicKeySize {
		return pk, fmt.Errorf("solana: public key: %w: got %d, want %d", ErrInvalidLength, len(b), PublicKeySize)
	}
	copy(pk[:], b)
	return pk, nil
}

// HashFromBytes constructs a Hash from a 32-byte slice. See
// PublicKeyFromBytes for the error contract.
func HashFromBytes(b []byte) (Hash, error) {
	var h Hash
	if len(b) != HashSize {
		return h, fmt.Errorf("solana: hash: %w: got %d, want %d", ErrInvalidLength, len(b), HashSize)
	}
	copy(h[:], b)
	return h, nil
}

// SignatureFromBytes constructs a Signature from a 64-byte slice. See
// PublicKeyFromBytes for the error contract.
func SignatureFromBytes(b []byte) (Signature, error) {
	var s Signature
	if len(b) != SignatureSize {
		return s, fmt.Errorf("solana: signature: %w: got %d, want %d", ErrInvalidLength, len(b), SignatureSize)
	}
	copy(s[:], b)
	return s, nil
}

// ---------------------------------------------------------------------
// Constructors from base58
// ---------------------------------------------------------------------

// decodeSizedBase58 decodes s as base58 and enforces an exact decoded
// length. label ("public key", "hash", ...) is used in error messages
// so that callers see which primitive the failure is about.
func decodeSizedBase58(s string, size int, label string) ([]byte, error) {
	if s == "" {
		return nil, fmt.Errorf("solana: %s: %w: empty string", label, ErrInvalidBase58)
	}
	b, err := base58.Decode(s)
	if err != nil {
		return nil, fmt.Errorf("solana: %s: %w: %v", label, ErrInvalidBase58, err)
	}
	if len(b) != size {
		return nil, fmt.Errorf("solana: %s: %w: decoded %d bytes, want %d", label, ErrInvalidLength, len(b), size)
	}
	return b, nil
}

// PublicKeyFromBase58 decodes a Solana address from its base58 form.
func PublicKeyFromBase58(s string) (PublicKey, error) {
	var pk PublicKey
	b, err := decodeSizedBase58(s, PublicKeySize, "public key")
	if err != nil {
		return pk, err
	}
	copy(pk[:], b)
	return pk, nil
}

// HashFromBase58 decodes a Solana blockhash or transaction hash from
// its base58 form.
func HashFromBase58(s string) (Hash, error) {
	var h Hash
	b, err := decodeSizedBase58(s, HashSize, "hash")
	if err != nil {
		return h, err
	}
	copy(h[:], b)
	return h, nil
}

// SignatureFromBase58 decodes an Ed25519 signature from its base58
// form.
func SignatureFromBase58(s string) (Signature, error) {
	var sig Signature
	b, err := decodeSizedBase58(s, SignatureSize, "signature")
	if err != nil {
		return sig, err
	}
	copy(sig[:], b)
	return sig, nil
}

// MustPublicKey is like PublicKeyFromBase58 but panics on invalid
// input. It is intended for package-level variable initializers that
// hardcode well-known program IDs, where a parse failure is a
// programmer error caught at startup rather than a runtime condition.
//
// Library code that receives user input must NOT use this function;
// use PublicKeyFromBase58 and propagate the error.
func MustPublicKey(s string) PublicKey {
	pk, err := PublicKeyFromBase58(s)
	if err != nil {
		panic(err)
	}
	return pk
}

// MustHash is like HashFromBase58 but panics on invalid input.
// Same contract as MustPublicKey.
func MustHash(s string) Hash {
	h, err := HashFromBase58(s)
	if err != nil {
		panic(err)
	}
	return h
}

// MustSignature is like SignatureFromBase58 but panics on invalid
// input. Same contract as MustPublicKey.
func MustSignature(s string) Signature {
	sig, err := SignatureFromBase58(s)
	if err != nil {
		panic(err)
	}
	return sig
}

// ---------------------------------------------------------------------
// Raw byte accessors
// ---------------------------------------------------------------------

// Bytes returns a defensive copy of the key's raw bytes. Mutating the
// returned slice does not affect the receiver.
func (p PublicKey) Bytes() []byte {
	out := make([]byte, PublicKeySize)
	copy(out, p[:])
	return out
}

// Equal reports whether p and other represent the same public key.
func (p PublicKey) Equal(other PublicKey) bool { return p == other }

// IsZero reports whether the key equals the all-zero PublicKey.
func (p PublicKey) IsZero() bool { return p == PublicKey{} }

// Bytes returns a defensive copy of the hash's raw bytes.
func (h Hash) Bytes() []byte {
	out := make([]byte, HashSize)
	copy(out, h[:])
	return out
}

// Equal reports whether h and other represent the same hash.
func (h Hash) Equal(other Hash) bool { return h == other }

// IsZero reports whether the hash equals the all-zero Hash.
func (h Hash) IsZero() bool { return h == Hash{} }

// Bytes returns a defensive copy of the signature's raw bytes.
func (s Signature) Bytes() []byte {
	out := make([]byte, SignatureSize)
	copy(out, s[:])
	return out
}

// Equal reports whether s and other represent the same signature.
func (s Signature) Equal(other Signature) bool { return s == other }

// IsZero reports whether the signature equals the all-zero Signature.
func (s Signature) IsZero() bool { return s == Signature{} }

// ---------------------------------------------------------------------
// String / fmt.Stringer
// ---------------------------------------------------------------------

// String returns the base58 representation of the public key, matching
// the canonical Solana address form.
func (p PublicKey) String() string { return base58.Encode(p[:]) }

// String returns the base58 representation of the hash.
func (h Hash) String() string { return base58.Encode(h[:]) }

// String returns the base58 representation of the signature.
func (s Signature) String() string { return base58.Encode(s[:]) }

// ---------------------------------------------------------------------
// JSON marshalling
// ---------------------------------------------------------------------

// jsonQuote wraps s in JSON double-quotes. The base58 alphabet contains
// no characters requiring JSON escaping, so a naive wrap is safe and
// avoids pulling encoding/json into the library hot path (see the
// project's architectural principles).
func jsonQuote(s string) []byte {
	out := make([]byte, 0, len(s)+2)
	out = append(out, '"')
	out = append(out, s...)
	out = append(out, '"')
	return out
}

// isJSONNull reports whether data is the exact JSON token "null".
// encoding/json strips surrounding whitespace before invoking
// UnmarshalJSON, so an exact compare is sufficient.
func isJSONNull(data []byte) bool {
	return len(data) == 4 && data[0] == 'n' && data[1] == 'u' && data[2] == 'l' && data[3] == 'l'
}

// unquoteJSONString strips the surrounding double-quotes of a JSON
// string literal. It does not interpret escape sequences because the
// base58 alphabet cannot contain any.
func unquoteJSONString(data []byte) (string, error) {
	if len(data) < 2 || data[0] != '"' || data[len(data)-1] != '"' {
		return "", fmt.Errorf("expected JSON string, got %q", data)
	}
	return string(data[1 : len(data)-1]), nil
}

// MarshalJSON implements json.Marshaler. The key is rendered as a JSON
// string holding its base58 form.
func (p PublicKey) MarshalJSON() ([]byte, error) { return jsonQuote(p.String()), nil }

// MarshalJSON implements json.Marshaler.
func (h Hash) MarshalJSON() ([]byte, error) { return jsonQuote(h.String()), nil }

// MarshalJSON implements json.Marshaler.
func (s Signature) MarshalJSON() ([]byte, error) { return jsonQuote(s.String()), nil }

// UnmarshalJSON implements json.Unmarshaler. The value must be a JSON
// string holding a base58-encoded 32-byte public key. The JSON token
// "null" is accepted and leaves the receiver unchanged.
func (p *PublicKey) UnmarshalJSON(data []byte) error {
	if isJSONNull(data) {
		return nil
	}
	s, err := unquoteJSONString(data)
	if err != nil {
		return fmt.Errorf("solana: public key: %w: %v", ErrInvalidBase58, err)
	}
	pk, err := PublicKeyFromBase58(s)
	if err != nil {
		return err
	}
	*p = pk
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (h *Hash) UnmarshalJSON(data []byte) error {
	if isJSONNull(data) {
		return nil
	}
	s, err := unquoteJSONString(data)
	if err != nil {
		return fmt.Errorf("solana: hash: %w: %v", ErrInvalidBase58, err)
	}
	out, err := HashFromBase58(s)
	if err != nil {
		return err
	}
	*h = out
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (s *Signature) UnmarshalJSON(data []byte) error {
	if isJSONNull(data) {
		return nil
	}
	str, err := unquoteJSONString(data)
	if err != nil {
		return fmt.Errorf("solana: signature: %w: %v", ErrInvalidBase58, err)
	}
	sig, err := SignatureFromBase58(str)
	if err != nil {
		return err
	}
	*s = sig
	return nil
}

// ---------------------------------------------------------------------
// Text marshalling (encoding.TextMarshaler / encoding.TextUnmarshaler)
// ---------------------------------------------------------------------

// MarshalText implements encoding.TextMarshaler by returning the base58
// form.
func (p PublicKey) MarshalText() ([]byte, error) { return []byte(p.String()), nil }

// MarshalText implements encoding.TextMarshaler.
func (h Hash) MarshalText() ([]byte, error) { return []byte(h.String()), nil }

// MarshalText implements encoding.TextMarshaler.
func (s Signature) MarshalText() ([]byte, error) { return []byte(s.String()), nil }

// UnmarshalText implements encoding.TextUnmarshaler.
func (p *PublicKey) UnmarshalText(text []byte) error {
	pk, err := PublicKeyFromBase58(string(text))
	if err != nil {
		return err
	}
	*p = pk
	return nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (h *Hash) UnmarshalText(text []byte) error {
	out, err := HashFromBase58(string(text))
	if err != nil {
		return err
	}
	*h = out
	return nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (s *Signature) UnmarshalText(text []byte) error {
	sig, err := SignatureFromBase58(string(text))
	if err != nil {
		return err
	}
	*s = sig
	return nil
}

// ---------------------------------------------------------------------
// database/sql integration
// ---------------------------------------------------------------------

// scanBase58String normalises src into a base58 string. It rejects
// unsupported source types with a descriptive error. Callers handle
// the nil case before invoking this helper.
func scanBase58String(src any, label string) (string, error) {
	switch v := src.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	default:
		return "", fmt.Errorf("solana: %s: cannot Scan %T into base58 string", label, src)
	}
}

// Value implements driver.Valuer by returning the base58 string form,
// which pairs with the string returned by String().
func (p PublicKey) Value() (driver.Value, error) { return p.String(), nil }

// Value implements driver.Valuer.
func (h Hash) Value() (driver.Value, error) { return h.String(), nil }

// Value implements driver.Valuer.
func (s Signature) Value() (driver.Value, error) { return s.String(), nil }

// Scan implements sql.Scanner. It accepts a string or a []byte holding
// the base58 form. A nil src leaves the receiver unchanged. Raw-byte
// columns must be converted by the caller via PublicKeyFromBytes.
func (p *PublicKey) Scan(src any) error {
	if src == nil {
		return nil
	}
	s, err := scanBase58String(src, "public key")
	if err != nil {
		return err
	}
	pk, err := PublicKeyFromBase58(s)
	if err != nil {
		return err
	}
	*p = pk
	return nil
}

// Scan implements sql.Scanner. See PublicKey.Scan for the contract.
func (h *Hash) Scan(src any) error {
	if src == nil {
		return nil
	}
	s, err := scanBase58String(src, "hash")
	if err != nil {
		return err
	}
	out, err := HashFromBase58(s)
	if err != nil {
		return err
	}
	*h = out
	return nil
}

// Scan implements sql.Scanner. See PublicKey.Scan for the contract.
func (s *Signature) Scan(src any) error {
	if src == nil {
		return nil
	}
	str, err := scanBase58String(src, "signature")
	if err != nil {
		return err
	}
	sig, err := SignatureFromBase58(str)
	if err != nil {
		return err
	}
	*s = sig
	return nil
}

// ReadOptionalPubkey reads a Solana COption<Pubkey>-shaped field from
// r: a u32 tag (0=None, 1=Some) followed by an unconditional 32-byte
// slot (the slot is always physically present in the wire layout —
// only the tag determines logical presence). Returns a non-nil
// *PublicKey when the tag is 1, nil when it is 0.
//
// This is the common shape used throughout SPL Token state for
// MintAuthority, FreezeAuthority, Delegate, and CloseAuthority. It is
// not the same as Rust's bincode Option<Pubkey> (which uses a 1-byte
// tag and a payload that is only present when Some); for that, read a
// Bool and a Bytes32 separately.
func ReadOptionalPubkey(r *encoding.Reader) *PublicKey {
	tag := r.U32()
	pk := PublicKey(r.Bytes32())
	if tag == 1 {
		return &pk
	}
	return nil
}
