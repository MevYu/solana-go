# Transactions

A **Transaction** is the signed pair of a [Message](Messages) and
a positional array of Ed25519 signatures. It is the unit every
Solana RPC `sendTransaction` call expects.

```go
type Transaction struct {
    Signatures []Signature
    Message    Message
}
```

## Wire format

```
[shortvec signatures] [64-byte sig × N] [serialised message]
```

The signature count is compact-u16. Each signature is exactly
`SignatureSize` (64) bytes. The message body follows immediately;
its own `(*Message).UnmarshalBinary` enforces end-of-input, so a
transaction with trailing garbage after the message is rejected.

## Constructing

```go
tx := solana.NewTransaction(message)
```

`NewTransaction` pre-allocates `Signatures` with zero-filled
placeholder entries — one per required signer, matching
`Header.NumRequiredSignatures`. You then fill those slots by
calling `Sign`.

## Signing

```go
func (tx *Transaction) Sign(ctx context.Context, signers ...Signer) error
```

Each `Signer` must own a key that appears in the first
`NumRequiredSignatures` entries of `Message.AccountKeys`.
Signature slots are filled **in place at the slot matching each
signer's position**. The lookup from public key to slot is **O(1)**:
the method builds one `map[PublicKey]int` per call and reuses it
for every signer.

### Incremental signing

`Sign` is safe to call more than once. Only the slots matching
the supplied signers are touched; other slots retain their
previous value. This supports multi-party signing flows:

```go
tx := solana.NewTransaction(*msg)

// Step 1: our payer signs slot 0.
_ = tx.Sign(ctx, payer)

// Serialise to disk / wire / Slack.
partial, _ := tx.Marshal()

// Step 2 (later, elsewhere): another party signs slot 1.
roundTripped := &solana.Transaction{}
_ = roundTripped.UnmarshalBinary(partial)
_ = roundTripped.Sign(ctx, other)
```

### Validation before mutation

`Sign` validates every signer against `AccountKeys` **before**
touching any signature slot. If any signer is not a required
signer, `Sign` returns an error naming the offending public key
and `tx.Signatures` is untouched. This means a failed multi-party
`Sign` call never leaves the transaction in a partially rewritten
state.

### Marshal once, sign many

The message bytes are marshaled exactly once and reused across
every signer, so adding more signers to a single call does not
scale the marshal cost.

## Sign errors

Common failure modes:

| Condition | Error |
|---|---|
| `NumRequiredSignatures == 0` | `message requires zero signatures` |
| `len(AccountKeys) < NumRequiredSignatures` | `message has N account keys but header requires M signatures` |
| Signer's pubkey not in required slots | `signer <pk> is not a required signer` |
| Signer returned an error (remote signer I/O) | wrapped `signer <pk>: <err>` |

All errors are plain wrapped errors without panics.

## Marshal / Unmarshal

```go
data, err := tx.Marshal()

out := &solana.Transaction{}
err = out.UnmarshalBinary(data)
```

- `Marshal` refuses to emit a transaction with more than `0xFFFF`
  signatures (protocol hard limit).
- `(*Transaction).UnmarshalBinary` rejects empty input, a missing
  message body, and any trailing bytes after the message. It
  implements `encoding.BinaryUnmarshaler`, paired with `Marshal` /
  `MarshalBinary`.
- The decoded `Transaction` does **not** alias the input buffer;
  signatures are copied out and the embedded `Message` is fully
  materialised by `(*Message).UnmarshalBinary` internally.

## Verify signatures locally

```go
if err := tx.VerifySignatures(); err != nil {
    // at least one non-zero slot failed Ed25519 verification
}
```

`VerifySignatures` is a **best-effort local sanity check**, not a
substitute for the validator's full rules:

- It re-serialises the message and runs stdlib
  `ed25519.Verify` on every non-zero slot.
- A slot containing the all-zero `Signature` is treated as
  unsigned and skipped, which supports the incremental signing
  flow — you can verify the partial transaction after slot 0 is
  filled without error.
- Tampering with any message field (`RecentBlockhash`,
  instruction data, account keys, …) breaks the re-serialised
  bytes and the stored signatures stop verifying.

## Example: sign and serialise

```go
payer, _ := solana.NewEd25519Keypair()

msg, _ := solana.NewMessage(
    payer.PublicKey(),
    []solana.Instruction{system.NewTransfer(payer.PublicKey(), recipient, 1000)},
    recentBlockhash,
)

tx := solana.NewTransaction(*msg)
if err := tx.Sign(ctx, payer); err != nil {
    return err
}

wire, err := tx.Marshal()
if err != nil {
    return err
}
_ = wire // pass to c.SendRawTransaction or persist offline
```

## Related

- [Signers](Signers) — the interface `Sign` consumes.
- [Send Transaction](Send-Transaction) — broadcasting a signed
  transaction.
- [Messages](Messages) — the unsigned body.
