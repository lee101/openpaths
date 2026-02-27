package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/openpath/openpath/internal/auth"
	"github.com/openpath/openpath/internal/billing"
	"github.com/openpath/openpath/internal/config"
	"github.com/openpath/openpath/internal/db"
	"github.com/openpath/openpath/internal/db/migrations"
	"github.com/openpath/openpath/internal/db/queries"
	"github.com/openpath/openpath/internal/metrics"
	"github.com/openpath/openpath/internal/provider"
	"github.com/openpath/openpath/internal/provider/anthropic"
	"github.com/openpath/openpath/internal/provider/google"
	"github.com/openpath/openpath/internal/provider/groq"
	"github.com/openpath/openpath/internal/provider/mistral"
	"github.com/openpath/openpath/internal/provider/openai"
	"github.com/openpath/openpath/internal/router"
	"github.com/openpath/openpath/internal/server"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to config file")
	flag.Parse()

	// Load config
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid config: %v", err)
	}

	ctx := context.Background()

	// Connect to database
	database, err := db.New(ctx, cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Run migrations
	if err := migrations.Run(ctx, database.Pool); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize query layers
	userQ := queries.NewUserQueries(database.Pool)
	apiKeyQ := queries.NewAPIKeyQueries(database.Pool)
	creditQ := queries.NewCreditQueries(database.Pool)
	usageQ := queries.NewUsageQueries(database.Pool)
	statsQ := queries.NewStatsQueries(database.Pool)

	// Initialize auth services
	jwtService := auth.NewJWTService(cfg.JWT.Secret, cfg.JWT.ExpirationHours)

	// Register providers
	registry := provider.NewRegistry()
	for _, provCfg := range cfg.Providers {
		if !provCfg.Enabled || provCfg.APIKey == "" {
			log.Printf("Skipping provider %s (disabled or no API key)", provCfg.Name)
			continue
		}

		var p provider.Provider
		switch provCfg.Name {
		case "openai":
			p = openai.New(provCfg.APIKey, provCfg.BaseURL)
		case "anthropic":
			p = anthropic.New(provCfg.APIKey, provCfg.BaseURL)
		case "google":
			p = google.New(provCfg.APIKey, provCfg.BaseURL)
		case "mistral":
			p = mistral.New(provCfg.APIKey, provCfg.BaseURL)
		case "groq":
			p = groq.New(provCfg.APIKey, provCfg.BaseURL)
		default:
			log.Printf("Unknown provider: %s", provCfg.Name)
			continue
		}

		registry.Register(p)
		log.Printf("Registered provider: %s", provCfg.Name)
	}

	// Initialize router
	modelRouter := router.New(registry, cfg.Models)

	// Initialize billing
	pricingTable := billing.NewPricingTable(cfg.Models)
	billingEngine := billing.NewEngine(pricingTable, creditQ)

	// Initialize metrics
	collector := metrics.NewCollector(usageQ, 0)
	collector.Start()
	defer collector.Stop()
	recorder := metrics.NewRecorder(collector)

	// Create and start server
	srv := server.New(&server.Dependencies{
		Config:     cfg,
		Router:     modelRouter,
		Billing:    billingEngine,
		Recorder:   recorder,
		JWTService: jwtService,
		UserQ:      userQ,
		APIKeyQ:    apiKeyQ,
		CreditQ:    creditQ,
		StatsQ:     statsQ,
	})

	// Graceful shutdown
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
	if err := srv.Shutdown(); err != nil {
		log.Printf("Shutdown error: %v", err)
	}
}
