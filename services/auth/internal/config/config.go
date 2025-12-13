package config

import (
	"os"

	"github.com/lpernett/godotenv"
)

type Config struct {
	DatabaseUrl string
}

func Load() (*Config, error) {
	err := godotenv.Load("../../.env")
	if err != nil {
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
