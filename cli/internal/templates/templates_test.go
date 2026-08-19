package templates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteProjectUsesCanonicalLayout(t *testing.T) {
	tmp := t.TempDir()
	target := filepath.Join(tmp, "demo")
	if err := WriteProject(target, "Demo Project"); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		"cloudcc-cli.config.json",
		".gitignore",
		"frontend/README.md",
		"frontend/pagecomponents",
		"frontend/build",
		"backend/classes",
		"backend/triggers",
		"backend/schedule",
		"backend/lib/ccopenapi-0.1.3.jar",
		"backend/lib/fastjson-1.2.83.jar",
		"backend/lib/reflections-0.9.12.jar",
		"sidecar",
	} {
		if _, err := os.Stat(filepath.Join(target, filepath.FromSlash(path))); err != nil {
			t.Fatalf("expected %s: %v", path, err)
		}
	}
	for _, path := range []string{
		".cloudcc",
		".claude",
		"doc",
		"package.json",
		"frontend/package.json",
		"frontend/vue.config.js",
		"frontend/babel.config.js",
		"src",
		"public",
		"frontend/src",
		"frontend/public",
		"lib",
		"plugins",
	} {
		if _, err := os.Stat(filepath.Join(target, filepath.FromSlash(path))); !os.IsNotExist(err) {
			t.Fatalf("unexpected legacy path %s, statErr=%v", path, err)
		}
	}
	frontendReadme, err := os.ReadFile(filepath.Join(target, "frontend", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(frontendReadme), "node") || strings.Contains(string(frontendReadme), "npm") {
		t.Fatalf("frontend README should not create a Node/npm dependency: %s", frontendReadme)
	}
	if !strings.Contains(string(frontendReadme), "pagecomponent") || !strings.Contains(string(frontendReadme), "prebuilt UMD") {
		t.Fatalf("frontend README should describe pagecomponent prebuilt bundle flow: %s", frontendReadme)
	}
}

func TestWriteProjectRejectsNonEmptyTarget(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "existing.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := WriteProject(tmp, "demo"); err == nil {
		t.Fatal("expected non-empty target rejection")
	}
}
