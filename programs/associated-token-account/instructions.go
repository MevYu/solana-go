// Package associatedtokenaccount provides typed instruction builders
// for the Associated Token Account program, the Solana program that
// canonicalises the mapping from (wallet, mint) pairs to a specific
// token account address.
//
// The associated token account for a given wallet and mint is a
// program-derived address computed deterministically from
// [wallet, tokenProgram, mint]. Any wallet that wants to hold
// balances in a given mint uses its ATA for that mint, and every
// dapp that sends tokens to a wallet addresses the wallet's ATA
// without coordination.
package associatedtokenaccount

import (
	"github.com/MevYu/solana-go"
	"github.com/MevYu/solana-go/programs/system"
)

// ProgramID is the canonical address of the Associated Token
// Account program.
var ProgramID = solana.MustPublicKey("ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL")

// FindAssociatedTokenAddress derives the associated token account
// address for (wallet, mint) under the given token program. Pass
// token.ProgramID for classic SPL tokens; for Token-2022 mints,
// pass token2022.ProgramID.
//
// The derivation uses the standard seed order
// [wallet, tokenProgram, mint] with the ATA program as the program
// id. It returns the PDA, the bump seed, and any error from
// FindProgramAddress.
func FindAssociatedTokenAddress(wallet, mint, tokenProgram solana.PublicKey) (solana.PublicKey, uint8, error) {
	seeds := [][]byte{
		wallet[:],
		tokenProgram[:],
		mint[:],
	}
	return solana.FindProgramAddress(seeds, ProgramID)
}

// buildAccountMetas constructs the standard account meta list used
// by both Create and CreateIdempotent. The ATA is derived
// internally so callers do not have to.
func buildAccountMetas(payer, wallet, mint, tokenProgram solana.PublicKey) ([]*solana.AccountMeta, error) {
	ata, _, err := FindAssociatedTokenAddress(wallet, mint, tokenProgram)
	if err != nil {
		return nil, err
	}
	return []*solana.AccountMeta{
		solana.NewAccountMeta(payer, true, true),
		solana.NewAccountMeta(ata, false, true),
		solana.NewAccountMeta(wallet, false, false),
		solana.NewAccountMeta(mint, false, false),
		solana.NewAccountMeta(system.ProgramID, false, false),
		solana.NewAccountMeta(tokenProgram, false, false),
	}, nil
}

// NewCreate builds an ATA Create instruction that allocates a new
// associated token account for wallet + mint and funds it with
// rent-exempt lamports paid by payer.
//
// Create fails if the account already exists. Use NewCreateIdempotent
// in retry-heavy flows where a prior attempt may have already
// created the account.
func NewCreate(payer, wallet, mint, tokenProgram solana.PublicKey) (solana.Instruction, error) {
	accs, err := buildAccountMetas(payer, wallet, mint, tokenProgram)
	if err != nil {
		return nil, err
	}
	return solana.NewInstruction(ProgramID, accs, []byte{}), nil // empty data selects the Create discriminator
}

// NewCreateIdempotent builds the idempotent variant of Create. It
// succeeds without error when the ATA already exists, making it
// the safer choice for retry-heavy flows and the default builder
// for most applications.
func NewCreateIdempotent(payer, wallet, mint, tokenProgram solana.PublicKey) (solana.Instruction, error) {
	accs, err := buildAccountMetas(payer, wallet, mint, tokenProgram)
	if err != nil {
		return nil, err
	}
	return solana.NewInstruction(ProgramID, accs, []byte{1}), nil // CreateIdempotent discriminator
}
