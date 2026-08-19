package msapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestPermissionSetReadShortcutUsesDedicatedMetadataServiceQuery(t *testing.T) {
	paths := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.RequestURI())
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/metadata/v1/permission-sets" {
			_, _ = w.Write([]byte(`{"count":1,"permissionSets":[{"id":"perm-001","name":"Sales","apiName":"sales"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"permissionSet":{"id":"perm-001"}}`))
	}))
	defer server.Close()

	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json",
		`{"use":"dev","dev":{"metadataService":{"url":"`+server.URL+`"}}}`)
	var stdout bytes.Buffer
	if err := HandleLowCodeShortcut("getList", "permission", []string{tmp, "sales team"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := HandleLowCodeShortcut("detail", "permission", []string{tmp, "sales"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}

	expected := []string{
		"/metadata/v1/permission-sets?filter=sales+team",
		"/metadata/v1/permission-sets?selector=sales",
		"/metadata/v1/permission-sets/perm-001",
	}
	if !reflect.DeepEqual(paths, expected) {
		t.Fatalf("unexpected permission-set query paths: got %#v want %#v", paths, expected)
	}
}

func TestPermissionSetReadShortcutRejectsMissingOrAmbiguousSelector(t *testing.T) {
	var stdout bytes.Buffer
	if err := HandleLowCodeShortcut("detail", "permission", []string{t.TempDir()}, &stdout, ""); err == nil {
		t.Fatal("expected missing permission-set selector to fail before request")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"count":2,"permissionSets":[{"id":"perm-001"},{"id":"perm-002"}]}`))
	}))
	defer server.Close()
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json",
		`{"use":"dev","dev":{"metadataService":{"url":"`+server.URL+`"}}}`)
	if err := HandleLowCodeShortcut("detail", "permission", []string{tmp, "duplicate"}, &stdout, tmp); err == nil {
		t.Fatal("expected ambiguous permission-set selector to fail")
	}
}
