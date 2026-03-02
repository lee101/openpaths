package metrics

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/model"
)

// Collector buffers usage logs in memory and flushes to the database periodically.
type Collector struct {
	mu       sync.Mutex
	buffer   []model.UsageLog
	usageQ   *queries.UsageQueries
	interval time.Duration
	done     chan struct{}
}

func NewCollector(usageQ *queries.UsageQueries, flushInterval time.Duration) *Collector {
	if flushInterval <= 0 {
		flushInterval = 5 * time.Second
	}
	return &Collector{
		usageQ:   usageQ,
		interval: flushInterval,
		done:     make(chan struct{}),
	}
}

// Start begins the periodic flush loop.
func (c *Collector) Start() {
	go func() {
		ticker := time.NewTicker(c.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				c.Flush()
			case <-c.done:
				c.Flush() // final flush
				return
			}
		}
	}()
}

// Stop signals the collector to stop and performs a final flush.
func (c *Collector) Stop() {
	close(c.done)
}

// Record adds a usage log to the buffer.
func (c *Collector) Record(logEntry model.UsageLog) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.buffer = append(c.buffer, logEntry)
}

// Flush writes all buffered logs to the database.
func (c *Collector) Flush() {
	c.mu.Lock()
	if len(c.buffer) == 0 {
		c.mu.Unlock()
		return
	}
	batch := c.buffer
	c.buffer = nil
	c.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for i := range batch {
		if err := c.usageQ.Insert(ctx, &batch[i]); err != nil {
			log.Printf("Failed to insert usage log: %v", err)
		}
	}
}
