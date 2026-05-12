package solana

import "encoding/json"

// TransactionMeta is the execution outcome of a Solana transaction,
// returned by RPC methods getTransaction and the block-level
// transaction listings. It carries the fee, the pre- and post-balances,
// any logs emitted, and (for transactions executed since recording was
// enabled) inner-instruction traces and token-balance snapshots.
type TransactionMeta struct {
	// Err is the raw JSON of the transaction-level error, or empty when
	// the transaction succeeded. The shape on the wire is the Solana
	// TransactionError enum: null, a plain string ("BlockhashNotFound"),
	// or an object such as {"InstructionError":[idx,...]}. Decode with
	// rpc.DecodeTransactionError, which accepts json.RawMessage.
	Err json.RawMessage `json:"err,omitempty"`

	// Fee is the fee charged for the transaction, in lamports.
	Fee uint64 `json:"fee"`

	// PreBalances and PostBalances hold the lamport balance of each
	// account in Message.AccountKeys (extended with ALT-loaded accounts)
	// immediately before and after the transaction executed.
	PreBalances  []uint64 `json:"preBalances"`
	PostBalances []uint64 `json:"postBalances"`

	// LogMessages is the ordered list of log lines emitted during
	// execution. Nil when log recording was disabled for this slot.
	LogMessages []string `json:"logMessages,omitempty"`

	// ComputeUnitsConsumed is the compute units actually used. Pointer
	// distinguishes "0 units" (rare but valid) from "field absent"
	// (older nodes that predate the field).
	ComputeUnitsConsumed *uint64 `json:"computeUnitsConsumed,omitempty"`

	// InnerInstructions is the per-top-level-instruction list of inner
	// instructions executed via CPI. Nil when inner-instruction
	// recording was disabled.
	InnerInstructions []InnerInstruction `json:"innerInstructions,omitempty"`

	// PreTokenBalances and PostTokenBalances are SPL Token account
	// balance snapshots immediately before and after execution. Nil
	// when token-balance recording was disabled.
	PreTokenBalances  []TokenBalance `json:"preTokenBalances,omitempty"`
	PostTokenBalances []TokenBalance `json:"postTokenBalances,omitempty"`

	// Status mirrors the legacy {"Ok":null} | {"Err":...} shape some
	// older nodes still emit alongside Err. Prefer Err for new code.
	Status *TxStatus `json:"status,omitempty"`

	// Rewards are validator/fee-payer reward distributions associated
	// with this transaction. Distinct from block-level rewards.
	Rewards []Reward `json:"rewards,omitempty"`

	// LoadedAddresses are the additional accounts a v0 transaction
	// resolved through Address Lookup Tables, split into writable and
	// read-only sets. Nil for legacy transactions.
	LoadedAddresses *LoadedAddresses `json:"loadedAddresses,omitempty"`
}

// TxStatus is the legacy {"Ok":...|"Err":...} shape that some Solana
// nodes still emit on TransactionMeta.Status. New code should read
// TransactionMeta.Err directly instead.
type TxStatus struct {
	Ok  any             `json:"Ok,omitempty"`
	Err json.RawMessage `json:"Err,omitempty"`
}

// InnerInstruction groups the inner (CPI-invoked) instructions that
// executed during a single top-level instruction.
type InnerInstruction struct {
	// Index is the position of the originating top-level instruction
	// within Message.Instructions.
	Index uint16 `json:"index"`

	// Instructions are the inner instructions in execution order.
	Instructions []CompiledInstruction `json:"instructions"`
}

// TokenBalance is a snapshot of an SPL Token account's balance at a
// transaction boundary, used in TransactionMeta.PreTokenBalances and
// TransactionMeta.PostTokenBalances.
type TokenBalance struct {
	// AccountIndex is the position of the token account in
	// Message.AccountKeys (extended with ALT-loaded accounts).
	AccountIndex uint16 `json:"accountIndex"`

	// Mint is the SPL Token mint that issued the balance.
	Mint PublicKey `json:"mint"`

	// Owner is the account that owns the token account. Absent on
	// older snapshots, which is why this is a value type with the
	// owner's zero-pubkey allowed.
	Owner PublicKey `json:"owner,omitempty"`

	// ProgramID is the SPL Token program (Token or Token-2022) the
	// account is associated with.
	ProgramID PublicKey `json:"programId,omitempty"`

	// UiTokenAmount is the balance, encoded as both raw amount and
	// decimal-adjusted UI form.
	UiTokenAmount UiTokenAmount `json:"uiTokenAmount"`
}

