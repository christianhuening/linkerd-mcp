# Linkerd MCP Server

A Model Context Protocol (MCP) server for interacting with Linkerd service mesh in Kubernetes clusters. This server enables AI agents to query service mesh health status and analyze connectivity policies between services.

## Features

- **Service Mesh Health Monitoring**: Check the health status of Linkerd control plane components
- **Connectivity Analysis**: Analyze Linkerd policies to determine allowed connectivity between services
- **Service Discovery**: List all services that are part of the Linkerd mesh

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

The server uses:
- **mcp-go**: Go implementation of the Model Context Protocol
- **client-go**: Official Kubernetes Go client
- **Linkerd Policy CRDs**: For analyzing service-to-service connectivity policies

## Future Enhancements

- [ ] Complete Linkerd policy CRD integration for detailed connectivity analysis
- [ ] Add support for analyzing traffic metrics
- [ ] Implement service mesh configuration validation
- [ ] Add support for multi-cluster Linkerd setups
- [ ] Provide detailed route and authorization policy insights

## License

MIT

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.
