package solana

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/klauspost/compress/zstd"
)

// compressZstd is a test helper that zstd-compresses src.
func compressZstd(t *testing.T, src []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w, err := zstd.NewWriter(&buf)
	if err != nil {
		t.Fatalf("zstd.NewWriter: %v", err)
	}
	if _, err := w.Write(src); err != nil {
		t.Fatalf("zstd write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("zstd close: %v", err)
	}
	return buf.Bytes()
}

// wireTuple builds the [value, encoding] JSON tuple as it appears on the
// Solana RPC wire.
func wireTuple(value, encoding string) []byte {
	b, _ := json.Marshal([2]string{value, encoding})
	return b
}

func TestEncodedData_Base64(t *testing.T) {
	payload := []byte{1, 2, 3, 4, 5}
	raw := wireTuple(base64.StdEncoding.EncodeToString(payload), "base64")

	var d EncodedData
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !bytes.Equal(d.Bytes, payload) {
		t.Errorf("Bytes: got %v, want %v", d.Bytes, payload)
	}
	if d.Encoding != EncodingBase64 {
		t.Errorf("Encoding: got %q, want base64", d.Encoding)
	}
}

func TestEncodedData_Base64ZSTD(t *testing.T) {
	payload := bytes.Repeat([]byte("solana"), 100)
	compressed := compressZstd(t, payload)
	raw := wireTuple(base64.StdEncoding.EncodeToString(compressed), "base64+zstd")

	var d EncodedData
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !bytes.Equal(d.Bytes, payload) {
		t.Errorf("decompressed mismatch: got %d bytes, want %d", len(d.Bytes), len(payload))
	}
}

func TestEncodedData_Base64ZSTD_BadBase64(t *testing.T) {
	raw := wireTuple("!!!not-base64!!!", "base64+zstd")
	var d EncodedData
	if err := json.Unmarshal(raw, &d); err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestEncodedData_Base64ZSTD_BadZstd(t *testing.T) {
	raw := wireTuple(base64.StdEncoding.EncodeToString([]byte("not zstd")), "base64+zstd")
	var d EncodedData
	if err := json.Unmarshal(raw, &d); err == nil {
		t.Fatal("expected error for invalid zstd payload")
	}
}

func TestEncodedData_Base64ZSTD_ConcurrentSafe(t *testing.T) {
	payload := []byte("concurrent zstd test payload")
	compressed := compressZstd(t, payload)
	raw := wireTuple(base64.StdEncoding.EncodeToString(compressed), "base64+zstd")

	done := make(chan error, 16)
	for i := 0; i < 16; i++ {
		go func() {
			var d EncodedData
			if err := json.Unmarshal(raw, &d); err != nil {
				done <- err
				return
			}
			if !bytes.Equal(d.Bytes, payload) {
				done <- nil
				return
			}
			done <- nil
		}()
	}
	for i := 0; i < 16; i++ {
		if err := <-done; err != nil {
			t.Errorf("concurrent unmarshal: %v", err)
		}
	}
}

func TestEncodedData_Base58(t *testing.T) {
	raw := wireTuple("3Mc6vR", "base58")
	var d EncodedData
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if d.Encoding != EncodingBase58 {
		t.Errorf("Encoding: got %q, want base58", d.Encoding)
	}
}

func TestEncodedData_Empty(t *testing.T) {
	raw := wireTuple("", "base64")
	var d EncodedData
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if d.Bytes != nil {
		t.Errorf("Bytes: expected nil for empty value, got %v", d.Bytes)
	}
	if d.Encoding != EncodingBase64 {
		t.Errorf("Encoding: got %q, want base64", d.Encoding)
	}
}

func TestEncodedData_JSONParsed(t *testing.T) {
	raw := wireTuple(`{"parsed":"object"}`, "jsonParsed")
	var d EncodedData
	if err := json.Unmarshal(raw, &d); err == nil {
		t.Fatal("expected error for jsonParsed encoding")
	}
}

func TestEncodedData_UnknownEncoding(t *testing.T) {
	raw := wireTuple("abc", "unknown")
	var d EncodedData
	if err := json.Unmarshal(raw, &d); err == nil {
		t.Fatal("expected error for unknown encoding")
	}
}

func TestEncodedData_BadShape(t *testing.T) {
	// jsonParsed-style object response cannot decode into a [2]string tuple.
	raw := []byte(`{"parsed":"object"}`)
	var d EncodedData
	if err := json.Unmarshal(raw, &d); err == nil {
		t.Fatal("expected error for object-shaped data")
	}
}

func TestEncodedData_RoundTrip_Base64(t *testing.T) {
	d := EncodedData{Bytes: []byte("hello"), Encoding: EncodingBase64}
	out, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	want := wireTuple(base64.StdEncoding.EncodeToString([]byte("hello")), "base64")
	if !bytes.Equal(out, want) {
		t.Errorf("Marshal: got %s, want %s", out, want)
	}

	var back EncodedData
	if err := json.Unmarshal(out, &back); err != nil {
		t.Fatalf("Unmarshal round-trip: %v", err)
	}
	if !bytes.Equal(back.Bytes, d.Bytes) {
		t.Errorf("round-trip Bytes mismatch")
	}
	if back.Encoding != d.Encoding {
		t.Errorf("round-trip Encoding mismatch")
	}
}

func TestEncodedData_RoundTrip_Base64ZSTD(t *testing.T) {
	payload := bytes.Repeat([]byte("solana"), 100)
	d := EncodedData{Bytes: payload, Encoding: EncodingBase64ZSTD}
	out, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var back EncodedData
	if err := json.Unmarshal(out, &back); err != nil {
		t.Fatalf("Unmarshal round-trip: %v", err)
	}
	if !bytes.Equal(back.Bytes, payload) {
		t.Errorf("round-trip Bytes mismatch: %d vs %d", len(back.Bytes), len(payload))
	}
}

func TestEncodedData_Marshal_EmptyEncoding(t *testing.T) {
	d := EncodedData{Bytes: []byte("x")}
	if _, err := json.Marshal(d); err == nil {
		t.Fatal("expected error marshaling with empty Encoding")
	}
}

func TestEncodedData_Marshal_JSONParsed(t *testing.T) {
	d := EncodedData{Encoding: EncodingJSONParsed}
	if _, err := json.Marshal(d); err == nil {
		t.Fatal("expected error marshaling jsonParsed encoding")
	}
}
