package main

import (
	"context"
	"fmt"
	"log"

	"github.com/GetLivreru/catalog-service/internal/app/config"
	hcategory "github.com/GetLivreru/catalog-service/internal/app/handler/http/category"
	rhealth "github.com/GetLivreru/catalog-service/internal/app/handler/http/health"
	hproduct "github.com/GetLivreru/catalog-service/internal/app/handler/http/product"
	rprocessor "github.com/GetLivreru/catalog-service/internal/app/processor/http"
	pcategory "github.com/GetLivreru/catalog-service/internal/app/repository/category"
	rcpostgres "github.com/GetLivreru/catalog-service/internal/app/repository/conn/postgres"
	pproduct "github.com/GetLivreru/catalog-service/internal/app/repository/product"
	scategory "github.com/GetLivreru/catalog-service/internal/app/service/category"
	sproduct "github.com/GetLivreru/catalog-service/internal/app/service/product"
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

	categoryRepo := pcategory.NewRepoFromPostgres(pgClient)
	productRepo := pproduct.NewRepoFromPostgres(pgClient)
	categorySvc := scategory.NewService(categoryRepo, productRepo)
	productSvc := sproduct.NewService(productRepo, categoryRepo)
	hCategory := hcategory.NewHandler(categorySvc)
	hProduct := hproduct.NewHandler(productSvc)
	hHealth := rhealth.NewHandler()

	httpServer := rprocessor.NewHTTP(hHealth, hCategory, hProduct, cfg.Processor.WebServer)
	if err := httpServer.Serve(); err != nil {
		return fmt.Errorf("HTTP server failed: %w", err)
	}
	return nil
}
