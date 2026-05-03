# Simulate Transaction

`simulateTransaction` runs a transaction against the cluster's
current state **without** committing anything on chain. It is
the canonical preflight for:

- Compute-unit measurement (to tune `SetComputeUnitLimit`).
- Log inspection and debugging.
- Post-execution account state inspection (optional on the
  server, passed via the `accounts` param).

```go
func (c *rpc.Client) SimulateTransaction(
    ctx context.Context,
    tx *solana.Transaction,
    opts ...rpc.CallOption,
) (*rpc.SimulateResult, error)
```

## Result

```go
type SimulateResult struct {
    Slot          uint64
    Err           any                       // transaction-level error (nil on success)
    Logs          []string                  // program log messages
    UnitsConsumed *uint64                   // CUs consumed, if reported
    Accounts      []*AccountInfo            // post-simulation account state
    ReturnData    *SimulationReturnData     // last instruction's return data
}

type SimulationReturnData struct {
    ProgramID PublicKey
    Data      AccountData // [value, encoding]; call .Bytes() to decode
}
```

The `Err` field is raw `any`. Feed it to
`rpc.DecodeTransactionError` to get a typed
`*TransactionError` / `*InstructionError`, or use the
higher-level [SimulateTransactionDecoded](Simulate-With-Decoded-Errors)
helper that does the decoding for you.

## Honoured options

| Option | Effect |
|---|---|
| `WithCommitment` | Commitment level to simulate against |
| `WithSigVerify(true)` | Enable on-server signature verification |
| `WithReplaceRecentBlockhash(true)` | Replace the blockhash with a fresh one before simulating |
| `WithMinContextSlot` | Require a minimum node slot |

**`WithSigVerify` and `WithReplaceRecentBlockhash` are mutually
exclusive at the server** — Solana rejects a request carrying
both. The Go SDK passes them through as-is; pick one.

## Signing requirements

- By default, the transaction must be signed.
- With `WithReplaceRecentBlockhash(true)`, the server re-signs
  with a dummy key and the input transaction's signatures are
  ignored. This is useful for simulating a transaction whose
  blockhash has already expired.

## Example: measure compute units

```go
sim, err := c.SimulateTransaction(ctx, tx)
if err != nil {
    return err
}
if sim.Err != nil {
    return fmt.Errorf("simulate failed: %v", sim.Err)
}
if sim.UnitsConsumed != nil {
    fmt.Printf("consumed %d CUs\n", *sim.UnitsConsumed)
}
for _, line := range sim.Logs {
    fmt.Println(line)
}
```

The measured CU count is a good input to
`computebudget.NewSetComputeUnitLimit` — set a limit 10–20%
above the measured count to absorb variability without paying
for the full 200k default.

## Example: inspect return data

Programs can emit up to 1024 bytes of return data via the
`sol_set_return_data` syscall. The last instruction's return
data, if any, is exposed on `SimulateResult`:

```go
sim, _ := c.SimulateTransaction(ctx, tx)
if sim.ReturnData != nil {
    raw, _ := sim.ReturnData.Data.Bytes()
    fmt.Printf("%s returned %x\n", sim.ReturnData.ProgramID, raw)
}
```

## Decoded flow (preferred)

```go
import (
    "errors"
    "github.com/MevYu/solana-go/helpers"
)

sim, err := c.SimulateTransaction(ctx, tx)
if err != nil {
    return err // transport error
}
if sim.Err != nil {
    decoded := rpc.DecodeTransactionError(sim.Err)
    var ie *rpc.InstructionError
    if errors.As(decoded, &ie) {
        fmt.Printf("instruction %d: %s\n", ie.Index, ie.Kind)
    }
    return decoded
}
fmt.Printf("%d log lines\n", len(sim.Logs))
```

See [Simulate With Decoded Errors](Simulate-With-Decoded-Errors)
for the full story.

## Related

- [Send Transaction](Send-Transaction) — once simulation is
  clean.
- [Compute Budget](Compute-Budget-Program) — to act on the CU
  measurement.
- [Error Handling](Error-Handling) — for the classifier and
  typed error shapes.
