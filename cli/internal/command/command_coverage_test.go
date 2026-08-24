package command

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type commandHTTPCall struct {
	Method      string
	Path        string
	RawQuery    string
	ServiceName string
	Body        map[string]any
}

func TestVersionDocumentationConfigAndProjectCommands(t *testing.T) {
	localCases := []struct {
		name    string
		args    []string
		wantOut string
		wantErr string
	}{
		{name: "help-empty", args: nil, wantOut: "CloudCC CLI Go"},
		{name: "version", args: []string{"--version"}, wantOut: "2.2.19-msapi"},
		{name: "help", args: []string{"help"}, wantOut: "Usage:"},
		{name: "doctor", args: []string{"doctor"}, wantOut: "node/npm: not required"},
		{name: "docs", args: []string{"docs"}, wantOut: "cloudcc doc"},
		{name: "stats", args: []string{"stats"}, wantOut: "Command stats are not collected"},
		{name: "changelog", args: []string{"changelog"}, wantErr: "Low-code metadata shortcuts"},
		{name: "doc-introduction", args: []string{"doc", "platform/overview", "introduction"}, wantOut: "CloudCC"},
		{name: "doc-devguide", args: []string{"doc", "platform/object", "devguide"}, wantOut: "object"},
	}
	for _, tc := range localCases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if exit := Run(tc.args, &stdout, &stderr, t.TempDir()); exit != 0 {
				t.Fatalf("expected exit 0, got %d, stderr=%s", exit, stderr.String())
			}
			if tc.wantOut != "" && !strings.Contains(stdout.String(), tc.wantOut) {
				t.Fatalf("expected stdout to contain %q, got %s", tc.wantOut, stdout.String())
			}
			if tc.wantErr != "" && !strings.Contains(stderr.String(), tc.wantErr) {
				t.Fatalf("expected stderr to contain %q, got %s", tc.wantErr, stderr.String())
			}
		})
	}

	tmp := t.TempDir()
	var stdout, stderr bytes.Buffer
	if exit := Run([]string{"create", "project", "demo"}, &stdout, &stderr, tmp); exit != 0 {
		t.Fatalf("create project failed: %s", stderr.String())
	}
	if _, err := os.Stat(filepath.Join(tmp, "demo", "cloudcc-cli.config.json")); err != nil {
		t.Fatalf("expected project config: %v", err)
	}
	projectConfig := readCommandTestJSON(t, filepath.Join(tmp, "demo", "cloudcc-cli.config.json"))
	projectDev := projectConfig["dev"].(map[string]any)
	projectMetadataService := projectDev["metadataService"].(map[string]any)
	if got := projectMetadataService["url"]; got != "https://dc52.apis.cloudcc.cn/metadata" {
		t.Fatalf("expected public MetadataService URL, got %#v", got)
	}

	configProject := t.TempDir()
	writeCommandTestFile(t, filepath.Join(configProject, "cloudcc-cli.config.json"), `{
	  "use":"dev",
	  "dev":{"accessToken":"dev-token","apiSvc":"http://127.0.0.1/api","setupSvc":"http://127.0.0.1/setup"},
	  "test":{"accessToken":"test-token","apiSvc":"http://127.0.0.1/api","setupSvc":"http://127.0.0.1/setup"}
	}`)
	stdout.Reset()
	stderr.Reset()
	if exit := Run([]string{"get", "config", configProject}, &stdout, &stderr, configProject); exit != 0 {
		t.Fatalf("get config failed: %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "dev-token") {
		t.Fatalf("expected active dev config, got %s", stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	if exit := Run([]string{"use", "config", "test", configProject}, &stdout, &stderr, configProject); exit != 0 {
		t.Fatalf("use config failed: %s", stderr.String())
	}
	root := readCommandTestJSON(t, filepath.Join(configProject, "cloudcc-cli.config.json"))
	if root["use"] != "test" {
		t.Fatalf("expected active env test, got %#v", root["use"])
	}
}

func TestGenericSkillSourceDoesNotContainProjectSpecificAssets(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test file path")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "../../.."))
	upperProjectKey := strings.ToUpper("ty" + "zy")
	lowerProjectKey := strings.ToLower(upperProjectKey)
	var matches []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if entry.IsDir() {
			switch {
			case rel == "tools":
				return filepath.SkipDir
			case strings.HasPrefix(rel, "tools/bin"):
				return filepath.SkipDir
			case rel == ".git":
				return filepath.SkipDir
			}
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := string(data)
		if strings.Contains(text, upperProjectKey) || strings.Contains(text, lowerProjectKey) {
			matches = append(matches, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) > 0 {
		t.Fatalf("generic skill source must not contain project-specific assets: %v", matches)
	}
}

func TestSkillRootConfigDefaultsPublicMetadataService(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test file path")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "../../.."))
	config := readCommandTestJSON(t, filepath.Join(root, "cloudcc-cli.config.json"))
	dev := config["dev"].(map[string]any)
	metadataService := dev["metadataService"].(map[string]any)
	if got := metadataService["url"]; got != "https://dc52.apis.cloudcc.cn/metadata" {
		t.Fatalf("expected skill root public MetadataService URL, got %#v", got)
	}
	if got := dev["executionMode"]; got != "msapi" {
		t.Fatalf("expected skill root executionMode msapi, got %#v", got)
	}
}

func TestOpenAPICommandsUseExpectedServiceNames(t *testing.T) {
	var calls []commandHTTPCall
	server := newCommandCoverageServer(t, &calls)
	defer server.Close()

	projectPath := newCommandCoverageProject(t, server.URL)
	serviceByAction := map[string]string{
		"query":     "cqueryWithRoleRight",
		"pageQuery": "pageQueryWithRoleRight",
		"create":    "insertWithRoleRight",
		"update":    "updateWithRoleRight",
		"delete":    "deleteWithRoleRight",
		"upsert":    "upsertWithRoleRight",
	}
	for action, serviceName := range serviceByAction {
		t.Run(action, func(t *testing.T) {
			calls = nil
			body := `{"objectApiName":"Account","data":{"id":"001","name":"Acme"}}`
			var stdout, stderr bytes.Buffer
			if exit := Run([]string{action, "openapi", projectPath, body}, &stdout, &stderr, projectPath); exit != 0 {
				t.Fatalf("%s openapi failed: %s", action, stderr.String())
			}
			assertCommandCall(t, calls, http.MethodPost, "/openApi/common")
			if calls[0].ServiceName != serviceName {
				t.Fatalf("expected service %s, got %s", serviceName, calls[0].ServiceName)
			}
		})
	}
}

func TestMetadataServiceCommandMatrix(t *testing.T) {
	var calls []commandHTTPCall
	server := newCommandCoverageServer(t, &calls)
	defer server.Close()
	t.Setenv("CLOUDCC_METADATA_SERVICE_URL", server.URL)

	projectPath := newCommandCoverageProject(t, server.URL)
	cases := []struct {
		name   string
		args   []string
		method string
		path   string
	}{
		{"capabilities", []string{"capabilities", "msapi", projectPath}, http.MethodGet, "/metadata/v1/capabilities"},
		{"capability-alias", []string{"capability", "metadata", projectPath}, http.MethodGet, "/metadata/v1/capabilities"},
		{"scan-summary", []string{"scan", "msapi", projectPath}, http.MethodGet, "/metadata/v1/scans/summary"},
		{"scan-standard-catalog", []string{"scan", "msapi", projectPath, "standard-catalog"}, http.MethodGet, "/metadata/v1/scans/standard-catalog"},
		{"scan-compare", []string{"scan", "msapi", projectPath, "compare", `{"source":"unit","checks":[]}`}, http.MethodPost, "/metadata/v1/scans:compare"},
		{"resolve", []string{"resolve", "msapi", projectPath, `{"references":[]}`}, http.MethodPost, "/metadata/v1/references:resolve"},
		{"references-alias", []string{"references", "metadata", projectPath, `{"references":[]}`}, http.MethodPost, "/metadata/v1/references:resolve"},
		{"normalize", []string{"normalize", "msapi", projectPath, "objects", `{"label":"A"}`}, http.MethodPost, "/metadata/v1/intents:normalize"},
		{"validate", []string{"validate", "msapi", projectPath, "objects", `{"label":"A"}`}, http.MethodPost, "/metadata/v1/intents:validate"},
		{"plan", []string{"plan", "msapi", projectPath, "objects", `{"label":"A"}`, "create"}, http.MethodPost, "/metadata/v1/plans"},
		{"metadata-domain-alias", []string{"plan", "objects", `{"label":"A"}`}, http.MethodPost, "/metadata/v1/plans"},
		{"apply", []string{"apply", "msapi", projectPath, "plan-001", `{"reason":"unit"}`}, http.MethodPost, "/metadata/v1/plans/plan-001:apply"},
		{"operation", []string{"operation", "msapi", projectPath, "op-001"}, http.MethodGet, "/metadata/v1/operations/op-001"},
		{"get-operation-alias", []string{"get", "msapi", projectPath, "op-001"}, http.MethodGet, "/metadata/v1/operations/op-001"},
		{"changes", []string{"changes", "msapi", projectPath, "op-001"}, http.MethodGet, "/metadata/v1/operations/op-001/changes"},
		{"rollback-plan", []string{"rollback-plan", "msapi", projectPath, "op-001"}, http.MethodPost, "/metadata/v1/operations/op-001:rollback-plan"},
		{"rollback", []string{"rollback", "msapi", projectPath, "op-001"}, http.MethodPost, "/metadata/v1/operations/op-001:rollback"},
		{"mutate", []string{"mutate", "msapi", projectPath, "objects", "patch", `{"id":"obj-001"}`}, http.MethodPost, "/metadata/v1/objects:mutate"},
		{"draft-create", []string{"draft-create", "msapi", projectPath, "objects", "create", `{"label":"A"}`}, http.MethodPost, "/metadata/v1/drafts"},
		{"draft-update", []string{"draft-update", "msapi", projectPath, "draft-001", `{"label":"B"}`}, http.MethodPatch, "/metadata/v1/drafts/draft-001"},
		{"draft-validate", []string{"draft-validate", "msapi", projectPath, "draft-001"}, http.MethodPost, "/metadata/v1/drafts/draft-001:validate"},
		{"draft-plan", []string{"draft-plan", "msapi", projectPath, "draft-001"}, http.MethodPost, "/metadata/v1/drafts/draft-001:plan"},
		{"draft-delete", []string{"draft-delete", "msapi", projectPath, "draft-001"}, http.MethodDelete, "/metadata/v1/drafts/draft-001"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls = nil
			var stdout, stderr bytes.Buffer
			if exit := Run(tc.args, &stdout, &stderr, projectPath); exit != 0 {
				t.Fatalf("command failed: args=%v stderr=%s", tc.args, stderr.String())
			}
			assertCommandCall(t, calls, tc.method, tc.path)
		})
	}

	calls = nil
	var stdout, stderr bytes.Buffer
	if exit := Run([]string{"plan", "msapi", projectPath, "classes", `{"name":"Demo"}`}, &stdout, &stderr, projectPath); exit == 0 {
		t.Fatal("expected high-code metadata plan to fail closed")
	}
	if len(calls) != 0 {
		t.Fatalf("high-code metadata guard should fail before HTTP, got calls %#v", calls)
	}
	if !strings.Contains(stderr.String(), "high-code writes stay on existing CloudCC resource/API paths") {
		t.Fatalf("expected high-code guard message, got %s", stderr.String())
	}
}

func TestLowCodeShortcutCommandMatrix(t *testing.T) {
	var calls []commandHTTPCall
	server := newCommandCoverageServer(t, &calls)
	defer server.Close()
	t.Setenv("CLOUDCC_METADATA_SERVICE_URL", server.URL)

	projectPath := newCommandCoverageProject(t, server.URL)
	resources := []string{
		"application", "button", "customSetting", "dupeCatcher", "fields", "globalSelectList",
		"identityProvider", "menu", "object", "pagelayout", "permission", "profile",
		"recordType", "role", "sharingRule", "singleSignOn", "validationRule", "view",
	}
	for _, resource := range resources {
		t.Run(resource+"-read", func(t *testing.T) {
			calls = nil
			var stdout, stderr bytes.Buffer
			args := []string{"get", resource, projectPath}
			if resource == "fields" || resource == "pagelayout" || resource == "recordType" {
				args = append(args, "obj-001")
			}
			if exit := Run(args, &stdout, &stderr, projectPath); exit != 0 {
				t.Fatalf("read shortcut failed: %s", stderr.String())
			}
			if resource == "fields" {
				assertCommandCall(t, calls, http.MethodGet, "/metadata/v1/fields")
				if calls[0].RawQuery != "object=obj-001" {
					t.Fatalf("expected object selector query, got %#v", calls[0])
				}
			} else if resource == "pagelayout" {
				assertCommandCall(t, calls, http.MethodGet, "/metadata/v1/layouts")
				if calls[0].RawQuery != "object=obj-001" {
					t.Fatalf("expected object selector query, got %#v", calls[0])
				}
			} else if resource == "recordType" {
				assertCommandCall(t, calls, http.MethodGet, "/metadata/v1/record-types")
				if calls[0].RawQuery != "object=obj-001" {
					t.Fatalf("expected object selector query, got %#v", calls[0])
				}
			} else if resource == "profile" {
				assertCommandCall(t, calls, http.MethodGet, "/metadata/v1/profiles")
			} else if resource == "globalSelectList" {
				assertCommandCall(t, calls, http.MethodGet, "/metadata/v1/global-select-lists")
			} else if resource == "permission" {
				assertCommandCall(t, calls, http.MethodGet, "/metadata/v1/permission-sets")
			} else if resource == "role" {
				assertCommandCall(t, calls, http.MethodGet, "/metadata/v1/roles")
			} else if resource == "sharingRule" {
				assertCommandCall(t, calls, http.MethodGet, "/metadata/v1/sharing-rules")
			} else {
				assertCommandCall(t, calls, http.MethodGet, "/metadata/v1/scans/standard-catalog")
			}
		})
		t.Run(resource+"-write", func(t *testing.T) {
			calls = nil
			var stdout, stderr bytes.Buffer
			if exit := Run([]string{"update", resource, projectPath, `{"id":"unit-id"}`}, &stdout, &stderr, projectPath); exit != 0 {
				t.Fatalf("write shortcut failed: %s", stderr.String())
			}
			assertCommandCall(t, calls, http.MethodPost, "/metadata/v1/plans")
		})
	}

	calls = nil
	var stdout, stderr bytes.Buffer
	if exit := Run([]string{"purge", "object", projectPath, "obj-001", "--execute", "--approval", "CLOUDCC_OBJECT_PHYSICAL_DELETE_APPROVED"}, &stdout, &stderr, projectPath); exit != 0 {
		t.Fatalf("purge object shortcut should create a plan, got stderr=%s", stderr.String())
	}
	assertCommandCall(t, calls, http.MethodPost, "/metadata/v1/plans")
}

func TestProfileShortcutsPreserveFiltersResolveUniqueDetailAndPlanDeleteByID(t *testing.T) {
	var calls []commandHTTPCall
	server := newCommandCoverageServer(t, &calls)
	defer server.Close()
	t.Setenv("CLOUDCC_METADATA_SERVICE_URL", server.URL)
	projectPath := newCommandCoverageProject(t, server.URL)

	var stdout, stderr bytes.Buffer
	if exit := Run([]string{"get", "profile", projectPath, "销售 + 服务"}, &stdout, &stderr, projectPath); exit != 0 {
		t.Fatalf("get profile failed: %s", stderr.String())
	}
	assertCommandCall(t, calls, http.MethodGet, "/metadata/v1/profiles")
	if calls[0].RawQuery != "filter=%E9%94%80%E5%94%AE+%2B+%E6%9C%8D%E5%8A%A1" {
		t.Fatalf("profile filter was not preserved: %#v", calls[0])
	}

	calls = nil
	stdout.Reset()
	stderr.Reset()
	if exit := Run([]string{"detail", "profile", projectPath, "sales_profile"}, &stdout, &stderr, projectPath); exit != 0 {
		t.Fatalf("detail profile failed: %s", stderr.String())
	}
	assertCommandCalls(t, calls, http.MethodGet, []string{"/metadata/v1/profiles", "/metadata/v1/profiles/profile-001"})
	if calls[0].RawQuery != "selector=sales_profile" {
		t.Fatalf("profile selector was not preserved: %#v", calls[0])
	}

	calls = nil
	stdout.Reset()
	stderr.Reset()
	if exit := Run([]string{"delete", "profile", projectPath, "销售简档"}, &stdout, &stderr, projectPath); exit != 0 {
		t.Fatalf("delete profile failed: %s", stderr.String())
	}
	if len(calls) != 2 || calls[0].Path != "/metadata/v1/profiles" || calls[1].Path != "/metadata/v1/plans" {
		t.Fatalf("expected unique resolution then plan, got %#v", calls)
	}
	spec, _ := calls[1].Body["spec"].(map[string]any)
	if calls[1].Body["domain"] != "profiles" || calls[1].Body["operation"] != "delete" || spec["id"] != "profile-001" {
		t.Fatalf("profile delete must plan by resolved id, got %#v", calls[1].Body)
	}
}

func TestProfileDestructiveSelectorRejectsDuplicatesBeforePlan(t *testing.T) {
	var calls []commandHTTPCall
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, commandHTTPCall{Method: r.Method, Path: r.URL.Path, RawQuery: r.URL.RawQuery})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"count":2,"profiles":[{"id":"profile-001","name":"销售简档"},{"id":"profile-002","name":"销售简档"}]}`))
	}))
	defer server.Close()
	t.Setenv("CLOUDCC_METADATA_SERVICE_URL", server.URL)
	projectPath := newCommandCoverageProject(t, server.URL)

	var stdout, stderr bytes.Buffer
	if exit := Run([]string{"delete", "profile", projectPath, "销售简档"}, &stdout, &stderr, projectPath); exit == 0 {
		t.Fatal("duplicate profile selector must fail closed")
	}
	if len(calls) != 1 || calls[0].Path != "/metadata/v1/profiles" {
		t.Fatalf("duplicate selector must fail before plan: %#v", calls)
	}
	if !strings.Contains(stderr.String(), "matched 2 profiles") || !strings.Contains(stderr.String(), "profile-001, profile-002") {
		t.Fatalf("expected actionable duplicate error, got %s", stderr.String())
	}
}

func TestLowCodeShortcutActionMatrix(t *testing.T) {
	var calls []commandHTTPCall
	server := newCommandCoverageServer(t, &calls)
	defer server.Close()
	t.Setenv("CLOUDCC_METADATA_SERVICE_URL", server.URL)

	projectPath := newCommandCoverageProject(t, server.URL)
	readActions := [][]string{
		{"get", "object", projectPath},
		{"detail", "object", projectPath, "obj-001"},
		{"getList", "recordType", projectPath, "001"},
		{"newInfo", "recordType", projectPath, "001"},
		{"editInfo", "recordType", projectPath, "rt-001"},
		{"validDelete", "recordType", projectPath, "rt-001"},
	}
	for _, args := range readActions {
		t.Run(strings.Join(args[:2], "-"), func(t *testing.T) {
			calls = nil
			var stdout, stderr bytes.Buffer
			if exit := Run(args, &stdout, &stderr, projectPath); exit != 0 {
				t.Fatalf("read action failed args=%v stderr=%s", args, stderr.String())
			}
			if args[0] == "detail" && args[1] == "object" {
				assertCommandCall(t, calls, http.MethodGet, "/metadata/v1/objects/obj-001")
			} else if args[1] == "recordType" && (args[0] == "getList" || args[0] == "newInfo") {
				assertCommandCall(t, calls, http.MethodGet, "/metadata/v1/record-types")
				if calls[0].RawQuery != "object=001" {
					t.Fatalf("expected record-type object selector query, got %#v", calls[0])
				}
			} else if args[1] == "recordType" {
				assertCommandCall(t, calls, http.MethodGet, "/metadata/v1/record-types/rt-001")
			} else {
				assertCommandCall(t, calls, http.MethodGet, "/metadata/v1/scans/standard-catalog")
			}
		})
	}

	writeActions := [][]string{
		{"create", "menu", projectPath, `{"id":"unit-id"}`},
		{"update", "menu", projectPath, `{"id":"unit-id"}`},
		{"save", "menu", projectPath, `{"id":"unit-id"}`},
		{"modify", "customSetting", projectPath, `{"id":"unit-id"}`},
		{"editSave", "recordType", projectPath, `{"id":"unit-id"}`},
		{"assign", "permission", projectPath, `{"id":"unit-id"}`},
		{"add", "permission", projectPath, `{"id":"unit-id"}`},
		{"remove", "permission", projectPath, `{"id":"unit-id"}`},
		{"delete", "menu", projectPath, "unit-id"},
		{"purge", "object", projectPath, "obj-001"},
	}
	for _, args := range writeActions {
		t.Run(strings.Join(args[:2], "-"), func(t *testing.T) {
			calls = nil
			var stdout, stderr bytes.Buffer
			if exit := Run(args, &stdout, &stderr, projectPath); exit != 0 {
				t.Fatalf("write action failed args=%v stderr=%s", args, stderr.String())
			}
			assertCommandCall(t, calls, http.MethodPost, "/metadata/v1/plans")
		})
	}
}

func TestHighCodeAndRemainingResourceCommands(t *testing.T) {
	var calls []commandHTTPCall
	server := newCommandCoverageServer(t, &calls)
	defer server.Close()
	projectPath := newCommandCoverageProject(t, server.URL)

	localCreates := [][]string{
		{"create", "classes", "DemoClass"},
		{"create", "triggers", "DemoTrigger"},
		{"create", "timer", "DemoTimer"},
		{"create", "script", "Account/DemoScript"},
		{"create", "html", "demo_html"},
		{"create", "pagecomponent", "cc-demo"},
	}
	for _, args := range localCreates {
		var stdout, stderr bytes.Buffer
		if exit := Run(args, &stdout, &stderr, projectPath); exit != 0 {
			t.Fatalf("local create failed args=%v stderr=%s", args, stderr.String())
		}
	}
	writeCommandTestFile(t, filepath.Join(projectPath, "frontend", "build", "component-cc-demo.umd.min.js"), `window.CCDemo=true;`)
	classFile := filepath.Join(projectPath, "backend", "classes", "DemoClass", "DemoClass.java")
	validatedClassSource := strings.TrimSpace(markedSourceFromTestFile(t, classFile))
	classSourceSum := sha256.Sum256([]byte(validatedClassSource))
	validationEvidencePath := filepath.Join(projectPath, "backend", "classes", "DemoClass", "validation.json")
	writeCommandTestFile(t, validationEvidencePath, fmt.Sprintf(`{"status":"passed","valid":true,"className":"DemoClass","sourceSha256":"%x","diagnostics":[]}`, classSourceSum))
	httpCases := []struct {
		name   string
		args   []string
		method string
		path   string
	}{
		{"publish-classes", []string{"publish", "classes", "DemoClass", "--validation-evidence", validationEvidencePath}, http.MethodPost, "/api/ccfag/save"},
		{"get-classes", []string{"get", "classes", projectPath}, http.MethodPost, "/api/ccfag/list"},
		{"detail-classes", []string{"detail", "classes", projectPath, "DemoClass"}, http.MethodPost, "/api/ccfag/detail"},
		{"pull-classes", []string{"pull", "classes", projectPath, "DemoClass"}, http.MethodPost, "/api/ccfag/detail"},
		{"pull-list-classes", []string{"pullList", "classes", projectPath, "DemoClass"}, http.MethodPost, "/api/ccfag/detail"},
		{"delete-classes", []string{"delete", "classes", projectPath, "cls-001"}, http.MethodPost, "/api/ccfag/delete"},
		{"publish-triggers", []string{"publish", "triggers", "DemoTrigger"}, http.MethodPost, "/api/triggerSetup/saveTrigger"},
		{"get-triggers", []string{"get", "triggers", projectPath}, http.MethodPost, "/api/triggerSetup/getTriggerByCondition"},
		{"detail-triggers", []string{"detail", "triggers", projectPath, "DemoTrigger"}, http.MethodPost, "/api/trigger/newobjtrigger"},
		{"delete-triggers", []string{"delete", "triggers", projectPath, "trg-001"}, http.MethodPost, "/api/triggerSetup/deleteTrigger"},
		{"publish-timer", []string{"publish", "timer", "DemoTimer"}, http.MethodPost, "/api/ccPeak/save"},
		{"get-timer", []string{"get", "timer", projectPath}, http.MethodPost, "/api/ccPeak/list"},
		{"detail-timer", []string{"detail", "timer", projectPath, "DemoTimer"}, http.MethodPost, "/api/ccPeak/detail"},
		{"delete-timer", []string{"delete", "timer", projectPath, "timer-001"}, http.MethodPost, "/api/ccPeak/delete"},
		{"publish-script", []string{"publish", "script", "Account/DemoScript"}, http.MethodPost, "/devconsole/script/saveClientScript"},
		{"get-script", []string{"get", "script", projectPath}, http.MethodPost, "/devconsole/script/pageClientScript"},
		{"detail-script", []string{"detail", "script", projectPath, `{"name":"DemoScript"}`}, http.MethodPost, "/devconsole/script/pageClientScript"},
		{"delete-script", []string{"delete", "script", projectPath, `{"id":"script-001"}`}, http.MethodPost, "/devconsole/script/pageClientScript"},
		{"publish-html", []string{"publish", "html", "demo_html", projectPath}, http.MethodPost, "/devconsole/htmlComponent/saveHtmlComponent"},
		{"create-static-resource", []string{"create", "staticResource", "demo_resource", "demo.js"}, http.MethodPost, "/api/staticResource/save"},
		{"get-static-resource", []string{"get", "staticResource", projectPath}, http.MethodPost, "/api/staticResource/list"},
		{"detail-static-resource", []string{"detail", "staticResource", projectPath, `{"id":"sr-001"}`}, http.MethodPost, "/api/staticResource/detail"},
		{"count-static-resource", []string{"count", "staticResource", projectPath}, http.MethodPost, "/api/staticResource/count"},
		{"delete-static-resource", []string{"delete", "staticResource", projectPath, "sr-001"}, http.MethodPost, "/api/staticResource/delete"},
		{"get-schedule-job", []string{"get", "scheduleJob", projectPath}, http.MethodPost, "/api/schedulAbleprg/list"},
		{"get-list-schedule-job", []string{"getList", "scheduleJob", projectPath}, http.MethodPost, "/api/lookup/getLookupData"},
		{"get-schedule-alias", []string{"get", "schedule", projectPath}, http.MethodPost, "/api/ccPeak/list"},
		{"detail-schedule-job", []string{"detail", "scheduleJob", projectPath, `{"id":"job-001"}`}, http.MethodPost, "/api/schedulAbleprg/edit"},
		{"create-schedule-job", []string{"create", "scheduleJob", projectPath, `{"name":"job"}`}, http.MethodPost, "/api/schedulAbleprg/save"},
		{"delete-schedule-job", []string{"delete", "scheduleJob", projectPath, "job-001"}, http.MethodPost, "/api/schedulAbleprg/delete"},
		{"get-brief", []string{"get", "brief", projectPath}, http.MethodPost, "/api/customObject/newPage"},
		{"get-user", []string{"get", "user", projectPath}, http.MethodPost, "/api/user/list"},
		{"view-user", []string{"view", "user", projectPath, `{"id":"user-001"}`}, http.MethodPost, "/api/user/view"},
		{"create-user", []string{"create", "user", projectPath, `{"name":"user"}`}, http.MethodPost, "/api/user/save"},
		{"update-user", []string{"update", "user", projectPath, `{"id":"user-001"}`}, http.MethodPost, "/api/user/save"},
		{"delete-user", []string{"delete", "user", projectPath, "user-001"}, http.MethodPost, "/api/user/delete"},
		{"publish-pagecomponent", []string{"publish", "pagecomponent", "cc-demo", projectPath}, http.MethodPost, "/devconsole/custom/pc/1.0/post/insertCustomComp"},
		{"get-pagecomponent", []string{"get", "pagecomponent", projectPath}, http.MethodPost, "/devconsole/custom/pc/1.0/post/pageCustomComp"},
		{"pull-pagecomponent", []string{"pull", "pagecomponent", "pc-001", projectPath}, http.MethodPost, "/devconsole/custom/pc/1.0/post/detailCustomComp"},
		{"delete-pagecomponent", []string{"delete", "pagecomponent", "pc-001", projectPath}, http.MethodPost, "/devconsole/custom/pc/1.0/post/deleteCustomComp"},
	}
	for _, tc := range httpCases {
		t.Run(tc.name, func(t *testing.T) {
			calls = nil
			var stdout, stderr bytes.Buffer
			if exit := Run(tc.args, &stdout, &stderr, projectPath); exit != 0 {
				t.Fatalf("command failed args=%v stderr=%s", tc.args, stderr.String())
			}
			if tc.name == "publish-classes" {
				assertCommandCalls(t, calls, tc.method, []string{"/api/ccfag/list", "/api/ccfag/validate", "/api/ccfag/save", "/api/ccfag/detail"})
				if !strings.Contains(stdout.String(), `"status": "published_and_verified"`) {
					t.Fatalf("expected verified class publish output, got %s", stdout.String())
				}
			} else if tc.name == "publish-triggers" {
				assertCommandCalls(t, calls, tc.method, []string{"/api/trigger/validate", "/api/triggerSetup/saveTrigger"})
			} else if tc.name == "publish-timer" {
				assertCommandCalls(t, calls, tc.method, []string{"/api/ccPeak/validate", "/api/ccPeak/save"})
			} else if tc.name == "publish-pagecomponent" {
				assertCommandCalls(t, calls, tc.method, []string{
					"/devconsole/custom/pc/1.0/post/insertCustomComp",
					"/devconsole/custom/pc/1.0/post/pageCustomPage",
					"/devconsole/custom/pc/1.0/post/detailCustomPage",
				})
				if !strings.Contains(stdout.String(), `"status":"stale_component_reference"`) {
					t.Fatalf("expected publish to report stale customPage reference, got %s", stdout.String())
				}
			} else if tc.name == "detail-triggers" {
				assertCommandCalls(t, calls, tc.method, []string{
					"/api/triggerSetup/getTriggerByCondition",
					"/api/trigger/newobjtrigger",
				})
			} else if tc.name == "delete-triggers" {
				assertCommandCalls(t, calls, tc.method, []string{
					"/api/triggerSetup/getTriggerByCondition",
					"/api/triggerSetup/deleteTrigger",
				})
			} else {
				assertCommandCall(t, calls, tc.method, tc.path)
				if tc.name == "get-list-schedule-job" {
					if calls[0].Body["prefix"] != "ccp" || calls[0].Body["page"] != float64(1) || calls[0].Body["pageSize"] != float64(9999) {
						t.Fatalf("expected scheduleJob class lookup defaults, got %#v", calls[0].Body)
					}
				}
			}
		})
	}

	var stdout, stderr bytes.Buffer
	if exit := Run([]string{"detail", "pagecomponent", "cc-demo", "", projectPath}, &stdout, &stderr, projectPath); exit != 0 {
		t.Fatalf("local pagecomponent detail failed: %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), `"fromLocal":true`) {
		t.Fatalf("expected local pagecomponent detail, got %s", stdout.String())
	}

	for _, args := range [][]string{{"get", "jsp"}, {"get", "site"}, {"get", "skill"}} {
		stdout.Reset()
		stderr.Reset()
		if exit := Run(args, &stdout, &stderr, projectPath); exit == 0 {
			t.Fatalf("expected deferred command to fail: %v", args)
		}
		if !strings.Contains(stderr.String(), "deferred") {
			t.Fatalf("expected deferred message for %v, got %s", args, stderr.String())
		}
	}
}

func markedSourceFromTestFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	start := strings.Index(text, "// @SOURCE_CONTENT_START")
	if start < 0 {
		return text
	}
	start = strings.Index(text[start:], "\n") + start + 1
	end := strings.Index(text, "// @SOURCE_CONTENT_END")
	if end < 0 || end < start {
		return text
	}
	return text[start:end]
}

func TestCustomPageHighCodeCommandsUseDevconsoleEnvelope(t *testing.T) {
	var calls []commandHTTPCall
	server := newCommandCoverageServer(t, &calls)
	defer server.Close()
	projectPath := newCommandCoverageProject(t, server.URL)

	cases := []struct {
		name       string
		args       []string
		wantPath   string
		wantOutput string
	}{
		{
			name:       "get customPage filtered by pageApi",
			args:       []string{"get", "customPage", projectPath, "workbench_page"},
			wantPath:   "/devconsole/custom/pc/1.0/post/pageCustomPage",
			wantOutput: `"pageApi":"workbench_page"`,
		},
		{
			name:       "detail customPage summarizes component refs",
			args:       []string{"detail", "customPage", projectPath, "workbench_page"},
			wantPath:   "/devconsole/custom/pc/1.0/post/detailCustomPage",
			wantOutput: `"componentRefs":[{"comId":"pc-old","embedded":false,"name":"component-cc-demo","workspaceUrl":"https://old.example/app"}]`,
		},
		{
			name:       "update customPage saves then reads back",
			args:       []string{"update", "customPage", projectPath, "workbench_page", `{"pageContent":[{"name":"component-cc-demo","comId":"pc-new","embedded":true}],"canvasStyleData":{"width":1440},"compList":[{"id":"pc-new","compUniName":"component-cc-demo"}]}`},
			wantPath:   "/devconsole/custom/pc/1.0/post/detailCustomPage",
			wantOutput: `"status":"updated"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls = nil
			var stdout, stderr bytes.Buffer
			if exit := Run(tc.args, &stdout, &stderr, projectPath); exit != 0 {
				t.Fatalf("command failed args=%v stderr=%s", tc.args, stderr.String())
			}
			if len(calls) == 0 || calls[0].Path != tc.wantPath {
				t.Fatalf("expected first call to %s, got %#v", tc.wantPath, calls)
			}
			head, _ := calls[0].Body["head"].(map[string]any)
			if head["source"] != "lightning-devconsole" || head["appType"] != "lightning-devconsole" {
				t.Fatalf("customPage commands must use lightning-devconsole envelope, got %#v", head)
			}
			if !strings.Contains(stdout.String(), tc.wantOutput) {
				t.Fatalf("expected stdout to contain %s, got %s", tc.wantOutput, stdout.String())
			}
		})
	}

	calls = nil
	var stdout, stderr bytes.Buffer
	exit := Run([]string{"update", "customPage", projectPath, "workbench_page", `{"pageLabel":"Workbench","pageApi":"workbench_page","compList":[{"id":"pc-new","compName":"wrong"}]}`}, &stdout, &stderr, projectPath)
	if exit == 0 {
		t.Fatal("expected invalid compList to fail before HTTP")
	}
	if len(calls) != 0 {
		t.Fatalf("invalid compList should not call HTTP, got %#v", calls)
	}
	if !strings.Contains(stderr.String(), "compUniName") {
		t.Fatalf("expected compUniName validation error, got %s", stderr.String())
	}
}

