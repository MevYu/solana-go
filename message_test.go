package solana

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mr-tron/base58"
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

	out := &Message{}
	if err := out.UnmarshalBinary(data); err != nil {
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

	out := &Message{}
	if err := out.UnmarshalBinary(data); err != nil {
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
	out := &Message{}
	if err := out.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}
	if len(out.AddressTableLookups) != 0 {
		t.Errorf("got %d lookups, want 0", len(out.AddressTableLookups))
	}
}

func TestMessage_EmptyInput(t *testing.T) {
	if err := new(Message).UnmarshalBinary(nil); err == nil {
		t.Fatal("expected error on nil input")
	}
	if err := new(Message).UnmarshalBinary([]byte{}); err == nil {
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

	err := new(Message).UnmarshalBinary(data)
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
		if err := new(Message).UnmarshalBinary(data[:i]); err == nil {
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
	if err := new(Message).UnmarshalBinary(data); err == nil {
		t.Fatal("expected error for trailing bytes")
	}
}

func TestMessage_UnmarshalDoesNotAliasInput(t *testing.T) {
	m := makeLegacyMessage(t)
	data, err := m.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	out := &Message{}
	if err := out.UnmarshalBinary(data); err != nil {
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
		Version:         MessageVersionLegacy,
		Header:          MessageHeader{NumRequiredSignatures: 1},
		AccountKeys:     []PublicKey{payer, {0x02}, {0x03}},
		RecentBlockhash: Hash{0x42},
		Instructions:    []CompiledInstruction{*ix},
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
		_ = new(Message).UnmarshalBinary(data)
	}
}

func TestMessageVersion_UnmarshalJSON(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want MessageVersion
	}{
		{"legacy_string", `"legacy"`, MessageVersionLegacy},
		{"empty_string", `""`, MessageVersionLegacy},
		{"null", `null`, MessageVersionLegacy},
		{"v0", `0`, MessageVersion0},
		{"v1_unsupported_but_decodable", `1`, MessageVersion(1)},
		{"v255_clamp_max", `255`, MessageVersion(255)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var v MessageVersion
			if err := json.Unmarshal([]byte(tc.in), &v); err != nil {
				t.Fatalf("Unmarshal(%q): %v", tc.in, err)
			}
			if v != tc.want {
				t.Errorf("got %d, want %d", v, tc.want)
			}
		})
	}
}

func TestMessageVersion_UnmarshalJSON_Invalid(t *testing.T) {
	cases := []string{
		`"unknown"`, // unknown string
		`-1`,        // out of uint8 range
		`256`,       // out of uint8 range
		`1.5`,       // not an integer
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			var v MessageVersion
			if err := json.Unmarshal([]byte(in), &v); err == nil {
				t.Errorf("expected error for %q, got %d", in, v)
			}
		})
	}
}

func TestMessageVersion_MarshalJSON(t *testing.T) {
	cases := []struct {
		in   MessageVersion
		want string
	}{
		{MessageVersionLegacy, `"legacy"`},
		{MessageVersion0, `0`},
		{MessageVersion(7), `7`},
	}
	for _, tc := range cases {
		got, err := json.Marshal(tc.in)
		if err != nil {
			t.Fatalf("Marshal(%d): %v", tc.in, err)
		}
		if string(got) != tc.want {
			t.Errorf("Marshal(%d) = %s, want %s", tc.in, got, tc.want)
		}
	}
}

func TestMessage_Signers(t *testing.T) {
	m := &Message{
		Header: MessageHeader{NumRequiredSignatures: 2},
		AccountKeys: []PublicKey{
			{0x01}, {0x02}, {0x03}, {0x04},
		},
	}
	got := m.Signers()
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0] != (PublicKey{0x01}) || got[1] != (PublicKey{0x02}) {
		t.Errorf("Signers = %+v, want first two keys", got)
	}
}

func TestMessage_Signers_Empty(t *testing.T) {
	m := &Message{
		Header:      MessageHeader{NumRequiredSignatures: 0},
		AccountKeys: []PublicKey{{0x01}},
	}
	if got := m.Signers(); len(got) != 0 {
		t.Errorf("Signers = %+v, want empty", got)
	}
}

