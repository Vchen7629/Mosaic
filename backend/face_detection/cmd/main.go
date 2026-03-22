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
	fd "mosaic-face-detection.com/gen"
	"mosaic-face-detection.com/internal/db"
	"mosaic-face-detection.com/internal/handler"
	"mosaic-face-detection.com/internal/observability"
	"mosaic-face-detection.com/internal/service"
)

type Config struct {
	ServerPort  string `envconfig:"SERVER_PORT" default:"40040"`
	MetricsPort string `envconfig:"METRICS_PORT" default:"9092"`
	CacheURL    string `envconfig:"CACHE_URL" default:""`
	DatabaseURL string `envconfig:"DATABASE_URL" default:""`
	ModelsDir   string `envconfig:"MODELS_DIR" default:"models"`
	RecPoolSize int    `envconfig:"REC_POOL_SIZE" default:"5"`
	ProdMode    bool   `envconfig:"PROD_MODE" default:"false"`
}

// handles starting the gRPC server
func gRPCServer(
	logger *slog.Logger,
	cfg *Config,
	recPool *service.RecognizerPool,
	client valkey.Client,
	pool *pgxpool.Pool,
) (*grpc.Server, error) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.ServerPort))
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	dbPool := db.NewDBPool(pool, logger)

	grpcServer := grpc.NewServer()
	fd.RegisterFaceDetectionServiceServer(
		grpcServer, handler.NewFaceDetectionServer(logger, recPool, client, dbPool),
	)

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	go func() {
		logger.Info("gRPC server listening on", "port", cfg.ServerPort)
		err = grpcServer.Serve(lis)
		if err != nil {
			logger.Error("failed to serve gRPC server", "err", err)
			os.Exit(1)
		}
	}()

	return grpcServer, nil
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

	var handler slog.Handler
	if cfg.ProdMode {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	} else {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	}
	logger := slog.New(handler).With("service", "face_detection")

	logger.Info("Starting gRPC server...")
	pool := db.ConnectionPool(logger, cfg.DatabaseURL)

	defer pool.Close()

	recPool, err := service.NewRecognizerPool(cfg.ModelsDir, cfg.RecPoolSize)
	if err != nil {
		logger.Error("failed to init recognizer pool", "err", err)
		os.Exit(1)
	}

	defer recPool.Close()

	client, err := valkey.NewClient(valkey.ClientOption{InitAddress: []string{cfg.CacheURL}})
	if err != nil {
		logger.Warn("error initializing caching client", "err", err)
	}

	observability.RegisterMetrics()

	go observabilityServer(logger, cfg, pool)

	grpcServer, err := gRPCServer(logger, cfg, recPool, client, pool)
	if err != nil {
		logger.Error("failed to start gRPC server", "err", err)
		os.Exit(1)
	}

	// go channel for listening to sigint/sigterm signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	// trigger sigChan channel if app recieves either SIGTERM or SIGINT indicating it should shutdown
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Block until shutdown signal recieved
	<-sigChan
	logger.Info("Shutting down gracefully...")

	grpcServer.GracefulStop()
	pool.Close()
	if client != nil {
		client.Close()
	}

	logger.Info("Closed gRPC and dbpool connection")
}

// method to load config values
func loadConfig() (*Config, error) {
	err := godotenv.Load("../.env")
	if err != nil {
		return nil, err
	}
	var cfg Config

	err = envconfig.Process("", &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
