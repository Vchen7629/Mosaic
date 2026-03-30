package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/valkey-io/valkey-go"
)

type TestCacheContainer struct {
	Container testcontainers.Container
	Client    valkey.Client
}

func createTestCache(ctx context.Context) (*TestCacheContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        "valkey/valkey:8",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor: wait.ForLog("Ready to accept connections").
			WithStartupTimeout(30 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start Valkey container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container host: %w", err)
	}

	port, err := container.MappedPort(ctx, "6379")
	if err != nil {
		return nil, fmt.Errorf("failed to get container port: %w", err)
	}

	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress:  []string{fmt.Sprintf("%s:%s", host, port.Port())},
		DisableCache: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create valkey client: %w", err)
	}

	return &TestCacheContainer{
		Container: container,
		Client:    client,
	}, nil
}

// SetupTestCache creates a Valkey container and returns a client.
// Automatically registers cleanup with t.Cleanup().
func SetupTestCache(t *testing.T) *TestCacheContainer {
	t.Helper()

	ctx := context.Background()
	testCache, err := createTestCache(ctx)
	if err != nil {
		t.Fatalf("Failed to setup test cache: %v", err)
	}

	t.Cleanup(func() { testCache.Client.Close() })
	t.Cleanup(func() {
		if err := testCache.Container.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate Valkey container: %v", err)
		}
	})

	return testCache
}

// SetupTestCacheForTestMain creates a Valkey container for use in TestMain.
// Returns the container and a cleanup function that must be called manually.
func SetupTestCacheForTestMain() (*TestCacheContainer, func()) {
	ctx := context.Background()
	testCache, err := createTestCache(ctx)
	if err != nil {
		panic(fmt.Sprintf("Failed to setup test cache: %v", err))
	}

	cleanup := func() {
		if testCache.Client != nil {
			testCache.Client.Close()
		}
		if testCache.Container != nil {
			_ = testCache.Container.Terminate(context.Background())
		}
	}

	return testCache, cleanup
}

// FlushCache deletes all keys in the Valkey instance.
func FlushCache(t *testing.T, client valkey.Client) {
	t.Helper()

	err := client.Do(context.Background(), client.B().Flushall().Build()).Error()
	if err != nil {
		t.Fatalf("Failed to flush cache: %v", err)
	}
}
