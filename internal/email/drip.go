package email

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DripConfig struct {
	Campaign    string      `json:"campaign"`
	Description string      `json:"description"`
	FromEmail   string      `json:"from_email"`
	FromName    string      `json:"from_name"`
	BaseURL     string      `json:"base_url"`
	Emails      []DripEmail `json:"emails"`
}

type DripEmail struct {
	ID          int    `json:"id"`
	DelayDays   int    `json:"delay_days"`
	Subject     string `json:"subject"`
	Template    string `json:"template"`
	Description string `json:"description"`
}

type DripRunner struct {
	db        *pgxpool.Pool
	config    DripConfig
	emailsDir string
}

// NewDripRunner loads the drip config and returns a runner.
func NewDripRunner(db *pgxpool.Pool, emailsDir string) (*DripRunner, error) {
	configPath := filepath.Join(emailsDir, "drip_config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("reading drip config: %w", err)
	}

	var config DripConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing drip config: %w", err)
	}

	return &DripRunner{
		db:        db,
		config:    config,
		emailsDir: emailsDir,
	}, nil
}

// EnsureTable creates the drip tracking table if it doesn't exist.
func (r *DripRunner) EnsureTable(ctx context.Context) error {
	_, err := r.db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS drip_emails_sent (
			id          BIGSERIAL PRIMARY KEY,
			user_id     TEXT NOT NULL,
			email_id    INT NOT NULL,
			sent_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(user_id, email_id)
		);
		CREATE INDEX IF NOT EXISTS idx_drip_user ON drip_emails_sent(user_id);
	`)
	return err
}

// SendWelcome sends the first drip email (id=1, delay=0) immediately for a new signup.
func (r *DripRunner) SendWelcome(ctx context.Context, userID, userEmail string) {
	go func() {
		if err := r.sendDripEmail(ctx, userID, userEmail, 1); err != nil {
			log.Printf("drip: failed to send welcome email to %s: %v", userEmail, err)
		}
	}()
}

// RunScheduled checks all users and sends any drip emails that are due.
// Call this periodically (e.g. every hour via a ticker).
func (r *DripRunner) RunScheduled(ctx context.Context) {
	for _, de := range r.config.Emails {
		if de.DelayDays == 0 {
			continue // welcome email is sent on signup
		}

		// Find users who signed up delay_days ago and haven't received this email yet.
		cutoff := time.Now().AddDate(0, 0, -de.DelayDays)
		rows, err := r.db.Query(ctx, `
			SELECT u.id, u.email FROM users u
			WHERE u.created_at <= $1
			  AND u.email != ''
			  AND u.disabled = false
			  AND NOT EXISTS (
				SELECT 1 FROM drip_emails_sent d
				WHERE d.user_id = u.id AND d.email_id = $2
			  )
			LIMIT 100
		`, cutoff, de.ID)
		if err != nil {
			log.Printf("drip: query error for email %d: %v", de.ID, err)
			continue
		}

		type userRow struct {
			ID    string
			Email string
		}
		var users []userRow
		for rows.Next() {
			var u userRow
			if err := rows.Scan(&u.ID, &u.Email); err != nil {
				log.Printf("drip: scan error: %v", err)
				continue
			}
			users = append(users, u)
		}
		rows.Close()

		for _, u := range users {
			if err := r.sendDripEmail(ctx, u.ID, u.Email, de.ID); err != nil {
				log.Printf("drip: failed to send email %d to %s: %v", de.ID, u.Email, err)
			}
			// Small delay between sends to avoid SES rate limits.
			time.Sleep(200 * time.Millisecond)
		}
	}
}

func (r *DripRunner) sendDripEmail(ctx context.Context, userID, userEmail string, emailID int) error {
	// Find the email config.
	var de *DripEmail
	for i := range r.config.Emails {
		if r.config.Emails[i].ID == emailID {
			de = &r.config.Emails[i]
			break
		}
	}
	if de == nil {
		return fmt.Errorf("unknown drip email id: %d", emailID)
	}

	// Check if already sent.
	var exists bool
	err := r.db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM drip_emails_sent WHERE user_id=$1 AND email_id=$2)",
		userID, emailID,
	).Scan(&exists)
	if err != nil {
		return fmt.Errorf("checking sent status: %w", err)
	}
	if exists {
		return nil
	}

	// Load template.
	templatePath := filepath.Join(r.emailsDir, de.Template)
	htmlBytes, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("reading template %s: %w", de.Template, err)
	}

	// Replace template variables.
	html := string(htmlBytes)
	unsubscribeURL := fmt.Sprintf("%s/account?unsubscribe=true", r.config.BaseURL)
	html = strings.ReplaceAll(html, "{{.UnsubscribeURL}}", unsubscribeURL)

	// Send the email.
	if err := Send(userEmail, de.Subject, html); err != nil {
		return fmt.Errorf("sending: %w", err)
	}

	// Record that we sent it.
	_, err = r.db.Exec(ctx,
		"INSERT INTO drip_emails_sent (user_id, email_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
		userID, emailID,
	)
	if err != nil {
		log.Printf("drip: failed to record sent email %d for user %s: %v", emailID, userID, err)
	}

	log.Printf("drip: sent email %d (%s) to %s", emailID, de.Subject, userEmail)
	return nil
}

// StartScheduler runs the drip scheduler in the background, checking every hour.
func (r *DripRunner) StartScheduler(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		// Run once immediately on startup.
		r.RunScheduled(ctx)

		for {
			select {
			case <-ticker.C:
				r.RunScheduled(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
	log.Println("drip: email scheduler started (hourly)")
}
