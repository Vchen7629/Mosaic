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
	at "mosaic-client.com/gen/audio_transcription"
	cb "mosaic-client.com/gen/conversation_briefing"
	fd "mosaic-client.com/gen/face_detection"
	"mosaic-client.com/internal/handler"
	"mosaic-client.com/internal/observability"
)

type Config struct {
	ServerPort string `envconfig:"SERVER_PORT" default:"8080"`
	ProdMode   bool   `envconfig:"PROD_MODE" default:"false"`
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("failed to load config values: %v", err)
	}

	logger := observability.StructuredLogger(cfg.ProdMode)

	logger.Info("Starting Mosaic backend server...")

	gwClient := handler.NewClient("https://api.verturus.com", handler.DefaultRetryConfig)

	atClient := at.NewAudioTranscriptionServiceClient(gwClient)
	fdClient := fd.NewFaceDetectionServiceClient(gwClient)
	cbClient := cb.NewConversationBriefingServiceClient(gwClient)

	go websocketServer(cfg, logger, atClient, fdClient, cbClient)

	// go channel for listening to sigint/sigterm signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	// trigger sigChan channel if app recieves either SIGTERM or SIGINT indicating it should shutdown
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Block until shutdown signal recieved
	<-sigChan
	logger.Info("Shutting down gracefully...")

	err = gwClient.Close()
	if err != nil {
		logger.Warn("grpc-web connection not closed properly", "err", err)
	}
	logger.Debug("Closed gRPC connection")
}

// this starts the websocket server used to communicate between frontend and this client
func websocketServer(
	cfg *Config,
	logger *slog.Logger,
	audio_client at.AudioTranscriptionServiceClient,
	face_client fd.FaceDetectionServiceClient,
	briefing_client cb.ConversationBriefingServiceClient,
) {
	router := http.NewServeMux()

	wsHandler := &handler.WebSocketHandler{
		Logger:         logger,
		AudioClient:    audio_client,
		FaceClient:     face_client,
		BriefingClient: briefing_client,
	}

	router.HandleFunc("/api/v1/ws", wsHandler.HandleWebSocket)

	server := http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.ServerPort),
		Handler: observability.HTTPLogger(router),
	}

	logger.Debug("[client] Server running on", "port", cfg.ServerPort)
	err := server.ListenAndServe()
	if err != nil {
		logger.Error("HTTP Server failed to start", "err", err)
		os.Exit(1)
	}
}

// method to load config values
func loadConfig() (*Config, error) {
	err := godotenv.Load(".env")
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
