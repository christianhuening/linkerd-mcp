package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type LinkerdMCPServer struct {
	clientset     *kubernetes.Clientset
	dynamicClient dynamic.Interface
}

func NewLinkerdMCPServer() (*LinkerdMCPServer, error) {
	config, err := getKubeConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	return &LinkerdMCPServer{
		clientset:     clientset,
		dynamicClient: dynamicClient,
	}, nil
}

func getKubeConfig() (*rest.Config, error) {
	// Try in-cluster config first (when running in Kubernetes)
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	// Fall back to kubeconfig file (for local development)
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = os.Getenv("HOME") + "/.kube/config"
	}

	config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func main() {
	linkerdServer, err := NewLinkerdMCPServer()
	if err != nil {
		log.Fatalf("Failed to initialize Linkerd MCP server: %v", err)
	}

	s := server.NewMCPServer(
		"linkerd-mcp",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// Register tool: Check mesh health
	checkMeshHealthTool := mcp.NewTool("check_mesh_health",
		mcp.WithDescription("Checks the health status of the Linkerd service mesh in the cluster"),
		mcp.WithString("namespace",
			mcp.Description("The namespace to check (defaults to 'linkerd')"),
		),
	)
	s.AddTool(checkMeshHealthTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := request.Params.Arguments.(map[string]interface{})
		return linkerdServer.checkMeshHealth(ctx, args)
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
	s.AddTool(analyzeConnectivityTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := request.Params.Arguments.(map[string]interface{})
		return linkerdServer.analyzeConnectivity(ctx, args)
	})

	// Register tool: List service mesh services
	listMeshedServicesTool := mcp.NewTool("list_meshed_services",
		mcp.WithDescription("Lists all services that are part of the Linkerd mesh"),
		mcp.WithString("namespace",
			mcp.Description("The namespace to filter services (optional, defaults to all namespaces)"),
		),
	)
	s.AddTool(listMeshedServicesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := request.Params.Arguments.(map[string]interface{})
		return linkerdServer.listMeshedServices(ctx, args)
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
	s.AddTool(getAllowedTargetsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := request.Params.Arguments.(map[string]interface{})
		return linkerdServer.getAllowedTargets(ctx, args)
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
	s.AddTool(getAllowedSourcesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := request.Params.Arguments.(map[string]interface{})
		return linkerdServer.getAllowedSources(ctx, args)
	})

	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func (s *LinkerdMCPServer) checkMeshHealth(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	namespace := "linkerd"
	if ns, ok := args["namespace"].(string); ok && ns != "" {
		namespace = ns
	}

	// Get Linkerd control plane pods
	pods, err := s.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "linkerd.io/control-plane-component",
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list control plane pods: %v", err)), nil
	}

	healthStatus := map[string]interface{}{
		"namespace":    namespace,
		"totalPods":    len(pods.Items),
		"healthyPods":  0,
		"unhealthyPods": 0,
		"components":   []map[string]interface{}{},
	}

	for _, pod := range pods.Items {
		component := pod.Labels["linkerd.io/control-plane-component"]
		healthy := true
		status := "Running"

		if pod.Status.Phase != corev1.PodPhase("Running") {
			healthy = false
			status = string(pod.Status.Phase)
		}

		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady && condition.Status != corev1.ConditionTrue {
				healthy = false
			}
		}

		if healthy {
			healthStatus["healthyPods"] = healthStatus["healthyPods"].(int) + 1
		} else {
			healthStatus["unhealthyPods"] = healthStatus["unhealthyPods"].(int) + 1
		}

		componentInfo := map[string]interface{}{
			"name":      pod.Name,
			"component": component,
			"healthy":   healthy,
			"status":    status,
		}

		healthStatus["components"] = append(healthStatus["components"].([]map[string]interface{}), componentInfo)
	}

	result, _ := json.MarshalIndent(healthStatus, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

func (s *LinkerdMCPServer) analyzeConnectivity(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	sourceNamespace, ok := args["source_namespace"].(string)
	if !ok || sourceNamespace == "" {
		return mcp.NewToolResultError("source_namespace is required"), nil
	}

	sourceService, ok := args["source_service"].(string)
	if !ok || sourceService == "" {
		return mcp.NewToolResultError("source_service is required"), nil
	}

	targetNamespace := sourceNamespace
	if tn, ok := args["target_namespace"].(string); ok && tn != "" {
		targetNamespace = tn
	}

	targetService, ok := args["target_service"].(string)
	if !ok || targetService == "" {
		return mcp.NewToolResultError("target_service is required"), nil
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

func (s *LinkerdMCPServer) listMeshedServices(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	namespace := ""
	if ns, ok := args["namespace"].(string); ok {
		namespace = ns
	}

	listOptions := metav1.ListOptions{}
	pods, err := s.clientset.CoreV1().Pods(namespace).List(ctx, listOptions)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list pods: %v", err)), nil
	}

	meshedServices := make(map[string]map[string]interface{})

	for _, pod := range pods.Items {
		// Check if pod has Linkerd proxy injected
		hasProxy := false
		for _, container := range pod.Spec.Containers {
			if container.Name == "linkerd-proxy" {
				hasProxy = true
				break
			}
		}

		if !hasProxy {
			continue
		}

		serviceName := pod.Labels["app"]
		if serviceName == "" {
			serviceName = pod.Labels["k8s-app"]
		}
		if serviceName == "" {
			continue
		}

		key := fmt.Sprintf("%s/%s", pod.Namespace, serviceName)
		if _, exists := meshedServices[key]; !exists {
			meshedServices[key] = map[string]interface{}{
				"namespace": pod.Namespace,
				"service":   serviceName,
				"pods":      []string{},
			}
		}

		meshedServices[key]["pods"] = append(
			meshedServices[key]["pods"].([]string),
			pod.Name,
		)
	}

	result := map[string]interface{}{
		"totalServices": len(meshedServices),
		"services":      meshedServices,
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(resultJSON)), nil
}

// getAllowedTargets finds all services that a given source service can communicate with
func (s *LinkerdMCPServer) getAllowedTargets(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	sourceNamespace, ok := args["source_namespace"].(string)
	if !ok || sourceNamespace == "" {
		return mcp.NewToolResultError("source_namespace is required"), nil
	}

	sourceService, ok := args["source_service"].(string)
	if !ok || sourceService == "" {
		return mcp.NewToolResultError("source_service is required"), nil
	}

	// Get the source service's pods to find their ServiceAccount
	pods, err := s.clientset.CoreV1().Pods(sourceNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", sourceService),
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list source pods: %v", err)), nil
	}

	if len(pods.Items) == 0 {
		return mcp.NewToolResultError(fmt.Sprintf("No pods found for service %s in namespace %s", sourceService, sourceNamespace)), nil
	}

	sourceServiceAccount := pods.Items[0].Spec.ServiceAccountName
	if sourceServiceAccount == "" {
		sourceServiceAccount = "default"
	}

	// Define Linkerd policy GVRs
	serverGVR := schema.GroupVersionResource{
		Group:    "policy.linkerd.io",
		Version:  "v1beta3",
		Resource: "servers",
	}

	authPolicyGVR := schema.GroupVersionResource{
		Group:    "policy.linkerd.io",
		Version:  "v1alpha1",
		Resource: "authorizationpolicies",
	}

	// Get all Servers in the cluster
	serverList, err := s.dynamicClient.Resource(serverGVR).Namespace("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list Servers: %v (ensure Linkerd policy CRDs are installed)", err)), nil
	}

	allowedTargets := []map[string]interface{}{}

	// For each Server, check if there's an AuthorizationPolicy allowing our source
	for _, server := range serverList.Items {
		serverNamespace := server.GetNamespace()
		serverName := server.GetName()

		// Get AuthorizationPolicies in the same namespace as the Server
		authPolicies, err := s.dynamicClient.Resource(authPolicyGVR).Namespace(serverNamespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			log.Printf("Warning: Failed to list AuthorizationPolicies in namespace %s: %v", serverNamespace, err)
			continue
		}

		// Check each policy to see if it allows our source
		for _, policy := range authPolicies.Items {
			targetRef, found, err := unstructured.NestedMap(policy.Object, "spec", "targetRef")
			if err != nil || !found {
				continue
			}

			// Check if this policy targets our Server
			targetName, _, _ := unstructured.NestedString(targetRef, "name")
			if targetName != serverName {
				continue
			}

			// Check if our source is allowed
			requiredAuths, found, err := unstructured.NestedSlice(policy.Object, "spec", "requiredAuthenticationRefs")
			if err != nil || !found {
				// If no authentication required, it's allowed by default (depending on mode)
				continue
			}

			// Check if any of the required authentications match our source
			allowed := false
			for _, authRef := range requiredAuths {
				authMap, ok := authRef.(map[string]interface{})
				if !ok {
					continue
				}

				authName, _, _ := unstructured.NestedString(authMap, "name")
				authKind, _, _ := unstructured.NestedString(authMap, "kind")

				// If it's a MeshTLSAuthentication, check if it matches our source
				if authKind == "MeshTLSAuthentication" || authKind == "NetworkAuthentication" {
					// Get the authentication resource
					authGVR := schema.GroupVersionResource{
						Group:    "policy.linkerd.io",
						Version:  "v1alpha1",
						Resource: "meshtlsauthentications",
					}
					if authKind == "NetworkAuthentication" {
						authGVR.Resource = "networkauthentications"
					}

					auth, err := s.dynamicClient.Resource(authGVR).Namespace(serverNamespace).Get(ctx, authName, metav1.GetOptions{})
					if err != nil {
						continue
					}

					// Check if the authentication matches our source
					identities, found, err := unstructured.NestedSlice(auth.Object, "spec", "identities")
					if err == nil && found {
						for _, identity := range identities {
							identityStr, ok := identity.(string)
							if !ok {
								continue
							}
							// Linkerd identities are in the format: {serviceaccount}.{namespace}.serviceaccount.identity.linkerd.cluster.local
							expectedIdentity := fmt.Sprintf("%s.%s.serviceaccount.identity.linkerd.cluster.local", sourceServiceAccount, sourceNamespace)
							if identityStr == expectedIdentity || identityStr == "*" {
								allowed = true
								break
							}
						}
					}

					// Check service accounts directly
					serviceAccounts, found, err := unstructured.NestedSlice(auth.Object, "spec", "serviceAccounts")
					if err == nil && found {
						for _, sa := range serviceAccounts {
							saMap, ok := sa.(map[string]interface{})
							if !ok {
								continue
							}
							saName, _, _ := unstructured.NestedString(saMap, "name")
							saNamespace, _, _ := unstructured.NestedString(saMap, "namespace")
							if saNamespace == "" {
								saNamespace = serverNamespace
							}
							if saName == sourceServiceAccount && saNamespace == sourceNamespace {
								allowed = true
								break
							}
						}
					}
				}

				if allowed {
					break
				}
			}

			if allowed {
				// Extract target service information from the Server
				podSelector, found, err := unstructured.NestedMap(server.Object, "spec", "podSelector")
				if err != nil || !found {
					continue
				}

				matchLabels, found, err := unstructured.NestedMap(podSelector, "matchLabels")
				if err != nil || !found {
					continue
				}

				port, found, err := unstructured.NestedFieldNoCopy(server.Object, "spec", "port")
				if err != nil || !found {
					continue
				}

				targetInfo := map[string]interface{}{
					"namespace":            serverNamespace,
					"server":               serverName,
					"labels":               matchLabels,
					"port":                 port,
					"authorizationPolicy":  policy.GetName(),
				}

				allowedTargets = append(allowedTargets, targetInfo)
			}
		}
	}

	result := map[string]interface{}{
		"source": map[string]string{
			"namespace":      sourceNamespace,
			"service":        sourceService,
			"serviceAccount": sourceServiceAccount,
		},
		"allowedTargets": allowedTargets,
		"totalTargets":   len(allowedTargets),
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(resultJSON)), nil
}

// getAllowedSources finds all services that can communicate with a given target service
func (s *LinkerdMCPServer) getAllowedSources(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	targetNamespace, ok := args["target_namespace"].(string)
	if !ok || targetNamespace == "" {
		return mcp.NewToolResultError("target_namespace is required"), nil
	}

	targetService, ok := args["target_service"].(string)
	if !ok || targetService == "" {
		return mcp.NewToolResultError("target_service is required"), nil
	}

	// Define Linkerd policy GVRs
	serverGVR := schema.GroupVersionResource{
		Group:    "policy.linkerd.io",
		Version:  "v1beta3",
		Resource: "servers",
	}

	authPolicyGVR := schema.GroupVersionResource{
		Group:    "policy.linkerd.io",
		Version:  "v1alpha1",
		Resource: "authorizationpolicies",
	}

	// Find Servers matching the target service
	servers, err := s.dynamicClient.Resource(serverGVR).Namespace(targetNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list Servers: %v (ensure Linkerd policy CRDs are installed)", err)), nil
	}

	allowedSources := []map[string]interface{}{}
	matchingServers := []string{}

	// Find servers for the target service
	for _, server := range servers.Items {
		podSelector, found, err := unstructured.NestedMap(server.Object, "spec", "podSelector")
		if err != nil || !found {
			continue
		}

		matchLabels, found, err := unstructured.NestedMap(podSelector, "matchLabels")
		if err != nil || !found {
			continue
		}

		// Check if the server matches our target service (by app label)
		if appLabel, ok := matchLabels["app"].(string); ok && appLabel == targetService {
			matchingServers = append(matchingServers, server.GetName())
		}
	}

	if len(matchingServers) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No Linkerd Servers found for service %s in namespace %s", targetService, targetNamespace)), nil
	}

	// Get AuthorizationPolicies for the matching servers
	authPolicies, err := s.dynamicClient.Resource(authPolicyGVR).Namespace(targetNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list AuthorizationPolicies: %v", err)), nil
	}

	// Map to track unique sources
	sourcesMap := make(map[string]map[string]interface{})

	for _, policy := range authPolicies.Items {
		targetRef, found, err := unstructured.NestedMap(policy.Object, "spec", "targetRef")
		if err != nil || !found {
			continue
		}

		// Check if this policy targets one of our matching servers
		targetName, _, _ := unstructured.NestedString(targetRef, "name")
		isMatchingServer := false
		for _, serverName := range matchingServers {
			if targetName == serverName {
				isMatchingServer = true
				break
			}
		}

		if !isMatchingServer {
			continue
		}

		// Get the required authentications
		requiredAuths, found, err := unstructured.NestedSlice(policy.Object, "spec", "requiredAuthenticationRefs")
		if err != nil || !found {
			continue
		}

		// Process each authentication reference
		for _, authRef := range requiredAuths {
			authMap, ok := authRef.(map[string]interface{})
			if !ok {
				continue
			}

			authName, _, _ := unstructured.NestedString(authMap, "name")
			authKind, _, _ := unstructured.NestedString(authMap, "kind")

			// Get the authentication resource
			var authGVR schema.GroupVersionResource
			if authKind == "MeshTLSAuthentication" {
				authGVR = schema.GroupVersionResource{
					Group:    "policy.linkerd.io",
					Version:  "v1alpha1",
					Resource: "meshtlsauthentications",
				}
			} else if authKind == "NetworkAuthentication" {
				authGVR = schema.GroupVersionResource{
					Group:    "policy.linkerd.io",
					Version:  "v1alpha1",
					Resource: "networkauthentications",
				}
			} else {
				continue
			}

			auth, err := s.dynamicClient.Resource(authGVR).Namespace(targetNamespace).Get(ctx, authName, metav1.GetOptions{})
			if err != nil {
				log.Printf("Warning: Failed to get authentication %s: %v", authName, err)
				continue
			}

			// Extract identities
			identities, found, err := unstructured.NestedSlice(auth.Object, "spec", "identities")
			if err == nil && found {
				for _, identity := range identities {
					identityStr, ok := identity.(string)
					if !ok {
						continue
					}

					// Parse Linkerd identity format: {serviceaccount}.{namespace}.serviceaccount.identity.linkerd.cluster.local
					if identityStr == "*" {
						key := "all-authenticated"
						sourcesMap[key] = map[string]interface{}{
							"type":                 "wildcard",
							"description":          "All authenticated services",
							"authenticationPolicy": authName,
							"authorizationPolicy":  policy.GetName(),
						}
					} else {
						key := identityStr
						sourcesMap[key] = map[string]interface{}{
							"identity":             identityStr,
							"authenticationPolicy": authName,
							"authorizationPolicy":  policy.GetName(),
						}
					}
				}
			}

			// Extract service accounts
			serviceAccounts, found, err := unstructured.NestedSlice(auth.Object, "spec", "serviceAccounts")
			if err == nil && found {
				for _, sa := range serviceAccounts {
					saMap, ok := sa.(map[string]interface{})
					if !ok {
						continue
					}
					saName, _, _ := unstructured.NestedString(saMap, "name")
					saNamespace, _, _ := unstructured.NestedString(saMap, "namespace")
					if saNamespace == "" {
						saNamespace = targetNamespace
					}

					key := fmt.Sprintf("%s/%s", saNamespace, saName)
					sourcesMap[key] = map[string]interface{}{
						"serviceAccount":       saName,
						"namespace":            saNamespace,
						"authenticationPolicy": authName,
						"authorizationPolicy":  policy.GetName(),
					}
				}
			}

			// Extract networks (for NetworkAuthentication)
			networks, found, err := unstructured.NestedSlice(auth.Object, "spec", "networks")
			if err == nil && found {
				for _, network := range networks {
					networkMap, ok := network.(map[string]interface{})
					if !ok {
						continue
					}
					cidr, _, _ := unstructured.NestedString(networkMap, "cidr")
					except, _, _ := unstructured.NestedSlice(networkMap, "except")

					key := fmt.Sprintf("network-%s", cidr)
					sourcesMap[key] = map[string]interface{}{
						"type":                 "network",
						"cidr":                 cidr,
						"except":               except,
						"authenticationPolicy": authName,
						"authorizationPolicy":  policy.GetName(),
					}
				}
			}
		}
	}

	// Convert map to slice
	for _, source := range sourcesMap {
		allowedSources = append(allowedSources, source)
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
