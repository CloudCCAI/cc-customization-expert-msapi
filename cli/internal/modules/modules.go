package modules

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"cloudcc-customization-expert-go/internal/compatibility"
	"cloudcc-customization-expert-go/internal/config"
	"cloudcc-customization-expert-go/internal/httpclient"
	"cloudcc-customization-expert-go/internal/jsonx"
	"cloudcc-customization-expert-go/internal/msapi"
	projecttemplates "cloudcc-customization-expert-go/internal/templates"
)

type endpoint struct {
	base string
	path string
}

var genericEndpoints = map[string]map[string]endpoint{
	"apiRegistrar": {
		"get":        {"setup", "/api/register/list"},
		"getList":    {"setup", "/api/register/list"},
		"detail":     {"setup", "/api/register/detail"},
		"create":     {"setup", "/api/register/register"},
		"register":   {"setup", "/api/register/register"},
		"update":     {"setup", "/api/register/update"},
		"modify":     {"setup", "/api/register/update"},
		"delete":     {"setup", "/api/register/delete"},
		"debug":      {"setup", "/api/register/debug"},
		"test":       {"setup", "/api/register/debug"},
		"logs":       {"setup", "/api/register/logs"},
		"logDetail":  {"setup", "/api/register/logDetail"},
		"log-detail": {"setup", "/api/register/logDetail"},
	},
	"approval": {
		"get":     {"setup", "/api/approvalsetup/list"},
		"getList": {"setup", "/api/approvalsetup/list"},
		"detail":  {"setup", "/api/approvalsetup/detail"},
		"newInfo": {"setup", "/api/approvalsetup/editAapproval"},
		"create":  {"setup", "/api/approvalsetup/saveApproval"},
		"update":  {"setup", "/api/approvalsetup/saveApproval"},
		"save":    {"setup", "/api/approvalsetup/saveApproval"},
		"delete":  {"setup", "/api/approvalsetup/deleteApproval"},
	},
	"approvalProcess": {
		"get":     {"setup", "/api/approvalsetup/list"},
		"getList": {"setup", "/api/approvalsetup/list"},
		"detail":  {"setup", "/api/approvalsetup/detail"},
		"newInfo": {"setup", "/api/approvalsetup/editAapproval"},
		"create":  {"setup", "/api/approvalsetup/saveApproval"},
		"update":  {"setup", "/api/approvalsetup/saveApproval"},
		"save":    {"setup", "/api/approvalsetup/saveApproval"},
		"delete":  {"setup", "/api/approvalsetup/deleteApproval"},
	},
	"application": {
		"get":    {"setup", "/api/newApp/getAppList"},
		"create": {"setup", "/api/newApp/save"},
		"update": {"setup", "/api/newApp/save"},
		"delete": {"setup", "/api/newApp/deleteApp"},
	},
	"button": {
		"get":    {"setup", "/api/buttonlink/listButton"},
		"detail": {"setup", "/api/buttonlink/detailButton"},
		"create": {"setup", "/api/buttonlink/saveButton"},
		"update": {"setup", "/api/buttonlink/saveButton"},
		"delete": {"setup", "/api/buttonlink/deleteButton"},
	},
	"customSetting": {
		"get":    {"setup", "/api/customsetting/list"},
		"detail": {"setup", "/api/customsetting/detail"},
		"create": {"setup", "/api/customsetting/save"},
		"update": {"setup", "/api/customsetting/modify"},
		"delete": {"setup", "/api/customsetting/deleteobj"},
		"modify": {"setup", "/api/customsetting/modify"},
	},
	"dupeCatcher": {
		"get":    {"setup", "/api/duplication/getList"},
		"detail": {"setup", "/api/duplication/detailfilter"},
		"create": {"setup", "/api/duplication/saveFilter"},
		"update": {"setup", "/api/duplication/saveFilter"},
		"delete": {"setup", "/api/duplication/deletefilter"},
	},
	"dashboard": {
		"get":     {"api", "/api/dashboard/getDashboardList"},
		"getList": {"api", "/api/dashboard/getDashboardList"},
		"detail":  {"api", "/api/dashboard/getDashboardList"},
		"create":  {"api", "/api/dashboard/addDashboard"},
		"update":  {"api", "/api/dashboard/updateDashboard"},
		"save":    {"api", "/api/dashboard/updateDashboard"},
		"delete":  {"api", "/api/dashboard/deleteDashboard"},
	},
	"fields": {
		"detail": {"setup", "/api/fieldSetup/detailField"},
		"create": {"setup", "/api/fieldSetup/save"},
		"update": {"setup", "/api/fieldSetup/save"},
		"delete": {"setup", "/api/fieldSetup/deleteFieldCompletely"},
	},
	"globalSelectList": {
		"get":    {"setup", "/api/globalSelectSetup/queryList"},
		"detail": {"setup", "/api/globalSelectSetup/detail"},
		"create": {"setup", "/api/globalSelectSetup/save"},
		"update": {"setup", "/api/globalSelectSetup/save"},
		"delete": {"setup", "/api/globalSelectSetup/delete"},
	},
	"identityProvider": {
		"get":      {"setup", "/api/samlSp/queryList"},
		"detail":   {"setup", "/api/samlSp/detail"},
		"create":   {"setup", "/api/samlSp/save"},
		"update":   {"setup", "/api/samlSp/save"},
		"delete":   {"setup", "/api/samlSp/delete"},
		"download": {"setup", "/api/spconfig/downloadXml"},
	},
	"menu": {
		"get":    {"setup", "/api/customTab/queryTabList"},
		"create": {"setup", "/api/customTab/tabSetDone"},
		"delete": {"setup", "/api/customTab/deleteTab"},
	},
	"pagelayout": {
		"get":    {"setup", "/api/layout/queryPageLayout"},
		"create": {"setup", "/api/layout/cloneLayout"},
		"delete": {"setup", "/api/layout/deleteButton"},
	},
	"permission": {
		"get":    {"setup", "/api/permissionGroup/queryPermsetsList"},
		"query":  {"setup", "/api/permissionGroup/queryPermsetsList"},
		"create": {"setup", "/api/permissionGroup/savePermsets"},
		"update": {"setup", "/api/permissionGroup/modifyPermsets"},
		"delete": {"setup", "/api/permissionGroup/deletePermsets"},
		"assign": {"setup", "/api/permissionGroup/addUsersetup"},
		"add":    {"setup", "/api/permissionGroup/addUsersetup"},
		"remove": {"setup", "/api/permissionGroup/deleteUsersetup"},
	},
	"profile": {
		"get":    {"setup", "/api/profile/listAll"},
		"detail": {"setup", "/api/profile/detailInfo"},
		"create": {"setup", "/api/profile/newProfile"},
		"update": {"setup", "/api/profile/saveProfile"},
		"delete": {"setup", "/api/profile/delProfile"},
	},
	"recordType": {
		"get":         {"setup", "/api/recordType/getRecordTypeList"},
		"getList":     {"setup", "/api/recordType/getRecordTypeList"},
		"newInfo":     {"setup", "/api/recordType/newRecordType"},
		"detail":      {"setup", "/api/recordType/getRecoderTypeDetail"},
		"create":      {"setup", "/api/recordType/saveRecordType"},
		"editInfo":    {"setup", "/api/recordType/editRecordType"},
		"editSave":    {"setup", "/api/recordType/editSave"},
		"update":      {"setup", "/api/recordType/editSave"},
		"validDelete": {"setup", "/api/recordType/validDelete"},
		"delete":      {"setup", "/api/recordType/deleteObj"},
	},
	"report": {
		"get":     {"api", "/api/report/tab/getReportList"},
		"getList": {"api", "/api/report/tab/getReportList"},
		"detail":  {"api", "/api/report/tab/getReportDetail"},
		"create":  {"api", "/api/report/tab/saveReport"},
		"update":  {"api", "/api/report/tab/saveReport"},
		"save":    {"api", "/api/report/tab/saveReport"},
		"delete":  {"api", "/api/report/base/deleteReport"},
	},
	"reports": {
		"get":     {"api", "/api/report/tab/getReportList"},
		"getList": {"api", "/api/report/tab/getReportList"},
		"detail":  {"api", "/api/report/tab/getReportDetail"},
		"create":  {"api", "/api/report/tab/saveReport"},
		"update":  {"api", "/api/report/tab/saveReport"},
		"save":    {"api", "/api/report/tab/saveReport"},
		"delete":  {"api", "/api/report/base/deleteReport"},
	},
	"reportTabular": {
		"create":   {"api", "/api/report/tab/saveReport"},
		"update":   {"api", "/api/report/tab/saveReport"},
		"save":     {"api", "/api/report/tab/saveReport"},
		"modify":   {"api", "/api/report/tab/saveReport"},
		"editSave": {"api", "/api/report/tab/saveReport"},
	},
	"reportSummary": {
		"create":   {"api", "/api/report/tab/saveReport"},
		"update":   {"api", "/api/report/tab/saveReport"},
		"save":     {"api", "/api/report/tab/saveReport"},
		"modify":   {"api", "/api/report/tab/saveReport"},
		"editSave": {"api", "/api/report/tab/saveReport"},
	},
	"reportMatrix": {
		"create":   {"api", "/api/report/tab/saveReport"},
		"update":   {"api", "/api/report/tab/saveReport"},
		"save":     {"api", "/api/report/tab/saveReport"},
		"modify":   {"api", "/api/report/tab/saveReport"},
		"editSave": {"api", "/api/report/tab/saveReport"},
	},
	"reportRatio": {
		"create":   {"api", "/api/report/tab/saveReport"},
		"update":   {"api", "/api/report/tab/saveReport"},
		"save":     {"api", "/api/report/tab/saveReport"},
		"modify":   {"api", "/api/report/tab/saveReport"},
		"editSave": {"api", "/api/report/tab/saveReport"},
	},
	"reportFolder": {
		"get":     {"api", "/api/report/tab/getReportFolders"},
		"getList": {"api", "/api/report/tab/getReportFolders"},
		"detail":  {"api", "/api/report/folder/getReportFolderInfo"},
		"create":  {"api", "/api/report/folder/addReportFolder"},
		"update":  {"api", "/api/report/folder/updateReportFolder"},
		"save":    {"api", "/api/report/folder/updateReportFolder"},
		"delete":  {"api", "/api/report/folder/deleteReportFolder"},
	},
	"role": {
		"get":      {"setup", "/api/role/queryRole"},
		"detail":   {"setup", "/api/role/editRole"},
		"create":   {"setup", "/api/role/saveRole"},
		"update":   {"setup", "/api/role/editSaveRole"},
		"assign":   {"setup", "/api/role/saveAssign"},
		"delete":   {"setup", "/api/role/deleteRole"},
		"editInfo": {"setup", "/api/role/editRole"},
	},
	"scheduleJob": {
		"get":     {"setup", "/api/schedulAbleprg/list"},
		"getList": {"setup", "/api/lookup/getLookupData"},
		"detail":  {"setup", "/api/schedulAbleprg/edit"},
		"create":  {"setup", "/api/schedulAbleprg/save"},
		"update":  {"setup", "/api/schedulAbleprg/save"},
		"delete":  {"setup", "/api/schedulAbleprg/delete"},
	},
	"singleSignOn": {
		"get":    {"setup", "/api/spconfig/list"},
		"detail": {"setup", "/api/spconfig/queryIdpById"},
		"create": {"setup", "/api/spconfig/save"},
		"update": {"setup", "/api/spconfig/save"},
		"delete": {"setup", "/api/spconfig/delete"},
	},
	"user": {
		"get":          {"setup", "/api/usermange/queryUserList"},
		"query":        {"setup", "/api/usermange/queryUserList"},
		"getList":      {"setup", "/api/usermange/queryUserList"},
		"views":        {"setup", "/api/usermange/queryUser"},
		"queryViews":   {"setup", "/api/usermange/queryUser"},
		"newInfo":      {"setup", "/api/usermange/addUserQuery"},
		"addUserQuery": {"setup", "/api/usermange/addUserQuery"},
		"detail":       {"setup", "/api/usermange/viewUser"},
		"view":         {"setup", "/api/usermange/viewUser"},
		"create":       {"setup", "/api/usermange/saveUser"},
		"save":         {"setup", "/api/usermange/saveUser"},
		"editInfo":     {"setup", "/api/usermange/editUserQuery"},
		"update":       {"setup", "/api/usermange/editandsave"},
		"editSave":     {"setup", "/api/usermange/editandsave"},
		"delete":       {"setup", "/api/usermange/editandsave"},
		"deactivate":   {"setup", "/api/usermange/editandsave"},
		"disable":      {"setup", "/api/usermange/editandsave"},
		"resetpw":      {"setup", "/api/usermange/resetpw"},
		"unlock":       {"setup", "/api/usermange/unlocked"},
		"unlocked":     {"setup", "/api/usermange/unlocked"},
		"unBindMfa":    {"setup", "/api/usermange/unBindMfa"},
		"mfa-unbind":   {"setup", "/api/usermange/unBindMfa"},
		"choseemail":   {"setup", "/api/usermange/choseemail"},
		"setSendFrom":  {"setup", "/api/usermange/setSendFrom"},
		"sendemail":    {"setup", "/api/usermange/sendemail"},
	},
	"validationRule": {
		"get":    {"setup", "/api/validateRule/queryByPrefix"},
		"create": {"setup", "/api/validateRule/save"},
		"update": {"setup", "/api/validateRule/save"},
		"delete": {"setup", "/api/validateRule/delete"},
	},
	"view": {
		"get":    {"setup", "/api/view/list/getViewList"},
		"detail": {"setup", "/api/view/getViewInfo"},
		"create": {"setup", "/api/view/saveView"},
		"update": {"setup", "/api/view/saveView"},
		"delete": {"setup", "/api/view/deleteView"},
	},
	"sharingRule": {
		"get":    {"setup", "/api/sharingSettings/queryRule"},
		"detail": {"setup", "/api/sharingSettings/toUpdateRule"},
		"create": {"setup", "/api/sharingSettings/insertRule"},
		"update": {"setup", "/api/sharingSettings/updateRule"},
		"delete": {"setup", "/api/sharingSettings/deleteRule"},
	},
}

