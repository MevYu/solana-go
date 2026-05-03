package jsonrpc

import (
	"context"
	"fmt"
)

// ContextValue is the generic shape every Solana RPC method that
// returns a context-wrapped value takes on the wire:
//
//	{"context": {"slot": <slot>}, "value": <T>}
//
// To decode this envelope, instantiate CallContext with
// ContextValue[T] as the type argument:
//
//	resp, err := jsonrpc.CallContext[jsonrpc.ContextValue[*AccountInfo]](
//	    ctx, c, "getAccountInfo", addr, cfg)
//	// resp.Context.Slot, resp.Value
//
// The type is also reused by ws.Client notification dispatchers,
// which decode the same envelope shape from raw JSON bytes.
type ContextValue[T any] struct {
	Context struct {
		Slot uint64 `json:"slot"`
	} `json:"context"`
	Value T `json:"value"`
}

// responseEnvelope is the JSON-RPC 2.0 response envelope parameterised
// over the typed result. Using a generic Result field lets the codec
// decode the whole response in a single Unmarshal pass with the
// concrete type known at compile time, so no interface dispatch and
// no per-instance reflection are needed beyond what the codec already
// caches for T.
type responseEnvelope[T any] struct {
	Result T      `json:"result"`
	Error  *Error `json:"error"`
}

// CallContext issues a JSON-RPC 2.0 request and returns the decoded
// typed result in a single Unmarshal pass.
//
// CallContext is a free generic function (Go does not allow type
// parameters on methods) that takes the *Client explicitly:
//
//	balance, err := jsonrpc.CallContext[uint64](ctx, c, "getBalance", addr)
//
// CallContext uses the configured RetryPolicy to transparently retry
// transient failures. The caller's context controls deadlines and
// cancellation; a cancelled or expired context returns immediately
// with the context error wrapped in a method-labelled message.
//
// On a JSON-RPC error response, CallContext returns *ErrRPC wrapping
// the code, message, data and raw body. Use errors.As to recover it.
// The zero value of T is returned on any error.
//
// For methods that return Solana's {context:{slot}, value} envelope,
// instantiate T as ContextValue[X]; the slot is then on
// result.Context.Slot and the payload on result.Value.
func CallContext[T any](ctx context.Context, c *Client, method string, args ...any) (T, error) {
	var zero T
	body, err := c.callRaw(ctx, method, args)
	if err != nil {
		return zero, err
	}

	var resp responseEnvelope[T]
	if err = c.codec.Unmarshal(body, &resp); err != nil {
		return zero, fmt.Errorf("solana rpc %s: decode response: %w", method, err)
	}
	if resp.Error != nil {
		return zero, &ErrRPC{
			Method: method,
			Code:   resp.Error.Code,
			Msg:    resp.Error.Message,
			Data:   resp.Error.Data,
			Body:   body,
		}
	}
	return resp.Result, nil
}
