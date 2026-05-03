package jsonrpc

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// These tests cover Client.CallContext when used with a *ContextValue[T]
// result, which is the canonical way to decode Solana's
// {context:{slot}, value} envelope after CallContextValue was removed.

func TestCallContext_ContextValue_Primitive(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":{"context":{"slot":42},"value":123}}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, Config{})
	var resp ContextValue[uint64]
	if err := c.CallContext(context.Background(), &resp, "getBalance", nil); err != nil {
		t.Fatal(err)
	}
	if resp.Context.Slot != 42 {
		t.Errorf("slot = %d, want 42", resp.Context.Slot)
	}
	if resp.Value != 123 {
		t.Errorf("value = %d, want 123", resp.Value)
	}
}

func TestCallContext_ContextValue_Struct(t *testing.T) {
	type Info struct {
		Lamports uint64 `json:"lamports"`
		Owner    string `json:"owner"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":{"context":{"slot":7},"value":{"lamports":500,"owner":"11111111111111111111111111111111"}}}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, Config{})
	var resp ContextValue[*Info]
	if err := c.CallContext(context.Background(), &resp, "getAccountInfo", nil); err != nil {
		t.Fatal(err)
	}
	if resp.Context.Slot != 7 {
		t.Errorf("slot = %d", resp.Context.Slot)
	}
	if resp.Value == nil || resp.Value.Lamports != 500 {
		t.Errorf("value = %+v", resp.Value)
	}
}

func TestCallContext_ContextValue_NilValue(t *testing.T) {
	type Info struct {
		Lamports uint64 `json:"lamports"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":{"context":{"slot":9},"value":null}}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, Config{})
	var resp ContextValue[*Info]
	if err := c.CallContext(context.Background(), &resp, "getAccountInfo", nil); err != nil {
		t.Fatal(err)
	}
	if resp.Context.Slot != 9 {
		t.Errorf("slot = %d", resp.Context.Slot)
	}
	if resp.Value != nil {
		t.Errorf("value = %+v, want nil", resp.Value)
	}
}

func TestCallContext_ContextValue_RPCError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"error":{"code":-32602,"message":"bad param"}}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, Config{})
	var resp ContextValue[uint64]
	err := c.CallContext(context.Background(), &resp, "getBalance", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
