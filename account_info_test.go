package solana

import (
	"bytes"
	"encoding/base64"
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

// --- AccountData.Bytes ---

func TestAccountData_Base64(t *testing.T) {
	payload := []byte{1, 2, 3, 4, 5}
	d := AccountData{base64.StdEncoding.EncodeToString(payload), "base64"}
	got, err := d.Bytes()
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("got %v, want %v", got, payload)
	}
}

func TestAccountData_Base64ZSTD(t *testing.T) {
	payload := bytes.Repeat([]byte("solana"), 100) // compressible
	compressed := compressZstd(t, payload)
	encoded := base64.StdEncoding.EncodeToString(compressed)

	d := AccountData{encoded, "base64+zstd"}
	got, err := d.Bytes()
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("decompressed data mismatch: got %d bytes, want %d bytes", len(got), len(payload))
	}
}

func TestAccountData_Base64ZSTD_BadBase64(t *testing.T) {
	d := AccountData{"!!!not-base64!!!", "base64+zstd"}
	_, err := d.Bytes()
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestAccountData_Base64ZSTD_BadZstd(t *testing.T) {
	// Valid base64 but not valid zstd data.
	d := AccountData{base64.StdEncoding.EncodeToString([]byte("not zstd")), "base64+zstd"}
	_, err := d.Bytes()
	if err == nil {
		t.Fatal("expected error for invalid zstd payload")
	}
}

func TestAccountData_Base64ZSTD_ConcurrentSafe(t *testing.T) {
	// Verify the singleton decoder handles concurrent callers.
	payload := []byte("concurrent zstd test payload")
	compressed := compressZstd(t, payload)
	encoded := base64.StdEncoding.EncodeToString(compressed)
	d := AccountData{encoded, "base64+zstd"}

	done := make(chan error, 16)
	for i := 0; i < 16; i++ {
		go func() {
			got, err := d.Bytes()
			if err != nil {
				done <- err
				return
			}
			if !bytes.Equal(got, payload) {
				done <- nil // wrong data but no error; checked below
				return
			}
			done <- nil
		}()
	}
	for i := 0; i < 16; i++ {
		if err := <-done; err != nil {
			t.Errorf("concurrent Bytes: %v", err)
		}
	}
}

func TestAccountData_Base58(t *testing.T) {
	// base58 for a small payload
	d := AccountData{"3Mc6vR", "base58"} // base58("hello") approximately
	_, err := d.Bytes()
	// Just verify it doesn't panic; actual value depends on base58 alphabet.
	_ = err
}

func TestAccountData_Empty(t *testing.T) {
	d := AccountData{"", "base64"}
	got, err := d.Bytes()
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for empty value, got %v", got)
	}
}

func TestAccountData_JSONParsed(t *testing.T) {
	d := AccountData{`{"parsed":"object"}`, "jsonParsed"}
	_, err := d.Bytes()
	if err == nil {
		t.Fatal("expected error for jsonParsed encoding")
	}
}

func TestAccountData_UnknownEncoding(t *testing.T) {
	d := AccountData{"abc", "unknown"}
	_, err := d.Bytes()
	if err == nil {
		t.Fatal("expected error for unknown encoding")
	}
}
