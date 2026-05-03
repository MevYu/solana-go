# Compute Budget Program

The **ComputeBudget** program is the metadata-only program
that tunes a transaction's compute-unit limit, its per-CU
priority fee, and (rarely) its heap size. Its address is
`ComputeBudget111111111111111111111111111111`.

ComputeBudget instructions take **no account inputs** —
they just carry data bytes. Include one or more at the start
of your instruction list to control the containing
transaction's resource budget.

Typed instruction builders live in
`github.com/MevYu/solana-go/programs/compute-budget` (Go package
name `computebudget`).

```go
import computebudget "github.com/MevYu/solana-go/programs/compute-budget"
```

## Program id

```go
var computebudget.ProgramID = solana.MustPublicKey(
    "ComputeBudget111111111111111111111111111111")
```

## Builders

### `NewSetComputeUnitLimit`

```go
func NewSetComputeUnitLimit(units uint32) solana.Instruction
```

Sets the compute-unit limit for the containing transaction.
Transactions that omit this instruction get Solana's default
limit (200,000 CUs as of late 2024). Specifying a tighter
limit:

- **Reduces the priority fee** you pay at a given per-CU
  price (the fee is `limit × price`).
- **Makes your transaction more competitive** for block space
  against cheaper transactions.

Measure the real CU cost via
[SimulateTransaction](Simulate-Transaction) first, then set
a limit 10–20% above the measured value to absorb variability.

### `NewSetComputeUnitPrice`

```go
func NewSetComputeUnitPrice(microLamports uint64) solana.Instruction
```

Sets the price the caller is willing to pay per compute unit,
in **micro-lamports** (1 lamport = 1,000,000 micro-lamports).
This is Solana's equivalent of an EVM priority fee.

Pair `c.GetRecentPrioritizationFees` with
[`helpers.PriorityFeeStatsFromFees`](Priority-Fee-Estimation) to
pick a reasonable value from recent cluster observations rather
than hardcoding.

### `NewRequestHeapFrame`

```go
func NewRequestHeapFrame(bytes uint32) solana.Instruction
```

Requests a larger program heap. `bytes` must be a multiple of
1024 and is capped at 256 KiB by the runtime. **Most programs
do not need this** — use it only when a program you call
specifically documents it.

## Wire format

ComputeBudget uses **single-byte tags** (unlike System's u32
tags):

| Tag | Instruction |
|---|---|
| 1 | RequestHeapFrame |
| 2 | SetComputeUnitLimit |
| 3 | SetComputeUnitPrice |

## Typical pattern

Put ComputeBudget instructions first in your instruction list:

```go
import (
    "github.com/MevYu/solana-go"
    computebudget "github.com/MevYu/solana-go/programs/compute-budget"
    "github.com/MevYu/solana-go/programs/system"
    "github.com/MevYu/solana-go/helpers"
)

// Estimate a priority fee from recent cluster data.
fees, _ := c.GetRecentPrioritizationFees(ctx, nil)
stats := helpers.PriorityFeeStatsFromFees(fees)

instructions := []solana.Instruction{
    computebudget.NewSetComputeUnitLimit(60_000),      // tight limit
    computebudget.NewSetComputeUnitPrice(stats.P75),    // 75th percentile fee
    system.NewTransfer(payer.PublicKey(), recipient, 1_000_000),
}

msg, _ := solana.NewMessage(payer.PublicKey(), instructions, recentBlockhash)
```

This pays at the 75th percentile of observed recent fees for
up to 60,000 CUs. The total priority fee is
`60_000 × P75 / 1_000_000` lamports.

## Why the order matters

ComputeBudget instructions must be processed before any
instruction whose cost they affect. In practice, always put
them first — the Solana runtime does not reorder instructions.

## Measuring CU cost

The loop is: build, simulate, measure, tighten, ship.

```go
tx := solana.NewTransaction(*msg)
_ = tx.Sign(ctx, payer)

sim, _ := c.SimulateTransaction(ctx, tx)
if sim.UnitsConsumed != nil {
    limit := uint32(*sim.UnitsConsumed + 10_000) // 10k headroom
    fmt.Printf("tighten limit to %d CUs\n", limit)
}
```

Feed `limit` into the next build's
`NewSetComputeUnitLimit` call. On a mature flow this usually
converges in one iteration.

## Related

- [Fee Methods](Fee-Methods) —
  `GetRecentPrioritizationFees`, the raw data source.
- [Priority Fee Estimation](Priority-Fee-Estimation) — the
  helper that computes percentiles.
- [Simulate Transaction](Simulate-Transaction) — how to
  measure CU cost for a specific transaction.
