package rpc

import (
	gojson "github.com/goccy/go-json"

	solana "github.com/MevYu/solana-go"
)

// marshalJSON is the package-internal entry point for the SDK's
// MarshalJSON helpers on Cfg types. It uses goccy/go-json to match
// the default Codec; callers that swap to StdCodec via WithCodec
// still get correct output because both codecs read MarshalJSON.
func marshalJSON(v any) ([]byte, error) { return gojson.Marshal(v) }

// DataSlice requests a subrange of account data to be returned,
// reducing bandwidth for large accounts.
type DataSlice struct {
	Offset uint64 `json:"offset"`
	Length uint64 `json:"length"`
}

// All Cfg structs below carry json tags so they can be passed
// directly to the JSON-RPC codec as the params object — there is
// no parallel wire-shape struct in the client package.
//
// Pointer fields combined with `omitempty` ensure that an unset
// option is dropped from the wire payload entirely, so the server
// applies its own default. String fields use `omitempty` too;
// fields where the SDK injects a non-server default (Encoding for
// SendTx/GetBlock, TransactionDetails for GetBlock) implement
// MarshalJSON to apply the default at marshal time.

// CommitmentCfg is the simplest config object accepted by RPC methods
// that only honour {commitment}: GetSupply, GetVoteAccounts,
// GetBlockProduction, GetMinimumBalanceForRentExemption,
// GetStakeMinimumDelegation, GetInflationGovernor, RequestAirdrop,
// GetTokenLargestAccounts, GetTokenSupply, GetTokenAccountBalance,
// GetBlocks, GetBlocksWithLimit.
type CommitmentCfg struct {
	Commitment solana.CommitmentLevel `json:"commitment,omitempty"`
}

// CommitmentWithMinSlotCfg is the config object for RPC methods that
// honour {commitment, minContextSlot}: GetBalance, GetSlot,
// GetBlockHeight, GetLatestBlockhash, GetEpochInfo, GetSlotLeader,
// GetTransactionCount, IsBlockhashValid, GetFeeForMessage,
// GetStakeActivation.
type CommitmentWithMinSlotCfg struct {
	Commitment     solana.CommitmentLevel `json:"commitment,omitempty"`
	MinContextSlot *uint64                `json:"minContextSlot,omitempty"`
}

// CommitmentWithEncodingCfg is the config object for WebSocket
// subscriptions that honour {commitment, encoding}:
// AccountSubscribe, ProgramSubscribe.
type CommitmentWithEncodingCfg struct {
	Commitment solana.CommitmentLevel `json:"commitment,omitempty"`
	Encoding   solana.Encoding        `json:"encoding,omitempty"`
}

// AccountInfoCfg is the config object for account-data RPC methods
// that honour {commitment, encoding, dataSlice, minContextSlot}:
// GetAccountInfo, GetMultipleAccounts, GetProgramAccounts,
// GetTokenAccountsByOwner, GetTokenAccountsByDelegate.
type AccountInfoCfg struct {
	Commitment     solana.CommitmentLevel `json:"commitment,omitempty"`
	Encoding       solana.Encoding        `json:"encoding,omitempty"`
	DataSlice      *DataSlice             `json:"dataSlice,omitempty"`
	MinContextSlot *uint64                `json:"minContextSlot,omitempty"`
}

// GetBlockCfg is the config object for GetBlock and BlockSubscribe.
// TransactionDetails defaults to "full" when empty; Encoding defaults
// to base64 when empty (both via MarshalJSON below).
type GetBlockCfg struct {
	Commitment                     solana.CommitmentLevel `json:"commitment,omitempty"`
	Encoding                       solana.Encoding        `json:"encoding,omitempty"`
	MaxSupportedTransactionVersion *uint64                `json:"maxSupportedTransactionVersion,omitempty"`
	TransactionDetails             string                 `json:"transactionDetails,omitempty"` // "full" | "accounts" | "signatures" | "none"
	Rewards                        *bool                  `json:"rewards,omitempty"`
}

// GetTransactionCfg is the config object for GetTransaction.
type GetTransactionCfg struct {
	Commitment                     solana.CommitmentLevel `json:"commitment,omitempty"`
	Encoding                       solana.Encoding        `json:"encoding,omitempty"`
	MaxSupportedTransactionVersion *uint64                `json:"maxSupportedTransactionVersion,omitempty"`
}

