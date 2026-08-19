package modules

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"cloudcc-customization-expert-go/internal/config"
	"cloudcc-customization-expert-go/internal/httpclient"
	"cloudcc-customization-expert-go/internal/jsonx"
)

const pageComponentBaseURL = "https://developer.apis.cloudcc.cn"
const pageComponentRootDir = "frontend/pagecomponents"

func pageComponentDir(projectPath string, name string) string {
	return filepath.Join(projectPath, filepath.FromSlash(pageComponentRootDir), name)
}

func pageComponentFrontendDir(projectPath string) string {
	return filepath.Join(projectPath, "frontend")
}

type pageComponentPublishData struct {
	Name          string
	Component     string
	CompName      string
	CompDesc      any
	BizType       any
	Category      any
	LoadModel     any
	BelongOrgFlag any
	BuildVersion  string
	VueContent    string
	VueData       map[string]any
	Dependencies  map[string]string
}

func handlePageComponent(action string, resource string, args []string, stdout io.Writer, stderr io.Writer, cwd string) error {
	switch action {
	case "create":
		return pageComponentCreate(args, stderr, cwd)
	case "package":
		return pageComponentPackage(args, stdout, cwd)
	case "bind":
		return pageComponentBind(args, stdout, cwd)
	case "publish":
		return pageComponentPublish(args, stdout, stderr, cwd)
	case "get":
		return pageComponentGet(args, stdout, cwd)
	case "detail":
		return pageComponentDetail(args, stdout, cwd)
	case "pull":
		return pageComponentPull(args, stderr, cwd)
	case "delete":
		return pageComponentDelete(args, stderr, cwd)
	default:
		return fmt.Errorf("unsupported pagecomponent action: %s", action)
	}
}

func pageComponentCreate(args []string, stderr io.Writer, cwd string) error {
	if len(args) < 1 || strings.TrimSpace(args[0]) == "" {
		return fmt.Errorf("cloudcc create pagecomponent <name>")
	}
	name := strings.TrimSpace(args[0])
	component := "component-" + name
	if !isValidPageComponentElement(component) {
		suggested := suggestPageComponentName(name)
		return fmt.Errorf("%q is not a valid pagecomponent element name; use lowercase letters and '-' only, for example: cloudcc create pagecomponent %s", component, suggested)
	}
	target := pageComponentDir(cwd, name)
	if err := os.MkdirAll(filepath.Join(target, "components"), 0755); err != nil {
		return err
	}
	vue := fmt.Sprintf(`<template>
  <div class="cc-container">
    <HelloWorld />
  </div>
</template>

<script>
import HelloWorld from "./components/HelloWorld.vue";

export default {
  components: {
    HelloWorld,
  },
  data() {
    return {
      componentInfo: {
        component: "%s",
        compName: "%s",
        compDesc: "请填写组件功能描述",
        loadModel: "lazy"
      }
    };
  },
};
</script>

<style lang="scss" scoped>
.cc-container {
  text-align: center;
  padding: 8px;
}
</style>
`, component, name)
	child := `<template>
  <div>Hello world</div>
</template>

<script>
export default {};
</script>

<style lang="scss" scoped>
</style>
`
	if err := os.WriteFile(filepath.Join(target, name+".vue"), []byte(vue), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(target, "components", "HelloWorld.vue"), []byte(child), 0644); err != nil {
		return err
	}
	cfg := map[string]any{
		"component": component,
		"compName":  name,
		"compDesc":  "Component description information",
		"loadModel": "lazy",
	}
	if err := jsonx.WriteObjectFile(filepath.Join(target, "config.json"), cfg); err != nil {
		return err
	}
	fmt.Fprintf(stderr, "Created pagecomponent: %s\n", target)
	return nil
}

func pageComponentPublish(args []string, stdout io.Writer, stderr io.Writer, cwd string) error {
	if len(args) < 1 || strings.TrimSpace(args[0]) == "" {
		return fmt.Errorf("cloudcc publish pagecomponent <name> [projectPath]")
	}
	name := strings.TrimSpace(args[0])
	projectPath := cwd
	if len(args) > 1 && strings.TrimSpace(args[1]) != "" {
		projectPath = args[1]
	}
	projectPath, _ = filepath.Abs(projectPath)
	cfg, err := config.Load(projectPath)
	if err != nil {
		return err
	}
	localConfigPath := filepath.Join(pageComponentDir(projectPath, name), "config.json")
	localConfig, err := jsonx.ReadObjectFile(localConfigPath)
	if err != nil {
		return fmt.Errorf("cannot read pagecomponent config: %w", err)
	}
	pub, err := readPageComponentPublishData(projectPath, name, localConfig, cfg)
	if err != nil {
		return err
	}
	jsPath, checked := pageComponentPrebuiltBundlePath(projectPath, name, pub, localConfig, cfg)
	if jsPath == "" {
		return fmt.Errorf("pagecomponent publish requires a prebuilt UMD bundle; checked %s. Build the bundle outside the Go skill runtime, or set bundlePath/prebuiltBundlePath in frontend/pagecomponents/%s/config.json. The Go CLI does not invoke Node/npm", strings.Join(checked, ", "), name)
	}
	fmt.Fprintf(stderr, "Publishing prebuilt pagecomponent bundle: %s\n", jsPath)
	jsContent, err := os.ReadFile(jsPath)
	if err != nil {
		return fmt.Errorf("cannot read prebuilt pagecomponent JS %s: %w", jsPath, err)
	}
	header := map[string]any{
		"accessToken": pageComponentAccessToken(cfg),
		"source":      firstAny(cfg["source"], "cloudcc_cli"),
	}
	if header["accessToken"] == "" {
		return fmt.Errorf("pagecomponent publish requires pluginToken or accessToken in config")
	}
	body := map[string]any{
		"id":             pageComponentConfigID(projectPath, localConfig),
		"compLabel":      pub.CompName,
		"compUniName":    pub.Component,
		"compContentJs":  string(jsContent),
		"compContentVue": mustJSONString(pub.Dependencies),
		"vueData":        mustJSONString(pub.VueData),
		"bizType":        pub.BizType,
		"compDesc":       pub.CompDesc,
		"category":       pub.Category,
		"loadModel":      firstAny(pub.LoadModel, "lazy"),
		"belongOrgFlag":  firstAny(pub.BelongOrgFlag, cfg["belongOrgFlag"], "custom"),
		"dependencies":   mustJSONString(pub.Dependencies),
	}
	var res map[string]any
	endpoint := strings.TrimRight(baseURL(cfg), "/") + pageComponentDevDispatch(cfg) + "/custom/pc/1.0/post/insertCustomComp"
	if err := httpclient.New().PostEnvelope(endpoint, body, header, &res); err != nil {
		return err
	}
	if code := fmt.Sprint(res["returnCode"]); code != "200" {
		return fmt.Errorf("Publish PageComponent Failed: %v", firstAny(res["returnInfo"], res["msg"], "unknown error"))
	}
	if pageComponentConfigID(projectPath, localConfig) == "" {
		if id := returnedPageComponentID(res); id != "" {
			setPageComponentConfigID(projectPath, localConfig, id)
			_ = jsonx.WriteObjectFile(localConfigPath, localConfig)
		}
	}
	publishedID := firstNonBlankString(returnedPageComponentID(res), pageComponentConfigID(projectPath, localConfig))
	if refs, err := pageComponentCustomPageReferenceHints(projectPath, pub.Component, publishedID); err == nil {
		res["customPageReferences"] = refs
	} else {
		res["customPageReferenceCheckError"] = err.Error()
	}
	return printJSON(stdout, res)
}

