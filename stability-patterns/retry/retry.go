// Package retry implements a retry pattern for idempotent operations
// that may fail transiently in distributed systems.
package retry

import (
	"context"
	"log"
	"time"
)

// Effector represents an operation that may fail transiently.
// Retry wraps an Effector to transparently retry failed calls.
type Effector func(context.Context) (string, error)

// Retry returns a wrapper that retries the given Effector on failure,
// waiting delay between attempts, up to maxRetries.
//
// Only use with idempotent operations to avoid side effects.
func Retry(effector Effector, maxRetries int, delay time.Duration) Effector {
	return func(ctx context.Context) (string, error) {
		for r := 0; ; r++ {
			response, err := effector(ctx)
			if err != nil || r >= maxRetries {
				return response, err
			}

			log.Printf("Attempt %d failed; retrying in %v", r+1, delay)
		}
	}
}
