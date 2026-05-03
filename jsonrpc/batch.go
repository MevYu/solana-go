package jsonrpc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
)

// ErrMissingResponse is the per-element Error populated when the
// server's batch response omitted the element's ID. It indicates a
// broken or misbehaving RPC endpoint — the JSON-RPC 2.0 spec
// requires one response per request ID (except for notifications,
// which this client does not issue).
var ErrMissingResponse = errors.New("no response for batch element")

// BatchElem is one call inside a BatchCallContext invocation.
//
//   - Method and Args describe the request (Args may be nil for no-arg
//     methods; the client will serialise it as []).
//   - Result is a pointer into which the decoded result is written;
//     set to nil to discard. For context-wrapped methods, point it at
//     a *ContextValue[T] to receive the full envelope.
//   - Error is populated by BatchCallContext on per-element failure:
//     an *ErrRPC for JSON-RPC error objects, a plain error for result
//     decode failures, or ErrMissingResponse when the server omitted
//     the response. Zero on success.
type BatchElem struct {
	Method string
	Args   []any
	Result any
	Error  error
}

// BatchCallContext sends b as a single JSON-RPC 2.0 batch request.
// The returned error is non-nil only on transport-level failures
// (HTTP failure, malformed response array, context cancellation).
// Per-element errors — including JSON-RPC error objects, result
// decode failures, and missing responses — are written into each
// BatchElem.Error so the caller can observe partial success.
//
// Each element is tagged with a fresh monotonic ID; the server's
// responses are matched back to the input by ID (JSON-RPC does not
// guarantee response order). The entire batch is retried as a unit
// under the configured retry policy; Solana's RPC server is
// stateless so reusing IDs on retry is safe.
//
// An empty b is a no-op and returns nil without issuing a request.
func (c *Client) BatchCallContext(ctx context.Context, b []BatchElem) error {
	if len(b) == 0 {
		return nil
	}

	reqs := make([]Request, len(b))
	idToIdx := make(map[uint64]int, len(b))
	for i := range b {
		id := c.nextID.Add(1)
		args := b[i].Args
		if args == nil {
			args = []any{}
		}
		reqs[i] = Request{
			Version: jsonrpcVersion,
			ID:      id,
			Method:  b[i].Method,
			Params:  args,
		}
		idToIdx[id] = i
	}

	label := batchLabel(b)
	body, err := c.sendRaw(ctx, label, func() ([]byte, error) {
		return c.codec.Marshal(reqs)
	})
	if err != nil {
		return err
	}

	// Response is the full envelope; for batches it decodes per-element.
	// Reusing the type keeps the JSON-RPC wire shape in one place.
	var resps []Response
	if err := c.codec.Unmarshal(body, &resps); err != nil {
		return fmt.Errorf("solana rpc %s: decode response array: %w", label, err)
	}

	seen := make([]bool, len(b))
	for i := range resps {
		idx, ok := idToIdx[resps[i].ID]
		if !ok {
			continue
		}
		seen[idx] = true
		elem := &b[idx]
		if resps[i].Error != nil {
			elem.Error = &ErrRPC{
				Method: elem.Method,
				Code:   resps[i].Error.Code,
				Msg:    resps[i].Error.Message,
				Data:   resps[i].Error.Data,
			}
			continue
		}
		if elem.Result != nil && len(resps[i].Result) > 0 {
			if err := c.codec.Unmarshal(resps[i].Result, elem.Result); err != nil {
				elem.Error = fmt.Errorf("solana rpc %s: decode result: %w", elem.Method, err)
			}
		}
	}
	for i := range seen {
		if !seen[i] {
			b[i].Error = ErrMissingResponse
		}
	}
	return nil
}

// batchLabel returns a short human-readable tag for a batch used in
// transport-level error messages. Includes the batch size and up to
// the first three method names.
func batchLabel(b []BatchElem) string {
	const maxMethods = 3
	var sb bytes.Buffer
	fmt.Fprintf(&sb, "batch[%d]:", len(b))
	n := len(b)
	if n > maxMethods {
		n = maxMethods
	}
	for i := 0; i < n; i++ {
		if i == 0 {
			sb.WriteByte(' ')
		} else {
			sb.WriteByte(',')
		}
		sb.WriteString(b[i].Method)
	}
	if len(b) > maxMethods {
		sb.WriteString(",...")
	}
	return sb.String()
}
