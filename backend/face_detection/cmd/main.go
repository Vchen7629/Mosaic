package main

import (
	"fmt"
	"log"
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
	ServerPort 	string 	`envconfig:"SERVER_PORT" default:"40040"`
	DatabaseURL string 	`envconfig:"DATABASE_URL" default:""`
	ModelsDir 	string 	`envconfig:"MODELS_DIR" default:"models"`
	RecPoolSize int		`envconfig:"REC_POOL_SIZE" default:"5"`	
}

// handles starting the gRPC server
func gRPCServer(
	cfg *Config,
	recPool *service.RecognizerPool, 
	pool *pgxpool.Pool,
) (*grpc.Server, error) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.ServerPort))
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	dbPool := db.NewDBPool(pool)

	grpcServer := grpc.NewServer()
	fd.RegisterFaceDetectionServiceServer(
		grpcServer, handler.NewFaceDetectionServer(recPool, dbPool),
	)

	go func() {
		log.Printf("face detection gRPC server listening on: %s", cfg.ServerPort)
		err = grpcServer.Serve(lis)
		if err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	return grpcServer, nil
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("failed to load config values: %v", err)
	}

	log.Println("Starting Service...")
	pool := db.ConnectionPool(cfg.DatabaseURL)

	defer pool.Close()

	recPool, err := service.NewRecognizerPool(cfg.ModelsDir, cfg.RecPoolSize)
	if err != nil {
		log.Fatalf("failed to init recognizer pool: %v", err)
	}
	
	defer recPool.Close()

	grpcServer, err := gRPCServer(cfg, recPool, pool)
	if err != nil {
		log.Fatalf("failed to start gRPC server: %v", err)
	}

	// go channel for listening to sigint/sigterm signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	// trigger sigChan channel if app recieves either SIGTERM or SIGINT indicating it should shutdown
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Block until shutdown signal recieved
	<-sigChan
	log.Println("Shutting down gracefully...")

	grpcServer.GracefulStop()
	pool.Close()

	log.Println("Closed gRPC and dbpool connection")
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