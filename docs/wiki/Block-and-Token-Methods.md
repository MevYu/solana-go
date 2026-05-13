# Block and Token Methods

Block retrieval and SPL Token balance / supply queries live in
`rpc/methods_block_token.go`.

## `GetBlock`

```go
func (c *Client) GetBlock(ctx context.Context, slot uint64, cfg ...rpc.GetBlockCfg) (*GetBlockResult, error)
```

Fetches the content of a single block by absolute slot. Returns
`nil, nil` if the slot is absent or skipped — treat a nil result
as a success signal that the slot was empty, not as an error.

### Cfg fields (`rpc.GetBlockCfg`)

- `Commitment`
- `Encoding` (default: `EncodingBase64`)
- `MaxSupportedTransactionVersion` (default: `0`, accepts v0 transactions)
- `TransactionDetails`, `Rewards`

### Result shape

```go
type GetBlockResult struct {
    Blockhash         string
    PreviousBlockhash string
    ParentSlot        uint64
    Transactions      []BlockTransaction
    BlockHeight       *uint64
    BlockTime         *int64
    Rewards           []solana.Reward
}

type BlockTransaction struct {
    Transaction solana.EncodedData      // [value, encoding] form, eagerly decoded
    Meta        *solana.TransactionMeta
    Version     any                     // "legacy" or 0
}
```

`Transactions[i].Transaction.Bytes` holds the wire bytes (the
`UnmarshalJSON` on `EncodedData` already decoded base64 / base58 /
base64+zstd). Hand it to `(*Transaction).UnmarshalBinary` to get
a typed `*Transaction`:

```go
for _, bt := range block.Transactions {
    tx := &solana.Transaction{}
    if err := tx.UnmarshalBinary(bt.Transaction.Bytes); err != nil {
        continue
    }
    _ = tx
}
```

The `Meta` field is the typed [TransactionMeta](Transaction-Query)
struct holding fee, logs, balance changes,
`InnerInstructions []InnerInstruction`,
`PreTokenBalances / PostTokenBalances []TokenBalance`,
`Rewards []Reward`, and `LoadedAddresses` — all decoded into
real Go structs, no `[]any` raw shapes.

## `GetBlocks`

```go
func (c *Client) GetBlocks(ctx context.Context, start uint64, end *uint64, cfg ...rpc.CommitmentCfg) ([]uint64, error)
```

Returns the list of confirmed block slots in the inclusive
range `[start, end]`. Pass `nil` for `end` to use the node's
latest confirmed slot.

Cfg: `rpc.CommitmentCfg{Commitment: …}`.

```go
slots, err := c.GetBlocks(ctx, 12_000_000, nil)
```

## SPL Token queries

### `GetTokenAccountBalance`

```go
func (c *Client) GetTokenAccountBalance(
    ctx context.Context,
    account solana.PublicKey,
    cfg ...rpc.CommitmentCfg,
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

Cfg: `rpc.CommitmentCfg{Commitment: …}`.

### `GetTokenSupply`

```go
func (c *Client) GetTokenSupply(ctx context.Context, mint solana.PublicKey, cfg ...rpc.CommitmentCfg) (*GetTokenSupplyResult, error)
```

Returns the total supply of an SPL Token mint. Same
`TokenAmount` shape.

Cfg: `rpc.CommitmentCfg{Commitment: …}`.

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
