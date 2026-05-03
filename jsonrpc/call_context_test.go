package jsonrpc

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCallContextValue_Primitive(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":{"context":{"slot":42},"value":123}}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, Config{})
	slot, v, err := CallContextValue[uint64](context.Background(), c, "getBalance", nil)
	if err != nil {
		t.Fatal(err)
	}
	if slot != 42 {
		t.Errorf("slot = %d, want 42", slot)
	}
	if v != 123 {
		t.Errorf("value = %d, want 123", v)
	}
}

func TestCallContextValue_Struct(t *testing.T) {
	type Info struct {
		Lamports uint64 `json:"lamports"`
		Owner    string `json:"owner"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":{"context":{"slot":7},"value":{"lamports":500,"owner":"11111111111111111111111111111111"}}}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, Config{})
	slot, v, err := CallContextValue[*Info](context.Background(), c, "getAccountInfo", nil)
	if err != nil {
		t.Fatal(err)
	}
	if slot != 7 {
		t.Errorf("slot = %d", slot)
	}
	if v == nil || v.Lamports != 500 {
		t.Errorf("value = %+v", v)
	}
}

func TestCallContextValue_NilValue(t *testing.T) {
	type Info struct {
		Lamports uint64 `json:"lamports"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":{"context":{"slot":9},"value":null}}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, Config{})
	slot, v, err := CallContextValue[*Info](context.Background(), c, "getAccountInfo", nil)
	if err != nil {
		t.Fatal(err)
	}
	if slot != 9 {
		t.Errorf("slot = %d", slot)
	}
	if v != nil {
		t.Errorf("value = %+v, want nil", v)
	}
}

func TestCallContextValue_RPCError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"error":{"code":-32602,"message":"bad param"}}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, Config{})
	_, _, err := CallContextValue[uint64](context.Background(), c, "getBalance", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