func Handle(action string, resource string, args []string, stdout io.Writer, stderr io.Writer, cwd string) error {
	if resource == "apiRegister" || resource == "api-registrar" || resource == "api-register" {
		resource = "apiRegistrar"
	}
	switch resource {
	case "config":
		return handleConfig(action, args, stdout, cwd)
	case "object":
		return handleObject(action, args, stdout, stderr, cwd)
	case "brief":
		return handleBrief(action, args, stdout, cwd)
	case "fields":
		if action == "get" {
			return handleFieldsGet(args, stdout, cwd)
		}
	case "classes":
		if action == "doctor" || action == "prepare" || action == "validate" || action == "test" {
			return handleClassDev(action, args, stdout, stderr, cwd)
		}
		if action == "publish" {
			return publishClassResource(args, stdout, stderr, cwd)
		}
		return handleCodeResource(action, resource, "classes", "ccfag", args, stdout, stderr, cwd)
	case "trigger", "triggers":
		return handleTrigger(action, args, stdout, stderr, cwd)
	case "timer", "schedule":
		return handleCodeResource(action, "timer", "schedule", "ccPeak", args, stdout, stderr, cwd)
	case "script":
		return handleScript(action, args, stdout, stderr, cwd)
	case "html":
		return handleHTML(action, args, stdout, stderr, cwd)
	case "staticResource":
		return handleStaticResource(action, args, stdout, stderr, cwd)
	case "customPage":
		return handleCustomPage(action, args, stdout, stderr, cwd)
	case "scheduleJob":
		return handleScheduleJob(action, args, stdout, cwd)
	case "pagecomponent":
		return handlePageComponent(action, resource, args, stdout, stderr, cwd)
	case "injectionPage":
		return handleInjectionPage(action, args, stdout, stderr, cwd)
	case "validationRule":
		return handleValidationRule(action, args, stdout, cwd)
	case "project":
		return handleProject(action, args, stderr, cwd)
	case "pagelayout":
		return handlePageLayout(action, args, stdout, stderr, cwd)
	case "menu":
		return handleMenu(action, args, stdout, cwd)
	case "user":
		return handleUser(action, args, stdout, cwd)
	case "jsp", "site", "skill":
		return fmt.Errorf("%s %s is deferred to P4 or a later task in the Go rewrite", action, resource)
	}
	if isReportShortcutResource(resource) {
		if byAction, ok := genericEndpoints[resource]; ok {
			if ep, ok := byAction[action]; ok {
				return handleReportShortcut(ep, action, resource, args, stdout, cwd)
			}
		}
	}
	if byAction, ok := genericEndpoints[resource]; ok {
		if ep, ok := byAction[action]; ok {
			if resource == "apiRegistrar" && isApiRegistrarRuntimeAction(action) {
				return callGenericRedacted(ep, action, resource, args, stdout, cwd)
			}
			return callGeneric(ep, action, resource, args, stdout, cwd)
		}
	}
	return fmt.Errorf("unsupported command: cloudcc %s %s", action, resource)
}

func isReportShortcutResource(resource string) bool {
	switch resource {
	case "report", "reports", "reportTabular", "reportSummary", "reportMatrix", "reportRatio", "reportFolder":
		return true
	default:
		return false
	}
}

func handleReportShortcut(ep endpoint, action string, resource string, args []string, stdout io.Writer, cwd string) error {
	projectPath := firstArg(args, cwd)
	body, err := msapi.NormalizeReportShortcut(action, resource, argsAfterProject(args))
	if err != nil {
		return err
	}
	cfg, err := config.Load(projectPath)
	if err != nil {
		return err
	}
	return postClass(stdout, cfg, ep.base, ep.path, body)
}

