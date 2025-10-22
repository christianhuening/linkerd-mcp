package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/christianhuening/linkerd-mcp/internal/server"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func main() {
	// Initialize the Linkerd MCP server
	linkerdServer, err := server.New()
	if err != nil {
		log.Fatalf("Failed to initialize Linkerd MCP server: %v", err)
	}

	// Create MCP server with tool capabilities
	s := mcpserver.NewMCPServer(
		"linkerd-mcp",
		"1.0.0",
		mcpserver.WithToolCapabilities(true),
	)

	// Register all tools
	linkerdServer.RegisterTools(s)

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Create HTTP mux
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := fmt.Fprintf(w, `{"status":"healthy","service":"linkerd-mcp","version":"1.0.0"}`); err != nil {
			log.Printf("Error writing health response: %v", err)
		}
	})

	// Readiness check endpoint
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := fmt.Fprintf(w, `{"status":"ready"}`); err != nil {
			log.Printf("Error writing ready response: %v", err)
		}
	})

	// Create StreamableHTTP server for MCP protocol (replaces deprecated SSE)
	// This mounts the MCP endpoints at /mcp/*
	streamableServer := mcpserver.NewStreamableHTTPServer(s)

	// Mount StreamableHTTP server at /mcp
	mux.Handle("/mcp/", http.StripPrefix("/mcp", streamableServer))

	// Create HTTP server with timeouts
	httpServer := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting MCP server on port %s", port)
		log.Printf("Health check: http://localhost:%s/health", port)
		log.Printf("Readiness check: http://localhost:%s/ready", port)
		log.Printf("MCP StreamableHTTP endpoint: http://localhost:%s/mcp", port)
		log.Printf("  - POST /mcp/initialize")
		log.Printf("  - POST /mcp/tools/list")
		log.Printf("  - POST /mcp/tools/call")
		log.Printf("  - GET /mcp/health")
		log.Printf("  - GET /mcp/capabilities")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
