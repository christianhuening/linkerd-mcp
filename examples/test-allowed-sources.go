//go:build ignore

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
		fmt.Println("Usage: go run test-allowed-sources.go <target-service> [namespace]")
		fmt.Println("\nExample:")
		fmt.Println("  go run examples/test-allowed-sources.go payment")
		fmt.Println("  go run examples/test-allowed-sources.go database demo-app")
		os.Exit(1)
	}

	targetService := os.Args[1]
	namespace := "demo-app"
	if len(os.Args) > 2 {
		namespace = os.Args[2]
	}

	fmt.Printf("=== Finding Allowed Sources for: %s ===\n\n", targetService)

	// Initialize Kubernetes clients
	fmt.Println("Initializing Kubernetes clients...")
	clients, err := config.NewKubernetesClients()
	if err != nil {
		log.Fatalf("Failed to initialize Kubernetes clients: %v", err)
	}
	fmt.Println("✓ Kubernetes clients initialized\n")

	// Create policy analyzer
	analyzer := policy.NewAnalyzer(clients.Clientset, clients.DynamicClient)

	// Get allowed sources
	fmt.Printf("Finding services that can access %s in namespace %s...\n", targetService, namespace)
	result, err := analyzer.GetAllowedSources(context.Background(), namespace, targetService)
	if err != nil {
		log.Fatalf("Failed to get allowed sources: %v", err)
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
	var sourcesResult map[string]interface{}
	if err := json.Unmarshal([]byte(text), &sourcesResult); err != nil {
		log.Fatalf("Failed to parse sources result: %v", err)
	}

	// Pretty print the result
	prettyJSON, err := json.MarshalIndent(sourcesResult, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}

	fmt.Println("\n=== Allowed Sources Result ===")
	fmt.Println(string(prettyJSON))

	// Print summary
	fmt.Println("\n=== Summary ===")
	target := sourcesResult["target"].(string)
	fmt.Printf("Target Service: %s\n", target)

	if sources, ok := sourcesResult["allowedSources"].([]interface{}); ok {
		fmt.Printf("Accessible by %d source(s):\n\n", len(sources))

		if len(sources) == 0 {
			fmt.Println("  ❌ No authorized sources found")
			fmt.Println("  This service cannot be accessed by any other services")
			fmt.Println("  (It may be completely locked down or have no Server/AuthorizationPolicy)")
		} else {
			for i, s := range sources {
				source := s.(map[string]interface{})
				sourceService := source["sourceService"].(string)
				sourceNamespace := source["sourceNamespace"].(string)

				fmt.Printf("  %d. ✅ %s (namespace: %s)\n", i+1, sourceService, sourceNamespace)

				if identity, ok := source["identity"].(string); ok && identity != "" {
					fmt.Printf("     Identity: %s\n", identity)
				}
				if policies, ok := source["authorizationPolicies"].([]interface{}); ok && len(policies) > 0 {
					fmt.Printf("     Policies: %v\n", policies)
				}
			}
		}
	}

	// Additional security note for sensitive services
	if targetService == "payment" || targetService == "database" {
		fmt.Println("\n⚠️  Security Note:")
		fmt.Printf("   %s is a sensitive service and should have restricted access\n", targetService)
		if sources, ok := sourcesResult["allowedSources"].([]interface{}); ok && len(sources) > 0 {
			fmt.Printf("   Currently accessible by %d service(s) - verify this is expected\n", len(sources))
		}
	}
}
