# Query Methods

The simplest RPC queries — balance, slot, block height,
blockhash — live in `rpc/methods_query.go`. Each is a one-line call
that returns a typed value.

## `GetBalance`

```go
func (c *Client) GetBalance(ctx context.Context, pubkey PublicKey, opts ...CallOption) (*GetBalanceResult, error)

type GetBalanceResult struct {
    Slot  uint64
    Value uint64 // lamports
}
```

Returns the balance (in lamports) of the account at `pubkey`.
The returned `Slot` is the slot at which the balance was read;
use it if you need to compare timing across calls.

Honoured options: `WithCommitment`, `WithMinContextSlot`.

```go
res, _ := c.GetBalance(ctx, wallet)
fmt.Printf("%d lamports @ slot %d\n", res.Value, res.Slot)
```

## `GetSlot`

```go
func (c *Client) GetSlot(ctx context.Context, opts ...CallOption) (uint64, error)
```

Returns the current absolute slot the node is processing.

Honoured options: `WithCommitment`, `WithMinContextSlot`.

## `GetBlockHeight`

```go
func (c *Client) GetBlockHeight(ctx context.Context, opts ...CallOption) (uint64, error)
```

Returns the current block height — the number of blocks
produced since genesis, regardless of forks. This is distinct
from the slot: some slots are skipped, so `block_height ≤ slot`
always.

Block height is the reference for the `LastValidBlockHeight`
field in `GetLatestBlockhash`: once the current block height
passes that value, a transaction committed to the blockhash
can no longer land.

Honoured options: `WithCommitment`, `WithMinContextSlot`.

## `GetLatestBlockhash`

```go
func (c *Client) GetLatestBlockhash(ctx context.Context, opts ...CallOption) (*LatestBlockhash, error)

type LatestBlockhash struct {
    Slot                 uint64
    Blockhash            Hash
    LastValidBlockHeight uint64
}
```

Returns the latest blockhash together with the highest block
height at which a transaction committing to that blockhash can
still land. Callers use the `Blockhash` in new transactions and
poll `LastValidBlockHeight` vs `GetBlockHeight` to decide when
to refresh.

Honoured options: `WithCommitment`, `WithMinContextSlot`.

### Blockhash lifetime

A Solana blockhash remains valid for roughly 150 slots
(≈60 seconds on mainnet). The authoritative signal is
`LastValidBlockHeight`; build your retry loop around that,
not around a wall-clock timeout:

```go
bh, _ := c.GetLatestBlockhash(ctx)
// ... construct, sign, send ...
for {
    height, _ := c.GetBlockHeight(ctx)
    if height > bh.LastValidBlockHeight {
        // blockhash expired; fetch a new one
        bh, _ = c.GetLatestBlockhash(ctx)
        // rebuild, re-sign, re-send
    }
    // poll signature status ...
}
```

`Client.SendAndConfirmTransaction` does this for you. See
[SendAndConfirmTransaction](SendAndConfirmTransaction).

## Related

- [Chain Info Methods](Chain-Info-Methods) — epoch info, slot
  leader, inflation rate.
- [Send Transaction](Send-Transaction) — where
  `GetLatestBlockhash` feeds in.
