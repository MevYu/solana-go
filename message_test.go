package solana

import (
	"bytes"
	"strings"
	"testing"
)

func makeLegacyMessage(t *testing.T) *Message {
	t.Helper()
	systemProgram, err := PublicKeyFromBase58(systemProgramID)
	if err != nil {
		t.Fatal(err)
	}
	tokenProgram, err := PublicKeyFromBase58(tokenProgramID)
	if err != nil {
		t.Fatal(err)
	}
	return &Message{
		Version: MessageVersionLegacy,
		Header: MessageHeader{
			NumRequiredSignatures:       1,
			NumReadonlySignedAccounts:   0,
			NumReadonlyUnsignedAccounts: 1,
		},
		AccountKeys: []PublicKey{
			{0x01, 0x02, 0x03},
			tokenProgram,
			systemProgram,
		},
		RecentBlockhash: Hash{0x42, 0x43, 0x44},
		Instructions: []CompiledInstruction{
			{
				ProgramIDIndex: 2,
				Accounts:       []uint8{0, 1},
				Data:           []byte{0x02, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03, 0x04},
			},
		},
	}
}

func makeV0Message(t *testing.T) *Message {
	t.Helper()
	systemProgram, err := PublicKeyFromBase58(systemProgramID)
	if err != nil {
		t.Fatal(err)
	}
	return &Message{
		Version: MessageVersion0,
		Header: MessageHeader{
			NumRequiredSignatures:       1,
			NumReadonlySignedAccounts:   0,
			NumReadonlyUnsignedAccounts: 1,
		},
		AccountKeys: []PublicKey{
			{0x11, 0x22, 0x33},
			systemProgram,
		},
		RecentBlockhash: Hash{0xde, 0xad, 0xbe, 0xef},
		Instructions: []CompiledInstruction{
			{
				ProgramIDIndex: 1,
				Accounts:       []uint8{0, 2},
				Data:           []byte{0x01},
			},
		},
		AddressTableLookups: []MessageAddressTableLookup{
			{
				AccountKey:      PublicKey{0xaa, 0xbb, 0xcc},
				WritableIndexes: []uint8{5, 6},
				ReadonlyIndexes: []uint8{7},
			},
		},
	}
}

func TestMessage_Legacy_RoundTrip(t *testing.T) {
	m := makeLegacyMessage(t)
	data, err := m.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	// A legacy message's first byte is numRequiredSignatures, which
	// must never have bit 7 set.
	if data[0]&versionPrefixMask != 0 {
		t.Fatalf("legacy first byte = %#x, must not set 0x80", data[0])
	}

	out, err := UnmarshalMessage(data)
	if err != nil {
		t.Fatal(err)
	}

	if out.Version != MessageVersionLegacy {
		t.Errorf("Version = %d, want legacy", out.Version)
	}
	if out.Header != m.Header {
		t.Errorf("Header = %+v, want %+v", out.Header, m.Header)
	}
	if len(out.AccountKeys) != len(m.AccountKeys) {
		t.Fatalf("AccountKeys len = %d, want %d", len(out.AccountKeys), len(m.AccountKeys))
	}
	for i := range m.AccountKeys {
		if !out.AccountKeys[i].Equal(m.AccountKeys[i]) {
			t.Errorf("AccountKeys[%d] mismatch", i)
		}
	}
	if !out.RecentBlockhash.Equal(m.RecentBlockhash) {
		t.Error("blockhash mismatch")
	}
	if len(out.Instructions) != 1 {
		t.Fatalf("Instructions len = %d", len(out.Instructions))
	}
	got := out.Instructions[0]
	want := m.Instructions[0]
	if got.ProgramIDIndex != want.ProgramIDIndex {
		t.Errorf("ProgramIDIndex = %d, want %d", got.ProgramIDIndex, want.ProgramIDIndex)
	}
	if !bytes.Equal(got.Accounts, want.Accounts) {
		t.Errorf("Accounts = %v, want %v", got.Accounts, want.Accounts)
	}
	if !bytes.Equal(got.Data, want.Data) {
		t.Errorf("Data = %x, want %x", got.Data, want.Data)
	}
	if len(out.AddressTableLookups) != 0 {
		t.Errorf("legacy message should have no ALTs, got %d", len(out.AddressTableLookups))
	}
}

