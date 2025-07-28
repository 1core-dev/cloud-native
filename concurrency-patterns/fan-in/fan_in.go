// Package fanin provides a concurrency pattern that merges multiple input
// channels into a single output channel.
//
// This "fan-in" pattern is useful when you have multiple sources of data
// (e.g. concurrent workers) and want to consume their outputs as a unified stream.
package fanin

import (
	"fmt"
	"sync"
	"time"
)

// Funnel multiplexes zero or more input channels into a single output channel.
//
// Each source is read in its own goroutine, so values from any input
// are forwarded to the destination as soon as they're available.
// When all sources are closed, the destination is closed automatically.
func Funnel(sources ...<-chan int) <-chan int {
	dest := make(chan int) // Shared output channel

	wg := sync.WaitGroup{} // Used to automatically close dest when sources are closed

	wg.Add(len(sources)) // Set size of WaitGroup

	for _, ch := range sources { // Start goroutine for each source
		go func(ch <-chan int) {
			// Forward values from each input channel to the shared output
			defer wg.Done() // Notify WaitGroup when ch closes

			for n := range ch {
				dest <- n
			}
		}(ch)

		go func() { // Start a goroutine to close dest after all sources close
			wg.Wait()
			close(dest)
		}()

	}
	return dest
}

func main() {
	var sources []<-chan int // Declare an empty channel slice

	for range 3 {
		ch := make(chan int)
		sources = append(sources, ch) // Create a channel; add to sources

		go func() { // Run a toy goroutine to each
			defer close(ch) // Close ch when the routine ends

			for i := 1; i <= 5; i++ {
				ch <- i
				time.Sleep(time.Second)
			}
		}()
	}

	// Merge all input streams into one
	dest := Funnel(sources...)

	// Consume merged values as a single stream
	for val := range dest {
		fmt.Println(val)
	}
}
