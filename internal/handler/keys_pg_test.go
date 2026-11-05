package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	fasthttprouter "github.com/fasthttp/router"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/auth"
	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/middleware"
	"github.com/openpaths/openpaths/internal/model"
)

type keysEnv struct {
	t       *testing.T
	router  *fasthttprouter.Router
	q       *queries.APIKeyQueries
	pool    *pgxpool.Pool
	user    string
	session string
}

func newKeysEnv(t *testing.T) *keysEnv {
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
	schema := fmt.Sprintf("keys_h_%d", time.Now().UnixNano())
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
	for _, f := range []string{"001_initial.sql", "028_guardrails.sql", "037_api_key_scopes.sql"} {
		mig, err := os.ReadFile(filepath.Join("..", "db", "migrations", f))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, strings.ReplaceAll(string(mig), `CREATE EXTENSION IF NOT EXISTS "pgcrypto";`, "")); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
	}

	e := &keysEnv{t: t, pool: pool, q: queries.NewAPIKeyQueries(pool)}
	e.user = e.newUser("owner@example.com")
	jwt := auth.NewJWTService("test-secret", 1)
	e.session, _ = jwt.Generate(e.user, "owner@example.com")

	h := NewAccountHandler(e.q, nil, nil, nil)
	account := middleware.Chain(middleware.DashboardAuth(e.q, jwt))
	keyed := middleware.Chain(middleware.APIKeyAuth(e.q))
	scoped := func(chain middleware.Middleware, scope string) middleware.Middleware {
		return func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
			return middleware.RequireScope(scope, chain(next))
		}
	}
	ok := func(ctx *fasthttp.RequestCtx) { ctx.SetStatusCode(200) }
	r := fasthttprouter.New()
	r.GET("/v1/keys/scopes", h.HandleKeyScopes)
	r.GET("/v1/keys/me", scoped(account, model.ScopeNone)(h.HandleKeyMe))
	r.GET("/v1/keys", scoped(account, model.ScopeKeysManage)(h.HandleListAPIKeys))
	r.POST("/v1/keys", scoped(account, model.ScopeKeysManage)(h.HandleCreateAPIKey))
	r.POST("/v1/keys/{id}/rotate", scoped(account, model.ScopeKeysManage)(h.HandleRotateAPIKey))
	r.DELETE("/v1/keys/{id}", scoped(account, model.ScopeKeysManage)(h.HandleRevokeAPIKey))
	r.POST("/v1/chat/completions", scoped(keyed, model.ScopeChat)(ok))
	r.POST("/v1/images/generations", scoped(keyed, model.ScopeImages)(ok))
	r.POST("/v1/unscoped", keyed(ok))
	e.router = r
	return e
}

func (e *keysEnv) newUser(email string) string {
	var id string
	if err := e.pool.QueryRow(context.Background(), `INSERT INTO users (email, password_hash) VALUES ($1, 'x') RETURNING id::text`, email).Scan(&id); err != nil {
		e.t.Fatal(err)
	}
	return id
}

type keysResp struct {
	status    int
	challenge string
	raw       string
	body      map[string]any
}

func (e *keysEnv) call(method, path string, headers map[string]string, body string) *keysResp {
	var req fasthttp.Request
	req.Header.SetMethod(method)
	req.SetRequestURI(path)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	req.SetBodyString(body)
	var ctx fasthttp.RequestCtx
	ctx.Init(&req, nil, nil)
	e.router.Handler(&ctx)
	r := &keysResp{status: ctx.Response.StatusCode(), raw: string(ctx.Response.Body()), challenge: string(ctx.Response.Header.Peek("WWW-Authenticate"))}
	json.Unmarshal(ctx.Response.Body(), &r.body)
	return r
}

func bearer(secret string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + secret}
}

func (e *keysEnv) mint(scopes ...string) (id, secret string) {
	e.t.Helper()
	b, _ := json.Marshal(map[string]any{"name": "t", "scopes": scopes})
	r := e.call("POST", "/v1/keys", bearer(e.session), string(b))
	if r.status != 201 {
		e.t.Fatalf("mint %v: %d %s", scopes, r.status, r.raw)
	}
	return r.body["id"].(string), r.body["key"].(string)
}

func errField(r *keysResp, field string) any {
	m, _ := r.body["error"].(map[string]any)
	return m[field]
}

