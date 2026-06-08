package rpc

import (
	"testing"

	solana "github.com/MevYu/solana-go"
)

func TestPriorityFeeStatsFromFees_TenSamples(t *testing.T) {
	fees := []solana.PrioritizationFee{
		{Slot: 100, PrioritizationFee: 1000},
		{Slot: 101, PrioritizationFee: 2000},
		{Slot: 102, PrioritizationFee: 3000},
		{Slot: 103, PrioritizationFee: 4000},
		{Slot: 104, PrioritizationFee: 5000},
		{Slot: 105, PrioritizationFee: 6000},
		{Slot: 106, PrioritizationFee: 7000},
		{Slot: 107, PrioritizationFee: 8000},
		{Slot: 108, PrioritizationFee: 9000},
		{Slot: 109, PrioritizationFee: 10000},
	}
	stats := PriorityFeeStatsFromFees(fees)
	if stats.Samples != 10 {
		t.Errorf("Samples = %d, want 10", stats.Samples)
	}
	if stats.P50 != 5000 {
		t.Errorf("P50 = %d, want 5000", stats.P50)
	}
	if stats.P75 != 7000 {
		t.Errorf("P75 = %d, want 7000", stats.P75)
	}
	if stats.P95 != 9000 {
		t.Errorf("P95 = %d, want 9000", stats.P95)
	}
	if stats.Max != 10000 {
		t.Errorf("Max = %d, want 10000", stats.Max)
	}
}

func TestPriorityFeeStatsFromFees_Empty(t *testing.T) {
	stats := PriorityFeeStatsFromFees(nil)
	if stats.Samples != 0 {
		t.Errorf("Samples = %d, want 0", stats.Samples)
	}
	if stats.P50 != 0 || stats.P75 != 0 || stats.P95 != 0 || stats.Max != 0 {
		t.Errorf("all stats should be zero on empty input, got %+v", stats)
	}
}

func TestPriorityFeeStatsFromFees_SingleSample(t *testing.T) {
	fees := []solana.PrioritizationFee{{Slot: 1, PrioritizationFee: 12345}}
	stats := PriorityFeeStatsFromFees(fees)
	if stats.Samples != 1 {
		t.Errorf("Samples = %d", stats.Samples)
	}
	if stats.P50 != 12345 || stats.P75 != 12345 || stats.P95 != 12345 || stats.Max != 12345 {
		t.Errorf("all stats should equal the single sample, got %+v", stats)
	}
}

func TestPriorityFeeStatsFromFees_UnsortedInput(t *testing.T) {
	fees := []solana.PrioritizationFee{
		{Slot: 1, PrioritizationFee: 3000},
		{Slot: 2, PrioritizationFee: 1000},
		{Slot: 3, PrioritizationFee: 2000},
	}
	stats := PriorityFeeStatsFromFees(fees)
	if stats.Max != 3000 {
		t.Errorf("Max = %d, want 3000", stats.Max)
	}
	if stats.P50 == 0 {
		t.Error("P50 should not be zero")
	}
}
