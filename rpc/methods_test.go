package rpc_test

import (
	"context"
	"encoding/json"
	"testing"

	solana "github.com/MevYu/solana-go"
	"github.com/MevYu/solana-go/rpc"
	"github.com/MevYu/solana-go/jsonrpc"
	"github.com/MevYu/solana-go/internal/testutil"
)

// contextEnvelope wraps a value in the Solana RPC context envelope:
// {"context": {"slot": N}, "value": V}
func contextEnvelope(slot uint64, value any) map[string]any {
	return map[string]any{
		"context": map[string]any{"slot": slot},
		"value":   value,
	}
}

// --- GetBalance ---

func TestGetBalance_Happy(t *testing.T) {
	srv := testutil.NewMockRPCServer(t, func(method string, _ json.RawMessage) (any, error) {
		if method != "getBalance" {
			t.Errorf("unexpected method %q", method)
		}
		return contextEnvelope(100, uint64(500_000_000)), nil
	})
	c := rpc.NewClientWith(srv.URL)
	res, err := c.GetBalance(context.Background(), solana.PublicKey{1})
	if err != nil {
		t.Fatalf("GetBalance: %v", err)
	}
	if res.Value != 500_000_000 {
		t.Errorf("Value = %d, want 500_000_000", res.Value)
	}
	if res.Slot != 100 {
		t.Errorf("Slot = %d, want 100", res.Slot)
	}
}

func TestGetBalance_ZeroBalance(t *testing.T) {
	srv := testutil.NewMockRPCServer(t, func(_ string, _ json.RawMessage) (any, error) {
		return contextEnvelope(1, uint64(0)), nil
	})
	c := rpc.NewClientWith(srv.URL)
	res, err := c.GetBalance(context.Background(), solana.PublicKey{})
	if err != nil {
		t.Fatalf("GetBalance: %v", err)
	}
	if res.Value != 0 {
		t.Errorf("Value = %d, want 0", res.Value)
	}
}

// --- GetSlot ---

func TestGetSlot_Happy(t *testing.T) {
	srv := testutil.NewMockRPCServer(t, func(_ string, _ json.RawMessage) (any, error) {
		return uint64(12345), nil
	})
	c := rpc.NewClientWith(srv.URL)
	slot, err := c.GetSlot(context.Background())
	if err != nil {
		t.Fatalf("GetSlot: %v", err)
	}
	if slot != 12345 {
		t.Errorf("slot = %d, want 12345", slot)
	}
}

// --- GetBlockHeight ---

func TestGetBlockHeight_Happy(t *testing.T) {
	srv := testutil.NewMockRPCServer(t, func(_ string, _ json.RawMessage) (any, error) {
		return uint64(999999), nil
	})
	c := rpc.NewClientWith(srv.URL)
	h, err := c.GetBlockHeight(context.Background())
	if err != nil {
		t.Fatalf("GetBlockHeight: %v", err)
	}
	if h != 999999 {
		t.Errorf("height = %d, want 999999", h)
	}
}

// --- GetLatestBlockhash ---

func TestGetLatestBlockhash_Happy(t *testing.T) {
	bh := solana.Hash{1, 2, 3, 4, 5}
	bhStr := bh.String()
	srv := testutil.NewMockRPCServer(t, func(_ string, _ json.RawMessage) (any, error) {
		return contextEnvelope(200, map[string]any{
			"blockhash":            bhStr,
			"lastValidBlockHeight": uint64(888),
		}), nil
	})
	c := rpc.NewClientWith(srv.URL)
	res, err := c.GetLatestBlockhash(context.Background())
	if err != nil {
		t.Fatalf("GetLatestBlockhash: %v", err)
	}
	if res.Blockhash != bh {
		t.Errorf("Blockhash = %s, want %s", res.Blockhash, bh)
	}
	if res.LastValidBlockHeight != 888 {
		t.Errorf("LastValidBlockHeight = %d, want 888", res.LastValidBlockHeight)
	}
	if res.Slot != 200 {
		t.Errorf("Slot = %d, want 200", res.Slot)
	}
}

