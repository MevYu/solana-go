package jsonrpc

import "errors"

// RawJSON holds a fragment of JSON bytes without decoding them. It
// implements json.Marshaler and json.Unmarshaler so it round-trips
// through any JSON library that honours those interfaces (stdlib
// encoding/json, goccy/go-json, jsoniter, ...).
//
// RawJSON is behaviourally equivalent to encoding/json.RawMessage
// but is defined here so the rpc package does not need to import
// encoding/json at the type level.
type RawJSON []byte

// MarshalJSON implements json.Marshaler by returning the raw bytes.
// A zero-length RawJSON marshals to the JSON token "null" so that
// absent optional fields are rendered correctly.
func (r RawJSON) MarshalJSON() ([]byte, error) {
	if len(r) == 0 {
		return []byte("null"), nil
	}
	return []byte(r), nil
}

// UnmarshalJSON implements json.Unmarshaler by copying the input
// bytes. The copy is required: JSON decoders typically reuse a
// scratch buffer across tokens, so the argument bytes are not
// guaranteed to remain valid after the call returns.
func (r *RawJSON) UnmarshalJSON(data []byte) error {
	if r == nil {
		return errors.New("rpc: RawJSON: nil receiver")
	}
	*r = append((*r)[:0], data...)
	return nil
}

// String returns the underlying bytes as a Go string. It is intended
// for logging and debugging; do not parse the result back into JSON
// via string operations.
func (r RawJSON) String() string { return string(r) }
