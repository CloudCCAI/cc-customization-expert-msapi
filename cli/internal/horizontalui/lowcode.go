package horizontalui

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"

	"cloudcc-customization-expert-go/internal/config"
	"cloudcc-customization-expert-go/internal/horizontal"
	"cloudcc-customization-expert-go/internal/jsonx"
)

type endpoint struct {
	path string
}

var endpoints = map[string]map[string]endpoint{
	"object": {
		"get":     {path: "/api/object/getObjectList"},
		"getList": {path: "/api/object/getObjectList"},
		"detail":  {path: "/api/object/getObjectInfo"},
	},
	"application": {
		"get":     {path: "/api/application/getApplicationList"},
		"getList": {path: "/api/application/getApplicationList"},
		"detail":  {path: "/api/application/getApplicationTab"},
	},
	"recordType": {
		"get":     {path: "/api/object/objectInfo/getRecordType"},
		"getList": {path: "/api/object/objectInfo/getRecordType"},
	},
	"view": {
		"get":      {path: "/api/view/list/getViewList"},
		"getList":  {path: "/api/view/list/getViewList"},
		"detail":   {path: "/api/view/getViewInfo"},
		"editInfo": {path: "/api/view/getViewInfo"},
		"create":   {path: "/api/view/saveView"},
		"save":     {path: "/api/view/saveView"},
		"update":   {path: "/api/view/saveView"},
		"editSave": {path: "/api/view/saveView"},
		"delete":   {path: "/api/view/deleteView"},
	},
	"pagelayout": {
		"detail": {path: "/api/object/form/setup/get"},
		"save":   {path: "/api/object/form/setup/save"},
		"update": {path: "/api/object/form/setup/save"},
	},
}

var aliases = map[string]string{
	"field": "fields", "fieldId": "fields",
	"pageLayout": "pagelayout", "layout": "pagelayout",
	"record-type": "recordType", "record-type-list": "recordType",
	"object-view": "view", "object-views": "view",
	"apiRegister": "apiRegistrar", "api-registrar": "apiRegistrar", "api-register": "apiRegistrar",
}

// HandleLowCode invokes only main-app contracts that are registered as public
// JSON resources in the horizontal source tree. Missing entries fail closed;
// they never fall back to Lightning setup-svc routes.
func HandleLowCode(action string, resource string, args []string, stdout io.Writer, cwd string) error {
	resource = normalizeResource(resource)
	byAction, ok := endpoints[resource]
	if !ok {
		return unsupported(action, resource)
	}
	ep, ok := byAction[action]
	if !ok {
		return unsupported(action, resource)
	}
	projectPath := cwd
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		projectPath = args[0]
	}
	rest := []string{}
	if len(args) > 1 {
		rest = args[1:]
	}
	body, query, err := request(action, resource, rest)
	if err != nil {
		return err
	}
	cfg, err := config.Load(projectPath)
	if err != nil {
		return err
	}
	client, err := horizontal.New(config.String(cfg, "mainAppUrl"))
	if err != nil {
		return err
	}
	session, err := client.AcquireSession(context.Background(), projectPath, cfg)
	if err != nil {
		return err
	}
	if resource == "object" && action == "detail" {
		selector := firstText(body["id"], body["objid"])
		resolved, resolveErr := resolveObjectDetailSelector(client, session.Binding, selector)
		if resolveErr != nil {
			return resolveErr
		}
		body["id"], body["objid"] = resolved, resolved
	}
	result, err := client.PostJSON(context.Background(), ep.path, session.Binding, query, body)
	if err != nil {
		return err
	}
	if err := rejectExplicitFailure(action, resource, result); err != nil {
		return err
	}
	if action == "detail" && (resource == "object" || resource == "application") && result["data"] == nil {
		return fmt.Errorf("horizontal UIAPI detail %s returned no data", resource)
	}
	if isMutation(action, resource) {
		if err := requireSuccess(action, resource, result); err != nil {
			return err
		}
		readback, err := verifyMutation(client, session.Binding, action, resource, body, result)
		if err != nil {
			return err
		}
		return writeJSON(stdout, map[string]any{
			"result":   result,
			"readback": readback,
			"verified": true,
		})
	}
	return writeJSON(stdout, result)
}

func resolveObjectDetailSelector(client *horizontal.Client, binding string, selector string) (string, error) {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return "", fmt.Errorf("horizontal UIAPI object detail requires an object id or API name")
	}
	list, err := client.PostJSON(context.Background(), "/api/object/getObjectList", binding, nil, map[string]any{})
	if err != nil {
		return "", fmt.Errorf("horizontal UIAPI object selector resolution failed: %w", err)
	}
	if err := rejectExplicitFailure("getList", "object", list); err != nil {
		return "", err
	}
	items, _ := list["data"].([]any)
	for _, item := range items {
		mapped, _ := item.(map[string]any)
		id := firstText(mapped["id"], mapped["objid"], mapped["objectId"])
		apiName := firstText(mapped["objectapi"], mapped["objectApi"], mapped["apiName"])
		if strings.EqualFold(selector, id) || (apiName != "" && strings.EqualFold(selector, apiName)) {
			return id, nil
		}
	}
	return selector, nil
}

