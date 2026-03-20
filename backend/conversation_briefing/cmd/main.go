package main

import (
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	cb "mosaic-conversation-briefing.com/gen"
	"mosaic-conversation-briefing.com/internal/db"
	"mosaic-conversation-briefing.com/internal/handler"
	"mosaic-conversation-briefing.com/internal/observability"
)

type Config struct {
	ServerPort  string `envconfig:"SERVER_PORT" default:"30030"`
	MetricsPort  string `envconfig:"METRICS_PORT" default:"8080"`
	DatabaseURL string `envconfig:"DATABASE_URL" default:""`
	LLMBaseURL  string `envconfig:"OLLAMA_BASE_URL" default:""`
	ProdMode 	bool   `envconfig:"PROD_MODE" default:"false"`
}

func gRPCServer(logger *slog.Logger, cfg *Config, pool *pgxpool.Pool) (*grpc.Server, error) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.ServerPort))
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	dbPool := db.NewDBPool(pool, logger)

	gRPCServer := grpc.NewServer()
	cb.RegisterConversationBriefingServiceServer(
		gRPCServer, handler.NewConvoBriefingServer(logger, dbPool, cfg.LLMBaseURL),
	)

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(gRPCServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	go func() {
		logger.Info("conversation briefing gRPC server listening on", "port", cfg.ServerPort)
		err = gRPCServer.Serve(lis)
		if err != nil {
			logger.Error("failed to serve gRPC server", "err", err,)
			os.Exit(1)
		}
	}()

	return gRPCServer, nil
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
	logger := slog.New(handler).With("service", "conversation_briefing")

	logger.Info("Starting gRPC server...")

	pool := db.ConnectionPool(logger, cfg.DatabaseURL)

	defer pool.Close()

	server, err := gRPCServer(logger, cfg, pool)
	if err != nil {
		logger.Error("failed to start gRPC server", "err", err)
		os.Exit(1)
	}

	observability.RegisterMetrics()

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		logger.Info("metrics server listening", "port", cfg.MetricsPort)
		err = http.ListenAndServe(":"+cfg.MetricsPort, mux)
		if err != nil {
			logger.Error("metrics server failed to start", "err", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Block until shutdown signal recieved
	<-sigChan
	logger.Info("Shutting down gracefully...")

	server.GracefulStop()
	pool.Close()

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
