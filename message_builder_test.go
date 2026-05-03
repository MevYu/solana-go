package solana

import (
	"bytes"
	"testing"
)

// testInstruction is a minimal Instruction used only to drive
// NewMessage tests without pulling in any program binding.
type testInstruction struct {
	pid  PublicKey
	meta []*AccountMeta
	data []byte
}

func (i *testInstruction) ProgramID() PublicKey     { return i.pid }
func (i *testInstruction) Accounts() []*AccountMeta { return i.meta }
func (i *testInstruction) Data() ([]byte, error)    { return i.data, nil }

func TestMustPublicKey_Valid(t *testing.T) {
	pk := MustPublicKey(tokenProgramID)
	if pk.IsZero() {
		t.Error("pk should not be zero")
	}
}

func TestMustPublicKey_InvalidPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for invalid base58")
		}
	}()
	_ = MustPublicKey("not a real base58 key")
}

func TestNewMessage_SingleInstruction(t *testing.T) {
	payer := PublicKey{0x01}
	recipient := PublicKey{0x02}
	systemProg := PublicKey{} // all zeros == system program

	ix := &testInstruction{
		pid: systemProg,
		meta: []*AccountMeta{
			NewAccountMeta(payer, true, true),
			NewAccountMeta(recipient, false, true),
		},
		data: []byte{0x02, 0x00, 0x00, 0x00, 0xe8, 0x03, 0, 0, 0, 0, 0, 0},
	}

	msg, err := NewMessage(payer, []Instruction{ix}, Hash{0x42})
	if err != nil {
		t.Fatal(err)
	}

	// Expected layout: [payer, recipient, systemProgram]
	if len(msg.AccountKeys) != 3 {
		t.Fatalf("AccountKeys len = %d, want 3", len(msg.AccountKeys))
	}
	if !msg.AccountKeys[0].Equal(payer) {
		t.Errorf("AccountKeys[0] should be payer")
	}
	if !msg.AccountKeys[1].Equal(recipient) {
		t.Errorf("AccountKeys[1] should be recipient")
	}
	if !msg.AccountKeys[2].Equal(systemProg) {
		t.Errorf("AccountKeys[2] should be system program")
	}

	// Header: 1 signer (payer), 0 readonly signed, 1 readonly unsigned (system)
	if msg.Header.NumRequiredSignatures != 1 {
		t.Errorf("NumRequiredSignatures = %d", msg.Header.NumRequiredSignatures)
	}
	if msg.Header.NumReadonlySignedAccounts != 0 {
		t.Errorf("NumReadonlySignedAccounts = %d", msg.Header.NumReadonlySignedAccounts)
	}
	if msg.Header.NumReadonlyUnsignedAccounts != 1 {
		t.Errorf("NumReadonlyUnsignedAccounts = %d", msg.Header.NumReadonlyUnsignedAccounts)
	}

	// Instruction should reference accounts [0, 1] and program 2.
	if len(msg.Instructions) != 1 {
		t.Fatalf("Instructions len = %d", len(msg.Instructions))
	}
	cix := msg.Instructions[0]
	if cix.ProgramIDIndex != 2 {
		t.Errorf("ProgramIDIndex = %d, want 2", cix.ProgramIDIndex)
	}
	if !bytes.Equal(cix.Accounts, []byte{0, 1}) {
		t.Errorf("Accounts = %v, want [0 1]", cix.Accounts)
	}
	if !bytes.Equal(cix.Data, ix.data) {
		t.Errorf("Data mismatch")
	}
}

