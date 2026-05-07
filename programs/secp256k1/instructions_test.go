package secp256k1

import (
	"encoding/binary"
	"testing"
)

func TestNewVerifyEthSignature_Layout(t *testing.T) {
	var ethAddr [20]byte
	var sig [64]byte
	for i := range ethAddr {
		ethAddr[i] = byte(i + 1)
	}
	for i := range sig {
		sig[i] = byte(i + 100)
	}
	msg := []byte("hello solana")
	recoveryID := uint8(1)

	ix, err := NewVerifyEthSignature(ethAddr, sig, recoveryID, msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ix.Accounts() != nil {
		t.Error("accounts must be nil")
	}

	data, err := ix.Data()
	if err != nil {
		t.Fatalf("Data: %v", err)
	}

	// byte 0: count = 1
	if data[0] != 1 {
		t.Errorf("count = %d, want 1", data[0])
	}

	// offsets in header (bytes 1–11)
	sigOff := binary.LittleEndian.Uint16(data[1:3])
	ethOff := binary.LittleEndian.Uint16(data[4:6])
	msgOff := binary.LittleEndian.Uint16(data[7:9])
	msgSz := binary.LittleEndian.Uint16(data[9:11])
	selfIx := uint8(0xff)

	if sigOff != 12 {
		t.Errorf("sigOffset = %d, want 12", sigOff)
	}
	if data[3] != selfIx {
		t.Errorf("sigInstructionIndex = %d, want 0xff", data[3])
	}
	if ethOff != 77 {
		t.Errorf("ethAddressOffset = %d, want 77", ethOff)
	}
	if data[6] != selfIx {
		t.Errorf("ethInstructionIndex = %d, want 0xff", data[6])
	}
	if msgOff != 97 {
		t.Errorf("messageDataOffset = %d, want 97", msgOff)
	}
	if msgSz != uint16(len(msg)) {
		t.Errorf("messageDataSize = %d, want %d", msgSz, len(msg))
	}
	if data[11] != selfIx {
		t.Errorf("msgInstructionIndex = %d, want 0xff", data[11])
	}

	// verify payload positions
	if data[12:76][0] != sig[0] || data[12:76][63] != sig[63] {
		t.Error("signature bytes mismatch")
	}
	if data[76] != recoveryID {
		t.Errorf("recoveryID = %d, want %d", data[76], recoveryID)
	}
	if data[77:97][0] != ethAddr[0] || data[77:97][19] != ethAddr[19] {
		t.Error("eth address bytes mismatch")
	}
	if string(data[97:]) != string(msg) {
		t.Error("message bytes mismatch")
	}

	expectedLen := 97 + len(msg)
	if len(data) != expectedLen {
		t.Errorf("total data len = %d, want %d", len(data), expectedLen)
	}
}

func TestNewVerifyEthSignature_EmptyMessage(t *testing.T) {
	ix, err := NewVerifyEthSignature([20]byte{}, [64]byte{}, 0, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := ix.Data()
	if len(data) != 97 {
		t.Errorf("empty message: data len = %d, want 97", len(data))
	}
	if binary.LittleEndian.Uint16(data[9:11]) != 0 {
		t.Error("messageDataSize should be 0 for nil message")
	}
}

func TestNewVerifyEthSignature_MessageTooLong(t *testing.T) {
	msg := make([]byte, 65536) // len > math.MaxUint16
	_, err := NewVerifyEthSignature([20]byte{}, [64]byte{}, 0, msg)
	if err == nil {
		t.Error("expected error for message length > 65535, got nil")
	}
}

func TestNewSignatureVerifyInstruction_TooManySigs(t *testing.T) {
	sigs := make([]Secp256k1SignatureOffsets, 256) // len > math.MaxUint8
	_, err := NewSignatureVerifyInstruction(sigs)
	if err == nil {
		t.Error("expected error for signature count > 255, got nil")
	}
}
