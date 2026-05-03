# Associated Token Account Program

The **Associated Token Account** (ATA) program canonicalises
the mapping from a `(wallet, mint)` pair to a deterministic
token account address. Any wallet holding balances in a given
mint uses its ATA; any dapp sending tokens to a wallet
addresses the wallet's ATA without coordination.

Typed instruction builders live in
`github.com/MevYu/solana-go/programs/associated-token-account`.

```go
import "github.com/MevYu/solana-go/programs/associated-token-account"
```

## Program id

```go
var associatedtokenaccount.ProgramID = solana.MustPublicKey(
    "ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL")
```

## Address derivation

```go
func FindAssociatedTokenAddress(
    wallet, mint, tokenProgram solana.PublicKey,
) (solana.PublicKey, uint8, error)
```

Derives the ATA for `(wallet, mint)` under the given token
program. The seed order is `[wallet, tokenProgram, mint]`
with `associatedtokenaccount.ProgramID` as the program id. Pass
`token.ProgramID` for classic SPL Token and
`token2022.ProgramID` for Token-2022.

```go
import (
    "github.com/MevYu/solana-go/programs/associated-token-account"
    "github.com/MevYu/solana-go/programs/token"
)

addr, bump, err := associatedtokenaccount.FindAssociatedTokenAddress(
    walletPubkey, mintPubkey, token.ProgramID,
)
```

The PDA is deterministic: anybody who knows `(wallet, mint,
tokenProgram)` derives the same address without coordination.
See [PDAs](PDAs) for how the derivation works under the hood.

## Instruction builders

### `NewCreate`

```go
func NewCreate(
    payer, wallet, mint, tokenProgram solana.PublicKey,
) (solana.Instruction, error)
```

Allocates a new associated token account for `(wallet, mint)`
and funds it with rent-exempt lamports paid by `payer`. The
ATA is derived internally, so callers do not pass it.

**Fails if the account already exists.** For retry-heavy flows,
use `NewCreateIdempotent` instead.

### `NewCreateIdempotent` (recommended)

```go
func NewCreateIdempotent(
    payer, wallet, mint, tokenProgram solana.PublicKey,
) (solana.Instruction, error)
```

Same as `NewCreate` but **succeeds without error when the ATA
already exists**. This is the safer default and the one to use
when you are not sure whether the ATA has been created yet.

## Standard account layout

Both builders emit the same six-account meta list:

1. `payer` — signer, writable (pays rent)
2. ATA address — non-signer, writable (the account being
   created / checked)
3. `wallet` — non-signer, read-only (owner of the new ATA)
4. `mint` — non-signer, read-only
5. System program — non-signer, read-only
6. Token program — non-signer, read-only

The ATA program PDA-signs the CPI into the Token program, so
no extra signers are needed beyond the payer.

## The discriminator difference

- `Create` has empty instruction data (`[]byte{}`).
- `CreateIdempotent` has a single-byte data blob (`[]byte{1}`).

Internally the two builders differ only in that byte.

## Example: send tokens to a wallet you don't control

The canonical "send tokens to anybody" flow is two instructions:

```go
import (
    associatedtokenaccount "github.com/MevYu/solana-go/programs/associated-token-account"
    "github.com/MevYu/solana-go/programs/token"
)

// 1. Compute or create the recipient's ATA idempotently.
createATA, _ := associatedtokenaccount.NewCreateIdempotent(
    payer.PublicKey(), recipientWallet, mint, token.ProgramID,
)
recipientATA, _, _ := associatedtokenaccount.FindAssociatedTokenAddress(
    recipientWallet, mint, token.ProgramID,
)

// 2. Transfer from our own ATA (computed the same way).
ourATA, _, _ := associatedtokenaccount.FindAssociatedTokenAddress(
    payer.PublicKey(), mint, token.ProgramID,
)
transfer := token.NewTransferChecked(
    ourATA, mint, recipientATA, payer.PublicKey(),
    amount, decimals,
)
```

Because `NewCreateIdempotent` succeeds even when the account
exists, you can always include it at the head of the instruction list
without conditional logic. The extra compute is negligible if the
account is already there.

Compose `[createATA, transfer]` into a transaction and send via the
shared flow in [Getting Started](Getting-Started).

## Related

- [SPL Token Program](SPL-Token-Program) — the program the
  ATA dispatches to via CPI.
- [Token-2022 Program](Token-2022-Program) — pass
  `token2022.ProgramID` instead of `token.ProgramID`.
- [PDAs](PDAs) — the derivation mechanism.
