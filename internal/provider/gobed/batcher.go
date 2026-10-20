package gobed

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// ErrSaturated is returned when the bounded queue is full; callers should
// shed load (429/503 + Retry-After) or divert to an overflow backend.
var ErrSaturated = errors.New("gobed: embedding queue saturated")

type job struct {
	run  func()
	done *sync.WaitGroup
}

// batcher is a bounded worker pool. Workers block for one job, then drain up
// to maxDrain more already-queued jobs (waiting up to window for stragglers),
// so concurrent small requests are served back to back on one core without a
// goroutine wakeup per request.
type batcher struct {
	q        chan job
	maxDrain int
	window   time.Duration
	queued   atomic.Int64
	once     sync.Once
}

func newBatcher(workers, queue, maxDrain int, window time.Duration) *batcher {
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
	}
	if queue <= 0 {
		queue = workers * 8
	}
	b := &batcher{q: make(chan job, queue), maxDrain: maxDrain, window: window}
	for i := 0; i < workers; i++ {
		go b.worker()
	}
	return b
}

func (b *batcher) worker() {
	var timer *time.Timer
	for j := range b.q {
		j.run()
		j.done.Done()
		for n := 0; n < b.maxDrain; n++ {
			select {
			case j = <-b.q:
				j.run()
				j.done.Done()
				continue
			default:
			}
			if b.window <= 0 {
				break
			}
			if timer == nil {
				timer = time.NewTimer(b.window)
			} else {
				timer.Reset(b.window)
			}
			select {
			case j = <-b.q:
				if !timer.Stop() {
					<-timer.C
				}
				j.run()
				j.done.Done()
				continue
			case <-timer.C:
			}
			break
		}
	}
}

// submit enqueues jobs without blocking. If the queue cannot take all of
// them, jobs already queued still complete (wg tracks only accepted ones) and
// ErrSaturated is returned after waiting for them.
func (b *batcher) runAll(ctx context.Context, jobs []func()) error {
	var wg sync.WaitGroup
	var sat bool
	for _, f := range jobs {
		wg.Add(1)
		select {
		case b.q <- job{run: f, done: &wg}:
		default:
			wg.Done()
			sat = true
		}
		if sat {
			break
		}
	}
	wg.Wait()
	if sat {
		return ErrSaturated
	}
	return ctx.Err()
}
