package system

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/MevYu/solana-go"
)

func TestProgramID_IsAllZero(t *testing.T) {
	if !ProgramID.IsZero() {
		t.Errorf("System ProgramID should be all zero")
	}
}

func TestNewTransfer_Encoding(t *testing.T) {
	from := solana.PublicKey{0x01}
	to := solana.PublicKey{0x02}
	ix := NewTransfer(from, to, 1_000_000_000)

	data, err := ix.Data()
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 12 {
		t.Fatalf("data len = %d, want 12", len(data))
	}
	if tag := binary.LittleEndian.Uint32(data[0:4]); tag != tagTransfer {
		t.Errorf("tag = %d, want %d", tag, tagTransfer)
	}
	if lamports := binary.LittleEndian.Uint64(data[4:12]); lamports != 1_000_000_000 {
		t.Errorf("lamports = %d", lamports)
	}

	accs := ix.Accounts()
	if len(accs) != 2 {
		t.Fatalf("accounts len = %d", len(accs))
	}
	if !accs[0].PublicKey.Equal(from) || !accs[0].IsSigner || !accs[0].IsWritable {
		t.Errorf("from meta wrong: %+v", accs[0])
	}
	if !accs[1].PublicKey.Equal(to) || accs[1].IsSigner || !accs[1].IsWritable {
		t.Errorf("to meta wrong: %+v", accs[1])
	}

	if !ix.ProgramID().Equal(ProgramID) {
		t.Errorf("program id mismatch")
	}
}

func TestNewCreateAccount_Encoding(t *testing.T) {
	from := solana.PublicKey{0x01}
	newAcc := solana.PublicKey{0x02}
	owner := solana.PublicKey{0x03, 0x04, 0x05}

	ix := NewCreateAccount(from, newAcc, 2_039_280, 165, owner)
	data, err := ix.Data()
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 52 {
		t.Fatalf("data len = %d, want 52", len(data))
	}
	if tag := binary.LittleEndian.Uint32(data[0:4]); tag != tagCreateAccount {
		t.Errorf("tag = %d", tag)
	}
	if lamports := binary.LittleEndian.Uint64(data[4:12]); lamports != 2_039_280 {
		t.Errorf("lamports = %d", lamports)
	}
	if space := binary.LittleEndian.Uint64(data[12:20]); space != 165 {
		t.Errorf("space = %d", space)
	}
	if !bytes.Equal(data[20:52], owner[:]) {
		t.Errorf("owner bytes mismatch")
	}

	accs := ix.Accounts()
	if len(accs) != 2 {
		t.Fatalf("accounts len = %d", len(accs))
	}
	if !accs[0].IsSigner || !accs[1].IsSigner {
		t.Error("both accounts should be signers")
	}
	if !accs[0].IsWritable || !accs[1].IsWritable {
		t.Error("both accounts should be writable")
	}
}

func TestNewAssign_Encoding(t *testing.T) {
	account := solana.PublicKey{0x01}
	owner := solana.PublicKey{0x99, 0xaa, 0xbb}
	ix := NewAssign(account, owner)
	data, _ := ix.Data()
	if len(data) != 36 {
		t.Fatalf("len = %d, want 36", len(data))
	}
	if tag := binary.LittleEndian.Uint32(data[0:4]); tag != tagAssign {
		t.Errorf("tag = %d", tag)
	}
	if !bytes.Equal(data[4:36], owner[:]) {
		t.Error("owner mismatch")
	}
	accs := ix.Accounts()
	if len(accs) != 1 || !accs[0].IsSigner || !accs[0].IsWritable {
		t.Errorf("meta wrong: %+v", accs)
	}
}

func TestNewAllocate_Encoding(t *testing.T) {
	account := solana.PublicKey{0x01}
	ix := NewAllocate(account, 1024)
	data, _ := ix.Data()
	if len(data) != 12 {
		t.Fatalf("len = %d", len(data))
	}
	if tag := binary.LittleEndian.Uint32(data[0:4]); tag != tagAllocate {
		t.Errorf("tag = %d", tag)
	}
	if space := binary.LittleEndian.Uint64(data[4:12]); space != 1024 {
		t.Errorf("space = %d", space)
	}
}

