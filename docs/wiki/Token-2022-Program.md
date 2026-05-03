# Token-2022 Program

**Token-2022** is the extension-capable successor to the
classic SPL Token program. Its address is
`TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb`.

Typed instruction builders live in
`github.com/MevYu/solana-go/programs/token2022`.

```go
import "github.com/MevYu/solana-go/programs/token2022"
```

## Why a separate package?

Token-2022 supports the same core instructions as classic SPL
Token (Transfer, TransferChecked, MintTo, Burn, CloseAccount,
InitializeMint2, InitializeAccount3) **with byte-identical wire
formats**. The only difference is the program id the
transaction dispatches on.

Because of that, the `token2022` package reuses the builders
from `programs/token` via a thin `wrapped` adapter that
substitutes Token-2022's program id. A single-word change at
the call site switches between the two:

```go
// Classic SPL Token
ix := token.NewTransferChecked(src, mint, dst, auth, amount, decimals)

// Token-2022
ix := token2022.NewTransferChecked(src, mint, dst, auth, amount, decimals)
```

## Program id

```go
var token2022.ProgramID = solana.MustPublicKey(
    "TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb")
```

The id is verified against the `declare_id!` call in the
`spl_token_2022` interface crate, not copied from folklore.

## Covered builders

All of these exist with the same signatures as in
[SPL Token](SPL-Token-Program):

- `NewTransfer`
- `NewTransferChecked`
- `NewMintTo`
- `NewBurn`
- `NewCloseAccount`
- `NewInitializeMint2`
- `NewInitializeAccount3`

The account metas, data bytes, and tags are identical; only
`ProgramID()` differs.

## The `Wrap` escape hatch

For instructions that are not yet covered by a typed builder
in `token2022` but are byte-identical to their SPL Token
counterparts, wrap the classic builder directly:

```go
import (
    "github.com/MevYu/solana-go"
    "github.com/MevYu/solana-go/programs/token"
    "github.com/MevYu/solana-go/programs/token2022"
)

rawIx := token.NewSomeInstruction(...)
ix := token2022.Wrap(rawIx) // now targets Token-2022's program id
```

`Wrap` substitutes the program id without touching accounts
or data.

## Extensions

Token-2022's real selling point is **extensions** — mint-level
or account-level features that classic SPL Token cannot
express. Typed builders for the most commonly used extensions
are exported from `programs/token2022`:

| Extension | Builders |
|---|---|
| Mint close authority | `NewInitializeMintCloseAuthority` |
| Non-transferable mint | `NewInitializeNonTransferableMint` |
| Permanent delegate | `NewInitializePermanentDelegate` |
| Default account state | `NewInitializeDefaultAccountState`, `NewUpdateDefaultAccountState` |
| Transfer fee | `NewInitializeTransferFeeConfig`, `NewSetTransferFee`, `NewWithdrawWithheldTokensFromMint`, `NewWithdrawWithheldTokensFromAccounts`, `NewHarvestWithheldTokensToMint` |
| Interest-bearing mint | `NewInitializeInterestBearingMint`, `NewUpdateInterestBearingMintRate` |
| Metadata pointer | `NewInitializeMetadataPointer`, `NewUpdateMetadataPointer` |
| Transfer hook | `NewInitializeTransferHook`, `NewUpdateTransferHook` |
| Required memo / CPI guard | `NewEnableRequiredMemoTransfers`, `NewDisableRequiredMemoTransfers`, `NewEnableCpiGuard`, `NewDisableCpiGuard` |

Most extension authority/address fields use spl-pod's
`OptionalNonZeroPubkey` on the wire: a fixed 32 bytes where the
all-zero pubkey represents `None`. Pass `nil` to the builder for
"no authority" and the encoder will emit the zero pubkey
automatically.

`ConfidentialTransfer` is intentionally **not** wrapped: the
zero-knowledge proof generation is out of scope for an SDK that
otherwise has zero crypto dependencies beyond ed25519.

Layouts for already-deployed mint and account state are read by
`token2022.DecodeMint` / `token2022.DecodeAccount`, which walk
the TLV extension data appended after the 165-byte base layout.

## Interop with ATA

Associated Token Accounts work for both token programs. Pass
the right program id to
`associatedtokenaccount.FindAssociatedTokenAddress` /
`associatedtokenaccount.NewCreateIdempotent`:

```go
addr, _, _ := associatedtokenaccount.FindAssociatedTokenAddress(
    wallet, mint, token2022.ProgramID,
)

createIx, _ := associatedtokenaccount.NewCreateIdempotent(
    payer, wallet, mint, token2022.ProgramID,
)
```

See [Associated Token Account](Associated-Token-Account-Program).

## Related

- [SPL Token Program](SPL-Token-Program) — the classic
  program these builders wrap.
- [Associated Token Account](Associated-Token-Account-Program)
  — supports both token programs.
