package health

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Checker provides health checking functionality for Linkerd mesh
type Checker struct {
	clientset kubernetes.Interface
}

// NewChecker creates a new health checker
func NewChecker(clientset kubernetes.Interface) *Checker {
	return &Checker{
		clientset: clientset,
	}
}

// CheckMeshHealth checks the health status of the Linkerd service mesh
func (c *Checker) CheckMeshHealth(ctx context.Context, namespace string) (*mcp.CallToolResult, error) {
	if namespace == "" {
		namespace = "linkerd"
	}

	// Get Linkerd control plane pods
	pods, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "linkerd.io/control-plane-component",
	})
	if err != nil {
		return mcp.NewToolResultError("Failed to list control plane pods: " + err.Error()), nil
	}

	healthStatus := map[string]interface{}{
		"namespace":     namespace,
		"totalPods":     len(pods.Items),
		"healthyPods":   0,
		"unhealthyPods": 0,
		"components":    []map[string]interface{}{},
	}

	for _, pod := range pods.Items {
		component := pod.Labels["linkerd.io/control-plane-component"]
		healthy := true
		status := "Running"

		if pod.Status.Phase != corev1.PodPhase("Running") {
			healthy = false
			status = string(pod.Status.Phase)
		}

		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady && condition.Status != corev1.ConditionTrue {
				healthy = false
			}
		}

		if healthy {
			healthStatus["healthyPods"] = healthStatus["healthyPods"].(int) + 1
		} else {
			healthStatus["unhealthyPods"] = healthStatus["unhealthyPods"].(int) + 1
		}

		componentInfo := map[string]interface{}{
			"name":      pod.Name,
			"component": component,
			"healthy":   healthy,
			"status":    status,
		}

		healthStatus["components"] = append(healthStatus["components"].([]map[string]interface{}), componentInfo)
	}

	result, _ := json.MarshalIndent(healthStatus, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}
