package ed25519

import (
	"fmt"
	"math"

	solana "github.com/MevYu/solana-go"
	"github.com/MevYu/solana-go/encoding"
)

// Ed25519SignatureOffsets describes the byte offsets within an ed25519
// precompile instruction where the signature components can be found.
// On-wire layout is 14 bytes, seven u16 fields in little-endian order.
// Unlike Secp256k1SignatureOffsets, every field — including the three
// instruction-index fields — is u16.
type Ed25519SignatureOffsets struct {
	SignatureOffset            uint16 // byte offset to the 64-byte ed25519 signature
	SignatureInstructionIndex  uint16 // which transaction instruction holds the signature; 0xFFFF = this instruction
	PublicKeyOffset            uint16 // byte offset to the 32-byte ed25519 public key
	PublicKeyInstructionIndex  uint16 // which transaction instruction holds the public key
	MessageDataOffset          uint16 // byte offset to the signed message data
	MessageDataSize            uint16 // length of the message data in bytes
	MessageInstructionIndex    uint16 // which transaction instruction holds the message
}

type ed25519Ix struct{ data []byte }

func (s *ed25519Ix) ProgramID() solana.PublicKey     { return ProgramID }
func (s *ed25519Ix) Accounts() []*solana.AccountMeta { return nil }
func (s *ed25519Ix) Data() ([]byte, error)           { return s.data, nil }

// NewRawInstruction wraps pre-encoded ed25519 precompile data as a
// solana.Instruction.
func NewRawInstruction(data []byte) solana.Instruction {
	return &ed25519Ix{data: data}
}

// NewInstruction is an alias for NewRawInstruction.
func NewInstruction(data []byte) solana.Instruction {
	return NewRawInstruction(data)
}

// NewVerifySignature builds a self-contained ed25519 verification
// instruction. All data (public key, signature, message) is packed inline
// in the instruction data field; no other instructions are needed.
//
//	publicKey — 32-byte ed25519 public key
//	signature — 64-byte ed25519 signature
//	message   — bytes that were signed; must be ≤ 65535 bytes
//
// The instruction has no accounts; the precompile reads everything from
// the instruction data.
func NewVerifySignature(publicKey [32]byte, signature [64]byte, message []byte) (solana.Instruction, error) {
	if len(message) > math.MaxUint16 {
		return nil, fmt.Errorf("ed25519: message length %d exceeds uint16 max (%d)", len(message), math.MaxUint16)
	}
	// Inline layout (matches Rust solana-sdk new_ed25519_instruction):
	//  byte   0:        count = 1
	//  byte   1:        padding = 0
	//  bytes  2–15:     Ed25519SignatureOffsets (14 bytes, 7×u16 LE)
	//  bytes 16–47:     publicKey (32 bytes)
	//  bytes 48–111:    signature (64 bytes)
	//  bytes 112–…:     message
	const (
		pubkeyOffset = uint16(16)
		sigOffset    = uint16(48)  // 16 + 32
		msgOffset    = uint16(112) // 48 + 64
		selfIx       = uint16(0xFFFF)
	)
	data := encoding.NewEncoder(int(msgOffset) + len(message)).
		U8(1).             // count
		U8(0).             // padding
		U16(sigOffset).U16(selfIx).
		U16(pubkeyOffset).U16(selfIx).
		U16(msgOffset).U16(uint16(len(message))).U16(selfIx).
		Raw(publicKey[:]).
		Raw(signature[:]).
		Raw(message).
		Bytes()
	return &ed25519Ix{data: data}, nil
}

// NewSignatureVerifyInstruction builds an ed25519 precompile instruction
// from a slice of Ed25519SignatureOffsets descriptors. Layout:
// [1 byte count] [1 byte padding] [14 bytes per entry, LE encoded].
// len(signatures) must not exceed 255 (the on-chain count field is u8).
func NewSignatureVerifyInstruction(signatures []Ed25519SignatureOffsets) (solana.Instruction, error) {
	const offsetsSize = 14
	if len(signatures) > math.MaxUint8 {
		return nil, fmt.Errorf("ed25519: signature count %d exceeds uint8 max (%d)", len(signatures), math.MaxUint8)
	}
	e := encoding.NewEncoder(2 + len(signatures)*offsetsSize).U8(byte(len(signatures))).U8(0)
	for _, sig := range signatures {
		e.U16(sig.SignatureOffset).U16(sig.SignatureInstructionIndex).
			U16(sig.PublicKeyOffset).U16(sig.PublicKeyInstructionIndex).
			U16(sig.MessageDataOffset).U16(sig.MessageDataSize).U16(sig.MessageInstructionIndex)
	}
	return &ed25519Ix{data: e.Bytes()}, nil
}
