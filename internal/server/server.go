package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	fasthttprouter "github.com/fasthttp/router"
	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/audio"
	"github.com/openpaths/openpaths/internal/auth"
	"github.com/openpaths/openpaths/internal/billing"
	"github.com/openpaths/openpaths/internal/config"
	"github.com/openpaths/openpaths/internal/crypto"
	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/discovery"
	"github.com/openpaths/openpaths/internal/handler"
	"github.com/openpaths/openpaths/internal/metrics"
	"github.com/openpaths/openpaths/internal/middleware"
	"github.com/openpaths/openpaths/internal/provider"
	"github.com/openpaths/openpaths/internal/router"
	"github.com/openpaths/openpaths/internal/storage"
	stripesvc "github.com/openpaths/openpaths/internal/stripe"
)

type Server struct {
	httpServer *fasthttp.Server
	port       int
}

type Dependencies struct {
	Config           *config.Config
	Router           *router.Router
	Billing          *billing.Engine
	Recorder         *metrics.Recorder
	JWTService       *auth.JWTService
	UserQ            *queries.UserQueries
	APIKeyQ          *queries.APIKeyQueries
	CreditQ          *queries.CreditQueries
	StripeDepositQ   *queries.StripeDepositQueries
	StripeReconciler *billing.Reconciler
	StatsQ           *queries.StatsQueries
	Transcribers     []provider.TranscriptionProvider
	Embedders        []provider.EmbeddingProvider
	AutoEmotion      *audio.AutoEmotion
	CryptoSvc        *crypto.Service
	Storage          storage.Store
	StripeSvc        *stripesvc.Service
	Discovery        *discovery.Service
	ModelMetaQ       *queries.ModelMetadataQueries
	FineTuneQ        *queries.FineTuneQueries
	FineTuneProvs    map[string]provider.FineTuneProvider
	ProviderKeyQ     *queries.ProviderKeyQueries
	OnRegister       handler.OnRegisterFunc
}

