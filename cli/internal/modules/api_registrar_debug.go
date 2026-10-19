package modules

import (
	"encoding/json"
	"fmt"
	"strings"

	"cloudcc-customization-expert-go/internal/config"
	"cloudcc-customization-expert-go/internal/jsonx"
)

func validateApiRegistrarDebugCommand(action string, args []string, cwd string) error {
	projectPath := firstArg(args, cwd)
	cfg, err := config.Load(projectPath)
	if err != nil {
		return err
	}
	if err := requireLightningRemote(cfg, "API registrar runtime action"); err != nil {
		return err
	}
	return validateApiRegistrarDebugArgs(action, args)
}

func validateApiRegistrarDebugArgs(action string, args []string) error {
	if len(args) < 2 || strings.TrimSpace(args[1]) == "" {
		return fmt.Errorf("cloudcc %s apiRegistrar <projectPath> <@debug.json|encodedBodyJson>", action)
	}
	body, err := jsonx.ParseEncodedObject(args[1], action+" apiRegistrar")
	if err != nil {
		return err
	}
	apiCode, ok := stringField(body, "apiCode")
	if !ok || strings.TrimSpace(apiCode) == "" {
		return fmt.Errorf("cloudcc %s apiRegistrar: apiCode must be a nonblank string", action)
	}
	method, ok := stringField(body, "method")
	if !ok || strings.TrimSpace(method) == "" {
		return fmt.Errorf("cloudcc %s apiRegistrar: method must be GET or POST", action)
	}
	method = strings.ToUpper(strings.TrimSpace(method))
	if method != "GET" && method != "POST" {
		return fmt.Errorf("cloudcc %s apiRegistrar: method must be GET or POST", action)
	}
	contentType, ok := stringField(body, "contentType")
	if !ok || strings.TrimSpace(contentType) == "" {
		return fmt.Errorf("cloudcc %s apiRegistrar: contentType must be JSON, FORM, MULTIPART, or XML", action)
	}
	contentType = strings.ToUpper(strings.TrimSpace(contentType))
	switch contentType {
	case "JSON":
		requestBodyValue, exists := body["requestBody"]
		if !exists {
			return nil
		}
		requestBody, ok := requestBodyValue.(string)
		if !ok {
			return fmt.Errorf("cloudcc %s apiRegistrar: JSON requestBody must be a string", action)
		}
		if strings.TrimSpace(requestBody) == "" {
			return nil
		}
		if !json.Valid([]byte(requestBody)) {
			return fmt.Errorf("cloudcc %s apiRegistrar: JSON requestBody must contain valid JSON", action)
		}
		if method == "GET" {
			var object map[string]any
			if err := json.Unmarshal([]byte(requestBody), &object); err != nil || object == nil {
				return fmt.Errorf("cloudcc %s apiRegistrar: JSON array or scalar requestBody only supports POST", action)
			}
		}
		return nil
	case "FORM", "MULTIPART":
		return nil
	case "XML":
		if method != "POST" {
			return fmt.Errorf("cloudcc %s apiRegistrar: XML requests only support POST", action)
		}
		requestBody, ok := stringField(body, "requestBody")
		if !ok || strings.TrimSpace(requestBody) == "" {
			return fmt.Errorf("cloudcc %s apiRegistrar: XML requestBody must be a nonblank raw XML string", action)
		}
		return nil
	default:
		return fmt.Errorf("cloudcc %s apiRegistrar: contentType must be JSON, FORM, MULTIPART, or XML", action)
	}
}

func stringField(body map[string]any, key string) (string, bool) {
	value, ok := body[key]
	if !ok {
		return "", false
	}
	text, ok := value.(string)
	return text, ok
}
