# Fee Methods

Airdrop, fee pricing, and priority-fee observation live in
`rpc/methods_fees.go`.

## `RequestAirdrop`

```go
func (c *Client) RequestAirdrop(
    ctx context.Context,
    pubkey solana.PublicKey,
    lamports uint64,
    cfg ...rpc.CommitmentCfg,
) (solana.Signature, error)
```

Asks the cluster to deposit `lamports` into `pubkey`. **Only
valid on devnet and testnet.** The mainnet faucet is
rate-limited and not accessible from all endpoints.

On success, the returned `Signature` is the airdrop
transaction's signature; poll it via `GetSignatureStatuses`
to wait for confirmation.

Cfg: `rpc.CommitmentCfg{Commitment: …}`.

```go
sig, err := c.RequestAirdrop(ctx, wallet, 1_000_000_000)
if err != nil {
    return err
}
// wait for it to land
```

## `GetFeeForMessage`

```go
func (c *Client) GetFeeForMessage(
    ctx context.Context,
    msg *solana.Message,
    cfg ...rpc.CommitmentWithMinSlotCfg,
) (*GetFeeForMessageResult, error)

type GetFeeForMessageResult struct {
    Slot uint64
    Fee  *uint64 // nil if the blockhash expired
}
```

Computes the fee the cluster would charge for a transaction
committing to `msg`. The method takes a typed `*Message` and
handles the marshal + base64 encoding inline, so callers do
not have to.

**Blockhash expiry is not an error.** If the message's recent
blockhash has expired, the server returns `Fee == nil`, which
this method surfaces by leaving `Fee` as a nil pointer. Treat
this as a signal to refresh and retry:

```go
res, err := c.GetFeeForMessage(ctx, msg)
if err != nil {
    return err
}
if res.Fee == nil {
    // refresh blockhash, rebuild, re-query
}
```

Cfg: `rpc.CommitmentWithMinSlotCfg{Commitment, MinContextSlot}`.

## `GetRecentPrioritizationFees`

```go
func (c *Client) GetRecentPrioritizationFees(
    ctx context.Context,
    addresses []solana.PublicKey,
) ([]PrioritizationFee, error)

type PrioritizationFee struct {
    Slot              uint64
    PrioritizationFee uint64 // micro-lamports per CU
}
```

Returns the prioritization fees observed in recent slots. If
`addresses` is non-empty, the server restricts the result to
slots containing at least one transaction that mentioned any
of those addresses. This is useful for fee estimation scoped
to a specific hot account (for example, a DEX pool).

The fees are in **micro-lamports per compute unit** (1 lamport
= 1,000,000 micro-lamports), matching the units
`ComputeBudget.SetComputeUnitPrice` expects.

```go
fees, _ := c.GetRecentPrioritizationFees(ctx, nil)
for _, f := range fees {
    fmt.Printf("slot %d: %d µL/CU\n", f.Slot, f.PrioritizationFee)
}
```

### Percentile statistics

Raw observations are almost never what you want. Pair the RPC
call with the pure-logic stats helper in `helpers`:

```go
import "github.com/MevYu/solana-go/helpers"

fees, _ := c.GetRecentPrioritizationFees(ctx, nil)
stats := helpers.PriorityFeeStatsFromFees(fees)
fmt.Printf("p50=%d p75=%d p95=%d max=%d over %d samples\n",
    stats.P50, stats.P75, stats.P95, stats.Max, stats.Samples)
```

See [Priority Fee Estimation](Priority-Fee-Estimation) for the
percentile interpretation and a recipe for building a budgeted
transaction around the result.

No cfg is accepted on this method.

## Related

- [Compute Budget](Compute-Budget-Program) — the program that
  applies a chosen priority fee to your transaction.
- [Priority Fee Estimation](Priority-Fee-Estimation) — the
  helper that consumes this method's output.
