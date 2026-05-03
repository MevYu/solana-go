# Benchmarks

Performance is a first-class concern of this project. Every
performance-sensitive function ships with a benchmark you can
run with a single command, so claims in the README, release
notes, or commit messages are reproducible by anyone with this
repo and Go installed.

## Running the suite

```bash
# Everything
go test -run=^$ -bench=. -benchmem ./...

# Codec comparison (goccy vs stdlib)
go test -run=^$ -bench=BenchmarkCodec -benchmem ./jsonrpc/

# Phase A response-decode suite
go test -run=^$ -bench=BenchmarkGet -benchmem ./benchmarks/

# Transaction signing
go test -run=^$ -bench=BenchmarkTransaction_Sign -benchmem ./

# Binary codec primitives
go test -run=^$ -bench=. -benchmem ./encoding/
```

Each benchmark reports allocations via `b.ReportAllocs()`, so
output includes both ns/op and B/op.

## What's benchmarked

Grouped by package:

### Root package

- `BenchmarkMessage_Marshal_Legacy`
- `BenchmarkMessage_Unmarshal_Legacy`
- `BenchmarkTransaction_Sign_{1,4,8,16}` — the suffix is the
  number of required signers; proves the O(1) slot lookup
  scales linearly
- `BenchmarkTransaction_Marshal`
- `BenchmarkTransaction_Unmarshal`

### `encoding`

- `BenchmarkEncoder_Uint64` — scalar little-endian encode
- `BenchmarkEncoder_Uint64_Batch128` — tight-loop batch encode
- `BenchmarkShortvec_Encode`
- `BenchmarkShortvec_Decode_3Byte`

### `jsonrpc`

- `BenchmarkCodec_Decode_GoJSONCodec` — default codec on a
  synthetic block response
- `BenchmarkCodec_Decode_StdCodec` — stdlib `encoding/json`
  baseline
- `BenchmarkCodec_Encode_GoJSONCodec`
- `BenchmarkCodec_Encode_StdCodec`

### `benchmarks`

The `benchmarks/` package hosts dedicated tests that consume
checked-in fixtures under `benchmarks/testdata/`. Three Phase A
benchmarks are in place:

- `BenchmarkGetAccountInfo_Decode`
- `BenchmarkGetTransaction_Decode`
- `BenchmarkGetBlock_Decode`

The current fixtures are shape-accurate **synthetic** samples;
replacing them with real mainnet captures is a Phase A
follow-up.

## Why the `benchmarks/` package exists separately

Benchmark-only fixtures, target structs, and helper
generators would bleed into the shipping API if they lived in
the root `solana` package. Keeping them in a sibling package
under `benchmarks/` means:

- Fixtures never import into production builds.
- The root package stays small and focused.
- `go test ./benchmarks/…` can be run in isolation in CI.

## Running against a real endpoint

The default suite is offline — it reads fixtures from disk.
For integration timings against a real RPC endpoint, use the
`integration` build tag:

```bash
go test -tags integration -run=^Test -v ./...
```

Integration tests hit devnet and are skipped by default so
offline CI stays green.

## Adding a new benchmark

The project's architectural rule is that **every
performance-sensitive function has a benchmark**. When you
add one:

1. Put it next to the code it measures (e.g.
   `transaction_test.go` for `Transaction.Sign`).
2. Use `b.ReportAllocs()` and reset the timer after setup.
3. If the benchmark exists to justify a choice (for example,
   "`sync.Pool` is worth the complexity"), include a comment
   naming the alternative it beats and by how much.

## Related

- [Architectural Principles](Architectural-Principles) — the
  rule requiring every performance-sensitive function to ship
  with a benchmark.
