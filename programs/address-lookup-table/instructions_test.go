package addresslookuptable

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/MevYu/solana-go"
	"github.com/MevYu/solana-go/programs/system"
)

func TestProgramID_Canonical(t *testing.T) {
	if ProgramID.String() != "AddressLookupTab1e1111111111111111111111111" {
		t.Errorf("ProgramID = %s", ProgramID.String())
	}
}

func TestDeriveLookupTableAddress_Deterministic(t *testing.T) {
	authority := solana.PublicKey{0x01, 0x02}
	pk1, bump1, err := DeriveLookupTableAddress(authority, 123456789)
	if err != nil {
		t.Fatal(err)
	}
	pk2, bump2, err := DeriveLookupTableAddress(authority, 123456789)
	if err != nil {
		t.Fatal(err)
	}
	if !pk1.Equal(pk2) || bump1 != bump2 {
		t.Errorf("same inputs produced different PDAs: %s,%d vs %s,%d", pk1, bump1, pk2, bump2)
	}
}

func TestDeriveLookupTableAddress_DifferentSlotsDifferentPDAs(t *testing.T) {
	authority := solana.PublicKey{0x01}
	pk1, _, _ := DeriveLookupTableAddress(authority, 100)
	pk2, _, _ := DeriveLookupTableAddress(authority, 101)
	if pk1.Equal(pk2) {
		t.Error("different slots should produce different PDAs")
	}
}

func TestNewCreateLookupTable_Encoding(t *testing.T) {
	authority := solana.PublicKey{0x01}
	payer := solana.PublicKey{0x02}
	slot := uint64(284518234)

	ix, table, err := NewCreateLookupTable(authority, payer, slot)
	if err != nil {
		t.Fatal(err)
	}
	expected, bump, _ := DeriveLookupTableAddress(authority, slot)
	if !table.Equal(expected) {
		t.Errorf("returned table address mismatch")
	}

	data, _ := ix.Data()
	if len(data) != 13 {
		t.Fatalf("data len = %d, want 13", len(data))
	}
	if tag := binary.LittleEndian.Uint32(data[0:4]); tag != tagCreateLookupTable {
		t.Errorf("tag = %d", tag)
	}
	if gotSlot := binary.LittleEndian.Uint64(data[4:12]); gotSlot != slot {
		t.Errorf("slot = %d", gotSlot)
	}
	if data[12] != bump {
		t.Errorf("bump = %d, want %d", data[12], bump)
	}

	accs := ix.Accounts()
	if len(accs) != 4 {
		t.Fatalf("accounts len = %d", len(accs))
	}
	if !accs[0].PublicKey.Equal(table) || !accs[0].IsWritable {
		t.Error("[0] should be table (writable)")
	}
	if !accs[1].IsSigner || accs[1].IsWritable {
		t.Error("[1] authority should be signer readonly")
	}
	if !accs[2].IsSigner || !accs[2].IsWritable {
		t.Error("[2] payer should be signer writable")
	}
	if !accs[3].PublicKey.Equal(system.ProgramID) {
		t.Error("[3] should be system program")
	}
}

func TestNewFreezeLookupTable_Encoding(t *testing.T) {
	table := solana.PublicKey{0xAA}
	authority := solana.PublicKey{0xBB}
	ix := NewFreezeLookupTable(table, authority)
	data, _ := ix.Data()
	if len(data) != 4 {
		t.Fatalf("data len = %d", len(data))
	}
	if tag := binary.LittleEndian.Uint32(data[0:4]); tag != tagFreezeLookupTable {
		t.Errorf("tag = %d", tag)
	}
	accs := ix.Accounts()
	if len(accs) != 2 || !accs[1].IsSigner {
		t.Errorf("accounts wrong: %+v", accs)
	}
}

func TestNewExtendLookupTable_Encoding(t *testing.T) {
	table := solana.PublicKey{0xAA}
	authority := solana.PublicKey{0xBB}
	payer := solana.PublicKey{0xCC}
	addr1 := solana.PublicKey{0x01}
	addr2 := solana.PublicKey{0x02}

	ix := NewExtendLookupTable(table, authority, payer, []solana.PublicKey{addr1, addr2})
	data, _ := ix.Data()
	// bincode wire: [u32 tag, u64 count, addr×count]; addresses start at 12.
	expectedLen := 4 + 8 + 32*2
	if len(data) != expectedLen {
		t.Fatalf("data len = %d, want %d", len(data), expectedLen)
	}
	if tag := binary.LittleEndian.Uint32(data[0:4]); tag != tagExtendLookupTable {
		t.Errorf("tag = %d", tag)
	}
	if count := binary.LittleEndian.Uint64(data[4:12]); count != 2 {
		t.Errorf("count = %d", count)
	}
	if !bytes.Equal(data[12:44], addr1[:]) {
		t.Error("addr1 mismatch")
	}
	if !bytes.Equal(data[44:76], addr2[:]) {
		t.Error("addr2 mismatch")
	}
	if len(ix.Accounts()) != 4 {
		t.Errorf("accounts len = %d", len(ix.Accounts()))
	}
}

func TestNewDeactivateLookupTable_Encoding(t *testing.T) {
	table := solana.PublicKey{0xAA}
	authority := solana.PublicKey{0xBB}
	ix := NewDeactivateLookupTable(table, authority)
	data, _ := ix.Data()
	if tag := binary.LittleEndian.Uint32(data[0:4]); tag != tagDeactivateLookupTable {
		t.Errorf("tag = %d", tag)
	}
}

func TestNewCloseLookupTable_Encoding(t *testing.T) {
	table := solana.PublicKey{0xAA}
	authority := solana.PublicKey{0xBB}
	recipient := solana.PublicKey{0xCC}
	ix := NewCloseLookupTable(table, authority, recipient)
	data, _ := ix.Data()
	if tag := binary.LittleEndian.Uint32(data[0:4]); tag != tagCloseLookupTable {
		t.Errorf("tag = %d", tag)
	}
	accs := ix.Accounts()
	if len(accs) != 3 {
		t.Fatalf("accounts len = %d", len(accs))
	}
	if !accs[2].IsWritable || !accs[2].PublicKey.Equal(recipient) {
		t.Errorf("recipient meta wrong: %+v", accs[2])
	}
}

func TestALT_IntegratesWithNewMessage(t *testing.T) {
	authority := solana.PublicKey{0x01}
	payer := solana.PublicKey{0x02}
	slot := uint64(123456)

	ix, _, err := NewCreateLookupTable(authority, payer, slot)
	if err != nil {
		t.Fatal(err)
	}
	msg, err := solana.NewMessage(payer, []solana.Instruction{ix}, solana.Hash{0x42})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := msg.Marshal(); err != nil {
		t.Errorf("Marshal: %v", err)
	}
}
