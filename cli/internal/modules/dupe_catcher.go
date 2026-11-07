package modules

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"cloudcc-customization-expert-go/internal/config"
	"cloudcc-customization-expert-go/internal/jsonx"
)

// handleDupeCatcher maps the aggregate and its owned rules to setup-web's
// authoritative Lightning endpoints. Horizontal UIAPI is rejected upstream.
func handleDupeCatcher(action string, args []string, stdout io.Writer, cwd string) error {
	ep, ok := genericEndpoints["dupeCatcher"][action]
	if !ok {
		return fmt.Errorf("unsupported dupeCatcher action: %s", action)
	}
	projectPath := firstArg(args, cwd)
	rest := argsAfterProject(args)
	cfg, err := config.Load(projectPath)
	if err != nil {
		return err
	}
	switch action {
	case "capabilities":
		if len(rest) != 0 {
			return fmt.Errorf("cloudcc capabilities dupeCatcher <projectPath>")
		}
		return json.NewEncoder(stdout).Encode(map[string]any{
			"domain": "dupe-catchers", "contractVersion": 2, "provider": "lightning-uiapi",
			"ruleComposition": "and", "matchOptions": []string{"exact", "contains", "firstNletters", "lastNletters"},
			"operations": []string{"get", "detail", "candidates", "create", "update", "delete", "detail-rule", "create-rule", "update-rule", "delete-rule"},
		})
	case "candidates":
		return dupeCatcherCandidates(projectPath, cfg, rest, stdout)
	case "detail-rule", "detailRule":
		if len(rest) != 2 || strings.TrimSpace(rest[0]) == "" || strings.TrimSpace(rest[1]) == "" {
			return fmt.Errorf("cloudcc detail-rule dupeCatcher <projectPath> <filterId> <ruleId>")
		}
		filterID, ruleID := strings.TrimSpace(rest[0]), strings.TrimSpace(rest[1])
		ownedRule, err := dupeCatcherOwnedRule(projectPath, cfg, filterID, ruleID)
		if err != nil {
			return err
		}
		var detail map[string]any
		if err := postClassResponse(projectPath, cfg, ep.base, ep.path, map[string]any{"ruleid": ruleID}, &detail); err != nil {
			return err
		}
		rule, ok := findDupeCatcherRule(detail, ruleID)
		if !ok {
			rule = ownedRule
		} else if parent := dupeCatcherText(rule, "filterid", "filterId"); parent != "" && parent != filterID {
			return fmt.Errorf("dupeCatcher rule %s belongs to filter %s, not %s", ruleID, parent, filterID)
		}
		return printJSON(stdout, map[string]any{"filterId": filterID, "rule": rule})
	case "create-rule", "createRule":
		if len(rest) != 2 || strings.TrimSpace(rest[0]) == "" {
			return fmt.Errorf("cloudcc create-rule dupeCatcher <projectPath> <filterId> <@rule.json>")
		}
		body, err := jsonx.ParseEncodedObject(rest[1], "create-rule dupeCatcher")
		if err != nil {
			return err
		}
		body, err = prepareDupeCatcherRule(body)
		if err != nil {
			return err
		}
		body["filterid"] = strings.TrimSpace(rest[0])
		if err := postClassResponse(projectPath, cfg, ep.base, ep.path, body, &map[string]any{}); err != nil {
			return err
		}
		return postClass(stdout, projectPath, cfg, "setup", "/api/duplication/detailfilter", map[string]any{"filterid": strings.TrimSpace(rest[0])})
	case "update-rule", "updateRule":
		if len(rest) != 3 || strings.TrimSpace(rest[0]) == "" || strings.TrimSpace(rest[1]) == "" {
			return fmt.Errorf("cloudcc update-rule dupeCatcher <projectPath> <filterId> <ruleId> <@patch.json>")
		}
		filterID, ruleID := strings.TrimSpace(rest[0]), strings.TrimSpace(rest[1])
		existing, err := dupeCatcherOwnedRule(projectPath, cfg, filterID, ruleID)
		if err != nil {
			return err
		}
		patch, err := jsonx.ParseEncodedObject(rest[2], "update-rule dupeCatcher")
		if err != nil {
			return err
		}
		body := normalizeDupeCatcherRule(existing)
		for key, value := range normalizeDupeCatcherRule(patch) {
			body[key] = value
		}
		body, err = prepareDupeCatcherRule(body)
		if err != nil {
			return err
		}
		body["filterid"], body["id"], body["ruleid"] = filterID, ruleID, ruleID
		if err := postClassResponse(projectPath, cfg, ep.base, ep.path, body, &map[string]any{}); err != nil {
			return err
		}
		readback, err := dupeCatcherOwnedRule(projectPath, cfg, filterID, ruleID)
		if err != nil {
			return fmt.Errorf("dupeCatcher rule update readback failed: %w", err)
		}
		if err := verifyDupeCatcherRule(body, readback); err != nil {
			return err
		}
		return printJSON(stdout, map[string]any{"filterId": filterID, "rule": readback})
	case "delete-rule", "deleteRule":
		if len(rest) != 2 || strings.TrimSpace(rest[0]) == "" || strings.TrimSpace(rest[1]) == "" {
			return fmt.Errorf("cloudcc delete-rule dupeCatcher <projectPath> <filterId> <ruleId>")
		}
		filterID, ruleID := strings.TrimSpace(rest[0]), strings.TrimSpace(rest[1])
		if _, err := dupeCatcherOwnedRule(projectPath, cfg, filterID, ruleID); err != nil {
			return err
		}
		if err := postClassResponse(projectPath, cfg, ep.base, ep.path, map[string]any{"ruleid": ruleID}, &map[string]any{}); err != nil {
			return err
		}
		var readback map[string]any
		if err := postClassResponse(projectPath, cfg, "setup", "/api/duplication/detailfilter",
			map[string]any{"filterid": filterID}, &readback); err != nil {
			return err
		}
		if _, ok := findDupeCatcherRule(readback, ruleID); ok {
			return fmt.Errorf("dupeCatcher rule %s still exists after delete", ruleID)
		}
		return printJSON(stdout, readback)
	default:
		return callGeneric(ep, action, "dupeCatcher", args, stdout, cwd)
	}
}

