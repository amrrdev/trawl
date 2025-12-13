package main

import (
	"flag"
	"log"

	"github.com/amrrdev/trawl/services/auth/internal/config"
	"github.com/amrrdev/trawl/services/auth/internal/database"
)

func main() {
	var (
		direction = flag.String("direction", "up", "Migration direction: up or down")
		steps     = flag.Int("steps", 0, "Number of steps to rollback (only for down)")
	)
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	switch *direction {
	case "up":
		log.Println("Running migrations...")
		if err := database.RunMigrations(cfg.DatabaseUrl); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
		log.Println("✅ Migrations completed successfully")
	case "down":
		log.Printf("Rolling back %d migrations...\n", *steps)
		if err := database.RollbackMigrations(cfg.DatabaseUrl, *steps); err != nil {
			log.Fatalf("Rollback failed: %v", err)
		}
		log.Println("✅ Rollback completed successfully")
	}
}
