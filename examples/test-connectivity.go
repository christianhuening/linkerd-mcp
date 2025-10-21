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
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run test-connectivity.go <source-service> <target-service> [target-namespace]")
		fmt.Println("\nExample:")
		fmt.Println("  go run examples/test-connectivity.go frontend api-gateway")
		fmt.Println("  go run examples/test-connectivity.go api-gateway catalog demo-app")
		os.Exit(1)
	}

	sourceService := os.Args[1]
	targetService := os.Args[2]
	targetNamespace := "demo-app"
	if len(os.Args) > 3 {
		targetNamespace = os.Args[3]
	}

	fmt.Printf("=== Testing Connectivity: %s → %s ===[0m\n\n", sourceService, targetService)

	// Initialize Kubernetes clients
	fmt.Println("Initializing Kubernetes clients...")
	clients, err := config.NewKubernetesClients()
	if err != nil {
		log.Fatalf("Failed to initialize Kubernetes clients: %v", err)
	}
	fmt.Println("✓ Kubernetes clients initialized\n")

	// Create policy analyzer
	analyzer := policy.NewAnalyzer(clients.Clientset, clients.DynamicClient)

	// Analyze connectivity
	fmt.Printf("Analyzing connectivity from %s to %s in namespace %s...\n", sourceService, targetService, targetNamespace)
	result, err := analyzer.AnalyzeConnectivity(context.Background(), "demo-app", sourceService, targetService, targetNamespace)
	if err != nil {
		log.Fatalf("Failed to analyze connectivity: %v", err)
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

	// Parse the JSON connectivity result
	var connectivityResult map[string]interface{}
	if err := json.Unmarshal([]byte(text), &connectivityResult); err != nil {
		log.Fatalf("Failed to parse connectivity result: %v", err)
	}

	// Pretty print the result
	prettyJSON, err := json.MarshalIndent(connectivityResult, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}

	fmt.Println("\n=== Connectivity Analysis Result ===")
	fmt.Println(string(prettyJSON))

	// Print summary
	fmt.Println("\n=== Summary ===")
	allowed := connectivityResult["allowed"].(bool)
	sourceObj := connectivityResult["source"].(map[string]interface{})
	targetObj := connectivityResult["target"].(map[string]interface{})

	var resultSourceService, resultSourceNs, resultTargetService, resultTargetNs string
	if val, ok := sourceObj["service"].(string); ok {
		resultSourceService = val
	}
	if val, ok := sourceObj["namespace"].(string); ok {
		resultSourceNs = val
	}
	if val, ok := targetObj["service"].(string); ok {
		resultTargetService = val
	}
	if val, ok := targetObj["namespace"].(string); ok {
		resultTargetNs = val
	}

	fmt.Printf("Source:  %s/%s\n", resultSourceNs, resultSourceService)
	fmt.Printf("Target:  %s/%s\n", resultTargetNs, resultTargetService)

	if allowed {
		fmt.Println("\n✅ ACCESS ALLOWED")
		if details, ok := connectivityResult["details"].(string); ok && details != "" {
			fmt.Printf("Reason:  %s\n", details)
		}
		if policies, ok := connectivityResult["authorizationPolicies"].([]interface{}); ok && len(policies) > 0 {
			fmt.Printf("\nAuthorizing Policies: %d\n", len(policies))
			for i, p := range policies {
				fmt.Printf("  %d. %s\n", i+1, p.(string))
			}
		}
	} else {
		fmt.Println("\n❌ ACCESS DENIED")
		if reason, ok := connectivityResult["reason"].(string); ok {
			fmt.Printf("Reason:  %s\n", reason)
		}
	}
}
