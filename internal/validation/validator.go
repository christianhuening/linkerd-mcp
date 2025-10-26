package validation

import (
	"context"
	"encoding/json"

	"github.com/christianhuening/linkerd-mcp/internal/validation/validators"
	"github.com/mark3labs/mcp-go/mcp"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// ConfigValidator orchestrates validation of Linkerd configuration
type ConfigValidator struct {
	serverValidator     *validators.ServerValidator
	authPolicyValidator *validators.AuthPolicyValidator
	meshTLSValidator    *validators.MeshTLSValidator
	proxyValidator      *validators.ProxyValidator
}

// NewConfigValidator creates a new configuration validator
func NewConfigValidator(clientset kubernetes.Interface, dynamicClient dynamic.Interface) *ConfigValidator {
	return &ConfigValidator{
		serverValidator:     validators.NewServerValidator(clientset, dynamicClient),
		authPolicyValidator: validators.NewAuthPolicyValidator(dynamicClient),
		meshTLSValidator:    validators.NewMeshTLSValidator(clientset, dynamicClient),
		proxyValidator:      validators.NewProxyValidator(clientset),
	}
}

// ValidateConfig validates Linkerd configuration based on parameters
func (cv *ConfigValidator) ValidateConfig(ctx context.Context, namespace, resourceType, resourceName string, includeWarnings bool) (*mcp.CallToolResult, error) {
	report := validators.ClusterValidationReport{
		Results: []validators.ValidationResult{},
		Summary: validators.ValidationSummary{},
	}

	// Determine which validators to run
	switch resourceType {
	case "server":
		results := cv.serverValidator.ValidateAll(ctx, namespace)
		cv.addResultsToReport(&report, results, resourceName, includeWarnings)
	case "authpolicy", "authorizationpolicy":
		results := cv.authPolicyValidator.ValidateAll(ctx, namespace)
		cv.addResultsToReport(&report, results, resourceName, includeWarnings)
	case "meshtls", "meshtlsauthentication":
		results := cv.meshTLSValidator.ValidateAll(ctx, namespace)
		cv.addResultsToReport(&report, results, resourceName, includeWarnings)
	case "proxy", "namespace":
		// Validate proxy configuration on namespaces
		if namespace == "" {
			results := cv.proxyValidator.ValidateAllNamespaces(ctx)
			cv.addResultsToReport(&report, results, resourceName, includeWarnings)
		} else {
			// Validate specific namespace and its pods
			results := cv.proxyValidator.ValidateAllPodsInNamespace(ctx, namespace)
			cv.addResultsToReport(&report, results, resourceName, includeWarnings)
		}
	case "all", "":
		// Validate all resource types
		serverResults := cv.serverValidator.ValidateAll(ctx, namespace)
		cv.addResultsToReport(&report, serverResults, resourceName, includeWarnings)

		authPolicyResults := cv.authPolicyValidator.ValidateAll(ctx, namespace)
		cv.addResultsToReport(&report, authPolicyResults, resourceName, includeWarnings)

		meshTLSResults := cv.meshTLSValidator.ValidateAll(ctx, namespace)
		cv.addResultsToReport(&report, meshTLSResults, resourceName, includeWarnings)

		// Validate proxy configuration
		if namespace == "" {
			proxyResults := cv.proxyValidator.ValidateAllNamespaces(ctx)
			cv.addResultsToReport(&report, proxyResults, resourceName, includeWarnings)
		} else {
			proxyResults := cv.proxyValidator.ValidateAllPodsInNamespace(ctx, namespace)
			cv.addResultsToReport(&report, proxyResults, resourceName, includeWarnings)
		}
	default:
		return mcp.NewToolResultError("Invalid resource_type. Must be one of: server, authpolicy, meshtls, proxy, all"), nil
	}

	report.Finalize()

	// Convert to JSON
	resultJSON, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("Failed to serialize validation results"), nil
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

func (cv *ConfigValidator) addResultsToReport(report *validators.ClusterValidationReport, results []validators.ValidationResult, resourceName string, includeWarnings bool) {
	for _, result := range results {
		// Filter by resource name if specified
		if resourceName != "" && result.Name != resourceName {
			continue
		}

		// Filter issues if warnings should not be included
		if !includeWarnings {
			filteredIssues := []validators.Issue{}
			for _, issue := range result.Issues {
				if issue.Severity == validators.SeverityError {
					filteredIssues = append(filteredIssues, issue)
				}
			}
			result.Issues = filteredIssues
			result.Valid = len(filteredIssues) == 0
		}

		report.AddResult(result)
	}
}
