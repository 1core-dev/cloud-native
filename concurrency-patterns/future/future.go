// Package future implements the Future pattern, which is used to manage
// asynchronous operations and allow the caller to retrieve results once
// the operation completes.
package future

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Future is an interface that represents an asynchronous operation. It
// provides a method to retrieve the result once the operation completes.
type Future interface {
	// Result will block until the async operation finishes and then return
	// the result of the operation, or any error that occurred.
	Result() (string, error)
}

// InnerFuture is a concrete implementation of Future that holds the result
// of an asynchronous operation and ensures the result is computed only once.
type InnerFuture struct {
	once  sync.Once
	wg    sync.WaitGroup
	res   string
	err   error
	resCh <-chan string
	errCh <-chan error
}

// Result waits for the async operation to complete, retrieves the result,
// and handles any errors that occurred. It guarantees that the result is
// computed only once.
func (f *InnerFuture) Result() (string, error) {
	// This ensures that the result is only computed once, no matter how many
	// times Result() is called.
	f.once.Do(func() {
		// Block until the operation completes and then fetch the result.
		f.wg.Add(1)
		defer f.wg.Done()

		// Wait for the result and any potential error to be available.
		f.res = <-f.resCh
		f.err = <-f.errCh
	})

	// Block until the result is ready.
	f.wg.Wait()

	return f.res, f.err
}

// SlowFunction starts an asynchronous task that will take some time to finish.
// It simulates an operation by sleeping for 2 seconds, after which it
// provides a result. The function returns a Future that can be used to
// retrieve the result later.
func SlowFunction(ctx context.Context) Future {
	resCh := make(chan string)
	errCh := make(chan error)

	// Perform the long-running operation asynchronously in a separate goroutine.
	go func() {
		select {
		// Simulate a task that takes 2 seconds to complete.
		case <-time.After(2 * time.Second):
			// Once the task is completed, send the result and indicate no error.
			resCh <- "I slept for 2 seconds"
			errCh <- nil
		// If the operation is cancelled, handle it by sending an error.
		case <-ctx.Done():
			// Propagate the cancellation error through the channel.
			resCh <- ""
			errCh <- ctx.Err()
		}
	}()

	// Return the Future so that the caller can wait for the result.
	return &InnerFuture{resCh: resCh, errCh: errCh}
}

func main() {
	ctx := context.Background()

	// Call SlowFunction to start the async task. This returns a Future.
	future := SlowFunction(ctx)

	// Do other work while the task is running asynchronously in the background.

	// Wait for the result of the async operation.
	res, err := future.Result()

	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(res)
}
