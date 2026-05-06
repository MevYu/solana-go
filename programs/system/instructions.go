package system

import (
	"github.com/MevYu/solana-go"
	"github.com/MevYu/solana-go/encoding"
)

// Instruction tag values for the System program. They are the
// first four bytes of every instruction's data payload and match
// the Solana runtime's SystemInstruction enum exactly.
const (
	tagCreateAccount          uint32 = 0
	tagAssign                 uint32 = 1
	tagTransfer               uint32 = 2
	tagCreateAccountWithSeed  uint32 = 3
	tagAdvanceNonceAccount    uint32 = 4
	tagWithdrawNonceAccount   uint32 = 5
	tagInitializeNonceAccount uint32 = 6
	tagAuthorizeNonceAccount  uint32 = 7
	tagAllocate               uint32 = 8
	tagTransferWithSeed       uint32 = 11
)

// genericIx is a small solana.Instruction implementation used by
// every builder in this package. It carries a program id, account
// meta list, and pre-encoded instruction data.
type genericIx struct {
	programID solana.PublicKey
	accounts  []*solana.AccountMeta
	data      []byte
}

func (g *genericIx) ProgramID() solana.PublicKey     { return g.programID }
func (g *genericIx) Accounts() []*solana.AccountMeta { return g.accounts }
func (g *genericIx) Data() ([]byte, error)           { return g.data, nil }

// NewTransfer builds a System.Transfer instruction that moves lamports
// from one account to another. The from account must sign the
// transaction. Both accounts are marked writable because their lamport
// balances change.
func NewTransfer(from, to solana.PublicKey, lamports uint64) solana.Instruction {
	return &genericIx{
		programID: ProgramID,
		accounts: []*solana.AccountMeta{
			solana.NewAccountMeta(from, true, true),
			solana.NewAccountMeta(to, false, true),
		},
		data: encoding.NewEncoder(12).U32(tagTransfer).U64(lamports).Bytes(),
	}
}

// NewCreateAccount builds a System.CreateAccount instruction that funds
// newAccount with the given lamports, allocates space bytes for its
// data, and assigns its owner program. Both from and newAccount must
// sign the transaction.
func NewCreateAccount(from, newAccount solana.PublicKey, lamports, space uint64, owner solana.PublicKey) solana.Instruction {
	return &genericIx{
		programID: ProgramID,
		accounts: []*solana.AccountMeta{
			solana.NewAccountMeta(from, true, true),
			solana.NewAccountMeta(newAccount, true, true),
		},
		data: encoding.NewEncoder(52).
			U32(tagCreateAccount).
			U64(lamports).
			U64(space).
			Raw(owner[:]).
			Bytes(),
	}
}

// NewAssign builds a System.Assign instruction that changes the owner
// program of an existing account. The account must currently be owned
// by the System program and must sign the transaction.
func NewAssign(account, owner solana.PublicKey) solana.Instruction {
	return &genericIx{
		programID: ProgramID,
		accounts: []*solana.AccountMeta{
			solana.NewAccountMeta(account, true, true),
		},
		data: encoding.NewEncoder(36).U32(tagAssign).Raw(owner[:]).Bytes(),
	}
}

// NewAllocate builds a System.Allocate instruction that allocates space
// bytes for an account's data. The account must sign the transaction
// and must currently be a System-owned account with zero-length data.
func NewAllocate(account solana.PublicKey, space uint64) solana.Instruction {
	return &genericIx{
		programID: ProgramID,
		accounts: []*solana.AccountMeta{
			solana.NewAccountMeta(account, true, true),
		},
		data: encoding.NewEncoder(12).U32(tagAllocate).U64(space).Bytes(),
	}
}

// NewAdvanceNonceAccount builds a System.AdvanceNonceAccount instruction
// that consumes the nonce and advances it to a new value. The nonce
// authority must sign the transaction.
func NewAdvanceNonceAccount(nonce, authority solana.PublicKey) solana.Instruction {
	return &genericIx{
		programID: ProgramID,
		accounts: []*solana.AccountMeta{
			solana.NewAccountMeta(nonce, false, true),
			solana.NewAccountMeta(solana.SysvarRecentBlockhashesPubkey, false, false),
			solana.NewAccountMeta(authority, true, false),
		},
		data: encoding.NewEncoder(4).U32(tagAdvanceNonceAccount).Bytes(),
	}
}

