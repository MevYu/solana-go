package addresslookuptable

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/MevYu/solana-go"
)

// makeMetaHeader builds a 56-byte LookupTableMetaSize buffer.
func makeMetaHeader(deactivationSlot, lastExtendedSlot uint64, startIndex uint8, authority *solana.PublicKey) []byte {
	buf := make([]byte, LookupTableMetaSize)
	binary.LittleEndian.PutUint32(buf[0:4], 1) // discriminant = LookupTable
	binary.LittleEndian.PutUint64(buf[4:12], deactivationSlot)
	binary.LittleEndian.PutUint64(buf[12:20], lastExtendedSlot)
	buf[20] = startIndex
	if authority != nil {
		buf[21] = 1
		copy(buf[22:54], authority[:])
	}
	return buf
}

func TestDecodeTableState_TooShort(t *testing.T) {
	if _, err := DecodeTableState(make([]byte, LookupTableMetaSize-1)); err == nil {
		t.Error("expected error for short data")
	}
}

func TestDecodeTableState_BadDiscriminant(t *testing.T) {
	buf := make([]byte, LookupTableMetaSize)
	buf[0] = 2
	if _, err := DecodeTableState(buf); err == nil {
		t.Error("expected error for bad discriminant")
	}
}

func TestDecodeTableState_ActiveWithAuthority(t *testing.T) {
	auth := solana.PublicKey{0x01, 0x02}
	buf := makeMetaHeader(math.MaxUint64, 500, 10, &auth)

	ts, err := DecodeTableState(buf)
	if err != nil {
		t.Fatal(err)
	}
	if ts.DeactivationSlot != math.MaxUint64 {
		t.Errorf("DeactivationSlot = %d", ts.DeactivationSlot)
	}
	if ts.LastExtendedSlot != 500 {
		t.Errorf("LastExtendedSlot = %d", ts.LastExtendedSlot)
	}
	if ts.LastExtendedSlotStartIndex != 10 {
		t.Errorf("StartIndex = %d", ts.LastExtendedSlotStartIndex)
	}
	if ts.Authority == nil || *ts.Authority != auth {
		t.Errorf("Authority = %v, want %v", ts.Authority, auth)
	}
	if !ts.IsActive() {
		t.Error("table should be active")
	}
	if len(ts.Addresses) != 0 {
		t.Errorf("Addresses = %d, want 0", len(ts.Addresses))
	}
}

func TestDecodeTableState_FrozenNoAuthority(t *testing.T) {
	buf := makeMetaHeader(math.MaxUint64, 100, 0, nil)
	ts, err := DecodeTableState(buf)
	if err != nil {
		t.Fatal(err)
	}
	if ts.Authority != nil {
		t.Error("Authority should be nil (frozen)")
	}
}

func TestDecodeTableState_Deactivated(t *testing.T) {
	buf := makeMetaHeader(12345, 100, 0, nil)
	ts, err := DecodeTableState(buf)
	if err != nil {
		t.Fatal(err)
	}
	if ts.IsActive() {
		t.Error("table should not be active after deactivation")
	}
	if ts.DeactivationSlot != 12345 {
		t.Errorf("DeactivationSlot = %d", ts.DeactivationSlot)
	}
}

func TestDecodeTableState_WithAddresses(t *testing.T) {
	addr1 := solana.PublicKey{0xAA}
	addr2 := solana.PublicKey{0xBB}
	buf := makeMetaHeader(math.MaxUint64, 200, 0, nil)
	buf = append(buf, addr1[:]...)
	buf = append(buf, addr2[:]...)

	ts, err := DecodeTableState(buf)
	if err != nil {
		t.Fatal(err)
	}
	if len(ts.Addresses) != 2 {
		t.Fatalf("Addresses = %d, want 2", len(ts.Addresses))
	}
	if ts.Addresses[0] != addr1 {
		t.Errorf("Addresses[0] = %v, want %v", ts.Addresses[0], addr1)
	}
	if ts.Addresses[1] != addr2 {
		t.Errorf("Addresses[1] = %v, want %v", ts.Addresses[1], addr2)
	}
}

func TestDecodeTableState_NonMultipleOf32(t *testing.T) {
	buf := makeMetaHeader(math.MaxUint64, 0, 0, nil)
	buf = append(buf, make([]byte, 33)...)
	if _, err := DecodeTableState(buf); err == nil {
		t.Error("expected error for non-32-aligned address section")
	}
}
