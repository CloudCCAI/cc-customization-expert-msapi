package msapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestRecordTypeReadShortcutUsesDedicatedMetadataServiceQuery(t *testing.T) {
	paths := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.RequestURI())
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"found":true}`))
	}))
	defer server.Close()

	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json",
		`{"use":"dev","dev":{"metadataService":{"url":"`+server.URL+`"}}}`)
	var stdout bytes.Buffer
	if err := HandleLowCodeShortcut("getList", "recordType", []string{tmp, "account", "profile-1"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := HandleLowCodeShortcut("detail", "recordType", []string{tmp, "retail", "account"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}

	expected := []string{
		"/metadata/v1/record-types?object=account&profileId=profile-1",
		"/metadata/v1/record-types/retail?object=account",
	}
	if !reflect.DeepEqual(paths, expected) {
		t.Fatalf("unexpected record type query paths: got %#v want %#v", paths, expected)
	}
}

func TestRecordTypeReadShortcutRequiresObjectOrSelector(t *testing.T) {
	var stdout bytes.Buffer
	if err := HandleLowCodeShortcut("getList", "recordType", []string{t.TempDir()}, &stdout, ""); err == nil {
		t.Fatal("expected missing object selector to fail before request")
	}
	if err := HandleLowCodeShortcut("detail", "recordType", []string{t.TempDir()}, &stdout, ""); err == nil {
		t.Fatal("expected missing record-type selector to fail before request")
	}
}
