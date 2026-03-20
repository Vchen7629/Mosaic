package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	at "mosaic-client.com/gen/audio_transcription"
	cb "mosaic-client.com/gen/conversation_briefing"
	fd "mosaic-client.com/gen/face_detection"
	"mosaic-client.com/internal/handler"
	"mosaic-client.com/internal/middleware"
)

type Config struct {
	ServerPort  string 	`envconfig:"SERVER_PORT" default:"8080"`
	ProdMode 	bool 	`envconfig:"PROD_MODE" default:"false"`
}

func WebsocketServer(
	cfg *Config,
	logger *slog.Logger,
	audio_client at.AudioTranscriptionServiceClient,
	face_client fd.FaceDetectionServiceClient,
	briefing_client cb.ConversationBriefingServiceClient,
) {
	router := http.NewServeMux()

	wsHandler := &handler.WebSocketHandler{
		Logger: logger,
		AudioClient: audio_client,
		FaceClient:  face_client,
		BriefingClient: briefing_client,
	}

	router.HandleFunc("/api/v1/ws", wsHandler.HandleWebSocket)

	server := http.Server{
		Addr: fmt.Sprintf(":%s", cfg.ServerPort),
		Handler: middleware.Logging(router),
	}

	logger.Debug("[client] Server running on http://localhost:8000")
	err := server.ListenAndServe()
	if err != nil {
		logger.Error("HTTP Server failed to start", "err", err)
		os.Exit(1)
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
	logger := slog.New(handler).With("service", "client")

	logger.Info("Starting Mosaic backend server...")

	audioConn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	faceConn, err := grpc.NewClient("localhost:40040", grpc.WithTransportCredentials(insecure.NewCredentials()))
	briefingConn, err := grpc.NewClient("localhost:30030", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("Unable to start gRPC client")
		os.Exit(1)
	}

	atClient := at.NewAudioTranscriptionServiceClient(audioConn)
	fdClient := fd.NewFaceDetectionServiceClient(faceConn)
	cbClient := cb.NewConversationBriefingServiceClient(briefingConn)

	go WebsocketServer(cfg, logger, atClient, fdClient, cbClient)

	// go channel for listening to sigint/sigterm signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	// trigger sigChan channel if app recieves either SIGTERM or SIGINT indicating it should shutdown
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Block until shutdown signal recieved
	<-sigChan
	logger.Info("Shutting down gracefully...")

	audioConn.Close()
	faceConn.Close()
	logger.Debug("Closed gRPC connection")
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
