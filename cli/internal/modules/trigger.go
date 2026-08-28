package modules

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"

	"cloudcc-customization-expert-go/internal/config"
	"cloudcc-customization-expert-go/internal/httpclient"
	"cloudcc-customization-expert-go/internal/jsonx"
)

const (
	triggerListEndpoint     = "/api/triggerSetup/getTriggerByCondition"
	triggerDetailEndpoint   = "/api/trigger/newobjtrigger"
	triggerValidateEndpoint = "/api/trigger/validate"
	triggerSaveEndpoint     = "/api/triggerSetup/saveTrigger"
	triggerDeleteEndpoint   = "/api/triggerSetup/deleteTrigger"
)

func triggerList(args []string, stdout io.Writer, cwd string) error {
	projectPath := firstArg(args, cwd)
	body := triggerListRequest("")
	if len(args) > 1 && strings.TrimSpace(args[1]) != "" {
		if parsed, err := jsonx.ParseEncodedObject(args[1], "cloudcc get trigger"); err == nil {
			for key, value := range parsed {
				body[key] = value
			}
		} else {
			body["sname"] = strings.TrimSpace(args[1])
		}
	}
	cfg, err := config.Load(projectPath)
	if err != nil {
		return err
	}
	res, err := triggerRequest(cfg, triggerListEndpoint, body, "Get Trigger List Failed")
	if err != nil {
		return err
	}
	return printJSON(stdout, res)
}

func triggerDetail(args []string, stdout io.Writer, cwd string) error {
	if len(args) < 2 || strings.TrimSpace(args[1]) == "" {
		return fmt.Errorf("cloudcc detail trigger <projectPath> <id|name|apiName>")
	}
	cfg, err := config.Load(firstArg(args, cwd))
	if err != nil {
		return err
	}
	id, err := resolveTriggerID(cfg, args[1])
	if err != nil {
		return err
	}
	res, err := triggerRequest(cfg, triggerDetailEndpoint, map[string]any{"id": id}, "Get Trigger Detail Failed")
	if err != nil {
		return err
	}
	if triggerDetailRecord(res) == nil {
		return fmt.Errorf("trigger %q resolved to %q but detail response contained no tp_sys_trigger record", args[1], id)
	}
	return printJSON(stdout, res)
}

func triggerDelete(args []string, stdout io.Writer, cwd string) error {
	if len(args) < 2 || strings.TrimSpace(args[1]) == "" {
		return fmt.Errorf("cloudcc delete trigger <projectPath> <id|name|apiName>")
	}
	cfg, err := config.Load(firstArg(args, cwd))
	if err != nil {
		return err
	}
	id, err := resolveTriggerID(cfg, args[1])
	if err != nil {
		return err
	}
	res, err := triggerRequest(cfg, triggerDeleteEndpoint, map[string]any{"id": id}, "Delete Trigger Failed")
	if err != nil {
		return err
	}
	return printJSON(stdout, res)
}

