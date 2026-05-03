// Package secp256k1 provides instruction builders for the Solana
// secp256k1 precompile program, which verifies Ethereum-style ECDSA
// signatures. It is a native precompile: it reads instruction data
// only and takes no account inputs.
package secp256k1

import solana "github.com/MevYu/solana-go"

// ProgramID is the canonical address of the Solana secp256k1
// precompile: KeccakSecp256k11111111111111111111111111111.
var ProgramID = solana.MustPublicKey("KeccakSecp256k11111111111111111111111111111")
