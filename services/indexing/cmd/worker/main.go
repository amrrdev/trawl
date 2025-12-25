package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/amrrdev/trawl/services/indexing/internal/queue"
	"github.com/amrrdev/trawl/services/indexing/internal/scylladb"
	"github.com/amrrdev/trawl/services/indexing/internal/worker"
	sharedQueue "github.com/amrrdev/trawl/services/shared/queue"
	"github.com/amrrdev/trawl/services/shared/storage"
	"github.com/lpernett/godotenv"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := godotenv.Load("../../.env"); err != nil {
		log.Println("Warning: .env file not found, using defaults")
	}

	minioEndpoint := getEnv("MINIO_ENDPOINT", "localhost:9000")
	minioAccessKey := getEnv("MINIO_ACCESS_KEY", "minioadmin")
	minioSecretKey := getEnv("MINIO_SECRET_KEY", "minioadmin123")
	minioBucket := getEnv("MINIO_BUCKET", "trawl-documents")
	rabbitmqURL := getEnv("RABBITMQ_URL", "amqp://rabbitmq_user:rabbitmq_password@localhost:5672/")
	indexingQueue := getEnv("RABBITMQ_INDEXING_QUEUE", "indexing_queue")
	dlqName := getEnv("RABBITMQ_DLQ", "indexing_dlq")
	scyllaHostsStr := getEnv("SCYLLADB_HOSTS", "127.0.0.1:9042")
	scyllaHosts := strings.Split(scyllaHostsStr, ",")

	// Initialize MinIO storage
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

	// Initialize ScyllaDB
	session, err := scylladb.Connect(scyllaHosts...)
	if err != nil {
		log.Fatalf("Failed to connect to ScyllaDB cluster: %v", err)
	}
	defer session.Close()
	log.Println("✓ Connected to ScyllaDB")

	// Initialize RabbitMQ
	rabbitClient, err := sharedQueue.NewRabbitMQ(rabbitmqURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rabbitClient.Close()
	log.Println("✓ Connected to RabbitMQ")

	// Initialize queue consumer
	consumer, err := queue.NewConsumer(rabbitClient, indexingQueue, dlqName)
	if err != nil {
		log.Fatalf("Failed to initialize consumer: %v", err)
	}
	defer consumer.Close()

	// Initialize worker
	indexingWorker := worker.NewIndexingWorker(consumer, storageClient, session)

	// Start the worker
	log.Println("🚀 Starting indexing worker...")
	if err := indexingWorker.Start(ctx); err != nil {
		log.Fatalf("Worker stopped with error: %v", err)
	}

	log.Println("👋 Worker shut down gracefully")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
