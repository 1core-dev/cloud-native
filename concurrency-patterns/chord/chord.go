// Package chord implements the Chord pattern for synchronizing multiple input
// channels. It outputs a slice of values when all source channels have emitted
// at least one value.
//
// This pattern is useful when you need to collect inputs from multiple independent
// sources and act once all sources are ready.
package chord

import (
	"fmt"
	"sync"
	"time"
)

// Chord accepts multiple source channels and returns a single destination channel
// that emits a slice containing values from all sources once each has provided a value.
//
// It waits for all sources to send a value, packages the values into a slice, and
// sends the slice to the destination channel. The pattern ensures that results
// are only sent once all channels have emitted.

func Chord(sources ...<-chan int) <-chan []int {
	type input struct { // Used to send inputs between goroutines
		idx, input int
	}

	dest := make(chan []int)   // The output channel
	inputs := make(chan input) // An intermediate channel
	wg := sync.WaitGroup{}     // Used to close channels when all sources are closed
	wg.Add(len(sources))

	// Start goroutines to collect values from each source channel
	for i, ch := range sources {
		go func(i int, ch <-chan int) {
			defer wg.Done() // Notify WaitGroup when ch is closed

			for n := range ch {
				inputs <- input{i, n} // Transfer input to next goroutine
			}
		}(i, ch)
	}

	// Close the 'inputs' channel once all sources have finished
	go func() {
		wg.Wait()
		close(inputs)
	}()

	// Collect values from 'inputs' and emit when all sources have sent data
	go func() {
		res := make([]int, len(sources))   // Slice for incoming inputs
		sent := make([]bool, len(sources)) // Slice to track sent status
		count := len(sources)              // Counter for channels

		for r := range inputs {
			res[r.idx] = r.input // Update incoming input

			if !sent[r.idx] { // First input from channel?
				sent[r.idx] = true
				count--
			}

			if count == 0 {
				c := make([]int, len(res)) // Copy and send inputs slice
				copy(c, res)
				dest <- c

				count = len(sources) // Reset counter
				clear(sent)          // Clear status tracker
			}
		}

		close(dest)
	}()

	return dest
}

func main() {
	ch1 := make(chan int)
	ch2 := make(chan int)
	ch3 := make(chan int)

	go func() {
		for n := 1; n <= 4; n++ {
			ch1 <- n
			ch1 <- n * 2 // Writing twice to ch1
			ch2 <- n
			ch3 <- n
			time.Sleep(2 * time.Second)
		}
		close(ch1) // Closing all input channels
		close(ch2) // cause res to be closed
		close(ch3) // as well
	}()

	res := Chord(ch1, ch2, ch3)
	for s := range res {
		fmt.Println(s) // Read results
	}
}
