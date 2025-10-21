# Linkerd MCP Server Makefile

# Variables
APP_NAME := linkerd-mcp
DOCKER_IMAGE := linkerd-mcp
DOCKER_TAG := latest
KIND_CLUSTER := linkerd-mcp
LINKERD_VERSION := edge-25.4.4
HELM_CHART := ./helm/linkerd-mcp
HELM_NAMESPACE := linkerd-mcp
KUBECONFIG := $(HOME)/.kube/config

# Go variables
GO := go
GOFLAGS := -v
LDFLAGS := -s -w
BUILD_DIR := .
COVERAGE_FILE := coverage.out

# Helm variables
HELM_RELEASE := linkerd-mcp

# Colors for output
CYAN := \033[0;36m
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m # No Color

.PHONY: help
help: ## Show this help message
	@echo "$(CYAN)Linkerd MCP Server - Available Commands$(NC)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-25s$(NC) %s\n", $$1, $$2}'
	@echo ""

##@ Development

.PHONY: build
build: ## Build the Go binary
	@echo "$(CYAN)Building $(APP_NAME)...$(NC)"
	$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(APP_NAME) .
	@echo "$(GREEN)✓ Build complete: $(APP_NAME)$(NC)"

.PHONY: build-debug
build-debug: ## Build the Go binary with debug symbols
	@echo "$(CYAN)Building $(APP_NAME) with debug symbols...$(NC)"
	$(GO) build $(GOFLAGS) -o $(APP_NAME) .
	@echo "$(GREEN)✓ Debug build complete: $(APP_NAME)$(NC)"

.PHONY: run
run: ## Run the application locally (requires kubeconfig)
	@echo "$(CYAN)Running $(APP_NAME)...$(NC)"
	$(GO) run main.go

.PHONY: clean
clean: ## Clean build artifacts
	@echo "$(CYAN)Cleaning build artifacts...$(NC)"
	rm -f $(APP_NAME)
	rm -f $(COVERAGE_FILE)
	rm -f coverage.html
	@echo "$(GREEN)✓ Clean complete$(NC)"

##@ Testing

.PHONY: test
test: ## Run all tests
	@echo "$(CYAN)Running tests...$(NC)"
	$(GO) test ./internal/... -v

.PHONY: test-coverage
test-coverage: ## Run tests with coverage report
	@echo "$(CYAN)Running tests with coverage...$(NC)"
	$(GO) test -v -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./internal/...
	$(GO) tool cover -func=$(COVERAGE_FILE)
	@echo "$(GREEN)✓ Coverage report generated: $(COVERAGE_FILE)$(NC)"

.PHONY: test-coverage-html
test-coverage-html: test-coverage ## Generate HTML coverage report
	@echo "$(CYAN)Generating HTML coverage report...$(NC)"
	$(GO) tool cover -html=$(COVERAGE_FILE) -o coverage.html
	@echo "$(GREEN)✓ HTML coverage report: coverage.html$(NC)"

.PHONY: test-race
test-race: ## Run tests with race detector
	@echo "$(CYAN)Running tests with race detector...$(NC)"
	$(GO) test -v -race ./...

.PHONY: test-all
test-all: ## Run all tests including main package
	@echo "$(CYAN)Running all tests...$(NC)"
	$(GO) test -v -race -coverprofile=$(COVERAGE_FILE) ./...

.PHONY: lint
lint: ## Run golangci-lint
	@echo "$(CYAN)Running linter...$(NC)"
	golangci-lint run --timeout=5m
	@echo "$(GREEN)✓ Linting complete$(NC)"

##@ Docker

.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "$(CYAN)Building Docker image $(DOCKER_IMAGE):$(DOCKER_TAG)...$(NC)"
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@echo "$(GREEN)✓ Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)$(NC)"

.PHONY: docker-build-local
docker-build-local: ## Build Docker image with 'local' tag
	@echo "$(CYAN)Building Docker image $(DOCKER_IMAGE):local...$(NC)"
	docker build -t $(DOCKER_IMAGE):local .
	@echo "$(GREEN)✓ Docker image built: $(DOCKER_IMAGE):local$(NC)"

.PHONY: docker-run
docker-run: ## Run Docker container locally (requires kubeconfig)
	@echo "$(CYAN)Running Docker container...$(NC)"
	docker run --rm -v $(KUBECONFIG):/root/.kube/config -p 8080:8080 $(DOCKER_IMAGE):$(DOCKER_TAG)

##@ Kind Cluster

.PHONY: kind-create
kind-create: ## Create kind cluster
	@echo "$(CYAN)Creating kind cluster: $(KIND_CLUSTER)...$(NC)"
	kind create cluster --name $(KIND_CLUSTER)
	@echo "$(GREEN)✓ Kind cluster created: $(KIND_CLUSTER)$(NC)"

