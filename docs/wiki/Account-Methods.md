# Account Methods

The two account-fetching methods on `rpc.Client`:

- `GetAccountInfo`
- `GetMultipleAccounts`

Both return [AccountInfo](Accounts) structures — the RPC form
that preserves the `[value, encoding]` data tuple.

## `GetAccountInfo`

```go
func (c *rpc.Client) GetAccountInfo(
    ctx context.Context,
    pubkey solana.PublicKey,
    cfg ...rpc.AccountInfoCfg,
) (*c.GetAccountInfoResult, error)

type GetAccountInfoResult struct {
    Slot    uint64
    Account *solana.AccountInfo // nil if the account does not exist
}

type rpc.AccountInfoCfg struct {
    Commitment     CommitmentLevel
    Encoding       Encoding // default: EncodingBase64
    DataSlice      *DataSlice
    MinContextSlot *uint64
}
```

### Absent accounts

The result's `Account` field is `nil` when the account does not
exist at the requested commitment. Treat this as a success case
— it is not an error.

```go
res, err := c.GetAccountInfo(ctx, addr)
if err != nil {
    return err
}
if res.Account == nil {
    log.Println("account does not exist")
    return nil
}
log.Printf("%d lamports, owner %s", res.Account.Lamports, res.Account.Owner)
```

### Decoding data

`Account.Data` is an [AccountData](Accounts) value; call
`.Bytes()` to get the raw bytes:

```go
raw, err := res.Account.Data.Bytes()
if err != nil {
    return err // unsupported encoding
}
// raw is nil if the account exists but holds no data
```

### Conversion to `Account`

```go
acct, err := res.Account.ToAccount()
```

Drops the encoding metadata and gives you the binary-compatible
`Account` shape instead, suitable for feeding into SPL Token /
Anchor / ALT decoders.

## `GetMultipleAccounts`

```go
func (c *rpc.Client) GetMultipleAccounts(
    ctx context.Context,
    addresses []solana.PublicKey,
    cfg ...rpc.AccountInfoCfg,
) (*c.GetMultipleAccountsResult, error)

type GetMultipleAccountsResult struct {
    Slot     uint64
    Accounts []*solana.AccountInfo // same length as addresses; nil entry = missing
}
```

A single round trip fetches many accounts. The result slice has
the same length as the input and preserves order; each entry is
`nil` if the matching account does not exist.

### Honoured options

Same as `GetAccountInfo`: `WithCommitment`, `WithEncoding`,
`WithDataSlice`, `WithMinContextSlot`.

### Example

```go
addrs := []solana.PublicKey{a, b, c}
res, err := c.GetMultipleAccounts(ctx, addrs,
    rpc.WithEncoding(solana.EncodingBase64),
    rpc.WithCommitment(rpc.CommitmentConfirmed),
)
if err != nil {
    return err
}
for i, info := range res.Accounts {
    if info == nil {
        fmt.Printf("%s: missing\n", addrs[i])
        continue
    }
    fmt.Printf("%s: %d lamports\n", addrs[i], info.Lamports)
}
```

## Choosing between them

- **One account** → `GetAccountInfo`.
- **2–100 accounts** → `GetMultipleAccounts`. This method
  batches server-side and avoids the per-request latency of
  many individual calls.
- **Accounts owned by a program** → `c.GetProgramAccounts(ctx, program, cfg...)`.

## Watching for live updates

For a continuously-updated account view rather than a point-in-time
snapshot, use the WebSocket [Account Subscriptions](Account-Subscriptions).
