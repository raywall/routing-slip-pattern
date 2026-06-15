package source

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalLoadsAndResolvesRelativeWorkflow(t *testing.T) {
	dir := t.TempDir()
	parent := filepath.Join(dir, "service", "main.yaml")
	child := filepath.Join(dir, "service", "child.yaml")
	if err := os.MkdirAll(filepath.Dir(parent), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(parent, []byte("parent"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(child, []byte("child"), 0644); err != nil {
		t.Fatal(err)
	}
	resolved, err := (Local{Path: parent}).Resolve("child.yaml")
	if err != nil {
		t.Fatal(err)
	}
	data, err := resolved.Load(context.Background())
	if err != nil || string(data) != "child" {
		t.Fatalf("data=%q err=%v", data, err)
	}
}

func TestAWSResolvesRelativeS3Workflow(t *testing.T) {
	resolved, err := (AWS{Type: "s3", Bucket: "workflows", Key: "service/main.yaml"}).Resolve("child.yaml")
	if err != nil {
		t.Fatal(err)
	}
	child := resolved.(AWS)
	if child.Key != "service/child.yaml" {
		t.Fatalf("key = %q", child.Key)
	}
}