// SignaturesForAddressCfg is the config object for GetSignaturesForAddress.
type SignaturesForAddressCfg struct {
	Commitment     solana.CommitmentLevel `json:"commitment,omitempty"`
	MinContextSlot *uint64                `json:"minContextSlot,omitempty"`
	Limit          *int                   `json:"limit,omitempty"`
	Before         string                 `json:"before,omitempty"`
	Until          string                 `json:"until,omitempty"`
}

// SignatureStatusesCfg is the config object for GetSignatureStatuses.
type SignatureStatusesCfg struct {
	SearchTransactionHistory *bool `json:"searchTransactionHistory,omitempty"`
}

// SendTxCfg is the config object for SendTransaction and
// SendRawTransaction. Encoding may be base58 or base64; base64 is the
// default and recommended choice (applied via MarshalJSON below).
type SendTxCfg struct {
	SkipPreflight       *bool                  `json:"skipPreflight,omitempty"`
	PreflightCommitment solana.CommitmentLevel `json:"preflightCommitment,omitempty"`
	MaxRetries          *uint                  `json:"maxRetries,omitempty"`
	MinContextSlot      *uint64                `json:"minContextSlot,omitempty"`
	Encoding            solana.Encoding        `json:"encoding,omitempty"`
}

// SimulateTxCfg is the config object for SimulateTransaction.
type SimulateTxCfg struct {
	Commitment             solana.CommitmentLevel `json:"commitment,omitempty"`
	SigVerify              *bool                  `json:"sigVerify,omitempty"`
	ReplaceRecentBlockhash *bool                  `json:"replaceRecentBlockhash,omitempty"`
	MinContextSlot         *uint64                `json:"minContextSlot,omitempty"`
	Encoding               solana.Encoding        `json:"encoding,omitempty"`
}

// LargestAccountsCfg is the config object for GetLargestAccounts.
// Filter is "circulating" or "nonCirculating".
type LargestAccountsCfg struct {
	Commitment solana.CommitmentLevel `json:"commitment,omitempty"`
	Filter     string                 `json:"filter,omitempty"`
}

// InflationRewardCfg is the config object for GetInflationReward.
type InflationRewardCfg struct {
	Commitment     solana.CommitmentLevel `json:"commitment,omitempty"`
	MinContextSlot *uint64                `json:"minContextSlot,omitempty"`
	Epoch          *uint64                `json:"epoch,omitempty"`
}

// LeaderScheduleCfg is the config object for GetLeaderSchedule.
type LeaderScheduleCfg struct {
	Commitment solana.CommitmentLevel `json:"commitment,omitempty"`
	Identity   string                 `json:"identity,omitempty"`
}

// LogsSubscribeCfg is the config object for LogsSubscribe.
type LogsSubscribeCfg struct {
	Commitment solana.CommitmentLevel `json:"commitment,omitempty"`
}

// SignatureSubscribeCfg is the config object for SignatureSubscribe.
type SignatureSubscribeCfg struct {
	Commitment solana.CommitmentLevel `json:"commitment,omitempty"`
}

// MarshalJSON injects the SDK's default encoding ("base64") into
// AccountInfoCfg before serialising. Account-data RPCs always need a
// concrete encoding so the SDK fills in base64 when the caller leaves
// it blank.
//
// Centralising the default here pays off because five typed methods
// (GetAccountInfo, GetMultipleAccounts, GetProgramAccounts,
// GetTokenAccountsByOwner, GetTokenAccountsByDelegate) share this Cfg.
// Other Cfg types with only one or two callers inject defaults inline
// in the calling method instead, which avoids the alias-MarshalJSON
// ceremony for the rare paths.
func (c AccountInfoCfg) MarshalJSON() ([]byte, error) {
	if c.Encoding == "" {
		c.Encoding = solana.EncodingBase64
	}
	type alias AccountInfoCfg
	return marshalJSON(alias(c))
}

// firstOrZero returns a pointer to opts[0] when non-empty, otherwise
// a pointer to a zero value of T. Useful in methods that take a
// variadic config so the caller can omit the argument entirely.
func firstOrZero[T any](opts []T) *T {
	if len(opts) == 0 {
		var z T
		return &z
	}
	return &opts[0]
}

// FirstOrZero is the exported form of firstOrZero, used by callers
// that build their own request body (helpers, third-party wrappers).
func FirstOrZero[T any](opts []T) *T { return firstOrZero(opts) }
