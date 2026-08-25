package testgovernance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Advise(projectPath string, change ChangeRequest) (Recommendation, error) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return Recommendation{}, err
	}
	if change.SchemaVersion != ChangeSchemaVersion {
		return Recommendation{}, fmt.Errorf("change schemaVersion must be %s", ChangeSchemaVersion)
	}
	if !validID(change.ChangeSetID) {
		return Recommendation{}, fmt.Errorf("changeSetId must use letters, digits, dot, underscore, or hyphen")
	}
	change.Phase = strings.ToLower(strings.TrimSpace(change.Phase))
	if change.Phase == "" {
		return Recommendation{}, fmt.Errorf("phase is required")
	}
	if len(change.Resources) == 0 {
		return Recommendation{}, fmt.Errorf("resources must contain at least one change")
	}

	impactMap, impactLoaded, err := loadImpactMap(filepath.Join(absPath, "test-assets", "catalog", "impact-map.json"))
	if err != nil {
		return Recommendation{}, err
	}
	scenarioIndex, scenarioLoaded, err := loadScenarioIndex(filepath.Join(absPath, "test-assets", "catalog", "scenario-index.json"))
	if err != nil {
		return Recommendation{}, err
	}

	recommendation := Recommendation{
		SchemaVersion:         RecommendationSchemaVersion,
		Advisory:              true,
		Blocking:              false,
		ChangeSetID:           change.ChangeSetID,
		Phase:                 change.Phase,
		Resources:             change.Resources,
		RecommendedScope:      phaseFloor(change.Phase),
		HumanDecisionRequired: true,
	}
	matchedEntries := 0
	for _, resource := range change.Resources {
		resource.Kind = normalizeKind(resource.Kind)
		if resource.Kind == "" {
			return Recommendation{}, fmt.Errorf("every resource requires kind")
		}
		resourceScope, resourceReasons, resourceRisks := defaultResourceRecommendation(resource)
		recommendation.RecommendedScope = maxScope(recommendation.RecommendedScope, resourceScope)
		recommendation.Reasons = append(recommendation.Reasons, resourceReasons...)
		recommendation.RiskTags = append(recommendation.RiskTags, resourceRisks...)
		recommendation.RiskTags = append(recommendation.RiskTags, resource.RiskTags...)
		if strings.TrimSpace(resource.Module) != "" {
			recommendation.AffectedModules = append(recommendation.AffectedModules, resource.Module)
		}
		for _, entry := range impactMap.Resources {
			if !impactEntryMatches(entry, resource) {
				continue
			}
			matchedEntries++
			recommendation.RecommendedScope = maxScope(recommendation.RecommendedScope, entry.RecommendedScope)
			recommendation.AffectedModules = append(recommendation.AffectedModules, entry.AffectedModules...)
			recommendation.ScenarioIDs = append(recommendation.ScenarioIDs, entry.ScenarioIDs...)
			recommendation.SuiteIDs = append(recommendation.SuiteIDs, entry.SuiteIDs...)
			recommendation.RiskTags = append(recommendation.RiskTags, entry.RiskTags...)
			if strings.TrimSpace(entry.Reason) != "" {
				recommendation.Reasons = append(recommendation.Reasons, entry.Reason)
			}
		}
	}

	recommendation.AffectedModules = sortedUnique(recommendation.AffectedModules)
	recommendation.ScenarioIDs = sortedUnique(recommendation.ScenarioIDs)
	recommendation.SuiteIDs = sortedUnique(recommendation.SuiteIDs)
	recommendation.RiskTags = sortedUnique(recommendation.RiskTags)
	recommendation.Reasons = sortedUnique(recommendation.Reasons)
	if !impactLoaded || !scenarioLoaded || matchedEntries == 0 || len(recommendation.ScenarioIDs) == 0 {
		recommendation.CatalogCoverage = "missing"
		recommendation.RiskTags = sortedUnique(append(recommendation.RiskTags, "catalog-gap"))
		recommendation.Reasons = sortedUnique(append(recommendation.Reasons, "Project impact or scenario catalog is incomplete; a human must review the suggested scope and add missing assets."))
	} else {
		recommendation.CatalogCoverage = "matched"
	}

	scenarioByID := map[string]ScenarioEntry{}
	for _, scenario := range scenarioIndex.Scenarios {
		scenarioByID[scenario.ScenarioID] = scenario
	}
	for _, id := range recommendation.ScenarioIDs {
		if scenario, ok := scenarioByID[id]; ok {
			recommendation.EstimatedMinutes += scenario.EstimatedMinutes
			recommendation.RiskTags = append(recommendation.RiskTags, scenario.RiskTags...)
		}
	}
	recommendation.RiskTags = sortedUnique(recommendation.RiskTags)
	recommendation.RiskLevel = riskLevel(recommendation.RecommendedScope, recommendation.RiskTags)
	recommendation.Reasons = append(recommendation.Reasons, "Recommendation is advisory and non-blocking; the human decision is the authoritative test scope.")
	recommendation.Reasons = sortedUnique(recommendation.Reasons)
	hash, err := recommendationHash(recommendation)
	if err != nil {
		return Recommendation{}, err
	}
	recommendation.RecommendationHash = hash
	return recommendation, nil
}

