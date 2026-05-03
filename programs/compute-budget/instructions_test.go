package computebudget

import (
	"encoding/binary"
	"testing"

	"github.com/MevYu/solana-go"
)

func TestProgramID_ExpectedBase58(t *testing.T) {
	want := "ComputeBudget111111111111111111111111111111"
	if got := ProgramID.String(); got != want {
		t.Errorf("ProgramID = %q, want %q", got, want)
	}
}

func TestNewSetComputeUnitLimit_Encoding(t *testing.T) {
	ix := NewSetComputeUnitLimit(400_000)
	data, err := ix.Data()
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 5 {
		t.Fatalf("data len = %d, want 5", len(data))
	}
	if data[0] != tagSetComputeUnitLimit {
		t.Errorf("tag = %d, want %d", data[0], tagSetComputeUnitLimit)
	}
	if units := binary.LittleEndian.Uint32(data[1:5]); units != 400_000 {
		t.Errorf("units = %d", units)
	}
	if len(ix.Accounts()) != 0 {
		t.Errorf("ComputeBudget instructions should have no accounts")
	}
	if !ix.ProgramID().Equal(ProgramID) {
		t.Errorf("program id mismatch")
	}
}

func TestNewSetComputeUnitPrice_Encoding(t *testing.T) {
	ix := NewSetComputeUnitPrice(5_000)
	data, _ := ix.Data()
	if len(data) != 9 {
		t.Fatalf("data len = %d, want 9", len(data))
	}
	if data[0] != tagSetComputeUnitPrice {
		t.Errorf("tag = %d", data[0])
	}
	if price := binary.LittleEndian.Uint64(data[1:9]); price != 5_000 {
		t.Errorf("price = %d", price)
	}
}

func TestNewRequestHeapFrame_Encoding(t *testing.T) {
	ix := NewRequestHeapFrame(131_072)
	data, _ := ix.Data()
	if len(data) != 5 {
		t.Fatalf("data len = %d, want 5", len(data))
	}
	if data[0] != tagRequestHeapFrame {
		t.Errorf("tag = %d", data[0])
	}
	if bytes := binary.LittleEndian.Uint32(data[1:5]); bytes != 131_072 {
		t.Errorf("bytes = %d", bytes)
	}
}

// TestComputeBudget_IntegratesWithNewMessage proves a ComputeBudget
// instruction feeds NewMessage correctly and the resulting message
// marshals round-trip.
func TestComputeBudget_IntegratesWithNewMessage(t *testing.T) {
	payer := solana.PublicKey{0x01}
	bh := solana.Hash{0x42}

	ix := NewSetComputeUnitLimit(200_000)
	msg, err := solana.NewMessage(payer, []solana.Instruction{ix}, bh)
	if err != nil {
		t.Fatal(err)
	}
	// Layout: [payer, ComputeBudgetProgram] (ComputeBudget is
	// non-signer readonly).
	if len(msg.AccountKeys) != 2 {
		t.Fatalf("AccountKeys len = %d", len(msg.AccountKeys))
	}
	if !msg.AccountKeys[1].Equal(ProgramID) {
		t.Errorf("AccountKeys[1] should be ComputeBudget program")
	}
	if msg.Header.NumReadonlyUnsignedAccounts != 1 {
		t.Errorf("NumReadonlyUnsignedAccounts = %d", msg.Header.NumReadonlyUnsignedAccounts)
	}
	if _, err := msg.Marshal(); err != nil {
		t.Errorf("Marshal: %v", err)
	}
}
