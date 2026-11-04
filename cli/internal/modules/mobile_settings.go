package modules

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

	"cloudcc-customization-expert-go/internal/config"
	"cloudcc-customization-expert-go/internal/jsonx"
)

var mobileSettingNames = map[string]string{
	"parameters": "parameters", "parameter": "parameters", "homepage": "homepage",
	"offlineCache": "offlineCache", "offline-cache": "offlineCache",
	"recommendedModules": "recommendedModules", "recommended-modules": "recommendedModules",
	"scanModules": "scanModules", "scan-modules": "scanModules",
	"homepageShortcuts": "homepageShortcuts", "homepage-shortcuts": "homepageShortcuts",
	"firstPage": "firstPage", "first-page": "firstPage",
}

var mobileSettingReadPaths = map[string]string{
	"parameters": "/api/mobileSetup/queryMobileSetup", "homepage": "/api/mobileSetup/getMobileMainSet",
	"offlineCache": "/api/mobileSetup/getGroupCacheObjects", "recommendedModules": "/api/mobileSetup/getMobilePreferFunctionsSet",
	"scanModules": "/api/mobileSetup/getMobileScanSet", "homepageShortcuts": "/api/mobileSetup/getHomePageShortcuts",
	"firstPage": "/api/mobileSetup/getBottomButton",
}

var mobileSettingWritePaths = map[string]string{
	"parameters": "/api/mobileSetup/saveMobileSetup", "homepage": "/api/mobileSetup/saveMobileMainSet",
	"offlineCache": "/api/mobileSetup/saveGroupCacheObjects", "recommendedModules": "/api/mobileSetup/saveMobilePreferFunctionsSet",
	"scanModules": "/api/mobileSetup/saveMobileScanSet", "homepageShortcuts": "/api/mobileSetup/saveHomePageShortcuts",
	"firstPage": "/api/mobileSetup/saveBottomButton",
}

func handleMobileSettings(action string, args []string, stdout io.Writer, cwd string) error {
	projectPath := firstArg(args, cwd)
	cfg, err := config.Load(projectPath)
	if err != nil {
		return err
	}
	rest := argsAfterProject(args)
	if len(rest) == 0 {
		return fmt.Errorf("cloudcc %s mobileSetting <projectPath> <parameters|homepage|offlineCache|recommendedModules|scanModules|homepageShortcuts|firstPage> [@spec.json]", action)
	}
	resource, ok := mobileSettingNames[strings.TrimSpace(rest[0])]
	if !ok {
		return fmt.Errorf("mobile_settings_resource_invalid: unsupported resource %q", rest[0])
	}
	rest = rest[1:]
	switch strings.TrimSpace(action) {
	case "get":
		if len(rest) != 0 {
			break
		}
		state, readErr := readMobileSetting(projectPath, cfg, resource)
		if readErr != nil {
			return readErr
		}
		return printJSON(stdout, state)
	case "candidates":
		if len(rest) != 0 {
			break
		}
		state, readErr := readMobileSetting(projectPath, cfg, resource)
		if readErr != nil {
			return readErr
		}
		return printJSON(stdout, mobileSettingCandidates(resource, state))
	case "update":
		if len(rest) != 1 {
			break
		}
		spec, parseErr := jsonx.ParseEncodedObject(rest[0], "update mobileSetting "+resource)
		if parseErr != nil {
			return parseErr
		}
		return updateMobileSetting(projectPath, cfg, resource, spec, stdout)
	}
	return fmt.Errorf("cloudcc %s mobileSetting <projectPath> %s%s", action, restResourceName(resource), map[bool]string{true: " <encodedSpecJson|@file>", false: ""}[action == "update"])
}

