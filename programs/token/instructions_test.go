package token

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/MevYu/solana-go"
)

func TestProgramID_Canonical(t *testing.T) {
	if ProgramID.String() != "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA" {
		t.Errorf("ProgramID = %s", ProgramID.String())
	}
}

func TestNewTransfer_Encoding(t *testing.T) {
	src := solana.PublicKey{0x01}
	dst := solana.PublicKey{0x02}
	auth := solana.PublicKey{0x03}
	ix := NewTransfer(src, dst, auth, 1_000_000)

	data, _ := ix.Data()
	if len(data) != 9 || data[0] != tagTransfer {
		t.Errorf("data = %x", data)
	}
	if amt := binary.LittleEndian.Uint64(data[1:9]); amt != 1_000_000 {
		t.Errorf("amount = %d", amt)
	}
	accs := ix.Accounts()
	if len(accs) != 3 {
		t.Fatalf("accounts len = %d", len(accs))
	}
	if accs[0].IsSigner || !accs[0].IsWritable {
		t.Errorf("source meta wrong")
	}
	if accs[2].IsWritable || !accs[2].IsSigner {
		t.Errorf("authority meta wrong")
	}
}

func TestNewTransferChecked_Encoding(t *testing.T) {
	src := solana.PublicKey{0x01}
	mint := solana.PublicKey{0x02}
	dst := solana.PublicKey{0x03}
	auth := solana.PublicKey{0x04}
	ix := NewTransferChecked(src, mint, dst, auth, 12345, 6)

	data, _ := ix.Data()
	if len(data) != 10 {
		t.Fatalf("data len = %d", len(data))
	}
	if data[0] != tagTransferChecked {
		t.Errorf("tag = %d", data[0])
	}
	if amt := binary.LittleEndian.Uint64(data[1:9]); amt != 12345 {
		t.Errorf("amount = %d", amt)
	}
	if data[9] != 6 {
		t.Errorf("decimals = %d", data[9])
	}
	accs := ix.Accounts()
	if len(accs) != 4 {
		t.Fatalf("accounts len = %d", len(accs))
	}
	if accs[1].IsWritable {
		t.Error("mint should be readonly")
	}
}

func TestNewMintTo_Encoding(t *testing.T) {
	mint := solana.PublicKey{0x01}
	dst := solana.PublicKey{0x02}
	auth := solana.PublicKey{0x03}
	ix := NewMintTo(mint, dst, auth, 500)
	data, _ := ix.Data()
	if data[0] != tagMintTo {
		t.Errorf("tag = %d", data[0])
	}
	if amt := binary.LittleEndian.Uint64(data[1:9]); amt != 500 {
		t.Errorf("amount = %d", amt)
	}
}

func TestNewBurn_Encoding(t *testing.T) {
	acc := solana.PublicKey{0x01}
	mint := solana.PublicKey{0x02}
	auth := solana.PublicKey{0x03}
	ix := NewBurn(acc, mint, auth, 100)
	data, _ := ix.Data()
	if data[0] != tagBurn {
		t.Errorf("tag = %d", data[0])
	}
	accs := ix.Accounts()
	if !accs[0].IsWritable || !accs[1].IsWritable {
		t.Error("account and mint should both be writable")
	}
}

func TestNewCloseAccount_Encoding(t *testing.T) {
	acc := solana.PublicKey{0x01}
	dst := solana.PublicKey{0x02}
	auth := solana.PublicKey{0x03}
	ix := NewCloseAccount(acc, dst, auth)
	data, _ := ix.Data()
	if len(data) != 1 || data[0] != tagCloseAccount {
		t.Errorf("data = %x", data)
	}
}

func TestNewInitializeMint2_WithFreezeAuthority(t *testing.T) {
	mint := solana.PublicKey{0x01}
	mintAuth := solana.PublicKey{0x02, 0x03}
	freezeAuth := solana.PublicKey{0x04, 0x05}
	ix := NewInitializeMint2(mint, 9, mintAuth, freezeAuth)
	data, _ := ix.Data()
	expectedLen := 1 + 1 + 32 + 1 + 32
	if len(data) != expectedLen {
		t.Fatalf("data len = %d, want %d", len(data), expectedLen)
	}
	if data[0] != tagInitializeMint2 {
		t.Errorf("tag = %d", data[0])
	}
	if data[1] != 9 {
		t.Errorf("decimals = %d", data[1])
	}
	if !bytes.Equal(data[2:34], mintAuth[:]) {
		t.Error("mint authority mismatch")
	}
	if data[34] != 1 {
		t.Error("has_freeze_authority flag should be 1")
	}
	if !bytes.Equal(data[35:67], freezeAuth[:]) {
		t.Error("freeze authority mismatch")
	}
}