func pageComponentPackage(args []string, stdout io.Writer, cwd string) error {
	if len(args) < 1 || strings.TrimSpace(args[0]) == "" {
		return fmt.Errorf("cloudcc package pagecomponent <name> [projectPath] --dry-run")
	}
	name := strings.TrimSpace(args[0])
	projectPath := cwd
	if len(args) > 1 && strings.TrimSpace(args[1]) != "" && !strings.HasPrefix(args[1], "--") {
		projectPath = args[1]
	}
	projectPath, _ = filepath.Abs(projectPath)
	localConfigPath := filepath.Join(pageComponentDir(projectPath, name), "config.json")
	localConfig, err := jsonx.ReadObjectFile(localConfigPath)
	if err != nil {
		return fmt.Errorf("cannot read pagecomponent config: %w", err)
	}
	pub, err := readPageComponentPublishData(projectPath, name, localConfig, map[string]any{})
	if err != nil {
		return err
	}
	bundlePath, checked := pageComponentPrebuiltBundlePath(projectPath, name, pub, localConfig, map[string]any{})
	files := make([]string, 0, len(pub.Dependencies)+1)
	for key := range pub.Dependencies {
		files = append(files, key)
	}
	files = append(files, filepath.ToSlash(filepath.Join(pageComponentRootDir, name, "config.json")))
	sort.Strings(files)
	return printJSON(stdout, map[string]any{
		"mode":              "pagecomponent-package-dry-run",
		"name":              name,
		"component":         pub.Component,
		"projectPath":       projectPath,
		"files":             files,
		"bundlePath":        slashRelOrAbs(projectPath, bundlePath),
		"checkedBundlePath": slashRelList(projectPath, checked),
		"safeDependencyPolicy": map[string]any{
			"include": []string{filepath.ToSlash(filepath.Join(pageComponentRootDir, name))},
			"exclude": []string{"cloudcc-cli.config.json", ".env", "token cache", ".claw-local"},
		},
	})
}

func pageComponentBind(args []string, stdout io.Writer, cwd string) error {
	if len(args) < 3 {
		return fmt.Errorf("cloudcc bind pagecomponent <projectPath> <pageApi> <componentIdOrName> [--embedded true|false] [--workspace-url <url>] [--dry-run]")
	}
	projectPath := args[0]
	pageAPI := args[1]
	componentInput := args[2]
	options := parsePageComponentBindOptions(args[3:])
	page, err := customPageDetailData(projectPath, pageAPI)
	if err != nil {
		return err
	}
	component, err := resolvePageComponentForBind(projectPath, componentInput)
	if err != nil {
		return err
	}
	componentID := strings.TrimSpace(fmt.Sprint(component["id"]))
	componentName := strings.TrimSpace(fmt.Sprint(firstAny(component["compUniName"], component["component"], component["name"])))
	if componentID == "" || componentID == "<nil>" {
		return fmt.Errorf("target pagecomponent id is required")
	}
	if componentName == "" || componentName == "<nil>" {
		return fmt.Errorf("target pagecomponent compUniName is required")
	}
	previousRefs := customPageComponentRefs(page)
	content := customPageObjectList(page["pageContent"])
	if len(content) == 0 {
		content = []map[string]any{{"name": componentName}}
	}
	targetIndex := 0
	for i, item := range content {
		name := strings.TrimSpace(fmt.Sprint(firstAny(item["name"], item["component"], item["compUniName"])))
		if name == componentName {
			targetIndex = i
			break
		}
	}
	item := content[targetIndex]
	item["name"] = componentName
	item["comId"] = componentID
	if options.embeddedSet {
		item["embedded"] = options.embedded
	}
	propObj, _ := item["propObj"].(map[string]any)
	if propObj == nil {
		propObj = map[string]any{}
	}
	if options.workspaceURL != "" {
		propObj["workspaceUrl"] = options.workspaceURL
	}
	item["propObj"] = propObj
	componentInfo, _ := item["componentInfo"].(map[string]any)
	if componentInfo == nil {
		componentInfo = map[string]any{}
	}
	componentInfo["id"] = componentID
	componentInfo["component"] = componentName
	componentInfo["loadModel"] = firstAny(component["loadModel"], componentInfo["loadModel"], "lazy")
	item["componentInfo"] = componentInfo
	content[targetIndex] = item

	payload := map[string]any{}
	for _, key := range []string{"id", "pageLabel", "pageApi", "orgId", "renderVersion", "canvasStyleData", "isTemplate"} {
		if v := page[key]; v != nil {
			payload[key] = v
		}
	}
	payload["pageApi"] = firstAny(page["pageApi"], pageAPI)
	payload["pageContent"] = mapsToAnyList(content)
	payload["compList"] = []any{map[string]any{"id": componentID, "compUniName": componentName}}
	if err := validateCustomPagePayload(payload); err != nil {
		return err
	}
	if options.dryRun {
		return printJSON(stdout, map[string]any{
			"status":  "dry_run",
			"pageApi": pageAPI,
			"previous": map[string]any{
				"componentRefs": previousRefs,
			},
			"proposed": map[string]any{
				"componentId":   componentID,
				"componentName": componentName,
				"payload":       payload,
			},
		})
	}
	cfg, err := config.Load(projectPath)
	if err != nil {
		return err
	}
	var res map[string]any
	if err := customPageSavePost(cfg, payload, &res); err != nil {
		return err
	}
	if err := requireCloudCCSuccess(res, "Bind PageComponent Failed"); err != nil {
		return err
	}
	current := map[string]any{}
	if data, err := customPageDetailData(projectPath, pageAPI); err == nil {
		current = customPageSummary(data)
	}
	return printJSON(stdout, map[string]any{
		"status":  "updated",
		"pageApi": pageAPI,
		"previous": map[string]any{
			"componentRefs": previousRefs,
		},
		"current": map[string]any{
			"componentId":   componentID,
			"componentName": componentName,
			"detail":        current,
		},
		"response": res,
	})
}