// --- GetAccountInfo ---

func TestGetAccountInfo_Happy(t *testing.T) {
	owner := solana.PublicKey{5, 6, 7}
	srv := testutil.NewMockRPCServer(t, func(_ string, _ json.RawMessage) (any, error) {
		return contextEnvelope(50, map[string]any{
			"lamports":   uint64(2_000_000),
			"owner":      owner.String(),
			"data":       []string{"", "base64"},
			"executable": false,
			"rentEpoch":  uint64(0),
			"space":      uint64(0),
		}), nil
	})
	c := rpc.NewClientWith(srv.URL)
	res, err := c.GetAccountInfo(context.Background(), solana.PublicKey{1})
	if err != nil {
		t.Fatalf("GetAccountInfo: %v", err)
	}
	if res.Account == nil {
		t.Fatal("expected non-nil account")
	}
	if res.Account.Lamports != 2_000_000 {
		t.Errorf("Lamports = %d, want 2_000_000", res.Account.Lamports)
	}
	if res.Account.Owner != owner {
		t.Errorf("Owner = %s, want %s", res.Account.Owner, owner)
	}
	if res.Slot != 50 {
		t.Errorf("Slot = %d, want 50", res.Slot)
	}
}

func TestGetAccountInfo_NullAccount(t *testing.T) {
	srv := testutil.NewMockRPCServer(t, func(_ string, _ json.RawMessage) (any, error) {
		// Solana returns null for accounts that don't exist
		return contextEnvelope(50, nil), nil
	})
	c := rpc.NewClientWith(srv.URL)
	res, err := c.GetAccountInfo(context.Background(), solana.PublicKey{99})
	if err != nil {
		t.Fatalf("GetAccountInfo: %v", err)
	}
	if res.Account != nil {
		t.Errorf("expected nil account for null value, got %+v", res.Account)
	}
}

// --- SendRawTransaction ---

func TestSendRawTransaction_Happy(t *testing.T) {
	wantSig := solana.Signature{1, 2, 3, 4}
	var gotMethod string
	var gotParams json.RawMessage
	srv := testutil.NewMockRPCServer(t, func(method string, params json.RawMessage) (any, error) {
		gotMethod = method
		gotParams = params
		return wantSig.String(), nil
	})
	c := rpc.NewClientWith(srv.URL)
	sig, err := c.SendRawTransaction(context.Background(), []byte{0, 1, 2, 3})
	if err != nil {
		t.Fatalf("SendRawTransaction: %v", err)
	}
	if gotMethod != "sendTransaction" {
		t.Errorf("method = %q, want sendTransaction", gotMethod)
	}
	if sig != wantSig {
		t.Errorf("sig = %s, want %s", sig, wantSig)
	}
	// The default-encoding (base64) injected by SendTxCfg.MarshalJSON
	// must reach the wire even when the caller passes no cfg.
	var arr []json.RawMessage
	if err := json.Unmarshal(gotParams, &arr); err != nil {
		t.Fatalf("decode params: %v", err)
	}
	if len(arr) != 2 {
		t.Fatalf("len(params) = %d, want 2", len(arr))
	}
	var optsObj map[string]any
	if err := json.Unmarshal(arr[1], &optsObj); err != nil {
		t.Fatalf("decode opts: %v", err)
	}
	if optsObj["encoding"] != "base64" {
		t.Errorf("encoding = %v, want base64", optsObj["encoding"])
	}
}

// --- GetEpochInfo ---