func TestNewInitializeMint2_NoFreezeAuthority(t *testing.T) {
	mint := solana.PublicKey{0x01}
	mintAuth := solana.PublicKey{0x02}
	ix := NewInitializeMint2(mint, 6, mintAuth, solana.PublicKey{})
	data, _ := ix.Data()
	expectedLen := 1 + 1 + 32 + 1
	if len(data) != expectedLen {
		t.Fatalf("data len = %d, want %d", len(data), expectedLen)
	}
	if data[34] != 0 {
		t.Error("has_freeze_authority flag should be 0")
	}
}

func TestNewInitializeAccount3_Encoding(t *testing.T) {
	acc := solana.PublicKey{0x01}
	mint := solana.PublicKey{0x02}
	owner := solana.PublicKey{0x03, 0x04}
	ix := NewInitializeAccount3(acc, mint, owner)
	data, _ := ix.Data()
	if len(data) != 33 {
		t.Fatalf("data len = %d", len(data))
	}
	if data[0] != tagInitializeAccount3 {
		t.Errorf("tag = %d", data[0])
	}
	if !bytes.Equal(data[1:33], owner[:]) {
		t.Error("owner mismatch")
	}
}

func TestNewInitializeMint_Encoding(t *testing.T) {
	mint := solana.PublicKey{0x01}
	mintAuth := solana.PublicKey{0x02}
	freezeAuth := solana.PublicKey{0x03}
	ix := NewInitializeMint(mint, 6, mintAuth, &freezeAuth)
	data, _ := ix.Data()
	// tag + decimals + mintAuth(32) + option(1) + freezeAuth(32) = 67
	if len(data) != 67 {
		t.Fatalf("data len = %d, want 67", len(data))
	}
	if data[0] != tagInitializeMint {
		t.Errorf("tag = %d, want %d", data[0], tagInitializeMint)
	}
	if data[1] != 6 {
		t.Errorf("decimals = %d, want 6", data[1])
	}
	if data[34] != 1 {
		t.Error("freeze option byte should be 1")
	}
	accs := ix.Accounts()
	if len(accs) != 2 {
		t.Fatalf("accounts = %d, want 2", len(accs))
	}
	if accs[1].PublicKey != solana.SysvarRentPubkey {
		t.Error("account[1] must be sysvar:rent")
	}
}

func TestNewInitializeMint_NoFreezeAuth(t *testing.T) {
	mint := solana.PublicKey{0x01}
	mintAuth := solana.PublicKey{0x02}
	ix := NewInitializeMint(mint, 9, mintAuth, nil)
	data, _ := ix.Data()
	// tag + decimals + mintAuth(32) + option(1) = 35
	if len(data) != 35 {
		t.Fatalf("data len = %d, want 35", len(data))
	}
	if data[34] != 0 {
		t.Error("freeze option byte should be 0")
	}
}

func TestNewInitializeAccount_Encoding(t *testing.T) {
	acc := solana.PublicKey{0x01}
	mint := solana.PublicKey{0x02}
	owner := solana.PublicKey{0x03}
	ix := NewInitializeAccount(acc, mint, owner)
	data, _ := ix.Data()
	if len(data) != 1 || data[0] != tagInitializeAccount {
		t.Errorf("data = %x, want [%d]", data, tagInitializeAccount)
	}
	accs := ix.Accounts()
	if len(accs) != 4 {
		t.Fatalf("accounts = %d, want 4", len(accs))
	}
	if accs[3].PublicKey != solana.SysvarRentPubkey {
		t.Error("accounts[3] must be sysvar:rent")
	}
}

func TestNewInitializeMultisig_Encoding(t *testing.T) {
	multisig := solana.PublicKey{0x01}
	s1 := solana.PublicKey{0x02}
	s2 := solana.PublicKey{0x03}
	ix := NewInitializeMultisig(multisig, 2, []solana.PublicKey{s1, s2})
	data, _ := ix.Data()
	if len(data) != 2 {
		t.Fatalf("data len = %d, want 2", len(data))
	}
	if data[0] != tagInitializeMultisig {
		t.Errorf("tag = %d, want %d", data[0], tagInitializeMultisig)
	}
	if data[1] != 2 {
		t.Errorf("m = %d, want 2", data[1])
	}
	accs := ix.Accounts()
	// multisig(w) + rent(r) + s1 + s2 = 4
	if len(accs) != 4 {
		t.Fatalf("accounts = %d, want 4", len(accs))
	}
	if accs[1].PublicKey != solana.SysvarRentPubkey {
		t.Error("accounts[1] must be sysvar:rent")
	}
}

func TestToken_IntegratesWithNewMessage(t *testing.T) {
	payer := solana.PublicKey{0x01}
	src := solana.PublicKey{0x02}
	dst := solana.PublicKey{0x03}
	mint := solana.PublicKey{0x04}

	ix := NewTransferChecked(src, mint, dst, payer, 1000, 6)
	msg, err := solana.NewMessage(payer, []solana.Instruction{ix}, solana.Hash{0x42})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := msg.Marshal(); err != nil {
		t.Errorf("Marshal: %v", err)
	}
}
