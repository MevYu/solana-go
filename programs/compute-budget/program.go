// Package computebudget provides typed instruction builders for
// the Solana ComputeBudget program.
//
// ComputeBudget instructions do not take account inputs; they are
// metadata instructions that tune the compute unit limit, the
// priority fee per compute unit, and the program heap size for the
// containing transaction. Include one or more at the start of your
// instruction list to control your transaction's resource budget.
package computebudget

import "github.com/MevYu/solana-go"

// ProgramID is the canonical address of the Solana ComputeBudget
// program: ComputeBudget111111111111111111111111111111.
var ProgramID = solana.MustPublicKey("ComputeBudget111111111111111111111111111111")