func argsAfterProject(args []string) []string {
	if len(args) <= 1 {
		return nil
	}
	return args[1:]
}

func handleScheduleJob(action string, args []string, stdout io.Writer, cwd string) error {
	byAction := genericEndpoints["scheduleJob"]
	ep, ok := byAction[action]
	if !ok {
		return fmt.Errorf("unsupported scheduleJob action: %s", action)
	}
	if action != "getList" {
		return callGeneric(ep, action, "scheduleJob", args, stdout, cwd)
	}

	projectPath := firstArg(args, cwd)
	body := map[string]any{
		"prefix":        "ccp",
		"searchKeyWord": "",
		"page":          1,
		"pageSize":      9999,
	}
	if len(args) > 1 && strings.TrimSpace(args[1]) != "" {
		overrides, err := jsonx.ParseEncodedObject(args[1], "getList scheduleJob")
		if err != nil {
			return err
		}
		for key, value := range overrides {
			body[key] = value
		}
	}
	cfg, err := config.Load(projectPath)
	if err != nil {
		return err
	}
	return postClass(stdout, cfg, ep.base, ep.path, body)
}

func handleUser(action string, args []string, stdout io.Writer, cwd string) error {
	ep, ok := genericEndpoints["user"][action]
	if !ok {
		return fmt.Errorf("unsupported user action: %s", action)
	}
	projectPath := firstArg(args, cwd)
	rest := argsAfterProject(args)
	body, err := userRequestBody(action, rest)
	if err != nil {
		return err
	}
	cfg, err := config.Load(projectPath)
	if err != nil {
		return err
	}
	return postClass(stdout, cfg, ep.base, ep.path, body)
}

func userRequestBody(action string, args []string) (map[string]any, error) {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		switch action {
		case "get", "query", "getList", "views", "queryViews", "newInfo", "addUserQuery":
			return map[string]any{}, nil
		default:
			return nil, fmt.Errorf("cloudcc %s user <projectPath> <encodedJson-or-userId>", action)
		}
	}
	if body, err := jsonx.ParseEncodedObject(args[0], "cloudcc "+action+" user"); err == nil {
		return normalizeUserRequestBody(action, body)
	}
	if action == "create" && len(args) >= 2 {
		form := map[string]any{
			"name":      strings.TrimSpace(args[0]),
			"profileId": strings.TrimSpace(args[1]),
		}
		if len(args) >= 3 && strings.TrimSpace(args[2]) != "" {
			form["email"] = strings.TrimSpace(args[2])
		}
		return userDataJsonBody(form, true)
	}
	id := strings.TrimSpace(args[0])
	switch action {
	case "detail", "view", "editInfo":
		return map[string]any{"userId": id, "id": id}, nil
	case "resetpw", "unlock", "unlocked", "unBindMfa", "mfa-unbind":
		return map[string]any{"userId": id, "id": id, "checkedid": id}, nil
	case "delete", "deactivate", "disable":
		return userDataJsonBody(map[string]any{"id": id, "isusing": "false"}, false)
	default:
		return nil, fmt.Errorf("cloudcc %s user expects an encoded JSON body", action)
	}
}

func normalizeUserRequestBody(action string, body map[string]any) (map[string]any, error) {
	switch action {
	case "create", "save":
		return normalizeUserSaveBody(body, true)
	case "update", "editSave":
		return normalizeUserSaveBody(body, false)
	case "delete", "deactivate", "disable":
		form := copyStringAnyMap(body)
		form["isusing"] = "false"
		return userDataJsonBody(form, false)
	default:
		return body, nil
	}
}

func normalizeUserSaveBody(body map[string]any, allowSendEmail bool) (map[string]any, error) {
	if _, exists := body["dataJson"]; exists {
		return body, nil
	}
	return userDataJsonBody(body, allowSendEmail)
}

func userDataJsonBody(form map[string]any, allowSendEmail bool) (map[string]any, error) {
	payload := copyStringAnyMap(form)
	sendEmail := false
	if allowSendEmail {
		sendEmail = truthy(payload["sendemail"]) || truthy(payload["sendEmail"]) || truthy(payload["isSendEmail"])
	}
	delete(payload, "sendemail")
	delete(payload, "sendEmail")
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	body := map[string]any{"dataJson": string(bodyBytes)}
	if allowSendEmail {
		body["sendemail"] = sendEmail
	}
	return body, nil
}

func truthy(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "1", "yes", "y":
			return true
		}
	case float64:
		return typed != 0
	case int:
		return typed != 0
	}
	return false
}

func handlePageLayout(action string, args []string, stdout io.Writer, _ io.Writer, cwd string) error {
	switch action {
	case "detail":
		return pageLayoutDetail(args, stdout, cwd)
	case "get":
		return pageLayoutList(args, stdout, cwd)
	case "update", "save":
		return pageLayoutSave(args, stdout, cwd)
	default:
		if _, ok := genericEndpoints["pagelayout"][action]; ok {
			return callGeneric(genericEndpoints["pagelayout"][action], action, "pagelayout", args, stdout, cwd)
		}
	}
	return fmt.Errorf("unsupported pagelayout action: %s", action)
}

func handleMenu(action string, args []string, stdout io.Writer, cwd string) error {
	if action != "update" {
		ep, ok := genericEndpoints["menu"][action]
		if !ok {
			return fmt.Errorf("unsupported menu action: %s", action)
		}
		return callGeneric(ep, action, "menu", args, stdout, cwd)
	}
	if len(args) < 2 {
		return fmt.Errorf("cloudcc update menu <projectPath> <encodedMenuJSON>")
	}
	projectPath := firstArg(args, cwd)
	body, err := jsonx.ParseEncodedObject(args[1], "cloudcc update menu")
	if err != nil {
		return err
	}
	menuID := strings.TrimSpace(fmt.Sprint(firstAny(body["id"], body["tabid"], body["tabId"])))
	if menuID == "" || menuID == "<nil>" {
		return fmt.Errorf("cloudcc update menu requires id, tabid, or tabId")
	}
	cfg, err := config.Load(projectPath)
	if err != nil {
		return err
	}
	var editResponse map[string]any
	if err := httpclient.New().PostClass(config.String(cfg, "setupSvc")+"/api/customTab/updatetab", map[string]any{"id": menuID}, config.String(cfg, "accessToken"), &editResponse); err != nil {
		return err
	}
	return postClass(stdout, cfg, "setup", "/api/customTab/updatesavetab", body)
}

func pageLayoutList(args []string, stdout io.Writer, cwd string) error {
	if len(args) < 2 {
		return fmt.Errorf("cloudcc get pagelayout <projectPath> <prefix>")
	}
	projectPath := firstArg(args, cwd)
	prefix := strings.TrimSpace(args[1])
	body := map[string]any{"prefix": prefix}
	cfg, err := config.Load(projectPath)
	if err != nil {
		return err
	}
	return postClass(stdout, cfg, "setup", "/api/layout/queryPageLayout", body)
}

func pageLayoutDetail(args []string, stdout io.Writer, cwd string) error {
	if len(args) < 3 {
		return fmt.Errorf("cloudcc detail pagelayout <projectPath> <objId> <layoutId> [type]")
	}
	projectPath := firstArg(args, cwd)
	body := map[string]any{
		"objId":    args[1],
		"layoutId": args[2],
	}
	if len(args) > 3 && strings.TrimSpace(args[3]) != "" {
		body["type"] = args[3]
	}
	cfg, err := config.Load(projectPath)
	if err != nil {
		return err
	}
	return postClass(stdout, cfg, "setup", "/api/modifyLayoutLightning/queryLayout", body)
}

func pageLayoutSave(args []string, stdout io.Writer, cwd string) error {
	if len(args) < 3 {
		return fmt.Errorf("cloudcc update pagelayout <projectPath> <layoutId> <encodedLayoutJSON>")
	}
	projectPath := firstArg(args, cwd)
	layoutId := args[1]
	layoutArg := args[2]
	layout, err := jsonx.ParseEncodedObject(layoutArg, "cloudcc update pagelayout")
	if err != nil {
		var raw map[string]any
		if err2 := json.Unmarshal([]byte(layoutArg), &raw); err2 == nil {
			layout = raw
		} else {
			return err
		}
	}
	layoutJSON, err := normalizePageLayoutJSON(layout, layoutId)
	if err != nil {
		return err
	}
	cfg, err := config.Load(projectPath)
	if err != nil {
		return err
	}
	body := map[string]any{
		"layoutId":   layoutId,
		"layoutJson": layoutJSON,
	}
	return postClass(stdout, cfg, "setup", "/api/modifyLayoutLightning/saveLayout", body)
}

