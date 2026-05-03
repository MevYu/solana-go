# Architectural Principles

These are **binding** rules for every pull request against the
repository. Most of them exist because a specific bug in an
older Go Solana library got through review, or because an
alternative ecosystem library got them wrong. Violating them
requires a motivated justification in the PR description and
usually ends with the PR being rewritten before merge.

## The ten principles

### 1. No `...interface{}` for typed arguments

Use **typed config structs** (`rpc.AccountInfoCfg`,
`rpc.SendTxCfg`, …) as the variadic last argument: methods accept
`cfg ...rpc.XxxCfg` so callers either omit the config entirely or
pass a single struct literal. We do not use `...interface{}`
variadic — it loses type information at the call site and makes
IDE autocomplete useless. We also do not use functional options
across the RPC API: a `WithCommitment` that 30 different methods
silently ignore is the exact opposite of the goal. See
[Call Options](Call-Options) for the full table of Cfg types.

Functional options remain in two places where they make sense:
constructor configuration (`jsonrpc.WithMaxIdleConnsPerHost(20)`,
`rpc.NewClientWith(...)`) and the send-and-confirm helper
(`rpc.WithSendCommitment(...)`), where the option's effect is
local to a single function instead of cross-cutting many methods.

### 2. No `panic` in library code

All "impossible" states return errors. The only exceptions are
`init()` and genuine programmer errors in test helpers.

`MustPublicKey` exists specifically for package-level
variable initialisers that hardcode well-known program IDs; it
is documented to panic and is explicitly **not** for caller
input. See `must.go`.

### 3. No silent argument-dropping

If the caller passes something the method can't honor, return
an error that **names the offending value**. Never silently
discard: bugs that "sometimes work and sometimes don't" are
the hardest to debug.

Example: `PublicKeyFromBytes` returns an error on a 31-byte or
33-byte input instead of truncating / padding. The older
`MevYu/go-solana` truncated silently.

### 4. Errors are wrapped with method context

A typical error reads:

```
solana rpc getBalance: context deadline exceeded
```

not:

```
context deadline exceeded
```

The RPC transport prepends `solana rpc <method>:` on every
error, and every helper wraps with `solana helpers: <fn>:`
or equivalent. The caller can always tell where the error
came from without a stack trace.

### 5. `context.Context` is the first argument of every public method that performs I/O

No exceptions. Cancellation must reach the transport layer,
the retry loop, and every remote signer. Local-only methods
(`PublicKeyFromBase58`, `(*Transaction).Marshal`) do not take
a context.

The `Signer.Sign` method takes a context even though the local
implementation (`Ed25519Keypair.Sign`) ignores it, specifically
so remote signers can enforce deadlines without
`Transaction.Sign` having to branch on signer type.

### 6. Public map and slice returns are defensive copies

Callers cannot mutate internal state through a returned value.
`PublicKey.Bytes()` returns a fresh slice each call:

```go
out := make([]byte, PublicKeySize)
copy(out, p[:])
return out
```

Similarly, `Ed25519Keypair.PrivateKey()` returns a defensive
copy so callers can zero the returned slice without touching
the signer's internal key material.

### 7. No `*big.Int` in hot paths

Lamports are `uint64`. If you think you need a bigger integer,
you're probably wrong. If you're right (Token-2022 amount
encoding, for example), it deserves a dedicated type with a
comment explaining why.

`u128` / `i128` in generated code are `[16]byte`; callers that
need arithmetic import `math/big` locally.

### 8. `encoding/json` is banned from the RPC hot path

Use a faster JSON codec. The default is
`github.com/goccy/go-json`, swappable via
`jsonrpc.WithCodec(jsonrpc.StdCodec())` if you cannot take the
dependency.

The `rpc` package's dependency on stdlib `encoding/json` is
confined to `codec_std.go`, a single file that is never
imported by the default build path. `Uint8Slice`,
`RawJSON`, and the `PublicKey` / `Hash` / `Signature`
marshallers do not recursively call `encoding/json` either —
they are handwritten byte-level parsers.

### 9. Every exported type has a doc comment naming the RPC method or protocol concept it represents

This is the contract that lets generated documentation on
pkg.go.dev actually describe the library. Exported types
without doc comments fail `go vet` style conventions and
should be rejected in review.

For Solana-specific types, the comment should name the RPC
method, wire-format concept, or runtime enum it models:

```go
// CompiledInstruction is an Instruction whose program and
// accounts have been resolved to indices into the enclosing
// Message's account key array.
type CompiledInstruction struct { ... }
```

### 10. Every performance-sensitive function has a benchmark

`sync.Pool` use must be justified by a benchmark showing the
win. Hot paths (binary encode/decode, JSON decode of RPC
responses, transaction signing) must be measured, not
assumed. See [Benchmarks](Benchmarks).

## Commit style

- Imperative, lowercase, no trailing period: `fix binary decoder off-by-one`.
- One logical change per commit. Architectural cleanups go
  in their own commits even when they touch many files.
- Every commit must leave `go build ./...`, `go vet ./...`,
  and `go test -race ./...` green.
- Breaking changes get a `BREAKING CHANGE:` footer in the
  commit body.

## Testing

- **Unit tests must run offline.** Tests that require a
  network endpoint live behind the `//go:build integration`
  tag and are skipped by the default `go test`.
- Use table-driven tests for edge cases; each table entry is
  a named subtest via `t.Run(tc.name, ...)`.
- Prefer deterministic seeds (`Ed25519KeypairFromSeed`) over
  `NewEd25519Keypair()` in tests so failures are reproducible.
- Coverage targets: >80% on core types (keys, message,
  transaction, binary), >60% elsewhere.

## Dependency policy

Every runtime dependency is a long-term maintenance cost.
Current dependencies:

- `github.com/mr-tron/base58` — base58 encoding, zero
  transitive deps.
- `github.com/goccy/go-json` — default JSON codec, zero
  transitive deps.
- `github.com/gorilla/websocket` — WebSocket transport for
  `ws.Client`.
- `github.com/tyler-smith/go-bip39` — BIP39 mnemonic seeds for
  `Ed25519KeypairFromMnemonic`.
- `filippo.io/edwards25519` — PDA on-curve check.
- `github.com/klauspost/compress` (transitive) — `base64+zstd`
  account-data decoding in `account_info.go`.

**Adding a new dependency requires a benchmark or
architectural justification in the PR that introduces it.**
Vendoring LGPL code (in particular, `go-ethereum`'s `rpc/`)
is explicitly forbidden — the transport layer in this repo
was written from scratch to avoid that encumbrance.

## Escape hatches

Three places have documented escape hatches for users who
need to bypass the typed layer:

- `*rpc.Client.CallContext(ctx, &out, method, params...)` —
  the raw JSON-RPC transport for methods not yet typed
  (promoted through `*rpc.Client` by embedding `*jsonrpc.Client`).
- `jsonrpc.CallContextValue[T](ctx, c.Client, method, params...)`
  — the same, but for context-wrapped responses.
- `token2022.Wrap(ix)` — substitute Token-2022's program id
  for any instruction that is byte-identical to its SPL
  Token counterpart.
- `jsonrpc.WithCodec(jsonrpc.StdCodec())` — drop the
  `goccy/go-json` dependency.

Using an escape hatch is not discouraged, but the design
direction is always "add a typed method eventually". File
an issue if you find a use case the typed layer does not yet
cover.

## Related

- [Benchmarks](Benchmarks) — runs and policy around
  measurement.
- [Error Handling](Error-Handling) — the wrap-with-context
  pattern in practice.
