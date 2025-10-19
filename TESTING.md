# Testing Guide for Linkerd MCP Server

This document describes the test suite for the Linkerd MCP server and how to run and extend the tests.

## Test Structure

```
internal/
├── config/           # No tests (simple initialization)
├── health/
│   └── checker_test.go       # 7 test cases
├── mesh/
│   └── services_test.go      # 8 test cases
├── policy/
│   ├── analyzer_test.go      # 7 test cases
│   ├── auth.go
│   ├── sources.go
│   └── targets.go
├── server/
│   └── server_test.go        # 11 test cases
└── testutil/
    ├── fixtures.go           # Test fixture helpers
    └── mcp_helpers.go        # MCP result parsing helpers
```

## Running Tests

### Run All Tests

```bash
go test ./internal/...
```

### Run Tests with Verbose Output

```bash
go test ./internal/... -v
```

### Run Tests for Specific Package

```bash
go test ./internal/health -v
go test ./internal/mesh -v
go test ./internal/policy -v
go test ./internal/server -v
```

### Run Specific Test

```bash
go test ./internal/health -run TestCheckMeshHealth_HealthyControlPlane -v
```

### Run Tests with Coverage

```bash
go test ./internal/... -cover
```

### Generate Coverage Report

```bash
go test ./internal/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

## Test Categories

### Unit Tests

Most tests are unit tests that use fake Kubernetes clients:

- **health package**: Tests mesh health checking logic
- **mesh package**: Tests service discovery
- **policy package**: Tests authorization policy analysis
- **server package**: Tests tool registration and argument parsing

### Integration Tests

Some tests require a Kubernetes cluster and are marked with `t.Skip()`:

```go
func TestNew_Success(t *testing.T) {
    t.Skip("Skipping integration test that requires Kubernetes config")
    // ...
}
```

To run integration tests against a real cluster:
```bash
# Remove t.Skip() from the test, then:
go test ./internal/server -v -run TestNew_Success
```

## Test Utilities

### testutil Package

The `testutil` package provides helpers for creating test fixtures and parsing results.

#### Creating Test Pods

```go
import "github.com/christianhuening/linkerd-mcp/internal/testutil"

// Regular pod
pod := testutil.CreatePod("pod-1", "default", "default", labels, corev1.PodRunning, true)

// Meshed pod (with linkerd-proxy)
meshedPod := testutil.CreateMeshedPod("web-1", "prod", "web")

// Control plane pod
cpPod := testutil.CreateLinkerdControlPlanePod("destination-1", "linkerd", "destination", corev1.PodRunning, true)
```

#### Creating Linkerd CRDs

```go
// Server CRD
server := testutil.CreateServer("backend-server", "prod", map[string]string{"app": "backend"}, 8080)

// AuthorizationPolicy
authPolicy := testutil.CreateAuthorizationPolicy(
    "allow-frontend",
    "prod",
    "backend-server",
    []map[string]string{{"name": "frontend-auth", "kind": "MeshTLSAuthentication"}},
)

// MeshTLSAuthentication with identities
meshAuth := testutil.CreateMeshTLSAuthentication(
    "frontend-auth",
    "prod",
    []string{"frontend-sa.prod.serviceaccount.identity.linkerd.cluster.local"},
    nil,
)

// MeshTLSAuthentication with service accounts
meshAuth := testutil.CreateMeshTLSAuthentication(
    "frontend-auth",
    "prod",
    nil,
    []map[string]string{{"name": "frontend-sa", "namespace": "prod"}},
)
```

#### Parsing MCP Results

```go
result, err := checker.CheckMeshHealth(ctx, "linkerd")

// Parse JSON from result
var healthStatus map[string]interface{}
if err := testutil.ParseJSONResult(result, &healthStatus); err != nil {
    t.Fatalf("Failed to parse result: %v", err)
}

