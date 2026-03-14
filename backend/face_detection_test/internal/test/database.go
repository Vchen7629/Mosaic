package test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type TestDBContainer struct {
	Container 	testcontainers.Container
	Pool		*pgxpool.Pool
	ConnStr		string
}

// core setup logic shared between SetupTestDatabase and SetupTestDatabaseForTestMain
func createTestDatabase(ctx context.Context) (*TestDBContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        "pgvector/pgvector:pg16",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_DB":       "testdb",
			"POSTGRES_USER":     "testuser",
			"POSTGRES_PASSWORD": "testpass",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start PostgreSQL container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container host: %w", err)
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		return nil, fmt.Errorf("failed to get container port: %w", err)
	}

	connStr := fmt.Sprintf("postgres://testuser:testpass@%s:%s/testdb?sslmode=disable",
		host, port.Port())

	// Retry connection with backoff (PostgreSQL may need extra time after log message)
	var pool *pgxpool.Pool
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		pool, err = pgxpool.Connect(ctx, connStr)
		if err == nil {
			break
		}
		if i < maxRetries-1 {
			time.Sleep(time.Second * time.Duration(i+1))
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database after %d retries: %w", maxRetries, err)
	}

	if err := runMigrations(ctx, pool); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &TestDBContainer{
		Container: container,
		Pool:      pool,
		ConnStr:   connStr,
	}, nil
}

// Creates a PostgreSQL container and returns a connection pool
// Automatically registers cleanup with t.Cleanup()
func SetupTestDatabase(t *testing.T) *TestDBContainer {
	t.Helper()

	ctx := context.Background()
	testDB, err := createTestDatabase(ctx)
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	// cleanup on test completion
	t.Cleanup(func() {
		if err := testDB.Container.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	})

	t.Cleanup(func() { testDB.Pool.Close() })

	return testDB
}

// SetupTestDatabaseForTestMain creates a database for use in TestMain
// Returns the container and a cleanup function that must be called manually
func SetupTestDatabaseForTestMain() (*TestDBContainer, func()) {
	ctx := context.Background()
	testDB, err := createTestDatabase(ctx)
	if err != nil {
		panic(fmt.Sprintf("Failed to setup test database: %v", err))
	}

	cleanup := func() {
		if testDB.Pool != nil {
			testDB.Pool.Close()
		}
		if testDB.Container != nil {
			_ = testDB.Container.Terminate(context.Background())
		}
	}

	return testDB, cleanup
}

// run the init.sql schema to create the tables
func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	schemaPath := filepath.Join("..", "..", "..", "database", "init.sql")
	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	_, err = pool.Exec(ctx, string(schema))
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	return nil
}

// truncate all tables for a fresh test state, resets data
func CleanupTables(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	ctx := context.Background()
	tables := []string{
		"patient",
		"visitor_face_embeddings",
		"conversation_records",
		"briefings",
	}

	for _, table := range tables {
		_, err := pool.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", table))
		if err != nil {
			t.Fatalf("Failed to truncate table %s: %v", table, err)
		}
	}
}