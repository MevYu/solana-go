package tx_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"testing"

	solana "github.com/MevYu/solana-go"
	"github.com/MevYu/solana-go/programs/system"
	"github.com/MevYu/solana-go/tx"
)

var testBlockhash = solana.Hash{1, 2, 3}

func newTransfer(from, to solana.PublicKey, lamports uint64) solana.Instruction {
	return system.NewTransfer(from, to, lamports)
}

func newKeypair(t *testing.T) (solana.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate keypair: %v", err)
	}
	var pk solana.PublicKey
	copy(pk[:], pub)
	return pk, priv
}

type localSigner struct {
	pub  solana.PublicKey
	priv ed25519.PrivateKey
}

func (s *localSigner) PublicKey() solana.PublicKey { return s.pub }
func (s *localSigner) Sign(_ context.Context, msg []byte) (solana.Signature, error) {
	raw := ed25519.Sign(s.priv, msg)
	var sig solana.Signature
	copy(sig[:], raw)
	return sig, nil
}

type errSigner struct {
	pub solana.PublicKey
	err error
}

func (e *errSigner) PublicKey() solana.PublicKey { return e.pub }
func (e *errSigner) Sign(_ context.Context, _ []byte) (solana.Signature, error) {
	return solana.Signature{}, e.err
}

// --- Builder.Build ---

func TestBuilder_Build_Happy(t *testing.T) {
	pub, priv := newKeypair(t)
	recipient := solana.PublicKey{9, 9, 9}
	signer := &localSigner{pub: pub, priv: priv}

	txn, err := tx.NewBuilder().
		SetFeePayer(pub).
		SetRecentBlockhash(testBlockhash).
		Add(newTransfer(pub, recipient, 1_000_000)).
		Sign(signer).
		Build(context.Background())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(txn.Signatures) == 0 {
		t.Fatal("no signature slots")
	}
	if txn.Signatures[0].IsZero() {
		t.Fatal("signature slot is zero — not signed")
	}
}

func TestBuilder_Build_NoFeePayer(t *testing.T) {
	pub, _ := newKeypair(t)
	_, err := tx.NewBuilder().
		SetRecentBlockhash(testBlockhash).
		Add(newTransfer(pub, solana.PublicKey{9}, 1)).
		Build(context.Background())
	if err == nil {
		t.Fatal("expected error for missing fee payer")
	}
}

func TestBuilder_Build_NoBlockhash(t *testing.T) {
	pub, _ := newKeypair(t)
	_, err := tx.NewBuilder().
		SetFeePayer(pub).
		Add(newTransfer(pub, solana.PublicKey{9}, 1)).
		Build(context.Background())
	if err == nil {
		t.Fatal("expected error for missing blockhash")
	}
}

func TestBuilder_Build_NoInstructions(t *testing.T) {
	pub, _ := newKeypair(t)
	_, err := tx.NewBuilder().
		SetFeePayer(pub).
		SetRecentBlockhash(testBlockhash).
		Build(context.Background())
	if err == nil {
		t.Fatal("expected error for empty instruction list")
	}
}

func TestBuilder_Build_NilInstruction(t *testing.T) {
	pub, _ := newKeypair(t)
	_, err := tx.NewBuilder().
		SetFeePayer(pub).
		SetRecentBlockhash(testBlockhash).
		Add(nil).
		Build(context.Background())
	if err == nil {
		t.Fatal("expected error for nil instruction")
	}
}

func TestBuilder_Build_SignerError(t *testing.T) {
	pub, _ := newKeypair(t)
	wantErr := errors.New("signing failed")
	_, err := tx.NewBuilder().
		SetFeePayer(pub).
		SetRecentBlockhash(testBlockhash).
		Add(newTransfer(pub, solana.PublicKey{9}, 1)).
		Sign(&errSigner{pub: pub, err: wantErr}).
		Build(context.Background())
	if err == nil {
		t.Fatal("expected error from signer")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v in chain, got %v", wantErr, err)
	}
}

func TestBuilder_Build_UnsignedWhenNoSigners(t *testing.T) {
	pub, _ := newKeypair(t)
	txn, err := tx.NewBuilder().
		SetFeePayer(pub).
		SetRecentBlockhash(testBlockhash).
		Add(newTransfer(pub, solana.PublicKey{9}, 1)).
		Build(context.Background())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	for i, sig := range txn.Signatures {
		if !sig.IsZero() {
			t.Errorf("slot %d: expected zero sig without signers", i)
		}
	}
}

func TestBuilder_Build_MultipleInstructions(t *testing.T) {
	pub, _ := newKeypair(t)
	txn, err := tx.NewBuilder().
		SetFeePayer(pub).
		SetRecentBlockhash(testBlockhash).
		Add(newTransfer(pub, solana.PublicKey{9}, 1)).
		Add(newTransfer(pub, solana.PublicKey{9}, 2)).
		Build(context.Background())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if got := len(txn.Message.Instructions); got != 2 {
		t.Fatalf("expected 2 instructions, got %d", got)
	}
}

func TestBuilder_Build_ResultIsSerializable(t *testing.T) {
	pub, priv := newKeypair(t)
	signer := &localSigner{pub: pub, priv: priv}

	txn, err := tx.NewBuilder().
		SetFeePayer(pub).
		SetRecentBlockhash(testBlockhash).
		Add(newTransfer(pub, solana.PublicKey{9}, 1_000_000)).
		Sign(signer).
		Build(context.Background())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	wire, err := txn.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	recovered, err := solana.UnmarshalTransaction(wire)
	if err != nil {
		t.Fatalf("UnmarshalTransaction: %v", err)
	}
	if err := recovered.VerifySignatures(); err != nil {
		t.Fatalf("VerifySignatures: %v", err)
	}
}

