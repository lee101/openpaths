package queries

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openpaths/openpaths/internal/auth"
	"github.com/openpaths/openpaths/internal/model"
)

func apiKeyTestSetup(t *testing.T) (*APIKeyQueries, *pgxpool.Pool, string) {
	t.Helper()
	pool := schemaTestPool(t, "apikeys", "", "001_initial.sql", "028_guardrails.sql", "037_api_key_scopes.sql")
	var user string
	if err := pool.QueryRow(context.Background(), `INSERT INTO users (email, password_hash) VALUES ('a@b.c', 'x') RETURNING id::text`).Scan(&user); err != nil {
		t.Fatal(err)
	}
	return NewAPIKeyQueries(pool), pool, user
}

func newSecret(t *testing.T) (raw, hash, prefix string) {
	t.Helper()
	raw, hash, prefix, err := auth.GenerateAPIKey()
	if err != nil {
		t.Fatal(err)
	}
	return
}

func TestAPIKeyStoresOnlyHash(t *testing.T) {
	q, pool, user := apiKeyTestSetup(t)
	ctx := context.Background()
	raw, hash, prefix := newSecret(t)
	k, err := q.CreateScoped(ctx, user, hash, prefix, "ci", []string{model.ScopeImages}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var stored, pre string
	pool.QueryRow(ctx, `SELECT key_hash, key_prefix FROM api_keys WHERE id = $1`, k.ID).Scan(&stored, &pre)
	if stored != auth.HashAPIKey(raw) || stored == raw || pre == raw || len(pre) >= len(raw) {
		t.Fatalf("secret stored: hash=%q prefix=%q", stored, pre)
	}
	got, err := q.ValidateKey(ctx, auth.HashAPIKey(raw))
	if err != nil || got.ID != k.ID || len(got.Scopes) != 1 || got.Scopes[0] != model.ScopeImages {
		t.Fatalf("validate: %+v %v", got, err)
	}
	if _, err := q.ValidateKey(ctx, auth.HashAPIKey(raw+"x")); !errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("wrong secret: %v", err)
	}
}

func TestAPIKeyLegacyRowKeepsUsageScopes(t *testing.T) {
	q, pool, user := apiKeyTestSetup(t)
	ctx := context.Background()
	raw, hash, _ := newSecret(t)
	if _, err := pool.Exec(ctx, `INSERT INTO api_keys (user_id, key_hash, key_prefix) VALUES ($1, $2, 'sk-op-legacy')`, user, hash); err != nil {
		t.Fatal(err)
	}
	k, err := q.ValidateKey(ctx, auth.HashAPIKey(raw))
	if err != nil {
		t.Fatal(err)
	}
	if k.Scopes != nil || !k.HasScope(model.ScopeChat) || !k.HasScope(model.ScopeAccountManage) || k.IsManagement() {
		t.Fatalf("legacy scopes: %+v", k.EffectiveScopes())
	}
}

func TestAPIKeyExpiryAndRevocation(t *testing.T) {
	q, _, user := apiKeyTestSetup(t)
	ctx := context.Background()

	rawExp, hashExp, preExp := newSecret(t)
	past := time.Now().Add(-time.Minute)
	if _, err := q.CreateScoped(ctx, user, hashExp, preExp, "old", nil, &past); err != nil {
		t.Fatal(err)
	}
	if _, err := q.ValidateKey(ctx, auth.HashAPIKey(rawExp)); !errors.Is(err, ErrKeyExpired) {
		t.Fatalf("expired: %v", err)
	}

	raw, hash, prefix := newSecret(t)
	k, _ := q.CreateScoped(ctx, user, hash, prefix, "live", nil, nil)
	if _, err := q.ValidateKey(ctx, auth.HashAPIKey(raw)); err != nil {
		t.Fatal(err)
	}
	if err := q.Revoke(ctx, k.ID, user); err != nil {
		t.Fatal(err)
	}
	if _, err := q.ValidateKey(ctx, auth.HashAPIKey(raw)); !errors.Is(err, ErrKeyRevoked) {
		t.Fatalf("revoked: %v", err)
	}
	if err := q.Revoke(ctx, k.ID, user); !errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("double revoke: %v", err)
	}
}

