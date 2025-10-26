package validators

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ProxyValidator validates Linkerd proxy configuration annotations
type ProxyValidator struct {
	clientset kubernetes.Interface
}

// NewProxyValidator creates a new proxy configuration validator
func NewProxyValidator(clientset kubernetes.Interface) *ProxyValidator {
	return &ProxyValidator{
		clientset: clientset,
	}
}

// ValidateNamespace validates proxy annotations on a namespace
func (v *ProxyValidator) ValidateNamespace(ctx context.Context, ns *corev1.Namespace) ValidationResult {
	result := ValidationResult{
		ResourceType: "Namespace",
		Name:         ns.Name,
		Namespace:    ns.Name,
		Issues:       []Issue{},
	}

	annotations := ns.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	// Validate injection annotation
	v.validateInjectionAnnotation(&result, annotations)

	// Validate resource annotations
	v.validateCPURequest(&result, annotations)
	v.validateCPULimit(&result, annotations)
	v.validateMemoryRequest(&result, annotations)
	v.validateMemoryLimit(&result, annotations)

	// Validate log level
	v.validateLogLevel(&result, annotations)

	// Validate proxy version
	v.validateProxyVersion(&result, annotations)

	// Validate wait time
	v.validateWaitBeforeExit(&result, annotations)

	result.Finalize()
	return result
}

// ValidatePod validates proxy annotations on a pod
func (v *ProxyValidator) ValidatePod(ctx context.Context, pod *corev1.Pod) ValidationResult {
	result := ValidationResult{
		ResourceType: "Pod",
		Name:         pod.Name,
		Namespace:    pod.Namespace,
		Issues:       []Issue{},
	}

	annotations := pod.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	// Check if pod has linkerd proxy
	hasProxy := false
	for _, container := range pod.Spec.Containers {
		if container.Name == "linkerd-proxy" {
			hasProxy = true
			break
		}
	}

	// Validate injection annotation
	v.validateInjectionAnnotation(&result, annotations)

	// If pod should be injected but isn't, warn
	if annotations["linkerd.io/inject"] == "enabled" && !hasProxy {
		result.AddIssue(SeverityWarning,
			"Pod is marked for injection but doesn't have linkerd-proxy container",
			"metadata.annotations[linkerd.io/inject]",
			"LNKD-P001",
			"Ensure the Linkerd proxy injector webhook is running")
	}

	// Validate resource annotations
	v.validateCPURequest(&result, annotations)
	v.validateCPULimit(&result, annotations)
	v.validateMemoryRequest(&result, annotations)
	v.validateMemoryLimit(&result, annotations)

	// Validate log level
	v.validateLogLevel(&result, annotations)

	// Validate proxy version
	v.validateProxyVersion(&result, annotations)

	// Validate wait time
	v.validateWaitBeforeExit(&result, annotations)

	result.Finalize()
	return result
}

func (v *ProxyValidator) validateInjectionAnnotation(result *ValidationResult, annotations map[string]string) {
	inject, exists := annotations["linkerd.io/inject"]
	if !exists {
		result.AddIssue(SeverityInfo,
			"No linkerd.io/inject annotation set",
			"metadata.annotations",
			"LNKD-P002",
			"Add 'linkerd.io/inject: enabled' to enable automatic proxy injection")
		return
	}

	validValues := map[string]bool{
		"enabled":  true,
		"disabled": true,
		"ingress":  true,
	}

	if !validValues[inject] {
		result.AddIssue(SeverityError,
			fmt.Sprintf("Invalid inject value '%s', must be: enabled, disabled, or ingress", inject),
			"metadata.annotations[linkerd.io/inject]",
			"LNKD-P003",
			"Set to 'enabled', 'disabled', or 'ingress'")
	}
}

func (v *ProxyValidator) validateCPURequest(result *ValidationResult, annotations map[string]string) {
	if cpu, exists := annotations["config.linkerd.io/proxy-cpu-request"]; exists {
		if !isValidResourceQuantity(cpu) {
			result.AddIssue(SeverityError,
				fmt.Sprintf("Invalid CPU request format: %s", cpu),
				"metadata.annotations[config.linkerd.io/proxy-cpu-request]",
				"LNKD-P004",
				"Use valid Kubernetes resource format (e.g., '100m', '0.1')")
		}
	}
}

