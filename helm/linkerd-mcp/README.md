# Linkerd MCP Helm Chart

A Helm chart for deploying the Linkerd Model Context Protocol (MCP) server on Kubernetes. This MCP server provides AI assistants with tools to observe and analyze Linkerd service mesh deployments.

## Prerequisites

- Kubernetes 1.29+
- Helm 3.0+
- Linkerd service mesh installed in your cluster

## Installing the Chart

To install the chart with the release name `linkerd-mcp`:

```bash
helm install linkerd-mcp ./helm/linkerd-mcp -n linkerd --create-namespace
```

## Uninstalling the Chart

To uninstall/delete the `linkerd-mcp` deployment:

```bash
helm uninstall linkerd-mcp -n linkerd
```

## Configuration

The following table lists the configurable parameters of the Linkerd MCP chart and their default values.

### Basic Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | `1` |
| `image.repository` | Image repository | `linkerd-mcp` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `image.tag` | Image tag (defaults to chart appVersion) | `""` |
| `imagePullSecrets` | Image pull secrets | `[]` |
| `nameOverride` | Override chart name | `""` |
| `fullnameOverride` | Override full name | `""` |

### Service Account

| Parameter | Description | Default |
|-----------|-------------|---------|
| `serviceAccount.create` | Create service account | `true` |
| `serviceAccount.automount` | Automount service account token | `true` |
| `serviceAccount.annotations` | Service account annotations | `{}` |
| `serviceAccount.name` | Service account name | `""` |

### Security Context

| Parameter | Description | Default |
|-----------|-------------|---------|
| `podSecurityContext.runAsNonRoot` | Run as non-root user | `true` |
| `podSecurityContext.runAsUser` | User ID | `65532` |
| `podSecurityContext.runAsGroup` | Group ID | `65532` |
| `podSecurityContext.fsGroup` | Filesystem group | `65532` |
| `podSecurityContext.seccompProfile.type` | Seccomp profile type | `RuntimeDefault` |
| `securityContext.readOnlyRootFilesystem` | Read-only root filesystem | `true` |
| `securityContext.allowPrivilegeEscalation` | Allow privilege escalation | `false` |
| `securityContext.capabilities.drop` | Capabilities to drop | `["ALL"]` |

### Service Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `service.enabled` | Enable service | `true` |
| `service.type` | Service type | `ClusterIP` |
| `service.port` | Service port | `8080` |
| `service.targetPort` | Target port | `8080` |
| `service.protocol` | Service protocol | `TCP` |
| `service.annotations` | Service annotations | `{}` |
| `service.labels` | Service labels | `{}` |

### Resources

| Parameter | Description | Default |
|-----------|-------------|---------|
| `resources.limits.cpu` | CPU limit | `500m` |
| `resources.limits.memory` | Memory limit | `256Mi` |
| `resources.requests.cpu` | CPU request | `100m` |
| `resources.requests.memory` | Memory request | `128Mi` |

### Autoscaling

| Parameter | Description | Default |
|-----------|-------------|---------|
| `autoscaling.enabled` | Enable HPA | `false` |
| `autoscaling.minReplicas` | Minimum replicas | `1` |
| `autoscaling.maxReplicas` | Maximum replicas | `3` |
| `autoscaling.targetCPUUtilizationPercentage` | Target CPU utilization | `80` |
| `autoscaling.targetMemoryUtilizationPercentage` | Target memory utilization | `80` |

### RBAC

| Parameter | Description | Default |
|-----------|-------------|---------|
| `rbac.create` | Create RBAC resources | `true` |
| `rbac.clusterRole` | Use ClusterRole | `true` |
| `rbac.rules` | RBAC rules | See values.yaml |

### Pod Disruption Budget

| Parameter | Description | Default |
|-----------|-------------|---------|
| `podDisruptionBudget.enabled` | Enable PDB | `false` |
| `podDisruptionBudget.minAvailable` | Minimum available pods | `1` |

### Network Policy

