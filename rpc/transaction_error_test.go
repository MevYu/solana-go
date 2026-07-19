package rpc

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestDecodeTransactionError_SuccessValues(t *testing.T) {
	tests := []struct {
		name string
		raw  any
	}{
		{name: "nil", raw: nil},
		{name: "nil RawMessage", raw: json.RawMessage(nil)},
		{name: "nil bytes", raw: []byte(nil)},
		{name: "JSON null", raw: json.RawMessage(" \n null \t")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := DecodeTransactionError(test.raw); err != nil {
				t.Fatalf("DecodeTransactionError() = %v, want nil", err)
			}
		})
	}
}

func TestDecodeTransactionError_TransactionVariants(t *testing.T) {
	t.Run("plain string", func(t *testing.T) {
		err := DecodeTransactionError("BlockhashNotFound")
		var transactionErr *TransactionError
		if !errors.As(err, &transactionErr) {
			t.Fatalf("expected *TransactionError, got %T (%v)", err, err)
		}
		if transactionErr.Kind != "BlockhashNotFound" || transactionErr.Details != nil {
			t.Fatalf("decoded error = %+v", transactionErr)
		}
	})

	t.Run("raw JSON data-bearing variant", func(t *testing.T) {
		err := DecodeTransactionError(json.RawMessage(`{"DuplicateInstruction":42}`))
		var transactionErr *TransactionError
		if !errors.As(err, &transactionErr) {
			t.Fatalf("expected *TransactionError, got %T (%v)", err, err)
		}
		if transactionErr.Kind != "DuplicateInstruction" {
			t.Fatalf("Kind = %q", transactionErr.Kind)
		}
		number, ok := transactionErr.Details.(json.Number)
		if !ok || number.String() != "42" {
			t.Fatalf("Details = %T(%v), want json.Number(42)", transactionErr.Details, transactionErr.Details)
		}
	})
}

func TestDecodeTransactionError_InstructionVariants(t *testing.T) {
	t.Run("named", func(t *testing.T) {
		err := DecodeTransactionError(json.RawMessage(`{"InstructionError":[0,"InvalidAccountData"]}`))
		var instructionErr *InstructionError
		if !errors.As(err, &instructionErr) {
			t.Fatalf("expected *InstructionError, got %T (%v)", err, err)
		}
		if instructionErr.Index != 0 || instructionErr.Kind != "InvalidAccountData" {
			t.Fatalf("decoded error = %+v", instructionErr)
		}
	})

	t.Run("custom", func(t *testing.T) {
		err := DecodeTransactionError(json.RawMessage(`{"InstructionError":[2,{"Custom":42}]}`))
		var instructionErr *InstructionError
		if !errors.As(err, &instructionErr) {
			t.Fatalf("expected *InstructionError, got %T (%v)", err, err)
		}
		if instructionErr.Index != 2 || instructionErr.Kind != "Custom" || instructionErr.CustomErrorCode != 42 {
			t.Fatalf("decoded error = %+v", instructionErr)
		}
	})

	t.Run("legacy data-bearing variant", func(t *testing.T) {
		err := DecodeTransactionError(json.RawMessage(`{"InstructionError":[1,{"BorshIoError":"decode failed"}]}`))
		var instructionErr *InstructionError
		if !errors.As(err, &instructionErr) {
			t.Fatalf("expected *InstructionError, got %T (%v)", err, err)
		}
		if instructionErr.Kind != "BorshIoError" || instructionErr.Details != "decode failed" {
			t.Fatalf("decoded error = %+v", instructionErr)
		}
	})
}

func TestDecodeTransactionError_RejectsMalformedValues(t *testing.T) {
	tests := []struct {
		name string
		raw  any
		want string
	}{
		{name: "invalid JSON", raw: json.RawMessage(`{`), want: "decode transaction error JSON"},
		{name: "empty object", raw: map[string]any{}, want: "exactly one variant"},
		{name: "multiple variants", raw: map[string]any{"A": nil, "B": nil}, want: "exactly one variant"},
		{name: "short instruction tuple", raw: json.RawMessage(`{"InstructionError":[0]}`), want: "must be [index, error]"},
		{name: "string index", raw: json.RawMessage(`{"InstructionError":["0","InvalidArgument"]}`), want: "decode InstructionError index"},
		{name: "negative index", raw: json.RawMessage(`{"InstructionError":[-1,"InvalidArgument"]}`), want: "decode InstructionError index"},
		{name: "fractional index", raw: json.RawMessage(`{"InstructionError":[1.5,"InvalidArgument"]}`), want: "decode InstructionError index"},
		{name: "index overflow", raw: json.RawMessage(`{"InstructionError":[256,"InvalidArgument"]}`), want: "exceeds maximum 255"},
		{name: "custom code string", raw: json.RawMessage(`{"InstructionError":[0,{"Custom":"42"}]}`), want: "decode custom program error code"},
		{name: "custom code overflow", raw: json.RawMessage(`{"InstructionError":[0,{"Custom":4294967296}]}`), want: "exceeds maximum 4294967295"},
		{name: "multiple instruction variants", raw: json.RawMessage(`{"InstructionError":[0,{"A":null,"B":null}]}`), want: "exactly one variant"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := DecodeTransactionError(test.raw)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("DecodeTransactionError() = %v, want error containing %q", err, test.want)
			}
		})
	}
}

func TestTransactionErrors_ErrorMessages(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "transaction",
			err:  &TransactionError{Kind: "BlockhashNotFound"},
			want: "solana: transaction error: BlockhashNotFound",
		},
		{
			name: "data-bearing transaction",
			err:  &TransactionError{Kind: "DuplicateInstruction", Details: 2},
			want: "solana: transaction error: DuplicateInstruction (2)",
		},
		{
			name: "named instruction",
			err:  &InstructionError{Index: 1, Kind: "InvalidAccountData"},
			want: "solana: instruction 1: InvalidAccountData",
		},
		{
			name: "custom instruction",
			err:  &InstructionError{Index: 3, Kind: "Custom", CustomErrorCode: 0x1234},
			want: "solana: instruction 3: custom program error 0x1234",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.err.Error(); got != test.want {
				t.Fatalf("Error() = %q, want %q", got, test.want)
			}
		})
	}
}
