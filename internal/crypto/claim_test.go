package crypto

import (
	"context"
	"testing"
	"time"
)

type claimStore struct {
	CryptoStore
	status   string
	released int
}

func (s *claimStore) ClaimPaid(_ context.Context, _, _ string, _ int64) (bool, error) {
	if s.status != "pending" {
		return false, nil
	}
	s.status = "paid"
	return true, nil
}

func (s *claimStore) ReleaseClaim(context.Context, string) error {
	s.status = "pending"
	s.released++
	return nil
}

func TestPriceFeedUnavailableUntilFetched(t *testing.T) {
	p := NewPriceFeed("", "")
	if p.SOLPrice() != 0 {
		t.Fatal("expected no price before first fetch")
	}
	p.solPrice, p.solAt = 150, time.Now()
	if p.SOLPrice() != 150*rateFactor {
		t.Fatal("fresh price should be served")
	}
	p.solAt = time.Now().Add(-time.Hour)
	if p.SOLPrice() != 0 {
		t.Fatal("stale price must not be served")
	}
}

func TestClaimOnlyOnce(t *testing.T) {
	st := &claimStore{status: "pending"}
	ok1, _ := st.ClaimPaid(context.Background(), "a", "tx", 1)
	ok2, _ := st.ClaimPaid(context.Background(), "a", "tx", 1)
	if !ok1 || ok2 {
		t.Fatalf("claim must succeed exactly once: %v %v", ok1, ok2)
	}
}
