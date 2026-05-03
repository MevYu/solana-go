# Vote Program

The **Vote Program** manages validator vote accounts: the
on-chain receivers of stake delegations and the place validators
sign their consensus votes. Most application code does not
build vote-program transactions directly — the relevant
operations are validator-operator workflows. The builders
exist so cluster operators can drive vote accounts from Go.

```go
import "github.com/MevYu/solana-go/programs/vote"
```

## Program id

```go
var vote.ProgramID = solana.MustPublicKey(
    "Vote111111111111111111111111111111111111111")
```

## Authority types

```go
const (
    VoteAuthorizeVoter      VoteAuthorize = 0
    VoteAuthorizeWithdrawer VoteAuthorize = 1
)
```

## Builders

### `NewInitializeAccount`

```go
func NewInitializeAccount(
    vote, nodePubkey, authorizedVoter, authorizedWithdrawer solana.PublicKey,
    commission uint8,
) solana.Instruction
```

Initialise a vote account. `nodePubkey` is the validator
identity that signs votes; it must sign the transaction.

### `NewAuthorize` / `NewAuthorizeChecked`

```go
func NewAuthorize(vote, authority, newAuthority solana.PublicKey, authType VoteAuthorize) solana.Instruction
func NewAuthorizeChecked(vote, authority, newAuthority solana.PublicKey, authType VoteAuthorize) solana.Instruction
```

Change the voter or withdrawer authority on the vote account.
`AuthorizeChecked` additionally requires `newAuthority` to sign.

### `NewWithdraw`

```go
func NewWithdraw(vote, withdrawAuthority, recipient solana.PublicKey, lamports uint64) solana.Instruction
```

Move lamports out of the vote account (the validator's
accumulated commission). Only the withdrawer authority can
sign.

### `NewUpdateValidatorIdentity`

```go
func NewUpdateValidatorIdentity(vote, newIdentity, withdrawAuthority solana.PublicKey) solana.Instruction
```

Rotate the validator identity that signs votes. Both the
withdrawer authority and the new identity must sign the
transaction.

### `NewUpdateCommission`

```go
func NewUpdateCommission(vote, withdrawAuthority solana.PublicKey, commission uint8) solana.Instruction
```

Set the validator's commission rate (0–100). Only the
withdrawer authority can sign.

## Wire format

Tags are little-endian `u32` (matching
`solana_vote_program::vote_instruction::VoteInstruction`):

| Tag | Instruction |
|---|---|
| 0 | InitializeAccount |
| 1 | Authorize |
| 2 | Vote *(not wrapped)* |
| 3 | Withdraw |
| 4 | UpdateValidatorIdentity |
| 5 | UpdateCommission |
| 6 | VoteSwitch *(not wrapped)* |
| 7 | AuthorizeChecked |

Higher tags (compact-vote-state, tower-sync) are not wrapped —
those are validator-internal operations driven by the validator
binary itself.

## Related

- [Stake Program](Stake-Program) — delegates stake to the
  account managed here.