// UiTokenAmount is the dual raw + decimal-formatted token amount shape
// the Solana RPC returns for any SPL Token balance.
type UiTokenAmount struct {
	// Address is the token account address. Set only on responses where
	// the address is not implied by surrounding context (e.g.
	// getTokenLargestAccounts).
	Address *PublicKey `json:"address,omitempty"`

	// Amount is the raw integer amount as a base-10 string, ignoring
	// decimals. String form preserves precision beyond float64.
	Amount string `json:"amount"`

	// Decimals is the mint's decimals setting.
	Decimals uint8 `json:"decimals"`

	// UiAmount is Amount divided by 10**Decimals as a float. May lose
	// precision for very large amounts; use UiAmountString for display.
	UiAmount float64 `json:"uiAmount"`

	// UiAmountString is Amount divided by 10**Decimals as a decimal
	// string, preserving full precision.
	UiAmountString string `json:"uiAmountString"`
}

// LoadedAddresses splits the v0-transaction ALT-loaded accounts into
// writable and read-only sets. Both fields are nil for legacy
// transactions.
type LoadedAddresses struct {
	Writable []PublicKey `json:"writable,omitempty"`
	ReadOnly []PublicKey `json:"readonly,omitempty"`
}

// Reward is a single lamport credit or debit attributed to an account,
// used in both block-level rewards and transaction-meta rewards.
type Reward struct {
	// Pubkey is the account that received (or, for negative Lamports,
	// lost) the reward.
	Pubkey PublicKey `json:"pubkey"`

	// Lamports is the change in balance; signed because slashing
	// rewards are negative.
	Lamports int64 `json:"lamports"`

	// PostBalance is the account's balance after the reward, in
	// lamports.
	PostBalance uint64 `json:"postBalance"`

	// RewardType is the kind of reward — "fee", "rent", "voting",
	// "staking", or future variants.
	RewardType string `json:"rewardType,omitempty"`

	// Commission is the vote-account commission percentage in effect
	// at the time the reward was paid. Only set on voting/staking
	// rewards.
	Commission *uint8 `json:"commission,omitempty"`
}

// SignatureStatus is the status of a single transaction signature, as
// returned by getSignatureStatuses. Decode SignatureStatus.Err with
// rpc.DecodeTransactionError.
type SignatureStatus struct {
	// Slot is the slot the transaction landed in.
	Slot uint64 `json:"slot"`

	// Confirmations is the number of blocks confirmed since the
	// transaction landed. Nil once the transaction is rooted.
	Confirmations *uint64 `json:"confirmations"`

	// Err is the raw JSON of the transaction error, or empty on
	// success. Same shape as TransactionMeta.Err.
	Err json.RawMessage `json:"err,omitempty"`

	// ConfirmationStatus is one of "processed", "confirmed",
	// "finalized" — the commitment level the cluster has applied.
	ConfirmationStatus string `json:"confirmationStatus,omitempty"`
}

// PrioritizationFee is one data point in the response to
// getRecentPrioritizationFees: the per-compute-unit fee, in
// micro-lamports, paid by at least one successfully landed
// transaction in the given slot.
type PrioritizationFee struct {
	Slot              uint64 `json:"slot"`
	PrioritizationFee uint64 `json:"prioritizationFee"`
}

// LatestBlockhash is the decoded value of getLatestBlockhash: the
// most recent blockhash and the last block height for which the
// blockhash will remain valid for use in a transaction.
//
// Slot is populated from the JSON-RPC context envelope after decode
// (rpc.Client.GetLatestBlockhash does this), so it has no json tag.
type LatestBlockhash struct {
	Slot                 uint64 `json:"-"`
	Blockhash            Hash   `json:"blockhash"`
	LastValidBlockHeight uint64 `json:"lastValidBlockHeight"`
}
