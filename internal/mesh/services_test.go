package mesh

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/christianhuening/linkerd-mcp/internal/testutil"
	"github.com/mark3labs/mcp-go/mcp"
	"k8s.io/client-go/kubernetes/fake"
)

func TestListMeshedServices_WithMeshedPods(t *testing.T) {
	// Create fake clientset with meshed pods
	clientset := fake.NewSimpleClientset(
		testutil.CreateMeshedPod("frontend-1", "prod", "frontend"),
		testutil.CreateMeshedPod("frontend-2", "prod", "frontend"),
		testutil.CreateMeshedPod("backend-1", "prod", "backend"),
		testutil.CreateMeshedPod("api-1", "staging", "api"),
	)

	lister := NewServiceLister(clientset)
	result, err := lister.ListMeshedServices(context.Background(), "")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	var response map[string]interface{}
	textContent, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatal("Expected text content")
	}
	if err := json.Unmarshal([]byte(textContent.Text), &response); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	totalServices := int(response["totalServices"].(float64))
	if totalServices != 3 {
		t.Errorf("Expected 3 services, got: %d", totalServices)
	}

	services := response["services"].(map[string]interface{})

	// Check frontend service
	frontendKey := "prod/frontend"
	if frontend, ok := services[frontendKey]; ok {
		frontendMap := frontend.(map[string]interface{})
		if frontendMap["namespace"] != "prod" {
			t.Errorf("Expected namespace 'prod', got: %v", frontendMap["namespace"])
		}
		if frontendMap["service"] != "frontend" {
			t.Errorf("Expected service 'frontend', got: %v", frontendMap["service"])
		}
		pods := frontendMap["pods"].([]interface{})
		if len(pods) != 2 {
			t.Errorf("Expected 2 pods for frontend, got: %d", len(pods))
		}
	} else {
		t.Errorf("Frontend service not found")
	}

	// Check backend service
	backendKey := "prod/backend"
	if backend, ok := services[backendKey]; ok {
		backendMap := backend.(map[string]interface{})
		pods := backendMap["pods"].([]interface{})
		if len(pods) != 1 {
			t.Errorf("Expected 1 pod for backend, got: %d", len(pods))
		}
	} else {
		t.Errorf("Backend service not found")
	}

	// Check api service in staging
	apiKey := "staging/api"
	if _, ok := services[apiKey]; !ok {
		t.Errorf("API service in staging not found")
	}
}

func TestListMeshedServices_FilterByNamespace(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		testutil.CreateMeshedPod("frontend-1", "prod", "frontend"),
		testutil.CreateMeshedPod("api-1", "staging", "api"),
	)

	lister := NewServiceLister(clientset)
	result, err := lister.ListMeshedServices(context.Background(), "prod")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	var response map[string]interface{}
	textContent, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatal("Expected text content")
	}
	if err := json.Unmarshal([]byte(textContent.Text), &response); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	totalServices := int(response["totalServices"].(float64))
	if totalServices != 1 {
		t.Errorf("Expected 1 service in prod, got: %d", totalServices)
	}

	services := response["services"].(map[string]interface{})
	if _, ok := services["prod/frontend"]; !ok {
		t.Error("Expected to find prod/frontend")
	}
	if _, ok := services["staging/api"]; ok {
		t.Error("Should not find staging/api when filtering by prod namespace")
	}
}

func TestListMeshedServices_NoMeshedPods(t *testing.T) {
	// Create pod without linkerd-proxy
	regularPod := testutil.CreatePod("app-1", "default", "default", map[string]string{"app": "myapp"}, "Running", true)

	clientset := fake.NewSimpleClientset(regularPod)

	lister := NewServiceLister(clientset)
	result, err := lister.ListMeshedServices(context.Background(), "")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	var response map[string]interface{}
	textContent, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatal("Expected text content")
	}
	if err := json.Unmarshal([]byte(textContent.Text), &response); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	totalServices := int(response["totalServices"].(float64))
	if totalServices != 0 {
		t.Errorf("Expected 0 services, got: %d", totalServices)
	}
}

func TestListMeshedServices_PodsWithoutAppLabel(t *testing.T) {
	// Create meshed pod but without app label
	podWithoutLabel := testutil.CreateMeshedPod("no-label-1", "default", "")
	podWithoutLabel.Labels = map[string]string{}

	clientset := fake.NewSimpleClientset(podWithoutLabel)

	lister := NewServiceLister(clientset)
	result, err := lister.ListMeshedServices(context.Background(), "")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	var response map[string]interface{}
	textContent, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatal("Expected text content")
	}
	if err := json.Unmarshal([]byte(textContent.Text), &response); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	totalServices := int(response["totalServices"].(float64))
	if totalServices != 0 {
		t.Errorf("Expected 0 services (no app label), got: %d", totalServices)
	}
}

func TestListMeshedServices_K8sAppLabel(t *testing.T) {
	// Create pod with k8s-app label instead of app
	pod := testutil.CreateMeshedPod("kube-pod-1", "kube-system", "")
	pod.Labels = map[string]string{"k8s-app": "kube-dns"}

	clientset := fake.NewSimpleClientset(pod)

	lister := NewServiceLister(clientset)
	result, err := lister.ListMeshedServices(context.Background(), "")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	var response map[string]interface{}
	textContent, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatal("Expected text content")
	}
	if err := json.Unmarshal([]byte(textContent.Text), &response); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	totalServices := int(response["totalServices"].(float64))
	if totalServices != 1 {
		t.Errorf("Expected 1 service with k8s-app label, got: %d", totalServices)
	}

	services := response["services"].(map[string]interface{})
	if service, ok := services["kube-system/kube-dns"]; ok {
		serviceMap := service.(map[string]interface{})
		if serviceMap["service"] != "kube-dns" {
			t.Errorf("Expected service name 'kube-dns', got: %v", serviceMap["service"])
		}
	} else {
		t.Error("Expected to find kube-system/kube-dns")
	}
}

func TestListMeshedServices_MultiplePodsPerService(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		testutil.CreateMeshedPod("web-1", "prod", "web"),
		testutil.CreateMeshedPod("web-2", "prod", "web"),
		testutil.CreateMeshedPod("web-3", "prod", "web"),
	)

	lister := NewServiceLister(clientset)
	result, err := lister.ListMeshedServices(context.Background(), "prod")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	var response map[string]interface{}
	textContent, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatal("Expected text content")
	}
	if err := json.Unmarshal([]byte(textContent.Text), &response); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	services := response["services"].(map[string]interface{})
	if service, ok := services["prod/web"]; ok {
		serviceMap := service.(map[string]interface{})
		pods := serviceMap["pods"].([]interface{})
		if len(pods) != 3 {
			t.Errorf("Expected 3 pods for web service, got: %d", len(pods))
		}

		// Verify all pod names are present
		podNames := make(map[string]bool)
		for _, pod := range pods {
			podNames[pod.(string)] = true
		}
		if !podNames["web-1"] || !podNames["web-2"] || !podNames["web-3"] {
			t.Error("Not all pod names found in the list")
		}
	} else {
		t.Error("Expected to find prod/web service")
	}
}

func TestNewServiceLister(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	lister := NewServiceLister(clientset)

	if lister == nil {
		t.Fatal("Expected lister to be created")
	}

	if lister.clientset != clientset {
		t.Error("Expected clientset to be set correctly")
	}
}