.PHONY: kind-delete
kind-delete: ## Delete kind cluster
	@echo "$(CYAN)Deleting kind cluster: $(KIND_CLUSTER)...$(NC)"
	kind delete cluster --name $(KIND_CLUSTER)
	@echo "$(GREEN)✓ Kind cluster deleted$(NC)"

.PHONY: kind-load
kind-load: docker-build-local ## Load Docker image into kind cluster
	@echo "$(CYAN)Loading image into kind cluster...$(NC)"
	kind load docker-image $(DOCKER_IMAGE):local --name $(KIND_CLUSTER)
	@echo "$(GREEN)✓ Image loaded into kind cluster$(NC)"

.PHONY: kind-status
kind-status: ## Check kind cluster status
	@echo "$(CYAN)Kind cluster status:$(NC)"
	@kind get clusters
	@echo ""
	@/usr/local/bin/kubectl cluster-info --context kind-$(KIND_CLUSTER)

##@ Linkerd

.PHONY: linkerd-install-cli
linkerd-install-cli: ## Install Linkerd CLI (edge-25.4.4)
	@echo "$(CYAN)Installing Linkerd CLI $(LINKERD_VERSION)...$(NC)"
	curl -sL https://run.linkerd.io/install-edge | sh
	@echo "$(GREEN)✓ Linkerd CLI installed$(NC)"
	@echo "$(YELLOW)Add to PATH: export PATH=\$$PATH:$(HOME)/.linkerd2/bin$(NC)"

.PHONY: linkerd-check-pre
linkerd-check-pre: ## Run Linkerd pre-installation checks
	@echo "$(CYAN)Running Linkerd pre-installation checks...$(NC)"
	linkerd check --pre

.PHONY: linkerd-install-crds
linkerd-install-crds: ## Install Linkerd CRDs
	@echo "$(CYAN)Installing Linkerd CRDs...$(NC)"
	linkerd install --crds | /usr/local/bin/kubectl apply -f -
	@echo "$(GREEN)✓ Linkerd CRDs installed$(NC)"

.PHONY: linkerd-install-gateway-api
linkerd-install-gateway-api: ## Install Gateway API CRDs
	@echo "$(CYAN)Installing Gateway API CRDs...$(NC)"
	/usr/local/bin/kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.2.1/standard-install.yaml
	@echo "$(GREEN)✓ Gateway API CRDs installed$(NC)"

.PHONY: linkerd-install
linkerd-install: linkerd-install-gateway-api linkerd-install-crds ## Install Linkerd control plane
	@echo "$(CYAN)Installing Linkerd control plane...$(NC)"
	linkerd install | /usr/local/bin/kubectl apply -f -
	@echo "$(CYAN)Waiting for Linkerd to be ready...$(NC)"
	/usr/local/bin/kubectl wait --for=condition=available --timeout=300s deployment --all -n linkerd
	@echo "$(GREEN)✓ Linkerd control plane installed$(NC)"

.PHONY: linkerd-check
linkerd-check: ## Run Linkerd health checks
	@echo "$(CYAN)Running Linkerd health checks...$(NC)"
	linkerd check

.PHONY: linkerd-viz-install
linkerd-viz-install: ## Install Linkerd Viz extension
	@echo "$(CYAN)Installing Linkerd Viz...$(NC)"
	linkerd viz install | /usr/local/bin/kubectl apply -f -
	@echo "$(GREEN)✓ Linkerd Viz installed$(NC)"

.PHONY: linkerd-viz-dashboard
linkerd-viz-dashboard: ## Open Linkerd Viz dashboard
	@echo "$(CYAN)Opening Linkerd Viz dashboard...$(NC)"
	linkerd viz dashboard

.PHONY: linkerd-uninstall
linkerd-uninstall: ## Uninstall Linkerd
	@echo "$(CYAN)Uninstalling Linkerd...$(NC)"
	linkerd viz uninstall | /usr/local/bin/kubectl delete -f - || true
	linkerd uninstall | /usr/local/bin/kubectl delete -f - || true
	@echo "$(GREEN)✓ Linkerd uninstalled$(NC)"

##@ Helm

.PHONY: helm-lint
helm-lint: ## Lint Helm chart
	@echo "$(CYAN)Linting Helm chart...$(NC)"
	helm lint $(HELM_CHART)
	@echo "$(GREEN)✓ Helm chart linted$(NC)"

.PHONY: helm-template
helm-template: ## Template Helm chart
	@echo "$(CYAN)Templating Helm chart...$(NC)"
	helm template $(HELM_RELEASE) $(HELM_CHART) --namespace $(HELM_NAMESPACE)

.PHONY: helm-template-prod
helm-template-prod: ## Template Helm chart with production values
	@echo "$(CYAN)Templating Helm chart with production values...$(NC)"
	helm template $(HELM_RELEASE) $(HELM_CHART) --namespace $(HELM_NAMESPACE) --values $(HELM_CHART)/values-production.yaml