func TestAPIKeyRotateImmediateAndGrace(t *testing.T) {
	q, pool, user := apiKeyTestSetup(t)
	ctx := context.Background()
	raw, hash, prefix := newSecret(t)
	exp := time.Now().Add(48 * time.Hour)
	old, _ := q.CreateScoped(ctx, user, hash, prefix, "svc", []string{model.ScopeChat}, &exp)
	pool.Exec(ctx, `UPDATE api_keys SET rate_limit_rpm = 900 WHERE id = $1`, old.ID)
	var guard string
	pool.QueryRow(ctx, `INSERT INTO guardrails (user_id, name) VALUES ($1, 'g') RETURNING id::text`, user).Scan(&guard)
	pool.Exec(ctx, `INSERT INTO guardrail_assignments VALUES ($1, 'api_key', $2)`, guard, old.ID)

	raw2, hash2, prefix2 := newSecret(t)
	nk, ok, err := q.Rotate(ctx, old.ID, user, hash2, prefix2, 0)
	if err != nil {
		t.Fatal(err)
	}
	if nk.RotatedFrom == nil || *nk.RotatedFrom != old.ID || nk.Name != "svc" || nk.RateLimitRPM != 900 || nk.ExpiresAt == nil || len(nk.Scopes) != 1 || ok.RevokedAt == nil {
		t.Fatalf("rotated: %+v old: %+v", nk, ok)
	}
	if _, err := q.ValidateKey(ctx, auth.HashAPIKey(raw)); !errors.Is(err, ErrKeyRevoked) {
		t.Fatalf("old key after immediate rotate: %v", err)
	}
	if _, err := q.ValidateKey(ctx, auth.HashAPIKey(raw2)); err != nil {
		t.Fatal(err)
	}
	var n int
	pool.QueryRow(ctx, `SELECT count(*) FROM guardrail_assignments WHERE target_type = 'api_key' AND target_id = $1`, nk.ID).Scan(&n)
	if n != 1 {
		t.Fatalf("guardrail assignment not carried over: %d", n)
	}
	if _, _, err := q.Rotate(ctx, old.ID, user, "h", "p", 0); !errors.Is(err, ErrKeyRevoked) {
		t.Fatalf("rotate revoked: %v", err)
	}

	raw3, hash3, prefix3 := newSecret(t)
	nk2, ok2, err := q.Rotate(ctx, nk.ID, user, hash3, prefix3, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if ok2.Revoked || ok2.RevokedAt == nil || !ok2.RevokedAt.After(time.Now().Add(59*time.Minute)) {
		t.Fatalf("grace window: %+v", ok2)
	}
	if _, err := q.ValidateKey(ctx, auth.HashAPIKey(raw2)); err != nil {
		t.Fatalf("old key inside grace: %v", err)
	}
	if _, err := q.ValidateKey(ctx, auth.HashAPIKey(raw3)); err != nil {
		t.Fatal(err)
	}
	pool.Exec(ctx, `UPDATE api_keys SET revoked_at = now() - interval '1 second' WHERE id = $1`, nk.ID)
	q.forgetKeys()
	if _, err := q.ValidateKey(ctx, auth.HashAPIKey(raw2)); !errors.Is(err, ErrKeyRevoked) {
		t.Fatalf("old key after grace: %v", err)
	}
	keys, err := q.ListByUser(ctx, user)
	if err != nil || len(keys) != 1 || keys[0].ID != nk2.ID {
		t.Fatalf("list after rotations: %+v %v", keys, err)
	}
	if _, _, err := q.Rotate(ctx, "00000000-0000-0000-0000-000000000000", user, "h", "p", 0); !errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("rotate unknown: %v", err)
	}
}
