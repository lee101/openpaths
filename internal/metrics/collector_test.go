package metrics

import (
	"testing"
	"time"

	"github.com/openpaths/openpaths/internal/model"
)

func TestCoalesceMergesRows(t *testing.T) {
	c := NewCollector(nil, time.Second)
	r := NewRecorder(c)
	for i := 0; i < 1000; i++ {
		r.RecordSuccessCoalesced("u", "k", "openpaths-embed", "gobed", 70, 2, 0, "", "", "", nil)
	}
	r.RecordSuccessCoalesced("u2", "k", "openpaths-embed", "gobed", 10, 4, 3, "", "", "", nil)
	rows := c.drainAgg(nil)
	if len(rows) != 2 {
		t.Fatalf("rows %d", len(rows))
	}
	for _, l := range rows {
		if l.UserID == "u" && (l.RequestCount != 1000 || l.TokensIn != 70000 || l.LatencyMs != 2) {
			t.Fatalf("bad row %+v", l)
		}
		if l.UserID == "u2" && (l.RequestCount != 1 || l.CostCents != 3) {
			t.Fatalf("bad row %+v", l)
		}
	}
	if len(c.drainAgg(nil)) != 0 {
		t.Fatal("not drained")
	}
	_ = model.UsageLog{}
}

func BenchmarkCoalesce(b *testing.B) {
	c := NewCollector(nil, time.Second)
	r := NewRecorder(c)
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			r.RecordSuccessCoalesced("u", "k", "openpaths-embed", "gobed", 70, 2, 0, "", "", "", nil)
		}
	})
}
