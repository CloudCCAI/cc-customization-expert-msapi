package modules

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"cloudcc-customization-expert-go/internal/config"
	"cloudcc-customization-expert-go/internal/jsonx"
)

func handleCcchat(action string, resource string, args []string, stdout io.Writer, cwd string) error {
	projectPath := firstArg(args, cwd)
	cfg, err := config.Load(projectPath)
	if err != nil {
		return err
	}
	rest := argsAfterProject(args)
	subresource := ""
	switch resource {
	case "ccchatSetting", "ccchat-settings":
		subresource = "settings"
	case "ccchatFeedTracking", "feedTracking", "ccchat-feed-tracking":
		subresource = "feedTracking"
	default:
		if len(rest) == 0 {
			return fmt.Errorf("cloudcc %s ccchat <projectPath> <settings|feedTracking> [object|@spec.json]", action)
		}
		subresource, rest = strings.TrimSpace(rest[0]), rest[1:]
	}
	if subresource == "settings" {
		return handleCcchatSettings(action, projectPath, cfg, rest, stdout)
	}
	if subresource == "feedTracking" {
		return handleCcchatFeedTracking(action, projectPath, cfg, rest, stdout)
	}
	return fmt.Errorf("ccchat_resource_invalid: expected settings or feedTracking, got %q", subresource)
}

func handleCcchatSettings(action string, projectPath string, cfg config.Config, args []string, stdout io.Writer) error {
	if action == "get" && len(args) == 0 {
		var response map[string]any
		if err := postCcchatResponse(projectPath, cfg, "/api/chatterGroup/queryChatterSetup", map[string]any{}, &response); err != nil {
			return err
		}
		if err := validateCcchatSettingsResponse(response); err != nil {
			return err
		}
		return printJSON(stdout, normalizeCcchatSettings(response))
	}
	if action != "update" || len(args) != 1 {
		return fmt.Errorf("cloudcc %s ccchat <projectPath> settings%s", action, map[bool]string{true: " <encodedSpecJson|@file>", false: ""}[action == "update"])
	}
	spec, err := jsonx.ParseEncodedObject(args[0], "update ccchat settings")
	if err != nil {
		return err
	}
	payload, err := ccchatSettingsPayload(spec)
	if err != nil {
		return err
	}
	if payload["allowInvite"] == false {
		var current map[string]any
		if err := postCcchatResponse(projectPath, cfg, "/api/chatterGroup/queryChatterSetup", map[string]any{}, &current); err != nil {
			return fmt.Errorf("CCChat settings could not preserve existing email domains: %w", err)
		}
		if err := validateCcchatSettingsResponse(current); err != nil {
			return err
		}
		payload["emailDomains"] = strings.Join(normalizeCcchatSettings(current)["emailDomains"].([]string), ";")
	}
	expected := normalizeCcchatSettings(map[string]any{"data": payload})
	var ignored map[string]any
	if err := postCcchatResponse(projectPath, cfg, "/api/chatterGroup/saveSetup", payload, &ignored); err != nil {
		return err
	}
	var readback map[string]any
	if err := postCcchatResponse(projectPath, cfg, "/api/chatterGroup/queryChatterSetup", map[string]any{}, &readback); err != nil {
		return fmt.Errorf("CCChat settings were submitted but readback failed: %w", err)
	}
	if err := validateCcchatSettingsResponse(readback); err != nil {
		return err
	}
	normalized := normalizeCcchatSettings(readback)
	if !reflect.DeepEqual(normalized, expected) {
		return fmt.Errorf("ccchat_readback_mismatch: saved settings do not match setup-service readback")
	}
	return printJSON(stdout, normalized)
}

