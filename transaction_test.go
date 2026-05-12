package solana

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------

func makeDeterministicKeypair(t testing.TB, marker byte) *Ed25519Keypair {
	t.Helper()
	seed := bytes.Repeat([]byte{marker}, ed25519.SeedSize)
	kp, err := Ed25519KeypairFromSeed(seed)
	if err != nil {
		t.Fatal(err)
	}
	return kp
}

// makeSimpleTx returns an unsigned Transaction whose first `nSigners`
// AccountKeys correspond to keypairs derived from deterministic
// seeds (0x01, 0x02, ...). The message is a one-instruction legacy
// message that references every key and uses the system program as
// the program id (the last account key).
func makeSimpleTx(t testing.TB, nSigners int) (*Transaction, []*Ed25519Keypair) {
	t.Helper()
	systemProgram, err := PublicKeyFromBase58(systemProgramID)
	if err != nil {
		t.Fatal(err)
	}
	kps := make([]*Ed25519Keypair, nSigners)
	keys := make([]PublicKey, 0, nSigners+1)
	for i := 0; i < nSigners; i++ {
		kps[i] = makeDeterministicKeypair(t, byte(i+1))
		keys = append(keys, kps[i].PublicKey())
	}
	keys = append(keys, systemProgram)

	ixAccounts := make(Uint8Slice, nSigners)
	for i := range ixAccounts {
		ixAccounts[i] = uint8(i)
	}

	msg := Message{
		Version: MessageVersionLegacy,
		Header: MessageHeader{
			NumRequiredSignatures:       uint8(nSigners),
			NumReadonlySignedAccounts:   0,
			NumReadonlyUnsignedAccounts: 1,
		},
		AccountKeys:     keys,
		RecentBlockhash: Hash{0xAA, 0xBB, 0xCC},
		Instructions: []CompiledInstruction{
			{
				ProgramIDIndex: uint8(nSigners),
				Accounts:       ixAccounts,
				Data:           []byte{0x02, 0, 0, 0, 1, 2, 3, 4},
			},
		},
	}
	return NewTransaction(msg), kps
}

// ---------------------------------------------------------------------
// Construction
// ---------------------------------------------------------------------

func TestNewTransaction_PreallocatesSignatureSlots(t *testing.T) {
	tx, _ := makeSimpleTx(t, 3)
	if len(tx.Signatures) != 3 {
		t.Fatalf("Signatures len = %d, want 3", len(tx.Signatures))
	}
	for i, sig := range tx.Signatures {
		if !sig.IsZero() {
			t.Errorf("Signatures[%d] should be zero, got %x", i, sig[:8])
		}
	}
}

func TestTransaction_MessageBytes_MatchesMarshalMessage(t *testing.T) {
	tx, _ := makeSimpleTx(t, 1)
	a, err := tx.MessageBytes()
	if err != nil {
		t.Fatal(err)
	}
	b, err := tx.Message.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatal("MessageBytes and Message.Marshal returned different bytes")
	}
}

// ---------------------------------------------------------------------
// Sign
// ---------------------------------------------------------------------

func TestTransaction_Sign_SingleSigner(t *testing.T) {
	tx, kps := makeSimpleTx(t, 1)
	if err := tx.Sign(context.Background(), kps[0]); err != nil {
		t.Fatal(err)
	}
	if tx.Signatures[0].IsZero() {
		t.Fatal("signature slot 0 still zero after Sign")
	}
	if err := tx.VerifySignatures(); err != nil {
		t.Fatalf("VerifySignatures: %v", err)
	}
}

func TestTransaction_Sign_MultipleSignersInAnyOrder(t *testing.T) {
	tx, kps := makeSimpleTx(t, 4)
	// Pass signers in a shuffled order to prove slot assignment is
	// by public key, not by signer argument position.
	if err := tx.Sign(context.Background(), kps[2], kps[0], kps[3], kps[1]); err != nil {
		t.Fatal(err)
	}
	for i, sig := range tx.Signatures {
		if sig.IsZero() {
			t.Errorf("slot %d still zero", i)
		}
	}
	if err := tx.VerifySignatures(); err != nil {
		t.Fatalf("VerifySignatures: %v", err)
	}
}

