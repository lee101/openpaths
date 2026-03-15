package main

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/openpaths/openpaths/internal/auth"
	"github.com/openpaths/openpaths/internal/billing"
	"github.com/openpaths/openpaths/internal/config"
	"github.com/openpaths/openpaths/internal/cron"
	"github.com/openpaths/openpaths/internal/crypto"
	"github.com/openpaths/openpaths/internal/db"
	"github.com/openpaths/openpaths/internal/db/migrations"
	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/discovery"
	"github.com/openpaths/openpaths/internal/metrics"
	"github.com/openpaths/openpaths/internal/provider"
	stripesvc "github.com/openpaths/openpaths/internal/stripe"
	"github.com/openpaths/openpaths/internal/provider/anthropic"
	"github.com/openpaths/openpaths/internal/provider/deepseek"
	"github.com/openpaths/openpaths/internal/provider/fal"
	gobedprov "github.com/openpaths/openpaths/internal/provider/gobed"
	"github.com/openpaths/openpaths/internal/provider/google"
	"github.com/openpaths/openpaths/internal/provider/groq"
	"github.com/openpaths/openpaths/internal/provider/minimax"
	"github.com/openpaths/openpaths/internal/provider/mistral"
	"github.com/openpaths/openpaths/internal/provider/netwrck"
	"github.com/openpaths/openpaths/internal/provider/nous"
	"github.com/openpaths/openpaths/internal/provider/openai"
	"github.com/openpaths/openpaths/internal/provider/openrouter"
	"github.com/openpaths/openpaths/internal/provider/textgenerator"
	"github.com/openpaths/openpaths/internal/provider/together"
	"github.com/openpaths/openpaths/internal/provider/xai"
	"github.com/openpaths/openpaths/internal/provider/zai"
	"github.com/openpaths/openpaths/internal/router"
	"github.com/openpaths/openpaths/internal/server"
	"github.com/openpaths/openpaths/internal/storage"
)

