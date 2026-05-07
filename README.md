# solana-go

A high-performance Go SDK for the Solana blockchain.

## Install

```bash
go get github.com/MevYu/solana-go
```

**Minimum Go version**: 1.24.

---

## Packages at a glance

| Package | Purpose |
|---------|---------|
| `github.com/MevYu/solana-go` | Core types: `PublicKey`, `Hash`, `Signature`, `Transaction`, `Message`, `Keypair` |
| `…/jsonrpc` | JSON-RPC 2.0 transport: `Client.CallContext`, retry, codec, `Config`, `*ErrRPC`, classifiers |
| `…/rpc` | Typed wrappers for every stable RPC method: `*rpc.Client` with one Go method per RPC |
| `…/ws` | WebSocket subscriptions: `ws.Client`, `AccountSubscribe`, `LogsSubscribe`, … |
| `…/encoding` | Solana wire format: `shortvec`, bincode, Borsh, chained `Encoder`/`Reader` |
| `…/helpers` | Pure-logic helpers: `PriorityFeeStatsFromFees` |
| `…/programs/system` | System Program instructions (transfer, create-account, nonces, …) |
| `…/programs/token` | SPL Token instructions + state decoding |
| `…/programs/token2022` | Token-2022 instructions (incl. extension builders) + state decoding |
| `…/programs/associated-token-account` | ATA derivation + create-instruction builders |
| `…/programs/compute-budget` | Compute-budget instructions (priority fees) |
| `…/programs/memo` | Memo Program instruction builder |
| `…/programs/address-lookup-table` | ALT instruction builders + state decoder |
| `…/programs/stake` | Stake Program instructions |
| `…/programs/vote` | Vote Program instructions |
| `…/programs/secp256k1` | secp256k1 precompile (Ethereum-style sig verify) |

---

## Quickstart

### Parse a public key

```go
import "github.com/MevYu/solana-go"

pk, err := solana.PublicKeyFromBase58("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")
if err != nil {
    log.Fatal(err)
}
fmt.Println(pk) // TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA
```

### Generate a keypair

```go
kp, err := solana.NewEd25519Keypair()
if err != nil {
    log.Fatal(err)
}
fmt.Println("public key:", kp.PublicKey())
```

### Typed RPC client

```go
import (
    "context"
    "github.com/MevYu/solana-go/jsonrpc"
    "github.com/MevYu/solana-go/rpc"
)

c := rpc.NewClient("https://api.mainnet-beta.solana.com", jsonrpc.Config{})

// Get balance
res, err := c.GetBalance(ctx, pk)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("balance: %d lamports at slot %d\n", res.Value, res.Slot)

// Get account info — typed config struct, no functional options
info, err := c.GetAccountInfo(ctx, pk, rpc.AccountInfoCfg{
    Commitment: solana.CommitmentConfirmed,
    Encoding:   solana.EncodingBase64,
})

// Get latest blockhash
bh, err := c.GetLatestBlockhash(ctx)

// Send a transaction (returns the signature once the node accepts it)
sig, err := c.SendTransaction(ctx, tx)
```

All ~50 stable Solana RPC methods are available as typed methods on
`*rpc.Client`. Method options are typed `rpc.XxxCfg` structs (see
`jsonrpc/options.go`); each method documents the exact Cfg type it
accepts so unhonoured fields are a compile error rather than a silent
no-op.

### Build a transaction

```go
import (
    "github.com/MevYu/solana-go"
    "github.com/MevYu/solana-go/programs/system"
)

msg, err := solana.NewMessage(
    payer.PublicKey(),
    []solana.Instruction{system.NewTransfer(payer.PublicKey(), recipient, lamports)},
    blockhash,
)
if err != nil {
    log.Fatal(err)
}
t := solana.NewTransaction(*msg)
if err := t.Sign(ctx, payer); err != nil {
    log.Fatal(err)
}
```

### Send and confirm a transaction

```go
sig, err := c.SendAndConfirmTransaction(ctx,
    func(ctx context.Context, blockhash solana.Hash) (*solana.Transaction, error) {
        msg, _ := solana.NewMessage(payer.PublicKey(),
            []solana.Instruction{
                system.NewTransfer(payer.PublicKey(), recipient, lamports),
            },
            blockhash,
        )
        t := solana.NewTransaction(*msg)
        if err := t.Sign(ctx, payer); err != nil {
            return nil, err
        }
        return t, nil
    },
    rpc.WithSendCommitment(solana.CommitmentConfirmed),
)
```

