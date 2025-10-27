package metrics

import (
	"fmt"
	"time"
)

// QueryBuilder helps construct PromQL queries for Linkerd metrics
type QueryBuilder struct {
	namespace string
}

// NewQueryBuilder creates a new query builder
func NewQueryBuilder(namespace string) *QueryBuilder {
	return &QueryBuilder{namespace: namespace}
}

// BuildServiceRequestRateQuery builds a query for service request rate (requests/sec)
// Measures inbound requests to the service
func (qb *QueryBuilder) BuildServiceRequestRateQuery(deployment, namespace string, window time.Duration) string {
	if namespace == "" {
		namespace = qb.namespace
	}
	return fmt.Sprintf(
		`sum(rate(request_total{deployment="%s", namespace="%s", direction="inbound"}[%s]))`,
		deployment, namespace, formatDuration(window),
	)
}

// BuildServiceSuccessRateQuery builds a query for service success rate (0-1)
// Measures the ratio of successful requests (non-failure) to total requests
func (qb *QueryBuilder) BuildServiceSuccessRateQuery(deployment, namespace string, window time.Duration) string {
	if namespace == "" {
		namespace = qb.namespace
	}
	return fmt.Sprintf(
		`sum(rate(response_total{deployment="%s", namespace="%s", classification!="failure", direction="inbound"}[%s])) / sum(rate(response_total{deployment="%s", namespace="%s", direction="inbound"}[%s]))`,
		deployment, namespace, formatDuration(window),
		deployment, namespace, formatDuration(window),
	)
}

// BuildServiceErrorRateQuery builds a query for service error rate (0-1)
// Measures the ratio of failed requests to total requests
func (qb *QueryBuilder) BuildServiceErrorRateQuery(deployment, namespace string, window time.Duration) string {
	if namespace == "" {
		namespace = qb.namespace
	}
	return fmt.Sprintf(
		`sum(rate(response_total{deployment="%s", namespace="%s", classification="failure", direction="inbound"}[%s])) / sum(rate(response_total{deployment="%s", namespace="%s", direction="inbound"}[%s]))`,
		deployment, namespace, formatDuration(window),
		deployment, namespace, formatDuration(window),
	)
}

// BuildServiceLatencyQuery builds a query for service latency at a given quantile
// quantile should be between 0 and 1 (e.g., 0.95 for p95)
func (qb *QueryBuilder) BuildServiceLatencyQuery(deployment, namespace string, quantile float64, window time.Duration) string {
	if namespace == "" {
		namespace = qb.namespace
	}
	return fmt.Sprintf(
		`histogram_quantile(%.2f, sum(rate(response_latency_ms_bucket{deployment="%s", namespace="%s", direction="inbound"}[%s])) by (le))`,
		quantile, deployment, namespace, formatDuration(window),
	)
}

// BuildServiceMeanLatencyQuery builds a query for mean latency
func (qb *QueryBuilder) BuildServiceMeanLatencyQuery(deployment, namespace string, window time.Duration) string {
	if namespace == "" {
		namespace = qb.namespace
	}
	return fmt.Sprintf(
		`sum(rate(response_latency_ms_sum{deployment="%s", namespace="%s", direction="inbound"}[%s])) / sum(rate(response_latency_ms_count{deployment="%s", namespace="%s", direction="inbound"}[%s]))`,
		deployment, namespace, formatDuration(window),
		deployment, namespace, formatDuration(window),
	)
}

// BuildTrafficBetweenServicesQuery builds a query for traffic from source to target
func (qb *QueryBuilder) BuildTrafficBetweenServicesQuery(srcDeployment, srcNamespace, dstDeployment, dstNamespace string, window time.Duration) string {
	if srcNamespace == "" {
		srcNamespace = qb.namespace
	}
	if dstNamespace == "" {
		dstNamespace = qb.namespace
	}
	return fmt.Sprintf(
		`sum(rate(request_total{deployment="%s", namespace="%s", dst_deployment="%s", dst_namespace="%s", direction="outbound"}[%s]))`,
		srcDeployment, srcNamespace, dstDeployment, dstNamespace, formatDuration(window),
	)
}

// BuildTrafficSuccessRateQuery builds a query for success rate between services
func (qb *QueryBuilder) BuildTrafficSuccessRateQuery(srcDeployment, srcNamespace, dstDeployment, dstNamespace string, window time.Duration) string {
	if srcNamespace == "" {
		srcNamespace = qb.namespace
	}
	if dstNamespace == "" {
		dstNamespace = qb.namespace
	}
	return fmt.Sprintf(
		`sum(rate(response_total{deployment="%s", namespace="%s", dst_deployment="%s", dst_namespace="%s", classification!="failure", direction="outbound"}[%s])) / sum(rate(response_total{deployment="%s", namespace="%s", dst_deployment="%s", dst_namespace="%s", direction="outbound"}[%s]))`,
		srcDeployment, srcNamespace, dstDeployment, dstNamespace, formatDuration(window),
		srcDeployment, srcNamespace, dstDeployment, dstNamespace, formatDuration(window),
	)
}