func TestGetEpochInfo_Happy(t *testing.T) {
	srv := testutil.NewMockRPCServer(t, func(_ string, _ json.RawMessage) (any, error) {
		return map[string]any{
			"absoluteSlot":     uint64(100000),
			"blockHeight":      uint64(99000),
			"epoch":            uint64(5),
			"slotIndex":        uint64(500),
			"slotsInEpoch":     uint64(432000),
			"transactionCount": uint64(1_000_000),
		}, nil
	})
	c := rpc.NewClientWith(srv.URL)
	info, err := c.GetEpochInfo(context.Background())
	if err != nil {
		t.Fatalf("GetEpochInfo: %v", err)
	}
	if info.AbsoluteSlot != 100000 {
		t.Errorf("AbsoluteSlot = %d, want 100000", info.AbsoluteSlot)
	}
	if info.Epoch != 5 {
		t.Errorf("Epoch = %d, want 5", info.Epoch)
	}
	if info.TransactionCount == nil || *info.TransactionCount != 1_000_000 {
		t.Errorf("TransactionCount mismatch")
	}
}

// --- GetInflationRate ---

func TestGetInflationRate_Happy(t *testing.T) {
	srv := testutil.NewMockRPCServer(t, func(_ string, _ json.RawMessage) (any, error) {
		return map[string]any{
			"total":      0.08,
			"validator":  0.076,
			"foundation": 0.004,
			"epoch":      uint64(100),
		}, nil
	})
	c := rpc.NewClientWith(srv.URL)
	rate, err := c.GetInflationRate(context.Background())
	if err != nil {
		t.Fatalf("GetInflationRate: %v", err)
	}
	if rate.Total != 0.08 {
		t.Errorf("Total = %f, want 0.08", rate.Total)
	}
	if rate.Epoch != 100 {
		t.Errorf("Epoch = %d, want 100", rate.Epoch)
	}
}

// --- GetMinimumBalanceForRentExemption ---

func TestGetMinimumBalanceForRentExemption_Happy(t *testing.T) {
	srv := testutil.NewMockRPCServer(t, func(_ string, _ json.RawMessage) (any, error) {
		return uint64(2_039_280), nil
	})
	c := rpc.NewClientWith(srv.URL)
	lamports, err := c.GetMinimumBalanceForRentExemption(context.Background(), 165)
	if err != nil {
		t.Fatalf("GetMinimumBalanceForRentExemption: %v", err)
	}
	if lamports != 2_039_280 {
		t.Errorf("lamports = %d, want 2_039_280", lamports)
	}
}

// --- GetSignatureStatuses ---

func TestGetSignatureStatuses_Happy(t *testing.T) {
	confirmations := uint64(10)
	srv := testutil.NewMockRPCServer(t, func(_ string, _ json.RawMessage) (any, error) {
		return contextEnvelope(300, []any{
			map[string]any{
				"slot":               uint64(250),
				"confirmations":      &confirmations,
				"err":                nil,
				"confirmationStatus": "finalized",
			},
		}), nil
	})
	c := rpc.NewClientWith(srv.URL)
	sig := solana.Signature{1}
	res, err := c.GetSignatureStatuses(context.Background(), []solana.Signature{sig})
	if err != nil {
		t.Fatalf("GetSignatureStatuses: %v", err)
	}
	if len(res.Statuses) != 1 {
		t.Fatalf("Statuses len = %d, want 1", len(res.Statuses))
	}
	st := res.Statuses[0]
	if st == nil {
		t.Fatal("status[0] is nil")
	}
	if st.Slot != 250 {
		t.Errorf("Slot = %d, want 250", st.Slot)
	}
	if st.ConfirmationStatus != "finalized" {
		t.Errorf("ConfirmationStatus = %q, want finalized", st.ConfirmationStatus)
	}
}

