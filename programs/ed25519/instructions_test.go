package ed25519

import (
	stded25519 "crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"testing"
)

func TestNewVerifySignature_Layout(t *testing.T) {
	var pubkey [32]byte
	var sig [64]byte
	for i := range pubkey {
		pubkey[i] = byte(i + 1)
	}
	for i := range sig {
		sig[i] = byte(i + 100)
	}
	msg := []byte("hello solana")

	ix, err := NewVerifySignature(pubkey, sig, msg)
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
	// byte 1: padding = 0
	if data[1] != 0 {
		t.Errorf("padding = %d, want 0", data[1])
	}

	// offsets struct (bytes 2–15, seven u16 LE).
	sigOff := binary.LittleEndian.Uint16(data[2:4])
	sigIxIdx := binary.LittleEndian.Uint16(data[4:6])
	pubOff := binary.LittleEndian.Uint16(data[6:8])
	pubIxIdx := binary.LittleEndian.Uint16(data[8:10])
	msgOff := binary.LittleEndian.Uint16(data[10:12])
	msgSz := binary.LittleEndian.Uint16(data[12:14])
	msgIxIdx := binary.LittleEndian.Uint16(data[14:16])
	const selfIx = uint16(0xFFFF)

	if sigOff != 48 {
		t.Errorf("sigOffset = %d, want 48", sigOff)
	}
	if sigIxIdx != selfIx {
		t.Errorf("sigInstructionIndex = %#x, want 0xFFFF", sigIxIdx)
	}
	if pubOff != 16 {
		t.Errorf("pubkeyOffset = %d, want 16", pubOff)
	}
	if pubIxIdx != selfIx {
		t.Errorf("pubkeyInstructionIndex = %#x, want 0xFFFF", pubIxIdx)
	}
	if msgOff != 112 {
		t.Errorf("messageDataOffset = %d, want 112", msgOff)
	}
	if msgSz != uint16(len(msg)) {
		t.Errorf("messageDataSize = %d, want %d", msgSz, len(msg))
	}
	if msgIxIdx != selfIx {
		t.Errorf("msgInstructionIndex = %#x, want 0xFFFF", msgIxIdx)
	}

	// Payload positions.
	if data[16] != pubkey[0] || data[16+31] != pubkey[31] {
		t.Error("pubkey bytes mismatch")
	}
	if data[48] != sig[0] || data[48+63] != sig[63] {
		t.Error("signature bytes mismatch")
	}
	if string(data[112:]) != string(msg) {
		t.Error("message bytes mismatch")
	}

	expectedLen := 112 + len(msg)
	if len(data) != expectedLen {
		t.Errorf("total data len = %d, want %d", len(data), expectedLen)
	}
}