func normalizeResource(resource string) string {
	resource = strings.TrimSpace(resource)
	if canonical, ok := aliases[resource]; ok {
		return canonical
	}
	return resource
}

func unsupported(action string, resource string) error {
	if resource == "apiRegistrar" {
		return fmt.Errorf("horizontal UIAPI does not expose an API-registrar management endpoint; use the MSAPI provider")
	}
	return fmt.Errorf("cloudcc %s %s is not supported by the horizontal UIAPI adapter", action, resource)
}

func isMutation(action string, resource string) bool {
	switch resource {
	case "view":
		switch action {
		case "create", "save", "update", "editSave", "delete":
			return true
		}
	case "pagelayout":
		return action == "save" || action == "update"
	}
	return false
}

func requireSuccess(action string, resource string, response map[string]any) error {
	if success, ok := response["result"].(bool); ok {
		if success {
			return nil
		}
		return fmt.Errorf("horizontal UIAPI %s %s failed: %s", action, resource, responseMessage(response))
	}
	if flag := strings.ToLower(strings.TrimSpace(fmt.Sprint(response["flag"]))); flag != "" && flag != "<nil>" {
		if flag == "success" || flag == "1" || flag == "true" {
			return nil
		}
		return fmt.Errorf("horizontal UIAPI %s %s failed: %s", action, resource, responseMessage(response))
	}
	return fmt.Errorf("horizontal UIAPI %s %s returned no decoded success condition", action, resource)
}

// rejectExplicitFailure makes read operations honor the business envelope.
// Some main-app resources return HTTP 200 even when result=false or flag=failed;
// treating those responses as successful hides permission and contract errors.
func rejectExplicitFailure(action string, resource string, response map[string]any) error {
	if success, ok := response["result"].(bool); ok && !success {
		return fmt.Errorf("horizontal UIAPI %s %s failed: %s", action, resource, responseMessage(response))
	}
	if rawFlag, exists := response["flag"]; exists {
		flag := strings.ToLower(strings.TrimSpace(fmt.Sprint(rawFlag)))
		if flag != "" && flag != "<nil>" && flag != "success" && flag != "1" && flag != "true" {
			return fmt.Errorf("horizontal UIAPI %s %s failed: %s", action, resource, responseMessage(response))
		}
	}
	return nil
}

func responseMessage(response map[string]any) string {
	for _, key := range []string{"returnInfo", "message", "msg", "returnCode"} {
		if value := strings.TrimSpace(fmt.Sprint(response[key])); value != "" && value != "<nil>" {
			return value
		}
	}
	return "operation rejected"
}

func verifyMutation(client *horizontal.Client, binding string, action string, resource string, body map[string]any, response map[string]any) (map[string]any, error) {
	switch resource {
	case "view":
		viewID := firstText(body["viewId"], body["id"], nestedValue(response, "data", "id"))
		if viewID == "" {
			return nil, fmt.Errorf("horizontal UIAPI %s view succeeded but did not provide a view id for readback", action)
		}
		readback, err := client.PostJSON(context.Background(), "/api/view/getViewInfo", binding, nil, map[string]any{
			"viewId": viewID,
			"objId":  firstText(body["objId"]),
		})
		if err != nil {
			return nil, fmt.Errorf("horizontal UIAPI %s view readback failed: %w", action, err)
		}
		viewInfo := nestedMap(readback, "data", "viewInfo")
		if action == "delete" {
			if len(viewInfo) != 0 {
				return nil, fmt.Errorf("horizontal UIAPI delete view was not confirmed by authoritative readback")
			}
			return readback, nil
		}
		if firstText(viewInfo["id"]) != viewID {
			return nil, fmt.Errorf("horizontal UIAPI %s view readback did not return %s", action, viewID)
		}
		return readback, nil
	case "pagelayout":
		layoutID := firstText(body["id"])
		objectAPI := firstText(body["objectApi"])
		if layoutID == "" || objectAPI == "" {
			return nil, fmt.Errorf("horizontal UIAPI %s pagelayout requires id and objectApi for readback", action)
		}
		readback, err := client.PostJSON(context.Background(), "/api/object/form/setup/get", binding,
			url.Values{"id": {layoutID}, "objectApi": {objectAPI}}, map[string]any{})
		if err != nil {
			return nil, fmt.Errorf("horizontal UIAPI %s pagelayout readback failed: %w", action, err)
		}
		if err := requireSuccess("detail", "pagelayout", readback); err != nil {
			return nil, fmt.Errorf("horizontal UIAPI %s pagelayout readback was not successful: %w", action, err)
		}
		return readback, nil
	default:
		return nil, fmt.Errorf("horizontal UIAPI mutation verification is not defined for %s", resource)
	}
}

