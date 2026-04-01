package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/valkey-io/valkey-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	cb "mosaic-conversation-briefing.com/gen"
	"mosaic-conversation-briefing.com/internal/db"
	"mosaic-conversation-briefing.com/internal/handler"
	"mosaic-conversation-briefing.com/internal/observability"
)

type Config struct {
	ServerPort        string        `envconfig:"SERVER_PORT" default:"30030"`
	MetricsPort       string        `envconfig:"METRICS_PORT" default:"9090"`
	CacheURL          string        `envconfig:"CACHE_URL" default:""`
	DatabaseURL       string        `envconfig:"DATABASE_URL" default:""`
	LLMBaseURL        string        `envconfig:"OLLAMA_BASE_URL" default:""`
	ProdMode          bool          `envconfig:"PROD_MODE" default:"false"`
	MaxConnectionIdle time.Duration `envconfig:"GRPC_MAX_CONN_IDLE" default:"30s"`
}

func (cfg *Config) serverOptions() []grpc.ServerOption {
	return []grpc.ServerOption{
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: cfg.MaxConnectionIdle,
			Time:              15 * time.Second,
			Timeout:           5 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	}
}

func gRPCServer(logger *slog.Logger, cfg *Config, client valkey.Client, pool *pgxpool.Pool) (*grpc.Server, error) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.ServerPort))
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	dbPool := db.NewDBPool(pool, logger)

	gRPCServer := grpc.NewServer(cfg.serverOptions()...)
	cb.RegisterConversationBriefingServiceServer(
		gRPCServer, handler.NewConvoBriefingServer(logger, client, dbPool, cfg.LLMBaseURL),
	)

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(gRPCServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	go func() {
		logger.Info("conversation briefing gRPC server listening on", "port", cfg.ServerPort)
		err = gRPCServer.Serve(lis)
		if err != nil {
			logger.Error("failed to serve gRPC server", "err", err)
			os.Exit(1)
		}
	}()

	return gRPCServer, nil
}

// exposes a /metrics prometheus and /ready endpoint for readiness
func observabilityServer(logger *slog.Logger, cfg *Config, pool *pgxpool.Pool) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		err := pool.Ping(ctx)
		if err != nil {
			http.Error(w, "service not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	mux.Handle("/metrics", promhttp.Handler())
	logger.Info("metrics server listening", "port", cfg.MetricsPort)
	err := http.ListenAndServe(":"+cfg.MetricsPort, mux)
	if err != nil {
		logger.Error("metrics server failed to start", "err", err)
	}
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("failed to load config values: %v", err)
	}

	logger := observability.StructuredLogger(cfg.ProdMode)

	logger.Info("Starting gRPC server...")

	pool := db.ConnectionPool(logger, cfg.DatabaseURL)

	defer pool.Close()

	client, err := valkey.NewClient(valkey.ClientOption{InitAddress: []string{cfg.CacheURL}})
	if err != nil {
		logger.Warn("error initializing caching client", "err", err)
	}

	server, err := gRPCServer(logger, cfg, client, pool)
	if err != nil {
		logger.Error("failed to start gRPC server", "err", err)
		os.Exit(1)
	}

	observability.RegisterMetrics()

	go observabilityServer(logger, cfg, pool)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Block until shutdown signal recieved
	<-sigChan
	logger.Info("Shutting down gracefully...")

	server.GracefulStop()
	pool.Close()
	if client != nil {
		client.Close()
	}

	logger.Info("Closed gRPC and dbpool connection")
}

// method to load config values
func loadConfig() (*Config, error) {
	godotenv.Load("../.env")

	var cfg Config

	err := envconfig.Process("", &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
