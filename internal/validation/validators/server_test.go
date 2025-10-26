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
	kubefake "k8s.io/client-go/kubernetes/fake"
)

var _ = Describe("ServerValidator", func() {
	var (
		ctx           context.Context
		validator     *validators.ServerValidator
		kubeClient    *kubefake.Clientset
		dynamicClient *fake.FakeDynamicClient
	)

	BeforeEach(func() {
		ctx = context.Background()

		scheme := runtime.NewScheme()
		gvrToListKind := map[schema.GroupVersionResource]string{
			{Group: "policy.linkerd.io", Version: "v1beta3", Resource: "servers"}: "ServerList",
		}

		kubeClient = kubefake.NewSimpleClientset()
		dynamicClient = fake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToListKind)
		validator = validators.NewServerValidator(kubeClient, dynamicClient)
	})

	Describe("Validate", func() {
		Context("with valid server configuration", func() {
			It("should pass validation", func() {
				// Create matching pod
				pod := testutil.CreatePod("backend-1", "prod", "default", map[string]string{"app": "backend"}, "Running", true)
				_, err := kubeClient.CoreV1().Pods("prod").Create(ctx, pod, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				server := testutil.CreateServer("backend-server", "prod", map[string]string{"app": "backend"}, 8080)

				result := validator.Validate(ctx, server)

				Expect(result.Valid).To(BeTrue())
				Expect(result.ResourceType).To(Equal("Server"))
				Expect(result.Name).To(Equal("backend-server"))
				Expect(result.Namespace).To(Equal("prod"))
				Expect(result.Issues).To(BeEmpty())
			})
		})

		Context("with invalid port", func() {
			It("should return error for port out of range", func() {
				server := testutil.CreateServer("bad-server", "prod", map[string]string{"app": "backend"}, 70000)

				result := validator.Validate(ctx, server)

				Expect(result.Valid).To(BeFalse())
				Expect(len(result.Issues)).To(BeNumerically(">", 0))

				var foundPortError bool
				for _, issue := range result.Issues {
					if issue.Field == "spec.port" && issue.Severity == validators.SeverityError {
						foundPortError = true
						Expect(issue.Code).To(Equal("LNKD-006"))
					}
				}
				Expect(foundPortError).To(BeTrue(), "should have port validation error")
			})
		})

		Context("with missing podSelector", func() {
			It("should return error", func() {
				// Create server without podSelector by manipulating the object
				server := testutil.CreateServer("no-selector", "prod", map[string]string{"app": "backend"}, 8080)
				delete(server.Object["spec"].(map[string]interface{}), "podSelector")

				result := validator.Validate(ctx, server)

				Expect(result.Valid).To(BeFalse())

				var foundError bool
				for _, issue := range result.Issues {
					if issue.Field == "spec.podSelector" && issue.Severity == validators.SeverityError {
						foundError = true
						Expect(issue.Code).To(Equal("LNKD-002"))
					}
				}
				Expect(foundError).To(BeTrue())
			})
		})

		Context("with no matching pods", func() {
			It("should return warning", func() {
				server := testutil.CreateServer("orphan-server", "prod", map[string]string{"app": "nonexistent"}, 8080)

				result := validator.Validate(ctx, server)

				var foundWarning bool
				for _, issue := range result.Issues {
					if issue.Severity == validators.SeverityWarning && issue.Code == "LNKD-004" {
						foundWarning = true
					}
				}
				Expect(foundWarning).To(BeTrue(), "should have warning for no matching pods")
			})
		})

		Context("with empty podSelector", func() {
			It("should return warning", func() {
				server := testutil.CreateServer("empty-selector", "prod", map[string]string{}, 8080)

				result := validator.Validate(ctx, server)

				var foundWarning bool
				for _, issue := range result.Issues {
					if issue.Field == "spec.podSelector.matchLabels" && issue.Severity == validators.SeverityWarning {
						foundWarning = true
						Expect(issue.Code).To(Equal("LNKD-003"))
					}
				}
				Expect(foundWarning).To(BeTrue())
			})
		})

		Context("with conflicting servers", func() {
			It("should detect port conflicts", func() {
				// Create first server
				server1 := testutil.CreateServer("server-1", "prod", map[string]string{"app": "backend"}, 8080)
				_, err := dynamicClient.Resource(schema.GroupVersionResource{
					Group:    "policy.linkerd.io",
					Version:  "v1beta3",
					Resource: "servers",
				}).Namespace("prod").Create(ctx, server1, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				// Create conflicting server with same port and labels
				server2 := testutil.CreateServer("server-2", "prod", map[string]string{"app": "backend"}, 8080)

				result := validator.Validate(ctx, server2)

				var foundConflict bool
				for _, issue := range result.Issues {
					if issue.Severity == validators.SeverityError && issue.Code == "LNKD-008" {
						foundConflict = true
					}
				}
				Expect(foundConflict).To(BeTrue(), "should detect conflict")
			})
		})
	})

	Describe("ValidateAll", func() {
		It("should validate all servers in a namespace", func() {
			server1 := testutil.CreateServer("server-1", "prod", map[string]string{"app": "backend"}, 8080)
			server2 := testutil.CreateServer("server-2", "prod", map[string]string{"app": "frontend"}, 8081)

			_, err := dynamicClient.Resource(schema.GroupVersionResource{
				Group:    "policy.linkerd.io",
				Version:  "v1beta3",
				Resource: "servers",
			}).Namespace("prod").Create(ctx, server1, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			_, err = dynamicClient.Resource(schema.GroupVersionResource{
				Group:    "policy.linkerd.io",
				Version:  "v1beta3",
				Resource: "servers",
			}).Namespace("prod").Create(ctx, server2, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			results := validator.ValidateAll(ctx, "prod")

			Expect(results).To(HaveLen(2))
			Expect(results[0].ResourceType).To(Equal("Server"))
			Expect(results[1].ResourceType).To(Equal("Server"))
		})
	})
})
