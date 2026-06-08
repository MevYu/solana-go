// Package memo provides typed instruction builders for the SPL Memo
// program, which records arbitrary UTF-8 strings in the transaction log.
package memo

import solana "github.com/MevYu/solana-go"

// ProgramID is the canonical address of the SPL Memo v2 program.
var ProgramID = solana.MustPublicKey("MemoSq4gqABAXKb96qnH8TysNcWxMyWCqXgDLGmfcHr")

// Log returns an Instruction that records memo in the transaction log.
// Optionally pass signer accounts (e.g. the fee payer) to have them
// validate the memo; the on-chain program verifies their signatures.
func Log(memo string, signers ...*solana.AccountMeta) solana.Instruction {
	return solana.NewInstruction(ProgramID, signers, []byte(memo))
}
