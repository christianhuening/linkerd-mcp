package validators

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

var serverGVR = schema.GroupVersionResource{
	Group:    "policy.linkerd.io",
	Version:  "v1beta3",
	Resource: "servers",
}

// ServerValidator validates Linkerd Server CRDs
type ServerValidator struct {
	clientset     kubernetes.Interface
	dynamicClient dynamic.Interface
}

// NewServerValidator creates a new Server validator
func NewServerValidator(clientset kubernetes.Interface, dynamicClient dynamic.Interface) *ServerValidator {
	return &ServerValidator{
		clientset:     clientset,
		dynamicClient: dynamicClient,
	}
}

// Validate validates a Server resource
func (v *ServerValidator) Validate(ctx context.Context, server *unstructured.Unstructured) ValidationResult {
	result := ValidationResult{
		ResourceType: "Server",
		Name:         server.GetName(),
		Namespace:    server.GetNamespace(),
		Issues:       []Issue{},
	}

	// Extract spec
	spec, found, err := unstructured.NestedMap(server.Object, "spec")
	if err != nil || !found {
		result.AddIssue(SeverityError, "Missing or invalid spec", "spec", "LNKD-001", "Add a valid spec field to the Server resource")
		result.Finalize()
		return result
	}

	// Validate podSelector
	v.validatePodSelector(ctx, &result, spec)

	// Validate port
	v.validatePort(ctx, &result, spec)

	// Validate proxyProtocol
	v.validateProxyProtocol(ctx, &result, spec)

	// Check for conflicts
	v.checkConflicts(ctx, &result, server, spec)

	result.Finalize()
	return result
}

func (v *ServerValidator) validatePodSelector(ctx context.Context, result *ValidationResult, spec map[string]interface{}) {
	podSelector, found, err := unstructured.NestedMap(spec, "podSelector")
	if err != nil || !found {
		result.AddIssue(SeverityError, "Missing podSelector", "spec.podSelector", "LNKD-002", "Add a podSelector to target specific pods")
		return
	}

	matchLabels, _, _ := unstructured.NestedStringMap(podSelector, "matchLabels")
	if len(matchLabels) == 0 {
		result.AddIssue(SeverityWarning, "Empty podSelector will match all pods", "spec.podSelector.matchLabels", "LNKD-003", "Specify matchLabels to target specific pods")
		return
	}

	// Check if any pods match the selector
	labelSelector := metav1.FormatLabelSelector(&metav1.LabelSelector{MatchLabels: matchLabels})
	pods, err := v.clientset.CoreV1().Pods(result.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err == nil && len(pods.Items) == 0 {
		result.AddIssue(SeverityWarning, "No pods match the podSelector", "spec.podSelector", "LNKD-004", "Ensure pods with matching labels exist or will be created")
	}
}

func (v *ServerValidator) validatePort(ctx context.Context, result *ValidationResult, spec map[string]interface{}) {
	port, found, err := unstructured.NestedInt64(spec, "port")
	if err != nil || !found {
		result.AddIssue(SeverityError, "Missing port specification", "spec.port", "LNKD-005", "Add a port number to the Server spec")
		return
	}

	if port < 1 || port > 65535 {
		result.AddIssue(SeverityError,
			fmt.Sprintf("Invalid port %d, must be between 1-65535", port),
			"spec.port",
			"LNKD-006",
			"Set port to a valid value between 1-65535")
	}
}

func (v *ServerValidator) validateProxyProtocol(ctx context.Context, result *ValidationResult, spec map[string]interface{}) {
	proxyProtocol, found, _ := unstructured.NestedString(spec, "proxyProtocol")
	if !found {
		// ProxyProtocol is optional, defaults to "unknown"
		return
	}

	validProtocols := map[string]bool{
		"unknown": true,
		"HTTP/1":  true,
		"HTTP/2":  true,
		"gRPC":    true,
		"opaque":  true,
		"TLS":     true,
	}

	if !validProtocols[proxyProtocol] {
		result.AddIssue(SeverityError,
			fmt.Sprintf("Invalid proxyProtocol '%s', must be one of: unknown, HTTP/1, HTTP/2, gRPC, opaque, TLS", proxyProtocol),
			"spec.proxyProtocol",
			"LNKD-007",
			"Set proxyProtocol to a valid value")
	}
}

func (v *ServerValidator) checkConflicts(ctx context.Context, result *ValidationResult, server *unstructured.Unstructured, spec map[string]interface{}) {
	// Get all servers in the namespace
	servers, err := v.dynamicClient.Resource(serverGVR).Namespace(result.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		// Don't fail validation if we can't check for conflicts
		return
	}

	currentPort, _, _ := unstructured.NestedInt64(spec, "port")
	currentPodSelector, _, _ := unstructured.NestedMap(spec, "podSelector")
	currentMatchLabels, _, _ := unstructured.NestedStringMap(currentPodSelector, "matchLabels")

	for _, otherServer := range servers.Items {
		// Skip self
		if otherServer.GetName() == server.GetName() {
			continue
		}

		otherSpec, _, _ := unstructured.NestedMap(otherServer.Object, "spec")
		otherPort, _, _ := unstructured.NestedInt64(otherSpec, "port")
		otherPodSelector, _, _ := unstructured.NestedMap(otherSpec, "podSelector")
		otherMatchLabels, _, _ := unstructured.NestedStringMap(otherPodSelector, "matchLabels")

		// Check if port and podSelector match
		if currentPort == otherPort && labelsOverlap(currentMatchLabels, otherMatchLabels) {
			result.AddIssue(SeverityError,
				fmt.Sprintf("Conflicts with Server '%s' on port %d", otherServer.GetName(), currentPort),
				"spec",
				"LNKD-008",
				fmt.Sprintf("Change port or podSelector to avoid conflict with '%s'", otherServer.GetName()))
		}
	}
}

// labelsOverlap checks if two label sets could select the same pods
func labelsOverlap(labels1, labels2 map[string]string) bool {
	if len(labels1) == 0 || len(labels2) == 0 {
		return true // Empty selector matches everything
	}

	// Simple check: if all labels in labels1 match those in labels2, they overlap
	for k, v := range labels1 {
		if v2, exists := labels2[k]; exists && v == v2 {
			return true
		}
	}
	return false
}

// ValidateAll validates all Server resources in a namespace
func (v *ServerValidator) ValidateAll(ctx context.Context, namespace string) []ValidationResult {
	var results []ValidationResult

	listOptions := metav1.ListOptions{}
	var servers *unstructured.UnstructuredList
	var err error

	if namespace == "" {
		servers, err = v.dynamicClient.Resource(serverGVR).List(ctx, listOptions)
	} else {
		servers, err = v.dynamicClient.Resource(serverGVR).Namespace(namespace).List(ctx, listOptions)
	}

	if err != nil {
		return results
	}

	for i := range servers.Items {
		result := v.Validate(ctx, &servers.Items[i])
		results = append(results, result)
	}

	return results
}
