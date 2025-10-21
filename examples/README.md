# Linkerd MCP Examples

This directory contains example programs demonstrating how to use the Linkerd MCP server tools.

## Available Examples

### test-mesh-health.go

Tests the `check_mesh_health` MCP tool to verify Linkerd control plane health.

**Usage:**
```bash
go run examples/test-mesh-health.go
```

**What it does:**
- Initializes Kubernetes clients (uses your local kubeconfig)
- Calls the `check_mesh_health` tool for the `linkerd` namespace
- Parses and displays the health status in a readable format
- Shows summary statistics (total/healthy/unhealthy pods)
- Lists individual component status

**Expected Output:**
```
=== Testing Linkerd MCP Mesh Health Endpoint ===

Initializing Kubernetes clients...
✓ Kubernetes clients initialized

Checking Linkerd mesh health in 'linkerd' namespace...
✓ Mesh health check complete

=== Linkerd Mesh Health Status ===
{
  "components": [
    {
      "component": "destination",
      "healthy": true,
      "name": "linkerd-destination-55bf8f8ddc-gzsvv",
      "status": "Running"
    },
    ...
  ],
  "healthyPods": 3,
  "namespace": "linkerd",
  "totalPods": 3,
  "unhealthyPods": 0
}

=== Summary ===
Namespace:      linkerd
Total Pods:     3
Healthy Pods:   3
Unhealthy Pods: 0

✓ All control plane components are healthy!

=== Component Details ===
✓ destination         linkerd-destination-55bf8f8ddc-gzsvv    Running
✓ identity            linkerd-identity-6bd7865b75-9glgt       Running
✓ proxy-injector      linkerd-proxy-injector-5ccdc76744-rzj4b Running
```

## Requirements

- Go 1.25+
- Access to a Kubernetes cluster with Linkerd installed
- Valid kubeconfig (usually `~/.kube/config`)

## Running Examples

All examples can be run directly from the project root:

```bash
# Run mesh health test
go run examples/test-mesh-health.go
```

Or build and run:

```bash
# Build
go build -o test-mesh-health examples/test-mesh-health.go

# Run
./test-mesh-health
```

## Adding More Examples

To add a new example:

1. Create a new `.go` file in this directory
2. Import the required internal packages
3. Document the example in this README
4. Test it works with: `go run examples/your-example.go`
