# Call Options

Every typed RPC method accepts a final variadic config struct argument
(zero or one value). This is the SDK's chosen alternative to
functional options: a single, typed, IDE-discoverable struct per
shape, rather than a forest of `WithXxx` functions where each method
silently ignores most of them.

```go
res, err := c.GetAccountInfo(ctx, pubkey, rpc.AccountInfoCfg{
    Commitment: solana.CommitmentConfirmed,
    Encoding:   solana.EncodingBase64,
    DataSlice:  &rpc.DataSlice{Offset: 0, Length: 64},
})
```

If a method has no options to set, omit the argument entirely:

```go
slot, err := c.GetSlot(ctx)                  // no config
balance, err := c.GetBalance(ctx, pubkey)    // no config
```

## The config types

All live in `package rpc`. Each method's godoc names the config type
it accepts, and only that type — passing the wrong shape is a
compile error.

| Type | Used by |
|---|---|
| `CommitmentCfg` | `GetSupply`, `GetVoteAccounts`, `GetBlockProduction`, `GetMinimumBalanceForRentExemption`, `GetStakeMinimumDelegation`, `GetInflationGovernor`, `RequestAirdrop`, `GetTokenLargestAccounts`, `GetTokenSupply`, `GetTokenAccountBalance`, `GetBlocks`, `GetBlocksWithLimit` |
| `CommitmentWithMinSlotCfg` | `GetBalance`, `GetSlot`, `GetBlockHeight`, `GetLatestBlockhash`, `GetEpochInfo`, `GetSlotLeader`, `GetTransactionCount`, `IsBlockhashValid`, `GetFeeForMessage`, `GetStakeActivation` |
| `CommitmentWithEncodingCfg` | `AccountSubscribe`, `ProgramSubscribe` |
| `AccountInfoCfg` | `GetAccountInfo`, `GetMultipleAccounts`, `GetProgramAccounts`, `GetTokenAccountsByOwner`, `GetTokenAccountsByDelegate` |
| `GetBlockCfg` | `GetBlock`, `BlockSubscribe` |
| `GetTransactionCfg` | `GetTransaction` |
| `SignaturesForAddressCfg` | `GetSignaturesForAddress` |
| `SignatureStatusesCfg` | `GetSignatureStatuses` |
| `SendTxCfg` | `SendTransaction`, `SendRawTransaction` |
| `SimulateTxCfg` | `SimulateTransaction` |
| `LargestAccountsCfg` | `GetLargestAccounts` |
| `InflationRewardCfg` | `GetInflationReward` |
| `LeaderScheduleCfg` | `GetLeaderSchedule` |
| `LogsSubscribeCfg` | `LogsSubscribe` |
| `SignatureSubscribeCfg` | `SignatureSubscribe` |

## Common fields

### `Commitment CommitmentLevel`

The commitment level for the query. Tiers, in order of strength:

| Constant | Returns |
|---|---|
| `solana.CommitmentProcessed` | Most recent block — may be rolled back |
| `solana.CommitmentConfirmed` | Block confirmed by a cluster supermajority |
| `solana.CommitmentFinalized` | Block rooted by a cluster supermajority — never rolls back |

The zero value is the empty string; the server treats it as the
endpoint's default (typically `Finalized`).

### `MinContextSlot *uint64`

Constrain the RPC query to return data no older than this slot.
The server rejects the call (`-32016 MinContextSlotNotReached`) if
its own slot is behind. Pass `nil` (the zero value) to skip the
constraint.

### `Encoding Encoding`

Wire encoding for account data / transaction data fields.

| Constant | Value | Notes |
|---|---|---|
| `solana.EncodingBase64` | `"base64"` | Default for account-data methods |
| `solana.EncodingBase64ZSTD` | `"base64+zstd"` | Decoded transparently via `klauspost/compress/zstd`; 40–80% bandwidth saving for large accounts |
| `solana.EncodingBase58` | `"base58"` | Slow for large payloads; avoid |
| `solana.EncodingJSON` | `"json"` | Few programs support this |
| `solana.EncodingJSONParsed` | `"jsonParsed"` | Server-side parse for SPL Token, stake, vote |

### `DataSlice *rpc.DataSlice`

Return only a `[Offset, Offset+Length)` window of account data.
Reduces bandwidth when only a header field is needed.

### `MaxSupportedTransactionVersion *uint64`

Tell the server which transaction versions you understand. Pass
`new(uint64)` (i.e. a pointer to `0`) to accept v0 transactions
(the current maximum). Omit to accept only legacy.

## Pagination

`SignaturesForAddressCfg` carries:

- `Limit *int` — cap returned items
- `Before string` — page anchor (signature)
- `Until string` — page anchor (signature)

```go
limit := 1000
res, err := c.GetSignaturesForAddress(ctx, addr, rpc.SignaturesForAddressCfg{
    Limit:  &limit,
    Before: lastSeen,
})
```

## Send / simulate-specific fields

### `SendTxCfg`

```go
skip := true
sig, err := c.SendTransaction(ctx, tx, rpc.SendTxCfg{
    SkipPreflight:       &skip,
    PreflightCommitment: solana.CommitmentConfirmed,
})
```

- `SkipPreflight *bool` — skip the server's preflight check. Use
  only after a local simulation, or when retrying a known-valid
  transaction.
- `PreflightCommitment` — commitment for the preflight, independent
  of any caller-side polling commitment.
- `MaxRetries *uint` — cap the leader-forwarding retries the server
  performs.
- `MinContextSlot *uint64`, `Encoding`.

### `SimulateTxCfg`

`SigVerify` and `ReplaceRecentBlockhash` are mutually exclusive at
the server level — set at most one.

```go
verify := true
res, err := c.SimulateTransaction(ctx, tx, rpc.SimulateTxCfg{SigVerify: &verify})
```

## Why structs over functional options

- A single struct type per shape compiles or doesn't; an unhonoured
  `With…` call silently does nothing.
- IDE autocomplete shows the exact set of options the method accepts,
  with godoc on each field, instead of a flat namespace where every
  `With…` looks equally valid for every method.
- Adding a field is a non-breaking change (zero value preserves
  behaviour); adding a `With…` to a function that previously didn't
  honour it is silently ignored.
