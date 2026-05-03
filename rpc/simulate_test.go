package rpc

import (
	"errors"
	"testing"
)

func TestDecodeTransactionError_Nil(t *testing.T) {
	if err := DecodeTransactionError(nil); err != nil {
		t.Errorf("nil should decode to nil, got %v", err)
	}
}

func TestDecodeTransactionError_StringKind(t *testing.T) {
	err := DecodeTransactionError("BlockhashNotFound")
	if err == nil {
		t.Fatal("expected error")
	}
	var te *TransactionError
	if !errors.As(err, &te) {
		t.Fatalf("expected *TransactionError, got %T", err)
	}
	if te.Kind != "BlockhashNotFound" {
		t.Errorf("Kind = %q", te.Kind)
	}
}

func TestDecodeTransactionError_InstructionErrorCustom(t *testing.T) {
	raw := map[string]any{
		"InstructionError": []any{
			float64(2),
			map[string]any{"Custom": float64(42)},
		},
	}
	err := DecodeTransactionError(raw)
	var ie *InstructionError
	if !errors.As(err, &ie) {
		t.Fatalf("expected *InstructionError, got %T", err)
	}
	if ie.Index != 2 {
		t.Errorf("Index = %d", ie.Index)
	}
	if ie.Kind != "Custom" {
		t.Errorf("Kind = %q", ie.Kind)
	}
	if ie.CustomErrorCode != 42 {
		t.Errorf("CustomErrorCode = %d", ie.CustomErrorCode)
	}
}

func TestDecodeTransactionError_InstructionErrorNamed(t *testing.T) {
	raw := map[string]any{
		"InstructionError": []any{float64(0), "InvalidAccountData"},
	}
	err := DecodeTransactionError(raw)
	var ie *InstructionError
	if !errors.As(err, &ie) {
		t.Fatalf("expected *InstructionError, got %T", err)
	}
	if ie.Index != 0 || ie.Kind != "InvalidAccountData" {
		t.Errorf("got %+v", ie)
	}
}

func TestDecodeTransactionError_InstructionErrorSubKind(t *testing.T) {
	raw := map[string]any{
		"InstructionError": []any{
			float64(1),
			map[string]any{"BorshIoError": "deserialisation failed"},
		},
	}
	err := DecodeTransactionError(raw)
	var ie *InstructionError
	if !errors.As(err, &ie) {
		t.Fatalf("expected *InstructionError, got %T", err)
	}
	if ie.Kind != "BorshIoError" {
		t.Errorf("Kind = %q", ie.Kind)
	}
}

func TestInstructionError_ErrorMessage(t *testing.T) {
	ie := &InstructionError{Index: 3, Kind: "Custom", CustomErrorCode: 0x1234}
	if ie.Error() == "" {
		t.Fatal("empty error message")
	}
}

func TestTransactionError_ErrorMessage(t *testing.T) {
	te := &TransactionError{Kind: "BlockhashNotFound"}
	if te.Error() == "" {
		t.Fatal("empty error message")
	}
}
