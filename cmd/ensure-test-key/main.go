// Command ensure-test-key repairs the private TEST_API_KEY used by live smoke tests.
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/openpaths/openpaths/internal/auth"
	"github.com/openpaths/openpaths/internal/config"
	"github.com/openpaths/openpaths/internal/db/queries"
)

const (
	testEmail = "smoke-test@openpaths.local"
	testName  = "production-smoke-test"
)

func main() {
	_ = godotenv.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	url := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if url == "" {
		cfg, cfgErr := config.Load("config.yaml")
		if cfgErr != nil {
			fatal("load database configuration: %v", cfgErr)
		}
		url = cfg.Database.URL
	}
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		fatal("connect: %v", err)
	}
	defer pool.Close()
	keyQ := queries.NewAPIKeyQueries(pool)
	current := strings.TrimSpace(os.Getenv("TEST_API_KEY"))
	if strings.HasPrefix(current, auth.APIKeyPrefix) {
		if _, err := keyQ.ValidateKey(ctx, auth.HashAPIKey(current)); err == nil {
			fmt.Println("TEST_API_KEY is already valid")
			return
		}
	}

	userQ := queries.NewUserQueries(pool)
	user, err := userQ.GetByEmail(ctx, testEmail)
	if err != nil {
		hash, hashErr := auth.HashPassword(randomSecret())
		if hashErr != nil {
			fatal("hash service password: %v", hashErr)
		}
		user, err = userQ.Create(ctx, testEmail, hash, "Production Smoke Test")
		if err != nil {
			fatal("create service user: %v", err)
		}
	}
	creditQ := queries.NewCreditQueries(pool)
	if err := creditQ.InitBalance(ctx, user.ID); err != nil {
		fatal("initialize service credit: %v", err)
	}
	if balance, err := creditQ.GetBalance(ctx, user.ID); err == nil && balance < 500 {
		if err := creditQ.Deposit(ctx, user.ID, 2000, "production smoke-test credit grant"); err != nil {
			fatal("grant service credit: %v", err)
		}
	}
	if keys, err := keyQ.ListByUser(ctx, user.ID); err == nil {
		for _, key := range keys {
			if key.Name == testName && !key.Revoked {
				if err := keyQ.Revoke(ctx, key.ID, user.ID); err != nil {
					fatal("revoke stale test key: %v", err)
				}
			}
		}
	}
	raw, hash, prefix, err := auth.GenerateAPIKey()
	if err != nil {
		fatal("generate key: %v", err)
	}
	if _, err := keyQ.Create(ctx, user.ID, hash, prefix, testName); err != nil {
		fatal("store key: %v", err)
	}
	if err := replaceEnv(".env", "TEST_API_KEY", raw); err != nil {
		fatal("update .env: %v", err)
	}
	fmt.Println("repaired TEST_API_KEY in .env (value not printed)")
}

func randomSecret() string {
	raw, _, _, err := auth.GenerateAPIKey()
	if err != nil {
		fatal("generate service password: %v", err)
	}
	return raw
}

func replaceEnv(path, key, value string) error {
	input, err := os.Open(path)
	if err != nil {
		return err
	}
	defer input.Close()
	lines := make([]string, 0, 128)
	found := false
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, key+"=") {
			line = key + "=" + value
			found = true
		}
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if !found {
		lines = append(lines, key+"="+value)
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(strings.Join(lines, "\n")+"\n"), info.Mode().Perm()); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "ensure-test-key: "+format+"\n", args...)
	os.Exit(1)
}
