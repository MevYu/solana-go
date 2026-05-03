package rpc

import (
	"testing"

	solana "github.com/MevYu/solana-go"
)

func TestStatusReachedCommitment(t *testing.T) {
	cases := []struct {
		status   string
		required solana.CommitmentLevel
		want     bool
	}{
		{"processed", solana.CommitmentProcessed, true},
		{"processed", solana.CommitmentConfirmed, false},
		{"processed", solana.CommitmentFinalized, false},
		{"confirmed", solana.CommitmentProcessed, true},
		{"confirmed", solana.CommitmentConfirmed, true},
		{"confirmed", solana.CommitmentFinalized, false},
		{"finalized", solana.CommitmentFinalized, true},
		{"finalized", solana.CommitmentConfirmed, true},
		{"finalized", solana.CommitmentProcessed, true},
		{"unknown", solana.CommitmentConfirmed, false},
	}
	for _, tc := range cases {
		got := statusReachedCommitment(tc.status, tc.required)
		if got != tc.want {
			t.Errorf("status=%q required=%q: got %v, want %v", tc.status, tc.required, got, tc.want)
		}
	}
}
