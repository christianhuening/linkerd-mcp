package policy

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/christianhuening/linkerd-mcp/internal/testutil"
	"github.com/mark3labs/mcp-go/mcp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

func setupPolicyTest() (*Analyzer, *kubefake.Clientset, *fake.FakeDynamicClient) {
	scheme := runtime.NewScheme()

	// Register custom list kinds for Linkerd CRDs
	gvrToListKind := map[schema.GroupVersionResource]string{
		{Group: "policy.linkerd.io", Version: "v1beta3", Resource: "servers"}:                  "ServerList",
		{Group: "policy.linkerd.io", Version: "v1alpha1", Resource: "authorizationpolicies"}:   "AuthorizationPolicyList",
		{Group: "policy.linkerd.io", Version: "v1alpha1", Resource: "meshtlsauthentications"}:  "MeshTLSAuthenticationList",
		{Group: "policy.linkerd.io", Version: "v1alpha1", Resource: "networkauthentications"}:  "NetworkAuthenticationList",
		{Group: "policy.linkerd.io", Version: "v1alpha1", Resource: "httproutes"}:              "HTTPRouteList",
	}

	kubeClient := kubefake.NewSimpleClientset()
	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToListKind)

	analyzer := NewAnalyzer(kubeClient, dynamicClient)
	return analyzer, kubeClient, dynamicClient
}

func TestAnalyzeConnectivity(t *testing.T) {
	analyzer, _, _ := setupPolicyTest()

	result, err := analyzer.AnalyzeConnectivity(
		context.Background(),
		"prod",
		"frontend",
		"prod",
		"backend",
	)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	var analysis map[string]interface{}
	textContent, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatal("Expected text content")
	}
	if err := json.Unmarshal([]byte(textContent.Text), &analysis); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	// Verify structure
	source := analysis["source"].(map[string]interface{})
	if source["namespace"] != "prod" {
		t.Errorf("Expected source namespace 'prod', got: %v", source["namespace"])
	}
	if source["service"] != "frontend" {
		t.Errorf("Expected source service 'frontend', got: %v", source["service"])
	}

	target := analysis["target"].(map[string]interface{})
	if target["namespace"] != "prod" {
		t.Errorf("Expected target namespace 'prod', got: %v", target["namespace"])
	}
	if target["service"] != "backend" {
		t.Errorf("Expected target service 'backend', got: %v", target["service"])
	}
}

func TestAnalyzeConnectivity_DefaultTargetNamespace(t *testing.T) {
	analyzer, _, _ := setupPolicyTest()

	result, err := analyzer.AnalyzeConnectivity(
		context.Background(),
		"prod",
		"frontend",
		"", // Empty target namespace
		"backend",
	)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	var analysis map[string]interface{}
	textContent, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatal("Expected text content")
	}
	if err := json.Unmarshal([]byte(textContent.Text), &analysis); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	target := analysis["target"].(map[string]interface{})
	if target["namespace"] != "prod" {
		t.Errorf("Expected target namespace to default to 'prod', got: %v", target["namespace"])
	}
}

func TestGetAllowedTargets_NoPodsFound(t *testing.T) {
	analyzer, _, _ := setupPolicyTest()

	result, err := analyzer.GetAllowedTargets(
		context.Background(),
		"prod",
		"nonexistent",
	)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should return error result
	if !result.IsError {
		t.Error("Expected error result when no pods found")
	}

	expectedError := "no pods found for service nonexistent in namespace prod"
	textContent, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatal("Expected text content")
	}
	if textContent.Text != expectedError {
		t.Errorf("Expected error message '%s', got: %s", expectedError, textContent.Text)
	}
}

