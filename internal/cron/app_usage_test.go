package cron

import "testing"

func TestParseTokenCount(t *testing.T) {
	tests := []struct {
		raw  string
		want int64
	}{
		{raw: "8.52T tokens", want: 8_520_000_000_000},
		{raw: "26.9B tokens", want: 26_900_000_000},
		{raw: "12.7M", want: 12_700_000},
		{raw: "430K", want: 430_000},
		{raw: "bad", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			if got := parseTokenCount(tt.raw); got != tt.want {
				t.Fatalf("parseTokenCount(%q) = %d, want %d", tt.raw, got, tt.want)
			}
		})
	}
}

func TestParseOpenRouterAppModels(t *testing.T) {
	body := `window.__data = {\"appModelAnalytics\":[{\"date\":\"2026-05-23\",\"model_permaslug\":\"openai/gpt-oss-120b\",\"total_tokens\":2500},{\"date\":\"bad\",\"model_permaslug\":\"ignored\",\"total_tokens\":1}],\"appName\":\"Hermes\"}`

	rows := parseOpenRouterAppModels(body)
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
	if rows[0].Model != "openai/gpt-oss-120b" {
		t.Fatalf("model = %q, want openai/gpt-oss-120b", rows[0].Model)
	}
	if rows[0].TotalTokens != 2500 {
		t.Fatalf("total tokens = %d, want 2500", rows[0].TotalTokens)
	}
}

func TestParseOpenRouterApps(t *testing.T) {
	body := `<a class="app-card" href="/apps/hermes-agent">
  <img alt="Favicon for https://nousresearch.com" src="https://example.com/icon.png">
  <span class="font-medium anything">Hermes Agent</span>
  <p class="truncate">Persistent open-source agent.</p>
  <span class="text-sm font-medium whitespace-nowrap">8.52T tokens</span>
</a>`

	apps := parseOpenRouterApps(body)
	if len(apps) != 1 {
		t.Fatalf("len(apps) = %d, want 1", len(apps))
	}
	app := apps[0]
	if app.Slug != "hermes-agent" || app.Name != "Hermes Agent" || app.URL != "https://nousresearch.com" {
		t.Fatalf("parsed app = %#v", app)
	}
	if app.TotalTokens != 8_520_000_000_000 {
		t.Fatalf("total tokens = %d, want 8520000000000", app.TotalTokens)
	}
}
