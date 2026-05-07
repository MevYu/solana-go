package solana

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/MevYu/solana-go/encoding"
)

// buildEntriesBuffer constructs a bincode Vec<Entry> payload for testing,
// using real wire-format transactions.
func buildEntriesBuffer(t *testing.T, entries []Entry) []byte {
	t.Helper()
	var out bytes.Buffer
	var scratch [8]byte

	binary.LittleEndian.PutUint64(scratch[:], uint64(len(entries)))
	out.Write(scratch[:])

	for _, e := range entries {
		binary.LittleEndian.PutUint64(scratch[:], e.NumHashes)
		out.Write(scratch[:])
		out.Write(e.Hash[:])
		binary.LittleEndian.PutUint64(scratch[:], uint64(len(e.Transactions)))
		out.Write(scratch[:])
		for _, tx := range e.Transactions {
			b, err := tx.Marshal()
			if err != nil {
				t.Fatalf("tx marshal: %v", err)
			}
			out.Write(b)
		}
	}
	return out.Bytes()
}

// sampleTransaction returns a small, valid legacy transaction used as a
// round-trip payload for streaming decode tests. Signatures are zeroed.
func sampleTransaction(payer PublicKey, blockhash Hash) Transaction {
	msg := Message{
		Version: MessageVersionLegacy,
		Header: MessageHeader{
			NumRequiredSignatures:       1,
			NumReadonlySignedAccounts:   0,
			NumReadonlyUnsignedAccounts: 1,
		},
		AccountKeys:     []PublicKey{payer, SystemProgramID},
		RecentBlockhash: blockhash,
		Instructions: []CompiledInstruction{
			{ProgramIDIndex: 1, Accounts: Uint8Slice{0}, Data: []byte{0x01, 0x02}},
		},
	}
	return Transaction{
		Signatures: []Signature{{}},
		Message:    msg,
	}
}

func TestDecodeEntries_Empty(t *testing.T) {
	buf := buildEntriesBuffer(t, nil)
	got, err := DecodeEntries(buf)
	if err != nil {
		t.Fatalf("DecodeEntries: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 entries, got %d", len(got))
	}
}

func TestDecodeEntries_Single(t *testing.T) {
	payer := PublicKey{1, 2, 3, 4}
	bh := Hash{9, 9, 9}
	tx := sampleTransaction(payer, bh)
	in := []Entry{{
		NumHashes:    42,
		Hash:         Hash{0xaa, 0xbb},
		Transactions: []Transaction{tx},
	}}
	buf := buildEntriesBuffer(t, in)

	got, err := DecodeEntries(buf)
	if err != nil {
		t.Fatalf("DecodeEntries: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("entries = %d, want 1", len(got))
	}
	e := got[0]
	if e.NumHashes != 42 {
		t.Errorf("NumHashes = %d", e.NumHashes)
	}
	if e.Hash != (Hash{0xaa, 0xbb}) {
		t.Errorf("Hash = %v", e.Hash)
	}
	if len(e.Transactions) != 1 {
		t.Fatalf("txs = %d, want 1", len(e.Transactions))
	}
	if e.Transactions[0].Message.AccountKeys[0] != payer {
		t.Errorf("payer mismatch")
	}
	if e.Transactions[0].Message.RecentBlockhash != bh {
		t.Errorf("blockhash mismatch")
	}
}

func TestDecodeEntries_MultipleTransactions(t *testing.T) {
	payer := PublicKey{1}
	bh := Hash{2}
	in := []Entry{{
		NumHashes: 1,
		Hash:      Hash{},
		Transactions: []Transaction{
			sampleTransaction(payer, bh),
			sampleTransaction(PublicKey{3}, Hash{4}),
			sampleTransaction(PublicKey{5}, Hash{6}),
		},
	}}
	buf := buildEntriesBuffer(t, in)

	got, err := DecodeEntries(buf)
	if err != nil {
		t.Fatalf("DecodeEntries: %v", err)
	}
	if len(got[0].Transactions) != 3 {
		t.Fatalf("txs = %d, want 3", len(got[0].Transactions))
	}
	if got[0].Transactions[1].Message.AccountKeys[0] != (PublicKey{3}) {
		t.Errorf("tx[1] payer mismatch")
	}
	if got[0].Transactions[2].Message.RecentBlockhash != (Hash{6}) {
		t.Errorf("tx[2] blockhash mismatch")
	}
}

func TestDecodeEntries_TruncatedBuffer(t *testing.T) {
	buf := make([]byte, 4) // < 8 bytes
	if _, err := DecodeEntries(buf); err == nil {
		t.Fatal("expected error on truncated vec length")
	}
}

func TestDecodeEntries_TrailingBytes(t *testing.T) {
	buf := buildEntriesBuffer(t, nil)
	buf = append(buf, 0xff, 0xff)
	if _, err := DecodeEntries(buf); err == nil {
		t.Fatal("expected trailing-bytes error")
	}
}

func TestDecodeTransaction_StreamAdvancesCursor(t *testing.T) {
	// Two transactions back-to-back in one buffer; verify DecodeTransaction
	// reads exactly the first one and leaves the second intact.
	tx1 := sampleTransaction(PublicKey{1}, Hash{2})
	tx2 := sampleTransaction(PublicKey{3}, Hash{4})

	b1, err := tx1.Marshal()
	if err != nil {
		t.Fatalf("tx1 marshal: %v", err)
	}
	b2, err := tx2.Marshal()
	if err != nil {
		t.Fatalf("tx2 marshal: %v", err)
	}
	buf := append(b1, b2...)

	d := encoding.NewDecoder(buf)
	got1, err := DecodeTransaction(d)
	if err != nil {
		t.Fatalf("DecodeTransaction #1: %v", err)
	}
	if got1.Message.AccountKeys[0] != (PublicKey{1}) {
		t.Errorf("tx1 account mismatch")
	}
	if d.Pos() != len(b1) {
		t.Errorf("cursor = %d, want %d", d.Pos(), len(b1))
	}

	got2, err := DecodeTransaction(d)
	if err != nil {
		t.Fatalf("DecodeTransaction #2: %v", err)
	}
	if got2.Message.RecentBlockhash != (Hash{4}) {
		t.Errorf("tx2 blockhash mismatch")
	}
	if d.Remaining() != 0 {
		t.Errorf("remaining = %d, want 0", d.Remaining())
	}
}

func TestUnmarshalTransaction_RejectsTrailing(t *testing.T) {
	tx := sampleTransaction(PublicKey{1}, Hash{2})
	b, err := tx.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	b = append(b, 0xff)
	if _, err := UnmarshalTransaction(b); err == nil {
		t.Fatal("expected trailing-bytes error")
	}
}