func triggerSaveSpec(action string, args []string, stdout io.Writer, cwd string) error {
	if len(args) < 2 {
		return fmt.Errorf("cloudcc %s trigger <projectPath> <encodedTriggerJson|@file>", action)
	}
	projectPath := firstArg(args, cwd)
	spec, err := jsonx.ParseEncodedObject(args[1], "cloudcc "+action+" trigger")
	if err != nil {
		return err
	}
	if action == "update" && strings.TrimSpace(fmt.Sprint(spec["id"])) == "" {
		return fmt.Errorf("cloudcc update trigger requires spec.id")
	}
	if sourceFile := strings.TrimSpace(fmt.Sprint(spec["sourceFile"])); sourceFile != "" && sourceFile != "<nil>" {
		if !filepath.IsAbs(sourceFile) {
			sourceFile = filepath.Join(projectPath, sourceFile)
		}
		source, readErr := readMarkedSource(sourceFile)
		if readErr != nil {
			return readErr
		}
		spec["triggerSource"] = source
	}
	if source := strings.TrimSpace(fmt.Sprint(spec["triggerSource"])); source != "" && source != "<nil>" && !triggerSpecSourceEncoded(spec) {
		spec["triggerSource"] = encodeJavaURLDecoderComponent(source)
	}
	if _, exists := spec["folderid"]; !exists {
		spec["folderid"] = firstAny(spec["folderId"], "wgd")
	}
	spec["isactive"] = normalizeTriggerIsActive(firstAny(spec["isactive"], spec["isActive"], "true"))
	delete(spec, "sourceFile")
	delete(spec, "sourceEncoded")
	delete(spec, "triggerSourceEncoded")
	cfg, err := config.Load(projectPath)
	if err != nil {
		return err
	}
	operationEdit := strings.TrimSpace(fmt.Sprint(spec["id"])) != "" && strings.TrimSpace(fmt.Sprint(spec["id"])) != "<nil>"
	if operationEdit {
		if detail, detailErr := triggerRequest(cfg, triggerDetailEndpoint, map[string]any{"id": spec["id"]}, "Resolve Trigger Version Failed"); detailErr == nil {
			if version := highCodeRecordVersion(detail); version != "" {
				spec["version"] = version
			}
		}
	} else {
		spec["version"] = highCodeDefaultVersion
	}
	res, err := triggerRequest(cfg, triggerSaveEndpoint, spec, "Save Trigger Failed")
	if err != nil {
		return err
	}
	return printJSON(stdout, res)
}

func publishTrigger(args []string, stdout io.Writer, stderr io.Writer, cwd string) error {
	if len(args) < 1 {
		return fmt.Errorf("cloudcc publish trigger <object/name> [projectPath]")
	}
	projectPath := cwd
	if len(args) > 1 && strings.TrimSpace(args[1]) != "" && !strings.HasPrefix(args[1], "--") {
		projectPath = args[1]
	}
	namePath := args[0]
	name := filepath.Base(namePath)
	srcDir := backendResourcePath(projectPath, "triggers", namePath)
	sourceFile := filepath.Join(srcDir, name+".java")
	source, err := readMarkedSource(sourceFile)
	if err != nil {
		return err
	}
	source = strings.TrimSpace(source)
	configPath := filepath.Join(srcDir, "config.json")
	cfgContent, _ := jsonx.ReadObjectFile(configPath)
	if cfgContent == nil {
		cfgContent = map[string]any{}
	}
	cfg, err := config.Load(projectPath)
	if err != nil {
		return err
	}
	triggerID := strings.TrimSpace(fmt.Sprint(configID(cfgContent)))
	operationEdit := triggerID != "" && triggerID != "<nil>"
	var preSaveDetail map[string]any
	if operationEdit {
		if detail, detailErr := triggerRequest(cfg, triggerDetailEndpoint, map[string]any{"id": triggerID}, "Resolve Trigger Version Failed"); detailErr == nil {
			preSaveDetail = detail
		}
	}
	publishVersion := highCodePublishVersion(cfgContent, preSaveDetail, operationEdit)
	validateBody := map[string]any{
		"id":             configID(cfgContent),
		"apiname":        cfgContent["apiname"],
		"apiName":        cfgContent["apiName"],
		"isactive":       normalizeTriggerIsActive(firstAny(cfgContent["isactive"], cfgContent["isActive"], "true")),
		"targetObjectId": cfgContent["targetObjectId"],
		"triggerTime":    cfgContent["triggerTime"],
		"remark":         cfgContent["remark"],
		"name":           firstAny(cfgContent["name"], name),
		"triggerSource":  source,
		"folderid":       firstAny(cfgContent["folderid"], cfgContent["folderId"], "wgd"),
	}
	putHighCodeVersion(validateBody, publishVersion)
	remoteValidation, err := validateRemoteCustomCode(cfg, "trigger", name, triggerValidateEndpoint, validateBody)
	if err != nil {
		_ = writeJSON(stdout, map[string]any{"status": "blocked_remote_validation", "resource": "trigger", "name": name, "remoteValidation": remoteValidation})
		return err
	}
	saveBody := copyStringAnyMap(validateBody)
	saveBody["triggerSource"] = encodeJavaURLDecoderComponent(source)
	fmt.Fprintln(stderr, "Remote CloudCC trigger validation passed; posting trigger, please wait...")
	res, err := triggerRequest(cfg, triggerSaveEndpoint, saveBody, "Publish Trigger Failed")
	if err != nil {
		return err
	}
	if data, _ := res["data"].(map[string]any); data != nil {
		if id := strings.TrimSpace(fmt.Sprint(data["id"])); id != "" && id != "<nil>" {
			cfgContent["id"] = id
		}
		if apiName := strings.TrimSpace(fmt.Sprint(firstAny(data["apiname"], data["apiName"]))); apiName != "" && apiName != "<nil>" {
			cfgContent["apiname"] = apiName
		}
		savedVersion := publishVersion
		if id := strings.TrimSpace(fmt.Sprint(cfgContent["id"])); id != "" && id != "<nil>" {
			if detail, detailErr := triggerRequest(cfg, triggerDetailEndpoint, map[string]any{"id": id}, "Read Trigger Version Failed"); detailErr == nil {
				if version := highCodeRecordVersion(detail); version != "" {
					savedVersion = version
				}
			}
		}
		if savedVersion != "" {
			cfgContent["version"] = savedVersion
		}
		if writeErr := jsonx.WriteObjectFile(configPath, cfgContent); writeErr != nil {
			return writeErr
		}
	}
	return writeJSON(stdout, map[string]any{"status": "published", "resource": "trigger", "name": name, "remoteValidation": remoteValidation, "saveResponse": res})
}