func TestCustomPageUpdateMergesCurrentIDAndNormalizesStringBackedFields(t *testing.T) {
	var calls []commandHTTPCall
	server := newCommandCoverageServer(t, &calls)
	defer server.Close()
	projectPath := newCommandCoverageProject(t, server.URL)

	var stdout, stderr bytes.Buffer
	exit := Run([]string{"update", "customPage", projectPath, "workbench_page", `{"pageContent":[{"name":"component-cc-demo","comId":"pc-new","componentInfo":{"id":"pc-new","component":"component-cc-demo"}}],"canvasStyleData":{"width":1440},"compList":[{"id":"pc-new","compUniName":"component-cc-demo"}]}`}, &stdout, &stderr, projectPath)
	if exit != 0 {
		t.Fatalf("update customPage failed: %s", stderr.String())
	}
	var saveBody map[string]any
	for _, call := range calls {
		if call.Path == "/devconsole/custom/pc/1.0/post/insertCustomPage" {
			saveBody, _ = call.Body["body"].(map[string]any)
			break
		}
	}
	if saveBody == nil {
		t.Fatalf("expected insertCustomPage call, got %#v", calls)
	}
	if saveBody["id"] != "page-001" || saveBody["pageLabel"] != "Workbench" || saveBody["pageApi"] != "workbench_page" {
		t.Fatalf("expected persisted identity fields to be merged, got %#v", saveBody)
	}
	if _, ok := saveBody["pageContent"].(string); !ok {
		t.Fatalf("pageContent must be a JSON string on the devconsole wire, got %#v", saveBody["pageContent"])
	}
	if _, ok := saveBody["canvasStyleData"].(string); !ok {
		t.Fatalf("canvasStyleData must be a JSON string on the devconsole wire, got %#v", saveBody["canvasStyleData"])
	}
	if _, ok := saveBody["compList"].([]any); !ok {
		t.Fatalf("compList must remain an array on the devconsole wire, got %#v", saveBody["compList"])
	}
}

