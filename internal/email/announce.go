package email

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/mail"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openpaths/openpaths/internal/model"
)

const (
	defaultAnnounceTemplate = "model-announcement.html"
	announceSampleSize      = 10
	announceAbortAfter      = 10
	maxAnnounceSubject      = 200
)

type announceUser struct {
	ID         string
	Email      string
	Disabled   bool
	Suppressed bool
	Status     string
}

type CatalogEntry struct {
	ModelID     string    `json:"model_id"`
	FirstSeenAt time.Time `json:"first_seen_at"`
}

type AnnounceStatusRow struct {
	Counts       map[string]int
	AnnouncedAt  *time.Time
	RecentErrors []string
}

type announceStore interface {
	SyncCatalog(ctx context.Context, ids []string) (added []string, seeded bool, err error)
	PendingCatalog(ctx context.Context) ([]CatalogEntry, error)
	MarkAnnounced(ctx context.Context, modelID string) error
	Users(ctx context.Context, modelID string) ([]announceUser, error)
	Reserve(ctx context.Context, modelID, userID, email string, retryPending bool) (bool, error)
	Mark(ctx context.Context, modelID, userID, status, errMsg string) error
	Status(ctx context.Context, modelID string) (AnnounceStatusRow, error)
	Lock(ctx context.Context, modelID string) (unlock func(), ok bool, err error)
}

type AnnounceError struct {
	Code    int
	Message string
}

func (e *AnnounceError) Error() string { return e.Message }

func announceErr(code int, format string, a ...any) *AnnounceError {
	return &AnnounceError{Code: code, Message: fmt.Sprintf(format, a...)}
}

type AnnounceRequest struct {
	ModelID      string `json:"model_id"`
	Template     string `json:"template"`
	Subject      string `json:"subject"`
	DryRun       *bool  `json:"dry_run"`
	Confirm      bool   `json:"confirm"`
	Limit        int    `json:"limit"`
	TestEmail    string `json:"test_email"`
	RetryPending bool   `json:"retry_pending"`
}

type Exclusions struct {
	Disabled    int `json:"disabled"`
	Test        int `json:"test"`
	Suppressed  int `json:"suppressed"`
	AlreadySent int `json:"already_sent"`
	Pending     int `json:"pending"`
	NoEmail     int `json:"no_email"`
	Duplicate   int `json:"duplicate"`
}

type Preview struct {
	Subject string `json:"subject"`
	HTML    string `json:"html"`
	Text    string `json:"text"`
}

type AnnounceResult struct {
	ModelID    string     `json:"model_id"`
	DryRun     bool       `json:"dry_run"`
	Started    bool       `json:"started"`
	TestSent   string     `json:"test_sent,omitempty"`
	Template   string     `json:"template"`
	Subject    string     `json:"subject"`
	Users      int        `json:"users"`
	Recipients int        `json:"recipients"`
	Selected   int        `json:"selected"`
	Retries    int        `json:"retries"`
	Sample     []string   `json:"sample"`
	Excluded   Exclusions `json:"excluded"`
	Preview    *Preview   `json:"preview,omitempty"`
}

