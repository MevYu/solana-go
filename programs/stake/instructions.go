package stake

import (
	solana "github.com/MevYu/solana-go"
	"github.com/MevYu/solana-go/encoding"
)

// Well-known sysvar and config addresses used by Stake instructions.
var (
	sysvarRent         = solana.MustPublicKey("SysvarRent111111111111111111111111111111111")
	sysvarClock        = solana.MustPublicKey("SysvarC1ock11111111111111111111111111111111")
	sysvarStakeHistory = solana.MustPublicKey("SysvarStakeHistory1111111111111111111111111")
	stakeConfig        = solana.MustPublicKey("StakeConfig11111111111111111111111111111111")
)

// Instruction tag values for the Stake program.
const (
	tagInitialize        uint32 = 0
	tagAuthorize         uint32 = 1
	tagDelegate          uint32 = 2
	tagSplit             uint32 = 3
	tagWithdraw          uint32 = 4
	tagDeactivate        uint32 = 5
	tagSetLockup         uint32 = 6
	tagMerge             uint32 = 7
	tagInitializeChecked uint32 = 9
	tagAuthorizeChecked  uint32 = 10
)

// StakeAuthorize identifies which authority field Authorize changes.
type StakeAuthorize uint32

const (
	AuthorizeStaker     StakeAuthorize = 0
	AuthorizeWithdrawer StakeAuthorize = 1
)

// Authorized holds the two pubkeys that are permitted to manage a
// stake account: one for staking operations and one for withdrawals.
type Authorized struct {
	Staker     solana.PublicKey
	Withdrawer solana.PublicKey
}

// Lockup specifies the conditions under which a stake account is
// locked. Until the unix timestamp and the epoch have both passed,
// only the custodian may withdraw funds.
type Lockup struct {
	UnixTimestamp int64
	Epoch         uint64
	Custodian     solana.PublicKey
}

// Initialize builds a Stake.Initialize instruction.
func Initialize(stakeAccount solana.PublicKey, authorized Authorized, lockup Lockup) solana.Instruction {
	return solana.NewInstruction(
		ProgramID,
		[]*solana.AccountMeta{
			solana.NewAccountMeta(stakeAccount, false, true),
			solana.NewAccountMeta(sysvarRent, false, false),
		},
		encoding.NewEncoder(116).
			U32(tagInitialize).
			Raw(authorized.Staker[:]).
			Raw(authorized.Withdrawer[:]).
			I64(lockup.UnixTimestamp).
			U64(lockup.Epoch).
			Raw(lockup.Custodian[:]).
			Bytes(),
	)
}

// Delegate builds a Stake.DelegateStake instruction.
func Delegate(stakeAccount, voteAccount, staker solana.PublicKey) solana.Instruction {
	return solana.NewInstruction(
		ProgramID,
		[]*solana.AccountMeta{
			solana.NewAccountMeta(stakeAccount, false, true),
			solana.NewAccountMeta(voteAccount, false, false),
			solana.NewAccountMeta(sysvarClock, false, false),
			solana.NewAccountMeta(sysvarStakeHistory, false, false),
			solana.NewAccountMeta(stakeConfig, false, false),
			solana.NewAccountMeta(staker, true, false),
		},
		encoding.NewEncoder(4).U32(tagDelegate).Bytes(),
	)
}

// Deactivate builds a Stake.Deactivate instruction.
func Deactivate(stakeAccount, staker solana.PublicKey) solana.Instruction {
	return solana.NewInstruction(
		ProgramID,
		[]*solana.AccountMeta{
			solana.NewAccountMeta(stakeAccount, false, true),
			solana.NewAccountMeta(sysvarClock, false, false),
			solana.NewAccountMeta(staker, true, false),
		},
		encoding.NewEncoder(4).U32(tagDeactivate).Bytes(),
	)
}

// Withdraw builds a Stake.Withdraw instruction.
func Withdraw(stakeAccount, destination, withdrawer solana.PublicKey, lamports uint64, custodian *solana.PublicKey) solana.Instruction {
	accounts := []*solana.AccountMeta{
		solana.NewAccountMeta(stakeAccount, false, true),
		solana.NewAccountMeta(destination, false, true),
		solana.NewAccountMeta(sysvarClock, false, false),
		solana.NewAccountMeta(sysvarStakeHistory, false, false),
		solana.NewAccountMeta(withdrawer, true, false),
	}
	if custodian != nil {
		accounts = append(accounts, solana.NewAccountMeta(*custodian, true, false))
	}
	return solana.NewInstruction(
		ProgramID,
		accounts,
		encoding.NewEncoder(12).U32(tagWithdraw).U64(lamports).Bytes(),
	)
}