func readMobileSetting(projectPath string, cfg config.Config, resource string) (map[string]any, error) {
	var response map[string]any
	if err := postMobileSettingResponse(projectPath, cfg, mobileSettingReadPaths[resource], map[string]any{}, &response); err != nil {
		return nil, err
	}
	data := mobileSettingData(response)
	state := map[string]any{"domain": "mobile-settings", "resource": resource}
	switch resource {
	case "parameters":
		for canonical, setup := range map[string]string{
			"mobileAppNewTaskEnabled": "istask", "wechatContentRequired": "isRequiredWeChatContent",
			"signInPhotoRequired": "mustTakePhotoWhenSignIn", "scanValidationRulesEnabled": "isEnableValidationRulesForScan",
			"recordCallDetails": "isRecordCallDetails", "allowModifyCallDetails": "isModifyCallDetails",
			"callLocationRequired": "isNecessaryToLocateWhenRecordCall"} {
			state[canonical] = truthyAny(data[setup])
		}
		image, err := readMobileEventImage(projectPath, cfg)
		if err != nil {
			return nil, err
		}
		state["eventImage"] = image
	case "homepage", "recommendedModules", "scanModules":
		field := map[string]string{"homepage": "components", "recommendedModules": "modules", "scanModules": "modules"}[resource]
		state[field] = mobileStatusRows(data, resource == "homepage")
	case "offlineCache":
		state["enabled"] = truthyAny(firstAny(data["mobileTabSet"], data["mobileCacheStep"]))
		state["overrideUserSelections"] = truthyAny(data["isCoverUserDefined"])
		selected, available := mobileSelectable(data["selectedAppTabList"], data["unselectedAppTabList"])
		state["selectedTabs"], state["availableTabs"] = selected, available
	case "homepageShortcuts":
		state["hidden"] = truthyAny(data["isHideHomePageShortcuts"])
		selected, available := mobileSelectable(data["selectedHomePageShortcutsList"], data["unselectedHomePageShortcutsList"])
		state["selectedItems"], state["availableItems"] = selected, available
	case "firstPage":
		selected, available := mobileSelectable(data["selectedBottomButtonList"], data["unselectedBottomButtonList"])
		state["selectedItems"], state["availableItems"] = selected, available
	}
	return state, nil
}

func updateMobileSetting(projectPath string, cfg config.Config, resource string, spec map[string]any, stdout io.Writer) error {
	current, err := readMobileSetting(projectPath, cfg, resource)
	if err != nil {
		return err
	}
	if err := validateMobileSettingPatch(resource, spec, current); err != nil {
		return err
	}
	desired := copyAnyMap(current)
	delete(desired, "domain")
	delete(desired, "resource")
	for key, value := range spec {
		switch key {
		case "eventImage":
			base, _ := desired[key].(map[string]any)
			merged := copyAnyMap(base)
			if patch, ok := value.(map[string]any); ok {
				for nestedKey, nestedValue := range patch {
					merged[nestedKey] = nestedValue
				}
			}
			desired[key] = merged
		case "components", "modules":
			desired[key] = mergeMobileStatusItems(desired[key], value)
		default:
			desired[key] = value
		}
	}
	if err := validateMobileSetting(resource, desired, current); err != nil {
		return err
	}
	delete(desired, "availableTabs")
	delete(desired, "availableItems")
	payload, err := mobileSettingPayload(resource, desired)
	if err != nil {
		return err
	}
	if resource == "parameters" {
		var rawResponse map[string]any
		if err := postMobileSettingResponse(projectPath, cfg, mobileSettingReadPaths[resource], map[string]any{}, &rawResponse); err != nil {
			return err
		}
		raw := mobileSettingData(rawResponse)
		for key, value := range payload {
			raw[key] = value
		}
		payload = raw
	}
	var ignored map[string]any
	if err := postMobileSettingResponse(projectPath, cfg, mobileSettingWritePaths[resource], payload, &ignored); err != nil {
		return err
	}
	if resource == "parameters" {
		if _, changed := spec["eventImage"]; changed {
			image, _ := desired["eventImage"].(map[string]any)
			if err := writeMobileEventImage(projectPath, cfg, image); err != nil {
				return err
			}
		}
	}
	readback, err := readMobileSetting(projectPath, cfg, resource)
	if err != nil {
		return err
	}
	delete(readback, "domain")
	delete(readback, "resource")
	if !mobileDesiredMatches(desired, readback) {
		return fmt.Errorf("mobile_settings_readback_mismatch: saved %s does not match setup-service readback", resource)
	}
	return printJSON(stdout, map[string]any{"domain": "mobile-settings", "resource": resource, "changed": true, "readback": readback})
}

