// Package stake provides typed instruction builders for the Solana
// Stake program. Each builder returns a solana.Instruction with the
// program id, account metas, and encoded data already filled in, so
// callers can assemble a transaction by simply collecting the
// instructions they want and passing them to solana.NewMessage.
package stake

import solana "github.com/MevYu/solana-go"

// ProgramID is the canonical address of the Solana Stake program:
// Stake11111111111111111111111111111111111111.
var ProgramID = solana.MustPublicKey("Stake11111111111111111111111111111111111111")
