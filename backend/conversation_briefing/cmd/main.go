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
	cb "mosaic-conversation-briefing.com/gen"
	"mosaic-conversation-briefing.com/internal/db"
	"mosaic-conversation-briefing.com/internal/handler"
)

type Config struct {
	ServerPort  string `envconfig:"SERVER_PORT" default:"30030"`
	DatabaseURL string `envconfig:"DATABASE_URL" default:""`
	LLMBaseURL  string `envconfig:"OLLAMA_BASE_URL" default:""`
}

func gRPCServer(cfg *Config, pool *pgxpool.Pool) (*grpc.Server, error) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.ServerPort))
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	dbPool := db.NewDBPool(pool)

	gRPCServer := grpc.NewServer()
	cb.RegisterConversationBriefingServiceServer(
		gRPCServer, handler.NewConvoBriefingServer(dbPool, cfg.LLMBaseURL),
	)

	go func() {
		log.Printf("conversation briefing gRPC server listening on: %s", cfg.ServerPort)
		err = gRPCServer.Serve(lis)
		if err != nil {
			log.Fatalf("failed to serve gRPC server: %v", err)
		}
	}()

	return gRPCServer, nil
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("failed to load config values: %v", err)
	}

	log.Println("Starting Service...")
	pool := db.ConnectionPool(cfg.DatabaseURL)

	defer pool.Close()

	server, err := gRPCServer(cfg, pool)
	if err != nil {
		log.Fatalf("failed to start gRPC server: %v", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Block until shutdown signal recieved
	<-sigChan
	log.Println("Shutting down gracefully...")

	server.GracefulStop()
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
