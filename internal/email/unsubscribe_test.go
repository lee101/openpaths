package email

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func TestUnsubscribeToken(t *testing.T) {
	tok := UnsubscribeToken("secret", "User@Example.com")
	if len(tok) != 32 {
		t.Fatalf("token length = %d", len(tok))
	}
	if tok != UnsubscribeToken("secret", "  user@example.com ") {
		t.Fatal("token not normalized across case/whitespace")
	}
	if !VerifyUnsubscribeToken("secret", "user@example.com", tok) {
		t.Fatal("valid token rejected")
	}
	if VerifyUnsubscribeToken("secret", "other@example.com", tok) {
		t.Fatal("token accepted for wrong email")
	}
	if VerifyUnsubscribeToken("wrong", "user@example.com", tok) {
		t.Fatal("token accepted with wrong secret")
	}
}

func TestRenderModelCards(t *testing.T) {
	dir := t.TempDir()
	notes := `{"sections":{"reasoning":[
		{"id":"sol","name":"GPT-5.6 Sol","note":"smart","settings":"effort high"},
		{"id":"gone-model","name":"Gone","note":"removed from catalog"}
	]}}`
	path := filepath.Join(dir, "model_notes.json")
	if err := os.WriteFile(path, []byte(notes), 0o644); err != nil {
		t.Fatal(err)
	}
	models := []model.ModelConfig{{
		ID: "gpt-5.6-sol", Aliases: []string{"sol"},
		InputPricePer1M: 5, OutputPricePer1M: 30, ContextWindow: 1050000,
	}}
	cards, err := renderModelCards(path, models)
	if err != nil {
		t.Fatal(err)
	}
	html := cards["reasoning"]
	for _, want := range []string{"GPT-5.6 Sol", "$5/M in", "$30/M out", "1.1M ctx", "effort high"} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %q in cards html", want)
		}
	}
	if strings.Contains(html, "Gone") {
		t.Error("card rendered for model absent from catalog")
	}
}

func TestReasoningTemplateHasCardPlaceholder(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "emails", "14-reasoning-models.html"))
	if err != nil {
		t.Skip("emails dir not available")
	}
	if !strings.Contains(string(data), "{{.ModelCards.reasoning}}") {
		t.Fatal("14-reasoning-models.html missing {{.ModelCards.reasoning}}")
	}
	if !strings.Contains(string(data), "{{.UnsubscribeURL}}") {
		t.Fatal("14-reasoning-models.html missing {{.UnsubscribeURL}}")
	}
}
