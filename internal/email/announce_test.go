package email

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/openpaths/openpaths/internal/config"
	"github.com/openpaths/openpaths/internal/model"
)

type fakeStore struct {
	mu        sync.Mutex
	users     []announceUser
	rows      map[string]*announceUser
	errs      map[string]string
	writes    int
	announced map[string]bool
	locked    map[string]bool
	catalog   map[string]*time.Time
}

func newFakeStore(users ...announceUser) *fakeStore {
	return &fakeStore{users: users, rows: map[string]*announceUser{}, errs: map[string]string{}, announced: map[string]bool{}, locked: map[string]bool{}, catalog: map[string]*time.Time{}}
}

func (f *fakeStore) SyncCatalog(ctx context.Context, ids []string) ([]string, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writes++
	now := time.Now()
	if len(f.catalog) == 0 {
		for _, id := range ids {
			f.catalog[id] = &now
		}
		return nil, len(ids) > 0, nil
	}
	var added []string
	for _, id := range ids {
		if _, ok := f.catalog[id]; !ok {
			f.catalog[id] = nil
			added = append(added, id)
		}
	}
	return added, false, nil
}

func (f *fakeStore) PendingCatalog(ctx context.Context) ([]CatalogEntry, error) { return nil, nil }

func (f *fakeStore) MarkAnnounced(ctx context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writes++
	f.announced[id] = true
	return nil
}

func (f *fakeStore) Users(ctx context.Context, modelID string) ([]announceUser, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]announceUser, len(f.users))
	for i, u := range f.users {
		if r := f.rows[modelID+"|"+u.ID]; r != nil {
			u.Status = r.Status
		}
		out[i] = u
	}
	return out, nil
}

func (f *fakeStore) Reserve(ctx context.Context, modelID, userID, email string, retryPending bool) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	k := modelID + "|" + userID
	r := f.rows[k]
	if r == nil {
		f.rows[k] = &announceUser{ID: userID, Email: email, Status: "pending"}
		f.writes++
		return true, nil
	}
	if r.Status == "error" || (retryPending && r.Status == "pending") {
		r.Status = "pending"
		f.writes++
		return true, nil
	}
	return false, nil
}

func (f *fakeStore) Mark(ctx context.Context, modelID, userID, status, errMsg string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writes++
	f.rows[modelID+"|"+userID].Status = status
	f.errs[modelID+"|"+userID] = errMsg
	return nil
}

func (f *fakeStore) Status(ctx context.Context, modelID string) (AnnounceStatusRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	row := AnnounceStatusRow{Counts: map[string]int{}}
	for k, r := range f.rows {
		if strings.HasPrefix(k, modelID+"|") {
			row.Counts[r.Status]++
		}
	}
	return row, nil
}

func (f *fakeStore) Lock(ctx context.Context, modelID string) (func(), bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.locked[modelID] {
		return nil, false, nil
	}
	f.locked[modelID] = true
	return func() {
		f.mu.Lock()
		delete(f.locked, modelID)
		f.mu.Unlock()
	}, true, nil
}

func (f *fakeStore) writeCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.writes
}

type fakeSender struct {
	mu   sync.Mutex
	msgs []Message
	fail map[string]bool
	gate chan struct{}
}

func (s *fakeSender) send(m Message) error {
	if s.gate != nil {
		<-s.gate
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fail[strings.ToLower(m.To)] {
		return fmt.Errorf("smtp 550 rejected %s", m.To)
	}
	s.msgs = append(s.msgs, m)
	return nil
}

func (s *fakeSender) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.msgs)
}

func (s *fakeSender) tos() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []string
	for _, m := range s.msgs {
		out = append(out, m.To)
	}
	return out
}

var testModels = []model.ModelConfig{
	{ID: "acme-1", Provider: "acme", InputPricePer1M: 1.5, OutputPricePer1M: 6, ContextWindow: 1000000, SupportsTools: true, Aliases: []string{"acme-latest-x"}},
	{ID: "old-1", Deprecated: true},
}

func testUsers() []announceUser {
	return []announceUser{
		{ID: "1", Email: "a@real.com"},
		{ID: "2", Email: "b@real.com"},
		{ID: "3", Email: "c@real.com", Disabled: true},
		{ID: "4", Email: "d@real.com", Suppressed: true},
		{ID: "5", Email: "e2e-user@real.com"},
		{ID: "6", Email: "x@example.com"},
		{ID: "7", Email: "f@real.com", Status: "sent"},
		{ID: "8", Email: "G@real.com"},
		{ID: "9", Email: "g@real.com"},
		{ID: "10", Email: ""},
	}
}

