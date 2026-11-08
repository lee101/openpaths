package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	fasthttprouter "github.com/fasthttp/router"
	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/agent"
	"github.com/openpaths/openpaths/internal/artindex"
	"github.com/openpaths/openpaths/internal/audio"
	"github.com/openpaths/openpaths/internal/auth"
	"github.com/openpaths/openpaths/internal/billing"
	"github.com/openpaths/openpaths/internal/config"
	"github.com/openpaths/openpaths/internal/crypto"
	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/discovery"
	"github.com/openpaths/openpaths/internal/email"
	"github.com/openpaths/openpaths/internal/handler"
	"github.com/openpaths/openpaths/internal/metrics"
	"github.com/openpaths/openpaths/internal/middleware"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/promptlib"
	"github.com/openpaths/openpaths/internal/provider"
	"github.com/openpaths/openpaths/internal/router"
	"github.com/openpaths/openpaths/internal/savedresp"
	"github.com/openpaths/openpaths/internal/seo"
	"github.com/openpaths/openpaths/internal/skillindex"
	"github.com/openpaths/openpaths/internal/storage"
	stripesvc "github.com/openpaths/openpaths/internal/stripe"
)

var activeLongRequests int64

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
	AppQ             *queries.AppQueries
	ModelProbeQ      *queries.ModelProbeQueries
	EvalQ            *queries.EvalQueries
	EvalRunner       handler.EvalSweeper
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
	VideoJobQ        *queries.VideoJobQueries
	Model3DJobQ      *queries.Model3DJobQueries
	ResearchJobQ     *queries.ResearchJobQueries
	OnRegister       handler.OnRegisterFunc
	ArtIndex         *artindex.Service
	ArtImageQ        *queries.ArtImageQueries
	PromptIndex      *promptlib.Index
	SkillIndex       *skillindex.Service
	SkillQ           *queries.SkillQueries
	Saver            *savedresp.Saver
	AgentQ           *queries.AgentQueries
	AgentEngine      *agent.Engine
	SharedChatQ      *queries.SharedChatQueries
	AccessQ          *queries.AccessQueries
	GuardQ           *queries.GuardQueries
	GuardrailQ       *queries.GuardrailQueries
	TeamQ            *queries.TeamQueries
	SuppressionQ     *queries.SuppressionQueries
	ArtifactQ        *queries.ArtifactQueries
	Announcer        *email.Announcer
}

