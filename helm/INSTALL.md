# Linkerd MCP Helm Chart Installation Guide

## Quick Start

### Prerequisites

1. Kubernetes cluster (1.29+) with kubectl access
2. Helm 3.0+ installed
3. Linkerd service mesh installed

### Basic Installation

Install the chart in the `linkerd` namespace:

```bash
helm install linkerd-mcp ./helm/linkerd-mcp \
  --namespace linkerd \
  --create-namespace
```

### Verify Installation

```bash
# Check the deployment
kubectl get all -n linkerd -l app.kubernetes.io/name=linkerd-mcp

# View logs
kubectl logs -n linkerd -l app.kubernetes.io/name=linkerd-mcp -f
```

## Production Deployment

For production environments, use the production values file:

```bash
helm install linkerd-mcp ./helm/linkerd-mcp \
  --namespace linkerd \
  --create-namespace \
  --values ./helm/linkerd-mcp/values-production.yaml
```

The production configuration includes:

- **3 replicas** with pod anti-affinity
- **Horizontal Pod Autoscaling** (3-10 replicas)
- **Pod Disruption Budget** (minimum 2 available)
- **Network Policies** enabled
- **Topology spread constraints** for HA
- **Higher resource allocations** (200m CPU, 256Mi RAM requests)

## Configuration Options

### Custom Image

```bash
helm install linkerd-mcp ./helm/linkerd-mcp \
  --namespace linkerd \
  --set image.repository=myregistry.io/linkerd-mcp \
  --set image.tag=v1.0.0
```

### Resource Limits

```bash
helm install linkerd-mcp ./helm/linkerd-mcp \
  --namespace linkerd \
  --set resources.requests.cpu=200m \
  --set resources.requests.memory=256Mi \
  --set resources.limits.cpu=1000m \
  --set resources.limits.memory=512Mi
```

### Enable Autoscaling

```bash
helm install linkerd-mcp ./helm/linkerd-mcp \
  --namespace linkerd \
  --set autoscaling.enabled=true \
  --set autoscaling.minReplicas=3 \
  --set autoscaling.maxReplicas=10
```

### Enable Network Policies

```bash
helm install linkerd-mcp ./helm/linkerd-mcp \
  --namespace linkerd \
  --set networkPolicy.enabled=true
```

## Upgrading

### Upgrade with Default Values

```bash
helm upgrade linkerd-mcp ./helm/linkerd-mcp \
  --namespace linkerd
```

### Upgrade with New Values

```bash
helm upgrade linkerd-mcp ./helm/linkerd-mcp \
  --namespace linkerd \
  --values ./helm/linkerd-mcp/values-production.yaml \
  --reuse-values
```

### Check Upgrade Status

```bash
helm status linkerd-mcp -n linkerd
helm history linkerd-mcp -n linkerd
```

## Rollback

If something goes wrong, rollback to the previous version:

```bash
# Rollback to previous version
helm rollback linkerd-mcp -n linkerd

# Rollback to specific revision
helm rollback linkerd-mcp 2 -n linkerd
```

## Uninstalling

```bash
helm uninstall linkerd-mcp -n linkerd
```

## Troubleshooting

### Check Pod Status

```bash
kubectl get pods -n linkerd -l app.kubernetes.io/name=linkerd-mcp
kubectl describe pod -n linkerd -l app.kubernetes.io/name=linkerd-mcp
```

### View Logs

```bash
kubectl logs -n linkerd -l app.kubernetes.io/name=linkerd-mcp --all-containers=true
```

### Check RBAC Permissions

```bash
kubectl auth can-i --as=system:serviceaccount:linkerd:linkerd-mcp get pods
kubectl auth can-i --as=system:serviceaccount:linkerd:linkerd-mcp list pods --all-namespaces
```

### Debug Security Context Issues

```bash
# Check security context
kubectl get pod -n linkerd -l app.kubernetes.io/name=linkerd-mcp \
  -o jsonpath='{.items[0].spec.securityContext}' | jq

# Check container security context
kubectl get pod -n linkerd -l app.kubernetes.io/name=linkerd-mcp \
  -o jsonpath='{.items[0].spec.containers[0].securityContext}' | jq
```