func normalizeDupeCatcherRule(source map[string]any) map[string]any {
	result := map[string]any{}
	for key, value := range source {
		result[key] = value
	}
	for target, aliases := range map[string][]string{
		"fieldid":      {"fieldId", "field"},
		"matchoption":  {"matchOption"},
		"firstletters": {"matchLength", "firstLetters"},
		"isblank":      {"blankMatches", "blank"},
	} {
		if _, exists := result[target]; !exists {
			for _, alias := range aliases {
				if value, ok := result[alias]; ok {
					result[target] = value
					break
				}
			}
		}
		for _, alias := range aliases {
			delete(result, alias)
		}
	}
	return result
}

func prepareDupeCatcherRule(source map[string]any) (map[string]any, error) {
	result := normalizeDupeCatcherRule(source)
	if dupeCatcherText(result, "fieldid") == "" {
		return nil, fmt.Errorf("dupeCatcher rule requires fieldId")
	}
	match := dupeCatcherText(result, "matchoption")
	switch match {
	case "exact", "contains":
		// setup-service persists firstletters independently; explicitly clear a
		// previous N-length when the matching mode changes.
		result["firstletters"] = nil
	case "firstNletters", "lastNletters":
		length, err := strconv.Atoi(dupeCatcherText(result, "firstletters"))
		if err != nil || length <= 0 {
			return nil, fmt.Errorf("dupeCatcher %s requires a positive matchLength", match)
		}
	default:
		return nil, fmt.Errorf("dupeCatcher matchOption must be exact, contains, firstNletters, or lastNletters")
	}
	if _, exists := result["isblank"]; !exists {
		result["isblank"] = false
	}
	return result, nil
}