func main() {
	_ = godotenv.Load()

	configPath := flag.String("config", "config.yaml", "Path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid config: %v", err)
	}

	ctx := context.Background()

	database, err := db.New(ctx, cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	if err := migrations.Run(ctx, database.Pool); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	userQ := queries.NewUserQueries(database.Pool)
	apiKeyQ := queries.NewAPIKeyQueries(database.Pool)
	creditQ := queries.NewCreditQueries(database.Pool)
	usageQ := queries.NewUsageQueries(database.Pool)
	statsQ := queries.NewStatsQueries(database.Pool)
	providerKeyQ := queries.NewProviderKeyQueries(database.Pool)

	jwtService := auth.NewJWTService(cfg.JWT.Secret, cfg.JWT.ExpirationHours)

	registry := provider.NewRegistry()
	var transcribers []provider.TranscriptionProvider
	var embedders []provider.EmbeddingProvider

	if gp, err := gobedprov.New(); err != nil {
		log.Printf("gobed: disabled (%v)", err)
	} else {
		embedders = append(embedders, gp)
		log.Printf("Registered embedding provider: gobed")
	}

	for _, provCfg := range cfg.Providers {
		if !provCfg.Enabled || provCfg.APIKey == "" {
			log.Printf("Skipping provider %s (disabled or no API key)", provCfg.Name)
			continue
		}

		var p provider.Provider
		switch provCfg.Name {
		case "openai":
			p = openai.New(provCfg.APIKey, provCfg.BaseURL)
			transcribers = append(transcribers, openai.NewTranscriber(provCfg.APIKey, provCfg.BaseURL))
		case "anthropic":
			p = anthropic.New(provCfg.APIKey, provCfg.BaseURL)
		case "google":
			p = google.New(provCfg.APIKey, provCfg.BaseURL)
		case "mistral":
			m := mistral.New(provCfg.APIKey, provCfg.BaseURL)
			embedders = append(embedders, m)
			p = m
		case "groq":
			p = groq.New(provCfg.APIKey, provCfg.BaseURL)
			transcribers = append([]provider.TranscriptionProvider{
				groq.NewTranscriber(provCfg.APIKey, provCfg.BaseURL),
			}, transcribers...)
		case "xai":
			p = xai.New(provCfg.APIKey, provCfg.BaseURL)
		case "deepseek":
			p = deepseek.New(provCfg.APIKey, provCfg.BaseURL)
		case "openrouter":
			p = openrouter.New(provCfg.APIKey, provCfg.BaseURL)
		case "together":
			p = together.New(provCfg.APIKey, provCfg.BaseURL)
		case "minimax":
			p = minimax.New(provCfg.APIKey)
		case "netwrck":
			p = netwrck.New(provCfg.APIKey, provCfg.BaseURL)
		case "nous":
			p = nous.New(provCfg.APIKey, provCfg.BaseURL)
		case "textgenerator":
			tg := textgenerator.New(provCfg.APIKey)
			embedders = append(embedders, tg)
			log.Printf("Registered embedding provider: textgenerator")
			continue
		case "zai":
			p = zai.New(provCfg.APIKey, provCfg.BaseURL)
		case "fal":
			f := fal.New(provCfg.APIKey)
			transcribers = append(transcribers, f)
			p = f
		default:
			log.Printf("Unknown provider: %s", provCfg.Name)
			continue
		}

		registry.Register(p)
		log.Printf("Registered provider: %s", provCfg.Name)
	}

	modelRouter := router.New(registry, cfg.Models)

	if len(embedders) > 0 {
		ar := router.NewAutoRouter(embedders[0])
		if err := ar.Init(ctx); err != nil {
			log.Printf("AutoRouter init failed (will use static routing): %v", err)
		} else {
			modelRouter.SetAutoRouter(ar)
			log.Printf("AutoRouter enabled with embedding-based model selection")
		}
	}

	pricingTable := billing.NewPricingTable(cfg.Models)
	billingEngine := billing.NewEngine(pricingTable, creditQ)

	var stripe *stripesvc.Service
	if cfg.Stripe.SecretKey != "" {
		stripe = stripesvc.NewService(cfg.Stripe.SecretKey)
		topupQ := queries.NewAutotopupQueries(database.Pool)
		autoTopup := billing.NewAutoTopupService(userQ, creditQ, topupQ, stripe, billingEngine)
		billingEngine.SetAutoTopup(autoTopup)
		log.Printf("Stripe auto-topup enabled")
	}

	collector := metrics.NewCollector(usageQ, 0)
	collector.Start()
	defer collector.Stop()
	recorder := metrics.NewRecorder(collector)

	var store storage.Store
	if cfg.Storage.Provider == "r2" && cfg.Storage.R2AccessKey != "" {
		store = storage.NewR2Store(storage.R2Config{
			Endpoint:  cfg.Storage.R2Endpoint,
			Bucket:    cfg.Storage.R2Bucket,
			AccessKey: cfg.Storage.R2AccessKey,
			SecretKey: cfg.Storage.R2SecretKey,
			PublicURL: cfg.Storage.R2PublicURL,
		})
	} else {
		baseURL := fmt.Sprintf("http://localhost:%d", cfg.Server.Port)
		if v := os.Getenv("APP_URL"); v != "" {
			baseURL = v
		}
		s, err := storage.NewLocalStore(cfg.Storage.LocalDir, baseURL)
		if err != nil {
			log.Fatalf("Failed to init storage: %v", err)
		}
		store = s
	}

	var cryptoSvc *crypto.Service
	if cfg.Crypto.Enabled {
		cryptoSvc = initCrypto(cfg, database, billingEngine)
	}

	modelMetaQ := queries.NewModelMetadataQueries(database.Pool)
	ftQ := queries.NewFineTuneQueries(database.Pool)
	disc := discovery.New(cfg.Providers, modelMetaQ)

	ftProviders := make(map[string]provider.FineTuneProvider)
	for _, provCfg := range cfg.Providers {
		if !provCfg.Enabled || provCfg.APIKey == "" {
			continue
		}
		switch provCfg.Name {
		case "mistral":
			ftProviders["mistral"] = mistral.New(provCfg.APIKey, provCfg.BaseURL)
		}
	}
	if len(ftProviders) > 0 {
		log.Printf("Fine-tuning providers: %d", len(ftProviders))
	}

	go func() {
		n, err := disc.DiscoverAll(ctx)
		if err != nil {
			log.Printf("Initial model discovery failed: %v", err)
		} else {
			log.Printf("Model discovery: indexed %d models", n)
		}
	}()

	codexRefresher := cron.NewCodexRefresher(providerKeyQ)
	codexRefresher.Start()
	defer codexRefresher.Stop()

	srv := server.New(&server.Dependencies{
		Config:       cfg,
		Router:       modelRouter,
		Billing:      billingEngine,
		Recorder:     recorder,
		JWTService:   jwtService,
		UserQ:        userQ,
		APIKeyQ:      apiKeyQ,
		CreditQ:      creditQ,
		StatsQ:       statsQ,
		Transcribers: transcribers,
		Embedders:    embedders,
		CryptoSvc:    cryptoSvc,
		Storage:      store,
		StripeSvc:    stripe,
		Discovery:      disc,
		ModelMetaQ:     modelMetaQ,
		FineTuneQ:      ftQ,
		FineTuneProvs:  ftProviders,
		ProviderKeyQ:   providerKeyQ,
	})

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	log.Println("OpenPath is running. Press Ctrl+C to stop.")
	<-done

	log.Println("Shutting down...")
	if cryptoSvc != nil {
		cryptoSvc.Stop()
	}
	if err := srv.Shutdown(); err != nil {
		log.Printf("Shutdown error: %v", err)
	}
}

