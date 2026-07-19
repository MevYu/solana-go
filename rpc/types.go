package rpc

import (
	"encoding/json"
	"math/big"

	"github.com/MevYu/solana-go"
)

// GetSignatureStatusesResult is the decoded response of GetSignatureStatuses.
type GetSignatureStatusesResult struct {
	Slot     uint64
	Statuses []*solana.SignatureStatus
}

// SimulationReturnData is the data returned by the last instruction of a simulated transaction.
type SimulationReturnData struct {
	ProgramID solana.PublicKey   `json:"programId"`
	Data      solana.EncodedData `json:"data"`
}

// SimulateResult is the decoded response of SimulateTransaction. Slot
// is filled from the JSON-RPC context envelope after decode; the
// remaining fields decode directly from the value object via their
// json: tags.
type SimulateResult struct {
	Slot     uint64                `json:"-"`
	Accounts []*solana.AccountInfo `json:"accounts"`
	// Error if transaction failed, null if transaction succeeded.
	// https://github.com/solana-labs/solana/blob/master/sdk/src/transaction.rs#L24
	Err json.RawMessage `json:"err"`
	// Fee this transaction was charged
	Fee uint64 `json:"fee"`

	// Array of *big.Int account balances from before the transaction was processed
	PreBalances []*big.Int `json:"preBalances"`

	// Array of *big.Int account balances after the transaction was processed
	PostBalances []*big.Int `json:"postBalances"`

	// List of inner instructions or omitted if inner instruction recording
	// was not yet enabled during this transaction
	InnerInstructions []solana.InnerInstruction `json:"innerInstructions"`

	// List of token balances from before the transaction was processed
	// or omitted if token balance recording was not yet enabled during this transaction
	PreTokenBalances []solana.TokenBalance `json:"preTokenBalances"`

	// List of token balances from after the transaction was processed
	// or omitted if token balance recording was not yet enabled during this transaction
	PostTokenBalances []solana.TokenBalance `json:"postTokenBalances"`

	LoadedAccountsDataSize *uint32                `json:"loadedAccountsDataSize"`
	LoadedAddresses        solana.LoadedAddresses `json:"loadedAddresses"`
	// Array of string log messages or omitted if log message
	// recording was not yet enabled during this transaction
	Logs []string `json:"logs"`

	ReplacementBlockhash *LastBlock            `json:"replacementBlockhash"`
	ReturnData           *SimulationReturnData `json:"returnData"`
	UnitsConsumed        *uint64               `json:"unitsConsumed"`
}

// BlockTransaction is a single entry in the transactions array of a
// GetBlock response. Transaction is fully decoded via
// Transaction.UnmarshalJSON when the encoding is base64 / base58 /
// base64+zstd.
type BlockTransaction struct {
	Transaction *solana.Transaction     `json:"transaction"`
	Meta        *solana.TransactionMeta `json:"meta"`
	Version     solana.MessageVersion   `json:"version"`
}

type LastBlock struct {
	// a Hash as base-58 encoded string
	Blockhash solana.Hash `json:"blockhash"`
	//  last block height at which the blockhash will be valid
	LastValidBlockHeight uint64 `json:"lastValidBlockHeight"`
}

// GetBlockResult is the decoded response of GetBlock.
type GetBlockResult struct {
	Blockhash         string             `json:"blockhash"`
	PreviousBlockhash string             `json:"previousBlockhash"`
	ParentSlot        uint64             `json:"parentSlot"`
	Transactions      []BlockTransaction `json:"transactions"`
	BlockHeight       *uint64            `json:"blockHeight"`
	BlockTime         *int64             `json:"blockTime"`
	Rewards           []solana.Reward    `json:"rewards,omitempty"`
}
