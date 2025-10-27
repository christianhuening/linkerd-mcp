package metrics

import (
	"time"
)

// TimeRange represents a time window for querying metrics
type TimeRange struct {
	Start time.Time     // Start of the time range
	End   time.Time     // End of the time range
	Step  time.Duration // Step duration for range queries
}

// ServiceMetrics contains comprehensive metrics for a single service
type ServiceMetrics struct {
	Service         string              `json:"service"`
	Namespace       string              `json:"namespace"`
	Deployment      string              `json:"deployment,omitempty"`
	TimeRange       TimeRange           `json:"timeRange"`
	RequestRate     float64             `json:"requestRate"`     // requests per second
	SuccessRate     float64             `json:"successRate"`     // percentage (0-100)
	ErrorRate       float64             `json:"errorRate"`       // percentage (0-100)
	Latency         LatencyMetrics      `json:"latency"`
	TopDestinations []TrafficFlow       `json:"topDestinations,omitempty"`
	TopSources      []TrafficFlow       `json:"topSources,omitempty"`
	ErrorsByStatus  map[string]int64    `json:"errorsByStatus,omitempty"` // HTTP status code -> count
}

// LatencyMetrics contains latency percentiles
type LatencyMetrics struct {
	P50  float64 `json:"p50"`  // 50th percentile in milliseconds
	P95  float64 `json:"p95"`  // 95th percentile in milliseconds
	P99  float64 `json:"p99"`  // 99th percentile in milliseconds
	Mean float64 `json:"mean"` // mean latency in milliseconds
}

// TrafficMetrics contains metrics for traffic between two services
type TrafficMetrics struct {
	Source         ServiceIdentifier `json:"source"`
	Target         ServiceIdentifier `json:"target"`
	TimeRange      TimeRange         `json:"timeRange"`
	RequestCount   int64             `json:"requestCount"`
	RequestRate    float64           `json:"requestRate"`    // requests per second
	SuccessRate    float64           `json:"successRate"`    // percentage (0-100)
	ErrorRate      float64           `json:"errorRate"`      // percentage (0-100)
	ErrorsByStatus map[string]int64  `json:"errorsByStatus"` // HTTP status code -> count
	LatencyP50     float64           `json:"latencyP50"`     // milliseconds
	LatencyP95     float64           `json:"latencyP95"`     // milliseconds
	LatencyP99     float64           `json:"latencyP99"`     // milliseconds
	BytesSent      int64             `json:"bytesSent,omitempty"`
	BytesReceived  int64             `json:"bytesReceived,omitempty"`
}

// ServiceIdentifier uniquely identifies a service
type ServiceIdentifier struct {
	Service    string `json:"service"`
	Namespace  string `json:"namespace"`
	Deployment string `json:"deployment,omitempty"`
}

// TrafficFlow represents traffic to/from a service
type TrafficFlow struct {
	Service      string  `json:"service"`
	Namespace    string  `json:"namespace"`
	Deployment   string  `json:"deployment,omitempty"`
	RequestCount int64   `json:"requestCount"`
	RequestRate  float64 `json:"requestRate"` // requests per second
}

// ServiceHealthSummary contains health status based on metrics
type ServiceHealthSummary struct {
	Service        string         `json:"service"`
	Namespace      string         `json:"namespace"`
	Deployment     string         `json:"deployment,omitempty"`
	HealthStatus   HealthStatus   `json:"healthStatus"`
	RequestRate    float64        `json:"requestRate"`
	SuccessRate    float64        `json:"successRate"`
	ErrorRate      float64        `json:"errorRate"`
	LatencyP95     float64        `json:"latencyP95"`
	Issues         []HealthIssue  `json:"issues,omitempty"`
}

// HealthStatus represents the overall health of a service
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// HealthIssue describes a health problem detected from metrics
type HealthIssue struct {
	Severity    string  `json:"severity"` // "critical", "warning", "info"
	Description string  `json:"description"`
	Metric      string  `json:"metric"`
	Value       float64 `json:"value"`
	Threshold   float64 `json:"threshold"`
}

// ServiceRanking represents a ranked list of services by a metric
type ServiceRanking struct {
	SortBy   string                     `json:"sortBy"`
	Services []ServiceMetricSummary     `json:"services"`
}

// ServiceMetricSummary contains summary metrics for ranking
type ServiceMetricSummary struct {
	Service     string  `json:"service"`
	Namespace   string  `json:"namespace"`
	Deployment  string  `json:"deployment,omitempty"`
	RequestRate float64 `json:"requestRate"`
	SuccessRate float64 `json:"successRate"`
	ErrorRate   float64 `json:"errorRate"`
	LatencyP95  float64 `json:"latencyP95"`
}

// HealthThresholds defines thresholds for health assessment
type HealthThresholds struct {
	ErrorRateWarning    float64 // Error rate % that triggers warning
	ErrorRateCritical   float64 // Error rate % that triggers critical
	LatencyP95Warning   float64 // P95 latency ms that triggers warning
	LatencyP95Critical  float64 // P95 latency ms that triggers critical
	SuccessRateWarning  float64 // Success rate % below which triggers warning
	SuccessRateCritical float64 // Success rate % below which triggers critical
}

// DefaultHealthThresholds returns sensible default thresholds
func DefaultHealthThresholds() HealthThresholds {
	return HealthThresholds{
		ErrorRateWarning:    5.0,   // 5% error rate
		ErrorRateCritical:   10.0,  // 10% error rate
		LatencyP95Warning:   1000,  // 1 second
		LatencyP95Critical:  5000,  // 5 seconds
		SuccessRateWarning:  95.0,  // 95% success rate
		SuccessRateCritical: 90.0,  // 90% success rate
	}
}

// ParseTimeRange parses a string like "5m", "1h", "24h" into a TimeRange
func ParseTimeRange(rangeStr string) (TimeRange, error) {
	now := time.Now()

	if rangeStr == "" {
		rangeStr = "5m" // default
	}

	duration, err := time.ParseDuration(rangeStr)
	if err != nil {
		return TimeRange{}, err
	}

	// Set step based on duration
	var step time.Duration
	switch {
	case duration <= 5*time.Minute:
		step = 10 * time.Second
	case duration <= time.Hour:
		step = 30 * time.Second
	case duration <= 24*time.Hour:
		step = 5 * time.Minute
	default:
		step = 15 * time.Minute
	}

	return TimeRange{
		Start: now.Add(-duration),
		End:   now,
		Step:  step,
	}, nil
}
