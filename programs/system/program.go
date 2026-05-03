// Package system provides typed instruction builders for the
// Solana System program. Each builder returns a solana.Instruction
// with the program id, account metas, and encoded data already
// filled in, so callers can assemble a transaction by simply
// collecting the instructions they want and passing them to
// solana.NewMessage.
package system

import "github.com/MevYu/solana-go"

// ProgramID is the canonical address of the Solana System program.
// Its on-chain pubkey is 32 zero bytes, encoded as the base58
// string "11111111111111111111111111111111".
var ProgramID = solana.PublicKey{}
