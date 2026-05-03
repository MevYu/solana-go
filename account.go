package solana

// Account describes the state of an on-chain Solana account as
// returned by JSON-RPC methods such as getAccountInfo.
type Account struct {
	// Lamports is the account balance, denominated in lamports
	// (10^-9 SOL).
	Lamports uint64
	// Owner is the program that controls the account. Only the owner
	// program may debit lamports or modify Data.
	Owner PublicKey
	// Data is the raw account data bytes. A zero-length slice means
	// the account exists but holds no program-specific data.
	Data []byte
	// Executable reports whether the account stores a loadable
	// program. Executable accounts are immutable after finalisation.
	Executable bool
	// RentEpoch is the epoch at which the account next owes rent.
	// It is 0 on rent-exempt accounts, the modern default.
	RentEpoch uint64
}