`*rpc.Client.SendAndConfirmTransaction` handles blockhash refresh on
expiry and polls `getSignatureStatuses` until the requested commitment
is reached.

### Simulate a transaction

```go
import "errors"

sim, err := c.SimulateTransaction(ctx, tx)
if err != nil {
    log.Fatal(err) // transport error
}
if sim.Err != nil {
    decoded := rpc.DecodeTransactionError(sim.Err)
    var ie *rpc.InstructionError
    if errors.As(decoded, &ie) {
        fmt.Printf("instruction %d failed: %s\n", ie.Index, ie.Kind)
    }
}
if sim.UnitsConsumed != nil {
    fmt.Println("units consumed:", *sim.UnitsConsumed)
}
```

### Estimate priority fee

```go
import "github.com/MevYu/solana-go/helpers"

fees, err := c.GetRecentPrioritizationFees(ctx,
    []solana.PublicKey{writableAccount},
)
if err != nil {
    log.Fatal(err)
}
stats := helpers.PriorityFeeStatsFromFees(fees)
fmt.Printf("p50: %d  p75: %d  p95: %d  micro-lamports/CU\n",
    stats.P50, stats.P75, stats.P95)
```

### WebSocket subscriptions

```go
import (
    "github.com/MevYu/solana-go/rpc"
    "github.com/MevYu/solana-go/ws"
)

wsc, err := ws.DialWebSocket(ctx, "wss://api.mainnet-beta.solana.com")
if err != nil {
    log.Fatal(err)
}
defer wsc.Close()

sub, err := wsc.AccountSubscribe(ctx, pk, rpc.CommitmentWithEncodingCfg{
    Commitment: solana.CommitmentConfirmed,
})
if err != nil {
    log.Fatal(err)
}
defer sub.Unsubscribe(ctx)

for n := range sub.Recv() {
    fmt.Println("account updated at slot", n.Slot)
}
```

Available subscriptions: `AccountSubscribe`, `LogsSubscribe`,
`ProgramSubscribe`, `SignatureSubscribe`, `SlotSubscribe`,
`SlotsUpdatesSubscribe`, `RootSubscribe`, `BlockSubscribe`.

### Sign with a remote signer

For hardware wallets, cloud HSMs, or any networked signing service:

```go
remote := solana.NewRemoteSigner(
    walletPublicKey,
    func(ctx context.Context, message []byte) (solana.Signature, error) {
        return callMyHSM(ctx, message)
    },
)

tx.Sign(ctx, localPayer, remote)
```

### Raw JSON-RPC escape hatch

```go
import (
    "github.com/MevYu/solana-go/jsonrpc"
    "github.com/MevYu/solana-go/rpc"
)

c := rpc.NewClient("https://api.mainnet-beta.solana.com", jsonrpc.Config{})

// CallContext is a free generic function — Go forbids type parameters
// on methods. Pass the embedded *jsonrpc.Client as c.Client.
balance, err := jsonrpc.CallContext[uint64](
    ctx, c.Client, "getBalance",
    "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA",
)

// For {context, value} envelopes, instantiate T as ContextValue[X]:
resp, err := jsonrpc.CallContext[jsonrpc.ContextValue[uint64]](
    ctx, c.Client, "getBalance",
    "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA",
)
// resp.Context.Slot, resp.Value
```

---

## What's included

| Area | Status |
|------|--------|
| Core types (`PublicKey`, `Hash`, `Signature`, `Account`) — base58 / JSON / text / SQL | ✅ |
| Message encoding: legacy and v0 with Address Lookup Tables | ✅ |
| `Transaction`: O(1) signer lookup, incremental multi-party signing | ✅ |
| `Signer`: local `Ed25519Keypair`, BIP-39 mnemonic, function-adapted `RemoteSigner` | ✅ |
| Zero-copy binary codec with Solana `shortvec` / Borsh support | ✅ |
| `jsonrpc.Client`: JSON-RPC 2.0, pluggable codec, exponential-backoff retry, classifiers | ✅ |
| ~50 typed RPC methods on `*rpc.Client` | ✅ |
| WebSocket subscriptions (8 subscription types) on `*ws.Client` | ✅ |
| `SendAndConfirmTransaction` with blockhash refresh | ✅ |
| `DecodeTransactionError` typed error decoding for `SimulateTransaction` / `GetTransaction` | ✅ |
| `PriorityFeeStatsFromFees` percentile statistics | ✅ |
| Program bindings: System, SPL Token, Token-2022 (incl. extensions), ATA, Compute Budget, Memo, ALT, Stake, Vote, Secp256k1 | ✅ |