func pageComponentCustomPageReferenceHints(projectPath string, componentName string, componentID string) ([]map[string]any, error) {
	cfg, err := config.Load(projectPath)
	if err != nil {
		return nil, err
	}
	var listRes map[string]any
	body := map[string]any{
		"pageNo":   1,
		"pageSize": 2000,
		"condition": map[string]any{
			"pageLabel": "",
			"pageApi":   "",
		},
	}
	if err := customPagePost(cfg, "/custom/pc/1.0/post/pageCustomPage", body, &listRes); err != nil {
		return nil, err
	}
	if err := requireCloudCCSuccess(listRes, "Check CustomPage References Failed"); err != nil {
		return nil, err
	}
	pages := responseList(listRes)
	hints := make([]map[string]any, 0)
	for _, page := range pages {
		pageAPI := strings.TrimSpace(fmt.Sprint(firstAny(page["pageApi"], page["apiName"], page["apiname"])))
		if pageAPI == "" || pageAPI == "<nil>" {
			continue
		}
		detail, err := customPageDetailData(projectPath, pageAPI)
		if err != nil {
			return nil, err
		}
		for _, ref := range customPageComponentRefs(detail) {
			refName := strings.TrimSpace(fmt.Sprint(ref["name"]))
			if componentName != "" && refName != componentName {
				continue
			}
			refID := strings.TrimSpace(fmt.Sprint(ref["comId"]))
			status := "same_component_latest_version"
			if componentID == "" || componentID == "<nil>" {
				status = "referenced_component_unknown"
			} else if refID != componentID {
				status = "stale_component_reference"
			}
			hints = append(hints, map[string]any{
				"status":               status,
				"pageApi":              firstAny(detail["pageApi"], pageAPI),
				"pageLabel":            firstAny(detail["pageLabel"], detail["label"], detail["name"]),
				"customPageId":         firstAny(detail["id"], detail["customPageId"]),
				"componentName":        refName,
				"currentComponentId":   refID,
				"publishedComponentId": componentID,
				"embedded":             firstAny(ref["embedded"], false),
				"workspaceUrl":         firstAny(ref["workspaceUrl"], ""),
			})
		}
	}
	if len(hints) == 0 {
		hints = append(hints, map[string]any{
			"status":               "not_referenced",
			"componentName":        componentName,
			"publishedComponentId": componentID,
		})
	}
	return hints, nil
}

func firstNonBlankString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && value != "<nil>" {
			return value
		}
	}
	return ""
}

func pageComponentGet(args []string, stdout io.Writer, cwd string) error {
	projectPath := firstArg(args, cwd)
	compName := ""
	if len(args) > 1 {
		compName = strings.TrimSpace(args[1])
	}
	cfg, err := config.Load(projectPath)
	if err != nil {
		return err
	}
	body := map[string]any{
		"pageNo":   1,
		"pageSize": 2000,
		"condition": map[string]any{
			"compName": compName,
			"dtBegin":  "",
			"dtEnd":    "",
		},
	}
	var res map[string]any
	if err := httpclient.New().PostEnvelope(strings.TrimRight(baseURL(cfg), "/")+pageComponentDevDispatch(cfg)+"/custom/pc/1.0/post/pageCustomComp", body, pageComponentQueryHeader(cfg), &res); err != nil {
		return err
	}
	if code := fmt.Sprint(res["returnCode"]); code != "200" {
		return fmt.Errorf("Get PageComponent List Failed: %v", firstAny(res["returnInfo"], res["msg"], "unknown error"))
	}
	var out []map[string]any
	if data, _ := res["data"].(map[string]any); data != nil {
		if list, _ := data["list"].([]any); list != nil {
			for _, item := range list {
				if m, _ := item.(map[string]any); m != nil {
					out = append(out, map[string]any{
						"name":           firstAny(m["compUniName"], m["name"]),
						"component":      firstAny(m["compUniName"], m["component"]),
						"compName":       firstAny(m["compLabel"], m["compName"]),
						"compDesc":       firstAny(m["compDesc"], ""),
						"id":             m["id"],
						"version":        firstAny(m["version"], nil),
						"category":       firstAny(m["category"], nil),
						"bizType":        firstAny(m["bizType"], nil),
						"createBy":       firstAny(m["createBy"], nil),
						"createDate":     firstAny(m["createDate"], nil),
						"lastModifyBy":   firstAny(m["lastModifyBy"], nil),
						"lastModifyDate": firstAny(m["lastModifyDate"], nil),
						"orgId":          firstAny(m["orgId"], nil),
						"orgName":        firstAny(m["orgName"], nil),
						"isEnabled":      firstAny(m["isEnabled"], nil),
						"isDeleted":      firstAny(m["isDeleted"], nil),
						"disableReason":  firstAny(m["disableReason"], nil),
						"belongOrgFlag":  firstAny(m["belongOrgFlag"], nil),
						"loadModel":      firstAny(m["loadModel"], nil),
						"apiName":        firstAny(m["apiName"], nil),
						"vueData":        firstAny(m["vueData"], nil),
						"fromLocal":      false,
					})
				}
			}
		}
	}
	return printJSON(stdout, out)
}

