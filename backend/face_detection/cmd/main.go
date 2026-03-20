package main

import (
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"google.golang.org/grpc"
	fd "mosaic-face-detection.com/gen"
	"mosaic-face-detection.com/internal/db"
	"mosaic-face-detection.com/internal/handler"
	"mosaic-face-detection.com/internal/service"
)

type Config struct {
	ServerPort  string `envconfig:"SERVER_PORT" default:"40040"`
	DatabaseURL string `envconfig:"DATABASE_URL" default:""`
	ModelsDir   string `envconfig:"MODELS_DIR" default:"models"`
	RecPoolSize int    `envconfig:"REC_POOL_SIZE" default:"5"`
	ProdMode	bool   `envconfig:"PROD_MODE" default:"false"`
}

// handles starting the gRPC server
func gRPCServer(
	logger *slog.Logger,
	cfg *Config,
	recPool *service.RecognizerPool,
	pool *pgxpool.Pool,
) (*grpc.Server, error) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.ServerPort))
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	dbPool := db.NewDBPool(pool, logger)

	grpcServer := grpc.NewServer()
	fd.RegisterFaceDetectionServiceServer(
		grpcServer, handler.NewFaceDetectionServer(logger, recPool, dbPool),
	)

	go func() {
		logger.Info("gRPC server listening on", "port", cfg.ServerPort)
		err = grpcServer.Serve(lis)
		if err != nil {
			logger.Error("failed to serve gRPC server", "err", err,)
			os.Exit(1)
		}
	}()

	return grpcServer, nil
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

	grpcServer, err := gRPCServer(logger, cfg, recPool, pool)
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
