package validators_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/christianhuening/linkerd-mcp/internal/validation/validators"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

var _ = Describe("ProxyValidator", func() {
	var (
		ctx        context.Context
		validator  *validators.ProxyValidator
		kubeClient *kubefake.Clientset
	)

	BeforeEach(func() {
		ctx = context.Background()
		kubeClient = kubefake.NewSimpleClientset()
		validator = validators.NewProxyValidator(kubeClient)
	})

	Describe("ValidateNamespace", func() {
		Context("with valid proxy injection enabled", func() {
			It("should pass validation", func() {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "prod",
						Annotations: map[string]string{
							"linkerd.io/inject": "enabled",
						},
					},
				}

				result := validator.ValidateNamespace(ctx, ns)

				Expect(result.Valid).To(BeTrue())
				Expect(result.ResourceType).To(Equal("Namespace"))
				Expect(result.Name).To(Equal("prod"))
			})
		})

		Context("with no injection annotation", func() {
			It("should return info message", func() {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
				}

				result := validator.ValidateNamespace(ctx, ns)

				var foundInfo bool
				for _, issue := range result.Issues {
					if issue.Severity == validators.SeverityInfo && issue.Code == "LNKD-P002" {
						foundInfo = true
					}
				}
				Expect(foundInfo).To(BeTrue())
			})
		})

		Context("with invalid injection value", func() {
			It("should return error", func() {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
						Annotations: map[string]string{
							"linkerd.io/inject": "invalid",
						},
					},
				}

				result := validator.ValidateNamespace(ctx, ns)

				Expect(result.Valid).To(BeFalse())
				var foundError bool
				for _, issue := range result.Issues {
					if issue.Severity == validators.SeverityError && issue.Code == "LNKD-P003" {
						foundError = true
					}
				}
				Expect(foundError).To(BeTrue())
			})
		})

		Context("with invalid CPU request format", func() {
			It("should return error", func() {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
						Annotations: map[string]string{
							"linkerd.io/inject":                    "enabled",
							"config.linkerd.io/proxy-cpu-request": "invalid",
						},
					},
				}

				result := validator.ValidateNamespace(ctx, ns)

				Expect(result.Valid).To(BeFalse())
				var foundError bool
				for _, issue := range result.Issues {
					if issue.Code == "LNKD-P004" {
						foundError = true
					}
				}
				Expect(foundError).To(BeTrue())
			})
		})

		Context("with valid CPU resource annotations", func() {
			It("should pass validation", func() {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
						Annotations: map[string]string{
							"linkerd.io/inject":                    "enabled",
							"config.linkerd.io/proxy-cpu-request": "100m",
							"config.linkerd.io/proxy-cpu-limit":   "1",
						},
					},
				}

				result := validator.ValidateNamespace(ctx, ns)

				Expect(result.Valid).To(BeTrue())
			})
		})

		Context("with CPU limit without request", func() {
			It("should return warning", func() {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
						Annotations: map[string]string{
							"linkerd.io/inject":                  "enabled",
							"config.linkerd.io/proxy-cpu-limit": "1",
						},
					},
				}

				result := validator.ValidateNamespace(ctx, ns)

				var foundWarning bool
				for _, issue := range result.Issues {
					if issue.Code == "LNKD-P006" {
						foundWarning = true
					}
				}
				Expect(foundWarning).To(BeTrue())
			})
		})

		Context("with CPU limit lower than request", func() {
			It("should return error", func() {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
						Annotations: map[string]string{
							"linkerd.io/inject":                    "enabled",
							"config.linkerd.io/proxy-cpu-request": "1",
							"config.linkerd.io/proxy-cpu-limit":   "500m",
						},
					},
				}

				result := validator.ValidateNamespace(ctx, ns)

				Expect(result.Valid).To(BeFalse())
				var foundError bool
				for _, issue := range result.Issues {
					if issue.Code == "LNKD-P007" {
						foundError = true
					}
				}
				Expect(foundError).To(BeTrue())
			})
		})

		Context("with valid memory annotations", func() {
			It("should pass validation", func() {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
						Annotations: map[string]string{
							"linkerd.io/inject":                       "enabled",
							"config.linkerd.io/proxy-memory-request": "64Mi",
							"config.linkerd.io/proxy-memory-limit":   "128Mi",
						},
					},
				}

				result := validator.ValidateNamespace(ctx, ns)

				Expect(result.Valid).To(BeTrue())
			})
		})

		Context("with invalid log level", func() {
			It("should return error", func() {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
						Annotations: map[string]string{
							"linkerd.io/inject":                   "enabled",
							"config.linkerd.io/proxy-log-level": "verbose",
						},
					},
				}

				result := validator.ValidateNamespace(ctx, ns)

				Expect(result.Valid).To(BeFalse())
				var foundError bool
				for _, issue := range result.Issues {
					if issue.Code == "LNKD-P012" {
						foundError = true
					}
				}
				Expect(foundError).To(BeTrue())
			})
		})

		Context("with debug log level", func() {
			It("should return warning", func() {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
						Annotations: map[string]string{
							"linkerd.io/inject":                   "enabled",
							"config.linkerd.io/proxy-log-level": "debug",
						},
					},
				}

				result := validator.ValidateNamespace(ctx, ns)

				var foundWarning bool
				for _, issue := range result.Issues {
					if issue.Code == "LNKD-P013" {
						foundWarning = true
					}
				}
				Expect(foundWarning).To(BeTrue())
			})
		})

		Context("with invalid proxy version format", func() {
			It("should return warning", func() {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
						Annotations: map[string]string{
							"linkerd.io/inject":                  "enabled",
							"config.linkerd.io/proxy-version": "v2.14.0",
						},
					},
				}

				result := validator.ValidateNamespace(ctx, ns)

				var foundWarning bool
				for _, issue := range result.Issues {
					if issue.Code == "LNKD-P014" {
						foundWarning = true
					}
				}
				Expect(foundWarning).To(BeTrue())
			})
		})

		Context("with valid proxy version", func() {
			It("should pass validation", func() {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
						Annotations: map[string]string{
							"linkerd.io/inject":                  "enabled",
							"config.linkerd.io/proxy-version": "stable-2.14.0",
						},
					},
				}

				result := validator.ValidateNamespace(ctx, ns)

				Expect(result.Valid).To(BeTrue())
			})
		})

		Context("with invalid wait-before-exit value", func() {
			It("should return error", func() {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
						Annotations: map[string]string{
							"linkerd.io/inject": "enabled",
							"config.alpha.linkerd.io/proxy-wait-before-exit-seconds": "invalid",
						},
					},
				}

				result := validator.ValidateNamespace(ctx, ns)

				Expect(result.Valid).To(BeFalse())
				var foundError bool
				for _, issue := range result.Issues {
					if issue.Code == "LNKD-P015" {
						foundError = true
					}
				}
				Expect(foundError).To(BeTrue())
			})
		})

		Context("with very long wait time", func() {
			It("should return warning", func() {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
						Annotations: map[string]string{
							"linkerd.io/inject": "enabled",
							"config.alpha.linkerd.io/proxy-wait-before-exit-seconds": "600",
						},
					},
				}

				result := validator.ValidateNamespace(ctx, ns)

				var foundWarning bool
				for _, issue := range result.Issues {
					if issue.Code == "LNKD-P016" {
						foundWarning = true
					}
				}
				Expect(foundWarning).To(BeTrue())
			})
		})
	})

	Describe("ValidatePod", func() {
		Context("with valid pod having linkerd proxy", func() {
			It("should pass validation", func() {
				pod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
						Annotations: map[string]string{
							"linkerd.io/inject": "enabled",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "app"},
							{Name: "linkerd-proxy"},
						},
					},
				}

				result := validator.ValidatePod(ctx, pod)

				Expect(result.Valid).To(BeTrue())
				Expect(result.ResourceType).To(Equal("Pod"))
			})
		})

		Context("with pod marked for injection but no proxy", func() {
			It("should return warning", func() {
				pod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
						Annotations: map[string]string{
							"linkerd.io/inject": "enabled",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "app"},
						},
					},
				}

				result := validator.ValidatePod(ctx, pod)

				var foundWarning bool
				for _, issue := range result.Issues {
					if issue.Code == "LNKD-P001" {
						foundWarning = true
					}
				}
				Expect(foundWarning).To(BeTrue())
			})
		})
	})

	Describe("ValidateAllNamespaces", func() {
		It("should validate all namespaces", func() {
			ns1 := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "prod",
					Annotations: map[string]string{
						"linkerd.io/inject": "enabled",
					},
				},
			}
			ns2 := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "staging",
					Annotations: map[string]string{
						"linkerd.io/inject": "disabled",
					},
				},
			}

			_, err := kubeClient.CoreV1().Namespaces().Create(ctx, ns1, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			_, err = kubeClient.CoreV1().Namespaces().Create(ctx, ns2, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			results := validator.ValidateAllNamespaces(ctx)

			Expect(results).To(HaveLen(2))
			Expect(results[0].ResourceType).To(Equal("Namespace"))
			Expect(results[1].ResourceType).To(Equal("Namespace"))
		})
	})
})
