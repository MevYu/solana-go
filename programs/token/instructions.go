package token

import (
	"github.com/MevYu/solana-go"
	"github.com/MevYu/solana-go/encoding"
)

// Instruction tag values for SPL Token.
const (
	tagInitializeMint     byte = 0
	tagInitializeAccount  byte = 1
	tagInitializeMultisig byte = 2
	tagTransfer           byte = 3
	tagApprove            byte = 4
	tagRevoke             byte = 5
	tagSetAuthority       byte = 6
	tagMintTo             byte = 7
	tagBurn               byte = 8
	tagCloseAccount       byte = 9
	tagFreezeAccount      byte = 10
	tagThawAccount        byte = 11
	tagTransferChecked    byte = 12
	tagApproveChecked     byte = 13
	tagMintToChecked      byte = 14
	tagBurnChecked        byte = 15
	tagSyncNative         byte = 17
	tagInitializeAccount3 byte = 18
	tagInitializeMint2    byte = 20
)

// AuthorityType identifies which authority field SetAuthority changes.
type AuthorityType uint8

const (
	AuthorityMintTokens    AuthorityType = 0 // Mint authority
	AuthorityFreezeAccount AuthorityType = 1 // Freeze authority on a mint
	AuthorityAccountOwner  AuthorityType = 2 // Owner of a token account
	AuthorityCloseAccount  AuthorityType = 3 // Close authority on a token account
)

// genericIx is the small solana.Instruction implementation used by every
// SPL Token builder.
type genericIx struct {
	accounts []*solana.AccountMeta
	data     []byte
}

func (g *genericIx) ProgramID() solana.PublicKey     { return ProgramID }
func (g *genericIx) Accounts() []*solana.AccountMeta { return g.accounts }
func (g *genericIx) Data() ([]byte, error)           { return g.data, nil }

// initMintData encodes the shared payload of InitializeMint and
// InitializeMint2: [tag u8] [decimals u8] [mintAuthority 32B] [option u8]
// [freezeAuthority 32B if option==1].
func initMintData(tag byte, decimals uint8, mintAuthority solana.PublicKey, freezeAuthority *solana.PublicKey) []byte {
	cap := 1 + 1 + solana.PublicKeySize + 1
	if freezeAuthority != nil {
		cap += solana.PublicKeySize
	}
	e := encoding.NewEncoder(cap).U8(tag).U8(decimals).Raw(mintAuthority[:])
	if freezeAuthority != nil {
		e.U8(1).Raw(freezeAuthority[:])
	} else {
		e.U8(0)
	}
	return e.Bytes()
}

// NewInitializeMint builds the legacy SPL Token InitializeMint instruction
// (tag 0). This form requires the rent sysvar as an account; prefer
// NewInitializeMint2 for new code.
//
// freezeAuthority may be nil to indicate no freeze authority.
func NewInitializeMint(mint solana.PublicKey, decimals uint8, mintAuthority solana.PublicKey, freezeAuthority *solana.PublicKey) solana.Instruction {
	return &genericIx{
		accounts: []*solana.AccountMeta{
			solana.NewAccountMeta(mint, false, true),
			solana.NewAccountMeta(solana.SysvarRentPubkey, false, false),
		},
		data: initMintData(tagInitializeMint, decimals, mintAuthority, freezeAuthority),
	}
}

// NewInitializeAccount builds the legacy SPL Token InitializeAccount
// instruction (tag 1). This form requires the rent sysvar; prefer
// NewInitializeAccount3 for new code.
func NewInitializeAccount(account, mint, owner solana.PublicKey) solana.Instruction {
	return &genericIx{
		accounts: []*solana.AccountMeta{
			solana.NewAccountMeta(account, false, true),
			solana.NewAccountMeta(mint, false, false),
			solana.NewAccountMeta(owner, false, false),
			solana.NewAccountMeta(solana.SysvarRentPubkey, false, false),
		},
		data: []byte{tagInitializeAccount},
	}
}

// NewInitializeMultisig builds an SPL Token InitializeMultisig instruction
// that initialises a multisig authority account requiring m of len(signers)
// signatures. signers are the constituent signer public keys.
func NewInitializeMultisig(multisig solana.PublicKey, m uint8, signers []solana.PublicKey) solana.Instruction {
	accounts := make([]*solana.AccountMeta, 0, 2+len(signers))
	accounts = append(accounts,
		solana.NewAccountMeta(multisig, false, true),
		solana.NewAccountMeta(solana.SysvarRentPubkey, false, false),
	)
	for _, s := range signers {
		accounts = append(accounts, solana.NewAccountMeta(s, false, false))
	}
	return &genericIx{
		accounts: accounts,
		data:     []byte{tagInitializeMultisig, m},
	}
}

