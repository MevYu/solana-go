package stake

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/MevYu/solana-go"
)

var (
	stakeAcc   = solana.PublicKey{0x01}
	voteAcc    = solana.PublicKey{0x02}
	authKey    = solana.PublicKey{0x03}
	newAuthKey = solana.PublicKey{0x04}
	destKey    = solana.PublicKey{0x05}
)

func mustData(t *testing.T, ix solana.Instruction) []byte {
	t.Helper()
	d, err := ix.Data()
	if err != nil {
		t.Fatalf("Data(): %v", err)
	}
	return d
}

// ─── Initialize ───────────────────────────────────────────────────────────────

func TestInitialize_Encoding(t *testing.T) {
	auth := Authorized{Staker: authKey, Withdrawer: newAuthKey}
	custodian := solana.PublicKey{0xFF}
	lk := Lockup{UnixTimestamp: 1_700_000_000, Epoch: 42, Custodian: custodian}
	ix := Initialize(stakeAcc, auth, lk)

	data := mustData(t, ix)
	if len(data) != 116 {
		t.Fatalf("data len = %d, want 116", len(data))
	}
	if tag := binary.LittleEndian.Uint32(data[0:4]); tag != tagInitialize {
		t.Errorf("tag = %d, want %d", tag, tagInitialize)
	}
	if !bytes.Equal(data[4:36], authKey[:]) {
		t.Error("Staker pubkey mismatch")
	}
	if !bytes.Equal(data[36:68], newAuthKey[:]) {
		t.Error("Withdrawer pubkey mismatch")
	}
	if ts := int64(binary.LittleEndian.Uint64(data[68:76])); ts != 1_700_000_000 {
		t.Errorf("UnixTimestamp = %d", ts)
	}
	if ep := binary.LittleEndian.Uint64(data[76:84]); ep != 42 {
		t.Errorf("Epoch = %d", ep)
	}
	if !bytes.Equal(data[84:116], custodian[:]) {
		t.Error("Custodian pubkey mismatch")
	}

	accs := ix.Accounts()
	if len(accs) != 2 {
		t.Fatalf("accounts = %d, want 2", len(accs))
	}
	if accs[0].PublicKey != stakeAcc || !accs[0].IsWritable || accs[0].IsSigner {
		t.Error("accounts[0] (stake) should be writable non-signer")
	}
	if accs[1].PublicKey != sysvarRent {
		t.Error("accounts[1] should be sysvarRent")
	}
}

// ─── Delegate ─────────────────────────────────────────────────────────────────

func TestDelegate_Encoding(t *testing.T) {
	ix := Delegate(stakeAcc, voteAcc, authKey)
	data := mustData(t, ix)
	if len(data) != 4 {
		t.Fatalf("data len = %d, want 4", len(data))
	}
	if tag := binary.LittleEndian.Uint32(data[0:4]); tag != tagDelegate {
		t.Errorf("tag = %d, want %d", tag, tagDelegate)
	}
	accs := ix.Accounts()
	if len(accs) != 6 {
		t.Fatalf("accounts = %d, want 6", len(accs))
	}
	if accs[0].PublicKey != stakeAcc || !accs[0].IsWritable {
		t.Error("accounts[0] should be writable stake account")
	}
	if accs[1].PublicKey != voteAcc {
		t.Error("accounts[1] should be vote account")
	}
	if accs[5].PublicKey != authKey || !accs[5].IsSigner {
		t.Error("accounts[5] should be signing staker")
	}
}

// ─── Deactivate ───────────────────────────────────────────────────────────────

func TestDeactivate_Encoding(t *testing.T) {
	ix := Deactivate(stakeAcc, authKey)
	data := mustData(t, ix)
	if tag := binary.LittleEndian.Uint32(data[0:4]); tag != tagDeactivate {
		t.Errorf("tag = %d, want %d", tag, tagDeactivate)
	}
	accs := ix.Accounts()
	if len(accs) != 3 {
		t.Fatalf("accounts = %d, want 3", len(accs))
	}
	if accs[2].PublicKey != authKey || !accs[2].IsSigner {
		t.Error("accounts[2] should be signing staker")
	}
}