func handleCcchatFeedTracking(action string, projectPath string, cfg config.Config, args []string, stdout io.Writer) error {
	if action == "getList" && len(args) == 0 {
		var response map[string]any
		if err := postCcchatResponse(projectPath, cfg, "/api/feedTrack/queryObjectList", map[string]any{}, &response); err != nil {
			return err
		}
		if err := validateCcchatObjectListResponse(response); err != nil {
			return err
		}
		return printJSON(stdout, map[string]any{"domain": "ccchat", "resource": "feedTracking", "items": ccchatRows(response)})
	}
	if (action == "get" || action == "candidates") && len(args) == 1 && strings.TrimSpace(args[0]) != "" {
		state, err := queryCcchatFeedTracking(projectPath, cfg, strings.TrimSpace(args[0]))
		if err != nil {
			return err
		}
		if action == "candidates" {
			return printJSON(stdout, map[string]any{"domain": "ccchat", "resource": "feedTrackingCandidates", "objectId": state["objectId"], "fields": state["candidateFields"], "limit": 20})
		}
		delete(state, "candidateFields")
		return printJSON(stdout, state)
	}
	object := ""
	enabled := false
	selectors := []string{}
	if action == "clear" && len(args) == 1 {
		object = strings.TrimSpace(args[0])
	} else if action == "update" && len(args) == 1 {
		spec, err := jsonx.ParseEncodedObject(args[0], "update ccchat feedTracking")
		if err != nil {
			return err
		}
		for key := range spec {
			if key != "object" && key != "enabled" && key != "trackedFields" {
				return fmt.Errorf("ccchat_feed_tracking_invalid: unknown or server-derived field %s", key)
			}
		}
		object = strings.TrimSpace(fmt.Sprint(spec["object"]))
		var ok bool
		enabled, ok = spec["enabled"].(bool)
		if !ok {
			return fmt.Errorf("ccchat_feed_tracking_invalid: enabled must be a boolean")
		}
		values, ok := spec["trackedFields"].([]any)
		if !ok {
			return fmt.Errorf("ccchat_feed_tracking_invalid: trackedFields must be an explicit array")
		}
		if len(values) > 20 {
			return fmt.Errorf("ccchat_field_limit_exceeded: at most 20 fields may be tracked")
		}
		for _, value := range values {
			selector, ok := value.(string)
			if !ok || strings.TrimSpace(selector) == "" {
				return fmt.Errorf("ccchat_field_not_candidate: every trackedFields item must be a non-empty string")
			}
			selectors = append(selectors, strings.TrimSpace(selector))
		}
	} else {
		return fmt.Errorf("cloudcc %s ccchat <projectPath> feedTracking %s", action,
			map[string]string{"getList": "", "get": "<object>", "candidates": "<object>", "clear": "<object>", "update": "<encodedSpecJson|@file>"}[action])
	}
	if object == "" {
		return fmt.Errorf("ccchat_object_required: object is required")
	}
	state, err := queryCcchatFeedTracking(projectPath, cfg, object)
	if err != nil {
		return err
	}
	bySelector := map[string][]map[string]any{}
	for _, field := range anyMaps(state["candidateFields"]) {
		id := strings.TrimSpace(fmt.Sprint(field["fieldId"]))
		api := strings.TrimSpace(fmt.Sprint(field["apiName"]))
		if id != "" {
			key := strings.ToLower(id)
			bySelector[key] = append(bySelector[key], field)
		}
		if api != "" {
			key := strings.ToLower(api)
			if key != strings.ToLower(id) {
				bySelector[key] = append(bySelector[key], field)
			}
		}
	}
	fields := []map[string]any{}
	seen := map[string]bool{}
	for _, selector := range selectors {
		matches := bySelector[strings.ToLower(selector)]
		if len(matches) == 0 {
			return fmt.Errorf("ccchat_field_not_candidate: %q is not eligible", selector)
		}
		if len(matches) != 1 {
			return fmt.Errorf("ccchat_field_ambiguous: %q matches more than one eligible field", selector)
		}
		field := matches[0]
		id := fmt.Sprint(field["fieldId"])
		idKey := strings.ToLower(id)
		if seen[idKey] {
			return fmt.Errorf("ccchat_field_duplicate: %q resolves more than once", selector)
		}
		seen[idKey] = true
		fields = append(fields, map[string]any{"fieldId": id, "api": field["apiName"], "fieldType": field["fieldType"], "datafield": field["datafield"]})
	}
	layout := map[string]any{"objid": state["objectId"], "prefix": state["prefix"], "istracked": enabled, "fields": fields}
	layoutJSON, _ := json.Marshal(layout)
	var ignored map[string]any
	if err := postCcchatResponse(projectPath, cfg, "/api/feedTrack/saveTrack", map[string]any{"layoutJson": string(layoutJSON)}, &ignored); err != nil {
		return err
	}
	readback, err := queryCcchatFeedTracking(projectPath, cfg, fmt.Sprint(state["objectId"]))
	if err != nil {
		return fmt.Errorf("CCChat feed tracking was submitted but readback failed: %w", err)
	}
	if !ccchatFeedTrackingMatches(readback, state, enabled, fields) {
		return fmt.Errorf("ccchat_readback_mismatch: saved feed tracking does not match setup-service readback")
	}
	delete(readback, "candidateFields")
	return printJSON(stdout, readback)
}

