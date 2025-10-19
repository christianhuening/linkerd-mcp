package mesh

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ServiceLister provides functionality for listing meshed services
type ServiceLister struct {
	clientset *kubernetes.Clientset
}

// NewServiceLister creates a new service lister
func NewServiceLister(clientset *kubernetes.Clientset) *ServiceLister {
	return &ServiceLister{
		clientset: clientset,
	}
}

// ListMeshedServices lists all services that are part of the Linkerd mesh
func (s *ServiceLister) ListMeshedServices(ctx context.Context, namespace string) (*mcp.CallToolResult, error) {
	listOptions := metav1.ListOptions{}
	pods, err := s.clientset.CoreV1().Pods(namespace).List(ctx, listOptions)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list pods: %v", err)), nil
	}

	meshedServices := make(map[string]map[string]interface{})

	for _, pod := range pods.Items {
		// Check if pod has Linkerd proxy injected
		hasProxy := false
		for _, container := range pod.Spec.Containers {
			if container.Name == "linkerd-proxy" {
				hasProxy = true
				break
			}
		}

		if !hasProxy {
			continue
		}

		serviceName := pod.Labels["app"]
		if serviceName == "" {
			serviceName = pod.Labels["k8s-app"]
		}
		if serviceName == "" {
			continue
		}

		key := fmt.Sprintf("%s/%s", pod.Namespace, serviceName)
		if _, exists := meshedServices[key]; !exists {
			meshedServices[key] = map[string]interface{}{
				"namespace": pod.Namespace,
				"service":   serviceName,
				"pods":      []string{},
			}
		}

		meshedServices[key]["pods"] = append(
			meshedServices[key]["pods"].([]string),
			pod.Name,
		)
	}

	result := map[string]interface{}{
		"totalServices": len(meshedServices),
		"services":      meshedServices,
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(resultJSON)), nil
}