func initCrypto(cfg *config.Config, database *db.DB, billingEngine *billing.Engine) *crypto.Service {
	seed, err := crypto.DecodeHex(cfg.Crypto.HDWalletSeed)
	if err != nil || len(seed) < 32 {
		log.Printf("HD wallet seed not configured, generating random seed")
		seed = make([]byte, 64)
		rand.Read(seed)
	}

	endpoints := make([]*crypto.RPCEndpoint, 0, 2)
	if cfg.Crypto.HeliusAPIKey != "" {
		endpoints = append(endpoints, &crypto.RPCEndpoint{
			Name:         "helius",
			URL:          fmt.Sprintf("https://mainnet.helius-rpc.com/?api-key=%s", cfg.Crypto.HeliusAPIKey),
			RateLimitRPS: 50.0,
		})
		log.Printf("Helius RPC configured (primary)")
	}
	endpoints = append(endpoints, &crypto.RPCEndpoint{
		Name:         "mainnet",
		URL:          cfg.Crypto.SolanaRPCURL,
		RateLimitRPS: 10.0,
	})

	rpcClient := crypto.NewRPCClient(endpoints)
	priceFeed := crypto.NewPriceFeed(cfg.Crypto.CodexTokenMint, cfg.Crypto.BagsAPIKey)
	priceFeed.Start()

	cryptoQ := queries.NewCryptoQueries(database.Pool)
	svc := crypto.NewService(crypto.Config{
		Enabled:        true,
		WalletPubkey:   cfg.Crypto.WalletPubkey,
		HDWalletSeed:   seed,
		CodexTokenMint: cfg.Crypto.CodexTokenMint,
		BagsAPIKey:     cfg.Crypto.BagsAPIKey,
		MinTopupUSD:    cfg.Crypto.MinTopupUSD,
	}, cryptoQ, billingEngine, rpcClient, priceFeed)
	svc.Start()

	log.Printf("Crypto payments enabled (wallet=%s)", cfg.Crypto.WalletPubkey)
	return svc
}