func TestCustomPageDetailPrefersAccessTokenAndSendsSingleIdentifier(t *testing.T) {
	var calls []commandHTTPCall
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		calls = append(calls, commandHTTPCall{Method: r.Method, Path: r.URL.Path, Body: body})
		w.Header().Set("Content-Type", "application/json")
		head, _ := body["head"].(map[string]any)
		requestBody, _ := body["body"].(map[string]any)
		if got := head["accessToken"]; got != "cloud-token" {
			_, _ = w.Write([]byte(`{"result":true,"returnCode":"500","returnInfo":"accessToken无效"}`))
			return
		}
		if _, hasID := requestBody["id"]; hasID {
			_, _ = w.Write([]byte(`{"result":true,"returnCode":"500","returnInfo":"页面加载异常"}`))
			return
		}
		if requestBody["pageApi"] != "customer_interaction_workbench" {
			_, _ = w.Write([]byte(`{"result":true,"returnCode":"500","returnInfo":"missing pageApi"}`))
			return
		}
		_, _ = w.Write([]byte(`{"result":true,"returnCode":"200","data":{"id":"507f1f77bcf86cd799439011","pageLabel":"客户工作台","pageApi":"customer_interaction_workbench","pageContent":[{"name":"component-workbench","comId":"6a4db950e4b0a577cbba1eca"}]}}`))
	}))
	defer server.Close()
	projectPath := newCommandCoverageProject(t, server.URL)

	var stdout, stderr bytes.Buffer
	if exit := Run([]string{"detail", "customPage", projectPath, "customer_interaction_workbench"}, &stdout, &stderr, projectPath); exit != 0 {
		t.Fatalf("detail customPage failed: %s", stderr.String())
	}
	if len(calls) != 1 {
		t.Fatalf("expected one HTTP call, got %#v", calls)
	}
	body, _ := calls[0].Body["body"].(map[string]any)
	if _, hasID := body["id"]; hasID {
		t.Fatalf("pageApi lookup must not also send id, got %#v", body)
	}
	if body["pageApi"] != "customer_interaction_workbench" {
		t.Fatalf("expected pageApi lookup body, got %#v", body)
	}
	if !strings.Contains(stdout.String(), `"pageApi":"customer_interaction_workbench"`) {
		t.Fatalf("expected detail output, got %s", stdout.String())
	}
}

