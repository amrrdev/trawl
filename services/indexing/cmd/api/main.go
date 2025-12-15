package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/amrrdev/trawl/services/indexing/internal/handler"
	"github.com/amrrdev/trawl/services/indexing/internal/server"
	"github.com/amrrdev/trawl/services/indexing/internal/service"
	"github.com/amrrdev/trawl/services/shared/jwt"
	"github.com/amrrdev/trawl/services/shared/middleware"
	"github.com/amrrdev/trawl/services/shared/storage"
	"github.com/lpernett/godotenv"
)

func main() {
	ctx := context.Background()

	// Load environment variables
	if err := godotenv.Load(".env"); err != nil {
		log.Println("Warning: .env file not found, using defaults")
	}

	// Load config from environment
	jwtSecret := getEnv("JWT_SECRET_KEY", "supersecretkey123")
	minioEndpoint := getEnv("MINIO_ENDPOINT", "localhost:9000")
	minioAccessKey := getEnv("MINIO_ACCESS_KEY", "minioadmin")
	minioSecretKey := getEnv("MINIO_SECRET_KEY", "minioadmin123")
	minioBucket := getEnv("MINIO_BUCKET", "trawl-documents")
	port := getEnv("INDEXING_PORT", ":8003")

	storageClient, err := storage.NewStorage(ctx, &storage.Config{
		Endpoint:  minioEndpoint,
		AccessKey: minioAccessKey,
		SecretKey: minioSecretKey,
		Bucket:    minioBucket,
		UseSSL:    false,
	})
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	log.Println("✓ Connected to MinIO")

	jwtService := jwt.NewService(jwtSecret, 24*time.Hour)
	authMiddleware := middleware.NewAuthMiddleware(jwtService)

	documentService := service.NewDocument(storageClient)
	documentHandler := handler.NewDocumentHandler(documentService)

	g := server.NewServer(documentHandler, authMiddleware)

	log.Printf("🚀 Indexing service starting on %s", port)
	if err := g.Run(port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
