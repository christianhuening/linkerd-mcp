package mesh_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/christianhuening/linkerd-mcp/internal/mesh"
	"github.com/christianhuening/linkerd-mcp/internal/testutil"
	"k8s.io/client-go/kubernetes/fake"
)

var _ = Describe("ServiceLister", func() {
	var (
		ctx       context.Context
		clientset *fake.Clientset
		lister    *mesh.ServiceLister
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("NewServiceLister", func() {
		It("should create a new service lister with clientset", func() {
			clientset = fake.NewSimpleClientset()
			lister = mesh.NewServiceLister(clientset)

			Expect(lister).NotTo(BeNil())
		})
	})

	Describe("ListMeshedServices", func() {
		Context("with meshed pods", func() {
			BeforeEach(func() {
				clientset = fake.NewSimpleClientset(
					testutil.CreateMeshedPod("frontend-1", "prod", "frontend"),
					testutil.CreateMeshedPod("frontend-2", "prod", "frontend"),
					testutil.CreateMeshedPod("backend-1", "prod", "backend"),
					testutil.CreateMeshedPod("api-1", "staging", "api"),
				)
				lister = mesh.NewServiceLister(clientset)
			})

			It("should list all meshed services across namespaces", func() {
				result, err := lister.ListMeshedServices(ctx, "")
				Expect(err).NotTo(HaveOccurred())

				var response map[string]interface{}
				err = testutil.ParseJSONResult(result, &response)
				Expect(err).NotTo(HaveOccurred())

				Expect(response["totalServices"]).To(BeNumerically("==", 3))

				services := response["services"].(map[string]interface{})

				// Check frontend service
				frontendService := services["prod/frontend"].(map[string]interface{})
				Expect(frontendService["namespace"]).To(Equal("prod"))
				Expect(frontendService["service"]).To(Equal("frontend"))
				frontendPods := frontendService["pods"].([]interface{})
				Expect(frontendPods).To(HaveLen(2))

				// Check backend service
				backendService := services["prod/backend"].(map[string]interface{})
				backendPods := backendService["pods"].([]interface{})
				Expect(backendPods).To(HaveLen(1))

				// Check api service in staging
				_, ok := services["staging/api"]
				Expect(ok).To(BeTrue(), "staging/api service should exist")
			})
		})

		Context("when filtering by namespace", func() {
			BeforeEach(func() {
				clientset = fake.NewSimpleClientset(
					testutil.CreateMeshedPod("frontend-1", "prod", "frontend"),
					testutil.CreateMeshedPod("api-1", "staging", "api"),
				)
				lister = mesh.NewServiceLister(clientset)
			})

			It("should only return services in the specified namespace", func() {
				result, err := lister.ListMeshedServices(ctx, "prod")
				Expect(err).NotTo(HaveOccurred())

				var response map[string]interface{}
				err = testutil.ParseJSONResult(result, &response)
				Expect(err).NotTo(HaveOccurred())

				Expect(response["totalServices"]).To(BeNumerically("==", 1))

				services := response["services"].(map[string]interface{})
				_, prodExists := services["prod/frontend"]
				Expect(prodExists).To(BeTrue(), "prod/frontend should exist")

				_, stagingExists := services["staging/api"]
				Expect(stagingExists).To(BeFalse(), "staging/api should not exist when filtering by prod namespace")
			})
		})

		Context("with no meshed pods", func() {
			BeforeEach(func() {
				regularPod := testutil.CreatePod("app-1", "default", "default", map[string]string{"app": "myapp"}, "Running", true)
				clientset = fake.NewSimpleClientset(regularPod)
				lister = mesh.NewServiceLister(clientset)
			})

			It("should return zero services", func() {
				result, err := lister.ListMeshedServices(ctx, "")
				Expect(err).NotTo(HaveOccurred())

				var response map[string]interface{}
				err = testutil.ParseJSONResult(result, &response)
				Expect(err).NotTo(HaveOccurred())

				Expect(response["totalServices"]).To(BeNumerically("==", 0))
			})
		})

		Context("with pods without app label", func() {
			BeforeEach(func() {
				podWithoutLabel := testutil.CreateMeshedPod("no-label-1", "default", "")
				podWithoutLabel.Labels = map[string]string{}
				clientset = fake.NewSimpleClientset(podWithoutLabel)
				lister = mesh.NewServiceLister(clientset)
			})

			It("should not list pods without app labels", func() {
				result, err := lister.ListMeshedServices(ctx, "")
				Expect(err).NotTo(HaveOccurred())

				var response map[string]interface{}
				err = testutil.ParseJSONResult(result, &response)
				Expect(err).NotTo(HaveOccurred())

				Expect(response["totalServices"]).To(BeNumerically("==", 0))
			})
		})

		Context("with k8s-app label", func() {
			BeforeEach(func() {
				pod := testutil.CreateMeshedPod("kube-pod-1", "kube-system", "")
				pod.Labels = map[string]string{"k8s-app": "kube-dns"}
				clientset = fake.NewSimpleClientset(pod)
				lister = mesh.NewServiceLister(clientset)
			})

			It("should recognize k8s-app label as service name", func() {
				result, err := lister.ListMeshedServices(ctx, "")
				Expect(err).NotTo(HaveOccurred())

				var response map[string]interface{}
				err = testutil.ParseJSONResult(result, &response)
				Expect(err).NotTo(HaveOccurred())

				Expect(response["totalServices"]).To(BeNumerically("==", 1))

				services := response["services"].(map[string]interface{})
				service := services["kube-system/kube-dns"].(map[string]interface{})
				Expect(service["service"]).To(Equal("kube-dns"))
			})
		})

		Context("with multiple pods per service", func() {
			BeforeEach(func() {
				clientset = fake.NewSimpleClientset(
					testutil.CreateMeshedPod("web-1", "prod", "web"),
					testutil.CreateMeshedPod("web-2", "prod", "web"),
					testutil.CreateMeshedPod("web-3", "prod", "web"),
				)
				lister = mesh.NewServiceLister(clientset)
			})

			It("should aggregate all pods under the same service", func() {
				result, err := lister.ListMeshedServices(ctx, "prod")
				Expect(err).NotTo(HaveOccurred())

				var response map[string]interface{}
				err = testutil.ParseJSONResult(result, &response)
				Expect(err).NotTo(HaveOccurred())

				services := response["services"].(map[string]interface{})
				service := services["prod/web"].(map[string]interface{})
				pods := service["pods"].([]interface{})
				Expect(pods).To(HaveLen(3))

				// Verify all pod names are present
				podNames := make(map[string]bool)
				for _, pod := range pods {
					podNames[pod.(string)] = true
				}
				Expect(podNames).To(HaveKey("web-1"))
				Expect(podNames).To(HaveKey("web-2"))
				Expect(podNames).To(HaveKey("web-3"))
			})
		})
	})
})
