# Simulate With Decoded Errors

`rpc.DecodeTransactionError` turns the raw JSON shape Solana
returns for transaction errors into typed Go errors you can
pattern-match on. Pair the decoder with `Client.SimulateTransaction`
(or `GetTransaction.Meta.Err`) directly.

## Typed error shapes

```go
type TransactionError struct {
    Kind string // e.g. "BlockhashNotFound", "AccountNotFound"
}

type InstructionError struct {
    Index           int    // 0-based instruction position
    Kind            string // e.g. "Custom", "InvalidArgument"
    CustomErrorCode uint32 // set when Kind == "Custom"
}
```

Both implement `error` with human-readable messages.

## `DecodeTransactionError`

```go
func DecodeTransactionError(raw any) error
```

Parses the server's raw `err` field. The input `raw` comes from
`simulateTransaction` or `getTransaction`'s `meta.err` and is one
of several untyped JSON shapes:

| Input | Result |
|---|---|
| `nil` | `nil` (success) |
| `"BlockhashNotFound"` (string) | `*TransactionError{Kind: "BlockhashNotFound"}` |
| `{"InstructionError": [0, "InvalidArgument"]}` | `*InstructionError{Index: 0, Kind: "InvalidArgument"}` |
| `{"InstructionError": [2, {"Custom": 42}]}` | `*InstructionError{Index: 2, Kind: "Custom", CustomErrorCode: 42}` |
| `{"InstructionError": [1, {"PrivilegeEscalation": null}]}` | `*InstructionError{Index: 1, Kind: "PrivilegeEscalation"}` |
| top-level map with one key | `*TransactionError{Kind: <key>}` |
| anything else | a plain `fmt.Errorf("unrecognized transaction error shape: ...")` |

Unknown shapes are **not silently discarded** — you get an error
you can log verbatim for a follow-up to the library.

The integer-type switch covers `float64`, `int`, `int32`, `int64`,
`uint32`, and `uint64`, so the decoder works against any JSON
codec (stdlib `encoding/json` produces `float64`; `goccy/go-json`
preserves integer types).

## End-to-end pattern

```go
import (
    "errors"
    "github.com/MevYu/solana-go/rpc"
)

sim, err := c.SimulateTransaction(ctx, tx)
if err != nil {
    return fmt.Errorf("simulate transport: %w", err)
}

if sim.Err != nil {
    decoded := rpc.DecodeTransactionError(sim.Err)
    var ie *rpc.InstructionError
    var te *rpc.TransactionError
    switch {
    case errors.As(decoded, &ie):
        log.Printf("instruction %d: %s", ie.Index, ie.Kind)
        if ie.Kind == "Custom" {
            log.Printf("  program error 0x%x", ie.CustomErrorCode)
        }
    case errors.As(decoded, &te):
        log.Printf("transaction-level: %s", te.Kind)
    }
    return decoded
}

fmt.Printf("sim ok: %d log lines\n", len(sim.Logs))
```

## Matching program-specific errors

Anchor programs encode their errors as `Custom(u32)` where the u32
is the Anchor-assigned error code. Match on the code to branch:

```go
var ie *rpc.InstructionError
if errors.As(decoded, &ie) && ie.Kind == "Custom" {
    switch ie.CustomErrorCode {
    case 6000: // NotInitialized
        return retryAfterInit(ctx)
    case 6001: // Unauthorized
        return fmt.Errorf("unauthorized caller")
    }
}
```

The Anchor error code list lives in the program's IDL. The SDK does
not ship an IDL code generator (out of scope; mirror the constants
manually or generate them separately).

## Decoding from `getTransaction`

`TransactionMeta.Err` is a `json.RawMessage`; `SimulateResult.Err` is
the already-unmarshaled `any` form. `rpc.DecodeTransactionError` accepts
both and treats nil / empty / `"null"` as success, so a single guard
works:

```go
res, _ := c.GetTransaction(ctx, sig)
if res != nil && res.Meta != nil {
    if err := rpc.DecodeTransactionError(res.Meta.Err); err != nil {
        log.Println(err)
    }
}
```

`SendAndConfirmTransaction` already runs `DecodeTransactionError` on
on-chain errors inside its confirmation loop, so the typed error is
the one returned to your caller — no manual decode needed in that
path.

## Related

- [Simulate Transaction](Simulate-Transaction) — the raw
  `SimulateTransaction` call.
- [Error Handling](Error-Handling) — the full classifier for
  transport and JSON-RPC errors.
- [SendAndConfirmTransaction](SendAndConfirmTransaction) — decodes
  on-chain errors automatically.
