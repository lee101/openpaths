package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"github.com/openpaths/openpaths/internal/config"
	"github.com/openpaths/openpaths/internal/db"
)

type userRecord struct {
	ID                    string
	Email                 string
	Name                  string
	StripeCustomerID      string
	StripePaymentMethodID string
}

type countRow struct {
	Table string
	Count int64
}

type stripeClient struct {
	secretKey string
	client    *http.Client
}

type stripeCustomer struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Deleted bool   `json:"deleted"`
}

type stripePaymentMethod struct {
	ID string `json:"id"`
}

func main() {
	var (
		email      = flag.String("email", "", "email address to GDPR-delete")
		configPath = flag.String("config", "config.yaml", "path to OpenPaths config.yaml")
		execute    = flag.Bool("execute", false, "actually delete data; without this flag the script is a dry run")
		skipStripe = flag.Bool("skip-stripe", false, "skip Stripe deletion")
	)
	flag.Parse()

	targetEmail := strings.TrimSpace(strings.ToLower(*email))
	if targetEmail == "" {
		log.Fatal("--email is required")
	}

	_ = godotenv.Load()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if cfg.Database.URL == "" {
		log.Fatal("database URL is empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	database, err := db.New(ctx, cfg.Database)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer database.Close()

	user, err := lookupUser(ctx, database.Pool, targetEmail)
	if err != nil {
		log.Fatalf("lookup user: %v", err)
	}
	if user == nil {
		log.Printf("No database user found for %q.", targetEmail)
	} else {
		log.Printf("Database user found: id=%s email=%s stripe_customer_id=%s stripe_payment_method_id=%s",
			user.ID, user.Email, printable(user.StripeCustomerID), printable(user.StripePaymentMethodID))
		counts, err := collectCounts(ctx, database.Pool, user.ID)
		if err != nil {
			log.Fatalf("collect row counts: %v", err)
		}
		logCounts("database rows that will be removed", counts)
	}

	if !*skipStripe {
		if cfg.Stripe.SecretKey == "" {
			log.Fatal("Stripe secret key is empty; use --skip-stripe only if Stripe deletion was completed separately")
		}
		stripe := &stripeClient{
			secretKey: cfg.Stripe.SecretKey,
			client:    &http.Client{Timeout: 30 * time.Second},
		}
		customers, err := stripe.resolveCustomers(ctx, targetEmail, user)
		if err != nil {
			log.Fatalf("resolve Stripe customers: %v", err)
		}
		log.Printf("Stripe customers matched: %d", len(customers))
		for _, customer := range customers {
			log.Printf("Stripe customer match: id=%s email=%s name=%s", customer.ID, customer.Email, customer.Name)
		}
		if *execute {
			if err := stripe.deleteCustomerData(ctx, customers, user); err != nil {
				log.Fatalf("delete Stripe data: %v", err)
			}
		}
	}

	if user != nil && *execute {
		if err := eraseDatabaseUser(ctx, database.Pool, user.ID); err != nil {
			log.Fatalf("erase database user: %v", err)
		}
		remaining, err := collectRemainingEmailMatches(ctx, database.Pool, targetEmail, user.ID)
		if err != nil {
			log.Fatalf("verify database deletion: %v", err)
		}
		logCounts("post-delete database matches", remaining)
	}
	textMatches, err := collectTextColumnMatches(ctx, database.Pool, targetEmail)
	if err != nil {
		log.Fatalf("scan text columns for email: %v", err)
	}
	logCounts("text-column email matches", textMatches)

	if !*execute {
		log.Printf("Dry run complete. Re-run with --execute to delete database and Stripe records.")
		return
	}
	log.Printf("GDPR deletion complete for %q.", targetEmail)
}

func printable(value string) string {
	if value == "" {
		return "<none>"
	}
	return value
}

func lookupUser(ctx context.Context, pool interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, email string) (*userRecord, error) {
	var user userRecord
	err := pool.QueryRow(ctx, `
		SELECT id::text, email, name,
		       COALESCE(stripe_customer_id, ''),
		       COALESCE(stripe_payment_method_id, '')
		FROM users
		WHERE lower(email) = lower($1)
		ORDER BY created_at ASC
		LIMIT 1
	`, email).Scan(&user.ID, &user.Email, &user.Name, &user.StripeCustomerID, &user.StripePaymentMethodID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func collectCounts(ctx context.Context, pool interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, userID string) ([]countRow, error) {
	queries := []struct {
		table string
		sql   string
	}{
		{"users", `SELECT count(*) FROM users WHERE id = $1`},
		{"api_keys", `SELECT count(*) FROM api_keys WHERE user_id = $1`},
		{"usage_logs", `SELECT count(*) FROM usage_logs WHERE user_id = $1 OR api_key_id IN (SELECT id FROM api_keys WHERE user_id = $1)`},
		{"credit_balances", `SELECT count(*) FROM credit_balances WHERE user_id = $1`},
		{"credit_transactions", `SELECT count(*) FROM credit_transactions WHERE user_id = $1`},
		{"crypto_checkout_intents", `SELECT count(*) FROM crypto_checkout_intents WHERE user_id = $1`},
		{"autotopup_charges", `SELECT count(*) FROM autotopup_charges WHERE user_id = $1`},
		{"stripe_deposits", `SELECT count(*) FROM stripe_deposits WHERE user_id = $1`},
		{"fine_tuning_jobs", `SELECT count(*) FROM fine_tuning_jobs WHERE user_id = $1`},
		{"fine_tuning_events", `SELECT count(*) FROM fine_tuning_events WHERE job_id IN (SELECT id FROM fine_tuning_jobs WHERE user_id = $1)`},
		{"fine_tuning_files", `SELECT count(*) FROM fine_tuning_files WHERE user_id = $1`},
		{"user_provider_keys", `SELECT count(*) FROM user_provider_keys WHERE user_id = $1`},
	}
	var counts []countRow
	for _, q := range queries {
		var count int64
		if err := pool.QueryRow(ctx, q.sql, userID).Scan(&count); err != nil {
			return nil, fmt.Errorf("%s: %w", q.table, err)
		}
		counts = append(counts, countRow{Table: q.table, Count: count})
	}
	if ok, err := tableExists(ctx, pool, "drip_emails_sent"); err != nil {
		return nil, err
	} else if ok {
		var count int64
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM drip_emails_sent WHERE user_id = $1`, userID).Scan(&count); err != nil {
			return nil, fmt.Errorf("drip_emails_sent: %w", err)
		}
		counts = append(counts, countRow{Table: "drip_emails_sent", Count: count})
	}
	sort.Slice(counts, func(i, j int) bool { return counts[i].Table < counts[j].Table })
	return counts, nil
}

func tableExists(ctx context.Context, pool interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, table string) (bool, error) {
	var exists bool
	if err := pool.QueryRow(ctx, `SELECT to_regclass('public.' || $1) IS NOT NULL`, table).Scan(&exists); err != nil {
		return false, fmt.Errorf("check table %s: %w", table, err)
	}
	return exists, nil
}

func logCounts(label string, counts []countRow) {
	log.Printf("%s:", label)
	for _, c := range counts {
		log.Printf("  %s: %d", c.Table, c.Count)
	}
}

func eraseDatabaseUser(ctx context.Context, pool interface {
	Begin(context.Context) (pgx.Tx, error)
}, userID string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	statements := []struct {
		label string
		sql   string
	}{
		{"usage_logs", `DELETE FROM usage_logs WHERE user_id = $1 OR api_key_id IN (SELECT id FROM api_keys WHERE user_id = $1)`},
		{"drip_emails_sent", `DELETE FROM drip_emails_sent WHERE user_id = $1`},
		{"users", `DELETE FROM users WHERE id = $1`},
	}
	for _, stmt := range statements {
		if stmt.label == "drip_emails_sent" {
			ok, err := tableExists(ctx, tx, "drip_emails_sent")
			if err != nil {
				return err
			}
			if !ok {
				continue
			}
		}
		tag, err := tx.Exec(ctx, stmt.sql, userID)
		if err != nil {
			return fmt.Errorf("delete %s: %w", stmt.label, err)
		}
		log.Printf("Deleted %d row(s) from %s.", tag.RowsAffected(), stmt.label)
	}
	return tx.Commit(ctx)
}

func collectRemainingEmailMatches(ctx context.Context, pool interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, email, userID string) ([]countRow, error) {
	queries := []struct {
		table string
		sql   string
		args  []any
	}{
		{"users_by_email", `SELECT count(*) FROM users WHERE lower(email) = lower($1)`, []any{email}},
		{"users_by_id", `SELECT count(*) FROM users WHERE id = $1`, []any{userID}},
		{"api_keys", `SELECT count(*) FROM api_keys WHERE user_id = $1`, []any{userID}},
		{"usage_logs", `SELECT count(*) FROM usage_logs WHERE user_id = $1`, []any{userID}},
	}
	var counts []countRow
	for _, q := range queries {
		var count int64
		if err := pool.QueryRow(ctx, q.sql, q.args...).Scan(&count); err != nil {
			return nil, fmt.Errorf("%s: %w", q.table, err)
		}
		counts = append(counts, countRow{Table: q.table, Count: count})
	}
	return counts, nil
}

func collectTextColumnMatches(ctx context.Context, pool interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}, email string) ([]countRow, error) {
	rows, err := pool.Query(ctx, `
		SELECT table_schema, table_name, column_name
		FROM information_schema.columns
		WHERE table_schema = 'public'
		  AND (
		    data_type IN ('text', 'character varying', 'character', 'json', 'jsonb')
		    OR udt_name = 'citext'
		  )
		ORDER BY table_name, ordinal_position
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []countRow
	pattern := "%" + email + "%"
	for rows.Next() {
		var schema, table, column string
		if err := rows.Scan(&schema, &table, &column); err != nil {
			return nil, err
		}
		sql := fmt.Sprintf(
			`SELECT count(*) FROM %s.%s WHERE %s::text ILIKE $1`,
			quoteIdentifier(schema),
			quoteIdentifier(table),
			quoteIdentifier(column),
		)
		var count int64
		if err := pool.QueryRow(ctx, sql, pattern).Scan(&count); err != nil {
			return nil, fmt.Errorf("%s.%s.%s: %w", schema, table, column, err)
		}
		if count > 0 {
			matches = append(matches, countRow{
				Table: fmt.Sprintf("%s.%s.%s", schema, table, column),
				Count: count,
			})
		}
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	if len(matches) == 0 {
		return []countRow{{Table: "<none>", Count: 0}}, nil
	}
	return matches, nil
}

func quoteIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func (s *stripeClient) resolveCustomers(ctx context.Context, email string, user *userRecord) ([]stripeCustomer, error) {
	byID := map[string]stripeCustomer{}
	if user != nil && user.StripeCustomerID != "" {
		customer, err := s.retrieveCustomer(ctx, user.StripeCustomerID)
		if err != nil {
			log.Printf("Stripe customer %s from DB could not be retrieved: %v", user.StripeCustomerID, err)
		} else if customer.ID != "" {
			byID[customer.ID] = customer
		}
	}
	listed, err := s.listCustomersByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	for _, customer := range listed {
		if customer.ID != "" {
			byID[customer.ID] = customer
		}
	}
	customers := make([]stripeCustomer, 0, len(byID))
	for _, customer := range byID {
		customers = append(customers, customer)
	}
	sort.Slice(customers, func(i, j int) bool { return customers[i].ID < customers[j].ID })
	return customers, nil
}

func (s *stripeClient) deleteCustomerData(ctx context.Context, customers []stripeCustomer, user *userRecord) error {
	seenPM := map[string]bool{}
	if user != nil && user.StripePaymentMethodID != "" {
		seenPM[user.StripePaymentMethodID] = true
	}
	for _, customer := range customers {
		paymentMethods, err := s.listPaymentMethods(ctx, customer.ID)
		if err != nil {
			log.Printf("Stripe payment methods for customer %s could not be listed: %v", customer.ID, err)
		}
		for _, pm := range paymentMethods {
			seenPM[pm.ID] = true
		}
	}
	for pmID := range seenPM {
		if err := s.detachPaymentMethod(ctx, pmID); err != nil {
			log.Printf("Stripe payment method %s detach failed: %v", pmID, err)
			continue
		}
		log.Printf("Detached Stripe payment method %s.", pmID)
	}
	for _, customer := range customers {
		if err := s.deleteCustomer(ctx, customer.ID); err != nil {
			return fmt.Errorf("delete Stripe customer %s: %w", customer.ID, err)
		}
		log.Printf("Deleted Stripe customer %s.", customer.ID)
	}
	return nil
}

func (s *stripeClient) retrieveCustomer(ctx context.Context, id string) (stripeCustomer, error) {
	var customer stripeCustomer
	err := s.request(ctx, http.MethodGet, "/v1/customers/"+url.PathEscape(id), nil, &customer)
	return customer, err
}

func (s *stripeClient) listCustomersByEmail(ctx context.Context, email string) ([]stripeCustomer, error) {
	values := url.Values{}
	values.Set("email", email)
	values.Set("limit", "100")
	var resp struct {
		Data []stripeCustomer `json:"data"`
	}
	if err := s.request(ctx, http.MethodGet, "/v1/customers?"+values.Encode(), nil, &resp); err != nil {
		return nil, fmt.Errorf("list customers by email: %w", err)
	}
	return resp.Data, nil
}

func (s *stripeClient) listPaymentMethods(ctx context.Context, customerID string) ([]stripePaymentMethod, error) {
	values := url.Values{}
	values.Set("customer", customerID)
	values.Set("type", "card")
	values.Set("limit", "100")
	var resp struct {
		Data []stripePaymentMethod `json:"data"`
	}
	if err := s.request(ctx, http.MethodGet, "/v1/payment_methods?"+values.Encode(), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (s *stripeClient) detachPaymentMethod(ctx context.Context, paymentMethodID string) error {
	var resp map[string]any
	return s.request(ctx, http.MethodPost, "/v1/payment_methods/"+url.PathEscape(paymentMethodID)+"/detach", nil, &resp)
}

func (s *stripeClient) deleteCustomer(ctx context.Context, customerID string) error {
	var resp struct {
		ID      string `json:"id"`
		Deleted bool   `json:"deleted"`
	}
	if err := s.request(ctx, http.MethodDelete, "/v1/customers/"+url.PathEscape(customerID), nil, &resp); err != nil {
		return err
	}
	if !resp.Deleted {
		return fmt.Errorf("Stripe did not confirm deletion")
	}
	return nil
}

func (s *stripeClient) request(ctx context.Context, method, path string, body io.Reader, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, "https://api.stripe.com"+path, body)
	if err != nil {
		return err
	}
	req.SetBasicAuth(s.secretKey, "")
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("stripe 404: %s", string(data))
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("stripe %d: %s", resp.StatusCode, string(data))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("parse stripe response: %w", err)
	}
	return nil
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.SetOutput(os.Stdout)
}
