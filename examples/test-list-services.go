//go:build ignore

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/christianhuening/linkerd-mcp/internal/config"
	"github.com/christianhuening/linkerd-mcp/internal/mesh"
	"github.com/mark3labs/mcp-go/mcp"
)

func main() {
	namespace := ""
	if len(os.Args) > 1 {
		namespace = os.Args[1]
	}

	fmt.Println("=== Listing Meshed Services ===\n")

	// Initialize Kubernetes clients
	fmt.Println("Initializing Kubernetes clients...")
	clients, err := config.NewKubernetesClients()
	if err != nil {
		log.Fatalf("Failed to initialize Kubernetes clients: %v", err)
	}
	fmt.Println("✓ Kubernetes clients initialized\n")

	// Create service lister
	lister := mesh.NewServiceLister(clients.Clientset)

	// List meshed services
	if namespace == "" {
		fmt.Println("Listing all meshed services across all namespaces...")
	} else {
		fmt.Printf("Listing meshed services in namespace: %s...\n", namespace)
	}

	result, err := lister.ListMeshedServices(context.Background(), namespace)
	if err != nil {
		log.Fatalf("Failed to list meshed services: %v", err)
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
	var servicesResult map[string]interface{}
	if err := json.Unmarshal([]byte(text), &servicesResult); err != nil {
		log.Fatalf("Failed to parse services result: %v", err)
	}

	// Pretty print the result
	prettyJSON, err := json.MarshalIndent(servicesResult, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}

	fmt.Println("\n=== Meshed Services Result ===")
	fmt.Println(string(prettyJSON))

	// Print summary
	fmt.Println("\n=== Summary ===")
	totalServices := int(servicesResult["totalServices"].(float64))
	fmt.Printf("Total meshed services: %d\n", totalServices)

	if servicesMap, ok := servicesResult["services"].(map[string]interface{}); ok && len(servicesMap) > 0 {
		fmt.Println("\n=== Service Details ===")

		// Group by namespace
		namespaceMap := make(map[string][]map[string]interface{})
		for _, svcData := range servicesMap {
			service := svcData.(map[string]interface{})
			ns := service["namespace"].(string)
			namespaceMap[ns] = append(namespaceMap[ns], service)
		}

		for ns, nsSvcs := range namespaceMap {
			fmt.Printf("\n📦 Namespace: %s (%d services)\n", ns, len(nsSvcs))
			for i, service := range nsSvcs {
				serviceName := service["service"].(string)
				pods := service["pods"].([]interface{})

				fmt.Printf("  %d. ✅ %s\n", i+1, serviceName)
				fmt.Printf("     Pods: %d\n", len(pods))
				for j, podName := range pods {
					fmt.Printf("       - %s\n", podName.(string))
					if j >= 2 {
						fmt.Printf("       ... and %d more\n", len(pods)-3)
						break
					}
				}
			}
		}
	} else {
		fmt.Println("\n❌ No meshed services found")
		if namespace == "" {
			fmt.Println("   No services with Linkerd proxy injection found in any namespace")
		} else {
			fmt.Printf("   No services with Linkerd proxy injection found in namespace: %s\n", namespace)
		}
	}

	fmt.Println("\n💡 Tip: Services must have the linkerd-proxy sidecar to appear in this list")
}
