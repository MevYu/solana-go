# System Program

The Solana **System program** handles the fundamental account
operations: creating new accounts, moving SOL, allocating
space, assigning ownership. Its address is
`11111111111111111111111111111111` — 32 zero bytes in base58.

Typed instruction builders live in
`github.com/MevYu/solana-go/programs/system`.

```go
import "github.com/MevYu/solana-go/programs/system"
```

## Program id

```go
var system.ProgramID = solana.PublicKey{} // 32 zero bytes
```

## Builders

Each builder returns a `solana.Instruction` ready to pass to
`solana.NewMessage`. All builders are pure — they encode the
instruction tag and data bytes without touching the network.

### `NewTransfer`

Moves lamports from one account to another.

```go
func NewTransfer(from, to solana.PublicKey, lamports uint64) solana.Instruction
```

- **`from`** — signer, writable. Source of the lamports.
- **`to`** — non-signer, writable. Destination. Both balances
  change, so both accounts are writable.

```go
ix := system.NewTransfer(payer.PublicKey(), recipient, 1_000_000)
```

### `NewCreateAccount`

Allocates a brand-new account, funds it with lamports, and
sets its owner program.

```go
func NewCreateAccount(
    from, newAccount solana.PublicKey,
    lamports, space uint64,
    owner solana.PublicKey,
) solana.Instruction
```

- **`from`** — signer, writable. Pays for the new account's
  rent lamports.
- **`newAccount`** — signer, writable. The new account's
  address. **Must be a fresh keypair** — typically generated
  via `solana.NewEd25519Keypair` and passed to `Transaction.Sign`
  alongside the payer.
- **`lamports`** — funding. Should be at least the rent-exempt
  minimum for the given `space` (look up via
  `c.GetMinimumBalanceForRentExemption`).
- **`space`** — byte size allocated for `newAccount.Data`.
- **`owner`** — the program that will control `newAccount`
  after creation (e.g. SPL Token program for a new mint).

```go
rent, _ := c.GetMinimumBalanceForRentExemption(ctx, 82)

ix := system.NewCreateAccount(
    payer.PublicKey(),
    newMint.PublicKey(),
    rent,
    82,           // mint account size
    token.ProgramID,
)
```

### `NewAssign`

Changes the owner program of an already-existing,
System-owned account.

```go
func NewAssign(account, owner solana.PublicKey) solana.Instruction
```

- **`account`** — signer, writable. Must currently be owned
  by the System program.
- **`owner`** — the new owner program.

### `NewAllocate`

Allocates `space` bytes for an account's data.

```go
func NewAllocate(account solana.PublicKey, space uint64) solana.Instruction
```

- **`account`** — signer, writable. Must currently be
  System-owned with zero-length data.

## Wire format

All System instructions use a **4-byte little-endian u32
tag** at the start of the data blob, matching the Solana
runtime's `SystemInstruction` enum:

| Tag | Instruction | Builder |
|---|---|---|
| 0 | CreateAccount | `NewCreateAccount` |
| 1 | Assign | `NewAssign` |
| 2 | Transfer | `NewTransfer` |
| 8 | Allocate | `NewAllocate` |

The tag values are `const`s in `instructions.go` and match
Solana upstream exactly.

## Building a transfer

The instruction itself is one line:

```go
ix := system.NewTransfer(payer.PublicKey(), recipient, 1_000_000)
```

Wrap it in a message + transaction + send-and-confirm using the
shared flow documented in [Getting Started](Getting-Started). For
fire-and-forget delivery use `c.SendTransaction(ctx, tx)`; for
"send, wait, refresh blockhash on expiry" use
`c.SendAndConfirmTransaction(ctx, builder, opts...)`.

## Related

- [Message Builder](Message-Builder) — compiles System
  instructions into a message.
- [Associated Token Account](Associated-Token-Account-Program)
  — uses `NewCreateAccount`-equivalent flow for token
  accounts.
- [Compute Budget](Compute-Budget-Program) — typically the
  first instruction in the list, before System transfers.
