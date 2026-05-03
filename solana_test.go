package solana

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
)

const (
	// systemProgramID decodes to 32 zero bytes; it is the canonical
	// base58 form of the Solana system program address and a useful
	// fixed-point for round-trip tests.
	systemProgramID = "11111111111111111111111111111111"
	// tokenProgramID is the SPL Token program. It exercises the full
	// base58 alphabet with a real-world address.
	tokenProgramID = "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"
)

// ---------------------------------------------------------------------
// Constructors from raw bytes
// ---------------------------------------------------------------------

func TestPublicKeyFromBytes_RoundTrip(t *testing.T) {
	in := make([]byte, PublicKeySize)
	for i := range in {
		in[i] = byte(i)
	}
	pk, err := PublicKeyFromBytes(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(pk.Bytes(), in) {
		t.Fatalf("round-trip mismatch: got %x want %x", pk.Bytes(), in)
	}
}

func TestPublicKeyFromBytes_InvalidLength(t *testing.T) {
	cases := []struct {
		name string
		in   []byte
	}{
		{"nil", nil},
		{"empty", []byte{}},
		{"short", make([]byte, PublicKeySize-1)},
		{"long", make([]byte, PublicKeySize+1)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := PublicKeyFromBytes(tc.in)
			if !errors.Is(err, ErrInvalidLength) {
				t.Fatalf("expected ErrInvalidLength, got %v", err)
			}
		})
	}
}

func TestPublicKey_BytesIsDefensiveCopy(t *testing.T) {
	in := make([]byte, PublicKeySize)
	for i := range in {
		in[i] = byte(i + 1)
	}
	pk, err := PublicKeyFromBytes(in)
	if err != nil {
		t.Fatal(err)
	}
	out := pk.Bytes()
	out[0] = 0xff
	if pk[0] == 0xff {
		t.Fatal("mutating the returned slice mutated the receiver")
	}
}

func TestPublicKey_EqualAndIsZero(t *testing.T) {
	var a, b PublicKey
	if !a.Equal(b) {
		t.Fatal("two zero keys should be Equal")
	}
	if !a.IsZero() {
		t.Fatal("zero-value PublicKey should report IsZero=true")
	}
	a[0] = 1
	if a.Equal(b) {
		t.Fatal("keys that differ in one byte should not be Equal")
	}
	if a.IsZero() {
		t.Fatal("non-zero PublicKey should report IsZero=false")
	}
}

func TestHashFromBytes_InvalidLength(t *testing.T) {
	if _, err := HashFromBytes(make([]byte, HashSize-1)); !errors.Is(err, ErrInvalidLength) {
		t.Fatalf("expected ErrInvalidLength, got %v", err)
	}
}

func TestSignatureFromBytes_InvalidLength(t *testing.T) {
	if _, err := SignatureFromBytes(make([]byte, SignatureSize-1)); !errors.Is(err, ErrInvalidLength) {
		t.Fatalf("expected ErrInvalidLength, got %v", err)
	}
}

func TestHash_BytesIsDefensiveCopy(t *testing.T) {
	var h Hash
	h[0] = 0x42
	out := h.Bytes()
	out[0] = 0xff
	if h[0] != 0x42 {
		t.Fatal("Hash.Bytes should return a defensive copy")
	}
}

func TestSignature_BytesIsDefensiveCopy(t *testing.T) {
	var s Signature
	s[0] = 0x42
	out := s.Bytes()
	out[0] = 0xff
	if s[0] != 0x42 {
		t.Fatal("Signature.Bytes should return a defensive copy")
	}
}

// ---------------------------------------------------------------------
// Base58 round-trips with real Solana program vectors
// ---------------------------------------------------------------------

func TestPublicKey_Base58_SystemProgram(t *testing.T) {
	pk, err := PublicKeyFromBase58(systemProgramID)
	if err != nil {
		t.Fatal(err)
	}
	var zero PublicKey
	if !pk.Equal(zero) {
		t.Fatalf("system program should decode to 32 zero bytes, got %x", pk)
	}
	if got := pk.String(); got != systemProgramID {
		t.Fatalf("round-trip mismatch: got %q want %q", got, systemProgramID)
	}
}

func TestPublicKey_Base58_TokenProgram(t *testing.T) {
	pk, err := PublicKeyFromBase58(tokenProgramID)
	if err != nil {
		t.Fatal(err)
	}
	if pk.IsZero() {
		t.Fatal("token program id should not decode to zero")
	}
	if got := pk.String(); got != tokenProgramID {
		t.Fatalf("round-trip mismatch: got %q want %q", got, tokenProgramID)
	}
}

func TestPublicKeyFromBase58_Empty(t *testing.T) {
	if _, err := PublicKeyFromBase58(""); !errors.Is(err, ErrInvalidBase58) {
		t.Fatalf("expected ErrInvalidBase58, got %v", err)
	}
}

func TestPublicKeyFromBase58_InvalidChars(t *testing.T) {
	// 'O' (uppercase o) is not in the base58 alphabet.
	_, err := PublicKeyFromBase58("OOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOO")
	if !errors.Is(err, ErrInvalidBase58) {
		t.Fatalf("expected ErrInvalidBase58, got %v", err)
	}
}

