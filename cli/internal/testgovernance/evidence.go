package testgovernance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type DecisionResult struct {
	Status             string        `json:"status"`
	ProjectPath        string        `json:"projectPath"`
	ChangeSetID        string        `json:"changeSetId"`
	RecommendationPath string        `json:"recommendationPath"`
	DecisionPath       string        `json:"decisionPath"`
	Decision           HumanDecision `json:"decision"`
}

type RunResult struct {
	Status       string      `json:"status"`
	ProjectPath  string      `json:"projectPath"`
	RunPath      string      `json:"runPath"`
	ManifestPath string      `json:"manifestPath"`
	Manifest     RunManifest `json:"manifest"`
}

func Decide(projectPath string, request DecisionRequest) (DecisionResult, error) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return DecisionResult{}, err
	}
	if request.SchemaVersion != DecisionSchemaVersion {
		return DecisionResult{}, fmt.Errorf("decision schemaVersion must be %s", DecisionSchemaVersion)
	}
	if err := validateRecommendation(request.Recommendation); err != nil {
		return DecisionResult{}, err
	}
	selectedScope := normalizeScope(request.SelectedScope)
	if _, ok := scopeRanks[selectedScope]; !ok {
		return DecisionResult{}, fmt.Errorf("selectedScope must be skip, smoke, feature-closure, affected-chain, or full-core")
	}
	if !strings.EqualFold(strings.TrimSpace(request.ConfirmedBy), "human") {
		return DecisionResult{}, fmt.Errorf("confirmedBy must be human")
	}
	if strings.TrimSpace(request.DecidedByRole) == "" {
		return DecisionResult{}, fmt.Errorf("decidedByRole is required")
	}
	if _, err := time.Parse(time.RFC3339, strings.TrimSpace(request.DecidedAt)); err != nil {
		return DecisionResult{}, fmt.Errorf("decidedAt must be RFC3339: %w", err)
	}
	recommendedScope := normalizeScope(request.Recommendation.RecommendedScope)
	if (selectedScope == "skip" || selectedScope != recommendedScope) && strings.TrimSpace(request.Reason) == "" {
		return DecisionResult{}, fmt.Errorf("reason is required when skipping or changing the recommended scope")
	}
	verificationState := "pending_execution"
	if selectedScope == "skip" {
		verificationState = "risk_accepted"
	}
	decision := HumanDecision{
		SchemaVersion:      DecisionSchemaVersion,
		ChangeSetID:        request.Recommendation.ChangeSetID,
		RecommendationHash: request.Recommendation.RecommendationHash,
		RecommendedScope:   recommendedScope,
		SelectedScope:      selectedScope,
		Confirmation:       "human",
		DecidedByRole:      strings.TrimSpace(request.DecidedByRole),
		DecidedAt:          strings.TrimSpace(request.DecidedAt),
		Reason:             strings.TrimSpace(request.Reason),
		VerificationState:  verificationState,
	}
	dir := filepath.Join(absPath, "evidence", "testing", "decisions", decision.ChangeSetID)
	if _, err := os.Stat(dir); err == nil {
		return DecisionResult{}, fmt.Errorf("decision already exists for changeSetId %s; decisions are immutable", decision.ChangeSetID)
	} else if !os.IsNotExist(err) {
		return DecisionResult{}, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return DecisionResult{}, err
	}
	recommendationPath := filepath.Join(dir, "recommendation.json")
	decisionPath := filepath.Join(dir, "human-decision.json")
	if err := writeJSON(recommendationPath, request.Recommendation); err != nil {
		return DecisionResult{}, err
	}
	if err := writeJSON(decisionPath, decision); err != nil {
		return DecisionResult{}, err
	}
	return DecisionResult{
		Status:             "recorded",
		ProjectPath:        absPath,
		ChangeSetID:        decision.ChangeSetID,
		RecommendationPath: relPath(absPath, recommendationPath),
		DecisionPath:       relPath(absPath, decisionPath),
		Decision:           decision,
	}, nil
}

