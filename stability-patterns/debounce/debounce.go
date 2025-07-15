// Package debounce provides wrappers to limit how often a function executes.
// It ensures that only the first or last call in a burst of calls is processed.
package debounce

import (
	"context"
	"sync"
	"time"
)

// Circuit defines a cancelable operation controlled by debounce logic.
type Circuit func(context.Context) (string, error)

// DebounceFirst allows only the first call through in a time window.
// Subsequent calls return cached result.
//
// Use when you want immediate response and ignore repeats.
func DebounceFirst(circuit Circuit, d time.Duration) Circuit {
	var (
		threshold time.Time
		result    string
		err       error
		mu        sync.Mutex
	)

	return func(ctx context.Context) (string, error) {
		mu.Lock()
		defer mu.Unlock()

		if time.Now().Before(threshold) {
			// Suppressed: return cached result
			return result, err
		}

		// Executed: store result and delay next execution window
		result, err = circuit(ctx)
		threshold = time.Now().Add(d)

		return result, err
	}
}

// DebounceFirstContext runs every call but cancels any prior still running.
//
// Use when each call has side effects but only one active call at a time is allowed.
func DebounceFirstContext(circuit Circuit, d time.Duration) Circuit {
	var (
		threshold  time.Time
		mu         sync.Mutex
		lastCtx    context.Context
		lastCancel context.CancelFunc
	)

	return func(ctx context.Context) (string, error) {
		mu.Lock()

		// Cancel prior call in progress
		if time.Now().Before(threshold) {
			lastCancel()
		}

		// Always invoke the function, but reset window
		lastCtx, lastCancel = context.WithCancel(ctx)
		threshold = time.Now().Add(d)

		mu.Unlock()

		result, err := circuit(lastCtx)

		return result, err
	}
}

// DebounceLast waits until calls stop coming for the duration d,
// then runs the last call only.
//
// Use this when you want to wait for a pause in activity
// before doing something, like waiting for a user to stop typing.
func DebounceLast(circuit Circuit, d time.Duration) Circuit {
	var (
		mu     sync.Mutex
		timer  *time.Timer
		cctx   context.Context
		cancel context.CancelFunc
	)

	return func(ctx context.Context) (string, error) {
		mu.Lock()

		// Cancel previous timer and execution
		if timer != nil {
			timer.Stop()
			cancel()
		}

		// Setup new context for this call
		cctx, cancel = context.WithCancel(ctx)
		ch := make(chan struct {
			result string
			err    error
		}, 1)

		// Schedule execution after delay
		timer = time.AfterFunc(d, func() {
			r, e := circuit(cctx)
			ch <- struct {
				result string
				err    error
			}{r, e}
		})

		mu.Unlock()

		select {
		case res := <-ch:
			return res.result, res.err // Call completed
		case <-cctx.Done():
			return "", cctx.Err() // Cancelled
		}
	}
}
