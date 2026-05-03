package jsonrpc

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// mustMarshal is a test helper using stdlib encoding/json, which is
// allowed in tests by project rules.
func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestClient_Call_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		body, _ := io.ReadAll(r.Body)
		var req Request
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("malformed request: %v", err)
		}
		if req.Version != "2.0" {
			t.Errorf("version = %q, want 2.0", req.Version)
		}
		if req.Method != "getBalance" {
			t.Errorf("method = %q", req.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(mustMarshal(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  map[string]any{"value": 12345},
		}))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, Config{})
	var result struct {
		Value uint64 `json:"value"`
	}
	if err := c.CallContext(context.Background(), &result, "getBalance", []any{"somepubkey"}...); err != nil {
		t.Fatal(err)
	}
	if result.Value != 12345 {
		t.Errorf("value = %d, want 12345", result.Value)
	}
}

func TestClient_Call_RPCError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req Request
		_ = json.Unmarshal(body, &req)
		_, _ = w.Write(mustMarshal(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"error": map[string]any{
				"code":    -32602,
				"message": "Invalid params",
			},
		}))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, Config{})
	err := c.CallContext(context.Background(), nil, "getBalance")
	if err == nil {
		t.Fatal("expected error")
	}
	var rpcErr *ErrRPC
	if !errors.As(err, &rpcErr) {
		t.Fatalf("expected *ErrRPC, got %T: %v", err, err)
	}
	if rpcErr.Code != -32602 {
		t.Errorf("Code = %d, want -32602", rpcErr.Code)
	}
	if rpcErr.Msg != "Invalid params" {
		t.Errorf("Msg = %q", rpcErr.Msg)
	}
	if rpcErr.Method != "getBalance" {
		t.Errorf("Method = %q, want getBalance", rpcErr.Method)
	}
}

func TestClient_Call_Retries5xx(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := hits.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`service unavailable`))
			return
		}
		body, _ := io.ReadAll(r.Body)
		var req Request
		_ = json.Unmarshal(body, &req)
		_, _ = w.Write(mustMarshal(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  "ok",
		}))
	}))
	defer srv.Close()

	// Microsecond-scale backoff so the test finishes fast.
	fastPolicy := &exponentialBackoff{
		MaxAttempts: 5,
		Base:        time.Microsecond,
		Max:         time.Microsecond,
	}

	c := NewClientWith(srv.URL, WithRetryPolicy(fastPolicy))
	var result string
	if err := c.CallContext(context.Background(), &result, "ping"); err != nil {
		t.Fatal(err)
	}
	if result != "ok" {
		t.Errorf("result = %q", result)
	}
	if hits.Load() != 3 {
		t.Errorf("hits = %d, want 3", hits.Load())
	}
}

func TestClient_Call_NoRetry4xx(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`bad request`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, Config{})
	err := c.CallContext(context.Background(), nil, "ping")
	if err == nil {
		t.Fatal("expected error")
	}
	if hits.Load() != 1 {
		t.Errorf("hits = %d, want 1 (no retry)", hits.Load())
	}
}

func TestClient_Call_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			return
		case <-time.After(time.Second):
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":null}`))
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, Config{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := c.CallContext(ctx, nil, "ping")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want wrapping context.Canceled", err)
	}
}

func TestClient_Call_NilResultDiscarded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req Request
		_ = json.Unmarshal(body, &req)
		_, _ = w.Write(mustMarshal(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  "ignored",
		}))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, Config{})
	if err := c.CallContext(context.Background(), nil, "ping"); err != nil {
		t.Fatal(err)
	}
}

type countingCodec struct {
	marshalCalls   atomic.Int32
	unmarshalCalls atomic.Int32
}

func (c *countingCodec) Marshal(v any) ([]byte, error) {
	c.marshalCalls.Add(1)
	return json.Marshal(v)
}

func (c *countingCodec) Unmarshal(data []byte, v any) error {
	c.unmarshalCalls.Add(1)
	return json.Unmarshal(data, v)
}

func TestClient_Call_CustomCodec(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req Request
		_ = json.Unmarshal(body, &req)
		_, _ = w.Write(mustMarshal(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  42,
		}))
	}))
	defer srv.Close()

	codec := &countingCodec{}
	c := NewClientWith(srv.URL, WithCodec(codec))
	var result int
	if err := c.CallContext(context.Background(), &result, "x"); err != nil {
		t.Fatal(err)
	}
	if codec.marshalCalls.Load() != 1 {
		t.Errorf("marshal calls = %d, want 1", codec.marshalCalls.Load())
	}
	// Single Unmarshal: the envelope's Result field is pre-populated with
	// the caller's typed pointer, so the codec decodes the whole response
	// (envelope + typed result) in one pass.
	if codec.unmarshalCalls.Load() != 1 {
		t.Errorf("unmarshal calls = %d, want 1", codec.unmarshalCalls.Load())
	}
	if result != 42 {
		t.Errorf("result = %d", result)
	}
}

