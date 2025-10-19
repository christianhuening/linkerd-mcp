package policy

import (
	"context"
	"fmt"
	"log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// getServiceAccountForService retrieves the service account used by a service
func (a *Analyzer) getServiceAccountForService(ctx context.Context, namespace, service string) (string, error) {
	pods, err := a.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", service),
	})
	if err != nil {
		return "", fmt.Errorf("failed to list source pods: %v", err)
	}

	if len(pods.Items) == 0 {
		return "", fmt.Errorf("no pods found for service %s in namespace %s", service, namespace)
	}

	serviceAccount := pods.Items[0].Spec.ServiceAccountName
	if serviceAccount == "" {
		serviceAccount = "default"
	}

	return serviceAccount, nil
}

// findAllowedTargets finds all targets that a source with given service account can access
func (a *Analyzer) findAllowedTargets(ctx context.Context, sourceNamespace, sourceServiceAccount string) ([]map[string]interface{}, error) {
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
	serverList, err := a.dynamicClient.Resource(serverGVR).Namespace("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list Servers: %v (ensure Linkerd policy CRDs are installed)", err)
	}

	allowedTargets := []map[string]interface{}{}

	// For each Server, check if there's an AuthorizationPolicy allowing our source
	for _, server := range serverList.Items {
		serverNamespace := server.GetNamespace()
		serverName := server.GetName()

		// Get AuthorizationPolicies in the same namespace as the Server
		authPolicies, err := a.dynamicClient.Resource(authPolicyGVR).Namespace(serverNamespace).List(ctx, metav1.ListOptions{})
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
			if allowed := a.checkSourceAllowed(ctx, policy, serverNamespace, sourceNamespace, sourceServiceAccount); allowed {
				// Extract target service information from the Server
				targetInfo := a.extractServerInfo(server, policy.GetName())
				if targetInfo != nil {
					allowedTargets = append(allowedTargets, targetInfo)
				}
			}
		}
	}

	return allowedTargets, nil
}

// extractServerInfo extracts relevant information from a Server resource
func (a *Analyzer) extractServerInfo(server unstructured.Unstructured, policyName string) map[string]interface{} {
	podSelector, found, err := unstructured.NestedMap(server.Object, "spec", "podSelector")
	if err != nil || !found {
		return nil
	}

	matchLabels, found, err := unstructured.NestedMap(podSelector, "matchLabels")
	if err != nil || !found {
		return nil
	}

	port, found, err := unstructured.NestedFieldNoCopy(server.Object, "spec", "port")
	if err != nil || !found {
		return nil
	}

	return map[string]interface{}{
		"namespace":           server.GetNamespace(),
		"server":              server.GetName(),
		"labels":              matchLabels,
		"port":                port,
		"authorizationPolicy": policyName,
	}
}
