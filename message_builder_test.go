package solana

import (
	"bytes"
	"testing"
)

// testInstruction is a minimal Instruction used only to drive
// NewMessage tests without pulling in any program binding.
type testInstruction struct {
	pid  PublicKey
	meta []*AccountMeta
	data []byte
}

func (i *testInstruction) ProgramID() PublicKey     { return i.pid }
func (i *testInstruction) Accounts() []*AccountMeta { return i.meta }
func (i *testInstruction) Data() ([]byte, error)    { return i.data, nil }

func TestMustPublicKey_Valid(t *testing.T) {
	pk := MustPublicKey(tokenProgramID)
	if pk.IsZero() {
		t.Error("pk should not be zero")
	}
}

func TestMustPublicKey_InvalidPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for invalid base58")
		}
	}()
	_ = MustPublicKey("not a real base58 key")
}

func TestNewMessage_SingleInstruction(t *testing.T) {
	payer := PublicKey{0x01}
	recipient := PublicKey{0x02}
	systemProg := PublicKey{} // all zeros == system program

	ix := &testInstruction{
		pid: systemProg,
		meta: []*AccountMeta{
			NewAccountMeta(payer, true, true),
			NewAccountMeta(recipient, false, true),
		},
		data: []byte{0x02, 0x00, 0x00, 0x00, 0xe8, 0x03, 0, 0, 0, 0, 0, 0},
	}

	msg, err := NewMessage(payer, []Instruction{ix}, Hash{0x42})
	if err != nil {
		t.Fatal(err)
	}

	// Expected layout: [payer, recipient, systemProgram]
	if len(msg.AccountKeys) != 3 {
		t.Fatalf("AccountKeys len = %d, want 3", len(msg.AccountKeys))
	}
	if !msg.AccountKeys[0].Equal(payer) {
		t.Errorf("AccountKeys[0] should be payer")
	}
	if !msg.AccountKeys[1].Equal(recipient) {
		t.Errorf("AccountKeys[1] should be recipient")
	}
	if !msg.AccountKeys[2].Equal(systemProg) {
		t.Errorf("AccountKeys[2] should be system program")
	}

	// Header: 1 signer (payer), 0 readonly signed, 1 readonly unsigned (system)
	if msg.Header.NumRequiredSignatures != 1 {
		t.Errorf("NumRequiredSignatures = %d", msg.Header.NumRequiredSignatures)
	}
	if msg.Header.NumReadonlySignedAccounts != 0 {
		t.Errorf("NumReadonlySignedAccounts = %d", msg.Header.NumReadonlySignedAccounts)
	}
	if msg.Header.NumReadonlyUnsignedAccounts != 1 {
		t.Errorf("NumReadonlyUnsignedAccounts = %d", msg.Header.NumReadonlyUnsignedAccounts)
	}

	// Instruction should reference accounts [0, 1] and program 2.
	if len(msg.Instructions) != 1 {
		t.Fatalf("Instructions len = %d", len(msg.Instructions))
	}
	cix := msg.Instructions[0]
	if cix.ProgramIDIndex != 2 {
		t.Errorf("ProgramIDIndex = %d, want 2", cix.ProgramIDIndex)
	}
	if !bytes.Equal(cix.Accounts, []byte{0, 1}) {
		t.Errorf("Accounts = %v, want [0 1]", cix.Accounts)
	}
	if !bytes.Equal(cix.Data, ix.data) {
		t.Errorf("Data mismatch")
	}
}

