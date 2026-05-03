package secp256k1

import (
	"fmt"
	"math"

	solana "github.com/MevYu/solana-go"
	"github.com/MevYu/solana-go/encoding"
)

// Secp256k1SignatureOffsets describes the byte offsets within a
// secp256k1 precompile instruction where the signature components
// can be found. On-wire layout is 11 bytes, all fields little-endian.
type Secp256k1SignatureOffsets struct {
	SignatureOffset            uint16 // byte offset to the 64-byte secp256k1 signature
	SignatureInstructionIndex  uint8  // which transaction instruction holds the signature
	EthAddressOffset           uint16 // byte offset to the 20-byte Ethereum address
	EthAddressInstructionIndex uint8  // which transaction instruction holds the eth address
	MessageDataOffset          uint16 // byte offset to the signed message data
	MessageDataSize            uint16 // length of the message data in bytes
	MessageInstructionIndex    uint8  // which transaction instruction holds the message
}

type secp256k1Ix struct{ data []byte }

func (s *secp256k1Ix) ProgramID() solana.PublicKey     { return ProgramID }
func (s *secp256k1Ix) Accounts() []*solana.AccountMeta { return nil }
func (s *secp256k1Ix) Data() ([]byte, error)           { return s.data, nil }

// NewRawInstruction wraps pre-encoded secp256k1 precompile data as a
// solana.Instruction.
func NewRawInstruction(data []byte) solana.Instruction {
	return &secp256k1Ix{data: data}
}

// NewInstruction is an alias for NewRawInstruction.
func NewInstruction(data []byte) solana.Instruction {
	return NewRawInstruction(data)
}

// NewVerifyEthSignature builds a self-contained Secp256k1 verification
// instruction. All data (signature, Ethereum address, message) is packed
// inline in the instruction data field; no other instructions are needed.
//
//   ethAddress  — 20-byte Ethereum address (keccak256(uncompressed pubkey)[12:])
//   signature   — 64-byte compact ECDSA signature (r ‖ s, no recovery bit)
//   recoveryID  — 0 or 1
//   message     — bytes that were signed; must be ≤ 65535 bytes
//
// The instruction has no accounts; the precompile reads everything from
// the instruction data.
func NewVerifyEthSignature(ethAddress [20]byte, signature [64]byte, recoveryID uint8, message []byte) (solana.Instruction, error) {
	if len(message) > math.MaxUint16 {
		return nil, fmt.Errorf("secp256k1: message length %d exceeds uint16 max (%d)", len(message), math.MaxUint16)
	}
	// Inline layout:
	//  byte   0:        count = 1
	//  bytes  1–11:     Secp256k1SignatureOffsets (11 bytes: 5×u16LE + 3×u8)
	//  bytes 12–75:     signature (64 bytes)
	//  byte  76:        recoveryID
	//  bytes 77–96:     ethAddress (20 bytes)
	//  bytes 97–…:      message
	const (
		sigOffset = uint16(12)
		ethOffset = uint16(77) // 12 + 64 + 1 recoveryID
		msgOffset = uint16(97) // 77 + 20
		selfIx    = uint8(0xff) // "this instruction"
	)
	data := encoding.NewEncoder(int(msgOffset)+len(message)).
		U8(1). // count
		U16(sigOffset).U8(selfIx).
		U16(ethOffset).U8(selfIx).
		U16(msgOffset).U16(uint16(len(message))).U8(selfIx).
		Raw(signature[:]).
		U8(recoveryID).
		Raw(ethAddress[:]).
		Raw(message).
		Bytes()
	return &secp256k1Ix{data: data}, nil
}

// NewSignatureVerifyInstruction builds a secp256k1 precompile
// instruction from a slice of Secp256k1SignatureOffsets descriptors.
// Layout: [1 byte count] [11 bytes per entry, LE encoded].
// len(signatures) must not exceed 255 (the on-chain count field is u8).
func NewSignatureVerifyInstruction(signatures []Secp256k1SignatureOffsets) (solana.Instruction, error) {
	const offsetsSize = 11
	if len(signatures) > math.MaxUint8 {
		return nil, fmt.Errorf("secp256k1: signature count %d exceeds uint8 max (%d)", len(signatures), math.MaxUint8)
	}
	e := encoding.NewEncoder(1 + len(signatures)*offsetsSize).U8(byte(len(signatures)))
	for _, sig := range signatures {
		e.U16(sig.SignatureOffset).U8(sig.SignatureInstructionIndex).
			U16(sig.EthAddressOffset).U8(sig.EthAddressInstructionIndex).
			U16(sig.MessageDataOffset).U16(sig.MessageDataSize).U8(sig.MessageInstructionIndex)
	}
	return &secp256k1Ix{data: e.Bytes()}, nil
}
