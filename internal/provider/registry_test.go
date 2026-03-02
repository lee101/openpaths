package provider

import (
	"context"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

type mockProvider struct {
	name string
}

func (m *mockProvider) Name() string { return m.name }
func (m *mockProvider) ChatCompletion(ctx context.Context, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error) {
	return nil, nil
}
func (m *mockProvider) ChatCompletionStream(ctx context.Context, req *model.ChatCompletionRequest) (<-chan StreamEvent, error) {
	return nil, nil
}
func (m *mockProvider) HealthCheck(ctx context.Context) error { return nil }

func TestRegistryRegisterAndGet(t *testing.T) {
	r := NewRegistry()

	p := &mockProvider{name: "test-provider"}
	r.Register(p)

	got, err := r.Get("test-provider")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name() != "test-provider" {
		t.Errorf("got name %q, want %q", got.Name(), "test-provider")
	}
}

func TestRegistryGetNotFound(t *testing.T) {
	r := NewRegistry()

	_, err := r.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestRegistryList(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockProvider{name: "openai"})
	r.Register(&mockProvider{name: "anthropic"})

	names := r.List()
	if len(names) != 2 {
		t.Fatalf("got %d providers, want 2", len(names))
	}

	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}
	if !nameSet["openai"] || !nameSet["anthropic"] {
		t.Errorf("expected openai and anthropic in list, got %v", names)
	}
}

func TestRegistryOverwrite(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockProvider{name: "openai"})
	r.Register(&mockProvider{name: "openai"}) // overwrite

	names := r.List()
	if len(names) != 1 {
		t.Fatalf("got %d providers, want 1 after overwrite", len(names))
	}
}
