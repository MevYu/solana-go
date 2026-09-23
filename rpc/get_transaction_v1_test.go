package rpc_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"testing"

	solana "github.com/MevYu/solana-go"
	"github.com/MevYu/solana-go/internal/testutil"
	"github.com/MevYu/solana-go/rpc"
)

func TestGetTransactionV1_ExplicitConfig(t *testing.T) {
	wire, err := os.ReadFile("../testdata/v1_transaction.bin")
	if err != nil {
		t.Fatal(err)
	}
	const signature = "2psiRktzwwp24HJBUySfduJchgeKoHYMEoEK2FvzX4DMYyXxEtRqhGXmPH8YWDcY6a1J6LZDbbBfzrXf9ZYSCf9v"
	sig, err := solana.SignatureFromBase58(signature)
	if err != nil {
		t.Fatal(err)
	}
	srv := testutil.NewMockRPCServer(t, func(method string, params json.RawMessage) (any, error) {
		if method != "getTransaction" {
			t.Errorf("method = %q", method)
		}
		var args []json.RawMessage
		if err := json.Unmarshal(params, &args); err != nil {
			t.Fatal(err)
		}
		var cfg rpc.GetTransactionCfg
		if err := json.Unmarshal(args[1], &cfg); err != nil {
			t.Fatal(err)
		}
		if cfg.Encoding != solana.EncodingBase64 || cfg.Commitment != solana.CommitmentConfirmed ||
			cfg.MaxSupportedTransactionVersion == nil || *cfg.MaxSupportedTransactionVersion != 1 {
			t.Errorf("config = %+v", cfg)
		}
		return map[string]any{
			"slot": 449468371, "version": 1,
			"transaction": [2]string{base64.StdEncoding.EncodeToString(wire), "base64"},
		}, nil
	})
	maxVer := uint64(1)
	got, err := rpc.NewClientWith(srv.URL).GetTransaction(context.Background(), sig, rpc.GetTransactionCfg{
		Encoding:                       solana.EncodingBase64,
		Commitment:                     solana.CommitmentConfirmed,
		MaxSupportedTransactionVersion: &maxVer,
	})
	if err != nil {
		t.Fatalf("GetTransaction: %v", err)
	}
	if got == nil || got.Version != solana.MessageVersion1 || got.Transaction == nil ||
		len(got.Transaction.Signatures) != 1 || got.Transaction.Signatures[0] != sig {
		t.Fatalf("unexpected v1 result: %+v", got)
	}
}
