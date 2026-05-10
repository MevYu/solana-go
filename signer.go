package solana

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"

	"github.com/mr-tron/base58"
)

// Signer is the interface implemented by anything that can produce
// Ed25519 signatures for a known Solana public key. It is the
// contract Transaction.Sign consumes to fill in a transaction's
// signature slots.
//
// The context is threaded through so that remote signers (hardware
// wallets, cloud HSMs, networked signing services) can enforce
// deadlines and cancellations. Local in-memory signers may ignore
// it, but the signature of the interface is fixed either way so
// Transaction.Sign does not have to branch on the signer kind.
type Signer interface {
	// PublicKey returns the Solana public key that this signer holds
	// the corresponding Ed25519 private key for.
	PublicKey() PublicKey

	// Sign produces an Ed25519 signature over message. The returned
	// Signature is consumed directly by Transaction.Sign; callers
	// should treat the exact bytes as opaque.
	Sign(ctx context.Context, message []byte) (Signature, error)
}

// ---------------------------------------------------------------------
// Local signer: Ed25519Keypair
// ---------------------------------------------------------------------

// Ed25519Keypair is a local, in-memory Signer backed by an Ed25519
// private key from the standard library's crypto/ed25519.
//
// This is the fastest path for signing when the private key material
// is already in process memory. For hardware wallets or remote HSMs,
// use RemoteSigner instead.
type Ed25519Keypair struct {
	public  PublicKey
	private ed25519.PrivateKey
}

// NewEd25519Keypair returns a fresh keypair generated from the
// system's cryptographically secure random source.
func NewEd25519Keypair() (*Ed25519Keypair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("solana: ed25519 keypair: %w", err)
	}
	var pk PublicKey
	copy(pk[:], pub)
	return &Ed25519Keypair{private: priv, public: pk}, nil
}

// Ed25519KeypairFromSeed constructs a keypair deterministically from
// a 32-byte Ed25519 seed (the unexpanded form, as distinct from the
// 64-byte expanded private key).
func Ed25519KeypairFromSeed(seed []byte) (*Ed25519Keypair, error) {
	if len(seed) != ed25519.SeedSize {
		return nil, fmt.Errorf("solana: ed25519 seed: %w: got %d, want %d",
			ErrInvalidLength, len(seed), ed25519.SeedSize)
	}
	priv := ed25519.NewKeyFromSeed(seed)
	pub := priv.Public().(ed25519.PublicKey)
	var pk PublicKey
	copy(pk[:], pub)
	return &Ed25519Keypair{private: priv, public: pk}, nil
}

// Ed25519KeypairFromPrivateKey constructs a keypair from an already
// expanded 64-byte Ed25519 private key. The input bytes are copied;
// the caller may zero the input after the call returns.
func Ed25519KeypairFromPrivateKey(priv []byte) (*Ed25519Keypair, error) {
	if len(priv) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("solana: ed25519 private key: %w: got %d, want %d",
			ErrInvalidLength, len(priv), ed25519.PrivateKeySize)
	}
	p := ed25519.PrivateKey(append([]byte(nil), priv...))
	pub := p.Public().(ed25519.PublicKey)
	var pk PublicKey
	copy(pk[:], pub)
	return &Ed25519Keypair{private: p, public: pk}, nil
}

// Ed25519KeypairFromBase58Key creates an Ed25519Keypair from a base58-encoded
// Solana private key.
// It accepts the standard 64-byte base58 private key string commonly used in
// the Solana ecosystem.
func Ed25519KeypairFromBase58Key(key string) (*Ed25519Keypair, error) {
	// empty string
	if len(key) == 0 {
		return nil, fmt.Errorf("solana: ed25519 keypair: empty key")
	}
	b, err := base58.Decode(key)
	// if err
	if err != nil {
		return nil, fmt.Errorf("solana: ed25519 keypair: %w", err)
	}
	return Ed25519KeypairFromSeed(b)
}

// PublicKey implements Signer.
func (k *Ed25519Keypair) PublicKey() PublicKey { return k.public }

// Sign implements Signer. The context is intentionally ignored: local
// Ed25519 signing is a pure computation with no I/O, and attempting
// to cancel it would only throw away work.
func (k *Ed25519Keypair) Sign(_ context.Context, message []byte) (Signature, error) {
	var sig Signature
	raw := ed25519.Sign(k.private, message)
	copy(sig[:], raw)
	return sig, nil
}

// PrivateKey returns a defensive copy of the signer's Ed25519 private
// key bytes in the 64-byte expanded form. The caller should zero the
// returned slice when it is no longer needed.
func (k *Ed25519Keypair) PrivateKey() []byte {
	out := make([]byte, len(k.private))
	copy(out, k.private)
	return out
}

// ---------------------------------------------------------------------
// Remote signer: function-adapted Signer for off-host key material
// ---------------------------------------------------------------------

// RemoteSignFunc is a caller-supplied function that produces an
// Ed25519 signature for message. Implementations perform whatever
// network or device I/O is needed to reach the key material
// (hardware wallet, cloud HSM, networked signing service).
type RemoteSignFunc func(ctx context.Context, message []byte) (Signature, error)

// RemoteSigner adapts an arbitrary RemoteSignFunc into a Signer. It
// decouples transaction signing from the transport used to reach the
// signing key, so callers can integrate with Ledger, YubiHSM, AWS
// KMS, GCP KMS, or any in-house signing service without this library
// taking an opinion on the wire protocol.
//
// A zero RemoteSigner is not usable; always construct one with
// NewRemoteSigner. The wrapped function receives the context passed
// to Transaction.Sign, so callers can enforce per-signature deadlines.
type RemoteSigner struct {
	pk PublicKey
	fn RemoteSignFunc
}

// NewRemoteSigner binds a public key to a signing function. The
// function must not be nil.
func NewRemoteSigner(pk PublicKey, fn RemoteSignFunc) (*RemoteSigner, error) {
	if fn == nil {
		return nil, fmt.Errorf("solana: remote signer: nil sign function")
	}
	return &RemoteSigner{pk: pk, fn: fn}, nil
}

// PublicKey implements Signer.
func (r *RemoteSigner) PublicKey() PublicKey { return r.pk }

// Sign implements Signer by delegating to the wrapped RemoteSignFunc.
func (r *RemoteSigner) Sign(ctx context.Context, message []byte) (Signature, error) {
	return r.fn(ctx, message)
}
