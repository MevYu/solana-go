package associatedtokenaccount

import (
	"testing"

	"github.com/MevYu/solana-go"
	"github.com/MevYu/solana-go/programs/system"
	"github.com/MevYu/solana-go/programs/token"
)

func TestProgramID_Canonical(t *testing.T) {
	if ProgramID.String() != "ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL" {
		t.Errorf("ProgramID = %s", ProgramID.String())
	}
}

func TestFindAssociatedTokenAddress_Deterministic(t *testing.T) {
	wallet := solana.PublicKey{0x01, 0x02, 0x03}
	mint := solana.PublicKey{0x04, 0x05, 0x06}

	ata1, bump1, err := FindAssociatedTokenAddress(wallet, mint, token.ProgramID)
	if err != nil {
		t.Fatal(err)
	}
	ata2, bump2, err := FindAssociatedTokenAddress(wallet, mint, token.ProgramID)
	if err != nil {
		t.Fatal(err)
	}
	if !ata1.Equal(ata2) {
		t.Errorf("same inputs produced different ATAs: %s vs %s", ata1, ata2)
	}
	if bump1 != bump2 {
		t.Errorf("bump mismatch: %d vs %d", bump1, bump2)
	}
}

func TestFindAssociatedTokenAddress_DifferentMintsDifferentATAs(t *testing.T) {
	wallet := solana.PublicKey{0x01}
	mint1 := solana.PublicKey{0x02}
	mint2 := solana.PublicKey{0x03}

	ata1, _, _ := FindAssociatedTokenAddress(wallet, mint1, token.ProgramID)
	ata2, _, _ := FindAssociatedTokenAddress(wallet, mint2, token.ProgramID)
	if ata1.Equal(ata2) {
		t.Error("different mints should produce different ATAs")
	}
}

func TestFindAssociatedTokenAddress_DifferentTokenProgramsDifferentATAs(t *testing.T) {
	wallet := solana.PublicKey{0x01}
	mint := solana.PublicKey{0x02}
	fakeTokenProgram := solana.PublicKey{0xFE}

	ata1, _, _ := FindAssociatedTokenAddress(wallet, mint, token.ProgramID)
	ata2, _, _ := FindAssociatedTokenAddress(wallet, mint, fakeTokenProgram)
	if ata1.Equal(ata2) {
		t.Error("different token programs should produce different ATAs")
	}
}

func TestNewCreate_AccountLayout(t *testing.T) {
	payer := solana.PublicKey{0x01}
	wallet := solana.PublicKey{0x02}
	mint := solana.PublicKey{0x03}

	ix, err := NewCreate(payer, wallet, mint, token.ProgramID)
	if err != nil {
		t.Fatal(err)
	}

	accs := ix.Accounts()
	if len(accs) != 6 {
		t.Fatalf("accounts len = %d, want 6", len(accs))
	}
	// [0] payer must be signer + writable
	if !accs[0].IsSigner || !accs[0].IsWritable || !accs[0].PublicKey.Equal(payer) {
		t.Errorf("payer meta wrong: %+v", accs[0])
	}
	// [1] ATA must be writable non-signer and match FindATA output
	expectedATA, _, _ := FindAssociatedTokenAddress(wallet, mint, token.ProgramID)
	if !accs[1].PublicKey.Equal(expectedATA) {
		t.Errorf("ATA meta mismatch")
	}
	if accs[1].IsSigner || !accs[1].IsWritable {
		t.Errorf("ATA meta role wrong")
	}
	// [2] wallet readonly
	if accs[2].IsWritable || !accs[2].PublicKey.Equal(wallet) {
		t.Errorf("wallet meta wrong")
	}
	// [3] mint readonly
	if accs[3].IsWritable || !accs[3].PublicKey.Equal(mint) {
		t.Errorf("mint meta wrong")
	}
	// [4] system program readonly
	if !accs[4].PublicKey.Equal(system.ProgramID) {
		t.Errorf("accs[4] should be system program")
	}
	// [5] token program readonly
	if !accs[5].PublicKey.Equal(token.ProgramID) {
		t.Errorf("accs[5] should be token program")
	}

	data, _ := ix.Data()
	if len(data) != 0 {
		t.Errorf("Create data should be empty, got %x", data)
	}
}

func TestNewCreateIdempotent_DataDiscriminator(t *testing.T) {
	payer := solana.PublicKey{0x01}
	wallet := solana.PublicKey{0x02}
	mint := solana.PublicKey{0x03}

	ix, err := NewCreateIdempotent(payer, wallet, mint, token.ProgramID)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := ix.Data()
	if len(data) != 1 || data[0] != 1 {
		t.Errorf("CreateIdempotent data = %x, want [01]", data)
	}
	if len(ix.Accounts()) != 6 {
		t.Errorf("accounts len = %d, want 6", len(ix.Accounts()))
	}
}

func TestATA_IntegratesWithNewMessage(t *testing.T) {
	payer := solana.PublicKey{0x01}
	wallet := solana.PublicKey{0x02}
	mint := solana.PublicKey{0x03}

	ix, err := NewCreateIdempotent(payer, wallet, mint, token.ProgramID)
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
