# Stake Program

The **Stake Program** delegates lamports to a vote account so they
participate in the Solana inflation reward schedule. A stake
account is a normal SOL-bearing account whose authorities and
lockup are tracked by the Stake program.

```go
import "github.com/MevYu/solana-go/programs/stake"
```

## Program id

```go
var stake.ProgramID = solana.MustPublicKey(
    "Stake11111111111111111111111111111111111111")
```

## Authorities

A stake account has two independent authorities:

- **Staker** — controls delegation (`Delegate`, `Deactivate`,
  `Split`, `Merge`).
- **Withdrawer** — controls funds (`Withdraw`, `Authorize`).

Both can be the same key (typical) or different keys. The
authority you change is selected by `StakeAuthorize`:

```go
const (
    StakeAuthorizeStaker     StakeAuthorize = 0
    StakeAuthorizeWithdrawer StakeAuthorize = 1
)
```

## Builders

### `Initialize`

```go
type Authorized struct { Staker, Withdrawer solana.PublicKey }
type Lockup struct {
    UnixTimestamp int64
    Epoch         uint64
    Custodian     solana.PublicKey
}

func Initialize(stakeAccount solana.PublicKey, authorized Authorized, lockup Lockup) solana.Instruction
```

Initialise a freshly created stake account with the given
authorities and (optional) lockup. Pair with
`system.NewCreateAccount` in the same transaction.

### `Delegate`

```go
func Delegate(stakeAccount, voteAccount, staker solana.PublicKey) solana.Instruction
```

Delegate the stake to `voteAccount`. The staker authority must
sign.

### `Deactivate` / `Withdraw`

```go
func Deactivate(stakeAccount, staker solana.PublicKey) solana.Instruction
func Withdraw(stakeAccount, destination, withdrawer solana.PublicKey,
    lamports uint64, custodian *solana.PublicKey) solana.Instruction
```

`Deactivate` schedules the stake to cool down; `Withdraw` moves
free lamports out (only lamports above the activated stake plus
rent-exempt minimum). Pass a non-nil `custodian` only when the
stake account is locked up and the lockup custodian is signing.

### `Split` / `Merge`

```go
func Split(stakeAccount, splitStake, staker solana.PublicKey, lamports uint64) solana.Instruction
func Merge(destination, source, staker solana.PublicKey) solana.Instruction
```

Split moves `lamports` from `stakeAccount` into a freshly
created `splitStake` (which must already exist as an empty
stake account). Merge consolidates `source` into `destination`.

### `Authorize` / `AuthorizeChecked`

```go
func Authorize(stakeAccount, authority, newAuthority solana.PublicKey,
    authType StakeAuthorize, custodian *solana.PublicKey) solana.Instruction
func AuthorizeChecked(stakeAccount, authority, newAuthority solana.PublicKey,
    authType StakeAuthorize, custodian *solana.PublicKey) solana.Instruction
```

`AuthorizeChecked` additionally requires the new authority to
sign the transaction, preventing typos that would otherwise lock
the account.

### `InitializeChecked` / `SetLockup`

```go
func InitializeChecked(stakeAccount, staker, withdrawer solana.PublicKey) solana.Instruction
func SetLockup(stakeAccount, authority solana.PublicKey,
    unixTimestamp *int64, epoch *uint64, custodian *solana.PublicKey) solana.Instruction
```

`SetLockup` updates only the fields whose pointer is non-nil;
pass `nil` for fields you want to leave unchanged.

## Related

- [Vote Program](Vote-Program) — the target of `Delegate`.
- [System Program](System-Program) — `CreateAccount` that
  pre-allocates the stake account before `Initialize`.
