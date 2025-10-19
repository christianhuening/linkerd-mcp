package main

import (
	"log"

	"github.com/christianhuening/linkerd-mcp/internal/server"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func main() {
	// Initialize the Linkerd MCP server
	linkerdServer, err := server.New()
	if err != nil {
		log.Fatalf("Failed to initialize Linkerd MCP server: %v", err)
	}

	// Create MCP server
	s := mcpserver.NewMCPServer(
		"linkerd-mcp",
		"1.0.0",
		mcpserver.WithToolCapabilities(true),
	)

	// Register all tools
	linkerdServer.RegisterTools(s)

	// Start serving
	if err := mcpserver.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
