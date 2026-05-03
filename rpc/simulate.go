package rpc

import "fmt"

// TransactionError is a top-level transaction error that is not tied to
// a specific instruction. It carries the symbolic name the Solana
// runtime reports, for example "BlockhashNotFound", "AccountNotFound",
// or "AlreadyProcessed".
type TransactionError struct {
	Kind string
}

func (e *TransactionError) Error() string {
	return "solana: transaction error: " + e.Kind
}

// InstructionError is a typed transaction error attributed to a single
// instruction in the transaction. Index is the 0-based position of the
// failing instruction; Kind is the symbolic name of the failure matching
// the Solana runtime's InstructionError enum; CustomErrorCode is set when
// Kind is "Custom" and carries the program-defined u32 error code.
type InstructionError struct {
	Index           int
	Kind            string
	CustomErrorCode uint32
}

func (e *InstructionError) Error() string {
	if e.Kind == "Custom" {
		return fmt.Sprintf("solana: instruction %d: custom program error 0x%x", e.Index, e.CustomErrorCode)
	}
	return fmt.Sprintf("solana: instruction %d: %s", e.Index, e.Kind)
}

// DecodeTransactionError parses the server's raw err field into either a
// *TransactionError, an *InstructionError, or nil.
//
// The raw value comes from simulateTransaction or getTransaction's
// meta.err and is the untyped JSON shape Solana returns:
//
//   - nil (success)
//   - a plain string like "BlockhashNotFound"
//   - {"InstructionError": [<idx>, "<kind>"]}
//   - {"InstructionError": [<idx>, {"Custom": <u32>}]}
//   - {"InstructionError": [<idx>, {"<Kind>": <sub>}]}
//
// Unrecognised shapes are returned as a plain fmt.Errorf rather than
// silently discarded.
func DecodeTransactionError(raw any) error {
	if raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case string:
		return &TransactionError{Kind: v}
	case map[string]any:
		if ieRaw, ok := v["InstructionError"]; ok {
			return decodeInstructionError(ieRaw)
		}
		for k := range v {
			return &TransactionError{Kind: k}
		}
	}
	return fmt.Errorf("solana helpers: unrecognized transaction error shape: %v", raw)
}

func decodeInstructionError(ieRaw any) error {
	arr, ok := ieRaw.([]any)
	if !ok || len(arr) < 2 {
		return fmt.Errorf("solana helpers: unrecognized InstructionError shape: %v", ieRaw)
	}
	idx := 0
	switch n := arr[0].(type) {
	case float64:
		idx = int(n)
	case int:
		idx = n
	case int32:
		idx = int(n)
	case int64:
		idx = int(n)
	case uint32:
		idx = int(n)
	case uint64:
		idx = int(n)
	}
	errVal := arr[1]
	switch ev := errVal.(type) {
	case string:
		return &InstructionError{Index: idx, Kind: ev}
	case map[string]any:
		if custom, ok := ev["Custom"]; ok {
			var code uint32
			switch c := custom.(type) {
			case float64:
				code = uint32(c)
			case int:
				code = uint32(c)
			case int32:
				code = uint32(c)
			case int64:
				code = uint32(c)
			case uint32:
				code = c
			case uint64:
				code = uint32(c)
			}
			return &InstructionError{Index: idx, Kind: "Custom", CustomErrorCode: code}
		}
		for k := range ev {
			return &InstructionError{Index: idx, Kind: k}
		}
	}
	return fmt.Errorf("solana helpers: unknown instruction error %d: %v", idx, errVal)
}
