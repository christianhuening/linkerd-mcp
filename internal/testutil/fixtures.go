package testutil

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// CreatePod creates a test pod with specified parameters
func CreatePod(name, namespace, serviceAccount string, labels map[string]string, phase corev1.PodPhase, ready bool) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: serviceAccount,
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "app:latest",
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: phase,
		},
	}

	if ready {
		pod.Status.Conditions = []corev1.PodCondition{
			{
				Type:   corev1.PodReady,
				Status: corev1.ConditionTrue,
			},
		}
	} else {
		pod.Status.Conditions = []corev1.PodCondition{
			{
				Type:   corev1.PodReady,
				Status: corev1.ConditionFalse,
			},
		}
	}

	return pod
}

// CreateMeshedPod creates a pod with Linkerd proxy injected
func CreateMeshedPod(name, namespace, app string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app": app,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "app:latest",
				},
				{
					Name:  "linkerd-proxy",
					Image: "cr.l5d.io/linkerd/proxy:stable-2.14.0",
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}
}

// CreateLinkerdControlPlanePod creates a Linkerd control plane pod
func CreateLinkerdControlPlanePod(name, namespace, component string, phase corev1.PodPhase, ready bool) *corev1.Pod {
	labels := map[string]string{
		"linkerd.io/control-plane-component": component,
	}
	return CreatePod(name, namespace, "default", labels, phase, ready)
}

// CreateServer creates a Linkerd Server CRD
func CreateServer(name, namespace string, podLabels map[string]string, port int64) *unstructured.Unstructured {
	// Convert podLabels to map[string]interface{} for proper deep copy support
	matchLabels := make(map[string]interface{})
	for k, v := range podLabels {
		matchLabels[k] = v
	}

	server := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "policy.linkerd.io/v1beta3",
			"kind":       "Server",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"podSelector": map[string]interface{}{
					"matchLabels": matchLabels,
				},
				"port": port,
			},
		},
	}
	return server
}

// CreateAuthorizationPolicy creates a Linkerd AuthorizationPolicy CRD
func CreateAuthorizationPolicy(name, namespace, targetServer string, authRefs []map[string]string) *unstructured.Unstructured {
	requiredAuths := []interface{}{}
	for _, ref := range authRefs {
		requiredAuths = append(requiredAuths, map[string]interface{}{
			"name": ref["name"],
			"kind": ref["kind"],
		})
	}

	policy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "policy.linkerd.io/v1alpha1",
			"kind":       "AuthorizationPolicy",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"targetRef": map[string]interface{}{
					"kind": "Server",
					"name": targetServer,
				},
				"requiredAuthenticationRefs": requiredAuths,
			},
		},
	}
	return policy
}

// CreateMeshTLSAuthentication creates a MeshTLSAuthentication CRD
func CreateMeshTLSAuthentication(name, namespace string, identities []string, serviceAccounts []map[string]string) *unstructured.Unstructured {
	spec := map[string]interface{}{}

	if len(identities) > 0 {
		identityList := []interface{}{}
		for _, id := range identities {
			identityList = append(identityList, id)
		}
		spec["identities"] = identityList
	}

	if len(serviceAccounts) > 0 {
		saList := []interface{}{}
		for _, sa := range serviceAccounts {
			saMap := map[string]interface{}{
				"name": sa["name"],
			}
			if ns, ok := sa["namespace"]; ok {
				saMap["namespace"] = ns
			}
			saList = append(saList, saMap)
		}
		spec["serviceAccounts"] = saList
	}

	auth := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "policy.linkerd.io/v1alpha1",
			"kind":       "MeshTLSAuthentication",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": spec,
		},
	}
	return auth
}

// CreateNetworkAuthentication creates a NetworkAuthentication CRD
func CreateNetworkAuthentication(name, namespace string, networks []map[string]interface{}) *unstructured.Unstructured {
	networkList := []interface{}{}
	for _, net := range networks {
		networkList = append(networkList, net)
	}

	auth := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "policy.linkerd.io/v1alpha1",
			"kind":       "NetworkAuthentication",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"networks": networkList,
			},
		},
	}
	return auth
}

// ToRuntimeObject converts unstructured to runtime.Object
func ToRuntimeObject(u *unstructured.Unstructured) runtime.Object {
	return u
}
