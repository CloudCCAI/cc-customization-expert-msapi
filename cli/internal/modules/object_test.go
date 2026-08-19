package modules

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestObjectGetDeletedUsesRecycleBinEndpoint(t *testing.T) {
	var captured map[string]any
	projectPath, server := validationRuleTestProject(t, &captured)
	defer server.Close()

	var stdout, stderr bytes.Buffer
	if err := Handle("get", "object", []string{projectPath, "deleted"}, &stdout, &stderr, projectPath); err != nil {
		t.Fatal(err)
	}
	if captured["__path"] != "/api/customObject/queryDeletedObjList" {
		t.Fatalf("unexpected deleted object list path: %#v", captured)
	}
}

func TestObjectPurgeDefaultsToDryRun(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := Handle("purge", "object", []string{t.TempDir(), "obj-001"}, &stdout, &stderr, ""); err != nil {
		t.Fatal(err)
	}

	var out map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["dryRun"] != true || out["endpoint"] != "/api/customObject/deletePhysics" || out["objid"] != "obj-001" {
		t.Fatalf("unexpected dry-run payload: %#v", out)
	}
}

func TestObjectPurgeExecuteRequiresApproval(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Handle("purge", "object", []string{t.TempDir(), "obj-001", "--execute"}, &stdout, &stderr, "")
	if err == nil {
		t.Fatal("expected approval error")
	}
}

func TestObjectPurgeExecuteUsesPhysicalDeleteEndpoint(t *testing.T) {
	var captured map[string]any
	projectPath, server := validationRuleTestProject(t, &captured)
	defer server.Close()

	var stdout, stderr bytes.Buffer
	if err := Handle("purge", "object", []string{
		projectPath,
		"obj-001",
		"--execute",
		"--approval",
		"CLOUDCC_OBJECT_PHYSICAL_DELETE_APPROVED",
	}, &stdout, &stderr, projectPath); err != nil {
		t.Fatal(err)
	}
	if captured["__path"] != "/api/customObject/deletePhysics" || captured["objid"] != "obj-001" {
		t.Fatalf("unexpected physical delete request: %#v", captured)
	}
}
