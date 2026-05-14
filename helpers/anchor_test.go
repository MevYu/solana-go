package helpers

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestAnchorSighash(t *testing.T) {
	// Reference: sha256("global:initialize")[:8] = af af 6d 1f 0d 98 9b ed.
	// Computed directly against crypto/sha256 to avoid hand-typed magic numbers.
	want := sha256.Sum256([]byte("global:initialize"))
	got := AnchorSighash("global", "initialize")
	if got != ([8]byte)(want[:8]) {
		t.Fatalf("AnchorSighash mismatch: got %x want %x", got, want[:8])
	}
}

func TestAnchorMethodDisc_KnownVectors(t *testing.T) {
	// Vectors cross-checked against the Anchor reference: each entry is
	// what `anchor idl parse` emits for the given snake_case method.
	cases := []struct {
		method string
		hex    string
	}{
		{"initialize", "afaf6d1f0d989bed"},
		{"swap", "f8c69e91e17587c8"},
	}
	for _, c := range cases {
		got := AnchorMethodDisc(c.method)
		if hex.EncodeToString(got[:]) != c.hex {
			t.Errorf("AnchorMethodDisc(%q) = %x, want %s", c.method, got, c.hex)
		}
	}
}

func TestAnchorAccountDisc_KnownVector(t *testing.T) {
	// sha256("account:Pool")[:8]; recomputed here so the test stays
	// self-checking if the algorithm is ever touched.
	want := sha256.Sum256([]byte("account:Pool"))
	got := AnchorAccountDisc("Pool")
	if got != ([8]byte)(want[:8]) {
		t.Fatalf("AnchorAccountDisc(Pool) = %x, want %x", got, want[:8])
	}
}