func TestTransaction_Sign_Incremental(t *testing.T) {
	tx, kps := makeSimpleTx(t, 3)
	// First pass fills only slot 0.
	if err := tx.Sign(context.Background(), kps[0]); err != nil {
		t.Fatal(err)
	}
	if tx.Signatures[0].IsZero() || !tx.Signatures[1].IsZero() || !tx.Signatures[2].IsZero() {
		t.Fatal("incremental: after first Sign only slot 0 should be filled")
	}
	// Second pass fills slots 1 and 2 without disturbing slot 0.
	sig0Before := tx.Signatures[0]
	if err := tx.Sign(context.Background(), kps[1], kps[2]); err != nil {
		t.Fatal(err)
	}
	if !tx.Signatures[0].Equal(sig0Before) {
		t.Fatal("incremental: slot 0 was disturbed by the second Sign call")
	}
	if tx.Signatures[1].IsZero() || tx.Signatures[2].IsZero() {
		t.Fatal("incremental: slots 1 and 2 should be filled")
	}
	if err := tx.VerifySignatures(); err != nil {
		t.Fatalf("VerifySignatures: %v", err)
	}
}

func TestTransaction_Sign_RejectsNonRequiredSigner(t *testing.T) {
	tx, _ := makeSimpleTx(t, 1)
	stranger := makeDeterministicKeypair(t, 0xFE)
	err := tx.Sign(context.Background(), stranger)
	if err == nil {
		t.Fatal("expected error for non-required signer")
	}
	if !strings.Contains(err.Error(), "not a required signer") {
		t.Errorf("error message: %v", err)
	}
	// The failed Sign must not have touched the signature slot.
	if !tx.Signatures[0].IsZero() {
		t.Fatal("failed Sign mutated signature slot")
	}
}

func TestTransaction_Sign_RejectsZeroRequired(t *testing.T) {
	msg := Message{
		Version:     MessageVersionLegacy,
		Header:      MessageHeader{NumRequiredSignatures: 0},
		AccountKeys: []PublicKey{{0x01}},
	}
	tx := NewTransaction(msg)
	kp := makeDeterministicKeypair(t, 0x01)
	if err := tx.Sign(context.Background(), kp); err == nil {
		t.Fatal("expected error when message requires zero signatures")
	}
}

func TestTransaction_Sign_MixesLocalAndRemote(t *testing.T) {
	tx, kps := makeSimpleTx(t, 2)
	remote, err := NewRemoteSigner(kps[1].PublicKey(), kps[1].Sign)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Sign(context.Background(), kps[0], remote); err != nil {
		t.Fatal(err)
	}
	if err := tx.VerifySignatures(); err != nil {
		t.Fatalf("VerifySignatures: %v", err)
	}
}

// ---------------------------------------------------------------------
// Marshal / Unmarshal
// ---------------------------------------------------------------------

func TestTransaction_Marshal_UnmarshalRoundTrip(t *testing.T) {
	tx, kps := makeSimpleTx(t, 3)
	if err := tx.Sign(context.Background(), kps[0], kps[1], kps[2]); err != nil {
		t.Fatal(err)
	}
	data, err := tx.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	out := &Transaction{}
	if err := out.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}
	if len(out.Signatures) != 3 {
		t.Fatalf("Signatures len = %d", len(out.Signatures))
	}
	for i := range out.Signatures {
		if !out.Signatures[i].Equal(tx.Signatures[i]) {
			t.Errorf("sig %d mismatch", i)
		}
	}
	if out.Message.Header != tx.Message.Header {
		t.Errorf("Header mismatch")
	}
	if !out.Message.RecentBlockhash.Equal(tx.Message.RecentBlockhash) {
		t.Errorf("blockhash mismatch")
	}
	// After round-trip the decoded transaction must still verify.
	if err := out.VerifySignatures(); err != nil {
		t.Fatalf("VerifySignatures after round-trip: %v", err)
	}
}