func TestNewMessage_MultipleInstructionsShareAccounts(t *testing.T) {
	payer := PublicKey{0x01}
	shared := PublicKey{0x02}
	prog1 := PublicKey{0x03}
	prog2 := PublicKey{0x04}

	ix1 := &testInstruction{
		pid: prog1,
		meta: []*AccountMeta{
			NewAccountMeta(payer, true, true),
			NewAccountMeta(shared, false, true),
		},
		data: []byte{0xaa},
	}
	ix2 := &testInstruction{
		pid: prog2,
		meta: []*AccountMeta{
			NewAccountMeta(shared, false, true),
		},
		data: []byte{0xbb},
	}

	msg, err := NewMessage(payer, []Instruction{ix1, ix2}, Hash{0x42})
	if err != nil {
		t.Fatal(err)
	}

	// Expected layout: [payer, shared, prog1, prog2]
	if len(msg.AccountKeys) != 4 {
		t.Fatalf("AccountKeys len = %d, want 4", len(msg.AccountKeys))
	}
	if !msg.AccountKeys[0].Equal(payer) {
		t.Errorf("[0] should be payer")
	}
	if !msg.AccountKeys[1].Equal(shared) {
		t.Errorf("[1] should be shared (non-signer writable)")
	}
	// The two programs are readonly-unsigned, ordered by first-seen.
	if !msg.AccountKeys[2].Equal(prog1) {
		t.Errorf("[2] should be prog1")
	}
	if !msg.AccountKeys[3].Equal(prog2) {
		t.Errorf("[3] should be prog2")
	}

	if msg.Header.NumRequiredSignatures != 1 {
		t.Errorf("NumRequiredSignatures = %d", msg.Header.NumRequiredSignatures)
	}
	if msg.Header.NumReadonlyUnsignedAccounts != 2 {
		t.Errorf("NumReadonlyUnsignedAccounts = %d, want 2", msg.Header.NumReadonlyUnsignedAccounts)
	}
}

func TestNewMessage_RoleUpgradeCumulative(t *testing.T) {
	// Same account appearing as signer+readonly then non-signer+writable
	// should end up as signer+writable.
	payer := PublicKey{0x01}
	shared := PublicKey{0x02}
	prog := PublicKey{0x99}

	ix1 := &testInstruction{
		pid: prog,
		meta: []*AccountMeta{
			NewAccountMeta(payer, true, true),
			NewAccountMeta(shared, true, false), // signer, readonly
		},
		data: []byte{0x01},
	}
	ix2 := &testInstruction{
		pid: prog,
		meta: []*AccountMeta{
			NewAccountMeta(shared, false, true), // non-signer, writable
		},
		data: []byte{0x02},
	}

	msg, err := NewMessage(payer, []Instruction{ix1, ix2}, Hash{})
	if err != nil {
		t.Fatal(err)
	}

	// shared should be signer+writable (upgraded from both).
	// Expected layout: [payer, shared, prog] all in the right buckets.
	if !msg.AccountKeys[0].Equal(payer) {
		t.Errorf("[0] should be payer")
	}
	if !msg.AccountKeys[1].Equal(shared) {
		t.Errorf("[1] should be shared; got AccountKeys=%v", msg.AccountKeys)
	}
	if msg.Header.NumRequiredSignatures != 2 {
		t.Errorf("NumRequiredSignatures = %d, want 2", msg.Header.NumRequiredSignatures)
	}
	if msg.Header.NumReadonlySignedAccounts != 0 {
		t.Errorf("NumReadonlySignedAccounts = %d, want 0", msg.Header.NumReadonlySignedAccounts)
	}
}

func TestNewMessage_EmptyInstructions(t *testing.T) {
	if _, err := NewMessage(PublicKey{0x01}, nil, Hash{}); err == nil {
		t.Fatal("expected error for empty instructions")
	}
}

func TestNewMessage_NilInstruction(t *testing.T) {
	if _, err := NewMessage(PublicKey{0x01}, []Instruction{nil}, Hash{}); err == nil {
		t.Fatal("expected error for nil instruction")
	}
}

