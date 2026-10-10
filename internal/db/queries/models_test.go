package queries

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestUpsertReportsInserted(t *testing.T) {
	dsn := os.Getenv("OPENPATHS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("OPENPATHS_TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := fmt.Sprintf("models_t_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE")
	cfg, _ := pgxpool.ParseConfig(dsn)
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	mig, err := os.ReadFile(filepath.Join("..", "migrations", "003_model_metadata.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(mig)); err != nil {
		t.Fatal(err)
	}

	q := NewModelMetadataQueries(pool)
	m := &ModelMetadata{Provider: "p", ModelID: "m1"}
	if ins, err := q.Upsert(ctx, m); err != nil || !ins {
		t.Fatalf("first upsert inserted=%v err=%v", ins, err)
	}
	m.DisplayName = "changed"
	if ins, err := q.Upsert(ctx, m); err != nil || ins {
		t.Fatalf("second upsert inserted=%v err=%v", ins, err)
	}
	if ins, _ := q.Upsert(ctx, &ModelMetadata{Provider: "p", ModelID: "m2"}); !ins {
		t.Fatal("new model id not reported inserted")
	}
	if ins, _ := q.Upsert(ctx, &ModelMetadata{Provider: "p2", ModelID: "m1"}); !ins {
		t.Fatal("same id on new provider not reported inserted")
	}
	if n, _ := q.CountByProvider(ctx, "p"); n != 2 {
		t.Fatalf("count %d", n)
	}
}
