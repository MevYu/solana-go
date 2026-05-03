package token2022

import (
	"encoding/binary"
	"fmt"

	solana "github.com/MevYu/solana-go"
	"github.com/MevYu/solana-go/programs/token"
)

// Extension type IDs for Token-2022 (spl-token-2022 ExtensionType enum).
const (
	ExtTransferFeeConfig     uint16 = 1
	ExtTransferFeeAmount     uint16 = 2
	ExtMintCloseAuthority    uint16 = 3
	ExtDefaultAccountState   uint16 = 6
	ExtImmutableOwner        uint16 = 7
	ExtMemoTransfer          uint16 = 8
	ExtNonTransferable       uint16 = 9
	ExtInterestBearingConfig uint16 = 10
	ExtCpiGuard              uint16 = 11
	ExtPermanentDelegate     uint16 = 12
	ExtMetadataPointer       uint16 = 16
	ExtTransferHook          uint16 = 18
)

// MintState is the decoded state of a Token-2022 mint account. The base
// fields are identical to the classic SPL Token mint layout (82 bytes);
// extensions are parsed from the TLV region that follows.
type MintState struct {
	MintAuthority   *solana.PublicKey
	Supply          uint64
	Decimals        uint8
	IsInitialized   bool
	FreezeAuthority *solana.PublicKey
	Extensions      MintExtensions
}

// MintExtensions holds the decoded extension data for a Token-2022 mint.
// Fields are nil when the extension is absent.
type MintExtensions struct {
	CloseAuthority    *solana.PublicKey
	DefaultState      *uint8
	NonTransferable   bool
	PermanentDelegate *solana.PublicKey
	TransferFeeConfig *TransferFeeConfig
	InterestBearing   *InterestBearingConfig
	MetadataPointer   *MetadataPointerConfig
	TransferHook      *TransferHookConfig
}

// TransferFeeConfig holds the transfer-fee extension data for a mint.
type TransferFeeConfig struct {
	TransferFeeConfigAuthority *solana.PublicKey
	WithdrawWithheldAuthority  *solana.PublicKey
	WithheldAmount             uint64
	OlderTransferFee           TransferFee
	NewerTransferFee           TransferFee
}

// TransferFee holds per-epoch fee parameters.
type TransferFee struct {
	Epoch                  uint64
	MaximumFee             uint64
	TransferFeeBasisPoints uint16
}

// InterestBearingConfig holds interest-bearing-mint extension data.
type InterestBearingConfig struct {
	RateAuthority           *solana.PublicKey
	InitializationTimestamp int64
	PreUpdateAverageRate    int16
	LastUpdateTimestamp     int64
	CurrentRate             int16
}

// MetadataPointerConfig holds the metadata-pointer extension data.
type MetadataPointerConfig struct {
	Authority       *solana.PublicKey
	MetadataAddress *solana.PublicKey
}

// TransferHookConfig holds the transfer-hook extension data.
type TransferHookConfig struct {
	Authority *solana.PublicKey
	ProgramID *solana.PublicKey
}

// Account is the decoded state of a Token-2022 token account. Base fields
// are identical to the classic SPL Token account layout (165 bytes).
//
// Named Account (not AccountState) because the package already exports an
// AccountState uint8 for the frozen/thawed enum used by the
// DefaultAccountState extension.
type Account struct {
	Mint            solana.PublicKey
	Owner           solana.PublicKey
	Amount          uint64
	Delegate        *solana.PublicKey
	State           uint8
	IsNative        *uint64
	DelegatedAmount uint64
	CloseAuthority  *solana.PublicKey
	Extensions      AccountExtensions
}

// AccountExtensions holds the decoded extension data for a Token-2022 account.
type AccountExtensions struct {
	WithheldAmount *uint64
	MemoTransfer   *bool
	ImmutableOwner bool
	CpiGuard       *bool
}

// DecodeMintState decodes a raw Token-2022 mint account data buffer.
// The buffer must be at least 82 bytes; extension TLV data is parsed
// when additional bytes are present.
//
// The first 82 bytes are byte-identical to a classic SPL Token mint, so
// this function delegates to token.DecodeMintState for that header and
// only owns the Token-2022-specific TLV section.
func DecodeMintState(data []byte) (*MintState, error) {
	base, err := token.DecodeMintState(data)
	if err != nil {
		return nil, fmt.Errorf("token2022: DecodeMintState: %w", err)
	}
	m := &MintState{
		MintAuthority:   base.MintAuthority,
		Supply:          base.Supply,
		Decimals:        base.Decimals,
		IsInitialized:   base.IsInitialized,
		FreezeAuthority: base.FreezeAuthority,
	}
	// Byte 165 is the AccountType discriminator (1=Mint); TLV begins at 166.
	if len(data) > 166 {
		for extType, raw := range parseTLV(data[166:]) {
			decodeMintExt(&m.Extensions, extType, raw)
		}
	}
	return m, nil
}

