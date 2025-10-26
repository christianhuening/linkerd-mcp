package validators

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

var (
	authPolicyGVR = schema.GroupVersionResource{
		Group:    "policy.linkerd.io",
		Version:  "v1alpha1",
		Resource: "authorizationpolicies",
	}
	meshTLSAuthGVR = schema.GroupVersionResource{
		Group:    "policy.linkerd.io",
		Version:  "v1alpha1",
		Resource: "meshtlsauthentications",
	}
	networkAuthGVR = schema.GroupVersionResource{
		Group:    "policy.linkerd.io",
		Version:  "v1alpha1",
		Resource: "networkauthentications",
	}
)

// AuthPolicyValidator validates Linkerd AuthorizationPolicy CRDs
type AuthPolicyValidator struct {
	dynamicClient dynamic.Interface
}

// NewAuthPolicyValidator creates a new AuthorizationPolicy validator
func NewAuthPolicyValidator(dynamicClient dynamic.Interface) *AuthPolicyValidator {
	return &AuthPolicyValidator{
		dynamicClient: dynamicClient,
	}
}

// Validate validates an AuthorizationPolicy resource
func (v *AuthPolicyValidator) Validate(ctx context.Context, policy *unstructured.Unstructured) ValidationResult {
	result := ValidationResult{
		ResourceType: "AuthorizationPolicy",
		Name:         policy.GetName(),
		Namespace:    policy.GetNamespace(),
		Issues:       []Issue{},
	}

	// Extract spec
	spec, found, err := unstructured.NestedMap(policy.Object, "spec")
	if err != nil || !found {
		result.AddIssue(SeverityError, "Missing or invalid spec", "spec", "LNKD-009", "Add a valid spec field to the AuthorizationPolicy")
		result.Finalize()
		return result
	}

	// Validate targetRef
	v.validateTargetRef(ctx, &result, spec)

	// Validate authentication references
	v.validateAuthRefs(ctx, &result, spec)

	result.Finalize()
	return result
}

func (v *AuthPolicyValidator) validateTargetRef(ctx context.Context, result *ValidationResult, spec map[string]interface{}) {
	targetRef, found, err := unstructured.NestedMap(spec, "targetRef")
	if err != nil || !found {
		result.AddIssue(SeverityError, "Missing targetRef", "spec.targetRef", "LNKD-010", "Add a targetRef to specify which Server this policy applies to")
		return
	}

	kind, _, _ := unstructured.NestedString(targetRef, "kind")
	name, _, _ := unstructured.NestedString(targetRef, "name")
	targetNamespace, found, _ := unstructured.NestedString(targetRef, "namespace")
	if !found {
		targetNamespace = result.Namespace
	}

	if kind != "Server" {
		result.AddIssue(SeverityError,
			fmt.Sprintf("Invalid targetRef.kind '%s', must be 'Server'", kind),
			"spec.targetRef.kind",
			"LNKD-011",
			"Set targetRef.kind to 'Server'")
		return
	}

	if name == "" {
		result.AddIssue(SeverityError, "Missing targetRef.name", "spec.targetRef.name", "LNKD-012", "Specify the name of the target Server")
		return
	}

	// Check if the target server exists
	_, err = v.dynamicClient.Resource(serverGVR).Namespace(targetNamespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		result.AddIssue(SeverityError,
			fmt.Sprintf("Target Server '%s' does not exist in namespace '%s'", name, targetNamespace),
			"spec.targetRef",
			"LNKD-013",
			fmt.Sprintf("Create Server '%s' or correct the targetRef", name))
	}
}

func (v *AuthPolicyValidator) validateAuthRefs(ctx context.Context, result *ValidationResult, spec map[string]interface{}) {
	authRefs, found, err := unstructured.NestedSlice(spec, "requiredAuthenticationRefs")
	if err != nil {
		result.AddIssue(SeverityError, "Invalid requiredAuthenticationRefs", "spec.requiredAuthenticationRefs", "LNKD-014", "Fix the requiredAuthenticationRefs format")
		return
	}

	if !found || len(authRefs) == 0 {
		result.AddIssue(SeverityWarning, "No authentication requirements specified", "spec.requiredAuthenticationRefs", "LNKD-015", "Add requiredAuthenticationRefs to enforce authentication")
		return
	}

	for i, ref := range authRefs {
		refMap, ok := ref.(map[string]interface{})
		if !ok {
			result.AddIssue(SeverityError,
				fmt.Sprintf("Invalid authentication ref at index %d", i),
				fmt.Sprintf("spec.requiredAuthenticationRefs[%d]", i),
				"LNKD-016",
				"Ensure each authentication ref has name and kind fields")
			continue
		}

		v.validateAuthRef(ctx, result, refMap, i)
	}
}

func (v *AuthPolicyValidator) validateAuthRef(ctx context.Context, result *ValidationResult, ref map[string]interface{}, index int) {
	kind, _, _ := unstructured.NestedString(ref, "kind")
	name, _, _ := unstructured.NestedString(ref, "name")
	refNamespace, found, _ := unstructured.NestedString(ref, "namespace")
	if !found {
		refNamespace = result.Namespace
	}

	if name == "" {
		result.AddIssue(SeverityError,
			fmt.Sprintf("Missing name in authentication ref at index %d", index),
			fmt.Sprintf("spec.requiredAuthenticationRefs[%d].name", index),
			"LNKD-017",
			"Specify the name of the authentication resource")
		return
	}

	var gvr schema.GroupVersionResource
	switch kind {
	case "MeshTLSAuthentication":
		gvr = meshTLSAuthGVR
	case "NetworkAuthentication":
		gvr = networkAuthGVR
	default:
		result.AddIssue(SeverityError,
			fmt.Sprintf("Invalid authentication kind '%s' at index %d, must be 'MeshTLSAuthentication' or 'NetworkAuthentication'", kind, index),
			fmt.Sprintf("spec.requiredAuthenticationRefs[%d].kind", index),
			"LNKD-018",
			"Set kind to 'MeshTLSAuthentication' or 'NetworkAuthentication'")
		return
	}

	// Check if the authentication resource exists
	_, err := v.dynamicClient.Resource(gvr).Namespace(refNamespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		result.AddIssue(SeverityError,
			fmt.Sprintf("%s '%s' does not exist in namespace '%s'", kind, name, refNamespace),
			fmt.Sprintf("spec.requiredAuthenticationRefs[%d]", index),
			"LNKD-019",
			fmt.Sprintf("Create %s '%s' or correct the reference", kind, name))
	}
}

// ValidateAll validates all AuthorizationPolicy resources in a namespace
func (v *AuthPolicyValidator) ValidateAll(ctx context.Context, namespace string) []ValidationResult {
	var results []ValidationResult

	listOptions := metav1.ListOptions{}
	var policies *unstructured.UnstructuredList
	var err error

	if namespace == "" {
		policies, err = v.dynamicClient.Resource(authPolicyGVR).List(ctx, listOptions)
	} else {
		policies, err = v.dynamicClient.Resource(authPolicyGVR).Namespace(namespace).List(ctx, listOptions)
	}

	if err != nil {
		return results
	}

	for i := range policies.Items {
		result := v.Validate(ctx, &policies.Items[i])
		results = append(results, result)
	}

	return results
}
