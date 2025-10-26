//go:build ignore

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/christianhuening/linkerd-mcp/internal/config"
	"github.com/christianhuening/linkerd-mcp/internal/health"
	"github.com/mark3labs/mcp-go/mcp"
)

func main() {
	fmt.Println("=== Testing Linkerd MCP Mesh Health Endpoint ===\n")

	// Initialize Kubernetes clients
	fmt.Println("Initializing Kubernetes clients...")
	clients, err := config.NewKubernetesClients()
	if err != nil {
		log.Fatalf("Failed to initialize Kubernetes clients: %v", err)
	}
	fmt.Println("✓ Kubernetes clients initialized\n")

	// Create health checker
	checker := health.NewChecker(clients.Clientset)

	// Check mesh health in linkerd namespace
	fmt.Println("Checking Linkerd mesh health in 'linkerd' namespace...")
	result, err := checker.CheckMeshHealth(context.Background(), "linkerd")
	if err != nil {
		log.Fatalf("Failed to check mesh health: %v", err)
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

	// Parse the JSON health status
	var healthStatus map[string]interface{}
	if err := json.Unmarshal([]byte(text), &healthStatus); err != nil {
		log.Fatalf("Failed to parse health status: %v", err)
	}

	// Pretty print the result
	prettyJSON, err := json.MarshalIndent(healthStatus, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}

	fmt.Println("✓ Mesh health check complete\n")
	fmt.Println("=== Linkerd Mesh Health Status ===")
	fmt.Println(string(prettyJSON))

	// Print summary
	fmt.Println("\n=== Summary ===")
	totalPods := int(healthStatus["totalPods"].(float64))
	healthyPods := int(healthStatus["healthyPods"].(float64))
	unhealthyPods := int(healthStatus["unhealthyPods"].(float64))
	namespace := healthStatus["namespace"].(string)

	fmt.Printf("Namespace:      %s\n", namespace)
	fmt.Printf("Total Pods:     %d\n", totalPods)
	fmt.Printf("Healthy Pods:   %d\n", healthyPods)
	fmt.Printf("Unhealthy Pods: %d\n", unhealthyPods)

	if unhealthyPods == 0 {
		fmt.Println("\n✓ All control plane components are healthy!")
	} else {
		fmt.Printf("\n⚠ Warning: %d unhealthy pods detected\n", unhealthyPods)
	}

	// Print individual component status
	if components, ok := healthStatus["components"].([]interface{}); ok {
		fmt.Println("\n=== Component Details ===")
		for _, comp := range components {
			c := comp.(map[string]interface{})
			healthy := "✓"
			if !c["healthy"].(bool) {
				healthy = "✗"
			}
			fmt.Printf("%s %-20s %-40s %s\n",
				healthy,
				c["component"].(string),
				c["name"].(string),
				c["status"].(string))
		}
	}
}
