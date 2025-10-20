package health

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/christianhuening/linkerd-mcp/internal/testutil"
	"github.com/mark3labs/mcp-go/mcp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCheckMeshHealth_HealthyControlPlane(t *testing.T) {
	// Create fake clientset with healthy control plane pods
	clientset := fake.NewSimpleClientset(
		testutil.CreateLinkerdControlPlanePod("destination-1", "linkerd", "destination", corev1.PodRunning, true),
		testutil.CreateLinkerdControlPlanePod("identity-1", "linkerd", "identity", corev1.PodRunning, true),
		testutil.CreateLinkerdControlPlanePod("proxy-injector-1", "linkerd", "proxy-injector", corev1.PodRunning, true),
	)

	checker := NewChecker(clientset)
	result, err := checker.CheckMeshHealth(context.Background(), "linkerd")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Parse the result
	var healthStatus map[string]interface{}
	if err := testutil.ParseJSONResult(result, &healthStatus); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	// Verify results
	if healthStatus["namespace"] != "linkerd" {
		t.Errorf("Expected namespace 'linkerd', got: %v", healthStatus["namespace"])
	}

	totalPods := int(healthStatus["totalPods"].(float64))
	if totalPods != 3 {
		t.Errorf("Expected 3 total pods, got: %d", totalPods)
	}

	healthyPods := int(healthStatus["healthyPods"].(float64))
	if healthyPods != 3 {
		t.Errorf("Expected 3 healthy pods, got: %d", healthyPods)
	}

	unhealthyPods := int(healthStatus["unhealthyPods"].(float64))
	if unhealthyPods != 0 {
		t.Errorf("Expected 0 unhealthy pods, got: %d", unhealthyPods)
	}
}

func TestCheckMeshHealth_UnhealthyControlPlane(t *testing.T) {
	// Create fake clientset with mix of healthy and unhealthy pods
	clientset := fake.NewSimpleClientset(
		testutil.CreateLinkerdControlPlanePod("destination-1", "linkerd", "destination", corev1.PodRunning, true),
		testutil.CreateLinkerdControlPlanePod("identity-1", "linkerd", "identity", corev1.PodFailed, false),
		testutil.CreateLinkerdControlPlanePod("proxy-injector-1", "linkerd", "proxy-injector", corev1.PodPending, false),
	)

	checker := NewChecker(clientset)
	result, err := checker.CheckMeshHealth(context.Background(), "linkerd")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	var healthStatus map[string]interface{}
	textContent, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatal("Expected text content")
	}
	if err := json.Unmarshal([]byte(textContent.Text), &healthStatus); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	healthyPods := int(healthStatus["healthyPods"].(float64))
	if healthyPods != 1 {
		t.Errorf("Expected 1 healthy pod, got: %d", healthyPods)
	}

	unhealthyPods := int(healthStatus["unhealthyPods"].(float64))
	if unhealthyPods != 2 {
		t.Errorf("Expected 2 unhealthy pods, got: %d", unhealthyPods)
	}

	// Verify component details
	components := healthStatus["components"].([]interface{})
	if len(components) != 3 {
		t.Errorf("Expected 3 components, got: %d", len(components))
	}

	// Check that failed pod is marked as unhealthy
	foundFailedPod := false
	for _, comp := range components {
		component := comp.(map[string]interface{})
		if component["name"] == "identity-1" {
			foundFailedPod = true
			if component["healthy"] != false {
				t.Errorf("Expected identity-1 to be unhealthy")
			}
			if component["status"] != "Failed" {
				t.Errorf("Expected status 'Failed', got: %v", component["status"])
			}
		}
	}
	if !foundFailedPod {
		t.Error("Failed pod not found in components")
	}
}

func TestCheckMeshHealth_EmptyNamespaceDefaultsToLinkerd(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		testutil.CreateLinkerdControlPlanePod("destination-1", "linkerd", "destination", corev1.PodRunning, true),
	)

	checker := NewChecker(clientset)
	result, err := checker.CheckMeshHealth(context.Background(), "")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	var healthStatus map[string]interface{}
	textContent, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatal("Expected text content")
	}
	if err := json.Unmarshal([]byte(textContent.Text), &healthStatus); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	if healthStatus["namespace"] != "linkerd" {
		t.Errorf("Expected default namespace 'linkerd', got: %v", healthStatus["namespace"])
	}
}

func TestCheckMeshHealth_NoControlPlanePods(t *testing.T) {
	// Create clientset with no control plane pods
	clientset := fake.NewSimpleClientset()

	checker := NewChecker(clientset)
	result, err := checker.CheckMeshHealth(context.Background(), "linkerd")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	var healthStatus map[string]interface{}
	textContent, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatal("Expected text content")
	}
	if err := json.Unmarshal([]byte(textContent.Text), &healthStatus); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	totalPods := int(healthStatus["totalPods"].(float64))
	if totalPods != 0 {
		t.Errorf("Expected 0 total pods, got: %d", totalPods)
	}

	components := healthStatus["components"].([]interface{})
	if len(components) != 0 {
		t.Errorf("Expected 0 components, got: %d", len(components))
	}
}

func TestCheckMeshHealth_CustomNamespace(t *testing.T) {
	// Create pods in custom namespace
	clientset := fake.NewSimpleClientset(
		testutil.CreateLinkerdControlPlanePod("destination-1", "custom-mesh", "destination", corev1.PodRunning, true),
	)

	checker := NewChecker(clientset)
	result, err := checker.CheckMeshHealth(context.Background(), "custom-mesh")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	var healthStatus map[string]interface{}
	textContent, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatal("Expected text content")
	}
	if err := json.Unmarshal([]byte(textContent.Text), &healthStatus); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	if healthStatus["namespace"] != "custom-mesh" {
		t.Errorf("Expected namespace 'custom-mesh', got: %v", healthStatus["namespace"])
	}

	totalPods := int(healthStatus["totalPods"].(float64))
	if totalPods != 1 {
		t.Errorf("Expected 1 total pod, got: %d", totalPods)
	}
}

func TestNewChecker(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	checker := NewChecker(clientset)

	if checker == nil {
		t.Fatal("Expected checker to be created")
	}

	if checker.clientset != clientset {
		t.Error("Expected clientset to be set correctly")
	}
}