func TestCustomPageDetailSendsOnlyIDForObjectID(t *testing.T) {
	var calls []commandHTTPCall
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		calls = append(calls, commandHTTPCall{Method: r.Method, Path: r.URL.Path, Body: body})
		w.Header().Set("Content-Type", "application/json")
		requestBody, _ := body["body"].(map[string]any)
		if _, hasPageAPI := requestBody["pageApi"]; hasPageAPI {
			_, _ = w.Write([]byte(`{"result":true,"returnCode":"500","returnInfo":"页面加载异常"}`))
			return
		}
		if requestBody["id"] != "507f1f77bcf86cd799439011" {
			_, _ = w.Write([]byte(`{"result":true,"returnCode":"500","returnInfo":"missing id"}`))
			return
		}
		_, _ = w.Write([]byte(`{"result":true,"returnCode":"200","data":{"id":"507f1f77bcf86cd799439011","pageLabel":"客户工作台","pageApi":"customer_interaction_workbench"}}`))
	}))
	defer server.Close()
	projectPath := newCommandCoverageProject(t, server.URL)

	var stdout, stderr bytes.Buffer
	if exit := Run([]string{"detail", "customPage", projectPath, "507f1f77bcf86cd799439011"}, &stdout, &stderr, projectPath); exit != 0 {
		t.Fatalf("detail customPage by id failed: %s", stderr.String())
	}
	body, _ := calls[0].Body["body"].(map[string]any)
	if _, hasPageAPI := body["pageApi"]; hasPageAPI {
		t.Fatalf("id lookup must not also send pageApi, got %#v", body)
	}
	if body["id"] != "507f1f77bcf86cd799439011" {
		t.Fatalf("expected id lookup body, got %#v", body)
	}
}

