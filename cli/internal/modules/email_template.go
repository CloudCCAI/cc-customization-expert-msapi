package modules

import (
	"fmt"
	"io"
	"strings"

	"cloudcc-customization-expert-go/internal/config"
	"cloudcc-customization-expert-go/internal/jsonx"
)

// handleEmailTemplate is the Lightning direct-provider adapter. Destructive
// actions fail closed because this channel has no authoritative reference graph.
func handleEmailTemplate(action, resource string, args []string, stdout io.Writer, cwd string) error {
	folder := resource == "emailTemplateFolder"
	if action == "delete" {
		return fmt.Errorf("email_template_uiapi_delete_unsupported: use MSAPI impact + plan/apply so references are checked")
	}
	ep, ok := genericEndpoints[resource][action]
	if !ok {
		return fmt.Errorf("unsupported %s action: %s", resource, action)
	}
	projectPath := firstArg(args, cwd)
	rest := argsAfterProject(args)
	cfg, err := config.Load(projectPath)
	if err != nil {
		return err
	}
	if action == "get" {
		body := map[string]any{"pageNum": 1, "pageSize": 10000, "backgroundSetting": true}
		if folder {
			body["foldertype"] = "email"
		} else {
			body["etype"] = "EMAIL"
			body["isQueryAll"] = "true"
		}
		if len(rest) > 0 {
			body["keyWord"] = strings.TrimSpace(rest[0])
		}
		return postClass(stdout, projectPath, cfg, ep.base, ep.path, body)
	}
	if action == "detail" {
		if len(rest) != 1 || strings.TrimSpace(rest[0]) == "" {
			return fmt.Errorf("%s selector is required", resource)
		}
		body := map[string]any{"id": strings.TrimSpace(rest[0])}
		if folder {
			body["foldertype"] = "email"
		}
		return postClass(stdout, projectPath, cfg, ep.base, ep.path, body)
	}
	var selector, raw string
	if action == "create" {
		if len(rest) != 1 {
			return fmt.Errorf("cloudcc create %s <projectPath> <@spec.json>", resource)
		}
		raw = rest[0]
	} else {
		if len(rest) != 2 || strings.TrimSpace(rest[0]) == "" {
			return fmt.Errorf("cloudcc %s %s <projectPath> <id> <@spec.json>", action, resource)
		}
		selector, raw = strings.TrimSpace(rest[0]), rest[1]
	}
	spec, err := jsonx.ParseEncodedObject(raw, action+" "+resource)
	if err != nil {
		return err
	}
	if selector != "" {
		var current map[string]any
		readPath := "/api/template/queryTemplateById"
		readBody := map[string]any{"id": selector}
		if folder {
			readPath = "/api/templateFolder/queryTemplateFolderById"
			readBody["foldertype"] = "email"
		}
		if err := postClassResponse(projectPath, cfg, "setup", readPath, readBody, &current); err != nil {
			return err
		}
		spec = emailTemplateMergeCurrent(current, spec, folder)
	}
	body, err := emailTemplateUIBody(spec, folder)
	if err != nil {
		return err
	}
	if selector != "" {
		body["id"] = selector
	}
	if err := postClassResponse(projectPath, cfg, ep.base, ep.path, body, &map[string]any{}); err != nil {
		return err
	}
	// The platform create response omits IDs. Query through the standard list to
	// prove visibility without inventing an ID or replaying the write.
	readPath := "/api/template/queryTemplateByFolder"
	readBody := map[string]any{"etype": "EMAIL", "isQueryAll": "true", "backgroundSetting": "true", "pageNum": 1, "pageSize": 10000}
	if folder {
		readPath = "/api/templateFolder/queryTemplateFolder"
		readBody = map[string]any{"foldertype": "email", "backgroundSetting": true, "pageNum": 1, "pageSize": 10000}
	}
	return postClass(stdout, projectPath, cfg, "setup", readPath, readBody)
}

