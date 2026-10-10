package queries

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func schemaTestPool(t *testing.T, prefix, bootstrap string, migrations ...string) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("OPENPATHS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("OPENPATHS_TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("%s_t_%d", prefix, time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	cfg, _ := pgxpool.ParseConfig(dsn)
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE")
		admin.Close()
	})
	if bootstrap != "" {
		if _, err := pool.Exec(ctx, bootstrap); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range migrations {
		mig, err := os.ReadFile(filepath.Join("..", "migrations", f))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, strings.ReplaceAll(string(mig), `CREATE EXTENSION IF NOT EXISTS "pgcrypto";`, "")); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
	}
	return pool
}

func TestGuardrailResolveAndListAssignments(t *testing.T) {
	pool := schemaTestPool(t, "guard", `CREATE TABLE users (id UUID PRIMARY KEY DEFAULT gen_random_uuid())`,
		"028_guardrails.sql", "029_guardrail_blocklists.sql", "030_guardrail_violation_email.sql")
	ctx := context.Background()
	var user, other string
	if err := pool.QueryRow(ctx, `INSERT INTO users DEFAULT VALUES RETURNING id::text`).Scan(&user); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO users DEFAULT VALUES RETURNING id::text`).Scan(&other); err != nil {
		t.Fatal(err)
	}
	var def, keyed, foreign string
	pool.QueryRow(ctx, `INSERT INTO guardrails (user_id, name) VALUES ($1, 'default') RETURNING id::text`, user).Scan(&def)
	pool.QueryRow(ctx, `INSERT INTO guardrails (user_id, name) VALUES ($1, 'keyed') RETURNING id::text`, user).Scan(&keyed)
	pool.QueryRow(ctx, `INSERT INTO guardrails (user_id, name) VALUES ($1, 'foreign') RETURNING id::text`, other).Scan(&foreign)
	pool.Exec(ctx, `INSERT INTO guardrail_assignments VALUES ($1, 'user', $2)`, def, user)
	pool.Exec(ctx, `INSERT INTO guardrail_assignments VALUES ($1, 'api_key', 'key-1')`, keyed)
	pool.Exec(ctx, `INSERT INTO guardrail_assignments VALUES ($1, 'api_key', 'key-3')`, foreign)

	q := NewGuardrailQueries(pool)
	cases := []struct {
		key, want string
		perKey    bool
	}{
		{"key-1", "keyed", true},
		{"key-2", "default", false},
		{"", "default", false},
		{"key-3", "default", false},
	}
	for _, c := range cases {
		got, err := q.ResolveForRequest(ctx, user, c.key)
		if err != nil || len(got) != 1 || got[0].Name != c.want {
			t.Fatalf("key %q: got %v err %v want %s", c.key, got, err, c.want)
		}
		if len(got[0].Assignments) != 1 || (got[0].Assignments[0].TargetType == "api_key") != c.perKey {
			t.Fatalf("key %q: assignments %+v", c.key, got[0].Assignments)
		}
	}
	if got, err := q.ResolveForRequest(ctx, other, "key-1"); err != nil || len(got) != 0 {
		t.Fatalf("foreign user: %v %v", got, err)
	}

	list, err := q.ListByUser(ctx, user)
	if err != nil || len(list) != 2 {
		t.Fatalf("list %v %v", list, err)
	}
	for _, g := range list {
		if len(g.Assignments) != 1 {
			t.Fatalf("%s assignments %+v", g.Name, g.Assignments)
		}
	}
	pool.Exec(ctx, `DELETE FROM guardrail_assignments WHERE guardrail_id = $1`, keyed)
	list, _ = q.ListByUser(ctx, user)
	for _, g := range list {
		if g.Assignments == nil {
			t.Fatalf("%s assignments nil", g.Name)
		}
	}
}