func pageComponentDetail(args []string, stdout io.Writer, cwd string) error {
	pageComponentName := ""
	pageComponentID := ""
	projectPath := cwd
	if len(args) > 0 {
		pageComponentName = strings.TrimSpace(args[0])
	}
	if len(args) > 1 {
		pageComponentID = strings.TrimSpace(args[1])
	}
	if len(args) > 2 && strings.TrimSpace(args[2]) != "" {
		projectPath = args[2]
	}
	if pageComponentName != "" {
		dir := pageComponentDir(projectPath, pageComponentName)
		localConfig, err := jsonx.ReadObjectFile(filepath.Join(dir, "config.json"))
		if err != nil {
			return fmt.Errorf("pagecomponent not found in local directory %s: %w", dir, err)
		}
		source, err := os.ReadFile(filepath.Join(dir, pageComponentName+".vue"))
		if err != nil {
			return fmt.Errorf("pagecomponent source not found: %w", err)
		}
		return printJSON(stdout, map[string]any{
			"name":      pageComponentName,
			"component": firstAny(localConfig["component"], "component-"+pageComponentName),
			"compName":  firstAny(localConfig["compName"], pageComponentName),
			"compDesc":  firstAny(localConfig["compDesc"], ""),
			"id":        pageComponentConfigID(projectPath, localConfig),
			"source":    string(source),
			"config":    localConfig,
			"published": pageComponentConfigID(projectPath, localConfig) != "",
			"fromLocal": true,
		})
	}
	if pageComponentID != "" {
		data, err := pageComponentDetailByID(projectPath, pageComponentID)
		if err != nil {
			return err
		}
		delete(data, "compContentJs")
		delete(data, "compContentVue")
		data["fromLocal"] = false
		return printJSON(stdout, data)
	}
	return fmt.Errorf("cloudcc detail pagecomponent <name> [id] [projectPath] or cloudcc detail pagecomponent \"\" <id> [projectPath]")
}

func pageComponentPull(args []string, stderr io.Writer, cwd string) error {
	if len(args) < 1 || strings.TrimSpace(args[0]) == "" {
		return fmt.Errorf("cloudcc pull pagecomponent <nameOrId> [projectPath]")
	}
	input := strings.TrimSpace(args[0])
	projectPath := cwd
	if len(args) > 1 && strings.TrimSpace(args[1]) != "" {
		projectPath = args[1]
	}
	pageComponentID := input
	configPath := filepath.Join(pageComponentDir(projectPath, input), "config.json")
	if localConfig, err := jsonx.ReadObjectFile(configPath); err == nil {
		if component := strings.TrimSpace(fmt.Sprint(localConfig["component"])); component != "" && component != "<nil>" {
			list, err := pageComponentList(projectPath, component)
			if err == nil {
				for _, item := range list {
					if fmt.Sprint(item["component"]) == component || fmt.Sprint(item["name"]) == component {
						if id := fmt.Sprint(item["id"]); id != "" && id != "<nil>" {
							pageComponentID = id
							break
						}
					}
				}
			}
		}
		if pageComponentID == input {
			if id := pageComponentConfigID(projectPath, localConfig); id != "" {
				pageComponentID = id
			}
		}
	}
	data, err := pageComponentDetailByID(projectPath, pageComponentID)
	if err != nil {
		return err
	}
	remoteName := strings.TrimSpace(fmt.Sprint(firstAny(data["compUniName"], data["name"], data["component"])))
	if remoteName == "" || remoteName == "<nil>" {
		return fmt.Errorf("cannot determine pagecomponent name from server data")
	}
	localName := strings.TrimPrefix(remoteName, "component-")
	pageDir := pageComponentDir(projectPath, localName)
	if err := os.MkdirAll(pageDir, 0755); err != nil {
		return err
	}
	if raw := strings.TrimSpace(fmt.Sprint(data["compContentVue"])); raw != "" && raw != "<nil>" {
		var files map[string]string
		if err := json.Unmarshal([]byte(raw), &files); err == nil {
			keys := make([]string, 0, len(files))
			for k := range files {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, key := range keys {
				if shouldSkipPulledPageComponentFile(key) {
					continue
				}
				target := resolvePulledPageComponentFile(key, projectPath, pageDir, localName, remoteName)
				if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
					return err
				}
				if err := os.WriteFile(target, []byte(files[key]), 0644); err != nil {
					return err
				}
			}
		} else {
			if err := os.WriteFile(filepath.Join(pageDir, localName+".vue"), []byte(raw), 0644); err != nil {
				return err
			}
		}
	} else if source := strings.TrimSpace(fmt.Sprint(data["source"])); source != "" && source != "<nil>" {
		if err := os.WriteFile(filepath.Join(pageDir, localName+".vue"), []byte(source), 0644); err != nil {
			return err
		}
	}
	localConfig := map[string]any{
		"component": strings.TrimSpace(fmt.Sprint(firstAny(data["compUniName"], data["component"], "component-"+localName))),
		"compName":  firstAny(data["compLabel"], data["compName"], localName),
		"compDesc":  firstAny(data["compDesc"], ""),
		"bizType":   firstAny(data["bizType"], ""),
		"category":  firstAny(data["category"], ""),
		"loadModel": firstAny(data["loadModel"], "lazy"),
	}
	setPageComponentConfigID(projectPath, localConfig, pageComponentID)
	if err := jsonx.WriteObjectFile(filepath.Join(pageDir, "config.json"), localConfig); err != nil {
		return err
	}
	fmt.Fprintf(stderr, "Success! PageComponent %q (ID: %s) pulled to %s\n", localName, pageComponentID, pageDir)
	return nil
}

func pageComponentDelete(args []string, stderr io.Writer, cwd string) error {
	if len(args) < 1 || strings.TrimSpace(args[0]) == "" {
		return fmt.Errorf("cloudcc delete pagecomponent <nameOrId> [projectPath]")
	}
	input := strings.TrimSpace(args[0])
	projectPath := cwd
	if len(args) > 1 && strings.TrimSpace(args[1]) != "" {
		projectPath = args[1]
	}
	pageComponentID := input
	if localConfig, err := jsonx.ReadObjectFile(filepath.Join(pageComponentDir(projectPath, input), "config.json")); err == nil {
		if id := pageComponentConfigID(projectPath, localConfig); id != "" {
			pageComponentID = id
		}
	}
	cfg, err := config.Load(projectPath)
	if err != nil {
		return err
	}
	body := map[string]any{"id": pageComponentID}
	var res map[string]any
	fmt.Fprintf(stderr, "Deleting pagecomponent (ID: %s), please wait...\n", pageComponentID)
	if err := httpclient.New().PostEnvelope(strings.TrimRight(baseURL(cfg), "/")+pageComponentDevDispatch(cfg)+"/custom/pc/1.0/post/deleteCustomComp", body, pageComponentQueryHeader(cfg), &res); err != nil {
		return err
	}
	if code := fmt.Sprint(res["returnCode"]); code != "200" {
		return fmt.Errorf("Delete PageComponent Failed: %v", firstAny(res["returnInfo"], res["msg"], "unknown error"))
	}
	fmt.Fprintf(stderr, "Success! PageComponent (ID: %s) deleted from server.\n", pageComponentID)
	return nil
}

