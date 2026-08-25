package testgovernance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAdvisoryTestingLifecycle(t *testing.T) {
	project := t.TempDir()
	initResult, err := Init(project, "demo-crm")
	if err != nil {
		t.Fatal(err)
	}
	if initResult.Status != "initialized" || len(initResult.Created) == 0 {
		t.Fatalf("unexpected init result: %+v", initResult)
	}
	if _, err := os.Stat(filepath.Join(project, ".claw")); !os.IsNotExist(err) {
		t.Fatalf("initializer must not create or overwrite .claw: %v", err)
	}

	scenarioPath := "test-assets/scenarios/opportunity/E2E-OPP-001.json"
	writeTestGovernanceFile(t, filepath.Join(project, filepath.FromSlash(scenarioPath)), `{"schemaVersion":"cloudcc-test-scenario/v1","scenarioId":"E2E-OPP-001"}`)
	writeTestGovernanceJSON(t, filepath.Join(project, "test-assets", "catalog", "impact-map.json"), ImpactMap{
		SchemaVersion: ImpactMapSchemaVersion,
		Resources: []ImpactEntry{{
			Kind: "trigger", Name: "OpportunityTrigger", Module: "opportunity",
			RecommendedScope: "affected-chain", AffectedModules: []string{"opportunity", "quote"},
			ScenarioIDs: []string{"E2E-OPP-001"}, SuiteIDs: []string{"opportunity-regression"},
			RiskTags: []string{"downstream-write"}, Reason: "Trigger updates quote state.",
		}},
	})
	writeTestGovernanceJSON(t, filepath.Join(project, "test-assets", "catalog", "scenario-index.json"), ScenarioIndex{
		SchemaVersion: ScenarioIndexSchemaVersion,
		Scenarios:     []ScenarioEntry{{ScenarioID: "E2E-OPP-001", Module: "opportunity", Path: scenarioPath, EstimatedMinutes: 12}},
	})

	recommendation, err := Advise(project, ChangeRequest{
		SchemaVersion: ChangeSchemaVersion,
		ChangeSetID:   "CHG-001",
		Phase:         "feature-development",
		Resources:     []ResourceChange{{Kind: "trigger", Name: "OpportunityTrigger", Module: "opportunity"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !recommendation.Advisory || recommendation.Blocking || recommendation.RecommendedScope != "affected-chain" || recommendation.CatalogCoverage != "matched" {
		t.Fatalf("unexpected recommendation: %+v", recommendation)
	}
	if recommendation.EstimatedMinutes != 12 || recommendation.RecommendationHash == "" || len(recommendation.ScenarioIDs) != 1 {
		t.Fatalf("missing recommendation evidence: %+v", recommendation)
	}

	decisionResult, err := Decide(project, DecisionRequest{
		SchemaVersion:  DecisionSchemaVersion,
		Recommendation: recommendation,
		SelectedScope:  "smoke",
		ConfirmedBy:    "human",
		DecidedByRole:  "project-manager",
		DecidedAt:      "2026-08-25T04:00:00Z",
		Reason:         "Feature is still changing; run the core path now and expand before UAT.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if decisionResult.Decision.SelectedScope != "smoke" || decisionResult.Decision.VerificationState != "pending_execution" {
		t.Fatalf("unexpected decision: %+v", decisionResult)
	}

	runResult, err := RecordRun(project, RunRequest{
		SchemaVersion:            RunSchemaVersion,
		RunID:                    "TEST-RUN-001",
		ChangeSetID:              "CHG-001",
		SourceRevision:           "revision-001",
		Environment:              "SIT",
		EnvironmentFingerprint:   "sit-fingerprint-001",
		TestAssetRevision:        "assets-revision-001",
		Status:                   "passed",
		StartedAt:                "2026-08-25T04:10:00Z",
		CompletedAt:              "2026-08-25T04:20:00Z",
		ScenarioResults:          []ScenarioResult{{ScenarioID: "E2E-OPP-001", Status: "passed", Evidence: "readbacks/opportunity.json"}},
		BusinessAcceptanceStatus: "pending",
	})
	if err != nil {
		t.Fatal(err)
	}
	if runResult.Manifest.SelectedScope != "smoke" || runResult.Manifest.VerificationState != "pending_uat" {
		t.Fatalf("unexpected run manifest: %+v", runResult.Manifest)
	}

	report, err := Doctor(project)
	if err != nil {
		t.Fatalf("expected doctor to pass: %v report=%+v", err, report)
	}
	if report.Status != "passed" || report.Summary["decisionCount"] != 1 || report.Summary["runCount"] != 1 || report.Summary["scenarioCount"] != 1 {
		t.Fatalf("unexpected doctor report: %+v", report)
	}
}

func TestInitDoesNotOverwriteExistingAssets(t *testing.T) {
	project := t.TempDir()
	if _, err := Init(project, "demo-crm"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(project, "test-assets", "README.md")
	if err := os.WriteFile(path, []byte("project-owned\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := Init(project, "demo-crm")
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "project-owned\n" || !containsFile(result.Skipped, "test-assets/README.md") {
		t.Fatalf("initializer overwrote existing asset or failed to report skip: result=%+v content=%q", result, b)
	}
}

func TestAdviseWithoutCatalogIsNonBlockingAndConservative(t *testing.T) {
	project := t.TempDir()
	recommendation, err := Advise(project, ChangeRequest{
		SchemaVersion: ChangeSchemaVersion,
		ChangeSetID:   "CHG-SHARED-001",
		Phase:         "development",
		Resources:     []ResourceChange{{Kind: "class", Name: "SharedBusinessService", Shared: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if recommendation.RecommendedScope != "full-core" || recommendation.CatalogCoverage != "missing" || recommendation.Blocking {
		t.Fatalf("unexpected conservative recommendation: %+v", recommendation)
	}
	if !containsString(recommendation.RiskTags, "catalog-gap") || !containsString(recommendation.RiskTags, "shared-resource") {
		t.Fatalf("expected catalog/shared risk tags: %+v", recommendation.RiskTags)
	}
}

func TestSkipDecisionRequiresReasonAndRecordsRiskAccepted(t *testing.T) {
	project := t.TempDir()
	if _, err := Init(project, "demo-crm"); err != nil {
		t.Fatal(err)
	}
	recommendation, err := Advise(project, ChangeRequest{
		SchemaVersion: ChangeSchemaVersion,
		ChangeSetID:   "CHG-SKIP-001",
		Phase:         "development",
		Resources:     []ResourceChange{{Kind: "layout", Name: "AccountLayout"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	base := DecisionRequest{
		SchemaVersion:  DecisionSchemaVersion,
		Recommendation: recommendation,
		SelectedScope:  "skip",
		ConfirmedBy:    "human",
		DecidedByRole:  "project-manager",
		DecidedAt:      "2026-08-25T05:00:00Z",
	}
	if _, err := Decide(project, base); err == nil || !strings.Contains(err.Error(), "reason is required") {
		t.Fatalf("expected missing reason rejection, got %v", err)
	}
	base.Reason = "Prototype layout is still being revised."
	result, err := Decide(project, base)
	if err != nil {
		t.Fatal(err)
	}
	if result.Decision.VerificationState != "risk_accepted" {
		t.Fatalf("expected risk_accepted, got %+v", result.Decision)
	}
	if _, err := Decide(project, base); err == nil || !strings.Contains(err.Error(), "immutable") {
		t.Fatalf("expected immutable decision rejection, got %v", err)
	}
}

func TestDoctorRejectsRecommendationDriftAndSensitiveFiles(t *testing.T) {
	project := t.TempDir()
	if _, err := Init(project, "demo-crm"); err != nil {
		t.Fatal(err)
	}
	recommendation, err := Advise(project, ChangeRequest{
		SchemaVersion: ChangeSchemaVersion,
		ChangeSetID:   "CHG-DRIFT-001",
		Phase:         "development",
		Resources:     []ResourceChange{{Kind: "field", Name: "Account.NewField"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decide(project, DecisionRequest{
		SchemaVersion:  DecisionSchemaVersion,
		Recommendation: recommendation,
		SelectedScope:  recommendation.RecommendedScope,
		ConfirmedBy:    "human",
		DecidedByRole:  "qa-lead",
		DecidedAt:      "2026-08-25T06:00:00Z",
	}); err != nil {
		t.Fatal(err)
	}
	recommendationPath := filepath.Join(project, "evidence", "testing", "decisions", "CHG-DRIFT-001", "recommendation.json")
	var drifted Recommendation
	readTestGovernanceJSON(t, recommendationPath, &drifted)
	drifted.RecommendedScope = "full-core"
	writeTestGovernanceJSON(t, recommendationPath, drifted)
	writeTestGovernanceFile(t, filepath.Join(project, "test-assets", "fixtures", ".env"), "TOKEN=should-not-be-here\n")
	report, err := Doctor(project)
	if err == nil || report.Status != "failed" {
		t.Fatalf("expected failed doctor report: err=%v report=%+v", err, report)
	}
	if !hasDoctorIssue(report.Errors, "recommendation_contract_invalid") || !hasDoctorIssue(report.Errors, "sensitive_file_forbidden") {
		t.Fatalf("expected drift and sensitive-file errors, got %+v", report.Errors)
	}
}

func writeTestGovernanceFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeTestGovernanceJSON(t *testing.T, path string, value any) {
	t.Helper()
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeTestGovernanceFile(t, path, string(append(b, '\n')))
}

func readTestGovernanceJSON(t *testing.T, path string, target any) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, target); err != nil {
		t.Fatal(err)
	}
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func hasDoctorIssue(issues []DoctorIssue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
