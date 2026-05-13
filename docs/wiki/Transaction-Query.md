# Transaction Query

`GetTransaction` fetches a single transaction by signature. It
lives in `rpc/methods_transaction.go`.

```go
func (c *Client) GetTransaction(
    ctx context.Context,
    sig solana.Signature,
    cfg ...rpc.GetTransactionCfg,
) (*GetTransactionResult, error)
```

## Result

```go
type GetTransactionResult struct {
    Slot        uint64
    BlockTime   *int64                  // UNIX timestamp, or nil if unknown
    Meta        *solana.TransactionMeta
    Transaction *solana.Transaction     // fully decoded structure
    Version     any                     // "legacy" or 0
}

// solana.TransactionMeta
type TransactionMeta struct {
    Err                  json.RawMessage  // empty / "null" on success
    Fee                  uint64
    PreBalances          []uint64
    PostBalances         []uint64
    LogMessages          []string
    ComputeUnitsConsumed *uint64
    InnerInstructions    []InnerInstruction
    PreTokenBalances     []TokenBalance
    PostTokenBalances    []TokenBalance
    Rewards              []Reward
    LoadedAddresses      *LoadedAddresses
}
```

## Returns nil, nil for unknown signatures

If the signature is not known to the node, `GetTransaction`
returns `nil, nil` — not an error. Treat a nil result as "did
not land, or landed but aged out of the node's index":

```go
res, err := c.GetTransaction(ctx, sig)
if err != nil {
    return err
}
if res == nil {
    fmt.Println("signature not found")
    return nil
}
```

## Cfg fields (`rpc.GetTransactionCfg`)

| Field | Default |
|---|---|
| `Commitment` | server default |
| `Encoding` | `EncodingBase64` |
| `MaxSupportedTransactionVersion` | `0` (accept v0) |

## Decoding the transaction body

`res.Transaction` is a fully decoded `*solana.Transaction`.
`Transaction.UnmarshalJSON` reads the `[value, encoding]` JSON
tuple, decodes base64 / base58 / base64+zstd to wire bytes, and
parses them into the structured type in one pass — no manual
`UnmarshalBinary` step:

```go
tx := res.Transaction
// tx.Message has the account keys, instructions, blockhash
// tx.Signatures has the signatures
for _, ix := range tx.Message.Instructions {
    _ = ix
}
```

`json` / `jsonParsed` encodings are not supported here — request a
binary encoding (base64 is the SDK default).

## Reading execution meta

`Meta.Fee` is the fee that was charged. `Meta.LogMessages`
holds the program log output. `Meta.ComputeUnitsConsumed` is
the actual CU cost — useful for retrospectively calibrating
`SetComputeUnitLimit`.

### Balance changes

```go
for i := range res.Meta.PreBalances {
    delta := int64(res.Meta.PostBalances[i]) - int64(res.Meta.PreBalances[i])
    if delta != 0 {
        fmt.Printf("account %d: %+d lamports\n", i, delta)
    }
}
```

The indices match the transaction's `Message.AccountKeys` (or
for v0 transactions, the concatenation of static keys and ALT
resolved keys — use `Meta.LoadedAddresses` to reconstruct the
full view).

### Typed `Meta.Err`

`Meta.Err` is `json.RawMessage` carrying the on-wire JSON of the
transaction error (or empty on success). Feed it to
`rpc.DecodeTransactionError`, which accepts `json.RawMessage`
directly, to get typed Go errors. See
[Simulate With Decoded Errors](Simulate-With-Decoded-Errors)
for details.

## Example: audit a transaction's impact

```go
res, err := c.GetTransaction(ctx, sig, rpc.GetTransactionCfg{
    Commitment: solana.CommitmentFinalized,
})
if err != nil {
    return err
}
if res == nil {
    return fmt.Errorf("signature %s not found", sig)
}

tx := res.Transaction

var cus uint64
if res.Meta.ComputeUnitsConsumed != nil {
    cus = *res.Meta.ComputeUnitsConsumed
}
fmt.Printf("slot %d, fee %d lamports, %d CUs, %d ixs\n",
    res.Slot, res.Meta.Fee, cus, len(tx.Message.Instructions))

for i, line := range res.Meta.LogMessages {
    fmt.Printf("  %3d: %s\n", i, line)
}

for i, key := range tx.Message.AccountKeys {
    pre  := res.Meta.PreBalances[i]
    post := res.Meta.PostBalances[i]
    if pre != post {
        fmt.Printf("  %s: %d -> %d\n", key, pre, post)
    }
}
```

## Unmodelled fields

`InnerInstructions`, `PreTokenBalances`, `PostTokenBalances`,
and `Rewards` are currently retained as `[]any`. Use them if
you need the raw shape; typed models will land in a follow-up.

## Related

- [Signature Methods](Signature-Methods) — `GetSignatureStatuses`
  for the cheaper "is it confirmed yet?" query.
- [Block & Token](Block-and-Token-Methods) — `GetBlock` returns
  the same transaction shape in a batch.