func TestNewMessage_PayerAlwaysFirst(t *testing.T) {
	// Even when the instruction doesn't mention the payer, it
	// should end up as AccountKeys[0] with signer+writable role.
	payer := PublicKey{0xFF}
	other := PublicKey{0x01}
	prog := PublicKey{0x10}

	ix := &testInstruction{
		pid: prog,
		meta: []*AccountMeta{
			NewAccountMeta(other, false, true),
		},
		data: []byte{0x00},
	}
	msg, err := NewMessage(payer, []Instruction{ix}, Hash{})
	if err != nil {
		t.Fatal(err)
	}
	if !msg.AccountKeys[0].Equal(payer) {
		t.Errorf("payer should be first, got AccountKeys=%v", msg.AccountKeys)
	}
	if msg.Header.NumRequiredSignatures != 1 {
		t.Errorf("NumRequiredSignatures = %d", msg.Header.NumRequiredSignatures)
	}
}

func TestNewMessageV0_SingleALT_RoutesNonSignerWritable(t *testing.T) {
	// A single ALT contains one address. When an instruction references
	// it as a non-signer writable, the builder should route it through
	// the table rather than into the static account list.
	payer := PublicKey{0x01}
	altKey := PublicKey{0xA0}
	tableAddr := PublicKey{0x02}
	prog := PublicKey{0x99}

	ix := &testInstruction{
		pid: prog,
		meta: []*AccountMeta{
			NewAccountMeta(payer, true, true),
			NewAccountMeta(tableAddr, false, true),
		},
		data: []byte{0xaa},
	}

	tables := []LoadedAddressLookupTable{
		{AccountKey: altKey, Addresses: []PublicKey{tableAddr}},
	}
	msg, err := NewMessageV0(payer, []Instruction{ix}, Hash{0x42}, tables)
	if err != nil {
		t.Fatal(err)
	}

	if msg.Version != MessageVersion0 {
		t.Fatalf("Version = %d, want v0", msg.Version)
	}
	// Static accounts: payer (signer+writable), prog (readonly-unsigned).
	if len(msg.AccountKeys) != 2 {
		t.Fatalf("AccountKeys len = %d, want 2 (payer + program only)", len(msg.AccountKeys))
	}
	if !msg.AccountKeys[0].Equal(payer) || !msg.AccountKeys[1].Equal(prog) {
		t.Errorf("static keys = %v, want [payer, prog]", msg.AccountKeys)
	}
	if len(msg.AddressTableLookups) != 1 {
		t.Fatalf("AddressTableLookups len = %d, want 1", len(msg.AddressTableLookups))
	}
	lk := msg.AddressTableLookups[0]
	if !lk.AccountKey.Equal(altKey) {
		t.Errorf("lookup AccountKey = %s, want %s", lk.AccountKey, altKey)
	}
	if !bytes.Equal(lk.WritableIndexes, []byte{0}) {
		t.Errorf("WritableIndexes = %v, want [0]", lk.WritableIndexes)
	}
	if len(lk.ReadonlyIndexes) != 0 {
		t.Errorf("ReadonlyIndexes = %v, want []", lk.ReadonlyIndexes)
	}

	// Compiled instruction accounts: ProgramIDIndex=1 (static), accounts=[0,2]:
	// payer at static index 0, tableAddr at index 2 (= len(static) + 0).
	cix := msg.Instructions[0]
	if cix.ProgramIDIndex != 1 {
		t.Errorf("ProgramIDIndex = %d, want 1", cix.ProgramIDIndex)
	}
	if !bytes.Equal(cix.Accounts, []byte{0, 2}) {
		t.Errorf("Accounts = %v, want [0 2]", cix.Accounts)
	}
}

