package rpc_test

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"

	solana "github.com/MevYu/solana-go"
	"github.com/MevYu/solana-go/internal/testutil"
	"github.com/MevYu/solana-go/rpc"
)

// TestGetTransaction_Fixture_FullDecode exercises every sub-type that
// landed in the root package: TransactionMeta + InnerInstructions +
// PreTokenBalances / PostTokenBalances + Rewards + LoadedAddresses,
// the structural Transaction decode via (*Transaction).UnmarshalJSON,
// and the MessageVersion envelope.
//
// The fixture is built in Go so wire bytes are known-good — no
// hand-base64ed strings to drift.
func TestGetTransaction_Fixture_FullDecode(t *testing.T) {
	const systemProgramID = "11111111111111111111111111111111"
	const tokenProgramID = "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"

	systemProg := solana.MustPublicKey(systemProgramID)
	tokenProg := solana.MustPublicKey(tokenProgramID)
	mint := solana.MustPublicKey("So11111111111111111111111111111111111111112")
	owner := solana.MustPublicKey("9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM")

	// Build a deterministic signed transfer transaction.
	seed := make([]byte, ed25519.SeedSize)
	seed[0] = 0x42
	payer, err := solana.Ed25519KeypairFromSeed(seed)
	if err != nil {
		t.Fatal(err)
	}
	recipient := solana.PublicKey{0x03}

	msg, err := solana.NewMessage(
		payer.PublicKey(),
		[]solana.Instruction{&fixtureInstruction{
			programID: systemProg,
			accounts: []*solana.AccountMeta{
				{PublicKey: payer.PublicKey(), IsSigner: true, IsWritable: true},
				{PublicKey: recipient, IsWritable: true},
			},
			data: []byte{0x02, 0x00, 0x00, 0x00, 0xE8, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		}},
		solana.Hash{0xAB, 0xCD},
	)
	if err != nil {
		t.Fatalf("NewMessage: %v", err)
	}
	tx := solana.NewTransaction(*msg)
	if err := tx.Sign(context.Background(), payer); err != nil {
		t.Fatalf("Sign: %v", err)
	}
	wire, err := tx.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	wireB64 := base64.StdEncoding.EncodeToString(wire)

	// Build the JSON-RPC `result` body that mirrors what a real
	// Solana node returns for getTransaction with encoding=base64.
	commission := uint8(5)
	cusConsumed := uint64(5000)
	resultJSON := `{
		"slot": 250000000,
		"blockTime": 1700000000,
		"version": 0,
		"transaction": ["` + wireB64 + `", "base64"],
		"meta": {
			"err": null,
			"fee": 5000,
			"preBalances":  [1000000000, 0, 1],
			"postBalances": [999990000,  5000, 1],
			"logMessages": [
				"Program 11111111111111111111111111111111 invoke [1]",
				"Program 11111111111111111111111111111111 success"
			],
			"computeUnitsConsumed": 5000,
			"innerInstructions": [
				{
					"index": 0,
					"instructions": [
						{"programIdIndex": 1, "accounts": [0, 2], "data": "AQID"}
					]
				}
			],
			"preTokenBalances": [
				{
					"accountIndex": 2,
					"mint":   "` + mint.String() + `",
					"owner":  "` + owner.String() + `",
					"programId": "` + tokenProg.String() + `",
					"uiTokenAmount": {
						"amount": "1000000000",
						"decimals": 9,
						"uiAmount": 1.0,
						"uiAmountString": "1"
					}
				}
			],
			"postTokenBalances": [
				{
					"accountIndex": 2,
					"mint":   "` + mint.String() + `",
					"owner":  "` + owner.String() + `",
					"programId": "` + tokenProg.String() + `",
					"uiTokenAmount": {
						"amount": "1500000000",
						"decimals": 9,
						"uiAmount": 1.5,
						"uiAmountString": "1.5"
					}
				}
			],
			"rewards": [
				{
					"pubkey": "` + payer.PublicKey().String() + `",
					"lamports": -500,
					"postBalance": 999989500,
					"rewardType": "fee",
					"commission": 5
				}
			],
			"loadedAddresses": {
				"writable": ["` + mint.String() + `"],
				"readonly": ["` + owner.String() + `"]
			}
		}
	}`

	srv := testutil.NewMockRPCServer(t, func(method string, _ json.RawMessage) (any, error) {
		if method != "getTransaction" {
			t.Errorf("unexpected method %q", method)
		}
		return json.RawMessage(resultJSON), nil
	})
	c := rpc.NewClientWith(srv.URL)

	got, err := c.GetTransaction(context.Background(), tx.Signatures[0])
	if err != nil {
		t.Fatalf("GetTransaction: %v", err)
	}
	if got == nil {
		t.Fatal("nil result")
	}

	// --- envelope ---
	if got.Slot != 250000000 {
		t.Errorf("Slot = %d", got.Slot)
	}
	if got.BlockTime == nil || *got.BlockTime != 1700000000 {
		t.Errorf("BlockTime = %v", got.BlockTime)
	}
	if got.Version != solana.MessageVersion0 {
		t.Errorf("Version = %d, want %d", got.Version, solana.MessageVersion0)
	}

	// --- Transaction (structural decode) ---
	if got.Transaction == nil {
		t.Fatal("Transaction is nil; UnmarshalJSON should have decoded the [value, encoding] tuple structurally")
	}
	if len(got.Transaction.Signatures) != 1 || !got.Transaction.Signatures[0].Equal(tx.Signatures[0]) {
		t.Errorf("signature mismatch")
	}
	if !got.Transaction.Message.RecentBlockhash.Equal(tx.Message.RecentBlockhash) {
		t.Errorf("blockhash mismatch")
	}
	if len(got.Transaction.Message.Instructions) != 1 {
		t.Fatalf("instructions len = %d", len(got.Transaction.Message.Instructions))
	}

	// --- Meta ---
	m := got.Meta
	if m == nil {
		t.Fatal("Meta is nil")
	}
	if err := rpc.DecodeTransactionError(m.Err); err != nil {
		t.Errorf("Err should be success (nil), got %v", err)
	}
	if m.Fee != 5000 {
		t.Errorf("Fee = %d", m.Fee)
	}
	if len(m.PreBalances) != 3 || m.PreBalances[0] != 1000000000 {
		t.Errorf("PreBalances = %v", m.PreBalances)
	}
	if len(m.PostBalances) != 3 || m.PostBalances[1] != 5000 {
		t.Errorf("PostBalances = %v", m.PostBalances)
	}
	if len(m.LogMessages) != 2 {
		t.Errorf("LogMessages len = %d", len(m.LogMessages))
	}
	if m.ComputeUnitsConsumed == nil || *m.ComputeUnitsConsumed != cusConsumed {
		t.Errorf("ComputeUnitsConsumed = %v, want %d", m.ComputeUnitsConsumed, cusConsumed)
	}

	// --- InnerInstructions ---
	if len(m.InnerInstructions) != 1 {
		t.Fatalf("InnerInstructions len = %d", len(m.InnerInstructions))
	}
	if m.InnerInstructions[0].Index != 0 {
		t.Errorf("InnerInstruction.Index = %d", m.InnerInstructions[0].Index)
	}
	if len(m.InnerInstructions[0].Instructions) != 1 {
		t.Fatalf("inner.Instructions len = %d", len(m.InnerInstructions[0].Instructions))
	}
	inner := m.InnerInstructions[0].Instructions[0]
	if inner.ProgramIDIndex != 1 || len(inner.Accounts) != 2 || inner.Accounts[1] != 2 {
		t.Errorf("inner CompiledInstruction = %+v", inner)
	}

	// --- TokenBalances ---
	if len(m.PreTokenBalances) != 1 || len(m.PostTokenBalances) != 1 {
		t.Fatal("token balance slices wrong length")
	}
	pre := m.PreTokenBalances[0]
	post := m.PostTokenBalances[0]
	if pre.AccountIndex != 2 {
		t.Errorf("pre.AccountIndex = %d", pre.AccountIndex)
	}
	if !pre.Mint.Equal(mint) {
		t.Errorf("pre.Mint = %s", pre.Mint)
	}
	if !pre.Owner.Equal(owner) {
		t.Errorf("pre.Owner = %s", pre.Owner)
	}
	if !pre.ProgramID.Equal(tokenProg) {
		t.Errorf("pre.ProgramID = %s", pre.ProgramID)
	}
	if pre.UiTokenAmount.Decimals != 9 || pre.UiTokenAmount.Amount != "1000000000" {
		t.Errorf("pre.UiTokenAmount = %+v", pre.UiTokenAmount)
	}
	if post.UiTokenAmount.UiAmountString != "1.5" {
		t.Errorf("post.UiTokenAmount.UiAmountString = %q", post.UiTokenAmount.UiAmountString)
	}

	// --- Rewards (signed lamports) ---
	if len(m.Rewards) != 1 {
		t.Fatalf("Rewards len = %d", len(m.Rewards))
	}
	r := m.Rewards[0]
	if r.Lamports != -500 {
		t.Errorf("Reward.Lamports = %d, want -500 (signed int64)", r.Lamports)
	}
	if r.PostBalance != 999989500 {
		t.Errorf("Reward.PostBalance = %d", r.PostBalance)
	}
	if r.RewardType != "fee" {
		t.Errorf("Reward.RewardType = %q", r.RewardType)
	}
	if r.Commission == nil || *r.Commission != commission {
		t.Errorf("Reward.Commission = %v", r.Commission)
	}

	// --- LoadedAddresses ---
	if m.LoadedAddresses == nil {
		t.Fatal("LoadedAddresses is nil")
	}
	if len(m.LoadedAddresses.Writable) != 1 || !m.LoadedAddresses.Writable[0].Equal(mint) {
		t.Errorf("LoadedAddresses.Writable = %v", m.LoadedAddresses.Writable)
	}
	if len(m.LoadedAddresses.ReadOnly) != 1 || !m.LoadedAddresses.ReadOnly[0].Equal(owner) {
		t.Errorf("LoadedAddresses.ReadOnly = %v", m.LoadedAddresses.ReadOnly)
	}
}

// TestGetTransaction_Fixture_LegacyVersion confirms the "legacy"
// string form is accepted by the MessageVersion JSON envelope.
func TestGetTransaction_Fixture_LegacyVersion(t *testing.T) {
	// Build minimal legacy transaction.
	seed := make([]byte, ed25519.SeedSize)
	seed[0] = 0x11
	payer, _ := solana.Ed25519KeypairFromSeed(seed)
	msg, _ := solana.NewMessage(
		payer.PublicKey(),
		[]solana.Instruction{&fixtureInstruction{
			programID: solana.MustPublicKey("11111111111111111111111111111111"),
			accounts:  []*solana.AccountMeta{{PublicKey: payer.PublicKey(), IsSigner: true, IsWritable: true}},
			data:      []byte{0xFF},
		}},
		solana.Hash{0x10},
	)
	tx := solana.NewTransaction(*msg)
	_ = tx.Sign(context.Background(), payer)
	wire, _ := tx.Marshal()
	wireB64 := base64.StdEncoding.EncodeToString(wire)

	body := `{
		"slot": 1, "blockTime": null, "meta": null, "version": "legacy",
		"transaction": ["` + wireB64 + `", "base64"]
	}`
	srv := testutil.NewMockRPCServer(t, func(method string, _ json.RawMessage) (any, error) {
		return json.RawMessage(body), nil
	})
	c := rpc.NewClientWith(srv.URL)
	got, err := c.GetTransaction(context.Background(), tx.Signatures[0])
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != solana.MessageVersionLegacy {
		t.Errorf("Version = %d, want legacy(%d)", got.Version, solana.MessageVersionLegacy)
	}
	if got.Transaction == nil || len(got.Transaction.Signatures) != 1 {
		t.Errorf("structural decode failed")
	}
}

// TestGetTransaction_Fixture_ErrPath verifies the err-payload path
// flows through DecodeTransactionError without manual unmarshal.
func TestGetTransaction_Fixture_ErrPath(t *testing.T) {
	body := `{
		"slot": 1, "blockTime": null, "version": 0, "transaction": null,
		"meta": {
			"err": {"InstructionError": [0, "InvalidAccountData"]},
			"fee": 5000,
			"preBalances": [], "postBalances": []
		}
	}`
	srv := testutil.NewMockRPCServer(t, func(method string, _ json.RawMessage) (any, error) {
		return json.RawMessage(body), nil
	})
	c := rpc.NewClientWith(srv.URL)
	got, err := c.GetTransaction(context.Background(), solana.Signature{})
	if err != nil {
		t.Fatal(err)
	}
	decoded := rpc.DecodeTransactionError(got.Meta.Err)
	var ie *rpc.InstructionError
	if !errors.As(decoded, &ie) {
		t.Fatalf("expected *InstructionError, got %T (%v)", decoded, decoded)
	}
	if ie.Index != 0 || ie.Kind != "InvalidAccountData" {
		t.Errorf("got %+v", ie)
	}
}

// fixtureInstruction is the minimum surface needed for NewMessage in
// these tests — a typed Instruction without pulling in the system
// program package and risking import cycles.
type fixtureInstruction struct {
	programID solana.PublicKey
	accounts  []*solana.AccountMeta
	data      []byte
}

func (i *fixtureInstruction) ProgramID() solana.PublicKey     { return i.programID }
func (i *fixtureInstruction) Accounts() []*solana.AccountMeta { return i.accounts }
func (i *fixtureInstruction) Data() ([]byte, error)           { return i.data, nil }
