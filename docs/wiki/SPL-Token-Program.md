# SPL Token Program

The **SPL Token** program is Solana's fungible and non-fungible
token standard. Its address is
`TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA`.

Typed instruction builders live in
`github.com/MevYu/solana-go/programs/token`.

```go
import "github.com/MevYu/solana-go/programs/token"
```

For the newer **Token-2022** program (extension-capable
successor), see [Token-2022 Program](Token-2022-Program).

## Program id

```go
var token.ProgramID = solana.MustPublicKey(
    "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")
```

## Builders

The package covers the common hot-path operations. Each builder
returns a `solana.Instruction` ready for `solana.NewMessage`.

### `NewTransfer` and `NewTransferChecked`

```go
func NewTransfer(source, destination, authority solana.PublicKey, amount uint64) solana.Instruction

func NewTransferChecked(
    source, mint, destination, authority solana.PublicKey,
    amount uint64, decimals uint8,
) solana.Instruction
```

`source` and `destination` are **token account** addresses
(not wallet addresses — see
[Associated Token Account](Associated-Token-Account-Program)
for the mapping). `authority` owns the source account and must
sign.

**Prefer `NewTransferChecked` over `NewTransfer`** everywhere
it is supported: the checked variant validates the mint and
decimals on chain, protecting against mint-confusion attacks
where a caller is tricked into signing a transfer for a
mint they did not expect.

### `NewMintTo`

```go
func NewMintTo(mint, destination, authority solana.PublicKey, amount uint64) solana.Instruction
```

Creates `amount` new tokens in `destination`. `authority` must
be the mint's mint authority.

### `NewBurn`

```go
func NewBurn(account, mint, authority solana.PublicKey, amount uint64) solana.Instruction
```

Destroys `amount` tokens from `account`. `authority` must be
the account's owner.

### `NewCloseAccount`

```go
func NewCloseAccount(account, destination, authority solana.PublicKey) solana.Instruction
```

Closes a zero-balance token account and returns its rent
lamports to `destination`. The token account must have a
zero balance.

### `NewInitializeMint2`

```go
func NewInitializeMint2(
    mint solana.PublicKey,
    decimals uint8,
    mintAuthority, freezeAuthority solana.PublicKey,
) solana.Instruction
```

Initialises a newly-created mint account. Unlike the legacy
`InitializeMint` (tag 0), this form **does not require the
rent sysvar** as an account input, so it is the recommended
way to create new mints.

Pass `freezeAuthority = solana.PublicKey{}` (the zero key) to
indicate no freeze authority; the option byte is encoded
accordingly.

### `NewInitializeAccount3`

```go
func NewInitializeAccount3(account, mint, owner solana.PublicKey) solana.Instruction
```

Initialises a newly-created token account. Like
`InitializeMint2`, this form takes `owner` as a parameter
instead of requiring the rent sysvar as an account input.

For new code, prefer the ATA flow instead — see
[Associated Token Account](Associated-Token-Account-Program).

## Instruction tags

The package uses single-byte tags (not u32 like System):

| Tag | Instruction |
|---|---|
| 3 | Transfer |
| 7 | MintTo |
| 8 | Burn |
| 9 | CloseAccount |
| 12 | TransferChecked |
| 18 | InitializeAccount3 |
| 20 | InitializeMint2 |

## Example: mint USDC-like tokens to an ATA

```go
import (
    associatedtokenaccount "github.com/MevYu/solana-go/programs/associated-token-account"
    "github.com/MevYu/solana-go/programs/token"
)

// 1. Ensure the recipient has an ATA for this mint.
createATA, _ := associatedtokenaccount.NewCreateIdempotent(
    payer.PublicKey(), recipient, mint, token.ProgramID,
)
recipientATA, _, _ := associatedtokenaccount.FindAssociatedTokenAddress(
    recipient, mint, token.ProgramID,
)

// 2. Mint tokens to it.
mintTo := token.NewMintTo(mint, recipientATA, payer.PublicKey(), 1_000_000)
```

Compose `createATA` and `mintTo` into a transaction and send via the
shared flow in [Getting Started](Getting-Started).

## Related

- [Token-2022 Program](Token-2022-Program) — the
  extension-capable successor.
- [Associated Token Account](Associated-Token-Account-Program)
  — the canonical wallet-to-token-account mapping.
- [Block & Token](Block-and-Token-Methods) —
  `GetTokenAccountBalance`, `GetTokenSupply`.
