package handler

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSitemapCacheSingleflightAndStale(t *testing.T) {
	c := newSitemapCache()
	now := time.Now()
	c.now = func() time.Time { return now }
	var calls int32
	gate := make(chan struct{})
	build := func(context.Context) ([]byte, error) {
		n := atomic.AddInt32(&calls, 1)
		<-gate
		return []byte{byte('0' + n)}, nil
	}
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b, err := c.get(1, build)
			if err != nil || string(b) != "1" {
				t.Errorf("got %q %v", b, err)
			}
		}()
	}
	time.Sleep(50 * time.Millisecond)
	close(gate)
	wg.Wait()
	if calls != 1 {
		t.Fatalf("calls=%d", calls)
	}
	b, _ := c.get(1, build)
	if string(b) != "1" || calls != 1 {
		t.Fatal("fresh hit rebuilt")
	}
	now = now.Add(sitemapFreshTTL + time.Second)
	b, _ = c.get(1, build)
	if string(b) != "1" {
		t.Fatalf("stale not served: %q", b)
	}
	time.Sleep(50 * time.Millisecond)
	b, _ = c.get(1, build)
	if string(b) != "2" {
		t.Fatalf("not refreshed: %q", b)
	}
}

func TestSitemapCacheErrorKeepsStale(t *testing.T) {
	c := newSitemapCache()
	now := time.Now()
	c.now = func() time.Time { return now }
	if _, err := c.get(2, func(context.Context) ([]byte, error) { return []byte("ok"), nil }); err != nil {
		t.Fatal(err)
	}
	now = now.Add(sitemapFreshTTL + time.Second)
	b, err := c.get(2, func(context.Context) ([]byte, error) { return nil, errors.New("db") })
	if err != nil || string(b) != "ok" {
		t.Fatalf("%q %v", b, err)
	}
}