func TestKeysScopeAllowDeny(t *testing.T) {
	e := newKeysEnv(t)
	_, chatKey := e.mint(model.ScopeChat)

	if r := e.call("POST", "/v1/chat/completions", bearer(chatKey), ""); r.status != 200 {
		t.Fatalf("bearer: %d %s", r.status, r.raw)
	}
	if r := e.call("POST", "/v1/chat/completions", map[string]string{"X-API-Key": chatKey}, ""); r.status != 200 {
		t.Fatalf("x-api-key: %d %s", r.status, r.raw)
	}
	r := e.call("POST", "/v1/images/generations", bearer(chatKey), "")
	if r.status != 403 || errField(r, "code") != "insufficient_scope" || errField(r, "required_scope") != model.ScopeImages || errField(r, "hint") == "" {
		t.Fatalf("deny: %d %s", r.status, r.raw)
	}
	if r.challenge != `Bearer error="insufficient_scope", scope="images"` {
		t.Fatalf("challenge %q", r.challenge)
	}
	if r := e.call("POST", "/v1/unscoped", bearer(chatKey), ""); r.status != 403 {
		t.Fatalf("undeclared route: %d", r.status)
	}
	r = e.call("POST", "/v1/chat/completions", nil, "")
	if r.status != 401 || r.challenge != "Bearer" || errField(r, "code") != "missing_api_key" {
		t.Fatalf("missing key: %d %s", r.status, r.raw)
	}
	if r := e.call("POST", "/v1/chat/completions", bearer("sk-op-nope"), ""); r.status != 401 || errField(r, "code") != "invalid_api_key" {
		t.Fatalf("bad key: %d %s", r.status, r.raw)
	}
}

func TestKeysManagementOnly(t *testing.T) {
	e := newKeysEnv(t)
	chatID, chatKey := e.mint(model.ScopeChat, model.ScopeImages)
	mgmtID, mgmtKey := e.mint(model.ScopeKeysManage)

	for _, c := range []struct{ method, path, body string }{
		{"GET", "/v1/keys", ""},
		{"POST", "/v1/keys", `{"name":"x","scopes":["chat.completions"]}`},
		{"POST", "/v1/keys/" + chatID + "/rotate", ""},
		{"DELETE", "/v1/keys/" + chatID, ""},
	} {
		r := e.call(c.method, c.path, bearer(chatKey), c.body)
		if r.status != 403 || errField(r, "required_scope") != model.ScopeKeysManage {
			t.Fatalf("%s %s with ordinary key: %d %s", c.method, c.path, r.status, r.raw)
		}
	}
	if r := e.call("POST", "/v1/chat/completions", bearer(mgmtKey), ""); r.status != 403 {
		t.Fatalf("management key got usage scope: %d", r.status)
	}

	r := e.call("GET", "/v1/keys", bearer(mgmtKey), "")
	if r.status != 200 || len(r.body["keys"].([]any)) != 2 {
		t.Fatalf("list: %d %s", r.status, r.raw)
	}
	for _, secret := range []string{chatKey, mgmtKey, "key_hash", auth.HashAPIKey(chatKey)} {
		if strings.Contains(r.raw, secret) {
			t.Fatalf("list leaks %q", secret)
		}
	}

	r = e.call("GET", "/v1/keys/me", bearer(mgmtKey), "")
	if r.status != 200 || r.body["is_management"] != true || r.body["id"] != mgmtID {
		t.Fatalf("me mgmt: %s", r.raw)
	}
	r = e.call("GET", "/v1/keys/me", bearer(chatKey), "")
	if r.status != 200 || r.body["is_management"] != false || len(r.body["scopes"].([]any)) != 2 {
		t.Fatalf("me ordinary: %s", r.raw)
	}
	r = e.call("GET", "/v1/keys/me", bearer(e.session), "")
	if r.status != 200 || r.body["is_management"] != true {
		t.Fatalf("me session: %s", r.raw)
	}
	r = e.call("GET", "/v1/keys/scopes", nil, "")
	if r.status != 200 || len(r.body["scopes"].([]any)) != len(model.ScopeCatalog()) {
		t.Fatalf("scopes: %s", r.raw)
	}

	r = e.call("POST", "/v1/keys", bearer(mgmtKey), `{"name":"made-by-mgmt","scopes":["embeddings"],"expires_in_days":7}`)
	if r.status != 201 || r.body["expires_at"] == nil || !strings.HasPrefix(r.body["key"].(string), auth.APIKeyPrefix) {
		t.Fatalf("mgmt create: %d %s", r.status, r.raw)
	}
	for _, bad := range []string{`{"scopes":["root"]}`, `{"scopes":[]}`, `{"expires_in_days":0}`, `not json`} {
		if r := e.call("POST", "/v1/keys", bearer(mgmtKey), bad); r.status != 400 {
			t.Fatalf("create %s: %d", bad, r.status)
		}
	}

	if r := e.call("DELETE", "/v1/keys/"+chatID, bearer(mgmtKey), ""); r.status != 200 {
		t.Fatalf("revoke: %d %s", r.status, r.raw)
	}
	r = e.call("POST", "/v1/chat/completions", bearer(chatKey), "")
	if r.status != 401 || errField(r, "code") != "revoked_api_key" {
		t.Fatalf("revoked key: %d %s", r.status, r.raw)
	}
	if r := e.call("DELETE", "/v1/keys/"+chatID, bearer(mgmtKey), ""); r.status != 404 {
		t.Fatalf("revoke twice: %d", r.status)
	}
	if r := e.call("DELETE", "/v1/keys/not-a-uuid", bearer(mgmtKey), ""); r.status != 404 {
		t.Fatalf("bad id: %d", r.status)
	}

	other := e.newUser("other@example.com")
	oh, oraw := auth.HashAPIKey("sk-op-other"), "sk-op-other"
	ok, _ := e.q.CreateScoped(context.Background(), other, oh, oraw[:8], "theirs", []string{model.ScopeChat}, nil)
	if r := e.call("DELETE", "/v1/keys/"+ok.ID, bearer(mgmtKey), ""); r.status != 404 {
		t.Fatalf("cross-user revoke: %d", r.status)
	}
}