func resolveTriggerID(cfg config.Config, selector string) (string, error) {
	selector = strings.TrimSpace(selector)
	listRes, err := triggerRequest(cfg, triggerListEndpoint, triggerListRequest(selector), "Resolve Trigger Failed")
	if err != nil {
		return "", err
	}
	matches := make([]map[string]any, 0)
	for _, item := range triggerListItems(listRes) {
		for _, key := range []string{"id", "name", "apiname", "apiName"} {
			if strings.EqualFold(strings.TrimSpace(fmt.Sprint(item[key])), selector) {
				matches = append(matches, item)
				break
			}
		}
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("ambiguous trigger selector %q matched %d tp_sys_trigger rows; use the unique id", selector, len(matches))
	}
	if len(matches) == 1 {
		id := strings.TrimSpace(fmt.Sprint(matches[0]["id"]))
		if id == "" || id == "<nil>" {
			return "", fmt.Errorf("trigger selector %q matched a row without id", selector)
		}
		return id, nil
	}
	// The global list filters name/API name but not id. Validate a possible direct
	// id through the canonical detail endpoint before any destructive operation.
	detail, detailErr := triggerRequest(cfg, triggerDetailEndpoint, map[string]any{"id": selector}, "Resolve Trigger ID Failed")
	if detailErr != nil {
		return "", detailErr
	}
	record := triggerDetailRecord(detail)
	if record == nil {
		return "", fmt.Errorf("trigger %q was not found in tp_sys_trigger", selector)
	}
	id := strings.TrimSpace(fmt.Sprint(firstAny(record["id"], selector)))
	if id == "" || id == "<nil>" {
		return "", fmt.Errorf("trigger %q detail response contained no id", selector)
	}
	return id, nil
}

func triggerListRequest(search string) map[string]any {
	return map[string]any{
		"shownum":  "2000",
		"showpage": "1",
		"sname":    strings.TrimSpace(search),
		"objId":    "",
		"fid":      "",
		"rptcond":  "lastmodifydate",
		"rptorder": "desc",
	}
}

