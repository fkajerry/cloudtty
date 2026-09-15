package workpool

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestQueueAllAndAddDoNotDeadlock guards against a deadlock between All() and Add().
//
// All() used to call q.Len() while already holding the read lock. Go's RWMutex is
// write-preferring: once a concurrent Add() blocks on Lock(), every subsequent
// RLock() blocks too, including the nested one inside Len(). All() then waits for
// the pending writer, and the writer waits for All()'s outer read lock. Neither
// ever makes progress.
//
// The queue is deliberately kept small (writers re-add the same few items, which
// Add() rejects as duplicates after taking the write lock) so that All() spends
// most of its time in the RLock -> Len window, which is where the race lives.
func TestQueueAllAndAddDoNotDeadlock(t *testing.T) {
	const (
		pairs       = 8
		items       = 8
		runFor      = 5 * time.Second
		stallWindow = 2 * time.Second
	)

	if runtime.GOMAXPROCS(0) < 2 {
		t.Skip("needs GOMAXPROCS >= 2 to interleave read and write lock acquisitions")
	}

	q := newQueue()
	for i := 0; i < items; i++ {
		q.Add(i)
	}

	var ops atomic.Uint64
	stop := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < pairs; i++ {
		wg.Add(2)

		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				q.All()
				ops.Add(1)
			}
		}()

		go func() {
			defer wg.Done()
			for n := 0; ; n++ {
				select {
				case <-stop:
					return
				default:
				}
				q.Add(n % items)
				ops.Add(1)
			}
		}()
	}

	var (
		deadline   = time.Now().Add(runFor)
		lastOps    = ops.Load()
		lastChange = time.Now()
	)
	for time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)

		switch cur := ops.Load(); {
		case cur != lastOps:
			lastOps, lastChange = cur, time.Now()
		case time.Since(lastChange) > stallWindow:
			t.Fatalf("deadlock: no queue operation completed in %s (%d ops total)", stallWindow, cur)
		}
	}

	close(stop)

	finished := make(chan struct{})
	go func() {
		wg.Wait()
		close(finished)
	}()

	select {
	case <-finished:
	case <-time.After(10 * time.Second):
		t.Fatal("deadlock: goroutines did not return after stop")
	}
}

// TestQueueGetIsExclusive ensures Get() never hands the same item to two callers.
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
