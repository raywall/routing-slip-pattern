package slip

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// MetricEvent is compatible with custom-business-metrics /v1/metrics.
type MetricEvent struct {
	Name      string            `json:"name"`
	Kind      string            `json:"kind"`
	Value     float64           `json:"value"`
	Unit      string            `json:"unit,omitempty"`
	Segment   string            `json:"segment,omitempty"`
	Workflow  string            `json:"workflow,omitempty"`
	Step      string            `json:"step,omitempty"`
	Status    string            `json:"status,omitempty"`
	Source    string            `json:"source,omitempty"`
	Tags      map[string]string `json:"tags,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// MetricsEmitter publishes routing slip lifecycle events.
type MetricsEmitter interface {
	Emit(ctx context.Context, event MetricEvent) error
}

// HTTPMetricsEmitter sends events to custom-business-metrics.
type HTTPMetricsEmitter struct {
	Endpoint string
	Client   *http.Client
}

// Emit publishes one event. Failures are returned to the middleware, which logs
// them without failing the workflow.
func (e HTTPMetricsEmitter) Emit(ctx context.Context, event MetricEvent) error {
	if e.Endpoint == "" {
		return nil
	}
	client := e.Client
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Second}
	}
	body, err := json.Marshal(map[string]any{"events": []MetricEvent{event}})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.Endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("metrics endpoint returned status %s", resp.Status)
	}
	return nil
}

// MetricsOptions controls metric dimensions and tags.
type MetricsOptions struct {
	Workflow string
	Segment  string
	Source   string
	Tags     map[string]string
}
