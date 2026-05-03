package token2022

import (
	"encoding/binary"
	"testing"

	"github.com/MevYu/solana-go"
)

func makeMintBase() []byte    { return make([]byte, 82) }
func makeAccountBase() []byte { return make([]byte, 165) }

func TestDecodeMintState_TooShort(t *testing.T) {
	if _, err := DecodeMintState(make([]byte, 81)); err == nil {
		t.Error("expected error for short data")
	}
}

func TestDecodeMintState_BaseOnly(t *testing.T) {
	data := makeMintBase()
	binary.LittleEndian.PutUint32(data[0:4], 1)
	data[4] = 0xAB
	binary.LittleEndian.PutUint64(data[36:44], 500)
	data[44] = 6
	data[45] = 1

	m, err := DecodeMintState(data)
	if err != nil {
		t.Fatal(err)
	}
	if m.MintAuthority == nil || m.MintAuthority[0] != 0xAB {
		t.Error("MintAuthority not decoded")
	}
	if m.Supply != 500 {
		t.Errorf("Supply = %d, want 500", m.Supply)
	}
	if m.Decimals != 6 {
		t.Errorf("Decimals = %d, want 6", m.Decimals)
	}
	if !m.IsInitialized {
		t.Error("IsInitialized should be true")
	}
}

func TestDecodeMintState_CloseAuthority(t *testing.T) {
	closeAuth := solana.PublicKey{0x01, 0x02, 0x03}
	buf := make([]byte, 165+1+4+32)
	buf[165] = 1 // AccountType = Mint
	binary.LittleEndian.PutUint16(buf[166:168], 3)
	binary.LittleEndian.PutUint16(buf[168:170], 32)
	copy(buf[170:202], closeAuth[:])

	m, err := DecodeMintState(buf)
	if err != nil {
		t.Fatal(err)
	}
	if m.Extensions.CloseAuthority == nil {
		t.Fatal("CloseAuthority should not be nil")
	}
	if *m.Extensions.CloseAuthority != closeAuth {
		t.Errorf("CloseAuthority = %v, want %v", *m.Extensions.CloseAuthority, closeAuth)
	}
}

func TestDecodeMintState_CloseAuthorityZero(t *testing.T) {
	buf := make([]byte, 165+1+4+32)
	buf[165] = 1
	binary.LittleEndian.PutUint16(buf[166:168], 3)
	binary.LittleEndian.PutUint16(buf[168:170], 32)

	m, err := DecodeMintState(buf)
	if err != nil {
		t.Fatal(err)
	}
	if m.Extensions.CloseAuthority != nil {
		t.Error("all-zero pubkey should decode as nil CloseAuthority")
	}
}

func TestDecodeMintState_NonTransferable(t *testing.T) {
	buf := make([]byte, 165+1+4)
	buf[165] = 1
	binary.LittleEndian.PutUint16(buf[166:168], 9)
	binary.LittleEndian.PutUint16(buf[168:170], 0)

	m, err := DecodeMintState(buf)
	if err != nil {
		t.Fatal(err)
	}
	if !m.Extensions.NonTransferable {
		t.Error("NonTransferable should be true")
	}
}

func TestDecodeMintState_DefaultAccountState(t *testing.T) {
	buf := make([]byte, 165+1+4+1)
	buf[165] = 1
	binary.LittleEndian.PutUint16(buf[166:168], 6)
	binary.LittleEndian.PutUint16(buf[168:170], 1)
	buf[170] = 2

	m, err := DecodeMintState(buf)
	if err != nil {
		t.Fatal(err)
	}
	if m.Extensions.DefaultState == nil || *m.Extensions.DefaultState != 2 {
		t.Errorf("DefaultState = %v, want 2", m.Extensions.DefaultState)
	}
}

func TestDecodeMintState_MultipleTLV(t *testing.T) {
	closeAuth := solana.PublicKey{0xCC}
	tlvSize := (4 + 32) + (4 + 1)
	buf := make([]byte, 165+1+tlvSize)
	buf[165] = 1

	off := 166
	binary.LittleEndian.PutUint16(buf[off:], 3)
	binary.LittleEndian.PutUint16(buf[off+2:], 32)
	copy(buf[off+4:off+36], closeAuth[:])
	off += 4 + 32

	binary.LittleEndian.PutUint16(buf[off:], 6)
	binary.LittleEndian.PutUint16(buf[off+2:], 1)
	buf[off+4] = 1

	m, err := DecodeMintState(buf)
	if err != nil {
		t.Fatal(err)
	}
	if m.Extensions.CloseAuthority == nil || *m.Extensions.CloseAuthority != closeAuth {
		t.Error("CloseAuthority not decoded in multi-TLV")
	}
	if m.Extensions.DefaultState == nil || *m.Extensions.DefaultState != 1 {
		t.Error("DefaultState not decoded in multi-TLV")
	}
}

