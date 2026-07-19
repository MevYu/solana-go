package rpc

import (
	"encoding/json"

	solana "github.com/MevYu/solana-go"
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

// SimulationInnerInstruction groups the inner instructions invoked by one
// top-level instruction during simulation. Unlike transaction metadata, the
// simulateTransaction RPC returns inner instructions in JSON-parsed form.
type SimulationInnerInstruction struct {
	Index        uint8                   `json:"index"`
	Instructions []SimulationInstruction `json:"instructions"`
}

// SimulationInstruction is the common wire shape of a parsed or partially
// decoded instruction returned by simulateTransaction. Parsed contains the
// program-specific JSON payload for parsed instructions. Accounts and Data
// are populated for partially decoded instructions.
type SimulationInstruction struct {
	Program     string             `json:"program,omitempty"`
	ProgramID   solana.PublicKey   `json:"programId"`
	Parsed      json.RawMessage    `json:"parsed,omitempty"`
	Accounts    []solana.PublicKey `json:"accounts,omitempty"`
	Data        solana.Base58Data  `json:"data,omitempty"`
	StackHeight *uint32            `json:"stackHeight,omitempty"`
}

// SimulateResult is the decoded response of SimulateTransaction. Slot
// is filled from the JSON-RPC context envelope after decode; the
// remaining fields decode directly from the value object via their
// json: tags.
type SimulateResult struct {
	Slot                   uint64                       `json:"-"`
	Accounts               []*solana.AccountInfo        `json:"accounts"`
	Err                    any                          `json:"err"`
	Fee                    *uint64                      `json:"fee"`
	PreBalances            []uint64                     `json:"preBalances"`
	PostBalances           []uint64                     `json:"postBalances"`
	InnerInstructions      []SimulationInnerInstruction `json:"innerInstructions"`
	PreTokenBalances       []solana.TokenBalance        `json:"preTokenBalances"`
	PostTokenBalances      []solana.TokenBalance        `json:"postTokenBalances"`
	LoadedAccountsDataSize *uint32                      `json:"loadedAccountsDataSize"`
	LoadedAddresses        *solana.LoadedAddresses      `json:"loadedAddresses"`
	Logs                   []string                     `json:"logs"`
	ReplacementBlockhash   *solana.LatestBlockhash      `json:"replacementBlockhash"`
	ReturnData             *SimulationReturnData        `json:"returnData"`
	UnitsConsumed          *uint64                      `json:"unitsConsumed"`
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
