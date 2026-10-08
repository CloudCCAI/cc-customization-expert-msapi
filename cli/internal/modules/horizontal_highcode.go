package modules

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"cloudcc-customization-expert-go/internal/config"
	"cloudcc-customization-expert-go/internal/horizontalaction"
	"cloudcc-customization-expert-go/internal/jsonx"
)

func horizontalJavaPublish(projectPath string, cfg config.Config, resource, name, source, configPath string, local map[string]any, stdout, stderr io.Writer) error {
	apiName := name
	if resource == "timer" {
		apiName = firstNonBlankString(cleanAny(firstAny(local["apiname"], local["apiName"])), name)
		if cleanAny(firstAny(local["apiname"], local["apiName"])) == "" {
			local["apiname"] = apiName
			if err := jsonx.WriteObjectFileAtomic(configPath, local); err != nil {
				return fmt.Errorf("cannot persist stable timer apiname: %w", err)
			}
		}
	}
	form := url.Values{"m": {"save"}, "trigger.id": {cleanAny(configID(local))}, "trigger.name": {name}, "trigger.folderid": {firstNonBlankString(anyString(firstAny(local["folderid"], local["folderId"])), "wgd")}, "trigger.source": {source}}
	if resource == "timer" {
		form.Set("trigger.apiname", apiName)
	}
	path := "/ccfag.action"
	if resource == "timer" {
		path = "/ccpeak.action"
	}
	receipt, err := horizontalaction.ExecuteResolvedPublish(context.Background(), projectPath, cfg, horizontalaction.PublishRequest{Resource: resource, Path: path, Form: form, IDField: "trigger.id", Identity: horizontalaction.PublishIdentity{Name: name, APIName: apiName}})
	if err != nil {
		return err
	}
	writeResolvedID(configPath, local, &receipt, stderr)
	return printJSON(stdout, receipt)
}

func horizontalCodeRemote(action, resource string, args []string, stdout io.Writer, cwd string) (bool, error) {
	projectPath := firstArg(args, cwd)
	cfg, err := config.Load(projectPath)
	if err != nil {
		return false, err
	}
	if !config.IsHorizontal(cfg) {
		return false, nil
	}
	form := url.Values{}
	mutating := action == "delete"
	switch action {
	case "get":
		form.Set("m", "list")
		form.Set("shownum", "2000")
		form.Set("showpage", "1")
		form.Set("rptcond", "lastmodifydate")
		form.Set("rptorder", "desc")
	case "pullList":
		form.Set("m", "list")
	case "detail", "pull":
		if len(args) < 2 {
			return true, fmt.Errorf("cloudcc %s %s <projectPath> <id>", action, resource)
		}
		form.Set("m", "detail")
		form.Set("id", args[1])
	case "delete":
		if len(args) < 2 {
			return true, fmt.Errorf("cloudcc delete %s <projectPath> <id>", resource)
		}
		form.Set("m", "delete")
		form.Set("id", args[1])
	default:
		return true, fmt.Errorf("unsupported horizontal %s action: %s", resource, action)
	}
	receipt, err := horizontalaction.ExecuteForm(context.Background(), projectPath, cfg, resource, action, form, mutating)
	if err != nil {
		return true, err
	}
	return true, printJSON(stdout, receipt)
}

