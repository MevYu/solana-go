package solana

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	"github.com/MevYu/solana-go/internal/hdwallet"

	bip39 "github.com/tyler-smith/go-bip39"
)

// DefaultDerivationPath is the standard Solana BIP44 HD wallet path.
// All components are hardened as required by SLIP-0010 / Ed25519.
const DefaultDerivationPath = "m/44'/501'/0'/0'"

var reDerivationPath = regexp.MustCompile(`^m(/\d+')*$`)

// Ed25519KeypairFromMnemonic derives an Ed25519Keypair from a BIP39
// mnemonic phrase.
//
// password is the optional BIP39 passphrase (empty string for none).
// path is the HD derivation path; pass "" to use DefaultDerivationPath.
// Valid path format: m/44'/501'/0'/0' (all components must be hardened).
//
// Example:
//
//	kp, err := solana.Ed25519KeypairFromMnemonic(phrase, "", "")
//	kp, err := solana.Ed25519KeypairFromMnemonic(phrase, "", "m/44'/501'/1'/0'")
func Ed25519KeypairFromMnemonic(mnemonic, password, path string) (*Ed25519Keypair, error) {
	seed, err := bip39.NewSeedWithErrorChecking(mnemonic, password)
	if err != nil {
		return nil, fmt.Errorf("solana: mnemonic: %w", err)
	}

	if path == "" {
		path = DefaultDerivationPath
	}
	if !reDerivationPath.MatchString(path) {
		return nil, fmt.Errorf("solana: mnemonic: invalid derivation path %q", path)
	}

	derived, err := hdwallet.Derived(path, seed)
	if err != nil {
		return nil, fmt.Errorf("solana: mnemonic: %w", err)
	}

	return Ed25519KeypairFromSeed(derived.PrivateKey[:ed25519.SeedSize])
}

// GenerateMnemonic generates a new random BIP39 mnemonic with the
// given bit size (128 for 12 words, 256 for 24 words).
func GenerateMnemonic(bitSize int) (string, error) {
	entropy, err := bip39.NewEntropy(bitSize)
	if err != nil {
		return "", fmt.Errorf("solana: generate mnemonic: %w", err)
	}
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", fmt.Errorf("solana: generate mnemonic: %w", err)
	}
	return mnemonic, nil
}

// Ed25519KeypairFromKeygenFile loads an Ed25519Keypair from a Solana
// CLI keygen JSON file. The file contains a JSON array of uint8 values
// representing the 64-byte expanded private key.
//
// Example file: [1,2,3,...,64]
func Ed25519KeypairFromKeygenFile(path string) (*Ed25519Keypair, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("solana: keygen file: %w", err)
	}
	var keyBytes []byte
	if err := json.Unmarshal(data, &keyBytes); err != nil {
		return nil, fmt.Errorf("solana: keygen file: decode: %w", err)
	}
	return Ed25519KeypairFromPrivateKey(keyBytes)
}
