package jsonrpc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCallContext_Primitive(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":42}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, Config{})
	v, err := CallContext[uint64](context.Background(), c, "getSlot")
	if err != nil {
		t.Fatal(err)
	}
	if v != 42 {
		t.Errorf("value = %d, want 42", v)
	}
}

func TestCallContext_Struct(t *testing.T) {
	type Info struct {
		Lamports uint64 `json:"lamports"`
		Owner    string `json:"owner"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":{"lamports":500,"owner":"11111111111111111111111111111111"}}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, Config{})
	v, err := CallContext[*Info](context.Background(), c, "getX")
	if err != nil {
		t.Fatal(err)
	}
	if v == nil || v.Lamports != 500 {
		t.Errorf("value = %+v", v)
	}
}

func TestCallContext_ContextValue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":{"context":{"slot":42},"value":123}}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, Config{})
	resp, err := CallContext[ContextValue[uint64]](context.Background(), c, "getBalance")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Context.Slot != 42 {
		t.Errorf("slot = %d, want 42", resp.Context.Slot)
	}
	if resp.Value != 123 {
		t.Errorf("value = %d, want 123", resp.Value)
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
	resp, err := CallContext[ContextValue[*Info]](context.Background(), c, "getAccountInfo")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Context.Slot != 9 {
		t.Errorf("slot = %d", resp.Context.Slot)
	}
	if resp.Value != nil {
		t.Errorf("value = %+v, want nil", resp.Value)
	}
}

func TestCallContext_RPCError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"error":{"code":-32602,"message":"bad param"}}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, Config{})
	_, err := CallContext[uint64](context.Background(), c, "getBalance")
	if err == nil {
		t.Fatal("expected error")
	}
	var rpcErr *ErrRPC
	if !errors.As(err, &rpcErr) {
		t.Fatalf("err type = %T, want *ErrRPC", err)
	}
	if rpcErr.Code != -32602 {
		t.Errorf("code = %d", rpcErr.Code)
	}
}