// ─── Withdraw ─────────────────────────────────────────────────────────────────

func TestWithdraw_NoCustodian(t *testing.T) {
	ix := Withdraw(stakeAcc, destKey, authKey, 1_000_000_000, nil)
	data := mustData(t, ix)
	if len(data) != 12 {
		t.Fatalf("data len = %d, want 12", len(data))
	}
	if tag := binary.LittleEndian.Uint32(data[0:4]); tag != tagWithdraw {
		t.Errorf("tag = %d, want %d", tag, tagWithdraw)
	}
	if lamports := binary.LittleEndian.Uint64(data[4:12]); lamports != 1_000_000_000 {
		t.Errorf("lamports = %d", lamports)
	}
	if len(ix.Accounts()) != 5 {
		t.Fatalf("accounts = %d, want 5 (no custodian)", len(ix.Accounts()))
	}
}

func TestWithdraw_WithCustodian(t *testing.T) {
	custodian := solana.PublicKey{0xAB}
	ix := Withdraw(stakeAcc, destKey, authKey, 500, &custodian)
	if len(ix.Accounts()) != 6 {
		t.Fatalf("accounts = %d, want 6 (with custodian)", len(ix.Accounts()))
	}
	if ix.Accounts()[5].PublicKey != custodian {
		t.Error("accounts[5] should be custodian")
	}
}

// ─── Split ────────────────────────────────────────────────────────────────────

func TestSplit_Encoding(t *testing.T) {
	splitStake := solana.PublicKey{0x06}
	ix := Split(stakeAcc, splitStake, authKey, 2_000_000_000)
	data := mustData(t, ix)
	if len(data) != 12 {
		t.Fatalf("data len = %d, want 12", len(data))
	}
	if tag := binary.LittleEndian.Uint32(data[0:4]); tag != tagSplit {
		t.Errorf("tag = %d, want %d", tag, tagSplit)
	}
	if lamports := binary.LittleEndian.Uint64(data[4:12]); lamports != 2_000_000_000 {
		t.Errorf("lamports = %d", lamports)
	}
}

// ─── Merge ────────────────────────────────────────────────────────────────────

func TestMerge_Encoding(t *testing.T) {
	src := solana.PublicKey{0x07}
	ix := Merge(stakeAcc, src, authKey)
	data := mustData(t, ix)
	if tag := binary.LittleEndian.Uint32(data[0:4]); tag != tagMerge {
		t.Errorf("tag = %d, want %d", tag, tagMerge)
	}
	accs := ix.Accounts()
	if len(accs) != 5 {
		t.Fatalf("accounts = %d, want 5", len(accs))
	}
	if accs[4].PublicKey != authKey || !accs[4].IsSigner {
		t.Error("accounts[4] should be signing staker")
	}
}

// ─── Authorize ────────────────────────────────────────────────────────────────

func TestAuthorize_Encoding_NoAuth(t *testing.T) {
	ix := Authorize(stakeAcc, authKey, newAuthKey, AuthorizeStaker, nil)
	data := mustData(t, ix)
	// data: u32(tag) + Pubkey(32) + u32(authType) = 40B
	if len(data) != 40 {
		t.Fatalf("data len = %d, want 40", len(data))
	}
	if tag := binary.LittleEndian.Uint32(data[0:4]); tag != tagAuthorize {
		t.Errorf("tag = %d, want %d", tag, tagAuthorize)
	}
	if !bytes.Equal(data[4:36], newAuthKey[:]) {
		t.Error("newAuthority pubkey mismatch")
	}
	if at := binary.LittleEndian.Uint32(data[36:40]); at != uint32(AuthorizeStaker) {
		t.Errorf("authType = %d, want %d", at, AuthorizeStaker)
	}
	accs := ix.Accounts()
	if len(accs) != 3 {
		t.Fatalf("accounts = %d, want 3", len(accs))
	}
	if accs[0].PublicKey != stakeAcc || !accs[0].IsWritable {
		t.Error("accounts[0] should be writable stake account")
	}
	if accs[1].PublicKey != sysvarClock {
		t.Error("accounts[1] should be sysvarClock")
	}
	if accs[2].PublicKey != authKey || !accs[2].IsSigner {
		t.Error("accounts[2] should be signing authority")
	}
}

