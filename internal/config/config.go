package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/openpath/openpath/internal/model"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server    ServerConfig         `yaml:"server"`
	Database  DatabaseConfig       `yaml:"database"`
	JWT       JWTConfig            `yaml:"jwt"`
	Providers []model.ProviderConfig `yaml:"providers"`
	Models    []model.ModelConfig  `yaml:"models"`
}

type ServerConfig struct {
	Port            int `yaml:"port"`
	ReadTimeout     int `yaml:"read_timeout_seconds"`
	WriteTimeout    int `yaml:"write_timeout_seconds"`
	MaxRequestBody  int `yaml:"max_request_body_mb"`
}

type DatabaseConfig struct {
	URL             string `yaml:"url"`
	MaxConns        int    `yaml:"max_conns"`
	MinConns        int    `yaml:"min_conns"`
}

type JWTConfig struct {
	Secret          string `yaml:"secret"`
	ExpirationHours int    `yaml:"expiration_hours"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	// Expand environment variables in the YAML
	expanded := os.ExpandEnv(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	cfg.applyDefaults()

	// Override provider API keys from environment
	for i := range cfg.Providers {
		envKey := fmt.Sprintf("%s_API_KEY", strings.ToUpper(cfg.Providers[i].Name))
		if v := os.Getenv(envKey); v != "" {
			cfg.Providers[i].APIKey = v
		}
	}

	if v := os.Getenv("DATABASE_URL"); v != "" {
		cfg.Database.URL = v
	}
	if v := os.Getenv("JWT_SECRET"); v != "" {
		cfg.JWT.Secret = v
	}

	return &cfg, nil
}

func (c *Config) applyDefaults() {
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Server.ReadTimeout == 0 {
		c.Server.ReadTimeout = 30
	}
	if c.Server.WriteTimeout == 0 {
		c.Server.WriteTimeout = 300
	}
	if c.Server.MaxRequestBody == 0 {
		c.Server.MaxRequestBody = 10
	}
	if c.Database.MaxConns == 0 {
		c.Database.MaxConns = 20
	}
	if c.Database.MinConns == 0 {
		c.Database.MinConns = 2
	}
	if c.JWT.ExpirationHours == 0 {
		c.JWT.ExpirationHours = 72
	}
	if c.Database.URL == "" {
		c.Database.URL = "postgres://openpath:openpath@localhost:5432/openpath?sslmode=disable"
	}
}

func (c *Config) Validate() error {
	if c.JWT.Secret == "" {
		return fmt.Errorf("jwt.secret is required (set JWT_SECRET env var)")
	}
	if len(c.Models) == 0 {
		return fmt.Errorf("at least one model must be configured")
	}
	return nil
}
