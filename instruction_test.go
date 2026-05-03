package solana

import "testing"

// fakeInstruction is a minimal Instruction implementation used to
// assert the interface shape and to exercise AccountMeta ordering.
type fakeInstruction struct {
	programID PublicKey
	accounts  []*AccountMeta
	data      []byte
}

func (i *fakeInstruction) ProgramID() PublicKey     { return i.programID }
func (i *fakeInstruction) Accounts() []*AccountMeta { return i.accounts }
func (i *fakeInstruction) Data() ([]byte, error)    { return i.data, nil }

func TestInstruction_InterfaceShape(t *testing.T) {
	var _ Instruction = (*fakeInstruction)(nil)
}

func TestNewAccountMeta(t *testing.T) {
	pk, err := PublicKeyFromBase58(tokenProgramID)
	if err != nil {
		t.Fatal(err)
	}
	m := NewAccountMeta(pk, true, false)
	if !m.PublicKey.Equal(pk) {
		t.Error("PublicKey mismatch")
	}
	if !m.IsSigner {
		t.Error("IsSigner should be true")
	}
	if m.IsWritable {
		t.Error("IsWritable should be false")
	}
}

func TestInstruction_AccountsPositional(t *testing.T) {
	payer, _ := PublicKeyFromBase58(tokenProgramID)
	var dest PublicKey
	dest[0] = 1

	inst := &fakeInstruction{
		programID: payer,
		accounts: []*AccountMeta{
			NewAccountMeta(payer, true, true),
			NewAccountMeta(dest, false, true),
		},
		data: []byte{0x02, 0x00, 0x00, 0x00},
	}

	accs := inst.Accounts()
	if len(accs) != 2 {
		t.Fatalf("len = %d, want 2", len(accs))
	}
	if !accs[0].PublicKey.Equal(payer) || !accs[0].IsSigner || !accs[0].IsWritable {
		t.Errorf("accs[0] = %+v", accs[0])
	}
	if !accs[1].PublicKey.Equal(dest) || accs[1].IsSigner || !accs[1].IsWritable {
		t.Errorf("accs[1] = %+v", accs[1])
	}
	if !inst.ProgramID().Equal(payer) {
		t.Error("ProgramID mismatch")
	}
	data, err := inst.Data()
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 4 {
		t.Errorf("data len = %d, want 4", len(data))
	}
}
