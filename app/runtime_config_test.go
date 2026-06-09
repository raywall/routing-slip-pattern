package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sync/atomic"
	"testing"

	"github.com/raywall/routing-slip-pattern/app/slip"
)

func TestLoadWorkflowConfigExpandsWorkflowRef(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "fiscal"), 0o755); err != nil {
		t.Fatal(err)
	}

	childPath := filepath.Join(dir, "fiscal", "emitir-nota.yaml")
	if err := os.WriteFile(childPath, []byte(`name: emitir-nota
steps:
  - id: validar
    name: validate
    params:
      required:
        - pedido_id
  - id: finalizar
    name: audit
    params:
      event: nota.completed
`), 0o644); err != nil {
		t.Fatal(err)
	}

	parentPath := filepath.Join(dir, "pedido.yaml")
	if err := os.WriteFile(parentPath, []byte(`name: pedido
steps:
  - id: inicio
    name: validate
    params:
      required:
        - pedido_id
  - id: emitir_nota
    name: workflow_ref
    params:
      file: fiscal/emitir-nota.yaml
  - id: fim
    name: audit
    params:
      event: pedido.completed
`), 0o644); err != nil {
		t.Fatal(err)
	}

	workflow, err := loadWorkflowConfig(parentPath)
	if err != nil {
		t.Fatalf("load workflow: %v", err)
	}

	if got, want := len(workflow.Steps), 4; got != want {
		t.Fatalf("steps = %d want %d", got, want)
	}
	if workflow.Steps[1].Name != "validate" || workflow.Steps[1].ID != "emitir_nota.validar" {
		t.Fatalf("expanded step[1] = %#v", workflow.Steps[1])
	}
	if workflow.Steps[2].Name != "audit" || workflow.Steps[2].ID != "emitir_nota.finalizar" {
		t.Fatalf("expanded step[2] = %#v", workflow.Steps[2])
	}
}