// NewTransfer builds an SPL Token Transfer instruction.
//
// source and destination are the token account addresses (not the wallet
// addresses). authority is the token account's owner and must sign the
// transaction.
//
// Prefer NewTransferChecked over this instruction: the checked variant
// validates the token's decimals on chain, protecting against mint
// confusion attacks.
func NewTransfer(source, destination, authority solana.PublicKey, amount uint64) solana.Instruction {
	return &genericIx{
		accounts: []*solana.AccountMeta{
			solana.NewAccountMeta(source, false, true),
			solana.NewAccountMeta(destination, false, true),
			solana.NewAccountMeta(authority, true, false),
		},
		data: encoding.NewEncoder(9).U8(tagTransfer).U64(amount).Bytes(),
	}
}

// NewTransferChecked builds an SPL Token TransferChecked instruction.
// Unlike NewTransfer, it requires the mint pubkey and the mint's decimals
// to be passed in, and the runtime verifies both match the token account
// on chain.
func NewTransferChecked(source, mint, destination, authority solana.PublicKey, amount uint64, decimals uint8) solana.Instruction {
	return &genericIx{
		accounts: []*solana.AccountMeta{
			solana.NewAccountMeta(source, false, true),
			solana.NewAccountMeta(mint, false, false),
			solana.NewAccountMeta(destination, false, true),
			solana.NewAccountMeta(authority, true, false),
		},
		data: encoding.NewEncoder(10).U8(tagTransferChecked).U64(amount).U8(decimals).Bytes(),
	}
}

// NewMintTo builds an SPL Token MintTo instruction that creates amount
// new tokens in destination. authority must be the mint's mint authority
// and must sign the transaction.
func NewMintTo(mint, destination, authority solana.PublicKey, amount uint64) solana.Instruction {
	return &genericIx{
		accounts: []*solana.AccountMeta{
			solana.NewAccountMeta(mint, false, true),
			solana.NewAccountMeta(destination, false, true),
			solana.NewAccountMeta(authority, true, false),
		},
		data: encoding.NewEncoder(9).U8(tagMintTo).U64(amount).Bytes(),
	}
}

// NewBurn builds an SPL Token Burn instruction that destroys amount
// tokens from account. authority must be the account's owner and must
// sign the transaction.
func NewBurn(account, mint, authority solana.PublicKey, amount uint64) solana.Instruction {
	return &genericIx{
		accounts: []*solana.AccountMeta{
			solana.NewAccountMeta(account, false, true),
			solana.NewAccountMeta(mint, false, true),
			solana.NewAccountMeta(authority, true, false),
		},
		data: encoding.NewEncoder(9).U8(tagBurn).U64(amount).Bytes(),
	}
}

// NewCloseAccount builds an SPL Token CloseAccount instruction that
// returns the token account's rent lamports to destination. The token
// account must have a zero balance and authority must sign the
// transaction.
func NewCloseAccount(account, destination, authority solana.PublicKey) solana.Instruction {
	return &genericIx{
		accounts: []*solana.AccountMeta{
			solana.NewAccountMeta(account, false, true),
			solana.NewAccountMeta(destination, false, true),
			solana.NewAccountMeta(authority, true, false),
		},
		data: []byte{tagCloseAccount},
	}
}

// NewInitializeMint2 builds an SPL Token InitializeMint2 instruction.
// Unlike the legacy InitializeMint (tag 0), this form does not require
// the rent sysvar as an account input, so it is the recommended way to
// create new mints.
//
// freezeAuthority may be the zero PublicKey to indicate no freeze
// authority; the encoded option byte is set accordingly.
func NewInitializeMint2(mint solana.PublicKey, decimals uint8, mintAuthority solana.PublicKey, freezeAuthority solana.PublicKey) solana.Instruction {
	var fa *solana.PublicKey
	if !freezeAuthority.IsZero() {
		fa = &freezeAuthority
	}
	return &genericIx{
		accounts: []*solana.AccountMeta{
			solana.NewAccountMeta(mint, false, true),
		},
		data: initMintData(tagInitializeMint2, decimals, mintAuthority, fa),
	}
}

// NewInitializeAccount3 builds an SPL Token InitializeAccount3
// instruction. The newer form (as opposed to InitializeAccount /
// InitializeAccount2) takes the owner as a parameter instead of requiring
// the rent sysvar as an account input, and is the recommended form for
// all new code.
func NewInitializeAccount3(account, mint, owner solana.PublicKey) solana.Instruction {
	return &genericIx{
		accounts: []*solana.AccountMeta{
			solana.NewAccountMeta(account, false, true),
			solana.NewAccountMeta(mint, false, false),
		},
		data: encoding.NewEncoder(33).U8(tagInitializeAccount3).Raw(owner[:]).Bytes(),
	}
}

// NewApprove builds an SPL Token Approve instruction that authorizes
// delegate to transfer up to amount tokens from source on behalf of owner.
func NewApprove(source, delegate, owner solana.PublicKey, amount uint64) solana.Instruction {
	return &genericIx{
		accounts: []*solana.AccountMeta{
			solana.NewAccountMeta(source, false, true),
			solana.NewAccountMeta(delegate, false, false),
			solana.NewAccountMeta(owner, true, false),
		},
		data: encoding.NewEncoder(9).U8(tagApprove).U64(amount).Bytes(),
	}
}