func New(deps *Dependencies) *Server {
	r := fasthttprouter.New()

	chatH := handler.NewChatHandler(deps.Router, deps.Billing, deps.Recorder, deps.UserQ, deps.ProviderKeyQ)
	modelsH := handler.NewModelsHandler(deps.Router)
	authH := handler.NewAuthHandler(deps.UserQ, deps.CreditQ, deps.APIKeyQ)
	if deps.OnRegister != nil {
		authH.SetOnRegister(deps.OnRegister)
	}
	accountH := handler.NewAccountHandler(deps.APIKeyQ, deps.CreditQ, deps.Billing, deps.StripeReconciler)
	creditsH := handler.NewCreditsHandler(deps.Billing)
	statsH := handler.NewStatsHandler(deps.StatsQ)
	acctStatsH := handler.NewAccountStatsHandler(deps.StatsQ)
	adminH := handler.NewAdminHandler(deps.UserQ)

	apiKeyChain := middleware.Chain(
		middleware.Recovery(),
		middleware.Logging(),
		middleware.APIKeyAuth(deps.APIKeyQ),
		middleware.BYOKLoader(deps.ProviderKeyQ),
		middleware.RateLimit(),
		middleware.BalanceCheck(deps.Billing),
	)

	searchChain := middleware.Chain(
		middleware.Recovery(),
		middleware.Logging(),
		middleware.APIKeyAuth(deps.APIKeyQ),
		middleware.BYOKLoader(deps.ProviderKeyQ),
		middleware.RateLimit(),
	)

	// accountChain: API key auth only — no balance check for account management
	accountChain := middleware.Chain(
		middleware.Recovery(),
		middleware.Logging(),
		middleware.APIKeyAuth(deps.APIKeyQ),
	)

	publicChain := middleware.Chain(
		middleware.Recovery(),
		middleware.Logging(),
	)

	r.POST("/v1/chat/completions", apiKeyChain(chatH.HandleChatCompletion))
	r.GET("/v1/models", apiKeyChain(modelsH.HandleListModels))
	r.GET("/v1/models/{model_id}", apiKeyChain(modelsH.HandleGetModel))

	searchH := handler.NewSearchHandler(
		handler.ExaSearchProviderConfig(deps.Config.Providers),
		handler.PapersSearchProviderConfig(deps.Config.Providers),
		deps.Billing,
		deps.Recorder,
	)
	r.POST("/v1/search", searchChain(searchH.HandleSearch))
	handler.LogExaSearchPricing()
	log.Printf("Search endpoint enabled at /v1/search")

	anthH := handler.NewAnthropicHandler(deps.Router, deps.Billing, deps.Recorder)
	r.POST("/v1/messages", apiKeyChain(anthH.HandleMessages))
	log.Printf("Anthropic-compatible /v1/messages endpoint enabled")

	imageH := handler.NewImageHandler(deps.Router, deps.Billing, deps.Recorder)
	imageH.SetStorage(deps.Storage)
	r.POST("/v1/images/generations", apiKeyChain(imageH.HandleImageGeneration))
	r.POST("/v1/images/edits", apiKeyChain(imageH.HandleImageGeneration))
	log.Printf("Image generation endpoint enabled")

	videoH := handler.NewVideoHandler(deps.Router, deps.Billing, deps.Recorder)
	r.POST("/v1/videos/generations", apiKeyChain(videoH.HandleVideoGeneration))
	log.Printf("Video generation endpoint enabled")

	musicH := handler.NewMusicHandler(deps.Router, deps.Billing, deps.Recorder)
	r.POST("/v1/music/generations", apiKeyChain(musicH.HandleMusicGeneration))
	log.Printf("Music generation endpoint enabled")

	speechH := handler.NewSpeechHandler(deps.Router, deps.Billing, deps.Recorder)
	speechH.SetAutoEmotion(deps.AutoEmotion)
	r.POST("/v1/audio/speech", apiKeyChain(speechH.HandleSpeechGeneration))
	r.POST("/v1/tts", apiKeyChain(speechH.HandleSpeechGeneration))
	log.Printf("Speech generation endpoint enabled")

	embeddingH := handler.NewEmbeddingHandler(deps.Router, deps.Billing, deps.Recorder, deps.Embedders)
	r.POST("/v1/embeddings", apiKeyChain(embeddingH.HandleEmbedding))
	log.Printf("Embedding endpoint enabled (%d fallback providers)", len(deps.Embedders))

	if len(deps.Transcribers) > 0 {
		transcriptionH := handler.NewTranscriptionHandler(deps.Router, deps.Billing, deps.Transcribers, deps.Recorder)
		r.POST("/v1/audio/transcriptions", apiKeyChain(transcriptionH.HandleTranscription))
		r.POST("/v1/stt", apiKeyChain(transcriptionH.HandleTranscription))
		log.Printf("Transcription endpoint enabled (%d providers)", len(deps.Transcribers))
	}

	if deps.Storage != nil {
		uploadH := handler.NewUploadHandler(deps.Storage)
		r.POST("/v1/files/upload", apiKeyChain(uploadH.HandleUpload))
		log.Printf("File upload endpoint enabled")
	}

	r.ServeFiles("/uploads/{filepath:*}", deps.Config.Storage.LocalDir)

	r.POST("/auth/register", publicChain(authH.HandleRegister))
	r.POST("/auth/login", publicChain(authH.HandleLogin))
	r.POST("/auth/logout", publicChain(authH.HandleLogout))

	r.GET("/account/keys", accountChain(accountH.HandleListAPIKeys))
	r.POST("/account/keys", accountChain(accountH.HandleCreateAPIKey))
	r.DELETE("/account/keys/{id}", accountChain(accountH.HandleRevokeAPIKey))
	r.GET("/account/balance", accountChain(accountH.HandleGetBalance))
	r.GET("/account/transactions", accountChain(accountH.HandleGetTransactions))
	r.POST("/account/credits/add", accountChain(creditsH.HandleAddCredits))

	if deps.ProviderKeyQ != nil {
		pkH := handler.NewProviderKeysHandler(deps.ProviderKeyQ)
		r.GET("/account/provider-keys", accountChain(pkH.HandleList))
		r.POST("/account/provider-keys", accountChain(pkH.HandleUpsert))
		r.POST("/account/provider-keys/bulk", accountChain(pkH.HandleBulkUpsert))
		r.DELETE("/account/provider-keys", accountChain(pkH.HandleDelete))
		log.Printf("BYOK provider keys endpoints enabled")
	}

	if deps.CryptoSvc != nil {
		cryptoH := handler.NewCryptoHandler(deps.CryptoSvc)
		r.POST("/crypto/checkout", accountChain(cryptoH.HandleCreateCheckout))
		r.GET("/crypto/checkout/{id}", publicChain(cryptoH.HandleGetCheckout))
		r.GET("/crypto/checkout/{id}/events", publicChain(cryptoH.HandleCheckoutEvents))
		r.GET("/crypto/prices", publicChain(cryptoH.HandlePrices))
		log.Printf("Crypto payment endpoints enabled")
	}

	if deps.StripeSvc != nil {
		atH := handler.NewAutotopupHandler(deps.UserQ, deps.StripeSvc)
		r.POST("/account/stripe/setup", accountChain(atH.HandleStripeSetup))
		r.POST("/account/stripe/confirm", accountChain(atH.HandleStripeConfirm))
		r.GET("/account/stripe/payment-methods", accountChain(atH.HandleListPaymentMethods))
		r.DELETE("/account/stripe/payment-methods/{id}", accountChain(atH.HandleDeletePaymentMethod))
		r.POST("/account/autotopup/settings", accountChain(atH.HandleUpdateAutotopupSettings))
		r.GET("/account/autotopup/settings", accountChain(atH.HandleGetAutotopupSettings))

		checkoutH := handler.NewCheckoutHandler(deps.StripeSvc, deps.UserQ, deps.Billing, deps.StripeDepositQ, deps.Config.Stripe.CreditsPriceID, deps.Config.Stripe.WebhookSecret)
		r.POST("/account/stripe/checkout", accountChain(checkoutH.HandleCreateCheckout))
		r.GET("/account/stripe/config", publicChain(checkoutH.HandleStripeConfig))
		r.POST("/stripe/webhooks", publicChain(checkoutH.HandleWebhook))

		log.Printf("Stripe checkout + auto-topup endpoints enabled (price: %s)", deps.Config.Stripe.CreditsPriceID)
	}

	r.GET("/stats/models", publicChain(statsH.HandleModelStats))
	r.GET("/stats/providers", publicChain(statsH.HandleProviderStats))
	r.GET("/stats/breakdown", publicChain(statsH.HandleUsageBreakdown))
	r.GET("/stats/timeseries", publicChain(statsH.HandleTimeSeries))

	r.GET("/account/stats/timeseries", accountChain(acctStatsH.HandleUserTimeSeries))
	r.GET("/account/stats/by-api-key", accountChain(acctStatsH.HandleUserSpendByAPIKey))
	r.GET("/account/stats/by-provider", accountChain(acctStatsH.HandleUserSpendByProvider))
	r.GET("/account/stats/by-api-key/{key_id}/models", accountChain(acctStatsH.HandleUserAPIKeyDrilldown))
	r.GET("/account/stats/by-provider/{provider}/models", accountChain(acctStatsH.HandleUserProviderDrilldown))
	r.GET("/admin/users/spend", accountChain(adminH.HandleUserSpend))

	if deps.FineTuneQ != nil && len(deps.FineTuneProvs) > 0 {
		ftH := handler.NewFineTuneHandler(deps.FineTuneQ, deps.FineTuneProvs, deps.Storage)
		r.POST("/v1/files", apiKeyChain(ftH.HandleUploadFile))
		r.GET("/v1/files", apiKeyChain(ftH.HandleListFiles))
		r.POST("/v1/fine_tuning/jobs", apiKeyChain(ftH.HandleCreateJob))
		r.GET("/v1/fine_tuning/jobs", apiKeyChain(ftH.HandleListJobs))
		r.GET("/v1/fine_tuning/jobs/{job_id}", apiKeyChain(ftH.HandleGetJob))
		r.POST("/v1/fine_tuning/jobs/{job_id}/cancel", apiKeyChain(ftH.HandleCancelJob))
		r.GET("/v1/fine_tuning/jobs/{job_id}/events", apiKeyChain(ftH.HandleListEvents))
		log.Printf("Fine-tuning endpoints enabled (%d providers)", len(deps.FineTuneProvs))
	}

	if deps.Discovery != nil {
		disc := deps.Discovery
		metaQ := deps.ModelMetaQ
		r.POST("/admin/discovery/run", accountChain(adminH.RequireAdmin(func(ctx *fasthttp.RequestCtx) {
			n, err := disc.DiscoverAll(ctx)
			if err != nil {
				handler.WriteJSONPublic(ctx, 500, map[string]any{"error": err.Error()})
				return
			}
			handler.WriteJSONPublic(ctx, 200, map[string]any{"indexed": n})
		})))
		r.GET("/admin/discovery/models", accountChain(adminH.RequireAdmin(func(ctx *fasthttp.RequestCtx) {
			prov := string(ctx.QueryArgs().Peek("provider"))
			var models []*queries.ModelMetadata
			var err error
			if prov != "" {
				models, err = metaQ.ListByProvider(ctx, prov)
			} else {
				models, err = metaQ.ListAll(ctx)
			}
			if err != nil {
				handler.WriteJSONPublic(ctx, 500, map[string]any{"error": err.Error()})
				return
			}
			handler.WriteJSONPublic(ctx, 200, map[string]any{"object": "list", "data": models, "count": len(models)})
		})))
		log.Printf("Model discovery endpoints enabled")
	}

	// OpenRouter provider endpoint - public, no auth required
	// This lets OpenRouter discover our models and route traffic to us
	orProvH := handler.NewOpenRouterProviderHandler(deps.Router)
	r.GET("/openrouter/models", publicChain(orProvH.HandleListModels))
	log.Printf("OpenRouter provider models endpoint enabled at /openrouter/models")

	r.GET("/health", func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(200)
		ctx.SetBodyString(`{"status":"ok"}`)
	})
	r.POST("/monitoring/frontend-errors", publicChain(handleFrontendErrorReport))

	r.GET("/sitemap.xml", func(ctx *fasthttp.RequestCtx) {
		if staticDir := deps.Config.Server.StaticDir; staticDir != "" {
			if data, err := os.ReadFile(filepath.Join(staticDir, "sitemap.xml")); err == nil {
				ctx.SetContentType("application/xml; charset=utf-8")
				ctx.SetStatusCode(200)
				ctx.SetBody(data)
				return
			}
		}

		const base = "https://openpaths.io"
		pages := []struct{ loc, priority, freq string }{
			{"/", "1.0", "daily"},
			{"/pricing", "0.9", "weekly"},
			{"/models", "0.9", "weekly"},
			{"/providers", "0.8", "weekly"},
			{"/stats", "0.7", "daily"},
			{"/docs", "0.9", "weekly"},
			{"/integrations", "0.9", "weekly"},
			{"/playground", "0.7", "monthly"},
			{"/search", "0.7", "monthly"},
			{"/blog", "0.8", "weekly"},
		}
		blogSlugs := []string{
			"openpaths-agent-integrations-hermes-openclaw",
			"openpaths-sdk-integrations",
			"how-openpaths-is-hosted-on-codex-infinity",
			"switch-to-openpaths-in-2-lines",
			"state-of-ai-models-march-2026",
			"how-auto-models-work",
			"ai-art-generation-compared",
			"choosing-the-right-llm",
			"image-resolution-handling",
			"video-generation-guide",
			"openpaths-vs-openrouter",
			"music-and-speech-models",
			"free-ai-models",
			"provider-openai",
			"provider-anthropic",
			"provider-google",
			"provider-xai",
			"provider-deepseek",
			"provider-mistral",
			"provider-groq",
			"provider-minimax",
			"provider-together",
			"provider-zai",
			"provider-openrouter",
			"provider-netwrck",
			"provider-text-generator",
			"provider-fal",
		}
		var b strings.Builder
		b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
		b.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
		for _, p := range pages {
			fmt.Fprintf(&b, "  <url>\n    <loc>%s%s</loc>\n    <changefreq>%s</changefreq>\n    <priority>%s</priority>\n  </url>\n", base, p.loc, p.freq, p.priority)
		}
		for _, slug := range blogSlugs {
			fmt.Fprintf(&b, "  <url>\n    <loc>%s/blog/%s</loc>\n    <changefreq>monthly</changefreq>\n    <priority>0.7</priority>\n  </url>\n", base, slug)
		}
		b.WriteString("</urlset>\n")
		ctx.SetContentType("text/xml; charset=utf-8")
		ctx.SetStatusCode(200)
		ctx.SetBodyString(b.String())
	})

	handler := r.Handler
	if staticDir := deps.Config.Server.StaticDir; staticDir != "" {
		handler = spaHandler(staticDir, r.Handler, deps.APIKeyQ, deps.UserQ)
		log.Printf("Serving frontend from %s", staticDir)
	}

	srv := &fasthttp.Server{
		Handler:                       handler,
		ReadTimeout:                   time.Duration(deps.Config.Server.ReadTimeout) * time.Second,
		WriteTimeout:                  time.Duration(deps.Config.Server.WriteTimeout) * time.Second,
		MaxRequestBodySize:            deps.Config.Server.MaxRequestBody * 1024 * 1024,
		DisableHeaderNamesNormalizing: true,
		TCPKeepalive:                  true,
		ReduceMemoryUsage:             false,
		NoDefaultServerHeader:         true,
		NoDefaultDate:                 true,
		NoDefaultContentType:          true,
	}

	return &Server{
		httpServer: srv,
		port:       deps.Config.Server.Port,
	}
}

