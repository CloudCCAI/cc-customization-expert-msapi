package jsonx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseEncodedObjectAcceptsJSONFile(t *testing.T) {
	tmp := t.TempDir()
	file := filepath.Join(tmp, "body.json")
	if err := os.WriteFile(file, []byte(`{"name":"go-only","enabled":true}`), 0644); err != nil {
		t.Fatal(err)
	}
	body, err := ParseEncodedObject("@"+file, "test")
	if err != nil {
		t.Fatal(err)
	}
	if body["name"] != "go-only" {
		t.Fatalf("unexpected body %#v", body)
	}
}

func TestParseEncodedObjectAcceptsRawJSON(t *testing.T) {
	body, err := ParseEncodedObject(`{"name":"raw-json"}`, "test")
	if err != nil {
		t.Fatal(err)
	}
	if body["name"] != "raw-json" {
		t.Fatalf("unexpected body %#v", body)
	}
}
