package server

import (
	"fmt"
	"log"
	"time"

	fasthttprouter "github.com/fasthttp/router"
	"github.com/valyala/fasthttp"

	"github.com/openpath/openpath/internal/auth"
	"github.com/openpath/openpath/internal/billing"
	"github.com/openpath/openpath/internal/config"
	"github.com/openpath/openpath/internal/db/queries"
	"github.com/openpath/openpath/internal/handler"
	"github.com/openpath/openpath/internal/metrics"
	"github.com/openpath/openpath/internal/middleware"
	"github.com/openpath/openpath/internal/router"
)

type Server struct {
	httpServer *fasthttp.Server
	port       int
}

type Dependencies struct {
	Config     *config.Config
	Router     *router.Router
	Billing    *billing.Engine
	Recorder   *metrics.Recorder
	JWTService *auth.JWTService
	UserQ      *queries.UserQueries
	APIKeyQ    *queries.APIKeyQueries
	CreditQ    *queries.CreditQueries
	StatsQ     *queries.StatsQueries
}

func New(deps *Dependencies) *Server {
	r := fasthttprouter.New()

	// Create handlers
	chatH := handler.NewChatHandler(deps.Router, deps.Billing, deps.Recorder)
	modelsH := handler.NewModelsHandler(deps.Router)
	authH := handler.NewAuthHandler(deps.UserQ, deps.CreditQ, deps.JWTService)
	accountH := handler.NewAccountHandler(deps.APIKeyQ, deps.CreditQ, deps.Billing)
	creditsH := handler.NewCreditsHandler(deps.Billing)
	statsH := handler.NewStatsHandler(deps.StatsQ)

	// Middleware chains
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

	// OpenAI-compatible API endpoints (API key auth)
	r.POST("/v1/chat/completions", apiKeyChain(chatH.HandleChatCompletion))
	r.GET("/v1/models", apiKeyChain(modelsH.HandleListModels))

	// Auth endpoints (public)
	r.POST("/auth/register", publicChain(authH.HandleRegister))
	r.POST("/auth/login", publicChain(authH.HandleLogin))

	// Account management (JWT auth)
	r.GET("/account/keys", jwtChain(accountH.HandleListAPIKeys))
	r.POST("/account/keys", jwtChain(accountH.HandleCreateAPIKey))
	r.DELETE("/account/keys/{id}", jwtChain(accountH.HandleRevokeAPIKey))
	r.GET("/account/balance", jwtChain(accountH.HandleGetBalance))
	r.GET("/account/transactions", jwtChain(accountH.HandleGetTransactions))
	r.POST("/account/credits/add", jwtChain(creditsH.HandleAddCredits))

	// Stats (public for now, can add auth later)
	r.GET("/stats/models", publicChain(statsH.HandleModelStats))
	r.GET("/stats/providers", publicChain(statsH.HandleProviderStats))
	r.GET("/stats/timeseries", publicChain(statsH.HandleTimeSeries))

	// Health check
	r.GET("/health", func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(200)
		ctx.SetBodyString(`{"status":"ok"}`)
	})

	srv := &fasthttp.Server{
		Handler:          r.Handler,
		ReadTimeout:      time.Duration(deps.Config.Server.ReadTimeout) * time.Second,
		WriteTimeout:     time.Duration(deps.Config.Server.WriteTimeout) * time.Second,
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
