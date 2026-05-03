package jsonrpc

import (
	"errors"
	"fmt"
	"testing"
)

func TestIsBlockhashExpired_SentinelMatch(t *testing.T) {
	wrapped := fmt.Errorf("wrapper: %w", ErrBlockhashExpired)
	if !IsBlockhashExpired(wrapped) {
		t.Error("sentinel should match")
	}
}

func TestIsBlockhashExpired_RPCError(t *testing.T) {
	err := &ErrRPC{
		Method: "sendTransaction",
		Code:   -32002,
		Msg:    "Transaction simulation failed: BlockhashNotFound",
	}
	if !IsBlockhashExpired(err) {
		t.Error("RPC error with BlockhashNotFound should classify as blockhash expired")
	}
}

func TestIsBlockhashExpired_WrappedRPCError(t *testing.T) {
	inner := &ErrRPC{Method: "sendTransaction", Msg: "blockhash not found"}
	wrapped := fmt.Errorf("helpers: %w", inner)
	if !IsBlockhashExpired(wrapped) {
		t.Error("wrapped RPC error should still classify")
	}
}

func TestIsBlockhashExpired_False(t *testing.T) {
	if IsBlockhashExpired(nil) {
		t.Error("nil should not classify as blockhash expired")
	}
	if IsBlockhashExpired(errors.New("unrelated failure")) {
		t.Error("unrelated error should not classify")
	}
}

func TestIsInsufficientFunds(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{nil, false},
		{ErrInsufficientFunds, true},
		{fmt.Errorf("wrapper: %w", ErrInsufficientFunds), true},
		{errors.New("InsufficientFundsForFee"), true},
		{errors.New("account lacks rent: InsufficientFundsForRent"), true},
		{errors.New("unrelated"), false},
	}
	for i, tc := range cases {
		if got := IsInsufficientFunds(tc.err); got != tc.want {
			t.Errorf("case %d: got %v, want %v (err=%v)", i, got, tc.want, tc.err)
		}
	}
}

func TestIsRateLimited(t *testing.T) {
	if !IsRateLimited(errors.New("http 429: rate limit exceeded")) {
		t.Error("rate limit message should classify")
	}
	if !IsRateLimited(ErrRateLimited) {
		t.Error("sentinel should match")
	}
	if IsRateLimited(nil) {
		t.Error("nil should not classify")
	}
}

func TestIsNodeBehind_ByRPCCode(t *testing.T) {
	err := &ErrRPC{
		Method: "getBlock",
		Code:   RPCErrCodeNodeUnhealthy,
		Msg:    "Node is behind by 123 slots",
	}
	if !IsNodeBehind(err) {
		t.Error("code -32005 should classify as node behind")
	}
}

func TestIsNodeBehind_ByMessage(t *testing.T) {
	err := errors.New("NodeUnhealthy: 456 slots behind")
	if !IsNodeBehind(err) {
		t.Error("message match should classify")
	}
}

func TestIsSlotSkipped_ByRPCCode(t *testing.T) {
	err := &ErrRPC{Code: RPCErrCodeSlotSkipped, Msg: "Slot 123 skipped"}
	if !IsSlotSkipped(err) {
		t.Error("code -32007 should classify as slot skipped")
	}
}

func TestIsBlockCleanedUp_ByRPCCode(t *testing.T) {
	err := &ErrRPC{Code: RPCErrCodeBlockCleanedUp, Msg: "Block cleaned up"}
	if !IsBlockCleanedUp(err) {
		t.Error("code -32001 should classify as block cleaned up")
	}
}

func TestIsAccountNotFound(t *testing.T) {
	if !IsAccountNotFound(errors.New("AccountNotFound")) {
		t.Error("should classify")
	}
	if IsAccountNotFound(errors.New("some other failure")) {
		t.Error("should not classify")
	}
}

func TestIsSignatureNotFound(t *testing.T) {
	if !IsSignatureNotFound(ErrSignatureNotFound) {
		t.Error("sentinel should match")
	}
	if !IsSignatureNotFound(errors.New("SignatureNotFound")) {
		t.Error("message should match")
	}
}

func TestIsTransactionExpired(t *testing.T) {
	if !IsTransactionExpired(ErrTransactionExpired) {
		t.Error("sentinel should match")
	}
	if !IsTransactionExpired(errors.New("TransactionExpired: blockhash lifetime ended")) {
		t.Error("message should match")
	}
	if IsTransactionExpired(nil) {
		t.Error("nil should not classify")
	}
}

func TestClassifiers_AllHandleNil(t *testing.T) {
	classifiers := []struct {
		name string
		fn   func(error) bool
	}{
		{"IsBlockhashExpired", IsBlockhashExpired},
		{"IsInsufficientFunds", IsInsufficientFunds},
		{"IsRateLimited", IsRateLimited},
		{"IsNodeBehind", IsNodeBehind},
		{"IsTransactionExpired", IsTransactionExpired},
		{"IsAccountNotFound", IsAccountNotFound},
		{"IsSignatureNotFound", IsSignatureNotFound},
		{"IsSlotSkipped", IsSlotSkipped},
		{"IsBlockCleanedUp", IsBlockCleanedUp},
	}
	for _, c := range classifiers {
		if c.fn(nil) {
			t.Errorf("%s(nil) returned true", c.name)
		}
	}
}
