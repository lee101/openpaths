package billing

import (
	"context"
	"testing"
	"time"
)

func TestReconciler_ThrottleSkipsWithinMinGap(t *testing.T) {
	r := &Reconciler{
		minGap:  10 * time.Second,
		lastRun: map[string]time.Time{},
	}

	if !r.shouldRun("u1") {
		t.Fatal("first call should run")
	}
	r.markRan("u1")

	if r.shouldRun("u1") {
		t.Fatal("second call within minGap should be throttled")
	}
	if !r.shouldRun("u2") {
		t.Fatal("throttle is per-user; u2 should be allowed")
	}
}

func TestReconciler_NilSafeReturnsZero(t *testing.T) {
	var r *Reconciler
	n, err := r.ReconcileUser(context.Background(), "u1")
	if err != nil || n != 0 {
		t.Fatalf("nil reconciler: want (0,nil), got (%d,%v)", n, err)
	}

	r2 := &Reconciler{lastRun: map[string]time.Time{}}
	n, err = r2.ReconcileUser(context.Background(), "u1")
	if err != nil || n != 0 {
		t.Fatalf("reconciler with nil stripe: want (0,nil), got (%d,%v)", n, err)
	}
}
