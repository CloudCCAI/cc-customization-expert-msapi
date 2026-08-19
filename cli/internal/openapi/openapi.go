package openapi

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"cloudcc-customization-expert-go/internal/config"
	"cloudcc-customization-expert-go/internal/httpclient"
	"cloudcc-customization-expert-go/internal/jsonx"
)

type spec struct {
	label        string
	serviceName  string
	wrapArray    bool
	successLabel string
}

var specs = map[string]spec{
	"query":     {label: "Query", serviceName: "cqueryWithRoleRight"},
	"pageQuery": {label: "Page Query", serviceName: "pageQueryWithRoleRight"},
	"create":    {label: "Create", serviceName: "insertWithRoleRight", wrapArray: true, successLabel: "Success! OpenAPI create completed."},
	"update":    {label: "Update", serviceName: "updateWithRoleRight", wrapArray: true, successLabel: "Success! OpenAPI update completed."},
	"delete":    {label: "Delete", serviceName: "deleteWithRoleRight", wrapArray: true, successLabel: "Success! OpenAPI delete completed."},
	"upsert":    {label: "Upsert", serviceName: "upsertWithRoleRight", wrapArray: true, successLabel: "Success! OpenAPI upsert completed."},
}

func Handle(action string, args []string, stdout io.Writer, stderr io.Writer, cwd string) error {
	if action == "doc" {
		return fmt.Errorf("use cloudcc doc platform/openapi introduction|devguide")
	}
	sp, ok := specs[action]
	if !ok {
		return fmt.Errorf("unsupported openapi action: %s", action)
	}
	projectPath := cwd
	if len(args) > 0 && args[0] != "" {
		projectPath = args[0]
	}
	encodedBody := ""
	if len(args) > 1 {
		encodedBody = args[1]
	}
	isMCP := len(args) > 2 && (args[2] == "true" || args[2] == "1")
	body, err := jsonx.ParseEncodedObject(encodedBody, sp.label+" OpenAPI")
	if err != nil {
		return err
	}
	cfg, err := config.Load(projectPath)
	if err != nil {
		return err
	}
	apiSvc := strings.TrimRight(first(config.String(cfg, "apiSvc"), config.String(cfg, "apisvc")), "/")
	accessToken := first(config.String(cfg, "accessToken"), config.String(cfg, "token"))
	if apiSvc == "" || accessToken == "" {
		return fmt.Errorf("OpenAPI Failed: apiSvc or accessToken is missing in resolved config")
	}
	payload := map[string]any{"serviceName": sp.serviceName}
	for k, v := range body {
		payload[k] = v
	}
	if data, ok := body["Data"]; ok {
		payload["data"] = normalizeData(data, sp.wrapArray)
		delete(payload, "Data")
	}
	if data, ok := body["data"]; ok {
		payload["data"] = normalizeData(data, sp.wrapArray)
	}
	var res map[string]any
	if err := httpclient.New().PostClass(apiSvc+"/openApi/common", payload, accessToken, &res); err != nil {
		return err
	}
	success := res["result"] == true || fmt.Sprint(res["returnCode"]) == "1"
	if !success {
		return fmt.Errorf("%s OpenAPI Failed: %s", sp.label, first(fmt.Sprint(res["returnInfo"]), fmt.Sprint(res["message"]), "Unknown error"))
	}
	if !isMCP {
		b, _ := json.Marshal(res)
		fmt.Fprintln(stdout, string(b))
		if sp.successLabel != "" {
			fmt.Fprintln(stderr)
			fmt.Fprintln(stderr, sp.successLabel)
			fmt.Fprintln(stderr)
		}
	}
	return nil
}

func normalizeData(v any, wrapArray bool) any {
	if s, ok := v.(string); ok {
		return s
	}
	value := v
	if wrapArray {
		if _, ok := v.([]any); !ok {
			value = []any{v}
		}
	}
	b, _ := json.Marshal(value)
	return string(b)
}

func first(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" && v != "<nil>" {
			return v
		}
	}
	return ""
}
