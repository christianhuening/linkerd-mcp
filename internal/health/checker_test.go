package health_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/christianhuening/linkerd-mcp/internal/health"
	"github.com/christianhuening/linkerd-mcp/internal/testutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/fake"
)

var _ = Describe("Checker", func() {
	var (
		ctx       context.Context
		clientset *fake.Clientset
		checker   *health.Checker
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("NewChecker", func() {
		It("should create a new checker with clientset", func() {
			clientset = fake.NewSimpleClientset()
			checker = health.NewChecker(clientset)

			Expect(checker).NotTo(BeNil())
		})
	})

	Describe("CheckMeshHealth", func() {
		Context("when control plane is healthy", func() {
			BeforeEach(func() {
				clientset = fake.NewSimpleClientset(
					testutil.CreateLinkerdControlPlanePod("destination-1", "linkerd", "destination", corev1.PodRunning, true),
					testutil.CreateLinkerdControlPlanePod("identity-1", "linkerd", "identity", corev1.PodRunning, true),
					testutil.CreateLinkerdControlPlanePod("proxy-injector-1", "linkerd", "proxy-injector", corev1.PodRunning, true),
				)
				checker = health.NewChecker(clientset)
			})

			It("should return healthy status for all pods", func() {
				result, err := checker.CheckMeshHealth(ctx, "linkerd")
				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())

				var healthStatus map[string]interface{}
				err = testutil.ParseJSONResult(result, &healthStatus)
				Expect(err).NotTo(HaveOccurred())

				Expect(healthStatus["namespace"]).To(Equal("linkerd"))
				Expect(healthStatus["totalPods"]).To(BeNumerically("==", 3))
				Expect(healthStatus["healthyPods"]).To(BeNumerically("==", 3))
				Expect(healthStatus["unhealthyPods"]).To(BeNumerically("==", 0))
			})
		})

		Context("when control plane has unhealthy pods", func() {
			BeforeEach(func() {
				clientset = fake.NewSimpleClientset(
					testutil.CreateLinkerdControlPlanePod("destination-1", "linkerd", "destination", corev1.PodRunning, true),
					testutil.CreateLinkerdControlPlanePod("identity-1", "linkerd", "identity", corev1.PodFailed, false),
					testutil.CreateLinkerdControlPlanePod("proxy-injector-1", "linkerd", "proxy-injector", corev1.PodPending, false),
				)
				checker = health.NewChecker(clientset)
			})

			It("should return mixed health status", func() {
				result, err := checker.CheckMeshHealth(ctx, "linkerd")
				Expect(err).NotTo(HaveOccurred())

				var healthStatus map[string]interface{}
				err = testutil.ParseJSONResult(result, &healthStatus)
				Expect(err).NotTo(HaveOccurred())

				Expect(healthStatus["healthyPods"]).To(BeNumerically("==", 1))
				Expect(healthStatus["unhealthyPods"]).To(BeNumerically("==", 2))

				components := healthStatus["components"].([]interface{})
				Expect(components).To(HaveLen(3))

				// Verify failed pod is marked as unhealthy
				foundFailedPod := false
				for _, comp := range components {
					component := comp.(map[string]interface{})
					if component["name"] == "identity-1" {
						foundFailedPod = true
						Expect(component["healthy"]).To(BeFalse())
						Expect(component["status"]).To(Equal("Failed"))
					}
				}
				Expect(foundFailedPod).To(BeTrue(), "Failed pod should be found in components")
			})
		})

		Context("when namespace is empty", func() {
			BeforeEach(func() {
				clientset = fake.NewSimpleClientset(
					testutil.CreateLinkerdControlPlanePod("destination-1", "linkerd", "destination", corev1.PodRunning, true),
				)
				checker = health.NewChecker(clientset)
			})

			It("should default to linkerd namespace", func() {
				result, err := checker.CheckMeshHealth(ctx, "")
				Expect(err).NotTo(HaveOccurred())

				var healthStatus map[string]interface{}
				err = testutil.ParseJSONResult(result, &healthStatus)
				Expect(err).NotTo(HaveOccurred())

				Expect(healthStatus["namespace"]).To(Equal("linkerd"))
			})
		})

		Context("when there are no control plane pods", func() {
			BeforeEach(func() {
				clientset = fake.NewSimpleClientset()
				checker = health.NewChecker(clientset)
			})

			It("should return zero pod counts", func() {
				result, err := checker.CheckMeshHealth(ctx, "linkerd")
				Expect(err).NotTo(HaveOccurred())

				var healthStatus map[string]interface{}
				err = testutil.ParseJSONResult(result, &healthStatus)
				Expect(err).NotTo(HaveOccurred())

				Expect(healthStatus["totalPods"]).To(BeNumerically("==", 0))
				components := healthStatus["components"].([]interface{})
				Expect(components).To(BeEmpty())
			})
		})

		Context("with custom namespace", func() {
			BeforeEach(func() {
				clientset = fake.NewSimpleClientset(
					testutil.CreateLinkerdControlPlanePod("destination-1", "custom-mesh", "destination", corev1.PodRunning, true),
				)
				checker = health.NewChecker(clientset)
			})

			It("should query the custom namespace", func() {
				result, err := checker.CheckMeshHealth(ctx, "custom-mesh")
				Expect(err).NotTo(HaveOccurred())

				var healthStatus map[string]interface{}
				err = testutil.ParseJSONResult(result, &healthStatus)
				Expect(err).NotTo(HaveOccurred())

				Expect(healthStatus["namespace"]).To(Equal("custom-mesh"))
				Expect(healthStatus["totalPods"]).To(BeNumerically("==", 1))
			})
		})
	})
})
