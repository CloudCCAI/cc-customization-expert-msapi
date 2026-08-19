package modules

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"cloudcc-customization-expert-go/internal/config"
	"cloudcc-customization-expert-go/internal/httpclient"
	"cloudcc-customization-expert-go/internal/jsonx"
)

func handleCustomPage(action string, args []string, stdout io.Writer, _ io.Writer, cwd string) error {
	switch action {
	case "get", "list":
		return customPageGet(args, stdout, cwd)
	case "detail":
		return customPageDetail(args, stdout, cwd)
	case "create":
		return customPageCreate(args, stdout, cwd)
	case "update":
		return customPageUpdate(args, stdout, cwd)
	case "delete":
		return customPageDelete(args, stdout, cwd)
	default:
		return fmt.Errorf("unsupported customPage action: %s", action)
	}
}

func handleInjectionPage(action string, args []string, stdout io.Writer, _ io.Writer, cwd string) error {
	if action != "verify" {
		return fmt.Errorf("unsupported injectionPage action: %s", action)
	}
	if len(args) < 2 {
		return fmt.Errorf("cloudcc verify injectionPage <projectPath> <pageApi> [--expected-component <id|name>] [--expected-component-id <id>] [--expected-component-version <version>] [--stale-policy warning|failure] [--snapshot <@json|json>]")
	}
	projectPath := args[0]
	pageAPI := args[1]
	opts, err := parseInjectionVerifyOptions(args[2:])
	if err != nil {
		return err
	}
	page, err := customPageDetailData(projectPath, pageAPI)
	if err != nil {
		return err
	}
	refs := customPageComponentRefs(page)
	actualIDs := make([]string, 0, len(refs))
	actualNames := make([]string, 0, len(refs))
	actualVersions := make([]string, 0, len(refs))
	for _, ref := range refs {
		if id := strings.TrimSpace(fmt.Sprint(ref["comId"])); id != "" && id != "<nil>" {
			actualIDs = append(actualIDs, id)
		}
		if name := strings.TrimSpace(fmt.Sprint(ref["name"])); name != "" && name != "<nil>" {
			actualNames = append(actualNames, name)
		}
		if version := strings.TrimSpace(fmt.Sprint(ref["version"])); version != "" && version != "<nil>" {
			actualVersions = append(actualVersions, version)
		}
	}
	expectedComponentID := strings.TrimSpace(opts.expectedComponentID)
	expectedComponentName := ""
	resolvedExpected := map[string]any{}
	if opts.expectedComponent != "" {
		if component, err := resolvePageComponentForVerify(projectPath, opts.expectedComponent); err == nil {
			expectedComponentID = firstNonBlankString(expectedComponentID, anyString(component["id"]))
			expectedComponentName = firstNonBlankString(anyString(firstAny(component["compUniName"], component["component"], component["name"])), opts.expectedComponent)
			resolvedExpected = map[string]any{
				"id":          anyString(component["id"]),
				"compUniName": firstNonBlankString(anyString(firstAny(component["compUniName"], component["component"], component["name"])), opts.expectedComponent),
				"version":     firstNonBlankString(anyString(component["version"]), anyString(component["renderVersion"]), anyString(component["buildVersion"])),
			}
			if opts.expectedComponentVersion == "" {
				opts.expectedComponentVersion = firstNonBlankString(anyString(component["version"]), anyString(component["renderVersion"]), anyString(component["buildVersion"]))
			}
		} else if expectedComponentID == "" {
			expectedComponentName = opts.expectedComponent
		}
	}
	status := "passed"
	issues := []string{}
	if expectedComponentID != "" && !containsString(actualIDs, expectedComponentID) {
		status = injectionStaleStatus(opts.stalePolicy)
		issues = append(issues, "stale_component_reference: customPage comId does not match expected pagecomponent id")
	}
	if expectedComponentName != "" && !containsString(actualNames, expectedComponentName) {
		status = injectionStaleStatus(opts.stalePolicy)
		issues = append(issues, "stale_component_reference: customPage pageContent does not reference expected component name")
	}
	if expectedComponentID == "" && expectedComponentName == "" && opts.expectedComponent != "" && !containsString(actualIDs, opts.expectedComponent) && !containsString(actualNames, opts.expectedComponent) {
		status = injectionStaleStatus(opts.stalePolicy)
		issues = append(issues, "stale_component_reference: customPage pageContent does not reference expected component")
	}
	if opts.expectedComponentVersion != "" && !containsString(actualVersions, opts.expectedComponentVersion) && !snapshotContainsVersion(opts.snapshot, opts.expectedComponentVersion) {
		status = injectionStaleStatus(opts.stalePolicy)
		issues = append(issues, "stale_component_reference: customPage or runtime snapshot does not match expected pagecomponent version")
	}
	if status != "passed" {
		// Stale component references are reported before runtime snapshot issues so a good
		// CRM render cannot hide an outdated customPage binding.
	} else if len(opts.snapshot) > 0 {
		if v, ok := opts.snapshot["hasElement"].(bool); ok && !v {
			status = "custom_page_missing"
			issues = append(issues, "CRM injection route did not render target component element")
		} else if v, ok := opts.snapshot["hasIframe"].(bool); ok && !v {
			if hasContent, _ := opts.snapshot["hasContent"].(bool); !hasContent {
				status = "component_not_mounted"
				issues = append(issues, "component element exists but no iframe or mounted content was detected")
			} else {
				status = "iframe_missing"
				issues = append(issues, "component mounted but expected iframe was not detected")
			}
		}
	}
	return printJSON(stdout, map[string]any{
		"status":                    status,
		"pageApi":                   pageAPI,
		"customPage":                customPageSummary(page),
		"actualComponentIds":        actualIDs,
		"actualComponents":          actualNames,
		"actualVersions":            actualVersions,
		"expectedComponent":         opts.expectedComponent,
		"expectedComponentId":       expectedComponentID,
		"expectedComponentName":     expectedComponentName,
		"expectedComponentVersion":  opts.expectedComponentVersion,
		"resolvedExpectedComponent": resolvedExpected,
		"stalePolicy":               opts.stalePolicy,
		"issues":                    issues,
		"snapshot":                  opts.snapshot,
	})
}

