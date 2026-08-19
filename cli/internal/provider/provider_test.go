package provider

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"cloudcc-customization-expert-go/internal/edition"
)

func TestUniversalAutoSelectsUIAPIWithoutMetadataService(t *testing.T) {
	withUniversalEdition(t)
	t.Setenv("CLOUDCC_METADATA_SERVICE_URL", "")
	t.Setenv("CLOUDCC_EXECUTION_MODE", "")
	selection, err := Resolve(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if selection.SelectedMode != ModeUIAPI || selection.Reason != "metadata_service_not_configured" {
		t.Fatalf("unexpected selection: %#v", selection)
	}
}

func TestUniversalAutoSelectsMSAPIAfterReadOnlyProbe(t *testing.T) {
	withUniversalEdition(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metadata/v1/capabilities" || r.Method != http.MethodGet {
			t.Fatalf("unexpected probe %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"domains":[]}`))
	}))
	defer server.Close()
	t.Setenv("CLOUDCC_METADATA_SERVICE_URL", server.URL)
	selection, err := Resolve(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if selection.SelectedMode != ModeMSAPI || selection.Reason != "metadata_service_capability_probe_passed" {
		t.Fatalf("unexpected selection: %#v", selection)
	}
}

func TestUniversalAutoRefusesFallbackWhenConfiguredProbeFails(t *testing.T) {
	withUniversalEdition(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not ready", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	t.Setenv("CLOUDCC_METADATA_SERVICE_URL", server.URL)
	if _, err := Resolve(t.TempDir()); err == nil {
		t.Fatal("expected configured probe failure to reject fallback")
	}
}

func TestStrictUIAPIPackageRejectsMSAPIRequest(t *testing.T) {
	originalPackage, originalSuffix := edition.PackageName, edition.VersionSuffix
	originalDefault, originalStrict := edition.DefaultExecutionMode, edition.StrictExecutionMode
	t.Cleanup(func() {
		edition.PackageName, edition.VersionSuffix = originalPackage, originalSuffix
		edition.DefaultExecutionMode, edition.StrictExecutionMode = originalDefault, originalStrict
	})
	edition.PackageName = "cc-customization-expert-uiapi"
	edition.VersionSuffix = "-uiapi"
	edition.DefaultExecutionMode = ModeUIAPI
	edition.StrictExecutionMode = ModeUIAPI
	t.Setenv("CLOUDCC_EXECUTION_MODE", ModeMSAPI)
	if _, err := Resolve(t.TempDir()); err == nil {
		t.Fatal("expected strict UIAPI package to reject MSAPI request")
	}
}

func TestProjectExecutionModeOverridesUniversalDefault(t *testing.T) {
	withUniversalEdition(t)
	project := t.TempDir()
	configBody := []byte(`{"use":"dev","dev":{"executionMode":"uiapi"}}`)
	if err := os.WriteFile(filepath.Join(project, "cloudcc-cli.config.json"), configBody, 0644); err != nil {
		t.Fatal(err)
	}
	selection, err := Resolve(project)
	if err != nil {
		t.Fatal(err)
	}
	if selection.SelectedMode != ModeUIAPI || selection.Reason != "explicit_execution_mode" {
		t.Fatalf("unexpected selection: %#v", selection)
	}
}

func withUniversalEdition(t *testing.T) {
	t.Helper()
	originalPackage, originalSuffix := edition.PackageName, edition.VersionSuffix
	originalDefault, originalStrict := edition.DefaultExecutionMode, edition.StrictExecutionMode
	t.Cleanup(func() {
		edition.PackageName, edition.VersionSuffix = originalPackage, originalSuffix
		edition.DefaultExecutionMode, edition.StrictExecutionMode = originalDefault, originalStrict
	})
	edition.PackageName = "cc-customization-expert-universal"
	edition.VersionSuffix = "-universal"
	edition.DefaultExecutionMode = ModeAuto
	edition.StrictExecutionMode = ""
}
