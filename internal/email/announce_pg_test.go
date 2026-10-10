package email

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openpaths/openpaths/internal/model"
)

func testPool(t *testing.T) *pgxpool.Pool {
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
	schema := fmt.Sprintf("announce_t_%d", time.Now().UnixNano())
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
	mig, err := os.ReadFile(filepath.Join("..", "db", "migrations", "034_model_announcements.sql"))
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`CREATE TABLE users (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), email TEXT NOT NULL UNIQUE, disabled BOOLEAN NOT NULL DEFAULT FALSE, created_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		`CREATE TABLE email_suppressions (email TEXT PRIMARY KEY)`,
		string(mig),
	} {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	return pool
}

func TestPGCatalogSync(t *testing.T) {
	ctx := context.Background()
	s := &pgAnnounceStore{db: testPool(t)}
	added, seeded, err := s.SyncCatalog(ctx, []string{"a", "b"})
	if err != nil || !seeded || len(added) != 0 {
		t.Fatalf("seed %v %v %v", added, seeded, err)
	}
	if p, _ := s.PendingCatalog(ctx); len(p) != 0 {
		t.Fatalf("seeded rows pending: %v", p)
	}
	added, seeded, err = s.SyncCatalog(ctx, []string{"a", "b", "c", "c"})
	if err != nil || seeded || len(added) != 1 || added[0] != "c" {
		t.Fatalf("new %v %v %v", added, seeded, err)
	}
	if added, _, _ = s.SyncCatalog(ctx, []string{"a", "b", "c"}); len(added) != 0 {
		t.Fatalf("repeat %v", added)
	}
	p, _ := s.PendingCatalog(ctx)
	if len(p) != 1 || p[0].ModelID != "c" {
		t.Fatalf("pending %v", p)
	}
	if err := s.MarkAnnounced(ctx, "c"); err != nil {
		t.Fatal(err)
	}
	if p, _ = s.PendingCatalog(ctx); len(p) != 0 {
		t.Fatalf("pending after announce %v", p)
	}
}

func TestPGReserveLockAndFullRun(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	s := &pgAnnounceStore{db: pool}
	for _, e := range []string{"a@real.com", "b@real.com", "c@real.com", "d@real.com"} {
		pool.Exec(ctx, `INSERT INTO users (email) VALUES ($1)`, e)
	}
	pool.Exec(ctx, `UPDATE users SET disabled = true WHERE email = 'c@real.com'`)
	pool.Exec(ctx, `INSERT INTO email_suppressions VALUES ('d@real.com')`)

	users, err := s.Users(ctx, "m")
	if err != nil || len(users) != 4 {
		t.Fatalf("%v %v", users, err)
	}
	byEmail := map[string]announceUser{}
	for _, u := range users {
		byEmail[u.Email] = u
	}
	if !byEmail["c@real.com"].Disabled || !byEmail["d@real.com"].Suppressed || byEmail["a@real.com"].Status != "" {
		t.Fatalf("flags %+v", byEmail)
	}

	uid := byEmail["a@real.com"].ID
	if ok, err := s.Reserve(ctx, "m", uid, "a@real.com", false); !ok || err != nil {
		t.Fatalf("first reserve %v %v", ok, err)
	}
	if ok, _ := s.Reserve(ctx, "m", uid, "a@real.com", false); ok {
		t.Fatal("double reserve of pending row")
	}
	if ok, _ := s.Reserve(ctx, "m", uid, "a@real.com", true); !ok {
		t.Fatal("retry_pending reserve refused")
	}
	s.Mark(ctx, "m", uid, "error", "boom")
	if ok, _ := s.Reserve(ctx, "m", uid, "a@real.com", false); !ok {
		t.Fatal("error row not retryable")
	}
	s.Mark(ctx, "m", uid, "sent", "")
	if ok, _ := s.Reserve(ctx, "m", uid, "a@real.com", true); ok {
		t.Fatal("sent row re-reserved")
	}
	if ok, _ := s.Reserve(ctx, "other", uid, "a@real.com", false); !ok {
		t.Fatal("reservations must be per model")
	}

	unlock, ok, err := s.Lock(ctx, "m")
	if !ok || err != nil {
		t.Fatalf("lock %v %v", ok, err)
	}
	if _, ok2, _ := s.Lock(ctx, "m"); ok2 {
		t.Fatal("second lock acquired")
	}
	uz, ok3, _ := s.Lock(ctx, "z")
	if !ok3 {
		t.Fatal("lock not per model")
	}
	uz()
	unlock()
	u2, ok, _ := s.Lock(ctx, "m")
	if !ok {
		t.Fatal("lock not released")
	}
	u2()

	snd := &fakeSender{}
	a := newAnnouncer(ctx, s, snd.send, filepath.Join("..", "..", "emails"), "secret", []model.ModelConfig{{ID: "m", InputPricePer1M: 1, OutputPricePer1M: 2}})
	res, err := a.Announce(ctx, AnnounceRequest{ModelID: "m"})
	if err != nil || res.Recipients != 1 || res.Excluded.AlreadySent != 1 || res.Excluded.Disabled != 1 || res.Excluded.Suppressed != 1 {
		t.Fatalf("dry %+v %v", res, err)
	}
	var rows int
	pool.QueryRow(ctx, `SELECT count(*) FROM model_announcements WHERE model_id = 'm'`).Scan(&rows)
	if rows != 1 {
		t.Fatalf("dry run wrote rows: %d", rows)
	}
	if _, err := a.Announce(ctx, AnnounceRequest{ModelID: "m", DryRun: ptr(false), Confirm: true}); err != nil {
		t.Fatal(err)
	}
	run := waitDone(t, a, "m")
	if run.Sent != 1 || snd.count() != 1 || snd.tos()[0] != "b@real.com" {
		t.Fatalf("%+v %v", run, snd.tos())
	}
	st, _ := a.Status(ctx, "m")
	if st.Counts["sent"] != 2 || st.AnnouncedAt == nil {
		t.Fatalf("status %+v", st)
	}

	pool.Exec(ctx, `DELETE FROM users WHERE email = 'a@real.com'`)
	pool.QueryRow(ctx, `SELECT count(*) FROM model_announcements WHERE email = 'a@real.com'`).Scan(&rows)
	if rows != 0 {
		t.Fatal("rows survive user deletion")
	}
}

func TestPGDripModelLaunchSkipsLateSignups(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	prev := Suppressed
	Suppressed = func(string) bool { return true }
	defer func() { Suppressed = prev }()

	for e, ts := range map[string]string{
		"old@real.com":    "2026-01-01 00:00+00",
		"inwin@real.com":  "2026-09-06 12:00+00",
		"late@real.com":   "2026-09-07 00:00:01+00",
		"newest@real.com": "2026-10-01 00:00+00",
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO users (email, created_at) VALUES ($1, $2::timestamptz)`, e, ts); err != nil {
			t.Fatal(err)
		}
	}
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "t.html"), []byte("<p>hi</p>"), 0o644)

	sentTo := func(id int) map[string]bool {
		rows, err := pool.Query(ctx, `SELECT u.email FROM drip_emails_sent d JOIN users u ON u.id::text = d.user_id WHERE d.email_id = $1`, id)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		out := map[string]bool{}
		for rows.Next() {
			var e string
			rows.Scan(&e)
			out[e] = true
		}
		return out
	}
	run := func(de DripEmail) {
		r := &DripRunner{db: pool, config: DripConfig{BaseURL: "https://x", Emails: []DripEmail{de}}, emailsDir: dir, secret: "s"}
		if err := r.EnsureTable(ctx); err != nil {
			t.Fatal(err)
		}
		r.RunScheduled(ctx)
	}

	run(DripEmail{ID: 23, DelayDays: 1, Subject: "s", Template: "t.html", Kind: KindModelLaunch, LaunchDate: "2026-08-23", MaxAgeDays: 14})
	got := sentTo(23)
	if len(got) != 2 || !got["old@real.com"] || !got["inwin@real.com"] {
		t.Fatalf("model_launch recipients: %v", got)
	}

	run(DripEmail{ID: 23, DelayDays: 1, Subject: "s", Template: "t.html", Kind: KindModelLaunch})
	if got = sentTo(23); len(got) != 2 {
		t.Fatalf("misconfigured model_launch entry sent mail: %v", got)
	}

	pool.Exec(ctx, `DELETE FROM drip_emails_sent`)
	run(DripEmail{ID: 5, DelayDays: 1, Subject: "s", Template: "t.html"})
	if got = sentTo(5); len(got) != 4 {
		t.Fatalf("regular drip must stay unfiltered: %v", got)
	}
}
