package validators_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/christianhuening/linkerd-mcp/internal/testutil"
	"github.com/christianhuening/linkerd-mcp/internal/validation/validators"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
)

var _ = Describe("AuthPolicyValidator", func() {
	var (
		ctx           context.Context
		validator     *validators.AuthPolicyValidator
		dynamicClient *fake.FakeDynamicClient
		serverGVR     schema.GroupVersionResource
		authPolicyGVR schema.GroupVersionResource
		meshTLSGVR    schema.GroupVersionResource
	)

	BeforeEach(func() {
		ctx = context.Background()

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

		meshTLSGVR = schema.GroupVersionResource{
			Group:    "policy.linkerd.io",
			Version:  "v1alpha1",
			Resource: "meshtlsauthentications",
		}

		scheme := runtime.NewScheme()
		gvrToListKind := map[schema.GroupVersionResource]string{
			serverGVR:     "ServerList",
			authPolicyGVR: "AuthorizationPolicyList",
			meshTLSGVR:    "MeshTLSAuthenticationList",
		}

		dynamicClient = fake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToListKind)
		validator = validators.NewAuthPolicyValidator(dynamicClient)
	})

	Describe("Validate", func() {
		Context("with valid authorization policy", func() {
			It("should pass validation", func() {
				// Create target server
				server := testutil.CreateServer("backend-server", "prod", map[string]string{"app": "backend"}, 8080)
				_, err := dynamicClient.Resource(serverGVR).Namespace("prod").Create(ctx, server, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				// Create auth resource
				meshAuth := testutil.CreateMeshTLSAuthentication("frontend-auth", "prod", []string{"*.prod.serviceaccount.identity.linkerd.cluster.local"}, nil)
				_, err = dynamicClient.Resource(meshTLSGVR).Namespace("prod").Create(ctx, meshAuth, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				// Create policy
				policy := testutil.CreateAuthorizationPolicy("allow-frontend", "prod", "backend-server",
					[]map[string]string{{"name": "frontend-auth", "kind": "MeshTLSAuthentication"}})

				result := validator.Validate(ctx, policy)

				Expect(result.Valid).To(BeTrue())
				Expect(result.ResourceType).To(Equal("AuthorizationPolicy"))
				Expect(result.Name).To(Equal("allow-frontend"))
			})
		})

		Context("with missing target server", func() {
			It("should return error", func() {
				policy := testutil.CreateAuthorizationPolicy("orphan-policy", "prod", "nonexistent-server",
					[]map[string]string{{"name": "some-auth", "kind": "MeshTLSAuthentication"}})

				result := validator.Validate(ctx, policy)

				Expect(result.Valid).To(BeFalse())

				var foundError bool
				for _, issue := range result.Issues {
					if issue.Field == "spec.targetRef" && issue.Severity == validators.SeverityError {
						foundError = true
						Expect(issue.Code).To(Equal("LNKD-013"))
					}
				}
				Expect(foundError).To(BeTrue())
			})
		})

		Context("with missing authentication reference", func() {
			It("should return error", func() {
				// Create target server
				server := testutil.CreateServer("backend-server", "prod", map[string]string{"app": "backend"}, 8080)
				_, err := dynamicClient.Resource(serverGVR).Namespace("prod").Create(ctx, server, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				// Create policy with non-existent auth ref
				policy := testutil.CreateAuthorizationPolicy("bad-policy", "prod", "backend-server",
					[]map[string]string{{"name": "nonexistent-auth", "kind": "MeshTLSAuthentication"}})

				result := validator.Validate(ctx, policy)

				Expect(result.Valid).To(BeFalse())

				var foundError bool
				for _, issue := range result.Issues {
					if issue.Severity == validators.SeverityError && issue.Code == "LNKD-019" {
						foundError = true
					}
				}
				Expect(foundError).To(BeTrue())
			})
		})

		Context("with no authentication refs", func() {
			It("should return warning", func() {
				// Create target server
				server := testutil.CreateServer("backend-server", "prod", map[string]string{"app": "backend"}, 8080)
				_, err := dynamicClient.Resource(serverGVR).Namespace("prod").Create(ctx, server, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				// Create policy with no auth refs
				policy := testutil.CreateAuthorizationPolicy("no-auth-policy", "prod", "backend-server", nil)

				result := validator.Validate(ctx, policy)

				var foundWarning bool
				for _, issue := range result.Issues {
					if issue.Severity == validators.SeverityWarning && issue.Code == "LNKD-015" {
						foundWarning = true
					}
				}
				Expect(foundWarning).To(BeTrue())
			})
		})
	})

	Describe("ValidateAll", func() {
		It("should validate all policies in a namespace", func() {
			server := testutil.CreateServer("backend-server", "prod", map[string]string{"app": "backend"}, 8080)
			_, err := dynamicClient.Resource(serverGVR).Namespace("prod").Create(ctx, server, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			policy1 := testutil.CreateAuthorizationPolicy("policy-1", "prod", "backend-server", nil)
			policy2 := testutil.CreateAuthorizationPolicy("policy-2", "prod", "backend-server", nil)

			_, err = dynamicClient.Resource(authPolicyGVR).Namespace("prod").Create(ctx, policy1, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			_, err = dynamicClient.Resource(authPolicyGVR).Namespace("prod").Create(ctx, policy2, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			results := validator.ValidateAll(ctx, "prod")

			Expect(results).To(HaveLen(2))
			Expect(results[0].ResourceType).To(Equal("AuthorizationPolicy"))
			Expect(results[1].ResourceType).To(Equal("AuthorizationPolicy"))
		})
	})
})
