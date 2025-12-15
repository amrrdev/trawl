package main

import (
	"context"
	"log"

	"github.com/amrrdev/trawl/services/auth/internal/config"
	"github.com/amrrdev/trawl/services/auth/internal/database"
	"github.com/amrrdev/trawl/services/auth/internal/handler"
	"github.com/amrrdev/trawl/services/auth/internal/repository"
	"github.com/amrrdev/trawl/services/auth/internal/server"
	"github.com/amrrdev/trawl/services/auth/internal/services"
	"github.com/amrrdev/trawl/services/shared/jwt"
	"github.com/amrrdev/trawl/services/shared/middleware"
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

	repo := repository.NewUserRepository(database.Pool)
	jwtService := jwt.NewService(config.JWTSecretKey, config.AccessTokenTTL)
	hashingService := services.NewHashingService()
	authService := services.NewAuthService(repo, hashingService, jwtService)
	authHandler := handler.NewAuthHandler(authService)
	authMiddleware := middleware.NewAuthMiddleware(jwtService)

	g := server.NewServer(authHandler, authMiddleware)

	if err := g.Run(); err != nil {
		log.Fatal(err)
	}
}