func TestNewMessageV0_SignerStaysStaticEvenIfInTable(t *testing.T) {
	// Signers must never be routed through an ALT; the runtime verifies
	// signatures before resolving table lookups.
	payer := PublicKey{0x01}
	altKey := PublicKey{0xA0}
	signerInTable := PublicKey{0x02}
	prog := PublicKey{0x99}

	ix := &testInstruction{
		pid: prog,
		meta: []*AccountMeta{
			NewAccountMeta(payer, true, true),
			NewAccountMeta(signerInTable, true, true), // signer + writable
		},
		data: []byte{0xaa},
	}
	tables := []LoadedAddressLookupTable{
		{AccountKey: altKey, Addresses: []PublicKey{signerInTable}},
	}
	msg, err := NewMessageV0(payer, []Instruction{ix}, Hash{}, tables)
	if err != nil {
		t.Fatal(err)
	}
	// The signer must end up static; no ALT lookups emitted at all.
	if len(msg.AddressTableLookups) != 0 {
		t.Fatalf("AddressTableLookups len = %d, want 0 (signer should stay static)", len(msg.AddressTableLookups))
	}
	if msg.Header.NumRequiredSignatures != 2 {
		t.Errorf("NumRequiredSignatures = %d, want 2", msg.Header.NumRequiredSignatures)
	}
	// Static layout: [payer, signerInTable, prog]
	if len(msg.AccountKeys) != 3 ||
		!msg.AccountKeys[0].Equal(payer) ||
		!msg.AccountKeys[1].Equal(signerInTable) ||
		!msg.AccountKeys[2].Equal(prog) {
		t.Errorf("AccountKeys = %v, want [payer, signerInTable, prog]", msg.AccountKeys)
	}
}

func TestNewMessageV0_MultiALT_FirstMatchWins(t *testing.T) {
	// When the same address is present in two ALTs, only the first table
	// should be referenced for that address.
	payer := PublicKey{0x01}
	alt1Key := PublicKey{0xA1}
	alt2Key := PublicKey{0xA2}
	shared := PublicKey{0x02}
	onlyIn2 := PublicKey{0x03}
	prog := PublicKey{0x99}

	ix := &testInstruction{
		pid: prog,
		meta: []*AccountMeta{
			NewAccountMeta(payer, true, true),
			NewAccountMeta(shared, false, true),
			NewAccountMeta(onlyIn2, false, false),
		},
		data: []byte{0xaa},
	}
	tables := []LoadedAddressLookupTable{
		{AccountKey: alt1Key, Addresses: []PublicKey{shared}},          // slot 0
		{AccountKey: alt2Key, Addresses: []PublicKey{shared, onlyIn2}}, // slot 0, 1
	}
	msg, err := NewMessageV0(payer, []Instruction{ix}, Hash{}, tables)
	if err != nil {
		t.Fatal(err)
	}

	if len(msg.AddressTableLookups) != 2 {
		t.Fatalf("AddressTableLookups len = %d, want 2", len(msg.AddressTableLookups))
	}
	// alt1 must carry shared (writable, slot 0); alt2 must carry only
	// onlyIn2 (readonly, slot 1) since shared was claimed by alt1.
	lk1 := msg.AddressTableLookups[0]
	lk2 := msg.AddressTableLookups[1]
	if !lk1.AccountKey.Equal(alt1Key) {
		t.Errorf("lookup[0].AccountKey = %s, want alt1", lk1.AccountKey)
	}
	if !bytes.Equal(lk1.WritableIndexes, []byte{0}) {
		t.Errorf("alt1 writable = %v, want [0]", lk1.WritableIndexes)
	}
	if len(lk1.ReadonlyIndexes) != 0 {
		t.Errorf("alt1 readonly = %v, want []", lk1.ReadonlyIndexes)
	}
	if !lk2.AccountKey.Equal(alt2Key) {
		t.Errorf("lookup[1].AccountKey = %s, want alt2", lk2.AccountKey)
	}
	if len(lk2.WritableIndexes) != 0 {
		t.Errorf("alt2 writable = %v, want []", lk2.WritableIndexes)
	}
	if !bytes.Equal(lk2.ReadonlyIndexes, []byte{1}) {
		t.Errorf("alt2 readonly = %v, want [1]", lk2.ReadonlyIndexes)
	}
}

