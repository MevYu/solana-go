# Error Handling

The SDK distinguishes three layers of error:

1. **Transport errors** — network failures, HTTP non-2xx,
   context cancellation. Returned as wrapped errors from the
   `rpc` package.
2. **JSON-RPC errors** — well-formed responses carrying a
   JSON-RPC error object. Returned as `*jsonrpc.ErrRPC`.
3. **Transaction / instruction errors** — the `err` field
   inside `simulateTransaction` / `getTransaction` results.
   Decoded with `rpc.DecodeTransactionError`.

Each layer has a dedicated type you can match on.

## Transport: wrapped errors

Every RPC call that fails wraps the underlying error with the
method name:

```
solana rpc getBalance: ...
```

Use standard `errors.Is` / `errors.As` patterns:

```go
_, err := c.GetBalance(ctx, addr)
if errors.Is(err, context.DeadlineExceeded) {
    // caller deadline fired
}
```

### Retries

The default `jsonrpc.RetryPolicy` retries only transient errors:

- HTTP 5xx
- HTTP 429 (rate limit)
- Non-context network errors (DNS, connection refused, EOF
  mid-stream, …)

Context cancellation and deadline errors are **never** retried.
The policy is exponential backoff with full jitter, capped at
10 seconds per attempt and 5 attempts total. Customise via
`jsonrpc.WithRetryPolicy(yourPolicy)` passed to
`rpc.NewClientWith(...)`, or via the `RetryPolicy` field of
`jsonrpc.Config` passed to `rpc.NewClient(url, cfg)`.

## JSON-RPC: `*jsonrpc.ErrRPC`

When the server returns a JSON-RPC error object, the client
wraps it in `*jsonrpc.ErrRPC`:

```go
type ErrRPC struct {
    Method string
    Code   int
    Msg    string
    Data   RawJSON // method-specific payload
    Body   []byte  // raw response body for diagnostics
}

func (e *ErrRPC) Error() string
```

Recover it with `errors.As`:

```go
_, err := c.GetBalance(ctx, addr)
var rpcErr *jsonrpc.ErrRPC
if errors.As(err, &rpcErr) {
    fmt.Println(rpcErr.Code, rpcErr.Msg)
    // rpcErr.Data is raw JSON; unmarshal into a typed struct
    // if the method documents one
}
```

### Known numeric codes

`jsonrpc/classify.go` defines the numeric codes Solana uses
for common failures:

| Constant | Code | Meaning |
|---|---|---|
| `RPCErrCodeBlockCleanedUp` | -32001 | Block pruned |
| `RPCErrCodeTransactionSimulationFail` | -32002 | Preflight / simulate failed |
| `RPCErrCodeSigVerifyFailed` | -32003 | Signature verification failed |
| `RPCErrCodeBlockNotAvailable` | -32004 | Block not available |
| `RPCErrCodeNodeUnhealthy` | -32005 | Node behind the cluster |
| `RPCErrCodeSlotSkipped` | -32007 | Slot skipped in leader schedule |
| `RPCErrCodeNoSnapshot` | -32008 | No snapshot available |
| `RPCErrCodeLongTermStorageSlotSkipped` | -32009 | Pruned from long-term storage |
| `RPCErrCodeKeyExcluded` | -32010 | Key filtered by the endpoint |
| `RPCErrCodeTransactionPrecompileFail` | -32011 | Precompile rejected the transaction |
| `RPCErrCodeScanError` | -32012 | Internal scan error |
| `RPCErrCodeTransactionHistoryOff` | -32013 | This node does not index history |
| `RPCErrCodeMinContextSlotNotReached` | -32016 | Server is behind the requested `MinContextSlot` |

## Classifier helpers

The `Is*` functions in the **`jsonrpc`** package match an error
against a symbolic condition, regardless of whether the cause is
a sentinel (`errors.Is`), an `*ErrRPC` with a known numeric
code, or a message substring. Use them in retry loops instead of
string matching:

| Helper | What it matches |
|---|---|
| `jsonrpc.IsBlockhashExpired` | `BlockhashNotFound`, `ErrBlockhashExpired` |
| `jsonrpc.IsInsufficientFunds` | `InsufficientFundsForFee`, `InsufficientFundsForRent`, sentinel |
| `jsonrpc.IsRateLimited` | HTTP 429, message substrings, sentinel |
| `jsonrpc.IsNodeBehind` | `-32005`, `Node is behind`, sentinel |
| `jsonrpc.IsTransactionExpired` | `TransactionExpired`, sentinel |
| `jsonrpc.IsAccountNotFound` | `AccountNotFound`, sentinel |
| `jsonrpc.IsSignatureNotFound` | `SignatureNotFound`, sentinel |
| `jsonrpc.IsSlotSkipped` | `-32007`, `-32009`, sentinel |
| `jsonrpc.IsBlockCleanedUp` | `-32001`, sentinel |

### Example: retry on rate limit

```go
for attempt := 0; attempt < 5; attempt++ {
    res, err := c.GetBalance(ctx, addr)
    if err == nil {
        return res, nil
    }
    if !jsonrpc.IsRateLimited(err) {
        return nil, err
    }
    // exponential backoff
    time.Sleep(time.Duration(1<<attempt) * time.Second)
}
return nil, fmt.Errorf("rate-limited after 5 attempts")
```

The default `RetryPolicy` already handles HTTP 429, but
caller-level backoff is still useful for strict rate limits
over the 5-attempt default.

## Sentinel errors

```go
// All defined in package jsonrpc (jsonrpc/classify.go).
var (
    ErrBlockhashExpired   = errors.New("solana: blockhash expired")
    ErrInsufficientFunds  = errors.New("solana: insufficient funds")
    ErrRateLimited        = errors.New("solana: rate limited")
    ErrNodeBehind         = errors.New("solana: node is behind the cluster")
    ErrTransactionExpired = errors.New("solana: transaction expired")
    ErrAccountNotFound    = errors.New("solana: account not found")
    ErrSignatureNotFound  = errors.New("solana: signature not found")
    ErrSlotSkipped        = errors.New("solana: slot skipped")
    ErrBlockCleanedUp     = errors.New("solana: block cleaned up")
)
```

The sentinels are never wrapped directly by the transport —
you always use the `Is*` classifiers, which both check
`errors.Is` against the sentinel **and** pattern-match the raw
RPC error. Raising one of these from your own code is
supported if you want to propagate a canonical shape.

## Transaction / instruction errors

```go
import "github.com/MevYu/solana-go/rpc"

err := rpc.DecodeTransactionError(rawErrFromRPC)

var te *rpc.TransactionError
var ie *rpc.InstructionError
switch {
case errors.As(err, &ie):
    fmt.Printf("instruction %d: %s", ie.Index, ie.Kind)
    if ie.Kind == "Custom" {
        fmt.Printf(" (program error 0x%x)", ie.CustomErrorCode)
    }
case errors.As(err, &te):
    fmt.Printf("transaction-level: %s", te.Kind)
}
```

See [Simulate With Decoded Errors](Simulate-With-Decoded-Errors)
for the full decoding rules and examples.

## Related

- [Send Transaction](Send-Transaction) — the most common source
  of JSON-RPC errors.
- [SendAndConfirmTransaction](SendAndConfirmTransaction) —
  automatically refreshes on `IsBlockhashExpired`.
