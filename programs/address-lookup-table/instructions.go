package addresslookuptable

import (
	"encoding/binary"
	"fmt"

	"github.com/MevYu/solana-go"
	"github.com/MevYu/solana-go/encoding"
	"github.com/MevYu/solana-go/programs/system"
)

// Instruction discriminator tags. ALT uses u32 tags (Borsh enum
// discriminator form), unlike SPL Token which uses single-byte
// tags.
const (
	tagCreateLookupTable     uint32 = 0
	tagFreezeLookupTable     uint32 = 1
	tagExtendLookupTable     uint32 = 2
	tagDeactivateLookupTable uint32 = 3
	tagCloseLookupTable      uint32 = 4
)

// DeriveLookupTableAddress computes the program-derived address for a
// lookup table owned by authority at recentSlot. The PDA is the table's
// on-chain address; pass it back as the first account meta to every
// subsequent ALT instruction referring to this table.
//
// The seed order is [authority, recentSlot as u64 LE bytes], matching
// the Solana runtime's derive_lookup_table_address.
func DeriveLookupTableAddress(authority solana.PublicKey, recentSlot uint64) (solana.PublicKey, uint8, error) {
	slotBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(slotBytes, recentSlot)
	seeds := [][]byte{authority[:], slotBytes}
	return solana.FindProgramAddress(seeds, ProgramID)
}

// NewCreateLookupTable builds an instruction that allocates a new address
// lookup table owned by authority. recentSlot must be a slot the runtime
// still considers "recent" — usually the current slot from client.GetSlot,
// or one of the last few.
//
// The derived table address is returned alongside the instruction so
// callers can immediately reference it in subsequent extend / freeze
// calls. Both authority and payer must sign the transaction.
func NewCreateLookupTable(authority, payer solana.PublicKey, recentSlot uint64) (solana.Instruction, solana.PublicKey, error) {
	table, bump, err := DeriveLookupTableAddress(authority, recentSlot)
	if err != nil {
		return nil, solana.PublicKey{}, fmt.Errorf("alt: CreateLookupTable: derive address: %w", err)
	}
	return solana.NewInstruction(ProgramID, []*solana.AccountMeta{
		solana.NewAccountMeta(table, false, true),
		solana.NewAccountMeta(authority, true, false),
		solana.NewAccountMeta(payer, true, true),
		solana.NewAccountMeta(system.ProgramID, false, false),
	}, encoding.NewEncoder(13).
		U32(tagCreateLookupTable).
		U64(recentSlot).
		U8(bump).
		Bytes()), table, nil
}

// NewFreezeLookupTable builds an instruction that freezes a lookup table,
// preventing any further extends. Only the table's authority can freeze
// it; freezing is permanent.
func NewFreezeLookupTable(lookupTable, authority solana.PublicKey) solana.Instruction {
	return solana.NewInstruction(ProgramID, []*solana.AccountMeta{
		solana.NewAccountMeta(lookupTable, false, true),
		solana.NewAccountMeta(authority, true, false),
	}, encoding.NewEncoder(4).U32(tagFreezeLookupTable).Bytes())
}

// NewExtendLookupTable builds an instruction that appends new addresses
// to a lookup table. Both authority and payer must sign; payer covers
// any additional rent required by the larger account.
//
// A lookup table can hold up to 256 addresses total. Calling Extend when
// the table is already at the limit will fail.
func NewExtendLookupTable(lookupTable, authority, payer solana.PublicKey, newAddresses []solana.PublicKey) solana.Instruction {
	// Data: [u32 tag, u64 count, addr×count]. The ALT program is
	// serialized with bincode (new_with_bincode), so Vec<Pubkey> carries
	// a u64 length prefix — not Borsh's u32.
	e := encoding.NewEncoder(4 + 8 + solana.PublicKeySize*len(newAddresses)).
		U32(tagExtendLookupTable).
		U64(uint64(len(newAddresses)))
	for _, a := range newAddresses {
		e.Raw(a[:])
	}
	return solana.NewInstruction(ProgramID, []*solana.AccountMeta{
		solana.NewAccountMeta(lookupTable, false, true),
		solana.NewAccountMeta(authority, true, false),
		solana.NewAccountMeta(payer, true, true),
		solana.NewAccountMeta(system.ProgramID, false, false),
	}, e.Bytes())
}

// NewDeactivateLookupTable builds an instruction that marks a lookup
// table as deactivated. Deactivation does not immediately close the
// account; the table enters a cooling-off period (currently ~500 slots)
// during which it cannot be referenced by new transactions but its rent
// is still held.
func NewDeactivateLookupTable(lookupTable, authority solana.PublicKey) solana.Instruction {
	return solana.NewInstruction(ProgramID, []*solana.AccountMeta{
		solana.NewAccountMeta(lookupTable, false, true),
		solana.NewAccountMeta(authority, true, false),
	}, encoding.NewEncoder(4).U32(tagDeactivateLookupTable).Bytes())
}

// NewCloseLookupTable builds an instruction that closes a deactivated
// lookup table and transfers its rent lamports to recipient. The table
// must be past the deactivation cooling-off period or the runtime will
// reject the call.
func NewCloseLookupTable(lookupTable, authority, recipient solana.PublicKey) solana.Instruction {
	return solana.NewInstruction(ProgramID, []*solana.AccountMeta{
		solana.NewAccountMeta(lookupTable, false, true),
		solana.NewAccountMeta(authority, true, false),
		solana.NewAccountMeta(recipient, false, true),
	}, encoding.NewEncoder(4).U32(tagCloseLookupTable).Bytes())
}