func (v *ProxyValidator) validateCPULimit(result *ValidationResult, annotations map[string]string) {
	if cpu, exists := annotations["config.linkerd.io/proxy-cpu-limit"]; exists {
		if !isValidResourceQuantity(cpu) {
			result.AddIssue(SeverityError,
				fmt.Sprintf("Invalid CPU limit format: %s", cpu),
				"metadata.annotations[config.linkerd.io/proxy-cpu-limit]",
				"LNKD-P005",
				"Use valid Kubernetes resource format (e.g., '1', '1000m')")
		}
	}

	// Check if limit is set but request is not
	request, hasRequest := annotations["config.linkerd.io/proxy-cpu-request"]
	limit, hasLimit := annotations["config.linkerd.io/proxy-cpu-limit"]
	if hasLimit && !hasRequest {
		result.AddIssue(SeverityWarning,
			"CPU limit is set without CPU request",
			"metadata.annotations",
			"LNKD-P006",
			"Set config.linkerd.io/proxy-cpu-request for better scheduling")
	} else if hasLimit && hasRequest {
		// Warn if limit is lower than request
		reqVal := parseResourceQuantity(request)
		limVal := parseResourceQuantity(limit)
		if reqVal > 0 && limVal > 0 && limVal < reqVal {
			result.AddIssue(SeverityError,
				"CPU limit is lower than CPU request",
				"metadata.annotations[config.linkerd.io/proxy-cpu-limit]",
				"LNKD-P007",
				"CPU limit must be greater than or equal to CPU request")
		}
	}
}

func (v *ProxyValidator) validateMemoryRequest(result *ValidationResult, annotations map[string]string) {
	if mem, exists := annotations["config.linkerd.io/proxy-memory-request"]; exists {
		if !isValidResourceQuantity(mem) {
			result.AddIssue(SeverityError,
				fmt.Sprintf("Invalid memory request format: %s", mem),
				"metadata.annotations[config.linkerd.io/proxy-memory-request]",
				"LNKD-P008",
				"Use valid Kubernetes resource format (e.g., '64Mi', '128Mi')")
		}
	}
}

func (v *ProxyValidator) validateMemoryLimit(result *ValidationResult, annotations map[string]string) {
	if mem, exists := annotations["config.linkerd.io/proxy-memory-limit"]; exists {
		if !isValidResourceQuantity(mem) {
			result.AddIssue(SeverityError,
				fmt.Sprintf("Invalid memory limit format: %s", mem),
				"metadata.annotations[config.linkerd.io/proxy-memory-limit]",
				"LNKD-P009",
				"Use valid Kubernetes resource format (e.g., '128Mi', '256Mi')")
		}
	}

	// Check if limit is set but request is not
	request, hasRequest := annotations["config.linkerd.io/proxy-memory-request"]
	limit, hasLimit := annotations["config.linkerd.io/proxy-memory-limit"]
	if hasLimit && !hasRequest {
		result.AddIssue(SeverityWarning,
			"Memory limit is set without memory request",
			"metadata.annotations",
			"LNKD-P010",
			"Set config.linkerd.io/proxy-memory-request for better scheduling")
	} else if hasLimit && hasRequest {
		// Warn if limit is lower than request
		reqVal := parseMemoryQuantity(request)
		limVal := parseMemoryQuantity(limit)
		if reqVal > 0 && limVal > 0 && limVal < reqVal {
			result.AddIssue(SeverityError,
				"Memory limit is lower than memory request",
				"metadata.annotations[config.linkerd.io/proxy-memory-limit]",
				"LNKD-P011",
				"Memory limit must be greater than or equal to memory request")
		}
	}
}

func (v *ProxyValidator) validateLogLevel(result *ValidationResult, annotations map[string]string) {
	if logLevel, exists := annotations["config.linkerd.io/proxy-log-level"]; exists {
		validLevels := map[string]bool{
			"trace": true,
			"debug": true,
			"info":  true,
			"warn":  true,
			"error": true,
		}

		if !validLevels[logLevel] {
			result.AddIssue(SeverityError,
				fmt.Sprintf("Invalid log level '%s', must be: trace, debug, info, warn, error", logLevel),
				"metadata.annotations[config.linkerd.io/proxy-log-level]",
				"LNKD-P012",
				"Set to one of: trace, debug, info, warn, error")
		}

		// Warn about trace/debug in production
		if logLevel == "trace" || logLevel == "debug" {
			result.AddIssue(SeverityWarning,
				fmt.Sprintf("Log level '%s' may impact performance and increase log volume", logLevel),
				"metadata.annotations[config.linkerd.io/proxy-log-level]",
				"LNKD-P013",
				"Consider using 'info' or 'warn' for production workloads")
		}
	}
}

