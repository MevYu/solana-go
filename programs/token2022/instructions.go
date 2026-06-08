// Package token2022 provides typed instruction builders for the
// SPL Token-2022 program (TokenzQd...), the extension-capable
// successor to the classic SPL Token program (Tokenkeg...).
//
// For the core instructions (Transfer, TransferChecked, MintTo,
// Burn, CloseAccount, InitializeMint2, InitializeAccount3), the
// wire format is byte-identical to classic SPL Token. This
// package reuses the builders from programs/token and wraps them
// to substitute Token-2022's program id.
//
// Extension instructions use a u8 discriminator (not u32). Families
// with multiple sub-operations use [family_tag u8, sub_tag u8, ...].
package token2022

import (
	"github.com/MevYu/solana-go"
	"github.com/MevYu/solana-go/encoding"
	"github.com/MevYu/solana-go/programs/token"
)

// ProgramID is the canonical address of the Token-2022 program,
// verified against the declare_id! call in the spl_token_2022
// interface crate at
// https://github.com/solana-program/token-2022.
var ProgramID = solana.MustPublicKey("TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb")

// wrapped substitutes a different program id for an inner
// solana.Instruction without touching its accounts or data. It is
// the mechanism we use to reuse the programs/token builders for
// Token-2022, since the wire format is identical except for the
// program id the transaction dispatches on.
type wrapped struct {
	inner solana.Instruction
}

func (w *wrapped) ProgramID() solana.PublicKey     { return ProgramID }
func (w *wrapped) Accounts() []*solana.AccountMeta { return w.inner.Accounts() }
func (w *wrapped) Data() ([]byte, error)           { return w.inner.Data() }

// Wrap wraps an arbitrary solana.Instruction so that its ProgramID
// resolves to Token-2022's program id. Use this as an escape hatch
// for instructions that are not yet covered by a typed builder in
// this package but are byte-identical to their SPL Token
// counterpart.
func Wrap(ix solana.Instruction) solana.Instruction {
	return &wrapped{inner: ix}
}

// NewTransfer builds a Token-2022 Transfer instruction. The wire
// format matches classic SPL Token exactly; only the dispatched
// program id differs.
func NewTransfer(source, destination, authority solana.PublicKey, amount uint64) solana.Instruction {
	return Wrap(token.NewTransfer(source, destination, authority, amount))
}

// NewTransferChecked builds a Token-2022 TransferChecked instruction.
func NewTransferChecked(source, mint, destination, authority solana.PublicKey, amount uint64, decimals uint8) solana.Instruction {
	return Wrap(token.NewTransferChecked(source, mint, destination, authority, amount, decimals))
}

// NewMintTo builds a Token-2022 MintTo instruction.
func NewMintTo(mint, destination, authority solana.PublicKey, amount uint64) solana.Instruction {
	return Wrap(token.NewMintTo(mint, destination, authority, amount))
}

// NewBurn builds a Token-2022 Burn instruction.
func NewBurn(account, mint, authority solana.PublicKey, amount uint64) solana.Instruction {
	return Wrap(token.NewBurn(account, mint, authority, amount))
}

// NewCloseAccount builds a Token-2022 CloseAccount instruction.
func NewCloseAccount(account, destination, authority solana.PublicKey) solana.Instruction {
	return Wrap(token.NewCloseAccount(account, destination, authority))
}

// NewInitializeMint2 builds a Token-2022 InitializeMint2 instruction.
// For mints that use Token-2022 extensions, additional instructions
// (typically InitializeXxxExtension) must be sent in the same transaction
// before the mint is finalised; those are not yet exposed by this package.
func NewInitializeMint2(mint solana.PublicKey, decimals uint8, mintAuthority, freezeAuthority solana.PublicKey) solana.Instruction {
	return Wrap(token.NewInitializeMint2(mint, decimals, mintAuthority, freezeAuthority))
}

// NewInitializeAccount3 builds a Token-2022 InitializeAccount3 instruction.
func NewInitializeAccount3(account, mint, owner solana.PublicKey) solana.Instruction {
	return Wrap(token.NewInitializeAccount3(account, mint, owner))
}

// optNonZeroKey returns p[:] when p is non-nil, or a 32-byte zero slice
// otherwise. It is the wire form of spl-pod's OptionalNonZeroPubkey:
// always 32 bytes, with the all-zero pubkey representing None. Used by
// most Token-2022 extension authority/address fields (mint-close,
// transfer-fee, interest-bearing, metadata-pointer, transfer-hook).
//
// This is NOT bincode/Borsh Option<Pubkey> (which would be 1-byte tag +
// payload) nor COption<Pubkey> (4-byte tag + 32-byte payload).
func optNonZeroKey(p *solana.PublicKey) []byte {
	if p == nil {
		return zeroPubkey[:]
	}
	return p[:]
}

