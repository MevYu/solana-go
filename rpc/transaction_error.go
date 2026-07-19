package rpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
)

// TransactionError is a top-level transaction error that is not tied to a
// specific instruction. Kind is the symbolic name reported by the Solana
// runtime. Details is populated for data-bearing variants such as
// DuplicateInstruction.
type TransactionError struct {
	Kind    string
	Details any
}

func (e *TransactionError) Error() string {
	if e.Details != nil {
		return fmt.Sprintf("solana: transaction error: %s (%v)", e.Kind, e.Details)
	}
	return "solana: transaction error: " + e.Kind
}

// InstructionError is a typed transaction error attributed to a single
// instruction. Index is the 0-based position of the failing instruction;
// Kind matches the Solana runtime's InstructionError enum. CustomErrorCode is
// populated for Custom errors, while Details preserves payloads carried by
// other data-bearing variants from older RPC nodes.
type InstructionError struct {
	Index           int
	Kind            string
	CustomErrorCode uint32
	Details         any
}

func (e *InstructionError) Error() string {
	if e.Kind == "Custom" {
		return fmt.Sprintf("solana: instruction %d: custom program error 0x%x", e.Index, e.CustomErrorCode)
	}
	if e.Details != nil {
		return fmt.Sprintf("solana: instruction %d: %s (%v)", e.Index, e.Kind, e.Details)
	}
	return fmt.Sprintf("solana: instruction %d: %s", e.Index, e.Kind)
}

// DecodeTransactionError parses a transaction err field into either a
// *TransactionError, an *InstructionError, or nil. The same wire shape is
// used by simulateTransaction, getTransaction metadata, signature statuses,
// and transaction-related WebSocket notifications.
//
// raw may be nil, json.RawMessage, []byte, a plain error-kind string, or the
// map[string]any shape produced by unmarshalling JSON into an interface.
func DecodeTransactionError(raw any) error {
	if raw == nil {
		return nil
	}

	switch v := raw.(type) {
	case json.RawMessage:
		return decodeTransactionErrorJSON(v)
	case []byte:
		return decodeTransactionErrorJSON(v)
	case string:
		return &TransactionError{Kind: v}
	case map[string]any:
		if len(v) != 1 {
			return fmt.Errorf("solana rpc: transaction error object must contain exactly one variant, got %d", len(v))
		}
		for kind, details := range v {
			if kind == "InstructionError" {
				return decodeInstructionError(details)
			}
			return &TransactionError{Kind: kind, Details: details}
		}
	}

	return fmt.Errorf("solana rpc: unrecognized transaction error shape: %v", raw)
}

func decodeTransactionErrorJSON(raw []byte) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil
	}
	if !json.Valid(trimmed) {
		var discard any
		err := json.Unmarshal(trimmed, &discard)
		return fmt.Errorf("solana rpc: decode transaction error JSON: %w", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return fmt.Errorf("solana rpc: decode transaction error JSON: %w", err)
	}
	return DecodeTransactionError(decoded)
}

func decodeInstructionError(raw any) error {
	parts, ok := raw.([]any)
	if !ok || len(parts) != 2 {
		return fmt.Errorf("solana rpc: InstructionError must be [index, error], got %v", raw)
	}

	index, err := decodeUnsigned(parts[0], uint64(^uint8(0)))
	if err != nil {
		return fmt.Errorf("solana rpc: decode InstructionError index: %w", err)
	}

	switch value := parts[1].(type) {
	case string:
		return &InstructionError{Index: int(index), Kind: value}
	case map[string]any:
		if len(value) != 1 {
			return fmt.Errorf("solana rpc: instruction error object must contain exactly one variant, got %d", len(value))
		}
		for kind, details := range value {
			if kind == "Custom" {
				code, err := decodeUnsigned(details, uint64(^uint32(0)))
				if err != nil {
					return fmt.Errorf("solana rpc: decode custom program error code: %w", err)
				}
				return &InstructionError{
					Index:           int(index),
					Kind:            "Custom",
					CustomErrorCode: uint32(code),
				}
			}
			return &InstructionError{Index: int(index), Kind: kind, Details: details}
		}
	}

	return fmt.Errorf("solana rpc: unrecognized instruction error at index %d: %v", index, parts[1])
}

func decodeUnsigned(raw any, max uint64) (uint64, error) {
	var value uint64
	switch number := raw.(type) {
	case json.Number:
		parsed, err := strconv.ParseUint(number.String(), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("expected an unsigned integer, got %q", number)
		}
		value = parsed
	case float64:
		if math.IsNaN(number) || math.IsInf(number, 0) || number < 0 || math.Trunc(number) != number || number > float64(max) {
			return 0, fmt.Errorf("expected an integer between 0 and %d, got %v", max, number)
		}
		value = uint64(number)
	case float32:
		converted := float64(number)
		if math.IsNaN(converted) || math.IsInf(converted, 0) || converted < 0 || math.Trunc(converted) != converted || converted > float64(max) {
			return 0, fmt.Errorf("expected an integer between 0 and %d, got %v", max, number)
		}
		value = uint64(number)
	case int:
		if number < 0 {
			return 0, fmt.Errorf("expected an unsigned integer, got %d", number)
		}
		value = uint64(number)
	case int8:
		if number < 0 {
			return 0, fmt.Errorf("expected an unsigned integer, got %d", number)
		}
		value = uint64(number)
	case int16:
		if number < 0 {
			return 0, fmt.Errorf("expected an unsigned integer, got %d", number)
		}
		value = uint64(number)
	case int32:
		if number < 0 {
			return 0, fmt.Errorf("expected an unsigned integer, got %d", number)
		}
		value = uint64(number)
	case int64:
		if number < 0 {
			return 0, fmt.Errorf("expected an unsigned integer, got %d", number)
		}
		value = uint64(number)
	case uint:
		value = uint64(number)
	case uint8:
		value = uint64(number)
	case uint16:
		value = uint64(number)
	case uint32:
		value = uint64(number)
	case uint64:
		value = number
	default:
		return 0, fmt.Errorf("expected an unsigned integer, got %T", raw)
	}

	if value > max {
		return 0, fmt.Errorf("integer %d exceeds maximum %d", value, max)
	}
	return value, nil
}
