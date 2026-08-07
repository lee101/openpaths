package promptcache

import (
	"log"
	"sort"
	"sync"
	"time"
)

// TTL is the cache breakpoint lifetime chosen for an Anthropic request.
type TTL int

const (
	TTLNone TTL = iota // do not cache (cold/one-shot prefix — writing would cost more)
	TTL5m              // 5-minute ephemeral cache (1.25x write)
	TTL1h              // 1-hour ephemeral cache (2x write)
)

func (t TTL) String() string {
	switch t {
	case TTL5m:
		return "5m"
	case TTL1h:
		return "1h"
	default:
		return "none"
	}
}

// Config tunes the optimizer. The zero value is filled with sane defaults by New.
type Config struct {
	Window      time.Duration // how far back observations are kept (default 2h)
	Recompute   time.Duration // how often decisions are recomputed (default 1m)
	MinSamples  int           // observations below this use ColdDefault (default 2)
	ColdDefault TTL           // decision for cold/unknown prefixes (default TTL5m)
	MaxPrefixes int           // cap on tracked prefixes (default 50000)
}

func (c *Config) withDefaults() {
	if c.Window <= 0 {
		c.Window = 2 * time.Hour
	}
	if c.Recompute <= 0 {
		c.Recompute = time.Minute
	}
	if c.MinSamples <= 0 {
		c.MinSamples = 2
	}
	if c.ColdDefault == TTLNone {
		c.ColdDefault = TTL5m
	}
	if c.MaxPrefixes <= 0 {
		c.MaxPrefixes = 50000
	}
}

type stat struct {
	model string
	last  int64   // unix-nano of most recent observation
	times []int64 // sorted unix-nano observations within the window
}

// Optimizer tracks per-prefix request timing and serves the current cheapest
// cache TTL decision for each prefix. It is safe for concurrent use.
type Optimizer struct {
	cfg Config
	now func() time.Time

	mu     sync.RWMutex
	stats  map[string]*stat
	decide map[string]TTL

	// realized telemetry (under mu)
	totalRead   int64   // cumulative cache-read (hit) tokens observed
	totalWrite  int64   // cumulative cache-write tokens observed
	netSavedUSD float64 // cumulative net $ saved vs. no caching

	stop   chan struct{}
	closed bool
}

// New builds an Optimizer with the given config (defaults applied).
func New(cfg Config) *Optimizer {
	cfg.withDefaults()
	return &Optimizer{
		cfg:    cfg,
		now:    time.Now,
		stats:  make(map[string]*stat),
		decide: make(map[string]TTL),
		stop:   make(chan struct{}),
	}
}

// Observe records that a request with the given prefix key (for an Anthropic
// model) arrived now. Cheap; safe to call on the hot path.
func (o *Optimizer) Observe(prefixKey, model string, t time.Time) {
	if o == nil {
		return
	}
	ns := t.UnixNano()
	o.mu.Lock()
	defer o.mu.Unlock()
	s := o.stats[prefixKey]
	if s == nil {
		if len(o.stats) >= o.cfg.MaxPrefixes {
			o.evictOldestLocked()
		}
		s = &stat{model: model}
		o.stats[prefixKey] = s
		o.decide[prefixKey] = o.cfg.ColdDefault
	}
	s.model = model
	s.last = ns
	s.times = append(s.times, ns)
}

// Decide returns the currently recommended TTL for a prefix. Unknown prefixes
// (not yet observed, or never recomputed) get ColdDefault.
func (o *Optimizer) Decide(prefixKey string) TTL {
	if o == nil {
		return TTLNone
	}
	o.mu.RLock()
	defer o.mu.RUnlock()
	if d, ok := o.decide[prefixKey]; ok {
		return d
	}
	return o.cfg.ColdDefault
}

// RecordResult feeds realized cache usage from a response back in so the
// optimizer can report actual savings. readTokens were billed upstream at the
// cache-hit rate (vs. base) and writeTokens at the write premium (vs. base);
// the net is the dollars saved versus not caching at all. Safe to call from the
// hot path; no-op when nil.
func (o *Optimizer) RecordResult(modelName string, readTokens, writeTokens int) {
	if o == nil || (readTokens == 0 && writeTokens == 0) {
		return
	}
	c := CostFor(modelName)
	readSaved := float64(readTokens) * (c.BaseInputPer1M - c.CacheHitPer1M) / 1_000_000
	// Writes cost a premium over base; attribute the 5m premium as a conservative
	// estimate (we cannot tell 5m vs 1h from the response).
	writeExtra := float64(writeTokens) * (c.Write5mPer1M - c.BaseInputPer1M) / 1_000_000

	o.mu.Lock()
	o.totalRead += int64(readTokens)
	o.totalWrite += int64(writeTokens)
	o.netSavedUSD += readSaved - writeExtra
	o.mu.Unlock()
}

