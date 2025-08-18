// Package workerpool provides a worker pool pattern for processing tasks concurrently.
// It uses a fixed number of workers to process jobs from a channel and send results
// back via another channel. This approach helps manage concurrency while controlling
// the number of active goroutines.
package main

import (
	"fmt"
	"sync"
	"time"
)

// worker processes jobs from the jobs channel and sends the results to the results channel.
// It runs in its own goroutine and processes jobs concurrently.
func worker(id int, jobs <-chan int, results chan<- int) {
	for j := range jobs {
		fmt.Println("Worker", id, "started job", j)
		time.Sleep(time.Second) // Simulate processing by sleeping for 1 second
		results <- j * 10       // Send result, job input multiplied by 10
	}
}

func main() {
	jobs := make(chan int, 25)
	results := make(chan int)
	wg := sync.WaitGroup{}

	for w := 1; w <= 3; w++ { // Spawn 3 workers processes
		go worker(w, jobs, results)
	}

	for j := range 25 { // Send jobs to workers
		wg.Add(1)
		jobs <- j
	}

	go func() {
		wg.Wait()
		close(jobs)
		close(results)
	}()

	for r := range results {
		fmt.Println("Got result:", r)
		wg.Done()
	}

}