func readPageComponentPublishData(projectPath string, name string, localConfig map[string]any, cfg config.Config) (pageComponentPublishData, error) {
	entry := filepath.Join(pageComponentDir(projectPath, name), name+".vue")
	vueBytes, err := os.ReadFile(entry)
	if err != nil {
		return pageComponentPublishData{}, err
	}
	vueContent := string(vueBytes)
	if !strings.Contains(vueContent, "scoped") {
		// Keep original warning behavior, but do not block publication.
	}
	data, err := parsePageComponentVueData(vueContent)
	if err != nil {
		data = map[string]any{}
		if componentInfo := parsePageComponentInfo(vueContent); len(componentInfo) > 0 {
			data["componentInfo"] = componentInfo
		}
	}
	fillPageComponentDefaults(data)
	componentInfo, _ := data["componentInfo"].(map[string]any)
	component := strings.TrimSpace(fmt.Sprint(firstAny(componentInfo["component"], localConfig["component"], "component-"+name)))
	compName := strings.TrimSpace(fmt.Sprint(firstAny(componentInfo["compName"], localConfig["compName"], name)))
	deps := collectPageComponentDependencies(entry, projectPath)
	deps = safePageComponentDependencies(projectPath, name, deps)
	buildVersion := strings.TrimSpace(fmt.Sprint(firstAny(componentInfo["buildVersion"], localConfig["buildVersion"], cfg["buildVersion"], "v1")))
	return pageComponentPublishData{
		Name:          name,
		Component:     component,
		CompName:      compName,
		CompDesc:      firstAny(componentInfo["compDesc"], localConfig["compDesc"], ""),
		BizType:       firstAny(componentInfo["bizType"], localConfig["bizType"], nil),
		Category:      firstAny(componentInfo["category"], localConfig["category"], nil),
		LoadModel:     firstAny(componentInfo["loadModel"], localConfig["loadModel"], "lazy"),
		BelongOrgFlag: firstAny(componentInfo["belongOrgFlag"], localConfig["belongOrgFlag"], cfg["belongOrgFlag"], "custom"),
		BuildVersion:  buildVersion,
		VueContent:    vueContent,
		VueData:       data,
		Dependencies:  deps,
	}, nil
}

type pageComponentBindOptions struct {
	embeddedSet  bool
	embedded     bool
	workspaceURL string
	dryRun       bool
}

func parsePageComponentBindOptions(args []string) pageComponentBindOptions {
	opts := pageComponentBindOptions{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--embedded":
			if i+1 < len(args) {
				opts.embeddedSet = true
				opts.embedded = strings.EqualFold(args[i+1], "true") || args[i+1] == "1"
				i++
			}
		case "--workspace-url":
			if i+1 < len(args) {
				opts.workspaceURL = args[i+1]
				i++
			}
		case "--dry-run":
			opts.dryRun = true
		}
	}
	return opts
}

func resolvePageComponentForBind(projectPath string, input string) (map[string]any, error) {
	if strings.HasPrefix(strings.TrimSpace(input), "component-") {
		if data, err := pageComponentDetailFromList(projectPath, input); err == nil {
			return data, nil
		}
	}
	if data, err := pageComponentDetailByID(projectPath, input); err == nil {
		return data, nil
	}
	return pageComponentDetailFromList(projectPath, input)
}

func pageComponentDetailFromList(projectPath string, input string) (map[string]any, error) {
	list, err := pageComponentList(projectPath, input)
	if err != nil {
		return nil, err
	}
	for _, item := range list {
		if fmt.Sprint(item["id"]) == input || fmt.Sprint(item["component"]) == input || fmt.Sprint(item["name"]) == input {
			id := strings.TrimSpace(fmt.Sprint(item["id"]))
			if id == "" || id == "<nil>" {
				break
			}
			return pageComponentDetailByID(projectPath, id)
		}
	}
	return nil, fmt.Errorf("pagecomponent %q not found", input)
}

func mapsToAnyList(items []map[string]any) []any {
	out := make([]any, 0, len(items))
	for _, item := range items {
		out = append(out, item)
	}
	return out
}

func safePageComponentDependencies(projectPath string, name string, deps map[string]string) map[string]string {
	out := map[string]string{}
	prefix := filepath.ToSlash(filepath.Join(pageComponentRootDir, name)) + "/"
	for key, value := range deps {
		normalized := filepath.ToSlash(strings.TrimPrefix(key, "./"))
		if !strings.HasPrefix(normalized, prefix) {
			continue
		}
		if isUnsafePageComponentDependency(normalized) {
			continue
		}
		out[normalized] = value
	}
	return out
}

func isUnsafePageComponentDependency(path string) bool {
	base := strings.ToLower(filepath.Base(filepath.ToSlash(path)))
	if base == "cloudcc-cli.config.json" || base == "cloudcc-cli.config.js" || base == ".env" || strings.HasSuffix(base, ".env") {
		return true
	}
	if strings.Contains(strings.ToLower(path), "token") || strings.Contains(strings.ToLower(path), "secret") {
		return true
	}
	return false
}

func slashRelOrAbs(base string, path string) string {
	if path == "" {
		return ""
	}
	if rel, err := filepath.Rel(base, path); err == nil && !strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(rel)
	}
	return filepath.ToSlash(path)
}

func slashRelList(base string, paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		out = append(out, slashRelOrAbs(base, path))
	}
	return out
}

func pageComponentPrebuiltBundlePath(projectPath string, name string, pub pageComponentPublishData, localConfig map[string]any, cfg config.Config) (string, []string) {
	candidates := pageComponentPrebuiltBundleCandidates(projectPath, name, pub, localConfig, cfg)
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, candidates
		}
	}
	return "", candidates
}