func horizontalTriggerPublish(projectPath string, cfg config.Config, name, source, configPath string, local map[string]any, stdout, stderr io.Writer) error {
	apiName := firstNonBlankString(cleanAny(firstAny(local["apiname"], local["apiName"])), name)
	if cleanAny(firstAny(local["apiname"], local["apiName"])) == "" {
		local["apiname"] = apiName
		if err := jsonx.WriteObjectFileAtomic(configPath, local); err != nil {
			return fmt.Errorf("cannot persist stable trigger apiname: %w", err)
		}
	}
	form := url.Values{
		"m": {"save"}, "trigger.id": {cleanAny(configID(local))}, "trigger.apiname": {apiName},
		"trigger.name": {firstNonBlankString(cleanAny(local["name"]), name)}, "trigger.triggerTime": {cleanAny(local["triggerTime"])},
		"trigger.targetObjectId": {cleanAny(local["targetObjectId"])}, "trigger.folderid": {firstNonBlankString(cleanAny(firstAny(local["folderid"], local["folderId"])), "wgd")},
		"trigger.isactive": {firstNonBlankString(cleanAny(firstAny(local["isactive"], local["isActive"])), "true")},
		"trigger.version":  {firstNonBlankString(cleanAny(local["version"]), "2")}, "trigger.remark": {cleanAny(local["remark"])},
		"trigger.triggerSource": {source},
	}
	receipt, err := horizontalaction.ExecuteResolvedPublish(context.Background(), projectPath, cfg, horizontalaction.PublishRequest{Resource: "trigger", Path: "/trigger.action", Form: form, IDField: "trigger.id", Identity: horizontalaction.PublishIdentity{Name: form.Get("trigger.name"), APIName: apiName}})
	if err != nil {
		return err
	}
	writeResolvedID(configPath, local, &receipt, stderr)
	return printJSON(stdout, receipt)
}

func handleVisualPage(action string, args []string, stdout io.Writer, stderr io.Writer, cwd string) error {
	if action == "create" {
		if len(args) < 1 {
			return fmt.Errorf("cloudcc create visualPage <name>")
		}
		mode, err := config.ProjectPlatformMode(cwd)
		if err != nil {
			return err
		}
		if mode != config.PlatformHorizontal {
			return fmt.Errorf("visualPage is a horizontal-only resource; use customPage for Lightning")
		}
		return createHorizontalVisualPage(args[0], stderr, cwd)
	}
	projectPath := cwd
	if action == "publish" {
		if len(args) < 1 {
			return fmt.Errorf("cloudcc publish visualPage <name> [projectPath]")
		}
		if len(args) > 1 {
			projectPath = args[1]
		}
	} else {
		projectPath = firstArg(args, cwd)
	}
	cfg, err := config.Load(projectPath)
	if err != nil {
		return err
	}
	if !config.IsHorizontal(cfg) {
		return fmt.Errorf("visualPage is supported only for platformMode=horizontal")
	}
	if action == "publish" {
		name := filepath.Base(args[0])
		if err := validateHorizontalAPIName("visualPage", name); err != nil {
			return err
		}
		dir := filepath.Join(projectPath, "visualpage", name)
		source, err := os.ReadFile(filepath.Join(dir, name+".jsp"))
		if err != nil {
			return err
		}
		if err := validateHorizontalVisualPage(string(source)); err != nil {
			return err
		}
		local, _ := jsonx.ReadObjectFile(filepath.Join(dir, "config.json"))
		if local == nil {
			local = map[string]any{}
		}
		form := url.Values{"m": {"save"}, "visualPage.id": {cleanAny(configID(local))}, "visualPage.label": {firstNonBlankString(cleanAny(local["label"]), name)}, "visualPage.name": {name}, "visualPage.functiontype": {firstNonBlankString(cleanAny(local["functiontype"]), "P")}, "visualPage.versiontype": {"oldsystem"}, "visualPage.folderid": {firstNonBlankString(cleanAny(local["folderid"]), "wgd")}, "visualPage.appliedmobile": {firstNonBlankString(cleanAny(local["appliedmobile"]), "false")}, "visualPage.pageSource": {string(source)}}
		configPath := filepath.Join(dir, "config.json")
		receipt, err := horizontalaction.ExecuteResolvedPublish(context.Background(), projectPath, cfg, horizontalaction.PublishRequest{Resource: "visualPage", Path: "/page.action", Form: form, IDField: "visualPage.id", Identity: horizontalaction.PublishIdentity{Name: name, Label: form.Get("visualPage.label")}})
		if err != nil {
			return err
		}
		writeResolvedID(configPath, local, &receipt, stderr)
		return printJSON(stdout, receipt)
	}
	form := url.Values{}
	mutating := action == "delete"
	switch action {
	case "get", "list":
		form.Set("m", "list")
		form.Set("shownum", "2000")
		form.Set("showpage", "1")
	case "detail", "pull":
		if len(args) < 2 {
			return fmt.Errorf("cloudcc %s visualPage <projectPath> <id>", action)
		}
		form.Set("m", "detail")
		form.Set("id", args[1])
	case "delete":
		if len(args) < 2 {
			return fmt.Errorf("cloudcc delete visualPage <projectPath> <id>")
		}
		form.Set("m", "delete")
		form.Set("id", args[1])
	case "assignProfiles":
		if len(args) < 3 {
			return fmt.Errorf("cloudcc assignProfiles visualPage <projectPath> <pageId> <profileIds>")
		}
		form.Set("vid", args[1])
		form.Set("pids", args[2])
		mutating = true
		return executeHorizontalVisualPagePermission(projectPath, cfg, "visualPageAccess", action, form, stdout)
	case "enableForProfile":
		if len(args) < 3 {
			return fmt.Errorf("cloudcc enableForProfile visualPage <projectPath> <profileId> <pageIds>")
		}
		form.Set("editpid", args[1])
		form.Set("vpageids", args[2])
		mutating = true
		return executeHorizontalVisualPagePermission(projectPath, cfg, "visualPageProfileEnable", action, form, stdout)
	default:
		return fmt.Errorf("unsupported visualPage action: %s", action)
	}
	receipt, err := horizontalaction.ExecuteForm(context.Background(), projectPath, cfg, "visualPage", action, form, mutating)
	if err != nil {
		return err
	}
	return printJSON(stdout, receipt)
}

