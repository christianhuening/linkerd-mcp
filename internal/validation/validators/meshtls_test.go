package validators_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/christianhuening/linkerd-mcp/internal/testutil"
	"github.com/christianhuening/linkerd-mcp/internal/validation/validators"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

var _ = Describe("MeshTLSValidator", func() {
	var (
		ctx           context.Context
		validator     *validators.MeshTLSValidator
		kubeClient    *kubefake.Clientset
		dynamicClient *fake.FakeDynamicClient
		meshTLSGVR    schema.GroupVersionResource
	)

	BeforeEach(func() {
		ctx = context.Background()

		meshTLSGVR = schema.GroupVersionResource{
			Group:    "policy.linkerd.io",
			Version:  "v1alpha1",
			Resource: "meshtlsauthentications",
		}

		scheme := runtime.NewScheme()
		gvrToListKind := map[schema.GroupVersionResource]string{
			meshTLSGVR: "MeshTLSAuthenticationList",
		}

		kubeClient = kubefake.NewSimpleClientset()
		dynamicClient = fake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToListKind)
		validator = validators.NewMeshTLSValidator(kubeClient, dynamicClient)
	})

	Describe("Validate", func() {
		Context("with valid identity configuration", func() {
			It("should pass validation", func() {
				meshAuth := testutil.CreateMeshTLSAuthentication("frontend-auth", "prod",
					[]string{"frontend-sa.prod.serviceaccount.identity.linkerd.cluster.local"}, nil)

				result := validator.Validate(ctx, meshAuth)

				Expect(result.Valid).To(BeTrue())
				Expect(result.ResourceType).To(Equal("MeshTLSAuthentication"))
				Expect(result.Name).To(Equal("frontend-auth"))
			})
		})

		Context("with valid serviceAccount configuration", func() {
			It("should pass validation when service account exists", func() {
				// Create service account
				sa := &corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "frontend-sa",
						Namespace: "prod",
					},
				}
				_, err := kubeClient.CoreV1().ServiceAccounts("prod").Create(ctx, sa, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				meshAuth := testutil.CreateMeshTLSAuthentication("frontend-auth", "prod", nil,
					[]map[string]string{{"name": "frontend-sa", "namespace": "prod"}})

				result := validator.Validate(ctx, meshAuth)

				Expect(result.Valid).To(BeTrue())
			})

			It("should warn when service account does not exist", func() {
				meshAuth := testutil.CreateMeshTLSAuthentication("frontend-auth", "prod", nil,
					[]map[string]string{{"name": "nonexistent-sa", "namespace": "prod"}})

				result := validator.Validate(ctx, meshAuth)

				var foundWarning bool
				for _, issue := range result.Issues {
					if issue.Severity == validators.SeverityWarning && issue.Code == "LNKD-027" {
						foundWarning = true
					}
				}
				Expect(foundWarning).To(BeTrue())
			})
		})

		Context("with wildcard identity", func() {
			It("should return warning", func() {
				meshAuth := testutil.CreateMeshTLSAuthentication("wildcard-auth", "prod",
					[]string{"*"}, nil)

				result := validator.Validate(ctx, meshAuth)

				var foundWarning bool
				for _, issue := range result.Issues {
					if issue.Severity == validators.SeverityWarning && issue.Code == "LNKD-022" {
						foundWarning = true
						Expect(issue.Message).To(ContainSubstring("Wildcard identity"))
					}
				}
				Expect(foundWarning).To(BeTrue())
			})
		})

		Context("with neither identities nor serviceAccounts", func() {
			It("should return error", func() {
				meshAuth := testutil.CreateMeshTLSAuthentication("empty-auth", "prod", nil, nil)

				result := validator.Validate(ctx, meshAuth)

				Expect(result.Valid).To(BeFalse())

				var foundError bool
				for _, issue := range result.Issues {
					if issue.Severity == validators.SeverityError && issue.Code == "LNKD-021" {
						foundError = true
					}
				}
				Expect(foundError).To(BeTrue())
			})
		})

		Context("with missing serviceAccount name", func() {
			It("should return error", func() {
				meshAuth := testutil.CreateMeshTLSAuthentication("bad-sa-auth", "prod", nil,
					[]map[string]string{{"namespace": "prod"}}) // Missing name

				result := validator.Validate(ctx, meshAuth)

				Expect(result.Valid).To(BeFalse())

				var foundError bool
				for _, issue := range result.Issues {
					if issue.Severity == validators.SeverityError && issue.Code == "LNKD-025" {
						foundError = true
					}
				}
				Expect(foundError).To(BeTrue())
			})
		})

		Context("with missing serviceAccount namespace", func() {
			It("should return error", func() {
				meshAuth := testutil.CreateMeshTLSAuthentication("bad-sa-auth", "prod", nil,
					[]map[string]string{{"name": "frontend-sa"}}) // Missing namespace

				result := validator.Validate(ctx, meshAuth)

				Expect(result.Valid).To(BeFalse())

				var foundError bool
				for _, issue := range result.Issues {
					if issue.Severity == validators.SeverityError && issue.Code == "LNKD-026" {
						foundError = true
					}
				}
				Expect(foundError).To(BeTrue())
			})
		})
	})

	Describe("ValidateAll", func() {
		It("should validate all MeshTLS authentications in a namespace", func() {
			auth1 := testutil.CreateMeshTLSAuthentication("auth-1", "prod",
				[]string{"*.prod.serviceaccount.identity.linkerd.cluster.local"}, nil)
			auth2 := testutil.CreateMeshTLSAuthentication("auth-2", "prod",
				[]string{"*.staging.serviceaccount.identity.linkerd.cluster.local"}, nil)

			_, err := dynamicClient.Resource(meshTLSGVR).Namespace("prod").Create(ctx, auth1, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			_, err = dynamicClient.Resource(meshTLSGVR).Namespace("prod").Create(ctx, auth2, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			results := validator.ValidateAll(ctx, "prod")

			Expect(results).To(HaveLen(2))
			Expect(results[0].ResourceType).To(Equal("MeshTLSAuthentication"))
			Expect(results[1].ResourceType).To(Equal("MeshTLSAuthentication"))
		})
	})
})
