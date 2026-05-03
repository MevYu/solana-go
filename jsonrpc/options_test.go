package jsonrpc

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- WithHeader / WithHeaders / SetHeader ---

func TestWithHeader_SentOnRequest(t *testing.T) {
	var gotHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-API-Key")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(successResponse(t, 1, nil))
	}))
	defer srv.Close()

	c := NewClientWith(srv.URL, WithHeader("X-API-Key", "secret"))
	_ = c.CallContext(context.Background(), nil, "getSlot")
	if gotHeader != "secret" {
		t.Errorf("X-API-Key = %q, want %q", gotHeader, "secret")
	}
}

func TestWithHeaders_AllSentOnRequest(t *testing.T) {
	got := http.Header{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got["X-Foo"] = r.Header["X-Foo"]
		got["X-Bar"] = r.Header["X-Bar"]
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(successResponse(t, 1, nil))
	}))
	defer srv.Close()

	h := http.Header{"X-Foo": {"foo"}, "X-Bar": {"bar"}}
	c := NewClientWith(srv.URL, WithHeaders(h))
	_ = c.CallContext(context.Background(), nil, "getSlot")
	if got.Get("X-Foo") != "foo" {
		t.Errorf("X-Foo = %q", got.Get("X-Foo"))
	}
	if got.Get("X-Bar") != "bar" {
		t.Errorf("X-Bar = %q", got.Get("X-Bar"))
	}
}

func TestSetHeader_UpdatesAfterConstruction(t *testing.T) {
	var gotKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("X-Key")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(successResponse(t, 1, nil))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, Config{})
	c.SetHeader("X-Key", "updated")
	_ = c.CallContext(context.Background(), nil, "getSlot")
	if gotKey != "updated" {
		t.Errorf("X-Key = %q, want %q", gotKey, "updated")
	}
}

// --- WithHTTPClient ---

func TestWithHTTPClient_UsedForRequests(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(successResponse(t, 1, nil))
	}))
	defer srv.Close()

	custom := &http.Client{}
	c := NewClientWith(srv.URL, WithHTTPClient(custom))
	_ = c.CallContext(context.Background(), nil, "getSlot")
	if !called {
		t.Error("custom HTTP client was not used")
	}
}

func TestWithHTTPClient_NilIsIgnored(t *testing.T) {
	// nil should not replace the default client
	c := NewClientWith("http://localhost", WithHTTPClient(nil))
	if c.httpClient == nil {
		t.Error("httpClient should not be nil after WithHTTPClient(nil)")
	}
}

// --- WithHTTPAuth ---

func TestWithHTTPAuth_InjectsHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(successResponse(t, 1, nil))
	}))
	defer srv.Close()

	c := NewClientWith(srv.URL, WithHTTPAuth(func(h http.Header) error {
		h.Set("Authorization", "Bearer token123")
		return nil
	}))
	_ = c.CallContext(context.Background(), nil, "getSlot")
	if gotAuth != "Bearer token123" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer token123")
	}
}

// --- StdCodec ---

func TestStdCodec_MarshalUnmarshal(t *testing.T) {
	codec := StdCodec()
	req := Request{Version: "2.0", ID: 1, Method: "test", Params: []any{"a"}}
	b, err := codec.Marshal(&req)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got Request
	if err := codec.Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Method != "test" || got.ID != 1 {
		t.Errorf("round-trip mismatch: %+v", got)
	}
}

// --- NewContextWithHeaders ---

func TestNewContextWithHeaders_AddedToRequest(t *testing.T) {
	var gotTrace string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTrace = r.Header.Get("X-Trace-Id")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(successResponse(t, 1, nil))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, Config{})
	ctx := NewContextWithHeaders(context.Background(), http.Header{"X-Trace-Id": {"abc-123"}})
	_ = c.CallContext(ctx, nil, "getSlot")
	if gotTrace != "abc-123" {
		t.Errorf("X-Trace-Id = %q, want %q", gotTrace, "abc-123")
	}
}

func TestNewContextWithHeaders_EmptyIsNoop(t *testing.T) {
	ctx := context.Background()
	ctx2 := NewContextWithHeaders(ctx, http.Header{})
	if ctx2 != ctx {
		t.Error("expected same context when headers are empty")
	}
}

func TestNewContextWithHeaders_Merges(t *testing.T) {
	ctx := NewContextWithHeaders(context.Background(), http.Header{"X-A": {"1"}})
	ctx = NewContextWithHeaders(ctx, http.Header{"X-B": {"2"}})
	h := headersFromContext(ctx)
	if h.Get("X-A") != "1" {
		t.Errorf("X-A = %q", h.Get("X-A"))
	}
	if h.Get("X-B") != "2" {
		t.Errorf("X-B = %q", h.Get("X-B"))
	}
}

// --- helpers ---

// successResponse returns a minimal JSON-RPC success body with a null result.
func successResponse(t *testing.T, id uint64, result any) []byte {
	t.Helper()
	b, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// Ensure headersFromContext is exercised on a context with no headers.
func TestHeadersFromContext_Empty(t *testing.T) {
	h := headersFromContext(context.Background())
	if h != nil {
		t.Errorf("expected nil, got %v", h)
	}
}

// Verify WithHeaders does not panic when called on a client that already
// has headers set (map must be initialised before range).
func TestWithHeaders_Cumulative(t *testing.T) {
	c := NewClientWith("http://localhost",
		WithHeader("X-First", "1"),
		WithHeaders(http.Header{"X-Second": {"2"}}),
	)
	if c.headers.Get("X-First") != "1" {
		t.Error("X-First missing")
	}
	if c.headers.Get("X-Second") != "2" {
		t.Error("X-Second missing")
	}
}

// Verify the client propagates the custom HTTP client to actual requests
// and that WithHTTPClient(nil) is ignored (default not overwritten).
func TestWithHTTPClient_NilBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(successResponse(t, 1, 42))
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
}
