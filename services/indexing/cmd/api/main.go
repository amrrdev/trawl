package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/amrrdev/trawl/services/indexing/internal/handler"
	"github.com/amrrdev/trawl/services/indexing/internal/queue"
	"github.com/amrrdev/trawl/services/indexing/internal/server"
	"github.com/amrrdev/trawl/services/indexing/internal/service"
	"github.com/amrrdev/trawl/services/shared/jwt"
	"github.com/amrrdev/trawl/services/shared/middleware"
	sharedQueue "github.com/amrrdev/trawl/services/shared/queue"
	"github.com/amrrdev/trawl/services/shared/storage"
	"github.com/lpernett/godotenv"
)

func main() {
	ctx := context.Background()

	// Load environment variables from root (2 levels up from cmd directory)
	if err := godotenv.Load("../../.env"); err != nil {
		log.Println("Warning: .env file not found, using defaults")
	}

	// Load config from environment
	jwtSecret := getEnv("JWT_SECRET_KEY", "very-secret-key")
	minioEndpoint := getEnv("MINIO_ENDPOINT", "localhost:9000")
	minioAccessKey := getEnv("MINIO_ACCESS_KEY", "minioadmin")
	minioSecretKey := getEnv("MINIO_SECRET_KEY", "minioadmin123")
	minioBucket := getEnv("MINIO_BUCKET", "trawl-documents")
	rabbitmqURL := getEnv("RABBITMQ_URL", "amqp://rabbitmq_user:rabbitmq_password@localhost:5672/")
	indexingQueue := getEnv("RABBITMQ_INDEXING_QUEUE", "indexing_queue")
	port := getEnv("INDEXING_PORT", ":8003")

	// Connect to MinIO
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

	// Connect to RabbitMQ
	rabbitClient, err := sharedQueue.NewRabbitMQ(rabbitmqURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rabbitClient.Close()
	log.Println("✓ Connected to RabbitMQ")

	// Initialize producer
	producer, err := queue.NewProducer(rabbitClient, indexingQueue)
	if err != nil {
		log.Fatalf("Failed to initialize producer: %v", err)
	}

	jwtService := jwt.NewService(jwtSecret, 24*time.Hour)
	authMiddleware := middleware.NewAuthMiddleware(jwtService)

	documentService := service.NewDocument(storageClient, producer)
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
