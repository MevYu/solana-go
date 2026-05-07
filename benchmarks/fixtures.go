package benchmarks

import _ "embed"

// Raw fixture bytes, embedded at compile time. Each fixture is the
// result payload of a specific Solana JSON-RPC method, not the full
// JSON-RPC envelope; the envelope parsing is the rpc.Client's
// concern, so measuring it here would conflate transport and
// decode.

//go:embed testdata/getAccountInfo.json
var getAccountInfoJSON []byte

//go:embed testdata/getTransaction.json
var getTransactionJSON []byte

//go:embed testdata/getBlock.json
var getBlockJSON []byte

// GetAccountInfoJSON returns the bytes of a synthetic getAccountInfo
// result payload. The fixture is a rent-exempt SPL Token account.
func GetAccountInfoJSON() []byte { return getAccountInfoJSON }

// GetTransactionJSON returns the bytes of a synthetic getTransaction
// result payload. The fixture is a single legacy system-transfer.
func GetTransactionJSON() []byte { return getTransactionJSON }

// GetBlockJSON returns the bytes of a synthetic getBlock result
// payload. The fixture holds a handful of transactions; it is sized
// for representative block decoding, not for real mainnet blocks.
func GetBlockJSON() []byte { return getBlockJSON }
