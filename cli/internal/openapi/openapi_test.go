package openapi

import "testing"

func TestNormalizeDataWrapsObject(t *testing.T) {
	got := normalizeData(map[string]any{"Name": "Acme"}, true)
	if got != `[{"Name":"Acme"}]` {
		t.Fatalf("unexpected normalized data: %v", got)
	}
}

func TestNormalizeDataKeepsString(t *testing.T) {
	got := normalizeData(`[{"Name":"Acme"}]`, true)
	if got != `[{"Name":"Acme"}]` {
		t.Fatalf("unexpected normalized data: %v", got)
	}
}