| Parameter | Description | Default |
|-----------|-------------|---------|
| `networkPolicy.enabled` | Enable network policy | `false` |
| `networkPolicy.policyTypes` | Policy types | `["Ingress", "Egress"]` |
| `networkPolicy.ingress` | Ingress rules | `[]` |
| `networkPolicy.egress` | Egress rules | See values.yaml |

### Environment Variables

| Parameter | Description | Default |
|-----------|-------------|---------|
| `env` | Environment variables | `[{"name": "LINKERD_NAMESPACE", "value": "linkerd"}]` |
| `envFrom` | Environment from ConfigMap/Secret | `[]` |

## Security Features

This Helm chart implements modern Kubernetes security best practices:

### Pod Security Standards (Restricted)

- **Non-root user**: Runs as UID 65532 (non-root)
- **Read-only root filesystem**: Prevents tampering with container filesystem
- **No privilege escalation**: Prevents gaining more privileges
- **Dropped capabilities**: All Linux capabilities are dropped
- **Seccomp profile**: Uses RuntimeDefault seccomp profile

### Additional Security

- **Service account token automounting**: Configurable
- **Network policies**: Optional ingress/egress restrictions
- **Pod disruption budgets**: Ensures availability during cluster operations
- **Resource limits**: Prevents resource exhaustion

## MCP Tools Provided

The server exposes the following MCP tools:

1. **check_mesh_health**: Checks the health status of the Linkerd control plane
   - Parameters: `namespace` (optional, defaults to "linkerd")
   - Returns: Status of all control plane components

2. **analyze_connectivity**: Analyzes Linkerd policies for service-to-service connectivity
   - Parameters: `source_namespace`, `source_service`, `target_namespace` (optional), `target_service`
   - Returns: Analysis of whether connectivity is allowed and applicable policies

3. **list_meshed_services**: Lists all services with Linkerd proxy injected
   - Parameters: `namespace` (optional)
   - Returns: All services in the mesh with their pods

4. **get_allowed_targets**: Find all services that a given source can communicate with
   - Parameters: `source_namespace`, `source_service`
   - Returns: List of all targets the source is authorized to access based on AuthorizationPolicies

5. **get_allowed_sources**: Find all services that can communicate with a given target
   - Parameters: `target_namespace`, `target_service`
   - Returns: List of all sources authorized to access the target based on AuthorizationPolicies

## Examples

### Install with custom values

```bash
helm install linkerd-mcp ./helm/linkerd-mcp -n linkerd \
  --set replicaCount=2 \
  --set resources.limits.memory=512Mi \
  --set autoscaling.enabled=true
```

### Install with network policies enabled

```bash
helm install linkerd-mcp ./helm/linkerd-mcp -n linkerd \
  --set networkPolicy.enabled=true
```

### Install with custom image

```bash
helm install linkerd-mcp ./helm/linkerd-mcp -n linkerd \
  --set image.repository=myregistry.io/linkerd-mcp \
  --set image.tag=v1.0.0 \
  --set image.pullPolicy=Always
```

### Enable Pod Disruption Budget for high availability

```bash
helm install linkerd-mcp ./helm/linkerd-mcp -n linkerd \
  --set replicaCount=3 \
  --set podDisruptionBudget.enabled=true \
  --set podDisruptionBudget.minAvailable=2
```

## Upgrading

To upgrade the release:

```bash
helm upgrade linkerd-mcp ./helm/linkerd-mcp -n linkerd
```

## Testing the Deployment

After installation, verify the deployment:

```bash
# Check pod status
kubectl get pods -n linkerd -l app.kubernetes.io/name=linkerd-mcp

# Check service
kubectl get svc -n linkerd -l app.kubernetes.io/name=linkerd-mcp

# View logs
kubectl logs -n linkerd -l app.kubernetes.io/name=linkerd-mcp

# Port forward to test locally
kubectl port-forward -n linkerd svc/linkerd-mcp 8080:8080
```

## Compliance

This chart is designed to comply with:

- **Kubernetes Pod Security Standards**: Restricted profile
- **CIS Kubernetes Benchmark**: Security controls
- **NIST 800-190**: Container security guidelines

## License

See the main project LICENSE file.