func validateMobileSettingPatch(resource string, spec, current map[string]any) error {
	allowed := map[string]map[string]bool{
		"parameters": {"mobileAppNewTaskEnabled": true, "wechatContentRequired": true, "signInPhotoRequired": true, "scanValidationRulesEnabled": true, "recordCallDetails": true, "allowModifyCallDetails": true, "callLocationRequired": true, "eventImage": true},
		"homepage":   {"components": true}, "offlineCache": {"enabled": true, "overrideUserSelections": true, "selectedTabs": true},
		"recommendedModules": {"modules": true}, "scanModules": {"modules": true}, "homepageShortcuts": {"hidden": true, "selectedItems": true}, "firstPage": {"selectedItems": true},
	}[resource]
	for key := range spec {
		if !allowed[key] {
			return fmt.Errorf("mobile_settings_spec_invalid: unknown %s field %s", resource, key)
		}
	}
	if resource == "offlineCache" {
		for _, key := range []string{"enabled", "overrideUserSelections", "selectedTabs"} {
			if _, ok := spec[key]; !ok {
				return fmt.Errorf("mobile_settings_spec_invalid: %s is required", key)
			}
		}
		if _, ok := spec["enabled"].(bool); !ok {
			return fmt.Errorf("mobile_settings_spec_invalid: enabled must be boolean")
		}
		if _, ok := spec["overrideUserSelections"].(bool); !ok {
			return fmt.Errorf("mobile_settings_spec_invalid: overrideUserSelections must be boolean")
		}
		if !mobileStringArray(spec["selectedTabs"]) {
			return fmt.Errorf("mobile_settings_spec_invalid: selectedTabs must be an explicit string array")
		}
	}
	if resource == "homepageShortcuts" {
		for _, key := range []string{"hidden", "selectedItems"} {
			if _, ok := spec[key]; !ok {
				return fmt.Errorf("mobile_settings_spec_invalid: %s is required", key)
			}
		}
		if _, ok := spec["hidden"].(bool); !ok {
			return fmt.Errorf("mobile_settings_spec_invalid: hidden must be boolean")
		}
		if !mobileStringArray(spec["selectedItems"]) {
			return fmt.Errorf("mobile_settings_spec_invalid: selectedItems must be an explicit string array")
		}
	}
	if resource == "firstPage" {
		if _, ok := spec["selectedItems"]; !ok {
			return fmt.Errorf("mobile_settings_spec_invalid: selectedItems is required")
		}
		if !mobileStringArray(spec["selectedItems"]) {
			return fmt.Errorf("mobile_settings_spec_invalid: selectedItems must be an explicit string array")
		}
	}
	if resource == "homepage" || resource == "recommendedModules" || resource == "scanModules" {
		field := map[string]string{"homepage": "components", "recommendedModules": "modules", "scanModules": "modules"}[resource]
		patches := anyMaps(spec[field])
		if len(patches) == 0 {
			return fmt.Errorf("mobile_settings_spec_invalid: %s must be a non-empty array", field)
		}
		candidates := map[string]bool{}
		for _, item := range anyMaps(current[field]) {
			candidates[strings.ToLower(fmt.Sprint(item["id"]))] = true
		}
		for _, item := range patches {
			allowedItem := map[string]bool{"id": true}
			if resource == "homepage" {
				allowedItem["state"] = true
			} else {
				allowedItem["enabled"] = true
			}
			for key := range item {
				if !allowedItem[key] {
					return fmt.Errorf("mobile_settings_spec_invalid: unknown %s item field %s", field, key)
				}
			}
			id := strings.ToLower(fmt.Sprint(item["id"]))
			if id == "" || !candidates[id] {
				return fmt.Errorf("mobile_settings_candidate_not_found: %v", item["id"])
			}
			if resource == "homepage" {
				state := fmt.Sprint(item["state"])
				if state != "enabled" && state != "disabled" && state != "closed" {
					return fmt.Errorf("mobile_settings_state_invalid: %s", state)
				}
			} else if _, ok := item["enabled"].(bool); !ok {
				return fmt.Errorf("mobile_settings_spec_invalid: enabled must be boolean")
			}
		}
	}
	if resource == "parameters" {
		for key, value := range spec {
			if key != "eventImage" {
				if _, ok := value.(bool); !ok {
					return fmt.Errorf("mobile_settings_spec_invalid: %s must be boolean", key)
				}
			}
		}
		if raw, ok := spec["eventImage"]; ok {
			image, ok := raw.(map[string]any)
			if !ok || len(image) == 0 {
				return fmt.Errorf("mobile_settings_spec_invalid: eventImage must be a non-empty object")
			}
			allowedImage := map[string]bool{"uploadMode": true, "cameraWatermarkEnabled": true, "uploadWatermarkEnabled": true, "watermarkStyle": true, "showFieldLabel": true, "watermarkFields": true}
			for key := range image {
				if !allowedImage[key] {
					return fmt.Errorf("mobile_settings_spec_invalid: unknown eventImage field %s", key)
				}
			}
			for _, key := range []string{"cameraWatermarkEnabled", "uploadWatermarkEnabled", "showFieldLabel"} {
				if value, exists := image[key]; exists {
					if _, ok := value.(bool); !ok {
						return fmt.Errorf("mobile_settings_spec_invalid: eventImage.%s must be boolean", key)
					}
				}
			}
			if value, exists := image["watermarkFields"]; exists && !mobileStringArray(value) {
				return fmt.Errorf("mobile_settings_spec_invalid: eventImage.watermarkFields must be an explicit string array")
			}
		}
	}
	return nil
}