func pageComponentPrebuiltBundleCandidates(projectPath string, name string, pub pageComponentPublishData, localConfig map[string]any, cfg config.Config) []string {
	rawValues := []any{
		localConfig["bundlePath"],
		localConfig["prebuiltBundlePath"],
		localConfig["prebuiltBundle"],
		localConfig["jsBundlePath"],
		cfg["pagecomponentBundlePath"],
		cfg["prebuiltBundlePath"],
	}
	componentDir := pageComponentDir(projectPath, name)
	candidates := []string{}
	seen := map[string]bool{}
	addCandidate := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" || path == "<nil>" {
			return
		}
		if !filepath.IsAbs(path) {
			for _, base := range []string{projectPath, componentDir} {
				abs := filepath.Clean(filepath.Join(base, filepath.FromSlash(path)))
				if !seen[abs] {
					seen[abs] = true
					candidates = append(candidates, abs)
				}
			}
			return
		}
		abs := filepath.Clean(path)
		if !seen[abs] {
			seen[abs] = true
			candidates = append(candidates, abs)
		}
	}
	for _, raw := range rawValues {
		addCandidate(fmt.Sprint(raw))
	}
	for _, path := range []string{
		filepath.Join(pageComponentFrontendDir(projectPath), "build", pub.Component+".umd.min.js"),
		filepath.Join(pageComponentFrontendDir(projectPath), "build", pub.Component+".umd.js"),
		filepath.Join(componentDir, "build", pub.Component+".umd.min.js"),
		filepath.Join(componentDir, "build", pub.Component+".umd.js"),
		filepath.Join(componentDir, "build", name+".umd.min.js"),
		filepath.Join(componentDir, "build", name+".umd.js"),
	} {
		addCandidate(path)
	}
	return candidates
}

func pageComponentDetailByID(projectPath string, pageComponentID string) (map[string]any, error) {
	cfg, err := config.Load(projectPath)
	if err != nil {
		return nil, err
	}
	var res map[string]any
	if err := httpclient.New().PostEnvelope(strings.TrimRight(baseURL(cfg), "/")+pageComponentDevDispatch(cfg)+"/custom/pc/1.0/post/detailCustomComp", map[string]any{"id": pageComponentID}, pageComponentQueryHeader(cfg), &res); err != nil {
		return nil, err
	}
	if ok, _ := res["result"].(bool); !ok {
		if code := fmt.Sprint(res["returnCode"]); code != "200" {
			return nil, fmt.Errorf("Get PageComponent Details Failed: %v", firstAny(res["returnInfo"], res["msg"], "unknown error"))
		}
	}
	data, _ := res["data"].(map[string]any)
	if data == nil {
		return nil, fmt.Errorf("Get PageComponent Details Failed: empty response data")
	}
	return data, nil
}

func pageComponentList(projectPath string, compName string) ([]map[string]any, error) {
	cfg, err := config.Load(projectPath)
	if err != nil {
		return nil, err
	}
	body := map[string]any{"pageNo": 1, "pageSize": 2000, "condition": map[string]any{"compName": compName, "dtBegin": "", "dtEnd": ""}}
	var res map[string]any
	if err := httpclient.New().PostEnvelope(strings.TrimRight(baseURL(cfg), "/")+pageComponentDevDispatch(cfg)+"/custom/pc/1.0/post/pageCustomComp", body, pageComponentQueryHeader(cfg), &res); err != nil {
		return nil, err
	}
	var out []map[string]any
	if data, _ := res["data"].(map[string]any); data != nil {
		if list, _ := data["list"].([]any); list != nil {
			for _, item := range list {
				if m, _ := item.(map[string]any); m != nil {
					out = append(out, map[string]any{
						"name":      firstAny(m["compUniName"], m["name"]),
						"component": firstAny(m["compUniName"], m["component"]),
						"id":        m["id"],
					})
				}
			}
		}
	}
	return out, nil
}

func parsePageComponentVueData(vueContent string) (map[string]any, error) {
	script := extractPageComponentScript(vueContent)
	if script == "" {
		return nil, fmt.Errorf("cannot find script export default in vue file")
	}
	dataIndex := regexp.MustCompile(`(?s)data\s*(?:\(\s*\)|:\s*function\s*\(\s*\)|:\s*\(\s*\)\s*=>)\s*\{`).FindStringIndex(script)
	if dataIndex == nil {
		return nil, fmt.Errorf("cannot find data function")
	}
	rest := script[dataIndex[1]:]
	returnIndex := strings.Index(rest, "return")
	if returnIndex < 0 {
		return nil, fmt.Errorf("cannot find data return statement")
	}
	objectStart := strings.Index(rest[returnIndex:], "{")
	if objectStart < 0 {
		return nil, fmt.Errorf("cannot find data return object")
	}
	start := dataIndex[1] + returnIndex + objectStart
	objectLiteral, err := extractBalancedJSObject(script, start)
	if err != nil {
		return nil, err
	}
	return parseJSObjectLiteral(objectLiteral)
}

func parsePageComponentInfo(vueContent string) map[string]any {
	idx := strings.Index(vueContent, "componentInfo")
	if idx < 0 {
		return nil
	}
	brace := strings.Index(vueContent[idx:], "{")
	if brace < 0 {
		return nil
	}
	objectLiteral, err := extractBalancedJSObject(vueContent, idx+brace)
	if err != nil {
		return nil
	}
	out, err := parseJSObjectLiteral(objectLiteral)
	if err != nil {
		return nil
	}
	return out
}

func extractPageComponentScript(vueContent string) string {
	re := regexp.MustCompile(`(?is)<script([^>]*)>(.*?)</script>`)
	for _, match := range re.FindAllStringSubmatch(vueContent, -1) {
		if strings.Contains(strings.ToLower(match[1]), "src=") {
			continue
		}
		if strings.Contains(match[2], "export default") {
			return match[2]
		}
	}
	return ""
}

