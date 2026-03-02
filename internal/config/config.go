package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/openpaths/openpaths/internal/model"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server    ServerConfig           `yaml:"server"`
	Database  DatabaseConfig         `yaml:"database"`
	JWT       JWTConfig              `yaml:"jwt"`
	Crypto    CryptoConfig           `yaml:"crypto"`
	Storage   StorageConfig          `yaml:"storage"`
	Providers []model.ProviderConfig `yaml:"providers"`
	Models    []model.ModelConfig    `yaml:"models"`
}

type StorageConfig struct {
	Provider  string `yaml:"provider"`
	LocalDir  string `yaml:"local_dir"`
	R2Endpoint string `yaml:"r2_endpoint"`
	R2Bucket   string `yaml:"r2_bucket"`
	R2AccessKey string `yaml:"r2_access_key"`
	R2SecretKey string `yaml:"r2_secret_key"`
	R2PublicURL string `yaml:"r2_public_url"`
}

type CryptoConfig struct {
	Enabled        bool   `yaml:"enabled"`
	WalletPubkey   string `yaml:"solana_wallet_pubkey"`
	HDWalletSeed   string `yaml:"hd_wallet_seed"`
	SolanaRPCURL   string `yaml:"solana_rpc_url"`
	HeliusAPIKey   string `yaml:"helius_api_key"`
	CodexTokenMint string `yaml:"codex_token_mint"`
	BagsAPIKey     string `yaml:"bags_api_key"`
	MinTopupUSD    float64 `yaml:"min_topup_usd"`
}

type ServerConfig struct {
	Port            int    `yaml:"port"`
	ReadTimeout     int    `yaml:"read_timeout_seconds"`
	WriteTimeout    int    `yaml:"write_timeout_seconds"`
	MaxRequestBody  int    `yaml:"max_request_body_mb"`
	StaticDir       string `yaml:"static_dir"`
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
		// Also check GEMINI_API_KEY for google provider
		if cfg.Providers[i].Name == "google" {
			if v := os.Getenv("GEMINI_API_KEY"); v != "" && cfg.Providers[i].APIKey == "" {
				cfg.Providers[i].APIKey = v
			}
		}
	}

	if v := os.Getenv("PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = p
		}
	}
	if v := os.Getenv("DATABASE_URL"); v != "" {
		cfg.Database.URL = v
	}
	if v := os.Getenv("JWT_SECRET"); v != "" {
		cfg.JWT.Secret = v
	}

	if v := os.Getenv("SOLANA_WALLET_PUBKEY"); v != "" {
		cfg.Crypto.WalletPubkey = v
	}
	if v := os.Getenv("HD_WALLET_SEED"); v != "" {
		cfg.Crypto.HDWalletSeed = v
	}
	if v := os.Getenv("SOLANA_RPC_URL"); v != "" {
		cfg.Crypto.SolanaRPCURL = v
	}
	if v := os.Getenv("HELIUS_API_KEY"); v != "" {
		cfg.Crypto.HeliusAPIKey = v
	}
	if v := os.Getenv("CODEX_TOKEN_MINT"); v != "" {
		cfg.Crypto.CodexTokenMint = v
	}
	if v := os.Getenv("BAGS_API_KEY"); v != "" {
		cfg.Crypto.BagsAPIKey = v
	}

	if v := os.Getenv("STORAGE_PROVIDER"); v != "" {
		cfg.Storage.Provider = v
	}
	if v := os.Getenv("R2_ENDPOINT"); v != "" {
		cfg.Storage.R2Endpoint = v
	}
	if v := os.Getenv("R2_BUCKET"); v != "" {
		cfg.Storage.R2Bucket = v
	}
	if v := os.Getenv("R2_ACCESS_KEY"); v != "" {
		cfg.Storage.R2AccessKey = v
	}
	if v := os.Getenv("R2_SECRET_KEY"); v != "" {
		cfg.Storage.R2SecretKey = v
	}
	if v := os.Getenv("R2_PUBLIC_URL"); v != "" {
		cfg.Storage.R2PublicURL = v
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
	if c.Crypto.SolanaRPCURL == "" {
		c.Crypto.SolanaRPCURL = "https://api.mainnet-beta.solana.com"
	}
	if c.Crypto.CodexTokenMint == "" {
		c.Crypto.CodexTokenMint = "HAK9cX1jfYmcNpr6keTkLvxehGPWKELXSu7GH2ofBAGS"
	}
	if c.Crypto.MinTopupUSD == 0 {
		c.Crypto.MinTopupUSD = 5
	}
	if c.Storage.LocalDir == "" {
		c.Storage.LocalDir = "./uploads"
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