var zeroPubkey [solana.PublicKeySize]byte

// ─── D1: Mint close authority (tag = 25) ────────────────────────────────────

// NewInitializeMintCloseAuthority builds an InitializeMintCloseAuthority
// instruction (tag 25). Call this before InitializeMint2 to allow the
// mint to be closed later. closeAuth may be nil to disable close.
func NewInitializeMintCloseAuthority(mint solana.PublicKey, closeAuth *solana.PublicKey) solana.Instruction {
	return solana.NewInstruction(
		ProgramID,
		[]*solana.AccountMeta{solana.NewAccountMeta(mint, false, true)},
		encoding.New().U8(25).Raw(optNonZeroKey(closeAuth)).Bytes(),
	)
}

// ─── D2: Non-transferable mint (tag = 32) ────────────────────────────────────

// NewInitializeNonTransferableMint builds an InitializeNonTransferableMint
// instruction (tag 32) that makes all tokens minted by this mint
// soul-bound (non-transferable between accounts).
func NewInitializeNonTransferableMint(mint solana.PublicKey) solana.Instruction {
	return solana.NewInstruction(
		ProgramID,
		[]*solana.AccountMeta{solana.NewAccountMeta(mint, false, true)},
		[]byte{32},
	)
}

// ─── D3: Permanent delegate (tag = 35) ───────────────────────────────────────

// NewInitializePermanentDelegate builds an InitializePermanentDelegate
// instruction (tag 35) that grants delegate irrevocable authority over
// all token accounts for this mint.
func NewInitializePermanentDelegate(mint, delegate solana.PublicKey) solana.Instruction {
	return solana.NewInstruction(
		ProgramID,
		[]*solana.AccountMeta{solana.NewAccountMeta(mint, false, true)},
		encoding.New().U8(35).Raw(delegate[:]).Bytes(),
	)
}

// ─── D4: Default account state (tag = 28) ────────────────────────────────────

// AccountState represents the frozen/thawed state of a token account.
type AccountState uint8

const (
	AccountStateUninitialized AccountState = 0
	AccountStateInitialized   AccountState = 1
	AccountStateFrozen        AccountState = 2
)

// NewInitializeDefaultAccountState builds an InitializeDefaultAccountState
// instruction (tag 28, sub 0) that sets the default state for newly created
// token accounts for this mint.
func NewInitializeDefaultAccountState(mint solana.PublicKey, state AccountState) solana.Instruction {
	return solana.NewInstruction(
		ProgramID,
		[]*solana.AccountMeta{solana.NewAccountMeta(mint, false, true)},
		[]byte{28, 0, byte(state)},
	)
}

// NewUpdateDefaultAccountState builds an UpdateDefaultAccountState instruction
// (tag 28, sub 1). freezeAuthority must sign.
func NewUpdateDefaultAccountState(mint, freezeAuthority solana.PublicKey, state AccountState) solana.Instruction {
	return solana.NewInstruction(
		ProgramID,
		[]*solana.AccountMeta{
			solana.NewAccountMeta(mint, false, true),
			solana.NewAccountMeta(freezeAuthority, true, false),
		},
		[]byte{28, 1, byte(state)},
	)
}

// ─── D5: Transfer fee family (tag = 26) ──────────────────────────────────────

// NewInitializeTransferFeeConfig builds an InitializeTransferFeeConfig
// instruction (tag 26, sub 0). Call before InitializeMint2. Either
// authority may be nil to disable that capability.
func NewInitializeTransferFeeConfig(
	mint solana.PublicKey,
	transferFeeAuthority, withdrawWithheldAuthority *solana.PublicKey,
	feeBasisPoints uint16, maximumFee uint64,
) solana.Instruction {
	return solana.NewInstruction(
		ProgramID,
		[]*solana.AccountMeta{solana.NewAccountMeta(mint, false, true)},
		// Both authorities are OptionalNonZeroPubkey (32 bytes raw, zero == None).
		encoding.New().
			U8(26).U8(0).
			Raw(optNonZeroKey(transferFeeAuthority)).
			Raw(optNonZeroKey(withdrawWithheldAuthority)).
			U16(feeBasisPoints).
			U64(maximumFee).
			Bytes(),
	)
}