func RecordRun(projectPath string, request RunRequest) (RunResult, error) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return RunResult{}, err
	}
	if request.SchemaVersion != RunSchemaVersion {
		return RunResult{}, fmt.Errorf("run schemaVersion must be %s", RunSchemaVersion)
	}
	if !validID(request.RunID) || !validID(request.ChangeSetID) {
		return RunResult{}, fmt.Errorf("runId and changeSetId must use safe identifier characters")
	}
	for label, value := range map[string]string{
		"sourceRevision": request.SourceRevision,
		"environment":    request.Environment,
		"startedAt":      request.StartedAt,
		"completedAt":    request.CompletedAt,
	} {
		if strings.TrimSpace(value) == "" {
			return RunResult{}, fmt.Errorf("%s is required", label)
		}
	}
	if _, err := time.Parse(time.RFC3339, request.StartedAt); err != nil {
		return RunResult{}, fmt.Errorf("startedAt must be RFC3339: %w", err)
	}
	if _, err := time.Parse(time.RFC3339, request.CompletedAt); err != nil {
		return RunResult{}, fmt.Errorf("completedAt must be RFC3339: %w", err)
	}
	status := strings.ToLower(strings.TrimSpace(request.Status))
	if !supportedRunStatus(status) {
		return RunResult{}, fmt.Errorf("unsupported run status %q", request.Status)
	}
	for _, result := range request.ScenarioResults {
		if !validID(result.ScenarioID) || !supportedScenarioStatus(strings.ToLower(strings.TrimSpace(result.Status))) {
			return RunResult{}, fmt.Errorf("invalid scenario result for %q", result.ScenarioID)
		}
	}

	decisionPath := filepath.Join(absPath, "evidence", "testing", "decisions", request.ChangeSetID, "human-decision.json")
	decision, err := readDecision(decisionPath)
	if err != nil {
		return RunResult{}, fmt.Errorf("run requires a valid human decision: %w", err)
	}
	if decision.SelectedScope == "skip" {
		if status != "skipped" && status != "unverified" {
			return RunResult{}, fmt.Errorf("a skip decision can only record skipped or unverified run status")
		}
		if len(request.ScenarioResults) > 0 {
			return RunResult{}, fmt.Errorf("a skip decision cannot claim executed scenario results")
		}
	} else if len(request.ScenarioResults) == 0 && status != "unverified" {
		return RunResult{}, fmt.Errorf("non-skip run requires scenarioResults unless status is unverified")
	}

	businessStatus := strings.ToLower(strings.TrimSpace(request.BusinessAcceptanceStatus))
	if businessStatus == "" {
		businessStatus = "pending"
	}
	switch businessStatus {
	case "pending", "not_performed", "accepted", "rejected":
	default:
		return RunResult{}, fmt.Errorf("unsupported businessAcceptanceStatus %q", request.BusinessAcceptanceStatus)
	}
	if (businessStatus == "accepted" || businessStatus == "rejected") && strings.TrimSpace(request.BusinessAcceptanceEvidence) == "" {
		return RunResult{}, fmt.Errorf("businessAcceptanceEvidence is required for accepted or rejected business acceptance")
	}
	request.Status = status
	request.BusinessAcceptanceStatus = businessStatus
	verificationState := verificationStateForRun(status, businessStatus)
	manifest := RunManifest{
		RunRequest:        request,
		DecisionPath:      relPath(absPath, decisionPath),
		SelectedScope:     decision.SelectedScope,
		VerificationState: verificationState,
	}
	runDir := filepath.Join(absPath, "evidence", "testing", "runs", request.RunID)
	if _, err := os.Stat(runDir); err == nil {
		return RunResult{}, fmt.Errorf("runId %s already exists; run evidence is immutable", request.RunID)
	} else if !os.IsNotExist(err) {
		return RunResult{}, err
	}
	for _, dir := range []string{"readbacks", "screenshots", "logs"} {
		if err := os.MkdirAll(filepath.Join(runDir, dir), 0o755); err != nil {
			return RunResult{}, err
		}
	}
	manifestPath := filepath.Join(runDir, "manifest.json")
	if err := writeJSON(manifestPath, manifest); err != nil {
		return RunResult{}, err
	}
	summary := fmt.Sprintf("# Test Run %s\n\n- Change set: `%s`\n- Selected scope: `%s`\n- Run status: `%s`\n- Verification state: `%s`\n- Business acceptance: `%s`\n\nThis file is an index. Put structured results in manifest.json and external runner evidence in the dedicated subdirectories.\n", request.RunID, request.ChangeSetID, decision.SelectedScope, status, verificationState, businessStatus)
	if err := os.WriteFile(filepath.Join(runDir, "summary.md"), []byte(summary), 0o644); err != nil {
		return RunResult{}, err
	}
	if err := os.WriteFile(filepath.Join(runDir, "defect-links.md"), []byte("# Defect Links\n\n- None recorded.\n"), 0o644); err != nil {
		return RunResult{}, err
	}
	return RunResult{
		Status:       "recorded",
		ProjectPath:  absPath,
		RunPath:      relPath(absPath, runDir),
		ManifestPath: relPath(absPath, manifestPath),
		Manifest:     manifest,
	}, nil
}

func supportedRunStatus(status string) bool {
	switch status {
	case "passed", "failed", "partial", "skipped", "unverified":
		return true
	default:
		return false
	}
}

func supportedScenarioStatus(status string) bool {
	switch status {
	case "passed", "failed", "skipped", "blocked", "not_run":
		return true
	default:
		return false
	}
}

func verificationStateForRun(status string, businessStatus string) string {
	switch status {
	case "passed":
		if businessStatus == "accepted" {
			return "verified"
		}
		return "pending_uat"
	case "partial":
		return "partially_verified"
	case "skipped":
		return "risk_accepted"
	case "failed":
		return "failed"
	default:
		return "unverified"
	}
}

func readDecision(path string) (HumanDecision, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return HumanDecision{}, err
	}
	var decision HumanDecision
	if err := json.Unmarshal(b, &decision); err != nil {
		return decision, err
	}
	if decision.SchemaVersion != DecisionSchemaVersion || !validID(decision.ChangeSetID) || decision.Confirmation != "human" {
		return decision, fmt.Errorf("invalid human decision contract")
	}
	if _, ok := scopeRanks[decision.SelectedScope]; !ok {
		return decision, fmt.Errorf("invalid selectedScope")
	}
	return decision, nil
}

func writeJSON(path string, value any) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o644)
}

func relPath(root string, path string) string {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(relative)
}
