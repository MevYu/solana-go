// Package hdwallet implements SLIP-0010 hierarchical deterministic key
// derivation for Ed25519. Solana uses hardened-only derivation paths of
// the form m/44'/501'/0'/0'.
package hdwallet

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/binary"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Key holds an Ed25519 private key and its chain code as produced by
// SLIP-0010 derivation.
type Key struct {
	PrivateKey []byte
	ChainCode  []byte
}

// CreateMasterKey derives the master key from a BIP39 seed using the
// SLIP-0010 Ed25519 domain string "ed25519 seed".
func CreateMasterKey(seed []byte) Key {
	mac := hmac.New(sha512.New, []byte("ed25519 seed"))
	mac.Write(seed)
	I := mac.Sum(nil)
	return Key{PrivateKey: I[:32], ChainCode: I[32:]}
}

// CKDPriv performs SLIP-0010 hardened child key derivation. index must
// have the hardened bit set (index >= 1<<31).
func CKDPriv(key Key, index uint32) Key {
	buf := make([]byte, 0, 37)
	buf = append(buf, 0x00)
	buf = append(buf, key.PrivateKey...)
	var idx [4]byte
	binary.BigEndian.PutUint32(idx[:], index)
	buf = append(buf, idx[:]...)

	mac := hmac.New(sha512.New, key.ChainCode)
	mac.Write(buf)
	I := mac.Sum(nil)
	return Key{PrivateKey: I[:32], ChainCode: I[32:]}
}

var validPath = regexp.MustCompile(`^m(/\d+')*$`)

// Derived derives a child key from seed following path. path must be of
// the form m/44'/501'/0'/0' (all components hardened). Returns an error
// if the path is malformed.
func Derived(path string, seed []byte) (Key, error) {
	if !validPath.MatchString(path) {
		return Key{}, fmt.Errorf("hdwallet: invalid derivation path %q", path)
	}
	key := CreateMasterKey(seed)
	for _, segment := range strings.Split(path, "/")[1:] {
		// each segment ends with '
		n, err := strconv.ParseUint(segment[:len(segment)-1], 10, 32)
		if err != nil {
			return Key{}, fmt.Errorf("hdwallet: parse segment %q: %w", segment, err)
		}
		key = CKDPriv(key, uint32(n)+1<<31)
	}
	return key, nil
}
