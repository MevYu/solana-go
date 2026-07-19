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
    cfg ...rpc.SimulateTxCfg,
) (*rpc.SimulateResult, error)

type SimulateTxCfg struct {
    Commitment             solana.CommitmentLevel
    SigVerify              *bool
    ReplaceRecentBlockhash *bool
    MinContextSlot         *uint64
    Encoding               solana.Encoding
    InnerInstructions      *bool
    Accounts               *SimulateAccountsCfg
}

type SimulateAccountsCfg struct {
    Encoding  solana.Encoding
    Addresses []solana.PublicKey
}
```

## Result

```go
type SimulateResult struct {
    Slot                   uint64
    Accounts               []*solana.AccountInfo
    Err                    any // nil on success
    Fee                    *uint64
    PreBalances            []uint64
    PostBalances           []uint64
    InnerInstructions      []SimulationInnerInstruction
    PreTokenBalances       []solana.TokenBalance
    PostTokenBalances      []solana.TokenBalance
    LoadedAccountsDataSize *uint32
    LoadedAddresses        *solana.LoadedAddresses
    Logs                   []string
    ReplacementBlockhash   *solana.LatestBlockhash
    ReturnData             *SimulationReturnData
    UnitsConsumed          *uint64
}

type SimulationReturnData struct {
    ProgramID PublicKey
    Data      EncodedData // decoded bytes plus the source encoding
}
```

Feed `Err` to `rpc.DecodeTransactionError` to get a typed
`*TransactionError` / `*InstructionError`. A successful simulation leaves
`Err` as `nil`, preserving the usual `Err != nil` guard for existing callers.

## Cfg fields

| Field | Effect |
|---|---|
| `Commitment` | Commitment level to simulate against |
| `SigVerify` | Enable on-server signature verification |
| `ReplaceRecentBlockhash` | Replace the blockhash with a fresh one before simulating |
| `MinContextSlot` | Require a minimum node slot |
| `Encoding` | Wire encoding (defaults to base64) |
| `InnerInstructions` | Include parsed CPI instructions in the result |
| `Accounts` | Return post-simulation snapshots for selected addresses |

`Accounts.Encoding` defaults to `base64`; the supported account snapshot
encodings are `base64` and `base64+zstd`. Solana rejects `base58` for this
specific simulation option.

**`SigVerify` and `ReplaceRecentBlockhash` are mutually
exclusive**. The SDK rejects a request that enables both before making
an RPC call.

## Signing requirements

- Signatures are verified only when `SigVerify` is true. Unsigned
  placeholder signature slots are otherwise accepted for simulation.
- With `ReplaceRecentBlockhash` enabled, the RPC node substitutes a fresh
  blockhash. This is useful when the transaction's blockhash has expired.

## Example: measure compute units

```go
sim, err := c.SimulateTransaction(ctx, tx)
if err != nil {
    return err
}
if txErr := rpc.DecodeTransactionError(sim.Err); txErr != nil {
    return txErr
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
    raw := sim.ReturnData.Data.Bytes
    fmt.Printf("%s returned %x\n", sim.ReturnData.ProgramID, raw)
}
```

## Decoded flow (preferred)

```go
import (
    "errors"
    "github.com/MevYu/solana-go/rpc"
)

sim, err := c.SimulateTransaction(ctx, tx)
if err != nil {
    return err // transport error
}
if decoded := rpc.DecodeTransactionError(sim.Err); decoded != nil {
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
