package vote

import (
	"encoding/binary"
	"testing"

	"github.com/MevYu/solana-go"
)

func TestProgramID_Canonical(t *testing.T) {
	want := solana.MustPublicKey("Vote11111111111111111111111111111111111111h")
	if ProgramID != want {
		t.Errorf("ProgramID = %s, want %s", ProgramID, want)
	}
}

func TestNewAuthorize_Encoding(t *testing.T) {
	vote := solana.PublicKey{0x01}
	auth := solana.PublicKey{0x02}
	newAuth := solana.PublicKey{0x03}
	ix := NewAuthorize(vote, auth, newAuth, VoteAuthorizeWithdrawer)

	data, _ := ix.Data()
	if len(data) != 40 {
		t.Fatalf("data len = %d, want 40", len(data))
	}
	if tag := binary.LittleEndian.Uint32(data[0:4]); tag != tagAuthorize {
		t.Errorf("tag = %d, want %d", tag, tagAuthorize)
	}
	for i, b := range newAuth {
		if data[4+i] != b {
			t.Errorf("newAuth[%d] = %d, want %d", i, data[4+i], b)
		}
	}
	if at := binary.LittleEndian.Uint32(data[36:40]); at != uint32(VoteAuthorizeWithdrawer) {
		t.Errorf("authType = %d, want %d", at, VoteAuthorizeWithdrawer)
	}
	accs := ix.Accounts()
	if len(accs) != 3 {
		t.Fatalf("accounts = %d, want 3", len(accs))
	}
	if accs[0].PublicKey != vote || !accs[0].IsWritable {
		t.Errorf("accounts[0] (vote) wrong")
	}
	if accs[2].PublicKey != auth || !accs[2].IsSigner {
		t.Errorf("accounts[2] (authority) wrong")
	}
}

func TestNewWithdraw_Encoding(t *testing.T) {
	vote := solana.PublicKey{0x01}
	wauth := solana.PublicKey{0x02}
	recip := solana.PublicKey{0x03}
	lamports := uint64(1_000_000_000)
	ix := NewWithdraw(vote, wauth, recip, lamports)

	data, _ := ix.Data()
	if len(data) != 12 {
		t.Fatalf("data len = %d, want 12", len(data))
	}
	if tag := binary.LittleEndian.Uint32(data[0:4]); tag != tagWithdraw {
		t.Errorf("tag = %d, want %d", tag, tagWithdraw)
	}
	if got := binary.LittleEndian.Uint64(data[4:12]); got != lamports {
		t.Errorf("lamports = %d, want %d", got, lamports)
	}
}

func TestNewUpdateCommission_Encoding(t *testing.T) {
	vote := solana.PublicKey{0x01}
	auth := solana.PublicKey{0x02}
	ix := NewUpdateCommission(vote, auth, 10)

	data, _ := ix.Data()
	if len(data) != 5 {
		t.Fatalf("data len = %d, want 5", len(data))
	}
	if tag := binary.LittleEndian.Uint32(data[0:4]); tag != tagUpdateCommission {
		t.Errorf("tag = %d, want %d", tag, tagUpdateCommission)
	}
	if data[4] != 10 {
		t.Errorf("commission = %d, want 10", data[4])
	}
}

func TestNewInitializeAccount_Encoding(t *testing.T) {
	vote := solana.PublicKey{0x01}
	node := solana.PublicKey{0x02}
	voter := solana.PublicKey{0x03}
	withdrawer := solana.PublicKey{0x04}
	ix := NewInitializeAccount(vote, node, voter, withdrawer, 10)

	data, _ := ix.Data()
	// [u32(0)] [node 32B] [voter 32B] [withdrawer 32B] [commission 1B] = 101
	if len(data) != 101 {
		t.Fatalf("data len = %d, want 101", len(data))
	}
	if tag := binary.LittleEndian.Uint32(data[0:4]); tag != tagInitializeAccount {
		t.Errorf("tag = %d, want %d", tag, tagInitializeAccount)
	}
	for i, b := range node {
		if data[4+i] != b {
			t.Errorf("nodePubkey[%d] = %d, want %d", i, data[4+i], b)
		}
	}
	if data[100] != 10 {
		t.Errorf("commission = %d, want 10", data[100])
	}
	accs := ix.Accounts()
	if len(accs) != 4 {
		t.Fatalf("accounts = %d, want 4", len(accs))
	}
	if accs[1].PublicKey != solana.SysvarRentPubkey {
		t.Error("accounts[1] must be sysvar:rent")
	}
	if accs[2].PublicKey != solana.SysvarClockPubkey {
		t.Error("accounts[2] must be sysvar:clock")
	}
	if !accs[3].IsSigner {
		t.Error("nodePubkey must be signer")
	}
}

func TestNewAuthorizeChecked_Encoding(t *testing.T) {
	vote := solana.PublicKey{0x01}
	auth := solana.PublicKey{0x02}
	newAuth := solana.PublicKey{0x03}
	ix := NewAuthorizeChecked(vote, auth, newAuth, VoteAuthorizeVoter)

	data, _ := ix.Data()
	if len(data) != 8 {
		t.Fatalf("data len = %d, want 8", len(data))
	}
	if tag := binary.LittleEndian.Uint32(data[0:4]); tag != tagAuthorizeChecked {
		t.Errorf("tag = %d, want %d", tag, tagAuthorizeChecked)
	}
	if at := binary.LittleEndian.Uint32(data[4:8]); at != uint32(VoteAuthorizeVoter) {
		t.Errorf("authType = %d, want %d", at, VoteAuthorizeVoter)
	}
	accs := ix.Accounts()
	if len(accs) != 4 {
		t.Fatalf("accounts = %d, want 4", len(accs))
	}
	if !accs[2].IsSigner || !accs[3].IsSigner {
		t.Error("both authority and newAuthority must sign")
	}
}

func TestNewUpdateValidatorIdentity_Encoding(t *testing.T) {
	vote := solana.PublicKey{0x01}
	newIdent := solana.PublicKey{0x02}
	wauth := solana.PublicKey{0x03}
	ix := NewUpdateValidatorIdentity(vote, newIdent, wauth)

	data, _ := ix.Data()
	if len(data) != 4 {
		t.Fatalf("data len = %d, want 4", len(data))
	}
	accs := ix.Accounts()
	if len(accs) != 3 {
		t.Fatalf("accounts = %d, want 3", len(accs))
	}
	if !accs[1].IsSigner {
		t.Errorf("newIdentity must be signer")
	}
}
