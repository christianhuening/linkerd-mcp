# Internal Packages

This directory contains the internal packages for the Linkerd MCP server. These packages are organized by domain responsibility to improve code maintainability and testability.

## Package Structure

```
internal/
├── config/         # Kubernetes client configuration
├── health/         # Mesh health checking
├── mesh/           # Meshed services discovery
├── policy/         # Authorization policy analysis
└── server/         # MCP server setup and tool registration
```

## Package Descriptions

### config
**Purpose**: Kubernetes client initialization and configuration management

**Files**:
- `kubernetes.go`: Creates and configures Kubernetes clientset and dynamic client

**Key Functions**:
- `NewKubernetesClients()`: Initializes both standard and dynamic Kubernetes clients
- `GetKubeConfig()`: Retrieves Kubernetes configuration (in-cluster or kubeconfig)

**Usage**:
```go
clients, err := config.NewKubernetesClients()
if err != nil {
    log.Fatal(err)
}
// clients.Clientset - Standard Kubernetes client
// clients.DynamicClient - Dynamic client for CRDs
```

---

### health
**Purpose**: Linkerd control plane health checking

**Files**:
- `checker.go`: Health checking logic for Linkerd mesh components

**Key Types**:
- `Checker`: Health checker with Kubernetes client

**Key Methods**:
- `CheckMeshHealth(ctx, namespace)`: Checks health of Linkerd control plane pods

**Usage**:
```go
checker := health.NewChecker(clientset)
result, err := checker.CheckMeshHealth(ctx, "linkerd")
```

---

### mesh
**Purpose**: Service mesh discovery and listing

**Files**:
- `services.go`: Lists services with Linkerd proxy injected

**Key Types**:
- `ServiceLister`: Service discovery with Kubernetes client

**Key Methods**:
- `ListMeshedServices(ctx, namespace)`: Lists all meshed services

**Usage**:
```go
lister := mesh.NewServiceLister(clientset)
result, err := lister.ListMeshedServices(ctx, "")
```

---

### policy
**Purpose**: Linkerd authorization policy analysis

**Files**:
- `analyzer.go`: Main analyzer and public API
- `targets.go`: Finding allowed targets for a source
- `sources.go`: Finding allowed sources for a target
- `auth.go`: Authentication policy matching logic

**Key Types**:
- `Analyzer`: Policy analyzer with Kubernetes and dynamic clients

**Key Methods**:
- `AnalyzeConnectivity(ctx, srcNs, srcSvc, tgtNs, tgtSvc)`: Analyzes connectivity between two services
- `GetAllowedTargets(ctx, sourceNs, sourceSvc)`: Finds all targets a source can access
- `GetAllowedSources(ctx, targetNs, targetSvc)`: Finds all sources that can access a target

**Implementation Details**:
- Uses dynamic client to query Linkerd CRDs (Server, AuthorizationPolicy, etc.)
- Parses MeshTLSAuthentication and NetworkAuthentication resources
- Matches service identities, service accounts, and network CIDRs

**Usage**:
```go
analyzer := policy.NewAnalyzer(clientset, dynamicClient)
result, err := analyzer.GetAllowedTargets(ctx, "prod", "frontend")
```

---

### server
**Purpose**: MCP server initialization and tool registration

**Files**:
- `server.go`: Main server setup and tool registration

**Key Types**:
- `LinkerdMCPServer`: Main server struct that coordinates all components

**Key Methods**:
- `New()`: Creates a new LinkerdMCPServer with all components
- `RegisterTools(mcpServer)`: Registers all MCP tools with the MCP server

**Registered Tools**:
1. `check_mesh_health`: Health status of Linkerd control plane
2. `analyze_connectivity`: Connectivity analysis between services
3. `list_meshed_services`: List all meshed services
4. `get_allowed_targets`: Find allowed targets for a source
5. `get_allowed_sources`: Find allowed sources for a target

**Usage**:
```go
linkerdServer, err := server.New()
if err != nil {
    log.Fatal(err)
}

mcpServer := mcpserver.NewMCPServer("linkerd-mcp", "1.0.0")
linkerdServer.RegisterTools(mcpServer)
```

---

## Architecture Diagram

