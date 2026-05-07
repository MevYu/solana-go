# Address Lookup Tables

Address Lookup Tables (ALTs) are the mechanism v0 transactions
use to reference accounts without consuming signature space: a
single 32-byte lookup table address in the transaction resolves
to up to 256 accounts on chain. Programs like Jupiter and
Whirlpool route through far more accounts than a legacy
transaction can fit by maintaining one or more ALTs per pool.

Typed instruction builders live in
`github.com/MevYu/solana-go/programs/addresslookuptable`.

```go
import "github.com/MevYu/solana-go/programs/addresslookuptable"
```

## Program id

```go
var addresslookuptable.ProgramID = solana.MustPublicKey(
    "AddressLookupTab1e1111111111111111111111111")
```

## Typical lifecycle

1. **Create** the table at a recent slot, deriving its PDA.
2. **Extend** the table with the addresses you want to bundle.
3. **Use** the table via `MessageAddressTableLookup` in v0
   transactions.
4. **Freeze** the table once complete (optional, permanent).
5. Eventually **deactivate** then **close** to reclaim rent.

## `DeriveLookupTableAddress`

```go
func DeriveLookupTableAddress(
    authority solana.PublicKey,
    recentSlot uint64,
) (solana.PublicKey, uint8, error)
```

Computes the PDA for a lookup table owned by `authority` at
`recentSlot`. The seed order is
`[authority, recentSlot as u64 LE bytes]`, matching the
Solana runtime's `derive_lookup_table_address`.

Pass the PDA back as the first account meta to every
subsequent ALT instruction referring to this table.

## Builders

### `NewCreateLookupTable`

```go
func NewCreateLookupTable(
    authority, payer solana.PublicKey,
    recentSlot uint64,
) (solana.Instruction, solana.PublicKey, error)
```

Allocates a new address lookup table owned by `authority`.
`recentSlot` must be a slot the runtime still considers
"recent" — usually the current slot from `c.GetSlot` or
one of the last few.

The **derived table address** is returned alongside the
instruction so callers can immediately reference it in
subsequent extend / freeze calls. Both `authority` and `payer`
must sign the transaction.

```go
slot, _ := c.GetSlot(ctx)
createIx, table, err := addresslookuptable.NewCreateLookupTable(
    authority.PublicKey(),
    payer.PublicKey(),
    slot,
)
```

### `NewExtendLookupTable`

```go
func NewExtendLookupTable(
    lookupTable, authority, payer solana.PublicKey,
    newAddresses []solana.PublicKey,
) solana.Instruction
```

Appends `newAddresses` to the table. Both `authority` and
`payer` must sign; `payer` covers any additional rent required
by the larger account.

**A lookup table can hold up to 256 addresses total.** Split
large address sets across multiple extend calls to stay under
the per-transaction size limit (each extend fits roughly 30
addresses per 1232-byte packet).

### `NewFreezeLookupTable`

```go
func NewFreezeLookupTable(lookupTable, authority solana.PublicKey) solana.Instruction
```

Freezes a table, preventing further extends. **Permanent** —
there is no unfreeze. Only the `authority` can freeze.

### `NewDeactivateLookupTable`

```go
func NewDeactivateLookupTable(lookupTable, authority solana.PublicKey) solana.Instruction
```

Marks the table as deactivated. Deactivation does not close
the account; the table enters a cooling-off period (~500
slots) during which it cannot be referenced by new
transactions but its rent is still held. Only the `authority`
can deactivate.

### `NewCloseLookupTable`

```go
func NewCloseLookupTable(lookupTable, authority, recipient solana.PublicKey) solana.Instruction
```

Closes a **deactivated** lookup table and returns its rent
lamports to `recipient`. The table must be past the
deactivation cooling-off period or the runtime rejects the
call.

## Wire format

Unlike SPL Token (single-byte tags), ALT uses **u32 tags**
(Borsh enum discriminator form):

| Tag | Instruction |
|---|---|
| 0 | CreateLookupTable |
| 1 | FreezeLookupTable |
| 2 | ExtendLookupTable |
| 3 | DeactivateLookupTable |
| 4 | CloseLookupTable |

`Extend` uses Borsh's `Vec<PublicKey>` encoding: a u32 length
prefix followed by the addresses.

## Referencing a table from a v0 transaction

ALT usage in a transaction is via
`MessageAddressTableLookup` on the `Message`:

```go
msg := solana.Message{
    Version: solana.MessageVersion0,
    // ... header, AccountKeys, RecentBlockhash, Instructions ...
    AddressTableLookups: []solana.MessageAddressTableLookup{
        {
            AccountKey:      tablePubkey,
            WritableIndexes: solana.Uint8Slice{0, 3, 5},
            ReadonlyIndexes: solana.Uint8Slice{1, 2, 4},
        },
    },
}
```

The indices refer to positions **within the lookup table**.
The runtime resolves them to real pubkeys at execution time
and appends them to the transaction's effective account list
(after the static `AccountKeys`).

### Building a v0 message with ALTs

`solana.NewMessage` emits legacy messages. For v0 messages that
resolve accounts through one or more lookup tables, use
`solana.NewMessageV0`:

```go
msg, err := solana.NewMessageV0(
    payer.PublicKey(),
    instructions,
    recentBlockhash,
    []solana.LoadedAddressLookupTable{
        {AccountKey: tableAddr1, Addresses: ts1.Addresses},
        {AccountKey: tableAddr2, Addresses: ts2.Addresses},
    },
)
```

`LoadedAddressLookupTable` lives in the root `solana` package
(the encoding package can't import `solana` due to cycles, so
the type travels with the message API). Build it from a
`*addresslookuptable.TableState` returned by
`addresslookuptable.DecodeTableState(rawData)`.

The builder picks the static-vs-table split per account
automatically: required signers and writable accounts the
runtime cannot resolve through a table stay in `AccountKeys`,
read-only accounts present in any provided table move into
`AddressTableLookups`. The builder validates that the total
account count fits in `u8` and returns an error otherwise.

## Related

- [Messages](Messages) — the wire format the table plugs into.
- [Compute Budget](Compute-Budget-Program) — pair with ALTs
  for maximum instruction throughput per transaction.
- [PDAs](PDAs) — how the table address is derived.