func TestCustomPageFailureHonorsReturnCodeBeforeResult(t *testing.T) {
	var calls []commandHTTPCall
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		calls = append(calls, commandHTTPCall{Method: r.Method, Path: r.URL.Path, Body: body})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":true,"returnCode":"500","returnInfo":"accessToken无效"}`))
	}))
	defer server.Close()
	projectPath := newCommandCoverageProject(t, server.URL)

	var stdout, stderr bytes.Buffer
	exit := Run([]string{"detail", "customPage", projectPath, "customer_interaction_workbench"}, &stdout, &stderr, projectPath)
	if exit == 0 {
		t.Fatalf("expected returnCode 500 to fail, stdout=%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "accessToken无效") {
		t.Fatalf("expected CloudCC returnInfo in stderr, got %s", stderr.String())
	}
}

func TestCustomPageSaveFailureIncludesRawResponseBody(t *testing.T) {
	var calls []commandHTTPCall
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		calls = append(calls, commandHTTPCall{Method: r.Method, Path: r.URL.Path, Body: body})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":true,"returnCode":"500","returnInfo":"系统发生异常","debugId":"save-001","data":{"reason":"compList envelope rejected"}}`))
	}))
	defer server.Close()
	projectPath := newCommandCoverageProject(t, server.URL)

	var stdout, stderr bytes.Buffer
	exit := Run([]string{"update", "customPage", projectPath, "customer_interaction_workbench", `{"pageLabel":"客户工作台","pageApi":"customer_interaction_workbench","pageContent":[{"name":"component-workbench","comId":"pc-new","componentInfo":{"id":"pc-new","component":"component-workbench"}}],"compList":[{"id":"pc-new","compUniName":"component-workbench"}]}`}, &stdout, &stderr, projectPath)
	if exit == 0 {
		t.Fatalf("expected save failure, stdout=%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "系统发生异常") || !strings.Contains(stderr.String(), "responseBody=") || !strings.Contains(stderr.String(), `"debugId":"save-001"`) {
		t.Fatalf("expected raw devconsole response body in error, got %s", stderr.String())
	}
}