func TestMessage_V0_RoundTrip(t *testing.T) {
	m := makeV0Message(t)
	data, err := m.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	// The first byte for v0 is exactly versionPrefixMask | 0 == 0x80.
	if data[0] != versionPrefixMask {
		t.Fatalf("v0 first byte = %#x, want %#x", data[0], versionPrefixMask)
	}

	out, err := UnmarshalMessage(data)
	if err != nil {
		t.Fatal(err)
	}
	if out.Version != MessageVersion0 {
		t.Errorf("Version = %d, want 0", out.Version)
	}
	if len(out.AddressTableLookups) != 1 {
		t.Fatalf("ALT lookups len = %d", len(out.AddressTableLookups))
	}
	lk := out.AddressTableLookups[0]
	want := m.AddressTableLookups[0]
	if !lk.AccountKey.Equal(want.AccountKey) {
		t.Error("ALT AccountKey mismatch")
	}
	if !bytes.Equal(lk.WritableIndexes, want.WritableIndexes) {
		t.Errorf("WritableIndexes = %v, want %v", lk.WritableIndexes, want.WritableIndexes)
	}
	if !bytes.Equal(lk.ReadonlyIndexes, want.ReadonlyIndexes) {
		t.Errorf("ReadonlyIndexes = %v, want %v", lk.ReadonlyIndexes, want.ReadonlyIndexes)
	}
}

func TestMessage_V0_EmptyLookups_RoundTrip(t *testing.T) {
	m := &Message{
		Version:      MessageVersion0,
		Header:       MessageHeader{NumRequiredSignatures: 1},
		AccountKeys:  []PublicKey{{0x01}},
		Instructions: []CompiledInstruction{},
	}
	data, err := m.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	out, err := UnmarshalMessage(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.AddressTableLookups) != 0 {
		t.Errorf("got %d lookups, want 0", len(out.AddressTableLookups))
	}
}

func TestMessage_EmptyInput(t *testing.T) {
	if _, err := UnmarshalMessage(nil); err == nil {
		t.Fatal("expected error on nil input")
	}
	if _, err := UnmarshalMessage([]byte{}); err == nil {
		t.Fatal("expected error on empty input")
	}
}

func TestMessage_UnsupportedVersion(t *testing.T) {
	// Fabricate a message whose first byte is versionPrefixMask | 1,
	// i.e. a v1 message which we don't support yet.
	data := []byte{
		versionPrefixMask | 1, // version 1
		0, 0, 0,               // header
		0, // account keys count
	}
	data = append(data, make([]byte, HashSize)...) // blockhash
	data = append(data, 0)                         // instructions count
	data = append(data, 0)                         // lookups count

	_, err := UnmarshalMessage(data)
	if err == nil {
		t.Fatal("expected error for unsupported version")
	}
	if !strings.Contains(err.Error(), "unsupported version") {
		t.Errorf("error should mention version: %v", err)
	}
}

func TestMessage_Legacy_RejectsLookups(t *testing.T) {
	m := &Message{
		Version:     MessageVersionLegacy,
		AccountKeys: []PublicKey{{0x01}},
		AddressTableLookups: []MessageAddressTableLookup{
			{AccountKey: PublicKey{0x02}},
		},
	}
	if _, err := m.Marshal(); err == nil {
		t.Fatal("expected error for legacy message with ALTs")
	}
}