func dupeCatcherOwnedRule(projectPath string, cfg config.Config, filterID string, ruleID string) (map[string]any, error) {
	var detail map[string]any
	if err := postClassResponse(projectPath, cfg, "setup", "/api/duplication/detailfilter",
		map[string]any{"filterid": filterID}, &detail); err != nil {
		return nil, err
	}
	rule, ok := findDupeCatcherRule(detail, ruleID)
	if !ok {
		return nil, fmt.Errorf("dupeCatcher rule %s does not belong to filter %s", ruleID, filterID)
	}
	if parent := dupeCatcherText(rule, "filterid", "filterId"); parent != "" && parent != filterID {
		return nil, fmt.Errorf("dupeCatcher rule %s belongs to filter %s, not %s", ruleID, parent, filterID)
	}
	return normalizeDupeCatcherRule(rule), nil
}

func findDupeCatcherRule(value any, ruleID string) (map[string]any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		id := dupeCatcherText(typed, "id", "ruleid", "ruleId")
		if id == ruleID && (dupeCatcherHas(typed, "fieldid", "fieldId") ||
			dupeCatcherHas(typed, "matchoption", "matchOption") || dupeCatcherHas(typed, "filterid", "filterId")) {
			return typed, true
		}
		for _, nested := range typed {
			if rule, ok := findDupeCatcherRule(nested, ruleID); ok {
				return rule, true
			}
		}
	case []any:
		for _, nested := range typed {
			if rule, ok := findDupeCatcherRule(nested, ruleID); ok {
				return rule, true
			}
		}
	}
	return nil, false
}

func verifyDupeCatcherRule(expected map[string]any, actual map[string]any) error {
	expected = normalizeDupeCatcherRule(expected)
	actual = normalizeDupeCatcherRule(actual)
	for _, key := range []string{"fieldid", "matchoption", "firstletters", "isblank"} {
		value, present := expected[key]
		if !present {
			continue
		}
		if strings.TrimSpace(fmt.Sprint(value)) != strings.TrimSpace(fmt.Sprint(actual[key])) {
			return fmt.Errorf("dupeCatcher rule readback mismatch for %s", key)
		}
	}
	return nil
}

func dupeCatcherText(values map[string]any, names ...string) string {
	for key, value := range values {
		for _, name := range names {
			if strings.EqualFold(key, name) {
				return strings.TrimSpace(fmt.Sprint(value))
			}
		}
	}
	return ""
}

func dupeCatcherHas(values map[string]any, names ...string) bool {
	for key := range values {
		for _, name := range names {
			if strings.EqualFold(key, name) {
				return true
			}
		}
	}
	return false
}

func dupeCatcherCandidates(projectPath string, cfg config.Config, args []string, stdout io.Writer) error {
	if len(args) < 1 || len(args) > 3 {
		return fmt.Errorf("cloudcc candidates dupeCatcher <projectPath> <objects|rule-fields|condition-fields> [filterId|object] [keyword]")
	}
	switch strings.TrimSpace(args[0]) {
	case "objects":
		if len(args) > 2 {
			return fmt.Errorf("objects candidates accept at most one keyword")
		}
		return postClass(stdout, projectPath, cfg, "setup", "/api/duplication/insetNewfilter", map[string]any{})
	case "rule-fields":
		if len(args) < 2 || strings.TrimSpace(args[1]) == "" {
			return fmt.Errorf("rule-fields candidates require filterId")
		}
		return postClass(stdout, projectPath, cfg, "setup", "/api/duplication/insetNewrule", map[string]any{"filterid": strings.TrimSpace(args[1])})
	case "condition-fields":
		if len(args) < 2 || strings.TrimSpace(args[1]) == "" {
			return fmt.Errorf("condition-fields candidates require object selector")
		}
		return postClass(stdout, projectPath, cfg, "setup", "/api/workFlowSetup/newWorkflow", map[string]any{"targetobjectid": strings.TrimSpace(args[1])})
	default:
		return fmt.Errorf("dupeCatcher candidate kind must be objects, rule-fields, or condition-fields")
	}
}
