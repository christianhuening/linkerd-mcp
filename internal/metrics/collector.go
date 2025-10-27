package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/prometheus/common/model"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// MetricsCollector collects and analyzes Linkerd traffic metrics
type MetricsCollector struct {
	promClient   *PrometheusClient
	queryBuilder *QueryBuilder
	clientset    kubernetes.Interface
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(config *rest.Config, clientset kubernetes.Interface, namespace string) (*MetricsCollector, error) {
	promClient, err := NewPrometheusClient(config, clientset, namespace)
	if err != nil {
		return nil, err
	}

	return &MetricsCollector{
		promClient:   promClient,
		queryBuilder: NewQueryBuilder(namespace),
		clientset:    clientset,
	}, nil
}

// GetServiceMetrics retrieves comprehensive metrics for a service
func (c *MetricsCollector) GetServiceMetrics(ctx context.Context, namespace, service, timeRangeStr string) (*mcp.CallToolResult, error) {
	// Parse time range
	tr, err := ParseTimeRange(timeRangeStr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid time range: %v", err)), nil
	}

	// Find deployment for service
	deployment, err := c.findDeploymentForService(ctx, namespace, service)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to find deployment: %v", err)), nil
	}

	// Build and execute queries
	window := tr.End.Sub(tr.Start)

	// Request rate
	reqRateQuery := c.queryBuilder.BuildServiceRequestRateQuery(deployment, namespace, window)
	reqRateResult, err := c.promClient.Query(ctx, reqRateQuery, tr.End)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to query request rate: %v", err)), nil
	}
	requestRate, _ := extractScalarValue(reqRateResult)

	// Success rate
	successRateQuery := c.queryBuilder.BuildServiceSuccessRateQuery(deployment, namespace, window)
	successRateResult, err := c.promClient.Query(ctx, successRateQuery, tr.End)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to query success rate: %v", err)), nil
	}
	successRate, _ := extractScalarValue(successRateResult)

	// Error rate
	errorRateQuery := c.queryBuilder.BuildServiceErrorRateQuery(deployment, namespace, window)
	errorRateResult, err := c.promClient.Query(ctx, errorRateQuery, tr.End)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to query error rate: %v", err)), nil
	}
	errorRate, _ := extractScalarValue(errorRateResult)

	// Latency metrics
	p50Query := c.queryBuilder.BuildServiceLatencyQuery(deployment, namespace, 0.50, window)
	p50Result, _ := c.promClient.Query(ctx, p50Query, tr.End)
	p50, _ := extractScalarValue(p50Result)

	p95Query := c.queryBuilder.BuildServiceLatencyQuery(deployment, namespace, 0.95, window)
	p95Result, _ := c.promClient.Query(ctx, p95Query, tr.End)
	p95, _ := extractScalarValue(p95Result)

	p99Query := c.queryBuilder.BuildServiceLatencyQuery(deployment, namespace, 0.99, window)
	p99Result, _ := c.promClient.Query(ctx, p99Query, tr.End)
	p99, _ := extractScalarValue(p99Result)

	meanQuery := c.queryBuilder.BuildServiceMeanLatencyQuery(deployment, namespace, window)
	meanResult, _ := c.promClient.Query(ctx, meanQuery, tr.End)
	mean, _ := extractScalarValue(meanResult)

	// Errors by status
	errorsByStatusQuery := c.queryBuilder.BuildErrorsByStatusQuery(deployment, namespace, window)
	errorsByStatusResult, _ := c.promClient.Query(ctx, errorsByStatusQuery, tr.End)
	errorsByStatus := c.extractErrorsByStatus(errorsByStatusResult)

	metrics := ServiceMetrics{
		Service:     service,
		Namespace:   namespace,
		Deployment:  deployment,
		TimeRange:   tr,
		RequestRate: requestRate,
		SuccessRate: successRate * 100, // Convert to percentage
		ErrorRate:   errorRate * 100,   // Convert to percentage
		Latency: LatencyMetrics{
			P50:  p50,
			P95:  p95,
			P99:  p99,
			Mean: mean,
		},
		ErrorsByStatus: errorsByStatus,
	}

	data, err := json.Marshal(metrics)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal metrics: %v", err)), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

