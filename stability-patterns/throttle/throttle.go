// Package throttle provides a token bucket rate limiter for effectful functions.
// It bounds execution rate to prevent overload and smooth traffic spikes.
package throttle

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Effector is a function that performs work under context control.
type Effector func(context.Context) (string, error)

// Throttle applies a token bucket limit to an Effector.
//
// It allows up to max calls in burst, with refill tokens added every interval.
// If no tokens remain, the call is rejected.
func Throttle(effector Effector, max uint, refill uint, d time.Duration) Effector {
	var (
		tokens = max // current token count
		once   sync.Once
		mu     sync.Mutex
	)

	return func(ctx context.Context) (string, error) {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}

		// Start background refill loop once
		once.Do(func() {
			ticker := time.NewTicker(d)

			go func() {
				defer ticker.Stop()

				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						mu.Lock()
						t := min(tokens+refill, max)
						tokens = t
						mu.Unlock()
					}

				}
			}()
		})
		mu.Lock()
		defer mu.Unlock()

		if tokens <= 0 {
			return "", fmt.Errorf("too many calls")
		}

		tokens--
		return effector(ctx)
	}

}
