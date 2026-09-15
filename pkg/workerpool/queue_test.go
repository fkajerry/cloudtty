package workpool

import (
	"sync"
	"testing"
	"time"
)

// TestQueueConcurrentAllAndAdd guards against a deadlock between All() and Add().
//
// All() used to call q.Len() while already holding the read lock. Go's RWMutex is
// write-preferring, so once a concurrent Add() blocks on Lock(), the nested RLock()
// inside Len() blocks too, and neither goroutine can ever make progress.
func TestQueueConcurrentAllAndAdd(t *testing.T) {
	q := newQueue()
	for i := 0; i < 128; i++ {
		q.Add(i)
	}

	done := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < 8; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					q.All()
				}
			}
		}()
		go func(base int) {
			defer wg.Done()
			for n := 0; ; n++ {
				select {
				case <-done:
					return
				default:
					q.Add(base*1_000_000 + n)
				}
			}
		}(i)
	}

	finished := make(chan struct{})
	go func() {
		wg.Wait()
		close(finished)
	}()

	time.Sleep(2 * time.Second)
	close(done)

	select {
	case <-finished:
	case <-time.After(10 * time.Second):
		t.Fatal("deadlock: goroutines calling All()/Add() did not finish")
	}
}

// TestQueueGetIsExclusive ensures Get() does not hand the same item to two callers.
// Get() used to mutate q.queue and q.dirty while holding only a read lock.
func TestQueueGetIsExclusive(t *testing.T) {
	const items = 2000

	q := newQueue()
	for i := 0; i < items; i++ {
		q.Add(i)
	}

	var mu sync.Mutex
	seen := map[interface{}]int{}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				item := q.Get()
				if item == nil {
					return
				}
				mu.Lock()
				seen[item]++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if len(seen) != items {
		t.Errorf("got %d distinct items, want %d", len(seen), items)
	}
	for item, count := range seen {
		if count != 1 {
			t.Errorf("item %v returned %d times, want 1", item, count)
		}
	}
	if q.Len() != 0 {
		t.Errorf("queue length is %d after draining, want 0", q.Len())
	}
}
