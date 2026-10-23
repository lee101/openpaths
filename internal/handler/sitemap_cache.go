package handler

import (
	"context"
	"sync"
	"time"
)

const (
	sitemapFreshTTL = 10 * time.Minute
	sitemapStaleTTL = 24 * time.Hour
	sitemapBuildTTL = 2 * time.Minute
)

type sitemapEntry struct {
	body    []byte
	builtAt time.Time
}

type sitemapCall struct {
	done chan struct{}
	body []byte
	err  error
}

type sitemapCache struct {
	mu      sync.Mutex
	entries map[int]sitemapEntry
	calls   map[int]*sitemapCall
	now     func() time.Time
}

func newSitemapCache() *sitemapCache {
	return &sitemapCache{entries: map[int]sitemapEntry{}, calls: map[int]*sitemapCall{}, now: time.Now}
}

func (c *sitemapCache) get(key int, build func(context.Context) ([]byte, error)) ([]byte, error) {
	c.mu.Lock()
	e, ok := c.entries[key]
	age := c.now().Sub(e.builtAt)
	if ok && age < sitemapFreshTTL {
		c.mu.Unlock()
		return e.body, nil
	}
	call, inflight := c.calls[key]
	if !inflight {
		call = &sitemapCall{done: make(chan struct{})}
		c.calls[key] = call
		go c.run(key, call, build)
	}
	if ok && age < sitemapStaleTTL {
		c.mu.Unlock()
		return e.body, nil
	}
	c.mu.Unlock()
	<-call.done
	return call.body, call.err
}

func (c *sitemapCache) run(key int, call *sitemapCall, build func(context.Context) ([]byte, error)) {
	ctx, cancel := context.WithTimeout(context.Background(), sitemapBuildTTL)
	defer cancel()
	body, err := build(ctx)
	c.mu.Lock()
	if err == nil {
		c.entries[key] = sitemapEntry{body: body, builtAt: c.now()}
	}
	delete(c.calls, key)
	c.mu.Unlock()
	call.body, call.err = body, err
	close(call.done)
}
