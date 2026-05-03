# Encoding Builder

The `encoding/` package layers three styles on top of one shared
`Encoder` / `Decoder`. Pick by scenario:

| Scenario | Write | Read |
|---|---|---|
| Fixed-shape instruction data | `encoding.New().U8(tag).U64(amount).Buf()` | `r := encoding.NewReader(data); v := r.U64(); ...` |
| Hand-crafted speed paths | `Encoder.WriteUint8` / `WriteBytes` / `WriteShortvec` (low-level) | `Decoder.ReadUint8` / `ReadBytes` / `ReadShortvec` |
| Reflection over `bin:"…"` tagged structs | not provided (always hand-write builders) | `encoding.DecodeTo(data, &v)` |

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
    Buf()

// 2. Precise capacity to avoid even the one-time grow.
encoding.NewEncoder(estimatedSize)

// 3. Pooled (sync.Pool) — for hot paths; remember to Release.
e := encoding.AcquireEncoder(64)
defer encoding.ReleaseEncoder(e)
```

Method conventions:

- numerics — `U8`, `U16`, `U32`, `U64`, `I8`, `I16`, `I32`,
  `I64`, `U128c`, `U256c`, `Bool`
- bytes-shaped — `Raw(b []byte)` (no length prefix),
  `Discriminator([8]byte)` (Anchor 8-byte tag),
  `Str(s string)` (bincode `u64`-prefixed)
- Rust `Option<T>` family — `OptU8`, `OptU32`, `OptU64`,
  `OptI64`, `OptU128`, `OptBool`, `OptRaw`: 1-byte tag +
  payload when non-nil. This is the **bincode** form of
  `Option<T>`. Token-2022 extensions that use spl-pod's
  `OptionalNonZeroPubkey` (32 bytes raw, zero == None) are
  not encoded with `OptRaw` — they emit raw 32-byte values
  with the zero pubkey representing absence.

Buffer length-prefixed `Vec<T>` is **not** wrapped because the
prefix differs across programs (bincode = `u64`,
Borsh = `u32`, ALT.Extend = `u32`, …); write the prefix
explicitly with `U32(uint32(n))` / `U64(uint64(n))` and loop
the elements.

Pubkeys in instruction data go through `Raw(pk[:])`; there is
intentionally no `Pubkey()` method on the encoder because the
encoding package must not import the root `solana` package
(cycle: root imports `encoding/`).

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
- `Skip(n)`, `Shortvec()`, `Str()`

`encoding.FromDecoder(d)` upgrades an existing `*Decoder`
(with established position) to a `Reader` for the rest of the
stream.

## Reflection-based decode

`encoding.DecodeTo(data, &v)` (and `Decoder.DecodeTo(&v)`)
walks `v` reflectively, respecting `bin:"…"` field tags. Use
this when:

- `v` is user-defined, possibly nested, with optional `*T`
  and slices.
- A one-shot decoder is more readable than 20 lines of
  `Reader` chains.

For pure performance, register a hand-written fast path with
`encoding.RegisterDecoder[T]` (the `*solana.PublicKey`,
`Hash`, `Signature`, `U128`, `U256` types are pre-registered).
Types implementing `encoding.Unmarshaler`
(`UnmarshalFromDecoder(*Decoder) error`) are also dispatched
directly without reflecting into their fields — this is how
`*Transaction`, `*Message`, `*Entry`, and the `Uint128/256`
types plug in.

`encoding.BorshDecodeTo(data, &v)` is the Borsh-specific
counterpart, with a per-decoder default prefix length so
nested `Vec<T>` and `Option<T>` decode with the right
prefix (`u32` for Borsh, `u64` for bincode).

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