func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("OpenPath server starting on %s", addr)
	return s.httpServer.ListenAndServe(addr)
}

func (s *Server) Shutdown() error {
	return s.httpServer.Shutdown()
}

func spaHandler(dir string, api fasthttp.RequestHandler, apiKeyQ *queries.APIKeyQueries, userQ *queries.UserQueries) fasthttp.RequestHandler {
	fs := &fasthttp.FS{
		Root:       dir,
		IndexNames: []string{"index.html"},
		Compress:   true,
	}
	fsHandler := fs.NewRequestHandler()

	indexData, err := os.ReadFile(filepath.Join(dir, "index.html"))
	if err != nil {
		log.Printf("WARN: could not pre-read index.html: %v", err)
	}

	return func(ctx *fasthttp.RequestCtx) {
		path := string(ctx.Path())
		if strings.HasPrefix(path, "/v1/") ||
			strings.HasPrefix(path, "/auth/") ||
			strings.HasPrefix(path, "/account/") ||
			strings.HasPrefix(path, "/crypto/") ||
			strings.HasPrefix(path, "/stripe/") ||
			strings.HasPrefix(path, "/stats/") ||
			strings.HasPrefix(path, "/admin/") ||
			strings.HasPrefix(path, "/uploads/") ||
			strings.HasPrefix(path, "/openrouter/") ||
			strings.HasPrefix(path, "/monitoring/") ||
			path == "/health" ||
			path == "/sitemap.xml" {
			api(ctx)
			return
		}
		if isSPARoute(path) && indexData != nil {
			ctx.SetStatusCode(200)
			ctx.SetContentType("text/html; charset=utf-8")
			ctx.SetBody(injectUserData(injectPageMeta(indexData, path), ctx, apiKeyQ, userQ))
			return
		}
		fsHandler(ctx)
		if ctx.Response.StatusCode() == fasthttp.StatusNotFound || ctx.Response.StatusCode() == fasthttp.StatusForbidden {
			if indexData == nil {
				api(ctx)
				return
			}
			ctx.Response.Reset()
			ctx.SetStatusCode(200)
			ctx.SetContentType("text/html; charset=utf-8")
			ctx.SetBody(injectUserData(injectPageMeta(indexData, path), ctx, apiKeyQ, userQ))
		} else if ctx.Response.StatusCode() == fasthttp.StatusOK && strings.HasSuffix(path, ".html") || path == "/" {
			// Also inject on direct index.html hits
			body := injectPageMeta(ctx.Response.Body(), path)
			ctx.Response.SetBody(injectUserData(body, ctx, apiKeyQ, userQ))
		}
	}
}

