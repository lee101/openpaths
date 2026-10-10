package email

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSignupCutoff(t *testing.T) {
	if c, err := (DripEmail{ID: 5, DelayDays: 3}).signupCutoff(); c != nil || err != nil {
		t.Fatalf("regular drip must be unfiltered: %v %v", c, err)
	}
	c, err := DripEmail{Kind: KindModelLaunch, LaunchDate: "2026-08-23", MaxAgeDays: 14}.signupCutoff()
	if err != nil || c == nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC); !c.Equal(want) {
		t.Fatalf("cutoff %v want %v", c, want)
	}
	for _, bad := range []DripEmail{
		{Kind: KindModelLaunch},
		{Kind: KindModelLaunch, LaunchDate: "2026-08-23"},
		{Kind: KindModelLaunch, LaunchDate: "nope", MaxAgeDays: 3},
	} {
		if _, err := bad.signupCutoff(); err == nil {
			t.Errorf("accepted %+v", bad)
		}
	}
}

func TestDripConfigModelLaunchEntriesHaveWindow(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "emails", "drip_config.json"))
	if err != nil {
		t.Skip(err)
	}
	var cfg DripConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	launch := map[int]bool{}
	for _, e := range cfg.Emails {
		if _, err := e.signupCutoff(); err != nil {
			t.Errorf("email %d: %v", e.ID, err)
		}
		if e.Kind == KindModelLaunch {
			launch[e.ID] = true
		}
	}
	for _, id := range []int{23, 24, 25, 26} {
		if !launch[id] {
			t.Errorf("email %d not marked model_launch", id)
		}
	}
}
