package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/openpaths/openpaths/internal/config"
	"github.com/openpaths/openpaths/internal/cron"
	"github.com/openpaths/openpaths/internal/db"
	"github.com/openpaths/openpaths/internal/db/migrations"
	"github.com/openpaths/openpaths/internal/db/queries"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()
	database, err := db.New(ctx, cfg.Database)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer database.Close()

	if err := migrations.Run(ctx, database.Pool); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	prober := cron.NewModelProber(queries.NewModelProbeQueries(database.Pool), cfg.Models)
	if prober == nil {
		log.Fatal("model prober init failed")
	}
	if os.Getenv("OPENPATHS_PROBE_API_KEY") == "" && os.Getenv("OPENPATHS_API_KEY") == "" && os.Getenv("APP_API_KEY") == "" {
		log.Fatal("set OPENPATHS_PROBE_API_KEY, OPENPATHS_API_KEY, or APP_API_KEY")
	}
	prober.RunOnce()
	log.Printf("model probe run complete")
}
