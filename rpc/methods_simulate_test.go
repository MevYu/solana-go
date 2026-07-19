package rpc_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	solana "github.com/MevYu/solana-go"
	"github.com/MevYu/solana-go/internal/testutil"
	"github.com/MevYu/solana-go/rpc"
)

func simulationTestTransaction() *solana.Transaction {
	return solana.NewTransaction(solana.Message{Version: solana.MessageVersionLegacy})
}

func TestSimulateTransaction_WireEncodingAndConfig(t *testing.T) {
	tx := simulationTestTransaction()
	wantBase64, err := tx.ToBase64()
	if err != nil {
		t.Fatal(err)
	}
	wantBase58, err := tx.ToBase58()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name         string
		encoding     solana.Encoding
		wantEncoding string
		wantPayload  string
	}{
		{name: "default base64", wantEncoding: "base64", wantPayload: wantBase64},
		{name: "explicit base58", encoding: solana.EncodingBase58, wantEncoding: "base58", wantPayload: wantBase58},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var gotParams json.RawMessage
			srv := testutil.NewMockRPCServer(t, func(method string, params json.RawMessage) (any, error) {
				if method != "simulateTransaction" {
					t.Errorf("method = %q, want simulateTransaction", method)
				}
				gotParams = append(gotParams[:0], params...)
				return contextEnvelope(77, map[string]any{"err": nil}), nil
			})

			includeInner := true
			accounts := &rpc.SimulateAccountsCfg{Addresses: []solana.PublicKey{solana.SystemProgramID}}
			cfg := rpc.SimulateTxCfg{
				Encoding:          test.encoding,
				InnerInstructions: &includeInner,
				Accounts:          accounts,
			}
			result, err := rpc.NewClientWith(srv.URL).SimulateTransaction(context.Background(), tx, cfg)
			if err != nil {
				t.Fatalf("SimulateTransaction: %v", err)
			}
			if result.Slot != 77 {
				t.Fatalf("Slot = %d, want 77", result.Slot)
			}
			if err := rpc.DecodeTransactionError(result.Err); err != nil {
				t.Fatalf("result error = %v, want success", err)
			}
			if accounts.Encoding != "" {
				t.Fatalf("SimulateTransaction mutated caller's account encoding to %q", accounts.Encoding)
			}

			var params []json.RawMessage
			if err := json.Unmarshal(gotParams, &params); err != nil {
				t.Fatalf("decode params: %v", err)
			}
			if len(params) != 2 {
				t.Fatalf("len(params) = %d, want 2", len(params))
			}
			var payload string
			if err := json.Unmarshal(params[0], &payload); err != nil {
				t.Fatalf("decode payload: %v", err)
			}
			if payload != test.wantPayload {
				t.Fatalf("payload = %q, want %q", payload, test.wantPayload)
			}
			var options struct {
				Encoding          string `json:"encoding"`
				InnerInstructions bool   `json:"innerInstructions"`
				Accounts          struct {
					Encoding  string   `json:"encoding"`
					Addresses []string `json:"addresses"`
				} `json:"accounts"`
			}
			if err := json.Unmarshal(params[1], &options); err != nil {
				t.Fatalf("decode options: %v", err)
			}
			if options.Encoding != test.wantEncoding {
				t.Fatalf("encoding = %q, want %q", options.Encoding, test.wantEncoding)
			}
			if !options.InnerInstructions {
				t.Fatal("innerInstructions = false, want true")
			}
			if options.Accounts.Encoding != "base64" {
				t.Fatalf("accounts.encoding = %q, want base64", options.Accounts.Encoding)
			}
			if len(options.Accounts.Addresses) != 1 || options.Accounts.Addresses[0] != solana.SystemProgramID.String() {
				t.Fatalf("accounts.addresses = %v", options.Accounts.Addresses)
			}
		})
	}
}

