package command

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cloudcc-customization-expert-go/internal/edition"
)

func TestLowCodeCreateObjectShortcutUsesMetadataServicePlan(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metadata/v1/plans" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method %s", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"planId":"plan_shortcut","operationId":"oper_shortcut","status":"PLANNED"}`))
	}))
	defer server.Close()
	t.Setenv("CLOUDCC_METADATA_SERVICE_URL", server.URL)

	projectPath := commandTestProject(t)
	var stdout, stderr bytes.Buffer
	exit := Run([]string{"create", "object", projectPath, "测试对象", "test_object", "说明"}, &stdout, &stderr, projectPath)
	if exit != 0 {
		t.Fatalf("expected success, stderr=%s", stderr.String())
	}
	if received["domain"] != "objects" || received["operation"] != "create" {
		t.Fatalf("unexpected plan envelope: %#v", received)
	}
	spec := received["spec"].(map[string]any)
	if spec["label"] != "测试对象" || spec["apiName"] != "test_object" {
		t.Fatalf("unexpected object spec: %#v", spec)
	}
	if !strings.Contains(stdout.String(), "plan_shortcut") {
		t.Fatalf("expected plan response, got %s", stdout.String())
	}
}

func TestLowCodeCreateObjectShortcutAcceptsAccessableFlag(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metadata/v1/plans" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"planId":"plan_accessable","status":"PLANNED"}`))
	}))
	defer server.Close()
	t.Setenv("CLOUDCC_METADATA_SERVICE_URL", server.URL)

	projectPath := commandTestProject(t)
	var stdout, stderr bytes.Buffer
	exit := Run([]string{"create", "object", projectPath, "合同", "contract", "合同管理", "--accessable", "0"}, &stdout, &stderr, projectPath)
	if exit != 0 {
		t.Fatalf("expected success, stderr=%s", stderr.String())
	}
	spec := received["spec"].(map[string]any)
	if spec["accessable"] != "0" {
		t.Fatalf("expected accessable 0 in object spec, got %#v", spec)
	}
}

func TestLowCodeCreateObjectShortcutRejectsUnsupportedAccessableFlag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("invalid accessable should be rejected before request, got %s", r.URL.Path)
	}))
	defer server.Close()
	t.Setenv("CLOUDCC_METADATA_SERVICE_URL", server.URL)

	projectPath := commandTestProject(t)
	var stdout, stderr bytes.Buffer
	exit := Run([]string{"create", "object", projectPath, "合同", "contract", "合同管理", "--accessable=3"}, &stdout, &stderr, projectPath)
	if exit == 0 {
		t.Fatalf("expected failure for unsupported accessable, stdout=%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "--accessable only supports 0") {
		t.Fatalf("expected accessable validation error, got %s", stderr.String())
	}
}

func TestLowCodeCreateObjectShortcutNormalizesImplicitPrefixAndChineseDescription(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metadata/v1/plans" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"planId":"plan_normalized","status":"PLANNED"}`))
	}))
	defer server.Close()
	t.Setenv("CLOUDCC_METADATA_SERVICE_URL", server.URL)

	projectPath := commandTestProject(t)
	var stdout, stderr bytes.Buffer
	exit := Run([]string{"create", "object", projectPath, "合同", "sun_contract", "V2.2 P0: contract header."}, &stdout, &stderr, projectPath)
	if exit != 0 {
		t.Fatalf("expected success, stderr=%s", stderr.String())
	}
	spec := received["spec"].(map[string]any)
	if spec["apiName"] != "contract" || spec["name"] != "contract" || spec["schemetableName"] != "contract" {
		t.Fatalf("expected implicit sun_ prefix to be removed, got %#v", spec)
	}
	if spec["description"] != "用于管理合同业务数据。" || spec["remark"] != "用于管理合同业务数据。" {
		t.Fatalf("expected Chinese default description, got %#v", spec)
	}
}

func TestLowCodeCreateObjectShortcutAcceptsJSONSpec(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metadata/v1/plans" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"planId":"plan_json","status":"PLANNED"}`))
	}))
	defer server.Close()
	t.Setenv("CLOUDCC_METADATA_SERVICE_URL", server.URL)

	projectPath := commandTestProject(t)
	specPath := filepath.Join(projectPath, "object.json")
	if err := os.WriteFile(specPath, []byte(`{"label":"JSON对象","apiName":"json_object"}`), 0644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	exit := Run([]string{"create", "object", projectPath, "@" + specPath}, &stdout, &stderr, projectPath)
	if exit != 0 {
		t.Fatalf("expected success, stderr=%s", stderr.String())
	}
	if received["domain"] != "objects" || received["operation"] != "create" {
		t.Fatalf("unexpected plan envelope: %#v", received)
	}
	spec := received["spec"].(map[string]any)
	if spec["label"] != "JSON对象" || spec["apiName"] != "json_object" {
		t.Fatalf("unexpected object spec: %#v", spec)
	}
}

