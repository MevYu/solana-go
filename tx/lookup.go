package tx

import (
	"context"
	"fmt"

	solana "github.com/MevYu/solana-go"
)

// AddressLookupTable associates an on-chain lookup table account with the
// indices of accounts within it that this transaction references. This is
// the wire-format representation used inside a compiled v0 Message; to
// build a v0 transaction from scratch use LoadedAddressLookupTable instead.
type AddressLookupTable struct {
	AccountKey      solana.PublicKey
	WritableIndexes []uint8
	ReadonlyIndexes []uint8
}

// LoadedAddressLookupTable is a type alias for solana.LoadedAddressLookupTable.
// It holds the full resolved addresses for a lookup table and is the input
// type for Builder.BuildV0.
type LoadedAddressLookupTable = solana.LoadedAddressLookupTable

// BuildV0 compiles the builder's instructions and the provided address
// lookup tables into a v0 versioned Message, constructs a Transaction,
// signs it with all registered signers, and returns the result.
//
// tables must contain the fully resolved ALT data (key + all addresses
// stored in the table). The builder maps instruction accounts to table
// slots automatically: any non-signer account found in a table is routed
// through that table; all signer accounts and program IDs remain static.
//
// Use BuildV0 instead of Build when the transaction references more
// accounts than the 35-account legacy limit, or when you need to save
// transaction bytes by referencing large shared account sets.
func (b *Builder) BuildV0(ctx context.Context, tables []LoadedAddressLookupTable) (*solana.Transaction, error) {
	if b.err != nil {
		return nil, b.err
	}
	if b.feePayer == (solana.PublicKey{}) {
		return nil, fmt.Errorf("solana/tx: fee payer not set")
	}
	if b.blockhash == (solana.Hash{}) {
		return nil, fmt.Errorf("solana/tx: recent blockhash not set")
	}
	if len(b.instructions) == 0 {
		return nil, fmt.Errorf("solana/tx: no instructions")
	}

	msg, err := solana.NewMessageV0(b.feePayer, b.instructions, b.blockhash, tables)
	if err != nil {
		return nil, fmt.Errorf("solana/tx: compile v0 message: %w", err)
	}

	transaction := solana.NewTransaction(*msg)
	if len(b.signers) > 0 {
		if err := transaction.Sign(ctx, b.signers...); err != nil {
			return nil, fmt.Errorf("solana/tx: sign: %w", err)
		}
	}
	return transaction, nil
}

// ToMessageAddressTableLookup converts an AddressLookupTable to the
// solana.MessageAddressTableLookup wire type.
func ToMessageAddressTableLookup(t AddressLookupTable) solana.MessageAddressTableLookup {
	return solana.MessageAddressTableLookup{
		AccountKey:      t.AccountKey,
		WritableIndexes: t.WritableIndexes,
		ReadonlyIndexes: t.ReadonlyIndexes,
	}
}
