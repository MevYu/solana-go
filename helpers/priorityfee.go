package helpers

import (
	"sort"

	solana "github.com/MevYu/solana-go"
)

// PriorityFeeStats summarises a window of recent prioritization fees as
// ordered percentiles. Values are in micro-lamports per compute unit,
// matching the units getRecentPrioritizationFees returns.
type PriorityFeeStats struct {
	// P50 is the median observed prioritization fee.
	P50 uint64
	// P75 is the 75th percentile observed prioritization fee.
	P75 uint64
	// P95 is the 95th percentile observed prioritization fee.
	P95 uint64
	// Max is the highest observed prioritization fee.
	Max uint64
	// Samples is the number of slots that contributed to the statistics.
	// A low Samples count means the percentiles are based on sparse data
	// and should be treated as advisory.
	Samples int
}

// PriorityFeeStatsFromFees computes ordered-percentile statistics from a
// pre-fetched slice of fees (typically from Client.GetRecentPrioritizationFees).
//
// The input slice is not mutated.
func PriorityFeeStatsFromFees(fees []solana.PrioritizationFee) *PriorityFeeStats {
	if len(fees) == 0 {
		return &PriorityFeeStats{}
	}
	values := make([]uint64, len(fees))
	for i, f := range fees {
		values[i] = f.PrioritizationFee
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	return &PriorityFeeStats{
		P50:     percentile(values, 50),
		P75:     percentile(values, 75),
		P95:     percentile(values, 95),
		Max:     values[len(values)-1],
		Samples: len(values),
	}
}

// percentile returns the value at the given percentile (0-100) of a
// sorted ascending slice. The implementation uses the
// highest-of-lowest-p%% interpretation:
//
//	idx = floor(p/100 * N) - 1, clamped to [0, N-1]
//
// For N=10 sorted ascending [1000..10000] this yields:
//
//	P50 -> sorted[4]  (5000)
//	P75 -> sorted[6]  (7000)
//	P95 -> sorted[8]  (9000)
//
// and for N=1 every percentile returns the single sample.
func percentile(sorted []uint64, p int) uint64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	idx := (p * n) / 100
	if idx > 0 {
		idx--
	}
	if idx >= n {
		idx = n - 1
	}
	return sorted[idx]
}
