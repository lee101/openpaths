package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/openpaths/openpaths/internal/config"
	"github.com/openpaths/openpaths/internal/db"
	"github.com/openpaths/openpaths/internal/email"
)

func main() {
	slug := flag.String("slug", "", "announcement id, used for dedupe")
	subject := flag.String("subject", "", "email subject")
	tmpl := flag.String("template", "", "html template path")
	only := flag.String("only", "", "send to this address only")
	limit := flag.Int("limit", 0, "max recipients (0 = all)")
	rate := flag.Duration("rate", 120*time.Millisecond, "delay between sends")
	baseURL := flag.String("base-url", "https://openpaths.io", "site base url")
	send := flag.Bool("send", false, "actually send; default is dry run")
	flag.Parse()
	if *slug == "" || *subject == "" || *tmpl == "" {
		log.Fatal("-slug, -subject and -template are required")
	}
	_ = godotenv.Load()
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	database, err := db.New(ctx, cfg.Database)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()
	pool := database.Pool
	if _, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS announcements_sent (
		slug TEXT NOT NULL, email TEXT NOT NULL, sent_at TIMESTAMPTZ NOT NULL DEFAULT now(), PRIMARY KEY (slug, email))`); err != nil {
		log.Fatal(err)
	}
	body, err := os.ReadFile(*tmpl)
	if err != nil {
		log.Fatal(err)
	}
	rows, err := pool.Query(ctx, `SELECT DISTINCT LOWER(u.email) FROM users u
		WHERE NOT u.disabled AND u.email LIKE '%@%'
		AND NOT EXISTS (SELECT 1 FROM email_suppressions s WHERE s.email = LOWER(u.email))
		AND NOT EXISTS (SELECT 1 FROM announcements_sent a WHERE a.slug = $1 AND a.email = LOWER(u.email))
		ORDER BY 1`, *slug)
	if err != nil {
		log.Fatal(err)
	}
	var addrs []string
	for rows.Next() {
		var a string
		if err := rows.Scan(&a); err != nil {
			log.Fatal(err)
		}
		if skip(a) || (*only != "" && a != strings.ToLower(*only)) {
			continue
		}
		addrs = append(addrs, a)
	}
	rows.Close()
	if *only != "" && len(addrs) == 0 {
		addrs = []string{strings.ToLower(*only)}
	}
	if *limit > 0 && len(addrs) > *limit {
		addrs = addrs[:*limit]
	}
	log.Printf("recipients: %d (send=%v)", len(addrs), *send)
	if !*send {
		return
	}
	var ok, fail int
	for _, a := range addrs {
		u := fmt.Sprintf("%s/unsubscribe?e=%s&t=%s", *baseURL, url.QueryEscape(a), email.UnsubscribeToken(cfg.JWT.Secret, a))
		html := strings.ReplaceAll(string(body), "{{.UnsubscribeURL}}", u)
		if err := email.SendWithUnsubscribe(a, *subject, html, u); err != nil {
			fail++
			log.Printf("fail %s: %v", a, err)
			continue
		}
		ok++
		if _, err := pool.Exec(ctx, `INSERT INTO announcements_sent (slug, email) VALUES ($1, $2) ON CONFLICT DO NOTHING`, *slug, a); err != nil {
			log.Printf("record %s: %v", a, err)
		}
		time.Sleep(*rate)
	}
	log.Printf("sent=%d failed=%d", ok, fail)
}

func skip(a string) bool {
	return strings.HasSuffix(a, ".local") || strings.HasSuffix(a, "@example.com") || strings.Contains(a, "smoke-test") || strings.HasSuffix(a, ".invalid")
}