func TestClient_Call_MalformedResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`this is not json`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, Config{})
	err := c.CallContext(context.Background(), nil, "x")
	if err == nil {
		t.Fatal("expected decode error")
	}
	if !strings.Contains(err.Error(), "decode response") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestClient_Call_EmptyParamsSent(t *testing.T) {
	// Solana RPC always expects `params: [...]`. Verify we send an
	// empty array even when the caller passes nil.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"params":[]`) {
			t.Errorf("expected params:[] in request body, got %s", body)
		}
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":null}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, Config{})
	if err := c.CallContext(context.Background(), nil, "x"); err != nil {
		t.Fatal(err)
	}
}

func TestClient_Endpoint(t *testing.T) {
	c := NewClient("https://example.com/rpc", Config{})
	if c.Endpoint() != "https://example.com/rpc" {
		t.Errorf("Endpoint = %q", c.Endpoint())
	}
}

func TestClient_Call_UniqueIDs(t *testing.T) {
	var mu sync.Mutex
	seen := make(map[uint64]int)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req Request
		_ = json.Unmarshal(body, &req)
		mu.Lock()
		seen[req.ID]++
		mu.Unlock()
		_, _ = w.Write(mustMarshal(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  nil,
		}))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, Config{})
	for i := 0; i < 10; i++ {
		if err := c.CallContext(context.Background(), nil, "x"); err != nil {
			t.Fatal(err)
		}
	}
	if len(seen) != 10 {
		t.Errorf("saw %d unique IDs, want 10", len(seen))
	}
}

// --- Transport configuration ---

func TestNewClient_HasCustomTransport(t *testing.T) {
	c := NewClient("http://localhost", Config{})
	tr, ok := c.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatal("expected *http.Transport")
	}
	if tr.MaxIdleConnsPerHost != DefaultMaxIdleConnsPerHost {
		t.Errorf("MaxIdleConnsPerHost = %d, want %d", tr.MaxIdleConnsPerHost, DefaultMaxIdleConnsPerHost)
	}
	if !tr.ForceAttemptHTTP2 {
		t.Error("ForceAttemptHTTP2 should be true")
	}
}

func TestWithMaxIdleConnsPerHost(t *testing.T) {
	c := NewClientWith("http://localhost", WithMaxIdleConnsPerHost(42))
	tr, ok := c.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatal("expected *http.Transport")
	}
	if tr.MaxIdleConnsPerHost != 42 {
		t.Errorf("MaxIdleConnsPerHost = %d, want 42", tr.MaxIdleConnsPerHost)
	}
}

func TestWithMaxIdleConnsPerHost_ZeroIgnored(t *testing.T) {
	c := NewClientWith("http://localhost", WithMaxIdleConnsPerHost(0))
	tr, ok := c.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatal("expected *http.Transport")
	}
	if tr.MaxIdleConnsPerHost != DefaultMaxIdleConnsPerHost {
		t.Errorf("zero should be ignored; got %d, want %d",
			tr.MaxIdleConnsPerHost, DefaultMaxIdleConnsPerHost)
	}
}

func TestWithMaxIdleConnsPerHost_WithHTTPClientWins(t *testing.T) {
	// WithHTTPClient takes priority; WithMaxIdleConnsPerHost should be ignored.
	custom := &http.Client{}
	c := NewClientWith("http://localhost",
		WithMaxIdleConnsPerHost(99),
		WithHTTPClient(custom),
	)
	if c.httpClient != custom {
		t.Error("WithHTTPClient should win over WithMaxIdleConnsPerHost")
	}
}

func TestNewClient_WithHTTPClient_Overrides(t *testing.T) {
	custom := &http.Client{}
	c := NewClientWith("http://localhost", WithHTTPClient(custom))
	if c.httpClient != custom {
		t.Error("WithHTTPClient should replace the default transport client")
	}
}

func TestClient_Call_PostBodyContainsMethod(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_, _ = w.Write(mustMarshal(t, map[string]any{
			"jsonrpc": "2.0", "id": 1, "result": 42,
		}))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, Config{})
	var n int
	if err := c.CallContext(context.Background(), &n, "getSlot"); err != nil {
		t.Fatalf("Call: %v", err)
	}
	if n != 42 {
		t.Errorf("result = %d, want 42", n)
	}
	if !strings.Contains(gotBody, "getSlot") {
		t.Errorf("body %q does not contain method name", gotBody)
	}
}
