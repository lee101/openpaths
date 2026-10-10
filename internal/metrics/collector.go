package metrics

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/model"
)

type aggKey struct {
	user, key, model, provider, app string
}

type aggRow struct {
	log   model.UsageLog
	latMs int64
}

type Collector struct {
	aggMu    sync.Mutex
	agg      map[aggKey]*aggRow
	ch       chan model.UsageLog
	usageQ   *queries.UsageQueries
	interval time.Duration
	done     chan struct{}
}

func NewCollector(usageQ *queries.UsageQueries, flushInterval time.Duration) *Collector {
	if flushInterval <= 0 {
		flushInterval = 5 * time.Second
	}
	return &Collector{
		agg:      make(map[aggKey]*aggRow),
		ch:       make(chan model.UsageLog, 65536),
		usageQ:   usageQ,
		interval: flushInterval,
		done:     make(chan struct{}),
	}
}

func (c *Collector) Start() {
	go func() {
		buf := make([]model.UsageLog, 0, 1024)
		ticker := time.NewTicker(c.interval)
		defer ticker.Stop()

		for {
			select {
			case entry := <-c.ch:
				buf = append(buf, entry)
				if len(buf) >= 1024 {
					c.flush(buf)
					buf = buf[:0]
				}
			case <-ticker.C:
				// drain remaining
				for {
					select {
					case entry := <-c.ch:
						buf = append(buf, entry)
					default:
						goto done
					}
				}
			done:
				buf = c.drainAgg(buf)
				if len(buf) > 0 {
					c.flush(buf)
					buf = buf[:0]
				}
			case <-c.done:
				close(c.ch)
				for entry := range c.ch {
					buf = append(buf, entry)
				}
				buf = c.drainAgg(buf)
				if len(buf) > 0 {
					c.flush(buf)
				}
				return
			}
		}
	}()
}

func (c *Collector) Stop() {
	close(c.done)
}

func (c *Collector) Record(logEntry model.UsageLog) {
	if logEntry.Coalesce {
		c.coalesce(logEntry)
		return
	}
	select {
	case c.ch <- logEntry:
	default:
		log.Printf("metrics: buffer full, dropping log")
	}
}

func (c *Collector) flush(batch []model.UsageLog) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if len(batch) > 1 && c.usageQ.InsertBatch(ctx, batch) == nil {
		return
	}
	for i := range batch {
		if err := c.usageQ.Insert(ctx, &batch[i]); err != nil {
			log.Printf("metrics flush: %v", err)
		}
	}
}

// Flush drains and writes all pending logs (for testing).
func (c *Collector) Flush() {
	buf := make([]model.UsageLog, 0, 64)
	for {
		select {
		case entry := <-c.ch:
			buf = append(buf, entry)
		default:
			if len(buf) > 0 {
				c.flush(buf)
			}
			return
		}
	}
}

func (c *Collector) coalesce(e model.UsageLog) {
	k := aggKey{e.UserID, e.APIKeyID, e.Model, e.Provider, e.AppID}
	n := e.RequestCount
	if n < 1 {
		n = 1
	}
	c.aggMu.Lock()
	row := c.agg[k]
	if row == nil {
		e.RequestCount = 0
		row = &aggRow{log: e}
		row.log.TokensIn, row.log.TokensOut, row.log.CostCents, row.log.LatencyMs = 0, 0, 0, 0
		c.agg[k] = row
	}
	row.log.RequestCount += n
	row.log.TokensIn += e.TokensIn
	row.log.TokensOut += e.TokensOut
	row.log.CostCents += e.CostCents
	row.latMs += int64(e.LatencyMs) * int64(n)
	c.aggMu.Unlock()
}

func (c *Collector) drainAgg(dst []model.UsageLog) []model.UsageLog {
	c.aggMu.Lock()
	rows := c.agg
	c.agg = make(map[aggKey]*aggRow, len(rows))
	c.aggMu.Unlock()
	for _, r := range rows {
		r.log.LatencyMs = int(r.latMs / int64(r.log.RequestCount))
		dst = append(dst, r.log)
	}
	return dst
}
