package ttlcache

import (
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	c := New[string, int](20*time.Millisecond, 2)
	c.Set("a", 1)
	if v, ok := c.Get("a"); !ok || v != 1 {
		t.Fatal("get")
	}
	c.Update("a", func(v int) int { return v + 4 })
	if v, _ := c.Get("a"); v != 5 {
		t.Fatal("update")
	}
	c.Set("b", 2)
	c.Set("c", 3)
	if _, ok := c.Get("c"); !ok {
		t.Fatal("set after full")
	}
	time.Sleep(30 * time.Millisecond)
	if _, ok := c.Get("c"); ok {
		t.Fatal("expiry")
	}
	c.Set("d", 4)
	c.Delete("d")
	if _, ok := c.Get("d"); ok {
		t.Fatal("delete")
	}
}