// NewWithdrawNonceAccount builds a System.WithdrawNonceAccount instruction
// that withdraws lamports from a nonce account to a destination. lamports
// must leave the nonce account with enough for rent-exemption, or
// withdraw all lamports to close it. The authority must sign the
// transaction.
func NewWithdrawNonceAccount(nonce, authority, to solana.PublicKey, lamports uint64) solana.Instruction {
	return &genericIx{
		programID: ProgramID,
		accounts: []*solana.AccountMeta{
			solana.NewAccountMeta(nonce, false, true),
			solana.NewAccountMeta(to, false, true),
			solana.NewAccountMeta(solana.SysvarRecentBlockhashesPubkey, false, false),
			solana.NewAccountMeta(solana.SysvarRentPubkey, false, false),
			solana.NewAccountMeta(authority, true, false),
		},
		data: encoding.NewEncoder(12).U32(tagWithdrawNonceAccount).U64(lamports).Bytes(),
	}
}

// NewInitializeNonceAccount builds a System.InitializeNonceAccount
// instruction that initializes a new nonce account with the given
// authority. The nonce account must already be funded and allocated by
// System.CreateAccount before calling this instruction.
func NewInitializeNonceAccount(nonce, authority solana.PublicKey) solana.Instruction {
	return &genericIx{
		programID: ProgramID,
		accounts: []*solana.AccountMeta{
			solana.NewAccountMeta(nonce, false, true),
			solana.NewAccountMeta(solana.SysvarRecentBlockhashesPubkey, false, false),
			solana.NewAccountMeta(solana.SysvarRentPubkey, false, false),
		},
		data: encoding.NewEncoder(36).U32(tagInitializeNonceAccount).Raw(authority[:]).Bytes(),
	}
}

// NewAuthorizeNonceAccount builds a System.AuthorizeNonceAccount instruction
// that changes the authority of a nonce account to newAuthority. The current
// authority must sign the transaction.
func NewAuthorizeNonceAccount(nonce, authority, newAuthority solana.PublicKey) solana.Instruction {
	return &genericIx{
		programID: ProgramID,
		accounts: []*solana.AccountMeta{
			solana.NewAccountMeta(nonce, false, true),
			solana.NewAccountMeta(authority, true, false),
		},
		data: encoding.NewEncoder(36).U32(tagAuthorizeNonceAccount).Raw(newAuthority[:]).Bytes(),
	}
}

// NewCreateAccountWithSeed builds a System.CreateAccountWithSeed instruction
// that creates newAccount at the address derived from base, seed, and the
// owner program. Unlike CreateAccount, only the funding account (and base,
// when distinct) needs to sign — newAccount is derived and does not need
// its own keypair.
//
// newAccount must equal CreateWithSeed(base, seed, owner) as computed by
// the Solana runtime.
func NewCreateAccountWithSeed(from, newAccount, base solana.PublicKey, seed string, lamports, space uint64, owner solana.PublicKey) solana.Instruction {
	data := encoding.NewEncoder(4 + solana.PublicKeySize + 8 + len(seed) + 8 + 8 + solana.PublicKeySize).
		U32(tagCreateAccountWithSeed).
		Raw(base[:]).
		StrU64(seed).
		U64(lamports).
		U64(space).
		Raw(owner[:]).
		Bytes()
	accounts := []*solana.AccountMeta{
		solana.NewAccountMeta(from, true, true),
		solana.NewAccountMeta(newAccount, false, true),
	}
	if base != from {
		accounts = append(accounts, solana.NewAccountMeta(base, true, false))
	}
	return &genericIx{programID: ProgramID, accounts: accounts, data: data}
}

// NewTransferWithSeed builds a System.TransferWithSeed instruction that
// moves lamports from a seed-derived address to a destination. The base
// account must sign; no separate keypair for the from account is needed.
//
// from must equal CreateWithSeed(base, seed, programID).
func NewTransferWithSeed(from, base, to solana.PublicKey, lamports uint64, seed string, programID solana.PublicKey) solana.Instruction {
	return &genericIx{
		programID: ProgramID,
		accounts: []*solana.AccountMeta{
			solana.NewAccountMeta(from, false, true),
			solana.NewAccountMeta(base, true, false),
			solana.NewAccountMeta(to, false, true),
		},
		data: encoding.NewEncoder(4 + 8 + 8 + len(seed) + solana.PublicKeySize).
			U32(tagTransferWithSeed).
			U64(lamports).
			StrU64(seed).
			Raw(programID[:]).
			Bytes(),
	}
}
