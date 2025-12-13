package main

import (
	"context"
	"log"

	"github.com/amrrdev/trawl/services/auth/internal/config"
	"github.com/amrrdev/trawl/services/auth/internal/database"

	"github.com/amrrdev/trawl/services/auth/internal/db"
)

func main() {
	ctx := context.Background()

	config, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	database, err := database.Connect(ctx, config.DatabaseUrl, database.DefaultConfig())
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	if err := database.HealthCheck(ctx); err != nil {
		log.Printf("Health check failed: %v", err)
	}

	stats := database.Stats()
	log.Printf("Active connections: %d", stats.AcquiredConns())

	queries := db.New(database.Pool)
	_ = queries
}