func New(deps *Dependencies) *Server {
	r := fasthttprouter.New()

	chatH := handler.NewChatHandler(deps.Router, deps.Billing, deps.Recorder, deps.UserQ, deps.ProviderKeyQ)
	chatH.SetResponseSaver(deps.Saver)
	chatH.SetAccess(deps.AccessQ)
	orgH := handler.NewOrgHandler(deps.AccessQ, deps.GuardQ, deps.TeamQ, deps.UserQ)
	modelsH := handler.NewModelsHandler(deps.Router)
	authH := handler.NewAuthHandler(deps.UserQ, deps.CreditQ, deps.APIKeyQ, deps.JWTService)
	if deps.OnRegister != nil {
		authH.SetOnRegister(deps.OnRegister)
	}
	accountH := handler.NewAccountHandler(deps.APIKeyQ, deps.CreditQ, deps.Billing, deps.StripeReconciler)
	statsH := handler.NewStatsHandler(deps.StatsQ, deps.AppQ, deps.ModelProbeQ)
	evalsH := handler.NewEvalsHandler(deps.EvalQ, deps.EvalRunner)
	creditsH := handler.NewCreditsHandler()
	artH := handler.NewArtHandler(deps.ArtIndex, deps.ArtImageQ)
	promptsH := handler.NewPromptsHandler(deps.PromptIndex)
	skillsH := handler.NewSkillsHandler(deps.SkillIndex, deps.SkillQ)
	agentsH := handler.NewAgentsHandler(deps.AgentQ, deps.AgentEngine)
	agentsH.SetStorage(deps.Storage)
	acctStatsH := handler.NewAccountStatsHandler(deps.StatsQ)
	adminH := handler.NewAdminHandler(deps.UserQ, deps.StatsQ)

	apiKeyChain := middleware.Chain(
		middleware.Recovery(),
		middleware.Logging(),
		middleware.APIKeyAuth(deps.APIKeyQ),
		middleware.AppAttribution(deps.AppQ),
		middleware.BYOKLoader(deps.ProviderKeyQ),
		middleware.RateLimit(),
		middleware.GuardrailCheck(middleware.GuardrailDeps{Q: deps.GuardrailQ, UserQ: deps.UserQ}),
		middleware.BalanceCheck(deps.Billing),
		middleware.NewRequestCoalescer(24*time.Hour),
	)

	searchChain := middleware.Chain(
		middleware.Recovery(),
		middleware.Logging(),
		middleware.APIKeyAuth(deps.APIKeyQ),
		middleware.AppAttribution(deps.AppQ),
		middleware.BYOKLoader(deps.ProviderKeyQ),
		middleware.RateLimit(),
		middleware.GuardrailCheck(middleware.GuardrailDeps{Q: deps.GuardrailQ, UserQ: deps.UserQ}),
	)

	realtimeChain := middleware.Chain(
		middleware.Recovery(),
		middleware.Logging(),
		middleware.RealtimeAuth(deps.APIKeyQ, deps.JWTService),
		middleware.AppAttribution(deps.AppQ),
		middleware.BYOKLoader(deps.ProviderKeyQ),
		middleware.RateLimit(),
	)

	// accountChain: API key or dashboard JWT — no balance check for account management
	accountChain := middleware.Chain(
		middleware.Recovery(),
		middleware.Logging(),
		middleware.DashboardAuth(deps.APIKeyQ, deps.JWTService),
	)

	publicChain := middleware.Chain(
		middleware.Recovery(),
		middleware.Logging(),
	)

	scoped := func(chain middleware.Middleware, scope string) middleware.Middleware {
		return func(h fasthttp.RequestHandler) fasthttp.RequestHandler {
			return middleware.RequireScope(scope, chain(h))
		}
	}

	r.POST("/v1/chat/completions", scoped(apiKeyChain, model.ScopeChat)(chatH.HandleChatCompletion))
	r.GET("/v1/models", scoped(apiKeyChain, model.ScopeModels)(modelsH.HandleListModels))
	r.GET("/v1/models/{model_id}", scoped(apiKeyChain, model.ScopeModels)(modelsH.HandleGetModel))
	realtimeH := handler.NewRealtimeHandler(deps.Router, deps.Billing, deps.Recorder, deps.Config.Providers)
	r.GET("/v1/realtime", scoped(realtimeChain, model.ScopeRealtime)(realtimeH.HandleRealtime))
	log.Printf("Realtime WebSocket endpoint enabled at /v1/realtime (openai, gemini native protocol)")

	searchH := handler.NewSearchHandler(
		handler.ExaSearchProviderConfig(deps.Config.Providers),
		handler.PapersSearchProviderConfig(deps.Config.Providers),
		deps.Billing,
		deps.Recorder,
	).SetAnswerProviders(
		handler.GeminiSearchProviderConfig(deps.Config.Providers),
		handler.OpenAISearchProviderConfig(deps.Config.Providers),
		handler.GrokSearchProviderConfig(deps.Config.Providers),
	).SetDeepSeekProvider(handler.DeepSeekSearchProviderConfig(deps.Config.Providers)).
		SetOpenRouterProvider(handler.OpenRouterSearchProviderConfig(deps.Config.Providers)).
		SetKimiProvider(handler.KimiSearchProviderConfig(deps.Config.Providers))
	r.POST("/v1/search", scoped(searchChain, model.ScopeSearch)(searchH.HandleSearch))
	handler.LogExaSearchPricing()
	log.Printf("Search endpoint enabled at /v1/search (exa, papers, deepseek, kimi, kimi-pro, kimi-fetch, openpaths, openrouter engines: exa/parallel/firecrawl/perplexity, gemini, openai, grok)")

	anthH := handler.NewAnthropicHandler(deps.Router, deps.Billing, deps.Recorder)
	anthH.SetAccess(deps.AccessQ)
	anthH.SetResponseSaver(deps.Saver)
	r.POST("/v1/messages", scoped(apiKeyChain, model.ScopeChat)(anthH.HandleMessages))
	log.Printf("Anthropic-compatible /v1/messages endpoint enabled")

	imageH := handler.NewImageHandler(deps.Router, deps.Billing, deps.Recorder)
	imageH.SetAccess(deps.AccessQ)
	imageH.SetStorage(deps.Storage)
	imageH.SetResponseSaver(deps.Saver)
	r.POST("/v1/images/generations", middleware.PaywallContract(scoped(apiKeyChain, model.ScopeImages)(imageH.HandleImageGeneration)))
	r.POST("/v1/images/edits", middleware.PaywallContract(scoped(apiKeyChain, model.ScopeImages)(imageH.HandleImageGeneration)))
	log.Printf("Image generation endpoint enabled")

	model3DH := handler.NewModel3DHandler(deps.Router, deps.Billing, deps.Recorder, deps.Model3DJobQ)
	model3DH.SetAccess(deps.AccessQ)
	r.POST("/v1/3d/generations", middleware.PaywallContract(trackLongRequest(scoped(apiKeyChain, model.Scope3D)(model3DH.HandleModel3DGeneration))))
	r.GET("/v1/3d/generations/{job_id}", middleware.PaywallContract(scoped(apiKeyChain, model.Scope3D)(model3DH.HandleModel3DGenerationJob)))
	r.GET("/v1/3d/generations/{job_id}/status", middleware.PaywallContract(scoped(apiKeyChain, model.Scope3D)(model3DH.HandleModel3DGenerationJob)))
	log.Printf("3D generation endpoint enabled")

	textTo3DH := handler.NewTextTo3DHandler(imageH, model3DH, deps.Billing)
	r.POST("/v1/3d/text-generations", middleware.PaywallContract(trackLongRequest(scoped(apiKeyChain, model.Scope3D)(textTo3DH.HandleTextTo3DGeneration))))
	r.GET("/v1/3d/text-generations/{job_id}", middleware.PaywallContract(scoped(apiKeyChain, model.Scope3D)(textTo3DH.HandleTextTo3DGenerationJob)))
	r.GET("/v1/3d/text-generations/{job_id}/status", middleware.PaywallContract(scoped(apiKeyChain, model.Scope3D)(textTo3DH.HandleTextTo3DGenerationJob)))
	log.Printf("Text-to-3D generation endpoint enabled")

	riggingH := handler.NewMeshRiggingHandler(deps.Router, deps.Billing, deps.Recorder)
	r.POST("/v1/3d/rigging", middleware.PaywallContract(trackLongRequest(scoped(apiKeyChain, model.Scope3D)(riggingH.HandleMeshRigging))))
	r.GET("/v1/3d/rigging/{job_id}", middleware.PaywallContract(scoped(apiKeyChain, model.Scope3D)(riggingH.HandleMeshRiggingJob)))
	r.GET("/v1/3d/rigging/{job_id}/status", middleware.PaywallContract(scoped(apiKeyChain, model.Scope3D)(riggingH.HandleMeshRiggingJob)))
	log.Printf("3D mesh rigging endpoint enabled")

	videoH := handler.NewVideoHandler(deps.Router, deps.Billing, deps.Recorder, deps.VideoJobQ)
	videoH.SetAccess(deps.AccessQ)
	videoH.SetStorage(deps.Storage)
	r.POST("/v1/videos/generations", middleware.PaywallContract(trackLongRequest(scoped(apiKeyChain, model.ScopeVideo)(videoH.HandleVideoGeneration))))
	r.POST("/v1/videos/edits", middleware.PaywallContract(trackLongRequest(scoped(apiKeyChain, model.ScopeVideo)(videoH.HandleVideoEdit))))
	r.POST("/v1/videos/extensions", middleware.PaywallContract(trackLongRequest(scoped(apiKeyChain, model.ScopeVideo)(videoH.HandleVideoExtension))))
	r.GET("/v1/videos/generations/{job_id}", middleware.PaywallContract(scoped(apiKeyChain, model.ScopeVideo)(videoH.HandleVideoGenerationJob)))
	r.GET("/v1/videos/generations/{job_id}/status", middleware.PaywallContract(scoped(apiKeyChain, model.ScopeVideo)(videoH.HandleVideoGenerationJob)))
	r.GET("/v1/videos/edits/{job_id}", middleware.PaywallContract(scoped(apiKeyChain, model.ScopeVideo)(videoH.HandleVideoGenerationJob)))
	r.GET("/v1/videos/edits/{job_id}/status", middleware.PaywallContract(scoped(apiKeyChain, model.ScopeVideo)(videoH.HandleVideoGenerationJob)))
	r.GET("/v1/videos/extensions/{job_id}", middleware.PaywallContract(scoped(apiKeyChain, model.ScopeVideo)(videoH.HandleVideoGenerationJob)))
	r.GET("/v1/videos/extensions/{job_id}/status", middleware.PaywallContract(scoped(apiKeyChain, model.ScopeVideo)(videoH.HandleVideoGenerationJob)))
	log.Printf("Video generation endpoint enabled")

	researchH := handler.NewResearchHandler(
		deps.Router, deps.Billing, deps.Recorder, deps.ResearchJobQ,
		handler.ExaSearchProviderConfig(deps.Config.Providers),
		handler.PapersSearchProviderConfig(deps.Config.Providers),
	)
	r.POST("/v1/deep-research", scoped(apiKeyChain, model.ScopeSearch)(researchH.HandleCreate))
	r.GET("/v1/deep-research", scoped(apiKeyChain, model.ScopeSearch)(researchH.HandleList))
	r.GET("/v1/deep-research/{job_id}", scoped(apiKeyChain, model.ScopeSearch)(researchH.HandleGet))
	log.Printf("Deep research endpoint enabled at /v1/deep-research")

	artifactsH := handler.NewArtifactHandler(deps.ArtifactQ, deps.Billing)
	// Public API: free get, billed search ($1/1000), list + create via API key.
	r.GET("/v1/artifacts", publicChain(artifactsH.HandleListPublic))
	r.GET("/v1/artifacts/search", scoped(searchChain, model.ScopeArtifacts)(artifactsH.HandleSearch))
	r.POST("/v1/artifacts", scoped(searchChain, model.ScopeArtifacts)(artifactsH.HandleCreate))
	r.GET("/v1/artifacts/{id}", publicChain(artifactsH.HandleGet))
	// Dashboard: owner-scoped CRUD + free search of own+public.
	r.GET("/account/artifacts", scoped(accountChain, model.ScopeArtifacts)(artifactsH.HandleListMine))
	r.POST("/account/artifacts", scoped(accountChain, model.ScopeArtifacts)(artifactsH.HandleCreate))
	r.GET("/account/artifacts/search", scoped(accountChain, model.ScopeArtifacts)(artifactsH.HandleSearchMine))
	r.GET("/account/artifacts/{id}", scoped(accountChain, model.ScopeArtifacts)(artifactsH.HandleGetMine))
	r.PUT("/account/artifacts/{id}", scoped(accountChain, model.ScopeArtifacts)(artifactsH.HandleUpdate))
	r.DELETE("/account/artifacts/{id}", scoped(accountChain, model.ScopeArtifacts)(artifactsH.HandleDelete))
	log.Printf("Artifacts endpoints enabled at /v1/artifacts and /account/artifacts")

	musicH := handler.NewMusicHandler(deps.Router, deps.Billing, deps.Recorder)
	r.POST("/v1/music/generations", middleware.PaywallContract(scoped(apiKeyChain, model.ScopeAudio)(musicH.HandleMusicGeneration)))
	log.Printf("Music generation endpoint enabled")

	structuredH := handler.NewStructuredHandler(deps.Router, deps.Billing, deps.Recorder)
	r.POST("/v1/extract", scoped(apiKeyChain, model.ScopeChat)(structuredH.HandleExtract))
	r.POST("/v1/decide", scoped(apiKeyChain, model.ScopeChat)(structuredH.HandleDecide))
	log.Printf("Structured extraction (Clairic) and decision (Clef) endpoints enabled")

	forecastH := handler.NewForecastingHandler(deps.Router, deps.Billing, deps.Recorder)
	r.POST("/v1/forecasts", scoped(apiKeyChain, model.ScopeChat)(forecastH.HandleForecast))
	r.POST("/v1/forecasts/generations", scoped(apiKeyChain, model.ScopeChat)(forecastH.HandleForecast))
	log.Printf("Time-series forecasting endpoint enabled")

	speechH := handler.NewSpeechHandler(deps.Router, deps.Billing, deps.Recorder)
	speechH.SetAccess(deps.AccessQ)
	speechH.SetAutoEmotion(deps.AutoEmotion)
	r.POST("/v1/audio/speech", scoped(apiKeyChain, model.ScopeAudio)(speechH.HandleSpeechGeneration))
	r.POST("/v1/tts", scoped(apiKeyChain, model.ScopeAudio)(speechH.HandleSpeechGeneration))
	log.Printf("Speech generation endpoint enabled")

	embeddingH := handler.NewEmbeddingHandler(deps.Router, deps.Billing, deps.Recorder, deps.Embedders)
	r.POST("/v1/embeddings", scoped(apiKeyChain, model.ScopeEmbeddings)(embeddingH.HandleEmbedding))
	decisionH := handler.NewDecisionHandler(deps.Router, deps.Billing, deps.Recorder, deps.Embedders)
	r.POST("/v1/systemone", scoped(apiKeyChain, model.ScopeEmbeddings)(decisionH.HandleDecision))
	r.POST("/v1/decisions", scoped(apiKeyChain, model.ScopeEmbeddings)(decisionH.HandleDecision))
	log.Printf("Embedding endpoint enabled (%d fallback providers)", len(deps.Embedders))

	// MCP server: exposes all models over the Model Context Protocol. Runs under
	// auth+byok WITHOUT the upfront balance gate so initialize/tools/list always
	// succeed; paid tools bill via the underlying handler (charge-after-serve).
	var transcriptionH *handler.TranscriptionHandler
	if len(deps.Transcribers) > 0 {
		transcriptionH = handler.NewTranscriptionHandler(deps.Router, deps.Billing, deps.Transcribers, deps.Recorder)
		r.POST("/v1/audio/transcriptions", scoped(apiKeyChain, model.ScopeAudio)(transcriptionH.HandleTranscription))
		r.POST("/v1/stt", scoped(apiKeyChain, model.ScopeAudio)(transcriptionH.HandleTranscription))
		log.Printf("Transcription endpoint enabled (%d providers)", len(deps.Transcribers))
	}

	mcpH := handler.NewMCPHandler(deps.Router, chatH, modelsH, imageH, embeddingH, searchH)
	mcpH.SetVideoHandler(videoH)
	mcpH.SetMusicHandler(musicH)
	mcpH.SetSpeechHandler(speechH)
	mcpH.SetTranscriptionHandler(transcriptionH)
	mcpH.SetTextTo3DHandler(textTo3DH)
	mcpChain := middleware.Chain(
		middleware.Recovery(),
		middleware.Logging(),
		middleware.APIKeyAuth(deps.APIKeyQ),
		middleware.AppAttribution(deps.AppQ),
		middleware.BYOKLoader(deps.ProviderKeyQ),
		middleware.RateLimit(),
	)
	r.POST("/mcp", scoped(mcpChain, model.ScopeMCP)(mcpH.HandleMCP))
	r.POST("/v1/mcp", scoped(mcpChain, model.ScopeMCP)(mcpH.HandleMCP))
	log.Printf("MCP server enabled at /mcp (chat, list_models, generate_image, generate_video, check_job, generate_music, text_to_speech, transcribe_audio, generate_3d, embed, web_search)")

	if deps.Storage != nil {
		uploadH := handler.NewUploadHandler(deps.Storage)
		r.POST("/v1/files/upload", scoped(apiKeyChain, model.ScopeFiles)(uploadH.HandleUpload))
		log.Printf("File upload endpoint enabled")
	}

	r.ServeFiles("/uploads/{filepath:*}", deps.Config.Storage.LocalDir)

	if deps.SuppressionQ != nil {
		unsubH := handler.NewUnsubscribeHandler(deps.SuppressionQ, deps.UserQ, deps.Config.JWT.Secret)
		r.GET("/unsubscribe", publicChain(unsubH.HandleUnsubscribe))
		r.POST("/unsubscribe", publicChain(unsubH.HandleUnsubscribe))
		r.POST("/account/unsubscribe", scoped(accountChain, model.ScopeAccountManage)(unsubH.HandleAccountUnsubscribe))
	}

	registerMin, registerHour, registerDay := middleware.RegisterIPLimits()
	loginMin, loginHour, loginDay := middleware.LoginIPLimits()
	registerChain := middleware.Chain(
		middleware.Recovery(),
		middleware.Logging(),
		middleware.IPRateLimit("register", registerMin, registerHour, registerDay),
	)
	loginChain := middleware.Chain(
		middleware.Recovery(),
		middleware.Logging(),
		middleware.IPRateLimit("login", loginMin, loginHour, loginDay),
	)

	r.POST("/auth/register", registerChain(authH.HandleRegister))
	r.POST("/auth/login", loginChain(authH.HandleLogin))
	r.POST("/auth/logout", publicChain(authH.HandleLogout))

	for _, prefix := range []string{"/account/keys", "/v1/keys"} {
		r.GET(prefix+"/scopes", publicChain(accountH.HandleKeyScopes))
		r.GET(prefix+"/me", scoped(accountChain, model.ScopeNone)(accountH.HandleKeyMe))
		r.GET(prefix, scoped(accountChain, model.ScopeKeysManage)(accountH.HandleListAPIKeys))
		r.POST(prefix, scoped(accountChain, model.ScopeKeysManage)(accountH.HandleCreateAPIKey))
		r.POST(prefix+"/{id}/rotate", scoped(accountChain, model.ScopeKeysManage)(accountH.HandleRotateAPIKey))
		r.DELETE(prefix+"/{id}", scoped(accountChain, model.ScopeKeysManage)(accountH.HandleRevokeAPIKey))
	}
	r.DELETE("/account/keys", scoped(accountChain, model.ScopeKeysManage)(accountH.HandleRevokeAPIKeys))
	r.GET("/account/balance", scoped(accountChain, model.ScopeBillingRead)(accountH.HandleGetBalance))
	r.GET("/account/transactions", scoped(accountChain, model.ScopeBillingRead)(accountH.HandleGetTransactions))

	guardrailH := handler.NewGuardrailHandler(deps.GuardrailQ, deps.APIKeyQ)
	r.GET("/account/guardrails", scoped(accountChain, model.ScopeAccountManage)(guardrailH.HandleList))
	r.POST("/account/guardrails", scoped(accountChain, model.ScopeAccountManage)(guardrailH.HandleCreate))
	r.GET("/account/guardrails/events", scoped(accountChain, model.ScopeAccountManage)(guardrailH.HandleEvents))
	r.GET("/account/guardrails/{id}", scoped(accountChain, model.ScopeAccountManage)(guardrailH.HandleGet))
	r.PUT("/account/guardrails/{id}", scoped(accountChain, model.ScopeAccountManage)(guardrailH.HandleUpdate))
	r.DELETE("/account/guardrails/{id}", scoped(accountChain, model.ScopeAccountManage)(guardrailH.HandleDelete))
	r.PUT("/account/guardrails/{id}/assignments", scoped(accountChain, model.ScopeAccountManage)(guardrailH.HandleAssignments))
	log.Printf("Guardrails endpoints enabled at /account/guardrails")

	// Model IAM + billshock guards + teams (opt-in, open by default).
	r.GET("/account/model-rules", scoped(accountChain, model.ScopeAccountManage)(orgH.HandleModelRules))
	r.POST("/account/model-rules", scoped(accountChain, model.ScopeAccountManage)(orgH.HandleModelRules))
	r.DELETE("/account/model-rules", scoped(accountChain, model.ScopeAccountManage)(orgH.HandleModelRules))
	r.GET("/account/billing-guards", scoped(accountChain, model.ScopeAccountManage)(orgH.HandleBillingGuards))
	r.POST("/account/billing-guards", scoped(accountChain, model.ScopeAccountManage)(orgH.HandleBillingGuards))
	r.GET("/account/orgs", scoped(accountChain, model.ScopeAccountManage)(orgH.HandleOrgs))
	r.POST("/account/orgs", scoped(accountChain, model.ScopeAccountManage)(orgH.HandleOrgs))
	r.POST("/account/orgs/{slug}/invites", scoped(accountChain, model.ScopeAccountManage)(orgH.HandleOrgInvite))
	r.POST("/account/orgs/{slug}/join", scoped(accountChain, model.ScopeAccountManage)(orgH.HandleOrgJoin))
	r.GET("/account/orgs/{slug}/members", scoped(accountChain, model.ScopeAccountManage)(orgH.HandleOrgMembers))
	r.DELETE("/account/orgs/{slug}/members/{user_id}", scoped(accountChain, model.ScopeAccountManage)(orgH.HandleOrgMembers))
	// Keep legacy credit-minting paths fail-closed. Credit issuance is driven by
	// verified Stripe webhooks/reconciliation, never by request-body amounts.
	r.POST("/account/credits", scoped(accountChain, model.ScopeAccountManage)(creditsH.HandleAddCredits))
	r.POST("/account/credits/add", scoped(accountChain, model.ScopeAccountManage)(creditsH.HandleAddCredits))

	if deps.ProviderKeyQ != nil {
		pkH := handler.NewProviderKeysHandler(deps.ProviderKeyQ)
		r.GET("/account/provider-keys", scoped(accountChain, model.ScopeAccountManage)(pkH.HandleList))
		r.POST("/account/provider-keys", scoped(accountChain, model.ScopeAccountManage)(pkH.HandleUpsert))
		r.POST("/account/provider-keys/bulk", scoped(accountChain, model.ScopeAccountManage)(pkH.HandleBulkUpsert))
		r.DELETE("/account/provider-keys", scoped(accountChain, model.ScopeAccountManage)(pkH.HandleDelete))
		openAIAuthH := handler.NewOpenAIOAuthHandler(deps.ProviderKeyQ, deps.Config.JWT.Secret)
		r.POST("/account/openai/start", scoped(accountChain, model.ScopeAccountManage)(openAIAuthH.HandleStart))
		r.POST("/account/openai/device/start", scoped(accountChain, model.ScopeAccountManage)(openAIAuthH.HandleDeviceStart))
		r.POST("/account/openai/device/poll", scoped(accountChain, model.ScopeAccountManage)(openAIAuthH.HandleDevicePoll))
		r.POST("/account/openai/browser/start", scoped(accountChain, model.ScopeAccountManage)(openAIAuthH.HandleBrowserStart))
		r.POST("/account/openai/browser/complete", scoped(accountChain, model.ScopeAccountManage)(openAIAuthH.HandleBrowserComplete))
		r.GET("/account/openai/callback", publicChain(openAIAuthH.HandleCallback))
		log.Printf("BYOK provider keys endpoints enabled")
	}

	if deps.CryptoSvc != nil {
		cryptoH := handler.NewCryptoHandler(deps.CryptoSvc)
		r.POST("/crypto/checkout", scoped(accountChain, model.ScopeAccountManage)(cryptoH.HandleCreateCheckout))
		r.GET("/crypto/checkout/{id}", publicChain(cryptoH.HandleGetCheckout))
		r.GET("/crypto/checkout/{id}/events", publicChain(cryptoH.HandleCheckoutEvents))
		r.GET("/crypto/prices", publicChain(cryptoH.HandlePrices))
		log.Printf("Crypto payment endpoints enabled")
	}

	if deps.StripeSvc != nil {
		atH := handler.NewAutotopupHandler(deps.UserQ, deps.StripeSvc)
		r.POST("/account/stripe/setup", scoped(accountChain, model.ScopeAccountManage)(atH.HandleStripeSetup))
		r.POST("/account/stripe/confirm", scoped(accountChain, model.ScopeAccountManage)(atH.HandleStripeConfirm))
		r.GET("/account/stripe/payment-methods", scoped(accountChain, model.ScopeAccountManage)(atH.HandleListPaymentMethods))
		r.DELETE("/account/stripe/payment-methods/{id}", scoped(accountChain, model.ScopeAccountManage)(atH.HandleDeletePaymentMethod))
		r.POST("/account/autotopup/settings", scoped(accountChain, model.ScopeAccountManage)(atH.HandleUpdateAutotopupSettings))
		r.GET("/account/autotopup/settings", scoped(accountChain, model.ScopeAccountManage)(atH.HandleGetAutotopupSettings))

		checkoutH := handler.NewCheckoutHandler(deps.StripeSvc, deps.UserQ, deps.Billing, deps.StripeDepositQ, deps.Config.Stripe.CreditsPriceID, deps.Config.Stripe.WebhookSecret)
		checkoutH.SetGuards(deps.GuardQ)
		r.POST("/account/stripe/checkout", scoped(accountChain, model.ScopeAccountManage)(checkoutH.HandleCreateCheckout))
		r.GET("/account/stripe/config", publicChain(checkoutH.HandleStripeConfig))
		r.POST("/stripe/webhooks", publicChain(checkoutH.HandleWebhook))

		log.Printf("Stripe checkout + auto-topup endpoints enabled (price: %s)", deps.Config.Stripe.CreditsPriceID)
	}

	r.GET("/stats/models", publicChain(statsH.HandleModelStats))
	r.GET("/stats/model-probes", publicChain(statsH.HandleModelProbes))
	r.GET("/stats/providers", publicChain(statsH.HandleProviderStats))
	r.GET("/stats/breakdown", publicChain(statsH.HandleUsageBreakdown))
	r.GET("/stats/timeseries", publicChain(statsH.HandleTimeSeries))
	r.GET("/stats/models/timeseries", publicChain(statsH.HandleModelDailyUsage))
	r.GET("/stats/apps/{slug}", publicChain(statsH.HandleAppDetailStats))

	// Live evals: public snapshot, admin-triggered sweeps.
	r.GET("/v1/evals/results", publicChain(evalsH.HandleResults))
	r.GET("/v1/evals/status", publicChain(evalsH.HandleStatus))
	r.POST("/v1/evals/run", scoped(accountChain, model.ScopeAccountManage)(adminH.RequireAdmin(evalsH.HandleRun)))
	r.GET("/stats/apps", publicChain(statsH.HandleAppStats))
	r.GET("/og/apps/{slug}.svg", publicChain(statsH.HandleAppOGImage))
	r.GET("/art/search", publicChain(artH.HandleSearch))
	r.GET("/art/status", publicChain(artH.HandleStatus))
	r.GET("/art/list", publicChain(artH.HandleList))
	r.GET("/art/item", publicChain(artH.HandleItem))
	r.GET("/art/tags", publicChain(artH.HandleTags))
	r.GET("/v1/art/search", publicChain(artH.HandleSearch))
	r.GET("/v1/art/status", publicChain(artH.HandleStatus))
	r.GET("/v1/art/list", publicChain(artH.HandleList))
	r.GET("/v1/art/item", publicChain(artH.HandleItem))
	r.GET("/v1/art/tags", publicChain(artH.HandleTags))
	r.POST("/v1/art/index", scoped(accountChain, model.ScopeAccountManage)(adminH.RequireAdmin(artH.HandleIndex)))
	r.POST("/v1/art/reindex", scoped(accountChain, model.ScopeAccountManage)(adminH.RequireAdmin(artH.HandleReindex)))

	// Prompt library (the /prompts directory). Read-only, public, gobed-powered search.
	// Served under /v1 only so the client-side /prompts SPA routes are untouched.
	r.GET("/v1/prompts", publicChain(promptsH.HandleList))
	r.GET("/v1/prompts/meta", publicChain(promptsH.HandleMeta))
	r.GET("/v1/prompts/{slug}", publicChain(promptsH.HandleGet))
	// Searchable agent-skill library (public reads, owner-scoped marketplace writes).
	r.GET("/v1/skills", publicChain(skillsH.HandleList))
	r.GET("/v1/skills/meta", publicChain(skillsH.HandleMeta))
	r.GET("/v1/skills/search", publicChain(skillsH.HandleSearch))
	r.POST("/v1/skills", scoped(accountChain, model.ScopeAccountManage)(skillsH.HandleCreate))
	r.POST("/v1/skills/reindex", scoped(accountChain, model.ScopeAccountManage)(adminH.RequireAdmin(skillsH.HandleReindex)))
	// One catch-all: fasthttp/router also claims the bare "/files" path for a
	// "{filepath:*}" route, so a separate "/files" registration panics.
	r.GET("/v1/skills/{slug}/files/{filepath:*}", publicChain(func(ctx *fasthttp.RequestCtx) {
		if raw, _ := ctx.UserValue("filepath").(string); strings.Trim(raw, "/") == "" {
			skillsH.HandleFiles(ctx)
			return
		}
		skillsH.HandleFileRaw(ctx)
	}))
	r.GET("/v1/skills/{slug}", publicChain(skillsH.HandleGet))
	r.PUT("/v1/skills/{slug}", scoped(accountChain, model.ScopeAccountManage)(skillsH.HandleUpdate))
	r.DELETE("/v1/skills/{slug}", scoped(accountChain, model.ScopeAccountManage)(skillsH.HandleDelete))

	// Agents: user-built tool-use agents with connected data sources.
	r.GET("/v1/agents/presets", publicChain(agentsH.HandlePresets))
	r.GET("/v1/agents", scoped(accountChain, model.ScopeAgents)(agentsH.HandleList))
	r.POST("/v1/agents", scoped(accountChain, model.ScopeAgents)(agentsH.HandleCreate))
	r.GET("/v1/agents/{id}", scoped(accountChain, model.ScopeAgents)(agentsH.HandleGet))
	r.PATCH("/v1/agents/{id}", scoped(accountChain, model.ScopeAgents)(agentsH.HandleUpdate))
	r.DELETE("/v1/agents/{id}", scoped(accountChain, model.ScopeAgents)(agentsH.HandleDelete))
	r.POST("/v1/agents/{id}/run", scoped(accountChain, model.ScopeAgents)(agentsH.HandleRun))
	r.POST("/v1/agents/{id}/run/stream", scoped(accountChain, model.ScopeAgents)(agentsH.HandleRunStream))
	r.GET("/v1/agents/{id}/runs", scoped(accountChain, model.ScopeAgents)(agentsH.HandleRuns))
	r.GET("/v1/agents/{id}/sources", scoped(accountChain, model.ScopeAgents)(agentsH.HandleListSources))
	r.POST("/v1/agents/{id}/sources", scoped(accountChain, model.ScopeAgents)(agentsH.HandleCreateSource))
	r.POST("/v1/agents/{id}/sources/upload", scoped(accountChain, model.ScopeAgents)(agentsH.HandleUploadSource))
	r.DELETE("/v1/agents/{id}/sources/{sid}", scoped(accountChain, model.ScopeAgents)(agentsH.HandleDeleteSource))
	r.GET("/v1/agents/{id}/search", scoped(accountChain, model.ScopeAgents)(agentsH.HandleSearch))

	// Share-a-chat: publish a transcript, read it publicly at /chat/:slug.
	sharedChatH := handler.NewSharedChatHandler(deps.SharedChatQ)
	r.POST("/v1/chats/share", scoped(accountChain, model.ScopeAccountManage)(sharedChatH.HandleShare))
	r.GET("/v1/chats/shared", publicChain(sharedChatH.HandleGetShared))

	r.GET("/account/stats/timeseries", scoped(accountChain, model.ScopeUsageRead)(acctStatsH.HandleUserTimeSeries))
	r.GET("/account/stats/by-api-key", scoped(accountChain, model.ScopeUsageRead)(acctStatsH.HandleUserSpendByAPIKey))
	r.GET("/account/stats/by-provider", scoped(accountChain, model.ScopeUsageRead)(acctStatsH.HandleUserSpendByProvider))
	r.GET("/account/stats/by-product", scoped(accountChain, model.ScopeUsageRead)(acctStatsH.HandleUserSpendByProduct))
	r.GET("/account/stats/by-model", scoped(accountChain, model.ScopeUsageRead)(acctStatsH.HandleUserSpendByModel))
	r.GET("/account/stats/by-app", scoped(accountChain, model.ScopeUsageRead)(acctStatsH.HandleUserSpendByApp))
	r.GET("/account/stats/activity", scoped(accountChain, model.ScopeUsageRead)(acctStatsH.HandleUserActivity))
	r.GET("/account/stats/recent", scoped(accountChain, model.ScopeUsageRead)(acctStatsH.HandleUserRecentUsage))
	r.GET("/account/stats/by-api-key/{key_id}/models", scoped(accountChain, model.ScopeUsageRead)(acctStatsH.HandleUserAPIKeyDrilldown))
	r.GET("/account/stats/by-provider/{provider}/models", scoped(accountChain, model.ScopeUsageRead)(acctStatsH.HandleUserProviderDrilldown))
	r.GET("/admin/users/spend", scoped(accountChain, model.ScopeAccountManage)(adminH.RequireAdmin(adminH.HandleUserSpend)))
	r.GET("/admin/users/{user_id}/usage", scoped(accountChain, model.ScopeAccountManage)(adminH.RequireAdmin(adminH.HandleUserUsage)))
	r.GET("/admin/openai-max-plan", scoped(accountChain, model.ScopeAccountManage)(adminH.RequireAdmin(adminH.HandleOpenAIMaxPlanStatus)))
	r.POST("/admin/openai-max-plan/refresh", scoped(accountChain, model.ScopeAccountManage)(adminH.RequireAdmin(adminH.HandleOpenAIMaxPlanRefresh)))

	// Private saved-response search + settings (response saving feature).
	usageH := handler.NewUsageHandler(deps.Saver, deps.UserQ)
	r.GET("/account/usage/settings", scoped(accountChain, model.ScopeUsageRead)(usageH.HandleGetSettings))
	r.POST("/account/usage/settings", scoped(accountChain, model.ScopeAccountManage)(usageH.HandleUpdateSettings))
	r.GET("/account/usage/responses", scoped(accountChain, model.ScopeUsageRead)(usageH.HandleSearch))
	r.GET("/account/usage/responses/{id}", scoped(accountChain, model.ScopeUsageRead)(usageH.HandleItem))

	if deps.FineTuneQ != nil && len(deps.FineTuneProvs) > 0 {
		ftH := handler.NewFineTuneHandler(deps.FineTuneQ, deps.FineTuneProvs, deps.Storage)
		r.POST("/v1/files", scoped(apiKeyChain, model.ScopeFiles)(ftH.HandleUploadFile))
		r.GET("/v1/files", scoped(apiKeyChain, model.ScopeFiles)(ftH.HandleListFiles))
		r.POST("/v1/fine_tuning/jobs", scoped(apiKeyChain, model.ScopeFineTuning)(ftH.HandleCreateJob))
		r.GET("/v1/fine_tuning/jobs", scoped(apiKeyChain, model.ScopeFineTuning)(ftH.HandleListJobs))
		r.GET("/v1/fine_tuning/jobs/{job_id}", scoped(apiKeyChain, model.ScopeFineTuning)(ftH.HandleGetJob))
		r.POST("/v1/fine_tuning/jobs/{job_id}/cancel", scoped(apiKeyChain, model.ScopeFineTuning)(ftH.HandleCancelJob))
		r.GET("/v1/fine_tuning/jobs/{job_id}/events", scoped(apiKeyChain, model.ScopeFineTuning)(ftH.HandleListEvents))
		log.Printf("Fine-tuning endpoints enabled (%d providers)", len(deps.FineTuneProvs))
	}

	if deps.Discovery != nil {
		disc := deps.Discovery
		metaQ := deps.ModelMetaQ
		r.POST("/admin/discovery/run", scoped(accountChain, model.ScopeAccountManage)(adminH.RequireAdmin(func(ctx *fasthttp.RequestCtx) {
			n, err := disc.DiscoverAll(ctx)
			if err != nil {
				handler.WriteJSONPublic(ctx, 500, map[string]any{"error": err.Error()})
				return
			}
			handler.WriteJSONPublic(ctx, 200, map[string]any{"indexed": n})
		})))
		r.GET("/admin/discovery/models", scoped(accountChain, model.ScopeAccountManage)(adminH.RequireAdmin(func(ctx *fasthttp.RequestCtx) {
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

	if deps.Announcer != nil {
		announceH := handler.NewAnnounceHandler(deps.Announcer, deps.Discovery)
		r.POST("/admin/announce/model", scoped(accountChain, model.ScopeAccountManage)(adminH.RequireAdmin(announceH.HandleAnnounce)))
		r.GET("/admin/announce/candidates", scoped(accountChain, model.ScopeAccountManage)(adminH.RequireAdmin(announceH.HandleCandidates)))
		r.GET("/admin/announce/{model_id:*}", scoped(accountChain, model.ScopeAccountManage)(adminH.RequireAdmin(announceH.HandleStatus)))
		log.Printf("Model announcement endpoints enabled")
	}

	// OpenRouter provider endpoint - public, no auth required
	// This lets OpenRouter discover our models and route traffic to us
	orProvH := handler.NewOpenRouterProviderHandler(deps.Router)
	r.GET("/openrouter/models", publicChain(orProvH.HandleListModels))
	log.Printf("OpenRouter provider models endpoint enabled at /openrouter/models")

	r.GET("/health", func(ctx *fasthttp.RequestCtx) {
		ctx.SetContentType("application/json")
		ctx.SetStatusCode(200)
		activeLong := atomic.LoadInt64(&activeLongRequests) + handler.ActiveVideoJobs()
		ctx.SetBodyString(fmt.Sprintf(`{"status":"ok","active_long_requests":%d,"active_video_jobs":%d}`,
			activeLong, handler.ActiveVideoJobs()))
	})
	r.POST("/monitoring/frontend-errors", publicChain(handleFrontendErrorReport))

	// Sitemap index: references the static pages sitemap plus the DB-backed art
	// tag + image sitemaps (paged, because the art catalog exceeds the 50k/url limit).
	r.GET("/sitemap.xml", func(ctx *fasthttp.RequestCtx) {
		ctx.SetContentType("text/xml; charset=utf-8")
		ctx.SetStatusCode(200)
		ctx.SetBodyString(handler.ArtSitemapIndexXML("https://openpaths.io", artH.ArtSitemapPages(ctx)))
	})
	seoCat := seo.New(deps.Router.ListModels())
	registerSEORoutes(r, seoCat)
	r.GET("/sitemap-art-tags.xml", publicChain(artH.HandleSitemapTags))
	// Clean per-page art sitemaps (/sitemap-art-1.xml …) plus the legacy ?p= route
	// so URLs Google already discovered keep resolving.
	r.GET("/sitemap-art-{page}.xml", publicChain(artH.HandleSitemapImages))
	r.GET("/sitemap-art.xml", publicChain(artH.HandleSitemapImages))

	// fasthttp/router does not auto-answer HEAD for GET-only routes; without these
	// a HEAD probe returns 405 Method Not Allowed, which crawlers and validators
	// (Google Search Console, sitemap linters) report as "sitemap could not be read".
	// We answer HEAD with the right status + content type without regenerating the
	// body (an art page is ~20MB), since a HEAD caller only inspects the headers.
	sitemapHead := func(ctx *fasthttp.RequestCtx) {
		ctx.SetContentType("text/xml; charset=utf-8")
		ctx.SetStatusCode(200)
	}
	r.HEAD("/sitemap.xml", sitemapHead)
	r.HEAD("/sitemap-pages.xml", sitemapHead)
	r.HEAD("/sitemap-models.xml", sitemapHead)
	r.HEAD("/sitemap-art-tags.xml", sitemapHead)
	r.HEAD("/sitemap-art-{page}.xml", sitemapHead)
	r.HEAD("/sitemap-art.xml", sitemapHead)

	// Static pages sitemap (previously the root sitemap): core pages, blog, prompts.
	r.GET("/sitemap-pages.xml", func(ctx *fasthttp.RequestCtx) {
		if staticDir := deps.Config.Server.StaticDir; staticDir != "" {
			if data, err := os.ReadFile(filepath.Join(staticDir, "sitemap.xml")); err == nil {
				ctx.SetContentType("text/xml; charset=utf-8")
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
			{"/apps", "0.7", "daily"},
			{"/docs", "0.9", "weekly"},
			{"/integrations", "0.9", "weekly"},
			{"/playground", "0.7", "monthly"},
			{"/fusion", "0.6", "monthly"},
			{"/tools", "0.8", "weekly"},
			{"/tools/google-tts", "0.7", "monthly"},
			{"/tools/lyria", "0.7", "monthly"},
			{"/text-to-image", "0.7", "monthly"},
			{"/image-edit", "0.7", "monthly"},
			{"/gpt-image", "0.8", "monthly"},
			{"/text-to-video", "0.7", "monthly"},
			{"/image-to-video", "0.7", "monthly"},
			{"/video-extension", "0.7", "monthly"},
			{"/character-animator", "0.7", "monthly"},
			{"/music-generator", "0.7", "monthly"},
			{"/remove-video-background", "0.7", "monthly"},
			{"/image-to-3d", "0.7", "monthly"},
			{"/text-to-3d", "0.7", "monthly"},
			{"/rig-3d", "0.7", "monthly"},
			{"/retexture-3d", "0.7", "monthly"},
			{"/search", "0.7", "monthly"},
			{"/art", "0.7", "daily"},
			{"/blog", "0.8", "weekly"},
		}
		blogSlugs := []string{
			"use-openpaths-openai-compatible-router-anywhere",
			"llm-creative-coding-shader-video-benchmark",
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
			"provider-manifoldgen",
			"provider-cutedsl",
			"provider-papers",
			"provider-openpaths",
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
		for _, p := range promptlib.SitemapPaths() {
			priority := "0.6"
			if p == "/prompts" {
				priority = "0.8"
			} else if strings.HasPrefix(p, "/prompts/category/") || strings.HasPrefix(p, "/prompts/model/") || strings.HasPrefix(p, "/prompts/type/") {
				priority = "0.7"
			}
			fmt.Fprintf(&b, "  <url>\n    <loc>%s%s</loc>\n    <changefreq>weekly</changefreq>\n    <priority>%s</priority>\n  </url>\n", base, p, priority)
		}
		b.WriteString("</urlset>\n")
		ctx.SetContentType("text/xml; charset=utf-8")
		ctx.SetStatusCode(200)
		ctx.SetBodyString(b.String())
	})

	handler := r.Handler
	if staticDir := deps.Config.Server.StaticDir; staticDir != "" {
		handler = spaHandler(staticDir, r.Handler, deps.APIKeyQ, deps.UserQ, deps.JWTService, deps.ArtImageQ, seoCat)
		log.Printf("Serving frontend from %s", staticDir)
	}

	// Clients that set base_url to ".../v1" produce /v1/v1/... paths; tolerate it
	inner := handler
	handler = func(ctx *fasthttp.RequestCtx) {
		if p := ctx.Path(); bytes.HasPrefix(p, []byte("/v1/v1/")) {
			ctx.Request.URI().SetPathBytes(p[3:])
		}
		inner(ctx)
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

func trackLongRequest(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		atomic.AddInt64(&activeLongRequests, 1)
		defer atomic.AddInt64(&activeLongRequests, -1)
		next(ctx)
	}
}

// artAPIPaths are the bare /art/* endpoints that must hit the API (not the SPA).
// /art, /art/i/:slug and /art/tag/:slug are SPA routes and must NOT be listed here.
func isArtAPIPath(path string) bool {
	switch path {
	case "/art/search", "/art/status", "/art/list", "/art/item", "/art/tags":
		return true
	}
	return false
}

func spaHandler(dir string, api fasthttp.RequestHandler, apiKeyQ *queries.APIKeyQueries, userQ *queries.UserQueries, jwtService *auth.JWTService, artMetaQ *queries.ArtImageQueries, seoCat *seo.Catalog) fasthttp.RequestHandler {
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
	modelSEO := newModelSEO(dir, seoCat)

	return func(ctx *fasthttp.RequestCtx) {
		path := string(ctx.Path())
		// POST /mcp is the MCP JSON-RPC server; GET /mcp is the docs SPA page.
		if path == "/mcp" && ctx.IsPost() {
			api(ctx)
			return
		}
		if strings.HasPrefix(path, "/v1/") ||
			strings.HasPrefix(path, "/auth/") ||
			strings.HasPrefix(path, "/account/") ||
			strings.HasPrefix(path, "/crypto/") ||
			strings.HasPrefix(path, "/stripe/") ||
			strings.HasPrefix(path, "/stats/") ||
			isArtAPIPath(path) ||
			strings.HasPrefix(path, "/og/") ||
			strings.HasPrefix(path, "/admin/") ||
			strings.HasPrefix(path, "/uploads/") ||
			path == "/openrouter/models" ||
			strings.HasPrefix(path, "/monitoring/") ||
			path == "/health" ||
			path == "/unsubscribe" ||
			path == "/llms.txt" ||
			path == "/llms-full.txt" ||
			strings.HasPrefix(path, "/sitemap") {
			api(ctx)
			return
		}
		if page, ok := modelSEO.render(indexData, path); ok {
			ctx.SetStatusCode(200)
			ctx.SetContentType("text/html; charset=utf-8")
			ctx.SetBody(injectUserData(page, ctx, apiKeyQ, userQ, jwtService))
			return
		}
		resolveMeta := func() pageMeta {
			if m, ok := resolveArtMeta(ctx, artMetaQ, path); ok {
				return m
			}
			return pageMetaForPath(path)
		}
		if page := readPrerendered(dir, path); page != nil {
			ctx.SetStatusCode(200)
			ctx.SetContentType("text/html; charset=utf-8")
			ctx.SetBody(injectUserData(page, ctx, apiKeyQ, userQ, jwtService))
			return
		}
		if isSPARoute(path) && indexData != nil {
			ctx.SetStatusCode(200)
			ctx.SetContentType("text/html; charset=utf-8")
			ctx.SetBody(injectUserData(injectPageMeta(indexData, resolveMeta()), ctx, apiKeyQ, userQ, jwtService))
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
			ctx.SetBody(injectUserData(injectPageMeta(indexData, resolveMeta()), ctx, apiKeyQ, userQ, jwtService))
		} else if ctx.Response.StatusCode() == fasthttp.StatusOK && strings.HasSuffix(path, ".html") || path == "/" {
			// Also inject on direct index.html hits
			body := injectPageMeta(ctx.Response.Body(), resolveMeta())
			ctx.Response.SetBody(injectUserData(body, ctx, apiKeyQ, userQ, jwtService))
		}
	}
}

// resolveArtMeta produces crawler-visible meta for art detail (/art/i/:slug) and
// tag (/art/tag/:slug) pages from the DB so each has a unique title/description/og:image.
func resolveArtMeta(ctx *fasthttp.RequestCtx, q *queries.ArtImageQueries, path string) (pageMeta, bool) {
	const base = "https://openpaths.io"
	if rest := strings.TrimPrefix(path, "/art/i/"); rest != path {
		slug := strings.Trim(rest, "/")
		if dec, err := url.PathUnescape(slug); err == nil {
			slug = dec
		}
		if slug == "" || q == nil {
			return pageMeta{}, false
		}
		item, err := q.GetBySlug(ctx, slug)
		if err != nil || item == nil {
			return pageMeta{}, false
		}
		prompt := strings.Join(strings.Fields(item.Prompt), " ")
		title := truncateMeta(prompt, 70)
		return pageMeta{
			Title:       title + " | AI Art Prompt | OpenPaths",
			Description: "AI-generated art for the prompt: " + truncateMeta(prompt, 150) + ". Search thousands of prompts and try them in the OpenPaths playground.",
			URL:         base + "/art/i/" + url.PathEscape(slug),
			Image:       item.ImageURL,
		}, true
	}
	if rest := strings.TrimPrefix(path, "/art/tag/"); rest != path {
		tag := strings.Trim(rest, "/")
		if dec, err := url.PathUnescape(tag); err == nil {
			tag = dec
		}
		if tag == "" {
			return pageMeta{}, false
		}
		display := strings.Title(strings.ReplaceAll(tag, "-", " ")) //nolint:staticcheck
		return pageMeta{
			Title:       display + " AI Art Prompts | OpenPaths",
			Description: "Browse AI-generated " + tag + " art and prompts. Search by visual intent and try any prompt against OpenPaths image models.",
			URL:         base + "/art/tag/" + url.PathEscape(tag),
		}, true
	}
	return pageMeta{}, false
}

func truncateMeta(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return strings.TrimSpace(string(runes[:max])) + "…"
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
	if path == "/compare" || strings.HasPrefix(path, "/compare/") {
		return true
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
	Image       string
}

func injectPageMeta(doc []byte, meta pageMeta) []byte {
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
	if meta.Image != "" {
		replacements = append(replacements,
			struct{ pattern, value string }{`(?is)<meta\s+property=["']og:image["']\s+content=["'][^"']*["']\s*/?>`, fmt.Sprintf(`<meta property="og:image" content="%s" />`, html.EscapeString(meta.Image))},
			struct{ pattern, value string }{`(?is)<meta\s+name=["']twitter:image["']\s+content=["'][^"']*["']\s*/?>`, fmt.Sprintf(`<meta name="twitter:image" content="%s" />`, html.EscapeString(meta.Image))},
		)
	}

	out := string(doc)
	for _, replacement := range replacements {
		re := cachedRegexp(replacement.pattern)
		if re.MatchString(out) {
			out = re.ReplaceAllString(out, replacement.value)
			continue
		}
		out = strings.Replace(out, "</head>", replacement.value+"\n</head>", 1)
	}

	return []byte(out)
}

// toolPageMeta mirrors src/data/tools.ts so crawler-visible meta matches the
// prerendered HTML and the client-rendered tags, including the per-tool OG card.
var toolPageMeta = map[string]pageMeta{
	"/tools": {
		Title:       "AI Tools - Image, Video, Music, Speech, 3D | OpenPaths",
		Description: "Hands-on studios for every OpenPaths generation endpoint: text to image, video, music, speech, 3D, model fusion, and the multi-model playground. Each tool is a thin wrapper over the API.",
		Image:       "https://openpaths.io/og/tools/index.webp",
	},
	"/tools/google-tts": {
		Title:       "Gemini Flash TTS Studio - Multi-speaker AI Voice | OpenPaths",
		Description: "Generate steerable single- and multi-speaker speech with Gemini 3.1 Flash TTS. Direct voice, style, pace, accent, scene, and emotion, then play, download, or copy the API code.",
	},
	"/tools/lyria": {
		Title:       "Lyria 3 Music Studio - AI Song & Instrumental Generator | OpenPaths",
		Description: "Generate complete songs, instrumentals, loops, and 30-second clips with Google Lyria 3 Pro and Clip. Direct genre, mood, structure, instruments, vocals, lyrics, and export as Opus.",
	},
	"/text-to-image": {
		Title:       "Text to Image API | OpenPaths",
		Description: "Generate images from text with the OpenPaths auto image endpoint - routes to GPT Image 2, RA1, Flux, and more with near-zero markup.",
	},
	"/image-edit": {
		Title:       "AI Image Style Transfer | OpenPaths",
		Description: "Upload an image and describe a new visual direction. OpenPaths routes the edit through GPT Image 2 and image-editing fallbacks.",
	},
	"/gpt-image": {
		Title:       "GPT Image 2.6 API - Sunburst and Flare | OpenPaths",
		Description: "Generate and edit images with GPT Image 2.6 and 2.5 Sunburst and Flare through one OpenAI-compatible API. $0.06 per 1K image, with real example outputs from every model.",
	},
	"/text-to-video": {
		Title:       "Text to Video API - Wan 3.0 | OpenPaths",
		Description: "Generate up-to-30s cinematic video with native audio from one prompt through Wan 3.0 on OpenPaths. Per-second pricing by resolution.",
	},
	"/image-to-video": {
		Title:       "Image to Video API - Wan 3.0 | OpenPaths",
		Description: "Animate a still into up-to-30s video with native audio through Wan 3.0 on OpenPaths. Optional end frame, per-second pricing by resolution.",
	},
	"/video-extension": {
		Title:       "Video Edit and Extension API | OpenPaths",
		Description: "Edit or extend an existing MP4 with Grok Imagine Video through OpenPaths. Describe the next beat and continue any clip.",
	},
	"/character-animator": {
		Title:       "Character Animator - Wan-Animate API | OpenPaths",
		Description: "Animate a reference character with a driving performance video through Wan-Animate on ManifoldGen, OpenPaths' own GPU studio. Standard, fast, and x-fast latency lanes with per-second pricing.",
	},
	"/music-generator": {
		Title:       "AI Music Generator - MiniMax-Music3 API | OpenPaths",
		Description: "Generate full songs with vocals from a prompt and optional lyrics through MiniMax-Music3 on ManifoldGen, OpenPaths' own GPU studio. Pin any length from 30 to 300 seconds.",
	},
	"/remove-video-background": {
		Title:       "Remove Video Background - Transparent WebM API | OpenPaths",
		Description: "Key any clip to alpha-transparent WebM through ManifoldGen, OpenPaths' own GPU studio. Optionally composite a solid backdrop color and keep the original audio.",
	},
	"/image-to-3d": {
		Title:       "Image to 3D API | OpenPaths",
		Description: "Generate textured GLB models from a single image through OpenPaths with Pixal3D, Meshy v6, and Tripo p1.",
	},
	"/text-to-3d": {
		Title:       "Text to 3D API | OpenPaths",
		Description: "Generate textured GLB models straight from a text prompt. OpenPaths auto-generates an image then converts it to 3D with Pixal3D.",
	},
	"/rig-3d": {
		Title:       "3D Auto-Rigging API | OpenPaths",
		Description: "Upload a humanoid GLB and get back a rigged character (GLB + FBX) with optional walk/run animation, via OpenPaths and Fal Meshy.",
	},
	"/retexture-3d": {
		Title:       "3D Retexture API | OpenPaths",
		Description: "Re-texture an existing 3D mesh from a reference image - upload a GLB and a style image, get back a textured GLB, via OpenPaths and Fal Trellis-2.",
	},
	"/playground": {
		Title:       "Playground - Test Any AI Model Side by Side | OpenPaths",
		Description: "Multi-pane studio for chat, image, video, and audio models. Compare any two models on the same prompt with live cost and latency, then copy the API snippet.",
	},
	"/fusion": {
		Title:       "Model Fusion Beta | OpenPaths",
		Description: "Run multiple models side by side with OpenRouter fusion, analyze the panel, and fuse the strongest result into one answer.",
	},
}

// toolOgSlug maps a tool route to its OG card slug (the route basename, except
// for the /tools/* studios which are already slugged).
func toolOgSlug(path string) string {
	if rest := strings.TrimPrefix(path, "/tools/"); rest != path {
		return rest
	}
	return strings.TrimPrefix(path, "/")
}

func pageMetaForPath(path string) pageMeta {
	if path == "" {
		path = "/"
	}
	if meta, ok := toolPageMeta[path]; ok {
		meta.URL = "https://openpaths.io" + path
		if meta.Image == "" {
			meta.Image = "https://openpaths.io/og/tools/" + toolOgSlug(path) + ".webp"
		}
		return meta
	}
	switch path {
	case "/pricing":
		return pageMeta{
			Title:       "OpenPaths Pricing | Near-Zero Markup AI Model Routing",
			Description: "OpenPaths keeps AI pricing as close to zero markup as practical, makes money from first-party AI services, and supports transparent pay-as-you-go pricing across text, embeddings, image, and video models.",
			URL:         "https://openpaths.io/pricing",
		}
	case "/evals":
		return pageMeta{
			Title:       "AI Model Evals, Pricing, and Speed | OpenPaths",
			Description: "Compare frontier AI model intelligence, coding, agentic performance, speed, and token pricing using the OpenPaths Artificial Analysis benchmark snapshot.",
			URL:         "https://openpaths.io/evals",
		}
	case "/compare":
		return pageMeta{
			Title:       "Compare AI Models by Evals, Speed, and Price | OpenPaths",
			Description: "Compare frontier AI models head-to-head using Artificial Analysis evals, pricing, context windows, speed, and benchmark run costs.",
			URL:         "https://openpaths.io/compare",
		}
	case "/stats":
		return pageMeta{
			Title:       "OpenPaths Stats | AI Model Usage",
			Description: "Daily model usage and public OpenPaths request breakdowns.",
			URL:         "https://openpaths.io/stats",
		}
	case "/art":
		return pageMeta{
			Title:       "AI Art Prompt Search | OpenPaths",
			Description: "Search a huge library of AI-generated art by prompt, style, scene, mood, and aspect ratio (square, portrait, wide). Browse subtags and try any prompt in the OpenPaths playground.",
			URL:         "https://openpaths.io/art",
		}
	case "/alternatives":
		return pageMeta{
			Title:       "OpenRouter Alternative | 0% AI Gateway Markup | OpenPaths",
			Description: "Compare OpenPaths with OpenRouter, OpenAI, Anthropic, LiteLLM, and more. One API, pooled credits, smart routing, and 0% platform markup.",
			URL:         "https://openpaths.io/alternatives",
		}
	default:
		if strings.HasPrefix(path, "/compare/") {
			return pageMeta{
				Title:       "AI Model Comparison | OpenPaths",
				Description: "Head-to-head AI model comparison with evals, pricing, context windows, speed, and benchmark run costs.",
				URL:         "https://openpaths.io" + path,
			}
		}
		return pageMeta{
			Title:       "OpenPaths - The Open Source Model Router",
			Description: "Search and we shall find. Neural learned paths for 1ms routing across 432+ large model providers and art generators. Try Open Pathways.",
			URL:         "https://openpaths.io" + path,
		}
	}
}

// injectUserData reads the op_session cookie and injects window.userData into index.html.
func injectUserData(html []byte, ctx *fasthttp.RequestCtx, apiKeyQ *queries.APIKeyQueries, userQ *queries.UserQueries, jwtService *auth.JWTService) []byte {
	sessionKey := string(ctx.Request.Header.Cookie("op_session"))
	if sessionKey == "" {
		return html
	}

	var userID string
	if strings.HasPrefix(sessionKey, auth.APIKeyPrefix) {
		apiKey, err := apiKeyQ.ValidateKey(ctx, auth.HashAPIKey(sessionKey))
		if err != nil {
			return html
		}
		userID = apiKey.UserID
	} else if jwtService != nil {
		claims, err := jwtService.Validate(sessionKey)
		if err != nil {
			return html
		}
		userID = claims.UserID
	} else {
		return html
	}

	user, err := userQ.GetByID(ctx, userID)
	if err != nil {
		return html
	}
	script := fmt.Sprintf(`<script>window.userData={id:%q,email:%q,name:%q,secret:%q,authenticated:true};</script>`,
		user.ID, user.Email, user.Name, sessionKey)
	return bytes.Replace(html, []byte("</head>"), []byte(script+"</head>"), 1)
}
