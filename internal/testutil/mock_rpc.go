package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MevYu/solana-go/jsonrpc"
	"github.com/MevYu/solana-go/rpc"
)

// RPCHandler is a function that handles a single JSON-RPC method call
// in a mock server. It receives the method name and the raw params
// JSON, and returns a result value to be serialised into the response.
type RPCHandler func(method string, params json.RawMessage) (any, error)

// NewMockRPCServer starts an httptest.Server that answers JSON-RPC 2.0
// requests by dispatching to handler. It registers a cleanup function
// that stops the server when the test ends.
//
// Example:
//
//	srv := testutil.NewMockRPCServer(t, func(method string, params json.RawMessage) (any, error) {
//	    if method == "getBalance" {
//	        return map[string]any{"context": map[string]any{"slot": 100}, "value": uint64(500)}, nil
//	    }
//	    return nil, fmt.Errorf("unexpected method %s", method)
//	})
//	client := rpc.NewClient(srv.URL, jsonrpc.Config{})
func NewMockRPCServer(t testing.TB, handler RPCHandler) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     any             `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		result, rpcErr := handler(req.Method, req.Params)
		w.Header().Set("Content-Type", "application/json")
		if rpcErr != nil {
			resp := map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"error": map[string]any{
					"code":    -32000,
					"message": rpcErr.Error(),
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		resp := map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  result,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// NewMockRPCClient creates a mock HTTP server using handler and returns
// an rpc.Client pointed at it.
func NewMockRPCClient(t testing.TB, handler RPCHandler) *rpc.Client {
	t.Helper()
	srv := NewMockRPCServer(t, handler)
	return rpc.NewClient(srv.URL, jsonrpc.Config{})
}