func TestAuthorize_Encoding_WithCustodian(t *testing.T) {
	custodian := solana.PublicKey{0xCC}
	ix := Authorize(stakeAcc, authKey, newAuthKey, AuthorizeWithdrawer, &custodian)
	accs := ix.Accounts()
	if len(accs) != 4 {
		t.Fatalf("accounts = %d, want 4 (with custodian)", len(accs))
	}
	if accs[3].PublicKey != custodian || !accs[3].IsSigner {
		t.Error("accounts[3] should be signing custodian")
	}
}

// ─── AuthorizeChecked ─────────────────────────────────────────────────────────

func TestAuthorizeChecked_Encoding_NoAuth(t *testing.T) {
	ix := AuthorizeChecked(stakeAcc, authKey, newAuthKey, AuthorizeStaker, nil)
	data := mustData(t, ix)
	if len(data) != 8 {
		t.Fatalf("data len = %d, want 8", len(data))
	}
	if tag := binary.LittleEndian.Uint32(data[0:4]); tag != tagAuthorizeChecked {
		t.Errorf("tag = %d, want %d", tag, tagAuthorizeChecked)
	}
	accs := ix.Accounts()
	if len(accs) != 4 {
		t.Fatalf("accounts = %d, want 4", len(accs))
	}
	if accs[2].PublicKey != authKey || !accs[2].IsSigner {
		t.Error("accounts[2] should be signing current authority")
	}
	if accs[3].PublicKey != newAuthKey || !accs[3].IsSigner {
		t.Error("accounts[3] should be signing new authority")
	}
}

func TestAuthorizeChecked_WithCustodian(t *testing.T) {
	custodian := solana.PublicKey{0xDD}
	ix := AuthorizeChecked(stakeAcc, authKey, newAuthKey, AuthorizeWithdrawer, &custodian)
	if len(ix.Accounts()) != 5 {
		t.Fatalf("accounts = %d, want 5", len(ix.Accounts()))
	}
}

// ─── InitializeChecked ────────────────────────────────────────────────────────

func TestInitializeChecked_Encoding(t *testing.T) {
	ix := InitializeChecked(stakeAcc, authKey, newAuthKey)
	data := mustData(t, ix)
	if len(data) != 4 {
		t.Fatalf("data len = %d, want 4", len(data))
	}
	if tag := binary.LittleEndian.Uint32(data[0:4]); tag != tagInitializeChecked {
		t.Errorf("tag = %d, want %d", tag, tagInitializeChecked)
	}
	accs := ix.Accounts()
	if len(accs) != 4 {
		t.Fatalf("accounts = %d, want 4", len(accs))
	}
	if accs[0].PublicKey != stakeAcc || !accs[0].IsWritable {
		t.Error("accounts[0] should be writable stake account")
	}
	if accs[1].PublicKey != sysvarRent {
		t.Error("accounts[1] should be sysvarRent")
	}
	if accs[2].PublicKey != authKey {
		t.Error("accounts[2] should be staker")
	}
	if accs[3].PublicKey != newAuthKey || !accs[3].IsSigner {
		t.Error("accounts[3] should be signing withdrawer")
	}
}

// ─── SetLockup ────────────────────────────────────────────────────────────────

// TestSetLockup_AllNil verifies all-None encoding: tag(4) + None(1) + None(1) + None(1) = 7B.
func TestSetLockup_AllNil(t *testing.T) {
	ix := SetLockup(stakeAcc, authKey, nil, nil, nil)
	data := mustData(t, ix)
	// [u32 tag=6] [u8 0] [u8 0] [u8 0]
	if len(data) != 7 {
		t.Fatalf("data len = %d, want 7", len(data))
	}
	if tag := binary.LittleEndian.Uint32(data[0:4]); tag != tagSetLockup {
		t.Errorf("tag = %d, want %d", tag, tagSetLockup)
	}
	if data[4] != 0 {
		t.Errorf("Option<timestamp> discriminant = %d, want 0 (None)", data[4])
	}
	if data[5] != 0 {
		t.Errorf("Option<epoch> discriminant = %d, want 0 (None)", data[5])
	}
	if data[6] != 0 {
		t.Errorf("Option<custodian> discriminant = %d, want 0 (None)", data[6])
	}
}

