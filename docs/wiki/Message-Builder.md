# Message Builder

`NewMessage` compiles a slice of typed `Instruction` values into
a legacy `Message` that is ready to sign. It is the function you
want in almost every real caller; hand-building a `Message`
struct is fine for tests and learning, but easy to get wrong.

```go
func NewMessage(
    payer PublicKey,
    instructions []Instruction,
    recentBlockhash Hash,
) (*Message, error)
```

## What it does

1. **Deduplicates** account keys across every instruction. The
   same pubkey mentioned twice counts once.
2. **Accumulates roles**. If account A is signer in one
   instruction and non-signer in another, A ends up in the
   signer bucket (roles are cumulative, never demoted).
3. **Orders the key array** by the four Solana role buckets:
   1. **s+w** — signer, writable (includes the payer, always
      first)
   2. **s+r** — signer, read-only
   3. **n+w** — non-signer, writable
   4. **n+r** — non-signer, read-only (program ids land here)
4. **Computes the header counters** from bucket sizes.
5. **Resolves each instruction's account list** to positional
   `Uint8Slice` indices into the final `AccountKeys`.

Within each bucket, accounts are ordered by the order they were
first seen. This matches `solana-web3.js` and produces
byte-for-byte stable output on repeated builds, which matters
for deterministic test fixtures.

## The payer is special

- The payer is always added first with role **signer + writable**,
  regardless of how individual instructions reference it.
- The payer always ends up at `AccountKeys[0]`.
- The payer is always the transaction's primary signer, so the
  signature at slot 0 of the eventual `Transaction.Signatures`
  matches it.

## Errors

`NewMessage` returns an error on:

- **No instructions** — every message must carry at least one.
- **A nil `Instruction`** — a slice entry that is `nil`.
- **A nil `AccountMeta`** — an instruction whose `Accounts()`
  list contains a `nil` pointer.
- **More than 255 account keys** — the wire format uses `uint8`
  indices and we refuse to emit an unencodable message.
- **An instruction whose `Data()` method returns an error** —
  the error is wrapped with the instruction's position so you
  can locate the faulty builder.

## Worked example: SOL transfer

```go
import (
    "github.com/MevYu/solana-go"
    "github.com/MevYu/solana-go/programs/system"
)

payer, _   := solana.NewEd25519Keypair()
recipient, _ := solana.PublicKeyFromBase58("...")

msg, err := solana.NewMessage(
    payer.PublicKey(),
    []solana.Instruction{
        system.NewTransfer(payer.PublicKey(), recipient, 1_000_000),
    },
    recentBlockhash,
)
```

Result:

- `AccountKeys[0]` = payer (s+w, always first)
- `AccountKeys[1]` = recipient (n+w — transfer writes to it)
- `AccountKeys[2]` = system program id (n+r — all program ids)
- `Header.NumRequiredSignatures` = 1
- `Header.NumReadonlyUnsignedAccounts` = 1 (the program id)
- The compiled instruction has `ProgramIDIndex = 2` and
  `Accounts = Uint8Slice{0, 1}`.

## Worked example: transfer + priority fee

```go
import (
    "github.com/MevYu/solana-go"
    "github.com/MevYu/solana-go/programs/system"
    computebudget "github.com/MevYu/solana-go/programs/compute-budget"
)

msg, err := solana.NewMessage(
    payer.PublicKey(),
    []solana.Instruction{
        computebudget.NewSetComputeUnitLimit(200_000),
        computebudget.NewSetComputeUnitPrice(5_000),
        system.NewTransfer(payer.PublicKey(), recipient, 1_000_000),
    },
    recentBlockhash,
)
```

Because the ComputeBudget instructions take no accounts, the
output has four keys (payer, recipient, systemProgram,
computeBudgetProgram) with the compute budget program in the
read-only non-signer bucket.

## v0 and Address Lookup Tables

`NewMessage` currently emits only legacy messages. A v0 builder
that resolves large account sets through Address Lookup Tables
is tracked as a follow-up; until it lands, construct v0 messages
by hand or use `programs/addresslookuptable` to manage the table yourself and
fill in `AddressTableLookups` manually.

## Related

- [Messages](Messages) — the output format and wire encoding.
- [System Program](System-Program) — the builders used in the
  examples.
- [Compute Budget](Compute-Budget-Program) — the priority-fee
  builders.