func TestSimulateTransaction_ValidatesInput(t *testing.T) {
	client := rpc.NewClientWith("http://invalid.invalid")
	tx := simulationTestTransaction()
	yes := true

	tests := []struct {
		name string
		tx   *solana.Transaction
		cfg  rpc.SimulateTxCfg
		want string
	}{
		{name: "nil transaction", tx: nil, want: "nil transaction"},
		{name: "unsupported transaction encoding", tx: tx, cfg: rpc.SimulateTxCfg{Encoding: solana.EncodingJSON}, want: "unsupported transaction encoding"},
		{
			name: "unsupported base58 account encoding",
			tx:   tx,
			cfg:  rpc.SimulateTxCfg{Accounts: &rpc.SimulateAccountsCfg{Encoding: solana.EncodingBase58}},
			want: "unsupported account encoding",
		},
		{
			name: "unsupported parsed account encoding",
			tx:   tx,
			cfg:  rpc.SimulateTxCfg{Accounts: &rpc.SimulateAccountsCfg{Encoding: solana.EncodingJSONParsed}},
			want: "unsupported account encoding",
		},
		{
			name: "incompatible flags",
			tx:   tx,
			cfg:  rpc.SimulateTxCfg{SigVerify: &yes, ReplaceRecentBlockhash: &yes},
			want: "cannot both be true",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := client.SimulateTransaction(context.Background(), test.tx, test.cfg)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("SimulateTransaction() = %v, want error containing %q", err, test.want)
			}
		})
	}
}

