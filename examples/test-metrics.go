//go:build ignore

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/christianhuening/linkerd-mcp/internal/config"
	"github.com/christianhuening/linkerd-mcp/internal/metrics"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run test-metrics.go <namespace> <service> [time_range]")
		fmt.Println("\nExample:")
		fmt.Println("  go run examples/test-metrics.go default frontend")
		fmt.Println("  go run examples/test-metrics.go prod api-gateway 1h")
		fmt.Println("\nTime range examples: 5m, 10m, 1h, 24h")
		os.Exit(1)
	}

	namespace := os.Args[1]
	service := os.Args[2]
	timeRange := "5m"
	if len(os.Args) > 3 {
		timeRange = os.Args[3]
	}

	fmt.Println("=== Testing Linkerd MCP Metrics Endpoint ===\n")

	// Initialize Kubernetes clients
	fmt.Println("Initializing Kubernetes clients...")
	clients, err := config.NewKubernetesClients()
	if err != nil {
		log.Fatalf("Failed to create Kubernetes clients: %v", err)
	}
	fmt.Println("✓ Kubernetes clients initialized")

	// Create metrics collector
	fmt.Println("Connecting to Prometheus...")
	collector, err := metrics.NewMetricsCollector(clients.Config, clients.Clientset, "linkerd")
	if err != nil {
		log.Fatalf("Failed to create metrics collector: %v\n", err)
	}
	fmt.Println("✓ Prometheus connection established")

	ctx := context.Background()

	// Test 1: Get service metrics
	fmt.Printf("\n--- Test 1: Get Service Metrics ---\n")
	fmt.Printf("Service: %s/%s\n", namespace, service)
	fmt.Printf("Time Range: %s\n\n", timeRange)

	result, err := collector.GetServiceMetrics(ctx, namespace, service, timeRange)
	if err != nil {
		log.Fatalf("Failed to get service metrics: %v", err)
	}

	// Parse and display the result
	var serviceMetrics metrics.ServiceMetrics
	if len(result.Content) > 0 && result.Content[0].Text != nil {
		if err := json.Unmarshal([]byte(*result.Content[0].Text), &serviceMetrics); err != nil {
			log.Fatalf("Failed to parse service metrics: %v", err)
		}

		fmt.Println("Service Metrics:")
		fmt.Printf("  Service: %s/%s\n", serviceMetrics.Namespace, serviceMetrics.Service)
		fmt.Printf("  Request Rate: %.2f req/s\n", serviceMetrics.RequestRate)
		fmt.Printf("  Success Rate: %.2f%%\n", serviceMetrics.SuccessRate)
		fmt.Printf("  Error Rate: %.2f%%\n", serviceMetrics.ErrorRate)
		fmt.Printf("  Latency:\n")
		fmt.Printf("    P50: %.2fms\n", serviceMetrics.Latency.P50)
		fmt.Printf("    P95: %.2fms\n", serviceMetrics.Latency.P95)
		fmt.Printf("    P99: %.2fms\n", serviceMetrics.Latency.P99)
		fmt.Printf("    Mean: %.2fms\n", serviceMetrics.Latency.Mean)

		if len(serviceMetrics.ErrorsByStatus) > 0 {
			fmt.Printf("  Errors by Status:\n")
			for status, count := range serviceMetrics.ErrorsByStatus {
				fmt.Printf("    %s: %d\n", status, count)
			}
		}
	} else if result.IsError != nil && *result.IsError {
		fmt.Printf("Error: %s\n", *result.Content[0].Text)
	}

	// Test 2: Get service health summary
	fmt.Printf("\n--- Test 2: Get Service Health Summary ---\n")
	fmt.Printf("Namespace: %s\n\n", namespace)

	healthResult, err := collector.GetServiceHealthSummary(ctx, namespace, timeRange, metrics.DefaultHealthThresholds())
	if err != nil {
		log.Fatalf("Failed to get service health summary: %v", err)
	}

	if len(healthResult.Content) > 0 && healthResult.Content[0].Text != nil {
		var healthSummary map[string]interface{}
		if err := json.Unmarshal([]byte(*healthResult.Content[0].Text), &healthSummary); err != nil {
			log.Fatalf("Failed to parse health summary: %v", err)
		}

		fmt.Println("Health Summary:")
		prettyJSON, _ := json.MarshalIndent(healthSummary, "  ", "  ")
		fmt.Println(string(prettyJSON))
	}

	// Test 3: Get top services
	fmt.Printf("\n--- Test 3: Get Top Services ---\n")
	fmt.Printf("Namespace: %s\n", namespace)
	fmt.Printf("Sort By: request_rate\n")
	fmt.Printf("Limit: 5\n\n")

	topResult, err := collector.GetTopServices(ctx, namespace, "request_rate", timeRange, 5)
	if err != nil {
		log.Fatalf("Failed to get top services: %v", err)
	}

	if len(topResult.Content) > 0 && topResult.Content[0].Text != nil {
		var topServices metrics.ServiceRanking
		if err := json.Unmarshal([]byte(*topResult.Content[0].Text), &topServices); err != nil {
			log.Fatalf("Failed to parse top services: %v", err)
		}

		fmt.Println("Top Services:")
		for i, svc := range topServices.Services {
			fmt.Printf("%d. %s\n", i+1, svc.Service)
			fmt.Printf("   Request Rate: %.2f req/s\n", svc.RequestRate)
			fmt.Printf("   Success Rate: %.2f%%\n", svc.SuccessRate)
			fmt.Printf("   Error Rate: %.2f%%\n", svc.ErrorRate)
			fmt.Printf("   P95 Latency: %.2fms\n", svc.LatencyP95)
		}
	}

	fmt.Println("\n✓ All metrics tests completed successfully")
}