// Split builds a Stake.Split instruction.
func Split(stakeAccount, splitStake, staker solana.PublicKey, lamports uint64) solana.Instruction {
	return solana.NewInstruction(
		ProgramID,
		[]*solana.AccountMeta{
			solana.NewAccountMeta(stakeAccount, false, true),
			solana.NewAccountMeta(splitStake, false, true),
			solana.NewAccountMeta(staker, true, false),
		},
		encoding.NewEncoder(12).U32(tagSplit).U64(lamports).Bytes(),
	)
}

// Merge builds a Stake.Merge instruction.
func Merge(destination, source, staker solana.PublicKey) solana.Instruction {
	return solana.NewInstruction(
		ProgramID,
		[]*solana.AccountMeta{
			solana.NewAccountMeta(destination, false, true),
			solana.NewAccountMeta(source, false, true),
			solana.NewAccountMeta(sysvarClock, false, false),
			solana.NewAccountMeta(sysvarStakeHistory, false, false),
			solana.NewAccountMeta(staker, true, false),
		},
		encoding.NewEncoder(4).U32(tagMerge).Bytes(),
	)
}

// Authorize builds a Stake.Authorize instruction that reassigns the staker
// or withdrawer authority. The current authority must sign. custodian is
// required only when the account is within its lockup period and the target
// authority is the withdrawer; pass nil to omit.
func Authorize(stakeAccount, authority, newAuthority solana.PublicKey, authType StakeAuthorize, custodian *solana.PublicKey) solana.Instruction {
	data := encoding.NewEncoder(40).
		U32(tagAuthorize).
		Raw(newAuthority[:]).
		U32(uint32(authType)).
		Bytes()
	accounts := []*solana.AccountMeta{
		solana.NewAccountMeta(stakeAccount, false, true),
		solana.NewAccountMeta(sysvarClock, false, false),
		solana.NewAccountMeta(authority, true, false),
	}
	if custodian != nil {
		accounts = append(accounts, solana.NewAccountMeta(*custodian, true, false))
	}
	return solana.NewInstruction(ProgramID, accounts, data)
}

// AuthorizeChecked builds a Stake.AuthorizeChecked instruction. Unlike
// Authorize, the new authority must also sign the transaction (preventing
// the caller from setting an authority they don't control).
func AuthorizeChecked(stakeAccount, authority, newAuthority solana.PublicKey, authType StakeAuthorize, custodian *solana.PublicKey) solana.Instruction {
	accounts := []*solana.AccountMeta{
		solana.NewAccountMeta(stakeAccount, false, true),
		solana.NewAccountMeta(sysvarClock, false, false),
		solana.NewAccountMeta(authority, true, false),
		solana.NewAccountMeta(newAuthority, true, false),
	}
	if custodian != nil {
		accounts = append(accounts, solana.NewAccountMeta(*custodian, true, false))
	}
	return solana.NewInstruction(
		ProgramID,
		accounts,
		encoding.NewEncoder(8).U32(tagAuthorizeChecked).U32(uint32(authType)).Bytes(),
	)
}

// InitializeChecked builds a Stake.InitializeChecked instruction. Like
// Initialize but the withdrawer authority must sign the transaction,
// making it safe to use when staker ≠ withdrawer.
func InitializeChecked(stakeAccount, staker, withdrawer solana.PublicKey) solana.Instruction {
	return solana.NewInstruction(
		ProgramID,
		[]*solana.AccountMeta{
			solana.NewAccountMeta(stakeAccount, false, true),
			solana.NewAccountMeta(sysvarRent, false, false),
			solana.NewAccountMeta(staker, false, false),
			solana.NewAccountMeta(withdrawer, true, false),
		},
		encoding.NewEncoder(4).U32(tagInitializeChecked).Bytes(),
	)
}

// SetLockup builds a Stake.SetLockup instruction that modifies the lockup
// conditions of a stake account. The current withdraw authority (or
// custodian if still within lockup) must sign. Pass nil for any field to
// leave it unchanged.
func SetLockup(stakeAccount, authority solana.PublicKey, unixTimestamp *int64, epoch *uint64, custodian *solana.PublicKey) solana.Instruction {
	e := encoding.NewEncoder(64).
		U32(tagSetLockup).
		OptI64(unixTimestamp).
		OptU64(epoch)
	if custodian != nil {
		e.U8(1).Raw(custodian[:])
	} else {
		e.U8(0)
	}
	data := e.Bytes()
	return solana.NewInstruction(
		ProgramID,
		[]*solana.AccountMeta{
			solana.NewAccountMeta(stakeAccount, false, true),
			solana.NewAccountMeta(authority, true, false),
		},
		data,
	)
}