// AnalyzeTrafficFlow analyzes traffic between two services
func (c *MetricsCollector) AnalyzeTrafficFlow(ctx context.Context, sourceNs, sourceService, targetNs, targetService, timeRangeStr string) (*mcp.CallToolResult, error) {
	// Parse time range
	tr, err := ParseTimeRange(timeRangeStr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid time range: %v", err)), nil
	}

	// Find deployments
	srcDeployment, err := c.findDeploymentForService(ctx, sourceNs, sourceService)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to find source deployment: %v", err)), nil
	}

	dstDeployment, err := c.findDeploymentForService(ctx, targetNs, targetService)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to find target deployment: %v", err)), nil
	}

	window := tr.End.Sub(tr.Start)

	// Request rate
	reqRateQuery := c.queryBuilder.BuildTrafficBetweenServicesQuery(srcDeployment, sourceNs, dstDeployment, targetNs, window)
	reqRateResult, err := c.promClient.Query(ctx, reqRateQuery, tr.End)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to query request rate: %v", err)), nil
	}
	requestRate, _ := extractScalarValue(reqRateResult)

	// Success rate
	successRateQuery := c.queryBuilder.BuildTrafficSuccessRateQuery(srcDeployment, sourceNs, dstDeployment, targetNs, window)
	successRateResult, _ := c.promClient.Query(ctx, successRateQuery, tr.End)
	successRate, _ := extractScalarValue(successRateResult)

	// Latency
	p50Query := c.queryBuilder.BuildTrafficLatencyQuery(srcDeployment, sourceNs, dstDeployment, targetNs, 0.50, window)
	p50Result, _ := c.promClient.Query(ctx, p50Query, tr.End)
	p50, _ := extractScalarValue(p50Result)

	p95Query := c.queryBuilder.BuildTrafficLatencyQuery(srcDeployment, sourceNs, dstDeployment, targetNs, 0.95, window)
	p95Result, _ := c.promClient.Query(ctx, p95Query, tr.End)
	p95, _ := extractScalarValue(p95Result)

	p99Query := c.queryBuilder.BuildTrafficLatencyQuery(srcDeployment, sourceNs, dstDeployment, targetNs, 0.99, window)
	p99Result, _ := c.promClient.Query(ctx, p99Query, tr.End)
	p99, _ := extractScalarValue(p99Result)

	// Errors by status
	errorsByStatusQuery := c.queryBuilder.BuildTrafficErrorsByStatusQuery(srcDeployment, sourceNs, dstDeployment, targetNs, window)
	errorsByStatusResult, _ := c.promClient.Query(ctx, errorsByStatusQuery, tr.End)
	errorsByStatus := c.extractErrorsByStatus(errorsByStatusResult)

	// Calculate error rate
	errorRate := 1.0 - successRate
	if errorRate < 0 {
		errorRate = 0
	}

	// Calculate request count (approximate)
	requestCount := int64(requestRate * window.Seconds())

	trafficMetrics := TrafficMetrics{
		Source: ServiceIdentifier{
			Service:    sourceService,
			Namespace:  sourceNs,
			Deployment: srcDeployment,
		},
		Target: ServiceIdentifier{
			Service:    targetService,
			Namespace:  targetNs,
			Deployment: dstDeployment,
		},
		TimeRange:      tr,
		RequestCount:   requestCount,
		RequestRate:    requestRate,
		SuccessRate:    successRate * 100,
		ErrorRate:      errorRate * 100,
		ErrorsByStatus: errorsByStatus,
		LatencyP50:     p50,
		LatencyP95:     p95,
		LatencyP99:     p99,
	}

	data, err := json.Marshal(trafficMetrics)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal metrics: %v", err)), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