// --- SetComputeUnitLimit / SetComputeUnitPrice ---

func TestSetComputeUnitLimit_Encoding(t *testing.T) {
	ix := tx.SetComputeUnitLimit(200_000)
	data, err := ix.Data()
	if err != nil {
		t.Fatalf("Data: %v", err)
	}
	if len(data) != 5 {
		t.Fatalf("want 5 bytes, got %d", len(data))
	}
	if data[0] != 2 {
		t.Errorf("discriminant: got %d, want 2", data[0])
	}
	if got := binary.LittleEndian.Uint32(data[1:]); got != 200_000 {
		t.Errorf("units: got %d, want 200000", got)
	}
	if ix.ProgramID() != tx.ComputeBudgetProgramID {
		t.Error("program id mismatch")
	}
	if ix.Accounts() != nil {
		t.Error("expected nil accounts")
	}
}

func TestSetComputeUnitPrice_Encoding(t *testing.T) {
	ix := tx.SetComputeUnitPrice(5_000)
	data, err := ix.Data()
	if err != nil {
		t.Fatalf("Data: %v", err)
	}
	if len(data) != 9 {
		t.Fatalf("want 9 bytes, got %d", len(data))
	}
	if data[0] != 3 {
		t.Errorf("discriminant: got %d, want 3", data[0])
	}
	if got := binary.LittleEndian.Uint64(data[1:]); got != 5_000 {
		t.Errorf("price: got %d, want 5000", got)
	}
}

// --- ToMessageAddressTableLookup ---

func TestToMessageAddressTableLookup(t *testing.T) {
	key := solana.PublicKey{99}
	alt := tx.AddressLookupTable{
		AccountKey:      key,
		WritableIndexes: []uint8{0, 2},
		ReadonlyIndexes: []uint8{1},
	}
	got := tx.ToMessageAddressTableLookup(alt)
	if got.AccountKey != key {
		t.Error("AccountKey mismatch")
	}
	if len(got.WritableIndexes) != 2 {
		t.Errorf("WritableIndexes len: %d", len(got.WritableIndexes))
	}
	if len(got.ReadonlyIndexes) != 1 {
		t.Errorf("ReadonlyIndexes len: %d", len(got.ReadonlyIndexes))
	}
}

// --- BuildV0 ---

func TestBuilder_BuildV0_Happy(t *testing.T) {
	pub, priv := newKeypair(t)
	recipient := solana.PublicKey{9, 9, 9}
	signer := &localSigner{pub: pub, priv: priv}

	// One ALT entry: the recipient — should be routed through the table.
	tableKey := solana.PublicKey{0xAB}
	tables := []tx.LoadedAddressLookupTable{
		{AccountKey: tableKey, Addresses: []solana.PublicKey{recipient}},
	}

	txn, err := tx.NewBuilder().
		SetFeePayer(pub).
		SetRecentBlockhash(testBlockhash).
		Add(newTransfer(pub, recipient, 1_000_000)).
		Sign(signer).
		BuildV0(context.Background(), tables)
	if err != nil {
		t.Fatalf("BuildV0: %v", err)
	}
	if txn.Message.Version != solana.MessageVersion0 {
		t.Errorf("expected v0 message, got %v", txn.Message.Version)
	}
	if len(txn.Message.AddressTableLookups) != 1 {
		t.Fatalf("expected 1 ALT lookup, got %d", len(txn.Message.AddressTableLookups))
	}
	if txn.Message.AddressTableLookups[0].AccountKey != tableKey {
		t.Errorf("ALT key mismatch")
	}
}

func TestBuilder_BuildV0_NoTables(t *testing.T) {
	pub, priv := newKeypair(t)
	signer := &localSigner{pub: pub, priv: priv}

	txn, err := tx.NewBuilder().
		SetFeePayer(pub).
		SetRecentBlockhash(testBlockhash).
		Add(newTransfer(pub, solana.PublicKey{9}, 1)).
		Sign(signer).
		BuildV0(context.Background(), nil)
	if err != nil {
		t.Fatalf("BuildV0 with no tables: %v", err)
	}
	if txn.Message.Version != solana.MessageVersion0 {
		t.Errorf("expected v0 message")
	}
	if len(txn.Message.AddressTableLookups) != 0 {
		t.Errorf("expected no lookups with empty tables")
	}
}

func TestBuilder_BuildV0_SignerNotMovedToTable(t *testing.T) {
	pub, priv := newKeypair(t)
	signer := &localSigner{pub: pub, priv: priv}

	// Payer is in the table — it must still be static (signers can't be in ALT).
	tables := []tx.LoadedAddressLookupTable{
		{AccountKey: solana.PublicKey{0xAB}, Addresses: []solana.PublicKey{pub}},
	}

	txn, err := tx.NewBuilder().
		SetFeePayer(pub).
		SetRecentBlockhash(testBlockhash).
		Add(newTransfer(pub, solana.PublicKey{9}, 1)).
		Sign(signer).
		BuildV0(context.Background(), tables)
	if err != nil {
		t.Fatalf("BuildV0: %v", err)
	}
	// Payer must appear in static AccountKeys.
	found := false
	for _, k := range txn.Message.AccountKeys {
		if k == pub {
			found = true
			break
		}
	}
	if !found {
		t.Error("payer must remain in static account keys, not table")
	}
}
