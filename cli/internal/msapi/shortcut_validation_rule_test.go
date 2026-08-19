package msapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidationRuleReadShortcutsUseDedicatedMetadataEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metadata/v1/validation-rules" {
			if got, want := r.URL.Query().Get("object"), "ctc"; got != want {
				t.Fatalf("object query = %q, want %q", got, want)
			}
			_, _ = w.Write([]byte(`{"mode":"read-only-validation-rules","count":1}`))
			return
		}
		if r.URL.Path == "/metadata/v1/validation-rules/rule-1" {
			_, _ = w.Write([]byte(`{"mode":"read-only-validation-rule-detail"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	project := t.TempDir()
	writeTestFile(t, project+"/cloudcc-cli.config.json",
		`{"use":"dev","dev":{"metadataService":{"url":"`+server.URL+`"}}}`)
	for _, tc := range []struct {
		name string
		action string
		args []string
		want string
	}{
		{"list", "get", []string{project, "ctc"}, "read-only-validation-rules"},
		{"filtered-list", "getList", []string{project, "ctc", "amount"}, "read-only-validation-rules"},
		{"detail", "detail", []string{project, "rule-1"}, "read-only-validation-rule-detail"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			if err := HandleLowCodeShortcut(tc.action, "validationRule", tc.args, &output, project); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(output.String(), tc.want) {
				t.Fatalf("output %q does not include %q", output.String(), tc.want)
			}
		})
	}
}