// NewApproveChecked builds an SPL Token ApproveChecked instruction (tag
// 13). Like NewApprove, it authorizes delegate to transfer up to amount
// tokens from source, but it also requires the mint pubkey and decimals —
// the runtime verifies both match the on-chain token account, guarding
// against mint-confusion attacks.
func NewApproveChecked(source, mint, delegate, owner solana.PublicKey, amount uint64, decimals uint8) solana.Instruction {
	return &genericIx{
		accounts: []*solana.AccountMeta{
			solana.NewAccountMeta(source, false, true),
			solana.NewAccountMeta(mint, false, false),
			solana.NewAccountMeta(delegate, false, false),
			solana.NewAccountMeta(owner, true, false),
		},
		data: encoding.NewEncoder(10).U8(tagApproveChecked).U64(amount).U8(decimals).Bytes(),
	}
}

// NewRevoke builds an SPL Token Revoke instruction that removes any
// previously approved delegate from source. owner must sign.
func NewRevoke(source, owner solana.PublicKey) solana.Instruction {
	return &genericIx{
		accounts: []*solana.AccountMeta{
			solana.NewAccountMeta(source, false, true),
			solana.NewAccountMeta(owner, true, false),
		},
		data: []byte{tagRevoke},
	}
}

// NewSetAuthority builds an SPL Token SetAuthority instruction that
// changes one of the authority fields on account. newAuthority is
// optional: pass nil to clear the authority (only valid for some
// authority types). currentAuthority must sign the transaction.
func NewSetAuthority(account solana.PublicKey, currentAuthority solana.PublicKey, newAuthority *solana.PublicKey, authType AuthorityType) solana.Instruction {
	cap := 1 + 1 + 1
	if newAuthority != nil {
		cap += 32
	}
	e := encoding.NewEncoder(cap).U8(tagSetAuthority).U8(byte(authType))
	if newAuthority != nil {
		e.U8(1).Raw(newAuthority[:])
	} else {
		e.U8(0)
	}
	return &genericIx{
		accounts: []*solana.AccountMeta{
			solana.NewAccountMeta(account, false, true),
			solana.NewAccountMeta(currentAuthority, true, false),
		},
		data: e.Bytes(),
	}
}

// NewFreezeAccount builds an SPL Token FreezeAccount instruction. The
// mint's freeze authority must sign the transaction.
func NewFreezeAccount(account, mint, freezeAuthority solana.PublicKey) solana.Instruction {
	return &genericIx{
		accounts: []*solana.AccountMeta{
			solana.NewAccountMeta(account, false, true),
			solana.NewAccountMeta(mint, false, false),
			solana.NewAccountMeta(freezeAuthority, true, false),
		},
		data: []byte{tagFreezeAccount},
	}
}

// NewThawAccount builds an SPL Token ThawAccount instruction that
// unfreezes a frozen token account. The mint's freeze authority must sign.
func NewThawAccount(account, mint, freezeAuthority solana.PublicKey) solana.Instruction {
	return &genericIx{
		accounts: []*solana.AccountMeta{
			solana.NewAccountMeta(account, false, true),
			solana.NewAccountMeta(mint, false, false),
			solana.NewAccountMeta(freezeAuthority, true, false),
		},
		data: []byte{tagThawAccount},
	}
}

// NewMintToChecked builds an SPL Token MintToChecked instruction. Like
// MintTo but validates decimals matches the mint on chain.
func NewMintToChecked(mint, destination, authority solana.PublicKey, amount uint64, decimals uint8) solana.Instruction {
	return &genericIx{
		accounts: []*solana.AccountMeta{
			solana.NewAccountMeta(mint, false, true),
			solana.NewAccountMeta(destination, false, true),
			solana.NewAccountMeta(authority, true, false),
		},
		data: encoding.NewEncoder(10).U8(tagMintToChecked).U64(amount).U8(decimals).Bytes(),
	}
}

// NewBurnChecked builds an SPL Token BurnChecked instruction. Like Burn
// but validates decimals matches the mint on chain.
func NewBurnChecked(account, mint, authority solana.PublicKey, amount uint64, decimals uint8) solana.Instruction {
	return &genericIx{
		accounts: []*solana.AccountMeta{
			solana.NewAccountMeta(account, false, true),
			solana.NewAccountMeta(mint, false, true),
			solana.NewAccountMeta(authority, true, false),
		},
		data: encoding.NewEncoder(10).U8(tagBurnChecked).U64(amount).U8(decimals).Bytes(),
	}
}

// NewSyncNative builds an SPL Token SyncNative instruction that syncs the
// lamport balance of a wrapped-SOL token account with its actual lamport
// balance. Call this after transferring SOL directly to a wrapped-SOL
// account.
func NewSyncNative(account solana.PublicKey) solana.Instruction {
	return &genericIx{
		accounts: []*solana.AccountMeta{
			solana.NewAccountMeta(account, false, true),
		},
		data: []byte{tagSyncNative},
	}
}
