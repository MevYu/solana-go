package token

import (
	"encoding/binary"
	"testing"

	solana "github.com/MevYu/solana-go"
)

// buildMintData constructs a minimal 82-byte SPL Token Mint layout.
// All optional fields are absent (COption tag = 0) unless their arguments are non-nil.
func buildMintData(mintAuth *solana.PublicKey, supply uint64, decimals uint8, initialized bool, freezeAuth *solana.PublicKey) []byte {
	data := make([]byte, 82)

	// Bytes 0-3: COption<Pubkey> mint_authority tag
	if mintAuth != nil {
		binary.LittleEndian.PutUint32(data[0:4], 1)
		copy(data[4:36], mintAuth[:])
	}

	// Bytes 36-43: supply
	binary.LittleEndian.PutUint64(data[36:44], supply)

	// Byte 44: decimals
	data[44] = decimals

	// Byte 45: is_initialized
	if initialized {
		data[45] = 1
	}

	// Bytes 46-49: COption<Pubkey> freeze_authority tag
	if freezeAuth != nil {
		binary.LittleEndian.PutUint32(data[46:50], 1)
		copy(data[50:82], freezeAuth[:])
	}

	return data
}

// buildAccountData constructs a minimal 165-byte SPL Token Account layout.
func buildAccountData(
	mint, owner solana.PublicKey,
	amount uint64,
	delegate *solana.PublicKey,
	state uint8,
	isNative *uint64,
	delegatedAmount uint64,
	closeAuth *solana.PublicKey,
) []byte {
	data := make([]byte, 165)

	copy(data[0:32], mint[:])
	copy(data[32:64], owner[:])
	binary.LittleEndian.PutUint64(data[64:72], amount)

	if delegate != nil {
		binary.LittleEndian.PutUint32(data[72:76], 1)
		copy(data[76:108], delegate[:])
	}

	data[108] = state

	if isNative != nil {
		binary.LittleEndian.PutUint32(data[109:113], 1)
		binary.LittleEndian.PutUint64(data[113:121], *isNative)
	}

	binary.LittleEndian.PutUint64(data[121:129], delegatedAmount)

	if closeAuth != nil {
		binary.LittleEndian.PutUint32(data[129:133], 1)
		copy(data[133:165], closeAuth[:])
	}

	return data
}

// --- DecodeMintState ---

func TestDecodeMintState_TooShort(t *testing.T) {
	_, err := DecodeMintState(make([]byte, 81))
	if err == nil {
		t.Fatal("expected error for 81-byte buffer")
	}
}

func TestDecodeMintState_Empty(t *testing.T) {
	_, err := DecodeMintState(nil)
	if err == nil {
		t.Fatal("expected error for nil buffer")
	}
}

func TestDecodeMintState_NoOptionalFields(t *testing.T) {
	data := buildMintData(nil, 1_000_000, 6, true, nil)
	m, err := DecodeMintState(data)
	if err != nil {
		t.Fatalf("DecodeMintState: %v", err)
	}
	if m.MintAuthority != nil {
		t.Error("expected nil MintAuthority")
	}
	if m.FreezeAuthority != nil {
		t.Error("expected nil FreezeAuthority")
	}
	if m.Supply != 1_000_000 {
		t.Errorf("Supply: got %d, want 1000000", m.Supply)
	}
	if m.Decimals != 6 {
		t.Errorf("Decimals: got %d, want 6", m.Decimals)
	}
	if !m.IsInitialized {
		t.Error("expected IsInitialized = true")
	}
}

func TestDecodeMintState_WithMintAuthority(t *testing.T) {
	auth := solana.PublicKey{1, 2, 3}
	data := buildMintData(&auth, 0, 9, true, nil)
	m, err := DecodeMintState(data)
	if err != nil {
		t.Fatalf("DecodeMintState: %v", err)
	}
	if m.MintAuthority == nil {
		t.Fatal("expected non-nil MintAuthority")
	}
	if *m.MintAuthority != auth {
		t.Errorf("MintAuthority: got %s, want %s", m.MintAuthority, auth)
	}
}

func TestDecodeMintState_WithFreezeAuthority(t *testing.T) {
	freeze := solana.PublicKey{9, 8, 7}
	data := buildMintData(nil, 500, 0, true, &freeze)
	m, err := DecodeMintState(data)
	if err != nil {
		t.Fatalf("DecodeMintState: %v", err)
	}
	if m.FreezeAuthority == nil {
		t.Fatal("expected non-nil FreezeAuthority")
	}
	if *m.FreezeAuthority != freeze {
		t.Errorf("FreezeAuthority: got %s, want %s", m.FreezeAuthority, freeze)
	}
}

func TestDecodeMintState_NotInitialized(t *testing.T) {
	data := buildMintData(nil, 0, 0, false, nil)
	m, err := DecodeMintState(data)
	if err != nil {
		t.Fatalf("DecodeMintState: %v", err)
	}
	if m.IsInitialized {
		t.Error("expected IsInitialized = false")
	}
}

func TestDecodeMintState_LargerBufferAllowed(t *testing.T) {
	// Buffers longer than 82 bytes (e.g. Token-2022 extension data) must not error.
	data := make([]byte, 200)
	copy(data, buildMintData(nil, 1, 6, true, nil))
	_, err := DecodeMintState(data)
	if err != nil {
		t.Fatalf("DecodeMintState on 200-byte buffer: %v", err)
	}
}