func TestLowCodeDeleteObjectShortcutUsesMetadataServicePlan(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metadata/v1/plans" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"planId":"plan_delete","status":"PLANNED"}`))
	}))
	defer server.Close()
	t.Setenv("CLOUDCC_METADATA_SERVICE_URL", server.URL)

	projectPath := commandTestProject(t)
	var stdout, stderr bytes.Buffer
	exit := Run([]string{"delete", "object", projectPath, "obj-001"}, &stdout, &stderr, projectPath)
	if exit != 0 {
		t.Fatalf("expected success, stderr=%s", stderr.String())
	}
	if received["domain"] != "objects" || received["operation"] != "delete" {
		t.Fatalf("unexpected plan envelope: %#v", received)
	}
	spec := received["spec"].(map[string]any)
	if spec["id"] != "obj-001" {
		t.Fatalf("unexpected delete spec: %#v", spec)
	}
}

func TestLowCodeGetObjectShortcutUsesMetadataServiceDetail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metadata/v1/objects/deleted" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method %s", r.Method)
		}
		_, _ = w.Write([]byte(`{"mode":"read-only-object-detail","selector":"deleted","found":true}`))
	}))
	defer server.Close()
	t.Setenv("CLOUDCC_METADATA_SERVICE_URL", server.URL)

	projectPath := commandTestProject(t)
	var stdout, stderr bytes.Buffer
	exit := Run([]string{"get", "object", projectPath, "deleted"}, &stdout, &stderr, projectPath)
	if exit != 0 {
		t.Fatalf("expected success, stderr=%s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "read-only-object-detail") {
		t.Fatalf("expected object detail response, got %s", stdout.String())
	}
}

func TestLowCodeGetFieldsShortcutUsesMetadataServiceObjectScope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metadata/v1/fields" || r.URL.Query().Get("object") != "a 41" {
			t.Fatalf("unexpected field query %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method %s", r.Method)
		}
		_, _ = w.Write([]byte(`{"mode":"read-only-object-fields","objectSelector":"a 41","found":true}`))
	}))
	defer server.Close()
	t.Setenv("CLOUDCC_METADATA_SERVICE_URL", server.URL)

	projectPath := commandTestProject(t)
	var stdout, stderr bytes.Buffer
	exit := Run([]string{"get", "fields", projectPath, "a 41"}, &stdout, &stderr, projectPath)
	if exit != 0 {
		t.Fatalf("expected success, stderr=%s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "read-only-object-fields") {
		t.Fatalf("expected field query response, got %s", stdout.String())
	}
}

