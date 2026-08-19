package modules

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type triggerTestCall struct {
	Path string
	Body map[string]any
}

func TestTriggerMetadataRoutesAndAliases(t *testing.T) {
	var calls []triggerTestCall
	server := newTriggerTestServer(t, &calls)
	defer server.Close()
	project := newTriggerTestProject(t, server.URL)

	for _, resource := range []string{"trigger", "triggers"} {
		calls = nil
		var stdout, stderr bytes.Buffer
		if err := Handle("get", resource, []string{project}, &stdout, &stderr, project); err != nil {
			t.Fatalf("get %s failed: %v", resource, err)
		}
		assertTriggerTestPaths(t, calls, "/api/triggerSetup/getTriggerByCondition")
		if got := calls[0].Body; got["shownum"] != "2000" || got["showpage"] != "1" || got["objId"] != "" {
			t.Fatalf("unexpected trigger list body: %#v", got)
		}
	}

	calls = nil
	var stdout, stderr bytes.Buffer
	if err := Handle("detail", "trigger", []string{project, "ContractRulesManager"}, &stdout, &stderr, project); err != nil {
		t.Fatalf("detail trigger by name failed: %v", err)
	}
	assertTriggerTestPaths(t, calls, "/api/triggerSetup/getTriggerByCondition", "/api/trigger/newobjtrigger")
	if calls[0].Body["sname"] != "ContractRulesManager" || calls[1].Body["id"] != "trg-001" {
		t.Fatalf("unexpected trigger detail resolution calls: %#v", calls)
	}

	calls = nil
	stdout.Reset()
	if err := Handle("delete", "trigger", []string{project, "ContractRulesManager"}, &stdout, &stderr, project); err != nil {
		t.Fatalf("delete trigger by name failed: %v", err)
	}
	assertTriggerTestPaths(t, calls, "/api/triggerSetup/getTriggerByCondition", "/api/triggerSetup/deleteTrigger")
	if calls[1].Body["id"] != "trg-001" {
		t.Fatalf("unexpected trigger delete body: %#v", calls[1].Body)
	}

	calls = nil
	stdout.Reset()
	spec := `{"id":"trg-001","name":"ContractRulesManager","apiname":"contract_rules","targetObjectId":"contract","triggerTime":"beforeUpdate","isactive":1,"version":"2","triggerSource":"int total = left + right;"}`
	if err := Handle("update", "trigger", []string{project, spec}, &stdout, &stderr, project); err != nil {
		t.Fatalf("update trigger failed: %v", err)
	}
	assertTriggerTestPaths(t, calls, "/api/triggerSetup/saveTrigger")
	encoded := fmt.Sprint(calls[0].Body["triggerSource"])
	decoded, err := url.QueryUnescape(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded != "int total = left + right;" || !strings.Contains(encoded, "%2B") {
		t.Fatalf("trigger source did not survive URLDecoder-compatible encoding: encoded=%q decoded=%q", encoded, decoded)
	}
	if got, ok := calls[0].Body["isactive"].(string); !ok || got != "true" {
		t.Fatalf("numeric-compatible trigger isactive must be string true, got %#v (%T)", calls[0].Body["isactive"], calls[0].Body["isactive"])
	}
}

func TestTriggerPublishPreservesPlusForJavaURLDecoder(t *testing.T) {
	var calls []triggerTestCall
	server := newTriggerTestServer(t, &calls)
	defer server.Close()
	project := newTriggerTestProject(t, server.URL)
	dir := filepath.Join(project, "backend", "triggers", "account", "PlusTrigger")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	source := "// @SOURCE_CONTENT_START\nint total = left + right;\ncounter++;\n// @SOURCE_CONTENT_END\n"
	if err := os.WriteFile(filepath.Join(dir, "PlusTrigger.java"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"id":"trg-plus","apiname":"plus_trigger","targetObjectId":"account","triggerTime":"beforeUpdate","isactive":1,"version":"2"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if err := Handle("publish", "trigger", []string{"account/PlusTrigger", project}, &stdout, &stderr, project); err != nil {
		t.Fatalf("publish trigger failed: %v", err)
	}
	assertTriggerTestPaths(t, calls, "/api/trigger/validate", "/api/triggerSetup/saveTrigger")
	if got := fmt.Sprint(calls[0].Body["triggerSource"]); got != "int total = left + right;\ncounter++;" {
		t.Fatalf("trigger validate must receive raw source, got %q", got)
	}
	encoded := fmt.Sprint(calls[1].Body["triggerSource"])
	decoded, err := url.QueryUnescape(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded != "int total = left + right;\ncounter++;" {
		t.Fatalf("unexpected trigger source round trip: encoded=%q decoded=%q", encoded, decoded)
	}
	if !strings.Contains(encoded, "%2B") {
		t.Fatalf("source plus must be encoded as %%2B for Java URLDecoder: %q", encoded)
	}
	if got, ok := calls[1].Body["isactive"].(string); !ok || got != "true" {
		t.Fatalf("publish trigger must send string isactive=true instead of 1, got %#v (%T)", calls[1].Body["isactive"], calls[1].Body["isactive"])
	}
}

func TestTriggerSaveNormalizesIsActiveValues(t *testing.T) {
	var calls []triggerTestCall
	server := newTriggerTestServer(t, &calls)
	defer server.Close()
	project := newTriggerTestProject(t, server.URL)

	var stdout, stderr bytes.Buffer
	spec := `{"id":"trg-001","name":"ContractRulesManager","targetObjectId":"contract","triggerTime":"beforeUpdate","isactive":1,"triggerSourceEncoded":true,"triggerSource":"return true;"}`
	if err := Handle("update", "trigger", []string{project, spec}, &stdout, &stderr, project); err != nil {
		t.Fatalf("update trigger failed: %v", err)
	}
	assertTriggerTestPaths(t, calls, "/api/triggerSetup/saveTrigger")
	if got, ok := calls[0].Body["isactive"].(string); !ok || got != "true" {
		t.Fatalf("update trigger must send string isactive=true instead of 1, got %#v (%T)", calls[0].Body["isactive"], calls[0].Body["isactive"])
	}

	calls = nil
	stdout.Reset()
	spec = `{"id":"trg-001","name":"ContractRulesManager","targetObjectId":"contract","triggerTime":"beforeUpdate","isActive":"0","triggerSourceEncoded":true,"triggerSource":"return false;"}`
	if err := Handle("save", "trigger", []string{project, spec}, &stdout, &stderr, project); err != nil {
		t.Fatalf("save trigger failed: %v", err)
	}
	assertTriggerTestPaths(t, calls, "/api/triggerSetup/saveTrigger")
	if got, ok := calls[0].Body["isactive"].(string); !ok || got != "false" {
		t.Fatalf("save trigger must send string isactive=false instead of 0, got %#v (%T)", calls[0].Body["isactive"], calls[0].Body["isactive"])
	}
}

func TestNormalizeTriggerIsActiveOnlyReturnsBooleanStrings(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value any
		want  string
	}{
		{name: "string true", value: "true", want: "true"},
		{name: "string false", value: "false", want: "false"},
		{name: "numeric one", value: 1, want: "true"},
		{name: "numeric zero", value: 0, want: "false"},
		{name: "bool true", value: true, want: "true"},
		{name: "bool false", value: false, want: "false"},
		{name: "unknown string coerces true", value: "enabled", want: "true"},
		{name: "blank coerces true", value: "", want: "true"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeTriggerIsActive(tc.value)
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
			if got != "true" && got != "false" {
				t.Fatalf("isactive must only be string true/false, got %q", got)
			}
		})
	}
}

func TestJavaPublishersPreservePlusForJavaURLDecoder(t *testing.T) {
	var calls []triggerTestCall
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		calls = append(calls, triggerTestCall{Path: r.URL.Path, Body: body})
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/ccfag/detail" && len(calls) >= 2 {
			encoded := fmt.Sprint(calls[len(calls)-2].Body["source"])
			decoded, _ := url.QueryUnescape(encoded)
			response, _ := json.Marshal(map[string]any{"result": true, "returnCode": "200", "data": map[string]any{"id": "ok", "source": decoded}})
			_, _ = w.Write(response)
			return
		}
		_, _ = w.Write([]byte(`{"result":true,"returnCode":"200","data":{"id":"ok"}}`))
	}))
	defer server.Close()
	project := newTriggerTestProject(t, server.URL)

	for _, tc := range []struct {
		resource string
		dir      string
		name     string
		path     string
	}{
		{resource: "classes", dir: "classes", name: "PlusClass", path: "/api/ccfag/save"},
		{resource: "timer", dir: "schedule", name: "PlusTimer", path: "/api/ccPeak/save"},
	} {
		dir := filepath.Join(project, "backend", tc.dir, tc.name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		source := "// @SOURCE_CONTENT_START\nint total = left + right;\ncounter++;\n// @SOURCE_CONTENT_END\n"
		if err := os.WriteFile(filepath.Join(dir, tc.name+".java"), []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"id":"asset-1","version":"2"}`), 0o644); err != nil {
			t.Fatal(err)
		}

		calls = nil
		var stdout, stderr bytes.Buffer
		publishArgs := []string{tc.name}
		if tc.resource == "classes" {
			evidencePath := filepath.Join(dir, "validation.json")
			evidence, _ := json.Marshal(classValidationResult{Status: "passed", Valid: true, ClassName: tc.name, SourceSHA256: sourceDigest("int total = left + right;\ncounter++;"), Diagnostics: []classCompileDiagnostic{}})
			if err := os.WriteFile(evidencePath, evidence, 0o644); err != nil {
				t.Fatal(err)
			}
			publishArgs = append(publishArgs, "--validation-evidence", evidencePath)
		}
		if err := Handle("publish", tc.resource, publishArgs, &stdout, &stderr, project); err != nil {
			t.Fatalf("publish %s failed: %v", tc.resource, err)
		}
		expectedPaths := []string{"/api/ccPeak/validate", tc.path}
		if tc.resource == "classes" {
			expectedPaths = []string{"/api/ccfag/validate", tc.path, "/api/ccfag/detail"}
		}
		assertTriggerTestPaths(t, calls, expectedPaths...)
		saveCallIndex := 1
		if tc.resource == "classes" {
			saveCallIndex = 1
		}
		encoded := fmt.Sprint(calls[saveCallIndex].Body["source"])
		decoded, err := url.QueryUnescape(encoded)
		if err != nil {
			t.Fatal(err)
		}
		if decoded != "int total = left + right;\ncounter++;" || !strings.Contains(encoded, "%2B") {
			t.Fatalf("%s source did not survive Java URLDecoder encoding: encoded=%q decoded=%q", tc.resource, encoded, decoded)
		}
	}
}

func TestPublishBlocksOnRemoteValidationFailureBeforeSave(t *testing.T) {
	for _, tc := range []struct {
		resource     string
		dir          string
		name         string
		validatePath string
		savePath     string
	}{
		{resource: "classes", dir: "classes", name: "BrokenClass", validatePath: "/api/ccfag/validate", savePath: "/api/ccfag/save"},
		{resource: "trigger", dir: "triggers", name: "BrokenTrigger", validatePath: "/api/trigger/validate", savePath: "/api/triggerSetup/saveTrigger"},
		{resource: "timer", dir: "schedule", name: "BrokenTimer", validatePath: "/api/ccPeak/validate", savePath: "/api/ccPeak/save"},
	} {
		t.Run(tc.resource, func(t *testing.T) {
			var calls []triggerTestCall
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				calls = append(calls, triggerTestCall{Path: r.URL.Path, Body: body})
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == tc.validatePath {
					_, _ = w.Write([]byte(`{"result":false,"returnInfo":"Compilation failed.\nline 2: cannot find symbol","data":{"valid":false,"message":"Compilation failed.","errors":[{"kind":"ERROR","userLine":2,"message":"cannot find symbol"}],"warnings":[]}}`))
					return
				}
				if r.URL.Path == "/api/ccfag/detail" {
					_, _ = w.Write([]byte(`{"result":true,"returnCode":"200","data":{"id":"asset-1","source":""}}`))
					return
				}
				_, _ = w.Write([]byte(`{"result":true,"returnCode":"200","data":{"id":"asset-1"}}`))
			}))
			defer server.Close()
			project := newTriggerTestProject(t, server.URL)
			dir := filepath.Join(project, "backend", tc.dir, tc.name)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			source := "// @SOURCE_CONTENT_START\nint broken = missing + value;\n// @SOURCE_CONTENT_END\n"
			if err := os.WriteFile(filepath.Join(dir, tc.name+".java"), []byte(source), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"id":"asset-1","apiname":"broken","targetObjectId":"account","triggerTime":"beforeUpdate","version":"2"}`), 0o644); err != nil {
				t.Fatal(err)
			}
			var stdout, stderr bytes.Buffer
			publishArgs := []string{tc.name, project}
			if tc.resource == "classes" {
				evidencePath := filepath.Join(dir, "validation.json")
				evidence, _ := json.Marshal(classValidationResult{Status: "passed", Valid: true, ClassName: tc.name, SourceSHA256: sourceDigest("int broken = missing + value;"), Diagnostics: []classCompileDiagnostic{}})
				if err := os.WriteFile(evidencePath, evidence, 0o644); err != nil {
					t.Fatal(err)
				}
				publishArgs = []string{tc.name, project, "--validation-evidence", evidencePath}
			}
			err := Handle("publish", tc.resource, publishArgs, &stdout, &stderr, project)
			if err == nil || !strings.Contains(err.Error(), "Compilation failed") || !strings.Contains(err.Error(), "responseBody=") {
				t.Fatalf("expected remote validation failure with response body, err=%v stdout=%s", err, stdout.String())
			}
			if !strings.Contains(stdout.String(), `"status": "blocked_remote_validation"`) || !strings.Contains(stdout.String(), `"remoteValidation"`) {
				t.Fatalf("expected structured blocked remote validation output, got %s", stdout.String())
			}
			for _, call := range calls {
				if call.Path == tc.savePath {
					t.Fatalf("save path %s must not be called after failed validation; calls=%#v", tc.savePath, calls)
				}
			}
			assertTriggerTestPaths(t, calls, tc.validatePath)
		})
	}
}