.PHONY: helm-install
helm-install: kind-load ## Install Helm chart to kind cluster
	@echo "$(CYAN)Creating namespace $(HELM_NAMESPACE)...$(NC)"
	/usr/local/bin/kubectl create namespace $(HELM_NAMESPACE) || true
	@echo "$(CYAN)Installing Helm chart...$(NC)"
	helm install $(HELM_RELEASE) $(HELM_CHART) \
		--namespace $(HELM_NAMESPACE) \
		--set image.tag=local \
		--set image.pullPolicy=Never
	@echo "$(GREEN)✓ Helm chart installed$(NC)"

.PHONY: helm-upgrade
helm-upgrade: kind-load ## Upgrade Helm release
	@echo "$(CYAN)Upgrading Helm release...$(NC)"
	helm upgrade $(HELM_RELEASE) $(HELM_CHART) \
		--namespace $(HELM_NAMESPACE) \
		--set image.tag=local \
		--set image.pullPolicy=Never
	@echo "$(GREEN)✓ Helm chart upgraded$(NC)"

.PHONY: helm-uninstall
helm-uninstall: ## Uninstall Helm release
	@echo "$(CYAN)Uninstalling Helm release...$(NC)"
	helm uninstall $(HELM_RELEASE) --namespace $(HELM_NAMESPACE)
	@echo "$(GREEN)✓ Helm release uninstalled$(NC)"

.PHONY: helm-reinstall
helm-reinstall: helm-uninstall helm-install ## Reinstall Helm release

##@ Kubernetes

.PHONY: k8s-status
k8s-status: ## Check Kubernetes deployment status
	@echo "$(CYAN)Checking deployment status...$(NC)"
	/usr/local/bin/kubectl get pods -n $(HELM_NAMESPACE)
	@echo ""
	/usr/local/bin/kubectl get svc -n $(HELM_NAMESPACE)

.PHONY: k8s-logs
k8s-logs: ## View pod logs
	@echo "$(CYAN)Fetching logs...$(NC)"
	/usr/local/bin/kubectl logs -n $(HELM_NAMESPACE) -l app.kubernetes.io/name=linkerd-mcp -c linkerd-mcp --tail=100 -f

.PHONY: k8s-logs-all
k8s-logs-all: ## View all pod logs (including init containers)
	@echo "$(CYAN)Fetching all logs...$(NC)"
	/usr/local/bin/kubectl logs -n $(HELM_NAMESPACE) -l app.kubernetes.io/name=linkerd-mcp --all-containers --tail=100

.PHONY: k8s-describe
k8s-describe: ## Describe pods
	@echo "$(CYAN)Describing pods...$(NC)"
	/usr/local/bin/kubectl describe pods -n $(HELM_NAMESPACE) -l app.kubernetes.io/name=linkerd-mcp

.PHONY: k8s-exec
k8s-exec: ## Execute shell in pod
	@echo "$(CYAN)Opening shell in pod...$(NC)"
	/usr/local/bin/kubectl exec -it -n $(HELM_NAMESPACE) $$(kubectl get pod -n $(HELM_NAMESPACE) -l app.kubernetes.io/name=linkerd-mcp -o jsonpath='{.items[0].metadata.name}') -c linkerd-mcp -- /bin/sh

.PHONY: k8s-port-forward
k8s-port-forward: ## Port-forward to service (8080:8080)
	@echo "$(CYAN)Port-forwarding to service...$(NC)"
	@echo "$(YELLOW)Access endpoints at:$(NC)"
	@echo "  Health:  http://localhost:8080/health"
	@echo "  Ready:   http://localhost:8080/ready"
	@echo "  MCP SSE: http://localhost:8080/sse"
	@echo ""
	/usr/local/bin/kubectl port-forward -n $(HELM_NAMESPACE) svc/linkerd-mcp 8080:8080

.PHONY: k8s-restart
k8s-restart: ## Restart pods
	@echo "$(CYAN)Restarting pods...$(NC)"
	/usr/local/bin/kubectl rollout restart deployment -n $(HELM_NAMESPACE) linkerd-mcp
	@echo "$(GREEN)✓ Deployment restarted$(NC)"

.PHONY: k8s-events
k8s-events: ## Show recent events
	@echo "$(CYAN)Recent events:$(NC)"
	/usr/local/bin/kubectl get events -n $(HELM_NAMESPACE) --sort-by='.lastTimestamp'

##@ Full Stack

.PHONY: setup-cluster
setup-cluster: kind-create linkerd-install ## Create kind cluster and install Linkerd
	@echo "$(GREEN)✓ Cluster setup complete$(NC)"

