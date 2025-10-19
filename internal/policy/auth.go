package policy

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// checkSourceAllowed checks if a source is allowed by an authorization policy
func (a *Analyzer) checkSourceAllowed(ctx context.Context, policy unstructured.Unstructured, serverNamespace, sourceNamespace, sourceServiceAccount string) bool {
	requiredAuths, found, err := unstructured.NestedSlice(policy.Object, "spec", "requiredAuthenticationRefs")
	if err != nil || !found {
		// If no authentication required, it's allowed by default (depending on mode)
		return false
	}

	// Check if any of the required authentications match our source
	for _, authRef := range requiredAuths {
		authMap, ok := authRef.(map[string]interface{})
		if !ok {
			continue
		}

		authName, _, _ := unstructured.NestedString(authMap, "name")
		authKind, _, _ := unstructured.NestedString(authMap, "kind")

		if authKind == "MeshTLSAuthentication" || authKind == "NetworkAuthentication" {
			if a.checkAuthenticationMatch(ctx, serverNamespace, authName, authKind, sourceNamespace, sourceServiceAccount) {
				return true
			}
		}
	}

	return false
}

// checkAuthenticationMatch checks if an authentication resource matches the source
func (a *Analyzer) checkAuthenticationMatch(ctx context.Context, serverNamespace, authName, authKind, sourceNamespace, sourceServiceAccount string) bool {
	authGVR := schema.GroupVersionResource{
		Group:    "policy.linkerd.io",
		Version:  "v1alpha1",
		Resource: "meshtlsauthentications",
	}
	if authKind == "NetworkAuthentication" {
		authGVR.Resource = "networkauthentications"
	}

	auth, err := a.dynamicClient.Resource(authGVR).Namespace(serverNamespace).Get(ctx, authName, metav1.GetOptions{})
	if err != nil {
		return false
	}

	// Check identities
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
				return true
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
				return true
			}
		}
	}

	return false
}