func handleFrontendErrorReport(ctx *fasthttp.RequestCtx) {
	const maxBody = 64 * 1024
	body := ctx.PostBody()
	if len(body) == 0 || len(body) > maxBody {
		ctx.SetStatusCode(fasthttp.StatusNoContent)
		return
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		ctx.SetStatusCode(fasthttp.StatusNoContent)
		return
	}

	entry := map[string]any{
		"received_at": time.Now().UTC().Format(time.RFC3339Nano),
		"remote_addr": string(ctx.RemoteAddr().String()),
		"user_agent":  string(ctx.UserAgent()),
		"payload":     payload,
	}
	line, err := json.Marshal(entry)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusNoContent)
		return
	}

	logPath := os.Getenv("OPENPATHS_FRONTEND_ERROR_LOG")
	if logPath == "" {
		logPath = filepath.Join("monitoring", "errors", "frontend_client.jsonl")
	}
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		log.Printf("frontend error log mkdir: %v", err)
		ctx.SetStatusCode(fasthttp.StatusNoContent)
		return
	}
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		log.Printf("frontend error log open: %v", err)
		ctx.SetStatusCode(fasthttp.StatusNoContent)
		return
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		log.Printf("frontend error log write: %v", err)
	}
	ctx.SetStatusCode(fasthttp.StatusNoContent)
}

