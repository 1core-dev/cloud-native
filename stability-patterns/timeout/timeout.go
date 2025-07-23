// Package timeout wraps a long-running function with context-based timeout logic.
// It allows the caller to stop waiting if the function takes too long.
//
// This is useful when dealing with third-party or legacy code that does not
// support context.Context. It isolates faults and avoids blocking on slow operations.
package timeout

import "context"

// SlowFunction defines a function that may run for an unbounded duration.
type SlowFunction func(string) (string, error)

// WithContext adds context-aware timeout support to a SlowFunction.
type WithContext func(context.Context, string) (string, error)

// Timeout wraps a SlowFunction, returning a context-aware version.
// If the context is done before the function returns, the error from ctx.Err() is returned.
func Timeout(fn SlowFunction) WithContext {
	return func(ctx context.Context, arg string) (string, error) {
		ch := make(chan struct {
			result string
			err    error
		}, 1)

		go func() {
			res, err := fn(arg)
			ch <- struct {
				result string
				err    error
			}{res, err}
		}()

		// Wait for either the slow function to finish or the context to be done (timeout/cancellation).
		select {
		case res := <-ch:
			// Slow function finished before the timeout.
			return res.result, res.err
		case <-ctx.Done():
			// Timeout or cancellation occurred before slow function completed.
			// Caller stops waiting but the slow function continues running in background.
			return "", ctx.Err()
		}
	}
}

// TimeoutWithPrecheck prevents starting work if the context is already done,
// avoiding unnecessary resource use and potential leaks.
func TimeoutWithPrecheck(fn SlowFunction) WithContext {
	return func(ctx context.Context, arg string) (string, error) {
		// Avoid launching the goroutine when timeout or cancellation already occurred.
		if err := ctx.Err(); err != nil {
			return "", err
		}

		ch := make(chan struct {
			result string
			err    error
		}, 1)

		// Run the slow operation concurrently to enable cancellation via context.
		go func() {
			res, err := fn(arg)
			ch <- struct {
				result string
				err    error
			}{res, err}
		}()

		// Return whichever happens first: function completes or context expires.
		select {
		case res := <-ch:
			return res.result, res.err
		case <-ctx.Done():
			// The function may still be running; caller stops waiting.
			return "", ctx.Err()
		}
	}
}