func TestTransaction_UnmarshalEmpty(t *testing.T) {
	if err := new(Transaction).UnmarshalBinary(nil); err == nil {
		t.Fatal("expected error on nil input")
	}
	if err := new(Transaction).UnmarshalBinary([]byte{}); err == nil {
		t.Fatal("expected error on empty input")
	}
}

func TestTransaction_Unmarshal_Truncated(t *testing.T) {
	tx, kps := makeSimpleTx(t, 2)
	if err := tx.Sign(context.Background(), kps[0], kps[1]); err != nil {
		t.Fatal(err)
	}
	data, err := tx.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i < len(data); i++ {
		if err := new(Transaction).UnmarshalBinary(data[:i]); err == nil {
			t.Fatalf("truncated at %d/%d should have errored", i, len(data))
		}
	}
}

func TestTransaction_Unmarshal_TrailingBytes(t *testing.T) {
	tx, kps := makeSimpleTx(t, 1)
	if err := tx.Sign(context.Background(), kps[0]); err != nil {
		t.Fatal(err)
	}
	data, err := tx.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, 0xFF)
	if err := new(Transaction).UnmarshalBinary(data); err == nil {
		t.Fatal("expected error for trailing bytes")
	}
}

// ---------------------------------------------------------------------
// VerifySignatures
// ---------------------------------------------------------------------

func TestTransaction_VerifySignatures_TamperedMessageFails(t *testing.T) {
	tx, kps := makeSimpleTx(t, 1)
	if err := tx.Sign(context.Background(), kps[0]); err != nil {
		t.Fatal(err)
	}
	// Tamper: flip a byte in the blockhash.
	tx.Message.RecentBlockhash[0] ^= 0xFF
	if err := tx.VerifySignatures(); err == nil {
		t.Fatal("expected verification failure after tampering with message")
	}
}

func TestTransaction_VerifySignatures_SkipsZeroSlots(t *testing.T) {
	tx, kps := makeSimpleTx(t, 3)
	// Sign only the first slot; slots 1 and 2 remain zero.
	if err := tx.Sign(context.Background(), kps[0]); err != nil {
		t.Fatal(err)
	}
	if err := tx.VerifySignatures(); err != nil {
		t.Fatalf("VerifySignatures should skip zero slots: %v", err)
	}
}

// ---------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------

func benchmarkTransactionSign(b *testing.B, nSigners int) {
	tx, kps := makeSimpleTx(b, nSigners)
	signers := make([]Signer, nSigners)
	for i := range kps {
		signers[i] = kps[i]
	}
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reset signature slots to zero so each iteration does the
		// same amount of work.
		for j := range tx.Signatures {
			tx.Signatures[j] = Signature{}
		}
		if err := tx.Sign(ctx, signers...); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTransaction_Sign_1(b *testing.B)  { benchmarkTransactionSign(b, 1) }
func BenchmarkTransaction_Sign_4(b *testing.B)  { benchmarkTransactionSign(b, 4) }
func BenchmarkTransaction_Sign_8(b *testing.B)  { benchmarkTransactionSign(b, 8) }
func BenchmarkTransaction_Sign_16(b *testing.B) { benchmarkTransactionSign(b, 16) }

func BenchmarkTransaction_Marshal(b *testing.B) {
	tx, kps := makeSimpleTx(b, 4)
	signers := []Signer{kps[0], kps[1], kps[2], kps[3]}
	if err := tx.Sign(context.Background(), signers...); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tx.Marshal()
	}
}

func BenchmarkTransaction_Unmarshal(b *testing.B) {
	tx, kps := makeSimpleTx(b, 4)
	signers := []Signer{kps[0], kps[1], kps[2], kps[3]}
	if err := tx.Sign(context.Background(), signers...); err != nil {
		b.Fatal(err)
	}
	data, err := tx.Marshal()
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = new(Transaction).UnmarshalBinary(data)
	}
}
