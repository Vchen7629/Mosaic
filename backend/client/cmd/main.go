package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	at "mosaic-client.com/gen/audio_transcription"
	fd "mosaic-client.com/gen/face_detection"
	"mosaic-client.com/internal/handler"
	"mosaic-client.com/internal/middleware"
)

func WebsocketServer(
	audio_client at.AudioTranscriptionServiceClient,
	face_client fd.FaceDetectionServiceClient,
) {
	router := http.NewServeMux()

	wsHandler := &handler.WebSocketHandler{
		AudioClient: audio_client,
		FaceClient:  face_client,
	}

	router.HandleFunc("/api/v1/ws", wsHandler.HandleWebSocket)

	server := http.Server{
		Addr: ":8000",
		Handler: middleware.Logging(router),
	}

	fmt.Println("[Go Backend] Server running on http://localhost:8000")
	err := server.ListenAndServe()
	if err != nil {
		log.Fatalf("HTTP Server failed to start with error: %v", err)
	}
}

func main() {
	fmt.Println("[Go Backend] Starting Mosaic backend server...")

	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("Unable to start gRPC client")
	}

	atClient := at.NewAudioTranscriptionServiceClient(conn)
	fdClient := fd.NewFaceDetectionServiceClient(conn)

	go WebsocketServer(atClient, fdClient)

	// go channel for listening to sigint/sigterm signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	// trigger sigChan channel if app recieves either SIGTERM or SIGINT indicating it should shutdown
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Block until shutdown signal recieved
	<-sigChan
	log.Println("Shutting down gracefully...")

	conn.Close()
	log.Println("Closed gRPC connection")
}