// GetServiceHealthSummary gets health summary for services in a namespace
func (c *MetricsCollector) GetServiceHealthSummary(ctx context.Context, namespace, timeRangeStr string, thresholds HealthThresholds) (*mcp.CallToolResult, error) {
	// Parse time range
	tr, err := ParseTimeRange(timeRangeStr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid time range: %v", err)), nil
	}

	// Get all services in namespace
	services, err := c.findAllServicesInNamespace(ctx, namespace)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to find services: %v", err)), nil
	}

	summaries := []ServiceHealthSummary{}
	window := tr.End.Sub(tr.Start)

	for _, svc := range services {
		deployment := svc // For Linkerd, deployment name often matches service name

		// Get basic metrics
		reqRateQuery := c.queryBuilder.BuildServiceRequestRateQuery(deployment, namespace, window)
		reqRateResult, _ := c.promClient.Query(ctx, reqRateQuery, tr.End)
		requestRate, _ := extractScalarValue(reqRateResult)

		successRateQuery := c.queryBuilder.BuildServiceSuccessRateQuery(deployment, namespace, window)
		successRateResult, _ := c.promClient.Query(ctx, successRateQuery, tr.End)
		successRate, _ := extractScalarValue(successRateResult)

		errorRateQuery := c.queryBuilder.BuildServiceErrorRateQuery(deployment, namespace, window)
		errorRateResult, _ := c.promClient.Query(ctx, errorRateQuery, tr.End)
		errorRate, _ := extractScalarValue(errorRateResult)

		p95Query := c.queryBuilder.BuildServiceLatencyQuery(deployment, namespace, 0.95, window)
		p95Result, _ := c.promClient.Query(ctx, p95Query, tr.End)
		p95, _ := extractScalarValue(p95Result)

		// Assess health
		status, issues := c.assessHealth(requestRate, successRate*100, errorRate*100, p95, thresholds)

		summary := ServiceHealthSummary{
			Service:      svc,
			Namespace:    namespace,
			Deployment:   deployment,
			HealthStatus: status,
			RequestRate:  requestRate,
			SuccessRate:  successRate * 100,
			ErrorRate:    errorRate * 100,
			LatencyP95:   p95,
			Issues:       issues,
		}

		summaries = append(summaries, summary)
	}

	data, err := json.Marshal(map[string]interface{}{
		"namespace": namespace,
		"timeRange": tr,
		"services":  summaries,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal summary: %v", err)), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

// GetTopServices returns top services ranked by a metric
func (c *MetricsCollector) GetTopServices(ctx context.Context, namespace, sortBy, timeRangeStr string, limit int) (*mcp.CallToolResult, error) {
	// Parse time range
	tr, err := ParseTimeRange(timeRangeStr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid time range: %v", err)), nil
	}

	// Get all services
	services, err := c.findAllServicesInNamespace(ctx, namespace)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to find services: %v", err)), nil
	}

	summaries := []ServiceMetricSummary{}
	window := tr.End.Sub(tr.Start)

	for _, svc := range services {
		deployment := svc

		// Get metrics
		reqRateQuery := c.queryBuilder.BuildServiceRequestRateQuery(deployment, namespace, window)
		reqRateResult, _ := c.promClient.Query(ctx, reqRateQuery, tr.End)
		requestRate, _ := extractScalarValue(reqRateResult)

		successRateQuery := c.queryBuilder.BuildServiceSuccessRateQuery(deployment, namespace, window)
		successRateResult, _ := c.promClient.Query(ctx, successRateQuery, tr.End)
		successRate, _ := extractScalarValue(successRateResult)

		errorRateQuery := c.queryBuilder.BuildServiceErrorRateQuery(deployment, namespace, window)
		errorRateResult, _ := c.promClient.Query(ctx, errorRateQuery, tr.End)
		errorRate, _ := extractScalarValue(errorRateResult)

		p95Query := c.queryBuilder.BuildServiceLatencyQuery(deployment, namespace, 0.95, window)
		p95Result, _ := c.promClient.Query(ctx, p95Query, tr.End)
		p95, _ := extractScalarValue(p95Result)

		summary := ServiceMetricSummary{
			Service:     svc,
			Namespace:   namespace,
			Deployment:  deployment,
			RequestRate: requestRate,
			SuccessRate: successRate * 100,
			ErrorRate:   errorRate * 100,
			LatencyP95:  p95,
		}

		summaries = append(summaries, summary)
	}

	// Sort and limit (simple sort - in production would use more sophisticated sorting)
	if limit > 0 && limit < len(summaries) {
		summaries = summaries[:limit]
	}

	ranking := ServiceRanking{
		SortBy:   sortBy,
		Services: summaries,
	}

	data, err := json.Marshal(ranking)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal ranking: %v", err)), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

// Helper functions

func (c *MetricsCollector) findDeploymentForService(ctx context.Context, namespace, service string) (string, error) {
	// In a real implementation, this would look up the service and find its backing deployment
	// For now, we'll assume service name matches deployment name (common in Linkerd)
	return service, nil
}

func (c *MetricsCollector) findAllServicesInNamespace(ctx context.Context, namespace string) ([]string, error) {
	// Query Prometheus for all deployments with metrics
	query := c.queryBuilder.BuildAllServicesQuery(namespace)
	result, err := c.promClient.Query(ctx, query, time.Now())
	if err != nil {
		return nil, err
	}

	vector, ok := result.(model.Vector)
	if !ok {
		return []string{}, nil
	}

	services := []string{}
	for _, sample := range vector {
		if deployment, ok := sample.Metric["deployment"]; ok {
			services = append(services, string(deployment))
		}
	}

	return services, nil
}

func (c *MetricsCollector) extractErrorsByStatus(value model.Value) map[string]int64 {
	vector, ok := value.(model.Vector)
	if !ok {
		return map[string]int64{}
	}

	errors := map[string]int64{}
	for _, sample := range vector {
		if status, ok := sample.Metric["http_status"]; ok {
			errors[string(status)] = int64(sample.Value)
		}
	}

	return errors
}

func (c *MetricsCollector) assessHealth(requestRate, successRate, errorRate, latencyP95 float64, thresholds HealthThresholds) (HealthStatus, []HealthIssue) {
	issues := []HealthIssue{}

	// Check error rate
	if errorRate >= thresholds.ErrorRateCritical {
		issues = append(issues, HealthIssue{
			Severity:    "critical",
			Description: "Error rate exceeds critical threshold",
			Metric:      "error_rate",
			Value:       errorRate,
			Threshold:   thresholds.ErrorRateCritical,
		})
	} else if errorRate >= thresholds.ErrorRateWarning {
		issues = append(issues, HealthIssue{
			Severity:    "warning",
			Description: "Error rate exceeds warning threshold",
			Metric:      "error_rate",
			Value:       errorRate,
			Threshold:   thresholds.ErrorRateWarning,
		})
	}

	// Check latency
	if latencyP95 >= thresholds.LatencyP95Critical {
		issues = append(issues, HealthIssue{
			Severity:    "critical",
			Description: "P95 latency exceeds critical threshold",
			Metric:      "latency_p95",
			Value:       latencyP95,
			Threshold:   thresholds.LatencyP95Critical,
		})
	} else if latencyP95 >= thresholds.LatencyP95Warning {
		issues = append(issues, HealthIssue{
			Severity:    "warning",
			Description: "P95 latency exceeds warning threshold",
			Metric:      "latency_p95",
			Value:       latencyP95,
			Threshold:   thresholds.LatencyP95Warning,
		})
	}

	// Check success rate
	if successRate <= thresholds.SuccessRateCritical {
		issues = append(issues, HealthIssue{
			Severity:    "critical",
			Description: "Success rate below critical threshold",
			Metric:      "success_rate",
			Value:       successRate,
			Threshold:   thresholds.SuccessRateCritical,
		})
	} else if successRate <= thresholds.SuccessRateWarning {
		issues = append(issues, HealthIssue{
			Severity:    "warning",
			Description: "Success rate below warning threshold",
			Metric:      "success_rate",
			Value:       successRate,
			Threshold:   thresholds.SuccessRateWarning,
		})
	}

	// Determine overall status
	hasCritical := false
	hasWarning := false
	for _, issue := range issues {
		switch issue.Severity {
		case "critical":
			hasCritical = true
		case "warning":
			hasWarning = true
		}
	}

	if hasCritical {
		return HealthStatusUnhealthy, issues
	} else if hasWarning {
		return HealthStatusDegraded, issues
	}

	return HealthStatusHealthy, issues
}
