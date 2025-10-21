# Linkerd MCP Demo Application

This demo application demonstrates the Linkerd MCP server's capabilities for analyzing service mesh connectivity and authorization policies.

## Architecture

The demo consists of a microservices-based e-commerce application with the following services:

```
┌──────────┐
│ Frontend │ (Web UI)
└────┬─────┘
     │ ✓ Allowed
     ▼
┌─────────────┐
│ API Gateway │
└─────┬───────┘
      │ ✓ Allowed to: catalog, cart, checkout
      ▼
┌─────────┬──────────┬──────────┐
│ Catalog │   Cart   │ Checkout │
└────┬────┴────┬─────┴────┬─────┘
     │         │          │ ✓ Allowed to: cart, payment
     │         │          ▼
     │         │     ┌─────────┐
     │         │     │ Payment │ (Only accessible by checkout)
     │         │     └────┬────┘
     │         │          │
     └─────────┴──────────┴─────┐
                                 ▼
                          ┌──────────┐
                          │ Database │ (Backend data store)
                          └──────────┘
```

## Authorization Policy Matrix

| Source Service | Can Access                          |
|----------------|-------------------------------------|
| frontend       | api-gateway                         |
| api-gateway    | catalog, cart, checkout             |
| cart           | catalog                             |
| checkout       | cart, payment                       |
| catalog        | database                            |
| cart           | database                            |
| checkout       | database                            |
| payment        | database                            |

## Key Features

1. **Strict mTLS Authentication**: All communication uses Linkerd's mutual TLS
2. **Fine-grained Authorization**: Each service has specific authorization policies
3. **Least Privilege**: Services can only access what they need
4. **Defense in Depth**: Payment service is only accessible by checkout

## Files

- `namespace.yaml` - Namespace with Linkerd injection enabled
- `deployments.yaml` - Service deployments and ServiceAccounts
- `servers.yaml` - Linkerd Server resources (policy targets)
- `auth-policies.yaml` - Linkerd AuthorizationPolicies and MeshTLSAuthentications
- `deploy.sh` - Deployment script
- `cleanup.sh` - Cleanup script

## Deployment

### Quick Start

```bash
# Deploy the demo application
./demo/deploy.sh

# Or use the Makefile
make demo-deploy
```

### Manual Deployment

```bash
# 1. Create namespace
kubectl apply -f demo/namespace.yaml

# 2. Deploy services
kubectl apply -f demo/deployments.yaml

# 3. Wait for pods to be ready
kubectl wait --for=condition=ready pod -l app -n demo-app --timeout=120s

# 4. Create Linkerd Servers
kubectl apply -f demo/servers.yaml

# 5. Create Authorization Policies
kubectl apply -f demo/auth-policies.yaml
```

### Verify Deployment

```bash
# Check all pods are running
kubectl get pods -n demo-app

# Check services
kubectl get svc -n demo-app

# Check Linkerd Servers
kubectl get servers -n demo-app

# Check Authorization Policies
kubectl get authorizationpolicies -n demo-app

# Check MeshTLS Authentications
kubectl get meshtlsauthentications -n demo-app
```

## Testing MCP Server Capabilities

Once deployed, you can use the MCP server to analyze the service mesh:

### 1. List Meshed Services

```bash
go run examples/test-list-services.go
```

Should show all 6 services with Linkerd proxy injected.

### 2. Analyze Specific Connectivity

```bash
# Can frontend reach api-gateway?
go run examples/test-connectivity.go frontend api-gateway

# Can api-gateway reach payment? (Should be DENIED)
go run examples/test-connectivity.go api-gateway payment

# Can checkout reach payment? (Should be ALLOWED)
go run examples/test-connectivity.go checkout payment
```

### 3. Get Allowed Targets

```bash
# What can api-gateway access?
go run examples/test-allowed-targets.go api-gateway

# What can checkout access?
go run examples/test-allowed-targets.go checkout
```

### 4. Get Allowed Sources

```bash
# Who can access payment?
go run examples/test-allowed-sources.go payment

# Who can access database?
go run examples/test-allowed-sources.go database
```

## Expected Test Results

### Frontend → API Gateway
- **Status**: ✅ ALLOWED
- **Reason**: MeshTLS authentication policy allows frontend identity

### API Gateway → Catalog
- **Status**: ✅ ALLOWED
- **Reason**: MeshTLS authentication policy allows api-gateway identity

### API Gateway → Payment
- **Status**: ❌ DENIED
- **Reason**: Payment only allows checkout identity

### Checkout → Payment
- **Status**: ✅ ALLOWED
- **Reason**: MeshTLS authentication policy allows checkout identity

### Catalog → Database
- **Status**: ✅ ALLOWED
- **Reason**: Database allows backend service identities

## Cleanup

```bash
# Delete all demo resources
./demo/cleanup.sh

# Or use the Makefile
make demo-cleanup
```

## Makefile Commands

```bash
make demo-deploy       # Deploy demo application
make demo-status       # Check demo application status
make demo-cleanup      # Remove demo application
make demo-test-all     # Run all MCP connectivity tests
```

## Notes

- All services use `hashicorp/http-echo` for simplicity
- The demo focuses on policy demonstration, not functional application logic
- Authorization policies are enforced at the Linkerd proxy level
- The MCP server queries Linkerd CRDs to analyze connectivity
