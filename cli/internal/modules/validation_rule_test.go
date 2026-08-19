package modules

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestValidationRuleGetUsesObjectPrefixBody(t *testing.T) {
	var captured map[string]any
	projectPath, server := validationRuleTestProject(t, &captured)
	defer server.Close()

	var stdout, stderr bytes.Buffer
	if err := Handle("get", "validationRule", []string{projectPath, "b00"}, &stdout, &stderr, projectPath); err != nil {
		t.Fatal(err)
	}
	if captured["prefix"] != "b00" || captured["objectPrefix"] != "b00" {
		t.Fatalf("unexpected validationRule list body: %#v", captured)
	}
	if captured["__path"] != "/api/validateRule/queryByPrefix" {
		t.Fatalf("unexpected validationRule list path: %#v", captured)
	}
}

func TestValidationRuleCreatePositionalBuildsDocumentedBody(t *testing.T) {
	var captured map[string]any
	projectPath, server := validationRuleTestProject(t, &captured)
	defer server.Close()

	var stdout, stderr bytes.Buffer
	err := Handle("create", "validationRule", []string{
		projectPath,
		"b00",
		"DATE_RANGE",
		"end_date < start_date",
		"结束日期不得早于开始日期",
	}, &stdout, &stderr, projectPath)
	if err != nil {
		t.Fatal(err)
	}
	if captured["__path"] != "/api/validateRule/save" || captured["objid"] != "b00" || captured["prefix"] != "b00" {
		t.Fatalf("unexpected validationRule save envelope: %#v", captured)
	}
	validate := captured["validate"].(map[string]any)
	expected := map[string]string{
		"name":         "DATE_RANGE",
		"ruleName":     "DATE_RANGE",
		"functionCode": "end_date < start_date",
		"ruleContent":  "end_date < start_date",
		"errorMessage": "结束日期不得早于开始日期",
		"isactive":     "false",
		"objId":        "b00",
	}
	for key, want := range expected {
		if validate[key] != want {
			t.Fatalf("validate[%s] = %#v, want %q; body=%#v", key, validate[key], want, captured)
		}
	}
}

func TestValidationRuleCreateAcceptsRawJSONBody(t *testing.T) {
	var captured map[string]any
	projectPath, server := validationRuleTestProject(t, &captured)
	defer server.Close()

	var stdout, stderr bytes.Buffer
	err := Handle("create", "validationRule", []string{
		projectPath,
		`{"prefix":"b01","name":"RAW_RULE","ruleContent":"amount < 0","errorMessage":"金额不可为负"}`,
	}, &stdout, &stderr, projectPath)
	if err != nil {
		t.Fatal(err)
	}
	if captured["prefix"] != "b01" || captured["name"] != "RAW_RULE" {
		t.Fatalf("unexpected raw validationRule body: %#v", captured)
	}
	if captured["__path"] != "/api/validateRule/save" {
		t.Fatalf("unexpected raw validationRule path: %#v", captured)
	}
}

func TestValidationRuleCreatePrefixOnlyReturnsNonInteractiveError(t *testing.T) {
	var captured map[string]any
	projectPath, server := validationRuleTestProject(t, &captured)
	defer server.Close()

	var stdout, stderr bytes.Buffer
	err := Handle("create", "validationRule", []string{projectPath, "b00"}, &stdout, &stderr, projectPath)
	if err == nil {
		t.Fatal("expected non-interactive error")
	}
	if captured != nil {
		t.Fatalf("prefix-only create must not call API, captured=%#v", captured)
	}
}

func TestValidationRuleDeleteBuildsRuleIDBody(t *testing.T) {
	var captured map[string]any
	projectPath, server := validationRuleTestProject(t, &captured)
	defer server.Close()

	var stdout, stderr bytes.Buffer
	if err := Handle("delete", "validationRule", []string{projectPath, "vr-001"}, &stdout, &stderr, projectPath); err != nil {
		t.Fatal(err)
	}
	if captured["id"] != "vr-001" || captured["validationRuleId"] != "vr-001" {
		t.Fatalf("unexpected validationRule delete body: %#v", captured)
	}
	if captured["__path"] != "/api/validateRule/delete" {
		t.Fatalf("unexpected validationRule delete path: %#v", captured)
	}
}

func validationRuleTestProject(t *testing.T, captured *map[string]any) (string, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("accessToken") != "token" {
			t.Fatalf("expected accessToken header, got %q", r.Header.Get("accessToken"))
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		body["__path"] = r.URL.Path
		*captured = body
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":true,"data":[]}`))
	}))
	projectPath := t.TempDir()
	pkg := `{"devConsoleConfig":{"accessToken":"token","setupSvc":"` + server.URL + `","apiSvc":"` + server.URL + `"}}`
	if err := os.WriteFile(filepath.Join(projectPath, "package.json"), []byte(pkg), 0644); err != nil {
		t.Fatal(err)
	}
	return projectPath, server
}
