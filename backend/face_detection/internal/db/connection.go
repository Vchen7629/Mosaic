package db

import (
	"context"
	"github.com/jackc/pgx/v4/pgxpool"
	"log"
	"time"
)

type DBPool struct {
	pool *pgxpool.Pool
}

// allows us to initialize the db pool in main.go
func NewDBPool(pool *pgxpool.Pool) *DBPool {
	return &DBPool{pool: pool}
}

// set up pgxpool db connection pool
func ConnectionPool(database_url string) *pgxpool.Pool {
	pool, err := pgxpool.ConnectConfig(
		context.Background(), databaseConfig(database_url),
	)
	if err != nil {
		log.Fatalf("Unable to connect to database %v\n", err)
	}

	return pool
}

// Pgxpool postgres connection pool config values
func databaseConfig(database_url string) *pgxpool.Config {
	config, err := pgxpool.ParseConfig(database_url)
	if err != nil {
		log.Fatalf("Unable to parse DATABASE_URL %v\n", err)
	}

	config.MaxConns = 50
	config.MinConns = 5
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute
	config.HealthCheckPeriod = time.Minute

	return config
}
