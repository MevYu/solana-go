package vote

import (
	"github.com/MevYu/solana-go"
	"github.com/MevYu/solana-go/encoding"
)

// Instruction tag values for the Vote program (little-endian u32).
// Source of truth: solana_vote_program::vote_instruction::VoteInstruction
// (variants are repr(u32), implicit ordinal tags).
const (
	tagInitializeAccount uint32 = 0
	tagAuthorize         uint32 = 1
	// tag 2 is Vote (not wrapped here)
	tagWithdraw                uint32 = 3
	tagUpdateValidatorIdentity uint32 = 4
	tagUpdateCommission        uint32 = 5
	// tag 6 is VoteSwitch (not wrapped here)
	tagAuthorizeChecked uint32 = 7
)

// VoteAuthorize identifies which authority field Authorize changes.
type VoteAuthorize uint32

const (
	VoteAuthorizeVoter      VoteAuthorize = 0 // Authorized voter
	VoteAuthorizeWithdrawer VoteAuthorize = 1 // Authorized withdrawer
)

type genericIx struct {
	accounts []*solana.AccountMeta
	data     []byte
}

func (g *genericIx) ProgramID() solana.PublicKey     { return ProgramID }
func (g *genericIx) Accounts() []*solana.AccountMeta { return g.accounts }
func (g *genericIx) Data() ([]byte, error)           { return g.data, nil }

// NewInitializeAccount builds a Vote.InitializeAccount instruction that
// creates and initialises a new vote account. nodePubkey must sign the
// transaction; it is the validator identity that will cast votes.
func NewInitializeAccount(vote, nodePubkey, authorizedVoter, authorizedWithdrawer solana.PublicKey, commission uint8) solana.Instruction {
	return &genericIx{
		accounts: []*solana.AccountMeta{
			solana.NewAccountMeta(vote, false, true),
			solana.NewAccountMeta(solana.SysvarRentPubkey, false, false),
			solana.NewAccountMeta(solana.SysvarClockPubkey, false, false),
			solana.NewAccountMeta(nodePubkey, true, false),
		},
		data: encoding.NewEncoder(101).
			U32(tagInitializeAccount).
			Raw(nodePubkey[:]).
			Raw(authorizedVoter[:]).
			Raw(authorizedWithdrawer[:]).
			U8(commission).
			Bytes(),
	}
}

// NewAuthorize builds a Vote.Authorize instruction that changes the voter
// or withdrawer authority on a vote account. The current authority
// (matching authType) must sign the transaction.
func NewAuthorize(vote, authority, newAuthority solana.PublicKey, authType VoteAuthorize) solana.Instruction {
	return &genericIx{
		accounts: []*solana.AccountMeta{
			solana.NewAccountMeta(vote, false, true),
			solana.NewAccountMeta(solana.SysvarClockPubkey, false, false),
			solana.NewAccountMeta(authority, true, false),
		},
		data: encoding.NewEncoder(40).
			U32(tagAuthorize).
			Raw(newAuthority[:]).
			U32(uint32(authType)).
			Bytes(),
	}
}

// NewWithdraw builds a Vote.Withdraw instruction that moves lamports from
// the vote account to a recipient. The withdrawer authority must sign
// the transaction.
func NewWithdraw(vote, withdrawAuthority, recipient solana.PublicKey, lamports uint64) solana.Instruction {
	return &genericIx{
		accounts: []*solana.AccountMeta{
			solana.NewAccountMeta(vote, false, true),
			solana.NewAccountMeta(recipient, false, true),
			solana.NewAccountMeta(withdrawAuthority, true, false),
		},
		data: encoding.NewEncoder(12).U32(tagWithdraw).U64(lamports).Bytes(),
	}
}

// NewUpdateValidatorIdentity builds a Vote.UpdateValidatorIdentity
// instruction that changes the validator identity on a vote account. Both
// the vote account's withdrawer authority and the new identity account
// must sign the transaction.
func NewUpdateValidatorIdentity(vote, newIdentity, withdrawAuthority solana.PublicKey) solana.Instruction {
	return &genericIx{
		accounts: []*solana.AccountMeta{
			solana.NewAccountMeta(vote, false, true),
			solana.NewAccountMeta(newIdentity, true, false),
			solana.NewAccountMeta(withdrawAuthority, true, false),
		},
		data: encoding.NewEncoder(4).U32(tagUpdateValidatorIdentity).Bytes(),
	}
}

// NewAuthorizeChecked builds a Vote.AuthorizeChecked instruction (tag 6),
// the checked variant of Authorize that additionally requires the new
// authority to sign the transaction.
func NewAuthorizeChecked(vote, authority, newAuthority solana.PublicKey, authType VoteAuthorize) solana.Instruction {
	return &genericIx{
		accounts: []*solana.AccountMeta{
			solana.NewAccountMeta(vote, false, true),
			solana.NewAccountMeta(solana.SysvarClockPubkey, false, false),
			solana.NewAccountMeta(authority, true, false),
			solana.NewAccountMeta(newAuthority, true, false),
		},
		data: encoding.NewEncoder(8).U32(tagAuthorizeChecked).U32(uint32(authType)).Bytes(),
	}
}

// NewUpdateCommission builds a Vote.UpdateCommission instruction that sets
// the validator's commission rate (0–100). The withdrawer authority must
// sign the transaction.
func NewUpdateCommission(vote, withdrawAuthority solana.PublicKey, commission uint8) solana.Instruction {
	return &genericIx{
		accounts: []*solana.AccountMeta{
			solana.NewAccountMeta(vote, false, true),
			solana.NewAccountMeta(withdrawAuthority, true, false),
		},
		data: encoding.NewEncoder(5).U32(tagUpdateCommission).U8(commission).Bytes(),
	}
}