func TestReportShortcutUsesMetadataServiceReadAndPlan(t *testing.T) {
	var paths []string
	var bodies []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		bodies = append(bodies, body)
		switch r.URL.Path {
		case "/metadata/v1/reports:query":
			_, _ = w.Write([]byte(`{"mode":"reports","rows":[]}`))
		case "/metadata/v1/plans":
			_, _ = w.Write([]byte(`{"planId":"plan_report","status":"PLANNED"}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("CLOUDCC_METADATA_SERVICE_URL", server.URL)

	projectPath := commandTestProject(t)
	var stdout, stderr bytes.Buffer
	exit := Run([]string{"get", "report", projectPath, "folder-1", "sales", "2", "50", "name", "desc"}, &stdout, &stderr, projectPath)
	if exit != 0 {
		t.Fatalf("expected get success, stderr=%s", stderr.String())
	}
	if paths[0] != "/metadata/v1/reports:query" ||
		bodies[0]["folderId"] != "folder-1" ||
		bodies[0]["searchKeyWord"] != "sales" ||
		bodies[0]["page"].(float64) != 2 ||
		bodies[0]["pageSize"].(float64) != 50 {
		t.Fatalf("unexpected report query request path=%v body=%#v", paths, bodies[0])
	}

	stdout.Reset()
	stderr.Reset()
	exit = Run([]string{"create", "report", projectPath, `{"id":"old","name":"Sales"}`}, &stdout, &stderr, projectPath)
	if exit != 0 {
		t.Fatalf("expected create success, stderr=%s", stderr.String())
	}
	plan := bodies[1]
	if plan["domain"] != "reports" || plan["operation"] != "create" {
		t.Fatalf("unexpected report plan envelope: %#v", plan)
	}
	spec := plan["spec"].(map[string]any)
	if spec["id"] != nil || spec["name"] != "Sales" {
		t.Fatalf("expected create report to strip id, got %#v", spec)
	}

	stdout.Reset()
	stderr.Reset()
	exit = Run([]string{"update", "report", projectPath, "report-1", `{"name":"Sales Updated"}`}, &stdout, &stderr, projectPath)
	if exit != 0 {
		t.Fatalf("expected update success, stderr=%s", stderr.String())
	}
	plan = bodies[2]
	if plan["domain"] != "reports" || plan["operation"] != "upsert" {
		t.Fatalf("unexpected report update plan envelope: %#v", plan)
	}
	spec = plan["spec"].(map[string]any)
	if spec["id"] != "report-1" || spec["name"] != "Sales Updated" {
		t.Fatalf("unexpected report update spec: %#v", spec)
	}

	stdout.Reset()
	stderr.Reset()
	exit = Run([]string{"delete", "report", projectPath, "report-1", "true"}, &stdout, &stderr, projectPath)
	if exit != 0 {
		t.Fatalf("expected delete success, stderr=%s", stderr.String())
	}
	plan = bodies[3]
	if plan["domain"] != "reports" || plan["operation"] != "delete" {
		t.Fatalf("unexpected report delete plan envelope: %#v", plan)
	}
	spec = plan["spec"].(map[string]any)
	if spec["id"] != "report-1" || spec["confirmdelete"] != "true" {
		t.Fatalf("unexpected report delete spec: %#v", spec)
	}
}

func TestReportMatrixShortcutBuildsCompleteMetadataServicePlan(t *testing.T) {
	var bodies []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metadata/v1/plans" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		bodies = append(bodies, body)
		_, _ = w.Write([]byte(`{"planId":"plan_matrix","status":"PLANNED"}`))
	}))
	defer server.Close()
	t.Setenv("CLOUDCC_METADATA_SERVICE_URL", server.URL)

	projectPath := commandTestProject(t)
	matrix := `{
		"id":"ignored",
		"name":"Matrix Sales",
		"source":{"objects":[{"objectId":"account"},{"objectId":"opportunity"}]},
		"fields":[{"fieldId":"field_name","objectId":"account"},{"fieldId":"field_amount","objectId":"opportunity"}],
		"rows":[{"fieldId":"owner_id","sort":"desc"}],
		"columns":[{"fieldId":"close_month","dateType":"month"}],
		"summaryFields":["field_amount"],
		"optionb":"inner",
		"bfindid":"field_account_id"
	}`
	var stdout, stderr bytes.Buffer
	exit := Run([]string{"create", "reportMatrix", projectPath, matrix}, &stdout, &stderr, projectPath)
	if exit != 0 {
		t.Fatalf("expected matrix create success, stderr=%s", stderr.String())
	}
	plan := bodies[0]
	if plan["domain"] != "reports" || plan["operation"] != "create" {
		t.Fatalf("unexpected matrix plan envelope: %#v", plan)
	}
	spec := plan["spec"].(map[string]any)
	if spec["id"] != nil || spec["reporttype"] != "Matrix" || spec["type"] != "Matrix" ||
		spec["islightning"] != "true" || spec["totalrecord"] != "1" || spec["scope"] != "user" {
		t.Fatalf("unexpected matrix defaults: %#v", spec)
	}
	groups := spec["groups"].(map[string]any)
	if len(groups["rows"].([]any)) != 1 || len(groups["columns"].([]any)) != 1 {
		t.Fatalf("expected row and column groups, got %#v", groups)
	}
	summaries := spec["summaries"].([]any)
	if summaries[0].(map[string]any)["fieldId"] != "field_amount" ||
		summaries[0].(map[string]any)["method"] != "sum" {
		t.Fatalf("unexpected summaries: %#v", summaries)
	}

	stdout.Reset()
	stderr.Reset()
	exit = Run([]string{"create", "reportMatrix", projectPath, `{"name":"Bad","fields":[{"fieldId":"field_name"}],"rows":[{"fieldId":"owner_id"}]}`}, &stdout, &stderr, projectPath)
	if exit == 0 || !strings.Contains(stderr.String(), "requires both row groups and column groups") {
		t.Fatalf("expected missing column group validation, stderr=%s", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	exit = Run([]string{"create", "reportMatrix", projectPath, `{"name":"Bad Chart","objecta":"account","fields":[{"fieldId":"field_name"}],"rows":[{"fieldId":"owner_id"}],"columns":[{"fieldId":"createdate"}],"summaryFields":["amount"],"isshowchart":"true","dashboardtype":"bar_0","xcon":"totalrecord"}`}, &stdout, &stderr, projectPath)
	if exit == 0 || !strings.Contains(stderr.String(), "requires ycon or chart.y") {
		t.Fatalf("expected missing chart y validation, stderr=%s", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	exit = Run([]string{"create", "reportMatrix", projectPath, `{"name":"Bad Date","objecta":"account","fields":[{"fieldId":"field_name"}],"rows":[{"fieldId":"owner_id"}],"columns":[{"fieldId":"createdate"}],"summaryFields":["amount"],"startdatestr":"2026-01-01","enddatestr":"2026-12-31"}`}, &stdout, &stderr, projectPath)
	if exit == 0 || !strings.Contains(stderr.String(), "requires datecon") {
		t.Fatalf("expected missing datecon validation, stderr=%s", stderr.String())
	}
}

func TestTypedReportShortcutsBuildCompleteMetadataServicePlans(t *testing.T) {
	var bodies []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metadata/v1/plans" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		bodies = append(bodies, body)
		_, _ = w.Write([]byte(`{"planId":"plan_report","status":"PLANNED"}`))
	}))
	defer server.Close()
	t.Setenv("CLOUDCC_METADATA_SERVICE_URL", server.URL)

	projectPath := commandTestProject(t)
	var stdout, stderr bytes.Buffer
	tabular := `{"name":"Customers","objecta":"account","fields":[{"fieldId":"field_name"}]}`
	exit := Run([]string{"create", "reportTabular", projectPath, tabular}, &stdout, &stderr, projectPath)
	if exit != 0 {
		t.Fatalf("expected tabular create success, stderr=%s", stderr.String())
	}
	spec := bodies[0]["spec"].(map[string]any)
	if spec["reporttype"] != "Tabular" || spec["type"] != "Tabular" || spec["totalrecord"] != "1" {
		t.Fatalf("unexpected tabular spec: %#v", spec)
	}

	stdout.Reset()
	stderr.Reset()
	summary := `{"name":"Owner Summary","objecta":"account","fields":[{"fieldId":"field_name"}],"rows":[{"fieldId":"owner_id"}],"summaryFields":["amount"]}`
	exit = Run([]string{"create", "reportSummary", projectPath, summary}, &stdout, &stderr, projectPath)
	if exit != 0 {
		t.Fatalf("expected summary create success, stderr=%s", stderr.String())
	}
	spec = bodies[1]["spec"].(map[string]any)
	if spec["reporttype"] != "Summary" || len(spec["summaries"].([]any)) != 1 {
		t.Fatalf("unexpected summary spec: %#v", spec)
	}
	groups := spec["groups"].(map[string]any)
	if len(groups["rows"].([]any)) != 1 {
		t.Fatalf("expected summary row group, got %#v", groups)
	}

	stdout.Reset()
	stderr.Reset()
	ratio := `{"name":"Growth","objecta":"account","dateFieldId":"createdate","ratioExpressions":[{"type":"TB","fieldId":"totalrecord"}]}`
	exit = Run([]string{"create", "reportRatio", projectPath, ratio}, &stdout, &stderr, projectPath)
	if exit != 0 {
		t.Fatalf("expected ratio create success, stderr=%s", stderr.String())
	}
	spec = bodies[2]["spec"].(map[string]any)
	if spec["reporttype"] != "ratio" || spec["transversegroupone"] != "createdate" ||
		spec["mainobjectcolumnid"] != "totalrecord,createdate" ||
		!strings.Contains(fmt.Sprint(spec["tbhbexpression"]), "TB") {
		t.Fatalf("unexpected ratio spec: %#v", spec)
	}

	stdout.Reset()
	stderr.Reset()
	exit = Run([]string{"create", "reportRatio", projectPath, `{"name":"Bad","objecta":"account","fields":[{"fieldId":"name"}]}`}, &stdout, &stderr, projectPath)
	if exit == 0 || !strings.Contains(stderr.String(), "requires date group field") {
		t.Fatalf("expected ratio date field validation, stderr=%s", stderr.String())
	}
}

func TestReportFolderShortcutUsesReportsDomainFolderPlans(t *testing.T) {
	var bodies []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metadata/v1/plans" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		bodies = append(bodies, body)
		_, _ = w.Write([]byte(`{"planId":"plan_folder","status":"PLANNED"}`))
	}))
	defer server.Close()
	t.Setenv("CLOUDCC_METADATA_SERVICE_URL", server.URL)

	projectPath := commandTestProject(t)
	var stdout, stderr bytes.Buffer
	exit := Run([]string{"create", "reportFolder", projectPath, "Sales Reports", `{"viewType":"2","purview":"1"}`}, &stdout, &stderr, projectPath)
	if exit != 0 {
		t.Fatalf("expected success, stderr=%s", stderr.String())
	}
	received := bodies[0]
	if received["domain"] != "reports" || received["operation"] != "folder-create" {
		t.Fatalf("unexpected reportFolder plan envelope: %#v", received)
	}
	spec := received["spec"].(map[string]any)
	if spec["name"] != "Sales Reports" || spec["viewType"] != "2" || spec["purview"] != "1" {
		t.Fatalf("unexpected reportFolder spec: %#v", spec)
	}

	stdout.Reset()
	stderr.Reset()
	exit = Run([]string{"update", "reportFolder", projectPath, "folder-1", `{"viewType":"1","purview":"2"}`}, &stdout, &stderr, projectPath)
	if exit != 0 {
		t.Fatalf("expected update success, stderr=%s", stderr.String())
	}
	received = bodies[1]
	if received["domain"] != "reports" || received["operation"] != "folder-update" {
		t.Fatalf("unexpected reportFolder update plan envelope: %#v", received)
	}
	spec = received["spec"].(map[string]any)
	if spec["id"] != "folder-1" || spec["viewType"] != "1" || spec["purview"] != "2" {
		t.Fatalf("unexpected reportFolder update spec: %#v", spec)
	}

	stdout.Reset()
	stderr.Reset()
	exit = Run([]string{"delete", "reportFolder", projectPath, "folder-1"}, &stdout, &stderr, projectPath)
	if exit != 0 {
		t.Fatalf("expected delete success, stderr=%s", stderr.String())
	}
	received = bodies[2]
	if received["domain"] != "reports" || received["operation"] != "folder-delete" {
		t.Fatalf("unexpected reportFolder delete plan envelope: %#v", received)
	}
	spec = received["spec"].(map[string]any)
	if spec["id"] != "folder-1" {
		t.Fatalf("unexpected reportFolder delete spec: %#v", spec)
	}
}

func TestUniversalUIAPIProviderPreservesLowCodeReportApprovalAndDashboardContracts(t *testing.T) {
	withUniversalCommandEdition(t)
	t.Setenv("CLOUDCC_EXECUTION_MODE", "uiapi")
	t.Setenv("CLOUDCC_METADATA_SERVICE_URL", "")

	type request struct {
		path string
		body map[string]any
	}
	var requests []request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		requests = append(requests, request{path: r.URL.Path, body: body})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":true,"returnCode":"200","data":{}}`))
	}))
	defer server.Close()
	projectPath := uiapiCommandTestProject(t, server.URL)

	var stdout, stderr bytes.Buffer
	run := func(args ...string) {
		stdout.Reset()
		stderr.Reset()
		if exit := Run(args, &stdout, &stderr, projectPath); exit != 0 {
			t.Fatalf("command %v failed: %s", args, stderr.String())
		}
	}

	run("create", "approval", projectPath, `{"name":"Sales approval"}`)
	run("update", "approval", projectPath, `{"id":"approval-1","name":"Sales approval"}`)
	run("delete", "approval", projectPath, "approval-1")
	run("get", "report", projectPath, "folder-1", "sales", "2", "50", "name", "desc")
	run("create", "report", projectPath, `{"id":"ignored","name":"Sales"}`)
	run("update", "report", projectPath, "report-1", `{"name":"Sales updated"}`)
	run("delete", "report", projectPath, "report-1")
	run("create", "reportMatrix", projectPath, `{"name":"Matrix","objecta":"account","fields":[{"fieldId":"name"}],"rows":[{"fieldId":"owner"}],"columns":[{"fieldId":"createdate"}],"summaryFields":["amount"]}`)
	run("create", "reportFolder", projectPath, "Sales Reports", `{"viewType":"2"}`)
	run("update", "reportFolder", projectPath, "folder-1", `{"viewType":"1"}`)
	run("delete", "reportFolder", projectPath, "folder-1")
	run("get", "dashboard", projectPath, `{"dashboardfolderid":"recentDashboard"}`)
	run("create", "dashboard", projectPath, `{"name":"Sales dashboard","islightning":"true"}`)
	run("update", "dashboard", projectPath, `{"id":"dashboard-1","name":"Sales dashboard"}`)
	run("delete", "dashboard", projectPath, "dashboard-1")

	wantPaths := []string{
		"/api/approvalsetup/saveApproval",
		"/api/approvalsetup/saveApproval",
		"/api/approvalsetup/deleteApproval",
		"/api/report/tab/getReportList",
		"/api/report/tab/saveReport",
		"/api/report/tab/saveReport",
		"/api/report/base/deleteReport",
		"/api/report/tab/saveReport",
		"/api/report/folder/addReportFolder",
		"/api/report/folder/updateReportFolder",
		"/api/report/folder/deleteReportFolder",
		"/api/dashboard/getDashboardList",
		"/api/dashboard/addDashboard",
		"/api/dashboard/updateDashboard",
		"/api/dashboard/deleteDashboard",
	}
	if len(requests) != len(wantPaths) {
		t.Fatalf("expected %d requests, got %#v", len(wantPaths), requests)
	}
	for index, wantPath := range wantPaths {
		if requests[index].path != wantPath {
			t.Fatalf("request %d: want %s, got %#v", index, wantPath, requests[index])
		}
	}
	if requests[3].body["folderId"] != "folder-1" || requests[3].body["page"] != float64(2) || requests[3].body["pageSize"] != float64(50) {
		t.Fatalf("report list arguments were not normalized: %#v", requests[3].body)
	}
	if requests[4].body["id"] != nil || requests[4].body["name"] != "Sales" || requests[5].body["id"] != "report-1" {
		t.Fatalf("report save bodies were not normalized: create=%#v update=%#v", requests[4].body, requests[5].body)
	}
	if requests[7].body["type"] != "Matrix" || requests[7].body["reporttype"] != "Matrix" {
		t.Fatalf("typed report defaults were not preserved: %#v", requests[7].body)
	}
	if requests[8].body["name"] != "Sales Reports" || requests[10].body["id"] != "folder-1" || requests[14].body["id"] != "dashboard-1" {
		t.Fatalf("UIAPI request bodies were not normalized: %#v", requests)
	}
}

