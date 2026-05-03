package solana

import (
	"fmt"
	"strconv"
)

// Uint8Slice is a slice of bytes that JSON-marshals as an array of
// numbers instead of a base64 string.
//
// Go's encoding/json has a special case that renders any slice whose
// element kind is uint8 (including named types like []byte) as a
// base64-encoded string. That is not the shape the Solana JSON-RPC
// uses for fields like CompiledInstruction.Accounts or
// MessageAddressTableLookup.WritableIndexes, which are transmitted as
// compact JSON arrays of small integers. Uint8Slice bypasses the
// special case by implementing json.Marshaler and json.Unmarshaler
// itself.
//
// The UnmarshalJSON method is a self-contained byte-level parser and
// does not recursively call encoding/json, keeping the decoding hot
// path free of reflection and of the stdlib JSON library.
type Uint8Slice []uint8

// MarshalJSON implements json.Marshaler by rendering the slice as a
// JSON array of decimal integers. A nil receiver marshals to "null";
// an empty non-nil slice marshals to "[]".
func (s Uint8Slice) MarshalJSON() ([]byte, error) {
	if s == nil {
		return []byte("null"), nil
	}
	if len(s) == 0 {
		return []byte("[]"), nil
	}
	// Upper bound per element: 3 decimal digits + 1 comma.
	buf := make([]byte, 0, 2+4*len(s))
	buf = append(buf, '[')
	for i, v := range s {
		if i > 0 {
			buf = append(buf, ',')
		}
		buf = strconv.AppendUint(buf, uint64(v), 10)
	}
	buf = append(buf, ']')
	return buf, nil
}

// UnmarshalJSON implements json.Unmarshaler. It accepts either the
// JSON token "null" (leaving the receiver nil) or a JSON array whose
// elements are decimal integers in the range 0..255 inclusive.
// Whitespace between tokens is tolerated.
func (s *Uint8Slice) UnmarshalJSON(data []byte) error {
	if isJSONNull(data) {
		*s = nil
		return nil
	}
	i := skipJSONWhitespace(data, 0)
	if i >= len(data) || data[i] != '[' {
		return fmt.Errorf("solana: Uint8Slice: expected JSON array, got %q", data)
	}
	i++
	out := Uint8Slice{}
	for {
		i = skipJSONWhitespace(data, i)
		if i >= len(data) {
			return fmt.Errorf("solana: Uint8Slice: unterminated array")
		}
		if data[i] == ']' {
			i++
			break
		}
		start := i
		for i < len(data) && data[i] >= '0' && data[i] <= '9' {
			i++
		}
		if i == start {
			return fmt.Errorf("solana: Uint8Slice: expected a decimal integer at byte %d", start)
		}
		n, err := strconv.ParseUint(string(data[start:i]), 10, 8)
		if err != nil {
			return fmt.Errorf("solana: Uint8Slice: %w", err)
		}
		out = append(out, uint8(n))
		i = skipJSONWhitespace(data, i)
		if i < len(data) && data[i] == ',' {
			i++
			continue
		}
		if i < len(data) && data[i] == ']' {
			i++
			break
		}
		return fmt.Errorf("solana: Uint8Slice: expected ',' or ']' at byte %d", i)
	}
	for i < len(data) {
		c := data[i]
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			return fmt.Errorf("solana: Uint8Slice: trailing garbage after array")
		}
		i++
	}
	*s = out
	return nil
}

func skipJSONWhitespace(data []byte, i int) int {
	for i < len(data) {
		c := data[i]
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			return i
		}
		i++
	}
	return i
}
