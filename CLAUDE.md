# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

This is a Model Context Protocol (MCP) server that enables AI agents to interact with Linkerd service mesh in Kubernetes clusters. The server exposes tools for health monitoring, service discovery, and authorization policy analysis.

The server uses **StreamableHTTP transport** (the modern MCP standard as of specification version 2025-03-26), which replaces the deprecated SSE transport.

## Commands

### Build & Run
```bash
# Build the binary
go build -o linkerd-mcp

# Run the server (uses local kubeconfig)
go run main.go
./linkerd-mcp

# Build with optimized size
go build -v -ldflags="-s -w" -o linkerd-mcp .
```

### Testing
```bash
# Run all tests
go test ./internal/... -v

# Run tests with coverage
go test ./internal/... -v -race -coverprofile=coverage.out -covermode=atomic

# Generate coverage HTML report
go tool cover -html=coverage.out -o coverage.html

# Run specific package tests
go test ./internal/health -v
go test ./internal/mesh -v
go test ./internal/policy -v
go test ./internal/server -v

# Run a single test
go test ./internal/health -run TestCheckMeshHealth_HealthyControlPlane -v
```

### Linting
```bash
# Run golangci-lint (CI uses this)
golangci-lint run --timeout=5m
```

### Docker
```bash
# Build Docker image
docker build -t linkerd-mcp:latest .

# Run in Docker (mount kubeconfig)
docker run --rm -v ~/.kube/config:/root/.kube/config linkerd-mcp:latest
```

### Helm
```bash
# Lint the Helm chart
helm lint helm/linkerd-mcp

# Template the chart to see rendered YAML
helm template linkerd-mcp helm/linkerd-mcp --namespace linkerd

# Install to Kubernetes
helm install linkerd-mcp helm/linkerd-mcp -n linkerd --create-namespace
```

## Architecture

The codebase uses a modular, domain-driven architecture with clean separation of concerns. The original monolithic main.go (~710 lines) was refactored into focused internal packages (~29 line main.go).

### Package Structure

```
linkerd-mcp/
├── main.go                    # Entry point - initializes server and registers tools
└── internal/
    ├── config/                # Kubernetes client initialization (in-cluster + kubeconfig)
    ├── health/                # Linkerd control plane health checking
    ├── mesh/                  # Service mesh discovery (finds meshed services/pods)
    ├── policy/                # Authorization policy analysis (4 files)
    │   ├── analyzer.go        # Public API and AnalyzeConnectivity
    │   ├── targets.go         # GetAllowedTargets - what can source reach
    │   ├── sources.go         # GetAllowedSources - who can reach target
    │   └── auth.go            # Authentication matching (MeshTLS, Network, ServiceAccount)
    ├── server/                # MCP server setup and tool registration
    └── testutil/              # Test helpers (fixtures, MCP result parsing)
```

### Component Flow

1. **main.go** initializes `LinkerdMCPServer` via `server.New()`
2. **server.New()** creates Kubernetes clients via `config.NewKubernetesClients()`
3. Clients are injected into domain components (health, mesh, policy)
4. **RegisterTools()** registers 5 MCP tools with handlers
5. Server runs using stdio transport (`mcpserver.ServeStdio`)

### Key Dependencies

- **mcp-go**: MCP protocol implementation - `github.com/mark3labs/mcp-go`
- **client-go**: Standard Kubernetes API client (Pods, Services, etc.)
- **dynamic client**: For querying Linkerd CRDs (Server, AuthorizationPolicy, MeshTLSAuthentication, etc.)

### MCP Tools Provided

1. `check_mesh_health` - Health status of Linkerd control plane pods
2. `analyze_connectivity` - Point-to-point connectivity analysis between services
3. `list_meshed_services` - Discover all services with linkerd-proxy injected
4. `get_allowed_targets` - Find all targets a source service can access
5. `get_allowed_sources` - Find all sources that can access a target service

## Linkerd Policy Analysis

The policy analyzer (`internal/policy/`) queries Linkerd CRDs to determine authorization:

- **Server**: Defines server-side policy targets (podSelector + port)
- **AuthorizationPolicy**: References Server + list of authentication refs
- **MeshTLSAuthentication**: Allows identities or service accounts (mTLS)
- **NetworkAuthentication**: Allows network CIDRs (unauthenticated/plaintext)

### Policy Analysis Flow

**For GetAllowedTargets:**
1. Find source pod → extract service account
2. List all Servers across namespaces
3. For each Server, check AuthorizationPolicies
4. Match authentication rules (identities, service accounts, networks)
5. Return list of authorized target services