func extractBalancedJSObject(source string, start int) (string, error) {
	if start < 0 || start >= len(source) || source[start] != '{' {
		return "", fmt.Errorf("object does not start with {")
	}
	depth := 0
	inSingle, inDouble, inTemplate, inLineComment, inBlockComment, escaped := false, false, false, false, false, false
	for i := start; i < len(source); i++ {
		ch := source[i]
		next := byte(0)
		if i+1 < len(source) {
			next = source[i+1]
		}
		if inLineComment {
			if ch == '\n' || ch == '\r' {
				inLineComment = false
			}
			continue
		}
		if inBlockComment {
			if ch == '*' && next == '/' {
				inBlockComment = false
				i++
			}
			continue
		}
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && (inSingle || inDouble || inTemplate) {
			escaped = true
			continue
		}
		if inSingle {
			if ch == '\'' {
				inSingle = false
			}
			continue
		}
		if inDouble {
			if ch == '"' {
				inDouble = false
			}
			continue
		}
		if inTemplate {
			if ch == '`' {
				inTemplate = false
			}
			continue
		}
		if ch == '/' && next == '/' {
			inLineComment = true
			i++
			continue
		}
		if ch == '/' && next == '*' {
			inBlockComment = true
			i++
			continue
		}
		switch ch {
		case '\'':
			inSingle = true
		case '"':
			inDouble = true
		case '`':
			inTemplate = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return source[start : i+1], nil
			}
		}
	}
	return "", fmt.Errorf("object braces are not balanced")
}

func parseJSObjectLiteral(literal string) (map[string]any, error) {
	jsonish := stripJSComments(literal)
	jsonish = regexp.MustCompile(`'([^'\\]*(?:\\.[^'\\]*)*)'`).ReplaceAllString(jsonish, `"$1"`)
	jsonish = regexp.MustCompile(`([,{]\s*)([A-Za-z_$][A-Za-z0-9_$]*)\s*:`).ReplaceAllString(jsonish, `$1"$2":`)
	jsonish = regexp.MustCompile(`,\s*([}\]])`).ReplaceAllString(jsonish, `$1`)
	var out map[string]any
	if err := json.Unmarshal([]byte(jsonish), &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]any{}
	}
	return out, nil
}

func stripJSComments(source string) string {
	var b strings.Builder
	inSingle, inDouble, inTemplate, inLineComment, inBlockComment, escaped := false, false, false, false, false, false
	for i := 0; i < len(source); i++ {
		ch := source[i]
		next := byte(0)
		if i+1 < len(source) {
			next = source[i+1]
		}
		if inLineComment {
			if ch == '\n' || ch == '\r' {
				inLineComment = false
				b.WriteByte(ch)
			}
			continue
		}
		if inBlockComment {
			if ch == '*' && next == '/' {
				inBlockComment = false
				i++
			}
			continue
		}
		if escaped {
			escaped = false
			b.WriteByte(ch)
			continue
		}
		if ch == '\\' && (inSingle || inDouble || inTemplate) {
			escaped = true
			b.WriteByte(ch)
			continue
		}
		if inSingle {
			if ch == '\'' {
				inSingle = false
			}
			b.WriteByte(ch)
			continue
		}
		if inDouble {
			if ch == '"' {
				inDouble = false
			}
			b.WriteByte(ch)
			continue
		}
		if inTemplate {
			if ch == '`' {
				inTemplate = false
			}
			b.WriteByte(ch)
			continue
		}
		if ch == '/' && next == '/' {
			inLineComment = true
			i++
			continue
		}
		if ch == '/' && next == '*' {
			inBlockComment = true
			i++
			continue
		}
		switch ch {
		case '\'':
			inSingle = true
		case '"':
			inDouble = true
		case '`':
			inTemplate = true
		}
		b.WriteByte(ch)
	}
	return b.String()
}

func fillPageComponentDefaults(data map[string]any) {
	if _, ok := data["propObj"]; !ok {
		data["propObj"] = map[string]any{}
		data["propOption"] = map[string]any{}
	}
	if _, ok := data["events"]; !ok {
		data["events"] = map[string]any{}
		data["eventsOption"] = map[string]any{}
	}
	if _, ok := data["style"]; !ok {
		data["style"] = map[string]any{"unit": "%", "width": 100, "height": 100, "top": 0, "left": 0, "rotate": 0, "opacity": 1}
		data["styleOption"] = map[string]any{
			"word":   map[string]any{"lable": "label.help", "type": "word", "link": "https://www.google.com"},
			"unit":   map[string]any{"lable": "label.custom.unit", "type": "option", "options": []map[string]any{{"value": "px", "label": "label.custom.pixel"}, {"value": "%", "label": "label.percent"}, {"value": "hw", "label": "label.custom.viewport"}}},
			"width":  map[string]any{"lable": "label.custom.width", "type": "input", "inputType": "number"},
			"height": map[string]any{"lable": "label.custom.height", "type": "input", "inputType": "number"},
			"top":    map[string]any{"lable": "label.dev.y.coordinate", "type": "input", "inputType": "number"},
			"left":   map[string]any{"lable": "label.dev.x.coordinate", "type": "input", "inputType": "number"},
		}
	}
}

func collectPageComponentDependencies(entryFile string, baseDir string) map[string]string {
	out := map[string]string{}
	visited := map[string]bool{}
	collectPageComponentDependency(entryFile, baseDir, visited, out)
	return out
}

func collectPageComponentDependency(file string, baseDir string, visited map[string]bool, out map[string]string) {
	abs, err := filepath.Abs(file)
	if err != nil {
		return
	}
	if visited[abs] {
		return
	}
	visited[abs] = true
	b, err := os.ReadFile(abs)
	if err != nil {
		return
	}
	rel, err := filepath.Rel(baseDir, abs)
	if err != nil {
		return
	}
	rel = filepath.ToSlash(rel)
	content := string(b)
	out[rel] = content
	for _, dep := range parsePageComponentDependencies(content, strings.ToLower(filepath.Ext(abs))) {
		if !strings.HasPrefix(dep, ".") && !strings.HasPrefix(dep, "/") {
			continue
		}
		if resolved := resolvePageComponentDependency(dep, abs); resolved != "" {
			collectPageComponentDependency(resolved, baseDir, visited, out)
		}
	}
}

func parsePageComponentDependencies(content string, ext string) []string {
	var deps []string
	if ext == ".vue" {
		if script := regexp.MustCompile(`(?is)<script[^>]*>(.*?)</script>`).FindStringSubmatch(content); len(script) > 1 {
			deps = append(deps, parseJSImportDependencies(script[1])...)
		}
		for _, style := range regexp.MustCompile(`(?is)<style[^>]*>(.*?)</style>`).FindAllStringSubmatch(content, -1) {
			deps = append(deps, parseCSSImportDependencies(style[1])...)
		}
		return deps
	}
	switch ext {
	case ".css", ".scss", ".sass", ".less":
		return parseCSSImportDependencies(content)
	default:
		return parseJSImportDependencies(content)
	}
}