type injectionVerifyOptions struct {
	expectedComponent        string
	expectedComponentID      string
	expectedComponentVersion string
	stalePolicy              string
	snapshot                 map[string]any
}

func parseInjectionVerifyOptions(args []string) (injectionVerifyOptions, error) {
	opts := injectionVerifyOptions{stalePolicy: "failure"}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--expected-component":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("--expected-component requires a value")
			}
			opts.expectedComponent = args[i+1]
			i++
		case "--expected-component-id":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("--expected-component-id requires a value")
			}
			opts.expectedComponentID = args[i+1]
			i++
		case "--expected-component-version", "--expected-version":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("%s requires a value", args[i])
			}
			opts.expectedComponentVersion = args[i+1]
			i++
		case "--stale-policy":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("--stale-policy requires warning or failure")
			}
			policy := strings.ToLower(strings.TrimSpace(args[i+1]))
			if policy != "warning" && policy != "failure" {
				return opts, fmt.Errorf("--stale-policy requires warning or failure")
			}
			opts.stalePolicy = policy
			i++
		case "--snapshot":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("--snapshot requires a JSON value or @file")
			}
			snapshot, err := customPagePayload(args[i+1], "cloudcc verify injectionPage --snapshot")
			if err != nil {
				return opts, err
			}
			opts.snapshot = snapshot
			i++
		}
	}
	return opts, nil
}

func injectionStaleStatus(policy string) string {
	if strings.EqualFold(policy, "warning") {
		return "warning"
	}
	return "stale_component_reference"
}

func snapshotContainsVersion(snapshot map[string]any, expected string) bool {
	expected = strings.TrimSpace(expected)
	if expected == "" || len(snapshot) == 0 {
		return false
	}
	for _, key := range []string{"componentVersion", "expectedVersion", "version", "buildVersion", "scriptVersion"} {
		if strings.TrimSpace(fmt.Sprint(snapshot[key])) == expected {
			return true
		}
	}
	for _, key := range []string{"scriptUrl", "script", "bundleUrl"} {
		if strings.Contains(fmt.Sprint(snapshot[key]), expected) {
			return true
		}
	}
	for _, key := range []string{"scriptUrls", "scripts", "resources"} {
		for _, item := range customPageObjectList(snapshot[key]) {
			if strings.Contains(fmt.Sprint(item), expected) {
				return true
			}
		}
		if items, _ := snapshot[key].([]any); items != nil {
			for _, item := range items {
				if strings.Contains(fmt.Sprint(item), expected) {
					return true
				}
			}
		}
	}
	return false
}

