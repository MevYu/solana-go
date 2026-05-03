# Signers

A `Signer` is anything that can produce an Ed25519 signature for
a known public key. It is the contract `Transaction.Sign`
consumes.

```go
type Signer interface {
    PublicKey() PublicKey
    Sign(ctx context.Context, message []byte) (Signature, error)
}
```

The context is threaded through **so that remote signers** —
hardware wallets, cloud HSMs, networked signing services — can
enforce deadlines and cancellations. Local in-memory signers
ignore it, but the method signature stays the same so
`Transaction.Sign` never has to branch on the signer kind.

## Built-in: `Ed25519Keypair`

A local, in-memory signer backed by `crypto/ed25519`. This is
the fastest path when the private key is already in process.

### Construction

```go
// Random fresh keypair
kp, err := solana.NewEd25519Keypair()

// Deterministic from a 32-byte seed
kp, err := solana.Ed25519KeypairFromSeed(seed)

// From an already-expanded 64-byte private key
kp, err := solana.Ed25519KeypairFromPrivateKey(priv)
```

`Ed25519KeypairFromPrivateKey` copies its input so the caller may
zero `priv` after the call.

### Using

```go
fmt.Println(kp.PublicKey())

// Low-level signing (you rarely call this directly)
sig, err := kp.Sign(ctx, message)
```

Most code passes the keypair straight to `Transaction.Sign`:

```go
_ = tx.Sign(ctx, kp)
```

### Private-key export

```go
raw := kp.PrivateKey() // 64-byte expanded form, fresh copy
// zero raw when done
```

The returned slice is a defensive copy. `Ed25519Keypair` never
exposes its internal buffer.

## Built-in: `RemoteSigner`

For anything whose key material is off-host — Ledger, YubiHSM,
AWS KMS, GCP KMS, an in-house signing service — wrap the transport
in a `RemoteSigner`:

```go
type RemoteSignFunc func(ctx context.Context, message []byte) (Signature, error)
```

```go
remote, err := solana.NewRemoteSigner(
    walletPublicKey,
    func(ctx context.Context, message []byte) (solana.Signature, error) {
        return callMyHSM(ctx, message)
    },
)
if err != nil {
    return err
}

_ = tx.Sign(ctx, remote)
```

`NewRemoteSigner` rejects a nil sign function at construction so
a zero `RemoteSigner` is never usable.

### Mixing local and remote signers

Because both types implement `Signer`, a single `Sign` call can
pass a mix:

```go
_ = tx.Sign(ctx, localPayer, remote1, remote2)
```

`Transaction.Sign` computes the message bytes once and hands
them to every signer, so the remote calls are independent and
can run in series (stdlib `ed25519.Verify` does not permit
parallelism in this library's hot path, but `RemoteSignFunc`
implementations are free to batch internally).

### Ordering does not matter

Signers can be passed in any order. `Transaction.Sign` looks up
each signer's slot by its public key in an O(1) map built once
per call, so slot assignment is by pubkey, not by argument
position. See `TestTransaction_Sign_MultipleSignersInAnyOrder`
in `transaction_test.go`.

## Writing your own signer

Any type that implements `PublicKey() PublicKey` and
`Sign(ctx, []byte) (Signature, error)` is a `Signer`. The contract
is:

- `PublicKey()` must be pure and return a stable value for the
  lifetime of the signer.
- `Sign` must produce a valid Ed25519 signature over `message`
  with the private key corresponding to `PublicKey()`.
- `Sign` should respect `ctx`. Remote signers in particular must
  abort on cancellation so stuck HSM calls don't wedge a
  transaction retry loop.

## Related

- [Transactions](Transactions) — how signers slot into the
  signing flow.
- [SendAndConfirmTransaction](SendAndConfirmTransaction) — the
  helper that uses signers inside a builder callback so
  blockhash refresh can re-sign automatically.
