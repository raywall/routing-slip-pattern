package slip

import (
	"math/rand"
	"strings"
	"time"
)

// ResiliencePolicy configures retry and failure behavior for a step.
type ResiliencePolicy struct {
	Retry     RetryPolicy     `yaml:"retry" json:"retry,omitempty"`
	OnFailure OnFailurePolicy `yaml:"on_failure" json:"on_failure,omitempty"`
}

// RetryPolicy controls retry attempts and delay strategy.
type RetryPolicy struct {
	Attempts          int    `yaml:"attempts" json:"attempts,omitempty"`
	Backoff           string `yaml:"backoff" json:"backoff,omitempty"`
	InitialIntervalMS int    `yaml:"initial_interval_ms" json:"initial_interval_ms,omitempty"`
	MaxIntervalMS     int    `yaml:"max_interval_ms" json:"max_interval_ms,omitempty"`
	Jitter            bool   `yaml:"jitter" json:"jitter,omitempty"`
}

// OnFailurePolicy controls what the router does after retries are exhausted.
type OnFailurePolicy struct {
	Action string `yaml:"action" json:"action,omitempty"`
	To     string `yaml:"to" json:"to,omitempty"`
}

func (p ResiliencePolicy) attempts() int {
	if p.Retry.Attempts <= 0 {
		return 1
	}
	return p.Retry.Attempts
}

func (p ResiliencePolicy) failureAction() string {
	action := strings.ToLower(strings.TrimSpace(p.OnFailure.Action))
	if action == "" {
		return "default"
	}
	return action
}

func (p ResiliencePolicy) backoffDuration(attempt int) time.Duration {
	if attempt <= 1 {
		return 0
	}
	initial := time.Duration(p.Retry.InitialIntervalMS) * time.Millisecond
	if initial <= 0 {
		initial = 100 * time.Millisecond
	}
	maxInterval := time.Duration(p.Retry.MaxIntervalMS) * time.Millisecond
	if maxInterval <= 0 {
		maxInterval = time.Second
	}
	delay := initial
	switch strings.ToLower(strings.TrimSpace(p.Retry.Backoff)) {
	case "fixed":
	case "none":
		delay = 0
	default:
		for i := 2; i < attempt; i++ {
			delay *= 2
			if delay >= maxInterval {
				delay = maxInterval
				break
			}
		}
	}
	if delay > maxInterval {
		delay = maxInterval
	}
	if p.Retry.Jitter && delay > 0 {
		delay += time.Duration(rand.Int63n(int64(delay / 2)))
	}
	return delay
}

func (p ResiliencePolicy) enabled() bool {
	return p.Retry.Attempts > 1 || strings.TrimSpace(p.OnFailure.Action) != ""
}