func TestNewMessageV0_DropsEmptyTable(t *testing.T) {
	// A table that contains no referenced account should not appear in
	// AddressTableLookups at all.
	payer := PublicKey{0x01}
	usedALT := PublicKey{0xA1}
	unusedALT := PublicKey{0xA2}
	addr := PublicKey{0x02}
	unused := PublicKey{0x03}
	prog := PublicKey{0x99}

	ix := &testInstruction{
		pid: prog,
		meta: []*AccountMeta{
			NewAccountMeta(payer, true, true),
			NewAccountMeta(addr, false, false),
		},
		data: []byte{0xaa},
	}
	tables := []LoadedAddressLookupTable{
		{AccountKey: usedALT, Addresses: []PublicKey{addr}},
		{AccountKey: unusedALT, Addresses: []PublicKey{unused}},
	}
	msg, err := NewMessageV0(payer, []Instruction{ix}, Hash{}, tables)
	if err != nil {
		t.Fatal(err)
	}
	if len(msg.AddressTableLookups) != 1 {
		t.Fatalf("AddressTableLookups len = %d, want 1 (unused ALT dropped)", len(msg.AddressTableLookups))
	}
	if !msg.AddressTableLookups[0].AccountKey.Equal(usedALT) {
		t.Errorf("lookup = %s, want usedALT", msg.AddressTableLookups[0].AccountKey)
	}
}

func TestNewMessageV0_SortsAndDedupsTableIndexes(t *testing.T) {
	// Within a single table the writable / readonly index lists must be
	// sorted ascending and each slot index must appear at most once even
	// when referenced from multiple instructions with different roles.
	payer := PublicKey{0x01}
	altKey := PublicKey{0xA0}
	a := PublicKey{0x10} // slot 5 (high), seen as writable then readonly
	b := PublicKey{0x11} // slot 1 (low), seen as readonly only
	c := PublicKey{0x12} // slot 3 (mid), seen as writable only
	prog := PublicKey{0x99}

	// Addresses: [_, b, _, c, _, a]
	addrs := make([]PublicKey, 6)
	addrs[1] = b
	addrs[3] = c
	addrs[5] = a
	tables := []LoadedAddressLookupTable{{AccountKey: altKey, Addresses: addrs}}

	ix1 := &testInstruction{
		pid: prog,
		meta: []*AccountMeta{
			NewAccountMeta(payer, true, true),
			NewAccountMeta(a, false, true), // a as writable
			NewAccountMeta(c, false, true), // c as writable
		},
		data: []byte{0x01},
	}
	ix2 := &testInstruction{
		pid: prog,
		meta: []*AccountMeta{
			NewAccountMeta(a, false, false), // a again, readonly — role is OR
			NewAccountMeta(b, false, false), // b readonly
		},
		data: []byte{0x02},
	}
	msg, err := NewMessageV0(payer, []Instruction{ix1, ix2}, Hash{}, tables)
	if err != nil {
		t.Fatal(err)
	}
	if len(msg.AddressTableLookups) != 1 {
		t.Fatalf("AddressTableLookups len = %d, want 1", len(msg.AddressTableLookups))
	}
	lk := msg.AddressTableLookups[0]
	// Writable indexes: a(5), c(3) → sorted [3, 5]. a's role was OR'd to writable.
	if !bytes.Equal(lk.WritableIndexes, []byte{3, 5}) {
		t.Errorf("WritableIndexes = %v, want [3 5]", lk.WritableIndexes)
	}
	// Readonly indexes: b(1) only — a is writable, not readonly.
	if !bytes.Equal(lk.ReadonlyIndexes, []byte{1}) {
		t.Errorf("ReadonlyIndexes = %v, want [1]", lk.ReadonlyIndexes)
	}
}

