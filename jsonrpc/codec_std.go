package jsonrpc

import (
	"encoding/json"
)

// StdCodec returns a Codec backed by stdlib encoding/json. It is
// kept available as an opt-in alternative to GoJSONCodec for
// environments where a hard dependency on github.com/goccy/go-json
// is not acceptable; pass it explicitly via WithCodec(StdCodec()).
//
// This file is the only place in the rpc package that imports
// encoding/json at the type level, so swapping out JSON libraries
// is a single-file change.
func StdCodec() Codec { return stdCodec{} }

type stdCodec struct{}

func (stdCodec) Marshal(v any) ([]byte, error)      { return json.Marshal(v) }
func (stdCodec) Unmarshal(data []byte, v any) error { return json.Unmarshal(data, v) }
