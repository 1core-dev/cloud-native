package circuitbreaker

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrServiceUnavailable is an error indicating that the service is unavailable.
var ErrServiceUnavailable = errors.New("service unavailable")

// Circuit represents a function that can be executed with a context.
type Circuit func(context.Context) (string, error)

// Breaker creates a circuit breaker that tracks failures and controls
// access to a service based on a failure threshold.
func Breaker(circuit Circuit, threshold int) Circuit {
	var (
		failures int
		last     time.Time
		mu       sync.RWMutex // Use RWMutex to allow concurrent reads
	)

	// Return a new circuit breaker function
	return func(ctx context.Context) (string, error) {
		mu.RLock() // Lock for read access before checking the failure count

		d := failures - threshold

		// If the number of failures exceeds the threshold, calculate retry delay
		if d >= 0 {
			shouldRetryAt := last.Add((2 << d) * time.Second)

			if !time.Now().After(shouldRetryAt) {
				mu.RUnlock() // Unlock read lock before returning early
				return "", ErrServiceUnavailable
			}
		}

		mu.RUnlock() // Unlock read lock after check

		// Execute the actual circuit function
		response, err := circuit(ctx)

		mu.Lock() // Lock for writing to shared resources (failures, last)
		defer mu.Unlock()

		last = time.Now() // Record time of the most recent attempt

		if err != nil {
			failures++
			// Circuit returned an error so we count the failure and return
			return response, err
		}

		failures = 0 // Reset failure count on success

		return response, nil
	}
}