func createHorizontalVisualPage(name string, stderr io.Writer, cwd string) error {
	name = filepath.Base(strings.TrimSpace(name))
	if err := validateHorizontalAPIName("visualPage", name); err != nil {
		return err
	}
	dir := filepath.Join(cwd, "visualpage", name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	source := `<cc:page type="normal" title="TODO: 页面标题" style="standard" showSidebar="false" showHeader="false" />
<!-- Horizontal VisualPage authoring guide:
  - userInfo is supplied by main-app. Prefer CCService/CCObject platform APIs over direct SQL.
  - Use the platform cc tag DSL; raw JSP scriptlets and jsp namespace tags are rejected.
  - Keep cc declarations and Java blocks small; never call System.exit or create unbounded for/while loops.
  - Do not read or update password, profile, role, licence, or enabled-state columns.
  - Do not use SELECT * or direct DELETE; use explicit fields and platform services.
  - Reference static resources through staticResource.action. Use only existing component API names in component DSL tags.
  - This file is compiled/transformed by main-app PageTemplate and is not a Lightning canvas page.
-->
<div class="cc-visual-page">
  <h1>TODO: 页面标题</h1>
  <p>TODO: 使用 CCService/CCObject 实现业务内容。</p>
</div>
`
	if err := os.WriteFile(filepath.Join(dir, name+".jsp"), []byte(source), 0644); err != nil {
		return err
	}
	if err := jsonx.WriteObjectFile(filepath.Join(dir, "config.json"), map[string]any{"name": name, "label": name, "functiontype": "P", "versiontype": "oldsystem", "folderid": "wgd", "appliedmobile": "false"}); err != nil {
		return err
	}
	fmt.Fprintf(stderr, "Created horizontal VisualPage: %s\n", dir)
	return nil
}

func validateHorizontalVisualPage(source string) error {
	trimmed := strings.TrimSpace(source)
	firstTagEnd := strings.Index(trimmed, "/>")
	if firstTagEnd < 0 || !strings.Contains(strings.ToLower(trimmed[:firstTagEnd+2]), "<cc:page") {
		return fmt.Errorf("horizontal visualPage must start with a self-closing <cc:page ... /> declaration")
	}
	compact := strings.ToLower(strings.Join(strings.Fields(trimmed), ""))
	for _, bad := range []string{"<%", "<jsp:"} {
		if strings.Contains(compact, bad) {
			return fmt.Errorf("horizontal visualPage contains forbidden raw JSP token %q", bad)
		}
	}
	forbiddenDefinition := regexp.MustCompile(`<cc!?>[^<]*(@|com\.g3cloud\.|com\.cloudcc\.|org\.apache\.cayenne)`)
	if match := forbiddenDefinition.FindStringSubmatch(compact); len(match) > 1 {
		return fmt.Errorf("horizontal visualPage cc block contains forbidden definition/import token %q", match[1])
	}
	return nil
}

func handleHorizontalCustomComponent(action string, args []string, stdout io.Writer, stderr io.Writer, cwd string) error {
	if action == "create" {
		if len(args) < 1 {
			return fmt.Errorf("cloudcc create customComponent <name>")
		}
		mode, err := config.ProjectPlatformMode(cwd)
		if err != nil {
			return err
		}
		if mode != config.PlatformHorizontal {
			return fmt.Errorf("customComponent is horizontal-only and is not Lightning pagecomponent")
		}
		return createHorizontalCustomComponent(args[0], stderr, cwd)
	}
	projectPath := cwd
	if action == "publish" {
		if len(args) < 1 {
			return fmt.Errorf("cloudcc publish customComponent <name> [projectPath]")
		}
		if len(args) > 1 {
			projectPath = args[1]
		}
	} else {
		projectPath = firstArg(args, cwd)
	}
	cfg, err := config.Load(projectPath)
	if err != nil {
		return err
	}
	if !config.IsHorizontal(cfg) {
		return fmt.Errorf("customComponent is horizontal-only; use pagecomponent for Lightning")
	}
	if action == "publish" {
		name := filepath.Base(args[0])
		if err := validateHorizontalAPIName("customComponent", name); err != nil {
			return err
		}
		dir := filepath.Join(projectPath, "customComponent", name)
		source, err := os.ReadFile(filepath.Join(dir, name+".cccomponent"))
		if err != nil {
			return err
		}
		if err := validateHorizontalComponent(string(source)); err != nil {
			return err
		}
		local, _ := jsonx.ReadObjectFile(filepath.Join(dir, "config.json"))
		if local == nil {
			local = map[string]any{}
		}
		form := url.Values{"m": {"save"}, "customComponent.id": {cleanAny(configID(local))}, "customComponent.label": {firstNonBlankString(cleanAny(local["label"]), name)}, "customComponent.name": {name}, "customComponent.description": {cleanAny(local["description"])}, "customComponent.visibility": {"1"}, "customComponent.isenable": {"1"}, "customComponent.content": {string(source)}, "rtnURL": {"/customComponent.action?m=index"}}
		configPath := filepath.Join(dir, "config.json")
		receipt, err := horizontalaction.ExecuteResolvedPublish(context.Background(), projectPath, cfg, horizontalaction.PublishRequest{Resource: "customComponent", Path: "/customComponent.action", Form: form, IDField: "customComponent.id", Identity: horizontalaction.PublishIdentity{Name: name, Label: form.Get("customComponent.label")}})
		if err != nil {
			return err
		}
		writeResolvedID(configPath, local, &receipt, stderr)
		return printJSON(stdout, receipt)
	}
	form := url.Values{}
	mutating := action == "delete"
	switch action {
	case "get", "list":
		form.Set("m", "index")
	case "detail", "pull":
		if len(args) < 2 {
			return fmt.Errorf("cloudcc %s customComponent <projectPath> <id>", action)
		}
		form.Set("m", "queryById")
		form.Set("id", args[1])
	case "used":
		if len(args) < 2 {
			return fmt.Errorf("cloudcc used customComponent <projectPath> <id>")
		}
		form.Set("m", "used")
		form.Set("id", args[1])
	case "delete":
		if len(args) < 2 {
			return fmt.Errorf("cloudcc delete customComponent <projectPath> <id>")
		}
		form.Set("m", "delete")
		form.Set("id", args[1])
	default:
		return fmt.Errorf("unsupported customComponent action: %s", action)
	}
	receipt, err := horizontalaction.ExecuteForm(context.Background(), projectPath, cfg, "customComponent", action, form, mutating)
	if err != nil {
		return err
	}
	return printJSON(stdout, receipt)
}

func createHorizontalCustomComponent(name string, stderr io.Writer, cwd string) error {
	name = filepath.Base(strings.TrimSpace(name))
	if err := validateHorizontalAPIName("customComponent", name); err != nil {
		return err
	}
	dir := filepath.Join(cwd, "customComponent", name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	source := `<cc:component>
<!-- Horizontal customComponent guide:
  - main-app provides UserInfo userInfo.
  - Use <cc>...</cc> for Java statements, <cc!>...</cc> for declarations, and
    <cc:outprint>...</cc:outprint> for expression output. Escape untrusted values explicitly.
  - Use <cc:forward> and <cc:param> only when a server-side forward is required.
  - Use CCService, CCObject, SendEmail, Util.*, and Math.* platform capabilities.
  - Raw JSP scriptlets, JSP namespace tags, and nested component references are forbidden.
  - Keep HTML tags balanced. For detailEmbed content, namespace JavaScript functions and DOM ids.
-->
<div class="cc-horizontal-component">
  TODO: 组件内容
</div>
</cc:component>
`
	if err := os.WriteFile(filepath.Join(dir, name+".cccomponent"), []byte(source), 0644); err != nil {
		return err
	}
	if err := jsonx.WriteObjectFile(filepath.Join(dir, "config.json"), map[string]any{"name": name, "label": name, "description": ""}); err != nil {
		return err
	}
	fmt.Fprintf(stderr, "Created horizontal customComponent: %s\n", dir)
	return nil
}

func validateHorizontalComponent(source string) error {
	compact := strings.ToLower(strings.Join(strings.Fields(source), ""))
	if !strings.HasPrefix(compact, "<cc:component>") || !strings.HasSuffix(compact, "</cc:component>") {
		return fmt.Errorf("horizontal customComponent must start with <cc:component> and end with </cc:component>")
	}
	for _, bad := range []string{"<%", "<jsp:", "<component:", "</component"} {
		if strings.Contains(compact, bad) {
			return fmt.Errorf("horizontal customComponent contains forbidden nested/raw JSP token %q", bad)
		}
	}
	forbiddenDefinition := regexp.MustCompile(`<cc!?>[^<]*(@|com\.g3cloud\.|com\.cloudcc\.|org\.apache\.cayenne)`)
	if match := forbiddenDefinition.FindStringSubmatch(compact); len(match) > 1 {
		return fmt.Errorf("horizontal customComponent cc block contains forbidden definition/import token %q", match[1])
	}
	return nil
}

func validateHorizontalAPIName(resource, name string) error {
	if ok, _ := regexp.MatchString(`^[A-Za-z][A-Za-z0-9_]{0,49}$`, name); !ok {
		return fmt.Errorf("%s name must match ^[A-Za-z][A-Za-z0-9_]{0,49}$", resource)
	}
	return nil
}

func handleHorizontalStaticResource(action string, args []string, projectPath string, cfg config.Config, stdout io.Writer) error {
	fields := url.Values{}
	filePath := ""
	mutating := action == "create" || action == "update" || action == "delete"
	switch action {
	case "create":
		if len(args) < 2 {
			return fmt.Errorf("cloudcc create staticResource <name> <filePath> [description]")
		}
		fields.Set("m", "save")
		if err := validateHorizontalAPIName("staticResource", args[0]); err != nil {
			return err
		}
		fields.Set("fileAPI", args[0])
		filePath = args[1]
		if len(args) > 2 {
			fields.Set("description", args[2])
		}
	case "update":
		if len(args) < 4 {
			return fmt.Errorf("cloudcc update staticResource <projectPath> <resourceId> <name> <filePath> [description]")
		}
		fields.Set("m", "save")
		if err := validateHorizontalAPIName("staticResource", args[2]); err != nil {
			return err
		}
		fields.Set("_id", args[1])
		fields.Set("fileAPI", args[2])
		filePath = args[3]
		if len(args) > 4 {
			fields.Set("description", args[4])
		}
	case "get", "list":
		fields.Set("m", "list")
	case "detail":
		if len(args) < 2 {
			return fmt.Errorf("cloudcc detail staticResource <projectPath> <resourceId>")
		}
		fields.Set("m", "detail")
		fields.Set("resourceId", args[1])
	case "delete":
		if len(args) < 2 {
			return fmt.Errorf("cloudcc delete staticResource <projectPath> <resourceId>")
		}
		fields.Set("m", "delete")
		fields.Set("resourceId", args[1])
	case "pull", "download":
		if len(args) < 3 {
			return fmt.Errorf("cloudcc %s staticResource <projectPath> <resourceId> <outputPath>", action)
		}
		receipt, err := horizontalaction.ExecuteStaticResourceDownload(context.Background(), projectPath, cfg, args[1], args[2])
		if err != nil {
			return err
		}
		return printJSON(stdout, receipt)
	default:
		return fmt.Errorf("unsupported horizontal staticResource action: %s", action)
	}
	if action == "create" {
		contentType := strings.ToLower(strings.TrimPrefix(filepath.Ext(filePath), "."))
		fields.Set("contentType", contentType)
		configPath := filepath.Join(projectPath, "staticResource", fields.Get("fileAPI"), "config.json")
		local, _ := jsonx.ReadObjectFile(configPath)
		if local == nil {
			local = map[string]any{}
		}
		if id := cleanAny(configID(local)); id != "" {
			fields.Set("_id", id)
		}
		receipt, err := horizontalaction.ExecuteResolvedPublish(context.Background(), projectPath, cfg, horizontalaction.PublishRequest{Resource: "staticResource", Path: "/staticResource.action", Form: fields, IDField: "_id", Identity: horizontalaction.PublishIdentity{APIName: fields.Get("fileAPI"), ContentType: contentType}, FilePath: filePath})
		if err != nil {
			return err
		}
		local["name"] = fields.Get("fileAPI")
		local["description"] = fields.Get("description")
		local["contentType"] = contentType
		local["sourceFile"] = filepath.Base(filePath)
		writeResolvedID(configPath, local, &receipt, io.Discard)
		return printJSON(stdout, receipt)
	}
	receipt, err := horizontalaction.ExecuteStaticResource(context.Background(), projectPath, cfg, action, fields, filePath, mutating)
	if err != nil {
		return err
	}
	return printJSON(stdout, receipt)
}

func writeResolvedID(configPath string, local map[string]any, receipt *horizontalaction.Receipt, stderr io.Writer) {
	if receipt.IDResolution != "resolved" || strings.TrimSpace(receipt.ID) == "" {
		fmt.Fprintln(stderr, "Horizontal publish was submitted, but the authoritative id could not be resolved; local config was not changed.")
		return
	}
	local["id"] = receipt.ID
	if err := jsonx.WriteObjectFileAtomic(configPath, local); err != nil {
		fmt.Fprintf(stderr, "Horizontal publish resolved id %s, but local config update failed: %v\n", receipt.ID, err)
		return
	}
	updated := true
	receipt.ConfigUpdated = &updated
}

func executeHorizontalVisualPagePermission(projectPath string, cfg config.Config, resource, operation string, form url.Values, stdout io.Writer) error {
	receipt, err := horizontalaction.ExecuteForm(context.Background(), projectPath, cfg, resource, operation, form, true)
	if err != nil {
		return err
	}
	receipt.Resource = "visualPage"
	return printJSON(stdout, receipt)
}

func cleanAny(v any) string {
	s := strings.TrimSpace(fmt.Sprint(v))
	if s == "<nil>" {
		return ""
	}
	return s
}
