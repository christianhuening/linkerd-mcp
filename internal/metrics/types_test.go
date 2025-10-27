package metrics_test

import (
	"time"

	"github.com/christianhuening/linkerd-mcp/internal/metrics"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Types", func() {
	Describe("ParseTimeRange", func() {
		Context("with valid duration strings", func() {
			It("should parse 5m correctly", func() {
				tr, err := metrics.ParseTimeRange("5m")

				Expect(err).NotTo(HaveOccurred())
				Expect(tr.End.Sub(tr.Start)).To(Equal(5 * time.Minute))
				Expect(tr.Step).To(Equal(10 * time.Second))
			})

			It("should parse 1h correctly", func() {
				tr, err := metrics.ParseTimeRange("1h")

				Expect(err).NotTo(HaveOccurred())
				Expect(tr.End.Sub(tr.Start)).To(Equal(1 * time.Hour))
				Expect(tr.Step).To(Equal(30 * time.Second))
			})

			It("should parse 24h correctly", func() {
				tr, err := metrics.ParseTimeRange("24h")

				Expect(err).NotTo(HaveOccurred())
				Expect(tr.End.Sub(tr.Start)).To(Equal(24 * time.Hour))
				Expect(tr.Step).To(Equal(5 * time.Minute))
			})

			It("should use default 5m for empty string", func() {
				tr, err := metrics.ParseTimeRange("")

				Expect(err).NotTo(HaveOccurred())
				Expect(tr.End.Sub(tr.Start)).To(Equal(5 * time.Minute))
			})
		})

		Context("with invalid duration strings", func() {
			It("should return error for invalid format", func() {
				_, err := metrics.ParseTimeRange("invalid")

				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("DefaultHealthThresholds", func() {
		It("should return sensible defaults", func() {
			thresholds := metrics.DefaultHealthThresholds()

			Expect(thresholds.ErrorRateWarning).To(Equal(5.0))
			Expect(thresholds.ErrorRateCritical).To(Equal(10.0))
			Expect(thresholds.LatencyP95Warning).To(Equal(1000.0))
			Expect(thresholds.LatencyP95Critical).To(Equal(5000.0))
			Expect(thresholds.SuccessRateWarning).To(Equal(95.0))
			Expect(thresholds.SuccessRateCritical).To(Equal(90.0))
		})
	})

	Describe("HealthStatus constants", func() {
		It("should have correct string values", func() {
			Expect(string(metrics.HealthStatusHealthy)).To(Equal("healthy"))
			Expect(string(metrics.HealthStatusDegraded)).To(Equal("degraded"))
			Expect(string(metrics.HealthStatusUnhealthy)).To(Equal("unhealthy"))
			Expect(string(metrics.HealthStatusUnknown)).To(Equal("unknown"))
		})
	})
})
