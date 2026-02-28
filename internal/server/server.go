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

	"github.com/openpath/openpath/internal/auth"
	"github.com/openpath/openpath/internal/billing"
	"github.com/openpath/openpath/internal/config"
	"github.com/openpath/openpath/internal/crypto"
	"github.com/openpath/openpath/internal/db/queries"
	"github.com/openpath/openpath/internal/handler"
	"github.com/openpath/openpath/internal/metrics"
	"github.com/openpath/openpath/internal/middleware"
	"github.com/openpath/openpath/internal/provider"
	"github.com/openpath/openpath/internal/router"
	"github.com/openpath/openpath/internal/storage"
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
	CryptoSvc    *crypto.Service
	Storage      storage.Store
}

func New(deps *Dependencies) *Server {
	r := fasthttprouter.New()

	chatH := handler.NewChatHandler(deps.Router, deps.Billing, deps.Recorder)
	modelsH := handler.NewModelsHandler(deps.Router)
	authH := handler.NewAuthHandler(deps.UserQ, deps.CreditQ, deps.JWTService)
	accountH := handler.NewAccountHandler(deps.APIKeyQ, deps.CreditQ, deps.Billing)
	creditsH := handler.NewCreditsHandler(deps.Billing)
	statsH := handler.NewStatsHandler(deps.StatsQ)

	apiKeyChain := middleware.Chain(
		middleware.Recovery(),
		middleware.Logging(),
		middleware.APIKeyAuth(deps.APIKeyQ),
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
	r.POST("/account/credits/add", jwtChain(creditsH.HandleAddCredits))

	if deps.CryptoSvc != nil {
		cryptoH := handler.NewCryptoHandler(deps.CryptoSvc)
		r.POST("/crypto/checkout", jwtChain(cryptoH.HandleCreateCheckout))
		r.GET("/crypto/checkout/{id}", publicChain(cryptoH.HandleGetCheckout))
		r.GET("/crypto/checkout/{id}/events", publicChain(cryptoH.HandleCheckoutEvents))
		r.GET("/crypto/prices", publicChain(cryptoH.HandlePrices))
		log.Printf("Crypto payment endpoints enabled")
	}

	r.GET("/stats/models", publicChain(statsH.HandleModelStats))
	r.GET("/stats/providers", publicChain(statsH.HandleProviderStats))
	r.GET("/stats/timeseries", publicChain(statsH.HandleTimeSeries))

	r.GET("/health", func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(200)
		ctx.SetBodyString(`{"status":"ok"}`)
	})

	handler := r.Handler
	if staticDir := deps.Config.Server.StaticDir; staticDir != "" {
		handler = spaHandler(staticDir, r.Handler)
		log.Printf("Serving frontend from %s", staticDir)
	}

	srv := &fasthttp.Server{
		Handler:            handler,
		ReadTimeout:        time.Duration(deps.Config.Server.ReadTimeout) * time.Second,
		WriteTimeout:       time.Duration(deps.Config.Server.WriteTimeout) * time.Second,
		MaxRequestBodySize: deps.Config.Server.MaxRequestBody * 1024 * 1024,
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
	indexPath := filepath.Join(dir, "index.html")

	return func(ctx *fasthttp.RequestCtx) {
		path := string(ctx.Path())
		if strings.HasPrefix(path, "/v1/") ||
			strings.HasPrefix(path, "/auth/") ||
			strings.HasPrefix(path, "/account/") ||
			strings.HasPrefix(path, "/crypto/") ||
			strings.HasPrefix(path, "/stats/") ||
			strings.HasPrefix(path, "/uploads/") ||
			path == "/health" {
			api(ctx)
			return
		}
		// try static file
		fsHandler(ctx)
		if ctx.Response.StatusCode() == fasthttp.StatusNotFound || ctx.Response.StatusCode() == fasthttp.StatusForbidden {
			ctx.Response.Reset()
			data, err := os.ReadFile(indexPath)
			if err != nil {
				api(ctx)
				return
			}
			ctx.SetStatusCode(200)
			ctx.SetContentType("text/html; charset=utf-8")
			ctx.SetBody(data)
		}
	}
}
