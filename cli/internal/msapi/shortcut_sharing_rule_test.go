package msapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestSharingRuleReadShortcutUsesDedicatedMetadataServiceQuery(t *testing.T) {
	paths := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.RequestURI())
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/metadata/v1/sharing-rules" {
			_, _ = w.Write([]byte(`{"count":1,"sharingRules":[{"id":"rule-sales","apiName":"rule_sales"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"sharingRule":{"id":"rule-sales"}}`))
	}))
	defer server.Close()

	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json",
		`{"use":"dev","dev":{"metadataService":{"url":"`+server.URL+`"}}}`)
	var stdout bytes.Buffer
	if err := HandleLowCodeShortcut("getList", "sharingRule", []string{tmp, "sales team"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := HandleLowCodeShortcut("detail", "sharingRule", []string{tmp, "rule_sales"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := HandleLowCodeShortcut("get", "sharingRule", []string{tmp, `{"objectId":"obj-contract"}`}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}

	expected := []string{
		"/metadata/v1/sharing-rules?filter=sales+team",
		"/metadata/v1/sharing-rules?selector=rule_sales",
		"/metadata/v1/sharing-rules/rule-sales",
		"/metadata/v1/sharing-rules?object=obj-contract",
	}
	if !reflect.DeepEqual(paths, expected) {
		t.Fatalf("unexpected sharing-rule query paths: got %#v want %#v", paths, expected)
	}
}

func TestSharingRuleReadShortcutRejectsMissingOrAmbiguousSelector(t *testing.T) {
	var stdout bytes.Buffer
	if err := HandleLowCodeShortcut("detail", "sharingRule", []string{t.TempDir()}, &stdout, ""); err == nil {
		t.Fatal("expected missing sharing-rule selector to fail before request")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"count":2,"sharingRules":[{"id":"rule-001"},{"id":"rule-002"}]}`))
	}))
	defer server.Close()
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json",
		`{"use":"dev","dev":{"metadataService":{"url":"`+server.URL+`"}}}`)
	if err := HandleLowCodeShortcut("detail", "sharingRule", []string{tmp, "duplicate"}, &stdout, tmp); err == nil {
		t.Fatal("expected ambiguous sharing-rule selector to fail")
	}
}
