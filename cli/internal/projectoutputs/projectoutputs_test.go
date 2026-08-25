package projectoutputs

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectOutputsInitAndDoctor(t *testing.T) {
	project := t.TempDir()
	result, err := Init(project, "demo-crm")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Created) != 3 {
		t.Fatalf("expected exactly three fixed files, got %+v", result.Created)
	}
	entries, err := os.ReadDir(filepath.Join(project, "outputs"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("initializer must not pre-create output categories, got %d entries", len(entries))
	}
	report, err := Doctor(project)
	if err != nil || report.Status != "passed" || report.Summary["outputCount"] != 0 {
		t.Fatalf("unexpected doctor result: err=%v report=%+v", err, report)
	}
}

func TestProjectOutputsInitDoesNotOverwrite(t *testing.T) {
	project := t.TempDir()
	if _, err := Init(project, "demo-crm"); err != nil {
		t.Fatal(err)
	}
	readme := filepath.Join(project, "outputs", "README.md")
	if err := os.WriteFile(readme, []byte("project-owned\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := Init(project, "demo-crm")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(readme)
	if string(b) != "project-owned\n" || !contains(result.Skipped, "outputs/README.md") {
		t.Fatalf("existing output asset was overwritten: result=%+v content=%q", result, b)
	}
}

func TestProjectOutputsDoctorValidatesDeliveredToolAndDigest(t *testing.T) {
	project := t.TempDir()
	if _, err := Init(project, "demo-crm"); err != nil {
		t.Fatal(err)
	}
	releasePath := filepath.Join(project, "outputs", "migration-tool", "release", "migration-tool.zip")
	if err := os.MkdirAll(filepath.Dir(releasePath), 0o755); err != nil {
		t.Fatal(err)
	}
	content := []byte("frozen-tool-package")
	if err := os.WriteFile(releasePath, content, 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	manifest := Manifest{
		SchemaVersion: SchemaVersion,
		ProjectCode:   "demo-crm",
		Outputs: []OutputEntry{{
			OutputID: "migration-tool", Kind: "tool", Title: "Customer Migration Tool", Status: "delivered", OwnerRole: "integration-agent",
			ReleaseArtifacts: []ReleaseArtifact{{Path: "outputs/migration-tool/release/migration-tool.zip", SHA256: hex.EncodeToString(sum[:])}},
		}},
	}
	writeManifest(t, project, manifest)
	report, err := Doctor(project)
	if err != nil || report.Status != "passed" || report.Summary["artifactCount"] != 1 {
		t.Fatalf("expected delivered tool to pass: err=%v report=%+v", err, report)
	}
}

func TestProjectOutputsDoctorRejectsUnsafeMissingAndSensitiveOutputs(t *testing.T) {
	project := t.TempDir()
	if _, err := Init(project, "demo-crm"); err != nil {
		t.Fatal(err)
	}
	manifest := Manifest{
		SchemaVersion: SchemaVersion,
		ProjectCode:   "demo-crm",
		Outputs: []OutputEntry{{
			OutputID: "ops-tool", Kind: "tool", Title: "Operations Tool", Status: "delivered",
			WorkingPaths:     []string{"../outside.sh"},
			ReleaseArtifacts: []ReleaseArtifact{{Path: "outputs/ops-tool/release/tool.zip", SHA256: strings.Repeat("0", 64)}},
		}},
	}
	writeManifest(t, project, manifest)
	secretPath := filepath.Join(project, "outputs", "ops-tool", ".env")
	if err := os.MkdirAll(filepath.Dir(secretPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secretPath, []byte("TOKEN=not-allowed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := Doctor(project)
	if err == nil || report.Status != "failed" {
		t.Fatalf("expected doctor failure: err=%v report=%+v", err, report)
	}
	for _, code := range []string{"output_path_unsafe", "output_reference_missing", "sensitive_output_file_forbidden"} {
		if !hasIssue(report.Errors, code) {
			t.Fatalf("expected %s in %+v", code, report.Errors)
		}
	}
}

func TestProjectOutputsDoctorRejectsSymlinkEscape(t *testing.T) {
	project := t.TempDir()
	outside := t.TempDir()
	if _, err := Init(project, "demo-crm"); err != nil {
		t.Fatal(err)
	}
	outsideArtifact := filepath.Join(outside, "external.zip")
	content := []byte("must-not-be-read-as-project-output")
	if err := os.WriteFile(outsideArtifact, content, 0o644); err != nil {
		t.Fatal(err)
	}
	linkPath := filepath.Join(project, "outputs", "linked-release.zip")
	if err := os.Symlink(outsideArtifact, linkPath); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	writeManifest(t, project, Manifest{
		SchemaVersion: SchemaVersion,
		ProjectCode:   "demo-crm",
		Outputs: []OutputEntry{{
			OutputID: "linked-release", Kind: "tool", Title: "Linked Release", Status: "delivered",
			ReleaseArtifacts: []ReleaseArtifact{{Path: "outputs/linked-release.zip", SHA256: hex.EncodeToString(sum[:])}},
		}},
	})
	report, err := Doctor(project)
	if err == nil || !hasIssue(report.Errors, "output_path_symlink_escape") {
		t.Fatalf("expected symbolic-link escape rejection: err=%v report=%+v", err, report)
	}
}

func writeManifest(t *testing.T, project string, manifest Manifest) {
	t.Helper()
	b, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "outputs", "output-manifest.json"), append(b, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func hasIssue(values []DoctorIssue, code string) bool {
	for _, value := range values {
		if value.Code == code {
			return true
		}
	}
	return false
}
