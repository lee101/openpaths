package handler

import (
	"testing"
	"time"
)

func TestVideoJobBudget(t *testing.T) {
	for id, want := range map[string]time.Duration{"wan3.0-video": 30 * time.Minute, "alibaba/wan3.0-video": 30 * time.Minute, "wan3.0": 30 * time.Minute, "wan3.0-video-prime": 30 * time.Minute, "wan3.0-prime": 30 * time.Minute, "video-dramatize": 60 * time.Minute, "wan-3.0-text-to-video": 15 * time.Minute} {
		if got := videoJobBudget(id); got != want {
			t.Fatalf("%s: %s want %s", id, got, want)
		}
	}
}
