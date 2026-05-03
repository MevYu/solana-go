package jsonrpc

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBatchCallContext_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqs []Request
		if err := json.Unmarshal(body, &reqs); err != nil {
			t.Fatalf("expected batch array, got: %s", body)
		}
		if len(reqs) != 2 {
			t.Fatalf("len(reqs) = %d, want 2", len(reqs))
		}
		resps := make([]map[string]any, len(reqs))
		for i, req := range reqs {
			resps[i] = map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result":  req.Method + "-ok",
			}
		}
		_, _ = w.Write(mustMarshal(t, resps))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, Config{})
	var a, b string
	batch := []BatchElem{
		{Method: "getSlot", Result: &a},
		{Method: "getBlockHeight", Result: &b},
	}
	if err := c.BatchCallContext(context.Background(), batch); err != nil {
		t.Fatal(err)
	}
	if a != "getSlot-ok" {
		t.Errorf("a = %q", a)
	}
	if b != "getBlockHeight-ok" {
		t.Errorf("b = %q", b)
	}
	for i, e := range batch {
		if e.Error != nil {
			t.Errorf("batch[%d].Error = %v", i, e.Error)
		}
	}
}

func TestBatchCallContext_OutOfOrderResponses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqs []Request
		_ = json.Unmarshal(body, &reqs)
		// Return responses in reverse order to exercise ID-based matching.
		resps := make([]map[string]any, len(reqs))
		for i, req := range reqs {
			resps[len(reqs)-1-i] = map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result":  req.Method,
			}
		}
		_, _ = w.Write(mustMarshal(t, resps))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, Config{})
	var a, b, d string
	batch := []BatchElem{
		{Method: "methodA", Result: &a},
		{Method: "methodB", Result: &b},
		{Method: "methodC", Result: &d},
	}
	if err := c.BatchCallContext(context.Background(), batch); err != nil {
		t.Fatal(err)
	}
	if a != "methodA" || b != "methodB" || d != "methodC" {
		t.Errorf("results out of order: %q %q %q", a, b, d)
	}
}

func TestBatchCallContext_PerElementRPCError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqs []Request
		_ = json.Unmarshal(body, &reqs)
		resps := make([]map[string]any, len(reqs))
		for i, req := range reqs {
			if req.Method == "bad" {
				resps[i] = map[string]any{
					"jsonrpc": "2.0",
					"id":      req.ID,
					"error": map[string]any{
						"code":    -32601,
						"message": "Method not found",
					},
				}
			} else {
				resps[i] = map[string]any{
					"jsonrpc": "2.0",
					"id":      req.ID,
					"result":  42,
				}
			}
		}
		_, _ = w.Write(mustMarshal(t, resps))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, Config{})
	var ok int
	batch := []BatchElem{
		{Method: "bad"},
		{Method: "good", Result: &ok},
	}
	if err := c.BatchCallContext(context.Background(), batch); err != nil {
		t.Fatalf("transport err = %v", err)
	}
	if batch[1].Error != nil {
		t.Errorf("batch[1].Error = %v, want nil", batch[1].Error)
	}
	if ok != 42 {
		t.Errorf("ok = %d, want 42", ok)
	}
	var rpcErr *ErrRPC
	if !errors.As(batch[0].Error, &rpcErr) {
		t.Fatalf("batch[0].Error = %T %v, want *ErrRPC", batch[0].Error, batch[0].Error)
	}
	if rpcErr.Code != -32601 {
		t.Errorf("Code = %d", rpcErr.Code)
	}
	if rpcErr.Method != "bad" {
		t.Errorf("Method = %q", rpcErr.Method)
	}
}

func TestBatchCallContext_MissingResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqs []Request
		_ = json.Unmarshal(body, &reqs)
		// Drop the last request from the response.
		var resps []map[string]any
		for i := 0; i < len(reqs)-1; i++ {
			resps = append(resps, map[string]any{
				"jsonrpc": "2.0",
				"id":      reqs[i].ID,
				"result":  "ok",
			})
		}
		_, _ = w.Write(mustMarshal(t, resps))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, Config{})
	var a, b string
	batch := []BatchElem{
		{Method: "m1", Result: &a},
		{Method: "m2", Result: &b},
	}
	if err := c.BatchCallContext(context.Background(), batch); err != nil {
		t.Fatal(err)
	}
	if batch[0].Error != nil {
		t.Errorf("batch[0].Error = %v", batch[0].Error)
	}
	if !errors.Is(batch[1].Error, ErrMissingResponse) {
		t.Errorf("batch[1].Error = %v, want ErrMissingResponse", batch[1].Error)
	}
}

func TestBatchCallContext_EmptyInput(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
	}))
	defer srv.Close()

	c := NewClient(srv.URL, Config{})
	if err := c.BatchCallContext(context.Background(), nil); err != nil {
		t.Fatalf("nil batch: %v", err)
	}
	if err := c.BatchCallContext(context.Background(), []BatchElem{}); err != nil {
		t.Fatalf("empty batch: %v", err)
	}
	if hits != 0 {
		t.Errorf("empty batch should not hit server; hits=%d", hits)
	}
}

func TestBatchCallContext_NilArgsSerialisedAsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"params":[]`) {
			t.Errorf("expected params:[] in body, got %s", body)
		}
		var reqs []Request
		_ = json.Unmarshal(body, &reqs)
		resps := make([]map[string]any, len(reqs))
		for i, req := range reqs {
			resps[i] = map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result":  nil,
			}
		}
		_, _ = w.Write(mustMarshal(t, resps))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, Config{})
	batch := []BatchElem{{Method: "getVersion"}}
	if err := c.BatchCallContext(context.Background(), batch); err != nil {
		t.Fatal(err)
	}
}

func TestBatchCallContext_DecodeResultError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqs []Request
		_ = json.Unmarshal(body, &reqs)
		_, _ = w.Write(mustMarshal(t, []map[string]any{{
			"jsonrpc": "2.0",
			"id":      reqs[0].ID,
			"result":  "not-a-number",
		}}))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, Config{})
	var n int
	batch := []BatchElem{{Method: "getSlot", Result: &n}}
	if err := c.BatchCallContext(context.Background(), batch); err != nil {
		t.Fatalf("transport err = %v", err)
	}
	if batch[0].Error == nil {
		t.Fatal("expected decode error on batch[0]")
	}
	if !strings.Contains(batch[0].Error.Error(), "decode result") {
		t.Errorf("err = %v", batch[0].Error)
	}
}

func TestBatchCallContext_MalformedResponseArray(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, Config{})
	batch := []BatchElem{{Method: "getSlot"}}
	err := c.BatchCallContext(context.Background(), batch)
	if err == nil {
		t.Fatal("expected transport error")
	}
	if !strings.Contains(err.Error(), "decode response array") {
		t.Errorf("err = %v", err)
	}
}

func TestBatchCallContext_UniqueIDsPerElement(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqs []Request
		_ = json.Unmarshal(body, &reqs)
		seen := make(map[uint64]bool)
		for _, req := range reqs {
			if seen[req.ID] {
				t.Errorf("duplicate ID %d in batch", req.ID)
			}
			seen[req.ID] = true
		}
		resps := make([]map[string]any, len(reqs))
		for i, req := range reqs {
			resps[i] = map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": nil}
		}
		_, _ = w.Write(mustMarshal(t, resps))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, Config{})
	batch := make([]BatchElem, 5)
	for i := range batch {
		batch[i] = BatchElem{Method: "x"}
	}
	if err := c.BatchCallContext(context.Background(), batch); err != nil {
		t.Fatal(err)
	}
}