### Test Network Policies

If network policies are enabled and the pod cannot communicate:

```bash
# Temporarily disable network policy for testing
kubectl annotate networkpolicy linkerd-mcp -n linkerd \
  kubectl.kubernetes.io/last-applied-configuration-

# Check network policy
kubectl describe networkpolicy linkerd-mcp -n linkerd
```

### Validate Rendered Templates

Before installing, preview the rendered templates:

```bash
helm template linkerd-mcp ./helm/linkerd-mcp \
  --namespace linkerd \
  --values ./helm/linkerd-mcp/values.yaml
```

## Advanced Configurations

### Using with Private Registry

```bash
# Create image pull secret
kubectl create secret docker-registry regcred \
  --docker-server=myregistry.io \
  --docker-username=myuser \
  --docker-password=mypassword \
  --docker-email=myemail@example.com \
  -n linkerd

# Install with pull secret
helm install linkerd-mcp ./helm/linkerd-mcp \
  --namespace linkerd \
  --set image.repository=myregistry.io/linkerd-mcp \
  --set imagePullSecrets[0].name=regcred
```

### Custom Service Account

```bash
helm install linkerd-mcp ./helm/linkerd-mcp \
  --namespace linkerd \
  --set serviceAccount.create=false \
  --set serviceAccount.name=my-custom-sa
```

### Custom RBAC Rules

Create a custom values file with additional RBAC rules:

```yaml
# custom-rbac-values.yaml
rbac:
  create: true
  clusterRole: true
  rules:
    - apiGroups: [""]
      resources: ["pods", "services", "namespaces"]
      verbs: ["get", "list", "watch"]
    - apiGroups: ["policy.linkerd.io"]
      resources: ["servers", "serverauthorizations", "authorizationpolicies", "httproutes"]
      verbs: ["get", "list", "watch"]
    - apiGroups: ["apps"]
      resources: ["deployments", "replicasets"]
      verbs: ["get", "list"]
    # Add your custom rules here
    - apiGroups: ["custom.io"]
      resources: ["customresources"]
      verbs: ["get", "list"]
```

```bash
helm install linkerd-mcp ./helm/linkerd-mcp \
  --namespace linkerd \
  --values custom-rbac-values.yaml
```

## Security Hardening

The chart follows Kubernetes security best practices:

1. **Non-root user**: Runs as UID 65532
2. **Read-only root filesystem**: `/tmp` and `/root/.cache` are mounted as emptyDir
3. **No privilege escalation**: `allowPrivilegeEscalation: false`
4. **Dropped capabilities**: All Linux capabilities dropped
5. **Seccomp profile**: Uses RuntimeDefault
6. **Network policies**: Optional restriction of ingress/egress

### Pod Security Standards

This chart is compliant with the **Restricted** Pod Security Standard.

To enforce at namespace level:

```bash
kubectl label namespace linkerd \
  pod-security.kubernetes.io/enforce=restricted \
  pod-security.kubernetes.io/audit=restricted \
  pod-security.kubernetes.io/warn=restricted
```

## Performance Tuning

### For High Load

```yaml
# high-performance-values.yaml
replicaCount: 5

resources:
  requests:
    cpu: 500m
    memory: 512Mi
  limits:
    cpu: 2000m
    memory: 1Gi

autoscaling:
  enabled: true
  minReplicas: 5
  maxReplicas: 20
  targetCPUUtilizationPercentage: 60
  targetMemoryUtilizationPercentage: 70
```

### For Low Resource Environments

```yaml
# low-resource-values.yaml
replicaCount: 1

resources:
  requests:
    cpu: 50m
    memory: 64Mi
  limits:
    cpu: 200m
    memory: 128Mi

autoscaling:
  enabled: false
```

## Monitoring

### Prometheus Integration

The chart includes a ServiceMonitor template for Prometheus Operator:

```bash
helm install linkerd-mcp ./helm/linkerd-mcp \
  --namespace linkerd \
  --set serviceMonitor.enabled=true \
  --set serviceMonitor.interval=30s
```

## Support

For issues and questions:
- GitHub Issues: https://github.com/christianhuening/linkerd-mcp/issues
- Documentation: See README.md in the chart directory
