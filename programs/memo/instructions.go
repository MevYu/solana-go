// Package memo provides typed instruction builders for the SPL Memo
// program, which records arbitrary UTF-8 strings in the transaction log.
package memo

import solana "github.com/MevYu/solana-go"

// ProgramID is the canonical address of the SPL Memo v2 program.
var ProgramID = solana.MustPublicKey("MemoSq4gqABAXKb96qnH8TysNcWxMyWCqXgDLGmfcHr")

// memoInstruction is a minimal Instruction for the Memo program.
type memoInstruction struct {
	signers []*solana.AccountMeta
	data    []byte
}

func (m *memoInstruction) ProgramID() solana.PublicKey     { return ProgramID }
func (m *memoInstruction) Accounts() []*solana.AccountMeta { return m.signers }
func (m *memoInstruction) Data() ([]byte, error)           { return m.data, nil }

// Log returns an Instruction that records memo in the transaction log.
// Optionally pass signer accounts (e.g. the fee payer) to have them
// validate the memo; the on-chain program verifies their signatures.
func Log(memo string, signers ...*solana.AccountMeta) solana.Instruction {
	return &memoInstruction{
		signers: signers,
		data:    []byte(memo),
	}
}