func TestGetCustomPageReadsDataListShape(t *testing.T) {
	var calls []commandHTTPCall
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		calls = append(calls, commandHTTPCall{Method: r.Method, Path: r.URL.Path, Body: body})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":true,"returnCode":"200","data":{"list":[{"id":"page-001","pageLabel":"客户工作台","pageApi":"customer_interaction_workbench","renderVersion":5}]}}`))
	}))
	defer server.Close()
	projectPath := newCommandCoverageProject(t, server.URL)

	var stdout, stderr bytes.Buffer
	if exit := Run([]string{"get", "customPage", projectPath, "customer_interaction_workbench"}, &stdout, &stderr, projectPath); exit != 0 {
		t.Fatalf("get customPage failed: %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), `"pageApi":"customer_interaction_workbench"`) || !strings.Contains(stdout.String(), `"renderVersion":5`) {
		t.Fatalf("expected data.list output, got %s", stdout.String())
	}
}

func TestCustomPageSavePreflightRejectsMismatchedPageContentAndCompList(t *testing.T) {
	var calls []commandHTTPCall
	server := newCommandCoverageServer(t, &calls)
	defer server.Close()
	projectPath := newCommandCoverageProject(t, server.URL)

	var stdout, stderr bytes.Buffer
	exit := Run([]string{"update", "customPage", projectPath, "workbench_page", `{"pageLabel":"Workbench","pageApi":"workbench_page","pageContent":[{"name":"component-cc-demo","comId":"pc-old","componentInfo":{"id":"pc-old","component":"component-cc-demo"}}],"compList":[{"id":"pc-new","compUniName":"component-cc-demo"}]}`}, &stdout, &stderr, projectPath)
	if exit == 0 {
		t.Fatalf("expected preflight failure, stdout=%s", stdout.String())
	}
	if len(calls) != 0 {
		t.Fatalf("preflight mismatch must not call HTTP, got %#v", calls)
	}
	if !strings.Contains(stderr.String(), "pageContent comId") || !strings.Contains(stderr.String(), "compList") {
		t.Fatalf("expected pageContent/compList validation error, got %s", stderr.String())
	}
}

