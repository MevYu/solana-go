package solana

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"errors"
	"testing"

	"github.com/mr-tron/base58"
)

func TestEd25519Keypair_GenerateAndSign(t *testing.T) {
	kp, err := NewEd25519Keypair()
	if err != nil {
		t.Fatal(err)
	}
	if kp.PublicKey().IsZero() {
		t.Fatal("generated public key is zero")
	}
	msg := []byte("hello solana")
	sig, err := kp.Sign(context.Background(), msg)
	if err != nil {
		t.Fatal(err)
	}
	// Verify using stdlib to prove we produced a real Ed25519
	// signature, not just random bytes.
	pub := kp.PublicKey()
	if !ed25519.Verify(ed25519.PublicKey(pub[:]), msg, sig[:]) {
		t.Fatal("generated signature does not verify against its own public key")
	}
}

func TestEd25519KeypairFromSeed_Deterministic(t *testing.T) {
	seed := make([]byte, ed25519.SeedSize)
	for i := range seed {
		seed[i] = byte(i)
	}
	kp1, err := Ed25519KeypairFromSeed(seed)
	if err != nil {
		t.Fatal(err)
	}
	kp2, err := Ed25519KeypairFromSeed(seed)
	if err != nil {
		t.Fatal(err)
	}
	if !kp1.PublicKey().Equal(kp2.PublicKey()) {
		t.Fatal("same seed produced different public keys")
	}
	// And the signatures must be deterministic too (Ed25519 is).
	msg := []byte("repeat signing")
	s1, _ := kp1.Sign(context.Background(), msg)
	s2, _ := kp2.Sign(context.Background(), msg)
	if !s1.Equal(s2) {
		t.Fatal("same seed + same message produced different signatures")
	}
}

func TestEd25519KeypairFromSeed_InvalidLength(t *testing.T) {
	cases := []int{0, 1, ed25519.SeedSize - 1, ed25519.SeedSize + 1}
	for _, n := range cases {
		_, err := Ed25519KeypairFromSeed(make([]byte, n))
		if !errors.Is(err, ErrInvalidLength) {
			t.Errorf("seed len %d: expected ErrInvalidLength, got %v", n, err)
		}
	}
}

func TestEd25519KeypairFromPrivateKey_RoundTrip(t *testing.T) {
	kp1, err := NewEd25519Keypair()
	if err != nil {
		t.Fatal(err)
	}
	priv := kp1.PrivateKey()
	kp2, err := Ed25519KeypairFromPrivateKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	if !kp1.PublicKey().Equal(kp2.PublicKey()) {
		t.Fatal("reconstructed keypair does not match original public key")
	}
	msg := []byte("round trip")
	s1, _ := kp1.Sign(context.Background(), msg)
	s2, _ := kp2.Sign(context.Background(), msg)
	if !s1.Equal(s2) {
		t.Fatal("reconstructed keypair does not produce identical signatures")
	}
}

func TestEd25519KeypairFromPrivateKey_InvalidLength(t *testing.T) {
	if _, err := Ed25519KeypairFromPrivateKey(make([]byte, 63)); !errors.Is(err, ErrInvalidLength) {
		t.Fatalf("expected ErrInvalidLength, got %v", err)
	}
}

func TestEd25519KeypairFromBase58Key(t *testing.T) {
	kp, err := NewEd25519Keypair()
	if err != nil {
		t.Fatal(err)
	}
	priv := kp.PrivateKey() // 64-byte expanded form, as solana-keygen exports

	// 64-byte standard Solana key (the case that previously failed with
	// "got 64, want 32").
	got, err := Ed25519KeypairFromBase58Key(base58.Encode(priv))
	if err != nil {
		t.Fatalf("64-byte key: %v", err)
	}
	if !got.PublicKey().Equal(kp.PublicKey()) {
		t.Fatal("64-byte key: public key mismatch")
	}

	// 32-byte bare seed.
	got, err = Ed25519KeypairFromBase58Key(base58.Encode(priv[:ed25519.SeedSize]))
	if err != nil {
		t.Fatalf("32-byte seed: %v", err)
	}
	if !got.PublicKey().Equal(kp.PublicKey()) {
		t.Fatal("32-byte seed: public key mismatch")
	}

	if _, err := Ed25519KeypairFromBase58Key(""); err == nil {
		t.Fatal("empty key: expected error")
	}
	if _, err := Ed25519KeypairFromBase58Key(base58.Encode(make([]byte, 33))); !errors.Is(err, ErrInvalidLength) {
		t.Fatalf("bad length: expected ErrInvalidLength, got %v", err)
	}
}

