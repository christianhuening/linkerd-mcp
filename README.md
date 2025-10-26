# Linkerd MCP Server

[![CI](https://github.com/christianhuening/linkerd-mcp/workflows/CI/badge.svg)](https://github.com/christianhuening/linkerd-mcp/actions/workflows/ci.yml)
[![Docker Build](https://github.com/christianhuening/linkerd-mcp/workflows/Docker%20Build/badge.svg)](https://github.com/christianhuening/linkerd-mcp/actions/workflows/docker.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/christianhuening/linkerd-mcp)](https://goreportcard.com/report/github.com/christianhuening/linkerd-mcp)
[![Go Version](https://img.shields.io/github/go-mod/go-version/christianhuening/linkerd-mcp)](https://go.dev/doc/devel/release)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A Model Context Protocol (MCP) server for interacting with Linkerd service mesh in Kubernetes clusters. This server enables AI agents to query service mesh health status and analyze connectivity policies between services.

DISCLAIMER: This project has been created with Claude.AI!

## Features

- **Service Mesh Health Monitoring**: Check the health status of Linkerd control plane components
- **Connectivity Analysis**: Analyze Linkerd policies to determine allowed connectivity between services
- **Service Discovery**: List all services that are part of the Linkerd mesh
- **Authorization Policy Analysis**: Query which services can access a target or what targets a source can reach
- **Configuration Validation**: Validate Linkerd resources (Servers, AuthorizationPolicies, MeshTLS) for correctness and best practices
- **Production Ready**: Modern security features, Helm charts, comprehensive tests, and CI/CD

## Quick Start

### Using with Claude Desktop (Recommended)

1. **Install the server:**
   ```bash
   # From source
   git clone https://github.com/christianhuening/linkerd-mcp.git
   cd linkerd-mcp
   go build -o linkerd-mcp

   # Or download pre-built binary from releases
   ```

2. **Configure Claude Desktop** (`~/Library/Application Support/Claude/claude_desktop_config.json`):
   ```json
   {
     "mcpServers": {
       "linkerd": {
         "command": "/path/to/linkerd-mcp",
         "args": [],
         "env": {
           "KUBECONFIG": "/Users/yourname/.kube/config"
         }
       }
     }
   }
   ```

3. **Use in Claude Desktop:**
   - "Check the health of my Linkerd mesh"
   - "List all services in the mesh"
   - "Validate my Linkerd configuration in the prod namespace"
   - "What services can the frontend service communicate with?"

### Using Docker

```bash
docker pull ghcr.io/christianhuening/linkerd-mcp:latest
docker run --rm -v ~/.kube/config:/root/.kube/config ghcr.io/christianhuening/linkerd-mcp:latest
```

### Using Helm (In-Cluster Deployment)

```bash
helm install linkerd-mcp ./helm/linkerd-mcp -n linkerd --create-namespace
```

### Testing with MCP Inspector

```bash
# Build from source
go build -o linkerd-mcp

# Run inspector
npx @modelcontextprotocol/inspector ./linkerd-mcp
```

This opens a web interface where you can test all MCP tools interactively.

## MCP Tools

### 1. `check_mesh_health`
Checks the health status of the Linkerd service mesh in the cluster.

**Arguments:**
- `namespace` (optional): Linkerd control plane namespace (default: "linkerd")

**Returns:** JSON with control plane pod status and health information

### 2. `analyze_connectivity`
Analyzes Linkerd policies to determine allowed connectivity between services.

**Arguments:**
- `source_namespace` (required): Source service namespace
- `source_service` (required): Source service name
- `target_namespace` (optional): Target service namespace (defaults to source namespace)
- `target_service` (required): Target service name

**Returns:** JSON with connectivity analysis and applicable policies

### 3. `list_meshed_services`
Lists all services that are part of the Linkerd mesh.

**Arguments:**
- `namespace` (optional): Filter by namespace (default: all namespaces)

**Returns:** JSON list of meshed services with their pods

### 4. `get_allowed_targets`
Find all services that a given source service can communicate with based on Linkerd authorization policies.

**Arguments:**
- `source_namespace` (required): Namespace of the source service
- `source_service` (required): Name of the source service

**Returns:** JSON list of all targets the source is authorized to access

### 5. `get_allowed_sources`
Find all services that can communicate with a given target service based on Linkerd authorization policies.

**Arguments:**
- `target_namespace` (required): Namespace of the target service
- `target_service` (required): Name of the target service

**Returns:** JSON list of all sources authorized to access the target

### 6. `validate_mesh_config`
Validates Linkerd service mesh configuration for correctness and best practices.

**Arguments:**
- `namespace` (optional): Namespace to validate (default: all namespaces)
- `resource_type` (optional): Resource type to validate - `server`, `authpolicy`, `meshtls`, `proxy`, `namespace`, or `all` (default: `all`)
- `resource_name` (optional): Specific resource name to validate
- `include_warnings` (optional): Include warnings in results (default: true)

**Returns:** JSON validation report with errors, warnings, and informational messages

**Supported Validations:**
- **Server Resources**: Port configuration, pod selectors, proxy protocol, port conflicts
- **AuthorizationPolicy Resources**: Target references, authentication references, policy consistency
- **MeshTLSAuthentication Resources**: Identity format, service account references
- **Proxy Configuration**: Injection annotations, CPU/memory resources, log levels, proxy versions (namespace and pod level)

**Example Usage (via Claude Desktop or MCP Inspector):**

When using Claude Desktop, you can ask:
- "Validate my Linkerd configuration"
- "Check for any configuration errors in the prod namespace"
- "Validate all Server resources and show me any issues"
- "Check proxy configuration in the default namespace"
- "Validate namespace annotations for proper Linkerd proxy injection"

Or use the MCP Inspector for testing:
```bash
npx @modelcontextprotocol/inspector ./linkerd-mcp
# Then call: validate_mesh_config with arguments: {}
```

## Prerequisites

- Go 1.23 or later
- Kubernetes cluster with Linkerd installed
- `kubectl` configured to access your cluster
- Docker (for containerization)

## Local Development

### Installation

1. Clone the repository:
```bash
git clone https://github.com/christianhuening/linkerd-mcp.git
cd linkerd-mcp
```

2. Install dependencies:
```bash
go mod download
```

3. Run the server:
```bash
go run main.go
```

The server will use your local kubeconfig (`~/.kube/config`) to connect to your Kubernetes cluster.

## Kubernetes Deployment

### Build and Push Docker Image

```bash
# Build the image
docker build -t your-registry/linkerd-mcp:latest .

# Push to your registry
docker push your-registry/linkerd-mcp:latest
```

### Deploy to Kubernetes

1. Update the image in k8s/deployment.yaml to point to your registry:
```yaml
image: your-registry/linkerd-mcp:latest
```

2. Deploy to your cluster:
```bash
kubectl apply -f k8s/deployment.yaml
```

This will create:
- ServiceAccount with necessary RBAC permissions
- ClusterRole with access to pods, services, and Linkerd policy CRDs
- Deployment running the MCP server
- Service exposing the MCP server

### Verify Deployment

```bash
# Check if the pod is running
kubectl get pods -n linkerd -l app=linkerd-mcp

# Check logs
kubectl logs -n linkerd -l app=linkerd-mcp
```

## RBAC Permissions

The server requires the following Kubernetes permissions:
- Read access to pods, services, and namespaces
- Read access to Linkerd policy CRDs (servers, serverauthorizations, authorizationpolicies, httproutes)
- Read access to deployments and replicasets

These are configured in k8s/deployment.yaml.

## Configuration

### Environment Variables

- `KUBECONFIG`: Path to kubeconfig file (for local development)
- `LINKERD_NAMESPACE`: Linkerd control plane namespace (default: "linkerd")

## Architecture

The server uses a modular architecture with clean separation of concerns:

```
linkerd-mcp/
├── main.go                    # Entry point (31 lines)
└── internal/
    ├── config/                # Kubernetes client configuration
    ├── health/                # Control plane health checking
    ├── mesh/                  # Service discovery
    ├── policy/                # Authorization policy analysis
    └── server/                # MCP server and tool registration
```

**Key Technologies:**
- **mcp-go**: Go implementation of the Model Context Protocol
- **client-go**: Official Kubernetes Go client
- **Linkerd Policy CRDs**: For analyzing service-to-service connectivity policies

See [internal/README.md](internal/README.md) for detailed architecture documentation.

## Testing

The project includes comprehensive unit tests with 67+ test cases using Ginkgo/Gomega BDD framework:

```bash
# Run all tests
go test ./internal/... -v

# Run tests with coverage
go test ./internal/... -cover

# Generate coverage report
go test ./internal/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

See [TESTING.md](TESTING.md) for complete testing documentation.

## CI/CD

The project uses GitHub Actions for continuous integration and deployment:

- **CI Workflow**: Runs tests, linting, and security scans on every PR
- **Docker Workflow**: Builds multi-platform images and pushes to GHCR
- **Release Workflow**: Creates releases with binaries for all platforms

See [.github/workflows/README.md](.github/workflows/README.md) for CI/CD documentation.

## Future Enhancements

- [ ] Complete Linkerd policy CRD integration for detailed connectivity analysis
- [ ] Add support for analyzing traffic metrics
- [x] Implement service mesh configuration validation
- [ ] Add support for multi-cluster Linkerd setups
- [ ] Provide detailed route and authorization policy insights
- [x] Add proxy configuration validation
- [ ] Implement best practice recommendations
- [ ] Add validation for HTTPRoute resources
- [ ] Add validation for ServiceProfile resources

## License

MIT

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.
