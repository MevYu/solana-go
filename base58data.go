package solana

import (
	"encoding/hex"
	"fmt"

	"github.com/mr-tron/base58"
)

// Base58Data is a byte slice that JSON-marshals to/from a base58
// string. It mirrors the go-solana type of the same name and is
// useful for instruction data fields on the wire.
type Base58Data []byte

// MarshalJSON implements json.Marshaler.
func (t Base58Data) MarshalJSON() ([]byte, error) {
	return jsonQuote(base58.Encode(t)), nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (t *Base58Data) UnmarshalJSON(data []byte) error {
	if isJSONNull(data) {
		*t = nil
		return nil
	}
	s, err := unquoteJSONString(data)
	if err != nil {
		return fmt.Errorf("solana: Base58Data: %w", err)
	}
	if s == "" {
		*t = []byte{}
		return nil
	}
	decoded, err := base58.Decode(s)
	if err != nil {
		return fmt.Errorf("solana: Base58Data: %w", err)
	}
	*t = decoded
	return nil
}

// String returns the base58 encoding.
func (t Base58Data) String() string {
	return base58.Encode(t)
}

// Base58 returns the base58 encoding.
func (t Base58Data) Base58() string {
	return base58.Encode(t)
}

// Hex returns the hex encoding.
func (t Base58Data) Hex() string {
	return hex.EncodeToString(t)
}
