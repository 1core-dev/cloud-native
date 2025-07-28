// Package fanout implements the Fan-Out concurrency pattern.
//
// Fan-Out reads from a single input channel and distributes values across
// multiple output channels. Each output has its own goroutine that competes
// to read from the input, enabling parallel consumption and implicit load
// balancing among workers.
package fanout

import (
	"fmt"
	"sync"
)

// Split distributes values from a single input to n output channels.
//
// Each output is served by a goroutine competing to pull from the shared input.
// This enables implicit load balancing and parallel consumption.
func Split(sources <-chan int, n int) []<-chan int {
	var dests []<-chan int // Declare the dests slice

	for range n { // Create n destination channels
		ch := make(chan int)
		dests = append(dests, ch)

		// Each output channel gets a goroutine that pulls from the shared input.
		// All goroutines compete for incoming data, enabling load distribution.
		go func() {
			defer close(ch)

			for val := range sources {
				ch <- val
			}
		}()
	}

	return dests
}

func main() {
	source := make(chan int) // The input channel

	// Fan-out the input stream to 5 independent output channels.
	dests := Split(source, 5)

	// Feed the source with number 1..100 to source and close it when we're done.
	go func() {
		for i := 0; i <= 100; i++ {
			source <- i
		}

		close(source)
	}()

	var wg sync.WaitGroup // Use WaitGroup to wait until the output channels all close
	wg.Add(len(dests))

	// Each output channel is handled by a separate consumer.
	// Since channels compete for input, values are evenly distributed.
	for i, d := range dests {
		go func(i int, d <-chan int) {
			defer wg.Done()

			for val := range d {
				fmt.Printf("#%d channel got %d value\n", i, val)
			}
		}(i, d)
	}
	wg.Wait()
}