func TestBindPageComponentUpdatesCustomPageReference(t *testing.T) {
	var calls []commandHTTPCall
	server := newCommandCoverageServer(t, &calls)
	defer server.Close()
	projectPath := newCommandCoverageProject(t, server.URL)

	var stdout, stderr bytes.Buffer
	if exit := Run([]string{"bind", "pagecomponent", projectPath, "workbench_page", "pc-new", "--embedded", "true", "--workspace-url", "https://x.example/app"}, &stdout, &stderr, projectPath); exit != 0 {
		t.Fatalf("bind pagecomponent failed: %s", stderr.String())
	}
	var paths []string
	for _, call := range calls {
		paths = append(paths, call.Path)
	}
	wantPaths := []string{
		"/devconsole/custom/pc/1.0/post/detailCustomPage",
		"/devconsole/custom/pc/1.0/post/detailCustomComp",
		"/devconsole/custom/pc/1.0/post/insertCustomPage",
		"/devconsole/custom/pc/1.0/post/detailCustomPage",
	}
	if strings.Join(paths, ",") != strings.Join(wantPaths, ",") {
		t.Fatalf("unexpected call sequence: %#v", paths)
	}
	saveBody, _ := calls[2].Body["body"].(map[string]any)
	if got := saveBody["pageApi"]; got != "workbench_page" {
		t.Fatalf("expected pageApi workbench_page, got %#v", got)
	}
	pageContentText, _ := saveBody["pageContent"].(string)
	var pageContent []any
	if err := json.Unmarshal([]byte(pageContentText), &pageContent); err != nil {
		t.Fatalf("expected pageContent JSON string, got %#v: %v", saveBody["pageContent"], err)
	}
	if len(pageContent) != 1 {
		t.Fatalf("expected one pageContent item, got %#v", saveBody["pageContent"])
	}
	item := pageContent[0].(map[string]any)
	if item["comId"] != "pc-new" || item["embedded"] != true {
		t.Fatalf("expected new component binding, got %#v", item)
	}
	propObj := item["propObj"].(map[string]any)
	if propObj["workspaceUrl"] != "https://x.example/app" {
		t.Fatalf("expected workspaceUrl update, got %#v", propObj)
	}
	compList := saveBody["compList"].([]any)
	comp := compList[0].(map[string]any)
	if comp["id"] != "pc-new" || comp["compUniName"] != "component-cc-demo" {
		t.Fatalf("expected compList with id and compUniName, got %#v", comp)
	}
	if !strings.Contains(stdout.String(), `"status":"updated"`) || !strings.Contains(stdout.String(), `"componentId":"pc-new"`) {
		t.Fatalf("expected bind summary, got %s", stdout.String())
	}
}

func TestBindPageComponentDryRunDoesNotSaveCustomPage(t *testing.T) {
	var calls []commandHTTPCall
	server := newCommandCoverageServer(t, &calls)
	defer server.Close()
	projectPath := newCommandCoverageProject(t, server.URL)

	var stdout, stderr bytes.Buffer
	if exit := Run([]string{"bind", "pagecomponent", projectPath, "workbench_page", "pc-new", "--embedded", "true", "--workspace-url", "https://x.example/app", "--dry-run"}, &stdout, &stderr, projectPath); exit != 0 {
		t.Fatalf("bind pagecomponent dry-run failed: %s", stderr.String())
	}
	var paths []string
	for _, call := range calls {
		paths = append(paths, call.Path)
		if call.Path == "/devconsole/custom/pc/1.0/post/insertCustomPage" {
			t.Fatalf("dry-run must not save customPage, calls=%#v", calls)
		}
	}
	wantPaths := []string{
		"/devconsole/custom/pc/1.0/post/detailCustomPage",
		"/devconsole/custom/pc/1.0/post/detailCustomComp",
	}
	if strings.Join(paths, ",") != strings.Join(wantPaths, ",") {
		t.Fatalf("unexpected dry-run call sequence: %#v", paths)
	}
	if !strings.Contains(stdout.String(), `"status":"dry_run"`) || !strings.Contains(stdout.String(), `"componentId":"pc-new"`) {
		t.Fatalf("expected dry-run bind summary, got %s", stdout.String())
	}
}