func mobileStringArray(value any) bool {
	switch items := value.(type) {
	case []string:
		return true
	case []any:
		for _, item := range items {
			if text, ok := item.(string); !ok || strings.TrimSpace(text) == "" {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func validateMobileSetting(resource string, desired map[string]any, current map[string]any) error {
	allowed := map[string]map[string]bool{
		"parameters": {"mobileAppNewTaskEnabled": true, "wechatContentRequired": true, "signInPhotoRequired": true, "scanValidationRulesEnabled": true, "recordCallDetails": true, "allowModifyCallDetails": true, "callLocationRequired": true, "eventImage": true},
		"homepage":   {"components": true}, "offlineCache": {"enabled": true, "overrideUserSelections": true, "selectedTabs": true},
		"recommendedModules": {"modules": true}, "scanModules": {"modules": true},
		"homepageShortcuts": {"hidden": true, "selectedItems": true}, "firstPage": {"selectedItems": true},
	}[resource]
	for key := range desired {
		if !allowed[key] && key != "availableTabs" && key != "availableItems" {
			return fmt.Errorf("mobile_settings_spec_invalid: unknown %s field %s", resource, key)
		}
	}
	selectionField, availableField, exact, limit := "", "", false, 0
	switch resource {
	case "offlineCache":
		selectionField, availableField, limit = "selectedTabs", "availableTabs", 10
	case "homepageShortcuts", "firstPage":
		selectionField, availableField, exact, limit = "selectedItems", "availableItems", true, 4
	}
	if selectionField != "" {
		selected := anyStrings(desired[selectionField])
		if (exact && len(selected) != limit) || (!exact && len(selected) > limit) {
			return fmt.Errorf("mobile_settings_selection_limit: invalid %s item count", selectionField)
		}
		if resource == "offlineCache" && truthyAny(desired["enabled"]) && len(selected) == 0 {
			return fmt.Errorf("mobile_settings_spec_invalid: selectedTabs is required when enabled=true")
		}
		candidates := map[string]bool{}
		for _, item := range anyMaps(current[availableField]) {
			candidates[strings.ToLower(fmt.Sprint(item["id"]))] = true
		}
		seen := map[string]bool{}
		for _, id := range selected {
			key := strings.ToLower(strings.TrimSpace(id))
			if seen[key] {
				return fmt.Errorf("mobile_settings_duplicate_selection: %s", id)
			}
			if !candidates[key] {
				return fmt.Errorf("mobile_settings_candidate_not_found: %s", id)
			}
			seen[key] = true
		}
	}
	if resource == "parameters" {
		image, _ := desired["eventImage"].(map[string]any)
		mode := fmt.Sprint(image["uploadMode"])
		if mode != "cameraAndAlbum" && mode != "cameraOnly" && mode != "albumOnly" {
			return fmt.Errorf("mobile_settings_spec_invalid: invalid eventImage.uploadMode")
		}
		style := fmt.Sprint(image["watermarkStyle"])
		if style != "topOpaque" && style != "bottomTranslucent" {
			return fmt.Errorf("mobile_settings_spec_invalid: invalid eventImage.watermarkStyle")
		}
		fields := anyStrings(image["watermarkFields"])
		if len(fields) > 5 {
			return fmt.Errorf("mobile_settings_selection_limit: watermarkFields supports at most 5 items")
		}
		if mode == "albumOnly" && truthyAny(image["cameraWatermarkEnabled"]) {
			return fmt.Errorf("mobile_settings_spec_invalid: camera watermark is unavailable for albumOnly")
		}
		if (truthyAny(image["cameraWatermarkEnabled"]) || truthyAny(image["uploadWatermarkEnabled"])) && len(fields) == 0 {
			return fmt.Errorf("mobile_settings_spec_invalid: watermarkFields is required when watermark is enabled")
		}
		candidates := map[string]bool{}
		currentImage, _ := current["eventImage"].(map[string]any)
		for _, key := range []string{"availableFields", "availableVariables"} {
			for _, item := range anyMaps(currentImage[key]) {
				candidates[strings.ToLower(fmt.Sprint(item["id"]))] = true
			}
		}
		seen := map[string]bool{}
		for _, id := range fields {
			key := strings.ToLower(id)
			if seen[key] {
				return fmt.Errorf("mobile_settings_duplicate_selection: %s", id)
			}
			if !candidates[key] {
				return fmt.Errorf("mobile_settings_candidate_not_found: %s", id)
			}
			seen[key] = true
		}
	}
	return nil
}

func mergeMobileStatusItems(current any, patch any) []map[string]any {
	result := anyMaps(current)
	byID := map[string]map[string]any{}
	for _, item := range result {
		byID[strings.ToLower(fmt.Sprint(item["id"]))] = item
	}
	for _, change := range anyMaps(patch) {
		if target := byID[strings.ToLower(fmt.Sprint(change["id"]))]; target != nil {
			for key, value := range change {
				target[key] = value
			}
		}
	}
	return result
}

func mobileSettingPayload(resource string, desired map[string]any) (map[string]any, error) {
	switch resource {
	case "parameters":
		payload := map[string]any{}
		for canonical, setup := range map[string]string{"mobileAppNewTaskEnabled": "istask", "wechatContentRequired": "isRequiredWeChatContent", "signInPhotoRequired": "mustTakePhotoWhenSignIn", "scanValidationRulesEnabled": "isEnableValidationRulesForScan", "recordCallDetails": "isRecordCallDetails", "allowModifyCallDetails": "isModifyCallDetails", "callLocationRequired": "isNecessaryToLocateWhenRecordCall"} {
			payload[setup] = fmt.Sprint(truthyAny(desired[canonical]))
		}
		if image, ok := desired["eventImage"].(map[string]any); ok {
			payload["isdisablum"] = fmt.Sprint(fmt.Sprint(image["uploadMode"]) == "cameraOnly")
			payload["isEnableAblumWatermark"] = fmt.Sprint(truthyAny(image["uploadWatermarkEnabled"]))
		}
		return payload, nil
	case "homepage":
		return map[string]any{"objPermission": mobileStatusPermission(desired["components"], true)}, nil
	case "recommendedModules", "scanModules":
		return map[string]any{"objPermission": mobileStatusPermission(desired["modules"], false)}, nil
	case "offlineCache":
		return map[string]any{"mobileTabSet": fmt.Sprint(truthyAny(desired["enabled"])), "isCoverUserDefined": fmt.Sprint(truthyAny(desired["overrideUserSelections"])), "appTabList": strings.Join(anyStrings(desired["selectedTabs"]), ",")}, nil
	case "homepageShortcuts":
		return map[string]any{"isHide": fmt.Sprint(truthyAny(desired["hidden"])), "appTabList": strings.Join(anyStrings(desired["selectedItems"]), ",")}, nil
	case "firstPage":
		return map[string]any{"bottomButtonIds": strings.Join(anyStrings(desired["selectedItems"]), ",")}, nil
	}
	return nil, fmt.Errorf("mobile_settings_resource_invalid: %s", resource)
}

func mobileSettingCandidates(resource string, state map[string]any) map[string]any {
	result := map[string]any{"domain": "mobile-settings", "resource": resource + "Candidates"}
	switch resource {
	case "homepage":
		result["items"] = state["components"]
	case "recommendedModules", "scanModules":
		result["items"] = state["modules"]
	case "offlineCache":
		result["items"] = state["availableTabs"]
	case "homepageShortcuts", "firstPage":
		result["items"] = state["availableItems"]
	case "parameters":
		if image, ok := state["eventImage"].(map[string]any); ok {
			result["objectFields"] = image["availableFields"]
			result["variables"] = image["availableVariables"]
			result["limit"] = 5
		}
	}
	return result
}

func readMobileEventImage(projectPath string, cfg config.Config) (map[string]any, error) {
	var detail, watermark map[string]any
	selector := map[string]any{"fieldId": "ffeEventPicture01", "fdtype": "IMG", "objid": "event"}
	if err := postMobileSettingResponse(projectPath, cfg, "/api/fieldSetup/editField", selector, &detail); err != nil {
		return nil, err
	}
	if err := postMobileSettingResponse(projectPath, cfg, "/api/fieldSetup/watermark", map[string]any{"objid": "event", "fieldId": "ffeEventPicture01"}, &watermark); err != nil {
		return nil, err
	}
	d := mobileSettingData(detail)
	w := mobileSettingData(watermark)
	field, _ := d["fieldObj"].(map[string]any)
	upload := fmt.Sprint(firstAny(w["uploadMode"], field["uploadMode"]))
	mode := map[string]string{"camera_album": "cameraAndAlbum", "camera": "cameraOnly", "album": "albumOnly"}[upload]
	if mode == "" {
		mode = "cameraAndAlbum"
	}
	style := map[string]string{"top_opaque": "topOpaque", "bottom_translucent": "bottomTranslucent"}[fmt.Sprint(firstAny(w["watermarkStyle"], field["watermarkStyle"]))]
	if style == "" {
		style = "topOpaque"
	}
	selected := []string{}
	for _, row := range anyMaps(firstAny(w["selectCheckFiledInfo"], w["selectedFields"])) {
		selected = append(selected, fmt.Sprint(firstAny(row["id"], row["fieldId"])))
	}
	return map[string]any{"objectId": "event", "fieldId": "ffeEventPicture01", "uploadMode": mode, "cameraWatermarkEnabled": truthyAny(firstAny(w["cameraWatermarkStatus"], field["cameraWatermarkStatus"])), "uploadWatermarkEnabled": truthyAny(firstAny(w["uploadWatermarkStatus"], field["uploadWatermarkStatus"])), "watermarkStyle": style, "showFieldLabel": truthyAny(firstAny(w["showFieldLabel"], field["showFieldLabel"], true)), "watermarkFields": selected, "availableFields": mobileCandidateRows(w["selectInfoList"]), "availableVariables": mobileCandidateRows(w["qualityList"])}, nil
}

func writeMobileEventImage(projectPath string, cfg config.Config, image map[string]any) error {
	var detail map[string]any
	if err := postMobileSettingResponse(projectPath, cfg, "/api/fieldSetup/editField", map[string]any{"fieldId": "ffeEventPicture01", "fdtype": "IMG", "objid": "event"}, &detail); err != nil {
		return err
	}
	d := mobileSettingData(detail)
	field, _ := d["fieldObj"].(map[string]any)
	obj := copyAnyMap(field)
	obj["schemetableId"] = "event"
	if obj["id"] == nil {
		obj["id"] = "ffeEventPicture01"
	}
	profiles := []map[string]any{}
	for _, row := range anyMaps(d["profileList"]) {
		profiles = append(profiles, map[string]any{"profileid": firstAny(row["id"], row["profileid"]), "permission": map[bool]string{true: "0,0", false: "1,0"}[fmt.Sprint(row["visible"]) == "0"]})
	}
	payload := map[string]any{"objid": "event", "obj": obj, "dtype": "IMG:图片", "fdtype": "IMG", "profileFieldJson": profiles, "uploadMode": map[string]string{"cameraAndAlbum": "camera_album", "cameraOnly": "camera", "albumOnly": "album"}[fmt.Sprint(image["uploadMode"])], "cameraWatermarkStatus": boolDigit(image["cameraWatermarkEnabled"]), "uploadWatermarkStatus": boolDigit(image["uploadWatermarkEnabled"]), "watermarkStyle": map[string]string{"topOpaque": "top_opaque", "bottomTranslucent": "bottom_translucent"}[fmt.Sprint(image["watermarkStyle"])], "showFieldLabel": boolDigit(image["showFieldLabel"]), "watermarkfield": strings.Join(anyStrings(image["watermarkFields"]), ",")}
	var ignored map[string]any
	return postMobileSettingResponse(projectPath, cfg, "/api/fieldSetup/saveField", payload, &ignored)
}

func postMobileSettingResponse(projectPath string, cfg config.Config, path string, body map[string]any, response *map[string]any) error {
	if err := postClassResponse(projectPath, cfg, "setup", path, body, response); err != nil {
		return err
	}
	if *response == nil || len(*response) == 0 {
		return fmt.Errorf("mobile_settings_setup_response_invalid: empty response")
	}
	if ok, exists := (*response)["result"].(bool); exists {
		if ok {
			return nil
		}
		return fmt.Errorf("mobile_settings_setup_request_failed: %v", firstAny((*response)["returnInfo"], (*response)["message"], (*response)["msg"]))
	}
	if ok, exists := (*response)["success"].(bool); exists {
		if ok {
			return nil
		}
		return fmt.Errorf("mobile_settings_setup_request_failed: %v", firstAny((*response)["returnInfo"], (*response)["message"], (*response)["msg"]))
	}
	return fmt.Errorf("mobile_settings_setup_response_invalid: response has no boolean result/success field")
}

func mobileSettingData(response map[string]any) map[string]any {
	raw := response["data"]
	if text, ok := raw.(string); ok {
		var parsed map[string]any
		if json.Unmarshal([]byte(text), &parsed) == nil {
			return parsed
		}
		var rows []any
		if json.Unmarshal([]byte(text), &rows) == nil {
			return map[string]any{"items": rows}
		}
	}
	if data, ok := raw.(map[string]any); ok {
		return data
	}
	if rows, ok := raw.([]any); ok {
		return map[string]any{"items": rows}
	}
	return response
}
func mobileStatusRows(data map[string]any, three bool) []map[string]any {
	rows := anyMaps(firstAny(data["items"], data["rows"], data["list"]))
	if len(rows) == 0 {
		rows = anyMaps(data)
	}
	out := []map[string]any{}
	for _, row := range rows {
		id := fmt.Sprint(firstAny(row["id"], row["relateId"]))
		status := fmt.Sprint(firstAny(row["obj_operate_type"], row["objOperateType"]))
		if id == "" {
			continue
		}
		item := map[string]any{"id": id, "label": fmt.Sprint(firstAny(row["name"], row["label"], id))}
		flags := strings.Split(status, ",")
		if three {
			state := "enabled"
			if len(flags) > 2 && flags[2] == "1" {
				state = "closed"
			} else if len(flags) > 1 && flags[1] == "1" {
				state = "disabled"
			}
			item["state"] = state
		} else {
			item["enabled"] = len(flags) == 0 || flags[0] != "0"
		}
		out = append(out, item)
	}
	return out
}
func mobileSelectable(selectedRaw, unselectedRaw any) ([]string, []map[string]any) {
	selected := []string{}
	available := []map[string]any{}
	for _, group := range []struct {
		raw any
		on  bool
	}{{selectedRaw, true}, {unselectedRaw, false}} {
		for _, row := range anyMaps(group.raw) {
			id := fmt.Sprint(firstAny(row["id"], row["code"]))
			if id == "" {
				continue
			}
			if group.on {
				selected = append(selected, id)
			}
			available = append(available, map[string]any{"id": id, "label": fmt.Sprint(firstAny(row["label"], row["tabname"], row["name"], id)), "selected": group.on})
		}
	}
	return selected, available
}
func mobileStatusPermission(raw any, three bool) string {
	var b strings.Builder
	for _, item := range anyMaps(raw) {
		status := "1,0"
		if three {
			switch fmt.Sprint(item["state"]) {
			case "disabled":
				status = "0,1,0"
			case "closed":
				status = "1,0,1"
			default:
				status = "1,0,0"
			}
		} else if !truthyAny(item["enabled"]) {
			status = "0,1"
		}
		fmt.Fprintf(&b, "%v#%s@@", item["id"], status)
	}
	return b.String()
}
func mobileCandidateRows(raw any) []map[string]any {
	out := []map[string]any{}
	for _, row := range anyMaps(raw) {
		id := fmt.Sprint(firstAny(row["id"], row["fieldId"]))
		if id != "" {
			out = append(out, map[string]any{"id": id, "label": fmt.Sprint(firstAny(row["value"], row["labelname"], row["label"], id))})
		}
	}
	return out
}
func mobileDesiredMatches(desired, readback map[string]any) bool {
	for key, want := range desired {
		if key == "availableTabs" || key == "availableItems" {
			continue
		}
		if key == "eventImage" {
			w, _ := want.(map[string]any)
			g, _ := readback[key].(map[string]any)
			for k, v := range w {
				if strings.HasPrefix(k, "available") {
					continue
				}
				if !reflect.DeepEqual(v, g[k]) {
					return false
				}
			}
			continue
		}
		if !mobileJSONEqual(want, readback[key]) {
			return false
		}
	}
	return true
}

func mobileJSONEqual(left, right any) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && string(leftJSON) == string(rightJSON)
}
func anyStrings(raw any) []string {
	out := []string{}
	switch rows := raw.(type) {
	case []string:
		return rows
	case []any:
		for _, v := range rows {
			if m, ok := v.(map[string]any); ok {
				out = append(out, fmt.Sprint(m["id"]))
			} else {
				out = append(out, fmt.Sprint(v))
			}
		}
	}
	return out
}
func copyAnyMap(source map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range source {
		out[k] = v
	}
	return out
}
func boolDigit(value any) string {
	if truthyAny(value) {
		return "1"
	}
	return "0"
}
func restResourceName(resource string) string {
	if value := map[string]string{"parameters": "parameters", "homepage": "homepage", "offlineCache": "offlineCache", "recommendedModules": "recommendedModules", "scanModules": "scanModules", "homepageShortcuts": "homepageShortcuts", "firstPage": "firstPage"}[resource]; value != "" {
		return value
	}
	return resource
}
