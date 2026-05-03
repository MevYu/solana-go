package solana

import (
	"crypto/sha256"
	"errors"
	"fmt"

	"filippo.io/edwards25519"
)

// MaxPDASeeds is the maximum number of seeds a program-derived
// address may be derived from, per the Solana runtime rules.
const MaxPDASeeds = 16

// MaxPDASeedLength is the maximum length in bytes of a single seed
// passed to CreateProgramAddress or FindProgramAddress.
const MaxPDASeedLength = 32

// pdaMarker is appended to the SHA256 input of every PDA
// derivation to domain-separate program-derived addresses from
// real Ed25519 public keys. It matches the marker string the
// Solana runtime uses.
var pdaMarker = []byte("ProgramDerivedAddress")

// CreateProgramAddress derives a program address from a sequence
// of seeds and a program id. It is the direct counterpart of the
// Solana runtime's create_program_address.
//
// The derivation is:
//
//	sha256(seed1 || seed2 || ... || programID || "ProgramDerivedAddress")
//
// The result is a 32-byte public key that must NOT lie on the
// ed25519 curve; if the hash happens to land on a curve point,
// CreateProgramAddress returns an error and the caller is
// expected to vary the seeds (typically by appending a "bump"
// byte via FindProgramAddress).
//
// Constraints:
//   - at most MaxPDASeeds seeds
//   - each seed at most MaxPDASeedLength bytes
func CreateProgramAddress(seeds [][]byte, programID PublicKey) (PublicKey, error) {
	if len(seeds) > MaxPDASeeds {
		return PublicKey{}, fmt.Errorf("solana: CreateProgramAddress: max %d seeds, got %d", MaxPDASeeds, len(seeds))
	}
	h := sha256.New()
	for i, seed := range seeds {
		if len(seed) > MaxPDASeedLength {
			return PublicKey{}, fmt.Errorf("solana: CreateProgramAddress: seed %d is %d bytes, max %d", i, len(seed), MaxPDASeedLength)
		}
		h.Write(seed)
	}
	h.Write(programID[:])
	h.Write(pdaMarker)
	sum := h.Sum(nil)

	var pk PublicKey
	copy(pk[:], sum)

	if isOnCurve(pk[:]) {
		return PublicKey{}, errors.New("solana: CreateProgramAddress: invalid seeds, produced on-curve point")
	}
	return pk, nil
}

// FindProgramAddress iterates over bump seeds from 255 down to 0
// until it finds one that produces an off-curve program address.
// It is the direct counterpart of the Solana runtime's
// find_program_address.
//
// The returned bump is the byte that was appended to seeds to
// produce the valid PDA; store it alongside the address if you
// plan to re-derive the same PDA later via CreateProgramAddress
// (re-deriving without the bump is possible but wastes CPU).
//
// Returns an error only in the astronomically unlikely event that
// every bump from 255 to 0 produces an on-curve point.
func FindProgramAddress(seeds [][]byte, programID PublicKey) (PublicKey, uint8, error) {
	if len(seeds) >= MaxPDASeeds {
		return PublicKey{}, 0, fmt.Errorf("solana: FindProgramAddress: seeds would exceed max after adding bump")
	}
	// Hoist the augmented-seeds buffer out of the loop. CreateProgramAddress
	// only reads each seed; mutating the bump byte in-place between iterations
	// is safe and avoids up to 256 redundant slice allocations.
	augmented := make([][]byte, len(seeds)+1)
	copy(augmented, seeds)
	var bumpBuf [1]byte
	augmented[len(seeds)] = bumpBuf[:]
	for bump := 255; bump >= 0; bump-- {
		bumpBuf[0] = byte(bump)
		pk, err := CreateProgramAddress(augmented, programID)
		if err == nil {
			return pk, uint8(bump), nil
		}
	}
	return PublicKey{}, 0, errors.New("solana: FindProgramAddress: no off-curve PDA found for any bump (statistically impossible)")
}

// isOnCurve reports whether a 32-byte value is a valid compressed
// Ed25519 public key. It is the check that distinguishes a real
// key (which a private key exists for) from a program-derived
// address (which by construction has no corresponding private key).
func isOnCurve(data []byte) bool {
	if len(data) != PublicKeySize {
		return false
	}
	_, err := new(edwards25519.Point).SetBytes(data)
	return err == nil
}