func TestDecodeAccount_TooShort(t *testing.T) {
	if _, err := DecodeAccount(make([]byte, 164)); err == nil {
		t.Error("expected error for short data")
	}
}

func TestDecodeAccount_BaseOnly(t *testing.T) {
	data := makeAccountBase()
	mint := solana.PublicKey{0x01}
	owner := solana.PublicKey{0x02}
	copy(data[0:32], mint[:])
	copy(data[32:64], owner[:])
	binary.LittleEndian.PutUint64(data[64:72], 1000)
	data[108] = 1

	a, err := DecodeAccount(data)
	if err != nil {
		t.Fatal(err)
	}
	if a.Mint != mint {
		t.Errorf("Mint = %v, want %v", a.Mint, mint)
	}
	if a.Owner != owner {
		t.Errorf("Owner = %v, want %v", a.Owner, owner)
	}
	if a.Amount != 1000 {
		t.Errorf("Amount = %d, want 1000", a.Amount)
	}
	if a.State != 1 {
		t.Errorf("State = %d, want 1", a.State)
	}
}

func TestDecodeAccount_ImmutableOwner(t *testing.T) {
	buf := make([]byte, 165+1+4)
	buf[165] = 2
	binary.LittleEndian.PutUint16(buf[166:168], 7)
	binary.LittleEndian.PutUint16(buf[168:170], 0)

	a, err := DecodeAccount(buf)
	if err != nil {
		t.Fatal(err)
	}
	if !a.Extensions.ImmutableOwner {
		t.Error("ImmutableOwner should be true")
	}
}

func TestDecodeAccount_WithheldAmount(t *testing.T) {
	buf := make([]byte, 165+1+4+8)
	buf[165] = 2
	binary.LittleEndian.PutUint16(buf[166:168], 2)
	binary.LittleEndian.PutUint16(buf[168:170], 8)
	binary.LittleEndian.PutUint64(buf[170:178], 42_000)

	a, err := DecodeAccount(buf)
	if err != nil {
		t.Fatal(err)
	}
	if a.Extensions.WithheldAmount == nil || *a.Extensions.WithheldAmount != 42_000 {
		t.Errorf("WithheldAmount = %v, want 42000", a.Extensions.WithheldAmount)
	}
}

func TestDecodeAccount_MemoTransfer(t *testing.T) {
	buf := make([]byte, 165+1+4+1)
	buf[165] = 2
	binary.LittleEndian.PutUint16(buf[166:168], 8)
	binary.LittleEndian.PutUint16(buf[168:170], 1)
	buf[170] = 1

	a, err := DecodeAccount(buf)
	if err != nil {
		t.Fatal(err)
	}
	if a.Extensions.MemoTransfer == nil || !*a.Extensions.MemoTransfer {
		t.Error("MemoTransfer should be true")
	}
}

func TestDecodeAccount_CpiGuard(t *testing.T) {
	buf := make([]byte, 165+1+4+1)
	buf[165] = 2
	binary.LittleEndian.PutUint16(buf[166:168], 11)
	binary.LittleEndian.PutUint16(buf[168:170], 1)
	buf[170] = 0

	a, err := DecodeAccount(buf)
	if err != nil {
		t.Fatal(err)
	}
	if a.Extensions.CpiGuard == nil || *a.Extensions.CpiGuard {
		t.Error("CpiGuard should be false (not nil)")
	}
}

func TestParseTLV_Truncated(t *testing.T) {
	data := make([]byte, 4+10)
	binary.LittleEndian.PutUint16(data[0:2], 3)
	binary.LittleEndian.PutUint16(data[2:4], 32)
	result := parseTLV(data)
	if len(result) != 0 {
		t.Errorf("truncated TLV should be ignored, got %d entries", len(result))
	}
}

func TestParseTLV_TypeZeroStops(t *testing.T) {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint16(data[0:2], 0)
	binary.LittleEndian.PutUint16(data[2:4], 0)
	result := parseTLV(data)
	if len(result) != 0 {
		t.Errorf("type-0 entry should stop parsing, got %d entries", len(result))
	}
}