func TestNewVerifySignature_EmptyMessage(t *testing.T) {
	ix, err := NewVerifySignature([32]byte{}, [64]byte{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := ix.Data()
	if len(data) != 112 {
		t.Errorf("empty message: data len = %d, want 112", len(data))
	}
	if binary.LittleEndian.Uint16(data[12:14]) != 0 {
		t.Error("messageDataSize should be 0 for nil message")
	}
}

func TestNewVerifySignature_MessageTooLong(t *testing.T) {
	msg := make([]byte, 65536) // len > math.MaxUint16
	_, err := NewVerifySignature([32]byte{}, [64]byte{}, msg)
	if err == nil {
		t.Error("expected error for message length > 65535, got nil")
	}
}

func TestNewSignatureVerifyInstruction_TwoEntries(t *testing.T) {
	entries := []Ed25519SignatureOffsets{
		{SignatureOffset: 100, SignatureInstructionIndex: 1, PublicKeyOffset: 200, PublicKeyInstructionIndex: 2, MessageDataOffset: 300, MessageDataSize: 50, MessageInstructionIndex: 3},
		{SignatureOffset: 400, SignatureInstructionIndex: 4, PublicKeyOffset: 500, PublicKeyInstructionIndex: 5, MessageDataOffset: 600, MessageDataSize: 75, MessageInstructionIndex: 6},
	}
	ix, err := NewSignatureVerifyInstruction(entries)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := ix.Data()
	// 2 + 2 entries × 14 bytes = 30.
	if len(data) != 30 {
		t.Fatalf("len = %d, want 30", len(data))
	}
	if data[0] != 2 {
		t.Errorf("count = %d, want 2", data[0])
	}
	if data[1] != 0 {
		t.Errorf("padding = %d, want 0", data[1])
	}
	// First entry at offset 2.
	if binary.LittleEndian.Uint16(data[2:4]) != 100 {
		t.Errorf("entry[0].SignatureOffset = %d, want 100", binary.LittleEndian.Uint16(data[2:4]))
	}
	if binary.LittleEndian.Uint16(data[4:6]) != 1 {
		t.Errorf("entry[0].SignatureInstructionIndex = %d, want 1", binary.LittleEndian.Uint16(data[4:6]))
	}
	if binary.LittleEndian.Uint16(data[12:14]) != 50 {
		t.Errorf("entry[0].MessageDataSize = %d, want 50", binary.LittleEndian.Uint16(data[12:14]))
	}
	// Second entry at offset 16.
	if binary.LittleEndian.Uint16(data[16:18]) != 400 {
		t.Errorf("entry[1].SignatureOffset = %d, want 400", binary.LittleEndian.Uint16(data[16:18]))
	}
	if binary.LittleEndian.Uint16(data[28:30]) != 6 {
		t.Errorf("entry[1].MessageInstructionIndex = %d, want 6", binary.LittleEndian.Uint16(data[28:30]))
	}
}

func TestNewSignatureVerifyInstruction_TooManySigs(t *testing.T) {
	sigs := make([]Ed25519SignatureOffsets, 256) // len > math.MaxUint8
	_, err := NewSignatureVerifyInstruction(sigs)
	if err == nil {
		t.Error("expected error for signature count > 255, got nil")
	}
}

// TestNewVerifySignature_VerifiesViaStdlib signs a real message with
// crypto/ed25519, builds the precompile instruction, then re-extracts
// pubkey, signature, and message from the instruction data using the
// encoded offsets and confirms stdlib ed25519.Verify accepts it. This
// mirrors what the on-chain precompile does, so any offset / payload
// layout bug in NewVerifySignature surfaces here.
func TestNewVerifySignature_VerifiesViaStdlib(t *testing.T) {
	pub, priv, err := stded25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	msg := []byte("verify me end-to-end")
	sig := stded25519.Sign(priv, msg)

	var pubArr [32]byte
	var sigArr [64]byte
	copy(pubArr[:], pub)
	copy(sigArr[:], sig)

	ix, err := NewVerifySignature(pubArr, sigArr, msg)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := ix.Data()

	// Extract via the encoded offsets, exactly as the precompile would.
	pubOff := binary.LittleEndian.Uint16(data[6:8])
	sigOff := binary.LittleEndian.Uint16(data[2:4])
	msgOff := binary.LittleEndian.Uint16(data[10:12])
	msgSz := binary.LittleEndian.Uint16(data[12:14])

	extractedPub := stded25519.PublicKey(data[pubOff : pubOff+32])
	extractedSig := data[sigOff : sigOff+64]
	extractedMsg := data[msgOff : msgOff+uint16(msgSz)]

	if !stded25519.Verify(extractedPub, extractedMsg, extractedSig) {
		t.Fatal("stdlib ed25519.Verify rejected the round-tripped signature; offsets/payload layout is wrong")
	}
}

func TestProgramID_Base58(t *testing.T) {
	got := ProgramID.String()
	want := "Ed25519SigVerify111111111111111111111111111"
	if got != want {
		t.Errorf("ProgramID = %s, want %s", got, want)
	}
}
