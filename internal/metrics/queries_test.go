package metrics_test

import (
	"time"

	"github.com/christianhuening/linkerd-mcp/internal/metrics"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("QueryBuilder", func() {
	var qb *metrics.QueryBuilder

	BeforeEach(func() {
		qb = metrics.NewQueryBuilder("linkerd")
	})

	Describe("BuildServiceRequestRateQuery", func() {
		It("should build correct PromQL query", func() {
			query := qb.BuildServiceRequestRateQuery("frontend", "default", 5*time.Minute)

			Expect(query).To(ContainSubstring(`deployment="frontend"`))
			Expect(query).To(ContainSubstring(`namespace="default"`))
			Expect(query).To(ContainSubstring(`direction="inbound"`))
			Expect(query).To(ContainSubstring("[5m]"))
		})

		It("should use default namespace if empty", func() {
			query := qb.BuildServiceRequestRateQuery("frontend", "", 5*time.Minute)

			Expect(query).To(ContainSubstring(`namespace="linkerd"`))
		})
	})

	Describe("BuildServiceSuccessRateQuery", func() {
		It("should build correct PromQL query", func() {
			query := qb.BuildServiceSuccessRateQuery("backend", "prod", 10*time.Minute)

			Expect(query).To(ContainSubstring(`deployment="backend"`))
			Expect(query).To(ContainSubstring(`namespace="prod"`))
			Expect(query).To(ContainSubstring(`classification!="failure"`))
			Expect(query).To(ContainSubstring("[10m]"))
		})
	})

	Describe("BuildServiceErrorRateQuery", func() {
		It("should build correct PromQL query", func() {
			query := qb.BuildServiceErrorRateQuery("api", "default", 5*time.Minute)

			Expect(query).To(ContainSubstring(`deployment="api"`))
			Expect(query).To(ContainSubstring(`classification="failure"`))
		})
	})

	Describe("BuildServiceLatencyQuery", func() {
		It("should build correct PromQL query for p95", func() {
			query := qb.BuildServiceLatencyQuery("frontend", "default", 0.95, 5*time.Minute)

			Expect(query).To(ContainSubstring("histogram_quantile(0.95"))
			Expect(query).To(ContainSubstring(`deployment="frontend"`))
			Expect(query).To(ContainSubstring("response_latency_ms_bucket"))
		})

		It("should build correct PromQL query for p50", func() {
			query := qb.BuildServiceLatencyQuery("backend", "prod", 0.50, 10*time.Minute)

			Expect(query).To(ContainSubstring("histogram_quantile(0.50"))
			Expect(query).To(ContainSubstring(`deployment="backend"`))
		})
	})

	Describe("BuildServiceMeanLatencyQuery", func() {
		It("should build correct PromQL query", func() {
			query := qb.BuildServiceMeanLatencyQuery("api", "default", 5*time.Minute)

			Expect(query).To(ContainSubstring("response_latency_ms_sum"))
			Expect(query).To(ContainSubstring("response_latency_ms_count"))
			Expect(query).To(ContainSubstring(`deployment="api"`))
		})
	})

	Describe("BuildTrafficBetweenServicesQuery", func() {
		It("should build correct PromQL query", func() {
			query := qb.BuildTrafficBetweenServicesQuery("frontend", "default", "backend", "default", 5*time.Minute)

			Expect(query).To(ContainSubstring(`deployment="frontend"`))
			Expect(query).To(ContainSubstring(`dst_deployment="backend"`))
			Expect(query).To(ContainSubstring(`direction="outbound"`))
		})

		It("should handle different namespaces", func() {
			query := qb.BuildTrafficBetweenServicesQuery("api", "prod", "database", "storage", 5*time.Minute)

			Expect(query).To(ContainSubstring(`namespace="prod"`))
			Expect(query).To(ContainSubstring(`dst_namespace="storage"`))
		})
	})

	Describe("BuildTrafficSuccessRateQuery", func() {
		It("should build correct PromQL query", func() {
			query := qb.BuildTrafficSuccessRateQuery("frontend", "default", "api", "default", 5*time.Minute)

			Expect(query).To(ContainSubstring(`deployment="frontend"`))
			Expect(query).To(ContainSubstring(`dst_deployment="api"`))
			Expect(query).To(ContainSubstring(`classification!="failure"`))
		})
	})

	Describe("BuildTrafficLatencyQuery", func() {
		It("should build correct PromQL query", func() {
			query := qb.BuildTrafficLatencyQuery("frontend", "default", "backend", "default", 0.99, 5*time.Minute)

			Expect(query).To(ContainSubstring("histogram_quantile(0.99"))
			Expect(query).To(ContainSubstring(`deployment="frontend"`))
			Expect(query).To(ContainSubstring(`dst_deployment="backend"`))
		})
	})

	Describe("BuildTopDestinationsQuery", func() {
		It("should build correct PromQL query", func() {
			query := qb.BuildTopDestinationsQuery("frontend", "default", 5*time.Minute, 10)

			Expect(query).To(ContainSubstring("topk(10"))
			Expect(query).To(ContainSubstring(`deployment="frontend"`))
			Expect(query).To(ContainSubstring("by (dst_deployment, dst_namespace)"))
		})
	})

	Describe("BuildTopSourcesQuery", func() {
		It("should build correct PromQL query", func() {
			query := qb.BuildTopSourcesQuery("backend", "default", 5*time.Minute, 5)

			Expect(query).To(ContainSubstring("topk(5"))
			Expect(query).To(ContainSubstring(`dst_deployment="backend"`))
			Expect(query).To(ContainSubstring("by (deployment, namespace)"))
		})
	})

	Describe("BuildErrorsByStatusQuery", func() {
		It("should build correct PromQL query", func() {
			query := qb.BuildErrorsByStatusQuery("api", "default", 5*time.Minute)

			Expect(query).To(ContainSubstring(`deployment="api"`))
			Expect(query).To(ContainSubstring(`http_status=~"5.."`))
			Expect(query).To(ContainSubstring("by (http_status)"))
		})
	})

	Describe("BuildTrafficErrorsByStatusQuery", func() {
		It("should build correct PromQL query", func() {
			query := qb.BuildTrafficErrorsByStatusQuery("frontend", "default", "api", "default", 5*time.Minute)

			Expect(query).To(ContainSubstring(`deployment="frontend"`))
			Expect(query).To(ContainSubstring(`dst_deployment="api"`))
			Expect(query).To(ContainSubstring(`http_status=~"5.."`))
		})
	})

	Describe("BuildAllServicesQuery", func() {
		It("should build correct PromQL query", func() {
			query := qb.BuildAllServicesQuery("default")

			Expect(query).To(ContainSubstring(`namespace="default"`))
			Expect(query).To(ContainSubstring("by (deployment)"))
		})

		It("should use default namespace if empty", func() {
			query := qb.BuildAllServicesQuery("")

			Expect(query).To(ContainSubstring(`namespace="linkerd"`))
		})
	})

	Describe("BuildByteSentQuery", func() {
		It("should build correct PromQL query", func() {
			query := qb.BuildByteSentQuery("frontend", "default", "backend", "default", 5*time.Minute)

			Expect(query).To(ContainSubstring("request_bytes_total"))
			Expect(query).To(ContainSubstring(`deployment="frontend"`))
			Expect(query).To(ContainSubstring(`dst_deployment="backend"`))
		})
	})

	Describe("BuildByteReceivedQuery", func() {
		It("should build correct PromQL query", func() {
			query := qb.BuildByteReceivedQuery("frontend", "default", "backend", "default", 5*time.Minute)

			Expect(query).To(ContainSubstring("response_bytes_total"))
			Expect(query).To(ContainSubstring(`deployment="frontend"`))
			Expect(query).To(ContainSubstring(`dst_deployment="backend"`))
		})
	})
})