**For GetAllowedSources:**
1. Find target service → match to Servers (by pod selector)
2. For each Server, list AuthorizationPolicies
3. Parse authentication refs (MeshTLS, Network)
4. Resolve identities/service accounts to source pods
5. Return list of authorized source services

## Testing Approach

Tests use **fake Kubernetes clients** (`k8s.io/client-go/kubernetes/fake`) for fast, isolated unit tests. Integration tests that require real clusters are marked with `t.Skip()`.

### Test Utilities

- `testutil.CreateMeshedPod()` - Create pod with linkerd-proxy sidecar
- `testutil.CreateServer()` - Create Linkerd Server CRD
- `testutil.CreateAuthorizationPolicy()` - Create AuthorizationPolicy CRD
- `testutil.CreateMeshTLSAuthentication()` - Create MeshTLS auth with identities/SAs
- `testutil.ParseJSONResult()` - Parse MCP CallToolResult JSON content

### Running Integration Tests

Integration tests are skipped by default. To run against a real cluster:
1. Remove `t.Skip()` from test function
2. Ensure `KUBECONFIG` points to valid cluster
3. Run: `go test ./internal/server -v -run TestNew_Success`

## CI/CD

GitHub Actions workflows (`.github/workflows/`):

- **ci.yml**: Runs on PR/push - lint, test (Go 1.25/1.25), build multi-platform binaries, security scans
- **docker.yml**: Builds and pushes multi-arch Docker images to GHCR
- **release.yml**: Creates GitHub releases with binaries for all platforms

## Environment Variables

- `KUBECONFIG`: Path to kubeconfig file (for local development)
- `LINKERD_NAMESPACE`: Override Linkerd control plane namespace (default: "linkerd")

## RBAC Requirements

When running in-cluster, the server needs:
- **pods, services, namespaces**: Read access (core API)
- **servers.policy.linkerd.io**: Read access
- **authorizationpolicies.policy.linkerd.io**: Read access
- **meshtlsauthentications.policy.linkerd.io**: Read access
- **networkauthentications.policy.linkerd.io**: Read access
- **httproutes.policy.linkerd.io**: Read access
- **deployments, replicasets**: Read access (for service account resolution)

See `helm/linkerd-mcp/templates/rbac.yaml` for complete ClusterRole definition.

## Adding New Tools

To add a new MCP tool:

1. Create new package under `internal/` (e.g., `internal/metrics/`)
2. Implement the tool logic with constructor pattern:
   ```go
   type Collector struct {
       clientset *kubernetes.Clientset
   }

   func NewCollector(clientset *kubernetes.Clientset) *Collector {
       return &Collector{clientset: clientset}
   }

   func (c *Collector) GetMetrics(ctx context.Context, namespace string) (*mcp.CallToolResult, error) {
       // Implementation
   }
   ```
3. Add component to `server.LinkerdMCPServer` struct
4. Initialize in `server.New()` constructor
5. Register tool in `server.RegisterTools()`:
   ```go
   metricsTool := mcp.NewTool("get_metrics",
       mcp.WithDescription("Get Linkerd metrics"),
       mcp.WithString("namespace", mcp.Description("Namespace to query")),
   )
   mcpServer.AddTool(metricsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
       args, _ := request.Params.Arguments.(map[string]interface{})
       namespace, _ := args["namespace"].(string)
       return s.metricsCollector.GetMetrics(ctx, namespace)
   })
   ```
6. Write unit tests using fake clients

## Kubernetes Client Patterns

### Standard vs Dynamic Client

- **Clientset** (`kubernetes.Clientset`): Use for built-in resources (Pods, Services, Namespaces)
- **DynamicClient** (`dynamic.Interface`): Use for CRDs (Linkerd Servers, AuthorizationPolicies)

### Dynamic Client Usage

```go
import (
    "k8s.io/apimachinery/pkg/runtime/schema"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Define GVR (GroupVersionResource)
serverGVR := schema.GroupVersionResource{
    Group:    "policy.linkerd.io",
    Version:  "v1beta3",
    Resource: "servers",
}

// List resources
list, err := dynamicClient.Resource(serverGVR).
    Namespace(namespace).
    List(ctx, metav1.ListOptions{})

// Parse unstructured data
for _, item := range list.Items {
    spec := item.Object["spec"].(map[string]interface{})
    podSelector := spec["podSelector"].(map[string]interface{})
    // ...
}
```

## Go Version

Project uses **Go 1.25** (see go.mod). CI tests against Go 1.25 and 1.25 for compatibility.
