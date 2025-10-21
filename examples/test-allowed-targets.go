package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/christianhuening/linkerd-mcp/internal/config"
	"github.com/christianhuening/linkerd-mcp/internal/policy"
	"github.com/mark3labs/mcp-go/mcp"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run test-allowed-targets.go <source-service> [namespace]")
		fmt.Println("\nExample:")
		fmt.Println("  go run examples/test-allowed-targets.go api-gateway")
		fmt.Println("  go run examples/test-allowed-targets.go checkout demo-app")
		os.Exit(1)
	}

	sourceService := os.Args[1]
	namespace := "demo-app"
	if len(os.Args) > 2 {
		namespace = os.Args[2]
	}

	fmt.Printf("=== Finding Allowed Targets for: %s ===\n\n", sourceService)

	// Initialize Kubernetes clients
	fmt.Println("Initializing Kubernetes clients...")
	clients, err := config.NewKubernetesClients()
	if err != nil {
		log.Fatalf("Failed to initialize Kubernetes clients: %v", err)
	}
	fmt.Println("✓ Kubernetes clients initialized\n")

	// Create policy analyzer
	analyzer := policy.NewAnalyzer(clients.Clientset, clients.DynamicClient)

	// Get allowed targets
	fmt.Printf("Finding services that %s can access in namespace %s...\n", sourceService, namespace)
	result, err := analyzer.GetAllowedTargets(context.Background(), namespace, sourceService)
	if err != nil {
		log.Fatalf("Failed to get allowed targets: %v", err)
	}

	// Parse the result
	if len(result.Content) == 0 {
		log.Fatal("No content in result")
	}

	// Get the text content using MCP helper
	textContent, ok := mcp.AsTextContent(result.Content[0])
	if !ok || textContent == nil {
		log.Fatal("Failed to parse content as text")
	}

	text := textContent.Text

	// Parse the JSON result
	var targetsResult map[string]interface{}
	if err := json.Unmarshal([]byte(text), &targetsResult); err != nil {
		log.Fatalf("Failed to parse targets result: %v", err)
	}

	// Pretty print the result
	prettyJSON, err := json.MarshalIndent(targetsResult, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}

	fmt.Println("\n=== Allowed Targets Result ===")
	fmt.Println(string(prettyJSON))

	// Print summary
	fmt.Println("\n=== Summary ===")
	source := targetsResult["source"].(string)
	fmt.Printf("Source Service: %s\n", source)

	if targets, ok := targetsResult["allowedTargets"].([]interface{}); ok {
		fmt.Printf("Can access %d target(s):\n\n", len(targets))

		if len(targets) == 0 {
			fmt.Println("  ❌ No authorized targets found")
			fmt.Println("  This service cannot access any other services in the mesh")
		} else {
			for i, t := range targets {
				target := t.(map[string]interface{})
				targetService := target["targetService"].(string)
				targetNamespace := target["targetNamespace"].(string)

				fmt.Printf("  %d. ✅ %s (namespace: %s)\n", i+1, targetService, targetNamespace)

				if server, ok := target["server"].(string); ok {
					fmt.Printf("     Server: %s\n", server)
				}
				if policies, ok := target["authorizationPolicies"].([]interface{}); ok && len(policies) > 0 {
					fmt.Printf("     Policies: %v\n", policies)
				}
			}
		}
	}
}
