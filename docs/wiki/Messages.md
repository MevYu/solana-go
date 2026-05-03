# Messages (legacy vs v0)

A Solana **Message** is the unsigned body of a transaction: a
header, an array of account keys, a recent blockhash, and a list
of compiled instructions. The SDK supports both wire formats —
legacy and v0 — through a single `Message` struct.

## The struct

```go
type Message struct {
    Version             MessageVersion
    Header              MessageHeader
    AccountKeys         []PublicKey
    RecentBlockhash     Hash
    Instructions        []CompiledInstruction
    AddressTableLookups []MessageAddressTableLookup
}
```

### `MessageVersion`

```go
const (
    MessageVersionLegacy MessageVersion = 0xFF // in-memory sentinel
    MessageVersion0      MessageVersion = 0    // wire value 0x80 | 0
)
```

- **Legacy** messages have no version prefix byte on the wire.
  Their serialised form begins directly with the header.
- **v0** messages prefix their first byte with `0x80`, leaving the
  low 7 bits for the version number. Bumping to v1 is a matter of
  adding a new constant and extending `validate` / `Marshal` /
  `UnmarshalMessage`.

The `MessageVersionLegacy` sentinel is `0xFF`, a value that can
never appear on the wire, so "legacy" and "versioned" never get
confused in memory.

### `MessageHeader`

```go
type MessageHeader struct {
    NumRequiredSignatures       uint8
    NumReadonlySignedAccounts   uint8
    NumReadonlyUnsignedAccounts uint8
}
```

The header tells the runtime where to draw the four role boundaries
within `AccountKeys`. See [Message Builder](Message-Builder) for
how `NewMessage` computes these counters automatically.

### `CompiledInstruction`

```go
type CompiledInstruction struct {
    ProgramIDIndex uint8
    Accounts       Uint8Slice // JSON-serialises as number array
    Data           []byte
}
```

`Accounts` is a `Uint8Slice`, not a `[]byte`, because Solana JSON-RPC
encodes these indices as a JSON array of integers (not a base64
string). `Uint8Slice` has a handwritten `MarshalJSON` /
`UnmarshalJSON` that bypasses Go's `[]byte` → base64 special case.

### `MessageAddressTableLookup` (v0 only)

```go
type MessageAddressTableLookup struct {
    AccountKey      PublicKey
    WritableIndexes Uint8Slice
    ReadonlyIndexes Uint8Slice
}
```

On v0 messages, addresses past the static `AccountKeys` list are
resolved through on-chain Address Lookup Tables. See
[Address Lookup Tables](Address-Lookup-Tables) for the full story.

## Wire format

A legacy message is:

```
[header (3 bytes)] [shortvec account_keys] [recent_blockhash (32)]
[shortvec instructions (program_idx, shortvec accounts, shortvec data)]
```

A v0 message prepends a single version prefix byte and appends a
shortvec list of address table lookups after the instructions.

`shortvec` is Solana's compact-u16 encoding (1–3 bytes). See
`internal/binary/shortvec.go`.

## `Marshal` and `UnmarshalMessage`

```go
data, err := msg.Marshal()              // []byte, error
out, err := solana.UnmarshalMessage(data) // *Message, error
```

The marshaller and unmarshaller have three strict properties:

1. **Trailing-byte rejection**: `UnmarshalMessage` checks
   `d.Remaining() != 0` after parsing the last field and returns
   an error if there is leftover data. A corrupted wire that
   happens to decode into a well-formed message prefix cannot
   silently drop its tail.
2. **No input aliasing**: every variable-length field is copied
   out of the decoder before returning, so callers may mutate or
   free the input buffer after `UnmarshalMessage` returns.
3. **Explicit version rejection**: `validate()` rejects legacy
   messages that carry ALT lookups (an obvious data bug) and
   `UnmarshalMessage` rejects versions past `maxMessageVersion`.

## Building a message by hand

```go
systemProgram, _ := solana.PublicKeyFromBase58("11111111111111111111111111111111")
msg := solana.Message{
    Version: solana.MessageVersionLegacy,
    Header: solana.MessageHeader{
        NumRequiredSignatures:       1,
        NumReadonlyUnsignedAccounts: 1,
    },
    AccountKeys: []solana.PublicKey{
        payer.PublicKey(),
        recipient,
        systemProgram,
    },
    RecentBlockhash: recentBlockhash,
    Instructions: []solana.CompiledInstruction{
        {
            ProgramIDIndex: 2,
            Accounts:       solana.Uint8Slice{0, 1},
            Data:           transferData,
        },
    },
}
```

Hand-constructing is fine for learning and tests, but for real
code prefer the builder in [Message Builder](Message-Builder),
which handles deduplication, role ordering, and index resolution
correctly.

## Related

- [Message Builder](Message-Builder) — compiling typed
  `Instruction` values into a ready-to-sign `Message`.
- [Transactions](Transactions) — wrapping a `Message` with
  signatures.
- [Address Lookup Tables](Address-Lookup-Tables) — the program
  that makes v0 messages interesting.
