# Signature Methods

Two signature-shaped queries live in
`rpc/methods_signatures.go`: point-in-time **status** lookup and
**history** search.

## `GetSignatureStatuses`

```go
func (c *rpc.Client) GetSignatureStatuses(
    ctx context.Context,
    sigs []solana.Signature,
    opts ...rpc.CallOption,
) (*rpc.GetSignatureStatusesResult, error)

type GetSignatureStatusesResult struct {
    Slot     uint64
    Statuses []*SignatureStatus // same length as input; nil entries unknown
}

type SignatureStatus struct {
    Slot               uint64
    Confirmations      *uint64 // nil once finalized
    Err                any     // nil on success
    ConfirmationStatus string  // "processed" | "confirmed" | "finalized"
}
```

Fetches the status of multiple transaction signatures in a
single round trip. The result slice has the same length as the
input; each entry is `nil` if the matching signature is not
known to the node (either it never landed, or it has aged out
of the node's recent cache).

### Honoured options

- `WithSearchTransactionHistory` — enables the transaction
  history search on the server. Signatures that have aged out
  of the recent cache can only be located this way.

### `Err` decoding

`SignatureStatus.Err` is the raw `any` Solana returns — either
`nil` for success or an object describing the failure. The
helper `rpc.DecodeTransactionError` parses it into a typed
`*TransactionError` or `*InstructionError`:

```go
import "github.com/MevYu/solana-go/helpers"

if s := res.Statuses[0]; s != nil && s.Err != nil {
    err := rpc.DecodeTransactionError(s.Err)

    var ie *rpc.InstructionError
    if errors.As(err, &ie) {
        fmt.Printf("instruction %d failed: %s\n", ie.Index, ie.Kind)
    }
}
```

See [Simulate With Decoded Errors](Simulate-With-Decoded-Errors)
for details.

### Example: polling for confirmation

```go
func waitForSignature(ctx context.Context, c *rpc.Client, sig solana.Signature) error {
    for {
        res, err := c.GetSignatureStatuses(ctx, []solana.Signature{sig},
            rpc.WithSearchTransactionHistory(true),
        )
        if err != nil {
            return err
        }
        s := res.Statuses[0]
        if s != nil {
            if s.Err != nil {
                return rpc.DecodeTransactionError(s.Err)
            }
            if s.ConfirmationStatus == "confirmed" || s.ConfirmationStatus == "finalized" {
                return nil
            }
        }
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(2 * time.Second):
        }
    }
}
```

Prefer [SendAndConfirmTransaction](SendAndConfirmTransaction)
for production flows — it includes this loop plus blockhash
refresh and decoded error reporting.

## `GetSignaturesForAddress`

```go
func (c *Client) GetSignaturesForAddress(
    ctx context.Context,
    addr PublicKey,
    opts ...CallOption,
) ([]*ConfirmedSignatureForAddress, error)

type ConfirmedSignatureForAddress struct {
    Signature          string
    Slot               uint64
    Err                any
    Memo               *string
    BlockTime          *int64
    ConfirmationStatus string
}
```

Fetches the transaction signatures that touched `addr`,
ordered newest first. Use the pagination options to walk
history:

- **`WithLimit(n)`** — cap the result count. Server default is
  1000.
- **`WithBefore(sig)`** — return signatures strictly older than
  `sig`.
- **`WithUntil(sig)`** — return signatures strictly newer than
  `sig`.

### Example: paginate an address's history

```go
var all []*c.ConfirmedSignatureForAddress
var before solana.Signature

for {
    opts := []rpc.CallOption{rpc.WithLimit(1000)}
    if !before.IsZero() {
        opts = append(opts, rpc.WithBefore(before))
    }

    page, err := c.GetSignaturesForAddress(ctx, addr, opts...)
    if err != nil {
        return err
    }
    if len(page) == 0 {
        break
    }
    all = append(all, page...)

    last, _ := solana.SignatureFromBase58(page[len(page)-1].Signature)
    before = last
}
```

Honoured options: `WithCommitment`, `WithMinContextSlot`,
`WithLimit`, `WithBefore`, `WithUntil`.

## Related

- [Send Transaction](Send-Transaction) — produces a signature
  to poll on.
- [SendAndConfirmTransaction](SendAndConfirmTransaction) — the
  helper that wraps the polling loop.
- [Advanced Subscriptions](Advanced-Subscriptions) — `SignatureSubscribe`
  for a push-based confirmation notification.
