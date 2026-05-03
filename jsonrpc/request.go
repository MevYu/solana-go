package jsonrpc

// jsonrpcVersion is the constant "2.0" embedded in every request and
// response envelope.
const jsonrpcVersion = "2.0"

// Request is a JSON-RPC 2.0 request. Version is always "2.0" for
// Solana RPC. Callers do not usually construct Request directly; use
// Client.Call instead.
type Request struct {
	Version string `json:"jsonrpc"`
	ID      uint64 `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

// Response is a JSON-RPC 2.0 response envelope. Exactly one of Result
// or Error is populated on a well-formed response. Result is kept as
// raw JSON so the client can decode it into an arbitrary typed value
// without this package needing to know its shape in advance.
type Response struct {
	Version string  `json:"jsonrpc"`
	ID      uint64  `json:"id"`
	Result  RawJSON `json:"result,omitempty"`
	Error   *Error  `json:"error,omitempty"`
}

// Error is a JSON-RPC 2.0 error object as defined by the JSON-RPC
// 2.0 specification. Solana's server populates Code and Message and
// may include a Data payload for method-specific context (for
// example, a decoded instruction error from simulateTransaction).
type Error struct {
	Code    int     `json:"code"`
	Message string  `json:"message"`
	Data    RawJSON `json:"data,omitempty"`
}