type AnnounceRun struct {
	ModelID    string     `json:"model_id"`
	State      string     `json:"state"`
	Total      int        `json:"total"`
	Sent       int        `json:"sent"`
	Failed     int        `json:"failed"`
	Skipped    int        `json:"skipped"`
	LastError  string     `json:"last_error,omitempty"`
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

type AnnounceStatus struct {
	ModelID      string         `json:"model_id"`
	Run          *AnnounceRun   `json:"run,omitempty"`
	Counts       map[string]int `json:"counts"`
	AnnouncedAt  *time.Time     `json:"announced_at,omitempty"`
	RecentErrors []string       `json:"recent_errors,omitempty"`
}

type Announcer struct {
	store     announceStore
	send      func(Message) error
	ready     func() error
	secret    string
	baseURL   string
	fromName  string
	emailsDir string
	models    []model.ModelConfig
	interval  time.Duration
	baseCtx   context.Context

	mu   sync.Mutex
	runs map[string]*AnnounceRun
}

func NewAnnouncer(ctx context.Context, db *pgxpool.Pool, emailsDir, secret string, models []model.ModelConfig) *Announcer {
	a := newAnnouncer(ctx, &pgAnnounceStore{db: db}, SendMessage, emailsDir, secret, models)
	a.interval = 200 * time.Millisecond
	a.ready = func() error {
		if os.Getenv("AWS_SMTP_USERNAME") == "" || os.Getenv("AWS_SMTP_PASSWORD") == "" {
			return fmt.Errorf("AWS_SMTP_USERNAME and AWS_SMTP_PASSWORD not set")
		}
		return nil
	}
	return a
}

func newAnnouncer(ctx context.Context, store announceStore, send func(Message) error, emailsDir, secret string, models []model.ModelConfig) *Announcer {
	a := &Announcer{
		store:     store,
		send:      send,
		secret:    secret,
		baseURL:   "https://openpaths.io",
		fromName:  "OpenPaths",
		emailsDir: emailsDir,
		models:    models,
		baseCtx:   ctx,
		runs:      map[string]*AnnounceRun{},
	}
	if data, err := os.ReadFile(filepath.Join(emailsDir, "drip_config.json")); err == nil {
		var c DripConfig
		if json.Unmarshal(data, &c) == nil {
			if c.BaseURL != "" {
				a.baseURL = strings.TrimRight(c.BaseURL, "/")
			}
			if c.FromName != "" {
				a.fromName = c.FromName
			}
		}
	}
	return a
}

var (
	testIDRe    = regexp.MustCompile(`(^|[-_/.])(test|e2e|smoke|dummy|mock)([-_/.]|$)`)
	virtualIDRe = regexp.MustCompile(`(^|/)auto(-|$)|-latest$`)
)

func CatalogCandidates(models []model.ModelConfig) []string {
	aliases := map[string]bool{}
	for _, m := range models {
		for _, al := range m.Aliases {
			aliases[strings.ToLower(al)] = true
		}
	}
	seen := map[string]bool{}
	var ids []string
	for _, m := range models {
		id := strings.TrimSpace(m.ID)
		low := strings.ToLower(id)
		if id == "" || m.Deprecated || aliases[low] || seen[low] || testIDRe.MatchString(low) || virtualIDRe.MatchString(low) {
			continue
		}
		seen[low] = true
		ids = append(ids, id)
	}
	return ids
}

func (a *Announcer) DetectNewModels(ctx context.Context) ([]string, error) {
	added, seeded, err := a.store.SyncCatalog(ctx, CatalogCandidates(a.models))
	if err != nil {
		return nil, err
	}
	if seeded {
		log.Printf("announce: seeded model_catalog_seen with existing catalog (marked announced)")
		return nil, nil
	}
	if len(added) > 0 {
		log.Printf("announce: %d new catalog model(s) pending announcement: %s", len(added), strings.Join(added, ", "))
	}
	return added, nil
}

func (a *Announcer) Candidates(ctx context.Context) ([]CatalogEntry, error) {
	return a.store.PendingCatalog(ctx)
}

func selectRecipients(users []announceUser, limit int, retryPending bool) (sel []announceUser, ex Exclusions, eligible int) {
	seen := map[string]bool{}
	for _, u := range users {
		if u.Status == "sent" {
			seen[strings.ToLower(strings.TrimSpace(u.Email))] = true
		}
	}
	for _, u := range users {
		addr := strings.ToLower(strings.TrimSpace(u.Email))
		switch {
		case addr == "":
			ex.NoEmail++
		case u.Disabled:
			ex.Disabled++
		case isTestEmail(addr):
			ex.Test++
		case u.Suppressed:
			ex.Suppressed++
		case u.Status == "sent":
			ex.AlreadySent++
		case u.Status == "pending" && !retryPending:
			ex.Pending++
		case seen[addr]:
			ex.Duplicate++
		default:
			seen[addr] = true
			eligible++
			if limit <= 0 || len(sel) < limit {
				sel = append(sel, u)
			}
		}
	}
	return sel, ex, eligible
}

type announceRender struct {
	subject string
	html    string
	baseURL string
	secret  string
	name    string
}

func (r *announceRender) unsubURL(addr string) string {
	return fmt.Sprintf("%s/unsubscribe?e=%s&t=%s", r.baseURL, url.QueryEscape(strings.ToLower(addr)), UnsubscribeToken(r.secret, addr))
}

func (r *announceRender) message(addr string) Message {
	u := r.unsubURL(addr)
	return Message{
		To:             addr,
		Subject:        r.subject,
		HTML:           strings.ReplaceAll(r.html, "{{.UnsubscribeURL}}", u),
		UnsubscribeURL: u,
		FromName:       r.name,
	}
}

func (a *Announcer) prepare(m *model.ModelConfig, req AnnounceRequest) (*announceRender, string, error) {
	tmpl := req.Template
	if tmpl == "" {
		tmpl = defaultAnnounceTemplate
	}
	if filepath.Base(tmpl) != tmpl || !strings.HasSuffix(tmpl, ".html") {
		return nil, "", announceErr(400, "template must be an .html filename inside the emails dir")
	}
	data, err := os.ReadFile(filepath.Join(a.emailsDir, tmpl))
	if err != nil {
		return nil, "", announceErr(400, "template %s not readable", tmpl)
	}
	card := BuildModelCard(filepath.Join(a.emailsDir, "model_notes.json"), m)
	h := string(data)
	for k, v := range map[string]string{
		"{{.Model.ID}}":      card.ID,
		"{{.Model.Name}}":    card.Name,
		"{{.Model.Price}}":   card.Price,
		"{{.Model.Context}}": card.Context,
		"{{.Model.Card}}":    card.HTML,
	} {
		h = strings.ReplaceAll(h, k, v)
	}
	if strings.Contains(h, "{{.ModelCards.") {
		cards, _ := renderModelCards(filepath.Join(a.emailsDir, "model_notes.json"), a.models)
		for section, c := range cards {
			h = strings.ReplaceAll(h, "{{.ModelCards."+section+"}}", c)
		}
	}
	h2 := strings.ReplaceAll(h, "{{.UnsubscribeURL}}", "")
	if i := strings.Index(h2, "{{."); i >= 0 {
		end := min(i+40, len(h2))
		return nil, "", announceErr(400, "unreplaced placeholder in template: %s", h2[i:end])
	}
	subject := SanitizeHeader(req.Subject)
	if subject == "" {
		subject = fmt.Sprintf("%s is live on OpenPaths", card.Name)
	}
	if r := []rune(subject); len(r) > maxAnnounceSubject {
		subject = string(r[:maxAnnounceSubject])
	}
	return &announceRender{subject: subject, html: h, baseURL: a.baseURL, secret: a.secret, name: a.fromName}, tmpl, nil
}

func (a *Announcer) Announce(ctx context.Context, req AnnounceRequest) (*AnnounceResult, error) {
	req.ModelID = strings.TrimSpace(req.ModelID)
	if req.ModelID == "" {
		return nil, announceErr(400, "model_id required")
	}
	m := FindModel(a.models, req.ModelID)
	if m == nil {
		return nil, announceErr(404, "model %s not in catalog", req.ModelID)
	}
	if m.Deprecated {
		return nil, announceErr(400, "model %s is deprecated", m.ID)
	}
	if req.Limit < 0 {
		return nil, announceErr(400, "limit must be >= 0")
	}
	dryRun := req.DryRun == nil || *req.DryRun
	rnd, tmpl, err := a.prepare(m, req)
	if err != nil {
		return nil, err
	}
	res := &AnnounceResult{ModelID: m.ID, DryRun: dryRun, Template: tmpl, Subject: rnd.subject, Sample: []string{}}

	if req.TestEmail != "" {
		addr, perr := mail.ParseAddress(req.TestEmail)
		if perr != nil {
			return nil, announceErr(400, "invalid test_email")
		}
		res.Recipients, res.Selected, res.Sample = 1, 1, []string{addr.Address}
		res.Preview = previewOf(rnd, addr.Address)
		if dryRun {
			return res, nil
		}
		if !req.Confirm {
			return nil, announceErr(400, "real run requires confirm:true")
		}
		if err := a.checkReady(); err != nil {
			return nil, err
		}
		msg := rnd.message(addr.Address)
		msg.Subject = "[TEST] " + msg.Subject
		if err := a.send(msg); err != nil {
			return nil, announceErr(502, "test send failed: %v", err)
		}
		res.TestSent = addr.Address
		return res, nil
	}

	users, err := a.store.Users(ctx, m.ID)
	if err != nil {
		return nil, announceErr(500, "loading users: %v", err)
	}
	sel, ex, eligible := selectRecipients(users, req.Limit, req.RetryPending)
	res.Users, res.Recipients, res.Selected, res.Excluded = len(users), eligible, len(sel), ex
	for _, u := range sel {
		if u.Status == "error" {
			res.Retries++
		}
	}
	for i := 0; i < len(sel) && i < announceSampleSize; i++ {
		res.Sample = append(res.Sample, sel[i].Email)
	}
	previewAddr := "preview@example.com"
	if len(sel) > 0 {
		previewAddr = sel[0].Email
	}
	res.Preview = previewOf(rnd, previewAddr)
	if dryRun {
		return res, nil
	}
	if !req.Confirm {
		return nil, announceErr(400, "real run requires confirm:true")
	}
	if err := a.checkReady(); err != nil {
		return nil, err
	}
	if len(sel) == 0 {
		return res, nil
	}

	a.mu.Lock()
	if r := a.runs[m.ID]; r != nil && r.State == "running" {
		a.mu.Unlock()
		return nil, announceErr(409, "announcement for %s already running", m.ID)
	}
	unlock, ok, err := a.store.Lock(ctx, m.ID)
	if err != nil {
		a.mu.Unlock()
		return nil, announceErr(500, "lock: %v", err)
	}
	if !ok {
		a.mu.Unlock()
		return nil, announceErr(409, "announcement for %s locked by another instance", m.ID)
	}
	run := &AnnounceRun{ModelID: m.ID, State: "running", Total: len(sel), StartedAt: time.Now()}
	a.runs[m.ID] = run
	a.mu.Unlock()

	complete := req.Limit <= 0
	go a.run(run, rnd, sel, unlock, req.RetryPending, complete)
	res.Started = true
	return res, nil
}

func previewOf(r *announceRender, addr string) *Preview {
	msg := r.message(addr)
	return &Preview{Subject: msg.Subject, HTML: msg.HTML, Text: HTMLToText(msg.HTML)}
}

func (a *Announcer) checkReady() error {
	if a.ready == nil {
		return nil
	}
	if err := a.ready(); err != nil {
		return announceErr(503, "email not configured: %v", err)
	}
	return nil
}

func (a *Announcer) update(run *AnnounceRun, f func(*AnnounceRun)) {
	a.mu.Lock()
	f(run)
	a.mu.Unlock()
}

func (a *Announcer) run(run *AnnounceRun, r *announceRender, sel []announceUser, unlock func(), retryPending, complete bool) {
	defer unlock()
	finish := func(state string) {
		a.update(run, func(x *AnnounceRun) {
			now := time.Now()
			x.State, x.FinishedAt = state, &now
		})
	}
	defer func() {
		if p := recover(); p != nil {
			log.Printf("announce %s: panic: %v", run.ModelID, p)
			a.update(run, func(x *AnnounceRun) { x.LastError = fmt.Sprint(p) })
			finish("failed")
		}
	}()

	ctx := a.baseCtx
	var tick <-chan time.Time
	if a.interval > 0 {
		t := time.NewTicker(a.interval)
		defer t.Stop()
		tick = t.C
	}
	consecutive := 0
	for _, u := range sel {
		if tick != nil {
			select {
			case <-tick:
			case <-ctx.Done():
				finish("canceled")
				return
			}
		} else if ctx.Err() != nil {
			finish("canceled")
			return
		}
		claimed, err := a.store.Reserve(ctx, run.ModelID, u.ID, u.Email, retryPending)
		if err != nil {
			a.update(run, func(x *AnnounceRun) { x.Failed++; x.LastError = "reserve: " + err.Error() })
			consecutive++
		} else if !claimed {
			a.update(run, func(x *AnnounceRun) { x.Skipped++ })
			continue
		} else if err := a.send(r.message(u.Email)); err != nil {
			msg := truncate(err.Error(), 500)
			if merr := a.store.Mark(ctx, run.ModelID, u.ID, "error", msg); merr != nil {
				log.Printf("announce %s: mark error %s: %v", run.ModelID, u.Email, merr)
			}
			a.update(run, func(x *AnnounceRun) { x.Failed++; x.LastError = msg })
			consecutive++
		} else {
			if merr := a.store.Mark(ctx, run.ModelID, u.ID, "sent", ""); merr != nil {
				log.Printf("announce %s: mark sent %s: %v", run.ModelID, u.Email, merr)
			}
			a.update(run, func(x *AnnounceRun) { x.Sent++ })
			consecutive = 0
		}
		if consecutive >= announceAbortAfter {
			log.Printf("announce %s: aborting after %d consecutive failures", run.ModelID, consecutive)
			finish("aborted")
			return
		}
	}
	if complete && run.Failed == 0 {
		if err := a.store.MarkAnnounced(ctx, run.ModelID); err != nil {
			log.Printf("announce %s: mark announced: %v", run.ModelID, err)
		}
	}
	log.Printf("announce %s: done sent=%d failed=%d skipped=%d", run.ModelID, run.Sent, run.Failed, run.Skipped)
	finish("done")
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

func (a *Announcer) Status(ctx context.Context, modelID string) (*AnnounceStatus, error) {
	m := FindModel(a.models, modelID)
	id := strings.TrimSpace(modelID)
	if m != nil {
		id = m.ID
	}
	row, err := a.store.Status(ctx, id)
	if err != nil {
		return nil, err
	}
	st := &AnnounceStatus{ModelID: id, Counts: row.Counts, AnnouncedAt: row.AnnouncedAt, RecentErrors: row.RecentErrors}
	a.mu.Lock()
	if r := a.runs[id]; r != nil {
		c := *r
		st.Run = &c
	}
	a.mu.Unlock()
	return st, nil
}
