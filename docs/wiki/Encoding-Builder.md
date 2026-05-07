# Encoding Builder

The `encoding/` package layers three styles on top of one shared
`Encoder` / `Decoder`. Pick by scenario:

| Scenario | Write | Read |
|---|---|---|
| Fixed-shape instruction data | `encoding.New().U8(tag).U64(amount).Bytes()` | `r := encoding.NewReader(data); v := r.U64(); ...` |
| Hand-crafted speed paths | `Encoder.WriteUint8` / `WriteBytes` / `WriteShortvec` (low-level) | `Decoder.ReadUint8` / `ReadBytes` / `ReadShortvec` |
| Reflective decode of plain Go structs | not provided (always hand-write builders) | `encoding.BinDecodeTo(data, &v)` (bincode) / `encoding.BorshDecodeTo(data, &v)` (Borsh) |

The chained `Encoder` (in `encoding/builder.go`) and `Reader`
(in `encoding/reader.go`) are wrappers around `Encoder` /
`Decoder` that expose every primitive as a one-shot method
returning the receiver (encoder) or value (reader). They do
not add a separate code path — all writes still go through
`Encoder.WriteUint*` / `Encoder.WriteBytes`.

## Encoder (writer)

```go
import "github.com/MevYu/solana-go/encoding"

// 1. Default 128-byte capacity — covers virtually every instruction.
data := encoding.New().
    U32(tag).
    U64(amount).
    Bool(flag).
    Raw(pubkey[:]).
    Bytes()

// 2. Precise capacity to avoid even the one-time grow.
e := encoding.NewEncoder(estimatedSize)
```

Method conventions:

- numerics — `U8`, `U16`, `U32`, `U64`, `I8`, `I16`, `I32`,
  `I64`, `U128c`, `U256c`, `Bool`
- bytes-shaped — `Raw(b []byte)` (no length prefix),
  `StrU64(s string)` (bincode `u64`-prefixed string)
- Rust `Option<T>` family — `OptU8`, `OptU32`, `OptU64`,
  `OptI64`, `OptBool`, `OptRaw`: 1-byte tag +
  payload when non-nil. This is the **bincode** form of
  `Option<T>`. Token-2022 extensions that use spl-pod's
  `OptionalNonZeroPubkey` (32 bytes raw, zero == None) are
  not encoded with `OptRaw` — they emit raw 32-byte values
  with the zero pubkey representing absence.

Length-prefixed `Vec<T>` is **not** wrapped because the
prefix differs across programs (bincode = `u64`,
Borsh = `u32`, ALT.Extend = `u32`, …); write the prefix
explicitly with `U32(uint32(n))` / `U64(uint64(n))` and loop
the elements.

Pubkeys in instruction data go through `Raw(pk[:])`; there is
intentionally no `Pubkey()` method on the encoder because the
encoding package must not import the root `solana` package
(cycle: root imports `encoding/`).

`Encoder.Bytes()` returns the buffer; `Encoder.Len()` reports
the current length. The package exposes no `sync.Pool` —
callers that allocate one `Encoder` per instruction are
already on the fast path (escape analysis stack-allocates
common cases; pooling did not pay off in benchmarks).

## Reader (sticky-error decoder)

```go
r := encoding.NewReader(data[:headerSize])
flag := r.U32()
mint := solana.PublicKey(r.Bytes32())   // [32]byte returned by value
supply := r.U64()
decimals := r.U8()
if err := r.Err(); err != nil {
    return nil, err
}
// or, to enforce no trailing bytes:
if err := r.Done(); err != nil {
    return nil, err   // *TrailingBytesError on extra bytes
}
```

The first short-buffer error is sticky: subsequent reads return
zero values, so callers sequence the happy path without per-call
err checks. `Err()` returns the sticky error; `Done()`
additionally requires the buffer was exhausted (returns
`*TrailingBytesError`). Common helpers:

- `Bytes32()` / `Bytes64()` for Pubkey/Signature-shaped fields
- `Bytes(n)` for variable-length slices
- `Read(out []byte)` to fill an existing destination
- `Skip(n)`, `StrU64()`

`Reader.Decoder()` returns the underlying `*Decoder`; `r.Pos()`
and `r.Remaining()` forward to it.

## Reflection-based decode

`encoding.BinDecodeTo(data, &v)` (bincode, u64-prefixed
`Vec<T>` / `String`) and `encoding.BorshDecodeTo(data, &v)`
(Borsh, u32-prefixed) walk `v` reflectively, decoding fields
in declaration order according to their Go types. Use these
when:

- `v` is user-defined, possibly nested, with optional `*T`
  and slices.
- A one-shot decoder is more readable than 20 lines of
  `Reader` chains.

`NewDecoder(b).UseBorsh().DecodeTo(&v)` is the lower-level
form when you already hold a `*Decoder` and want to switch
prefix width. `BinDecodeTo` and `BorshDecodeTo` differ
**only** in the `Vec<T>` / `String` length-prefix width
(u64 vs u32); everything else (struct walking, fixed-size
arrays, primitives) is identical.

There are no struct tags. Field order in the Go struct must
match the wire layout. Unexported fields are skipped (handy
for in-memory bookkeeping); for an exported field that should
not be on the wire, use a wrapper struct that holds only the
serialised subset.

For pure performance, types implementing
`encoding.Unmarshaler` (`UnmarshalFromDecoder(*Decoder) error`)
are dispatched directly without reflecting into their fields —
this is how `*Transaction`, `*Message`, `*Entry`, and the
`Uint128/256` types plug in. Implement `Unmarshaler` on a type
you control to bypass the reflective plan.

Reflection-based **encode** is intentionally not provided;
all instruction builders are hand-written in
`programs/*/instructions.go`, which keeps the wire layout
obvious next to the Solana runtime source.

## Streaming transaction decode

`UnmarshalTransaction(data)` and `UnmarshalMessage(data)` take
a whole-buffer view and reject trailing bytes. To decode a
transaction out of the middle of a larger stream, use the
decoder-position variants:

- `DecodeTransaction(d *encoding.Decoder) (*Transaction, error)`
  — reads one transaction from the current position, advances
  the cursor.
- `DecodeMessage(d *encoding.Decoder) (*Message, error)` —
  same for Message; does not enforce "no trailing bytes"
  (that's the stream caller's job).

This unlocks Jito ShredStream entry decoding, where a single
buffer contains a `Vec<Entry>` with each `Entry.Transactions`
holding a variable-length sequence of transactions. Use
`solana.DecodeEntries(data)` for that case.

## Length constants

Always reference `solana.PublicKeySize` (= 32),
`solana.HashSize` (= 32), and `solana.SignatureSize` (= 64)
instead of literal magic numbers in arithmetic and length
checks. Wire-layout-documenting offsets (e.g. `data[4:36]`
inside a known 82-byte SPL Mint) may keep the literals when
they make the layout clearer.