func TestSimulateResult_UnmarshalFullResponse(t *testing.T) {
	replacementHash := "HUhwFvKfmVhZ5AbkqK7N2Kt3i6hPz3jQeByN6wA9RvXb"
	body := []byte(`{
		"accounts": null,
		"err": null,
		"fee": 5000,
		"preBalances": [10000, 20],
		"postBalances": [4990, 20],
		"innerInstructions": [{
			"index": 0,
			"instructions": [
				{
					"program": "system",
					"programId": "11111111111111111111111111111111",
					"parsed": {"type": "transfer", "info": {"lamports": 10}},
					"stackHeight": 2
				},
				{
					"programId": "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA",
					"accounts": ["11111111111111111111111111111111"],
					"data": "2",
					"stackHeight": 2
				}
			]
		}],
		"preTokenBalances": [{
			"accountIndex": 1,
			"mint": "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA",
			"owner": "11111111111111111111111111111111",
			"programId": "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA",
			"uiTokenAmount": {"amount": "0", "decimals": 0, "uiAmount": null, "uiAmountString": "0"}
		}],
		"postTokenBalances": [{
			"accountIndex": 1,
			"mint": "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA",
			"owner": "11111111111111111111111111111111",
			"programId": "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA",
			"uiTokenAmount": {"amount": "10", "decimals": 0, "uiAmount": 10, "uiAmountString": "10"}
		}],
		"loadedAccountsDataSize": 413,
		"loadedAddresses": {
			"writable": ["11111111111111111111111111111111"],
			"readonly": ["TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"]
		},
		"logs": ["Program log: hello"],
		"replacementBlockhash": {
			"blockhash": "` + replacementHash + `",
			"lastValidBlockHeight": 999
		},
		"returnData": {
			"programId": "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA",
			"data": ["AQID", "base64"]
		},
		"unitsConsumed": 2200
	}`)

	var result rpc.SimulateResult
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if result.Err != nil {
		t.Fatalf("Err = %v, want nil", result.Err)
	}
	if err := rpc.DecodeTransactionError(result.Err); err != nil {
		t.Fatalf("Err = %v, want success", err)
	}
	if result.Fee == nil || *result.Fee != 5000 {
		t.Fatalf("Fee = %v, want 5000", result.Fee)
	}
	if len(result.PreBalances) != 2 || result.PreBalances[0] != 10000 || result.PostBalances[0] != 4990 {
		t.Fatalf("balances = %v -> %v", result.PreBalances, result.PostBalances)
	}
	if len(result.InnerInstructions) != 1 || len(result.InnerInstructions[0].Instructions) != 2 {
		t.Fatalf("InnerInstructions = %+v", result.InnerInstructions)
	}
	parsed := result.InnerInstructions[0].Instructions[0]
	if parsed.Program != "system" || parsed.ProgramID != solana.SystemProgramID || len(parsed.Parsed) == 0 {
		t.Fatalf("parsed instruction = %+v", parsed)
	}
	partiallyDecoded := result.InnerInstructions[0].Instructions[1]
	if partiallyDecoded.ProgramID != solana.TokenProgramID || len(partiallyDecoded.Accounts) != 1 || partiallyDecoded.Accounts[0] != solana.SystemProgramID {
		t.Fatalf("partially decoded instruction = %+v", partiallyDecoded)
	}
	if len(partiallyDecoded.Data) != 1 || partiallyDecoded.Data[0] != 1 {
		t.Fatalf("partially decoded data = %v, want [1]", partiallyDecoded.Data)
	}
	if len(result.PreTokenBalances) != 1 || result.PreTokenBalances[0].UiTokenAmount.UiAmount != nil {
		t.Fatalf("PreTokenBalances = %+v, want a null UI amount", result.PreTokenBalances)
	}
	if len(result.PostTokenBalances) != 1 || result.PostTokenBalances[0].UiTokenAmount.Amount != "10" {
		t.Fatalf("PostTokenBalances = %+v", result.PostTokenBalances)
	}
	postUIAmount := result.PostTokenBalances[0].UiTokenAmount.UiAmount
	if postUIAmount == nil || *postUIAmount != 10 {
		t.Fatalf("PostTokenBalances UI amount = %v, want 10", postUIAmount)
	}
	if result.LoadedAccountsDataSize == nil || *result.LoadedAccountsDataSize != 413 {
		t.Fatalf("LoadedAccountsDataSize = %v", result.LoadedAccountsDataSize)
	}
	if result.LoadedAddresses == nil || len(result.LoadedAddresses.Writable) != 1 || len(result.LoadedAddresses.ReadOnly) != 1 {
		t.Fatalf("LoadedAddresses = %+v", result.LoadedAddresses)
	}
	if result.ReplacementBlockhash == nil || result.ReplacementBlockhash.Blockhash.String() != replacementHash || result.ReplacementBlockhash.LastValidBlockHeight != 999 {
		t.Fatalf("ReplacementBlockhash = %+v", result.ReplacementBlockhash)
	}
	if result.ReturnData == nil || result.ReturnData.ProgramID != solana.TokenProgramID || string(result.ReturnData.Data.Bytes) != "\x01\x02\x03" {
		t.Fatalf("ReturnData = %+v", result.ReturnData)
	}
	if result.UnitsConsumed == nil || *result.UnitsConsumed != 2200 {
		t.Fatalf("UnitsConsumed = %v", result.UnitsConsumed)
	}
}

func TestSimulateResult_NullFee(t *testing.T) {
	var result rpc.SimulateResult
	if err := json.Unmarshal([]byte(`{"err":null,"fee":null}`), &result); err != nil {
		t.Fatal(err)
	}
	if result.Fee != nil {
		t.Fatalf("Fee = %v, want nil", result.Fee)
	}
	if result.Err != nil {
		t.Fatalf("Err = %v, want nil", result.Err)
	}
}

func TestSimulateResult_DecodesExecutionError(t *testing.T) {
	var result rpc.SimulateResult
	if err := json.Unmarshal([]byte(`{"err":{"InstructionError":[2,{"Custom":6001}]}}`), &result); err != nil {
		t.Fatal(err)
	}
	if result.Err == nil {
		t.Fatal("Err = nil, want decoded transaction error value")
	}

	decoded := rpc.DecodeTransactionError(result.Err)
	var instructionErr *rpc.InstructionError
	if !errors.As(decoded, &instructionErr) {
		t.Fatalf("expected *rpc.InstructionError, got %T (%v)", decoded, decoded)
	}
	if instructionErr.Index != 2 || instructionErr.Kind != "Custom" || instructionErr.CustomErrorCode != 6001 {
		t.Fatalf("decoded error = %+v", instructionErr)
	}
}
