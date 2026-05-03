# Keys, Hashes, Signatures

The three fixed-size primitives that every Solana program uses
are modelled by three small array types in the root package.

## Types

| Type | Size | Purpose |
|---|---|---|
| `PublicKey` | 32 bytes | Account, program, or PDA address |
| `Hash` | 32 bytes | Blockhash or transaction hash |
| `Signature` | 64 bytes | Ed25519 signature |

They are all value types (`[N]byte` under the hood) so they are
stack-allocable, hashable as map keys, and comparable with `==`.
The canonical textual form for all three is base58, matching the
wire format every Solana tool uses.

## Size constants

```go
const (
    PublicKeySize = ed25519.PublicKeySize // 32
    HashSize      = 32
    SignatureSize = ed25519.SignatureSize // 64
)
```

## Constructing from raw bytes

```go
pk, err := solana.PublicKeyFromBytes(raw) // raw must be exactly 32
h,  err := solana.HashFromBytes(raw)      // 32
sig,err := solana.SignatureFromBytes(raw) // 64
```

A wrong-length slice returns an error wrapping `ErrInvalidLength`:

```go
var solana.ErrInvalidLength
// "solana: public key: solana: invalid length: got 31, want 32"
```

The constructors **never silently truncate or pad**. This is a
common bug in older Solana-Go ports; see the `TestPublicKeyFromBytes_InvalidLength`
tests for the covered edges.

## Constructing from base58

```go
pk, err := solana.PublicKeyFromBase58("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")
h,  err := solana.HashFromBase58("...")
sig,err := solana.SignatureFromBase58("...")
```

Errors wrap either `ErrInvalidBase58` (bad characters, empty
string) or `ErrInvalidLength` (decoded to wrong size).

## Accessors

All three types expose the same small surface:

```go
pk.Bytes()   // defensive copy of the raw bytes
pk.String()  // base58
pk.Equal(q)  // value equality
pk.IsZero()  // true iff all bytes are zero
```

`Bytes()` returns a fresh slice each call. Mutating the returned
slice does not affect the receiver. This is required by the
project's architectural principles: callers must never be able to
corrupt internal state by writing through a returned slice.

## Marshalling

All three types implement:

- `json.Marshaler` / `json.Unmarshaler` — renders as a JSON string
  holding the base58 form. A JSON `null` on unmarshal leaves the
  receiver unchanged.
- `encoding.TextMarshaler` / `encoding.TextUnmarshaler` — for
  tools like YAML, TOML, URL query strings, CLI flag parsing.
- `database/sql.Scanner` / `driver.Valuer` — round-trips as a
  base58 string in any SQL column that accepts `TEXT` or
  `VARCHAR`. Scanning from `nil` leaves the receiver unchanged.

```go
type Wallet struct {
    Owner solana.PublicKey `json:"owner"`
    Fee   solana.Hash      `json:"fee_reference_hash"`
}

var w Wallet
_ = json.Unmarshal([]byte(`{"owner":"Token..."}`), &w)
```

The JSON codec path is handwritten: it does not recursively call
`encoding/json` for the base58 string, and it never pulls
`encoding/json` onto the RPC hot path.

## Well-known addresses

For program IDs that you know at compile time (System program,
SPL Token, ATA, …), use `MustPublicKey`:

```go
var TokenProgramID = solana.MustPublicKey(
    "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")
```

`MustPublicKey` panics on failure and is only intended
for package-level initialisers with a hardcoded literal. **Never
use it with caller input.** See `must.go`.

## Related

- [Signers](Signers) — producing Ed25519 signatures for a key.
- [PDAs](PDAs) — deriving program-derived addresses.
- [Accounts](Accounts) — the richer on-chain state type.
