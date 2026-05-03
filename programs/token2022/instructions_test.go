package token2022

import (
	"bytes"
	"testing"

	"github.com/MevYu/solana-go"
	"github.com/MevYu/solana-go/programs/token"
)

func TestProgramID_Canonical(t *testing.T) {
	if ProgramID.String() != "TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb" {
		t.Errorf("ProgramID = %s", ProgramID.String())
	}
	if ProgramID.Equal(token.ProgramID) {
		t.Error("Token-2022 ProgramID must differ from classic SPL Token")
	}
}

func TestNewTransfer_ProgramIDSubstituted(t *testing.T) {
	src := solana.PublicKey{0x01}
	dst := solana.PublicKey{0x02}
	auth := solana.PublicKey{0x03}

	ix22 := NewTransfer(src, dst, auth, 1000)
	ixLegacy := token.NewTransfer(src, dst, auth, 1000)

	// Program ids must differ.
	if ix22.ProgramID().Equal(ixLegacy.ProgramID()) {
		t.Error("Token-2022 and legacy Token should have distinct program ids")
	}
	if !ix22.ProgramID().Equal(ProgramID) {
		t.Error("ix22 ProgramID mismatch")
	}

	// But wire data must be byte-identical (Token-2022 Transfer is
	// the same instruction format as classic SPL Token Transfer).
	d22, _ := ix22.Data()
	dLegacy, _ := ixLegacy.Data()
	if !bytes.Equal(d22, dLegacy) {
		t.Errorf("data should match: %x vs %x", d22, dLegacy)
	}

	// And account metas must be identical.
	a22 := ix22.Accounts()
	aLegacy := ixLegacy.Accounts()
	if len(a22) != len(aLegacy) {
		t.Fatalf("account counts differ: %d vs %d", len(a22), len(aLegacy))
	}
	for i := range a22 {
		if !a22[i].PublicKey.Equal(aLegacy[i].PublicKey) {
			t.Errorf("account[%d] pubkey mismatch", i)
		}
		if a22[i].IsSigner != aLegacy[i].IsSigner {
			t.Errorf("account[%d] IsSigner mismatch", i)
		}
		if a22[i].IsWritable != aLegacy[i].IsWritable {
			t.Errorf("account[%d] IsWritable mismatch", i)
		}
	}
}

func TestNewMintTo_ProgramIDSubstituted(t *testing.T) {
	mint := solana.PublicKey{0x01}
	dst := solana.PublicKey{0x02}
	auth := solana.PublicKey{0x03}

	ix := NewMintTo(mint, dst, auth, 500)
	if !ix.ProgramID().Equal(ProgramID) {
		t.Errorf("ProgramID = %s", ix.ProgramID())
	}
	// Wire format must match legacy.
	d22, _ := ix.Data()
	dLegacy, _ := token.NewMintTo(mint, dst, auth, 500).Data()
	if !bytes.Equal(d22, dLegacy) {
		t.Errorf("data mismatch")
	}
}

func TestWrap_SubstitutesProgramID(t *testing.T) {
	inner := token.NewCloseAccount(
		solana.PublicKey{0x01},
		solana.PublicKey{0x02},
		solana.PublicKey{0x03},
	)
	wrapped := Wrap(inner)
	if wrapped.ProgramID().Equal(inner.ProgramID()) {
		t.Error("Wrap should substitute ProgramID")
	}
	if !wrapped.ProgramID().Equal(ProgramID) {
		t.Error("Wrap should substitute Token-2022 ProgramID")
	}
}

func TestToken2022_IntegratesWithNewMessage(t *testing.T) {
	payer := solana.PublicKey{0x01}
	src := solana.PublicKey{0x02}
	dst := solana.PublicKey{0x03}
	mint := solana.PublicKey{0x04}

	ix := NewTransferChecked(src, mint, dst, payer, 100, 6)
	msg, err := solana.NewMessage(payer, []solana.Instruction{ix}, solana.Hash{0x42})
	if err != nil {
		t.Fatal(err)
	}
	// The Token-2022 program id should appear as a readonly
	// unsigned account in the compiled message, NOT the classic
	// Token program id.
	foundToken2022 := false
	for _, k := range msg.AccountKeys {
		if k.Equal(ProgramID) {
			foundToken2022 = true
		}
		if k.Equal(token.ProgramID) {
			t.Error("classic Token ProgramID should not appear in Token-2022 message")
		}
	}
	if !foundToken2022 {
		t.Error("Token-2022 ProgramID missing from AccountKeys")
	}
	if _, err := msg.Marshal(); err != nil {
		t.Errorf("Marshal: %v", err)
	}
}
