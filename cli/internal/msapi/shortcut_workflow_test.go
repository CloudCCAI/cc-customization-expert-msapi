package msapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWorkflowReadShortcutsUseDedicatedMetadataEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metadata/v1/workflows" {
			if got, want := r.URL.Query().Get("filter"), "客户"; got != want {
				t.Fatalf("filter query = %q, want %q", got, want)
			}
			_, _ = w.Write([]byte(`{"mode":"read-only-workflows","count":1}`))
			return
		}
		if r.URL.Path == "/metadata/v1/workflows/wf-1" {
			_, _ = w.Write([]byte(`{"mode":"read-only-workflow-detail"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	project := t.TempDir()
	writeTestFile(t, project+"/cloudcc-cli.config.json",
		`{"use":"dev","dev":{"metadataService":{"url":"`+server.URL+`"}}}`)

	for _, tc := range []struct {
		name     string
		action   string
		args     []string
		wantMode string
	}{
		{"list", "get", []string{project, "客户"}, "read-only-workflows"},
		{"detail", "detail", []string{project, "wf-1"}, "read-only-workflow-detail"},
		{"edit-info", "editInfo", []string{project, "wf-1"}, "read-only-workflow-detail"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			if err := HandleLowCodeShortcut(tc.action, "workflow", tc.args, &output, project); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(output.String(), tc.wantMode) {
				t.Fatalf("output %q does not include %q", output.String(), tc.wantMode)
			}
		})
	}
}

func TestWorkflowWriteShortcutsCreateMetadataPlans(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metadata/v1/plans" {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if got, want := body["domain"], "workflows"; got != want {
			t.Fatalf("domain = %v, want %v", got, want)
		}
		_, _ = w.Write([]byte(`{"planId":"plan-workflow","status":"PLANNED"}`))
	}))
	defer server.Close()

	project := t.TempDir()
	writeTestFile(t, project+"/cloudcc-cli.config.json",
		`{"use":"dev","dev":{"metadataService":{"url":"`+server.URL+`"}}}`)

	for _, tc := range []struct {
		name   string
		action string
		args   []string
	}{
		{"create-json", "create", []string{project, `{"id":"wf-1","targetObjectId":"obj","name":"客户工作流"}`}},
		{"update-json", "update", []string{project, "wf-1", `{"name":"客户工作流更新"}`}},
		{"delete", "delete", []string{project, "wf-1"}},
		{"enable", "enable", []string{project, "wf-1"}},
		{"disable", "disable", []string{project, "wf-1"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			if err := HandleLowCodeShortcut(tc.action, "workflowRule", tc.args, &output, project); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(output.String(), "plan-workflow") {
				t.Fatalf("output %q does not include plan id", output.String())
			}
		})
	}
}
