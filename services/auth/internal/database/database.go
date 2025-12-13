package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Database struct {
	Pool *pgxpool.Pool
}

type Config struct {
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}

func DefaultConfig() *Config {
	return &Config{
		MaxConns:          25,
		MinConns:          5,
		MaxConnLifetime:   time.Hour,
		MaxConnIdleTime:   time.Minute * 30,
		HealthCheckPeriod: time.Minute,
	}
}
func Connect(ctx context.Context, databaseUrl string, cfg *Config) (*Database, error) {

	config, err := pgxpool.ParseConfig(databaseUrl)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database URL: %w", err)
	}
	if cfg != nil {
		if cfg.MaxConns > 0 {
			config.MaxConns = cfg.MaxConns
		}
		if cfg.MinConns > 0 {
			config.MinConns = cfg.MinConns
		}
		if cfg.MaxConnLifetime > 0 {
			config.MaxConnLifetime = cfg.MaxConnLifetime
		}
		if cfg.MaxConnIdleTime > 0 {
			config.MaxConnIdleTime = cfg.MaxConnIdleTime
		}
		if cfg.HealthCheckPeriod > 0 {
			config.HealthCheckPeriod = cfg.HealthCheckPeriod
		}
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	return &Database{
		Pool: pool,
	}, nil
}

func (db *Database) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}

func (db *Database) HealthCheck(ctx context.Context) error {
	if db.Pool == nil {
		return fmt.Errorf("database pool is nil")
	}

	if err := db.Pool.Ping(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	return nil
}

func (db *Database) Stats() *pgxpool.Stat {
	if db.Pool == nil {
		return nil
	}
	return db.Pool.Stat()
}