func TestGetSignatureStatuses_UnconfirmedNil(t *testing.T) {
	srv := testutil.NewMockRPCServer(t, func(_ string, _ json.RawMessage) (any, error) {
		// Solana returns null entries for unknown signatures
		return contextEnvelope(300, []any{nil}), nil
	})
	c := rpc.NewClientWith(srv.URL)
	sig := solana.Signature{2}
	res, err := c.GetSignatureStatuses(context.Background(), []solana.Signature{sig})
	if err != nil {
		t.Fatalf("GetSignatureStatuses: %v", err)
	}
	if len(res.Statuses) != 1 {
		t.Fatalf("Statuses len = %d, want 1", len(res.Statuses))
	}
	if res.Statuses[0] != nil {
		t.Errorf("expected nil status for unknown sig, got %+v", res.Statuses[0])
	}
}

// --- GetSignaturesForAddress ---

func TestGetSignaturesForAddress_Happy(t *testing.T) {
	sig := solana.Signature{9, 8, 7}
	srv := testutil.NewMockRPCServer(t, func(_ string, _ json.RawMessage) (any, error) {
		return []any{
			map[string]any{
				"signature":          sig.String(),
				"slot":               uint64(400),
				"err":                nil,
				"confirmationStatus": "finalized",
			},
		}, nil
	})
	c := rpc.NewClientWith(srv.URL)
	results, err := c.GetSignaturesForAddress(context.Background(), solana.PublicKey{1})
	if err != nil {
		t.Fatalf("GetSignaturesForAddress: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len = %d, want 1", len(results))
	}
	if results[0].Slot != 400 {
		t.Errorf("Slot = %d, want 400", results[0].Slot)
	}
	if results[0].Signature != sig.String() {
		t.Errorf("Signature = %q, want %q", results[0].Signature, sig.String())
	}
}

// --- GetHealth ---

func TestGetHealth_Happy(t *testing.T) {
	srv := testutil.NewMockRPCServer(t, func(_ string, _ json.RawMessage) (any, error) {
		return "ok", nil
	})
	c := rpc.NewClientWith(srv.URL)
	health, err := c.GetHealth(context.Background())
	if err != nil {
		t.Fatalf("GetHealth: %v", err)
	}
	if health != "ok" {
		t.Errorf("health = %q, want ok", health)
	}
}

// --- GetVersion ---

func TestGetVersion_Happy(t *testing.T) {
	srv := testutil.NewMockRPCServer(t, func(_ string, _ json.RawMessage) (any, error) {
		return map[string]any{
			"solana-core": "1.18.0",
			"feature-set": uint32(1234567890),
		}, nil
	})
	c := rpc.NewClientWith(srv.URL)
	ver, err := c.GetVersion(context.Background())
	if err != nil {
		t.Fatalf("GetVersion: %v", err)
	}
	if ver.SolanaCore != "1.18.0" {
		t.Errorf("SolanaCore = %q, want 1.18.0", ver.SolanaCore)
	}
}

// --- IsBlockhashValid ---

func TestIsBlockhashValid_True(t *testing.T) {
	srv := testutil.NewMockRPCServer(t, func(_ string, _ json.RawMessage) (any, error) {
		return contextEnvelope(500, true), nil
	})
	c := rpc.NewClientWith(srv.URL)
	valid, err := c.IsBlockhashValid(context.Background(), solana.Hash{1})
	if err != nil {
		t.Fatalf("IsBlockhashValid: %v", err)
	}
	if !valid {
		t.Error("expected valid = true")
	}
}

func TestIsBlockhashValid_False(t *testing.T) {
	srv := testutil.NewMockRPCServer(t, func(_ string, _ json.RawMessage) (any, error) {
		return contextEnvelope(500, false), nil
	})
	c := rpc.NewClientWith(srv.URL)
	valid, err := c.IsBlockhashValid(context.Background(), solana.Hash{99})
	if err != nil {
		t.Fatalf("IsBlockhashValid: %v", err)
	}
	if valid {
		t.Error("expected valid = false")
	}
}

// --- GetMinimumLedgerSlot ---

