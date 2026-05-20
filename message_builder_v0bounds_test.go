package solana

import "testing"

// Instruction account indices are u8. When static + ALT-resolved accounts
// exceed 256, compileInstructions' uint8(idx) would silently truncate
// indices >=256 into wrong accounts; NewMessageV0 must reject instead.
func TestNewMessageV0_TotalAccountsOver256Errors(t *testing.T) {
	payer := PublicKey{0x01}
	prog := PublicKey{0x99}
	altKey := PublicKey{0xA0}

	// 257 distinct non-signer accounts, all resolvable via one ALT.
	// Static keys stay at 2 (payer + program); the overflow comes from
	// the table-resolved accounts, exercising the post-resolution check.
	const n = 257
	addrs := make([]PublicKey, n)
	metas := []*AccountMeta{NewAccountMeta(payer, true, true)}
	for i := range addrs {
		addrs[i] = PublicKey{0x10, byte(i), byte(i >> 8)}
		metas = append(metas, NewAccountMeta(addrs[i], false, false))
	}
	ix := &testInstruction{pid: prog, meta: metas, data: []byte{0x01}}
	tables := []LoadedAddressLookupTable{{AccountKey: altKey, Addresses: addrs}}

	if _, err := NewMessageV0(payer, []Instruction{ix}, Hash{0x42}, tables); err == nil {
		t.Fatal("expected error for >256 total accounts, got nil")
	}
}

// NewTransactionFromInstructionsV0 wraps NewMessageV0 into an unsigned
// versioned transaction.
func TestNewTransactionFromInstructionsV0(t *testing.T) {
	payer := PublicKey{0x01}
	prog := PublicKey{0x99}
	altKey := PublicKey{0xA0}
	tableAddr := PublicKey{0x02}

	ix := &testInstruction{
		pid:  prog,
		meta: []*AccountMeta{NewAccountMeta(payer, true, true), NewAccountMeta(tableAddr, false, true)},
		data: []byte{0xaa},
	}
	tables := []LoadedAddressLookupTable{{AccountKey: altKey, Addresses: []PublicKey{tableAddr}}}

	tx, err := NewTransactionFromInstructionsV0([]Instruction{ix}, Hash{0x42}, payer, tables)
	if err != nil {
		t.Fatalf("NewTransactionFromInstructionsV0: %v", err)
	}
	if tx.Message.Version != MessageVersion0 {
		t.Errorf("Version = %d, want v0", tx.Message.Version)
	}
	if len(tx.Message.AddressTableLookups) != 1 {
		t.Errorf("AddressTableLookups len = %d, want 1", len(tx.Message.AddressTableLookups))
	}
}
