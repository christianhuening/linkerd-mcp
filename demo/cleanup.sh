#!/bin/bash

set -e

CYAN='\033[0;36m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

echo -e "${CYAN}=== Cleaning up Linkerd MCP Demo Application ===${NC}\n"

echo -e "${CYAN}Deleting demo-app namespace and all resources...${NC}"
kubectl delete namespace demo-app --timeout=60s || {
    echo "Warning: Namespace deletion timed out, but cleanup initiated"
}

echo -e "${GREEN}✓ Demo application cleaned up${NC}"