func resolvePageComponentForVerify(projectPath string, input string) (map[string]any, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("pagecomponent is required")
	}
	if strings.HasPrefix(input, "component-") {
		if data, err := pageComponentDetailFromList(projectPath, input); err == nil {
			return data, nil
		}
	}
	if data, err := pageComponentDetailByID(projectPath, input); err == nil {
		return data, nil
	}
	return pageComponentDetailFromList(projectPath, input)
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func customPageGet(args []string, stdout io.Writer, cwd string) error {
	projectPath := firstArg(args, cwd)
	pageAPI := ""
	if len(args) > 1 {
		pageAPI = strings.TrimSpace(args[1])
	}
	cfg, err := config.Load(projectPath)
	if err != nil {
		return err
	}
	body := map[string]any{
		"pageNo":   1,
		"pageSize": 2000,
		"condition": map[string]any{
			"pageLabel": "",
			"pageApi":   pageAPI,
		},
	}
	var res map[string]any
	if err := customPagePost(cfg, "/custom/pc/1.0/post/pageCustomPage", body, &res); err != nil {
		return err
	}
	if err := requireCloudCCSuccess(res, "Get CustomPage List Failed"); err != nil {
		return err
	}
	return printJSON(stdout, customPageListOutput(res))
}

func customPageDetail(args []string, stdout io.Writer, cwd string) error {
	if len(args) < 2 {
		return fmt.Errorf("cloudcc detail customPage <projectPath> <pageApi|id>")
	}
	data, err := customPageDetailData(args[0], args[1])
	if err != nil {
		return err
	}
	return printJSON(stdout, customPageSummary(data))
}

func customPageCreate(args []string, stdout io.Writer, cwd string) error {
	if len(args) < 2 {
		return fmt.Errorf("cloudcc create customPage <projectPath> <payload|@file>")
	}
	payload, err := customPagePayload(args[1], "cloudcc create customPage")
	if err != nil {
		return err
	}
	if err := validateCustomPagePayload(payload); err != nil {
		return err
	}
	return customPageSave(args[0], "", payload, stdout)
}

func customPageUpdate(args []string, stdout io.Writer, cwd string) error {
	if len(args) < 3 {
		return fmt.Errorf("cloudcc update customPage <projectPath> <pageApi|id> <payload|@file>")
	}
	payload, err := customPagePayload(args[2], "cloudcc update customPage")
	if err != nil {
		return err
	}
	if err := validateCustomPagePayload(payload); err != nil {
		return err
	}
	current, err := customPageDetailData(args[0], args[1])
	if err != nil {
		return err
	}
	merged, err := mergeCustomPageUpdatePayload(current, payload)
	if err != nil {
		return err
	}
	if err := validateCustomPagePayload(merged); err != nil {
		return err
	}
	return customPageSave(args[0], args[1], merged, stdout)
}

func customPageDelete(args []string, stdout io.Writer, cwd string) error {
	if len(args) < 2 {
		return fmt.Errorf("cloudcc delete customPage <projectPath> <pageApi|id>")
	}
	cfg, err := config.Load(args[0])
	if err != nil {
		return err
	}
	var res map[string]any
	body := customPageIdentifierBody(args[1])
	if err := customPagePost(cfg, "/custom/pc/1.0/post/deleteCustomPage", body, &res); err != nil {
		return err
	}
	if err := requireCloudCCSuccess(res, "Delete CustomPage Failed"); err != nil {
		return err
	}
	return printJSON(stdout, map[string]any{"status": "deleted", "target": args[1], "response": res})
}

func customPageSave(projectPath string, identifier string, payload map[string]any, stdout io.Writer) error {
	cfg, err := config.Load(projectPath)
	if err != nil {
		return err
	}
	var res map[string]any
	if err := customPageSavePost(cfg, payload, &res); err != nil {
		return err
	}
	if err := requireCloudCCSuccess(res, "Save CustomPage Failed"); err != nil {
		return err
	}
	pageAPI := strings.TrimSpace(fmt.Sprint(firstAny(payload["pageApi"], identifier)))
	current := map[string]any{}
	if pageAPI != "" {
		if data, err := customPageDetailData(projectPath, pageAPI); err == nil {
			current = customPageSummary(data)
		}
	}
	return printJSON(stdout, map[string]any{
		"status":   "updated",
		"pageApi":  pageAPI,
		"response": res,
		"current":  current,
	})
}

