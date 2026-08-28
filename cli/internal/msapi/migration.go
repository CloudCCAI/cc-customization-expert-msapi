package msapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var sourceInvalidCodes = map[string]bool{
	"field_storage_column_missing":                  true,
	"formula_definition_required":                   true,
	"invalid_field_length":                          true,
	"invalid_field_precision":                       true,
	"picklist_values_required":                      true,
	"source_field_api_missing":                      true,
	"source_filtered_rollup_missing_condition_rows": true,
}

func (c *client) migrate(stdout io.Writer, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("cloudcc migrate msapi [projectPath] <compare|classify|physical-preflight|archive-evidence> <@request.json|json>")
	}
	request, err := readObjectArg(args, 1, "cloudcc migrate msapi "+args[0])
	if err != nil {
		return err
	}
	var result map[string]any
	switch strings.ToLower(strings.TrimSpace(args[0])) {
	case "compare", "classify":
		result = classifyMigration(request)
	case "physical-preflight", "preflight":
		result = physicalMigrationPreflight(request)
	case "archive-evidence", "checkpoint", "resume":
		result, err = c.archiveMigrationEvidence(request)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("cloudcc migrate msapi [projectPath] <compare|classify|physical-preflight|archive-evidence> <@request.json|json>")
	}
	body, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, string(body))
	return err
}

func classifyMigration(request map[string]any) map[string]any {
	items := objectList(request["items"])
	classified := make([]map[string]any, 0, len(items))
	counts := map[string]int{}
	retry := make([]map[string]any, 0)
	for _, item := range items {
		classification, reason := migrationClassification(item)
		copy := cloneMap(item)
		copy["classification"] = classification
		copy["reason"] = reason
		classified = append(classified, copy)
		counts[classification]++
		if classification != "source_invalid" && classification != "verified" && classification != "ready" {
			retry = append(retry, migrationRetryItem(item, classification))
		}
	}
	return map[string]any{
		"mode":                 "migration-classification",
		"items":                classified,
		"classificationCounts": counts,
		"retryWorklist":        retry,
		"conservation": map[string]any{
			"input": len(items), "classified": len(classified), "balanced": len(items) == len(classified),
		},
	}
}

func migrationClassification(item map[string]any) (string, string) {
	errorCode := stringValue(item["errorCode"])
	if boolValue(item["sourceInvalid"]) || sourceInvalidCodes[errorCode] ||
		explicitlyFalse(item, "sourceMetadataValid") || explicitlyFalse(item, "sourcePhysicalValid") {
		return "source_invalid", firstNonEmpty(stringValue(item["reason"]), errorCode, "source metadata is incomplete or invalid")
	}
	if boolValue(item["skillGap"]) {
		return "skill_gap", firstNonEmpty(stringValue(item["reason"]), "skill migration mapping is incomplete")
	}
	if boolValue(item["serviceBug"]) {
		return "service_bug", firstNonEmpty(stringValue(item["reason"]), "MetadataService rejected valid source metadata")
	}
	if explicitlyFalse(item, "targetStandardMetadataExists") || explicitlyFalse(item, "targetStandardRuntimeCapable") {
		return "platform_baseline_gap", "target standard-object metadata or runtime capability is missing; provision or upgrade the platform"
	}
	if explicitlyFalse(item, "targetPhysicalValid") {
		return "target_physical_gap", "target physical table or required column is missing"
	}
	if strings.EqualFold(stringValue(item["status"]), "VERIFIED") {
		return "verified", "operation and readback are verified"
	}
	return "ready", "no blocking source, target, skill, or service issue was declared"
}

func physicalMigrationPreflight(request map[string]any) map[string]any {
	objects := objectList(request["objects"])
	results := make([]map[string]any, 0, len(objects))
	counts := map[string]int{}
	for _, object := range objects {
		classification := "ready"
		reason := "physical prerequisites are present"
		origin := strings.ToLower(stringValue(object["origin"]))
		kind := strings.ToLower(stringValue(object["storageKind"]))
		sourceColumns := stringSet(object["sourceColumns"])
		targetColumns := stringSet(object["targetColumns"])
		requiredColumns := stringSet(object["requiredColumns"])
		if requiredColumns == nil {
			requiredColumns = sourceColumns
		}
		missingSource := setDifference(requiredColumns, sourceColumns)
		missingTarget := setDifference(requiredColumns, targetColumns)
		if len(missingSource) > 0 || explicitlyFalse(object, "sourceTableExists") {
			classification, reason = "source_invalid", "source physical table or referenced columns are missing"
		} else if origin == "standard" && (explicitlyFalse(object, "targetMetadataExists") || explicitlyFalse(object, "targetTableExists")) {
			classification, reason = "platform_baseline_gap", "target standard object baseline is not initialized; an empty table is insufficient"
		} else if len(missingTarget) > 0 || explicitlyFalse(object, "targetTableExists") {
			classification = "target_physical_gap"
			if kind == "datatable" {
				reason = "target datatable slot capacity is missing required columns"
			} else {
				reason = "target special table is missing required named columns"
			}
		}
		row := cloneMap(object)
		row["classification"] = classification
		row["reason"] = reason
		row["missingSourceColumns"] = missingSource
		row["missingTargetColumns"] = missingTarget
		results = append(results, row)
		counts[classification]++
	}
	return map[string]any{
		"mode": "migration-physical-preflight", "objects": results, "classificationCounts": counts,
		"conservation": map[string]any{"input": len(objects), "classified": len(results), "balanced": len(objects) == len(results)},
	}
}

