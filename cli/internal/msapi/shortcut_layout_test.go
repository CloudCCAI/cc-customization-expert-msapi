package msapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestPageLayoutReadShortcutUsesDedicatedMetadataServiceQuery(t *testing.T) {
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
	if err := HandleLowCodeShortcut("get", "pagelayout", []string{tmp, "account"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := HandleLowCodeShortcut("detail", "pagelayout", []string{tmp, "account", "account_layout"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}

	expected := []string{
		"/metadata/v1/layouts?object=account",
		"/metadata/v1/layouts/account_layout?object=account",
	}
	if !reflect.DeepEqual(paths, expected) {
		t.Fatalf("unexpected layout query paths: got %#v want %#v", paths, expected)
	}
}

func TestPageLayoutReadShortcutRequiresObjectAndDetailSelector(t *testing.T) {
	var stdout bytes.Buffer
	if err := HandleLowCodeShortcut("get", "pagelayout", []string{t.TempDir()}, &stdout, ""); err == nil {
		t.Fatal("expected missing object selector to fail before request")
	}
	if err := HandleLowCodeShortcut("detail", "pagelayout", []string{t.TempDir(), "account"}, &stdout, ""); err == nil {
		t.Fatal("expected missing layout selector to fail before request")
	}
}
