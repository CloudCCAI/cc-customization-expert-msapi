package modules

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateProjectCommandWritesCanonicalLayout(t *testing.T) {
	tmp := t.TempDir()
	var stdout, stderr bytes.Buffer
	if err := Handle("create", "project", []string{"demo"}, &stdout, &stderr, tmp); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		"demo/cloudcc-cli.config.json",
		"demo/frontend/pagecomponents",
		"demo/backend/classes",
		"demo/backend/triggers",
		"demo/backend/schedule",
		"demo/sidecar",
	} {
		if _, err := os.Stat(filepath.Join(tmp, filepath.FromSlash(path))); err != nil {
			t.Fatalf("expected generated path %s: %v", path, err)
		}
	}
}

func TestCreateClassWritesUnderBackend(t *testing.T) {
	tmp := t.TempDir()
	var stderr bytes.Buffer
	if err := createJavaResource("classes", "classes", []string{"DemoClass"}, &stderr, tmp); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "backend", "classes", "DemoClass", "DemoClass.java")); err != nil {
		t.Fatalf("expected backend class source: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "classes")); !os.IsNotExist(err) {
		t.Fatalf("unexpected legacy classes directory, statErr=%v", err)
	}
}
