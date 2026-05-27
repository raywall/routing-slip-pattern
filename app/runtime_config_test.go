package main

import (
	"os"
	"path/filepath"
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
