package main

import (
	"context"
	"flag"
	"log"

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

	runner := cron.NewEvalRunner(
		queries.NewEvalQueries(database.Pool),
		queries.NewAPIKeyQueries(database.Pool),
		queries.NewUserQueries(database.Pool),
		queries.NewCreditQueries(database.Pool),
		cfg.Models,
	)
	if runner == nil {
		log.Fatal("eval runner init failed")
	}
	runner.RunAsync()
	<-runner.Done()
	log.Printf("live evals sweep complete")
}