func TestGetAllowedTargets_WithPods(t *testing.T) {
	analyzer, kubeClient, dynamicClient := setupPolicyTest()

	// Add source pod
	pod := testutil.CreatePod("frontend-1", "prod", "frontend-sa", map[string]string{"app": "frontend"}, "Running", true)
	kubeClient.CoreV1().Pods("prod").Create(context.Background(), pod, metav1.CreateOptions{})

	// Add Server CRD
	server := testutil.CreateServer("backend-server", "prod", map[string]string{"app": "backend"}, 8080)
	dynamicClient.Resource(serverGVR).Namespace("prod").Create(context.Background(), server, metav1.CreateOptions{})

	// Add AuthorizationPolicy
	authPolicy := testutil.CreateAuthorizationPolicy(
		"allow-frontend",
		"prod",
		"backend-server",
		[]map[string]string{{"name": "frontend-auth", "kind": "MeshTLSAuthentication"}},
	)
	dynamicClient.Resource(authPolicyGVR).Namespace("prod").Create(context.Background(), authPolicy, metav1.CreateOptions{})

	// Add MeshTLSAuthentication allowing frontend
	meshAuth := testutil.CreateMeshTLSAuthentication(
		"frontend-auth",
		"prod",
		[]string{"frontend-sa.prod.serviceaccount.identity.linkerd.cluster.local"},
		nil,
	)
	dynamicClient.Resource(meshTLSAuthGVR).Namespace("prod").Create(context.Background(), meshAuth, metav1.CreateOptions{})

	result, err := analyzer.GetAllowedTargets(context.Background(), "prod", "frontend")

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

	source := response["source"].(map[string]interface{})
	if source["serviceAccount"] != "frontend-sa" {
		t.Errorf("Expected serviceAccount 'frontend-sa', got: %v", source["serviceAccount"])
	}

	allowedTargets := response["allowedTargets"].([]interface{})
	totalTargets := int(response["totalTargets"].(float64))

	if totalTargets != len(allowedTargets) {
		t.Errorf("totalTargets (%d) doesn't match allowedTargets length (%d)", totalTargets, len(allowedTargets))
	}
}

func TestGetAllowedSources_NoServersFound(t *testing.T) {
	analyzer, _, _ := setupPolicyTest()

	result, err := analyzer.GetAllowedSources(context.Background(), "prod", "backend")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expectedMessage := "No Linkerd Servers found for service backend in namespace prod"
	textContent, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatal("Expected text content")
	}
	if textContent.Text != expectedMessage {
		t.Errorf("Expected message '%s', got: %s", expectedMessage, textContent.Text)
	}
}

func TestGetAllowedSources_WithServersAndPolicies(t *testing.T) {
	analyzer, _, dynamicClient := setupPolicyTest()

	// Add Server for backend
	server := testutil.CreateServer("backend-server", "prod", map[string]string{"app": "backend"}, 8080)
	dynamicClient.Resource(serverGVR).Namespace("prod").Create(context.Background(), server, metav1.CreateOptions{})

	// Add AuthorizationPolicy
	authPolicy := testutil.CreateAuthorizationPolicy(
		"allow-all-auth",
		"prod",
		"backend-server",
		[]map[string]string{{"name": "all-auth", "kind": "MeshTLSAuthentication"}},
	)
	dynamicClient.Resource(authPolicyGVR).Namespace("prod").Create(context.Background(), authPolicy, metav1.CreateOptions{})

	// Add MeshTLSAuthentication with wildcard
	meshAuth := testutil.CreateMeshTLSAuthentication(
		"all-auth",
		"prod",
		[]string{"*"},
		nil,
	)
	dynamicClient.Resource(meshTLSAuthGVR).Namespace("prod").Create(context.Background(), meshAuth, metav1.CreateOptions{})

	result, err := analyzer.GetAllowedSources(context.Background(), "prod", "backend")

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

	matchingServers := response["matchingServers"].([]interface{})
	if len(matchingServers) != 1 {
		t.Errorf("Expected 1 matching server, got: %d", len(matchingServers))
	}

	allowedSources := response["allowedSources"].([]interface{})
	if len(allowedSources) < 1 {
		t.Error("Expected at least 1 allowed source")
	}

	// Check for wildcard source
	foundWildcard := false
	for _, src := range allowedSources {
		source := src.(map[string]interface{})
		if source["type"] == "wildcard" {
			foundWildcard = true
			if source["description"] != "All authenticated services" {
				t.Errorf("Expected description 'All authenticated services', got: %v", source["description"])
			}
		}
	}
	if !foundWildcard {
		t.Error("Expected to find wildcard source")
	}
}

