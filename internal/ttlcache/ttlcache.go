package ttlcache

import (
	"sync"
	"time"
)

type entry[V any] struct {
	v   V
	exp int64
}

type Cache[K comparable, V any] struct {
	mu  sync.RWMutex
	m   map[K]entry[V]
	ttl time.Duration
	max int
}

func New[K comparable, V any](ttl time.Duration, max int) *Cache[K, V] {
	return &Cache[K, V]{m: make(map[K]entry[V]), ttl: ttl, max: max}
}

func (c *Cache[K, V]) Get(k K) (V, bool) {
	c.mu.RLock()
	e, ok := c.m[k]
	c.mu.RUnlock()
	if !ok || e.exp < time.Now().UnixNano() {
		var zero V
		return zero, false
	}
	return e.v, true
}

func (c *Cache[K, V]) Set(k K, v V) {
	now := time.Now().UnixNano()
	c.mu.Lock()
	if len(c.m) >= c.max {
		for key, e := range c.m {
			if e.exp < now {
				delete(c.m, key)
			}
		}
		if len(c.m) >= c.max {
			clear(c.m)
		}
	}
	c.m[k] = entry[V]{v: v, exp: now + int64(c.ttl)}
	c.mu.Unlock()
}

func (c *Cache[K, V]) Delete(k K) {
	c.mu.Lock()
	delete(c.m, k)
	c.mu.Unlock()
}

func (c *Cache[K, V]) Clear() {
	c.mu.Lock()
	clear(c.m)
	c.mu.Unlock()
}

// Update applies fn to a live entry in place; it is a no-op when absent or expired.
func (c *Cache[K, V]) Update(k K, fn func(V) V) {
	c.mu.Lock()
	if e, ok := c.m[k]; ok && e.exp >= time.Now().UnixNano() {
		e.v = fn(e.v)
		c.m[k] = e
	}
	c.mu.Unlock()
}