func TestUniversalUIAPIProviderUsesSourceProvenCRUDRoutes(t *testing.T) {
	withUniversalCommandEdition(t)
	t.Setenv("CLOUDCC_EXECUTION_MODE", "uiapi")
	t.Setenv("CLOUDCC_METADATA_SERVICE_URL", "")

	type request struct {
		path string
		body map[string]any
	}
	var requests []request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		requests = append(requests, request{path: r.URL.Path, body: body})
		_, _ = w.Write([]byte(`{"result":true,"returnCode":"200","data":{}}`))
	}))
	defer server.Close()
	projectPath := uiapiCommandTestProject(t, server.URL)

	cases := []struct {
		action   string
		resource string
		value    string
		path     string
	}{
		{"get", "application", `{}`, "/api/newApp/getAppList"},
		{"create", "application", `{"name":"Sales"}`, "/api/newApp/save"},
		{"update", "application", `{"id":"app-1","name":"Sales"}`, "/api/newApp/save"},
		{"delete", "application", "app-1", "/api/newApp/deleteApp"},
		{"get", "button", `{}`, "/api/buttonlink/listButton"},
		{"create", "button", `{"name":"Approve"}`, "/api/buttonlink/saveButton"},
		{"update", "button", `{"id":"button-1","name":"Approve"}`, "/api/buttonlink/saveButton"},
		{"delete", "button", "button-1", "/api/buttonlink/deleteButton"},
		{"get", "customSetting", `{}`, "/api/customsetting/list"},
		{"create", "customSetting", `{"name":"Settings"}`, "/api/customsetting/save"},
		{"update", "customSetting", `{"id":"setting-1"}`, "/api/customsetting/modify"},
		{"delete", "customSetting", "setting-1", "/api/customsetting/deleteobj"},
		{"get", "dupeCatcher", `{}`, "/api/duplication/getList"},
		{"create", "dupeCatcher", `{"name":"Duplicate account"}`, "/api/duplication/saveFilter"},
		{"update", "dupeCatcher", `{"id":"dupe-1"}`, "/api/duplication/saveFilter"},
		{"delete", "dupeCatcher", "dupe-1", "/api/duplication/deletefilter"},
		{"create", "fields", `{"prefix":"account","label":"Level"}`, "/api/fieldSetup/save"},
		{"update", "fields", `{"id":"field-1","prefix":"account"}`, "/api/fieldSetup/save"},
		{"delete", "fields", "field-1", "/api/fieldSetup/deleteFieldCompletely"},
		{"get", "globalSelectList", `{"pageNum":"1","pageSize":"20"}`, "/api/globalSelectSetup/queryList"},
		{"create", "globalSelectList", `{"name":"Stage"}`, "/api/globalSelectSetup/save"},
		{"update", "globalSelectList", `{"id":"global-1","name":"Stage"}`, "/api/globalSelectSetup/save"},
		{"delete", "globalSelectList", "global-1", "/api/globalSelectSetup/delete"},
		{"get", "identityProvider", `{}`, "/api/samlSp/queryList"},
		{"create", "identityProvider", `{"name":"Partner IdP"}`, "/api/samlSp/save"},
		{"update", "identityProvider", `{"id":"idp-1","name":"Partner IdP"}`, "/api/samlSp/save"},
		{"delete", "identityProvider", "idp-1", "/api/samlSp/delete"},
		{"get", "menu", `{}`, "/api/customTab/queryTabList"},
		{"create", "menu", `{"type":"object","p1":"account","p2":"Accounts"}`, "/api/customTab/tabSetDone"},
		{"get", "pagelayout", "account", "/api/layout/queryPageLayout"},
		{"create", "pagelayout", `{"objId":"account","layoutId":"layout-source"}`, "/api/layout/cloneLayout"},
		{"delete", "pagelayout", "layout-1", "/api/layout/deleteButton"},
		{"get", "permission", `{}`, "/api/permissionGroup/queryPermsetsList"},
		{"create", "permission", `{"name":"Sales permission set"}`, "/api/permissionGroup/savePermsets"},
		{"update", "permission", `{"id":"permission-1","name":"Sales permission set"}`, "/api/permissionGroup/modifyPermsets"},
		{"delete", "permission", "permission-1", "/api/permissionGroup/deletePermsets"},
		{"assign", "permission", `{"id":"permission-1","ids":["user-1"]}`, "/api/permissionGroup/addUsersetup"},
		{"remove", "permission", `{"id":"permission-1","ids":["user-1"]}`, "/api/permissionGroup/deleteUsersetup"},
		{"get", "profile", `{}`, "/api/profile/listAll"},
		{"create", "profile", `{"copyFromId":"aaa000003","newProfileName":"Sales"}`, "/api/profile/newProfile"},
		{"update", "profile", `{"id":"profile-1","name":"Sales"}`, "/api/profile/saveProfile"},
		{"delete", "profile", "profile-1", "/api/profile/delProfile"},
		{"get", "recordType", `{"prefix":"account"}`, "/api/recordType/getRecordTypeList"},
		{"create", "recordType", `{"prefix":"account","label":"Retail"}`, "/api/recordType/saveRecordType"},
		{"update", "recordType", `{"id":"record-type-1","label":"Retail"}`, "/api/recordType/editSave"},
		{"delete", "recordType", "record-type-1", "/api/recordType/deleteObj"},
		{"get", "role", `{}`, "/api/role/queryRole"},
		{"create", "role", `{"name":"Sales"}`, "/api/role/saveRole"},
		{"update", "role", `{"id":"role-1","name":"Sales"}`, "/api/role/editSaveRole"},
		{"assign", "role", `{"id":"role-1","userIds":["user-1"]}`, "/api/role/saveAssign"},
		{"delete", "role", "role-1", "/api/role/deleteRole"},
		{"update", "menu", `{"id":"menu-1","type":"object","p2":"Accounts"}`, "/api/customTab/updatesavetab"},
		{"delete", "menu", "menu-1", "/api/customTab/deleteTab"},
		{"get", "sharingRule", `{}`, "/api/sharingSettings/queryRule"},
		{"create", "sharingRule", `{"name":"Sales"}`, "/api/sharingSettings/insertRule"},
		{"update", "sharingRule", `{"id":"sharing-1"}`, "/api/sharingSettings/updateRule"},
		{"delete", "sharingRule", "sharing-1", "/api/sharingSettings/deleteRule"},
		{"get", "singleSignOn", `{}`, "/api/spconfig/list"},
		{"create", "singleSignOn", `{"name":"Partner SSO"}`, "/api/spconfig/save"},
		{"update", "singleSignOn", `{"id":"sso-1","name":"Partner SSO"}`, "/api/spconfig/save"},
		{"delete", "singleSignOn", "sso-1", "/api/spconfig/delete"},
		{"get", "validationRule", `{"prefix":"account"}`, "/api/validateRule/queryByPrefix"},
		{"create", "validationRule", `{"name":"Required"}`, "/api/validateRule/save"},
		{"update", "validationRule", `{"id":"rule-1","name":"Required"}`, "/api/validateRule/save"},
		{"delete", "validationRule", "rule-1", "/api/validateRule/delete"},
		{"get", "view", `{"objId":"account"}`, "/api/view/list/getViewList"},
		{"create", "view", `{"objId":"account","label":"Open accounts"}`, "/api/view/saveView"},
		{"update", "view", `{"id":"view-1","label":"Open accounts"}`, "/api/view/saveView"},
		{"delete", "view", "view-1", "/api/view/deleteView"},
	}

	var stdout, stderr bytes.Buffer
	for _, tc := range cases {
		stdout.Reset()
		stderr.Reset()
		if exit := Run([]string{tc.action, tc.resource, projectPath, tc.value}, &stdout, &stderr, projectPath); exit != 0 {
			t.Fatalf("%s %s failed: %s", tc.action, tc.resource, stderr.String())
		}
	}
	stdout.Reset()
	stderr.Reset()
	if exit := Run([]string{"get", "fields", projectPath, "account"}, &stdout, &stderr, projectPath); exit != 0 {
		t.Fatalf("get fields failed: %s", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if exit := Run([]string{"update", "pagelayout", projectPath, "layout-1", `{"sections":[{"sectionId":"section-1","sortOrder":1}]}`}, &stdout, &stderr, projectPath); exit != 0 {
		t.Fatalf("update pagelayout failed: %s", stderr.String())
	}
	if len(requests) != len(cases)+3 {
		t.Fatalf("expected %d requests, got %#v", len(cases)+3, requests)
	}
	requestOffset := 0
	for index, tc := range cases {
		requestIndex := index + requestOffset
		if tc.action == "update" && tc.resource == "menu" {
			if requests[requestIndex].path != "/api/customTab/updatetab" || requests[requestIndex].body["id"] != "menu-1" {
				t.Fatalf("menu update preflight route/body mismatch: %#v", requests[requestIndex])
			}
			requestIndex++
			requestOffset++
		}
		if requests[requestIndex].path != tc.path {
			t.Fatalf("%s %s: want %s, got %#v", tc.action, tc.resource, tc.path, requests[requestIndex])
		}
		if tc.action == "delete" && requests[requestIndex].body["id"] == nil {
			t.Fatalf("%s %s: delete body must preserve id, got %#v", tc.action, tc.resource, requests[requestIndex].body)
		}
	}
	if fieldRequest := requests[len(cases)+1]; fieldRequest.path != "/api/fieldSetup/queryField" || fieldRequest.body["prefix"] != "account" {
		t.Fatalf("get fields route/body mismatch: %#v", fieldRequest)
	}
	if layoutUpdate := requests[len(cases)+2]; layoutUpdate.path != "/api/modifyLayoutLightning/saveLayout" || layoutUpdate.body["layoutId"] != "layout-1" || layoutUpdate.body["layoutJson"] == nil {
		t.Fatalf("update pagelayout route/body mismatch: %#v", layoutUpdate)
	}
}

func TestUniversalUIAPIProviderUsesSourceProvenObjectLifecycleRoutes(t *testing.T) {
	withUniversalCommandEdition(t)
	t.Setenv("CLOUDCC_EXECUTION_MODE", "uiapi")
	t.Setenv("CLOUDCC_METADATA_SERVICE_URL", "")

	type request struct {
		path string
		body map[string]any
	}
	var requests []request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		requests = append(requests, request{path: r.URL.Path, body: body})
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/customObject/newPage" {
			_, _ = w.Write([]byte(`{"result":true,"returnCode":"200","data":{"objList":[{"id":"profile-1"}]}}`))
			return
		}
		_, _ = w.Write([]byte(`{"result":true,"returnCode":"200","data":{}}`))
	}))
	defer server.Close()
	projectPath := uiapiCommandTestProject(t, server.URL)

	var stdout, stderr bytes.Buffer
	run := func(args ...string) {
		stdout.Reset()
		stderr.Reset()
		if exit := Run(args, &stdout, &stderr, projectPath); exit != 0 {
			t.Fatalf("command %v failed: %s", args, stderr.String())
		}
	}

	run("get", "object", projectPath, "deleted")
	run("create", "object", projectPath, "Customer", "customer", "Relationship data")
	run("update", "object", projectPath, `{"objid":"object-1","obj":{"label":"Customer updated"}}`)
	run("delete", "object", projectPath, "object-1")
	run("purge", "object", projectPath, "object-1", "--execute", "--approval", "CLOUDCC_OBJECT_PHYSICAL_DELETE_APPROVED")

	wantPaths := []string{
		"/api/customObject/queryDeletedObjList",
		"/api/customObject/newPage",
		"/api/customObject/saveButton",
		"/api/customObject/editPage",
		"/api/customObject/saveButton",
		"/api/customObject/deleteLogic",
		"/api/customObject/deletePhysics",
	}
	if len(requests) != len(wantPaths) {
		t.Fatalf("expected %d requests, got %#v", len(wantPaths), requests)
	}
	for index, wantPath := range wantPaths {
		if requests[index].path != wantPath {
			t.Fatalf("request %d: want %s, got %#v", index, wantPath, requests[index])
		}
	}
	if requests[2].body["iscreatperm"] != "true" || requests[2].body["profileFieldArr"] == nil {
		t.Fatalf("object creation body was not normalized: %#v", requests[2].body)
	}
	if requests[3].body["objid"] != "object-1" || requests[4].body["objid"] != "object-1" || requests[6].body["objid"] != "object-1" {
		t.Fatalf("object lifecycle identifiers were not preserved: %#v", requests)
	}
}

