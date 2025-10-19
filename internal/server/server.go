package server

import (
	"context"

	"github.com/christianhuening/linkerd-mcp/internal/config"
	"github.com/christianhuening/linkerd-mcp/internal/health"
	"github.com/christianhuening/linkerd-mcp/internal/mesh"
	"github.com/christianhuening/linkerd-mcp/internal/policy"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// LinkerdMCPServer represents the MCP server for Linkerd
type LinkerdMCPServer struct {
	healthChecker   *health.Checker
	serviceLister   *mesh.ServiceLister
	policyAnalyzer  *policy.Analyzer
}

// New creates a new LinkerdMCPServer
func New() (*LinkerdMCPServer, error) {
	clients, err := config.NewKubernetesClients()
	if err != nil {
		return nil, err
	}

	return &LinkerdMCPServer{
		healthChecker:  health.NewChecker(clients.Clientset),
		serviceLister:  mesh.NewServiceLister(clients.Clientset),
		policyAnalyzer: policy.NewAnalyzer(clients.Clientset, clients.DynamicClient),
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
}