func (v *ProxyValidator) validateProxyVersion(result *ValidationResult, annotations map[string]string) {
	if version, exists := annotations["config.linkerd.io/proxy-version"]; exists {
		// Basic version format validation (e.g., stable-2.14.0, edge-24.1.1)
		validVersion := regexp.MustCompile(`^(stable|edge)-\d+\.\d+\.\d+$`)
		if !validVersion.MatchString(version) {
			result.AddIssue(SeverityWarning,
				fmt.Sprintf("Proxy version '%s' doesn't match expected format (stable-X.Y.Z or edge-X.Y.Z)", version),
				"metadata.annotations[config.linkerd.io/proxy-version]",
				"LNKD-P014",
				"Use format: stable-2.14.0 or edge-24.1.1")
		}
	}
}

func (v *ProxyValidator) validateWaitBeforeExit(result *ValidationResult, annotations map[string]string) {
	if wait, exists := annotations["config.alpha.linkerd.io/proxy-wait-before-exit-seconds"]; exists {
		seconds, err := strconv.Atoi(wait)
		if err != nil || seconds < 0 {
			result.AddIssue(SeverityError,
				fmt.Sprintf("Invalid wait-before-exit-seconds value: %s", wait),
				"metadata.annotations[config.alpha.linkerd.io/proxy-wait-before-exit-seconds]",
				"LNKD-P015",
				"Must be a non-negative integer")
		}

		if seconds > 300 {
			result.AddIssue(SeverityWarning,
				fmt.Sprintf("Very long wait time (%d seconds) may delay pod termination", seconds),
				"metadata.annotations[config.alpha.linkerd.io/proxy-wait-before-exit-seconds]",
				"LNKD-P016",
				"Consider a shorter wait time (typically 0-60 seconds)")
		}
	}
}

// ValidateAllNamespaces validates proxy configuration for all namespaces
func (v *ProxyValidator) ValidateAllNamespaces(ctx context.Context) []ValidationResult {
	var results []ValidationResult

	namespaces, err := v.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return results
	}

	for i := range namespaces.Items {
		result := v.ValidateNamespace(ctx, &namespaces.Items[i])
		results = append(results, result)
	}

	return results
}

// ValidateAllPodsInNamespace validates proxy configuration for all pods in a namespace
func (v *ProxyValidator) ValidateAllPodsInNamespace(ctx context.Context, namespace string) []ValidationResult {
	var results []ValidationResult

	var pods *corev1.PodList
	var err error

	if namespace == "" {
		pods, err = v.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	} else {
		pods, err = v.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	}

	if err != nil {
		return results
	}

	for i := range pods.Items {
		result := v.ValidatePod(ctx, &pods.Items[i])
		results = append(results, result)
	}

	return results
}

// Helper functions

func isValidResourceQuantity(s string) bool {
	// Match Kubernetes resource quantity format: number with optional suffix
	// Examples: 100m, 0.1, 1, 128Mi, 1Gi
	pattern := regexp.MustCompile(`^\d+(\.\d+)?(m|Mi|Gi|Ki|M|G|K)?$`)
	return pattern.MatchString(s)
}

func parseResourceQuantity(s string) float64 {
	// Simple parser for CPU quantities
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "m") {
		val, _ := strconv.ParseFloat(strings.TrimSuffix(s, "m"), 64)
		return val / 1000 // millicores to cores
	}
	val, _ := strconv.ParseFloat(s, 64)
	return val
}

func parseMemoryQuantity(s string) float64 {
	// Simple parser for memory quantities (convert to bytes)
	s = strings.TrimSpace(s)
	multipliers := map[string]float64{
		"Ki": 1024,
		"Mi": 1024 * 1024,
		"Gi": 1024 * 1024 * 1024,
		"K":  1000,
		"M":  1000 * 1000,
		"G":  1000 * 1000 * 1000,
	}

	for suffix, multiplier := range multipliers {
		if strings.HasSuffix(s, suffix) {
			val, _ := strconv.ParseFloat(strings.TrimSuffix(s, suffix), 64)
			return val * multiplier
		}
	}

	val, _ := strconv.ParseFloat(s, 64)
	return val
}
