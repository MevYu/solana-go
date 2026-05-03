package jsonrpc

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDefaultRetryPolicy_MaxAttemptsEnforced(t *testing.T) {
	p := DefaultRetryPolicy()
	if _, retry := p.ShouldRetry(5, errors.New("x")); retry {
		t.Error("should not retry at attempt 5 (MaxAttempts)")
	}
	if _, retry := p.ShouldRetry(10, errors.New("x")); retry {
		t.Error("should not retry at attempt 10")
	}
}

func TestDefaultRetryPolicy_NoRetryOnContextErrors(t *testing.T) {
	p := DefaultRetryPolicy()
	if _, retry := p.ShouldRetry(1, context.Canceled); retry {
		t.Error("should not retry context.Canceled")
	}
	if _, retry := p.ShouldRetry(1, context.DeadlineExceeded); retry {
		t.Error("should not retry context.DeadlineExceeded")
	}
}

func TestDefaultRetryPolicy_RetriesOnServerAndRateLimit(t *testing.T) {
	p := DefaultRetryPolicy()
	for _, code := range []int{500, 502, 503, 504, 429} {
		if _, retry := p.ShouldRetry(1, &httpError{StatusCode: code}); !retry {
			t.Errorf("should retry %d", code)
		}
	}
}

func TestDefaultRetryPolicy_NoRetryOn4xx(t *testing.T) {
	p := DefaultRetryPolicy()
	for _, code := range []int{400, 401, 403, 404, 422} {
		if _, retry := p.ShouldRetry(1, &httpError{StatusCode: code}); retry {
			t.Errorf("should not retry %d", code)
		}
	}
}

func TestDefaultRetryPolicy_RetriesGenericError(t *testing.T) {
	p := DefaultRetryPolicy()
	// A plain error (simulating a network failure) is retried by default.
	if _, retry := p.ShouldRetry(1, errors.New("connection reset by peer")); !retry {
		t.Error("generic error should be retried")
	}
}

// TestDefaultRetryPolicy_DelayGrowsBetweenAttempts verifies that the
// backoff upper bound grows as attempts increase, using a sampling
// probe because full jitter means individual samples can be low.
func TestDefaultRetryPolicy_DelayGrowsBetweenAttempts(t *testing.T) {
	p := DefaultRetryPolicy()
	err := &httpError{StatusCode: 503}
	maxAt := func(attempt int) time.Duration {
		var m time.Duration
		for i := 0; i < 128; i++ {
			d, _ := p.ShouldRetry(attempt, err)
			if d > m {
				m = d
			}
		}
		return m
	}
	m1 := maxAt(1)
	m2 := maxAt(2)
	m3 := maxAt(3)
	if m2 <= m1 {
		t.Errorf("attempt 2 max (%v) should exceed attempt 1 max (%v)", m2, m1)
	}
	if m3 <= m2 {
		t.Errorf("attempt 3 max (%v) should exceed attempt 2 max (%v)", m3, m2)
	}
}

func TestDefaultRetryPolicy_DelayClampedToMax(t *testing.T) {
	p := DefaultRetryPolicy()
	err := &httpError{StatusCode: 503}
	// At attempt 4 the raw exponential would be 200ms << 3 == 1.6s,
	// still under the 10s cap. At attempt 20 the raw exponential
	// would overflow; the clamp should keep samples under 10s.
	for _, attempt := range []int{4, 10, 20} {
		for i := 0; i < 64; i++ {
			d, _ := p.ShouldRetry(attempt, err)
			if d > 10*time.Second {
				t.Fatalf("attempt %d: delay %v exceeds 10s cap", attempt, d)
			}
		}
	}
}
