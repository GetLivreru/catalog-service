package main

import (
	"context"
	"fmt"
	"log"

	"github.com/GetLivreru/catalog-service/internal/app/config"
	rhealth "github.com/GetLivreru/catalog-service/internal/app/handler/http/health"
	rprocessor "github.com/GetLivreru/catalog-service/internal/app/processor/http"
	rcpostgres "github.com/GetLivreru/catalog-service/internal/app/repository/conn/postgres"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx := context.Background()
	config.Load()

	cfg := config.Root
	pgClient, err := rcpostgres.NewClient(ctx, cfg.Repository.Postgres)
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}
	defer func() {
		if err := pgClient.GetRawBunDB().Close(); err != nil {
			log.Printf("Failed to close PostgreSQL: %v", err)
		}
	}()

	oldVer, newVer, err := pgClient.Migrate(ctx)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	if oldVer != newVer {
		log.Printf("Database migrated old_version=%d new_version=%d", oldVer, newVer)
	} else {
		log.Printf("Database is up to date version=%d", newVer)
	}

	hHealth := rhealth.NewHandler()

	httpServer := rprocessor.NewHTTP(hHealth, cfg.Processor.WebServer)
	if err := httpServer.Serve(); err != nil {
		return fmt.Errorf("HTTP server failed: %w", err)
	}
	return nil
}
