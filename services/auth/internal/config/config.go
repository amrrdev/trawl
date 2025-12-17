package config

import (
	"fmt"
	"os"
	"time"

	"github.com/lpernett/godotenv"
)

type Config struct {
	DatabaseUrl    string
	JWTSecretKey   string
	AccessTokenTTL time.Duration
}

func Load() (*Config, error) {
	// Load .env from root (2 levels up from cmd directory)
	err := godotenv.Load("../../.env")
	if err != nil {
		// Not fatal - will use defaults
		fmt.Println("Warning: .env file not found, using defaults")
	}

	ttl, err := time.ParseDuration(
		getEnvOrDefault("ACCESS_TOKEN_TTL", "1h"),
	)
	if err != nil {
		return nil, fmt.Errorf("invalid ACCESS_TOKEN_TTL")
	}

	return &Config{
		DatabaseUrl:    getEnvOrDefault("DATABASE_URL", "postgres://search-flow_user:search-flow_password@localhost:5432/search-flow-db?sslmode=disable"),
		JWTSecretKey:   getEnvOrDefault("JWT_SECRET_KEY", "very-secret-key"),
		AccessTokenTTL: ttl,
	}, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
