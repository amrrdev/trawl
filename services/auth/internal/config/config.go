package config

import (
	"os"

	"github.com/lpernett/godotenv"
)

type Config struct {
	DatabaseUrl string
}

func Load() (*Config, error) {
	// Load .env from project root (2 levels up from services/auth)
	err := godotenv.Load("../../.env")
	if err != nil {
		// If .env doesn't exist, continue with environment variables
		// This is fine in Docker/production where env vars are set directly
		return &Config{
			DatabaseUrl: getEnvOrDefault("DATABASE_URL", "postgres://search-flow_user:search-flow_password@localhost:5432/search-flow-db?sslmode=disable"),
		}, nil
	}

	return &Config{
		DatabaseUrl: getEnvOrDefault("DATABASE_URL", "postgres://search-flow_user:search-flow_password@localhost:5432/search-flow-db?sslmode=disable"),
	}, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
