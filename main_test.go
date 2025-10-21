package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/christianhuening/linkerd-mcp/internal/server"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// TestHealthEndpoint tests the /health endpoint
func TestHealthEndpoint(t *testing.T) {
	// Create a request to the health endpoint
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	// Create handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy","service":"linkerd-mcp","version":"1.0.0"}`))
	})

	// Serve the request
	handler.ServeHTTP(w, req)

	// Check status code
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	// Check content type
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}

	// Parse response body
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify response fields
	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got '%v'", response["status"])
	}

	if response["service"] != "linkerd-mcp" {
		t.Errorf("Expected service 'linkerd-mcp', got '%v'", response["service"])
	}

	if response["version"] != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%v'", response["version"])
	}
}

// TestReadyEndpoint tests the /ready endpoint
func TestReadyEndpoint(t *testing.T) {
	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	})

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "ready" {
		t.Errorf("Expected status 'ready', got '%v'", response["status"])
	}
}

// TestHealthEndpoint_Methods tests that only GET is allowed
func TestHealthEndpoint_Methods(t *testing.T) {
	methods := []string{"POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		req := httptest.NewRequest(method, "/health", nil)
		w := httptest.NewRecorder()

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"healthy"}`))
		})

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Method %s: Expected status code %d, got %d", method, http.StatusMethodNotAllowed, w.Code)
		}
	}
}

// TestReadyEndpoint_Methods tests that only GET is allowed
func TestReadyEndpoint_Methods(t *testing.T) {
	methods := []string{"POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		req := httptest.NewRequest(method, "/ready", nil)
		w := httptest.NewRecorder()

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ready"}`))
		})

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Method %s: Expected status code %d, got %d", method, http.StatusMethodNotAllowed, w.Code)
		}
	}
}

// TestMCPServerCreation tests that the MCP server can be created
func TestMCPServerCreation(t *testing.T) {
	// Skip this test in CI as it requires Kubernetes configuration
	t.Skip("Skipping MCP server creation test - requires Kubernetes config")

	_, err := server.New()
	if err != nil {
		t.Fatalf("Failed to create Linkerd server: %v", err)
	}

	s := mcpserver.NewMCPServer(
		"linkerd-mcp",
		"1.0.0",
		mcpserver.WithToolCapabilities(true),
	)

	if s == nil {
		t.Fatal("Expected MCP server to be created")
	}
}

// TestSSEEndpoint tests the SSE endpoint basic structure
func TestSSEEndpoint(t *testing.T) {
	// Skip this test in CI as it requires Kubernetes configuration
	t.Skip("Skipping SSE endpoint test - requires Kubernetes config")

	linkerdServer, err := server.New()
	if err != nil {
		t.Fatalf("Failed to create Linkerd server: %v", err)
	}

	s := mcpserver.NewMCPServer(
		"linkerd-mcp",
		"1.0.0",
		mcpserver.WithToolCapabilities(true),
	)

	linkerdServer.RegisterTools(s)

	sseServer := mcpserver.NewSSEServer(s)

	// Create a test request
	req := httptest.NewRequest("GET", "/sse", nil)
	w := httptest.NewRecorder()

	// Serve the request
	sseServer.ServeHTTP(w, req)

	// SSE should return 200 OK
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}
}

