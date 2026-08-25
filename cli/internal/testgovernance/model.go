package testgovernance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const (
	ChangeSchemaVersion         = "cloudcc-test-change/v1"
	RecommendationSchemaVersion = "cloudcc-test-recommendation/v1"
	ImpactMapSchemaVersion      = "cloudcc-test-impact-map/v1"
	ScenarioIndexSchemaVersion  = "cloudcc-test-scenario-index/v1"
	DecisionSchemaVersion       = "cloudcc-test-decision/v1"
	RunSchemaVersion            = "cloudcc-test-run/v1"
)

var safeIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

var scopeRanks = map[string]int{
	"skip":            0,
	"smoke":           1,
	"feature-closure": 2,
	"affected-chain":  3,
	"full-core":       4,
}

type ResourceChange struct {
	Kind     string   `json:"kind"`
	Name     string   `json:"name,omitempty"`
	Module   string   `json:"module,omitempty"`
	Events   []string `json:"events,omitempty"`
	Shared   bool     `json:"shared,omitempty"`
	RiskTags []string `json:"riskTags,omitempty"`
}

type ChangeRequest struct {
	SchemaVersion string           `json:"schemaVersion"`
	ChangeSetID   string           `json:"changeSetId"`
	Phase         string           `json:"phase"`
	Description   string           `json:"description,omitempty"`
	Resources     []ResourceChange `json:"resources"`
}

type ImpactEntry struct {
	Kind             string   `json:"kind"`
	Name             string   `json:"name,omitempty"`
	Module           string   `json:"module,omitempty"`
	RecommendedScope string   `json:"recommendedScope,omitempty"`
	AffectedModules  []string `json:"affectedModules,omitempty"`
	ScenarioIDs      []string `json:"scenarioIds,omitempty"`
	SuiteIDs         []string `json:"suiteIds,omitempty"`
	RiskTags         []string `json:"riskTags,omitempty"`
	Reason           string   `json:"reason,omitempty"`
}

type ImpactMap struct {
	SchemaVersion string        `json:"schemaVersion"`
	Resources     []ImpactEntry `json:"resources"`
}

type ScenarioEntry struct {
	ScenarioID       string   `json:"scenarioId"`
	Module           string   `json:"module,omitempty"`
	Title            string   `json:"title,omitempty"`
	Path             string   `json:"path,omitempty"`
	RiskTags         []string `json:"riskTags,omitempty"`
	EstimatedMinutes int      `json:"estimatedMinutes,omitempty"`
}

type ScenarioIndex struct {
	SchemaVersion string          `json:"schemaVersion"`
	Scenarios     []ScenarioEntry `json:"scenarios"`
}

type Recommendation struct {
	SchemaVersion         string           `json:"schemaVersion"`
	Advisory              bool             `json:"advisory"`
	Blocking              bool             `json:"blocking"`
	ChangeSetID           string           `json:"changeSetId"`
	Phase                 string           `json:"phase"`
	Resources             []ResourceChange `json:"resources"`
	RecommendedScope      string           `json:"recommendedScope"`
	RiskLevel             string           `json:"riskLevel"`
	RiskTags              []string         `json:"riskTags"`
	AffectedModules       []string         `json:"affectedModules"`
	ScenarioIDs           []string         `json:"scenarioIds"`
	SuiteIDs              []string         `json:"suiteIds"`
	EstimatedMinutes      int              `json:"estimatedMinutes"`
	CatalogCoverage       string           `json:"catalogCoverage"`
	Reasons               []string         `json:"reasons"`
	HumanDecisionRequired bool             `json:"humanDecisionRequired"`
	RecommendationHash    string           `json:"recommendationHash"`
}

type DecisionRequest struct {
	SchemaVersion  string         `json:"schemaVersion"`
	Recommendation Recommendation `json:"recommendation"`
	SelectedScope  string         `json:"selectedScope"`
	ConfirmedBy    string         `json:"confirmedBy"`
	DecidedByRole  string         `json:"decidedByRole"`
	DecidedAt      string         `json:"decidedAt"`
	Reason         string         `json:"reason,omitempty"`
}

type HumanDecision struct {
	SchemaVersion      string `json:"schemaVersion"`
	ChangeSetID        string `json:"changeSetId"`
	RecommendationHash string `json:"recommendationHash"`
	RecommendedScope   string `json:"recommendedScope"`
	SelectedScope      string `json:"selectedScope"`
	Confirmation       string `json:"confirmation"`
	DecidedByRole      string `json:"decidedByRole"`
	DecidedAt          string `json:"decidedAt"`
	Reason             string `json:"reason,omitempty"`
	VerificationState  string `json:"verificationState"`
}

type ScenarioResult struct {
	ScenarioID string `json:"scenarioId"`
	Status     string `json:"status"`
	Evidence   string `json:"evidence,omitempty"`
}

type RunRequest struct {
	SchemaVersion              string           `json:"schemaVersion"`
	RunID                      string           `json:"runId"`
	ChangeSetID                string           `json:"changeSetId"`
	SourceRevision             string           `json:"sourceRevision"`
	Environment                string           `json:"environment"`
	EnvironmentFingerprint     string           `json:"environmentFingerprint,omitempty"`
	TestAssetRevision          string           `json:"testAssetRevision,omitempty"`
	Status                     string           `json:"status"`
	StartedAt                  string           `json:"startedAt"`
	CompletedAt                string           `json:"completedAt"`
	ScenarioResults            []ScenarioResult `json:"scenarioResults"`
	BusinessAcceptanceStatus   string           `json:"businessAcceptanceStatus,omitempty"`
	BusinessAcceptanceEvidence string           `json:"businessAcceptanceEvidence,omitempty"`
}

type RunManifest struct {
	RunRequest
	DecisionPath      string `json:"decisionPath"`
	SelectedScope     string `json:"selectedScope"`
	VerificationState string `json:"verificationState"`
}

func validID(value string) bool {
	return safeIDPattern.MatchString(strings.TrimSpace(value))
}

func normalizeScope(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func maxScope(current string, candidate string) string {
	current = normalizeScope(current)
	candidate = normalizeScope(candidate)
	if scopeRanks[candidate] > scopeRanks[current] {
		return candidate
	}
	return current
}

func sortedUnique(values []string) []string {
	seen := map[string]bool{}
	var result []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func recommendationHash(recommendation Recommendation) (string, error) {
	recommendation.RecommendationHash = ""
	b, err := json.Marshal(recommendation)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func validateRecommendation(recommendation Recommendation) error {
	if recommendation.SchemaVersion != RecommendationSchemaVersion {
		return fmt.Errorf("recommendation schemaVersion must be %s", RecommendationSchemaVersion)
	}
	if !recommendation.Advisory || recommendation.Blocking {
		return fmt.Errorf("recommendation must be advisory and non-blocking")
	}
	if !validID(recommendation.ChangeSetID) {
		return fmt.Errorf("recommendation changeSetId is invalid")
	}
	if _, ok := scopeRanks[normalizeScope(recommendation.RecommendedScope)]; !ok || normalizeScope(recommendation.RecommendedScope) == "skip" {
		return fmt.Errorf("recommendation recommendedScope is invalid")
	}
	expected, err := recommendationHash(recommendation)
	if err != nil {
		return err
	}
	if strings.TrimSpace(recommendation.RecommendationHash) == "" || recommendation.RecommendationHash != expected {
		return fmt.Errorf("recommendationHash mismatch")
	}
	return nil
}