func TestPublicKeyFromBase58_InvalidLength(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"29 zeros", "11111111111111111111111111111"},
		{"37 zeros", "1111111111111111111111111111111111111"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := PublicKeyFromBase58(tc.in)
			if !errors.Is(err, ErrInvalidLength) {
				t.Fatalf("expected ErrInvalidLength, got %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------
// JSON
// ---------------------------------------------------------------------

func TestPublicKey_JSON_RoundTrip(t *testing.T) {
	pk, err := PublicKeyFromBase58(tokenProgramID)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(pk)
	if err != nil {
		t.Fatal(err)
	}
	want := `"` + tokenProgramID + `"`
	if string(data) != want {
		t.Fatalf("MarshalJSON = %s, want %s", data, want)
	}
	var out PublicKey
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if !out.Equal(pk) {
		t.Fatal("round-trip mismatch")
	}
}

func TestPublicKey_JSON_Null(t *testing.T) {
	pk, _ := PublicKeyFromBase58(tokenProgramID)
	before := pk
	if err := json.Unmarshal([]byte("null"), &pk); err != nil {
		t.Fatal(err)
	}
	if !pk.Equal(before) {
		t.Fatal("null should leave the receiver unchanged")
	}
}

func TestPublicKey_JSON_UnmarshalInvalid(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"not a string", "42"},
		{"empty string", `""`},
		{"invalid base58", `"OOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOO"`},
		{"wrong length", `"11111111111111111111111111111"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var pk PublicKey
			if err := json.Unmarshal([]byte(tc.in), &pk); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

// ---------------------------------------------------------------------
// Text
// ---------------------------------------------------------------------

func TestPublicKey_Text_RoundTrip(t *testing.T) {
	pk, _ := PublicKeyFromBase58(tokenProgramID)
	text, err := pk.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	if string(text) != tokenProgramID {
		t.Fatalf("MarshalText = %s, want %s", text, tokenProgramID)
	}
	var out PublicKey
	if err := out.UnmarshalText(text); err != nil {
		t.Fatal(err)
	}
	if !out.Equal(pk) {
		t.Fatal("round-trip mismatch")
	}
}

// ---------------------------------------------------------------------
// database/sql
// ---------------------------------------------------------------------

func TestPublicKey_SQL(t *testing.T) {
	pk, _ := PublicKeyFromBase58(tokenProgramID)
	v, err := pk.Value()
	if err != nil {
		t.Fatal(err)
	}
	s, ok := v.(string)
	if !ok {
		t.Fatalf("Value should return string, got %T", v)
	}
	if s != tokenProgramID {
		t.Fatalf("Value = %q, want %q", s, tokenProgramID)
	}

	var fromString PublicKey
	if err := fromString.Scan(tokenProgramID); err != nil {
		t.Fatal(err)
	}
	if !fromString.Equal(pk) {
		t.Fatal("Scan(string) round-trip mismatch")
	}

	var fromBytes PublicKey
	if err := fromBytes.Scan([]byte(tokenProgramID)); err != nil {
		t.Fatal(err)
	}
	if !fromBytes.Equal(pk) {
		t.Fatal("Scan([]byte) round-trip mismatch")
	}

	var fromNil PublicKey
	if err := fromNil.Scan(nil); err != nil {
		t.Fatal(err)
	}
	if !fromNil.IsZero() {
		t.Fatal("Scan(nil) should leave receiver zero")
	}

	var fromInvalid PublicKey
	if err := fromInvalid.Scan(42); err == nil {
		t.Fatal("Scan(int) should error")
	}
}

// ---------------------------------------------------------------------
// Hash / Signature smoke tests
// ---------------------------------------------------------------------

func TestHash_Base58_RoundTrip(t *testing.T) {
	var h Hash
	for i := range h {
		h[i] = byte(i)
	}
	s := h.String()
	out, err := HashFromBase58(s)
	if err != nil {
		t.Fatal(err)
	}
	if !out.Equal(h) {
		t.Fatal("round-trip mismatch")
	}
}

func TestHash_JSON_RoundTrip(t *testing.T) {
	var h Hash
	for i := range h {
		h[i] = byte(255 - i)
	}
	data, err := json.Marshal(h)
	if err != nil {
		t.Fatal(err)
	}
	var out Hash
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if !out.Equal(h) {
		t.Fatal("round-trip mismatch")
	}
}

func TestHashFromBase58_Errors(t *testing.T) {
	if _, err := HashFromBase58(""); !errors.Is(err, ErrInvalidBase58) {
		t.Fatalf("expected ErrInvalidBase58, got %v", err)
	}
	if _, err := HashFromBase58("1"); !errors.Is(err, ErrInvalidLength) {
		t.Fatalf("expected ErrInvalidLength, got %v", err)
	}
}

func TestSignature_Base58_RoundTrip(t *testing.T) {
	var sig Signature
	for i := range sig {
		sig[i] = byte(i * 3)
	}
	s := sig.String()
	out, err := SignatureFromBase58(s)
	if err != nil {
		t.Fatal(err)
	}
	if !out.Equal(sig) {
		t.Fatal("round-trip mismatch")
	}
}

func TestSignature_JSON_RoundTrip(t *testing.T) {
	var sig Signature
	for i := range sig {
		sig[i] = byte(i + 7)
	}
	data, err := json.Marshal(sig)
	if err != nil {
		t.Fatal(err)
	}
	var out Signature
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if !out.Equal(sig) {
		t.Fatal("round-trip mismatch")
	}
}

func TestSignatureFromBase58_Errors(t *testing.T) {
	if _, err := SignatureFromBase58(""); !errors.Is(err, ErrInvalidBase58) {
		t.Fatalf("expected ErrInvalidBase58, got %v", err)
	}
	if _, err := SignatureFromBase58("1"); !errors.Is(err, ErrInvalidLength) {
		t.Fatalf("expected ErrInvalidLength, got %v", err)
	}
}