// --- DecodeAccountState ---

func TestDecodeAccountState_TooShort(t *testing.T) {
	_, err := DecodeAccountState(make([]byte, 164))
	if err == nil {
		t.Fatal("expected error for 164-byte buffer")
	}
}

func TestDecodeAccountState_NoOptionalFields(t *testing.T) {
	mint := solana.PublicKey{1}
	owner := solana.PublicKey{2}
	data := buildAccountData(mint, owner, 999, nil, 1, nil, 0, nil)
	a, err := DecodeAccountState(data)
	if err != nil {
		t.Fatalf("DecodeAccountState: %v", err)
	}
	if a.Mint != mint {
		t.Errorf("Mint: got %s, want %s", a.Mint, mint)
	}
	if a.Owner != owner {
		t.Errorf("Owner: got %s, want %s", a.Owner, owner)
	}
	if a.Amount != 999 {
		t.Errorf("Amount: got %d, want 999", a.Amount)
	}
	if a.State != 1 {
		t.Errorf("State: got %d, want 1", a.State)
	}
	if a.Delegate != nil {
		t.Error("expected nil Delegate")
	}
	if a.IsNative != nil {
		t.Error("expected nil IsNative")
	}
	if a.CloseAuthority != nil {
		t.Error("expected nil CloseAuthority")
	}
}

func TestDecodeAccountState_WithDelegate(t *testing.T) {
	mint := solana.PublicKey{1}
	owner := solana.PublicKey{2}
	delegate := solana.PublicKey{3}
	data := buildAccountData(mint, owner, 100, &delegate, 1, nil, 50, nil)
	a, err := DecodeAccountState(data)
	if err != nil {
		t.Fatalf("DecodeAccountState: %v", err)
	}
	if a.Delegate == nil {
		t.Fatal("expected non-nil Delegate")
	}
	if *a.Delegate != delegate {
		t.Errorf("Delegate: got %s, want %s", a.Delegate, delegate)
	}
	if a.DelegatedAmount != 50 {
		t.Errorf("DelegatedAmount: got %d, want 50", a.DelegatedAmount)
	}
}

func TestDecodeAccountState_WithNative(t *testing.T) {
	mint := solana.PublicKey{1}
	owner := solana.PublicKey{2}
	rent := uint64(2_039_280)
	data := buildAccountData(mint, owner, 5_000_000, nil, 1, &rent, 0, nil)
	a, err := DecodeAccountState(data)
	if err != nil {
		t.Fatalf("DecodeAccountState: %v", err)
	}
	if a.IsNative == nil {
		t.Fatal("expected non-nil IsNative")
	}
	if *a.IsNative != rent {
		t.Errorf("IsNative: got %d, want %d", *a.IsNative, rent)
	}
}

func TestDecodeAccountState_WithCloseAuthority(t *testing.T) {
	mint := solana.PublicKey{1}
	owner := solana.PublicKey{2}
	closeAuth := solana.PublicKey{7}
	data := buildAccountData(mint, owner, 0, nil, 1, nil, 0, &closeAuth)
	a, err := DecodeAccountState(data)
	if err != nil {
		t.Fatalf("DecodeAccountState: %v", err)
	}
	if a.CloseAuthority == nil {
		t.Fatal("expected non-nil CloseAuthority")
	}
	if *a.CloseAuthority != closeAuth {
		t.Errorf("CloseAuthority: got %s, want %s", a.CloseAuthority, closeAuth)
	}
}

func TestDecodeAccountState_FrozenState(t *testing.T) {
	mint := solana.PublicKey{1}
	owner := solana.PublicKey{2}
	data := buildAccountData(mint, owner, 0, nil, 2 /* frozen */, nil, 0, nil)
	a, err := DecodeAccountState(data)
	if err != nil {
		t.Fatalf("DecodeAccountState: %v", err)
	}
	if a.State != 2 {
		t.Errorf("State: got %d, want 2 (frozen)", a.State)
	}
}

func TestDecodeAccountState_AllFieldsSet(t *testing.T) {
	mint := solana.PublicKey{10}
	owner := solana.PublicKey{20}
	delegate := solana.PublicKey{30}
	rent := uint64(1_000_000)
	closeAuth := solana.PublicKey{40}
	data := buildAccountData(mint, owner, 12345, &delegate, 1, &rent, 6789, &closeAuth)
	a, err := DecodeAccountState(data)
	if err != nil {
		t.Fatalf("DecodeAccountState: %v", err)
	}
	if a.Amount != 12345 {
		t.Errorf("Amount: %d", a.Amount)
	}
	if a.DelegatedAmount != 6789 {
		t.Errorf("DelegatedAmount: %d", a.DelegatedAmount)
	}
	if a.Delegate == nil || *a.Delegate != delegate {
		t.Error("Delegate mismatch")
	}
	if a.IsNative == nil || *a.IsNative != rent {
		t.Error("IsNative mismatch")
	}
	if a.CloseAuthority == nil || *a.CloseAuthority != closeAuth {
		t.Error("CloseAuthority mismatch")
	}
}