func mergeCustomPageUpdatePayload(current map[string]any, patch map[string]any) (map[string]any, error) {
	currentID := anyString(firstAny(current["id"], current["customPageId"]))
	patchID := anyString(patch["id"])
	if currentID != "" && patchID != "" && patchID != currentID {
		return nil, fmt.Errorf("customPage update id %q does not match current id %q", patchID, currentID)
	}
	currentPageAPI := anyString(firstAny(current["pageApi"], current["apiName"], current["apiname"]))
	patchPageAPI := anyString(patch["pageApi"])
	if currentPageAPI != "" && patchPageAPI != "" && patchPageAPI != currentPageAPI {
		return nil, fmt.Errorf("customPage update pageApi %q does not match current pageApi %q", patchPageAPI, currentPageAPI)
	}

	merged := map[string]any{}
	for _, key := range []string{
		"id", "pageLabel", "pageApi", "pageContent", "canvasStyleData", "compList",
		"orgId", "renderVersion", "version", "isTemplate", "apiName", "createBy", "createDate",
	} {
		if value, exists := current[key]; exists && value != nil {
			merged[key] = value
		}
	}
	for key, value := range patch {
		merged[key] = value
	}
	if currentID != "" {
		merged["id"] = currentID
	}
	if currentPageAPI != "" {
		merged["pageApi"] = currentPageAPI
	}
	return merged, nil
}

func customPageDetailData(projectPath string, identifier string) (map[string]any, error) {
	cfg, err := config.Load(projectPath)
	if err != nil {
		return nil, err
	}
	var res map[string]any
	body := customPageIdentifierBody(identifier)
	if err := customPagePost(cfg, "/custom/pc/1.0/post/detailCustomPage", body, &res); err != nil {
		return nil, err
	}
	if err := requireCloudCCSuccess(res, "Get CustomPage Details Failed"); err != nil {
		return nil, err
	}
	data, _ := res["data"].(map[string]any)
	if data == nil {
		return nil, fmt.Errorf("Get CustomPage Details Failed: empty response data")
	}
	return data, nil
}

func customPageIdentifierBody(identifier string) map[string]any {
	identifier = strings.TrimSpace(identifier)
	if isCloudCCObjectID(identifier) {
		return map[string]any{"id": identifier}
	}
	return map[string]any{"pageApi": identifier}
}

func isCloudCCObjectID(identifier string) bool {
	if len(identifier) != 24 {
		return false
	}
	for _, ch := range identifier {
		if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')) {
			return false
		}
	}
	return true
}

func customPagePost(cfg config.Config, apiPath string, body map[string]any, out any) error {
	head := customPageDevconsoleHeader(cfg)
	envelope := map[string]any{"head": head, "body": body}
	return httpclient.New().PostRaw(strings.TrimRight(baseURL(cfg), "/")+pageComponentDevDispatch(cfg)+apiPath, envelope, nil, out)
}

func customPageSavePost(cfg config.Config, payload map[string]any, out any) error {
	wirePayload, err := customPageWirePayload(payload)
	if err != nil {
		return err
	}
	return customPagePost(cfg, "/custom/pc/1.0/post/insertCustomPage", wirePayload, out)
}

func customPageWirePayload(payload map[string]any) (map[string]any, error) {
	wire := make(map[string]any, len(payload))
	for key, value := range payload {
		wire[key] = value
	}
	if isBlankAny(wire["pageApi"]) {
		return nil, fmt.Errorf("customPage pageApi is required")
	}
	if isBlankAny(wire["pageLabel"]) {
		return nil, fmt.Errorf("customPage pageLabel is required")
	}
	for _, field := range []string{"pageContent", "canvasStyleData"} {
		value, exists := wire[field]
		if !exists || value == nil {
			continue
		}
		text, err := customPageJSONString(value, field)
		if err != nil {
			return nil, err
		}
		wire[field] = text
	}
	if value, exists := wire["compList"]; exists {
		items, err := customPageObjectListStrict(value, "compList")
		if err != nil {
			return nil, err
		}
		wire["compList"] = mapsToAnyList(items)
	}
	return wire, nil
}

func customPageJSONString(value any, field string) (string, error) {
	if text, ok := value.(string); ok {
		text = strings.TrimSpace(text)
		if text == "" {
			return "", nil
		}
		var decoded any
		if err := json.Unmarshal([]byte(text), &decoded); err != nil {
			return "", fmt.Errorf("customPage %s must contain valid JSON: %w", field, err)
		}
		value = decoded
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("customPage %s could not be encoded as JSON: %w", field, err)
	}
	return string(encoded), nil
}

