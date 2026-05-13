package rpc

import solana "github.com/MevYu/solana-go"

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
	Slot          uint64                `json:"-"`
	Err           any                   `json:"err"`
	Logs          []string              `json:"logs"`
	UnitsConsumed *uint64               `json:"unitsConsumed"`
	Accounts      []*solana.AccountInfo `json:"accounts"`
	ReturnData    *SimulationReturnData `json:"returnData"`
}

// BlockTransaction is a single entry in the transactions array of a
// GetBlock response. Transaction is fully decoded via
// Transaction.UnmarshalJSON when the encoding is base64 / base58 /
// base64+zstd.
type BlockTransaction struct {
	Transaction *solana.Transaction     `json:"transaction"`
	Meta        *solana.TransactionMeta `json:"meta"`
	Version     any                     `json:"version"`
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
