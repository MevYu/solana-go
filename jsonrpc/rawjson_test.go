package jsonrpc

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestRawJSON_MarshalEmpty(t *testing.T) {
	var r RawJSON
	out, err := r.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "null" {
		t.Errorf("got %q, want null", out)
	}
}

func TestRawJSON_MarshalPassthrough(t *testing.T) {
	r := RawJSON(`{"key":"value"}`)
	out, err := r.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, r) {
		t.Errorf("got %s, want %s", out, r)
	}
}

func TestRawJSON_UnmarshalCopy(t *testing.T) {
	data := []byte(`{"k":42}`)
	var r RawJSON
	if err := r.UnmarshalJSON(data); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(r, data) {
		t.Errorf("got %s, want %s", r, data)
	}
	// Mutating the input must not affect the stored RawJSON.
	data[0] = 0xff
	if r[0] == 0xff {
		t.Error("UnmarshalJSON did not copy bytes")
	}
}

func TestRawJSON_UnmarshalNilReceiver(t *testing.T) {
	var r *RawJSON
	if err := r.UnmarshalJSON([]byte(`"x"`)); err == nil {
		t.Fatal("expected error on nil receiver")
	}
}

func TestRawJSON_WithStdlibJSON(t *testing.T) {
	// Round-trip through encoding/json to prove compatibility with
	// the canonical JSON library shape. Using encoding/json in tests
	// is allowed by project rules.
	type envelope struct {
		Inner RawJSON `json:"inner"`
	}
	e := envelope{Inner: RawJSON(`{"a":1}`)}
	data, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte(`"inner":{"a":1}`)) {
		t.Errorf("unexpected marshal output: %s", data)
	}
	var back envelope
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	if string(back.Inner) != `{"a":1}` {
		t.Errorf("got %s", back.Inner)
	}
}

func TestRawJSON_String(t *testing.T) {
	r := RawJSON(`[1,2,3]`)
	if r.String() != "[1,2,3]" {
		t.Errorf("got %q", r.String())
	}
}
