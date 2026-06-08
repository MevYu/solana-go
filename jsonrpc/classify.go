package jsonrpc

import (
	"errors"
	"strings"
)

// Solana JSON-RPC error codes. These are the numeric codes the server
// returns in the JSON-RPC error object; some are documented, most are
// folklore. The Is* classifiers below match against these codes first
// and fall back to message substring matching when the code is
// ambiguous or missing.
const (
	RPCErrCodeBlockCleanedUp             = -32001
	RPCErrCodeTransactionSimulationFail  = -32002
	RPCErrCodeSigVerifyFailed            = -32003
	RPCErrCodeBlockNotAvailable          = -32004
	RPCErrCodeNodeUnhealthy              = -32005
	RPCErrCodeSlotSkipped                = -32007
	RPCErrCodeNoSnapshot                 = -32008
	RPCErrCodeLongTermStorageSlotSkipped = -32009
	RPCErrCodeKeyExcluded                = -32010
	RPCErrCodeTransactionPrecompileFail  = -32011
	RPCErrCodeScanError                  = -32012
	RPCErrCodeTransactionHistoryOff      = -32013
	RPCErrCodeMinContextSlotNotReached   = -32016
)

// Sentinel errors for common Solana RPC failure modes. Use the Is*
// classifiers below rather than comparing errors.Is directly, because
// the classifiers also match *ErrRPC values and substring patterns.
var (
	// ErrBlockhashExpired indicates that a transaction's recent
	// blockhash is no longer valid. Refresh via GetLatestBlockhash
	// and rebuild the transaction.
	ErrBlockhashExpired = errors.New("solana: blockhash expired")

	// ErrInsufficientFunds indicates the payer's balance was too low
	// to cover the transaction fee or an instruction's lamport transfer.
	ErrInsufficientFunds = errors.New("solana: insufficient funds")

	// ErrRateLimited indicates the RPC node is throttling the caller.
	// Back off and retry, or switch endpoints.
	ErrRateLimited = errors.New("solana: rate limited")

	// ErrNodeBehind indicates the RPC node is behind the current
	// cluster slot and cannot satisfy the query at the requested
	// commitment level.
	ErrNodeBehind = errors.New("solana: node is behind the cluster")

	// ErrTransactionExpired indicates the transaction's blockhash
	// passed its LastValidBlockHeight before landing, and the cluster
	// will not process it. This is distinct from ErrBlockhashExpired
	// in timing: ErrBlockhashExpired is returned during send,
	// ErrTransactionExpired during confirm.
	ErrTransactionExpired = errors.New("solana: transaction expired")

	// ErrAccountNotFound indicates the queried account does not exist
	// at the requested commitment level.
	ErrAccountNotFound = errors.New("solana: account not found")

	// ErrSignatureNotFound indicates the queried transaction signature
	// is not known to the node. This can mean the transaction never
	// landed, or that the node does not have transaction history enabled.
	ErrSignatureNotFound = errors.New("solana: signature not found")

	// ErrSlotSkipped indicates the requested slot was skipped in the
	// leader schedule and no block was produced for it.
	ErrSlotSkipped = errors.New("solana: slot skipped")

	// ErrBlockCleanedUp indicates the requested block has been pruned
	// from the node's local storage and is no longer available.
	ErrBlockCleanedUp = errors.New("solana: block cleaned up")
)

// IsBlockhashExpired reports whether err indicates a stale blockhash.
// Use this in retry loops to decide whether to refresh the blockhash
// and rebuild the transaction.
func IsBlockhashExpired(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrBlockhashExpired) {
		return true
	}
	// ErrRPC.Error() embeds its Msg, so matching err.Error() also covers a
	// wrapped *ErrRPC — no separate errors.As(&ErrRPC) pass needed.
	return messageContainsAny(err.Error(),
		"BlockhashNotFound",
		"blockhash not found",
		"Blockhash not found",
	)
}

// IsInsufficientFunds reports whether err indicates the payer lacked
// the lamports required to cover fees or an instruction.
func IsInsufficientFunds(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrInsufficientFunds) {
		return true
	}
	return messageContainsAny(err.Error(),
		"InsufficientFundsForFee",
		"InsufficientFundsForRent",
		"insufficient funds",
		"Insufficient funds",
	)
}

// IsRateLimited reports whether err indicates the server is throttling
// the caller. HTTP 429 at the transport layer is also treated as
// rate-limiting for the classifier's purposes.
func IsRateLimited(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrRateLimited) {
		return true
	}
	return messageContainsAny(err.Error(),
		"rate limit",
		"Rate limit",
		"too many requests",
		"Too Many Requests",
		"HTTP 429",
	)
}

// IsNodeBehind reports whether err indicates the RPC node is behind
// the current cluster slot. This usually means the caller should pick
// a different endpoint or retry after a short wait.
func IsNodeBehind(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrNodeBehind) {
		return true
	}
	var rpcErr *ErrRPC
	if errors.As(err, &rpcErr) {
		if rpcErr.Code == RPCErrCodeNodeUnhealthy {
			return true
		}
	}
	return messageContainsAny(err.Error(),
		"Node is behind",
		"node is behind",
		"NodeUnhealthy",
	)
}

// IsTransactionExpired reports whether err indicates a transaction
// expired (passed its LastValidBlockHeight before landing). This is
// distinct from IsBlockhashExpired: the former happens during the
// confirm loop, the latter during the send call.
func IsTransactionExpired(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrTransactionExpired) {
		return true
	}
	return messageContainsAny(err.Error(),
		"TransactionExpired",
		"transaction expired",
	)
}

// IsAccountNotFound reports whether err indicates the queried account
// does not exist.
func IsAccountNotFound(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrAccountNotFound) {
		return true
	}
	return messageContainsAny(err.Error(),
		"AccountNotFound",
		"account not found",
	)
}

// IsSignatureNotFound reports whether err indicates the queried
// signature is not known to the node.
func IsSignatureNotFound(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrSignatureNotFound) {
		return true
	}
	return messageContainsAny(err.Error(),
		"SignatureNotFound",
		"signature not found",
	)
}

// IsSlotSkipped reports whether err indicates the requested slot was
// skipped in the leader schedule.
func IsSlotSkipped(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrSlotSkipped) {
		return true
	}
	var rpcErr *ErrRPC
	if errors.As(err, &rpcErr) {
		if rpcErr.Code == RPCErrCodeSlotSkipped || rpcErr.Code == RPCErrCodeLongTermStorageSlotSkipped {
			return true
		}
	}
	return messageContainsAny(err.Error(),
		"SlotSkipped",
		"slot was skipped",
	)
}

// IsBlockCleanedUp reports whether err indicates the requested block
// has been pruned from the node's storage.
func IsBlockCleanedUp(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrBlockCleanedUp) {
		return true
	}
	var rpcErr *ErrRPC
	if errors.As(err, &rpcErr) {
		if rpcErr.Code == RPCErrCodeBlockCleanedUp {
			return true
		}
	}
	return messageContainsAny(err.Error(),
		"BlockCleanedUp",
		"block was cleaned up",
	)
}

// messageContainsAny returns true if s contains any of the provided
// substrings.
func messageContainsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