func TestMessage_Marshal_UnknownVersion(t *testing.T) {
	m := &Message{
		Version:     MessageVersion(42),
		AccountKeys: []PublicKey{{0x01}},
	}
	if _, err := m.Marshal(); err == nil {
		t.Fatal("expected error for unknown version")
	}
}

func TestMessage_TruncatedFails(t *testing.T) {
	m := makeV0Message(t)
	data, err := m.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(data); i++ {
		if _, err := UnmarshalMessage(data[:i]); err == nil {
			t.Fatalf("truncated at %d/%d should have errored", i, len(data))
		}
	}
}

func TestMessage_TrailingBytes(t *testing.T) {
	m := makeLegacyMessage(t)
	data, err := m.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, 0xff)
	if _, err := UnmarshalMessage(data); err == nil {
		t.Fatal("expected error for trailing bytes")
	}
}

func TestMessage_UnmarshalDoesNotAliasInput(t *testing.T) {
	m := makeLegacyMessage(t)
	data, err := m.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	out, err := UnmarshalMessage(data)
	if err != nil {
		t.Fatal(err)
	}
	// Zero the input buffer after Unmarshal; the returned Message
	// must not observe any of those zeros in its fields.
	for i := range data {
		data[i] = 0
	}
	if !out.AccountKeys[0].Equal(m.AccountKeys[0]) {
		t.Error("AccountKeys aliased the input buffer")
	}
	if !bytes.Equal(out.Instructions[0].Data, m.Instructions[0].Data) {
		t.Error("Instruction.Data aliased the input buffer")
	}
}

// ---------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------

func legacyBenchMessage() *Message {
	sp, _ := PublicKeyFromBase58(systemProgramID)
	return &Message{
		Version: MessageVersionLegacy,
		Header: MessageHeader{
			NumRequiredSignatures:       1,
			NumReadonlyUnsignedAccounts: 1,
		},
		AccountKeys:     []PublicKey{{0x01}, {0x02}, sp},
		RecentBlockhash: Hash{0x42},
		Instructions: []CompiledInstruction{
			{
				ProgramIDIndex: 2,
				Accounts:       []uint8{0, 1},
				Data:           []byte{0x02, 0, 0, 0, 1, 2, 3, 4},
			},
		},
	}
}

func TestMessage_SerializedSize_Legacy(t *testing.T) {
	m := makeLegacyMessage(t)
	wire, err := m.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if got := m.SerializedSize(); got != len(wire) {
		t.Errorf("SerializedSize() = %d, len(Marshal()) = %d", got, len(wire))
	}
}

func TestMessage_SerializedSize_V0(t *testing.T) {
	m := makeV0Message(t)
	wire, err := m.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if got := m.SerializedSize(); got != len(wire) {
		t.Errorf("SerializedSize() = %d, len(Marshal()) = %d", got, len(wire))
	}
}

func TestTransaction_SerializedSize(t *testing.T) {
	payer := PublicKey{0x01}
	ix := &CompiledInstruction{ProgramIDIndex: 0, Accounts: []byte{1, 2}, Data: []byte{0xAA, 0xBB}}
	msg := Message{
		Version: MessageVersionLegacy,
		Header:  MessageHeader{NumRequiredSignatures: 1},
		AccountKeys: []PublicKey{payer, {0x02}, {0x03}},
		RecentBlockhash: Hash{0x42},
		Instructions: []CompiledInstruction{*ix},
	}
	tx := NewTransaction(msg)
	wire, err := tx.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if got := tx.SerializedSize(); got != len(wire) {
		t.Errorf("SerializedSize() = %d, len(Marshal()) = %d", got, len(wire))
	}
}

func BenchmarkMessage_Marshal_Legacy(b *testing.B) {
	m := legacyBenchMessage()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = m.Marshal()
	}
}

func BenchmarkMessage_Unmarshal_Legacy(b *testing.B) {
	m := legacyBenchMessage()
	data, _ := m.Marshal()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = UnmarshalMessage(data)
	}
}
