// Package tx provides a fluent transaction builder for Solana.
// It assembles Instructions into a signed Transaction without requiring
// any network access: blockhash and signers are provided by the caller.
//
// Example:
//
//	tx, err := tx.NewBuilder().
//	    SetFeePayer(payer.PublicKey()).
//	    Add(systemProgram.Transfer(...)).
//	    Add(memoProgram.Log("hello")).
//	    SetRecentBlockhash(bh).
//	    Sign(ctx, payer).
//	    Build()
package tx

import (
	"context"
	"fmt"

	solana "github.com/MevYu/solana-go"
)

// Builder is a fluent transaction assembler. All mutating methods
// return the receiver so calls can be chained. Build finalises the
// transaction; after Build returns the Builder must not be reused.
type Builder struct {
	feePayer     solana.PublicKey
	instructions []solana.Instruction
	blockhash    solana.Hash
	signers      []solana.Signer
	err          error
}

// NewBuilder returns an empty Builder.
func NewBuilder() *Builder { return &Builder{} }

// SetFeePayer sets the transaction fee payer. The fee payer is the
// first signer and its public key is placed first in the account
// keys list.
func (b *Builder) SetFeePayer(payer solana.PublicKey) *Builder {
	b.feePayer = payer
	return b
}

// Add appends an instruction to the transaction. Instructions are
// executed in the order they are added.
func (b *Builder) Add(ix solana.Instruction) *Builder {
	if ix == nil {
		b.err = fmt.Errorf("solana/tx: nil instruction")
		return b
	}
	b.instructions = append(b.instructions, ix)
	return b
}

// SetRecentBlockhash sets the recent blockhash the transaction
// commits to. This must be set before calling Build.
func (b *Builder) SetRecentBlockhash(bh solana.Hash) *Builder {
	b.blockhash = bh
	return b
}

// Sign registers signers that will be asked to sign the serialized
// message when Build is called. Multiple Sign calls accumulate signers.
func (b *Builder) Sign(signers ...solana.Signer) *Builder {
	b.signers = append(b.signers, signers...)
	return b
}

// Build compiles the instructions into a Message, constructs the
// Transaction, calls Sign on all registered signers, and returns the
// result. ctx is forwarded to each signer's Sign method.
//
// Build returns an error if:
//   - an instruction was nil (recorded by Add)
//   - the blockhash is zero
//   - the fee payer is zero
//   - message compilation fails (e.g. too many accounts)
//   - any signer returns an error
func (b *Builder) Build(ctx context.Context) (*solana.Transaction, error) {
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

	msg, err := solana.NewMessage(b.feePayer, b.instructions, b.blockhash)
	if err != nil {
		return nil, fmt.Errorf("solana/tx: compile message: %w", err)
	}

	transaction := solana.NewTransaction(*msg)
	if len(b.signers) > 0 {
		if err := transaction.Sign(ctx, b.signers...); err != nil {
			return nil, fmt.Errorf("solana/tx: sign: %w", err)
		}
	}
	return transaction, nil
}