func normalizePageLayoutJSON(layout map[string]any, layoutId string) (string, error) {
	rawSections, ok := layout["sections"]
	if !ok {
		switch data := layout["data"].(type) {
		case map[string]any:
			rawSections = data["sections"]
		default:
			rawSections = layout["data"]
		}
	}
	sections, ok := rawSections.([]any)
	if !ok {
		return "", fmt.Errorf("invalid layout payload: sections is required and must be an array")
	}
	for _, section := range sections {
		m, ok := section.(map[string]any)
		if !ok {
			return "", fmt.Errorf("invalid layout payload: each section must be an object")
		}
		sectionID := firstAny(m["sectionId"], m["sectionid"])
		if sectionID == nil {
			return "", fmt.Errorf("invalid layout payload: each section must include sectionId")
		}
		m["sectionId"] = sectionID
		delete(m, "sortOrder")
		delete(m, "categoriesAllowed")
		delete(m, "canChangeColumns")
		delete(m, "canDeleteSection")
	}
	if layoutId == "" {
		if v := firstAny(layout["layoutid"], layout["layoutId"]); v != nil && fmt.Sprint(v) != "" {
			layoutId = fmt.Sprint(v)
		}
		if data, ok := layout["data"].(map[string]any); ok {
			if v := firstAny(data["layoutid"], data["layoutId"]); v != nil && fmt.Sprint(v) != "" {
				layoutId = fmt.Sprint(v)
			}
		}
	}
	if layoutId == "" {
		return "", fmt.Errorf("layoutId is required")
	}
	b, err := json.Marshal(map[string]any{"sections": sections})
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func handleProject(action string, args []string, stderr io.Writer, cwd string) error {
	if action != "create" {
		return fmt.Errorf("unsupported project action: %s", action)
	}
	if len(args) < 1 || strings.TrimSpace(args[0]) == "" {
		return fmt.Errorf("cloudcc create project <name|.>")
	}
	target := strings.TrimSpace(args[0])
	if target == "." {
		target = cwd
	} else if !filepath.IsAbs(target) {
		target = filepath.Join(cwd, target)
	}
	if err := projecttemplates.WriteProject(target, filepath.Base(target)); err != nil {
		return err
	}
	fmt.Fprintf(stderr, "Created CloudCC project: %s\n", target)
	fmt.Fprintln(stderr, "Compatibility check pending: provide CloudCCDev/setupSvc and MetadataService config, then run cloudcc doctor provider <projectPath>.")
	return nil
}

func handleConfig(action string, args []string, stdout io.Writer, cwd string) error {
	projectPath := firstArg(args, cwd)
	switch action {
	case "get":
		cfg, err := config.Load(projectPath)
		if err != nil {
			return err
		}
		cfg["compatibility"] = compatibility.CheckAll(projectPath)
		return printJSON(stdout, cfg)
	case "use":
		if len(args) == 0 {
			return fmt.Errorf("cloudcc use config <env> [projectPath]")
		}
		projectPath = cwd
		if len(args) > 1 {
			projectPath = args[1]
		}
		return config.Use(projectPath, args[0])
	default:
		return fmt.Errorf("unsupported config action: %s", action)
	}
}

func callGeneric(ep endpoint, action string, resource string, args []string, stdout io.Writer, cwd string) error {
	return callGenericWithPrinter(ep, action, resource, args, stdout, cwd, postClass)
}

func callGenericRedacted(ep endpoint, action string, resource string, args []string, stdout io.Writer, cwd string) error {
	return callGenericWithPrinter(ep, action, resource, args, stdout, cwd, postClassRedacted)
}

func callGenericWithPrinter(ep endpoint, action string, resource string, args []string, stdout io.Writer, cwd string, printer func(io.Writer, config.Config, string, string, map[string]any) error) error {
	projectPath := firstArg(args, cwd)
	body := map[string]any{}
	if len(args) > 1 && args[1] != "" {
		parsed, err := jsonx.ParseEncodedObject(args[1], action+" "+resource)
		if err != nil {
			if action == "delete" {
				body = map[string]any{"id": args[1], resource + "id": args[1]}
			} else {
				return err
			}
		} else {
			body = parsed
		}
	}
	cfg, err := config.Load(projectPath)
	if err != nil {
		return err
	}
	return printer(stdout, cfg, ep.base, ep.path, body)
}

func isApiRegistrarRuntimeAction(action string) bool {
	switch action {
	case "debug", "test", "logs", "logDetail", "log-detail":
		return true
	default:
		return false
	}
}

func handleBrief(action string, args []string, stdout io.Writer, cwd string) error {
	if action != "get" {
		return fmt.Errorf("unsupported brief action: %s", action)
	}
	cfg, err := config.Load(firstArg(args, cwd))
	if err != nil {
		return err
	}
	var res map[string]any
	if err := httpclient.New().PostClass(config.String(cfg, "setupSvc")+"/api/customObject/newPage", map[string]any{"id": ""}, config.String(cfg, "accessToken"), &res); err != nil {
		return err
	}
	if data, _ := res["data"].(map[string]any); data != nil {
		if list, ok := data["objList"]; ok {
			return printJSON(stdout, list)
		}
	}
	return printJSON(stdout, res)
}

func handleObject(action string, args []string, stdout io.Writer, stderr io.Writer, cwd string) error {
	switch action {
	case "get":
		return objectGet(args, stdout, cwd)
	case "create":
		return objectCreate(args, stdout, stderr, cwd)
	case "update":
		return objectUpdate(args, stdout, cwd)
	case "delete":
		if len(args) < 2 {
			return fmt.Errorf("cloudcc delete object <projectPath> <objid>")
		}
		cfg, err := config.Load(args[0])
		if err != nil {
			return err
		}
		return postClass(stdout, cfg, "setup", "/api/customObject/deleteLogic", map[string]any{"objid": args[1]})
	case "purge":
		return objectPurge(args, stdout, cwd)
	default:
		return fmt.Errorf("unsupported object action: %s", action)
	}
}

func objectGet(args []string, stdout io.Writer, cwd string) error {
	projectPath := firstArg(args, cwd)
	kind := ""
	if len(args) > 1 {
		kind = args[1]
	}
	cfg, err := config.Load(projectPath)
	if err != nil {
		return err
	}
	client := httpclient.New()
	if kind == "deleted" || kind == "recycle" || kind == "recycle-bin" {
		var res map[string]any
		if err := client.PostClass(config.String(cfg, "setupSvc")+"/api/customObject/queryDeletedObjList", map[string]any{}, config.String(cfg, "accessToken"), &res); err != nil {
			return err
		}
		return printJSON(stdout, res)
	}
	var out []map[string]any
	if kind == "" || kind == "chat" || kind == "standard" {
		var res map[string]any
		if err := client.PostClass(config.String(cfg, "setupSvc")+"/api/customObject/standardObjList", map[string]any{}, config.String(cfg, "accessToken"), &res); err != nil {
			return err
		}
		if list, ok := res["data"].([]any); ok {
			for _, item := range list {
				if m, _ := item.(map[string]any); m != nil {
					out = append(out, map[string]any{
						"id":              m["id"],
						"label":           firstAny(m["objname"], m["label"]),
						"objname":         m["label"],
						"objprefix":       m["objprefix"],
						"schemetableName": m["label"],
						"apiname":         m["label"],
					})
				}
			}
		}
	}
	if kind == "" || kind == "chat" || kind == "custom" {
		var res map[string]any
		if err := client.PostClass(config.String(cfg, "setupSvc")+"/api/customObject/list", map[string]any{}, config.String(cfg, "accessToken"), &res); err != nil {
			return err
		}
		if data, _ := res["data"].(map[string]any); data != nil {
			if list, ok := data["objList"].([]any); ok {
				for _, item := range list {
					if m, _ := item.(map[string]any); m != nil {
						out = append(out, map[string]any{
							"id":              m["id"],
							"label":           firstAny(m["objLabel"], m["label"]),
							"objname":         firstAny(m["schemetable_name"], m["objname"]),
							"objprefix":       m["prefix"],
							"schemetableName": m["schemetableName"],
							"apiname":         m["schemetableName"],
						})
					}
				}
			}
		}
	}
	if kind == "trigger" {
		var res map[string]any
		if err := client.PostClass(config.String(cfg, "setupSvc")+"/api/trigger/newobjtrigger", map[string]any{}, config.String(cfg, "accessToken"), &res); err != nil {
			return err
		}
		if data, _ := res["data"].(map[string]any); data != nil {
			if list, ok := data["targetObjList"].([]any); ok {
				for _, item := range list {
					if m, _ := item.(map[string]any); m != nil {
						out = append(out, m)
					}
				}
			}
		}
	}
	return printJSON(stdout, out)
}

func objectPurge(args []string, stdout io.Writer, cwd string) error {
	if len(args) < 2 {
		return fmt.Errorf("cloudcc purge object <projectPath> <objid> [--execute --approval CLOUDCC_OBJECT_PHYSICAL_DELETE_APPROVED]")
	}
	projectPath := firstArg(args, cwd)
	objID := strings.TrimSpace(args[1])
	if objID == "" {
		return fmt.Errorf("cloudcc purge object: objid is required")
	}
	execute, approval := destructiveFlags(args[2:])
	body := map[string]any{"objid": objID}
	if !execute {
		return printJSON(stdout, map[string]any{
			"dryRun":             true,
			"operation":          "customObjectPhysicalDelete",
			"endpoint":           "/api/customObject/deletePhysics",
			"objid":              objID,
			"approvalRequired":   "CLOUDCC_OBJECT_PHYSICAL_DELETE_APPROVED",
			"executeFlag":        "--execute",
			"requestBodyPreview": body,
			"warning":            "This physically deletes a logically deleted custom object through setup-svc and cannot be rolled back by deleteLogicCancel.",
		})
	}
	if approval != "CLOUDCC_OBJECT_PHYSICAL_DELETE_APPROVED" {
		return fmt.Errorf("cloudcc purge object requires --approval CLOUDCC_OBJECT_PHYSICAL_DELETE_APPROVED when --execute is used")
	}
	cfg, err := config.Load(projectPath)
	if err != nil {
		return err
	}
	return postClass(stdout, cfg, "setup", "/api/customObject/deletePhysics", body)
}

func objectCreate(args []string, stdout io.Writer, stderr io.Writer, cwd string) error {
	var accessable string
	var err error
	args, accessable, err = objectCreateAccessableFlag(args)
	if err != nil {
		return err
	}
	if len(args) < 3 {
		return fmt.Errorf("cloudcc create object <projectPath> <label> [nameLabel] <businessDescription> [--accessable <0|1|2>]")
	}
	projectPath, label := args[0], args[1]
	nameLabel, remarkSuffix := "", ""
	if len(args) == 3 {
		nameLabel = labelToSlug(label) + "_custom_object"
		remarkSuffix = args[2]
	} else {
		nameLabel = args[2]
		remarkSuffix = args[3]
	}
	cfg, err := config.Load(projectPath)
	if err != nil {
		return err
	}
	profiles, err := loadProfileIDs(cfg)
	if err != nil {
		return err
	}
	perms := make([]map[string]any, 0, len(profiles))
	for i, id := range profiles {
		perm := "1,1,1,1,0,0"
		if i == 0 {
			perm = "1,1,1,1,1,1"
		}
		perms = append(perms, map[string]any{"profileid": id, "permission": perm})
	}
	body := map[string]any{
		"iscreatperm":     "true",
		"profileFieldArr": perms,
		"nameLabel":       nameLabel,
		"obj": map[string]any{
			"label":           label,
			"schemetableName": nameLabel,
			"dataType":        "V",
			"showFormat":      "{yyyy}{mm}{dd}{000}",
			"beginIndex":      "0",
			"isquickcreated":  "false",
			"islbs":           "false",
			"isreportcreated": "false",
			"remark":          fmt.Sprintf("用于管理与「%s」相关的业务数据 %s", label, strings.TrimSpace(remarkSuffix)),
		},
	}
	if accessable != "" {
		body["obj"].(map[string]any)["accessable"] = accessable
	}
	fmt.Fprintln(stderr, "Creating, please wait...")
	return postClass(stdout, cfg, "setup", "/api/customObject/saveButton", body)
}

func objectUpdate(args []string, stdout io.Writer, cwd string) error {
	if len(args) < 2 {
		return fmt.Errorf("cloudcc update object <projectPath> <encodedCustomObjectJSON>")
	}
	projectPath := firstArg(args, cwd)
	body, err := jsonx.ParseEncodedObject(args[1], "cloudcc update object")
	if err != nil {
		return err
	}
	objID := strings.TrimSpace(fmt.Sprint(firstAny(body["objid"], body["objId"], body["id"])))
	if objID == "" || objID == "<nil>" {
		return fmt.Errorf("cloudcc update object requires objid, objId, or id")
	}
	cfg, err := config.Load(projectPath)
	if err != nil {
		return err
	}
	var editResponse map[string]any
	if err := httpclient.New().PostClass(config.String(cfg, "setupSvc")+"/api/customObject/editPage", map[string]any{"objid": objID}, config.String(cfg, "accessToken"), &editResponse); err != nil {
		return err
	}
	return postClass(stdout, cfg, "setup", "/api/customObject/saveButton", body)
}

func loadProfileIDs(cfg config.Config) ([]string, error) {
	var res map[string]any
	if err := httpclient.New().PostClass(config.String(cfg, "setupSvc")+"/api/customObject/newPage", map[string]any{"id": ""}, config.String(cfg, "accessToken"), &res); err != nil {
		return nil, err
	}
	var ids []string
	if data, _ := res["data"].(map[string]any); data != nil {
		if list, ok := data["objList"].([]any); ok {
			for _, item := range list {
				if m, _ := item.(map[string]any); m != nil {
					if id := fmt.Sprint(m["id"]); id != "" && id != "<nil>" {
						ids = append(ids, id)
					}
				}
			}
		}
	}
	return ids, nil
}

func handleFieldsGet(args []string, stdout io.Writer, cwd string) error {
	if len(args) < 2 {
		return fmt.Errorf("cloudcc get fields <projectPath> <prefix>")
	}
	cfg, err := config.Load(args[0])
	if err != nil {
		return err
	}
	return postClass(stdout, cfg, "setup", "/api/fieldSetup/queryField", map[string]any{"prefix": args[1]})
}

func handleCodeResource(action string, resource string, dir string, apiName string, args []string, stdout io.Writer, stderr io.Writer, cwd string) error {
	switch action {
	case "create":
		return createJavaResource(dir, resource, args, stderr, cwd)
	case "publish":
		return publishJavaResource(dir, apiName, args, stdout, stderr, cwd)
	case "get", "pullList":
		projectPath := firstArg(args, cwd)
		cfg, err := config.Load(projectPath)
		if err != nil {
			return err
		}
		listPath := "/api/" + apiName + "/list"
		if action == "pullList" {
			listPath = "/api/" + apiName + "/detail"
		}
		body := map[string]any{}
		if len(args) > 1 {
			body["name"] = args[1]
		}
		return postClass(stdout, cfg, "setup", listPath, body)
	case "detail", "pull":
		if len(args) < 2 {
			return fmt.Errorf("cloudcc %s %s <projectPath> <name>", action, resource)
		}
		cfg, err := config.Load(args[0])
		if err != nil {
			return err
		}
		return postClass(stdout, cfg, "setup", "/api/"+apiName+"/detail", map[string]any{"name": args[1]})
	case "delete":
		if len(args) < 2 {
			return fmt.Errorf("cloudcc delete %s <projectPath> <id>", resource)
		}
		cfg, err := config.Load(args[0])
		if err != nil {
			return err
		}
		return postClass(stdout, cfg, "setup", "/api/"+apiName+"/delete", map[string]any{"id": args[1]})
	default:
		return fmt.Errorf("unsupported %s action: %s", resource, action)
	}
}

func backendResourcePath(projectPath string, dir string, parts ...string) string {
	all := append([]string{projectPath, "backend", dir}, parts...)
	return filepath.Join(all...)
}

func handleTrigger(action string, args []string, stdout io.Writer, stderr io.Writer, cwd string) error {
	switch action {
	case "create":
		if len(args) > 1 && looksLikeTriggerSpec(args[1]) {
			return triggerSaveSpec(action, args, stdout, cwd)
		}
		return createTriggerResource(args, stderr, cwd)
	case "publish":
		return publishTrigger(args, stdout, stderr, cwd)
	case "get":
		return triggerList(args, stdout, cwd)
	case "detail", "pull", "pullList":
		return triggerDetail(args, stdout, cwd)
	case "delete":
		return triggerDelete(args, stdout, cwd)
	case "update", "save":
		return triggerSaveSpec(action, args, stdout, cwd)
	default:
		return fmt.Errorf("unsupported trigger action: %s", action)
	}
}

func createJavaResource(dir string, resource string, args []string, stderr io.Writer, cwd string) error {
	if len(args) < 1 {
		return fmt.Errorf("cloudcc create %s <name>", resource)
	}
	name := filepath.Base(args[0])
	target := backendResourcePath(cwd, dir, args[0])
	if err := os.MkdirAll(target, 0755); err != nil {
		return err
	}
	pkg := dir
	if dir == "schedule" {
		pkg = "schedule"
	}
	source := fmt.Sprintf("package %s.%s;\n\n// Local editor package only; CloudCC injects the runtime package and imports at publish time.\n// @SOURCE_CONTENT_START\npublic class %s {\n    private final UserInfo userInfo;\n\n    public %s(UserInfo userInfo) {\n        this.userInfo = userInfo;\n    }\n\n    public Object execute() {\n        // TODO: implement business logic\n        return null;\n    }\n}\n// @SOURCE_CONTENT_END\n", pkg, name, name, name)
	if resource == "timer" {
		source = fmt.Sprintf("package schedule.%s;\n\nimport com.cloudcc.core.*;\n\npublic class %s extends CCSchedule {\n    public %s() {\n        // @SOURCE_CONTENT_START\n        // TODO: implement schedule logic\n        // @SOURCE_CONTENT_END\n    }\n}\n", name, name, name)
	}
	if err := os.WriteFile(filepath.Join(target, name+".java"), []byte(source), 0644); err != nil {
		return err
	}
	test := fmt.Sprintf("package %s.%s;\n\npublic class %sTest {\n}\n", pkg, name, name)
	_ = os.WriteFile(filepath.Join(target, name+"Test.java"), []byte(test), 0644)
	_ = jsonx.WriteObjectFile(filepath.Join(target, "config.json"), map[string]any{"name": name, "version": highCodeDefaultVersion, "interface": true})
	fmt.Fprintf(stderr, "Created %s resource: %s\n", resource, target)
	return nil
}

func publishJavaResource(dir string, apiName string, args []string, stdout io.Writer, stderr io.Writer, cwd string) error {
	if len(args) < 1 {
		return fmt.Errorf("cloudcc publish <resource> <name>")
	}
	name := filepath.Base(args[0])
	projectPath := cwd
	if len(args) > 1 && strings.TrimSpace(args[1]) != "" && !strings.HasPrefix(args[1], "--") {
		projectPath = args[1]
	}
	srcDir := backendResourcePath(projectPath, dir, args[0])
	sourceFile := filepath.Join(srcDir, name+".java")
	source, err := readMarkedSource(sourceFile)
	if err != nil {
		return err
	}
	source = strings.TrimSpace(source)
	cfgContent, _ := jsonx.ReadObjectFile(filepath.Join(srcDir, "config.json"))
	if cfgContent == nil {
		cfgContent = map[string]any{}
	}
	cfg, err := config.Load(projectPath)
	if err != nil {
		return err
	}
	timerID := strings.TrimSpace(fmt.Sprint(configID(cfgContent)))
	operationEdit := timerID != "" && timerID != "<nil>"
	var preSaveDetail map[string]any
	if operationEdit {
		preSaveDetail, _ = setupSvcDetail(cfg, "/api/"+apiName+"/detail", timerID, "timer detail")
	}
	publishVersion := highCodePublishVersion(cfgContent, preSaveDetail, operationEdit)
	endpoint := "/api/" + apiName + "/save"
	body := map[string]any{
		"id":       configID(cfgContent),
		"name":     name,
		"source":   encodeJavaURLDecoderComponent(source),
		"folderId": "wgd",
	}
	putHighCodeVersion(body, publishVersion)
	remoteValidation, err := validateRemoteCustomCode(cfg, "timer", name, "/api/"+apiName+"/validate", body)
	if err != nil {
		_ = writeJSON(stdout, map[string]any{"status": "blocked_remote_validation", "resource": "timer", "name": name, "remoteValidation": remoteValidation})
		return err
	}
	fmt.Fprintln(stderr, "Remote CloudCC timer validation passed; posting, please wait...")
	var saveResponse map[string]any
	if err := postClassResponse(cfg, "setup", endpoint, body, &saveResponse); err != nil {
		return err
	}
	if message := cloudCCResponseFailure(saveResponse); message != "" {
		_ = writeJSON(stdout, map[string]any{"status": "publish_failed", "resource": "timer", "name": name, "remoteValidation": remoteValidation, "saveResponse": saveResponse})
		return responseFailureError("CloudCC timer save failed", saveResponse)
	}
	if id := recursiveFirstStringValue(saveResponse, "id"); id != "" {
		cfgContent["id"] = id
		savedVersion := publishVersion
		if detail, detailErr := setupSvcDetail(cfg, "/api/"+apiName+"/detail", id, "timer detail"); detailErr == nil {
			if version := highCodeRecordVersion(detail); version != "" {
				savedVersion = version
			}
		}
		if savedVersion != "" {
			cfgContent["version"] = savedVersion
		}
		_ = jsonx.WriteObjectFile(filepath.Join(srcDir, "config.json"), cfgContent)
	}
	return writeJSON(stdout, map[string]any{"status": "published", "resource": "timer", "name": name, "remoteValidation": remoteValidation, "saveResponse": saveResponse})
}

func publishClassResource(args []string, stdout io.Writer, stderr io.Writer, cwd string) error {
	name, opts, err := parseNamedClassDevOptions(args, cwd)
	if err != nil {
		return fmt.Errorf("cloudcc publish classes <name> [projectPath]: %w", err)
	}
	srcDir := backendResourcePath(opts.ProjectPath, "classes", name)
	source, err := readMarkedSource(filepath.Join(srcDir, name+".java"))
	if err != nil {
		return err
	}
	source = strings.TrimSpace(source)
	validation := classValidationResult{}
	if opts.ValidationEvidence != "" {
		b, readErr := os.ReadFile(opts.ValidationEvidence)
		if readErr != nil {
			return fmt.Errorf("cannot read validation evidence: %w", readErr)
		}
		if decodeErr := json.Unmarshal(b, &validation); decodeErr != nil {
			return fmt.Errorf("invalid validation evidence: %w", decodeErr)
		}
		if !validation.Valid || validation.ClassName != name || validation.SourceSHA256 != sourceDigest(source) {
			return fmt.Errorf("validation evidence is not passed or does not match the current %s source", name)
		}
	} else {
		validation, err = validateClass(name, opts)
		if err != nil {
			_ = writeJSON(stdout, map[string]any{"status": "blocked_local_validation", "validation": validation})
			return err
		}
	}
	cfgContent, _ := jsonx.ReadObjectFile(filepath.Join(srcDir, "config.json"))
	if cfgContent == nil {
		cfgContent = map[string]any{}
	}
	cfg, err := config.Load(opts.ProjectPath)
	if err != nil {
		return err
	}
	publishURL, err := classPublishBaseURL(cfg)
	if err != nil {
		return err
	}
	accessToken := firstNonBlankString(strings.TrimSpace(os.Getenv("CLOUDCC_ACCESS_TOKEN")), config.String(cfg, "accessToken"))
	if accessToken == "" {
		return fmt.Errorf("CloudCC publish authentication is missing; configure CloudCCDev credentials or CLOUDCC_ACCESS_TOKEN")
	}
	classID := strings.TrimSpace(fmt.Sprint(configID(cfgContent)))
	if classID == "" {
		classID, err = lookupClassID(publishURL, accessToken, name)
		if err != nil {
			return fmt.Errorf("cannot establish idempotent class publish target: %w", err)
		}
	}
	var preSaveDetail map[string]any
	if classID != "" {
		preSaveDetail, _ = classDetail(publishURL, accessToken, classID)
	}
	publishVersion := highCodePublishVersion(cfgContent, preSaveDetail, classID != "")
	body := map[string]any{
		"id":       classID,
		"name":     name,
		"source":   encodeJavaURLDecoderComponent(source),
		"folderId": "wgd",
	}
	putHighCodeVersion(body, publishVersion)
	remoteValidation, err := validateRemoteCustomCodeWithBase(publishURL, accessToken, "classes", name, "/api/ccfag/validate", body)
	if err != nil {
		_ = writeJSON(stdout, map[string]any{"status": "blocked_remote_validation", "resource": "classes", "name": name, "localValidation": validation, "remoteValidation": remoteValidation})
		return err
	}
	endpoint := publishURL + "/api/ccfag/save"
	fmt.Fprintln(stderr, "Local and remote CloudCC class validation passed; publishing through the target gateway and reading back...")
	var saveResponse map[string]any
	if err := httpclient.New().PostClass(endpoint, body, accessToken, &saveResponse); err != nil {
		return err
	}
	if message := cloudCCResponseFailure(saveResponse); message != "" {
		_ = writeJSON(stdout, map[string]any{"status": "publish_failed", "validation": validation, "remoteValidation": remoteValidation, "saveResponse": saveResponse})
		return fmt.Errorf("CloudCC class save failed: %s", message)
	}
	classID = firstNonBlankString(
		responseDataString(saveResponse),
		recursiveStringValue(saveResponse, "id"),
		classID,
	)
	if classID == "" {
		_ = writeJSON(stdout, map[string]any{"status": "readback_failed", "validation": validation, "remoteValidation": remoteValidation, "saveResponse": saveResponse})
		return fmt.Errorf("class saved but response did not return the class id required for readback")
	}
	var detailResponse map[string]any
	if err := httpclient.New().PostClass(publishURL+"/api/ccfag/detail", map[string]any{"id": classID}, accessToken, &detailResponse); err != nil {
		return fmt.Errorf("class saved but readback failed: %w", err)
	}
	readbackSource := recursiveFirstStringValue(detailResponse, "source", "triggerSource", "sourcecode")
	if readbackSource == "" {
		_ = writeJSON(stdout, map[string]any{"status": "readback_failed", "validation": validation, "remoteValidation": remoteValidation, "saveResponse": saveResponse, "detailResponse": detailResponse})
		return fmt.Errorf("class saved but detail readback did not return source")
	}
	readbackDigest := sourceDigest(readbackSource)
	if readbackDigest != validation.SourceSHA256 {
		_ = writeJSON(stdout, map[string]any{"status": "readback_mismatch", "validation": validation, "remoteValidation": remoteValidation, "publishedSourceSha256": validation.SourceSHA256, "readbackSourceSha256": readbackDigest})
		return fmt.Errorf("class saved but readback source does not match the locally validated source")
	}
	cfgContent["id"] = classID
	savedVersion := firstNonBlankString(highCodeRecordVersion(detailResponse), publishVersion)
	if savedVersion != "" {
		cfgContent["version"] = savedVersion
	}
	if err := jsonx.WriteObjectFile(filepath.Join(srcDir, "config.json"), cfgContent); err != nil {
		return fmt.Errorf("class published but local config id could not be updated: %w", err)
	}
	return writeJSON(stdout, map[string]any{
		"status":               "published_and_verified",
		"className":            name,
		"publishGateway":       publishURL,
		"sourceSha256":         validation.SourceSHA256,
		"localValidation":      validation,
		"remoteValidation":     remoteValidation,
		"saveResponse":         saveResponse,
		"readbackSourceSha256": readbackDigest,
	})
}

const highCodeDefaultVersion = "3"

func highCodePublishVersion(cfgContent map[string]any, currentRecord any, operationEdit bool) string {
	if !operationEdit {
		return highCodeDefaultVersion
	}
	if version := highCodeRecordVersion(currentRecord); version != "" {
		return version
	}
	if currentRecord != nil {
		return "2"
	}
	return firstNonBlankString(anyString(cfgContent["version"]), anyString(cfgContent["Version"]))
}

func highCodeRecordVersion(record any) string {
	return recursiveFirstStringValue(record, "version")
}

func putHighCodeVersion(body map[string]any, version string) {
	version = strings.TrimSpace(version)
	if version != "" {
		body["version"] = version
	}
}

func setupSvcDetail(cfg config.Config, path string, id string, label string) (map[string]any, error) {
	var response map[string]any
	if err := postClassResponse(cfg, "setup", path, map[string]any{"id": id}, &response); err != nil {
		return nil, err
	}
	if message := cloudCCResponseFailure(response); message != "" {
		return nil, fmt.Errorf("%s failed: %s", label, message)
	}
	return response, nil
}

func classDetail(setupURL string, accessToken string, id string) (map[string]any, error) {
	var response map[string]any
	if err := httpclient.New().PostClass(setupURL+"/api/ccfag/detail", map[string]any{"id": id}, accessToken, &response); err != nil {
		return nil, err
	}
	if message := cloudCCResponseFailure(response); message != "" {
		return nil, fmt.Errorf("class detail failed: %s", message)
	}
	return response, nil
}

func lookupClassID(setupURL string, accessToken string, name string) (string, error) {
	var response map[string]any
	body := map[string]any{
		"sname": name, "fid": "", "shownum": "100", "showpage": "1",
		"rptcond": "lastmodifydate", "rptorder": "desc",
	}
	if err := httpclient.New().PostClass(setupURL+"/api/ccfag/list", body, accessToken, &response); err != nil {
		return "", err
	}
	if message := cloudCCResponseFailure(response); message != "" {
		return "", fmt.Errorf("class list failed: %s", message)
	}
	data, _ := response["data"].(map[string]any)
	items, _ := data["list"].([]any)
	var matches []string
	for _, item := range items {
		record, _ := item.(map[string]any)
		recordName := recursiveFirstStringValue(record, "name", "apiname", "apiName")
		if strings.EqualFold(strings.TrimSpace(recordName), strings.TrimSpace(name)) {
			if id := recursiveStringValue(record, "id"); id != "" {
				matches = append(matches, id)
			}
		}
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("multiple existing classes named %s were returned; refusing to choose one", name)
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	return "", nil
}

func cloudCCResponseFailure(response map[string]any) string {
	for _, key := range []string{"success", "result"} {
		if value, exists := response[key]; exists {
			if value == false || strings.EqualFold(fmt.Sprint(value), "false") {
				return firstNonBlankString(recursiveStringValue(response, "returnInfo"), recursiveStringValue(response, "message"), recursiveStringValue(response, "msg"), key+"=false")
			}
		}
	}
	if code := firstNonBlankString(recursiveStringValue(response, "returnCode"), recursiveStringValue(response, "code")); code != "" && code != "200" && code != "1" && code != "0" {
		return firstNonBlankString(recursiveStringValue(response, "returnInfo"), recursiveStringValue(response, "message"), recursiveStringValue(response, "msg"), "code="+code)
	}
	return ""
}

func validateRemoteCustomCode(cfg config.Config, resource string, name string, path string, body map[string]any) (map[string]any, error) {
	base := strings.TrimRight(config.String(cfg, "setupSvc"), "/")
	accessToken := firstNonBlankString(strings.TrimSpace(os.Getenv("CLOUDCC_ACCESS_TOKEN")), config.String(cfg, "accessToken"))
	return validateRemoteCustomCodeWithBase(base, accessToken, resource, name, path, body)
}

func validateRemoteCustomCodeWithBase(base string, accessToken string, resource string, name string, path string, body map[string]any) (map[string]any, error) {
	var response map[string]any
	if err := httpclient.New().PostClass(strings.TrimRight(base, "/")+path, body, accessToken, &response); err != nil {
		return response, err
	}
	if message := cloudCCResponseFailure(response); message != "" || !remoteValidationDataValid(response) {
		if message == "" {
			message = firstNonBlankString(recursiveStringValue(response, "message"), "validation failed")
		}
		return response, responseFailureError("Remote "+resource+" validation failed", response)
	}
	return response, nil
}

func remoteValidationDataValid(response map[string]any) bool {
	data, _ := response["data"].(map[string]any)
	if data == nil {
		return true
	}
	if value, exists := data["valid"]; exists {
		switch v := value.(type) {
		case bool:
			return v
		case string:
			return strings.EqualFold(strings.TrimSpace(v), "true")
		default:
			return strings.EqualFold(strings.TrimSpace(fmt.Sprint(v)), "true")
		}
	}
	return true
}

func responseFailureError(label string, response map[string]any) error {
	raw, _ := json.Marshal(response)
	message := firstNonBlankString(
		recursiveStringValue(response, "returnInfo"),
		recursiveStringValue(response, "message"),
		recursiveStringValue(response, "msg"),
		recursiveStringValue(response, "errormsg"),
		"unknown error",
	)
	return fmt.Errorf("%s: %s; responseBody=%s", label, message, string(raw))
}

func copyStringAnyMap(input map[string]any) map[string]any {
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func responseDataString(response map[string]any) string {
	value, exists := response["data"]
	if !exists || value == nil {
		return ""
	}
	if _, nested := value.(map[string]any); nested {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func recursiveFirstStringValue(value any, keys ...string) string {
	for _, key := range keys {
		if found := recursiveStringValue(value, key); found != "" {
			return found
		}
	}
	return ""
}

func recursiveStringValue(value any, key string) string {
	switch typed := value.(type) {
	case map[string]any:
		if direct, exists := typed[key]; exists && direct != nil && fmt.Sprint(direct) != "<nil>" {
			if text, ok := direct.(string); ok {
				return text
			}
			if _, nested := direct.(map[string]any); !nested {
				return fmt.Sprint(direct)
			}
		}
		for _, childKey := range []string{"data", "body", "resultData", "trigger", "record", "item"} {
			if found := recursiveStringValue(typed[childKey], key); found != "" {
				return found
			}
		}
	case []any:
		for _, item := range typed {
			if found := recursiveStringValue(item, key); found != "" {
				return found
			}
		}
	}
	return ""
}

func handleScript(action string, args []string, stdout io.Writer, stderr io.Writer, cwd string) error {
	switch action {
	case "create":
		if len(args) < 1 {
			return fmt.Errorf("cloudcc create script <object/name>")
		}
		name := filepath.Base(args[0])
		target := filepath.Join(cwd, "script", args[0])
		if err := os.MkdirAll(target, 0755); err != nil {
			return err
		}
		body := "function main($CCDK, obj) {\n    // TODO: implement client script\n}\n"
		if err := os.WriteFile(filepath.Join(target, name+".js"), []byte(body), 0644); err != nil {
			return err
		}
		return jsonx.WriteObjectFile(filepath.Join(target, "config.json"), map[string]any{"name": name})
	case "publish":
		if len(args) < 1 {
			return fmt.Errorf("cloudcc publish script <object/name>")
		}
		name := filepath.Base(args[0])
		srcDir := filepath.Join(cwd, "script", args[0])
		content, err := os.ReadFile(filepath.Join(srcDir, name+".js"))
		if err != nil {
			return err
		}
		bodyText, err := extractMainBody(string(content))
		if err != nil {
			return err
		}
		local, _ := jsonx.ReadObjectFile(filepath.Join(srcDir, "config.json"))
		local["scriptContent"] = strings.TrimSpace(bodyText)
		cfg, err := config.Load(cwd)
		if err != nil {
			return err
		}
		var res map[string]any
		if err := httpclient.New().PostEnvelope(baseURL(cfg)+"/devconsole/script/saveClientScript", local, map[string]any(cfg), &res); err != nil {
			return err
		}
		return printJSON(stdout, res)
	case "get", "pull", "pullList", "detail", "delete":
		return callDevConsoleScript(action, args, stdout, cwd)
	default:
		return fmt.Errorf("unsupported script action: %s", action)
	}
}

func callDevConsoleScript(action string, args []string, stdout io.Writer, cwd string) error {
	projectPath := firstArg(args, cwd)
	body := map[string]any{}
	if len(args) > 1 {
		parsed, err := jsonx.ParseEncodedObject(args[1], action+" script")
		if err == nil {
			body = parsed
		}
	}
	cfg, err := config.Load(projectPath)
	if err != nil {
		return err
	}
	var res map[string]any
	if err := httpclient.New().PostEnvelope(baseURL(cfg)+"/devconsole/script/pageClientScript", body, map[string]any(cfg), &res); err != nil {
		return err
	}
	return printJSON(stdout, res)
}

func handleHTML(action string, args []string, stdout io.Writer, stderr io.Writer, cwd string) error {
	switch action {
	case "create":
		if len(args) < 1 {
			return fmt.Errorf("cloudcc create html <apiName>")
		}
		apiName := args[0]
		target := filepath.Join(cwd, "html", apiName)
		if err := os.MkdirAll(target, 0755); err != nil {
			return err
		}
		_ = os.WriteFile(filepath.Join(target, "index.html"), []byte("<!doctype html>\n<html><body></body></html>\n"), 0644)
		return jsonx.WriteObjectFile(filepath.Join(target, "config.json"), map[string]any{"apiName": apiName, "htmlLabel": apiName})
	case "publish":
		if len(args) < 1 {
			return fmt.Errorf("cloudcc publish html <apiName> [projectPath]")
		}
		projectPath := cwd
		if len(args) > 1 {
			projectPath = args[1]
		}
		apiName := args[0]
		dir := filepath.Join(projectPath, "html", apiName)
		local, err := jsonx.ReadObjectFile(filepath.Join(dir, "config.json"))
		if err != nil {
			return err
		}
		html, err := os.ReadFile(filepath.Join(dir, "index.html"))
		if err != nil {
			return err
		}
		local["htmlContent"] = string(html)
		cfg, err := config.Load(projectPath)
		if err != nil {
			return err
		}
		var res map[string]any
		if err := httpclient.New().PostRaw(baseURL(cfg)+"/devconsole/htmlComponent/saveHtmlComponent", local, map[string]string{"accessToken": config.String(cfg, "pluginToken")}, &res); err != nil {
			return err
		}
		return printJSON(stdout, res)
	default:
		return fmt.Errorf("unsupported html action: %s", action)
	}
}

func handleStaticResource(action string, args []string, stdout io.Writer, stderr io.Writer, cwd string) error {
	switch action {
	case "create":
		if len(args) < 2 {
			return fmt.Errorf("cloudcc create staticResource <name> <filePath>")
		}
		cfg, err := config.Load(cwd)
		if err != nil {
			return err
		}
		body := map[string]any{"name": args[0], "filePath": args[1]}
		return postClass(stdout, cfg, "setup", "/api/staticResource/save", body)
	case "get":
		return callGeneric(endpoint{"setup", "/api/staticResource/list"}, action, "staticResource", args, stdout, cwd)
	case "detail":
		return callGeneric(endpoint{"setup", "/api/staticResource/detail"}, action, "staticResource", args, stdout, cwd)
	case "delete":
		return callGeneric(endpoint{"setup", "/api/staticResource/delete"}, action, "staticResource", args, stdout, cwd)
	case "count":
		return callGeneric(endpoint{"setup", "/api/staticResource/count"}, action, "staticResource", args, stdout, cwd)
	default:
		return fmt.Errorf("unsupported staticResource action: %s", action)
	}
}

func postClass(stdout io.Writer, cfg config.Config, base string, apiPath string, body map[string]any) error {
	var res map[string]any
	if err := postClassResponse(cfg, base, apiPath, body, &res); err != nil {
		return err
	}
	return printJSON(stdout, res)
}

func postClassRedacted(stdout io.Writer, cfg config.Config, base string, apiPath string, body map[string]any) error {
	var res map[string]any
	if err := postClassResponse(cfg, base, apiPath, body, &res); err != nil {
		return err
	}
	return printJSONNoHTMLEscape(stdout, redactApiRegistrarLogValue(res))
}

func postClassResponse(cfg config.Config, base string, apiPath string, body map[string]any, res *map[string]any) error {
	var svc string
	if base == "api" {
		svc = strings.TrimRight(config.String(cfg, "apiSvc"), "/")
	} else {
		svc = strings.TrimRight(config.String(cfg, "setupSvc"), "/")
	}
	accessToken := firstNonBlankString(strings.TrimSpace(os.Getenv("CLOUDCC_ACCESS_TOKEN")), config.String(cfg, "accessToken"))
	if err := httpclient.New().PostClass(svc+apiPath, body, accessToken, res); err != nil {
		return err
	}
	return nil
}

func destructiveFlags(args []string) (bool, string) {
	execute := false
	approval := ""
	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		switch {
		case arg == "--execute":
			execute = true
		case arg == "--approval" && i+1 < len(args):
			i++
			approval = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--approval="):
			approval = strings.TrimSpace(strings.TrimPrefix(arg, "--approval="))
		}
	}
	return execute, approval
}

func objectCreateAccessableFlag(args []string) ([]string, string, error) {
	filtered := make([]string, 0, len(args))
	accessable := ""
	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		switch {
		case arg == "--accessable":
			if i+1 >= len(args) {
				return nil, "", fmt.Errorf("cloudcc create object --accessable requires one of 0, 1, or 2")
			}
			i++
			value, err := normalizeObjectCreateAccessable(args[i])
			if err != nil {
				return nil, "", err
			}
			accessable = value
		case strings.HasPrefix(arg, "--accessable="):
			value, err := normalizeObjectCreateAccessable(strings.TrimPrefix(arg, "--accessable="))
			if err != nil {
				return nil, "", err
			}
			accessable = value
		default:
			filtered = append(filtered, args[i])
		}
	}
	return filtered, accessable, nil
}

func normalizeObjectCreateAccessable(value string) (string, error) {
	value = strings.TrimSpace(value)
	switch value {
	case "0", "1", "2":
		return value, nil
	default:
		return "", fmt.Errorf("cloudcc create object --accessable only supports 0 (private), 1 (public read), or 2 (public read/write)")
	}
}

func readMarkedSource(file string) (string, error) {
	b, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	text := string(b)
	re := regexp.MustCompile(`(?s)//\s*@SOURCE_CONTENT_START\r?\n(.*?)\r?\n\s*//\s*@SOURCE_CONTENT_END`)
	if m := re.FindStringSubmatch(text); len(m) > 1 {
		return m[1], nil
	}
	return text, nil
}

func extractMainBody(content string) (string, error) {
	idx := strings.Index(content, "function main")
	if idx < 0 {
		return "", fmt.Errorf("no function main($CCDK, obj) found")
	}
	open := strings.Index(content[idx:], "{")
	if open < 0 {
		return "", fmt.Errorf("main function has no body")
	}
	start := idx + open + 1
	depth := 1
	inSingle, inDouble, inTemplate, escaped := false, false, false, false
	for i := start; i < len(content); i++ {
		ch := content[i]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' {
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
				return content[start:i], nil
			}
		}
	}
	return "", fmt.Errorf("main function braces are not balanced")
}

func configID(data map[string]any) any {
	if v := data["id"]; v != nil {
		return v
	}
	return ""
}

func baseURL(cfg config.Config) string {
	if v := config.String(cfg, "baseUrl"); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "https://developer.apis.cloudcc.cn"
}

func printJSON(w io.Writer, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(b))
	return err
}

func printJSONNoHTMLEscape(w io.Writer, v any) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(v)
}

func firstArg(args []string, fallback string) string {
	if len(args) == 0 || args[0] == "" {
		return fallback
	}
	return args[0]
}

func firstAny(values ...any) any {
	for _, v := range values {
		if v != nil && fmt.Sprint(v) != "" && fmt.Sprint(v) != "<nil>" {
			return v
		}
	}
	return nil
}

func labelToSlug(label string) string {
	s := strings.ToLower(strings.TrimSpace(label))
	re := regexp.MustCompile(`[^a-z0-9]+`)
	s = re.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	if s == "" {
		return "custom_object"
	}
	return s
}