// Start launches the periodic recompute loop.
func (o *Optimizer) Start() {
	if o == nil {
		return
	}
	go o.loop()
}

// Stop halts the recompute loop.
func (o *Optimizer) Stop() {
	if o == nil {
		return
	}
	o.mu.Lock()
	if !o.closed {
		o.closed = true
		close(o.stop)
	}
	o.mu.Unlock()
}

func (o *Optimizer) loop() {
	ticker := time.NewTicker(o.cfg.Recompute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			o.recompute()
		case <-o.stop:
			return
		}
	}
}

// recompute prunes the window and rebuilds the per-prefix decision map.
func (o *Optimizer) recompute() {
	cutoff := o.now().Add(-o.cfg.Window).UnixNano()
	var n5, n1h, nNone, nCold int

	o.mu.Lock()
	for key, s := range o.stats {
		s.times = pruneSorted(s.times, cutoff)
		if len(s.times) == 0 {
			delete(o.stats, key)
			delete(o.decide, key)
			continue
		}
		d := decideTTL(s.times, CostFor(s.model), o.cfg.MinSamples, o.cfg.ColdDefault)
		o.decide[key] = d
		switch d {
		case TTL5m:
			n5++
		case TTL1h:
			n1h++
		case TTLNone:
			nNone++
		}
		if len(s.times) < o.cfg.MinSamples {
			nCold++
		}
	}
	tracked := len(o.stats)
	read, write, saved := o.totalRead, o.totalWrite, o.netSavedUSD
	o.mu.Unlock()

	if tracked > 0 {
		log.Printf("promptcache: %d prefixes tracked (5m=%d 1h=%d none=%d cold=%d); realized cache read=%d write=%d net_saved=$%.4f",
			tracked, n5, n1h, nNone, nCold, read, write, saved)
	}
}

// evictOldestLocked drops the least-recently-used prefix. Caller holds o.mu.
func (o *Optimizer) evictOldestLocked() {
	var oldestKey string
	var oldest int64
	for k, s := range o.stats {
		if oldestKey == "" || s.last < oldest {
			oldestKey, oldest = k, s.last
		}
	}
	if oldestKey != "" {
		delete(o.stats, oldestKey)
		delete(o.decide, oldestKey)
	}
}

// pruneSorted drops entries older than cutoff from an ascending slice.
func pruneSorted(times []int64, cutoff int64) []int64 {
	i := sort.Search(len(times), func(i int) bool { return times[i] >= cutoff })
	if i == 0 {
		return times
	}
	return append(times[:0], times[i:]...)
}

// decideTTL picks the cheapest of {none, 5m, 1h} for the given observation
// timestamps. A cache hit refreshes the TTL, so a write is only paid when an
// inter-arrival gap exceeds the TTL. Base price and token count cancel in the
// argmin, but real prices keep the comparison valid if multipliers ever differ.
func decideTTL(times []int64, cost ModelCost, minSamples int, cold TTL) TTL {
	n := len(times)
	if n < minSamples {
		return cold
	}
	const fiveMin = int64(5 * time.Minute)
	const oneHour = int64(time.Hour)

	gaps5, gaps1h := 0, 0
	for i := 1; i < n; i++ {
		g := times[i] - times[i-1]
		if g > fiveMin {
			gaps5++
		}
		if g > oneHour {
			gaps1h++
		}
	}
	writes5 := 1 + gaps5
	writes1h := 1 + gaps1h
	hits5 := n - writes5
	hits1h := n - writes1h

	costNone := float64(n) * cost.BaseInputPer1M
	cost5 := float64(writes5)*cost.Write5mPer1M + float64(hits5)*cost.CacheHitPer1M
	cost1h := float64(writes1h)*cost.Write1hPer1M + float64(hits1h)*cost.CacheHitPer1M

	best, bestCost := TTLNone, costNone
	if cost5 < bestCost {
		best, bestCost = TTL5m, cost5
	}
	if cost1h < bestCost {
		best = TTL1h
	}
	return best
}
