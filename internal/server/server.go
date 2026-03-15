package server

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	fasthttprouter "github.com/fasthttp/router"
	"github.com/valyala/fasthttp"

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
	Config       *config.Config
	Router       *router.Router
	Billing      *billing.Engine
	Recorder     *metrics.Recorder
	JWTService   *auth.JWTService
	UserQ        *queries.UserQueries
	APIKeyQ      *queries.APIKeyQueries
	CreditQ      *queries.CreditQueries
	StatsQ       *queries.StatsQueries
	Transcribers []provider.TranscriptionProvider
	Embedders    []provider.EmbeddingProvider
	CryptoSvc      *crypto.Service
	Storage        storage.Store
	StripeSvc      *stripesvc.Service
	Discovery      *discovery.Service
	ModelMetaQ     *queries.ModelMetadataQueries
	FineTuneQ      *queries.FineTuneQueries
	FineTuneProvs  map[string]provider.FineTuneProvider
	ProviderKeyQ   *queries.ProviderKeyQueries
}

func New(deps *Dependencies) *Server {
	r := fasthttprouter.New()

	chatH := handler.NewChatHandler(deps.Router, deps.Billing, deps.Recorder)
	modelsH := handler.NewModelsHandler(deps.Router)
	authH := handler.NewAuthHandler(deps.UserQ, deps.CreditQ, deps.APIKeyQ, deps.JWTService)
	accountH := handler.NewAccountHandler(deps.APIKeyQ, deps.CreditQ, deps.Billing)
	statsH := handler.NewStatsHandler(deps.StatsQ)

	apiKeyChain := middleware.Chain(
		middleware.Recovery(),
		middleware.Logging(),
		middleware.APIKeyAuth(deps.APIKeyQ),
		middleware.BYOKLoader(deps.ProviderKeyQ),
		middleware.RateLimit(),
		middleware.BalanceCheck(deps.Billing),
	)

	jwtChain := middleware.Chain(
		middleware.Recovery(),
		middleware.Logging(),
		middleware.JWTAuth(deps.JWTService),
	)

	publicChain := middleware.Chain(
		middleware.Recovery(),
		middleware.Logging(),
	)

	r.POST("/v1/chat/completions", apiKeyChain(chatH.HandleChatCompletion))
	r.GET("/v1/models", apiKeyChain(modelsH.HandleListModels))
	r.GET("/v1/models/{model_id}", apiKeyChain(modelsH.HandleGetModel))

	anthH := handler.NewAnthropicHandler(deps.Router, deps.Billing, deps.Recorder)
	r.POST("/v1/messages", apiKeyChain(anthH.HandleMessages))
	log.Printf("Anthropic-compatible /v1/messages endpoint enabled")

	imageH := handler.NewImageHandler(deps.Router, deps.Billing, deps.Recorder)
	r.POST("/v1/images/generations", apiKeyChain(imageH.HandleImageGeneration))
	log.Printf("Image generation endpoint enabled")

	videoH := handler.NewVideoHandler(deps.Router, deps.Billing, deps.Recorder)
	r.POST("/v1/videos/generations", apiKeyChain(videoH.HandleVideoGeneration))
	log.Printf("Video generation endpoint enabled")

	musicH := handler.NewMusicHandler(deps.Router, deps.Billing, deps.Recorder)
	r.POST("/v1/music/generations", apiKeyChain(musicH.HandleMusicGeneration))
	log.Printf("Music generation endpoint enabled")

	speechH := handler.NewSpeechHandler(deps.Router, deps.Billing, deps.Recorder)
	r.POST("/v1/audio/speech", apiKeyChain(speechH.HandleSpeechGeneration))
	log.Printf("Speech generation endpoint enabled")

	embeddingH := handler.NewEmbeddingHandler(deps.Router, deps.Billing, deps.Recorder, deps.Embedders)
	r.POST("/v1/embeddings", apiKeyChain(embeddingH.HandleEmbedding))
	log.Printf("Embedding endpoint enabled (%d fallback providers)", len(deps.Embedders))

	if len(deps.Transcribers) > 0 {
		transcriptionH := handler.NewTranscriptionHandler(deps.Transcribers, deps.Router.HealthTracker(), deps.Recorder)
		r.POST("/v1/audio/transcriptions", apiKeyChain(transcriptionH.HandleTranscription))
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

	r.GET("/account/keys", jwtChain(accountH.HandleListAPIKeys))
	r.POST("/account/keys", jwtChain(accountH.HandleCreateAPIKey))
	r.DELETE("/account/keys/{id}", jwtChain(accountH.HandleRevokeAPIKey))
	r.GET("/account/balance", jwtChain(accountH.HandleGetBalance))
	r.GET("/account/transactions", jwtChain(accountH.HandleGetTransactions))

	if deps.ProviderKeyQ != nil {
		pkH := handler.NewProviderKeysHandler(deps.ProviderKeyQ)
		r.GET("/account/provider-keys", jwtChain(pkH.HandleList))
		r.POST("/account/provider-keys", jwtChain(pkH.HandleUpsert))
		r.POST("/account/provider-keys/bulk", jwtChain(pkH.HandleBulkUpsert))
		r.DELETE("/account/provider-keys", jwtChain(pkH.HandleDelete))
		log.Printf("BYOK provider keys endpoints enabled")
	}

	if deps.CryptoSvc != nil {
		cryptoH := handler.NewCryptoHandler(deps.CryptoSvc)
		r.POST("/crypto/checkout", jwtChain(cryptoH.HandleCreateCheckout))
		r.GET("/crypto/checkout/{id}", publicChain(cryptoH.HandleGetCheckout))
		r.GET("/crypto/checkout/{id}/events", publicChain(cryptoH.HandleCheckoutEvents))
		r.GET("/crypto/prices", publicChain(cryptoH.HandlePrices))
		log.Printf("Crypto payment endpoints enabled")
	}

	if deps.StripeSvc != nil {
		atH := handler.NewAutotopupHandler(deps.UserQ, deps.StripeSvc)
		r.POST("/account/stripe/setup", jwtChain(atH.HandleStripeSetup))
		r.POST("/account/stripe/confirm", jwtChain(atH.HandleStripeConfirm))
		r.GET("/account/stripe/payment-methods", jwtChain(atH.HandleListPaymentMethods))
		r.DELETE("/account/stripe/payment-methods/{id}", jwtChain(atH.HandleDeletePaymentMethod))
		r.POST("/account/autotopup/settings", jwtChain(atH.HandleUpdateAutotopupSettings))
		r.GET("/account/autotopup/settings", jwtChain(atH.HandleGetAutotopupSettings))

		checkoutH := handler.NewCheckoutHandler(deps.StripeSvc, deps.UserQ, deps.Billing, deps.Config.Stripe.CreditsPriceID, deps.Config.Stripe.WebhookSecret)
		r.POST("/account/stripe/checkout", jwtChain(checkoutH.HandleCreateCheckout))
		r.GET("/account/stripe/config", publicChain(checkoutH.HandleStripeConfig))
		r.POST("/stripe/webhooks", publicChain(checkoutH.HandleWebhook))

		log.Printf("Stripe checkout + auto-topup endpoints enabled (price: %s)", deps.Config.Stripe.CreditsPriceID)
	}

	r.GET("/stats/models", publicChain(statsH.HandleModelStats))
	r.GET("/stats/providers", publicChain(statsH.HandleProviderStats))
	r.GET("/stats/timeseries", publicChain(statsH.HandleTimeSeries))

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
		r.POST("/admin/discovery/run", jwtChain(func(ctx *fasthttp.RequestCtx) {
			n, err := disc.DiscoverAll(ctx)
			if err != nil {
				handler.WriteJSONPublic(ctx, 500, map[string]any{"error": err.Error()})
				return
			}
			handler.WriteJSONPublic(ctx, 200, map[string]any{"indexed": n})
		}))
		r.GET("/admin/discovery/models", publicChain(func(ctx *fasthttp.RequestCtx) {
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
		}))
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

	r.GET("/sitemap.xml", func(ctx *fasthttp.RequestCtx) {
		const base = "https://openpaths.io"
		pages := []struct{ loc, priority, freq string }{
			{"/", "1.0", "daily"},
			{"/models", "0.9", "weekly"},
			{"/providers", "0.8", "weekly"},
			{"/docs", "0.9", "weekly"},
			{"/playground", "0.7", "monthly"},
			{"/blog", "0.8", "weekly"},
		}
		blogSlugs := []string{
			"switch-to-openpaths-in-2-lines",
			"state-of-ai-models-march-2026",
			"how-auto-models-work",
			"ai-art-generation-compared",
			"choosing-the-right-llm",
			"image-resolution-handling",
			"video-generation-guide",
			"openpath-vs-openrouter",
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
		handler = spaHandler(staticDir, r.Handler)
		log.Printf("Serving frontend from %s", staticDir)
	}

	srv := &fasthttp.Server{
		Handler:                       handler,
		ReadTimeout:                   time.Duration(deps.Config.Server.ReadTimeout) * time.Second,
		WriteTimeout:                  time.Duration(deps.Config.Server.WriteTimeout) * time.Second,
		MaxRequestBodySize:            deps.Config.Server.MaxRequestBody * 1024 * 1024,
		DisableHeaderNamesNormalizing: true,
		TCPKeepalive:                  true,
		ReduceMemoryUsage:            false,
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

func spaHandler(dir string, api fasthttp.RequestHandler) fasthttp.RequestHandler {
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
			path == "/health" {
			api(ctx)
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
			ctx.SetBody(indexData)
		}
	}
}
