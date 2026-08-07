package handler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openpaths/openpaths/internal/agent"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/storage"
)

func TestMaterializeDocumentImagesStoresLinks(t *testing.T) {
	uploadDir := filepath.Join(t.TempDir(), "uploads")
	store, err := storage.NewLocalStore(uploadDir, "https://assets.test")
	if err != nil {
		t.Fatal(err)
	}
	h := &AgentsHandler{store: store}
	converted := agent.ConvertedDocument{
		Markdown: "Before\n\n{{IMAGE}}\n\nAfter",
		Images: []agent.DocumentImage{{
			Placeholder: "{{IMAGE}}",
			Alt:         "Quarterly [chart]",
			WebP:        []byte("webp bytes"),
		}},
	}

	markdown, kept := h.materializeDocumentImages(context.Background(), converted)
	if kept != 1 || !strings.Contains(markdown, "![Quarterly chart](https://assets.test/uploads/") || strings.Contains(markdown, "{{IMAGE}}") {
		t.Fatalf("materialized markdown = %q, kept = %d", markdown, kept)
	}
	files, err := os.ReadDir(uploadDir)
	if err != nil || len(files) != 1 || filepath.Ext(files[0].Name()) != ".webp" {
		t.Fatalf("stored files = %v, err = %v", files, err)
	}
}

func TestPublicAgentSourcesRedactsDatabaseDSN(t *testing.T) {
	source := model.AgentDataSource{Kind: "database", Meta: map[string]any{
		"dsn": "postgres://readonly:secret@example.test/db", "driver": "pgx",
	}}
	public := publicAgentSource(source)
	if _, exists := public.Meta["dsn"]; exists {
		t.Fatal("public source exposed database DSN")
	}
	if public.Meta["connected"] != true || public.Meta["driver"] != "pgx" {
		t.Fatalf("public metadata = %#v", public.Meta)
	}
	if source.Meta["dsn"] == nil {
		t.Fatal("redaction mutated persisted source metadata")
	}
}
