package policy

import (
	"context"
	"fmt"
	"log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// findServersForService finds all Server resources for a given service
func (a *Analyzer) findServersForService(ctx context.Context, namespace, service string) ([]string, error) {
	serverGVR := schema.GroupVersionResource{
		Group:    "policy.linkerd.io",
		Version:  "v1beta3",
		Resource: "servers",
	}

	servers, err := a.dynamicClient.Resource(serverGVR).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list Servers: %v (ensure Linkerd policy CRDs are installed)", err)
	}

	matchingServers := []string{}

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
		if appLabel, ok := matchLabels["app"].(string); ok && appLabel == service {
			matchingServers = append(matchingServers, server.GetName())
		}
	}

	return matchingServers, nil
}

// findAllowedSources finds all sources that can access the given servers
func (a *Analyzer) findAllowedSources(ctx context.Context, namespace string, matchingServers []string) ([]map[string]interface{}, error) {
	authPolicyGVR := schema.GroupVersionResource{
		Group:    "policy.linkerd.io",
		Version:  "v1alpha1",
		Resource: "authorizationpolicies",
	}

	authPolicies, err := a.dynamicClient.Resource(authPolicyGVR).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list AuthorizationPolicies: %v", err)
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

			sources := a.extractSourcesFromAuth(ctx, namespace, authName, authKind, policy.GetName())
			for key, source := range sources {
				sourcesMap[key] = source
			}
		}
	}

	// Convert map to slice
	allowedSources := []map[string]interface{}{}
	for _, source := range sourcesMap {
		allowedSources = append(allowedSources, source)
	}

	return allowedSources, nil
}

// extractSourcesFromAuth extracts sources from an authentication resource
func (a *Analyzer) extractSourcesFromAuth(ctx context.Context, namespace, authName, authKind, policyName string) map[string]map[string]interface{} {
	sources := make(map[string]map[string]interface{})

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
		return sources
	}

	auth, err := a.dynamicClient.Resource(authGVR).Namespace(namespace).Get(ctx, authName, metav1.GetOptions{})
	if err != nil {
		log.Printf("Warning: Failed to get authentication %s: %v", authName, err)
		return sources
	}

	// Extract identities
	identities, found, err := unstructured.NestedSlice(auth.Object, "spec", "identities")
	if err == nil && found {
		for _, identity := range identities {
			identityStr, ok := identity.(string)
			if !ok {
				continue
			}

			if identityStr == "*" {
				key := "all-authenticated"
				sources[key] = map[string]interface{}{
					"type":                 "wildcard",
					"description":          "All authenticated services",
					"authenticationPolicy": authName,
					"authorizationPolicy":  policyName,
				}
			} else {
				key := identityStr
				sources[key] = map[string]interface{}{
					"identity":             identityStr,
					"authenticationPolicy": authName,
					"authorizationPolicy":  policyName,
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
				saNamespace = namespace
			}

			key := fmt.Sprintf("%s/%s", saNamespace, saName)
			sources[key] = map[string]interface{}{
				"serviceAccount":       saName,
				"namespace":            saNamespace,
				"authenticationPolicy": authName,
				"authorizationPolicy":  policyName,
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
			sources[key] = map[string]interface{}{
				"type":                 "network",
				"cidr":                 cidr,
				"except":               except,
				"authenticationPolicy": authName,
				"authorizationPolicy":  policyName,
			}
		}
	}

	return sources
}