func nestedValue(value map[string]any, keys ...string) any {
	var current any = value
	for _, key := range keys {
		mapped, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = mapped[key]
	}
	return current
}

func nestedMap(value map[string]any, keys ...string) map[string]any {
	mapped, _ := nestedValue(value, keys...).(map[string]any)
	return mapped
}

func firstText(values ...any) string {
	for _, value := range values {
		if text := strings.TrimSpace(fmt.Sprint(value)); text != "" && text != "<nil>" {
			return text
		}
	}
	return ""
}

func request(action string, resource string, args []string) (map[string]any, url.Values, error) {
	body := map[string]any{}
	query := url.Values{}
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		if parsed, err := jsonx.ParseEncodedObject(args[0], "cloudcc "+action+" "+resource); err == nil {
			body = parsed
			args = args[1:]
		}
	}
	switch resource {
	case "object":
		if action == "detail" && len(args) > 0 {
			body["id"], body["objid"] = args[0], args[0]
		} else if len(args) > 0 {
			body["searchKeyWord"] = args[0]
		}
	case "application":
		if action == "detail" && len(args) > 0 {
			body["id"] = args[0]
		}
	case "recordType":
		copyIfMissing(body, "objectApi", "objectAPI", "objApi")
		if len(args) > 0 && firstText(body["id"], body["objectApi"]) == "" {
			body["objectApi"] = args[0]
		}
		if firstText(body["id"], body["objectApi"]) == "" {
			return nil, nil, fmt.Errorf("cloudcc %s recordType requires an object API name, record id, or encoded JSON body", action)
		}
		// The legacy JSONObject controller calls getString for both keys even
		// though callers may select by either record id or object API.
		if _, exists := body["id"]; !exists {
			body["id"] = ""
		}
		if _, exists := body["objectApi"]; !exists {
			body["objectApi"] = ""
		}
	case "view":
		copyIfMissing(body, "viewId", "id", "viewid")
		copyIfMissing(body, "objId", "objid", "objectId")
		switch action {
		case "get", "getList":
			if len(args) > 0 {
				body["objId"] = args[0]
			}
		case "detail", "editInfo", "delete":
			if len(args) > 0 {
				body["viewId"] = args[0]
			}
			if len(args) > 1 {
				body["objId"] = args[1]
			}
		}
	case "pagelayout":
		copyIfMissing(body, "id", "layoutId", "layoutid")
		copyIfMissing(body, "objectApi", "objApi", "objectName")
		if action == "detail" {
			if len(args) >= 2 {
				body["objectApi"], body["id"] = args[0], args[1]
			}
			for _, key := range []string{"objectApi", "id"} {
				if value := strings.TrimSpace(fmt.Sprint(body[key])); value != "" && value != "<nil>" {
					query.Set(key, value)
				}
			}
			if query.Get("objectApi") == "" || query.Get("id") == "" {
				return nil, nil, fmt.Errorf("cloudcc detail pagelayout <projectPath> <objectApi> <layoutId> or an encoded JSON body is required")
			}
		} else if len(body) == 0 {
			return nil, nil, fmt.Errorf("cloudcc %s pagelayout requires an encoded JSON body", action)
		}
	}
	return body, query, nil
}

func copyIfMissing(body map[string]any, target string, sources ...string) {
	if firstText(body[target]) != "" {
		return
	}
	for _, source := range sources {
		if firstText(body[source]) != "" {
			body[target] = body[source]
			return
		}
	}
}

func writeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	return encoder.Encode(sanitizeOutput(value, ""))
}

func sanitizeOutput(value any, key string) any {
	if isSensitiveKey(key) {
		return "[REDACTED]"
	}
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for childKey, childValue := range typed {
			out[childKey] = sanitizeOutput(childValue, childKey)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for index, item := range typed {
			out[index] = sanitizeOutput(item, "")
		}
		return out
	case string:
		return redactBindingInURL(typed)
	default:
		return value
	}
}

func isSensitiveKey(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(key), "_", ""), "-", ""))
	switch normalized {
	case "binding", "token", "accesstoken", "refreshtoken", "password", "secret":
		return true
	default:
		return false
	}
}

func redactBindingInURL(value string) string {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return value
	}
	query := parsed.Query()
	changed := false
	for key := range query {
		if strings.EqualFold(key, "binding") {
			query.Set(key, "[REDACTED]")
			changed = true
		}
	}
	if !changed {
		return value
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}
