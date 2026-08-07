package promptcache

import (
	"testing"
	"time"
)

func TestCostForFallback(t *testing.T) {
	cases := map[string]float64{
		"claude-opus-4-8":             5,
		"claude-opus-4-8-20260101":    5, // dated id -> prefix match
		"anthropic/claude-sonnet-5":   2, // routing prefix stripped
		"anthropic/claude-sonnet-4-6": 3,
		"claude-haiku-4-5":            1,
		"claude-opus-4-1":             15,
		"some-unknown-model":          5, // -> DefaultModel (opus 4.8)
	}
	for model, wantBase := range cases {
		if got := CostFor(model).BaseInputPer1M; got != wantBase {
			t.Errorf("CostFor(%q).BaseInputPer1M = %v, want %v", model, got, wantBase)
		}
	}
}

func ns(min float64) int64 { return int64(min * float64(time.Minute)) }

func TestDecideTTL(t *testing.T) {
	cost := CostFor("claude-opus-4-8")
	min := 2

	tests := []struct {
		name string
		gaps []float64 // inter-arrival gaps in minutes (n = len+1)
		want TTL
	}{
		// Single observation -> cold default.
		{"single", nil, TTL5m},
		// Tight reuse (every 1 min): one 5m write covers everything -> 5m wins.
		{"tight-1m", []float64{1, 1, 1, 1, 1}, TTL5m},
		// Reuse every ~30 min: 5m would re-write each time (1.25 each), 1h keeps
		// it warm as hits (0.1) -> 1h wins.
		{"every-30m", []float64{30, 30, 30, 30, 30}, TTL1h},
		// Rare reuse, gaps > 1h: every request re-writes regardless; caching is
		// pure overhead -> none.
		{"every-2h", []float64{120, 120}, TTLNone},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var times []int64
			var cur int64 = 0
			times = append(times, cur)
			for _, g := range tc.gaps {
				cur += ns(g)
				times = append(times, cur)
			}
			if got := decideTTL(times, cost, min, TTL5m); got != tc.want {
				t.Errorf("decideTTL(%s) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

func TestObserveDecideRecompute(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)
	o := New(Config{Window: 2 * time.Hour, MinSamples: 2})
	o.now = func() time.Time { return base.Add(3 * time.Minute) }

	key := "prefixA"
	// Four observations 18 min apart, all within one hour: 5m would re-write each
	// time but 1h keeps it warm as hits -> 1h should win after recompute.
	for i := 0; i < 4; i++ {
		o.Observe(key, "claude-opus-4-8", base.Add(time.Duration(i)*18*time.Minute))
	}

	// Before recompute, cold default.
	if got := o.Decide(key); got != TTL5m {
		t.Fatalf("pre-recompute Decide = %v, want 5m (cold default)", got)
	}

	o.now = func() time.Time { return base.Add(56 * time.Minute) }
	o.recompute()
	if got := o.Decide(key); got != TTL1h {
		t.Errorf("post-recompute Decide = %v, want 1h", got)
	}

	// After the window passes, the prefix is pruned away and reverts to default.
	o.now = func() time.Time { return base.Add(3 * time.Hour) }
	o.recompute()
	if got := o.Decide(key); got != TTL5m {
		t.Errorf("post-prune Decide = %v, want 5m (cold default)", got)
	}
}

func TestEviction(t *testing.T) {
	o := New(Config{MaxPrefixes: 2})
	now := time.Unix(1_700_000_000, 0)
	o.Observe("a", "claude-opus-4-8", now)
	o.Observe("b", "claude-opus-4-8", now.Add(time.Second))
	o.Observe("c", "claude-opus-4-8", now.Add(2*time.Second)) // evicts "a"

	o.mu.RLock()
	_, hasA := o.stats["a"]
	n := len(o.stats)
	o.mu.RUnlock()
	if hasA {
		t.Errorf("expected oldest prefix 'a' to be evicted")
	}
	if n != 2 {
		t.Errorf("tracked prefixes = %d, want 2 (MaxPrefixes)", n)
	}
}