func TestNewMessageV0_CompiledInstructionIndexes(t *testing.T) {
	// Verify the full account index layout: static keys first, then
	// per-table writable runs concatenated, then per-table readonly runs.
	// Compiled instruction Accounts entries must point into the right
	// region for each meta.
	payer := PublicKey{0x01}
	alt1Key := PublicKey{0xA1}
	alt2Key := PublicKey{0xA2}
	w1 := PublicKey{0x10} // alt1, writable
	r1 := PublicKey{0x11} // alt1, readonly
	w2 := PublicKey{0x20} // alt2, writable
	r2 := PublicKey{0x21} // alt2, readonly
	prog := PublicKey{0x99}

	ix := &testInstruction{
		pid: prog,
		meta: []*AccountMeta{
			NewAccountMeta(payer, true, true),
			NewAccountMeta(w1, false, true),
			NewAccountMeta(r1, false, false),
			NewAccountMeta(w2, false, true),
			NewAccountMeta(r2, false, false),
		},
		data: []byte{0xaa},
	}
	tables := []LoadedAddressLookupTable{
		{AccountKey: alt1Key, Addresses: []PublicKey{w1, r1}}, // w1=slot0, r1=slot1
		{AccountKey: alt2Key, Addresses: []PublicKey{w2, r2}}, // w2=slot0, r2=slot1
	}
	msg, err := NewMessageV0(payer, []Instruction{ix}, Hash{}, tables)
	if err != nil {
		t.Fatal(err)
	}
	// Static: [payer, prog] → indices 0, 1.
	// Writable section (size 2): alt1.w1=2, alt2.w2=3.
	// Readonly section (size 2): alt1.r1=4, alt2.r2=5.
	// Instruction meta order: [payer, w1, r1, w2, r2] → [0, 2, 4, 3, 5].
	wantAccounts := []byte{0, 2, 4, 3, 5}
	cix := msg.Instructions[0]
	if !bytes.Equal(cix.Accounts, wantAccounts) {
		t.Errorf("Accounts = %v, want %v", cix.Accounts, wantAccounts)
	}
	if cix.ProgramIDIndex != 1 {
		t.Errorf("ProgramIDIndex = %d, want 1", cix.ProgramIDIndex)
	}
}

func TestNewMessageV0_RoundTrip(t *testing.T) {
	// End-to-end: build a v0 message, marshal, unmarshal, compare bytes.
	payer := PublicKey{0x01}
	altKey := PublicKey{0xA0}
	addr := PublicKey{0x02}
	prog := PublicKey{0x99}

	ix := &testInstruction{
		pid: prog,
		meta: []*AccountMeta{
			NewAccountMeta(payer, true, true),
			NewAccountMeta(addr, false, true),
		},
		data: []byte{0xde, 0xad, 0xbe, 0xef},
	}
	tables := []LoadedAddressLookupTable{
		{AccountKey: altKey, Addresses: []PublicKey{addr}},
	}
	msg, err := NewMessageV0(payer, []Instruction{ix}, Hash{0x42, 0x42}, tables)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := msg.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	out := new(Message)
	if err := out.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	if out.Version != MessageVersion0 {
		t.Errorf("Version = %d, want v0", out.Version)
	}
	again, err := out.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, again) {
		t.Errorf("round-trip mismatch:\n  got  %x\n  want %x", again, raw)
	}
}