---

## Performance design

- **Single-pass JSON decode** for all `{context, value}` responses via
  `jsonrpc.CallContext[ContextValue[T]]` — halves codec work on the hot path.
- **Tagged public types** (`SimulateResult`, `LatestBlockhash`,
  `SupplyResult`, `GetTransactionResult`, WS notifications, …) decode
  straight from the wire — no internal wire-shape struct middleman.
- **Precise encoder pre-allocation** in `Message.Marshal` and
  `Transaction.Marshal` — eliminates buffer growth and copy-out.
- **Lazy `*time.Timer` + `defer Stop()`** in all retry and poll loops
  — no goroutine leaks on context cancellation.
- **Hoisted seed-augment slice** in `FindProgramAddress` — up to 256
  redundant allocations per call collapse to zero.

---

## Architectural principles

1. **Typed config structs over functional options for RPC methods.**
   Every method takes `cfg ...rpc.XxxCfg` so unhonoured fields are a
   compile error, not a silent no-op. Functional options remain only
   for constructor configuration (`jsonrpc.WithMaxIdleConnsPerHost`)
   and the send-and-confirm helper (`rpc.WithSendCommitment`).
2. **No `panic` in library code.** All "impossible" states return
   errors. `MustPublicKey` / `MustHash` / `MustSignature` exist
   specifically for package-level program-id constants and explicitly
   panic — never use them on user input.
3. **No silent argument-dropping.** Errors name the offending value.
4. **Errors are wrapped with method context.** A typical error reads
   `solana rpc getBalance: …`, not a bare `Forbidden`.
5. **`context.Context` is the first argument of every public method
   that does I/O.**
6. **Public map and slice returns are defensive copies.** Callers
   cannot mutate internal state through a returned value.
7. **No `*big.Int` in hot paths.** Lamports are `uint64`.
8. **`encoding/json` is not on the RPC hot path.** Default codec is
   `goccy/go-json`; swap with `jsonrpc.WithCodec(jsonrpc.StdCodec())`
   if you can't take the dependency.
9. **Every exported type has a doc comment** naming the RPC method or
   protocol concept it represents.
10. **Hard caller-input limits up front.** Methods enforce server
    caps locally (`MaxGetMultipleAccountsAddresses`,
    `MaxGetSignatureStatusesSignatures`,
    `MaxGetRecentPrioritizationFeesAddresses`) so callers see a
    precise error instead of a vague "Invalid params" from the
    server.

---

## Dependencies

| Dependency | Purpose |
|-----------|---------|
| [`filippo.io/edwards25519`](https://github.com/FiloSottile/edwards25519) | PDA off-curve check |
| [`github.com/goccy/go-json`](https://github.com/goccy/go-json) | Fast JSON codec (default) |
| [`github.com/gorilla/websocket`](https://github.com/gorilla/websocket) | WebSocket transport |
| [`github.com/mr-tron/base58`](https://github.com/mr-tron/base58) | Base58 encoding |
| [`github.com/tyler-smith/go-bip39`](https://github.com/tyler-smith/go-bip39) | BIP-39 mnemonic seeds |
| [`github.com/klauspost/compress`](https://github.com/klauspost/compress) | `base64+zstd` account-data decoding (transitive) |

All MIT or BSD licensed.

---

## API documentation

Full API documentation: <https://pkg.go.dev/github.com/MevYu/solana-go>

---

## Contributing

- `go build ./...` must be green
- `go vet ./...` must be green
- `go test -race ./...` must be green
- Every new exported identifier needs a doc comment
- Performance-sensitive changes should ship with a benchmark

## License

Apache-2.0. See [LICENSE](LICENSE).
