# SendAndConfirmTransaction

`*rpc.Client.SendAndConfirmTransaction` is the high-level helper for
the common "build, send, wait for confirmation, refresh blockhash on
expiry" flow. It composes several typed RPC calls into a single
operation and handles the state machine most real applications
re-implement.

This API lives in `package rpc` (not `helpers/`): the helpers
package is intentionally pure-logic and has no RPC entry points.

## API

```go
type TransactionBuilder func(ctx context.Context, blockhash solana.Hash) (*solana.Transaction, error)

func (c *Client) SendAndConfirmTransaction(
    ctx context.Context,
    build TransactionBuilder,
    opts ...SendAndConfirmOption,
) (solana.Signature, error)
```

The builder is a **callback** that constructs and signs a fresh
transaction for a given blockhash. The client calls it once per send
attempt, so when the blockhash expires and the helper refreshes, your
builder is invoked again with the new blockhash and re-signs the same
instructions.

## Why a builder callback?

A signed Solana transaction is pinned to a specific blockhash. If
the blockhash expires before the transaction lands, the transaction
is **dead forever**: you cannot update its blockhash without
re-signing, and you cannot re-sign without the private key. Either
the helper resigns automatically, or the caller does.

By taking a builder, `SendAndConfirmTransaction` can refresh the
blockhash and re-sign without ever holding the private key itself.
The caller's builder captures the signers in a closure; the helper
only sees a `*solana.Transaction`.

## Options

All options live in `package rpc`:

| Option | Default | Meaning |
|---|---|---|
| `rpc.WithSendCommitment(c)` | `CommitmentConfirmed` | Commitment required to consider confirmed |
| `rpc.WithConfirmTimeout(d)` | 60 s | Cap total wait time |
| `rpc.WithPollInterval(d)` | 2 s | Delay between `getSignatureStatuses` polls |
| `rpc.WithMaxBlockhashRetries(n)` | 3 | Max refresh+rebuild+resend cycles |
| `rpc.WithSendSkipPreflight(b)` | false | Skip server-side preflight simulate |

## Example

```go
import (
    "context"
    "log"
    "time"

    "github.com/MevYu/solana-go"
    "github.com/MevYu/solana-go/jsonrpc"
    "github.com/MevYu/solana-go/programs/system"
    "github.com/MevYu/solana-go/rpc"
)

ctx := context.Background()
c := rpc.NewClient("https://api.mainnet-beta.solana.com", jsonrpc.Config{})

sig, err := c.SendAndConfirmTransaction(ctx,
    func(ctx context.Context, blockhash solana.Hash) (*solana.Transaction, error) {
        msg, err := solana.NewMessage(
            payer.PublicKey(),
            []solana.Instruction{
                system.NewTransfer(payer.PublicKey(), recipient, 1_000_000),
            },
            blockhash,
        )
        if err != nil {
            return nil, err
        }
        tx := solana.NewTransaction(*msg)
        if err := tx.Sign(ctx, payer); err != nil {
            return nil, err
        }
        return tx, nil
    },
    rpc.WithSendCommitment(solana.CommitmentConfirmed),
    rpc.WithConfirmTimeout(45*time.Second),
)
if err != nil {
    return err // error is typed where possible (see below)
}
log.Println("landed:", sig)
```

## Lifecycle

Under the hood, each "try" does:

1. `GetLatestBlockhash` — fetch the newest blockhash.
2. Build + sign via the callback, passing `blockhash`.
3. `SendTransaction` — broadcast.
4. Poll `GetSignatureStatuses` every `pollInterval`:
   - **success** → return the signature.
   - **transaction error** → decode via
     `rpc.DecodeTransactionError`, return the typed error.
   - **blockhash age-out** (server's current block height passed
     `LastValidBlockHeight`) → abort the current try, refresh, retry.

Retries happen up to `WithMaxBlockhashRetries` + 1 times total.
Beyond that, the helper returns the last error wrapped as
`exhausted blockhash retries: ...`.

## Typed errors

When the transaction fails on chain, the returned error is the typed
decoded version:

```go
var ie *rpc.InstructionError
if errors.As(err, &ie) {
    fmt.Printf("instruction %d failed: %s\n", ie.Index, ie.Kind)
    if ie.Kind == "Custom" {
        fmt.Printf("  program error 0x%x\n", ie.CustomErrorCode)
    }
}
```

Transport errors, preflight errors, and expiry-exhausted errors are
returned as wrapped plain errors. See
[Error Handling](Error-Handling) for the full classification.

## `SendAndConfirmSignedTransaction`

```go
func (c *Client) SendAndConfirmSignedTransaction(
    ctx context.Context,
    tx *solana.Transaction,
    opts ...SendAndConfirmOption,
) (solana.Signature, error)
```

Use this when the caller already has a fully signed transaction and
wants a simple fire-and-wait flow. Unlike `SendAndConfirmTransaction`,
it performs exactly **one send attempt** and **cannot refresh the
blockhash** on expiry — if the blockhash ages out during the wait,
the helper returns a blockhash-expiry error immediately.

Prefer the builder form for long-running flows or whenever the
blockhash might expire (the 60-second default wait is close to the
~90-second blockhash lifetime on mainnet).

## Related

- [Send Transaction](Send-Transaction) — the low-level call.
- [Simulate With Decoded Errors](Simulate-With-Decoded-Errors) —
  simulate first, then send (recommended).
- [Priority Fee Estimation](Priority-Fee-Estimation) — pair with
  this helper for a complete production flow.
