package msapi

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func handleCompactLayoutShortcut(action, project string, args []string, stdout io.Writer, cwd string) error {
	const domain = "compact-layouts"
	const base = "/metadata/v1/" + domain
	var spec map[string]any
	var object string
	var path string
	switch action {
	case "create", "update", "assign", "delete":
		if len(args) != 1 {
			return fmt.Errorf("cloudcc %s compactLayout <projectPath> <specJson|@file>", action)
		}
		var err error
		spec, err = parseObject(args[0], action+" compactLayout")
		if err != nil {
			return err
		}
		object, _ = spec["object"].(string)
	case "get", "capabilities", "assignments", "candidates", "detail", "impact":
		expected := 1
		if action == "detail" || action == "candidates" {
			expected = 2
		} else if action == "impact" {
			expected = 3
		}
		if len(args) != expected || (expected > 1 && strings.TrimSpace(args[1]) == "") ||
			(action == "impact" && args[2] != "delete") {
			return fmt.Errorf("invalid compactLayout %s arguments; run cloudcc doc compactLayout devguide", action)
		}
		object = args[0]
		switch action {
		case "get":
			path = base
		case "capabilities", "assignments", "candidates":
			path = base + "/" + action
		case "detail", "impact":
			path = base + "/" + url.PathEscape(strings.TrimSpace(args[1]))
			if action == "impact" {
				path += "/delete-impact"
			}
		}
	default:
		return fmt.Errorf("unsupported compactLayout action: %s", action)
	}
	object = strings.TrimSpace(object)
	if object == "" {
		return fmt.Errorf("compactLayout requires a nonblank object selector")
	}
	values := url.Values{"object": {object}}
	client, _, err := newClient([]string{project}, cwd)
	if err != nil {
		return err
	}
	capabilityPath := base + "/capabilities?" + values.Encode()
	if action == "capabilities" {
		return client.getJSON(stdout, capabilityPath)
	}
	capability, err := client.requestJSONMap(http.MethodGet, capabilityPath, nil)
	if err != nil {
		return fmt.Errorf("compactLayout capability discovery failed; MetadataService 1.2.20+ is required: %w", err)
	}
	supported := false
	if actions, ok := capability["actions"].([]any); ok && capability["available"] == true {
		for _, item := range actions {
			if item == action {
				supported = true
				break
			}
		}
	}
	if !supported {
		return fmt.Errorf("compactLayout %s is unavailable for object %s", action, object)
	}
	if spec != nil {
		return planShortcut(stdout, cwd, project, domain, spec, action)
	}
	if action == "candidates" {
		switch args[1] {
		case "fields", "recordTypes", "layouts":
			values.Set("kind", args[1])
		default:
			return fmt.Errorf("compactLayout candidate kind must be fields, recordTypes, or layouts")
		}
	}
	return client.getJSON(stdout, path+"?"+values.Encode())
}