// TestSystem_IntegratesWithNewMessage proves the builder output
// feeds NewMessage correctly, producing a signable message.
func TestSystem_IntegratesWithNewMessage(t *testing.T) {
	payer := solana.PublicKey{0x01}
	recipient := solana.PublicKey{0x02}
	bh := solana.Hash{0x42}

	ix := NewTransfer(payer, recipient, 100_000)
	msg, err := solana.NewMessage(payer, []solana.Instruction{ix}, bh)
	if err != nil {
		t.Fatal(err)
	}
	if msg.Header.NumRequiredSignatures != 1 {
		t.Errorf("NumRequiredSignatures = %d", msg.Header.NumRequiredSignatures)
	}
	// Marshalling the message should succeed end-to-end.
	if _, err := msg.Marshal(); err != nil {
		t.Errorf("Marshal: %v", err)
	}
}

func TestNewAdvanceNonceAccount_Encoding(t *testing.T) {
	nonce := solana.PublicKey{0x01}
	auth := solana.PublicKey{0x02}
	ix := NewAdvanceNonceAccount(nonce, auth)

	data, _ := ix.Data()
	if len(data) != 4 {
		t.Fatalf("data len = %d, want 4", len(data))
	}
	if tag := binary.LittleEndian.Uint32(data[0:4]); tag != tagAdvanceNonceAccount {
		t.Errorf("tag = %d, want %d", tag, tagAdvanceNonceAccount)
	}
	accs := ix.Accounts()
	if len(accs) != 3 {
		t.Fatalf("accounts = %d, want 3", len(accs))
	}
	if accs[0].PublicKey != nonce || !accs[0].IsWritable {
		t.Errorf("accounts[0] wrong")
	}
	if accs[2].PublicKey != auth || !accs[2].IsSigner {
		t.Errorf("accounts[2] (authority) wrong")
	}
}

func TestNewWithdrawNonceAccount_Encoding(t *testing.T) {
	nonce := solana.PublicKey{0x01}
	auth := solana.PublicKey{0x02}
	to := solana.PublicKey{0x03}
	lamports := uint64(500_000_000)
	ix := NewWithdrawNonceAccount(nonce, auth, to, lamports)

	data, _ := ix.Data()
	if len(data) != 12 {
		t.Fatalf("data len = %d, want 12", len(data))
	}
	if tag := binary.LittleEndian.Uint32(data[0:4]); tag != tagWithdrawNonceAccount {
		t.Errorf("tag = %d, want %d", tag, tagWithdrawNonceAccount)
	}
	if got := binary.LittleEndian.Uint64(data[4:12]); got != lamports {
		t.Errorf("lamports = %d, want %d", got, lamports)
	}
}

func TestNewInitializeNonceAccount_Encoding(t *testing.T) {
	nonce := solana.PublicKey{0x01}
	auth := solana.PublicKey{0x02}
	ix := NewInitializeNonceAccount(nonce, auth)

	data, _ := ix.Data()
	if len(data) != 36 {
		t.Fatalf("data len = %d, want 36", len(data))
	}
	if tag := binary.LittleEndian.Uint32(data[0:4]); tag != tagInitializeNonceAccount {
		t.Errorf("tag = %d, want %d", tag, tagInitializeNonceAccount)
	}
	if !bytes.Equal(data[4:36], auth[:]) {
		t.Errorf("authority pubkey mismatch in data")
	}
}

func TestNewAuthorizeNonceAccount_Encoding(t *testing.T) {
	nonce := solana.PublicKey{0x01}
	auth := solana.PublicKey{0x02}
	newAuth := solana.PublicKey{0x03}
	ix := NewAuthorizeNonceAccount(nonce, auth, newAuth)

	data, _ := ix.Data()
	if len(data) != 36 {
		t.Fatalf("data len = %d, want 36", len(data))
	}
	if tag := binary.LittleEndian.Uint32(data[0:4]); tag != tagAuthorizeNonceAccount {
		t.Errorf("tag = %d, want %d", tag, tagAuthorizeNonceAccount)
	}
	if !bytes.Equal(data[4:36], newAuth[:]) {
		t.Errorf("new authority pubkey mismatch")
	}
}

