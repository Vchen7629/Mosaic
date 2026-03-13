package main

import (
	"fmt"
	"log"
	"net/http"

	"mosaic-client.com/internal/handler"
	"mosaic-client.com/internal/middleware"
)

func main() {
	fmt.Println("[Go Backend] Starting Mosaic backend server...")

	router := http.NewServeMux()

	router.HandleFunc("/api/v1/ws", handler.HandleWebSocket)

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