func TestTriggerRejectsAmbiguousNameAndBusinessFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/triggerSetup/getTriggerByCondition":
			_, _ = w.Write([]byte(`{"result":true,"data":{"list":[{"id":"trg-1","name":"Duplicate"},{"id":"trg-2","name":"Duplicate"}]}}`))
		case "/api/triggerSetup/saveTrigger":
			_, _ = w.Write([]byte(`{"result":false,"returnInfo":"compile failed","data":null}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	project := newTriggerTestProject(t, server.URL)

	var stdout, stderr bytes.Buffer
	err := Handle("delete", "trigger", []string{project, "Duplicate"}, &stdout, &stderr, project)
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("expected ambiguous trigger selector failure, got %v", err)
	}

	err = Handle("update", "trigger", []string{project, `{"id":"trg-1","name":"Broken","targetObjectId":"account","triggerTime":"beforeUpdate","triggerSource":"x + y"}`}, &stdout, &stderr, project)
	if err == nil || !strings.Contains(err.Error(), "compile failed") || !strings.Contains(err.Error(), "responseBody=") {
		t.Fatalf("expected trigger business failure with raw response, got %v", err)
	}
}

func newTriggerTestServer(t *testing.T, calls *[]triggerTestCall) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		*calls = append(*calls, triggerTestCall{Path: r.URL.Path, Body: body})
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/triggerSetup/getTriggerByCondition":
			_, _ = w.Write([]byte(`{"result":true,"data":{"list":[{"id":"trg-001","name":"ContractRulesManager","apiname":"contract_rules","targetObjectId":"contract"}],"allpage":"1"}}`))
		case "/api/trigger/newobjtrigger":
			_, _ = w.Write([]byte(`{"result":true,"data":{"trigger":{"id":"trg-001","name":"ContractRulesManager","apiname":"contract_rules","triggerSource":"return true;"}}}`))
		case "/api/trigger/validate":
			_, _ = w.Write([]byte(`{"result":true,"returnCode":"200","data":{"valid":true,"message":"Compilation succeeded.","errors":[],"warnings":[]}}`))
		case "/api/triggerSetup/saveTrigger":
			_, _ = w.Write([]byte(`{"result":true,"data":{"id":"trg-001","apiname":"contract_rules"}}`))
		case "/api/triggerSetup/deleteTrigger":
			_, _ = w.Write([]byte(`{"result":true,"data":null}`))
		default:
			http.NotFound(w, r)
		}
	}))
}

func newTriggerTestProject(t *testing.T, serverURL string) string {
	t.Helper()
	project := t.TempDir()
	content := `{"use":"dev","dev":{"accessToken":"token","setupSvc":"` + serverURL + `","apiSvc":"` + serverURL + `","baseUrl":"` + serverURL + `"}}`
	if err := os.WriteFile(filepath.Join(project, "cloudcc-cli.config.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return project
}

func assertTriggerTestPaths(t *testing.T, calls []triggerTestCall, paths ...string) {
	t.Helper()
	if len(calls) != len(paths) {
		t.Fatalf("expected paths %v, got %#v", paths, calls)
	}
	for i, path := range paths {
		if calls[i].Path != path {
			t.Fatalf("expected call %d path %s, got %#v", i, path, calls[i])
		}
	}
}
