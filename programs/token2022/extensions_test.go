package token2022

import (
	"encoding/binary"
	"testing"

	"github.com/MevYu/solana-go"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

func mustData(t *testing.T, ix solana.Instruction) []byte {
	t.Helper()
	d, err := ix.Data()
	if err != nil {
		t.Fatalf("Data: %v", err)
	}
	return d
}

var (
	mintKey  = solana.PublicKey{0x01}
	authKey  = solana.PublicKey{0x02}
	destKey  = solana.PublicKey{0x03}
	ownerKey = solana.PublicKey{0x04}
)

// ─── D1: mint close authority ─────────────────────────────────────────────────

func TestNewInitializeMintCloseAuthority_Some(t *testing.T) {
	k := authKey
	ix := NewInitializeMintCloseAuthority(mintKey, &k)
	data := mustData(t, ix)
	if data[0] != 25 {
		t.Errorf("tag = %d, want 25", data[0])
	}
	// OptionalNonZeroPubkey: 32 bytes raw, no option tag.
	// 1 (instruction tag) + 32 (pubkey) = 33.
	if len(data) != 33 {
		t.Errorf("len = %d, want 33", len(data))
	}
	for i, b := range k {
		if data[1+i] != b {
			t.Errorf("close_authority[%d] = %d, want %d", i, data[1+i], b)
		}
	}
	if len(ix.Accounts()) != 1 || !ix.Accounts()[0].IsWritable {
		t.Error("single writable mint account expected")
	}
}

func TestNewInitializeMintCloseAuthority_None(t *testing.T) {
	data := mustData(t, NewInitializeMintCloseAuthority(mintKey, nil))
	// 1 (tag) + 32 (zero pubkey == None) = 33.
	if len(data) != 33 || data[0] != 25 {
		t.Errorf("data len = %d, want 33; tag = %d", len(data), data[0])
	}
	for i := 1; i < 33; i++ {
		if data[i] != 0 {
			t.Errorf("None should be 32 zero bytes; data[%d] = %d", i, data[i])
		}
	}
}

// ─── D2: non-transferable ─────────────────────────────────────────────────────

func TestNewInitializeNonTransferableMint(t *testing.T) {
	data := mustData(t, NewInitializeNonTransferableMint(mintKey))
	if len(data) != 1 || data[0] != 32 {
		t.Errorf("data = %x, want [32]", data)
	}
}

// ─── D3: permanent delegate ───────────────────────────────────────────────────

func TestNewInitializePermanentDelegate(t *testing.T) {
	data := mustData(t, NewInitializePermanentDelegate(mintKey, authKey))
	if data[0] != 35 {
		t.Errorf("tag = %d, want 35", data[0])
	}
	if len(data) != 33 {
		t.Errorf("len = %d, want 33", len(data))
	}
}

// ─── D4: default account state ───────────────────────────────────────────────

func TestNewInitializeDefaultAccountState(t *testing.T) {
	data := mustData(t, NewInitializeDefaultAccountState(mintKey, AccountStateFrozen))
	if len(data) != 3 || data[0] != 28 || data[1] != 0 || data[2] != byte(AccountStateFrozen) {
		t.Errorf("data = %x", data)
	}
	if len(NewInitializeDefaultAccountState(mintKey, AccountStateFrozen).Accounts()) != 1 {
		t.Error("want 1 account")
	}
}

func TestNewUpdateDefaultAccountState(t *testing.T) {
	ix := NewUpdateDefaultAccountState(mintKey, authKey, AccountStateInitialized)
	data := mustData(t, ix)
	if len(data) != 3 || data[0] != 28 || data[1] != 1 || data[2] != byte(AccountStateInitialized) {
		t.Errorf("data = %x", data)
	}
	accs := ix.Accounts()
	if len(accs) != 2 || !accs[1].IsSigner {
		t.Error("want 2 accounts, authority must sign")
	}
}

// ─── D5: transfer fee ─────────────────────────────────────────────────────────

func TestNewInitializeTransferFeeConfig(t *testing.T) {
	tfAuth := authKey
	wdAuth := destKey
	ix := NewInitializeTransferFeeConfig(mintKey, &tfAuth, &wdAuth, 500, 1_000_000)
	data := mustData(t, ix)
	if data[0] != 26 || data[1] != 0 {
		t.Errorf("tags = [%d %d], want [26 0]", data[0], data[1])
	}
	// 2 (tags) + 32 (OptionalNonZeroPubkey) + 32 (OptionalNonZeroPubkey) + 2 (u16) + 8 (u64) = 76
	if len(data) != 76 {
		t.Errorf("len = %d, want 76", len(data))
	}
	bpOffset := 2 + 32 + 32
	if bp := binary.LittleEndian.Uint16(data[bpOffset : bpOffset+2]); bp != 500 {
		t.Errorf("feeBasisPoints = %d, want 500", bp)
	}
	if mf := binary.LittleEndian.Uint64(data[bpOffset+2 : bpOffset+10]); mf != 1_000_000 {
		t.Errorf("maximumFee = %d, want 1000000", mf)
	}
}

func TestNewSetTransferFee(t *testing.T) {
	data := mustData(t, NewSetTransferFee(mintKey, authKey, 100, 9999))
	if data[0] != 26 || data[1] != 5 {
		t.Errorf("tags = [%d %d]", data[0], data[1])
	}
	if len(data) != 12 {
		t.Errorf("len = %d, want 12", len(data))
	}
}

func TestNewWithdrawWithheldTokensFromMint(t *testing.T) {
	ix := NewWithdrawWithheldTokensFromMint(mintKey, destKey, authKey)
	data := mustData(t, ix)
	if len(data) != 2 || data[0] != 26 || data[1] != 2 {
		t.Errorf("data = %x", data)
	}
	if len(ix.Accounts()) != 3 {
		t.Error("want 3 accounts")
	}
}

func TestNewWithdrawWithheldTokensFromAccounts(t *testing.T) {
	sources := []solana.PublicKey{{0xAA}, {0xBB}}
	ix := NewWithdrawWithheldTokensFromAccounts(mintKey, destKey, authKey, sources)
	data := mustData(t, ix)
	if data[0] != 26 || data[1] != 3 || data[2] != 2 {
		t.Errorf("data = %x", data)
	}
	if len(ix.Accounts()) != 5 {
		t.Errorf("accounts = %d, want 5", len(ix.Accounts()))
	}
}

func TestNewHarvestWithheldTokensToMint(t *testing.T) {
	sources := []solana.PublicKey{{0xCC}}
	ix := NewHarvestWithheldTokensToMint(mintKey, sources)
	data := mustData(t, ix)
	if data[0] != 26 || data[1] != 4 {
		t.Errorf("data = %x", data)
	}
	if len(ix.Accounts()) != 2 {
		t.Error("want mint + 1 source")
	}
}

// ─── D6: interest bearing ─────────────────────────────────────────────────────

func TestNewInitializeInterestBearingMint(t *testing.T) {
	rAuth := authKey
	ix := NewInitializeInterestBearingMint(mintKey, &rAuth, 300)
	data := mustData(t, ix)
	if data[0] != 33 || data[1] != 0 {
		t.Errorf("tags = [%d %d]", data[0], data[1])
	}
	// 2 (tags) + 32 (OptionalNonZeroPubkey) + 2 (i16) = 36
	if len(data) != 36 {
		t.Errorf("len = %d, want 36", len(data))
	}
	if rate := int16(binary.LittleEndian.Uint16(data[34:36])); rate != 300 {
		t.Errorf("rate = %d, want 300", rate)
	}
}

func TestNewUpdateInterestBearingMintRate(t *testing.T) {
	data := mustData(t, NewUpdateInterestBearingMintRate(mintKey, authKey, -50))
	if data[0] != 33 || data[1] != 1 {
		t.Errorf("tags = [%d %d]", data[0], data[1])
	}
	if rate := int16(binary.LittleEndian.Uint16(data[2:4])); rate != -50 {
		t.Errorf("rate = %d, want -50", rate)
	}
}

// ─── D7: metadata pointer ─────────────────────────────────────────────────────

func TestNewInitializeMetadataPointer(t *testing.T) {
	a, m := authKey, destKey
	data := mustData(t, NewInitializeMetadataPointer(mintKey, &a, &m))
	if data[0] != 39 || data[1] != 0 {
		t.Errorf("tags = [%d %d]", data[0], data[1])
	}
	// 2 (tags) + 32 (OptionalNonZeroPubkey) + 32 (OptionalNonZeroPubkey) = 66
	if len(data) != 66 {
		t.Errorf("len = %d, want 66", len(data))
	}
}

func TestNewUpdateMetadataPointer(t *testing.T) {
	m := destKey
	data := mustData(t, NewUpdateMetadataPointer(mintKey, authKey, &m))
	if data[0] != 39 || data[1] != 1 {
		t.Errorf("tags = [%d %d]", data[0], data[1])
	}
}

// ─── D8: transfer hook ────────────────────────────────────────────────────────

func TestNewInitializeTransferHook(t *testing.T) {
	a, p := authKey, destKey
	data := mustData(t, NewInitializeTransferHook(mintKey, &a, &p))
	if data[0] != 36 || data[1] != 0 {
		t.Errorf("tags = [%d %d]", data[0], data[1])
	}
	// 2 (tags) + 32 (OptionalNonZeroPubkey) + 32 (OptionalNonZeroPubkey) = 66
	if len(data) != 66 {
		t.Errorf("len = %d, want 66", len(data))
	}
}

func TestNewUpdateTransferHook(t *testing.T) {
	p := destKey
	data := mustData(t, NewUpdateTransferHook(mintKey, authKey, &p))
	if data[0] != 36 || data[1] != 1 {
		t.Errorf("tags = [%d %d]", data[0], data[1])
	}
}

// ─── D9: account-level extensions ────────────────────────────────────────────

func TestNewEnableRequiredMemoTransfers(t *testing.T) {
	data := mustData(t, NewEnableRequiredMemoTransfers(ownerKey, authKey))
	if len(data) != 2 || data[0] != 30 || data[1] != 0 {
		t.Errorf("data = %x", data)
	}
}

func TestNewDisableRequiredMemoTransfers(t *testing.T) {
	data := mustData(t, NewDisableRequiredMemoTransfers(ownerKey, authKey))
	if len(data) != 2 || data[0] != 30 || data[1] != 1 {
		t.Errorf("data = %x", data)
	}
}

func TestNewEnableCpiGuard(t *testing.T) {
	data := mustData(t, NewEnableCpiGuard(ownerKey, authKey))
	if len(data) != 2 || data[0] != 34 || data[1] != 0 {
		t.Errorf("data = %x", data)
	}
}

func TestNewDisableCpiGuard(t *testing.T) {
	data := mustData(t, NewDisableCpiGuard(ownerKey, authKey))
	if len(data) != 2 || data[0] != 34 || data[1] != 1 {
		t.Errorf("data = %x", data)
	}
}