func isSPARoute(path string) bool {
	if path == "" || path == "/" {
		return false
	}
	base := path
	if i := strings.LastIndexByte(base, '/'); i >= 0 {
		base = base[i+1:]
	}
	return !strings.Contains(base, ".")
}

type pageMeta struct {
	Title       string
	Description string
	URL         string
}

func injectPageMeta(doc []byte, path string) []byte {
	meta := pageMetaForPath(path)
	if meta.Title == "" || meta.Description == "" || meta.URL == "" {
		return doc
	}

	replacements := []struct {
		pattern string
		value   string
	}{
		{`(?is)<title>.*?</title>`, fmt.Sprintf("<title>%s</title>", html.EscapeString(meta.Title))},
		{`(?is)<meta\s+name=["']description["']\s+content=["'][^"']*["']\s*/?>`, fmt.Sprintf(`<meta name="description" content="%s" />`, html.EscapeString(meta.Description))},
		{`(?is)<meta\s+property=["']og:url["']\s+content=["'][^"']*["']\s*/?>`, fmt.Sprintf(`<meta property="og:url" content="%s" />`, html.EscapeString(meta.URL))},
		{`(?is)<meta\s+property=["']og:title["']\s+content=["'][^"']*["']\s*/?>`, fmt.Sprintf(`<meta property="og:title" content="%s" />`, html.EscapeString(meta.Title))},
		{`(?is)<meta\s+property=["']og:description["']\s+content=["'][^"']*["']\s*/?>`, fmt.Sprintf(`<meta property="og:description" content="%s" />`, html.EscapeString(meta.Description))},
		{`(?is)<meta\s+name=["']twitter:title["']\s+content=["'][^"']*["']\s*/?>`, fmt.Sprintf(`<meta name="twitter:title" content="%s" />`, html.EscapeString(meta.Title))},
		{`(?is)<meta\s+name=["']twitter:description["']\s+content=["'][^"']*["']\s*/?>`, fmt.Sprintf(`<meta name="twitter:description" content="%s" />`, html.EscapeString(meta.Description))},
		{`(?is)<link\s+rel=["']canonical["']\s+href=["'][^"']*["']\s*/?>`, fmt.Sprintf(`<link rel="canonical" href="%s" />`, html.EscapeString(meta.URL))},
	}

	out := string(doc)
	for _, replacement := range replacements {
		re := regexp.MustCompile(replacement.pattern)
		if re.MatchString(out) {
			out = re.ReplaceAllString(out, replacement.value)
			continue
		}
		out = strings.Replace(out, "</head>", replacement.value+"\n</head>", 1)
	}

	return []byte(out)
}