func triggerRequest(cfg config.Config, endpoint string, body map[string]any, label string) (map[string]any, error) {
	var res map[string]any
	base := strings.TrimRight(config.String(cfg, "setupSvc"), "/")
	if err := httpclient.New().PostClass(base+endpoint, body, config.String(cfg, "accessToken"), &res); err != nil {
		return nil, err
	}
	if result, exists := res["result"].(bool); exists && !result {
		return nil, triggerFailure(label, res)
	}
	code := strings.TrimSpace(fmt.Sprint(firstAny(res["returnCode"], res["code"])))
	if code != "" && code != "<nil>" && code != "1" && code != "200" && code != "000-000" && !strings.Contains(code, "-000-") && !strings.EqualFold(code, "success") {
		return nil, triggerFailure(label, res)
	}
	return res, nil
}

func triggerFailure(label string, res map[string]any) error {
	raw, _ := json.Marshal(res)
	message := firstAny(res["returnInfo"], res["msg"], res["errormsg"], "unknown error")
	return fmt.Errorf("%s: %v; responseBody=%s", label, message, string(raw))
}

func triggerListItems(res map[string]any) []map[string]any {
	containers := []any{res["list"], res["data"]}
	if data, _ := res["data"].(map[string]any); data != nil {
		containers = append(containers, data["list"], data["rows"], data["records"])
	}
	for _, raw := range containers {
		list, ok := raw.([]any)
		if !ok {
			continue
		}
		items := make([]map[string]any, 0, len(list))
		for _, item := range list {
			if row, ok := item.(map[string]any); ok {
				items = append(items, row)
			}
		}
		return items
	}
	return nil
}

func triggerDetailRecord(res map[string]any) map[string]any {
	for _, raw := range []any{res["trigger"], res["data"]} {
		if row, ok := raw.(map[string]any); ok {
			if trigger, ok := row["trigger"].(map[string]any); ok {
				return trigger
			}
			if id := strings.TrimSpace(fmt.Sprint(row["id"])); id != "" && id != "<nil>" {
				return row
			}
		}
	}
	if items := triggerListItems(res); len(items) == 1 {
		return items[0]
	}
	return nil
}

func triggerSpecSourceEncoded(spec map[string]any) bool {
	for _, key := range []string{"sourceEncoded", "triggerSourceEncoded"} {
		switch value := spec[key].(type) {
		case bool:
			if value {
				return true
			}
		case string:
			if strings.EqualFold(strings.TrimSpace(value), "true") {
				return true
			}
		}
	}
	return false
}

func normalizeTriggerIsActive(value any) string {
	switch v := value.(type) {
	case bool:
		if v {
			return "true"
		}
		return "false"
	case float64:
		if v == 0 {
			return "false"
		}
		return "true"
	case int:
		if v == 0 {
			return "false"
		}
		return "true"
	}
	text := strings.TrimSpace(fmt.Sprint(value))
	switch strings.ToLower(text) {
	case "0", "false":
		return "false"
	default:
		return "true"
	}
}

func encodeJavaURLDecoderComponent(source string) string {
	return url.QueryEscape(source)
}

func looksLikeTriggerSpec(value string) bool {
	value = strings.TrimSpace(value)
	return strings.HasPrefix(value, "{") || strings.HasPrefix(value, "%7B") || strings.HasPrefix(value, "%7b") || strings.HasPrefix(value, "@")
}

func createTriggerResource(args []string, stderr io.Writer, cwd string) error {
	if len(args) == 0 {
		return fmt.Errorf("cloudcc create trigger <name> [projectPath]")
	}
	projectPath := cwd
	if len(args) > 1 && strings.TrimSpace(args[1]) != "" && !looksLikeTriggerSpec(args[1]) {
		projectPath = args[1]
	}
	return createJavaResource("triggers", "trigger", args[:1], stderr, projectPath)
}
