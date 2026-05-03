package solana

// LAMPORTS_PER_SOL is the number of lamports in one SOL.
// Use this to convert between human-readable SOL amounts and the
// on-chain lamport denomination: 1 SOL = 1,000,000,000 lamports.
const LAMPORTS_PER_SOL = uint64(1_000_000_000)

// Well-known program addresses as PublicKey values. These are the
// canonical addresses for Solana's native and core programs.
var (
	// SystemProgramID is the System Program, which handles SOL transfers,
	// account creation, and program deployments.
	SystemProgramID = MustPublicKey("11111111111111111111111111111111")

	// TokenProgramID is the SPL Token program for the classic token standard.
	TokenProgramID = MustPublicKey("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")

	// Token2022ProgramID is the SPL Token-2022 program (extensions).
	Token2022ProgramID = MustPublicKey("TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb")

	// AssociatedTokenProgramID is the Associated Token Account program.
	AssociatedTokenProgramID = MustPublicKey("ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJe1bRS")

	// ComputeBudgetProgramID is the Compute Budget program for priority fees.
	ComputeBudgetProgramID = MustPublicKey("ComputeBudget111111111111111111111111111111")

	// SysvarRentPubkey is the address of the Rent sysvar account.
	SysvarRentPubkey = MustPublicKey("SysvarRent111111111111111111111111111111111")

	// SysvarClockPubkey is the address of the Clock sysvar account.
	SysvarClockPubkey = MustPublicKey("SysvarC1ock11111111111111111111111111111111")

	// SysvarRecentBlockhashesPubkey is the address of the RecentBlockhashes sysvar.
	SysvarRecentBlockhashesPubkey = MustPublicKey("SysvarRecentB1ockHashes11111111111111111111")

	// SysvarSlotHashesPubkey is the address of the SlotHashes sysvar.
	SysvarSlotHashesPubkey = MustPublicKey("SysvarS1otHashes111111111111111111111111111")

	// SysvarSlotHistoryPubkey is the address of the SlotHistory sysvar.
	SysvarSlotHistoryPubkey = MustPublicKey("SysvarS1otHistory11111111111111111111111111")

	// SysvarStakeHistoryPubkey is the address of the StakeHistory sysvar.
	SysvarStakeHistoryPubkey = MustPublicKey("SysvarStakeHistory1111111111111111111111111")

	// SysvarEpochSchedulePubkey is the address of the EpochSchedule sysvar.
	SysvarEpochSchedulePubkey = MustPublicKey("SysvarEpochSchedu1e111111111111111111111111")

	// SysvarInstructionsPubkey is the address of the Instructions sysvar,
	// used to inspect other instructions in the same transaction.
	SysvarInstructionsPubkey = MustPublicKey("Sysvar1nstructions1111111111111111111111111")

	// SysvarEpochRewardsPubkey is the address of the EpochRewards sysvar.
	SysvarEpochRewardsPubkey = MustPublicKey("SysvarEpochRewards1111111111111111111111111")
)
