package addresslookuptable

import (
	"fmt"
	"math"

	solana "github.com/MevYu/solana-go"
	"github.com/MevYu/solana-go/encoding"
)

// LookupTableMetaSize is the fixed byte length of the serialized
// LookupTableMeta header, matching the Solana runtime constant.
const LookupTableMetaSize = 56

// TableState holds the decoded on-chain state of an Address Lookup Table.
type TableState struct {
	// DeactivationSlot is set to math.MaxUint64 (u64::MAX) while the table
	// is active. Once NewDeactivateLookupTable is called it is set to the
	// deactivation slot; the table cannot be closed until that slot is no
	// longer recent.
	DeactivationSlot uint64

	// LastExtendedSlot is the slot during which the table was last extended.
	LastExtendedSlot uint64

	// LastExtendedSlotStartIndex is the first index added during the last
	// extend operation.
	LastExtendedSlotStartIndex uint8

	// Authority is the account that may sign Extend, Freeze, Deactivate,
	// and Close. Nil when the table has been frozen (permanently immutable).
	Authority *solana.PublicKey

	// Addresses is the ordered list of public keys stored in the table.
	// Transactions reference entries by their 0-based index.
	Addresses []solana.PublicKey
}

// IsActive reports whether the lookup table is active (not yet deactivated).
func (t *TableState) IsActive() bool {
	return t.DeactivationSlot == math.MaxUint64
}

// DecodeTableState decodes the raw account data of an Address Lookup Table.
// The buffer must be at least LookupTableMetaSize (56) bytes.
//
// Layout:
//
//	[0:4]   discriminant u32 (must be 1 for LookupTable)
//	[4:12]  deactivation_slot u64
//	[12:20] last_extended_slot u64
//	[20]    last_extended_slot_start_index u8
//	[21]    authority tag u8 (0=None, 1=Some)
//	[22:54] authority pubkey (32 B; only valid when tag==1)
//	[54:56] padding u16
//	[56:]   addresses (each 32 B)
func DecodeTableState(data []byte) (*TableState, error) {
	if len(data) < LookupTableMetaSize {
		return nil, fmt.Errorf("addresslookuptable: data too short (%d < %d)", len(data), LookupTableMetaSize)
	}

	r := encoding.NewReader(data[:LookupTableMetaSize])
	if disc := r.U32(); disc != 1 {
		return nil, fmt.Errorf("addresslookuptable: unexpected discriminant %d (want 1)", disc)
	}

	ts := &TableState{
		DeactivationSlot:           r.U64(),
		LastExtendedSlot:           r.U64(),
		LastExtendedSlotStartIndex: r.U8(),
	}
	authTag := r.U8()
	authority := solana.PublicKey(r.Bytes32())
	if authTag == 1 {
		ts.Authority = &authority
	}
	r.Skip(2) // padding u16

	if err := r.Err(); err != nil {
		return nil, fmt.Errorf("addresslookuptable: %w", err)
	}

	tail := data[LookupTableMetaSize:]
	if len(tail)%solana.PublicKeySize != 0 {
		return nil, fmt.Errorf("addresslookuptable: address section length %d not a multiple of %d", len(tail), solana.PublicKeySize)
	}
	ts.Addresses = make([]solana.PublicKey, len(tail)/solana.PublicKeySize)
	for i := range ts.Addresses {
		copy(ts.Addresses[i][:], tail[i*solana.PublicKeySize:(i+1)*solana.PublicKeySize])
	}
	return ts, nil
}
