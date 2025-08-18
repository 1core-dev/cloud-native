// Package sharding implements a thread-safe map with partitioned data (sharding)
// to reduce lock contention in highly concurrent environments.

// ShardedMap splits data into multiple shards, each protected by its own read-write lock.
// This allows concurrent operations on different parts of the map without blocking each other.

// The pattern is particularly useful for scenarios where:
//  - You need efficient parallel access to a shared key-value store.
//  - Lock contention from frequent reads/writes becomes a bottleneck.
//  - Data fits within a single service instance and does not require horizontal sharding across distributed systems.

// Sharding improves performance by partitioning data within a single service instance (vertical sharding),
// making it ideal for services with high concurrency but without the complexity of distributed data systems.

package sharding

import (
	"fmt"
	"hash/fnv"
	"reflect"
	"sync"
)

// Shard represents a single partition of a ShardedMap.
// Each shard is an independent, lock-protected map that stores a subset of keys.
type Shard[K comparable, V any] struct {
	sync.RWMutex         // compose from sync.RWMutex
	items        map[K]V // contains the shard's data
}

// ShardedMap is a map abstraction composed of multiple shards.
// It provides concurrent access to key-value pairs with reduced lock contention.
type ShardedMap[K comparable, V any] []*Shard[K, V]

// NewShardedMap creates and returns a ShardedMap with the specified number of shards.
// Each shard is initialized and protected with its own read-write mutex.
func NewShardedMap[K comparable, V any](nshards int) ShardedMap[K, V] {
	shards := make([]*Shard[K, V], nshards) // Initialize a *Shards slice

	// for i := 0; i < nshards; i++ {
	for i := range nshards {
		shard := make(map[K]V)
		shards[i] = &Shard[K, V]{items: shard} // A ShardedMap is a slice
	}

	return shards
}

// Get retrieves the value associated with the given key.
// A read lock is acquired on the appropriate shard.
func (m ShardedMap[K, V]) Get(key K) V {
	shard := m.getShard(key)
	shard.RLock()
	defer shard.RUnlock()

	return shard.items[key]
}

// Set inserts or updates the value associated with the given key.
// A write lock is acquired on the appropriate shard.
func (m ShardedMap[K, V]) Set(key K, value V) {
	shard := m.getShard(key)
	shard.Lock()
	defer shard.Unlock()

	shard.items[key] = value
}

// Keys returns all keys from all shards as a single slice.
// Each shard is read concurrently, and keys are aggregated safely.
func (m ShardedMap[K, V]) Keys() []K {
	var keys []K      // Declare an empty keys slice
	var mu sync.Mutex // Mutex for write safety to keys

	var wg sync.WaitGroup // Create a wait group and add a
	wg.Add(len(m))        // wait value for each slice

	for _, shard := range m { // Run a goroutine for each slice in m
		go func(s *Shard[K, V]) {
			s.RLock() // Establish a read lock on s

			defer wg.Done()   // Tell the WaitGroup it's done
			defer s.RUnlock() // Release of the read lock

			for key := range s.items { // Get the slice's keys
				mu.Lock()
				keys = append(keys, key)
				mu.Unlock()
			}
		}(shard)
	}

	wg.Wait() // Block until all goroutines are done

	return keys // Return combined keys slice
}

// getShardIndex returns the index of the shard corresponding to the given key.
// It uses FNV-1a hashing on the key’s string representation to ensure even distribution.
func (m ShardedMap[K, V]) getShardIndex(key K) int {
	str := reflect.ValueOf(key).String() // Get string representation o key
	hash := fnv.New32a()                 // Get hash implementation
	hash.Write([]byte(str))              // Write bytes to the hash
	sum := int(hash.Sum32())             // Get the resulting checksum
	return sum % len(m)                  // Mod by len(m) to get index
}

// getShard returns the shard responsible for the given key.
func (m ShardedMap[K, V]) getShard(key K) *Shard[K, V] {
	index := m.getShardIndex(key)
	return m[index]
}

func main() {
	m := NewShardedMap[string, int](3)
	keys := []string{"key1", "key2", "key3", "key4", "key5", "key6", "key7"}

	for i, k := range keys {
		m.Set(k, i+1)

		fmt.Printf("%s: shard=%d value=%d\n", k, m.getShardIndex(k), m.Get(k))
	}

	fmt.Println(m.Keys())
}
