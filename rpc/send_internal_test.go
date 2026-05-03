package rpc

import (
	"testing"

)

func TestStatusReachedCommitment(t *testing.T) {
	cases := []struct {
		status   string
		required CommitmentLevel
		want     bool
	}{
		{"processed", CommitmentProcessed, true},
		{"processed", CommitmentConfirmed, false},
		{"processed", CommitmentFinalized, false},
		{"confirmed", CommitmentProcessed, true},
		{"confirmed", CommitmentConfirmed, true},
		{"confirmed", CommitmentFinalized, false},
		{"finalized", CommitmentFinalized, true},
		{"finalized", CommitmentConfirmed, true},
		{"finalized", CommitmentProcessed, true},
		{"unknown", CommitmentConfirmed, false},
	}
	for _, tc := range cases {
		got := statusReachedCommitment(tc.status, tc.required)
		if got != tc.want {
			t.Errorf("status=%q required=%q: got %v, want %v", tc.status, tc.required, got, tc.want)
		}
	}
}
