package solana

// AccountMeta describes an account's role in an instruction: which
// account is referenced, whether it must sign the transaction, and
// whether the instruction may write to it.
//
// Order matters: programs read their inputs positionally, so the
// slices returned by Instruction.Accounts are ordered exactly as the
// target program expects them.
type AccountMeta struct {
	PublicKey  PublicKey
	IsSigner   bool
	IsWritable bool
}

// NewAccountMeta constructs an AccountMeta.
func NewAccountMeta(pk PublicKey, isSigner, isWritable bool) *AccountMeta {
	return &AccountMeta{
		PublicKey:  pk,
		IsSigner:   isSigner,
		IsWritable: isWritable,
	}
}

// Instruction is the common interface implemented by every program
// instruction builder. An Instruction names its program, lists the
// accounts it consumes in positional order, and returns its
// serialized data bytes.
//
// Implementations must not panic on invalid configuration; they
// return a descriptive error from Data() instead.
type Instruction interface {
	// ProgramID returns the public key of the program that will run
	// this instruction.
	ProgramID() PublicKey

	// Accounts returns the account metas in positional order. The
	// returned slice is read-only; callers must not mutate it.
	Accounts() []*AccountMeta

	// Data returns the serialized instruction data or an error
	// describing why the instruction cannot be built.
	Data() ([]byte, error)
}

// instructionData is the canonical Instruction implementation returned by
// NewInstruction: a program id, positional account metas, and pre-encoded
// data. It replaces the per-package wrapper types every program builder
// used to define.
type instructionData struct {
	programID PublicKey
	accounts  []*AccountMeta
	data      []byte
}

func (i *instructionData) ProgramID() PublicKey     { return i.programID }
func (i *instructionData) Accounts() []*AccountMeta { return i.accounts }
func (i *instructionData) Data() ([]byte, error)    { return i.data, nil }

// NewInstruction builds an Instruction from a program id, positional
// account metas, and pre-encoded instruction data. Program packages use it
// instead of each defining their own Instruction wrapper. accounts may be
// nil for precompile-style instructions that consume no accounts.
//
// The accounts slice is retained, not copied; callers must not mutate it
// after the call, matching the read-only contract of Instruction.Accounts.
func NewInstruction(programID PublicKey, accounts []*AccountMeta, data []byte) Instruction {
	return &instructionData{programID: programID, accounts: accounts, data: data}
}
