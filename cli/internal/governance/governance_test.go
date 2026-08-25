package governance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidatePassesGenericProjectStandardGovernance(t *testing.T) {
	project := t.TempDir()
	standardPath := "docs/delivery/demo-manufacturing-crm/00-governance/standards/01-process-diagram-standard.md"
	writeGovernanceTestFile(t, filepath.Join(project, "AGENTS.md"), "Before process work read STD-001 at `"+standardPath+"`.\n")
	writeGovernanceTestFile(t, filepath.Join(project, "docs/delivery/demo-manufacturing-crm/00-governance/standards/00-standard-index.md"), `---
kind: project-standard-index
title: Demo standards
status: active
---
| ID | Standard |
|---|---|
| STD-001 | [Process](01-process-diagram-standard.md) |
`)
	writeGovernanceTestFile(t, filepath.Join(project, standardPath), `---
kind: project-standard
standard_id: STD-001
title: Process diagram standard
status: active
version: 1.0
owner_role: project-manager
effective_date: 2026-08-25
review_trigger: Project phase or diagram tooling changes
---
# Standard
`)
	writeGovernanceTestFile(t, filepath.Join(project, "docs/delivery/demo-manufacturing-crm/01-blueprint/processes/00-process-index.md"), `---
kind: process-diagram-index
title: Process index
status: active
---
- Standard: ../../00-governance/standards/01-process-diagram-standard.md
- Facts: ../../../../specs/FEAT-001-example.md
- Priority: ../../../../../.claw/task-board.md
`)

	report, err := Validate(project)
	if err != nil {
		t.Fatalf("expected governance validation to pass: %v, report=%+v", err, report)
	}
	if report.Status != "passed" || report.Summary["standardCount"] != 1 || report.Summary["activeCount"] != 1 {
		t.Fatalf("unexpected report: %+v", report)
	}
	standard := report.DeliveryRoots[0].Standards[0]
	if !standard.Indexed || !standard.GateRead {
		t.Fatalf("expected indexed standard with AGENTS read gate: %+v", standard)
	}
}

func TestValidateFailsMissingReadGateDuplicateIDAndArchiveConflict(t *testing.T) {
	project := t.TempDir()
	writeGovernanceTestFile(t, filepath.Join(project, "AGENTS.md"), "# AGENTS\n")
	indexPath := filepath.Join(project, "docs/delivery/demo-crm/00-governance/standards/00-standard-index.md")
	writeGovernanceTestFile(t, indexPath, `---
kind: project-standard-index
title: Demo standards
status: active
---
| STD-001 | [One](01-one.md) |
| STD-001 | [Two](archive/02-two-v1.0.md) |
`)
	base := `---
kind: project-standard
standard_id: STD-001
title: Demo
status: %s
version: 1.0
owner_role: project-manager
effective_date: 2026-08-25
review_trigger: Tooling changes
---
`
	writeGovernanceTestFile(t, filepath.Join(project, "docs/delivery/demo-crm/00-governance/standards/01-one.md"), strings.Replace(base, "%s", "active", 1))
	writeGovernanceTestFile(t, filepath.Join(project, "docs/delivery/demo-crm/00-governance/standards/archive/02-two-v1.0.md"), strings.Replace(base, "%s", "active", 1))

	report, err := Validate(project)
	if err == nil || report.Status != "failed" {
		t.Fatalf("expected failed report, err=%v report=%+v", err, report)
	}
	for _, code := range []string{"active_standard_read_gate_missing", "duplicate_standard_id", "active_standard_in_archive", "unstable_standard_filename"} {
		if !hasIssueCode(report.Errors, code) {
			t.Fatalf("expected issue %s, got %+v", code, report.Errors)
		}
	}
}

func TestValidateReturnsNotAdoptedWithoutDeliveryTree(t *testing.T) {
	report, err := Validate(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "not_adopted" || !report.ReadOnly {
		t.Fatalf("unexpected not-adopted report: %+v", report)
	}
}

func hasIssueCode(issues []Issue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func writeGovernanceTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