// NewSetTransferFee builds a SetTransferFee instruction (tag 26, sub 5).
// authority is the transfer fee config authority and must sign.
func NewSetTransferFee(mint, authority solana.PublicKey, feeBasisPoints uint16, maximumFee uint64) solana.Instruction {
	return solana.NewInstruction(
		ProgramID,
		[]*solana.AccountMeta{
			solana.NewAccountMeta(mint, false, true),
			solana.NewAccountMeta(authority, true, false),
		},
		encoding.New().U8(26).U8(5).U16(feeBasisPoints).U64(maximumFee).Bytes(),
	)
}

// NewWithdrawWithheldTokensFromMint builds a WithdrawWithheldTokensFromMint
// instruction (tag 26, sub 2). withdrawAuthority must sign.
func NewWithdrawWithheldTokensFromMint(mint, destination, withdrawAuthority solana.PublicKey) solana.Instruction {
	return solana.NewInstruction(
		ProgramID,
		[]*solana.AccountMeta{
			solana.NewAccountMeta(mint, false, true),
			solana.NewAccountMeta(destination, false, true),
			solana.NewAccountMeta(withdrawAuthority, true, false),
		},
		[]byte{26, 2},
	)
}

// NewWithdrawWithheldTokensFromAccounts builds a
// WithdrawWithheldTokensFromAccounts instruction (tag 26, sub 3).
// withdrawAuthority must sign; sources are the accounts to harvest from.
func NewWithdrawWithheldTokensFromAccounts(mint, destination, withdrawAuthority solana.PublicKey, sources []solana.PublicKey) solana.Instruction {
	accounts := make([]*solana.AccountMeta, 0, 3+len(sources))
	accounts = append(accounts,
		solana.NewAccountMeta(mint, false, true),
		solana.NewAccountMeta(destination, false, true),
		solana.NewAccountMeta(withdrawAuthority, true, false),
	)
	for _, s := range sources {
		accounts = append(accounts, solana.NewAccountMeta(s, false, true))
	}
	return solana.NewInstruction(ProgramID, accounts, []byte{26, 3, byte(len(sources))})
}

// NewHarvestWithheldTokensToMint builds a HarvestWithheldTokensToMint
// instruction (tag 26, sub 4) that moves withheld fees from source token
// accounts back to the mint account.
func NewHarvestWithheldTokensToMint(mint solana.PublicKey, sources []solana.PublicKey) solana.Instruction {
	accounts := make([]*solana.AccountMeta, 0, 1+len(sources))
	accounts = append(accounts, solana.NewAccountMeta(mint, false, true))
	for _, s := range sources {
		accounts = append(accounts, solana.NewAccountMeta(s, false, true))
	}
	return solana.NewInstruction(ProgramID, accounts, []byte{26, 4})
}

// ─── D6: Interest-bearing mint (tag = 33) ────────────────────────────────────

// NewInitializeInterestBearingMint builds an InitializeInterestBearingMint
// instruction (tag 33, sub 0). rateAuthority may be nil. rate is in basis
// points per year (signed; negative = deflation).
func NewInitializeInterestBearingMint(mint solana.PublicKey, rateAuthority *solana.PublicKey, rate int16) solana.Instruction {
	return solana.NewInstruction(
		ProgramID,
		[]*solana.AccountMeta{solana.NewAccountMeta(mint, false, true)},
		// rate_authority is OptionalNonZeroPubkey (32 bytes raw, zero == None).
		encoding.New().
			U8(33).U8(0).
			Raw(optNonZeroKey(rateAuthority)).
			I16(rate).
			Bytes(),
	)
}

// NewUpdateInterestBearingMintRate builds an UpdateInterestBearingMintRate
// instruction (tag 33, sub 1). rateAuthority must sign.
func NewUpdateInterestBearingMintRate(mint, rateAuthority solana.PublicKey, rate int16) solana.Instruction {
	return solana.NewInstruction(
		ProgramID,
		[]*solana.AccountMeta{
			solana.NewAccountMeta(mint, false, true),
			solana.NewAccountMeta(rateAuthority, true, false),
		},
		encoding.New().U8(33).U8(1).I16(rate).Bytes(),
	)
}

// ─── D7: Metadata pointer (tag = 39) ─────────────────────────────────────────

