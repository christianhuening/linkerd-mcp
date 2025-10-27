package metrics

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/api"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// PrometheusClient wraps the Prometheus API client
type PrometheusClient struct {
	api       prometheusv1.API
	namespace string
}

// NewPrometheusClient creates a new Prometheus client
// It attempts to connect to the Linkerd Prometheus instance
func NewPrometheusClient(config *rest.Config, clientset kubernetes.Interface, namespace string) (*PrometheusClient, error) {
	if namespace == "" {
		namespace = "linkerd" // default Linkerd namespace
	}

	// Get Prometheus URL from environment or use default
	promURL := os.Getenv("LINKERD_PROMETHEUS_URL")
	if promURL == "" {
		// Default to in-cluster service
		promURL = fmt.Sprintf("http://prometheus.%s.svc.cluster.local:9090", namespace)
	}

	// Create Prometheus API client
	client, err := api.NewClient(api.Config{
		Address: promURL,
		RoundTripper: &http.Transport{
			MaxIdleConns:       10,
			IdleConnTimeout:    30 * time.Second,
			DisableCompression: true,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus client: %w", err)
	}

	return &PrometheusClient{
		api:       prometheusv1.NewAPI(client),
		namespace: namespace,
	}, nil
}

// Query executes an instant Prometheus query
func (c *PrometheusClient) Query(ctx context.Context, query string, ts time.Time) (model.Value, error) {
	result, warnings, err := c.api.Query(ctx, query, ts)
	if err != nil {
		return nil, fmt.Errorf("prometheus query failed: %w", err)
	}

	if len(warnings) > 0 {
		// Log warnings but don't fail
		for _, w := range warnings {
			fmt.Printf("Prometheus warning: %s\n", w)
		}
	}

	return result, nil
}

// QueryRange executes a range Prometheus query
func (c *PrometheusClient) QueryRange(ctx context.Context, query string, tr TimeRange) (model.Value, error) {
	r := prometheusv1.Range{
		Start: tr.Start,
		End:   tr.End,
		Step:  tr.Step,
	}

	result, warnings, err := c.api.QueryRange(ctx, query, r)
	if err != nil {
		return nil, fmt.Errorf("prometheus range query failed: %w", err)
	}

	if len(warnings) > 0 {
		for _, w := range warnings {
			fmt.Printf("Prometheus warning: %s\n", w)
		}
	}

	return result, nil
}

// GetLabelValues returns all values for a given label
func (c *PrometheusClient) GetLabelValues(ctx context.Context, label string, startTime, endTime time.Time) ([]string, error) {
	matches := []string{}
	labelValues, warnings, err := c.api.LabelValues(ctx, label, matches, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get label values: %w", err)
	}

	if len(warnings) > 0 {
		for _, w := range warnings {
			fmt.Printf("Prometheus warning: %s\n", w)
		}
	}

	values := make([]string, len(labelValues))
	for i, v := range labelValues {
		values[i] = string(v)
	}

	return values, nil
}

// CheckHealth verifies Prometheus is accessible
func (c *PrometheusClient) CheckHealth(ctx context.Context) error {
	// Simple query to verify connectivity
	query := "up"
	_, err := c.Query(ctx, query, time.Now())
	return err
}

// extractScalarValue extracts a float64 value from a Prometheus query result
func extractScalarValue(value model.Value) (float64, error) {
	switch v := value.(type) {
	case model.Vector:
		if len(v) == 0 {
			return 0, nil
		}
		return float64(v[0].Value), nil
	case *model.Scalar:
		return float64(v.Value), nil
	default:
		return 0, fmt.Errorf("unexpected value type: %T", value)
	}
}