```
main.go
   │
   └─> internal/server
          │
          ├─> internal/config (Kubernetes clients)
          │
          ├─> internal/health (Health checker)
          │      └─> Clientset
          │
          ├─> internal/mesh (Service lister)
          │      └─> Clientset
          │
          └─> internal/policy (Policy analyzer)
                 ├─> Clientset
                 └─> DynamicClient
                     ├─> targets.go (find allowed targets)
                     ├─> sources.go (find allowed sources)
                     └─> auth.go (authentication matching)
```

## Design Principles

1. **Separation of Concerns**: Each package has a single, well-defined responsibility
2. **Dependency Injection**: Clients are passed to constructors, making testing easier
3. **Internal Package**: Using Go's `internal/` pattern prevents external imports
4. **Domain-Driven**: Packages are organized by business domain (health, policy, mesh)
5. **Modularity**: Each component can be tested and maintained independently

## Adding New Features

### Adding a New Tool

1. Create a new package under `internal/` if it's a new domain
2. Implement the tool logic in the new package
3. Add the component to `server.LinkerdMCPServer`
4. Register the tool in `server.RegisterTools()`

### Example: Adding a Metrics Tool

```go
// 1. Create internal/metrics/collector.go
package metrics

type Collector struct {
    clientset *kubernetes.Clientset
}

func NewCollector(clientset *kubernetes.Clientset) *Collector {
    return &Collector{clientset: clientset}
}

func (c *Collector) GetMetrics(ctx context.Context, namespace string) (*mcp.CallToolResult, error) {
    // Implementation
}

// 2. Update internal/server/server.go
type LinkerdMCPServer struct {
    healthChecker   *health.Checker
    serviceLister   *mesh.ServiceLister
    policyAnalyzer  *policy.Analyzer
    metricsCollector *metrics.Collector  // Add this
}

func New() (*LinkerdMCPServer, error) {
    clients, err := config.NewKubernetesClients()
    if err != nil {
        return nil, err
    }

    return &LinkerdMCPServer{
        healthChecker:    health.NewChecker(clients.Clientset),
        serviceLister:    mesh.NewServiceLister(clients.Clientset),
        policyAnalyzer:   policy.NewAnalyzer(clients.Clientset, clients.DynamicClient),
        metricsCollector: metrics.NewCollector(clients.Clientset),  // Add this
    }, nil
}

// 3. Register the tool in RegisterTools()
func (s *LinkerdMCPServer) RegisterTools(mcpServer *server.MCPServer) {
    // ... existing tools ...

    // Add metrics tool
    metricsTool := mcp.NewTool("get_metrics",
        mcp.WithDescription("Get metrics from the Linkerd mesh"),
        mcp.WithString("namespace", mcp.Description("Namespace to query")),
    )
    mcpServer.AddTool(metricsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        args, _ := request.Params.Arguments.(map[string]interface{})
        namespace, _ := args["namespace"].(string)
        return s.metricsCollector.GetMetrics(ctx, namespace)
    })
}
```

## Testing

Each package can be tested independently by mocking the Kubernetes clients:

```go
package health_test

import (
    "testing"
    "github.com/christianhuening/linkerd-mcp/internal/health"
    "k8s.io/client-go/kubernetes/fake"
)

func TestCheckMeshHealth(t *testing.T) {
    // Create fake clientset
    clientset := fake.NewSimpleClientset()

    // Create checker with fake client
    checker := health.NewChecker(clientset)

    // Test
    result, err := checker.CheckMeshHealth(context.Background(), "linkerd")
    // Assertions...
}
```

## Benefits of This Architecture

1. **Maintainability**: Easy to locate and modify specific functionality
2. **Testability**: Each package can be unit tested independently
3. **Reusability**: Components can be reused in different contexts
4. **Scalability**: Easy to add new features without affecting existing code
5. **Readability**: Clear package boundaries make the code easier to understand
6. **Type Safety**: Strong typing at package boundaries reduces errors

## Migration Notes

The original `main.go` had **~710 lines** of code. After refactoring:
- `main.go`: **29 lines** (96% reduction)
- Well-organized internal packages with clear responsibilities
- Easier to navigate and maintain
- Better separation of concerns