func withUniversalCommandEdition(t *testing.T) {
	t.Helper()
	originalPackage, originalSuffix := edition.PackageName, edition.VersionSuffix
	originalDefault, originalStrict := edition.DefaultExecutionMode, edition.StrictExecutionMode
	t.Cleanup(func() {
		edition.PackageName, edition.VersionSuffix = originalPackage, originalSuffix
		edition.DefaultExecutionMode, edition.StrictExecutionMode = originalDefault, originalStrict
	})
	edition.PackageName = "cc-customization-expert-universal"
	edition.VersionSuffix = "-universal"
	edition.DefaultExecutionMode = "auto"
	edition.StrictExecutionMode = ""
}

func uiapiCommandTestProject(t *testing.T, serverURL string) string {
	t.Helper()
	projectPath := t.TempDir()
	config := `{"use":"dev","dev":{"accessToken":"cloud-token","apiSvc":"` + serverURL + `","setupSvc":"` + serverURL + `"}}`
	if err := os.WriteFile(filepath.Join(projectPath, "cloudcc-cli.config.json"), []byte(config), 0644); err != nil {
		t.Fatal(err)
	}
	return projectPath
}

func commandTestProject(t *testing.T) string {
	t.Helper()
	projectPath := t.TempDir()
	config := `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:9"}}}`
	if err := os.WriteFile(filepath.Join(projectPath, "cloudcc-cli.config.json"), []byte(config), 0644); err != nil {
		t.Fatal(err)
	}
	return projectPath
}