func parseJSImportDependencies(content string) []string {
	var deps []string
	for _, match := range regexp.MustCompile(`(?m)import\s+(?:(?:\{[^}]*\}|\*\s+as\s+\w+|\w+)\s+from\s+)?["']([^"']+)["']`).FindAllStringSubmatch(content, -1) {
		deps = append(deps, match[1])
	}
	for _, match := range regexp.MustCompile(`require\s*\(\s*["']([^"']+)["']\s*\)`).FindAllStringSubmatch(content, -1) {
		deps = append(deps, match[1])
	}
	return deps
}

func parseCSSImportDependencies(content string) []string {
	var deps []string
	for _, match := range regexp.MustCompile(`@import\s+["']([^"']+)["']`).FindAllStringSubmatch(content, -1) {
		deps = append(deps, match[1])
	}
	return deps
}

func resolvePageComponentDependency(dep string, currentFile string) string {
	base := dep
	if strings.HasPrefix(base, "/") {
		base = "." + base
	}
	currentDir := filepath.Dir(currentFile)
	extensions := []string{"", ".vue", ".js", ".jsx", ".ts", ".tsx", ".css", ".scss", ".sass", ".less"}
	for _, ext := range extensions {
		candidate := filepath.Clean(filepath.Join(currentDir, base+ext))
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	dir := filepath.Clean(filepath.Join(currentDir, base))
	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		for _, ext := range []string{".vue", ".js", ".jsx", ".ts", ".tsx"} {
			candidate := filepath.Join(dir, "index"+ext)
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				return candidate
			}
		}
	}
	return ""
}

func resolvePulledPageComponentFile(filePath string, projectPath string, pageDir string, localName string, remoteName string) string {
	normalized := filepath.ToSlash(strings.TrimPrefix(filePath, "./"))
	if strings.HasPrefix(normalized, "../") {
		return filepath.Join(projectPath, strings.TrimLeft(strings.TrimPrefix(normalized, "../"), "/"))
	}
	if !strings.HasPrefix(normalized, pageComponentRootDir+"/") {
		if strings.HasPrefix(normalized, "frontend/") || strings.HasPrefix(normalized, "backend/") || strings.HasPrefix(normalized, "sidecar/") {
			return filepath.Join(projectPath, filepath.FromSlash(normalized))
		}
	}
	rest := strings.TrimPrefix(normalized, pageComponentRootDir+"/")
	names := []string{localName, remoteName}
	for guard := 0; guard < 64; guard++ {
		changed := false
		for _, name := range names {
			if name != "" && strings.HasPrefix(rest, name+"/") {
				rest = strings.TrimPrefix(rest, name+"/")
				changed = true
				break
			}
		}
		if !changed {
			break
		}
	}
	for guard := 0; guard < 64; guard++ {
		parts := strings.Split(rest, "/")
		if len(parts) < 2 {
			break
		}
		firstNorm := normalizePageComponentToken(parts[0])
		if firstNorm == "" || (firstNorm != normalizePageComponentToken(localName) && firstNorm != normalizePageComponentToken(remoteName)) {
			break
		}
		rest = strings.Join(parts[1:], "/")
	}
	if rest == "" || rest == "." {
		return pageDir
	}
	return filepath.Join(pageDir, filepath.FromSlash(rest))
}

func shouldSkipPulledPageComponentFile(filePath string) bool {
	switch filepath.Base(filepath.ToSlash(filePath)) {
	case "package.json", "cloudcc-cli.config.js", "cloudcc-cli.config.json":
		return true
	default:
		return false
	}
}

func normalizePageComponentToken(name string) string {
	s := strings.ToLower(regexp.MustCompile(`[^a-z0-9]`).ReplaceAllString(name, ""))
	s = strings.TrimPrefix(s, "component")
	s = strings.TrimSuffix(s, "component")
	return s
}

func pageComponentConfigID(projectPath string, localConfig map[string]any) string {
	if root, err := config.Root(projectPath); err == nil {
		if use := strings.TrimSpace(fmt.Sprint(root["use"])); use != "" && use != "<nil>" {
			if v := strings.TrimSpace(fmt.Sprint(localConfig[use+"id"])); v != "" && v != "<nil>" {
				return v
			}
		}
	}
	if v := strings.TrimSpace(fmt.Sprint(localConfig["id"])); v != "" && v != "<nil>" {
		return v
	}
	return ""
}

func setPageComponentConfigID(projectPath string, localConfig map[string]any, id string) {
	key := "id"
	if root, err := config.Root(projectPath); err == nil {
		if use := strings.TrimSpace(fmt.Sprint(root["use"])); use != "" && use != "<nil>" {
			key = use + "id"
		}
	}
	localConfig[key] = id
	delete(localConfig, "id")
}

func pageComponentAccessToken(cfg config.Config) string {
	if token := config.String(cfg, "pluginToken"); token != "" {
		return token
	}
	return config.String(cfg, "accessToken")
}

func pageComponentDevDispatch(cfg config.Config) string {
	if v := config.String(cfg, "devSvcDispatch"); v != "" {
		return v
	}
	return "/devconsole"
}

func pageComponentQueryHeader(cfg config.Config) map[string]any {
	return map[string]any{
		"appType":     "lightning-setup",
		"appVersion":  "0.0.1",
		"accessToken": firstAny(config.String(cfg, "accessToken"), config.String(cfg, "pluginToken")),
		"source":      "lightning-setup",
		"version":     "public",
	}
}

func returnedPageComponentID(res map[string]any) string {
	if data, _ := res["data"].(map[string]any); data != nil {
		if id := strings.TrimSpace(fmt.Sprint(data["id"])); id != "" && id != "<nil>" {
			return id
		}
	}
	if id := strings.TrimSpace(fmt.Sprint(res["data"])); id != "" && id != "<nil>" {
		return id
	}
	return ""
}

func mustJSONString(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func isValidPageComponentElement(name string) bool {
	return regexp.MustCompile(`^[a-z][a-z-]*-[a-z-]+$`).MatchString(name)
}

func suggestPageComponentName(name string) string {
	s := regexp.MustCompile(`([a-z0-9])([A-Z])`).ReplaceAllString(name, `${1}-${2}`)
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.ToLower(s)
	s = regexp.MustCompile(`[^a-z-]+`).ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if !strings.Contains(s, "-") {
		s = "cc-" + s
	}
	return s
}