func TestLoadWorkflowConfigExpandsWorkspaceRelativeWorkflowRef(t *testing.T) {
	dir := t.TempDir()
	for _, service := range []string{"service-first", "service-last"} {
		if err := os.Mkdir(filepath.Join(dir, service), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	childPath := filepath.Join(dir, "service-last", "B.yaml")
	if err := os.WriteFile(childPath, []byte(`name: B
steps:
  - id: finish
    name: audit
    params:
      event: b.completed
`), 0o644); err != nil {
		t.Fatal(err)
	}

	parentPath := filepath.Join(dir, "service-first", "A.yaml")
	if err := os.WriteFile(parentPath, []byte(`name: A
steps:
  - id: call-b
    name: workflow_ref
    params:
      file: service-last/B
`), 0o644); err != nil {
		t.Fatal(err)
	}

	workflow, err := loadWorkflowConfig(parentPath)
	if err != nil {
		t.Fatalf("load workflow: %v", err)
	}
	if got, want := len(workflow.Steps), 1; got != want {
		t.Fatalf("steps = %d want %d", got, want)
	}
	if workflow.Steps[0].ID != "call-b.finish" {
		t.Fatalf("unexpected referenced step: %#v", workflow.Steps[0])
	}
}

func TestLoadWorkflowConfigRejectsWorkflowRefCycle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "self.yaml")
	if err := os.WriteFile(path, []byte(`name: self
steps:
  - id: self
    name: workflow_ref
    params:
      file: self.yaml
`), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := loadWorkflowConfig(path); err == nil {
		t.Fatal("expected cycle error")
	}
}

func TestLoadEcommerceDistributedCaseWorkflow(t *testing.T) {
	workflow, err := loadWorkflowConfig("../workflows/ecommerce-distributed/order-fulfillment-main.yaml")
	if err != nil {
		t.Fatalf("load ecommerce case workflow: %v", err)
	}
	if workflow.Name != "ecommerce-order-fulfillment" {
		t.Fatalf("workflow name = %q", workflow.Name)
	}
	if got, want := len(workflow.Steps), 21; got != want {
		t.Fatalf("expanded steps = %d want %d", got, want)
	}
	if workflow.Steps[2].ID != "context.graphql-enrich" {
		t.Fatalf("unexpected first referenced step: %#v", workflow.Steps[2])
	}
}

func TestNewCorrelationUUIDGeneratesUniqueUUIDv4(t *testing.T) {
	pattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	seen := map[string]bool{}

	for i := 0; i < 100; i++ {
		id := newCorrelationUUID()
		if !pattern.MatchString(id) {
			t.Fatalf("correlation id %q is not uuid v4", id)
		}
		if seen[id] {
			t.Fatalf("duplicated correlation id %q", id)
		}
		seen[id] = true
	}
}

func TestProcessPayloadGeneratesCorrelationIDWhenMissing(t *testing.T) {
	cfg := AppConfig{}
	applyConfigDefaults(&cfg)
	cfg.Features.PersistentStateEnabled = false
	cfg.Metrics.Endpoint = ""

	workflow := WorkflowConfig{
		Name:              "test-workflow",
		ErrorPolicy:       "stop",
		MessageIDPath:     "id",
		CorrelationIDPath: "correlation_id",
		Steps: []StepConfig{
			{
				Name: "audit",
				Params: map[string]any{
					"event": "test.completed",
				},
			},
		},
	}

	runtime, err := newAppRuntime(&cfg, &workflow, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}

	payload := map[string]any{"id": "MSG-1"}
	msg, err := runtime.processPayload(context.Background(), payload, map[string]string{"trigger": "test"})
	if err != nil {
		t.Fatalf("process payload: %v", err)
	}

	if msg.CorrelationID == "" {
		t.Fatal("expected generated correlation id")
	}
	if got := payload["correlation_id"]; got != msg.CorrelationID {
		t.Fatalf("payload correlation_id = %v want %q", got, msg.CorrelationID)
	}
	if msg.Headers["correlation_id"] != msg.CorrelationID {
		t.Fatalf("header correlation_id = %q want %q", msg.Headers["correlation_id"], msg.CorrelationID)
	}
}

func TestProcessPayloadRejectsConcurrentDuplicateMessageID(t *testing.T) {
	cfg := AppConfig{}
	applyConfigDefaults(&cfg)
	cfg.Features.PersistentStateEnabled = true
	cfg.StateStore.Type = "memory"
	cfg.StateStore.ProcessingLock.TTLSeconds = 30
	cfg.Metrics.Endpoint = ""

	workflow := WorkflowConfig{
		Name:              "test-workflow",
		ErrorPolicy:       "stop",
		MessageIDPath:     "id",
		CorrelationIDPath: "correlation_id",
		Steps: []StepConfig{
			{Name: "blocking_count"},
		},
	}

	runtime, err := newAppRuntime(&cfg, &workflow, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	handler := &blockingCountHandler{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	runtime.router.MustRegister(handler)

	firstDone := make(chan error, 1)
	go func() {
		_, err := runtime.processPayload(context.Background(), map[string]any{"id": "MSG-DUP"}, map[string]string{"trigger": "test"})
		firstDone <- err
	}()
	<-handler.started

	_, err = runtime.processPayload(context.Background(), map[string]any{"id": "MSG-DUP"}, map[string]string{"trigger": "test"})
	if !slip.IsProcessingLocked(err) {
		t.Fatalf("expected processing lock error, got %v", err)
	}

	close(handler.release)
	if err := <-firstDone; err != nil {
		t.Fatalf("first process: %v", err)
	}
	if got := handler.count.Load(); got != 1 {
		t.Fatalf("handler count = %d want 1", got)
	}
	msg, err := runtime.processPayload(context.Background(), map[string]any{"id": "MSG-DUP"}, map[string]string{"trigger": "test"})
	if err != nil {
		t.Fatalf("completed duplicate process: %v", err)
	}
	if msg.Status != "completed" {
		t.Fatalf("duplicate status = %q want completed", msg.Status)
	}
	if got := handler.count.Load(); got != 1 {
		t.Fatalf("handler count after completed duplicate = %d want 1", got)
	}
}

func TestComposedWorkflowUsesSingleCorrelationID(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "inventory"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "inventory", "reserve.yaml"), []byte(`name: reserve
steps:
  - id: reserve_audit
    name: audit
    params:
      event: reserve.completed
`), 0o644); err != nil {
		t.Fatal(err)
	}
	parentPath := filepath.Join(dir, "order.yaml")
	if err := os.WriteFile(parentPath, []byte(`name: order
message_id_path: order_id
correlation_id_path: correlation_id
steps:
  - id: start
    name: audit
    params:
      event: order.started
  - id: reserve
    name: workflow_ref
    params:
      workflow: inventory/reserve
  - id: finish
    name: audit
    params:
      event: order.finished
`), 0o644); err != nil {
		t.Fatal(err)
	}
	workflow, err := loadWorkflowConfig(parentPath)
	if err != nil {
		t.Fatalf("load workflow: %v", err)
	}

	cfg := AppConfig{}
	applyConfigDefaults(&cfg)
	cfg.Features.PersistentStateEnabled = false
	cfg.Metrics.Endpoint = ""

	runtime, err := newAppRuntime(&cfg, workflow, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	payload := map[string]any{"order_id": "ORD-COMPOSED"}
	msg, err := runtime.processPayload(context.Background(), payload, map[string]string{"trigger": "test"})
	if err != nil {
		t.Fatalf("process payload: %v", err)
	}
	if msg.CorrelationID == "" {
		t.Fatal("expected generated correlation id")
	}
	if payload["correlation_id"] != msg.CorrelationID {
		t.Fatalf("payload correlation_id = %v want %q", payload["correlation_id"], msg.CorrelationID)
	}
	if got, want := len(msg.History), 3; got != want {
		t.Fatalf("history len = %d want %d", got, want)
	}
	for _, entry := range msg.History {
		if msg.Headers["correlation_id"] != msg.CorrelationID {
			t.Fatalf("correlation changed while executing %s", entry.Step)
		}
	}
}

type blockingCountHandler struct {
	started chan struct{}
	release chan struct{}
	count   atomic.Int64
}

func (h *blockingCountHandler) Name() string { return "blocking_count" }

func (h *blockingCountHandler) Handle(ctx context.Context, msg *slip.Message, params map[string]any) (bool, error) {
	h.count.Add(1)
	close(h.started)
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case <-h.release:
		return true, nil
	}
}