// DecodeAccount decodes a raw Token-2022 token account data buffer.
// The buffer must be at least 165 bytes; extension TLV data is parsed
// when additional bytes are present.
//
// The first 165 bytes are byte-identical to a classic SPL Token account,
// so this function delegates to token.DecodeAccountState for that header
// and only owns the Token-2022-specific TLV section.
func DecodeAccount(data []byte) (*Account, error) {
	base, err := token.DecodeAccountState(data)
	if err != nil {
		return nil, fmt.Errorf("token2022: DecodeAccount: %w", err)
	}
	a := &Account{
		Mint:            base.Mint,
		Owner:           base.Owner,
		Amount:          base.Amount,
		Delegate:        base.Delegate,
		State:           base.State,
		IsNative:        base.IsNative,
		DelegatedAmount: base.DelegatedAmount,
		CloseAuthority:  base.CloseAuthority,
	}
	// Byte 165 is the AccountType discriminator (2=Account); TLV begins at 166.
	if len(data) > 166 {
		for extType, raw := range parseTLV(data[166:]) {
			decodeAccountExt(&a.Extensions, extType, raw)
		}
	}
	return a, nil
}

// parseTLV parses Token-2022 TLV extension bytes into a map of
// extension-type → raw bytes. Stops at the first uninitialized or
// truncated entry.
func parseTLV(data []byte) map[uint16][]byte {
	result := make(map[uint16][]byte, 4)
	for len(data) >= 4 {
		extType := binary.LittleEndian.Uint16(data[0:2])
		extLen := int(binary.LittleEndian.Uint16(data[2:4]))
		if extType == 0 || 4+extLen > len(data) {
			break
		}
		result[extType] = data[4 : 4+extLen]
		data = data[4+extLen:]
	}
	return result
}

// optPubkey decodes a 32-byte field where all-zeros means "no authority"
// (Token-2022 OptionalNonZeroPubkey encoding).
func optPubkey(b []byte) *solana.PublicKey {
	if len(b) < solana.PublicKeySize {
		return nil
	}
	var zero solana.PublicKey
	if solana.PublicKey(b[:solana.PublicKeySize]) == zero {
		return nil
	}
	pk := solana.PublicKey{}
	copy(pk[:], b[:solana.PublicKeySize])
	return &pk
}

func decodeMintExt(ext *MintExtensions, t uint16, raw []byte) {
	switch t {
	case ExtMintCloseAuthority:
		if len(raw) >= solana.PublicKeySize {
			ext.CloseAuthority = optPubkey(raw)
		}
	case ExtDefaultAccountState:
		if len(raw) >= 1 {
			s := raw[0]
			ext.DefaultState = &s
		}
	case ExtNonTransferable:
		ext.NonTransferable = true
	case ExtPermanentDelegate:
		if len(raw) >= solana.PublicKeySize {
			ext.PermanentDelegate = optPubkey(raw)
		}
	case ExtTransferFeeConfig:
		if len(raw) >= 108 {
			ext.TransferFeeConfig = &TransferFeeConfig{
				TransferFeeConfigAuthority: optPubkey(raw[0:32]),
				WithdrawWithheldAuthority:  optPubkey(raw[32:64]),
				WithheldAmount:             binary.LittleEndian.Uint64(raw[64:72]),
				OlderTransferFee: TransferFee{
					Epoch:                  binary.LittleEndian.Uint64(raw[72:80]),
					MaximumFee:             binary.LittleEndian.Uint64(raw[80:88]),
					TransferFeeBasisPoints: binary.LittleEndian.Uint16(raw[88:90]),
				},
				NewerTransferFee: TransferFee{
					Epoch:                  binary.LittleEndian.Uint64(raw[90:98]),
					MaximumFee:             binary.LittleEndian.Uint64(raw[98:106]),
					TransferFeeBasisPoints: binary.LittleEndian.Uint16(raw[106:108]),
				},
			}
		}
	case ExtInterestBearingConfig:
		if len(raw) >= 52 {
			ext.InterestBearing = &InterestBearingConfig{
				RateAuthority:           optPubkey(raw[0:32]),
				InitializationTimestamp: int64(binary.LittleEndian.Uint64(raw[32:40])),
				PreUpdateAverageRate:    int16(binary.LittleEndian.Uint16(raw[40:42])),
				LastUpdateTimestamp:     int64(binary.LittleEndian.Uint64(raw[42:50])),
				CurrentRate:             int16(binary.LittleEndian.Uint16(raw[50:52])),
			}
		}
	case ExtMetadataPointer:
		if len(raw) >= 64 {
			ext.MetadataPointer = &MetadataPointerConfig{
				Authority:       optPubkey(raw[0:32]),
				MetadataAddress: optPubkey(raw[32:64]),
			}
		}
	case ExtTransferHook:
		if len(raw) >= 64 {
			ext.TransferHook = &TransferHookConfig{
				Authority: optPubkey(raw[0:32]),
				ProgramID: optPubkey(raw[32:64]),
			}
		}
	}
}

func decodeAccountExt(ext *AccountExtensions, t uint16, raw []byte) {
	switch t {
	case ExtTransferFeeAmount:
		if len(raw) >= 8 {
			v := binary.LittleEndian.Uint64(raw[0:8])
			ext.WithheldAmount = &v
		}
	case ExtMemoTransfer:
		if len(raw) >= 1 {
			b := raw[0] != 0
			ext.MemoTransfer = &b
		}
	case ExtImmutableOwner:
		ext.ImmutableOwner = true
	case ExtCpiGuard:
		if len(raw) >= 1 {
			b := raw[0] != 0
			ext.CpiGuard = &b
		}
	}
}
