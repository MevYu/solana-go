// Package testutil provides shared test helpers for solana-go.
// It is intended for use in _test packages only; import it with
//
//	import "github.com/MevYu/solana-go/internal/testutil"
package testutil

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	solana "github.com/MevYu/solana-go"
)

// MustNewKeypair generates a fresh Ed25519 keypair for use in tests.
// It calls t.Fatal on failure.
func MustNewKeypair(t testing.TB) (solana.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("testutil: MustNewKeypair: %v", err)
	}
	var pk solana.PublicKey
	copy(pk[:], pub)
	return pk, priv
}

// MustPublicKey decodes a base58 public key and calls t.Fatal on failure.
func MustPublicKey(t testing.TB, s string) solana.PublicKey {
	t.Helper()
	pk, err := solana.PublicKeyFromBase58(s)
	if err != nil {
		t.Fatalf("testutil: MustPublicKey(%q): %v", s, err)
	}
	return pk
}

// MustHash decodes a base58 hash and calls t.Fatal on failure.
func MustHash(t testing.TB, s string) solana.Hash {
	t.Helper()
	h, err := solana.HashFromBase58(s)
	if err != nil {
		t.Fatalf("testutil: MustHash(%q): %v", s, err)
	}
	return h
}

// ZeroHash returns a Hash with all bytes set to zero. Useful as a
// placeholder blockhash in unit tests that don't exercise blockhash
// validation.
func ZeroHash() solana.Hash { return solana.Hash{} }
