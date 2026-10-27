package license

import (
	"context"
	"sync"
	"time"
)

type Result int

const (
	// Unknown: Steam answered, but not in the shape that lists games.
	Unknown Result = iota
	Owned
	NotOwned
	// Failed: Steam could not be asked, or its answer was not JSON.
	Failed
)

const (
	cacheLifetime = 7 * 24 * time.Hour
	maxCached     = 100_000
)

func (r Result) cacheable() bool {
	return r == Owned || r == NotOwned
}

type cacheEntry struct {
	result    Result
	expiresAt time.Time
}

func (e cacheEntry) expired(now time.Time) bool {
	return !now.Before(e.expiresAt)
}

type pendingCheck struct {
	done   chan struct{}
	result Result
	// The fetching request went away without an answer, so the waiters fetch for themselves.
	abandoned bool
}

type Cache[K comparable] struct {
	mu      sync.Mutex
	entries map[K]cacheEntry
	pending map[K]*pendingCheck
	limit   int
	now     func() time.Time
}

func NewCache[K comparable](now func() time.Time) *Cache[K] {
	return &Cache[K]{entries: map[K]cacheEntry{}, pending: map[K]*pendingCheck{}, limit: maxCached, now: now}
}

// Overlapping checks of one key share one fetch, so a burst of requests for one account costs
// one upstream call.
func (c *Cache[K]) Check(ctx context.Context, key K, fetch func() Result) Result {
	for {
		result, pending, fetching := c.join(key)
		switch {
		case pending == nil:
			return result
		case fetching:
			return c.fetchFor(ctx, key, pending, fetch)
		}
		select {
		case <-pending.done:
		case <-ctx.Done():
			return Failed
		}
		if !pending.abandoned {
			return pending.result
		}
	}
}

func (c *Cache[K]) join(key K) (Result, *pendingCheck, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if result, ok := c.lookupLocked(key); ok {
		return result, nil, false
	}
	if pending, ok := c.pending[key]; ok {
		return Unknown, pending, false
	}
	pending := &pendingCheck{done: make(chan struct{})}
	c.pending[key] = pending
	return Unknown, pending, true
}

// The deferred settle also runs when fetch panics, so the waiters are never left hanging.
func (c *Cache[K]) fetchFor(ctx context.Context, key K, pending *pendingCheck, fetch func() Result) (result Result) {
	result = Failed
	defer func() { c.settle(key, pending, result, ctx.Err() != nil) }()
	return fetch()
}

func (c *Cache[K]) settle(key K, pending *pendingCheck, result Result, canceled bool) {
	c.mu.Lock()
	if result.cacheable() {
		c.storeLocked(key, result)
	}
	delete(c.pending, key)
	pending.result = result
	pending.abandoned = canceled && !result.cacheable()
	c.mu.Unlock()
	close(pending.done)
}

func (c *Cache[K]) lookupLocked(key K) (Result, bool) {
	entry, ok := c.entries[key]
	if !ok {
		return Unknown, false
	}
	if entry.expired(c.now()) {
		delete(c.entries, key)
		return Unknown, false
	}
	return entry.result, true
}

func (c *Cache[K]) storeLocked(key K, result Result) {
	now := c.now()
	if _, replacing := c.entries[key]; !replacing && len(c.entries) >= c.limit {
		c.makeRoom(now)
	}
	c.entries[key] = cacheEntry{result: result, expiresAt: now.Add(cacheLifetime)}
}

// makeRoom drops expired entries, then the entry closest to expiring if the cache is still full.
func (c *Cache[K]) makeRoom(now time.Time) {
	var soonestKey K
	var soonest time.Time
	for key, entry := range c.entries {
		if entry.expired(now) {
			delete(c.entries, key)
			continue
		}
		if soonest.IsZero() || entry.expiresAt.Before(soonest) {
			soonestKey, soonest = key, entry.expiresAt
		}
	}
	if len(c.entries) >= c.limit {
		delete(c.entries, soonestKey)
	}
}
