# Block and Token Methods

Block retrieval and SPL Token balance / supply queries live in
`rpc/methods_block_token.go`.

## `GetBlock`

```go
func (c *Client) GetBlock(ctx context.Context, slot uint64, opts ...CallOption) (*GetBlockResult, error)
```

Fetches the content of a single block by absolute slot. Returns
`nil, nil` if the slot is absent or skipped — treat a nil result
as a success signal that the slot was empty, not as an error.

### Honoured options

- `WithCommitment`
- `WithEncoding` (default: `EncodingBase64`)
- `WithMaxSupportedTransactionVersion` (default: `0`, which
  means v0 transactions are accepted)

### Result shape

```go
type GetBlockResult struct {
    Blockhash         string
    PreviousBlockhash string
    ParentSlot        uint64
    Transactions      []BlockTransaction
    BlockHeight       *uint64
    BlockTime         *int64
    Rewards           []any
}

type BlockTransaction struct {
    Transaction AccountData      // [value, encoding] form
    Meta        *TransactionMeta
    Version     any              // "legacy" or 0
}
```

`Transactions[i].Transaction.Bytes()` decodes the wire bytes;
`solana.UnmarshalTransaction(bytes)` gives you a typed
`*Transaction`.

```go
for _, bt := range block.Transactions {
    raw, _ := bt.Transaction.Bytes()
    tx, _ := solana.UnmarshalTransaction(raw)
    _ = tx
}
```

The `Meta` field is the typed [TransactionMeta](Transaction-Query)
struct holding fee, logs, balance changes.

### Rewards, inner instructions, token balances

`Rewards`, `InnerInstructions`, and `Pre/PostTokenBalances` are
currently retained as `[]any` until richer typed models land.
Use them if you need the raw shape; expect typed variants in
follow-up releases.

## `GetBlocks`

```go
func (c *Client) GetBlocks(ctx context.Context, start uint64, end *uint64, opts ...CallOption) ([]uint64, error)
```

Returns the list of confirmed block slots in the inclusive
range `[start, end]`. Pass `nil` for `end` to use the node's
latest confirmed slot.

Honoured options: `WithCommitment`.

```go
slots, err := c.GetBlocks(ctx, 12_000_000, nil)
```

## SPL Token queries

### `GetTokenAccountBalance`

```go
func (c *Client) GetTokenAccountBalance(
    ctx context.Context,
    account PublicKey,
    opts ...CallOption,
) (*GetTokenAccountBalanceResult, error)

type TokenAmount struct {
    Amount         string   // raw unscaled value
    Decimals       uint8
    UIAmount       *float64 // scaled by decimals
    UIAmountString string
}
```

Returns the balance of an SPL Token **account** (not a wallet —
pass the associated token account address). The raw `Amount`
is a decimal string because Solana's JSON-RPC emits u64 values
as strings to survive clients that can't represent `2^53 + 1`.

Honoured options: `WithCommitment`.

### `GetTokenSupply`

```go
func (c *Client) GetTokenSupply(ctx context.Context, mint PublicKey, opts ...CallOption) (*GetTokenSupplyResult, error)
```

Returns the total supply of an SPL Token mint. Same
`TokenAmount` shape.

Honoured options: `WithCommitment`.

## Example: compute a wallet's USDC balance

```go
import (
    "github.com/MevYu/solana-go"
    associatedtokenaccount "github.com/MevYu/solana-go/programs/associated-token-account"
    "github.com/MevYu/solana-go/programs/token"
)

usdcMint, _ := solana.PublicKeyFromBase58("EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v")
wallet,   _ := solana.PublicKeyFromBase58("...")

tokenAccount, _, _ := associatedtokenaccount.FindAssociatedTokenAddress(
    wallet, usdcMint, token.ProgramID,
)

res, err := c.GetTokenAccountBalance(ctx, tokenAccount)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("USDC: %s (%d decimals)\n", res.Value.Amount, res.Value.Decimals)
```

## Related

- [Transaction Query](Transaction-Query) for fetching a single
  transaction by signature.
- [Advanced Subscriptions](Advanced-Subscriptions) for
  streaming block updates via WebSocket.
