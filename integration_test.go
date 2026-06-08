//go:build integration

// Package solana_test integration tests run against a live Solana
// validator. They are gated behind the "integration" build tag so
// that the default `go test ./...` does not require network access
// or a running validator.
//
// Running:
//
//	go test -tags=integration -run=Integration -count=1 ./...
//
// By default the tests target http://localhost:8899 (the
// solana-test-validator default); override with the SOLANA_RPC_URL
// environment variable to point at a devnet or custom endpoint.
// Tests skip themselves if the endpoint is not reachable, so CI can
// invoke them unconditionally.
package solana_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/MevYu/solana-go"
	rpc "github.com/MevYu/solana-go/jsonrpc"
	"github.com/MevYu/solana-go/programs/system"
	client "github.com/MevYu/solana-go/rpc"
)

// defaultEndpoint is the solana-test-validator default RPC URL.
const defaultEndpoint = "http://localhost:8899"

// newIntegrationClient returns a Client pointed at the integration
// endpoint, or skips the test if the endpoint is not reachable.
// Reachability is probed by a short-timeout GetSlot call, which
// also warms up the HTTP connection.
func newIntegrationClient(t *testing.T) *client.Client {
	t.Helper()
	endpoint := os.Getenv("SOLANA_RPC_URL")
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	c := client.NewClient(endpoint, rpc.Config{})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := c.GetSlot(ctx); err != nil {
		t.Skipf("no validator reachable at %s: %v", endpoint, err)
	}
	return c
}

// TestIntegration_GetSlot is the simplest possible smoke test:
// fetch the current slot and verify it is non-zero. If this fails,
// nothing else will work.
func TestIntegration_GetSlot(t *testing.T) {
	c := newIntegrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	slot, err := c.GetSlot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if slot == 0 {
		t.Error("slot should be > 0 on a live validator")
	}
}

// TestIntegration_GetLatestBlockhash verifies that blockhash
// retrieval returns a non-zero hash that parses correctly. This
// exercises the full rpc+codec+type path that
// SendAndConfirmTransaction relies on.
func TestIntegration_GetLatestBlockhash(t *testing.T) {
	c := newIntegrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bh, err := c.GetLatestBlockhash(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if bh.Blockhash.IsZero() {
		t.Error("blockhash should not be zero")
	}
	if bh.LastValidBlockHeight == 0 {
		t.Error("LastValidBlockHeight should be > 0")
	}
}

// TestIntegration_AirdropAndTransfer exercises the end-to-end
// happy path: airdrop lamports to a fresh keypair, send a transfer
// to a second fresh keypair via SendAndConfirmTransaction, verify
// both balances, verify the transfer signature confirms.
//
// This test depends on the validator having a funded faucet
// (solana-test-validator always does). Devnet will occasionally
// rate-limit the airdrop; the test skips gracefully in that case.
func TestIntegration_AirdropAndTransfer(t *testing.T) {
	c := newIntegrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	payer, err := solana.NewEd25519Keypair()
	if err != nil {
		t.Fatal(err)
	}
	recipient, err := solana.NewEd25519Keypair()
	if err != nil {
		t.Fatal(err)
	}

	// Airdrop 1 SOL to the payer. On devnet this sometimes fails
	// with a rate-limit error; skip in that case so CI remains
	// green when a shared endpoint is throttled.
	airdropSig, err := c.RequestAirdrop(ctx, payer.PublicKey(), 1_000_000_000)
	if err != nil {
		if rpc.IsRateLimited(err) {
			t.Skipf("airdrop rate-limited: %v", err)
		}
		t.Fatal(err)
	}
	t.Logf("airdrop signature: %s", airdropSig)

	// Wait for the airdrop to confirm by polling the balance.
	var payerBalance uint64
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		res, err := c.GetBalance(ctx, payer.PublicKey())
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		if res.Value > 0 {
			payerBalance = res.Value
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if payerBalance == 0 {
		t.Fatal("airdrop did not credit payer within 30 seconds")
	}

	// Send 0.1 SOL from payer to recipient via SendAndConfirmTransaction.
	const transferLamports = 100_000_000
	build := func(ctx context.Context, bh solana.Hash) (*solana.Transaction, error) {
		ix := system.NewTransfer(payer.PublicKey(), recipient.PublicKey(), transferLamports)
		msg, err := solana.NewMessage(payer.PublicKey(), []solana.Instruction{ix}, bh)
		if err != nil {
			return nil, err
		}
		tx := solana.NewTransaction(*msg)
		if err := tx.Sign(ctx, payer); err != nil {
			return nil, err
		}
		return tx, nil
	}
	transferSig, err := c.SendAndConfirmTransaction(ctx, build,
		client.WithConfirmTimeout(45*time.Second),
		client.WithPollInterval(500*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("SendAndConfirmTransaction: %v", err)
	}
	t.Logf("transfer signature: %s", transferSig)

	// Verify recipient balance equals the transfer amount.
	rbal, err := c.GetBalance(ctx, recipient.PublicKey())
	if err != nil {
		t.Fatal(err)
	}
	if rbal.Value != transferLamports {
		t.Errorf("recipient balance = %d, want %d", rbal.Value, transferLamports)
	}

	// Verify the signature status via GetSignatureStatuses.
	statuses, err := c.GetSignatureStatuses(ctx, []solana.Signature{transferSig})
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses.Statuses) != 1 || statuses.Statuses[0] == nil {
		t.Fatalf("transfer signature not found in status lookup")
	}
	if statuses.Statuses[0].Err != nil {
		t.Errorf("transfer failed on chain: %v", statuses.Statuses[0].Err)
	}
}

// TestIntegration_PriorityFeeQuery exercises the
// GetRecentPrioritizationFees path end-to-end, summarising the
// returned window via client.PriorityFeeStatsFromFees.
func TestIntegration_PriorityFeeQuery(t *testing.T) {
	c := newIntegrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fees, err := c.GetRecentPrioritizationFees(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	stats := client.PriorityFeeStatsFromFees(fees)
	// On test-validator the fees are typically zero, so we only
	// assert the call round-trips and returns a non-nil struct.
	if stats == nil {
		t.Fatal("stats should not be nil")
	}
	t.Logf("p50=%d p75=%d p95=%d max=%d samples=%d",
		stats.P50, stats.P75, stats.P95, stats.Max, stats.Samples)
}