func customPageDevconsoleHeader(cfg config.Config) map[string]any {
	return map[string]any{
		"appType":     "lightning-devconsole",
		"appVersion":  "0.0.1",
		"accessToken": firstAny(config.String(cfg, "accessToken"), config.String(cfg, "pluginToken")),
		"source":      "lightning-devconsole",
		"version":     firstAny(config.String(cfg, "version"), "public"),
	}
}

func customPagePayload(input string, label string) (map[string]any, error) {
	payload, err := jsonx.ParseEncodedObject(input, label)
	if err == nil {
		return payload, nil
	}
	var raw map[string]any
	if jsonErr := json.Unmarshal([]byte(input), &raw); jsonErr == nil && raw != nil {
		return raw, nil
	}
	return nil, err
}

func validateCustomPagePayload(payload map[string]any) error {
	compList, err := customPageObjectListStrict(payload["compList"], "compList")
	if err != nil {
		return err
	}
	pageContent, err := customPageObjectListStrict(payload["pageContent"], "pageContent")
	if err != nil {
		return err
	}
	compIDs := map[string]bool{}
	for _, item := range compList {
		if isBlankAny(item["id"]) {
			return fmt.Errorf("customPage compList entries must include id")
		}
		if isBlankAny(item["compUniName"]) {
			return fmt.Errorf("customPage compList entries must include compUniName; compName alone is not accepted")
		}
		compIDs[anyString(item["id"])] = true
	}
	for i, item := range pageContent {
		comID := anyString(item["comId"])
		componentInfo, _ := item["componentInfo"].(map[string]any)
		componentInfoID := ""
		if componentInfo != nil {
			componentInfoID = anyString(componentInfo["id"])
		}
		if comID == "" && componentInfoID != "" {
			return fmt.Errorf("customPage pageContent[%d] with componentInfo.id must include matching comId", i)
		}
		if comID != "" && componentInfoID != "" && comID != componentInfoID {
			return fmt.Errorf("customPage pageContent[%d] comId %q must match componentInfo.id %q", i, comID, componentInfoID)
		}
		if comID != "" {
			if len(compIDs) == 0 {
				return fmt.Errorf("customPage compList is required when pageContent references comId %q", comID)
			}
			if !compIDs[comID] {
				return fmt.Errorf("customPage pageContent comId %q is not present in compList ids", comID)
			}
		}
	}
	return nil
}

func customPageObjectListStrict(raw any, field string) ([]map[string]any, error) {
	switch v := raw.(type) {
	case nil:
		return nil, nil
	case []map[string]any:
		return v, nil
	case []any:
		out := make([]map[string]any, 0, len(v))
		for i, item := range v {
			m, _ := item.(map[string]any)
			if m == nil {
				return nil, fmt.Errorf("customPage %s[%d] must be an object", field, i)
			}
			out = append(out, m)
		}
		return out, nil
	case map[string]any:
		return []map[string]any{v}, nil
	case string:
		text := strings.TrimSpace(v)
		if text == "" {
			return nil, nil
		}
		var arr []map[string]any
		if err := json.Unmarshal([]byte(text), &arr); err == nil {
			return arr, nil
		}
		var anyArr []any
		if err := json.Unmarshal([]byte(text), &anyArr); err == nil {
			return customPageObjectListStrict(anyArr, field)
		}
		var obj map[string]any
		if err := json.Unmarshal([]byte(text), &obj); err == nil {
			return []map[string]any{obj}, nil
		}
		return nil, fmt.Errorf("customPage %s must be a JSON object or array when supplied as a string", field)
	default:
		return nil, fmt.Errorf("customPage %s must be an object, array, or JSON string", field)
	}
}

func isBlankAny(v any) bool {
	return anyString(v) == ""
}

func anyString(v any) string {
	text := strings.TrimSpace(fmt.Sprint(v))
	if text == "" || text == "<nil>" {
		return ""
	}
	return text
}

func customPageListOutput(res map[string]any) []map[string]any {
	items := responseList(res)
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, customPageSummary(item))
	}
	return out
}