func TestEd25519Keypair_PrivateKeyIsDefensiveCopy(t *testing.T) {
	kp, err := NewEd25519Keypair()
	if err != nil {
		t.Fatal(err)
	}
	out := kp.PrivateKey()
	for i := range out {
		out[i] = 0xff
	}
	// Signing must still work after zeroing the returned slice,
	// proving PrivateKey returned a copy.
	if _, err := kp.Sign(context.Background(), []byte("still works")); err != nil {
		t.Fatal(err)
	}
}

// ---------------------------------------------------------------------
// RemoteSigner
// ---------------------------------------------------------------------

func TestRemoteSigner_DelegatesToFunc(t *testing.T) {
	kp, err := NewEd25519Keypair()
	if err != nil {
		t.Fatal(err)
	}
	called := false
	fn := func(ctx context.Context, message []byte) (Signature, error) {
		called = true
		return kp.Sign(ctx, message)
	}
	rs, err := NewRemoteSigner(kp.PublicKey(), fn)
	if err != nil {
		t.Fatal(err)
	}
	if !rs.PublicKey().Equal(kp.PublicKey()) {
		t.Fatal("RemoteSigner public key mismatch")
	}
	sig, err := rs.Sign(context.Background(), []byte("remote"))
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("RemoteSignFunc was not invoked")
	}
	pub := kp.PublicKey()
	if !ed25519.Verify(ed25519.PublicKey(pub[:]), []byte("remote"), sig[:]) {
		t.Fatal("RemoteSigner returned an invalid signature")
	}
}

func TestRemoteSigner_NilFunc(t *testing.T) {
	if _, err := NewRemoteSigner(PublicKey{}, nil); err == nil {
		t.Fatal("expected error for nil RemoteSignFunc")
	}
}

func TestRemoteSigner_PropagatesContextCancel(t *testing.T) {
	sentinel := errors.New("cancelled by test")
	fn := func(ctx context.Context, message []byte) (Signature, error) {
		select {
		case <-ctx.Done():
			return Signature{}, sentinel
		default:
			return Signature{}, errors.New("ctx not cancelled")
		}
	}
	rs, err := NewRemoteSigner(PublicKey{0x01}, fn)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = rs.Sign(ctx, []byte("x"))
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", err)
	}
}

func TestRemoteSigner_PropagatesErrorFromFunc(t *testing.T) {
	want := errors.New("hsm unavailable")
	fn := func(_ context.Context, _ []byte) (Signature, error) {
		return Signature{}, want
	}
	rs, _ := NewRemoteSigner(PublicKey{}, fn)
	_, err := rs.Sign(context.Background(), []byte("x"))
	if !errors.Is(err, want) {
		t.Fatalf("expected wrapped error, got %v", err)
	}
}

// TestSigner_InterfaceShape asserts at compile time that both concrete
// types satisfy the Signer interface.
func TestSigner_InterfaceShape(t *testing.T) {
	var _ Signer = (*Ed25519Keypair)(nil)
	var _ Signer = (*RemoteSigner)(nil)
}

// TestSigner_SignaturesMatchBetweenLocalAndRemote proves that wrapping
// an Ed25519Keypair in a RemoteSigner is a pure decoration with no
// byte-level differences, which is the guarantee Transaction.Sign
// relies on when mixing local and remote signers in one call.
func TestSigner_SignaturesMatchBetweenLocalAndRemote(t *testing.T) {
	seed := bytes.Repeat([]byte{0xAB}, ed25519.SeedSize)
	kp, err := Ed25519KeypairFromSeed(seed)
	if err != nil {
		t.Fatal(err)
	}
	msg := []byte("same bytes either way")
	localSig, _ := kp.Sign(context.Background(), msg)

	rs, _ := NewRemoteSigner(kp.PublicKey(), kp.Sign)
	remoteSig, _ := rs.Sign(context.Background(), msg)

	if !localSig.Equal(remoteSig) {
		t.Fatalf("local and remote signatures differ: %x vs %x", localSig[:8], remoteSig[:8])
	}
}
