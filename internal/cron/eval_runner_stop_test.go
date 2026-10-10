package cron

import (
	"testing"
	"time"
)

func TestEvalRunnerStopCancelsContext(t *testing.T) {
	r := &EvalRunner{stop: make(chan struct{})}
	ctx, cancel := r.stopContext()
	defer cancel()
	r.Stop()
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("context not cancelled by Stop")
	}
}

func TestEvalRunnerSweepOnceSkipsWhenRunning(t *testing.T) {
	r := &EvalRunner{stop: make(chan struct{})}
	r.running.Store(true)
	ctx, cancel := r.stopContext()
	defer cancel()
	if r.sweepOnce(ctx) {
		t.Fatal("sweep started while another was in flight")
	}
	if !r.running.Load() {
		t.Fatal("running flag cleared by skipped sweep")
	}
}