func TestGetAllowedSources_WithServiceAccounts(t *testing.T) {
	analyzer, _, dynamicClient := setupPolicyTest()

	// Add Server
	server := testutil.CreateServer("api-server", "prod", map[string]string{"app": "api"}, 8080)
	dynamicClient.Resource(serverGVR).Namespace("prod").Create(context.Background(), server, metav1.CreateOptions{})

	// Add AuthorizationPolicy
	authPolicy := testutil.CreateAuthorizationPolicy(
		"allow-frontend",
		"prod",
		"api-server",
		[]map[string]string{{"name": "frontend-auth", "kind": "MeshTLSAuthentication"}},
	)
	dynamicClient.Resource(authPolicyGVR).Namespace("prod").Create(context.Background(), authPolicy, metav1.CreateOptions{})

	// Add MeshTLSAuthentication with service accounts
	meshAuth := testutil.CreateMeshTLSAuthentication(
		"frontend-auth",
		"prod",
		nil,
		[]map[string]string{
			{"name": "frontend-sa", "namespace": "prod"},
			{"name": "admin-sa", "namespace": "admin"},
		},
	)
	dynamicClient.Resource(meshTLSAuthGVR).Namespace("prod").Create(context.Background(), meshAuth, metav1.CreateOptions{})

	result, err := analyzer.GetAllowedSources(context.Background(), "prod", "api")

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

	allowedSources := response["allowedSources"].([]interface{})
	if len(allowedSources) != 2 {
		t.Errorf("Expected 2 allowed sources, got: %d", len(allowedSources))
	}

	// Verify service accounts are present
	foundFrontend := false
	foundAdmin := false
	for _, src := range allowedSources {
		source := src.(map[string]interface{})
		if sa, ok := source["serviceAccount"]; ok {
			if sa == "frontend-sa" {
				foundFrontend = true
				if source["namespace"] != "prod" {
					t.Errorf("Expected namespace 'prod' for frontend-sa, got: %v", source["namespace"])
				}
			}
			if sa == "admin-sa" {
				foundAdmin = true
				if source["namespace"] != "admin" {
					t.Errorf("Expected namespace 'admin' for admin-sa, got: %v", source["namespace"])
				}
			}
		}
	}

	if !foundFrontend {
		t.Error("Expected to find frontend-sa in allowed sources")
	}
	if !foundAdmin {
		t.Error("Expected to find admin-sa in allowed sources")
	}
}

func TestNewAnalyzer(t *testing.T) {
	kubeClient := kubefake.NewSimpleClientset()
	dynamicClient := fake.NewSimpleDynamicClient(runtime.NewScheme())

	analyzer := NewAnalyzer(kubeClient, dynamicClient)

	if analyzer == nil {
		t.Fatal("Expected analyzer to be created")
	}

	if analyzer.clientset != kubeClient {
		t.Error("Expected clientset to be set correctly")
	}

	if analyzer.dynamicClient != dynamicClient {
		t.Error("Expected dynamicClient to be set correctly")
	}
}

// Define GVRs used in tests (same as in the actual code)
var (
	serverGVR = schema.GroupVersionResource{
		Group:    "policy.linkerd.io",
		Version:  "v1beta3",
		Resource: "servers",
	}

	authPolicyGVR = schema.GroupVersionResource{
		Group:    "policy.linkerd.io",
		Version:  "v1alpha1",
		Resource: "authorizationpolicies",
	}

	meshTLSAuthGVR = schema.GroupVersionResource{
		Group:    "policy.linkerd.io",
		Version:  "v1alpha1",
		Resource: "meshtlsauthentications",
	}
)
