package slip

import (
	"context"
	"errors"
	"testing"
)

type failOnceHandler struct {
	failed bool
}

func (h *failOnceHandler) Name() string { return "fail_once" }

func (h *failOnceHandler) Handle(ctx context.Context, msg *Message, params map[string]any) (bool, error) {
	if !h.failed {
		h.failed = true
		return false, errors.New("temporary failure")
	}
	msg.Set("resumed", true)
	return true, nil
}

func TestRouterResumesFromFailedStep(t *testing.T) {
	store := NewMemoryStateStore()
	handler := &failOnceHandler{}
	router := NewRouter(WithStateStore(store))
	router.MustRegister(handler)
	router.MustRegister(testSetHandler{name: "after_resume", key: "after", value: true})

	msg := NewMessage("MSG-RESUME", map[string]any{"input": "ok"})
	msg.AttachSlip(NewSlip().
		Step("fail_once").
		Step("after_resume").
		Build(),
	)

	if err := router.Process(context.Background(), msg); err == nil {
		t.Fatal("expected first run to fail")
	}

	snapshot, err := store.Load(context.Background(), msg.ID)
	if err != nil {
		t.Fatalf("load snapshot: %v", err)
	}
	if snapshot.Cursor != 0 {
		t.Fatalf("expected cursor to point to failed step 0, got %d", snapshot.Cursor)
	}

	resumed := MessageFromSnapshot(snapshot)
	if err := router.Process(context.Background(), resumed); err != nil {
		t.Fatalf("resume process: %v", err)
	}
	if resumed.Cursor() != 2 {
		t.Fatalf("expected cursor at end, got %d", resumed.Cursor())
	}
	if value, _ := resumed.Get("resumed"); value != true {
		t.Fatalf("expected failed step to be re-executed on resume")
	}
	if value, _ := resumed.Get("after"); value != true {
		t.Fatalf("expected next step to run after resume")
	}
}

func TestRouterAppliesCursorController(t *testing.T) {
	router := NewRouter(WithMiddleware(RecoveryMiddleware()))
	router.MustRegister(testJumpHandler{})
	router.MustRegister(testSetHandler{name: "skip_me", key: "skipped", value: true})
	router.MustRegister(testSetHandler{name: "finish", key: "finished", value: true})

	msg := NewMessage("MSG-JUMP", map[string]any{})
	msg.AttachSlip([]StepDef{
		{Name: "jump", Params: map[string]any{"to": "finish-step"}},
		{Name: "skip_me"},
		{ID: "finish-step", Name: "finish"},
	})

	if err := router.Process(context.Background(), msg); err != nil {
		t.Fatalf("process: %v", err)
	}
	if value, _ := msg.Get("skipped"); value == true {
		t.Fatal("expected skipped step not to run")
	}
	if value, _ := msg.Get("finished"); value != true {
		t.Fatal("expected finish step to run")
	}
}

type testJumpHandler struct{}

func (testJumpHandler) Name() string { return "jump" }

func (testJumpHandler) Handle(ctx context.Context, msg *Message, params map[string]any) (bool, error) {
	return true, nil
}

func (testJumpHandler) NextCursor(msg *Message, step StepDef, currentIndex int) (int, bool, error) {
	index, ok := msg.FindStepIndex(step.Params["to"].(string))
	return index, ok, nil
}

type testSetHandler struct {
	name  string
	key   string
	value any
}

func (h testSetHandler) Name() string { return h.name }

func (h testSetHandler) Handle(ctx context.Context, msg *Message, params map[string]any) (bool, error) {
	msg.Set(h.key, h.value)
	return true, nil
}
