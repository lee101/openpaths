package metrics

import (
	"context"
	"log"
	"time"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/model"
)

type Collector struct {
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
		ch:       make(chan model.UsageLog, 4096),
		usageQ:   usageQ,
		interval: flushInterval,
		done:     make(chan struct{}),
	}
}

func (c *Collector) Start() {
	go func() {
		buf := make([]model.UsageLog, 0, 256)
		ticker := time.NewTicker(c.interval)
		defer ticker.Stop()

		for {
			select {
			case entry := <-c.ch:
				buf = append(buf, entry)
				if len(buf) >= 256 {
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
				if len(buf) > 0 {
					c.flush(buf)
					buf = buf[:0]
				}
			case <-c.done:
				close(c.ch)
				for entry := range c.ch {
					buf = append(buf, entry)
				}
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
	select {
	case c.ch <- logEntry:
	default:
		log.Printf("metrics: buffer full, dropping log")
	}
}

func (c *Collector) flush(batch []model.UsageLog) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

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
