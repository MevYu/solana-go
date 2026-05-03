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
// It is exported so callers can decode directly with Client.Call
// when they have a reason to own the envelope, but the common path
// is to use CallContextValue, which hides the envelope entirely.
type ContextValue[T any] struct {
	Context struct {
		Slot uint64 `json:"slot"`
	} `json:"context"`
	Value T `json:"value"`
}

// contextEnvelope is the full JSON-RPC response shape for methods
// that return the {context, value} wrapper. Using a generic typed
// result field lets the codec decode the entire response—including
// the context slot and typed value—in a single Unmarshal pass,
// eliminating the second pass that Call's two-step Response+RawJSON
// approach requires.
type contextEnvelope[T any] struct {
	Result ContextValue[T] `json:"result"`
	Error  *Error          `json:"error"`
}

// CallContextValue issues a JSON-RPC call that is known to return
// the {context:{slot}, value:T} envelope and returns the decoded
// slot and value.
//
// Unlike CallContext, which decodes the full response envelope in
// two passes (first into Response with a RawJSON result field,
// then into the caller's type), CallContextValue decodes the
// entire wire response—including the context slot and typed
// value—in a single Unmarshal call. This halves codec work per
// call for all context-wrapped methods (getAccountInfo, getBalance,
// …).
//
// args is passed variadically. The zero value of T is returned on error.
func CallContextValue[T any](ctx context.Context, c *Client, method string, args ...any) (slot uint64, value T, err error) {
	body, err := c.callRaw(ctx, method, args)
	if err != nil {
		return 0, value, err
	}

	var resp contextEnvelope[T]
	if err = c.codec.Unmarshal(body, &resp); err != nil {
		return 0, value, fmt.Errorf("solana rpc %s: decode response: %w", method, err)
	}
	if resp.Error != nil {
		return 0, value, &ErrRPC{
			Method: method,
			Code:   resp.Error.Code,
			Msg:    resp.Error.Message,
			Data:   resp.Error.Data,
			Body:   body,
		}
	}
	return resp.Result.Context.Slot, resp.Result.Value, nil
}