func pageMetaForPath(path string) pageMeta {
	if path == "" {
		path = "/"
	}
	switch path {
	case "/pricing":
		return pageMeta{
			Title:       "OpenPaths Pricing | Near-Zero Markup AI Model Routing",
			Description: "OpenPaths keeps AI pricing as close to zero markup as practical, makes money from first-party AI services, and supports transparent pay-as-you-go pricing across text, embeddings, image, and video models.",
			URL:         "https://openpaths.io/pricing",
		}
	case "/stats":
		return pageMeta{
			Title:       "OpenPaths Stats | Anonymous AI Model Usage",
			Description: "Anonymous aggregate OpenPaths usage by task, provider, and model without exposing users, API keys, or prompt data.",
			URL:         "https://openpaths.io/stats",
		}
	default:
		return pageMeta{
			Title:       "OpenPaths - The Open Source Model Router",
			Description: "Search and we shall find. Neural learned paths for 1ms routing across 432+ large model providers and art generators. Try Open Pathways.",
			URL:         "https://openpaths.io" + path,
		}
	}
}

// injectUserData reads the op_session cookie and injects window.userData into index.html.
func injectUserData(html []byte, ctx *fasthttp.RequestCtx, apiKeyQ *queries.APIKeyQueries, userQ *queries.UserQueries) []byte {
	sessionKey := string(ctx.Request.Header.Cookie("op_session"))
	if sessionKey == "" {
		return html
	}
	apiKey, err := apiKeyQ.ValidateKey(ctx, auth.HashAPIKey(sessionKey))
	if err != nil {
		return html
	}
	user, err := userQ.GetByID(ctx, apiKey.UserID)
	if err != nil {
		return html
	}
	script := fmt.Sprintf(`<script>window.userData={id:%q,email:%q,name:%q,secret:%q,authenticated:true};</script>`,
		user.ID, user.Email, user.Name, sessionKey)
	return bytes.Replace(html, []byte("</head>"), []byte(script+"</head>"), 1)
}
