# Priority Fee Estimation

`helpers.PriorityFeeStatsFromFees` computes ordered-percentile
statistics over the recent prioritization fees the cluster has
observed, so you can pick a per-CU price that is high enough to land
but not wastefully high.

The pattern is two steps:

1. Fetch the fee window with `c.GetRecentPrioritizationFees(ctx, addrs)`.
2. Pass the result to `helpers.PriorityFeeStatsFromFees(...)`.

`helpers/` is intentionally pure-logic and does not perform any RPC
calls of its own; the split lets callers cache the fee window across
many transactions, query multiple endpoints, or compute custom
scoping without re-implementing the percentile math.

```go
import (
    "github.com/MevYu/solana-go/rpc"
    "github.com/MevYu/solana-go/helpers"
)
```

## API

```go
func PriorityFeeStatsFromFees(fees []rpc.PrioritizationFee) *PriorityFeeStats

type PriorityFeeStats struct {
    P50     uint64 // median percentile
    P75     uint64
    P95     uint64
    Max     uint64
    Samples int    // slot-window size the stats are computed over
}
```

Fees are in **micro-lamports per compute unit**, the same units that
`computebudget.NewSetComputeUnitPrice` expects.

### Address scoping

When you call `c.GetRecentPrioritizationFees(ctx, addrs)` with a
non-empty `addrs`, the cluster scopes returned fees to slots in which
at least one transaction mentioned any of those addresses. This
matters for hot programs: the cluster-wide median may be far below
the median needed to land in slots where a specific AMM or lending
program is contested.

```go
fees, err := c.GetRecentPrioritizationFees(ctx, []solana.PublicKey{dexProgram})
if err != nil { return err }
stats := helpers.PriorityFeeStatsFromFees(fees)
```

## Interpretation

The SDK uses a **floor-based nearest-rank** percentile:

```
idx = floor(p/100 * N) - 1, clamped to [0, N-1]
```

For `N = 10` sorted-ascending samples `[1000, 2000, …, 10000]`:

- `P50` → sorted[4] = 5000
- `P75` → sorted[6] = 7000
- `P95` → sorted[8] = 9000

This matches the sorted quartile that is strictly exceeded by the
given fraction of samples, which is the interpretation most
priority-fee libraries converge on.

For very small N (1–4), the algorithm pins all percentiles toward
the lower samples; treat the result as advisory in those cases.

## Zero-sample handling

When the cluster reports **no recent fees at all**, the helper
returns a zero-valued `PriorityFeeStats` — not an error. Callers
should treat that as a signal to use a conservative fallback rather
than as a failure:

```go
fees, err := c.GetRecentPrioritizationFees(ctx, nil)
if err != nil { return err }
stats := helpers.PriorityFeeStatsFromFees(fees)
price := stats.P75
if stats.Samples == 0 {
    price = 5_000 // sensible cluster-default fallback
}
```

## Recommended values

Pick the percentile based on how contested you expect block space
to be:

| Percentile | When to use |
|---|---|
| **P50** | Quiet slots, testnet, background jobs, no time pressure |
| **P75** | Default for interactive operations |
| **P95** | Hot mempool, arbitrage, liquidations, high-value settles |
| **Max** | Panic button: the highest fee observed in the window |

Above the 95th percentile you start competing with arb bots; below
the 50th you are betting on the cluster being quiet.

## Full example

```go
import (
    "github.com/MevYu/solana-go"
    computebudget "github.com/MevYu/solana-go/programs/compute-budget"
    "github.com/MevYu/solana-go/programs/system"
    "github.com/MevYu/solana-go/helpers"
)

// Fetch and summarise, scoped to the program the transaction touches.
fees, err := c.GetRecentPrioritizationFees(ctx,
    []solana.PublicKey{dexProgram},
)
if err != nil { return err }
stats := helpers.PriorityFeeStatsFromFees(fees)

price := stats.P75
if stats.Samples == 0 {
    price = 5_000
}

instructions := []solana.Instruction{
    computebudget.NewSetComputeUnitLimit(80_000),
    computebudget.NewSetComputeUnitPrice(price),
    // ... your program instructions ...
}

msg, _ := solana.NewMessage(payer.PublicKey(), instructions, recentBlockhash)
```

## Related

- [Fee Methods](Fee-Methods) — the underlying
  `GetRecentPrioritizationFees` call.
- [Compute Budget](Compute-Budget-Program) — the program that
  applies the chosen price.