func queryCcchatFeedTracking(projectPath string, cfg config.Config, object string) (map[string]any, error) {
	var response map[string]any
	if err := postCcchatResponse(projectPath, cfg, "/api/feedTrack/queryObjectFieldList", map[string]any{"objId": object}, &response); err != nil {
		return nil, err
	}
	data := ccchatData(response)
	returnedObject := strings.TrimSpace(fmt.Sprint(firstAny(data["objid"], data["objectId"])))
	prefix := strings.TrimSpace(fmt.Sprint(data["prefix"]))
	fieldsRaw, hasFields := data["fields"]
	if returnedObject == "" || prefix == "" || !hasFields || !isCcchatArray(fieldsRaw) {
		return nil, fmt.Errorf("ccchat_setup_response_invalid: feedTracking.read is missing object identity, prefix, or fields")
	}
	fields := anyMaps(data["fields"])
	candidates := make([]map[string]any, 0, len(fields))
	tracked := []map[string]any{}
	for _, field := range fields {
		if strings.TrimSpace(fmt.Sprint(firstAny(field["id"], field["fieldId"]))) == "" ||
			strings.TrimSpace(fmt.Sprint(firstAny(field["apiname"], field["api"], field["apiName"]))) == "" ||
			strings.TrimSpace(fmt.Sprint(firstAny(field["type"], field["fieldType"]))) == "" ||
			strings.TrimSpace(fmt.Sprint(firstAny(field["datafield"], field["dataField"]))) == "" {
			return nil, fmt.Errorf("ccchat_setup_response_invalid: feedTracking candidate is missing identity or physical mapping")
		}
		item := map[string]any{
			"fieldId": firstAny(field["id"], field["fieldId"]), "apiName": firstAny(field["apiname"], field["api"], field["apiName"]),
			"label": firstAny(field["label"], field["name"]), "fieldType": firstAny(field["type"], field["fieldType"]),
			"datafield": firstAny(field["datafield"], field["dataField"]), "tracked": truthyAny(firstAny(field["istracked"], field["isTracked"])),
		}
		candidates = append(candidates, item)
		if item["tracked"] == true {
			tracked = append(tracked, item)
		}
	}
	return map[string]any{"domain": "ccchat", "resource": "feedTracking", "objectId": firstAny(data["objid"], data["objectId"], object),
		"prefix": data["prefix"], "enabled": truthyAny(firstAny(data["istracked"], data["isTracked"])),
		"trackedFields": tracked, "candidateFields": candidates}, nil
}

func normalizeCcchatSettings(response map[string]any) map[string]any {
	data := ccchatData(response)
	domains := []string{}
	seen := map[string]bool{}
	for _, value := range strings.Split(strings.TrimSpace(fmt.Sprint(data["emailDomains"])), ";") {
		value = strings.ToLower(strings.TrimSpace(value))
		if value != "" && !seen[value] {
			domains = append(domains, value)
			seen[value] = true
		}
	}
	sort.Strings(domains)
	return map[string]any{"domain": "ccchat", "resource": "settings", "enabled": truthyAny(firstAny(data["isEnable"], data["enabled"])),
		"allowFeedEdit": truthyAny(data["allowFeedEdit"]), "allowGroupArchived": truthyAny(data["allowGroupArchived"]),
		"allowInvite": truthyAny(data["allowInvite"]), "emailDomains": domains}
}