func TestMessage_ResolvedAccountKeys_Legacy(t *testing.T) {
	m := makeLegacyMessage(t)
	got, err := m.ResolvedAccountKeys(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(m.AccountKeys) {
		t.Fatalf("len = %d, want %d", len(got), len(m.AccountKeys))
	}
	for i := range m.AccountKeys {
		if got[i] != m.AccountKeys[i] {
			t.Errorf("key[%d] mismatch", i)
		}
	}
}

func TestMessage_ResolvedAccountKeys_V0(t *testing.T) {
	altA := PublicKey{0xa1}
	altB := PublicKey{0xb1}
	wA1, wA2 := PublicKey{0xa1, 0x01}, PublicKey{0xa1, 0x02}
	rA1 := PublicKey{0xa1, 0x03}
	wB1 := PublicKey{0xb1, 0x01}
	rB1, rB2 := PublicKey{0xb1, 0x02}, PublicKey{0xb1, 0x03}

	m := &Message{
		Version:     MessageVersion0,
		Header:      MessageHeader{NumRequiredSignatures: 1},
		AccountKeys: []PublicKey{{0x01}, {0x02}},
		AddressTableLookups: []MessageAddressTableLookup{
			{AccountKey: altA, WritableIndexes: Uint8Slice{0, 1}, ReadonlyIndexes: Uint8Slice{2}},
			{AccountKey: altB, WritableIndexes: Uint8Slice{0}, ReadonlyIndexes: Uint8Slice{1, 2}},
		},
	}
	alts := map[PublicKey][]PublicKey{
		altA: {wA1, wA2, rA1},
		altB: {wB1, rB1, rB2},
	}
	got, err := m.ResolvedAccountKeys(alts)
	if err != nil {
		t.Fatal(err)
	}
	want := []PublicKey{
		{0x01}, {0x02}, // static
		wA1, wA2, wB1, // writable across all ALTs
		rA1, rB1, rB2, // readonly across all ALTs
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d (%+v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d] = %x, want %x", i, got[i], want[i])
		}
	}
}

func TestMessage_ResolvedAccountKeys_MissingALT(t *testing.T) {
	m := &Message{
		Version:     MessageVersion0,
		AccountKeys: []PublicKey{{0x01}},
		AddressTableLookups: []MessageAddressTableLookup{
			{AccountKey: PublicKey{0xaa}, WritableIndexes: Uint8Slice{0}},
		},
	}
	_, err := m.ResolvedAccountKeys(nil)
	if err == nil || !strings.Contains(err.Error(), "not in alts map") {
		t.Errorf("err = %v, want missing-ALT error", err)
	}
}

func TestMessage_ResolvedAccountKeys_IndexOutOfRange(t *testing.T) {
	alt := PublicKey{0xaa}
	m := &Message{
		Version:     MessageVersion0,
		AccountKeys: []PublicKey{{0x01}},
		AddressTableLookups: []MessageAddressTableLookup{
			{AccountKey: alt, WritableIndexes: Uint8Slice{5}},
		},
	}
	alts := map[PublicKey][]PublicKey{alt: {{0xff}}}
	_, err := m.ResolvedAccountKeys(alts)
	if err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Errorf("err = %v, want out-of-range error", err)
	}
}

func TestMessageVersion_RoundTrip(t *testing.T) {
	for _, v := range []MessageVersion{MessageVersionLegacy, MessageVersion0, MessageVersion(42)} {
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		var back MessageVersion
		if err := json.Unmarshal(raw, &back); err != nil {
			t.Fatal(err)
		}
		if back != v {
			t.Errorf("round-trip %d -> %s -> %d", v, raw, back)
		}
	}
}

// ######## CompiledInstruction base58 data ########

// getTransaction/getBlock return meta.innerInstructions[].instructions[].data
// as base58. Before CompiledInstruction.Data was typed Base58Data, the
// encoding/json default decoded []byte as base64 and failed with
// "illegal base64 data ..." on every transaction containing CPIs.
func TestCompiledInstructionUnmarshalJSONBase58Data(t *testing.T) {
	payload := []byte{3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 42}
	dataB58 := base58.Encode(payload)

	raw := `{"programIdIndex":4,"accounts":[1,2,3],"data":"` + dataB58 + `","stackHeight":2}`

	var ci CompiledInstruction
	if err := json.Unmarshal([]byte(raw), &ci); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if ci.ProgramIDIndex != 4 {
		t.Errorf("ProgramIDIndex = %d, want 4", ci.ProgramIDIndex)
	}
	if got := []uint8(ci.Accounts); len(got) != 3 || got[0] != 1 || got[2] != 3 {
		t.Errorf("Accounts = %v, want [1 2 3]", got)
	}
	if string(ci.Data) != string(payload) {
		t.Errorf("Data = %x, want %x", ci.Data, payload)
	}

	// round-trip: data must re-marshal as base58, not base64
	out, err := json.Marshal(ci)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var back CompiledInstruction
	if err := json.Unmarshal(out, &back); err != nil {
		t.Fatalf("re-unmarshal failed: %v", err)
	}
	if string(back.Data) != string(payload) {
		t.Errorf("round-trip Data = %x, want %x", back.Data, payload)
	}
}

// ######## Signers clamp ########

// Signers must not panic when a message decoded from untrusted bytes
// claims more required signers than it has account keys.
func TestMessageSigners_ClampsToAccountKeys(t *testing.T) {
	m := &Message{
		AccountKeys: []PublicKey{{0x01}, {0x02}},
	}
	m.Header.NumRequiredSignatures = 5 // > len(AccountKeys)

	got := m.Signers()
	if len(got) != 2 {
		t.Fatalf("Signers len = %d, want 2 (clamped)", len(got))
	}
	if !got[0].Equal(PublicKey{0x01}) || !got[1].Equal(PublicKey{0x02}) {
		t.Errorf("Signers = %v", got)
	}
}