// Or get raw text
text, err := testutil.GetTextFromResult(result)
```

## Test Coverage by Package

### health Package (7 tests)

1. **TestCheckMeshHealth_HealthyControlPlane**
   - Tests healthy Linkerd control plane
   - Verifies all pods are marked as healthy

2. **TestCheckMeshHealth_UnhealthyControlPlane**
   - Tests mix of healthy and unhealthy pods
   - Verifies correct status reporting

3. **TestCheckMeshHealth_EmptyNamespaceDefaultsToLinkerd**
   - Tests default namespace behavior

4. **TestCheckMeshHealth_NoControlPlanePods**
   - Tests empty cluster scenario

5. **TestCheckMeshHealth_CustomNamespace**
   - Tests non-default namespace

6. **Additional tests**: Constructor and edge cases

### mesh Package (8 tests)

1. **TestListMeshedServices_WithMeshedPods**
   - Tests discovery of multiple meshed services
   - Verifies pod grouping by service

2. **TestListMeshedServices_FilterByNamespace**
   - Tests namespace filtering

3. **TestListMeshedServices_NoMeshedPods**
   - Tests when no pods have linkerd-proxy

4. **TestListMeshedServices_PodsWithoutAppLabel**
   - Tests pods missing service labels

5. **TestListMeshedServices_K8sAppLabel**
   - Tests alternative k8s-app label

6. **TestListMeshedServices_MultiplePodsPerService**
   - Tests multiple replicas per service

7. **Additional tests**: Constructor and edge cases

### policy Package (7 tests)

1. **TestAnalyzeConnectivity**
   - Tests basic connectivity analysis

2. **TestAnalyzeConnectivity_DefaultTargetNamespace**
   - Tests namespace defaulting

3. **TestGetAllowedTargets_NoPodsFound**
   - Tests error handling for missing pods

4. **TestGetAllowedTargets_WithPods**
   - Tests target discovery with policies

5. **TestGetAllowedSources_NoServersFound**
   - Tests when no servers exist

6. **TestGetAllowedSources_WithServersAndPolicies**
   - Tests wildcard authentication

7. **TestGetAllowedSources_WithServiceAccounts**
   - Tests service account-based auth

### server Package (11 tests)

1. **TestNew_Success**
   - Integration test for server creation (skipped by default)

2. **TestRegisterTools**
   - Tests tool registration

3. **TestLinkerdMCPServer_Structure**
   - Tests struct initialization

4-9. **TestToolRegistration_***
   - Tests argument parsing for each tool

10-11. **TestToolRegistration_EmptyArguments** / **NilArguments**
    - Tests edge cases

## Writing New Tests

### Example: Adding a Test for a New Feature

```go
package mypackage

import (
    "context"
    "testing"

    "github.com/christianhuening/linkerd-mcp/internal/testutil"
    "k8s.io/client-go/kubernetes/fake"
)

func TestMyNewFeature(t *testing.T) {
    // 1. Setup: Create fake clientset with test data
    clientset := fake.NewSimpleClientset(
        testutil.CreateMeshedPod("test-1", "default", "test"),
    )

    // 2. Create component under test
    component := NewComponent(clientset)

    // 3. Execute
    result, err := component.DoSomething(context.Background(), "input")

    // 4. Assert
    if err != nil {
        t.Fatalf("Expected no error, got: %v", err)
    }

    var response map[string]interface{}
    if err := testutil.ParseJSONResult(result, &response); err != nil {
        t.Fatalf("Failed to parse result: %v", err)
    }

    if response["expected_field"] != "expected_value" {
        t.Errorf("Expected 'expected_value', got: %v", response["expected_field"])
    }
}
```

### Best Practices

1. **Arrange-Act-Assert**: Structure tests clearly
2. **Descriptive names**: `TestFunction_Scenario` format
3. **One assertion focus**: Each test should verify one thing
4. **Use test fixtures**: Leverage testutil helpers
5. **Clean up**: Use defer for cleanup if needed
6. **Document edge cases**: Comment why edge cases exist

### Table-Driven Tests

For testing multiple scenarios:

```go
func TestMultipleScenarios(t *testing.T) {
    tests := []struct {
        name      string
        input     string
        expected  string
        wantError bool
    }{
        {"valid input", "test", "result", false},
        {"empty input", "", "", true},
        {"special chars", "test@123", "result", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := Function(tt.input)

            if tt.wantError && err == nil {
                t.Error("Expected error but got none")
            }

            if !tt.wantError && err != nil {
                t.Errorf("Unexpected error: %v", err)
            }

            if result != tt.expected {
                t.Errorf("Expected %s, got %s", tt.expected, result)
            }
        })
    }
}
```

## Continuous Integration

### GitHub Actions Example

```yaml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: Run tests
        run: go test ./internal/... -v -cover
      - name: Generate coverage
        run: go test ./internal/... -coverprofile=coverage.out
      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.out
```

## Troubleshooting

### Common Issues

**Issue**: Tests fail with "no such file or directory"
**Solution**: Run `go mod tidy` to ensure dependencies are installed

**Issue**: Type mismatch with fake clients
**Solution**: Use `kubernetes.Clientset` interface, not concrete type

**Issue**: MCP Content parsing errors
**Solution**: Use `testutil.ParseJSONResult()` or `testutil.GetTextFromResult()`

**Issue**: Dynamic client tests fail
**Solution**: Ensure GVR (GroupVersionResource) is correctly defined

## Test Metrics

Current test coverage (as of last update):

- **health**: 7 test cases covering all public methods
- **mesh**: 8 test cases covering service discovery
- **policy**: 7 test cases covering policy analysis
- **server**: 11 test cases covering tool registration
- **testutil**: Helper functions (tested indirectly)

**Total**: 33+ test cases

## Future Improvements

- [ ] Add integration tests with real Kubernetes cluster
- [ ] Add performance/benchmark tests
- [ ] Add fuzz testing for input validation
- [ ] Increase code coverage to >80%
- [ ] Add end-to-end MCP protocol tests
- [ ] Add tests for error recovery scenarios

## Resources

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Kubernetes Fake Client](https://pkg.go.dev/k8s.io/client-go/kubernetes/fake)
- [MCP Go Library](https://github.com/mark3labs/mcp-go)
- [Table Driven Tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