.PHONY: deploy-full
deploy-full: docker-build-local kind-load helm-install ## Build, load, and deploy everything
	@echo "$(GREEN)✓ Full deployment complete$(NC)"
	@echo ""
	@echo "$(CYAN)Checking deployment status...$(NC)"
	@/usr/local/bin/kubectl get pods -n $(HELM_NAMESPACE)

.PHONY: redeploy
redeploy: docker-build-local kind-load helm-upgrade k8s-restart ## Rebuild and redeploy
	@echo "$(GREEN)✓ Redeployment complete$(NC)"
	@echo ""
	@echo "$(CYAN)Checking deployment status...$(NC)"
	@/usr/local/bin/kubectl get pods -n $(HELM_NAMESPACE)

.PHONY: teardown
teardown: helm-uninstall linkerd-uninstall kind-delete ## Tear down everything
	@echo "$(GREEN)✓ Teardown complete$(NC)"

.PHONY: reset
reset: teardown setup-cluster deploy-full ## Reset everything (full teardown and setup)
	@echo "$(GREEN)✓ Reset complete$(NC)"

##@ CI/CD

.PHONY: ci-test
ci-test: lint test-coverage ## Run CI tests locally
	@echo "$(GREEN)✓ CI tests complete$(NC)"

.PHONY: ci-build
ci-build: build docker-build ## Run CI build locally
	@echo "$(GREEN)✓ CI build complete$(NC)"

.PHONY: ci-all
ci-all: ci-test ci-build ## Run all CI checks locally
	@echo "$(GREEN)✓ All CI checks complete$(NC)"

##@ Utilities

.PHONY: deps
deps: ## Download Go dependencies
	@echo "$(CYAN)Downloading dependencies...$(NC)"
	$(GO) mod download
	@echo "$(GREEN)✓ Dependencies downloaded$(NC)"

.PHONY: tidy
tidy: ## Tidy Go modules
	@echo "$(CYAN)Tidying Go modules...$(NC)"
	$(GO) mod tidy
	@echo "$(GREEN)✓ Modules tidied$(NC)"

.PHONY: verify
verify: ## Verify Go modules
	@echo "$(CYAN)Verifying Go modules...$(NC)"
	$(GO) mod verify
	@echo "$(GREEN)✓ Modules verified$(NC)"

.PHONY: fmt
fmt: ## Format Go code
	@echo "$(CYAN)Formatting code...$(NC)"
	$(GO) fmt ./...
	@echo "$(GREEN)✓ Code formatted$(NC)"

.PHONY: vet
vet: ## Run go vet
	@echo "$(CYAN)Running go vet...$(NC)"
	$(GO) vet ./...
	@echo "$(GREEN)✓ Vet complete$(NC)"

.PHONY: check-tools
check-tools: ## Check if required tools are installed
	@echo "$(CYAN)Checking required tools...$(NC)"
	@command -v go >/dev/null 2>&1 || { echo "$(RED)✗ Go is not installed$(NC)"; exit 1; }
	@echo "$(GREEN)✓ Go $$(go version)$(NC)"
	@command -v docker >/dev/null 2>&1 || { echo "$(RED)✗ Docker is not installed$(NC)"; exit 1; }
	@echo "$(GREEN)✓ Docker $$(docker --version)$(NC)"
	@command -v kind >/dev/null 2>&1 || { echo "$(RED)✗ kind is not installed$(NC)"; exit 1; }
	@echo "$(GREEN)✓ kind $$(kind version)$(NC)"
	@command -v helm >/dev/null 2>&1 || { echo "$(RED)✗ Helm is not installed$(NC)"; exit 1; }
	@echo "$(GREEN)✓ Helm $$(helm version --short)$(NC)"
	@command -v kubectl >/dev/null 2>&1 || { echo "$(RED)✗ kubectl is not installed$(NC)"; exit 1; }
	@echo "$(GREEN)✓ kubectl $$(kubectl version --client --short 2>/dev/null || echo 'installed')$(NC)"
	@command -v linkerd >/dev/null 2>&1 || { echo "$(YELLOW)⚠ Linkerd CLI is not installed (run: make linkerd-install-cli)$(NC)"; }
	@command -v linkerd >/dev/null 2>&1 && echo "$(GREEN)✓ Linkerd $$(linkerd version --client --short)$(NC)" || true
	@command -v golangci-lint >/dev/null 2>&1 || { echo "$(YELLOW)⚠ golangci-lint is not installed$(NC)"; }
	@command -v golangci-lint >/dev/null 2>&1 && echo "$(GREEN)✓ golangci-lint $$(golangci-lint version --format short 2>/dev/null || echo 'installed')$(NC)" || true

.PHONY: all
all: clean deps build test lint docker-build ## Run all build and test steps
	@echo "$(GREEN)✓ All tasks complete$(NC)"

.DEFAULT_GOAL := help
