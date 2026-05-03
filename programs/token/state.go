package token

import (
	"fmt"

	solana "github.com/MevYu/solana-go"
	"github.com/MevYu/solana-go/encoding"
)

// MintState is the on-chain state of an SPL Token mint account. The
// binary layout follows the SPL Token program's Mint struct (82 bytes
// for a standard non-extended mint).
type MintState struct {
	// MintAuthority is the optional authority that can mint new tokens.
	// Nil if the mint authority has been permanently disabled.
	MintAuthority *solana.PublicKey
	// Supply is the total number of tokens in circulation (base units).
	Supply uint64
	// Decimals is the number of base-10 digits to the right of the decimal.
	Decimals uint8
	// IsInitialized is true after the mint has been initialized.
	IsInitialized bool
	// FreezeAuthority is the optional authority that can freeze token accounts.
	FreezeAuthority *solana.PublicKey
}

// DecodeMintState decodes a raw SPL Token Mint account data buffer.
// The buffer must be at least 82 bytes long.
func DecodeMintState(data []byte) (*MintState, error) {
	if len(data) < 82 {
		return nil, fmt.Errorf("token: DecodeMintState: data too short (%d < 82 bytes)", len(data))
	}
	r := encoding.NewReader(data[:82])
	m := &MintState{
		MintAuthority:   solana.ReadOptionalPubkey(r),
		Supply:          r.U64(),
		Decimals:        r.U8(),
		IsInitialized:   r.Bool(),
		FreezeAuthority: solana.ReadOptionalPubkey(r),
	}
	if err := r.Err(); err != nil {
		return nil, fmt.Errorf("token: DecodeMintState: %w", err)
	}
	return m, nil
}

// AccountState is the on-chain state of an SPL Token account (token wallet).
type AccountState struct {
	// Mint is the address of the mint this account holds tokens of.
	Mint solana.PublicKey
	// Owner is the wallet address that controls this token account.
	Owner solana.PublicKey
	// Amount is the number of tokens held (base units).
	Amount uint64
	// Delegate is the optional delegate authority.
	Delegate *solana.PublicKey
	// State is the account state: 0=Uninitialized, 1=Initialized, 2=Frozen.
	State uint8
	// IsNative is set if the token account holds wrapped SOL. When
	// non-nil, it records the rent-exempt reserve.
	IsNative *uint64
	// DelegatedAmount is the amount approved to the delegate.
	DelegatedAmount uint64
	// CloseAuthority is the optional close authority.
	CloseAuthority *solana.PublicKey
}

// DecodeAccountState decodes a raw SPL Token account data buffer.
// The buffer must be at least 165 bytes long.
func DecodeAccountState(data []byte) (*AccountState, error) {
	if len(data) < 165 {
		return nil, fmt.Errorf("token: DecodeAccountState: data too short (%d < 165 bytes)", len(data))
	}
	r := encoding.NewReader(data[:165])
	a := &AccountState{
		Mint:     solana.PublicKey(r.Bytes32()),
		Owner:    solana.PublicKey(r.Bytes32()),
		Amount:   r.U64(),
		Delegate: solana.ReadOptionalPubkey(r),
		State:    r.U8(),
	}

	// COption<u64> for IsNative: u32 tag + unconditional u64 slot.
	isNativeTag := r.U32()
	isNativeVal := r.U64()
	if isNativeTag == 1 {
		a.IsNative = &isNativeVal
	}

	a.DelegatedAmount = r.U64()
	a.CloseAuthority = solana.ReadOptionalPubkey(r)

	if err := r.Err(); err != nil {
		return nil, fmt.Errorf("token: DecodeAccountState: %w", err)
	}
	return a, nil
}
