#!/bin/bash

set -e

CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

echo -e "${CYAN}=== Deploying Linkerd MCP Demo Application ===${NC}\n"

# 1. Create namespace
echo -e "${CYAN}Creating namespace...${NC}"
kubectl apply -f demo/namespace.yaml
echo -e "${GREEN}✓ Namespace created${NC}\n"

# 2. Deploy services
echo -e "${CYAN}Deploying services...${NC}"
kubectl apply -f demo/deployments.yaml
echo -e "${GREEN}✓ Services deployed${NC}\n"

# 3. Wait for pods to be ready
echo -e "${CYAN}Waiting for pods to be ready (this may take a minute)...${NC}"
kubectl wait --for=condition=ready pod -l app -n demo-app --timeout=180s || {
    echo -e "${YELLOW}Warning: Some pods may not be ready yet${NC}"
    echo -e "${YELLOW}You can check status with: kubectl get pods -n demo-app${NC}\n"
}
echo -e "${GREEN}✓ Pods are ready${NC}\n"

# 4. Create Linkerd Servers
echo -e "${CYAN}Creating Linkerd Server resources...${NC}"
kubectl apply -f demo/servers.yaml
echo -e "${GREEN}✓ Servers created${NC}\n"

# 5. Create Authorization Policies
echo -e "${CYAN}Creating Authorization Policies...${NC}"
kubectl apply -f demo/auth-policies.yaml
echo -e "${GREEN}✓ Authorization policies created${NC}\n"

# Display summary
echo -e "${GREEN}=== Deployment Complete! ===${NC}\n"

echo -e "${CYAN}Deployed Services:${NC}"
kubectl get pods -n demo-app

echo -e "\n${CYAN}Linkerd Servers:${NC}"
kubectl get servers -n demo-app

echo -e "\n${CYAN}Authorization Policies:${NC}"
kubectl get authorizationpolicies -n demo-app

echo -e "\n${GREEN}Demo application is ready!${NC}"
echo -e "\nNext steps:"
echo -e "  ${CYAN}1.${NC} Run mesh health check:  ${YELLOW}make example-mesh-health${NC}"
echo -e "  ${CYAN}2.${NC} Test connectivity:      ${YELLOW}go run examples/test-connectivity.go${NC}"
echo -e "  ${CYAN}3.${NC} View all services:      ${YELLOW}kubectl get all -n demo-app${NC}"
echo -e "  ${CYAN}4.${NC} Check policies:         ${YELLOW}kubectl get servers,authorizationpolicies -n demo-app${NC}"
