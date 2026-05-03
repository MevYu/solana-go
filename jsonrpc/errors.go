package jsonrpc

import "strconv"

// ErrRPC is the error type returned when the Solana RPC server
// responded with a JSON-RPC 2.0 error object. It carries the method
// name, the numeric code, the human-readable message, any
// server-attached Data payload, and the raw response body for
// diagnostics.
//
// Use errors.As to recover *ErrRPC from a wrapped error:
//
//	var rpcErr *rpc.ErrRPC
//	if errors.As(err, &rpcErr) {
//		// inspect rpcErr.Code, rpcErr.Data, etc.
//	}
type ErrRPC struct {
	Method string
	Code   int
	Msg    string
	Data   RawJSON
	Body   []byte
}

func (e *ErrRPC) Error() string {
	return "solana rpc " + e.Method + ": rpc error " + strconv.Itoa(e.Code) + ": " + e.Msg
}

// httpError is an internal transport-level error describing a
// non-2xx HTTP response. It is not exposed to callers: the retry
// policy uses it to decide whether to retry, and the client wraps
// it in a more descriptive error before returning to the caller.
type httpError struct {
	StatusCode int
	Body       []byte
}

func (e *httpError) Error() string {
	return "http " + strconv.Itoa(e.StatusCode) + ": " + truncate(e.Body, 256)
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}