func (c *client) archiveMigrationEvidence(request map[string]any) (map[string]any, error) {
	items := objectList(request["items"])
	evidenceDir := c.migrationPath(firstNonEmpty(stringValue(request["evidenceDir"]), "artifacts/migration-evidence"))
	checkpointPath := c.migrationPath(firstNonEmpty(stringValue(request["checkpointFile"]), filepath.Join(evidenceDir, "checkpoint.json")))
	checkpoint := readCheckpoint(checkpointPath)
	completed := stringSet(checkpoint["completedOperationIds"])
	if completed == nil {
		completed = map[string]bool{}
	}
	archived := make([]map[string]any, 0)
	retry := make([]map[string]any, 0)
	for _, item := range items {
		operationID := stringValue(item["operationId"])
		if operationID == "" {
			retry = append(retry, migrationRetryItem(item, "missing_operation_id"))
			continue
		}
		if completed[operationID] {
			archived = append(archived, map[string]any{"operationId": operationID, "status": "checkpoint_skipped"})
			continue
		}
		operation, err := c.requestJSONMap(http.MethodGet, "/metadata/v1/operations/"+operationID, nil)
		if err != nil {
			retry = append(retry, migrationRetryItemWithReason(item, "operation_read_failed", err.Error()))
			continue
		}
		status := strings.ToUpper(stringValue(operation["status"]))
		operationDir := filepath.Join(evidenceDir, safeFilePart(operationID))
		if err := writeMigrationJSON(filepath.Join(operationDir, "operation.json"), operation); err != nil {
			return nil, err
		}
		if status != "VERIFIED" {
			retry = append(retry, migrationRetryItem(item, status))
			continue
		}
		changes, err := c.requestJSONMap(http.MethodGet, "/metadata/v1/operations/"+operationID+"/changes", nil)
		if err != nil {
			retry = append(retry, migrationRetryItemWithReason(item, "changes_read_failed", err.Error()))
			continue
		}
		rollback, err := c.requestJSONMap(http.MethodPost, "/metadata/v1/operations/"+operationID+":rollback-plan", map[string]any{})
		if err != nil {
			retry = append(retry, migrationRetryItemWithReason(item, "rollback_plan_failed", err.Error()))
			continue
		}
		if err := writeMigrationJSON(filepath.Join(operationDir, "changes.json"), changes); err != nil {
			return nil, err
		}
		if err := writeMigrationJSON(filepath.Join(operationDir, "rollback-plan.json"), rollback); err != nil {
			return nil, err
		}
		completed[operationID] = true
		archived = append(archived, map[string]any{"operationId": operationID, "status": "VERIFIED", "evidenceDir": operationDir})
		checkpoint["completedOperationIds"] = sortedSet(completed)
		checkpoint["retryWorklist"] = retry
		if err := writeMigrationJSON(checkpointPath, checkpoint); err != nil {
			return nil, err
		}
	}
	result := map[string]any{
		"mode": "migration-evidence-archive", "archived": archived, "retryWorklist": retry,
		"checkpointFile": checkpointPath,
		"conservation":   map[string]any{"input": len(items), "archivedOrSkipped": len(archived), "retry": len(retry), "balanced": len(items) == len(archived)+len(retry)},
	}
	checkpoint["completedOperationIds"] = sortedSet(completed)
	checkpoint["retryWorklist"] = retry
	checkpoint["conservation"] = result["conservation"]
	if err := writeMigrationJSON(checkpointPath, checkpoint); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *client) migrationPath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(c.projectPath, path)
}

func readCheckpoint(path string) map[string]any {
	body, err := os.ReadFile(path)
	if err != nil {
		return map[string]any{"schemaVersion": "cloudcc-migration-checkpoint/v1"}
	}
	var result map[string]any
	if json.Unmarshal(body, &result) != nil || result == nil {
		return map[string]any{"schemaVersion": "cloudcc-migration-checkpoint/v1"}
	}
	return result
}

func writeMigrationJSON(path string, value any) error {
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".cloudcc-migration-*.json")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if _, err := temp.Write(body); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(tempPath, path)
}

func migrationRetryItem(item map[string]any, classification string) map[string]any {
	return map[string]any{
		"id": item["id"], "planId": item["planId"], "operationId": item["operationId"],
		"classification": classification,
	}
}

func migrationRetryItemWithReason(item map[string]any, classification string, reason string) map[string]any {
	retry := migrationRetryItem(item, classification)
	retry["reason"] = reason
	return retry
}

func objectList(value any) []map[string]any {
	values, _ := value.([]any)
	result := make([]map[string]any, 0, len(values))
	for _, value := range values {
		if item, ok := value.(map[string]any); ok {
			result = append(result, item)
		}
	}
	return result
}

func cloneMap(value map[string]any) map[string]any {
	copy := make(map[string]any, len(value)+2)
	for key, item := range value {
		copy[key] = item
	}
	return copy
}

func explicitlyFalse(value map[string]any, key string) bool {
	raw, exists := value[key]
	return exists && !boolValue(raw)
}

func stringSet(value any) map[string]bool {
	values, ok := value.([]any)
	if !ok {
		return nil
	}
	result := map[string]bool{}
	for _, value := range values {
		if text := strings.TrimSpace(stringValue(value)); text != "" {
			result[strings.ToLower(text)] = true
		}
	}
	return result
}

func setDifference(expected map[string]bool, actual map[string]bool) []string {
	missing := make([]string, 0)
	for value := range expected {
		if !actual[value] {
			missing = append(missing, value)
		}
	}
	sort.Strings(missing)
	return missing
}

func sortedSet(values map[string]bool) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func safeFilePart(value string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
