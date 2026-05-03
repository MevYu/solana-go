package rpc_test

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	solana "github.com/MevYu/solana-go"
	"github.com/MevYu/solana-go/internal/testutil"
	"github.com/MevYu/solana-go/jsonrpc"
	"github.com/MevYu/solana-go/rpc"
)

// multiMethodHandler dispatches JSON-RPC method calls to a map of
// per-method handlers for use with testutil.NewMockRPCServer. A value
// in the map may be a static `any` result or a `func() any` that
// produces a fresh result for every call.
func multiMethodHandler(t testing.TB, handlers map[string]any) testutil.RPCHandler {
	t.Helper()
	return func(method string, _ json.RawMessage) (any, error) {
		raw, ok := handlers[method]
		if !ok {
			t.Errorf("unexpected method: %s", method)
			return nil, nil
		}
		if fn, ok := raw.(func() any); ok {
			return fn(), nil
		}
		return raw, nil
	}
}

// buildTestTx constructs a minimal, signed legacy transaction suitable
// for driving the send/confirm happy-path tests.
func buildTestTx(t *testing.T, blockhash solana.Hash) *solana.Transaction {
	t.Helper()
	kp, err := solana.NewEd25519Keypair()
	if err != nil {
		t.Fatal(err)
	}
	msg := solana.Message{
		Version: solana.MessageVersionLegacy,
		Header: solana.MessageHeader{
			NumRequiredSignatures:       1,
			NumReadonlyUnsignedAccounts: 1,
		},
		AccountKeys:     []solana.PublicKey{kp.PublicKey(), solana.SystemProgramID},
		RecentBlockhash: blockhash,
		Instructions: []solana.CompiledInstruction{
			{
				ProgramIDIndex: 1,
				Accounts:       solana.Uint8Slice{0},
				Data:           []byte{0x00},
			},
		},
	}
	tx := solana.NewTransaction(msg)
	if err := tx.Sign(context.Background(), kp); err != nil {
		t.Fatal(err)
	}
	return tx
}

func TestSendAndConfirmTransaction_Happy(t *testing.T) {
	sigStr := (solana.Signature{}).String()

	var pollCount atomic.Int32
	handlers := map[string]any{
		"getLatestBlockhash": map[string]any{
			"context": map[string]any{"slot": uint64(100)},
			"value": map[string]any{
				"blockhash":            "HUhwFvKfmVhZ5AbkqK7N2Kt3i6hPz3jQeByN6wA9RvXb",
				"lastValidBlockHeight": uint64(1000),
			},
		},
		"sendTransaction": sigStr,
		"getSignatureStatuses": func() any {
			// First poll: "processed"; second poll: "confirmed".
			n := pollCount.Add(1)
			status := "processed"
			if n >= 2 {
				status = "confirmed"
			}
			return map[string]any{
				"context": map[string]any{"slot": uint64(110)},
				"value": []any{
					map[string]any{
						"slot":               uint64(105),
						"confirmations":      uint64(2),
						"err":                nil,
						"confirmationStatus": status,
					},
				},
			}
		},
		"getBlockHeight": uint64(500),
	}
	srv := testutil.NewMockRPCServer(t, multiMethodHandler(t, handlers))
	c := rpc.NewClientWith(srv.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	builder := func(ctx context.Context, bh solana.Hash) (*solana.Transaction, error) {
		return buildTestTx(t, bh), nil
	}

	_, err := c.SendAndConfirmTransaction(ctx, builder,
		rpc.WithPollInterval(10*time.Millisecond),
		rpc.WithConfirmTimeout(3*time.Second),
	)
	if err != nil {
		t.Fatalf("SendAndConfirmTransaction: %v", err)
	}
	if pollCount.Load() < 2 {
		t.Errorf("expected at least 2 polls, got %d", pollCount.Load())
	}
}

func TestSendAndConfirmTransaction_NilBuilder(t *testing.T) {
	c := rpc.NewClientWith("https://example.com", jsonrpc.Config{})
	_, err := c.SendAndConfirmTransaction(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil builder")
	}
}

func TestSendAndConfirmSignedTransaction_Happy(t *testing.T) {
	sigStr := (solana.Signature{}).String()

	handlers := map[string]any{
		"sendTransaction": sigStr,
		"getSignatureStatuses": map[string]any{
			"context": map[string]any{"slot": uint64(110)},
			"value": []any{
				map[string]any{
					"slot":               uint64(105),
					"confirmations":      uint64(5),
					"err":                nil,
					"confirmationStatus": "confirmed",
				},
			},
		},
	}
	srv := testutil.NewMockRPCServer(t, multiMethodHandler(t, handlers))
	c := rpc.NewClientWith(srv.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx := buildTestTx(t, solana.Hash{0x42})
	_, err := c.SendAndConfirmSignedTransaction(ctx, tx,
		rpc.WithPollInterval(10*time.Millisecond),
		rpc.WithConfirmTimeout(3*time.Second),
	)
	if err != nil {
		t.Fatalf("SendAndConfirmSignedTransaction: %v", err)
	}
}

func TestSendAndConfirmSignedTransaction_Nil(t *testing.T) {
	c := rpc.NewClientWith("https://example.com", jsonrpc.Config{})
	_, err := c.SendAndConfirmSignedTransaction(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil transaction")
	}
}