// TestSetLockup_TimestampOnly verifies 1-byte Some discriminant for Option<i64>.
// Layout: tag(4) + Some(1) + i64(8) + None(1) + None(1) = 15B.
func TestSetLockup_TimestampOnly(t *testing.T) {
	ts := int64(1_700_000_000)
	ix := SetLockup(stakeAcc, authKey, &ts, nil, nil)
	data := mustData(t, ix)
	if len(data) != 15 {
		t.Fatalf("data len = %d, want 15", len(data))
	}
	// byte 4: Some discriminant for timestamp (must be 1, not 0 or 4-byte u32)
	if data[4] != 1 {
		t.Errorf("Option<timestamp> Some discriminant = %d, want 1 (1-byte u8)", data[4])
	}
	if got := int64(binary.LittleEndian.Uint64(data[5:13])); got != ts {
		t.Errorf("timestamp = %d, want %d", got, ts)
	}
	if data[13] != 0 {
		t.Errorf("Option<epoch> discriminant = %d, want 0 (None)", data[13])
	}
	if data[14] != 0 {
		t.Errorf("Option<custodian> discriminant = %d, want 0 (None)", data[14])
	}
}

// TestSetLockup_EpochOnly verifies 1-byte Some discriminant for Option<u64>.
// Layout: tag(4) + None(1) + Some(1) + u64(8) + None(1) = 15B.
func TestSetLockup_EpochOnly(t *testing.T) {
	ep := uint64(999)
	ix := SetLockup(stakeAcc, authKey, nil, &ep, nil)
	data := mustData(t, ix)
	if len(data) != 15 {
		t.Fatalf("data len = %d, want 15", len(data))
	}
	if data[4] != 0 {
		t.Errorf("Option<timestamp> discriminant = %d, want 0 (None)", data[4])
	}
	if data[5] != 1 {
		t.Errorf("Option<epoch> Some discriminant = %d, want 1 (1-byte u8)", data[5])
	}
	if got := binary.LittleEndian.Uint64(data[6:14]); got != ep {
		t.Errorf("epoch = %d, want %d", got, ep)
	}
	if data[14] != 0 {
		t.Errorf("Option<custodian> discriminant = %d, want 0 (None)", data[14])
	}
}

// TestSetLockup_AllSome verifies full encoding with all fields set.
// Layout: tag(4) + Some(1)+i64(8) + Some(1)+u64(8) + Some(1)+pubkey(32) = 55B.
func TestSetLockup_AllSome(t *testing.T) {
	ts := int64(100)
	ep := uint64(200)
	custodian := solana.PublicKey{0xAB}
	ix := SetLockup(stakeAcc, authKey, &ts, &ep, &custodian)
	data := mustData(t, ix)
	if len(data) != 55 {
		t.Fatalf("data len = %d, want 55", len(data))
	}
	// Verify custodian
	if data[22] != 1 {
		t.Errorf("Option<custodian> discriminant = %d, want 1", data[22])
	}
	if !bytes.Equal(data[23:55], custodian[:]) {
		t.Error("custodian pubkey mismatch")
	}
}

// TestSetLockup_Accounts verifies the account list.
func TestSetLockup_Accounts(t *testing.T) {
	ix := SetLockup(stakeAcc, authKey, nil, nil, nil)
	accs := ix.Accounts()
	if len(accs) != 2 {
		t.Fatalf("accounts = %d, want 2", len(accs))
	}
	if accs[0].PublicKey != stakeAcc || !accs[0].IsWritable {
		t.Error("accounts[0] should be writable stake account")
	}
	if accs[1].PublicKey != authKey || !accs[1].IsSigner {
		t.Error("accounts[1] should be signing authority")
	}
}
