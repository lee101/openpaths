package queries

import (
	"context"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func TestUsageInsertBatch(t *testing.T) {
	pool := schemaTestPool(t, "usage", "", "001_initial.sql", "014_app_attribution.sql", "024_usage_ttft.sql")
	ctx := context.Background()
	var user, key string
	if err := pool.QueryRow(ctx, `INSERT INTO users (email, password_hash) VALUES ('a@b.c', 'x') RETURNING id::text`).Scan(&user); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO api_keys (user_id, key_hash, key_prefix) VALUES ($1, 'h', 'p') RETURNING id::text`, user).Scan(&key); err != nil {
		t.Fatal(err)
	}
	q := NewUsageQueries(pool)
	logs := make([]model.UsageLog, 5)
	for i := range logs {
		logs[i] = model.UsageLog{UserID: user, APIKeyID: key, Model: "m", Provider: "p", TokensIn: i, StatusCode: 200, AppCategories: []string{"x"}}
	}
	if err := q.InsertBatch(ctx, logs); err != nil {
		t.Fatal(err)
	}
	var n, sum int
	pool.QueryRow(ctx, `SELECT count(*), sum(tokens_in) FROM usage_logs`).Scan(&n, &sum)
	if n != 5 || sum != 10 {
		t.Fatalf("n=%d sum=%d", n, sum)
	}
	bad := append([]model.UsageLog{}, logs[:2]...)
	bad[1].UserID = "not-a-uuid"
	if err := q.InsertBatch(ctx, bad); err == nil {
		t.Fatal("expected error")
	}
	if err := q.InsertBatch(ctx, nil); err != nil {
		t.Fatal(err)
	}
	pool.QueryRow(ctx, `SELECT count(*) FROM usage_logs`).Scan(&n)
	if n != 5 {
		t.Fatalf("failed batch leaked rows: %d", n)
	}
}
