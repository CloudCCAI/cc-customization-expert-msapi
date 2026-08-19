package msapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestGlobalSelectReadShortcutUsesDedicatedMetadataServiceQuery(t *testing.T) {
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
	if err := HandleLowCodeShortcut("getList", "globalSelectList", []string{tmp, "2", "50"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := HandleLowCodeShortcut("detail", "globalSelectList", []string{tmp, "season"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}

	expected := []string{
		"/metadata/v1/global-select-lists?page=2&pageSize=50",
		"/metadata/v1/global-select-lists/season",
	}
	if !reflect.DeepEqual(paths, expected) {
		t.Fatalf("unexpected global select query paths: got %#v want %#v", paths, expected)
	}
}

func TestGlobalSelectReadShortcutValidatesPagination(t *testing.T) {
	var stdout bytes.Buffer
	if err := HandleLowCodeShortcut("getList", "globalSelectList", []string{t.TempDir(), "0"}, &stdout, ""); err == nil {
		t.Fatal("expected invalid page to fail before request")
	}
}
