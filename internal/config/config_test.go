package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndParse(t *testing.T) {
	// Clear env vars that would override config values
	origOpenAI := os.Getenv("OPENAI_API_KEY")
	os.Unsetenv("OPENAI_API_KEY")
	defer func() {
		if origOpenAI != "" {
			os.Setenv("OPENAI_API_KEY", origOpenAI)
		}
	}()

	yaml := `
server:
  port: 9090
  read_timeout_seconds: 15

database:
  url: "postgres://test:test@localhost:5432/test"

jwt:
  secret: "test-secret"

providers:
  - name: openai
    base_url: "https://api.openai.com"
    api_key: "sk-test"
    enabled: true

models:
  - id: "gpt-4o"
    provider: openai
    provider_model_id: "gpt-4o"
    input_price_per_1m: 2.50
    output_price_per_1m: 10.00
    context_window: 128000
    max_output_tokens: 16384
    supports_streaming: true
`

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("got port %d, want 9090", cfg.Server.Port)
	}
	if cfg.Server.ReadTimeout != 15 {
		t.Errorf("got read timeout %d, want 15", cfg.Server.ReadTimeout)
	}
	if cfg.Database.URL != "postgres://test:test@localhost:5432/test" {
		t.Errorf("got db url %q", cfg.Database.URL)
	}
	if len(cfg.Providers) != 1 {
		t.Fatalf("got %d providers, want 1", len(cfg.Providers))
	}
	if cfg.Providers[0].APIKey != "sk-test" {
		t.Errorf("got api key %q", cfg.Providers[0].APIKey)
	}
	if len(cfg.Models) != 1 {
		t.Fatalf("got %d models, want 1", len(cfg.Models))
	}
}

func TestDefaults(t *testing.T) {
	yaml := `
jwt:
  secret: "test"
models:
  - id: "test"
    provider: test
    provider_model_id: "test"
    input_price_per_1m: 1.0
    output_price_per_1m: 1.0
`

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("default port should be 8080, got %d", cfg.Server.Port)
	}
	if cfg.Server.WriteTimeout != 300 {
		t.Errorf("default write timeout should be 300, got %d", cfg.Server.WriteTimeout)
	}
	if cfg.Database.MaxConns != 20 {
		t.Errorf("default max conns should be 20, got %d", cfg.Database.MaxConns)
	}
	if cfg.JWT.ExpirationHours != 72 {
		t.Errorf("default jwt expiration should be 72, got %d", cfg.JWT.ExpirationHours)
	}
	if cfg.Stripe.CreditsPriceID != defaultStripeCreditsPriceID {
		t.Errorf("default stripe credits price id should be %q, got %q", defaultStripeCreditsPriceID, cfg.Stripe.CreditsPriceID)
	}
}

func TestEnvOverride(t *testing.T) {
	yaml := `
jwt:
  secret: "default"
providers:
  - name: openai
    base_url: "https://api.openai.com"
    api_key: ""
    enabled: true
models:
  - id: "test"
    provider: openai
    provider_model_id: "test"
    input_price_per_1m: 1.0
    output_price_per_1m: 1.0
`

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	os.Setenv("OPENAI_API_KEY", "sk-from-env")
	os.Setenv("JWT_SECRET", "secret-from-env")
	os.Setenv("DATABASE_URL", "postgres://env:env@localhost/env")
	defer func() {
		os.Unsetenv("OPENAI_API_KEY")
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("DATABASE_URL")
	}()

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Providers[0].APIKey != "sk-from-env" {
		t.Errorf("got api key %q, want sk-from-env", cfg.Providers[0].APIKey)
	}
	if cfg.JWT.Secret != "secret-from-env" {
		t.Errorf("got jwt secret %q, want secret-from-env", cfg.JWT.Secret)
	}
	if cfg.Database.URL != "postgres://env:env@localhost/env" {
		t.Errorf("got db url %q", cfg.Database.URL)
	}
}

func TestValidateNoSecret(t *testing.T) {
	cfg := &Config{}
	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for missing secret")
	}
}

func TestValidateNoModels(t *testing.T) {
	cfg := &Config{JWT: JWTConfig{Secret: "test"}}
	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for no models")
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte("{{invalid yaml"), 0644)

	_, err := Load(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}
