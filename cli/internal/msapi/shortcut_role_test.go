package msapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestRoleReadShortcutUsesDedicatedMetadataServiceQuery(t *testing.T) {
	paths := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.RequestURI())
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/metadata/v1/roles" {
			_, _ = w.Write([]byte(`{"count":1,"roles":[{"id":"role-sales","name":"Sales"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"role":{"id":"role-sales"}}`))
	}))
	defer server.Close()

	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json",
		`{"use":"dev","dev":{"metadataService":{"url":"`+server.URL+`"}}}`)
	var stdout bytes.Buffer
	if err := HandleLowCodeShortcut("getList", "role", []string{tmp, "sales team"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := HandleLowCodeShortcut("detail", "role", []string{tmp, "Sales"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}

	expected := []string{
		"/metadata/v1/roles?filter=sales+team",
		"/metadata/v1/roles?selector=Sales",
		"/metadata/v1/roles/role-sales",
	}
	if !reflect.DeepEqual(paths, expected) {
		t.Fatalf("unexpected role query paths: got %#v want %#v", paths, expected)
	}
}

func TestRoleReadShortcutRejectsMissingOrAmbiguousSelector(t *testing.T) {
	var stdout bytes.Buffer
	if err := HandleLowCodeShortcut("detail", "role", []string{t.TempDir()}, &stdout, ""); err == nil {
		t.Fatal("expected missing role selector to fail before request")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"count":2,"roles":[{"id":"role-001"},{"id":"role-002"}]}`))
	}))
	defer server.Close()
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json",
		`{"use":"dev","dev":{"metadataService":{"url":"`+server.URL+`"}}}`)
	if err := HandleLowCodeShortcut("detail", "role", []string{tmp, "duplicate"}, &stdout, tmp); err == nil {
		t.Fatal("expected ambiguous role selector to fail")
	}
}