// BuildTrafficLatencyQuery builds a query for latency between services
func (qb *QueryBuilder) BuildTrafficLatencyQuery(srcDeployment, srcNamespace, dstDeployment, dstNamespace string, quantile float64, window time.Duration) string {
	if srcNamespace == "" {
		srcNamespace = qb.namespace
	}
	if dstNamespace == "" {
		dstNamespace = qb.namespace
	}
	return fmt.Sprintf(
		`histogram_quantile(%.2f, sum(rate(response_latency_ms_bucket{deployment="%s", namespace="%s", dst_deployment="%s", dst_namespace="%s", direction="outbound"}[%s])) by (le))`,
		quantile, srcDeployment, srcNamespace, dstDeployment, dstNamespace, formatDuration(window),
	)
}

// BuildTopDestinationsQuery builds a query to find top destinations from a source
func (qb *QueryBuilder) BuildTopDestinationsQuery(srcDeployment, srcNamespace string, window time.Duration, limit int) string {
	if srcNamespace == "" {
		srcNamespace = qb.namespace
	}
	return fmt.Sprintf(
		`topk(%d, sum(rate(request_total{deployment="%s", namespace="%s", direction="outbound"}[%s])) by (dst_deployment, dst_namespace))`,
		limit, srcDeployment, srcNamespace, formatDuration(window),
	)
}

// BuildTopSourcesQuery builds a query to find top sources to a destination
func (qb *QueryBuilder) BuildTopSourcesQuery(dstDeployment, dstNamespace string, window time.Duration, limit int) string {
	if dstNamespace == "" {
		dstNamespace = qb.namespace
	}
	return fmt.Sprintf(
		`topk(%d, sum(rate(request_total{dst_deployment="%s", dst_namespace="%s", direction="outbound"}[%s])) by (deployment, namespace))`,
		limit, dstDeployment, dstNamespace, formatDuration(window),
	)
}

// BuildErrorsByStatusQuery builds a query for errors grouped by HTTP status code
func (qb *QueryBuilder) BuildErrorsByStatusQuery(deployment, namespace string, window time.Duration) string {
	if namespace == "" {
		namespace = qb.namespace
	}
	return fmt.Sprintf(
		`sum(rate(response_total{deployment="%s", namespace="%s", direction="inbound", http_status=~"5.."}[%s])) by (http_status)`,
		deployment, namespace, formatDuration(window),
	)
}

// BuildTrafficErrorsByStatusQuery builds a query for errors between services grouped by HTTP status
func (qb *QueryBuilder) BuildTrafficErrorsByStatusQuery(srcDeployment, srcNamespace, dstDeployment, dstNamespace string, window time.Duration) string {
	if srcNamespace == "" {
		srcNamespace = qb.namespace
	}
	if dstNamespace == "" {
		dstNamespace = qb.namespace
	}
	return fmt.Sprintf(
		`sum(rate(response_total{deployment="%s", namespace="%s", dst_deployment="%s", dst_namespace="%s", direction="outbound", http_status=~"5.."}[%s])) by (http_status)`,
		srcDeployment, srcNamespace, dstDeployment, dstNamespace, formatDuration(window),
	)
}

// BuildAllServicesQuery builds a query to find all services in a namespace
func (qb *QueryBuilder) BuildAllServicesQuery(namespace string) string {
	if namespace == "" {
		namespace = qb.namespace
	}
	return fmt.Sprintf(
		`count(request_total{namespace="%s", direction="inbound"}) by (deployment)`,
		namespace,
	)
}

// BuildByteSentQuery builds a query for bytes sent
func (qb *QueryBuilder) BuildByteSentQuery(srcDeployment, srcNamespace, dstDeployment, dstNamespace string, window time.Duration) string {
	if srcNamespace == "" {
		srcNamespace = qb.namespace
	}
	if dstNamespace == "" {
		dstNamespace = qb.namespace
	}
	return fmt.Sprintf(
		`sum(rate(request_bytes_total{deployment="%s", namespace="%s", dst_deployment="%s", dst_namespace="%s", direction="outbound"}[%s]))`,
		srcDeployment, srcNamespace, dstDeployment, dstNamespace, formatDuration(window),
	)
}

// BuildByteReceivedQuery builds a query for bytes received
func (qb *QueryBuilder) BuildByteReceivedQuery(srcDeployment, srcNamespace, dstDeployment, dstNamespace string, window time.Duration) string {
	if srcNamespace == "" {
		srcNamespace = qb.namespace
	}
	if dstNamespace == "" {
		dstNamespace = qb.namespace
	}
	return fmt.Sprintf(
		`sum(rate(response_bytes_total{deployment="%s", namespace="%s", dst_deployment="%s", dst_namespace="%s", direction="outbound"}[%s]))`,
		srcDeployment, srcNamespace, dstDeployment, dstNamespace, formatDuration(window),
	)
}

// formatDuration formats a time.Duration for use in PromQL (e.g., "5m", "1h")
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}