func TestMinimumLedgerSlot_Happy(t *testing.T) {
	srv := testutil.NewMockRPCServer(t, func(_ string, _ json.RawMessage) (any, error) {
		return uint64(1000), nil
	})
	c := rpc.NewClientWith(srv.URL)
	slot, err := c.MinimumLedgerSlot(context.Background())
	if err != nil {
		t.Fatalf("MinimumLedgerSlot: %v", err)
	}
	if slot != 1000 {
		t.Errorf("slot = %d, want 1000", slot)
	}
}

// --- NewClient vs New ---

func TestNewClient_IsAlias(t *testing.T) {
	// Both constructors should produce a working rpc.
	srv := testutil.NewMockRPCServer(t, func(_ string, _ json.RawMessage) (any, error) {
		return uint64(42), nil
	})
	c1 := rpc.NewClient(srv.URL, jsonrpc.Config{})
	c2 := rpc.NewClientWith(srv.URL)

	s1, err := c1.GetSlot(context.Background())
	if err != nil {
		t.Fatalf("c1.GetSlot: %v", err)
	}
	s2, err := c2.GetSlot(context.Background())
	if err != nil {
		t.Fatalf("c2.GetSlot: %v", err)
	}
	if s1 != 42 || s2 != 42 {
		t.Errorf("slots = %d, %d; want 42, 42", s1, s2)
	}
}

// --- input-size validation ---

func TestGetMultipleAccounts_OverLimit(t *testing.T) {
	c := rpc.NewClientWith("http://invalid.invalid")
	addrs := make([]solana.PublicKey, rpc.MaxGetMultipleAccountsAddresses+1)
	if _, err := c.GetMultipleAccounts(context.Background(), addrs); err == nil {
		t.Fatal("want error for over-limit input, got nil")
	}
}

func TestGetSignatureStatuses_OverLimit(t *testing.T) {
	c := rpc.NewClientWith("http://invalid.invalid")
	sigs := make([]solana.Signature, rpc.MaxGetSignatureStatusesSignatures+1)
	if _, err := c.GetSignatureStatuses(context.Background(), sigs); err == nil {
		t.Fatal("want error for over-limit input, got nil")
	}
}

func TestGetRecentPrioritizationFees_OverLimit(t *testing.T) {
	c := rpc.NewClientWith("http://invalid.invalid")
	addrs := make([]solana.PublicKey, rpc.MaxGetRecentPrioritizationFeesAddresses+1)
	if _, err := c.GetRecentPrioritizationFees(context.Background(), addrs); err == nil {
		t.Fatal("want error for over-limit input, got nil")
	}
}

// --- SimulateTransaction wire decode ---

// TestSimulateResult_UnmarshalReturnData verifies that
// SimulateResult.ReturnData.ProgramID is decoded from a base58 string
// by PublicKey.UnmarshalJSON, without any wire-shape struct middleman.
// This is the contract that lets us drop the simulateValue alias.
func TestSimulateResult_UnmarshalReturnData(t *testing.T) {
	wantPid := solana.MustPublicKey("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")
	body := []byte(`{
		"err": null,
		"logs": ["Program log: hello"],
		"unitsConsumed": 2200,
		"accounts": null,
		"returnData": {"programId": "` + wantPid.String() + `", "data": ["AAEC", "base64"]}
	}`)
	var res rpc.SimulateResult
	if err := json.Unmarshal(body, &res); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if res.ReturnData == nil {
		t.Fatal("ReturnData == nil")
	}
	if res.ReturnData.ProgramID != wantPid {
		t.Errorf("ProgramID = %s, want %s", res.ReturnData.ProgramID, wantPid)
	}
	if len(res.Logs) != 1 || res.Logs[0] != "Program log: hello" {
		t.Errorf("Logs = %v", res.Logs)
	}
	if res.UnitsConsumed == nil || *res.UnitsConsumed != 2200 {
		t.Errorf("UnitsConsumed = %v, want 2200", res.UnitsConsumed)
	}
}