func TestNewCreateAccountWithSeed_Encoding(t *testing.T) {
	from := solana.PublicKey{0x01}
	newAcc := solana.PublicKey{0x02}
	base := solana.PublicKey{0x03}
	owner := solana.PublicKey{0x04}
	seed := "hello"

	ix := NewCreateAccountWithSeed(from, newAcc, base, seed, 2_039_280, 165, owner)
	data, err := ix.Data()
	if err != nil {
		t.Fatal(err)
	}

	// Layout: u32(tag) + base(32) + u64(seedLen) + seed + u64(lamports) + u64(space) + owner(32)
	want := 4 + 32 + 8 + len(seed) + 8 + 8 + 32
	if len(data) != want {
		t.Fatalf("data len = %d, want %d", len(data), want)
	}
	if tag := binary.LittleEndian.Uint32(data[0:4]); tag != tagCreateAccountWithSeed {
		t.Errorf("tag = %d, want %d", tag, tagCreateAccountWithSeed)
	}
	if !bytes.Equal(data[4:36], base[:]) {
		t.Error("base pubkey mismatch")
	}
	if seedLen := binary.LittleEndian.Uint64(data[36:44]); seedLen != uint64(len(seed)) {
		t.Errorf("seedLen = %d, want %d", seedLen, len(seed))
	}
	if string(data[44:44+len(seed)]) != seed {
		t.Error("seed bytes mismatch")
	}
	off := 44 + len(seed)
	if lamports := binary.LittleEndian.Uint64(data[off : off+8]); lamports != 2_039_280 {
		t.Errorf("lamports = %d", lamports)
	}
	if space := binary.LittleEndian.Uint64(data[off+8 : off+16]); space != 165 {
		t.Errorf("space = %d", space)
	}
	if !bytes.Equal(data[off+16:off+48], owner[:]) {
		t.Error("owner mismatch")
	}

	accs := ix.Accounts()
	if len(accs) != 3 {
		t.Fatalf("accounts = %d, want 3", len(accs))
	}
	if accs[0].PublicKey != from || !accs[0].IsSigner || !accs[0].IsWritable {
		t.Error("accounts[0] (from) should be signer+writable")
	}
	if accs[1].PublicKey != newAcc || !accs[1].IsWritable {
		t.Error("accounts[1] (newAcc) should be writable")
	}
	if accs[2].PublicKey != base || !accs[2].IsSigner {
		t.Error("accounts[2] (base) should be signer")
	}
}

func TestNewCreateAccountWithSeed_BaseEqualsFrom(t *testing.T) {
	from := solana.PublicKey{0x01}
	newAcc := solana.PublicKey{0x02}
	owner := solana.PublicKey{0x03}
	ix := NewCreateAccountWithSeed(from, newAcc, from, "seed", 1000, 0, owner)
	accs := ix.Accounts()
	if len(accs) != 2 {
		t.Fatalf("accounts = %d, want 2 (base==from)", len(accs))
	}
}

func TestNewTransferWithSeed_Encoding(t *testing.T) {
	from := solana.PublicKey{0x01}
	base := solana.PublicKey{0x02}
	to := solana.PublicKey{0x03}
	programID := solana.PublicKey{0x04}
	seed := "stake"
	lamports := uint64(500_000_000)

	ix := NewTransferWithSeed(from, base, to, lamports, seed, programID)
	data, err := ix.Data()
	if err != nil {
		t.Fatal(err)
	}

	want := 4 + 8 + 8 + len(seed) + 32
	if len(data) != want {
		t.Fatalf("data len = %d, want %d", len(data), want)
	}
	if tag := binary.LittleEndian.Uint32(data[0:4]); tag != tagTransferWithSeed {
		t.Errorf("tag = %d, want %d", tag, tagTransferWithSeed)
	}
	if got := binary.LittleEndian.Uint64(data[4:12]); got != lamports {
		t.Errorf("lamports = %d, want %d", got, lamports)
	}
	if seedLen := binary.LittleEndian.Uint64(data[12:20]); seedLen != uint64(len(seed)) {
		t.Errorf("seedLen = %d", seedLen)
	}
	if string(data[20:20+len(seed)]) != seed {
		t.Error("seed bytes mismatch")
	}
	off := 20 + len(seed)
	if !bytes.Equal(data[off:off+32], programID[:]) {
		t.Error("programID mismatch")
	}

	accs := ix.Accounts()
	if len(accs) != 3 {
		t.Fatalf("accounts = %d, want 3", len(accs))
	}
	if accs[0].PublicKey != from || !accs[0].IsWritable || accs[0].IsSigner {
		t.Error("accounts[0] (from) should be writable non-signer")
	}
	if accs[1].PublicKey != base || !accs[1].IsSigner {
		t.Error("accounts[1] (base) should be signer")
	}
	if accs[2].PublicKey != to || !accs[2].IsWritable {
		t.Error("accounts[2] (to) should be writable")
	}
}
