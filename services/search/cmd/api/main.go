package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/amrrdev/trawl/services/search/internal/handler"
	"github.com/amrrdev/trawl/services/search/internal/scylladb"
	"github.com/amrrdev/trawl/services/search/internal/server"
	"github.com/amrrdev/trawl/services/search/internal/service"
	"github.com/amrrdev/trawl/services/shared/jwt"
	"github.com/amrrdev/trawl/services/shared/middleware"
	"github.com/amrrdev/trawl/services/shared/storage"
	"github.com/lpernett/godotenv"
)

func main() {
	ctx := context.Background()

	if err := godotenv.Load("../../.env"); err != nil {
		log.Println("Warning: .env file not found, using defaults")
	}

	jwtSecret := getEnv("JWT_SECRET_KEY", "very-secret-key")
	port := getEnv("SEARCH_PORT", ":8004")
	minioEndpoint := getEnv("MINIO_ENDPOINT", "localhost:9000")
	minioAccessKey := getEnv("MINIO_ACCESS_KEY", "minioadmin")
	minioSecretKey := getEnv("MINIO_SECRET_KEY", "minioadmin123")
	minioBucket := getEnv("MINIO_BUCKET", "trawl-documents")
	scyllaHostsStr := getEnv("SCYLLADB_HOSTS", "127.0.0.1:9042")
	scyllaHosts := strings.Split(scyllaHostsStr, ",")

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

	session, err := scylladb.Connect(scyllaHosts...)
	if err != nil {
		log.Fatalf("Failed to connect to ScyllaDB cluster: %v", err)
	}
	defer session.Close()
	log.Println("✓ Connected to ScyllaDB")

	jwtService := jwt.NewService(jwtSecret, 24*time.Hour)
	authMiddleware := middleware.NewAuthMiddleware(jwtService)

	searchService := service.NewSearch(session, storageClient)
	searchHandler := handler.NewSearchHandler(searchService)

	g := server.NewServer(searchHandler, authMiddleware)

	log.Printf("🚀 Search service starting on %s", port)
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
