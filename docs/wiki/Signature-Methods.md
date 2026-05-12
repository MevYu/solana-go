# Signature Methods

Two signature-shaped queries live in
`rpc/methods_signatures.go`: point-in-time **status** lookup and
**history** search.

## `GetSignatureStatuses`

```go
func (c *rpc.Client) GetSignatureStatuses(
    ctx context.Context,
    sigs []solana.Signature,
    cfg ...rpc.SignatureStatusesCfg,
) (*rpc.GetSignatureStatusesResult, error)

type GetSignatureStatusesResult struct {
    Slot     uint64
    Statuses []*solana.SignatureStatus // same length as input; nil entries unknown
}

// solana.SignatureStatus
type SignatureStatus struct {
    Slot               uint64
    Confirmations      *uint64         // nil once finalized
    Err                json.RawMessage // empty / "null" on success
    ConfirmationStatus string          // "processed" | "confirmed" | "finalized"
}
```

Fetches the status of multiple transaction signatures in a
single round trip. The result slice has the same length as the
input; each entry is `nil` if the matching signature is not
known to the node (either it never landed, or it has aged out
of the node's recent cache).

### Cfg fields (`rpc.SignatureStatusesCfg`)

- `SearchTransactionHistory *bool` — enables the transaction
  history search on the server. Signatures that have aged out
  of the recent cache can only be located this way.

### `Err` decoding

`SignatureStatus.Err` is the raw JSON Solana returns — empty (or
`"null"`) for success, or the err-object describing the failure.
`rpc.DecodeTransactionError` accepts `json.RawMessage` directly,
returns `nil` for the success cases, and otherwise parses into a
typed `*TransactionError` or `*InstructionError`:

```go
import "github.com/MevYu/solana-go/rpc"

if s := res.Statuses[0]; s != nil {
    if err := rpc.DecodeTransactionError(s.Err); err != nil {
        var ie *rpc.InstructionError
        if errors.As(err, &ie) {
            fmt.Printf("instruction %d failed: %s\n", ie.Index, ie.Kind)
        }
    }
}
```

See [Simulate With Decoded Errors](Simulate-With-Decoded-Errors)
for details.

### Example: polling for confirmation

```go
func waitForSignature(ctx context.Context, c *rpc.Client, sig solana.Signature) error {
    for {
        searchHistory := true
        res, err := c.GetSignatureStatuses(ctx, []solana.Signature{sig},
            rpc.SignatureStatusesCfg{SearchTransactionHistory: &searchHistory},
        )
        if err != nil {
            return err
        }
        s := res.Statuses[0]
        if s != nil {
            if txErr := rpc.DecodeTransactionError(s.Err); txErr != nil {
                return txErr
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
    addr solana.PublicKey,
    cfg ...rpc.SignaturesForAddressCfg,
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
ordered newest first. Use the cfg fields to walk history:

- **`Limit *int`** — cap the result count. Server default is 1000.
- **`Before string`** — return signatures strictly older than `Before`.
- **`Until string`** — return signatures strictly newer than `Until`.
- **`Commitment`**, **`MinContextSlot`**.

### Example: paginate an address's history

```go
var all []*rpc.ConfirmedSignatureForAddress
limit := 1000
var before string

for {
    cfg := rpc.SignaturesForAddressCfg{Limit: &limit}
    if before != "" {
        cfg.Before = before
    }

    page, err := c.GetSignaturesForAddress(ctx, addr, cfg)
    if err != nil {
        return err
    }
    if len(page) == 0 {
        break
    }
    all = append(all, page...)
    before = page[len(page)-1].Signature
}
```

## Related

- [Send Transaction](Send-Transaction) — produces a signature
  to poll on.
- [SendAndConfirmTransaction](SendAndConfirmTransaction) — the
  helper that wraps the polling loop.
- [Advanced Subscriptions](Advanced-Subscriptions) — `SignatureSubscribe`
  for a push-based confirmation notification.
