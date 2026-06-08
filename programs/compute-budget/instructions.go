package computebudget

import (
	"github.com/MevYu/solana-go"
	"github.com/MevYu/solana-go/encoding"
)

// Instruction tag values for the ComputeBudget program. Unlike
// System, ComputeBudget uses a single-byte tag (not a u32).
const (
	tagRequestHeapFrame    byte = 1
	tagSetComputeUnitLimit byte = 2
	tagSetComputeUnitPrice byte = 3
)

// NewSetComputeUnitLimit builds an instruction that sets the compute unit
// limit for the containing transaction. Transactions that omit this
// instruction get Solana's default limit (200,000 CUs as of late 2024).
// Specifying a tighter limit reduces the priority fee you pay at a given
// per-CU price and makes your transaction more competitive.
func NewSetComputeUnitLimit(units uint32) solana.Instruction {
	return solana.NewInstruction(ProgramID, nil, encoding.NewEncoder(5).U8(tagSetComputeUnitLimit).U32(units).Bytes())
}

// NewSetComputeUnitPrice builds an instruction that sets the price the
// caller is willing to pay per compute unit, in micro-lamports
// (1 lamport = 1,000,000 micro-lamports). This is the Solana equivalent
// of an EVM priority fee. Pick a reasonable value by calling
// Client.GetRecentPrioritizationFees and summarising with
// rpc.PriorityFeeStatsFromFees.
func NewSetComputeUnitPrice(microLamports uint64) solana.Instruction {
	return solana.NewInstruction(ProgramID, nil, encoding.NewEncoder(9).U8(tagSetComputeUnitPrice).U64(microLamports).Bytes())
}

// NewRequestHeapFrame builds an instruction that requests a larger program
// heap. The size must be a multiple of 1024 and is capped at 256 KiB by
// the runtime. Most programs do not need this.
func NewRequestHeapFrame(bytes uint32) solana.Instruction {
	return solana.NewInstruction(ProgramID, nil, encoding.NewEncoder(5).U8(tagRequestHeapFrame).U32(bytes).Bytes())
}
