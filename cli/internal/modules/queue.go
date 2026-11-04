package modules

import (
	"fmt"
	"io"
	"strings"

	"cloudcc-customization-expert-go/internal/config"
	"cloudcc-customization-expert-go/internal/jsonx"
)

// handleQueue adapts the friendly queue CLI model to the platform queueSetup
// contract. Lightning writes always use the standard service and read back the
// aggregate; horizontal routing fails closed before reaching this adapter.
func handleQueue(action string, args []string, stdout io.Writer, cwd string) error {
	ep, ok := genericEndpoints["queue"][action]
	if !ok {
		return fmt.Errorf("unsupported queue action: %s", action)
	}
	projectPath := firstArg(args, cwd)
	rest := argsAfterProject(args)
	cfg, err := config.Load(projectPath)
	if err != nil {
		return err
	}
	if action == "get" {
		body := map[string]any{"page": 1, "pageSize": 9999}
		if len(rest) > 0 {
			body["searchKeyWord"] = strings.TrimSpace(rest[0])
		}
		return postClass(stdout, projectPath, cfg, ep.base, ep.path, body)
	}
	if action == "capabilities" {
		return postClass(stdout, projectPath, cfg, ep.base, ep.path, map[string]any{})
	}
	if action == "candidates" {
		if len(rest) < 1 {
			return fmt.Errorf("cloudcc candidates queue <projectPath> <objects|members> [type] [keyword]")
		}
		kind := strings.TrimSpace(rest[0])
		path := "/api/queueSetup/getAvailableTabs"
		body := map[string]any{}
		if kind == "members" {
			if len(rest) < 2 {
				return fmt.Errorf("queue member candidates require user, role, or roleAndSubordinates")
			}
			codes := map[string]string{"user": "2", "role": "3", "roleAndSubordinates": "4"}
			code := codes[strings.TrimSpace(rest[1])]
			if code == "" {
				return fmt.Errorf("unsupported queue member type %q", rest[1])
			}
			path = "/api/queueSetup/getAvailableMembers"
			body["memberType"] = code
			if len(rest) > 2 {
				body["keyword"] = strings.TrimSpace(rest[2])
			}
		} else if kind != "objects" {
			return fmt.Errorf("queue candidate kind must be objects or members")
		} else if len(rest) > 1 {
			body["keyword"] = strings.TrimSpace(rest[1])
		}
		return postClass(stdout, projectPath, cfg, "setup", path, body)
	}
	if action == "detail" || action == "impact" || action == "delete" {
		if len(rest) < 1 || strings.TrimSpace(rest[0]) == "" {
			return fmt.Errorf("queue selector is required")
		}
		body := map[string]any{"id": strings.TrimSpace(rest[0])}
		if action == "impact" && (len(rest) != 2 || strings.TrimSpace(rest[1]) != "delete") {
			return fmt.Errorf("cloudcc impact queue <projectPath> <selector> delete")
		}
		if action == "delete" {
			var impact map[string]any
			if err := postClassResponse(projectPath, cfg, "setup", "/api/queueSetup/checkBeforeDeleting", body, &impact); err != nil {
				return err
			}
			if value, ok := impact["result"].(bool); ok && !value {
				return fmt.Errorf("queue_delete_owner_reference: queue still owns business records")
			}
		}
		return postClass(stdout, projectPath, cfg, ep.base, ep.path, body)
	}
	body, err := queueWriteBody(action, rest)
	if err != nil {
		return err
	}
	if action == "update" || action == "assign" {
		var current map[string]any
		if err := postClassResponse(projectPath, cfg, "setup", "/api/queueSetup/query",
			map[string]any{"id": queueString(body["id"])}, &current); err != nil {
			return err
		}
		body = queueMergeCurrent(current, body)
	}
	var response map[string]any
	if err := postClassResponse(projectPath, cfg, ep.base, ep.path, body, &response); err != nil {
		return err
	}
	id := queueString(body["id"])
	if id == "" {
		id = queueResponseID(response)
	}
	if id == "" {
		return fmt.Errorf("queue_setup_response_invalid: save response omitted queue id; readback cannot be proven")
	}
	return postClass(stdout, projectPath, cfg, "setup", "/api/queueSetup/query", map[string]any{"id": id})
}

func queueWriteBody(action string, args []string) (map[string]any, error) {
	if action == "create" {
		if len(args) != 1 {
			return nil, fmt.Errorf("cloudcc create queue <projectPath> <encodedSpecJson|@file>")
		}
		body, err := jsonx.ParseEncodedObject(args[0], "create queue")
		if err != nil {
			return nil, err
		}
		return queueCanonicalSave(body, "add", "")
	}
	if len(args) != 2 || strings.TrimSpace(args[0]) == "" {
		return nil, fmt.Errorf("cloudcc %s queue <projectPath> <selector> <encodedSpecJson|@file>", action)
	}
	body, err := jsonx.ParseEncodedObject(args[1], action+" queue")
	if err != nil {
		return nil, err
	}
	if action == "assign" {
		for _, key := range []string{"objects", "members"} {
			if instruction, ok := body[key].(map[string]any); ok && strings.ToLower(queueString(instruction["mode"])) != "replace" {
				return nil, fmt.Errorf("Lightning UIAPI queue assignment supports mode=replace only; use MSAPI for add/remove semantics")
			}
		}
	}
	return queueCanonicalSave(body, "edit", strings.TrimSpace(args[0]))
}

func queueCanonicalSave(source map[string]any, option string, id string) (map[string]any, error) {
	result := map[string]any{"option": option}
	if id != "" {
		result["id"] = id
	}
	copyKey := func(from, to string) {
		if value, ok := source[from]; ok {
			result[to] = value
		}
	}
	copyKey("apiName", "code")
	copyKey("label", "name")
	copyKey("email", "email")
	copyKey("sendEmailToMembers", "sendToMember")
	copyKey("description", "description")
	if value, ok := source["supportedObjects"]; ok {
		result["tabs"] = value
	}
	if value, ok := source["members"]; ok {
		result["members"] = value
	}
	if value, ok := source["objects"].(map[string]any); ok {
		result["tabs"] = value["values"]
	}
	if value, ok := source["members"].(map[string]any); ok {
		result["members"] = value["values"]
	}
	return result, nil
}

func queueMergeCurrent(response map[string]any, patch map[string]any) map[string]any {
	current := queuePayload(response)
	result := map[string]any{"option": "edit", "id": patch["id"]}
	aliases := map[string][]string{
		"code": {"code", "apiName"}, "name": {"name", "label"}, "email": {"email"},
		"sendToMember": {"sendToMember", "sendEmailToMembers"}, "description": {"description"},
		"tabs": {"tabs", "supportedObjects"}, "members": {"members"},
	}
	for target, keys := range aliases {
		for _, key := range keys {
			if value, ok := current[key]; ok {
				result[target] = value
				break
			}
		}
	}
	for key, value := range patch {
		if key != "option" {
			result[key] = value
		}
	}
	return result
}

func queuePayload(response map[string]any) map[string]any {
	for _, key := range []string{"data", "resultData", "queue"} {
		if nested, ok := response[key].(map[string]any); ok {
			return queuePayload(nested)
		}
	}
	return response
}

func queueResponseID(response map[string]any) string {
	for _, key := range []string{"id", "queueId"} {
		if value := queueString(response[key]); value != "" {
			return value
		}
	}
	for _, key := range []string{"data", "resultData", "queue"} {
		if nested, ok := response[key].(map[string]any); ok {
			if value := queueResponseID(nested); value != "" {
				return value
			}
		}
	}
	return ""
}
func queueString(value any) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}
