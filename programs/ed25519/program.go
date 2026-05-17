// Package ed25519 provides instruction builders for the Solana ed25519
// precompile program, which verifies Ed25519 signatures. It is a native
// precompile: it reads instruction data only and takes no account inputs.
package ed25519

import solana "github.com/MevYu/solana-go"

// ProgramID is the canonical address of the Solana ed25519 precompile:
// Ed25519SigVerify111111111111111111111111111.
var ProgramID = solana.MustPublicKey("Ed25519SigVerify111111111111111111111111111")
