package testgovernance

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type DoctorIssue struct {
	Code    string `json:"code"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
}

type DoctorReport struct {
	Status      string         `json:"status"`
	ProjectPath string         `json:"projectPath"`
	ReadOnly    bool           `json:"readOnly"`
	ProjectCode string         `json:"projectCode,omitempty"`
	Errors      []DoctorIssue  `json:"errors"`
	Warnings    []DoctorIssue  `json:"warnings"`
	Summary     map[string]int `json:"summary"`
}

func WriteDoctor(projectPath string, stdout io.Writer) error {
	report, err := Doctor(projectPath)
	if encodeErr := json.NewEncoder(stdout).Encode(report); encodeErr != nil {
		return encodeErr
	}
	return err
}

func Doctor(projectPath string) (DoctorReport, error) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return DoctorReport{}, err
	}
	report := DoctorReport{
		Status:      "passed",
		ProjectPath: absPath,
		ReadOnly:    true,
		Summary: map[string]int{
			"decisionCount":    0,
			"runCount":         0,
			"scenarioCount":    0,
			"impactEntryCount": 0,
			"errorCount":       0,
			"warningCount":     0,
		},
	}
	deliveryBase := filepath.Join(absPath, "docs", "delivery")
	entries, err := os.ReadDir(deliveryBase)
	if os.IsNotExist(err) {
		report.Status = "not_adopted"
		return report, nil
	}
	if err != nil {
		return report, err
	}
	var projectCodes []string
	for _, entry := range entries {
		if entry.IsDir() {
			if _, err := os.Stat(filepath.Join(deliveryBase, entry.Name(), "07-testing-cutover")); err == nil {
				projectCodes = append(projectCodes, entry.Name())
			}
		}
	}
	if len(projectCodes) == 0 {
		report.Status = "not_adopted"
		return report, nil
	}
	sort.Strings(projectCodes)
	if len(projectCodes) > 1 {
		report.Errors = append(report.Errors, DoctorIssue{Code: "multiple_testing_delivery_roots", Path: "docs/delivery", Message: "test-governance doctor requires one adopted project delivery root"})
	}
	report.ProjectCode = projectCodes[0]
	deliveryRoot := filepath.Join(deliveryBase, report.ProjectCode)
	for _, rel := range requiredTestingPaths() {
		path := filepath.Join(absPath, filepath.FromSlash(strings.ReplaceAll(rel, "{{DELIVERY_ROOT}}", relPath(absPath, deliveryRoot))))
		if _, err := os.Stat(path); err != nil {
			report.Errors = append(report.Errors, DoctorIssue{Code: "required_asset_missing", Path: relPath(absPath, path), Message: "required test-governance asset is missing"})
		}
	}

	impactMap, impactLoaded, impactErr := loadImpactMap(filepath.Join(absPath, "test-assets", "catalog", "impact-map.json"))
	if impactErr != nil {
		report.Errors = append(report.Errors, DoctorIssue{Code: "impact_map_invalid", Path: "test-assets/catalog/impact-map.json", Message: impactErr.Error()})
	} else if impactLoaded {
		report.Summary["impactEntryCount"] = len(impactMap.Resources)
		for _, entry := range impactMap.Resources {
			if normalizeKind(entry.Kind) == "" {
				report.Errors = append(report.Errors, DoctorIssue{Code: "impact_entry_kind_missing", Path: "test-assets/catalog/impact-map.json", Message: "impact entry kind is required"})
			}
			if entry.RecommendedScope != "" {
				if _, ok := scopeRanks[normalizeScope(entry.RecommendedScope)]; !ok || normalizeScope(entry.RecommendedScope) == "skip" {
					report.Errors = append(report.Errors, DoctorIssue{Code: "impact_entry_scope_invalid", Path: "test-assets/catalog/impact-map.json", Message: "impact entry recommendedScope is invalid"})
				}
			}
		}
	}
	scenarioIndex, scenarioLoaded, scenarioErr := loadScenarioIndex(filepath.Join(absPath, "test-assets", "catalog", "scenario-index.json"))
	if scenarioErr != nil {
		report.Errors = append(report.Errors, DoctorIssue{Code: "scenario_index_invalid", Path: "test-assets/catalog/scenario-index.json", Message: scenarioErr.Error()})
	} else if scenarioLoaded {
		report.Summary["scenarioCount"] = len(scenarioIndex.Scenarios)
		seen := map[string]bool{}
		for _, scenario := range scenarioIndex.Scenarios {
			if !validID(scenario.ScenarioID) {
				report.Errors = append(report.Errors, DoctorIssue{Code: "scenario_id_invalid", Path: "test-assets/catalog/scenario-index.json", Message: "scenarioId is missing or invalid"})
			} else if seen[scenario.ScenarioID] {
				report.Errors = append(report.Errors, DoctorIssue{Code: "scenario_id_duplicate", Path: "test-assets/catalog/scenario-index.json", Message: "scenarioId must be unique"})
			}
			seen[scenario.ScenarioID] = true
			if scenario.Path != "" {
				target := filepath.Join(absPath, filepath.FromSlash(scenario.Path))
				if _, err := os.Stat(target); err != nil {
					report.Errors = append(report.Errors, DoctorIssue{Code: "scenario_asset_missing", Path: scenario.Path, Message: "scenario index references a missing asset"})
				}
			}
		}
	}

	validateDecisionTree(absPath, &report)
	validateRunTree(absPath, &report)
	validateSensitiveFiles(absPath, &report)
	if report.Summary["impactEntryCount"] == 0 || report.Summary["scenarioCount"] == 0 {
		report.Warnings = append(report.Warnings, DoctorIssue{Code: "catalog_not_populated", Path: "test-assets/catalog", Message: "advisory recommendations will report catalog gaps until impact and scenario catalogs are populated"})
	}
	report.Summary["errorCount"] = len(report.Errors)
	report.Summary["warningCount"] = len(report.Warnings)
	if len(report.Errors) > 0 {
		report.Status = "failed"
		return report, fmt.Errorf("test-governance doctor found %d error(s)", len(report.Errors))
	}
	return report, nil
}

func requiredTestingPaths() []string {
	return []string{
		"{{DELIVERY_ROOT}}/00-governance/standards/10-test-governance-standard.md",
		"{{DELIVERY_ROOT}}/07-testing-cutover/00-testing-cutover-index.md",
		"{{DELIVERY_ROOT}}/07-testing-cutover/01-uat-scenario-matrix.md",
		"{{DELIVERY_ROOT}}/07-testing-cutover/02-permission-acceptance-matrix.md",
		"{{DELIVERY_ROOT}}/07-testing-cutover/03-cutover-runbook.md",
		"{{DELIVERY_ROOT}}/07-testing-cutover/04-rollback-runbook.md",
		"{{DELIVERY_ROOT}}/07-testing-cutover/05-test-impact-matrix.md",
		"{{DELIVERY_ROOT}}/07-testing-cutover/06-test-data-catalog.md",
		"{{DELIVERY_ROOT}}/07-testing-cutover/07-environment-account-matrix.md",
		"{{DELIVERY_ROOT}}/07-testing-cutover/08-requirement-test-traceability.md",
		"{{DELIVERY_ROOT}}/08-release-evidence/00-release-evidence-index.md",
		"{{DELIVERY_ROOT}}/08-release-evidence/01-msapi-plan-apply-changes.md",
		"{{DELIVERY_ROOT}}/08-release-evidence/02-production-readiness-gate.md",
		"{{DELIVERY_ROOT}}/08-release-evidence/03-post-release-verification.md",
		"test-assets/README.md",
		"test-assets/catalog/impact-map.json",
		"test-assets/catalog/scenario-index.json",
		"test-assets/schemas/change.schema.json",
		"test-assets/schemas/decision.schema.json",
		"test-assets/schemas/run.schema.json",
		"evidence/testing/decisions",
		"evidence/testing/runs",
	}
}

func validateDecisionTree(projectPath string, report *DoctorReport) {
	root := filepath.Join(projectPath, "evidence", "testing", "decisions")
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		report.Summary["decisionCount"]++
		if !validID(entry.Name()) {
			report.Errors = append(report.Errors, DoctorIssue{Code: "decision_id_invalid", Path: relPath(projectPath, filepath.Join(root, entry.Name())), Message: "decision directory name is invalid"})
			continue
		}
		recommendationPath := filepath.Join(root, entry.Name(), "recommendation.json")
		decisionPath := filepath.Join(root, entry.Name(), "human-decision.json")
		var recommendation Recommendation
		if err := readJSON(recommendationPath, &recommendation); err != nil {
			report.Errors = append(report.Errors, DoctorIssue{Code: "recommendation_invalid", Path: relPath(projectPath, recommendationPath), Message: err.Error()})
			continue
		}
		if err := validateRecommendation(recommendation); err != nil {
			report.Errors = append(report.Errors, DoctorIssue{Code: "recommendation_contract_invalid", Path: relPath(projectPath, recommendationPath), Message: err.Error()})
		}
		decision, err := readDecision(decisionPath)
		if err != nil {
			report.Errors = append(report.Errors, DoctorIssue{Code: "decision_invalid", Path: relPath(projectPath, decisionPath), Message: err.Error()})
			continue
		}
		if decision.ChangeSetID != entry.Name() || decision.RecommendationHash != recommendation.RecommendationHash {
			report.Errors = append(report.Errors, DoctorIssue{Code: "decision_recommendation_mismatch", Path: relPath(projectPath, decisionPath), Message: "decision identity or recommendation hash does not match"})
		}
	}
}

func validateRunTree(projectPath string, report *DoctorReport) {
	root := filepath.Join(projectPath, "evidence", "testing", "runs")
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		report.Summary["runCount"]++
		manifestPath := filepath.Join(root, entry.Name(), "manifest.json")
		var manifest RunManifest
		if err := readJSON(manifestPath, &manifest); err != nil {
			report.Errors = append(report.Errors, DoctorIssue{Code: "run_manifest_invalid", Path: relPath(projectPath, manifestPath), Message: err.Error()})
			continue
		}
		if manifest.SchemaVersion != RunSchemaVersion || manifest.RunID != entry.Name() || !validID(manifest.ChangeSetID) || !supportedRunStatus(manifest.Status) {
			report.Errors = append(report.Errors, DoctorIssue{Code: "run_contract_invalid", Path: relPath(projectPath, manifestPath), Message: "run manifest identity, schema, or status is invalid"})
		}
		decisionPath := filepath.Join(projectPath, filepath.FromSlash(manifest.DecisionPath))
		if _, err := readDecision(decisionPath); err != nil {
			report.Errors = append(report.Errors, DoctorIssue{Code: "run_decision_missing", Path: relPath(projectPath, manifestPath), Message: "run references a missing or invalid human decision"})
		}
	}
}

func validateSensitiveFiles(projectPath string, report *DoctorReport) {
	for _, rootRel := range []string{"test-assets", "evidence/testing"} {
		root := filepath.Join(projectPath, filepath.FromSlash(rootRel))
		_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return nil
			}
			lower := strings.ToLower(entry.Name())
			if lower == ".env" || strings.HasSuffix(lower, ".pem") || strings.HasSuffix(lower, ".key") || strings.Contains(lower, "credentials") {
				report.Errors = append(report.Errors, DoctorIssue{Code: "sensitive_file_forbidden", Path: relPath(projectPath, path), Message: "test assets and evidence must not store credential files"})
				return nil
			}
			if strings.EqualFold(filepath.Ext(path), ".json") {
				var payload any
				if err := readJSON(path, &payload); err == nil {
					for _, key := range sensitiveJSONKeys(payload) {
						report.Errors = append(report.Errors, DoctorIssue{Code: "sensitive_json_value_forbidden", Path: relPath(projectPath, path), Message: "non-empty sensitive field is forbidden: " + key})
					}
				}
			}
			return nil
		})
	}
}

func sensitiveJSONKeys(value any) []string {
	var result []string
	var walk func(any)
	walk = func(current any) {
		switch typed := current.(type) {
		case map[string]any:
			for key, nested := range typed {
				normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(key, "_", ""), "-", ""))
				switch normalized {
				case "password", "token", "accesstoken", "secret", "privatekey", "opensecretkey":
					if strings.TrimSpace(fmt.Sprint(nested)) != "" {
						result = append(result, key)
					}
				}
				walk(nested)
			}
		case []any:
			for _, nested := range typed {
				walk(nested)
			}
		}
	}
	walk(value)
	return sortedUnique(result)
}

func readJSON(path string, target any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, target)
}