func ccchatSettingsPayload(spec map[string]any) (map[string]any, error) {
	for key := range spec {
		if key != "enabled" && key != "allowFeedEdit" && key != "allowGroupArchived" && key != "allowInvite" && key != "emailDomains" {
			return nil, fmt.Errorf("ccchat_settings_invalid: unknown field %s", key)
		}
	}
	result := map[string]any{}
	for _, field := range []string{"enabled", "allowFeedEdit", "allowGroupArchived", "allowInvite"} {
		value, ok := spec[field].(bool)
		if !ok {
			return nil, fmt.Errorf("ccchat_settings_invalid: %s must be a boolean", field)
		}
		result[field] = value
	}
	result["isEnable"] = fmt.Sprint(result["enabled"])
	delete(result, "enabled")
	domains := []string{}
	if result["allowInvite"] == true {
		values, ok := spec["emailDomains"].([]any)
		if !ok || len(values) == 0 {
			return nil, fmt.Errorf("ccchat_settings_invalid: emailDomains is required when allowInvite=true")
		}
		seen := map[string]bool{}
		validDomain := regexp.MustCompile(`(?i)^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,63}$`)
		for _, value := range values {
			rawDomain, ok := value.(string)
			if !ok {
				return nil, fmt.Errorf("ccchat_settings_invalid: every emailDomains item must be a string")
			}
			domain := strings.ToLower(strings.TrimSpace(rawDomain))
			if !validDomain.MatchString(domain) {
				return nil, fmt.Errorf("ccchat_settings_invalid: invalid email domain %q", domain)
			}
			if !seen[domain] {
				domains = append(domains, domain)
				seen[domain] = true
			}
		}
		result["emailDomains"] = strings.Join(domains, ";")
	} else if _, exists := spec["emailDomains"]; exists {
		return nil, fmt.Errorf("ccchat_settings_invalid: emailDomains must be omitted when allowInvite=false")
	}
	return result, nil
}

func postCcchatResponse(projectPath string, cfg config.Config, apiPath string, body map[string]any, response *map[string]any) error {
	if err := postClassResponse(projectPath, cfg, "setup", apiPath, body, response); err != nil {
		return err
	}
	if *response == nil || len(*response) == 0 {
		return fmt.Errorf("CCChat setup-service request failed: empty response")
	}
	if result, exists := (*response)["result"].(bool); exists {
		if result {
			return nil
		}
		return fmt.Errorf("CCChat setup-service request failed: %s",
			redactApiRegistrarLogString(fmt.Sprint(firstAny((*response)["returnInfo"], (*response)["message"], (*response)["msg"], "request rejected"))))
	}
	if success, exists := (*response)["success"].(bool); exists {
		if success {
			return nil
		}
		return fmt.Errorf("CCChat setup-service request failed: %s",
			redactApiRegistrarLogString(fmt.Sprint(firstAny((*response)["returnInfo"], (*response)["message"], (*response)["msg"], "request rejected"))))
	}
	return fmt.Errorf("CCChat setup-service request failed: response has no boolean result/success field")
}

func ccchatFeedTrackingMatches(readback map[string]any, before map[string]any, enabled bool, fields []map[string]any) bool {
	if truthyAny(readback["enabled"]) != enabled ||
		fmt.Sprint(readback["objectId"]) != fmt.Sprint(before["objectId"]) ||
		fmt.Sprint(readback["prefix"]) != fmt.Sprint(before["prefix"]) {
		return false
	}
	want := make([]string, 0, len(fields))
	for _, field := range fields {
		want = append(want, strings.ToLower(strings.TrimSpace(fmt.Sprint(field["fieldId"]))))
	}
	got := []string{}
	for _, field := range anyMaps(readback["trackedFields"]) {
		got = append(got, strings.ToLower(strings.TrimSpace(fmt.Sprint(field["fieldId"]))))
	}
	sort.Strings(want)
	sort.Strings(got)
	return reflect.DeepEqual(want, got)
}