func emailTemplateMergeCurrent(response, patch map[string]any, folder bool) map[string]any {
	current := emailTemplatePayload(response)
	base := map[string]any{}
	if folder {
		base["name"] = current["name"]
		base["accessMode"] = map[bool]string{true: "readOnly", false: "readWrite"}[fmt.Sprint(current["purview"]) == "0"]
		base["visibility"] = map[string]any{"scope": map[bool]string{true: "selected", false: "all"}[fmt.Sprint(current["viewtype"]) == "2"]}
	} else {
		for source, target := range map[string]string{"apiCode": "apiName", "name": "label", "folderid": "folder", "templateFolderId": "folder", "title": "subject", "encoding": "encoding", "remark": "description"} {
			if value, ok := current[source]; ok {
				base[target] = value
			}
		}
		base["enabled"] = strings.EqualFold(fmt.Sprint(current["is_enable"]), "true") || strings.EqualFold(fmt.Sprint(current["isEnable"]), "true")
		isText := strings.EqualFold(fmt.Sprint(current["is_text"]), "true") || strings.EqualFold(fmt.Sprint(current["isText"]), "true")
		content := current["mainbody"]
		base["body"] = map[string]any{"format": map[bool]string{true: "text", false: "html"}[isText], "content": content}
	}
	for key, value := range patch {
		base[key] = value
	}
	return base
}

func emailTemplatePayload(value map[string]any) map[string]any {
	for _, key := range []string{"data", "resultData", "returnInfo", "template", "folder"} {
		if nested, ok := value[key].(map[string]any); ok {
			return emailTemplatePayload(nested)
		}
		if list, ok := value[key].([]any); ok && len(list) > 0 {
			if nested, ok := list[0].(map[string]any); ok {
				return emailTemplatePayload(nested)
			}
		}
	}
	return value
}

func emailTemplateUIBody(spec map[string]any, folder bool) (map[string]any, error) {
	if folder {
		out := map[string]any{"foldertype": "email"}
		if v, ok := spec["name"]; ok {
			out["name"] = v
		}
		mode := strings.TrimSpace(fmt.Sprint(spec["accessMode"]))
		if mode == "" {
			mode = "readWrite"
		}
		if mode != "readOnly" && mode != "readWrite" {
			return nil, fmt.Errorf("accessMode must be readOnly or readWrite")
		}
		out["purview"] = map[bool]string{true: "0", false: "1"}[mode == "readOnly"]
		visibility, _ := spec["visibility"].(map[string]any)
		scope := strings.TrimSpace(fmt.Sprint(visibility["scope"]))
		if scope != "all" {
			return nil, fmt.Errorf("selected/owner-only folder visibility requires an advertised typed-subject contract")
		}
		out["viewtype"] = "0"
		out["accessibleuserset"] = ""
		return out, nil
	}
	body, ok := spec["body"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("body must be an object")
	}
	if _, ok := body["contentFile"]; ok {
		return nil, fmt.Errorf("UIAPI contentFile is unavailable; use MSAPI so the CLI expands and hashes the file")
	}
	format := strings.TrimSpace(fmt.Sprint(body["format"]))
	if format != "text" && format != "html" {
		return nil, fmt.Errorf("body.format must be text or html")
	}
	out := map[string]any{"etype": "EMAIL", "apiCode": spec["apiName"], "name": spec["label"], "templateFolderId": spec["folder"],
		"title": spec["subject"], "mainbody": body["content"], "isText": map[bool]string{true: "true", false: "false"}[format == "text"],
		"isEnable": "true", "encoding": "GBK", "remark": ""}
	if v, ok := spec["enabled"].(bool); ok {
		out["isEnable"] = fmt.Sprint(v)
	}
	for from, to := range map[string]string{"encoding": "encoding", "description": "remark"} {
		if v, ok := spec[from]; ok {
			out[to] = v
		}
	}
	return out, nil
}
