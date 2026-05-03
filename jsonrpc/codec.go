package jsonrpc

// Codec encodes and decodes JSON for the RPC client. The default
// implementation is StdCodec, which uses stdlib encoding/json. For
// allocation-sensitive workloads, swap in a faster library (such as
// github.com/goccy/go-json) via the WithCodec client option.
//
// Implementations must be safe for concurrent use.
type Codec interface {
	Marshal(v any) ([]byte, error)
	Unmarshal(data []byte, v any) error
}
