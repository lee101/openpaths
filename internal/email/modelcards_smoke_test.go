package email

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openpaths/openpaths/internal/config"
)

// Renders the real model_notes.json against the real catalog so stale ids in
// either file fail loudly. Set SMOKE_OUT to also write the rendered email.
func TestRenderRealCatalogCards(t *testing.T) {
	root := filepath.Join("..", "..")
	cfg, err := config.Load(filepath.Join(root, "config.yaml"))
	if err != nil {
		t.Skipf("config.yaml not loadable: %v", err)
	}
	cards, err := renderModelCards(filepath.Join(root, "emails", "model_notes.json"), cfg.Models)
	if err != nil {
		t.Fatal(err)
	}
	reasoning := cards["reasoning"]
	if n := strings.Count(reasoning, "border-radius:8px"); n < 5 {
		t.Fatalf("only %d reasoning cards rendered; model_notes.json ids drifted from catalog", n)
	}

	tmpl, err := os.ReadFile(filepath.Join(root, "emails", "14-reasoning-models.html"))
	if err != nil {
		t.Fatal(err)
	}
	html := strings.ReplaceAll(string(tmpl), "{{.ModelCards.reasoning}}", reasoning)
	html = strings.ReplaceAll(html, "{{.UnsubscribeURL}}", "https://openpaths.io/unsubscribe?e=test%40example.com&t=deadbeef")
	if strings.Contains(html, "{{.") {
		t.Fatalf("unreplaced placeholder remains: %s", html[strings.Index(html, "{{."):][:40])
	}
	if out := os.Getenv("SMOKE_OUT"); out != "" {
		if err := os.WriteFile(out, []byte(html), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