func TestVerifyInjectionPageReportsStaleReference(t *testing.T) {
	var calls []commandHTTPCall
	server := newCommandCoverageServer(t, &calls)
	defer server.Close()
	projectPath := newCommandCoverageProject(t, server.URL)

	var stdout, stderr bytes.Buffer
	if exit := Run([]string{"verify", "injectionPage", projectPath, "workbench_page", "--expected-component", "pc-new", "--snapshot", `{"hasElement":true,"hasIframe":false}`}, &stdout, &stderr, projectPath); exit != 0 {
		t.Fatalf("verify injectionPage failed: %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), `"status":"stale_component_reference"`) {
		t.Fatalf("expected stale reference diagnostic, got %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"actualComponentIds":["pc-old"]`) {
		t.Fatalf("expected actual component ids, got %s", stdout.String())
	}
}

func TestVerifyInjectionPageResolvesExpectedComponentNameAndRejectsStaleID(t *testing.T) {
	var calls []commandHTTPCall
	server := newCommandCoverageServer(t, &calls)
	defer server.Close()
	projectPath := newCommandCoverageProject(t, server.URL)

	var stdout, stderr bytes.Buffer
	if exit := Run([]string{"verify", "injectionPage", projectPath, "workbench_page", "--expected-component", "component-cc-demo"}, &stdout, &stderr, projectPath); exit != 0 {
		t.Fatalf("verify injectionPage failed: %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), `"status":"stale_component_reference"`) {
		t.Fatalf("expected stale reference when component name resolves to latest id, got %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"expectedComponentId":"pc-001"`) || !strings.Contains(stdout.String(), `"actualComponentIds":["pc-old"]`) {
		t.Fatalf("expected resolved component id and actual stale id in output, got %s", stdout.String())
	}
}

func newCommandCoverageServer(t *testing.T, calls *[]commandHTTPCall) *httptest.Server {
	t.Helper()
	savedClassSource := ""
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		call := commandHTTPCall{Method: r.Method, Path: r.URL.Path, RawQuery: r.URL.RawQuery}
		call.Body = body
		if serviceName, _ := body["serviceName"].(string); serviceName != "" {
			call.ServiceName = serviceName
		}
		*calls = append(*calls, call)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/ccfag/save":
			savedClassSource, _ = url.QueryUnescape(fmt.Sprint(body["source"]))
			_, _ = w.Write([]byte(`{"code":200,"returnCode":"200","result":true,"data":{"id":"class-001"}}`))
		case "/api/ccfag/detail":
			response, _ := json.Marshal(map[string]any{"code": 200, "returnCode": "200", "result": true, "data": map[string]any{"id": "class-001", "source": savedClassSource}})
			_, _ = w.Write(response)
		case "/openApi/common":
			_, _ = w.Write([]byte(`{"result":true,"returnCode":"1","data":{"id":"ok"}}`))
		case "/devconsole/custom/pc/1.0/post/pageCustomComp":
			_, _ = w.Write([]byte(`{"code":200,"returnCode":"200","result":true,"data":{"list":[{"id":"pc-001","compUniName":"component-cc-demo","compLabel":"CC Demo"}]}}`))
		case "/devconsole/custom/pc/1.0/post/detailCustomComp":
			requestBody, _ := body["body"].(map[string]any)
			id := "pc-001"
			component := "component-cc-remote"
			if requestBody["id"] == "pc-new" {
				id = "pc-new"
				component = "component-cc-demo"
			}
			_, _ = w.Write([]byte(`{"code":200,"returnCode":"200","result":true,"data":{"id":"` + id + `","compUniName":"` + component + `","compLabel":"Remote","loadModel":"lazy","vueData":"{\"componentInfo\":{\"component\":\"` + component + `\",\"loadModel\":\"lazy\"},\"propObj\":{\"workspaceUrl\":\"https://new.example/app\"}}","compContentVue":"{\"frontend/pagecomponents/cc-remote/cc-remote.vue\":\"<template><div /></template>\"}"}}`))
		case "/devconsole/custom/pc/1.0/post/deleteCustomComp", "/devconsole/custom/pc/1.0/post/insertCustomComp":
			_, _ = w.Write([]byte(`{"code":200,"returnCode":"200","result":true,"data":{"id":"pc-001"}}`))
		case "/devconsole/custom/pc/1.0/post/pageCustomPage":
			_, _ = w.Write([]byte(`{"code":200,"returnCode":"200","result":true,"data":{"list":[{"id":"page-001","pageLabel":"Workbench","pageApi":"workbench_page","renderVersion":"V1.0"}]}}`))
		case "/devconsole/custom/pc/1.0/post/detailCustomPage":
			_, _ = w.Write([]byte(`{"code":200,"returnCode":"200","result":true,"data":{"id":"page-001","pageLabel":"Workbench","pageApi":"workbench_page","renderVersion":"V1.0","orgId":"org-001","canvasStyleData":{"width":1200},"pageContent":[{"name":"component-cc-demo","comId":"pc-old","embedded":false,"propObj":{"workspaceUrl":"https://old.example/app"},"componentInfo":{"id":"pc-old","component":"component-cc-demo","loadModel":"lazy"}}],"compList":[{"id":"pc-old","compUniName":"component-cc-demo"}]}}`))
		case "/devconsole/custom/pc/1.0/post/insertCustomPage":
			_, _ = w.Write([]byte(`{"code":200,"returnCode":"200","result":true,"data":{"id":"page-002","renderVersion":"V2.0"}}`))
		case "/api/triggerSetup/getTriggerByCondition":
			_, _ = w.Write([]byte(`{"code":200,"returnCode":"200","result":true,"data":{"list":[{"id":"trg-001","name":"DemoTrigger","apiname":"demo_trigger"}]}}`))
		case "/api/trigger/newobjtrigger":
			_, _ = w.Write([]byte(`{"code":200,"returnCode":"200","result":true,"data":{"trigger":{"id":"trg-001","name":"DemoTrigger","apiname":"demo_trigger"}}}`))
		case "/api/triggerSetup/deleteTrigger":
			_, _ = w.Write([]byte(`{"code":200,"returnCode":"200","result":true}`))
		case "/metadata/v1/profiles":
			_, _ = w.Write([]byte(`{"service":"cc-metadata-service","mode":"read-only-profiles","count":1,"profiles":[{"id":"profile-001","name":"销售简档","apiName":"sales_profile"}]}`))
		case "/metadata/v1/profiles/profile-001":
			_, _ = w.Write([]byte(`{"service":"cc-metadata-service","mode":"read-only-profile-detail","profile":{"id":"profile-001","name":"销售简档","apiName":"sales_profile"},"userReferenceCount":0}`))
		default:
			_, _ = w.Write([]byte(`{"service":"mock","code":200,"returnCode":"200","result":true,"data":{"objList":[{"id":"profile-001"}],"list":[]}}`))
		}
	}))
}

func newCommandCoverageProject(t *testing.T, serverURL string) string {
	t.Helper()
	projectPath := t.TempDir()
	writeCommandTestFile(t, filepath.Join(projectPath, "cloudcc-cli.config.json"), `{"use":"dev","dev":{"metadataService":{"url":"`+serverURL+`","token":"metadata-token"},"accessToken":"cloud-token","pluginToken":"plugin-token","apiSvc":"`+serverURL+`","setupSvc":"`+serverURL+`","baseUrl":"`+serverURL+`","devSvcDispatch":"/devconsole"}}`)
	writeCommandTestFile(t, filepath.Join(projectPath, "package.json"), `{"devConsoleConfig":{"accessToken":"cloud-token","pluginToken":"plugin-token","apiSvc":"`+serverURL+`","setupSvc":"`+serverURL+`","baseUrl":"`+serverURL+`","devSvcDispatch":"/devconsole"}}`)
	return projectPath
}

func assertCommandCall(t *testing.T, calls []commandHTTPCall, method string, path string) {
	t.Helper()
	if len(calls) != 1 {
		t.Fatalf("expected one HTTP call, got %#v", calls)
	}
	if calls[0].Method != method || calls[0].Path != path {
		t.Fatalf("expected %s %s, got %#v", method, path, calls[0])
	}
}

func assertCommandCalls(t *testing.T, calls []commandHTTPCall, method string, paths []string) {
	t.Helper()
	if len(calls) != len(paths) {
		t.Fatalf("expected HTTP calls %v, got %#v", paths, calls)
	}
	for i, path := range paths {
		if calls[i].Method != method || calls[i].Path != path {
			t.Fatalf("expected call %d to be %s %s, got %#v", i, method, path, calls[i])
		}
	}
}

func writeCommandTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func readCommandTestJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out
}
