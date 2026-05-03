package tx

import (
	"encoding/binary"

	solana "github.com/MevYu/solana-go"
)

// ComputeBudgetProgramID is the on-chain address of the Compute Budget
// program, which controls per-transaction compute units and priority fees.
var ComputeBudgetProgramID = solana.MustPublicKey("ComputeBudget111111111111111111111111111111")

const (
	computeBudgetInstructionSetComputeUnitLimit = uint8(2)
	computeBudgetInstructionSetComputeUnitPrice = uint8(3)
)

// computeBudgetInstruction is a minimal Instruction implementation for
// Compute Budget program calls.
type computeBudgetInstruction struct {
	programID solana.PublicKey
	data      []byte
}

func (i *computeBudgetInstruction) ProgramID() solana.PublicKey    { return i.programID }
func (i *computeBudgetInstruction) Accounts() []*solana.AccountMeta { return nil }
func (i *computeBudgetInstruction) Data() ([]byte, error)          { return i.data, nil }

// SetComputeUnitLimit returns an Instruction that sets the per-transaction
// compute unit limit. Use this to cap compute consumption and avoid
// overpaying on compute fees.
//
// units should be determined by simulation (SimulateTransaction) with a
// safety margin (e.g. actual * 1.1).
func SetComputeUnitLimit(units uint32) solana.Instruction {
	data := make([]byte, 5)
	data[0] = computeBudgetInstructionSetComputeUnitLimit
	binary.LittleEndian.PutUint32(data[1:], units)
	return &computeBudgetInstruction{programID: ComputeBudgetProgramID, data: data}
}

// SetComputeUnitPrice returns an Instruction that sets the priority fee
// in micro-lamports per compute unit. Higher fees increase the probability
// of landing a transaction during congestion.
//
// microLamports is the per-compute-unit fee in micro-lamports (1 lamport =
// 1,000,000 micro-lamports).
func SetComputeUnitPrice(microLamports uint64) solana.Instruction {
	data := make([]byte, 9)
	data[0] = computeBudgetInstructionSetComputeUnitPrice
	binary.LittleEndian.PutUint64(data[1:], microLamports)
	return &computeBudgetInstruction{programID: ComputeBudgetProgramID, data: data}
}
