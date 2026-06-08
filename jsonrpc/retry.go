package jsonrpc

import (
	"context"
	"errors"
	"math/rand/v2"
	"time"
)

// RetryPolicy decides whether a failed attempt should be retried
// and how long to wait before the next attempt. Implementations are
// called once per failure; they should be pure with respect to the
// inputs (except for jitter) and safe for concurrent use.
type RetryPolicy interface {
	// ShouldRetry is invoked after each failed attempt. attempt is
	// the 1-based count of attempts so far. err is the reason the
	// last attempt failed. When retry is true, the caller waits for
	// delay before trying again; when false, delay is ignored and
	// the error is returned to the user.
	ShouldRetry(attempt int, err error) (delay time.Duration, retry bool)
}

// DefaultRetryPolicy returns the policy used by NewClient when the
// caller does not supply one: up to 5 attempts, exponential backoff
// starting at 200ms (capped at 10s), with full jitter on the delay.
//
// Only transient errors are retried: HTTP 5xx and 429 responses, and
// non-context network errors. Context cancellation and deadline
// errors are never retried.
func DefaultRetryPolicy() RetryPolicy {
	return &exponentialBackoff{
		MaxAttempts: 5,
		Base:        200 * time.Millisecond,
		Max:         10 * time.Second,
	}
}

// exponentialBackoff is the default RetryPolicy implementation. It
// is exported only through DefaultRetryPolicy to keep the knobs
// inside the package until a concrete user need for tuning emerges.
type exponentialBackoff struct {
	MaxAttempts int
	Base        time.Duration
	Max         time.Duration
}

func (e *exponentialBackoff) ShouldRetry(attempt int, err error) (time.Duration, bool) {
	if attempt >= e.MaxAttempts {
		return 0, false
	}
	if !isRetryable(err) {
		return 0, false
	}
	// Exponential growth, clamped to Max, then full jitter.
	shift := uint(attempt - 1)
	if shift > 10 {
		shift = 10 // avoid int64 overflow on pathological attempt counts
	}
	delay := e.Base << shift
	if delay > e.Max || delay <= 0 {
		delay = e.Max
	}
	return time.Duration(rand.Int64N(int64(delay))), true
}

// isRetryable classifies an error as a transient failure worth
// retrying. It is called by the default RetryPolicy and is package
// private because its heuristics are paired with the default policy.
func isRetryable(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var httpErr *httpError
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode >= 500 || httpErr.StatusCode == 429
	}
	// Anything else (network error, DNS failure, I/O error, ...) is
	// treated as potentially transient.
	return true
}