func TestNewMessage_MultipleInstructionsShareAccounts(t *testing.T) {
	payer := PublicKey{0x01}
	shared := PublicKey{0x02}
	prog1 := PublicKey{0x03}
	prog2 := PublicKey{0x04}

	ix1 := &testInstruction{
		pid: prog1,
		meta: []*AccountMeta{
			NewAccountMeta(payer, true, true),
			NewAccountMeta(shared, false, true),
		},
		data: []byte{0xaa},
	}
	ix2 := &testInstruction{
		pid: prog2,
		meta: []*AccountMeta{
			NewAccountMeta(shared, false, true),
		},
		data: []byte{0xbb},
	}

	msg, err := NewMessage(payer, []Instruction{ix1, ix2}, Hash{0x42})
	if err != nil {
		t.Fatal(err)
	}

	// Expected layout: [payer, shared, prog1, prog2]
	if len(msg.AccountKeys) != 4 {
		t.Fatalf("AccountKeys len = %d, want 4", len(msg.AccountKeys))
	}
	if !msg.AccountKeys[0].Equal(payer) {
		t.Errorf("[0] should be payer")
	}
	if !msg.AccountKeys[1].Equal(shared) {
		t.Errorf("[1] should be shared (non-signer writable)")
	}
	// The two programs are readonly-unsigned, ordered by first-seen.
	if !msg.AccountKeys[2].Equal(prog1) {
		t.Errorf("[2] should be prog1")
	}
	if !msg.AccountKeys[3].Equal(prog2) {
		t.Errorf("[3] should be prog2")
	}

	if msg.Header.NumRequiredSignatures != 1 {
		t.Errorf("NumRequiredSignatures = %d", msg.Header.NumRequiredSignatures)
	}
	if msg.Header.NumReadonlyUnsignedAccounts != 2 {
		t.Errorf("NumReadonlyUnsignedAccounts = %d, want 2", msg.Header.NumReadonlyUnsignedAccounts)
	}
}

func TestNewMessage_RoleUpgradeCumulative(t *testing.T) {
	// Same account appearing as signer+readonly then non-signer+writable
	// should end up as signer+writable.
	payer := PublicKey{0x01}
	shared := PublicKey{0x02}
	prog := PublicKey{0x99}

	ix1 := &testInstruction{
		pid: prog,
		meta: []*AccountMeta{
			NewAccountMeta(payer, true, true),
			NewAccountMeta(shared, true, false), // signer, readonly
		},
		data: []byte{0x01},
	}
	ix2 := &testInstruction{
		pid: prog,
		meta: []*AccountMeta{
			NewAccountMeta(shared, false, true), // non-signer, writable
		},
		data: []byte{0x02},
	}

	msg, err := NewMessage(payer, []Instruction{ix1, ix2}, Hash{})
	if err != nil {
		t.Fatal(err)
	}

	// shared should be signer+writable (upgraded from both).
	// Expected layout: [payer, shared, prog] all in the right buckets.
	if !msg.AccountKeys[0].Equal(payer) {
		t.Errorf("[0] should be payer")
	}
	if !msg.AccountKeys[1].Equal(shared) {
		t.Errorf("[1] should be shared; got AccountKeys=%v", msg.AccountKeys)
	}
	if msg.Header.NumRequiredSignatures != 2 {
		t.Errorf("NumRequiredSignatures = %d, want 2", msg.Header.NumRequiredSignatures)
	}
	if msg.Header.NumReadonlySignedAccounts != 0 {
		t.Errorf("NumReadonlySignedAccounts = %d, want 0", msg.Header.NumReadonlySignedAccounts)
	}
}

func TestNewMessage_EmptyInstructions(t *testing.T) {
	if _, err := NewMessage(PublicKey{0x01}, nil, Hash{}); err == nil {
		t.Fatal("expected error for empty instructions")
	}
}

func TestNewMessage_NilInstruction(t *testing.T) {
	if _, err := NewMessage(PublicKey{0x01}, []Instruction{nil}, Hash{}); err == nil {
		t.Fatal("expected error for nil instruction")
	}
}

func TestNewMessage_PayerAlwaysFirst(t *testing.T) {
	// Even when the instruction doesn't mention the payer, it
	// should end up as AccountKeys[0] with signer+writable role.
	payer := PublicKey{0xFF}
	other := PublicKey{0x01}
	prog := PublicKey{0x10}

	ix := &testInstruction{
		pid: prog,
		meta: []*AccountMeta{
			NewAccountMeta(other, false, true),
		},
		data: []byte{0x00},
	}
	msg, err := NewMessage(payer, []Instruction{ix}, Hash{})
	if err != nil {
		t.Fatal(err)
	}
	if !msg.AccountKeys[0].Equal(payer) {
		t.Errorf("payer should be first, got AccountKeys=%v", msg.AccountKeys)
	}
	if msg.Header.NumRequiredSignatures != 1 {
		t.Errorf("NumRequiredSignatures = %d", msg.Header.NumRequiredSignatures)
	}
}