func newTestAnnouncer(t *testing.T, st *fakeStore, s *fakeSender) *Announcer {
	t.Helper()
	a := newAnnouncer(context.Background(), st, s.send, filepath.Join("..", "..", "emails"), "secret", testModels)
	return a
}

func ptr[T any](v T) *T { return &v }

func waitDone(t *testing.T, a *Announcer, id string) *AnnounceRun {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		st, _ := a.Status(context.Background(), id)
		if st.Run != nil && st.Run.State != "running" {
			return st.Run
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("run did not finish")
	return nil
}

func TestSelectRecipients(t *testing.T) {
	sel, ex, eligible := selectRecipients(testUsers(), 0, false)
	want := Exclusions{Disabled: 1, Test: 2, Suppressed: 1, AlreadySent: 1, NoEmail: 1, Duplicate: 1}
	if ex != want {
		t.Fatalf("exclusions %+v want %+v", ex, want)
	}
	if eligible != 3 || len(sel) != 3 {
		t.Fatalf("eligible %d sel %d", eligible, len(sel))
	}
	sel, _, eligible = selectRecipients(testUsers(), 2, false)
	if len(sel) != 2 || eligible != 3 {
		t.Fatalf("limit: sel %d eligible %d", len(sel), eligible)
	}
	pend := []announceUser{{ID: "1", Email: "a@real.com", Status: "pending"}, {ID: "2", Email: "b@real.com", Status: "error"}}
	sel, ex, _ = selectRecipients(pend, 0, false)
	if len(sel) != 1 || sel[0].ID != "2" || ex.Pending != 1 {
		t.Fatalf("pending handling: %+v %+v", sel, ex)
	}
	if sel, _, _ = selectRecipients(pend, 0, true); len(sel) != 2 {
		t.Fatalf("retry_pending: %+v", sel)
	}
}

func TestCatalogCandidates(t *testing.T) {
	got := CatalogCandidates([]model.ModelConfig{
		{ID: "real-1", Aliases: []string{"real-alias", "openpaths/auto-x"}},
		{ID: "real-alias"},
		{ID: "Real-1"},
		{ID: "dead", Deprecated: true},
		{ID: "e2e-model"},
		{ID: "foo-test"},
		{ID: "latest-thing"},
		{ID: "openpaths/auto-code"},
		{ID: "auto"},
		{ID: "chat-latest"},
		{ID: "contest-model"},
		{ID: ""},
	})
	if strings.Join(got, ",") != "real-1,latest-thing,contest-model" {
		t.Fatalf("got %v", got)
	}
}

func TestCatalogCandidatesRealConfig(t *testing.T) {
	cfg, err := config.Load(filepath.Join("..", "..", "config.yaml"))
	if err != nil {
		t.Skipf("config: %v", err)
	}
	ids := CatalogCandidates(cfg.Models)
	if len(ids) < 20 {
		t.Fatalf("only %d candidates", len(ids))
	}
	for _, id := range ids {
		if virtualIDRe.MatchString(strings.ToLower(id)) || testIDRe.MatchString(strings.ToLower(id)) {
			t.Errorf("leaked %s", id)
		}
	}
}

func TestDetectNewModelsSeedsThenReportsNew(t *testing.T) {
	st := newFakeStore()
	a := newAnnouncer(context.Background(), st, (&fakeSender{}).send, "x", "s", []model.ModelConfig{{ID: "m1"}, {ID: "m2"}})
	added, err := a.DetectNewModels(context.Background())
	if err != nil || len(added) != 0 || st.catalog["m1"] == nil || st.catalog["m2"] == nil {
		t.Fatalf("seed: %v %v", added, err)
	}
	a.models = append(a.models, model.ModelConfig{ID: "m3"})
	added, _ = a.DetectNewModels(context.Background())
	if len(added) != 1 || added[0] != "m3" || st.catalog["m3"] != nil {
		t.Fatalf("new: %v", added)
	}
	if added, _ = a.DetectNewModels(context.Background()); len(added) != 0 {
		t.Fatalf("repeat: %v", added)
	}
}

func TestDryRunDefaultWritesAndSendsNothing(t *testing.T) {
	st, snd := newFakeStore(testUsers()...), &fakeSender{}
	a := newTestAnnouncer(t, st, snd)
	res, err := a.Announce(context.Background(), AnnounceRequest{ModelID: "acme-1"})
	if err != nil {
		t.Fatal(err)
	}
	if !res.DryRun || res.Started || st.writeCount() != 0 || snd.count() != 0 {
		t.Fatalf("dry run side effects: %+v writes=%d sent=%d", res, st.writeCount(), snd.count())
	}
	if res.Users != 10 || res.Recipients != 3 || res.Selected != 3 || len(res.Sample) != 3 {
		t.Fatalf("counts %+v", res)
	}
	if res.Excluded.Suppressed != 1 || res.Excluded.Disabled != 1 || res.Excluded.Test != 2 || res.Excluded.AlreadySent != 1 {
		t.Fatalf("exclusions %+v", res.Excluded)
	}
	p := res.Preview
	if res.Subject != "acme-1 is live on OpenPaths" || p == nil || p.Subject != res.Subject {
		t.Fatalf("subject %q", res.Subject)
	}
	for _, want := range []string{"acme-1", "$1.5/M in", "$6/M out", "1.0M ctx", "/unsubscribe?e=a%40real.com&t=" + UnsubscribeToken("secret", "a@real.com")} {
		if !strings.Contains(p.HTML, want) {
			t.Errorf("preview html missing %q", want)
		}
	}
	if strings.Contains(p.HTML, "{{.") || strings.Contains(p.Text, "<") || !strings.Contains(p.Text, "acme-1") {
		t.Fatal("bad preview render")
	}
}

func TestDryRunSampleCapped(t *testing.T) {
	var us []announceUser
	for i := 0; i < 50; i++ {
		us = append(us, announceUser{ID: fmt.Sprint(i), Email: fmt.Sprintf("u%d@real.com", i)})
	}
	a := newTestAnnouncer(t, newFakeStore(us...), &fakeSender{})
	res, _ := a.Announce(context.Background(), AnnounceRequest{ModelID: "acme-1", Limit: 20})
	if len(res.Sample) != announceSampleSize || res.Selected != 20 || res.Recipients != 50 {
		t.Fatalf("%d %d %d", len(res.Sample), res.Selected, res.Recipients)
	}
}

func TestRealRunRequiresConfirm(t *testing.T) {
	st, snd := newFakeStore(testUsers()...), &fakeSender{}
	a := newTestAnnouncer(t, st, snd)
	_, err := a.Announce(context.Background(), AnnounceRequest{ModelID: "acme-1", DryRun: ptr(false)})
	if ae, ok := err.(*AnnounceError); !ok || ae.Code != 400 || !strings.Contains(ae.Message, "confirm") {
		t.Fatalf("err %v", err)
	}
	_, err = a.Announce(context.Background(), AnnounceRequest{ModelID: "acme-1", Confirm: true})
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.Announce(context.Background(), AnnounceRequest{ModelID: "acme-1", DryRun: ptr(false), TestEmail: "me@x.com"})
	if err == nil {
		t.Fatal("test send without confirm accepted")
	}
	if st.writeCount() != 0 || snd.count() != 0 {
		t.Fatal("side effects without confirm")
	}
}

func TestRealRunSendsOnceRetriesErrorsAndMarksAnnounced(t *testing.T) {
	st, snd := newFakeStore(testUsers()...), &fakeSender{fail: map[string]bool{"b@real.com": true}}
	a := newTestAnnouncer(t, st, snd)
	req := AnnounceRequest{ModelID: "acme-1", DryRun: ptr(false), Confirm: true}

	res, err := a.Announce(context.Background(), req)
	if err != nil || !res.Started {
		t.Fatalf("%+v %v", res, err)
	}
	run := waitDone(t, a, "acme-1")
	if run.State != "done" || run.Sent != 2 || run.Failed != 1 || run.Total != 3 {
		t.Fatalf("run %+v", run)
	}
	if got := strings.Join(snd.tos(), ","); got != "a@real.com,G@real.com" {
		t.Fatalf("sent to %s", got)
	}
	if st.announced["acme-1"] {
		t.Fatal("marked announced despite failure")
	}
	m := snd.msgs[0]
	if m.UnsubscribeURL == "" || !strings.Contains(m.HTML, m.UnsubscribeURL) || m.FromName != "OpenPaths" {
		t.Fatalf("message %+v", m.UnsubscribeURL)
	}
	stt, _ := a.Status(context.Background(), "acme-1")
	if stt.Counts["sent"] != 2 || stt.Counts["error"] != 1 {
		t.Fatalf("counts %v", stt.Counts)
	}

	delete(snd.fail, "b@real.com")
	res, _ = a.Announce(context.Background(), req)
	if res.Selected != 1 || res.Retries != 1 || res.Excluded.AlreadySent != 3 {
		t.Fatalf("retry plan %+v", res)
	}
	run = waitDone(t, a, "acme-1")
	if run.Sent != 1 || run.Failed != 0 || snd.count() != 3 || !st.announced["acme-1"] {
		t.Fatalf("retry run %+v sent=%d announced=%v", run, snd.count(), st.announced)
	}

	res, _ = a.Announce(context.Background(), req)
	if res.Started || res.Selected != 0 || snd.count() != 3 {
		t.Fatalf("third run %+v", res)
	}
}

func TestRealRunLimitDoesNotMarkAnnounced(t *testing.T) {
	st, snd := newFakeStore(testUsers()...), &fakeSender{}
	a := newTestAnnouncer(t, st, snd)
	a.Announce(context.Background(), AnnounceRequest{ModelID: "acme-1", DryRun: ptr(false), Confirm: true, Limit: 1})
	waitDone(t, a, "acme-1")
	if snd.count() != 1 || st.announced["acme-1"] {
		t.Fatalf("sent %d announced %v", snd.count(), st.announced)
	}
}

func TestSingleFlight(t *testing.T) {
	st, snd := newFakeStore(testUsers()...), &fakeSender{gate: make(chan struct{})}
	a := newTestAnnouncer(t, st, snd)
	req := AnnounceRequest{ModelID: "acme-1", DryRun: ptr(false), Confirm: true}
	if _, err := a.Announce(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	_, err := a.Announce(context.Background(), req)
	if ae, ok := err.(*AnnounceError); !ok || ae.Code != 409 {
		t.Fatalf("second concurrent run: %v", err)
	}
	a2 := newTestAnnouncer(t, st, snd)
	_, err = a2.Announce(context.Background(), req)
	if ae, ok := err.(*AnnounceError); !ok || ae.Code != 409 {
		t.Fatalf("other instance (advisory lock): %v", err)
	}
	close(snd.gate)
	waitDone(t, a, "acme-1")
	if snd.count() != 3 {
		t.Fatalf("sent %d", snd.count())
	}
}

func TestAbortsAfterConsecutiveFailures(t *testing.T) {
	var us []announceUser
	fail := map[string]bool{}
	for i := 0; i < 30; i++ {
		e := fmt.Sprintf("u%d@real.com", i)
		us = append(us, announceUser{ID: fmt.Sprint(i), Email: e})
		fail[e] = true
	}
	snd := &fakeSender{fail: fail}
	a := newTestAnnouncer(t, newFakeStore(us...), snd)
	a.Announce(context.Background(), AnnounceRequest{ModelID: "acme-1", DryRun: ptr(false), Confirm: true})
	run := waitDone(t, a, "acme-1")
	if run.State != "aborted" || run.Failed != announceAbortAfter {
		t.Fatalf("%+v", run)
	}
}

func TestRateLimiterPaces(t *testing.T) {
	var us []announceUser
	for i := 0; i < 5; i++ {
		us = append(us, announceUser{ID: fmt.Sprint(i), Email: fmt.Sprintf("u%d@real.com", i)})
	}
	a := newTestAnnouncer(t, newFakeStore(us...), &fakeSender{})
	a.interval = 20 * time.Millisecond
	start := time.Now()
	a.Announce(context.Background(), AnnounceRequest{ModelID: "acme-1", DryRun: ptr(false), Confirm: true})
	waitDone(t, a, "acme-1")
	if d := time.Since(start); d < 90*time.Millisecond {
		t.Fatalf("5 sends in %v, not paced", d)
	}
}

func TestTestEmail(t *testing.T) {
	st, snd := newFakeStore(testUsers()...), &fakeSender{}
	a := newTestAnnouncer(t, st, snd)
	res, err := a.Announce(context.Background(), AnnounceRequest{ModelID: "acme-1", TestEmail: "me@x.com"})
	if err != nil || res.TestSent != "" || snd.count() != 0 || res.Preview == nil {
		t.Fatalf("dry test %+v %v", res, err)
	}
	res, err = a.Announce(context.Background(), AnnounceRequest{ModelID: "acme-1", TestEmail: "me@x.com", DryRun: ptr(false), Confirm: true})
	if err != nil || res.TestSent != "me@x.com" || snd.count() != 1 {
		t.Fatalf("real test %+v %v", res, err)
	}
	if !strings.HasPrefix(snd.msgs[0].Subject, "[TEST] ") || st.writeCount() != 0 {
		t.Fatal("test send must be tagged and write nothing")
	}
	if _, err = a.Announce(context.Background(), AnnounceRequest{ModelID: "acme-1", TestEmail: "bad\r\nBcc: x"}); err == nil {
		t.Fatal("bad test_email accepted")
	}
}

func TestAnnounceValidation(t *testing.T) {
	a := newTestAnnouncer(t, newFakeStore(), &fakeSender{})
	for _, tc := range []struct {
		req  AnnounceRequest
		code int
	}{
		{AnnounceRequest{}, 400},
		{AnnounceRequest{ModelID: "nope"}, 404},
		{AnnounceRequest{ModelID: "old-1"}, 400},
		{AnnounceRequest{ModelID: "acme-1", Template: "../drip_config.json"}, 400},
		{AnnounceRequest{ModelID: "acme-1", Template: "missing.html"}, 400},
		{AnnounceRequest{ModelID: "acme-1", Limit: -1}, 400},
	} {
		_, err := a.Announce(context.Background(), tc.req)
		if ae, ok := err.(*AnnounceError); !ok || ae.Code != tc.code {
			t.Errorf("%+v: %v", tc.req, err)
		}
	}
	res, err := a.Announce(context.Background(), AnnounceRequest{ModelID: "acme-latest-x", Subject: "Hi\r\nBcc: x@y.com"})
	if err != nil || res.ModelID != "acme-1" || strings.ContainsAny(res.Subject, "\r\n") {
		t.Fatalf("alias/subject: %+v %v", res, err)
	}
}

func TestReadyGateBlocksRealRun(t *testing.T) {
	st := newFakeStore(testUsers()...)
	a := newTestAnnouncer(t, st, &fakeSender{})
	a.ready = func() error { return fmt.Errorf("no creds") }
	_, err := a.Announce(context.Background(), AnnounceRequest{ModelID: "acme-1", DryRun: ptr(false), Confirm: true})
	if ae, ok := err.(*AnnounceError); !ok || ae.Code != 503 || st.writeCount() != 0 {
		t.Fatalf("%v", err)
	}
	if _, err := a.Announce(context.Background(), AnnounceRequest{ModelID: "acme-1"}); err != nil {
		t.Fatalf("dry-run must not need creds: %v", err)
	}
}

func TestCustomTemplateWithModelCards(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "t.html"), []byte(`<p>{{.Model.ID}} {{.Model.Price}} {{.Model.Context}}</p>{{.ModelCards.x}}<a href="{{.UnsubscribeURL}}">u</a>`), 0o644)
	os.WriteFile(filepath.Join(dir, "model_notes.json"), []byte(`{"sections":{"x":[{"id":"acme-1","name":"Acme One","note":"nice"}]}}`), 0o644)
	a := newAnnouncer(context.Background(), newFakeStore(testUsers()...), (&fakeSender{}).send, dir, "s", testModels)
	res, err := a.Announce(context.Background(), AnnounceRequest{ModelID: "acme-1", Template: "t.html"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Preview.HTML, "Acme One") || !strings.Contains(res.Preview.HTML, "$1.5/M in") || res.Subject != "Acme One is live on OpenPaths" {
		t.Fatalf("%s | %s", res.Subject, res.Preview.HTML)
	}
	os.WriteFile(filepath.Join(dir, "bad.html"), []byte(`{{.Nope}}`), 0o644)
	if _, err := a.Announce(context.Background(), AnnounceRequest{ModelID: "acme-1", Template: "bad.html"}); err == nil {
		t.Fatal("unreplaced placeholder accepted")
	}
}