func loadImpactMap(path string) (ImpactMap, bool, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return ImpactMap{SchemaVersion: ImpactMapSchemaVersion}, false, nil
	}
	if err != nil {
		return ImpactMap{}, false, err
	}
	var result ImpactMap
	if err := json.Unmarshal(b, &result); err != nil {
		return result, false, fmt.Errorf("invalid impact map: %w", err)
	}
	if result.SchemaVersion != ImpactMapSchemaVersion {
		return result, false, fmt.Errorf("impact map schemaVersion must be %s", ImpactMapSchemaVersion)
	}
	return result, true, nil
}

func loadScenarioIndex(path string) (ScenarioIndex, bool, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return ScenarioIndex{SchemaVersion: ScenarioIndexSchemaVersion}, false, nil
	}
	if err != nil {
		return ScenarioIndex{}, false, err
	}
	var result ScenarioIndex
	if err := json.Unmarshal(b, &result); err != nil {
		return result, false, fmt.Errorf("invalid scenario index: %w", err)
	}
	if result.SchemaVersion != ScenarioIndexSchemaVersion {
		return result, false, fmt.Errorf("scenario index schemaVersion must be %s", ScenarioIndexSchemaVersion)
	}
	return result, true, nil
}

func phaseFloor(phase string) string {
	switch strings.ToLower(strings.TrimSpace(phase)) {
	case "prototype", "rapid-development", "rapid_development", "development":
		return "smoke"
	case "feature-development", "feature_development", "feature":
		return "feature-closure"
	case "stabilization", "sit":
		return "affected-chain"
	case "uat", "pre-production", "pre_production", "release-candidate", "release_candidate", "production":
		return "full-core"
	default:
		return "feature-closure"
	}
}

func defaultResourceRecommendation(resource ResourceChange) (string, []string, []string) {
	kind := normalizeKind(resource.Kind)
	scope := "feature-closure"
	reasons := []string{fmt.Sprintf("Changed resource kind %s requires at least feature-level closure review.", kind)}
	var risks []string
	switch kind {
	case "label", "translation", "layout", "pagelayout", "view":
		scope = "smoke"
		reasons = []string{"Presentation-oriented change normally starts with focused smoke and role-visible readback."}
	case "field", "object", "record-type", "recordtype", "report", "dashboard", "pagecomponent", "custompage", "script":
		scope = "feature-closure"
		risks = append(risks, "data-or-experience")
	case "trigger", "class", "timer", "workflow", "approval", "validation-rule", "dupe-catcher", "integration", "openapi":
		scope = "affected-chain"
		risks = append(risks, "automation-or-integration")
		reasons = []string{"Business logic or automation change can affect downstream records and requires affected-chain review."}
	case "permission", "profile", "role", "sharing-rule", "field-security", "global-select-list", "data-migration", "platform-upgrade":
		scope = "full-core"
		risks = append(risks, "shared-or-governance")
		reasons = []string{"Shared security, data model, migration, or platform change can cross module boundaries and suggests full core regression."}
	}
	if resource.Shared {
		scope = "full-core"
		risks = append(risks, "shared-resource")
		reasons = append(reasons, "Resource is marked shared, so the recommendation expands to full core review.")
	}
	return scope, reasons, risks
}

func normalizeKind(kind string) string {
	kind = strings.ToLower(strings.TrimSpace(kind))
	kind = strings.ReplaceAll(kind, "_", "-")
	return kind
}

func impactEntryMatches(entry ImpactEntry, resource ResourceChange) bool {
	if normalizeKind(entry.Kind) != normalizeKind(resource.Kind) {
		return false
	}
	if strings.TrimSpace(entry.Name) != "" && !strings.EqualFold(strings.TrimSpace(entry.Name), strings.TrimSpace(resource.Name)) {
		return false
	}
	if strings.TrimSpace(entry.Module) != "" && !strings.EqualFold(strings.TrimSpace(entry.Module), strings.TrimSpace(resource.Module)) {
		return false
	}
	return true
}

func riskLevel(scope string, tags []string) string {
	rank := scopeRanks[normalizeScope(scope)]
	if rank >= 4 || len(tags) >= 4 {
		return "high"
	}
	if rank >= 3 || len(tags) >= 2 {
		return "medium"
	}
	return "low"
}
