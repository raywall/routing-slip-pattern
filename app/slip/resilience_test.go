package slip

import (
	"context"
	"errors"
	"testing"
)

type retryThenSuccessHandler struct {
	calls int
}

func (h *retryThenSuccessHandler) Name() string { return "retry_then_success" }

func (h *retryThenSuccessHandler) Handle(_ context.Context, msg *Message, _ map[string]any) (bool, error) {
	h.calls++
	if h.calls < 3 {
		return false, errors.New("temporary failure")
	}
	msg.Set("processed", true)
	return true, nil
}

type alwaysFailHandler struct{}

func (alwaysFailHandler) Name() string { return "always_fail" }

func (alwaysFailHandler) Handle(context.Context, *Message, map[string]any) (bool, error) {
	return false, errors.New("permanent failure")
}

func TestRouterRetriesStepUsingResiliencePolicy(t *testing.T) {
	handler := &retryThenSuccessHandler{}
	router := NewRouter(WithMiddleware(TracingMiddleware()))
	router.MustRegister(handler)

	msg := NewMessage("msg-1", nil)
	msg.AttachSlip([]StepDef{{
		Name: "retry_then_success",
		Resilience: ResiliencePolicy{
			Retry: RetryPolicy{Attempts: 3, Backoff: "none"},
		},
	}})

	if err := router.Process(context.Background(), msg); err != nil {
		t.Fatalf("Process returned error: %v", err)
	}
	if handler.calls != 3 {
		t.Fatalf("calls = %d, want 3", handler.calls)
	}
	if msg.Attempt != 3 {
		t.Fatalf("message attempt = %d, want 3", msg.Attempt)
	}
	if len(msg.History) != 1 || msg.History[0].Attempt != 3 || msg.History[0].Status != "success" {
		t.Fatalf("unexpected history: %#v", msg.History)
	}
}

func TestRouterFailurePolicyJumpsAfterRetries(t *testing.T) {
	router := NewRouter(WithMiddleware(TracingMiddleware()))
	router.MustRegister(alwaysFailHandler{})
	router.MustRegister(testSetHandler{name: "fallback", key: "fallback", value: true})

	msg := NewMessage("msg-1", nil)
	msg.AttachSlip([]StepDef{
		{
			ID:   "fail",
			Name: "always_fail",
			Resilience: ResiliencePolicy{
				Retry:     RetryPolicy{Attempts: 2, Backoff: "none"},
				OnFailure: OnFailurePolicy{Action: "jump", To: "fallback"},
			},
		},
		{ID: "fallback", Name: "fallback"},
	})

	if err := router.Process(context.Background(), msg); err != nil {
		t.Fatalf("Process returned error: %v", err)
	}
	value, _ := msg.Get("fallback")
	if value != true {
		t.Fatalf("fallback value = %#v", value)
	}
	if len(msg.History) != 2 || msg.History[0].Status != "jumped" {
		t.Fatalf("unexpected history: %#v", msg.History)
	}
}
