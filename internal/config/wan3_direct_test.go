package config

import (
	"testing"

	"github.com/openpaths/openpaths/internal/billing"
)

func TestWan3DirectConfigAndPricing(t *testing.T) {
	cfg, err := Load("../../config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, p := range cfg.Providers {
		found = found || (p.Name == "alibaba" && p.Enabled)
	}
	if !found {
		t.Fatal("alibaba provider missing")
	}
	pt := billing.NewPricingTable(cfg.Models)
	for _, tc := range []struct {
		resolution string
		seconds    float64
		video      bool
		want       int64
	}{{"480P", 5, false, 2500}, {"720P", 10, false, 10000}, {"1080P", 5, false, 10000}, {"", 5, false, 10000}, {"720P", 15, true, 15000}} {
		got, err := pt.CalculateVideoCostForDuration("wan3.0-video", tc.seconds, tc.video, 3, tc.resolution)
		if err != nil || got != tc.want {
			t.Fatalf("%+v: got %d err %v", tc, got, err)
		}
	}
}