// NewInitializeMetadataPointer builds an InitializeMetadataPointer
// instruction (tag 39, sub 0). Either authority or metadataAddress may be nil.
func NewInitializeMetadataPointer(mint solana.PublicKey, authority, metadataAddress *solana.PublicKey) solana.Instruction {
	return solana.NewInstruction(
		ProgramID,
		[]*solana.AccountMeta{solana.NewAccountMeta(mint, false, true)},
		// authority and metadata_address are both OptionalNonZeroPubkey (32B raw).
		encoding.New().
			U8(39).U8(0).
			Raw(optNonZeroKey(authority)).
			Raw(optNonZeroKey(metadataAddress)).
			Bytes(),
	)
}

// NewUpdateMetadataPointer builds an UpdateMetadataPointer instruction
// (tag 39, sub 1). authority must sign. metadataAddress may be nil to clear.
func NewUpdateMetadataPointer(mint, authority solana.PublicKey, metadataAddress *solana.PublicKey) solana.Instruction {
	return solana.NewInstruction(
		ProgramID,
		[]*solana.AccountMeta{
			solana.NewAccountMeta(mint, false, true),
			solana.NewAccountMeta(authority, true, false),
		},
		// metadata_address is OptionalNonZeroPubkey (32B raw, zero == clear).
		encoding.New().
			U8(39).U8(1).
			Raw(optNonZeroKey(metadataAddress)).
			Bytes(),
	)
}

// ─── D8: Transfer hook (tag = 36) ────────────────────────────────────────────

// NewInitializeTransferHook builds an InitializeTransferHook instruction
// (tag 36, sub 0). Either authority or programId may be nil.
func NewInitializeTransferHook(mint solana.PublicKey, authority, programId *solana.PublicKey) solana.Instruction {
	return solana.NewInstruction(
		ProgramID,
		[]*solana.AccountMeta{solana.NewAccountMeta(mint, false, true)},
		// authority and program_id are both OptionalNonZeroPubkey (32B raw).
		encoding.New().
			U8(36).U8(0).
			Raw(optNonZeroKey(authority)).
			Raw(optNonZeroKey(programId)).
			Bytes(),
	)
}

// NewUpdateTransferHook builds an UpdateTransferHook instruction
// (tag 36, sub 1). authority must sign. programId may be nil to clear.
func NewUpdateTransferHook(mint, authority solana.PublicKey, programId *solana.PublicKey) solana.Instruction {
	return solana.NewInstruction(
		ProgramID,
		[]*solana.AccountMeta{
			solana.NewAccountMeta(mint, false, true),
			solana.NewAccountMeta(authority, true, false),
		},
		// program_id is OptionalNonZeroPubkey (32B raw, zero == clear).
		encoding.New().
			U8(36).U8(1).
			Raw(optNonZeroKey(programId)).
			Bytes(),
	)
}

// ─── D9: Account-level extensions ────────────────────────────────────────────

// NewEnableRequiredMemoTransfers builds an EnableRequiredMemoTransfers
// instruction (tag 30, sub 0). owner must sign.
func NewEnableRequiredMemoTransfers(account, owner solana.PublicKey) solana.Instruction {
	return solana.NewInstruction(
		ProgramID,
		[]*solana.AccountMeta{
			solana.NewAccountMeta(account, false, true),
			solana.NewAccountMeta(owner, true, false),
		},
		[]byte{30, 0},
	)
}

// NewDisableRequiredMemoTransfers builds a DisableRequiredMemoTransfers
// instruction (tag 30, sub 1). owner must sign.
func NewDisableRequiredMemoTransfers(account, owner solana.PublicKey) solana.Instruction {
	return solana.NewInstruction(
		ProgramID,
		[]*solana.AccountMeta{
			solana.NewAccountMeta(account, false, true),
			solana.NewAccountMeta(owner, true, false),
		},
		[]byte{30, 1},
	)
}

// NewEnableCpiGuard builds an EnableCpiGuard instruction (tag 34, sub 0)
// that prevents instructions from CPI-calling token transfers on behalf
// of this account. owner must sign.
func NewEnableCpiGuard(account, owner solana.PublicKey) solana.Instruction {
	return solana.NewInstruction(
		ProgramID,
		[]*solana.AccountMeta{
			solana.NewAccountMeta(account, false, true),
			solana.NewAccountMeta(owner, true, false),
		},
		[]byte{34, 0},
	)
}

// NewDisableCpiGuard builds a DisableCpiGuard instruction (tag 34, sub 1).
// owner must sign.
func NewDisableCpiGuard(account, owner solana.PublicKey) solana.Instruction {
	return solana.NewInstruction(
		ProgramID,
		[]*solana.AccountMeta{
			solana.NewAccountMeta(account, false, true),
			solana.NewAccountMeta(owner, true, false),
		},
		[]byte{34, 1},
	)
}
