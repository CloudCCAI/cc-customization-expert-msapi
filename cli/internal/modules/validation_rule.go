package modules

import (
	"fmt"
	"io"
	"strings"

	"cloudcc-customization-expert-go/internal/config"
	"cloudcc-customization-expert-go/internal/jsonx"
)

func handleValidationRule(action string, args []string, stdout io.Writer, cwd string) error {
	switch action {
	case "get":
		return validationRuleGet(args, stdout, cwd)
	case "create":
		return validationRuleCreate(args, stdout, cwd)
	case "delete":
		return validationRuleDelete(args, stdout, cwd)
	default:
		if ep, ok := genericEndpoints["validationRule"][action]; ok {
			return callGeneric(ep, action, "validationRule", args, stdout, cwd)
		}
		return fmt.Errorf("unsupported validationRule action: %s", action)
	}
}

func validationRuleGet(args []string, stdout io.Writer, cwd string) error {
	if len(args) < 2 {
		return fmt.Errorf("cloudcc get validationRule <projectPath> <objectPrefix>")
	}
	projectPath := firstArg(args, cwd)
	prefix := strings.TrimSpace(args[1])
	if prefix == "" {
		return fmt.Errorf("objectPrefix is required")
	}
	return postValidationRule(projectPath, "/api/validateRule/queryByPrefix", map[string]any{
		"prefix":       prefix,
		"objectPrefix": prefix,
	}, stdout)
}

func validationRuleCreate(args []string, stdout io.Writer, cwd string) error {
	if len(args) < 2 {
		return fmt.Errorf("cloudcc create validationRule <projectPath> <objectPrefix> [ruleName] [ruleContent] [errorMessage]")
	}
	projectPath := firstArg(args, cwd)
	bodyArg := strings.TrimSpace(args[1])
	if len(args) == 2 {
		body, err := jsonx.ParseEncodedObject(bodyArg, "create validationRule")
		if err == nil {
			return postValidationRule(projectPath, "/api/validateRule/save", body, stdout)
		}
		return fmt.Errorf("interactive validationRule creation is not implemented in the Go CLI; provide <ruleName> <ruleContent> <errorMessage> or @body.json")
	}
	if strings.HasPrefix(bodyArg, "@") || strings.HasPrefix(bodyArg, "{") {
		body, err := jsonx.ParseEncodedObject(args[1], "create validationRule")
		if err != nil {
			return err
		}
		return postValidationRule(projectPath, "/api/validateRule/save", body, stdout)
	}
	if len(args) < 5 {
		return fmt.Errorf("cloudcc create validationRule <projectPath> <objectPrefix> <ruleName> <ruleContent> <errorMessage>")
	}
	prefix := strings.TrimSpace(args[1])
	name := strings.TrimSpace(args[2])
	ruleContent := strings.TrimSpace(args[3])
	errorMessage := strings.TrimSpace(args[4])
	if prefix == "" || name == "" || ruleContent == "" || errorMessage == "" {
		return fmt.Errorf("objectPrefix, ruleName, ruleContent, and errorMessage are required")
	}
	body := map[string]any{
		"objid":        prefix,
		"prefix":       prefix,
		"objectPrefix": prefix,
		"validate": map[string]any{
			"name":         name,
			"ruleName":     name,
			"functionCode": ruleContent,
			"ruleContent":  ruleContent,
			"errorMessage": errorMessage,
			"isactive":     "false",
			"objId":        prefix,
		},
	}
	return postValidationRule(projectPath, "/api/validateRule/save", body, stdout)
}

func validationRuleDelete(args []string, stdout io.Writer, cwd string) error {
	if len(args) < 2 {
		return fmt.Errorf("cloudcc delete validationRule <projectPath> <ruleId>")
	}
	projectPath := firstArg(args, cwd)
	id := strings.TrimSpace(args[1])
	if id == "" {
		return fmt.Errorf("ruleId is required")
	}
	return postValidationRule(projectPath, "/api/validateRule/delete", map[string]any{
		"id":               id,
		"validationRuleId": id,
	}, stdout)
}

func postValidationRule(projectPath string, endpoint string, body map[string]any, stdout io.Writer) error {
	cfg, err := config.Load(projectPath)
	if err != nil {
		return err
	}
	return postClass(stdout, cfg, "setup", endpoint, body)
}