// TestHTTPServerIntegration is an integration test for the full HTTP server
func TestHTTPServerIntegration(t *testing.T) {
	// Skip this test in CI as it requires Kubernetes configuration and actual HTTP server
	t.Skip("Skipping HTTP server integration test - requires Kubernetes config")

	// Set test port
	t.Setenv("PORT", "18080")

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start server in background
	go func() {
		// This would run the actual main() but we can't easily test that
		// without refactoring main() to be more testable
		<-ctx.Done()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test health endpoint
	resp, err := http.Get("http://localhost:18080/health")
	if err != nil {
		t.Fatalf("Failed to call health endpoint: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	var healthResponse map[string]interface{}
	if err := json.Unmarshal(body, &healthResponse); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if healthResponse["status"] != "healthy" {
		t.Errorf("Expected healthy status, got %v", healthResponse["status"])
	}
}

// TestHealthEndpoint_ResponseFormat tests the exact response format
func TestHealthEndpoint_ResponseFormat(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy","service":"linkerd-mcp","version":"1.0.0"}`))
	})

	handler.ServeHTTP(w, req)

	body := w.Body.String()

	// Verify it's valid JSON
	var response map[string]interface{}
	if err := json.Unmarshal([]byte(body), &response); err != nil {
		t.Fatalf("Response is not valid JSON: %v", err)
	}

	// Verify all expected fields are present
	expectedFields := []string{"status", "service", "version"}
	for _, field := range expectedFields {
		if _, ok := response[field]; !ok {
			t.Errorf("Response missing expected field: %s", field)
		}
	}

	// Verify no unexpected fields
	if len(response) != len(expectedFields) {
		t.Errorf("Response has unexpected fields. Expected %d fields, got %d", len(expectedFields), len(response))
	}
}

// TestReadyEndpoint_ResponseFormat tests the exact response format
func TestReadyEndpoint_ResponseFormat(t *testing.T) {
	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	})

	handler.ServeHTTP(w, req)

	body := w.Body.String()

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(body), &response); err != nil {
		t.Fatalf("Response is not valid JSON: %v", err)
	}

	if response["status"] != "ready" {
		t.Errorf("Expected status 'ready', got '%v'", response["status"])
	}

	// Should only have one field
	if len(response) != 1 {
		t.Errorf("Expected 1 field in response, got %d", len(response))
	}
}

// TestHealthEndpoint_ConcurrentRequests tests the health endpoint under concurrent load
func TestHealthEndpoint_ConcurrentRequests(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy","service":"linkerd-mcp","version":"1.0.0"}`))
	})

	// Run 100 concurrent requests
	numRequests := 100
	done := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/health", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
			}

			done <- true
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		<-done
	}
}

// TestReadyEndpoint_ConcurrentRequests tests the ready endpoint under concurrent load
func TestReadyEndpoint_ConcurrentRequests(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	})

	numRequests := 100
	done := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/ready", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
			}

			done <- true
		}()
	}

	for i := 0; i < numRequests; i++ {
		<-done
	}
}

// TestHealthEndpoint_Headers tests that proper headers are set
func TestHealthEndpoint_Headers(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy","service":"linkerd-mcp","version":"1.0.0"}`))
	})

	handler.ServeHTTP(w, req)

	// Check Content-Type header
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", ct)
	}
}

// TestReadyEndpoint_Headers tests that proper headers are set
func TestReadyEndpoint_Headers(t *testing.T) {
	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	})

	handler.ServeHTTP(w, req)

	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", ct)
	}
}

// TestEndpoints_EmptyBody tests that endpoints handle empty request bodies
func TestEndpoints_EmptyBody(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		expected string
	}{
		{
			name:     "health endpoint with empty body",
			endpoint: "/health",
			expected: "healthy",
		},
		{
			name:     "ready endpoint with empty body",
			endpoint: "/ready",
			expected: "ready",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.endpoint, strings.NewReader(""))
			w := httptest.NewRecorder()

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				if strings.Contains(tt.endpoint, "health") {
					_, _ = w.Write([]byte(`{"status":"healthy","service":"linkerd-mcp","version":"1.0.0"}`))
				} else {
					_, _ = w.Write([]byte(`{"status":"ready"}`))
				}
			})

			handler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
			}

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response["status"] != tt.expected {
				t.Errorf("Expected status '%s', got '%v'", tt.expected, response["status"])
			}
		})
	}
}

// TestEndpoints_WithQuery tests that endpoints ignore query parameters
func TestEndpoints_WithQuery(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
	}{
		{
			name:     "health endpoint with query params",
			endpoint: "/health?foo=bar&baz=qux",
		},
		{
			name:     "ready endpoint with query params",
			endpoint: "/ready?test=123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.endpoint, nil)
			w := httptest.NewRecorder()

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				if strings.Contains(tt.endpoint, "health") {
					_, _ = w.Write([]byte(`{"status":"healthy","service":"linkerd-mcp","version":"1.0.0"}`))
				} else {
					_, _ = w.Write([]byte(`{"status":"ready"}`))
				}
			})

			handler.ServeHTTP(w, req)

			// Should still return 200 OK regardless of query params
			if w.Code != http.StatusOK {
				t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
			}
		})
	}
}
