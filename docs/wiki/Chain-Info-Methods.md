# Chain Info Methods

Cluster-wide state queries live in `rpc/methods_chain.go`. None of
them need account addresses — they report the current state of
the network as a whole.

## `GetEpochInfo`

```go
func (c *Client) GetEpochInfo(ctx context.Context, cfg ...rpc.CommitmentWithMinSlotCfg) (*EpochInfo, error)

type EpochInfo struct {
    AbsoluteSlot     uint64
    BlockHeight      uint64
    Epoch            uint64
    SlotIndex        uint64
    SlotsInEpoch     uint64
    TransactionCount *uint64
}
```

Returns the current epoch, the slot within that epoch, the
total number of slots in the epoch, the current absolute slot,
block height, and (optionally) the total transaction count
since genesis.

Cfg: `rpc.CommitmentWithMinSlotCfg{Commitment, MinContextSlot}`.

## `GetSupply`

```go
func (c *Client) GetSupply(ctx context.Context, cfg ...rpc.CommitmentCfg) (*SupplyResult, error)

type SupplyResult struct {
    Slot                   uint64
    Total                  uint64 // lamports
    Circulating            uint64
    NonCirculating         uint64
    NonCirculatingAccounts []string
}
```

Returns the circulating and total SOL supply (in lamports).
`NonCirculatingAccounts` lists the accounts holding
non-circulating supply.

Cfg: `rpc.CommitmentCfg{Commitment: …}`.

## `GetInflationRate`

```go
func (c *Client) GetInflationRate(ctx context.Context) (*InflationRate, error)

type InflationRate struct {
    Total      float64
    Validator  float64
    Foundation float64
    Epoch      uint64
}
```

Returns the current inflation rate broken down by validator and
foundation share. No options are honoured.

## `GetMinimumBalanceForRentExemption`

```go
func (c *Client) GetMinimumBalanceForRentExemption(
    ctx context.Context,
    dataSize uint64,
    cfg ...rpc.CommitmentCfg,
) (uint64, error)
```

Returns the minimum lamports an account of the given size must
hold to be rent-exempt. Pass this value as the `lamports`
argument to `system.NewCreateAccount` to fund a new account
just above the rent-exempt threshold.

Cfg: `rpc.CommitmentCfg{Commitment: …}`.

```go
size := uint64(82) // SPL Token mint account size
rent, _ := c.GetMinimumBalanceForRentExemption(ctx, size)
fmt.Println("rent-exempt lamports:", rent)
```

## `GetSlotLeader`

```go
func (c *Client) GetSlotLeader(ctx context.Context, cfg ...rpc.CommitmentWithMinSlotCfg) (solana.PublicKey, error)
```

Returns the public key of the current slot leader. Useful for
leader-aware transaction forwarding or leader-schedule based
optimisations.

Cfg: `rpc.CommitmentWithMinSlotCfg{Commitment, MinContextSlot}`.

## `GetBlockTime`

```go
func (c *Client) GetBlockTime(ctx context.Context, slot uint64) (*int64, error)
```

Returns the UNIX timestamp at which the given slot was
produced, or `nil` if the node has not indexed it yet.

## Related

- [Query Methods](Query-Methods) — `GetBalance`, `GetSlot`,
  `GetLatestBlockhash`, `GetBlockHeight`.
- [Fee Methods](Fee-Methods) — `RequestAirdrop`,
  `GetFeeForMessage`, `GetRecentPrioritizationFees`.
