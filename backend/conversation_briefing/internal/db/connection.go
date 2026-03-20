package db

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

type DBPool struct {
	pool *pgxpool.Pool
	logger *slog.Logger
}

// allows us to initialize the db pool in main.go
func NewDBPool(pool *pgxpool.Pool, logger *slog.Logger) *DBPool {
	return &DBPool{pool: pool, logger: logger}
}

// set up pgxpool db connection pool
func ConnectionPool(logger *slog.Logger, database_url string) *pgxpool.Pool {
	pool, err := pgxpool.ConnectConfig(
		context.Background(), databaseConfig(logger, database_url),
	)
	if err != nil {
		logger.Error("Unable to connect to database", "err", err)
		os.Exit(1)
	}

	logger.Info("successfully created db connection pool")
	return pool
}

// Pgxpool postgres connection pool config values
func databaseConfig(logger *slog.Logger, database_url string) *pgxpool.Config {
	config, err := pgxpool.ParseConfig(database_url)
	if err != nil {
		logger.Error("Unable to parse DATABASE_URL", "err", err)
		os.Exit(1)
	}

	config.MaxConns = 50
	config.MinConns = 5
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute
	config.HealthCheckPeriod = time.Minute

	return config
}
