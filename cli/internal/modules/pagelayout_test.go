package modules

import (
	"encoding/json"
	"testing"
)

func TestNormalizePageLayoutJSONStripsRuntimeFields(t *testing.T) {
	layout := map[string]any{
		"sections": []any{
			map[string]any{
				"sectionId":         "sec-1",
				"sectionName":       "基本信息",
				"labelKey":          "基本信息",
				"showDetailHeader":  true,
				"showEditHeader":    true,
				"sortOrder":         "v",
				"categoriesAllowed": map[string]any{"Control": true},
				"canChangeColumns":  true,
				"canDeleteSection":  false,
				"columns":           []any{[]any{}, []any{}},
			},
		},
	}
	raw, err := normalizePageLayoutJSON(layout, "layout-1")
	if err != nil {
		t.Fatal(err)
	}

	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatal(err)
	}
	sections, ok := out["sections"].([]any)
	if !ok || len(sections) != 1 {
		t.Fatalf("expected sections payload, got %#v", out["sections"])
	}
	m := sections[0].(map[string]any)
	if _, ok := m["sortOrder"]; ok {
		t.Fatalf("sortOrder should be removed before save")
	}
	if _, ok := m["categoriesAllowed"]; ok {
		t.Fatalf("categoriesAllowed should be removed before save")
	}
	if _, ok := m["canChangeColumns"]; ok {
		t.Fatalf("canChangeColumns should be removed before save")
	}
	if _, ok := m["canDeleteSection"]; ok {
		t.Fatalf("canDeleteSection should be removed before save")
	}
}

func TestNormalizePageLayoutJSONAcceptsDataWrapper(t *testing.T) {
	layout := map[string]any{
		"data": map[string]any{
			"layoutid": "layout-2",
			"sections": []any{
				map[string]any{
					"sectionId":        "sec-2",
					"sectionName":      "联系人",
					"labelKey":         "联系人",
					"showDetailHeader": true,
					"showEditHeader":   true,
					"columns":          []any{[]any{}},
				},
			},
		},
	}
	raw, err := normalizePageLayoutJSON(layout, "")
	if err != nil {
		t.Fatal(err)
	}

	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatal(err)
	}
	sections, ok := out["sections"].([]any)
	if !ok || len(sections) != 1 {
		t.Fatalf("expected normalized sections, got %#v", out["sections"])
	}
}

func TestNormalizePageLayoutJSONRequiresSectionId(t *testing.T) {
	_, err := normalizePageLayoutJSON(map[string]any{
		"sections": []any{map[string]any{"sectionName": "no-id"}},
	}, "layout-3")
	if err == nil {
		t.Fatal("expected sectionId validation error")
	}
}
