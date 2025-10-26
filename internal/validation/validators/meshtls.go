package validators

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// MeshTLSValidator validates Linkerd MeshTLSAuthentication CRDs
type MeshTLSValidator struct {
	clientset     kubernetes.Interface
	dynamicClient dynamic.Interface
}

// NewMeshTLSValidator creates a new MeshTLSAuthentication validator
func NewMeshTLSValidator(clientset kubernetes.Interface, dynamicClient dynamic.Interface) *MeshTLSValidator {
	return &MeshTLSValidator{
		clientset:     clientset,
		dynamicClient: dynamicClient,
	}
}

// Validate validates a MeshTLSAuthentication resource
func (v *MeshTLSValidator) Validate(ctx context.Context, auth *unstructured.Unstructured) ValidationResult {
	result := ValidationResult{
		ResourceType: "MeshTLSAuthentication",
		Name:         auth.GetName(),
		Namespace:    auth.GetNamespace(),
		Issues:       []Issue{},
	}

	// Extract spec
	spec, found, err := unstructured.NestedMap(auth.Object, "spec")
	if err != nil || !found {
		result.AddIssue(SeverityError, "Missing or invalid spec", "spec", "LNKD-020", "Add a valid spec field to the MeshTLSAuthentication")
		result.Finalize()
		return result
	}

	// Validate identities and serviceAccounts
	v.validateIdentitiesAndServiceAccounts(ctx, &result, spec)

	result.Finalize()
	return result
}

func (v *MeshTLSValidator) validateIdentitiesAndServiceAccounts(ctx context.Context, result *ValidationResult, spec map[string]interface{}) {
	identities, hasIdentities, _ := unstructured.NestedStringSlice(spec, "identities")
	serviceAccounts, hasSAs, _ := unstructured.NestedSlice(spec, "serviceAccounts")

	// At least one must be specified
	if (!hasIdentities || len(identities) == 0) && (!hasSAs || len(serviceAccounts) == 0) {
		result.AddIssue(SeverityError,
			"Must specify at least one identity or serviceAccount",
			"spec",
			"LNKD-021",
			"Add either spec.identities or spec.serviceAccounts")
		return
	}

	// Validate identities
	if hasIdentities {
		v.validateIdentities(result, identities)
	}

	// Validate serviceAccounts
	if hasSAs {
		v.validateServiceAccounts(ctx, result, serviceAccounts)
	}
}

func (v *MeshTLSValidator) validateIdentities(result *ValidationResult, identities []string) {
	for i, identity := range identities {
		if identity == "*" {
			result.AddIssue(SeverityWarning,
				"Wildcard identity '*' allows all authenticated services",
				"spec.identities",
				"LNKD-022",
				"Consider restricting to specific identities for better security")
			continue
		}

		// Validate identity format (should be like: service-account.namespace.serviceaccount.identity.linkerd.cluster.local)
		if !isValidIdentityFormat(identity) {
			result.AddIssue(SeverityWarning,
				fmt.Sprintf("Identity '%s' at index %d may not be in the correct format", identity, i),
				fmt.Sprintf("spec.identities[%d]", i),
				"LNKD-023",
				"Identity should follow format: <sa>.<ns>.serviceaccount.identity.linkerd.cluster.local")
		}
	}
}

func (v *MeshTLSValidator) validateServiceAccounts(ctx context.Context, result *ValidationResult, serviceAccounts []interface{}) {
	for i, sa := range serviceAccounts {
		saMap, ok := sa.(map[string]interface{})
		if !ok {
			result.AddIssue(SeverityError,
				fmt.Sprintf("Invalid serviceAccount format at index %d", i),
				fmt.Sprintf("spec.serviceAccounts[%d]", i),
				"LNKD-024",
				"ServiceAccount must have name and namespace fields")
			continue
		}

		name, _, _ := unstructured.NestedString(saMap, "name")
		namespace, _, _ := unstructured.NestedString(saMap, "namespace")

		if name == "" {
			result.AddIssue(SeverityError,
				fmt.Sprintf("Missing serviceAccount name at index %d", i),
				fmt.Sprintf("spec.serviceAccounts[%d].name", i),
				"LNKD-025",
				"Specify the serviceAccount name")
			continue
		}

		if namespace == "" {
			result.AddIssue(SeverityError,
				fmt.Sprintf("Missing serviceAccount namespace at index %d", i),
				fmt.Sprintf("spec.serviceAccounts[%d].namespace", i),
				"LNKD-026",
				"Specify the serviceAccount namespace")
			continue
		}

		// Check if service account exists
		_, err := v.clientset.CoreV1().ServiceAccounts(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			result.AddIssue(SeverityWarning,
				fmt.Sprintf("ServiceAccount '%s' does not exist in namespace '%s'", name, namespace),
				fmt.Sprintf("spec.serviceAccounts[%d]", i),
				"LNKD-027",
				fmt.Sprintf("Create ServiceAccount '%s' in namespace '%s' or verify the reference", name, namespace))
		}
	}
}

func isValidIdentityFormat(identity string) bool {
	// Basic validation: should contain .serviceaccount.identity.linkerd
	return strings.Contains(identity, ".serviceaccount.identity.linkerd")
}

// ValidateAll validates all MeshTLSAuthentication resources in a namespace
func (v *MeshTLSValidator) ValidateAll(ctx context.Context, namespace string) []ValidationResult {
	var results []ValidationResult

	listOptions := metav1.ListOptions{}
	var auths *unstructured.UnstructuredList
	var err error

	if namespace == "" {
		auths, err = v.dynamicClient.Resource(meshTLSAuthGVR).List(ctx, listOptions)
	} else {
		auths, err = v.dynamicClient.Resource(meshTLSAuthGVR).Namespace(namespace).List(ctx, listOptions)
	}

	if err != nil {
		return results
	}

	for i := range auths.Items {
		result := v.Validate(ctx, &auths.Items[i])
		results = append(results, result)
	}

	return results
}
