// Package token provides typed instruction builders for the SPL
// Token program. The builders cover the common hot-path
// operations: transfer, mint, burn, close, initialize. Every
// builder returns a solana.Instruction ready for solana.NewMessage.
//
// This package binds the classic SPL Token program at
// TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA. For Token-2022
// support (the SPL Token program extension at TokenzQd...), see
// the sibling programs/token2022 package (to land in a follow-up
// PR).
package token

import "github.com/MevYu/solana-go"

// ProgramID is the canonical address of the SPL Token program.
var ProgramID = solana.MustPublicKey("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")
