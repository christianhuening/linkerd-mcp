package server

import (
	"context"

	"github.com/christianhuening/linkerd-mcp/internal/config"
	"github.com/christianhuening/linkerd-mcp/internal/health"
	"github.com/christianhuening/linkerd-mcp/internal/mesh"
	"github.com/christianhuening/linkerd-mcp/internal/metrics"
	"github.com/christianhuening/linkerd-mcp/internal/policy"
	"github.com/christianhuening/linkerd-mcp/internal/validation"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// LinkerdMCPServer represents the MCP server for Linkerd
type LinkerdMCPServer struct {
	healthChecker    *health.Checker
	serviceLister    *mesh.ServiceLister
	policyAnalyzer   *policy.Analyzer
	configValidator  *validation.ConfigValidator
	metricsCollector *metrics.MetricsCollector
}

// New creates a new LinkerdMCPServer
func New() (*LinkerdMCPServer, error) {
	clients, err := config.NewKubernetesClients()
	if err != nil {
		return nil, err
	}

	// Create metrics collector (gracefully handle errors - metrics are optional)
	metricsCollector, err := metrics.NewMetricsCollector(clients.Config, clients.Clientset, "linkerd")
	if err != nil {
		// Log warning but don't fail - Prometheus may not be available
		metricsCollector = nil
	}

	return &LinkerdMCPServer{
		healthChecker:    health.NewChecker(clients.Clientset),
		serviceLister:    mesh.NewServiceLister(clients.Clientset),
		policyAnalyzer:   policy.NewAnalyzer(clients.Clientset, clients.DynamicClient),
		configValidator:  validation.NewConfigValidator(clients.Clientset, clients.DynamicClient),
		metricsCollector: metricsCollector,
	}, nil
}

// RegisterTools registers all MCP tools with the server
func (s *LinkerdMCPServer) RegisterTools(mcpServer *server.MCPServer) {
	// Register tool: Check mesh health
	checkMeshHealthTool := mcp.NewTool("check_mesh_health",
		mcp.WithDescription("Checks the health status of the Linkerd service mesh in the cluster"),
		mcp.WithString("namespace",
			mcp.Description("The namespace to check (defaults to 'linkerd')"),
		),
	)
	mcpServer.AddTool(checkMeshHealthTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := request.Params.Arguments.(map[string]interface{})
		namespace, _ := args["namespace"].(string)
		return s.healthChecker.CheckMeshHealth(ctx, namespace)
	})

	// Register tool: Analyze connectivity policies
	analyzeConnectivityTool := mcp.NewTool("analyze_connectivity",
		mcp.WithDescription("Analyzes Linkerd policies to determine allowed connectivity between services"),
		mcp.WithString("source_namespace",
			mcp.Required(),
			mcp.Description("The namespace of the source service"),
		),
		mcp.WithString("source_service",
			mcp.Required(),
			mcp.Description("The name of the source service"),
		),
		mcp.WithString("target_namespace",
			mcp.Description("The namespace of the target service (defaults to source_namespace)"),
		),
		mcp.WithString("target_service",
			mcp.Required(),
			mcp.Description("The name of the target service"),
		),
	)
	mcpServer.AddTool(analyzeConnectivityTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := request.Params.Arguments.(map[string]interface{})
		sourceNamespace, _ := args["source_namespace"].(string)
		sourceService, _ := args["source_service"].(string)
		targetNamespace, _ := args["target_namespace"].(string)
		targetService, _ := args["target_service"].(string)
		return s.policyAnalyzer.AnalyzeConnectivity(ctx, sourceNamespace, sourceService, targetNamespace, targetService)
	})

	// Register tool: List service mesh services
	listMeshedServicesTool := mcp.NewTool("list_meshed_services",
		mcp.WithDescription("Lists all services that are part of the Linkerd mesh"),
		mcp.WithString("namespace",
			mcp.Description("The namespace to filter services (optional, defaults to all namespaces)"),
		),
	)
	mcpServer.AddTool(listMeshedServicesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := request.Params.Arguments.(map[string]interface{})
		namespace, _ := args["namespace"].(string)
		return s.serviceLister.ListMeshedServices(ctx, namespace)
	})

	// Register tool: Get allowed targets for a source
	getAllowedTargetsTool := mcp.NewTool("get_allowed_targets",
		mcp.WithDescription("Find all services that a given source service can communicate with based on Linkerd authorization policies"),
		mcp.WithString("source_namespace",
			mcp.Required(),
			mcp.Description("The namespace of the source service"),
		),
		mcp.WithString("source_service",
			mcp.Required(),
			mcp.Description("The name of the source service"),
		),
	)
	mcpServer.AddTool(getAllowedTargetsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := request.Params.Arguments.(map[string]interface{})
		sourceNamespace, _ := args["source_namespace"].(string)
		sourceService, _ := args["source_service"].(string)
		return s.policyAnalyzer.GetAllowedTargets(ctx, sourceNamespace, sourceService)
	})

	// Register tool: Get allowed sources for a target
	getAllowedSourcesTool := mcp.NewTool("get_allowed_sources",
		mcp.WithDescription("Find all services that can communicate with a given target service based on Linkerd authorization policies"),
		mcp.WithString("target_namespace",
			mcp.Required(),
			mcp.Description("The namespace of the target service"),
		),
		mcp.WithString("target_service",
			mcp.Required(),
			mcp.Description("The name of the target service"),
		),
	)
	mcpServer.AddTool(getAllowedSourcesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := request.Params.Arguments.(map[string]interface{})
		targetNamespace, _ := args["target_namespace"].(string)
		targetService, _ := args["target_service"].(string)
		return s.policyAnalyzer.GetAllowedSources(ctx, targetNamespace, targetService)
	})

	// Register tool: Validate mesh configuration
	validateMeshConfigTool := mcp.NewTool("validate_mesh_config",
		mcp.WithDescription("Validate Linkerd service mesh configuration"),
		mcp.WithString("namespace",
			mcp.Description("Namespace to validate (empty for all namespaces)"),
		),
		mcp.WithString("resource_type",
			mcp.Description("Resource type to validate (server|authpolicy|meshtls|all)"),
		),
		mcp.WithString("resource_name",
			mcp.Description("Specific resource name to validate"),
		),
		mcp.WithBoolean("include_warnings",
			mcp.Description("Include warnings in results (default: true)"),
		),
	)
	mcpServer.AddTool(validateMeshConfigTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := request.Params.Arguments.(map[string]interface{})
		namespace, _ := args["namespace"].(string)
		resourceType, _ := args["resource_type"].(string)
		resourceName, _ := args["resource_name"].(string)
		includeWarnings := true
		if v, ok := args["include_warnings"].(bool); ok {
			includeWarnings = v
		}
		return s.configValidator.ValidateConfig(ctx, namespace, resourceType, resourceName, includeWarnings)
	})

	// Only register metrics tools if collector is available
	if s.metricsCollector != nil {
		// Register tool: Get service metrics
		getServiceMetricsTool := mcp.NewTool("get_service_metrics",
			mcp.WithDescription("Get traffic metrics for a service (request rate, latency, success rate)"),
			mcp.WithString("namespace",
				mcp.Required(),
				mcp.Description("The namespace of the service"),
			),
			mcp.WithString("service",
				mcp.Required(),
				mcp.Description("The name of the service"),
			),
			mcp.WithString("time_range",
				mcp.Description("Time range for metrics (e.g., '5m', '1h', '24h'). Default: 5m"),
			),
		)
		mcpServer.AddTool(getServiceMetricsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, _ := request.Params.Arguments.(map[string]interface{})
			namespace, _ := args["namespace"].(string)
			service, _ := args["service"].(string)
			timeRange, _ := args["time_range"].(string)
			return s.metricsCollector.GetServiceMetrics(ctx, namespace, service, timeRange)
		})

		// Register tool: Analyze traffic flow
		analyzeTrafficFlowTool := mcp.NewTool("analyze_traffic_flow",
			mcp.WithDescription("Analyze traffic metrics between two services"),
			mcp.WithString("source_namespace",
				mcp.Required(),
				mcp.Description("The namespace of the source service"),
			),
			mcp.WithString("source_service",
				mcp.Required(),
				mcp.Description("The name of the source service"),
			),
			mcp.WithString("target_namespace",
				mcp.Description("The namespace of the target service (defaults to source_namespace)"),
			),
			mcp.WithString("target_service",
				mcp.Required(),
				mcp.Description("The name of the target service"),
			),
			mcp.WithString("time_range",
				mcp.Description("Time range for metrics (e.g., '5m', '1h', '24h'). Default: 5m"),
			),
		)
		mcpServer.AddTool(analyzeTrafficFlowTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, _ := request.Params.Arguments.(map[string]interface{})
			sourceNs, _ := args["source_namespace"].(string)
			sourceService, _ := args["source_service"].(string)
			targetNs, _ := args["target_namespace"].(string)
			targetService, _ := args["target_service"].(string)
			timeRange, _ := args["time_range"].(string)
			if targetNs == "" {
				targetNs = sourceNs
			}
			return s.metricsCollector.AnalyzeTrafficFlow(ctx, sourceNs, sourceService, targetNs, targetService, timeRange)
		})

		// Register tool: Get service health summary
		getServiceHealthSummaryTool := mcp.NewTool("get_service_health_summary",
			mcp.WithDescription("Get health summary for all services in a namespace based on metrics"),
			mcp.WithString("namespace",
				mcp.Required(),
				mcp.Description("The namespace to check"),
			),
			mcp.WithString("time_range",
				mcp.Description("Time range for metrics (e.g., '5m', '1h', '24h'). Default: 5m"),
			),
		)
		mcpServer.AddTool(getServiceHealthSummaryTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, _ := request.Params.Arguments.(map[string]interface{})
			namespace, _ := args["namespace"].(string)
			timeRange, _ := args["time_range"].(string)
			thresholds := metrics.DefaultHealthThresholds()
			return s.metricsCollector.GetServiceHealthSummary(ctx, namespace, timeRange, thresholds)
		})

		// Register tool: Get top services
		getTopServicesTool := mcp.NewTool("get_top_services",
			mcp.WithDescription("Get services ranked by traffic metrics"),
			mcp.WithString("namespace",
				mcp.Required(),
				mcp.Description("The namespace to query"),
			),
			mcp.WithString("sort_by",
				mcp.Description("Sort by metric: 'request_rate', 'error_rate', 'latency_p95'. Default: request_rate"),
			),
			mcp.WithString("time_range",
				mcp.Description("Time range for metrics (e.g., '5m', '1h', '24h'). Default: 5m"),
			),
			mcp.WithNumber("limit",
				mcp.Description("Number of top services to return. Default: 10"),
			),
		)
		mcpServer.AddTool(getTopServicesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, _ := request.Params.Arguments.(map[string]interface{})
			namespace, _ := args["namespace"].(string)
			sortBy, _ := args["sort_by"].(string)
			timeRange, _ := args["time_range"].(string)
			limit := 10
			if l, ok := args["limit"].(float64); ok {
				limit = int(l)
			}
			if sortBy == "" {
				sortBy = "request_rate"
			}
			return s.metricsCollector.GetTopServices(ctx, namespace, sortBy, timeRange, limit)
		})
	}
}
