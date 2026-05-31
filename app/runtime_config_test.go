package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"testing"
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
