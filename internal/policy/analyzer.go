package policy

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// Analyzer provides Linkerd policy analysis functionality
type Analyzer struct {
	clientset     kubernetes.Interface
	dynamicClient dynamic.Interface
}

// NewAnalyzer creates a new policy analyzer
func NewAnalyzer(clientset kubernetes.Interface, dynamicClient dynamic.Interface) *Analyzer {
	return &Analyzer{
		clientset:     clientset,
		dynamicClient: dynamicClient,
	}
}

// AnalyzeConnectivity analyzes connectivity policies between source and target services
func (a *Analyzer) AnalyzeConnectivity(ctx context.Context, sourceNamespace, sourceService, targetNamespace, targetService string) (*mcp.CallToolResult, error) {
	if targetNamespace == "" {
		targetNamespace = sourceNamespace
	}

	// Query Linkerd policies (ServerAuthorizations, AuthorizationPolicies)
	// This would integrate with Linkerd's policy CRDs
	analysis := map[string]interface{}{
		"source": map[string]string{
			"namespace": sourceNamespace,
			"service":   sourceService,
		},
		"target": map[string]string{
			"namespace": targetNamespace,
			"service":   targetService,
		},
		"allowed":      true,
		"policies":     []string{},
		"explanation":  "Policy analysis implementation pending - requires Linkerd CRD integration",
	}

	result, _ := json.MarshalIndent(analysis, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

// GetAllowedTargets finds all services that a given source service can communicate with
func (a *Analyzer) GetAllowedTargets(ctx context.Context, sourceNamespace, sourceService string) (*mcp.CallToolResult, error) {
	serviceAccount, err := a.getServiceAccountForService(ctx, sourceNamespace, sourceService)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	allowedTargets, err := a.findAllowedTargets(ctx, sourceNamespace, serviceAccount)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result := map[string]interface{}{
		"source": map[string]string{
			"namespace":      sourceNamespace,
			"service":        sourceService,
			"serviceAccount": serviceAccount,
		},
		"allowedTargets": allowedTargets,
		"totalTargets":   len(allowedTargets),
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(resultJSON)), nil
}

// GetAllowedSources finds all services that can communicate with a given target service
func (a *Analyzer) GetAllowedSources(ctx context.Context, targetNamespace, targetService string) (*mcp.CallToolResult, error) {
	matchingServers, err := a.findServersForService(ctx, targetNamespace, targetService)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(matchingServers) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No Linkerd Servers found for service %s in namespace %s", targetService, targetNamespace)), nil
	}

	allowedSources, err := a.findAllowedSources(ctx, targetNamespace, matchingServers)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result := map[string]interface{}{
		"target": map[string]string{
			"namespace": targetNamespace,
			"service":   targetService,
		},
		"matchingServers": matchingServers,
		"allowedSources":  allowedSources,
		"totalSources":    len(allowedSources),
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(resultJSON)), nil
}
