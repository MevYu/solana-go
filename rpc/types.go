package rpc

import solana "github.com/MevYu/solana-go"

// LatestBlockhash is the decoded response of GetLatestBlockhash.
type LatestBlockhash struct {
	Slot                 uint64      `json:"-"`
	Blockhash            solana.Hash `json:"blockhash"`
	LastValidBlockHeight uint64      `json:"lastValidBlockHeight"`
}

// SignatureStatus is the status of a single transaction signature.
type SignatureStatus struct {
	Slot               uint64  `json:"slot"`
	Confirmations      *uint64 `json:"confirmations"`
	Err                any     `json:"err"`
	ConfirmationStatus string  `json:"confirmationStatus"`
}

// GetSignatureStatusesResult is the decoded response of GetSignatureStatuses.
type GetSignatureStatusesResult struct {
	Slot     uint64
	Statuses []*SignatureStatus
}

// SimulationReturnData is the data returned by the last instruction of a simulated transaction.
type SimulationReturnData struct {
	ProgramID solana.PublicKey   `json:"programId"`
	Data      solana.EncodedData `json:"data"`
}

// SimulateResult is the decoded response of SimulateTransaction. Slot
// is filled from the JSON-RPC context envelope after decode; the
// remaining fields decode directly from the value object via their
// json: tags. PublicKey decodes from a base58 string via its own
// UnmarshalJSON.
type SimulateResult struct {
	Slot          uint64                `json:"-"`
	Err           any                   `json:"err"`
	Logs          []string              `json:"logs"`
	UnitsConsumed *uint64               `json:"unitsConsumed"`
	Accounts      []*solana.AccountInfo `json:"accounts"`
	ReturnData    *SimulationReturnData `json:"returnData"`
}

// PrioritizationFee is a single data point in a GetRecentPrioritizationFees response.
type PrioritizationFee struct {
	Slot              uint64 `json:"slot"`
	PrioritizationFee uint64 `json:"prioritizationFee"`
}

// TransactionMeta summarises execution outcome, balance changes, and logs.
type TransactionMeta struct {
	Err                  any      `json:"err"`
	Fee                  uint64   `json:"fee"`
	PreBalances          []uint64 `json:"preBalances"`
	PostBalances         []uint64 `json:"postBalances"`
	LogMessages          []string `json:"logMessages"`
	ComputeUnitsConsumed uint64   `json:"computeUnitsConsumed"`
	InnerInstructions    []any    `json:"innerInstructions,omitempty"`
	PreTokenBalances     []any    `json:"preTokenBalances,omitempty"`
	PostTokenBalances    []any    `json:"postTokenBalances,omitempty"`
	Rewards              []any    `json:"rewards,omitempty"`
	LoadedAddresses      *struct {
		Writable []string `json:"writable"`
		Readonly []string `json:"readonly"`
	} `json:"loadedAddresses,omitempty"`
}

// BlockTransaction is a single entry in the transactions array of a GetBlock response.
type BlockTransaction struct {
	Transaction solana.EncodedData `json:"transaction"`
	Meta        *TransactionMeta   `json:"meta"`
	Version     any                `json:"version"`
}

// GetBlockResult is the decoded response of GetBlock.
type GetBlockResult struct {
	Blockhash         string             `json:"blockhash"`
	PreviousBlockhash string             `json:"previousBlockhash"`
	ParentSlot        uint64             `json:"parentSlot"`
	Transactions      []BlockTransaction `json:"transactions"`
	BlockHeight       *uint64            `json:"blockHeight"`
	BlockTime         *int64             `json:"blockTime"`
	Rewards           []any              `json:"rewards"`
}
