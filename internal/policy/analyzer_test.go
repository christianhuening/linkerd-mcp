package policy_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/christianhuening/linkerd-mcp/internal/policy"
	"github.com/christianhuening/linkerd-mcp/internal/testutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

var (
	serverGVR = schema.GroupVersionResource{
		Group:    "policy.linkerd.io",
		Version:  "v1beta3",
		Resource: "servers",
	}

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
)

var _ = Describe("Analyzer", func() {
	var (
		ctx           context.Context
		analyzer      *policy.Analyzer
		kubeClient    *kubefake.Clientset
		dynamicClient *fake.FakeDynamicClient
	)

	BeforeEach(func() {
		ctx = context.Background()

		scheme := runtime.NewScheme()
		gvrToListKind := map[schema.GroupVersionResource]string{
			{Group: "policy.linkerd.io", Version: "v1beta3", Resource: "servers"}:                 "ServerList",
			{Group: "policy.linkerd.io", Version: "v1alpha1", Resource: "authorizationpolicies"}:  "AuthorizationPolicyList",
			{Group: "policy.linkerd.io", Version: "v1alpha1", Resource: "meshtlsauthentications"}: "MeshTLSAuthenticationList",
			{Group: "policy.linkerd.io", Version: "v1alpha1", Resource: "networkauthentications"}: "NetworkAuthenticationList",
			{Group: "policy.linkerd.io", Version: "v1alpha1", Resource: "httproutes"}:             "HTTPRouteList",
		}

		kubeClient = kubefake.NewSimpleClientset()
		dynamicClient = fake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToListKind)
		analyzer = policy.NewAnalyzer(kubeClient, dynamicClient)
	})

	Describe("NewAnalyzer", func() {
		It("should create a new analyzer with clients", func() {
			Expect(analyzer).NotTo(BeNil())
		})
	})

	Describe("AnalyzeConnectivity", func() {
		It("should analyze connectivity between services", func() {
			result, err := analyzer.AnalyzeConnectivity(ctx, "prod", "frontend", "prod", "backend")
			Expect(err).NotTo(HaveOccurred())

			var analysis map[string]interface{}
			err = testutil.ParseJSONResult(result, &analysis)
			Expect(err).NotTo(HaveOccurred())

			source := analysis["source"].(map[string]interface{})
			Expect(source["namespace"]).To(Equal("prod"))
			Expect(source["service"]).To(Equal("frontend"))

			target := analysis["target"].(map[string]interface{})
			Expect(target["namespace"]).To(Equal("prod"))
			Expect(target["service"]).To(Equal("backend"))
		})

		Context("when target namespace is empty", func() {
			It("should default to source namespace", func() {
				result, err := analyzer.AnalyzeConnectivity(ctx, "prod", "frontend", "", "backend")
				Expect(err).NotTo(HaveOccurred())

				var analysis map[string]interface{}
				err = testutil.ParseJSONResult(result, &analysis)
				Expect(err).NotTo(HaveOccurred())

				target := analysis["target"].(map[string]interface{})
				Expect(target["namespace"]).To(Equal("prod"))
			})
		})
	})

	Describe("GetAllowedTargets", func() {
		Context("when no pods are found", func() {
			It("should return an error result", func() {
				result, err := analyzer.GetAllowedTargets(ctx, "prod", "nonexistent")
				Expect(err).NotTo(HaveOccurred())
				Expect(result.IsError).To(BeTrue())

				var textContent string
				err = testutil.GetTextFromResult(result, &textContent)
				Expect(err).NotTo(HaveOccurred())
				Expect(textContent).To(Equal("no pods found for service nonexistent in namespace prod"))
			})
		})

		Context("with pods and authorization policies", func() {
			BeforeEach(func() {
				// Add source pod
				pod := testutil.CreatePod("frontend-1", "prod", "frontend-sa", map[string]string{"app": "frontend"}, "Running", true)
				_, err := kubeClient.CoreV1().Pods("prod").Create(ctx, pod, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				// Add Server CRD
				server := testutil.CreateServer("backend-server", "prod", map[string]string{"app": "backend"}, 8080)
				_, err = dynamicClient.Resource(serverGVR).Namespace("prod").Create(ctx, server, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				// Add AuthorizationPolicy
				authPolicy := testutil.CreateAuthorizationPolicy(
					"allow-frontend",
					"prod",
					"backend-server",
					[]map[string]string{{"name": "frontend-auth", "kind": "MeshTLSAuthentication"}},
				)
				_, err = dynamicClient.Resource(authPolicyGVR).Namespace("prod").Create(ctx, authPolicy, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				// Add MeshTLSAuthentication allowing frontend
				meshAuth := testutil.CreateMeshTLSAuthentication(
					"frontend-auth",
					"prod",
					[]string{"frontend-sa.prod.serviceaccount.identity.linkerd.cluster.local"},
					nil,
				)
				_, err = dynamicClient.Resource(meshTLSAuthGVR).Namespace("prod").Create(ctx, meshAuth, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
			})

			It("should return allowed targets for the source service", func() {
				result, err := analyzer.GetAllowedTargets(ctx, "prod", "frontend")
				Expect(err).NotTo(HaveOccurred())

				var response map[string]interface{}
				err = testutil.ParseJSONResult(result, &response)
				Expect(err).NotTo(HaveOccurred())

				source := response["source"].(map[string]interface{})
				Expect(source["serviceAccount"]).To(Equal("frontend-sa"))

				allowedTargets := response["allowedTargets"].([]interface{})
				totalTargets := int(response["totalTargets"].(float64))
				Expect(totalTargets).To(Equal(len(allowedTargets)))
			})
		})
	})

	Describe("GetAllowedSources", func() {
		Context("when no servers are found", func() {
			It("should return a message about no servers", func() {
				result, err := analyzer.GetAllowedSources(ctx, "prod", "backend")
				Expect(err).NotTo(HaveOccurred())

				var textContent string
				err = testutil.GetTextFromResult(result, &textContent)
				Expect(err).NotTo(HaveOccurred())
				Expect(textContent).To(Equal("No Linkerd Servers found for service backend in namespace prod"))
			})
		})

		Context("with servers and wildcard authentication", func() {
			BeforeEach(func() {
				// Add Server for backend
				server := testutil.CreateServer("backend-server", "prod", map[string]string{"app": "backend"}, 8080)
				_, err := dynamicClient.Resource(serverGVR).Namespace("prod").Create(ctx, server, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				// Add AuthorizationPolicy
				authPolicy := testutil.CreateAuthorizationPolicy(
					"allow-all-auth",
					"prod",
					"backend-server",
					[]map[string]string{{"name": "all-auth", "kind": "MeshTLSAuthentication"}},
				)
				_, err = dynamicClient.Resource(authPolicyGVR).Namespace("prod").Create(ctx, authPolicy, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				// Add MeshTLSAuthentication with wildcard
				meshAuth := testutil.CreateMeshTLSAuthentication(
					"all-auth",
					"prod",
					[]string{"*"},
					nil,
				)
				_, err = dynamicClient.Resource(meshTLSAuthGVR).Namespace("prod").Create(ctx, meshAuth, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
			})

			It("should return allowed sources including wildcard", func() {
				result, err := analyzer.GetAllowedSources(ctx, "prod", "backend")
				Expect(err).NotTo(HaveOccurred())

				var response map[string]interface{}
				err = testutil.ParseJSONResult(result, &response)
				Expect(err).NotTo(HaveOccurred())

				matchingServers := response["matchingServers"].([]interface{})
				Expect(matchingServers).To(HaveLen(1))

				allowedSources := response["allowedSources"].([]interface{})
				Expect(len(allowedSources)).To(BeNumerically(">=", 1))

				// Check for wildcard source
				foundWildcard := false
				for _, src := range allowedSources {
					source := src.(map[string]interface{})
					if source["type"] == "wildcard" {
						foundWildcard = true
						Expect(source["description"]).To(Equal("All authenticated services"))
					}
				}
				Expect(foundWildcard).To(BeTrue(), "should find wildcard source")
			})
		})

		Context("with service account authentication", func() {
			BeforeEach(func() {
				// Add Server
				server := testutil.CreateServer("api-server", "prod", map[string]string{"app": "api"}, 8080)
				_, err := dynamicClient.Resource(serverGVR).Namespace("prod").Create(ctx, server, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				// Add AuthorizationPolicy
				authPolicy := testutil.CreateAuthorizationPolicy(
					"allow-frontend",
					"prod",
					"api-server",
					[]map[string]string{{"name": "frontend-auth", "kind": "MeshTLSAuthentication"}},
				)
				_, err = dynamicClient.Resource(authPolicyGVR).Namespace("prod").Create(ctx, authPolicy, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				// Add MeshTLSAuthentication with service accounts
				meshAuth := testutil.CreateMeshTLSAuthentication(
					"frontend-auth",
					"prod",
					nil,
					[]map[string]string{
						{"name": "frontend-sa", "namespace": "prod"},
						{"name": "admin-sa", "namespace": "admin"},
					},
				)
				_, err = dynamicClient.Resource(meshTLSAuthGVR).Namespace("prod").Create(ctx, meshAuth, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
			})

			It("should return service accounts as allowed sources", func() {
				result, err := analyzer.GetAllowedSources(ctx, "prod", "api")
				Expect(err).NotTo(HaveOccurred())

				var response map[string]interface{}
				err = testutil.ParseJSONResult(result, &response)
				Expect(err).NotTo(HaveOccurred())

				allowedSources := response["allowedSources"].([]interface{})
				Expect(allowedSources).To(HaveLen(2))

				// Verify service accounts are present
				foundFrontend := false
				foundAdmin := false
				for _, src := range allowedSources {
					source := src.(map[string]interface{})
					if sa, ok := source["serviceAccount"]; ok {
						if sa == "frontend-sa" {
							foundFrontend = true
							Expect(source["namespace"]).To(Equal("prod"))
						}
						if sa == "admin-sa" {
							foundAdmin = true
							Expect(source["namespace"]).To(Equal("admin"))
						}
					}
				}

				Expect(foundFrontend).To(BeTrue(), "should find frontend-sa in allowed sources")
				Expect(foundAdmin).To(BeTrue(), "should find admin-sa in allowed sources")
			})
		})
	})
})