func customPageSummary(data map[string]any) map[string]any {
	return map[string]any{
		"id":              firstAny(data["id"], data["customPageId"]),
		"pageLabel":       firstAny(data["pageLabel"], data["label"], data["name"]),
		"pageApi":         firstAny(data["pageApi"], data["apiName"], data["apiname"]),
		"renderVersion":   firstAny(data["renderVersion"], data["version"]),
		"componentRefs":   customPageComponentRefs(data),
		"canvasStyleData": firstAny(data["canvasStyleData"], nil),
	}
}

func customPageComponentRefs(data map[string]any) []map[string]any {
	content := customPageObjectList(data["pageContent"])
	refs := make([]map[string]any, 0, len(content))
	for _, item := range content {
		propObj, _ := item["propObj"].(map[string]any)
		ref := map[string]any{
			"name":         firstAny(item["name"], item["component"], item["compUniName"]),
			"comId":        firstAny(item["comId"], item["id"]),
			"embedded":     firstAny(item["embedded"], false),
			"workspaceUrl": "",
		}
		if version := anyString(firstAny(item["version"], item["renderVersion"], nil)); version != "" {
			ref["version"] = version
		}
		componentInfo, _ := item["componentInfo"].(map[string]any)
		if componentInfo != nil {
			if version := anyString(firstAny(ref["version"], componentInfo["version"], componentInfo["renderVersion"], componentInfo["buildVersion"], "")); version != "" {
				ref["version"] = version
			}
		}
		if propObj != nil {
			ref["workspaceUrl"] = firstAny(propObj["workspaceUrl"], propObj["url"], "")
		}
		refs = append(refs, ref)
	}
	return refs
}

func responseList(res map[string]any) []map[string]any {
	data, _ := res["data"].(map[string]any)
	if data == nil {
		return nil
	}
	return customPageObjectList(firstAny(data["list"], data["records"], data["data"]))
}

func customPageObjectList(raw any) []map[string]any {
	switch v := raw.(type) {
	case []map[string]any:
		return v
	case []any:
		out := make([]map[string]any, 0, len(v))
		for _, item := range v {
			if m, _ := item.(map[string]any); m != nil {
				out = append(out, m)
			}
		}
		return out
	case map[string]any:
		return []map[string]any{v}
	case string:
		text := strings.TrimSpace(v)
		if text == "" {
			return nil
		}
		var arr []map[string]any
		if err := json.Unmarshal([]byte(text), &arr); err == nil {
			return arr
		}
		var anyArr []any
		if err := json.Unmarshal([]byte(text), &anyArr); err == nil {
			return customPageObjectList(anyArr)
		}
		var obj map[string]any
		if err := json.Unmarshal([]byte(text), &obj); err == nil {
			return []map[string]any{obj}
		}
	}
	return nil
}

func requireCloudCCSuccess(res map[string]any, label string) error {
	code := strings.TrimSpace(fmt.Sprint(firstAny(res["returnCode"], res["code"], "")))
	if code != "" {
		if code == "200" || code == "1" || strings.EqualFold(code, "OK") || strings.HasSuffix(strings.ToUpper(code), "OK") {
			return nil
		}
		return cloudCCFailure(label, res, firstAny(res["returnInfo"], res["msg"], "unknown error"))
	}
	if ok, exists := res["result"].(bool); exists {
		if ok {
			return nil
		}
		return cloudCCFailure(label, res, firstAny(res["returnInfo"], res["msg"], "unknown error"))
	}
	if status := strings.TrimSpace(fmt.Sprint(firstAny(res["status"], ""))); status != "" && status != "200" {
		return cloudCCFailure(label, res, firstAny(res["returnInfo"], res["msg"], "unknown error"))
	}
	if res == nil {
		return fmt.Errorf("%s: empty response", label)
	}
	if _, hasError := res["error"]; hasError {
		return cloudCCFailure(label, res, firstAny(res["error"], res["message"], "unknown error"))
	}
	if _, hasMessage := res["message"]; hasMessage && len(res) <= 2 {
		return cloudCCFailure(label, res, firstAny(res["message"], "unknown error"))
	}
	if len(res) == 0 {
		return fmt.Errorf("%s: empty response", label)
	}
	if _, hasData := res["data"]; hasData {
		return nil
	}
	return nil
}

func cloudCCFailure(label string, res map[string]any, message any) error {
	return fmt.Errorf("%s: %v; responseBody=%s", label, message, cloudCCResponseBody(res))
}

func cloudCCResponseBody(res map[string]any) string {
	if res == nil {
		return "null"
	}
	b, err := json.Marshal(res)
	if err != nil {
		return fmt.Sprint(res)
	}
	return string(b)
}
