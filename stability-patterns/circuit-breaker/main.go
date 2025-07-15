// Package circuitbreaker protects services from overload.
//
// It blocks calls after N failures, applies exponential backoff while open,
// and resets on success.
package circuitbreaker

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrServiceUnavailable signals that the circuit is currently open.
var ErrServiceUnavailable = errors.New("service unavailable")

// Circuit is a function that can be cancelled with context.
type Circuit func(context.Context) (string, error)

// Breaker wraps a function with circuit breaker logic.
// It tracks failures. After 'threshold' failures, it opens the circuit.
// While open, it blocks calls for some time using exponential backoff.
// If a call succeeds, it resets the failure counter.
func Breaker(circuit Circuit, threshold int) Circuit {
	var (
		failures int       // how many times the function failed
		last     time.Time // when the last attempt happened
		mu       sync.RWMutex
	)

	// Return a new circuit breaker function
	return func(ctx context.Context) (string, error) {
		mu.RLock()

		d := failures - threshold

		// Too many failures: wait before retrying
		if d >= 0 {
			shouldRetryAt := last.Add((2 << d) * time.Second)

			if !time.Now().After(shouldRetryAt) {
				mu.RUnlock()
				return "", ErrServiceUnavailable
			}
		}

		mu.RUnlock()

		// Execute the actual circuit function
		response, err := circuit(ctx)

		mu.Lock()
		defer mu.Unlock()

		last = time.Now()

		if err != nil {
			failures++
			return response, err
		}

		// Success: reset the failure count
		failures = 0

		return response, nil
	}
}
