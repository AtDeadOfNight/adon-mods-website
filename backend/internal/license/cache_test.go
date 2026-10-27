package license

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

func overlappingChecks(t *testing.T, cache *Cache[string], key string, checks int, answer Result) int32 {
	t.Helper()
	release := make(chan struct{})
	var fetches atomic.Int32
	fetch := func() Result {
		fetches.Add(1)
		<-release
		return answer
	}
	results := make(chan Result, checks)
	for range checks {
		go func() { results <- cache.Check(t.Context(), key, fetch) }()
	}
	synctest.Wait()
	close(release)
	for range checks {
		if result := <-results; result != answer {
			t.Fatalf("result %v, want %v", result, answer)
		}
	}
	return fetches.Load()
}

func TestOverlappingChecksShareOneFetch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		if fetches := overlappingChecks(t, NewCache[string](time.Now), "owner", 5, Owned); fetches != 1 {
			t.Fatalf("%d fetches for overlapping checks, want 1", fetches)
		}
	})
}

func TestWaitersShareAFailureWithoutCachingIt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cache := NewCache[string](time.Now)
		if fetches := overlappingChecks(t, cache, "flaky", 3, Failed); fetches != 1 {
			t.Fatalf("%d fetches for overlapping checks, want 1", fetches)
		}
		if result := cache.Check(t.Context(), "flaky", func() Result { return Owned }); result != Owned {
			t.Fatalf("after the failure: %v, want %v", result, Owned)
		}
	})
}

func TestWaiterFetchesWhenTheFetchingRequestLeaves(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cache := NewCache[string](time.Now)
		leaving, leave := context.WithCancel(t.Context())
		leaderResult := make(chan Result)
		go func() {
			leaderResult <- cache.Check(leaving, "owner", func() Result {
				<-leaving.Done()
				return Failed
			})
		}()
		synctest.Wait()
		waiterResult := make(chan Result)
		go func() {
			waiterResult <- cache.Check(t.Context(), "owner", func() Result { return Owned })
		}()
		synctest.Wait()
		leave()
		if result := <-leaderResult; result != Failed {
			t.Fatalf("the request that left got %v", result)
		}
		if result := <-waiterResult; result != Owned {
			t.Fatalf("the waiting request got %v instead of its own answer", result)
		}
	})
}

func TestWaiterThatLeavesFails(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cache := NewCache[string](time.Now)
		release := make(chan struct{})
		defer close(release)
		go cache.Check(t.Context(), "owner", func() Result {
			<-release
			return Owned
		})
		synctest.Wait()
		leaving, leave := context.WithCancel(t.Context())
		leave()
		if result := cache.Check(leaving, "owner", func() Result { return Owned }); result != Failed {
			t.Fatalf("a waiter whose request left got %v", result)
		}
	})
}

func TestPanickingFetchReleasesTheKey(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cache := NewCache[string](time.Now)
		func() {
			defer func() { recover() }()
			cache.Check(t.Context(), "owner", func() Result { panic("upstream parser bug") })
		}()
		if result := cache.Check(t.Context(), "owner", func() Result { return Owned }); result != Owned {
			t.Fatalf("result %v, want %v", result, Owned)
		}
	})
}

type testClock struct{ now time.Time }

func (c *testClock) read() time.Time            { return c.now }
func (c *testClock) advance(step time.Duration) { c.now = c.now.Add(step) }

func countingFetch(result Result, calls *int) func() Result {
	return func() Result {
		*calls++
		return result
	}
}

func TestCacheKeepsAnswersForAWeek(t *testing.T) {
	clock := &testClock{now: testNow}
	cache := NewCache[string](clock.read)
	for _, result := range []Result{Owned, NotOwned} {
		calls := 0
		key := fmt.Sprint("account", result)
		cache.Check(t.Context(), key, countingFetch(result, &calls))
		clock.advance(cacheLifetime - time.Millisecond)
		if got := cache.Check(t.Context(), key, countingFetch(Unknown, &calls)); got != result || calls != 1 {
			t.Fatalf("%v within 7 days: %v after %d fetches", result, got, calls)
		}
		clock.advance(time.Millisecond)
		if got := cache.Check(t.Context(), key, countingFetch(Unknown, &calls)); got != Unknown || calls != 2 {
			t.Fatalf("%v after 7 days: %v after %d fetches", result, got, calls)
		}
	}
	for _, result := range []Result{Unknown, Failed} {
		calls := 0
		cache.Check(t.Context(), "flaky", countingFetch(result, &calls))
		cache.Check(t.Context(), "flaky", countingFetch(result, &calls))
		if calls != 2 {
			t.Fatalf("%v must not be cached, fetched %d times", result, calls)
		}
	}
}

func TestCacheCap(t *testing.T) {
	clock := &testClock{now: testNow}
	cache := NewCache[int64](clock.read)
	cache.limit = 3
	check := func(key int64, result Result) {
		cache.Check(t.Context(), key, func() Result { return result })
	}
	cached := func(key int64) (Result, bool) {
		fetched := false
		result := cache.Check(t.Context(), key, func() Result {
			fetched = true
			return Unknown
		})
		return result, !fetched
	}
	expect := func(step string, present []int64, absent []int64) {
		t.Helper()
		if len(cache.entries) != cache.limit {
			t.Fatalf("%s: %d entries, want %d", step, len(cache.entries), cache.limit)
		}
		for _, key := range present {
			if _, ok := cached(key); !ok {
				t.Fatalf("%s: entry %d lost", step, key)
			}
		}
		for _, key := range absent {
			if _, ok := cached(key); ok {
				t.Fatalf("%s: entry %d kept", step, key)
			}
		}
	}

	check(1, NotOwned)
	clock.advance(time.Hour)
	check(2, Owned)
	clock.advance(time.Hour)
	check(3, Owned)
	clock.advance(cacheLifetime - 2*time.Hour)
	check(4, NotOwned)
	expect("the expired entry is pruned first", []int64{2, 3, 4}, []int64{1})

	clock.advance(time.Minute)
	check(5, Owned)
	expect("the entry closest to expiring makes room", []int64{3, 4, 5}, []int64{2})
	if result, _ := cached(5); result != Owned {
		t.Fatalf("entry 5 is %v", result)
	}
}
