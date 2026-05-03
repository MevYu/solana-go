package solana

// CommitmentLevel describes the level of confirmation required for
// a query. Solana defines three tiers, from fastest-but-reversible
// to slowest-but-permanent.
type CommitmentLevel string

const (
	// CommitmentProcessed queries the most recent block. It has the
	// lowest latency but may be rolled back.
	CommitmentProcessed CommitmentLevel = "processed"

	// CommitmentConfirmed queries the most recent block that has
	// been voted on by a supermajority of the cluster.
	CommitmentConfirmed CommitmentLevel = "confirmed"

	// CommitmentFinalized queries the most recent block that a
	// supermajority of the cluster has rooted, so it will never be
	// rolled back. This is the safest default for money movement.
	CommitmentFinalized CommitmentLevel = "finalized"
)

// EnumRpcCommitment is an alias for CommitmentLevel, matching the
// go-solana naming convention.
type EnumRpcCommitment = CommitmentLevel

// go-solana-compatible commitment constants.
const (
	RpcCommitmentProcessed = CommitmentProcessed
	RpcCommitmentConfirmed = CommitmentConfirmed
	RpcCommitmentFinalized = CommitmentFinalized
)

// Encoding is the on-wire encoding for binary payloads in RPC
// responses that carry account data or transaction data.
type Encoding string

const (
	// EncodingJSONParsed asks the server to parse the data into a
	// program-specific JSON object (for example SPL Token account
	// state). Only programs the server recognises support this.
	EncodingJSONParsed Encoding = "jsonParsed"

	// EncodingJSON asks the server to return the data as a structured
	// JSON object. Very few programs support this form.
	EncodingJSON Encoding = "json"

	// EncodingBase58 asks the server to return the data as a
	// base58-encoded string. Slow for large payloads; avoid.
	EncodingBase58 Encoding = "base58"

	// EncodingBase64 asks the server to return the data as a
	// base64-encoded string. The most common choice.
	EncodingBase64 Encoding = "base64"

	// EncodingBase64ZSTD asks the server to return the data as
	// zstd-compressed base64. Saves bandwidth on large accounts but
	// requires a zstd decoder on the client side.
	EncodingBase64ZSTD Encoding = "base64+zstd"
)

// EncodingEnum is an alias for Encoding, matching the go-solana
// naming convention.
type EncodingEnum = Encoding

// go-solana-compatible encoding constants.
const (
	EncodingBase64Zstd Encoding = EncodingBase64ZSTD
	EncodingJsonParsed Encoding = EncodingJSONParsed
	EncodingJson       Encoding = EncodingJSON
)

// EnumTxDetailLevel is the transaction detail level for
// getBlock-style calls.
type EnumTxDetailLevel string

const (
	TxDetailLevelNone       EnumTxDetailLevel = "none"
	TxDetailLevelFull       EnumTxDetailLevel = "full"
	TxDetailLevelAccounts   EnumTxDetailLevel = "accounts"
	TxDetailLevelSignatures EnumTxDetailLevel = "signatures"
)

// EnumCirculateFilter filters accounts by circulation status.
type EnumCirculateFilter string

const (
	FilterCirculating    EnumCirculateFilter = "circulating"
	FilterNonCirculating EnumCirculateFilter = "nonCirculating"
)