func TestNewMessageV0_ResolveMatchesOriginalAccounts(t *testing.T) {
	// Round-trip a built v0 message through marshal+unmarshal and then
	// resolve each compiled instruction's account indices via
	// ResolvedAccountKeys. The resolved pubkeys must match the original
	// instruction's meta in the original order — this is the invariant
	// that NewMessageV0's index assignment must preserve.
	payer := PublicKey{0x01}
	alt1Key := PublicKey{0xA1}
	alt2Key := PublicKey{0xA2}
	w1 := PublicKey{0x10}
	r1 := PublicKey{0x11}
	w2 := PublicKey{0x20}
	r2 := PublicKey{0x21}
	prog := PublicKey{0x99}

	originalMeta := []*AccountMeta{
		NewAccountMeta(payer, true, true),
		NewAccountMeta(w1, false, true),
		NewAccountMeta(r1, false, false),
		NewAccountMeta(w2, false, true),
		NewAccountMeta(r2, false, false),
	}
	ix := &testInstruction{pid: prog, meta: originalMeta, data: []byte{0x77}}
	tables := []LoadedAddressLookupTable{
		{AccountKey: alt1Key, Addresses: []PublicKey{w1, r1}},
		{AccountKey: alt2Key, Addresses: []PublicKey{w2, r2}},
	}

	msg, err := NewMessageV0(payer, []Instruction{ix}, Hash{0x42}, tables)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := msg.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	out := new(Message)
	if err := out.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	alts := map[PublicKey][]PublicKey{
		alt1Key: tables[0].Addresses,
		alt2Key: tables[1].Addresses,
	}
	resolved, err := out.ResolvedAccountKeys(alts)
	if err != nil {
		t.Fatal(err)
	}
	cix := out.Instructions[0]
	for i, idx := range cix.Accounts {
		got := resolved[idx]
		want := originalMeta[i].PublicKey
		if !got.Equal(want) {
			t.Errorf("meta[%d]: resolved=%s want=%s (compiled idx=%d, layout=%v)",
				i, got, want, idx, resolved)
		}
	}
	resolvedProg := resolved[cix.ProgramIDIndex]
	if !resolvedProg.Equal(prog) {
		t.Errorf("program: resolved=%s want=%s", resolvedProg, prog)
	}
}

func TestNewMessageV0_EmptyInstructions(t *testing.T) {
	if _, err := NewMessageV0(PublicKey{0x01}, nil, Hash{}, nil); err == nil {
		t.Fatal("expected error for empty instructions")
	}
}

func TestNewMessageV0_NilInstruction(t *testing.T) {
	if _, err := NewMessageV0(PublicKey{0x01}, []Instruction{nil}, Hash{}, nil); err == nil {
		t.Fatal("expected error for nil instruction")
	}
}

func TestNewMessageV0_NoTables_EquivalentToLegacyLayout(t *testing.T) {
	// With no ALTs supplied, NewMessageV0 should produce the same static
	// account layout as NewMessage (only difference: Version + empty
	// AddressTableLookups).
	payer := PublicKey{0x01}
	other := PublicKey{0x02}
	prog := PublicKey{0x99}

	mk := func() *testInstruction {
		return &testInstruction{
			pid: prog,
			meta: []*AccountMeta{
				NewAccountMeta(payer, true, true),
				NewAccountMeta(other, false, true),
			},
			data: []byte{0xaa},
		}
	}
	legacy, err := NewMessage(payer, []Instruction{mk()}, Hash{0x42})
	if err != nil {
		t.Fatal(err)
	}
	v0, err := NewMessageV0(payer, []Instruction{mk()}, Hash{0x42}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(legacy.AccountKeys) != len(v0.AccountKeys) {
		t.Fatalf("AccountKeys len differ: legacy=%d v0=%d", len(legacy.AccountKeys), len(v0.AccountKeys))
	}
	for i := range legacy.AccountKeys {
		if !legacy.AccountKeys[i].Equal(v0.AccountKeys[i]) {
			t.Errorf("AccountKeys[%d] differ: legacy=%s v0=%s", i, legacy.AccountKeys[i], v0.AccountKeys[i])
		}
	}
	if legacy.Header != v0.Header {
		t.Errorf("Header differs: legacy=%+v v0=%+v", legacy.Header, v0.Header)
	}
	if len(v0.AddressTableLookups) != 0 {
		t.Errorf("v0 AddressTableLookups should be empty when no tables supplied, got %d", len(v0.AddressTableLookups))
	}
}
