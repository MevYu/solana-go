package jsonrpc

import (
	gojson "github.com/goccy/go-json"
)

// GoJSONCodec returns a Codec backed by github.com/goccy/go-json,
// the fastest general-purpose JSON library in the Go ecosystem.
//
// It is API-compatible with stdlib encoding/json and honours the
// json.Marshaler and json.Unmarshaler interfaces, so every type in
// this module that implements either (PublicKey, Hash, Signature,
// Uint8Slice, RawJSON, ...) round-trips correctly through it
// without any additional plumbing.
//
// GoJSONCodec is the default Codec used by NewClient. StdCodec is
// kept available for environments where a hard dependency on
// goccy/go-json is not acceptable; pass WithCodec(StdCodec()) to
// opt out.
func GoJSONCodec() Codec { return gojsonCodec{} }

type gojsonCodec struct{}

func (gojsonCodec) Marshal(v any) ([]byte, error)      { return gojson.Marshal(v) }
func (gojsonCodec) Unmarshal(data []byte, v any) error { return gojson.Unmarshal(data, v) }