func TestKeysRotateGrace(t *testing.T) {
	e := newKeysEnv(t)
	_, mgmtKey := e.mint(model.ScopeKeysManage)
	id, secret := e.mint(model.ScopeChat)

	if r := e.call("POST", "/v1/keys/"+id+"/rotate", bearer(mgmtKey), `{"grace_seconds":86401}`); r.status != 400 {
		t.Fatalf("grace cap: %d", r.status)
	}
	r := e.call("POST", "/v1/keys/"+id+"/rotate", bearer(mgmtKey), `{"grace_seconds":3600}`)
	if r.status != 200 || r.body["rotated_from"] != id || r.body["old_key"].(map[string]any)["revoked_at"] == nil {
		t.Fatalf("rotate: %d %s", r.status, r.raw)
	}
	newSecret := r.body["key"].(string)
	if newSecret == secret || r.body["secret"] != newSecret {
		t.Fatal("rotation did not mint a new secret")
	}
	if r := e.call("POST", "/v1/chat/completions", bearer(secret), ""); r.status != 200 {
		t.Fatalf("old key inside grace: %d", r.status)
	}
	if r := e.call("POST", "/v1/chat/completions", bearer(newSecret), ""); r.status != 200 {
		t.Fatalf("new key: %d", r.status)
	}
	if r := e.call("POST", "/v1/keys/"+id+"/rotate", bearer(mgmtKey), ""); r.status != 409 {
		t.Fatalf("rotate already-rotated: %d", r.status)
	}

	var rotated string
	e.pool.QueryRow(context.Background(), `SELECT id::text FROM api_keys WHERE rotated_from = $1`, id).Scan(&rotated)
	r = e.call("POST", "/v1/keys/"+rotated+"/rotate", bearer(mgmtKey), "")
	if r.status != 200 {
		t.Fatalf("immediate rotate: %d %s", r.status, r.raw)
	}
	if r := e.call("POST", "/v1/chat/completions", bearer(newSecret), ""); r.status != 401 {
		t.Fatalf("old key after immediate rotate: %d", r.status)
	}
}

func TestKeysExpiryAndLegacy(t *testing.T) {
	e := newKeysEnv(t)
	ctx := context.Background()

	past := time.Now().Add(-time.Hour)
	e.q.CreateScoped(ctx, e.user, auth.HashAPIKey("sk-op-expired"), "sk-op-exp", "e", []string{model.ScopeChat}, &past)
	r := e.call("POST", "/v1/chat/completions", bearer("sk-op-expired"), "")
	if r.status != 401 || errField(r, "code") != "expired_api_key" {
		t.Fatalf("expired: %d %s", r.status, r.raw)
	}

	e.pool.Exec(ctx, `INSERT INTO api_keys (user_id, key_hash, key_prefix) VALUES ($1, $2, 'sk-op-leg')`, e.user, auth.HashAPIKey("sk-op-legacy"))
	if r := e.call("POST", "/v1/images/generations", bearer("sk-op-legacy"), ""); r.status != 200 {
		t.Fatalf("legacy key on usage route: %d %s", r.status, r.raw)
	}
	if r := e.call("GET", "/v1/keys", bearer("sk-op-legacy"), ""); r.status != 403 {
		t.Fatalf("legacy key on /keys: %d", r.status)
	}
	r = e.call("GET", "/v1/keys/me", bearer("sk-op-legacy"), "")
	if r.status != 200 || r.body["is_management"] != false {
		t.Fatalf("legacy me: %s", r.raw)
	}
}
