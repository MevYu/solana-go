# Accounts

Solana's on-chain state model is a flat array of accounts. Every
account has a balance in lamports, an owner program, opaque data
bytes, and a few flags. The SDK models accounts at two layers.

## `Account` — the binary shape

`Account` is the wire-level representation:

```go
type Account struct {
    Lamports   uint64
    Owner      PublicKey
    Data       []byte
    Executable bool
    RentEpoch  uint64
}
```

Use this when you want to feed account state into a binary
decoder (SPL Token mint state, Anchor account, ALT layout, …).

## `AccountInfo` — the RPC shape

`AccountInfo` is what the `getAccountInfo` and
`getMultipleAccounts` methods return over JSON-RPC:

```go
type AccountInfo struct {
    Lamports   uint64      `json:"lamports"`
    Owner      PublicKey   `json:"owner"`
    Data       AccountData `json:"data"`
    Executable bool        `json:"executable"`
    RentEpoch  uint64      `json:"rentEpoch"`
    Space      uint64      `json:"space"`
}
```

It differs from `Account` in two ways:

1. `Data` is an `AccountData` (see below) preserving the RPC's
   `[value, encoding]` two-element form.
2. `Space` is present; Solana's RPC reports the allocated byte
   size even when `Data` is truncated by a data slice.

## `AccountData` — the two-element form

Solana returns account data as a JSON array
`["<encoded_value>", "<encoding_name>"]`. `AccountData` is a
`[2]string` that decodes directly without a custom
`UnmarshalJSON`:

```go
type AccountData [2]string

func (d AccountData) Value() string       { return d[0] }
func (d AccountData) Encoding() Encoding  { return Encoding(d[1]) }
func (d AccountData) Bytes() ([]byte, error)
```

`Bytes()` dispatches on the encoding:

- `base64` → `base64.StdEncoding.DecodeString`
- `base58` → `base58.Decode`
- `base64+zstd` → base64 decode then `klauspost/compress/zstd`
  decompress (40–80% bandwidth saving for large accounts)
- `jsonParsed` / `json` → error (the server already parsed the
  data to a structured object; call sites that want `jsonParsed`
  should decode the object directly with their typed struct)

An empty `Value` decodes to `nil, nil` without error, matching
accounts that exist but hold no data.

## Converting between the two

```go
info, err := c.GetAccountInfo(ctx, pubkey)
if err != nil { return err }
if info.Account == nil {
    // account does not exist at the requested commitment
}
acct, err := info.Account.ToAccount() // *AccountInfo -> *Account
```

`ToAccount()` calls `Data.Bytes()` and propagates any decode
error, so a mint request that the server returned as `base64+zstd`
will surface as an explicit error rather than silently return
`nil` data.

## Example: fetch and decode

```go
import (
    "github.com/MevYu/solana-go/jsonrpc"
    "github.com/MevYu/solana-go/rpc"
)

c := rpc.NewClient("https://api.mainnet-beta.solana.com", jsonrpc.Config{})
ctx := context.Background()

pk, _ := solana.PublicKeyFromBase58("...")
res, err := c.GetAccountInfo(ctx, pk)
if err != nil {
    log.Fatal(err)
}
if res.Account == nil {
    log.Fatal("account does not exist")
}

raw, err := res.Account.Data.Bytes()
if err != nil {
    log.Fatal(err) // unsupported encoding
}
fmt.Printf("%d lamports owned by %s, %d bytes\n",
    res.Account.Lamports, res.Account.Owner, len(raw))
```

## Related

- [Account Methods](Account-Methods) — `GetAccountInfo` and
  `GetMultipleAccounts`.
- [Call Options](Call-Options) — `AccountInfoCfg{Encoding,
  DataSlice, Commitment}`.
