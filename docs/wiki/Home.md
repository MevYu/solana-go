# solana-go

Welcome to the `github.com/MevYu/solana-go` wiki — the long-form
documentation for a high-performance Go SDK for the Solana
blockchain.

The wiki tracks the current `main` branch.

## What this SDK gives you

- **Core primitives** — `PublicKey`, `Hash`, `Signature`, `Account`
  with zero-allocation base58, JSON, text, and SQL serialisation.
- **Transactions and messages** — legacy and v0 wire formats,
  builder that compiles typed instructions into a positional
  account layout, and a `Signer` interface that accepts both local
  `Ed25519Keypair` and function-adapted `RemoteSigner` (HSM /
  Ledger / cloud KMS).
- **Typed JSON-RPC client** — `rpc.Client` (backed by
  `rpc.Client` transport) exposes one Go method per stable Solana
  RPC method, with typed functional options for commitment /
  encoding / pagination, and a pluggable codec that defaults to
  `goccy/go-json`. The generic helper `jsonrpc.CallContextValue[T]`
  decodes `{context, value}` envelopes in a single pass without
  per-method boilerplate, keeping public result types free of JSON
  tags.
- **WebSocket subscriptions** — `ws.Client` exposes
  `AccountSubscribe`, `LogsSubscribe`, `SlotSubscribe`,
  `RootSubscribe`, `ProgramSubscribe`, `SignatureSubscribe`,
  `BlockSubscribe`, `SlotsUpdatesSubscribe`, all with buffered
  channels and a drop-oldest back-pressure policy.
- **Program bindings** — System, SPL Token, Token-2022 (incl.
  extensions), Associated Token Account, Address Lookup Tables,
  Compute Budget, Memo, Stake, Vote, Secp256k1.
- **High-level helpers** — `*rpc.Client.SendAndConfirmTransaction`
  with automatic blockhash refresh, `helpers.PriorityFeeStatsFromFees`
  percentile statistics,
  `SimulateTransactionDecoded` with typed `InstructionError`.

## Where to start

- **New to the SDK?** Read **[Getting Started](Getting-Started)**.
- **Building a transaction?** See
  [Transactions](Transactions) and
  [Message Builder](Message-Builder).
- **Calling an RPC method?** See [RPC Client](RPC-Client) and the
  per-method pages in the sidebar.
- **Streaming updates?** See [WebSocket Client](WebSocket-Client).
- **Contributing?** Read
  [Architectural Principles](Architectural-Principles) first.

## Design philosophy

The project is a clean-room rewrite with a hard bias toward:

1. **Performance that is measurable** — every claim has a
   checked-in benchmark; see [Benchmarks](Benchmarks).
2. **API honesty** — typed `rpc.XxxCfg` config structs (not
   `...interface{}`), no silent argument dropping, no `panic` in
   library code, errors wrapped with method context,
   `context.Context` threaded through every I/O.
3. **Minimal dependencies** — five runtime deps, all MIT or BSD.
4. **No LGPL code** — the RPC transport is our own, not a
   `go-ethereum` port.

## Links

- Source: <https://github.com/MevYu/solana-go>
- API reference: <https://pkg.go.dev/github.com/MevYu/solana-go>
- License: Apache-2.0
