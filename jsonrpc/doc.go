// Package rpc is a minimal, clean-room JSON-RPC 2.0 client for the
// Solana HTTP endpoint.
//
// The package is deliberately small and has no third-party
// dependencies. It does not import the root solana package; it
// speaks pure JSON over HTTP so the higher-level typed wrappers can
// live one layer up without a dependency cycle.
//
// A caller wanting a faster JSON library can swap stdlib
// encoding/json out at construction time via WithCodec, without
// touching the transport. See StdCodec for the default.
package jsonrpc