func ccchatData(response map[string]any) map[string]any {
	raw := response["data"]
	if text, ok := raw.(string); ok {
		var parsed map[string]any
		if json.Unmarshal([]byte(text), &parsed) == nil {
			return parsed
		}
	}
	if data, ok := raw.(map[string]any); ok {
		return data
	}
	return response
}

func validateCcchatSettingsResponse(response map[string]any) error {
	data := ccchatData(response)
	for _, names := range [][]string{{"isEnable", "enabled"}, {"allowFeedEdit"}, {"allowGroupArchived"}, {"allowInvite"}} {
		value, found := ccchatFirstPresent(data, names...)
		if !found || !ccchatBooleanCompatible(value) {
			return fmt.Errorf("ccchat_setup_response_invalid: settings.read field %s is missing or invalid", strings.Join(names, "/"))
		}
	}
	return nil
}

func validateCcchatObjectListResponse(response map[string]any) error {
	raw, exists := response["data"]
	if !exists {
		raw = response
	}
	if text, ok := raw.(string); ok {
		var decoded any
		if json.Unmarshal([]byte(text), &decoded) != nil {
			return fmt.Errorf("ccchat_setup_response_invalid: feedTracking.list data is not valid JSON")
		}
		raw = decoded
	}
	if isCcchatArray(raw) {
		return nil
	}
	if object, ok := raw.(map[string]any); ok {
		for _, key := range []string{"objList", "objects", "list", "rows"} {
			if value, present := object[key]; present && isCcchatArray(value) {
				return nil
			}
		}
	}
	return fmt.Errorf("ccchat_setup_response_invalid: feedTracking.list object array is missing")
}

func ccchatFirstPresent(data map[string]any, names ...string) (any, bool) {
	for _, name := range names {
		if value, exists := data[name]; exists && value != nil {
			return value, true
		}
	}
	return nil, false
}

func ccchatBooleanCompatible(value any) bool {
	if _, ok := value.(bool); ok {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(fmt.Sprint(value))) {
	case "true", "false", "1", "0", "yes", "no":
		return true
	}
	return false
}

func isCcchatArray(value any) bool {
	switch value.(type) {
	case []any, []map[string]any:
		return true
	default:
		return false
	}
}

func ccchatRows(response map[string]any) []map[string]any {
	data := ccchatData(response)
	var rows []map[string]any
	for _, key := range []string{"objList", "objects", "list", "rows"} {
		if candidate := anyMaps(data[key]); len(candidate) > 0 {
			rows = candidate
			break
		}
	}
	if len(rows) == 0 {
		rows = anyMaps(response["data"])
	}
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		count := 0
		if value := firstAny(row["numsTracked"], row["trackednums"], row["trackedCount"]); value != nil {
			count, _ = strconv.Atoi(strings.TrimSpace(fmt.Sprint(value)))
		}
		result = append(result, map[string]any{
			"objectId":     firstAny(row["id"], row["objId"], row["objectId"]),
			"apiName":      firstAny(row["apiName"], row["apiname"]),
			"label":        firstAny(row["label"], row["name"], row["objLabel"]),
			"prefix":       row["prefix"],
			"enabled":      truthyAny(firstAny(row["isTracked"], row["istracked"], row["tracked"])),
			"trackedCount": count,
		})
	}
	return result
}

func anyMaps(raw any) []map[string]any {
	values, _ := raw.([]any)
	result := []map[string]any{}
	for _, value := range values {
		if row, ok := value.(map[string]any); ok {
			result = append(result, row)
		}
	}
	if typed, ok := raw.([]map[string]any); ok {
		return typed
	}
	return result
}

func truthyAny(value any) bool {
	if result, ok := value.(bool); ok {
		return result
	}
	switch strings.ToLower(strings.TrimSpace(fmt.Sprint(value))) {
	case "true", "1", "yes":
		return true
	}
	return false
}
