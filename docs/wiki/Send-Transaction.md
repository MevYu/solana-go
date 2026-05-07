# Send Transaction

Broadcasting a signed transaction lives in
`rpc/methods_transaction.go`. The client offers two forms:

- `SendTransaction` — takes a typed `*Transaction`, marshals
  it, and calls `sendTransaction`.
- `SendRawTransaction` — takes pre-marshaled wire bytes.

Both return the first signature of the transaction — the
payer's — which doubles as the transaction identifier for
subsequent queries.

## `SendTransaction`

```go
func (c *rpc.Client) SendTransaction(
    ctx context.Context,
    tx *solana.Transaction,
    cfg ...rpc.SendTxCfg,
) (solana.Signature, error)
```

The transaction must already be signed by every required
signer; call [Transaction.Sign](Transactions) first.

### `rpc.SendTxCfg`

```go
type SendTxCfg struct {
    SkipPreflight       *bool
    PreflightCommitment CommitmentLevel
    MaxRetries          *uint
    MinContextSlot      *uint64
    Encoding            Encoding // base64 (default) or base58
}
```

There is no post-send commitment field — the send path does not
take a confirmation argument. Use the separate confirmation
loop or [SendAndConfirmTransaction](SendAndConfirmTransaction)
for that.

### Example

```go
sig, err := c.SendTransaction(ctx, tx, rpc.SendTxCfg{
    PreflightCommitment: solana.CommitmentConfirmed,
})
if err != nil {
    // inspect err with errors.As(*jsonrpc.ErrRPC) or the Is* classifiers
    return err
}
fmt.Println("submitted:", sig)
```

## `SendRawTransaction`

```go
func (c *rpc.Client) SendRawTransaction(
    ctx context.Context,
    raw []byte,
    cfg ...rpc.SendTxCfg,
) (solana.Signature, error)
```

Identical contract to `SendTransaction`, but you supply the wire
bytes directly. Use this when:

- You cached the marshaled form in Redis / on disk.
- You received the bytes from a remote builder / co-signer and
  want to avoid the parse-then-remarshal round trip.
- You have a test that constructs bytes by hand.

## Preflight explained

`sendTransaction` runs a preflight simulation server-side before
accepting the wire bytes — it validates signatures and simulates
the transaction at a chosen commitment to catch obvious errors
early. Preflight failures surface as JSON-RPC errors with the
same `err` shape as `simulateTransaction`.

### When to skip preflight

Skip preflight only when:

- You already called `SimulateTransaction` and got a clean
  result.
- You are retrying a transaction that the cluster rejected for
  a temporary reason (e.g. `BlockhashNotFound` on one endpoint
  but valid on another).

```go
skip := true
maxRetries := uint(0)
sig, err := c.SendTransaction(ctx, tx, rpc.SendTxCfg{
    SkipPreflight: &skip,
    MaxRetries:    &maxRetries, // don't let the server retry on our behalf
})
```

## After `Send` returns

A successful return means **the node accepted the bytes** —
not that the transaction landed. The transaction is now in the
cluster's pipeline. To wait for confirmation:

1. Poll `GetSignatureStatuses` until the signature reports the
   desired `ConfirmationStatus`.
2. Or subscribe via `SignatureSubscribe` on the WebSocket
   client for a one-shot push notification.
3. Or use `c.SendAndConfirmTransaction` on `*rpc.Client`,
   which does all of the above plus automatic blockhash refresh.

See [Signature Methods](Signature-Methods) for the polling
API and [SendAndConfirmTransaction](SendAndConfirmTransaction)
for the full helper.

## Error handling

The send call can fail for many reasons; the most common are:

| Condition | Signal |
|---|---|
| Blockhash expired before send | `jsonrpc.IsBlockhashExpired(err)` |
| Payer is broke | `jsonrpc.IsInsufficientFunds(err)` |
| Server throttling | `jsonrpc.IsRateLimited(err)` |
| Node behind the cluster | `jsonrpc.IsNodeBehind(err)` |
| Program error during preflight | `*jsonrpc.ErrRPC` with `-32002` code |

See [Error Handling](Error-Handling) for the full classifier
index and a retry pattern.

## Related

- [Simulate Transaction](Simulate-Transaction) — preflight
  the transaction yourself to catch errors with decoded logs
  before the send.
- [SendAndConfirmTransaction](SendAndConfirmTransaction) — the
  helper that combines send + confirm + blockhash refresh.
