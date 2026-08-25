package msapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

var lowCodeShortcutDomains = map[string]string{
	"approval":         "approval-processes",
	"approvalProcess":  "approval-processes",
	"application":      "applications",
	"button":           "buttons",
	"customSetting":    "custom-settings",
	"dupeCatcher":      "dupe-catchers",
	"dashboard":        "dashboards",
	"fields":           "fields",
	"globalSelectList": "global-select-lists",
	"identityProvider": "identity-providers",
	"menu":             "menus",
	"object":           "objects",
	"pagelayout":       "layouts",
	"permission":       "permissions",
	"profile":          "profiles",
	"recordType":       "record-types",
	"report":           "reports",
	"reportMatrix":     "reports",
	"reportRatio":      "reports",
	"reportSummary":    "reports",
	"reportTabular":    "reports",
	"reports":          "reports",
	"reportFolder":     "reports",
	"role":             "roles",
	"sharingRule":      "sharing-rules",
	"singleSignOn":     "single-sign-ons",
	"validationRule":   "validation-rules",
	"view":             "object-views",
	"workflow":         "workflows",
	"workflowRule":     "workflows",
}

var lowCodeShortcutActions = map[string]bool{
	"get":         true,
	"detail":      true,
	"getList":     true,
	"newInfo":     true,
	"editInfo":    true,
	"validDelete": true,
	"create":      true,
	"update":      true,
	"save":        true,
	"modify":      true,
	"editSave":    true,
	"assign":      true,
	"add":         true,
	"remove":      true,
	"delete":      true,
	"enable":      true,
	"disable":     true,
	"activate":    true,
	"deactivate":  true,
	"purge":       true,
}

// IsLowCodeShortcut returns true for legacy low-code metadata CLI shortcuts that
// must no longer fall through to setup-svc backed modules.
func IsLowCodeShortcut(action string, resource string) bool {
	if !lowCodeShortcutActions[strings.TrimSpace(action)] {
		return false
	}
	_, ok := lowCodeShortcutDomains[strings.TrimSpace(resource)]
	return ok
}

// HandleLowCodeShortcut converts legacy low-code metadata shortcuts into the
// MetadataService channel. Writes create plans only; callers must apply the plan
// explicitly with cloudcc apply msapi <planId>.
func HandleLowCodeShortcut(action string, resource string, args []string, stdout io.Writer, cwd string) error {
	domain, ok := lowCodeShortcutDomains[strings.TrimSpace(resource)]
	if !ok {
		return fmt.Errorf("unsupported low-code metadata shortcut resource: %s", resource)
	}
	projectPath, rest := shortcutProjectPath(args, cwd)
	if resource == "profile" {
		return handleProfileShortcut(action, projectPath, rest, stdout, cwd)
	}
	if resource == "permission" && isShortcutRead(action) {
		return handlePermissionSetReadShortcut(action, projectPath, rest, stdout, cwd)
	}
	if resource == "role" && isShortcutRead(action) {
		return handleRoleReadShortcut(action, projectPath, rest, stdout, cwd)
	}
	if resource == "sharingRule" && isShortcutRead(action) {
		return handleSharingRuleReadShortcut(action, projectPath, rest, stdout, cwd)
	}
	if resource == "validationRule" && isShortcutRead(action) {
		return handleValidationRuleReadShortcut(action, projectPath, rest, stdout, cwd)
	}
	if (resource == "workflow" || resource == "workflowRule") && isShortcutRead(action) {
		return handleWorkflowReadShortcut(action, projectPath, rest, stdout, cwd)
	}
	if resource == "object" && isShortcutRead(action) {
		return handleObjectReadShortcut(action, projectPath, rest, stdout, cwd)
	}
	if resource == "fields" && isShortcutRead(action) {
		return handleFieldsReadShortcut(action, projectPath, rest, stdout, cwd)
	}
	if resource == "globalSelectList" && isShortcutRead(action) {
		return handleGlobalSelectReadShortcut(action, projectPath, rest, stdout, cwd)
	}
	if resource == "recordType" && isShortcutRead(action) {
		return handleRecordTypeReadShortcut(action, projectPath, rest, stdout, cwd)
	}
	if resource == "pagelayout" && isShortcutRead(action) {
		return handlePageLayoutReadShortcut(action, projectPath, rest, stdout, cwd)
	}
	if resource == "view" && isShortcutRead(action) {
		return handleObjectViewReadShortcut(action, projectPath, rest, stdout, cwd)
	}
	if resource == "reportMatrix" || resource == "reportRatio" ||
		resource == "reportSummary" || resource == "reportTabular" {
		return handleTypedReportShortcut(action, resource, projectPath, rest, stdout, cwd)
	}
	if resource == "report" || resource == "reports" {
		return handleReportShortcut(action, projectPath, rest, stdout, cwd)
	}
	if resource == "reportFolder" {
		return handleReportFolderShortcut(action, projectPath, rest, stdout, cwd)
	}
	if isShortcutRead(action) {
		return Handle("scan", "msapi", []string{projectPath, "standard-catalog"}, stdout, cwd)
	}
	spec, operation, err := shortcutPlanSpec(action, resource, rest)
	if err != nil {
		return err
	}
	body, err := json.Marshal(spec)
	if err != nil {
		return err
	}
	return Handle("plan", "msapi", []string{projectPath, domain, string(body), operation}, stdout, cwd)
}

func handleObjectViewReadShortcut(action string, projectPath string, args []string, stdout io.Writer, cwd string) error {
	c, _, err := newClient([]string{projectPath}, cwd)
	if err != nil {
		return err
	}
	if action == "detail" || action == "editInfo" || (action == "get" && len(args) == 1) {
		if len(args) != 1 || strings.TrimSpace(args[0]) == "" {
			return fmt.Errorf("cloudcc %s view <projectPath> <view-id>", action)
		}
		return c.writeJSON(stdout, http.MethodPost, "/metadata/v1/object-views:detail", map[string]any{"id": strings.TrimSpace(args[0])})
	}
	if len(args) > 1 {
		return fmt.Errorf("cloudcc %s view <projectPath> [object-id-or-apiName]", action)
	}
	body := map[string]any{}
	if len(args) == 1 && strings.TrimSpace(args[0]) != "" {
		body["objectId"] = strings.TrimSpace(args[0])
	}
	return c.writeJSON(stdout, http.MethodPost, "/metadata/v1/object-views:query", body)
}

func handleObjectReadShortcut(action string, projectPath string, args []string, stdout io.Writer, cwd string) error {
	c, _, err := newClient([]string{projectPath}, cwd)
	if err != nil {
		return err
	}
	if action == "getList" || action == "newInfo" || len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return c.getJSON(stdout, "/metadata/v1/scans/standard-catalog")
	}
	if len(args) > 1 {
		return fmt.Errorf("cloudcc %s object <projectPath> [id-or-apiName]", action)
	}
	return c.getJSON(stdout, "/metadata/v1/objects/"+url.PathEscape(strings.TrimSpace(args[0])))
}

func handleFieldsReadShortcut(action string, projectPath string, args []string, stdout io.Writer, cwd string) error {
	if len(args) != 1 || strings.TrimSpace(args[0]) == "" {
		return fmt.Errorf("cloudcc %s fields <projectPath> <object-id-apiName-or-prefix>", action)
	}
	c, _, err := newClient([]string{projectPath}, cwd)
	if err != nil {
		return err
	}
	return c.getJSON(stdout, "/metadata/v1/fields?object="+url.QueryEscape(strings.TrimSpace(args[0])))
}

func handleValidationRuleReadShortcut(action string, projectPath string, args []string, stdout io.Writer, cwd string) error {
	action = strings.TrimSpace(action)
	c, _, err := newClient([]string{projectPath}, cwd)
	if err != nil {
		return err
	}
	if action == "detail" || action == "editInfo" || action == "validDelete" {
		if len(args) != 1 || strings.TrimSpace(args[0]) == "" {
			return fmt.Errorf("cloudcc %s validationRule <projectPath> <rule-id>", action)
		}
		return c.getJSON(stdout, "/metadata/v1/validation-rules/"+url.PathEscape(strings.TrimSpace(args[0])))
	}
	if len(args) < 1 || len(args) > 2 || strings.TrimSpace(args[0]) == "" {
		return fmt.Errorf("cloudcc %s validationRule <projectPath> <object-id-apiName-or-prefix> [filter-or-rule-selector]", action)
	}
	values := url.Values{}
	values.Set("object", strings.TrimSpace(args[0]))
	if len(args) == 2 && strings.TrimSpace(args[1]) != "" {
		values.Set("filter", strings.TrimSpace(args[1]))
	}
	return c.getJSON(stdout, "/metadata/v1/validation-rules?"+values.Encode())
}

func handleGlobalSelectReadShortcut(action string, projectPath string, args []string, stdout io.Writer, cwd string) error {
	action = strings.TrimSpace(action)
	if action == "detail" || action == "editInfo" || action == "validDelete" ||
		(action == "get" && len(args) == 1) {
		if len(args) != 1 || strings.TrimSpace(args[0]) == "" {
			return fmt.Errorf("cloudcc %s globalSelectList <projectPath> <id-name-or-label>", action)
		}
		c, _, err := newClient([]string{projectPath}, cwd)
		if err != nil {
			return err
		}
		return c.getJSON(stdout, "/metadata/v1/global-select-lists/"+url.PathEscape(strings.TrimSpace(args[0])))
	}
	if len(args) > 2 {
		return fmt.Errorf("cloudcc %s globalSelectList <projectPath> [page] [pageSize]", action)
	}
	page, pageSize := 1, 20
	var err error
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		page, err = strconv.Atoi(strings.TrimSpace(args[0]))
		if err != nil || page < 1 {
			return fmt.Errorf("globalSelectList page must be a positive integer")
		}
	}
	if len(args) > 1 && strings.TrimSpace(args[1]) != "" {
		pageSize, err = strconv.Atoi(strings.TrimSpace(args[1]))
		if err != nil || pageSize < 1 || pageSize > 500 {
			return fmt.Errorf("globalSelectList pageSize must be between 1 and 500")
		}
	}
	c, _, err := newClient([]string{projectPath}, cwd)
	if err != nil {
		return err
	}
	return c.getJSON(stdout, "/metadata/v1/global-select-lists?page="+strconv.Itoa(page)+"&pageSize="+strconv.Itoa(pageSize))
}

func handleRecordTypeReadShortcut(action string, projectPath string, args []string, stdout io.Writer, cwd string) error {
	action = strings.TrimSpace(action)
	c, _, err := newClient([]string{projectPath}, cwd)
	if err != nil {
		return err
	}
	if action == "detail" || action == "editInfo" || action == "validDelete" {
		if len(args) < 1 || len(args) > 2 || strings.TrimSpace(args[0]) == "" {
			return fmt.Errorf("cloudcc %s recordType <projectPath> <record-type-id-apiName-or-name> [object-id-apiName-or-prefix]", action)
		}
		path := "/metadata/v1/record-types/" + url.PathEscape(strings.TrimSpace(args[0]))
		if len(args) == 2 && strings.TrimSpace(args[1]) != "" {
			path += "?object=" + url.QueryEscape(strings.TrimSpace(args[1]))
		}
		return c.getJSON(stdout, path)
	}
	if len(args) < 1 || len(args) > 2 || strings.TrimSpace(args[0]) == "" {
		return fmt.Errorf("cloudcc %s recordType <projectPath> <object-id-apiName-or-prefix> [profileId]", action)
	}
	path := "/metadata/v1/record-types?object=" + url.QueryEscape(strings.TrimSpace(args[0]))
	if len(args) == 2 && strings.TrimSpace(args[1]) != "" {
		path += "&profileId=" + url.QueryEscape(strings.TrimSpace(args[1]))
	}
	return c.getJSON(stdout, path)
}

func handlePageLayoutReadShortcut(action string, projectPath string, args []string, stdout io.Writer, cwd string) error {
	action = strings.TrimSpace(action)
	c, _, err := newClient([]string{projectPath}, cwd)
	if err != nil {
		return err
	}
	if action == "detail" || action == "editInfo" {
		if len(args) < 2 || len(args) > 3 || strings.TrimSpace(args[0]) == "" || strings.TrimSpace(args[1]) == "" {
			return fmt.Errorf("cloudcc %s pagelayout <projectPath> <object-id-apiName-or-prefix> <layout-id-apiName-or-name> [type]", action)
		}
		path := "/metadata/v1/layouts/" + url.PathEscape(strings.TrimSpace(args[1])) +
			"?object=" + url.QueryEscape(strings.TrimSpace(args[0]))
		return c.getJSON(stdout, path)
	}
	if len(args) != 1 || strings.TrimSpace(args[0]) == "" {
		return fmt.Errorf("cloudcc %s pagelayout <projectPath> <object-id-apiName-or-prefix>", action)
	}
	return c.getJSON(stdout, "/metadata/v1/layouts?object="+url.QueryEscape(strings.TrimSpace(args[0])))
}

func handleProfileShortcut(action string, projectPath string, args []string, stdout io.Writer, cwd string) error {
	c, _, err := newClient([]string{projectPath}, cwd)
	if err != nil {
		return err
	}
	switch strings.TrimSpace(action) {
	case "get", "getList", "newInfo":
		filter, err := profileShortcutValue(args, false)
		if err != nil {
			return err
		}
		return c.getJSON(stdout, profileListPath(filter, ""))
	case "detail", "editInfo", "validDelete":
		selector, err := profileShortcutValue(args, true)
		if err != nil {
			return err
		}
		profile, err := c.resolveUniqueProfile(selector)
		if err != nil {
			return err
		}
		return c.getJSON(stdout, "/metadata/v1/profiles/"+url.PathEscape(strings.TrimSpace(fmt.Sprint(profile["id"]))))
	case "delete", "remove":
		selector, err := profileShortcutValue(args, true)
		if err != nil {
			return fmt.Errorf("cloudcc %s profile <projectPath> <id-or-name-or-apiName>: %w", action, err)
		}
		profile, err := c.resolveUniqueProfile(selector)
		if err != nil {
			return err
		}
		profileID := strings.TrimSpace(fmt.Sprint(profile["id"]))
		body := map[string]any{
			"domain":    "profiles",
			"operation": "delete",
			"mode":      "delete",
			"spec": map[string]any{
				"id": profileID,
			},
			"context": defaultContext(),
		}
		return c.writeJSON(stdout, http.MethodPost, "/metadata/v1/plans", body)
	default:
		spec, operation, err := shortcutPlanSpec(action, "profile", args)
		if err != nil {
			return err
		}
		body := map[string]any{
			"domain":    "profiles",
			"operation": operation,
			"mode":      operation,
			"spec":      spec,
			"context":   defaultContext(),
		}
		return c.writeJSON(stdout, http.MethodPost, "/metadata/v1/plans", body)
	}
}

func handlePermissionSetReadShortcut(action string, projectPath string, args []string, stdout io.Writer, cwd string) error {
	c, _, err := newClient([]string{projectPath}, cwd)
	if err != nil {
		return err
	}
	action = strings.TrimSpace(action)
	if action == "getList" || action == "newInfo" || (action == "get" && len(args) == 0) {
		if len(args) > 1 {
			return fmt.Errorf("cloudcc %s permission <projectPath> [filter]", action)
		}
		filter := ""
		if len(args) == 1 {
			filter = strings.TrimSpace(args[0])
		}
		return c.getJSON(stdout, permissionSetListPath(filter, ""))
	}
	selector, err := permissionSetShortcutValue(args)
	if err != nil {
		return fmt.Errorf("cloudcc %s permission <projectPath> <id-name-or-apiName>: %w", action, err)
	}
	permissionSet, err := c.resolveUniquePermissionSet(selector)
	if err != nil {
		return err
	}
	return c.getJSON(stdout, "/metadata/v1/permission-sets/"+
		url.PathEscape(strings.TrimSpace(fmt.Sprint(permissionSet["id"]))))
}

func handleRoleReadShortcut(action string, projectPath string, args []string, stdout io.Writer, cwd string) error {
	c, _, err := newClient([]string{projectPath}, cwd)
	if err != nil {
		return err
	}
	action = strings.TrimSpace(action)
	if action == "getList" || action == "newInfo" || (action == "get" && len(args) == 0) {
		if len(args) > 1 {
			return fmt.Errorf("cloudcc %s role <projectPath> [filter]", action)
		}
		filter := ""
		if len(args) == 1 {
			filter = strings.TrimSpace(args[0])
		}
		return c.getJSON(stdout, roleListPath(filter, ""))
	}
	selector, err := roleShortcutValue(args)
	if err != nil {
		return fmt.Errorf("cloudcc %s role <projectPath> <role-id-or-name>: %w", action, err)
	}
	role, err := c.resolveUniqueRole(selector)
	if err != nil {
		return err
	}
	return c.getJSON(stdout, "/metadata/v1/roles/"+
		url.PathEscape(strings.TrimSpace(fmt.Sprint(role["id"]))))
}

func roleShortcutValue(args []string) (string, error) {
	if len(args) != 1 || strings.TrimSpace(args[0]) == "" {
		return "", fmt.Errorf("role selector is required (role id or name)")
	}
	value := strings.TrimSpace(args[0])
	if !looksLikeJSONArg(value) {
		return value, nil
	}
	body, err := parseObject(value, "role selector")
	if err != nil {
		return "", err
	}
	resolved := firstShortcutValue(body, "selector", "id", "roleId", "roleid", "name", "rolename", "filter")
	if resolved == "" {
		return "", fmt.Errorf("role selector JSON must contain id, roleId, roleid, name, or rolename")
	}
	return resolved, nil
}

func roleListPath(filter string, selector string) string {
	values := url.Values{}
	if strings.TrimSpace(filter) != "" {
		values.Set("filter", strings.TrimSpace(filter))
	}
	if strings.TrimSpace(selector) != "" {
		values.Set("selector", strings.TrimSpace(selector))
	}
	if len(values) == 0 {
		return "/metadata/v1/roles"
	}
	return "/metadata/v1/roles?" + values.Encode()
}

func (c *client) resolveUniqueRole(selector string) (map[string]any, error) {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return nil, fmt.Errorf("role selector is required (role id or name)")
	}
	response, err := c.requestJSONMap(http.MethodGet, roleListPath("", selector), nil)
	if err != nil {
		return nil, err
	}
	rows, _ := response["roles"].([]any)
	if len(rows) == 0 {
		return nil, fmt.Errorf("role selector %q matched 0 roles", selector)
	}
	if len(rows) > 1 {
		ids := make([]string, 0, len(rows))
		for _, row := range rows {
			if item, ok := row.(map[string]any); ok {
				ids = append(ids, strings.TrimSpace(fmt.Sprint(item["id"])))
			}
		}
		return nil, fmt.Errorf("role selector %q matched %d roles: %s", selector, len(rows), strings.Join(ids, ", "))
	}
	role, ok := rows[0].(map[string]any)
	if !ok || strings.TrimSpace(fmt.Sprint(role["id"])) == "" {
		return nil, fmt.Errorf("role selector %q returned an invalid role", selector)
	}
	return role, nil
}

func handleSharingRuleReadShortcut(action string, projectPath string, args []string, stdout io.Writer, cwd string) error {
	c, _, err := newClient([]string{projectPath}, cwd)
	if err != nil {
		return err
	}
	action = strings.TrimSpace(action)
	if action == "getList" || action == "newInfo" || (action == "get" && len(args) == 0) {
		if len(args) > 1 {
			return fmt.Errorf("cloudcc %s sharingRule <projectPath> [filter-or-objectId]", action)
		}
		value := ""
		if len(args) == 1 {
			value = strings.TrimSpace(args[0])
		}
		return c.getJSON(stdout, sharingRuleListPath(value, "", ""))
	}
	selector, objectID, err := sharingRuleShortcutValues(args)
	if err != nil {
		return fmt.Errorf("cloudcc %s sharingRule <projectPath> <rule-id-name-apiName-or-json-selector>: %w", action, err)
	}
	if objectID != "" && selector == "" {
		return c.getJSON(stdout, sharingRuleListPath("", "", objectID))
	}
	rule, err := c.resolveUniqueSharingRule(selector, objectID)
	if err != nil {
		return err
	}
	return c.getJSON(stdout, "/metadata/v1/sharing-rules/"+
		url.PathEscape(strings.TrimSpace(fmt.Sprint(rule["id"]))))
}

func sharingRuleShortcutValues(args []string) (string, string, error) {
	if len(args) != 1 || strings.TrimSpace(args[0]) == "" {
		return "", "", fmt.Errorf("sharing-rule selector is required (id, apiName, label, or JSON selector)")
	}
	value := strings.TrimSpace(args[0])
	if !looksLikeJSONArg(value) {
		return value, "", nil
	}
	body, err := parseObject(value, "sharing-rule selector")
	if err != nil {
		return "", "", err
	}
	selector := firstShortcutValue(body, "selector", "id", "ruleId", "ruleid", "apiName", "apiname", "label", "name", "filter")
	objectID := firstShortcutValue(body, "object", "objid", "objectId", "targetObjectId")
	if selector == "" && objectID == "" {
		return "", "", fmt.Errorf("sharing-rule selector JSON must contain id, ruleId, apiName, label, or objectId")
	}
	return selector, objectID, nil
}

func sharingRuleListPath(filter string, selector string, objectID string) string {
	values := url.Values{}
	if strings.TrimSpace(filter) != "" {
		values.Set("filter", strings.TrimSpace(filter))
	}
	if strings.TrimSpace(selector) != "" {
		values.Set("selector", strings.TrimSpace(selector))
	}
	if strings.TrimSpace(objectID) != "" {
		values.Set("object", strings.TrimSpace(objectID))
	}
	if len(values) == 0 {
		return "/metadata/v1/sharing-rules"
	}
	return "/metadata/v1/sharing-rules?" + values.Encode()
}

func (c *client) resolveUniqueSharingRule(selector string, objectID string) (map[string]any, error) {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return nil, fmt.Errorf("sharing-rule selector is required")
	}
	response, err := c.requestJSONMap(http.MethodGet, sharingRuleListPath("", selector, objectID), nil)
	if err != nil {
		return nil, err
	}
	rows, _ := response["sharingRules"].([]any)
	if len(rows) == 0 {
		return nil, fmt.Errorf("sharing-rule selector %q matched 0 sharing rules", selector)
	}
	if len(rows) > 1 {
		ids := make([]string, 0, len(rows))
		for _, row := range rows {
			if item, ok := row.(map[string]any); ok {
				ids = append(ids, strings.TrimSpace(fmt.Sprint(item["id"])))
			}
		}
		return nil, fmt.Errorf("sharing-rule selector %q matched %d sharing rules: %s", selector, len(rows), strings.Join(ids, ", "))
	}
	rule, ok := rows[0].(map[string]any)
	if !ok || strings.TrimSpace(fmt.Sprint(rule["id"])) == "" {
		return nil, fmt.Errorf("sharing-rule selector %q returned an invalid sharing rule", selector)
	}
	return rule, nil
}

func permissionSetShortcutValue(args []string) (string, error) {
	if len(args) != 1 || strings.TrimSpace(args[0]) == "" {
		return "", fmt.Errorf("permission-set selector is required (id, name, or apiName)")
	}
	value := strings.TrimSpace(args[0])
	if !looksLikeJSONArg(value) {
		return value, nil
	}
	body, err := parseObject(value, "permission-set selector")
	if err != nil {
		return "", err
	}
	resolved := firstShortcutValue(body, "selector", "id", "permissionSetId", "permsetsid", "apiName", "name", "filter")
	if resolved == "" {
		return "", fmt.Errorf("permission-set selector JSON must contain id, permissionSetId, permsetsid, apiName, or name")
	}
	return resolved, nil
}

func permissionSetListPath(filter string, selector string) string {
	values := url.Values{}
	if strings.TrimSpace(filter) != "" {
		values.Set("filter", strings.TrimSpace(filter))
	}
	if strings.TrimSpace(selector) != "" {
		values.Set("selector", strings.TrimSpace(selector))
	}
	if len(values) == 0 {
		return "/metadata/v1/permission-sets"
	}
	return "/metadata/v1/permission-sets?" + values.Encode()
}

func (c *client) resolveUniquePermissionSet(selector string) (map[string]any, error) {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return nil, fmt.Errorf("permission-set selector is required (id, name, or apiName)")
	}
	response, err := c.requestJSONMap(http.MethodGet, permissionSetListPath("", selector), nil)
	if err != nil {
		return nil, err
	}
	rows, _ := response["permissionSets"].([]any)
	if len(rows) == 0 {
		return nil, fmt.Errorf("permission-set selector %q matched 0 permission sets", selector)
	}
	if len(rows) > 1 {
		ids := make([]string, 0, len(rows))
		for _, row := range rows {
			if item, ok := row.(map[string]any); ok {
				ids = append(ids, strings.TrimSpace(fmt.Sprint(item["id"])))
			}
		}
		return nil, fmt.Errorf("permission-set selector %q matched %d permission sets: %s",
			selector, len(rows), strings.Join(ids, ", "))
	}
	permissionSet, ok := rows[0].(map[string]any)
	if !ok || strings.TrimSpace(fmt.Sprint(permissionSet["id"])) == "" {
		return nil, fmt.Errorf("permission-set selector %q returned an invalid permission set", selector)
	}
	return permissionSet, nil
}

func handleReportShortcut(action string, projectPath string, args []string, stdout io.Writer, cwd string) error {
	switch strings.TrimSpace(action) {
	case "get", "getList":
		body, err := reportListShortcutBody(args)
		if err != nil {
			return err
		}
		c, _, err := newClient([]string{projectPath}, cwd)
		if err != nil {
			return err
		}
		return c.writeJSON(stdout, http.MethodPost, "/metadata/v1/reports:query", body)
	case "detail", "editInfo":
		body, err := reportDetailShortcutBody(args)
		if err != nil {
			return err
		}
		c, _, err := newClient([]string{projectPath}, cwd)
		if err != nil {
			return err
		}
		return c.writeJSON(stdout, http.MethodPost, "/metadata/v1/reports:detail", body)
	case "create":
		body, err := reportSaveShortcutBody(args, false)
		if err != nil {
			return err
		}
		return planShortcut(stdout, cwd, projectPath, "reports", body, "create")
	case "update", "save", "modify", "editSave":
		body, err := reportSaveShortcutBody(args, true)
		if err != nil {
			return err
		}
		return planShortcut(stdout, cwd, projectPath, "reports", body, "upsert")
	case "delete", "remove":
		body, err := reportDeleteShortcutBody(args)
		if err != nil {
			return err
		}
		return planShortcut(stdout, cwd, projectPath, "reports", body, "delete")
	default:
		return fmt.Errorf("unsupported report shortcut action: %s", action)
	}
}

func handleTypedReportShortcut(action string, resource string, projectPath string, args []string, stdout io.Writer, cwd string) error {
	switch strings.TrimSpace(action) {
	case "create":
		body, err := typedReportSaveShortcutBody(resource, args, false)
		if err != nil {
			return err
		}
		return planShortcut(stdout, cwd, projectPath, "reports", body, "create")
	case "update", "save", "modify", "editSave":
		body, err := typedReportSaveShortcutBody(resource, args, true)
		if err != nil {
			return err
		}
		return planShortcut(stdout, cwd, projectPath, "reports", body, "upsert")
	default:
		return fmt.Errorf("unsupported %s shortcut action: %s", resource, action)
	}
}

func handleReportFolderShortcut(action string, projectPath string, args []string, stdout io.Writer, cwd string) error {
	switch strings.TrimSpace(action) {
	case "get", "getList":
		body, err := reportFolderListShortcutBody(args)
		if err != nil {
			return err
		}
		c, _, err := newClient([]string{projectPath}, cwd)
		if err != nil {
			return err
		}
		return c.writeJSON(stdout, http.MethodPost, "/metadata/v1/report-folders:query", body)
	case "detail", "editInfo":
		body, err := reportFolderDetailShortcutBody(args)
		if err != nil {
			return err
		}
		c, _, err := newClient([]string{projectPath}, cwd)
		if err != nil {
			return err
		}
		return c.writeJSON(stdout, http.MethodPost, "/metadata/v1/report-folders:detail", body)
	case "create", "add":
		body, err := reportFolderCreateShortcutBody(args, false)
		if err != nil {
			return err
		}
		return planShortcut(stdout, cwd, projectPath, "reports", body, "folder-create")
	case "update", "save", "modify", "editSave":
		body, err := reportFolderCreateShortcutBody(args, true)
		if err != nil {
			return err
		}
		return planShortcut(stdout, cwd, projectPath, "reports", body, "folder-update")
	case "delete", "remove":
		body, err := reportFolderDeleteShortcutBody(args)
		if err != nil {
			return err
		}
		return planShortcut(stdout, cwd, projectPath, "reports", body, "folder-delete")
	default:
		return fmt.Errorf("unsupported reportFolder shortcut action: %s", action)
	}
}

// NormalizeReportShortcut converts the stable report and report-folder CLI
// arguments into the request body accepted by both providers. It deliberately
// contains no MetadataService I/O, so UIAPI execution retains the same input
// validation and report-type defaults as MSAPI execution.
func NormalizeReportShortcut(action string, resource string, args []string) (map[string]any, error) {
	action = strings.TrimSpace(action)
	resource = strings.TrimSpace(resource)
	switch resource {
	case "report", "reports":
		switch action {
		case "get", "getList":
			return reportListShortcutBody(args)
		case "detail", "editInfo":
			return reportDetailShortcutBody(args)
		case "create":
			return reportSaveShortcutBody(args, false)
		case "update", "save", "modify", "editSave":
			return reportSaveShortcutBody(args, true)
		case "delete", "remove":
			return reportDeleteShortcutBody(args)
		}
	case "reportTabular", "reportSummary", "reportMatrix", "reportRatio":
		if action == "create" {
			return typedReportSaveShortcutBody(resource, args, false)
		}
		if action == "update" || action == "save" || action == "modify" || action == "editSave" {
			return typedReportSaveShortcutBody(resource, args, true)
		}
	case "reportFolder":
		switch action {
		case "get", "getList":
			return reportFolderListShortcutBody(args)
		case "detail", "editInfo":
			return reportFolderDetailShortcutBody(args)
		case "create", "add":
			return reportFolderCreateShortcutBody(args, false)
		case "update", "save", "modify", "editSave":
			return reportFolderCreateShortcutBody(args, true)
		case "delete", "remove":
			return reportFolderDeleteShortcutBody(args)
		}
	}
	return nil, fmt.Errorf("unsupported %s shortcut action: %s", resource, action)
}

func planShortcut(stdout io.Writer, cwd string, projectPath string, domain string, spec map[string]any, operation string) error {
	body, err := json.Marshal(spec)
	if err != nil {
		return err
	}
	return Handle("plan", "msapi", []string{projectPath, domain, string(body), operation}, stdout, cwd)
}

func reportListShortcutBody(args []string) (map[string]any, error) {
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		if body, err := parseObject(args[0], "cloudcc get report"); err == nil {
			setReportListShortcutDefaults(body)
			return body, nil
		}
	}
	body := map[string]any{}
	fields := []string{"folderId", "searchKeyWord", "page", "pageSize", "orderField", "orderType"}
	for i, field := range fields {
		if len(args) <= i || strings.TrimSpace(args[i]) == "" {
			continue
		}
		value := strings.TrimSpace(args[i])
		if field == "page" || field == "pageSize" {
			n, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("cloudcc get report: %s must be an integer", field)
			}
			body[field] = n
		} else {
			body[field] = value
		}
	}
	setReportListShortcutDefaults(body)
	return body, nil
}

func setReportListShortcutDefaults(body map[string]any) {
	if body["page"] == nil {
		body["page"] = 1
	}
	if body["pageSize"] == nil {
		body["pageSize"] = 20
	}
}

func reportDetailShortcutBody(args []string) (map[string]any, error) {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return nil, fmt.Errorf("cloudcc detail report <projectPath> <reportId|encodedBodyJson>")
	}
	if body, err := parseObject(args[0], "cloudcc detail report"); err == nil {
		return body, nil
	}
	return map[string]any{"id": strings.TrimSpace(args[0])}, nil
}

func reportSaveShortcutBody(args []string, requireID bool) (map[string]any, error) {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		if requireID {
			return nil, fmt.Errorf("cloudcc update report <projectPath> <reportId> <encodedReportJson> OR cloudcc update report <projectPath> <encodedReportJson>")
		}
		return nil, fmt.Errorf("cloudcc create report <projectPath> <encodedReportJson>")
	}
	reportID := ""
	bodyArg := args[0]
	if requireID && len(args) > 1 {
		reportID = strings.TrimSpace(args[0])
		bodyArg = args[1]
	}
	body, err := parseObject(bodyArg, "cloudcc save report")
	if err != nil {
		return nil, err
	}
	if requireID {
		if reportID != "" {
			body["id"] = reportID
		}
		if firstShortcutValue(body, "id") == "" {
			return nil, fmt.Errorf("cloudcc update report: report id is required")
		}
	} else {
		delete(body, "id")
	}
	return body, nil
}

func reportMatrixSaveShortcutBody(args []string, requireID bool) (map[string]any, error) {
	return typedReportSaveShortcutBody("reportMatrix", args, requireID)
}

func typedReportSaveShortcutBody(resource string, args []string, requireID bool) (map[string]any, error) {
	body, err := reportSaveShortcutBody(args, requireID)
	if err != nil {
		if requireID {
			return nil, fmt.Errorf("cloudcc update %s <projectPath> <reportId> <encodedReportJson|@file> OR cloudcc update %s <projectPath> <encodedReportJson|@file>: %w", resource, resource, err)
		}
		return nil, fmt.Errorf("cloudcc create %s <projectPath> <encodedReportJson|@file>: %w", resource, err)
	}
	switch resource {
	case "reportTabular":
		normalizeTabularReportSpec(body)
		err = validateTabularReportSpec(body)
	case "reportSummary":
		normalizeSummaryReportSpec(body)
		err = validateSummaryReportSpec(body)
	case "reportRatio":
		normalizeRatioReportSpec(body)
		err = validateRatioReportSpec(body)
	default:
		normalizeMatrixReportSpec(body)
		err = validateMatrixReportSpec(body)
	}
	if err != nil {
		return nil, err
	}
	return body, nil
}

func normalizeTabularReportSpec(body map[string]any) {
	normalizeReportBaseSpec(body, "Tabular")
}

func normalizeSummaryReportSpec(body map[string]any) {
	normalizeReportBaseSpec(body, "Summary")
	ensureMatrixGroups(body)
	ensureMatrixSummaries(body)
}

func normalizeMatrixReportSpec(body map[string]any) {
	normalizeReportBaseSpec(body, "Matrix")
	ensureMatrixGroups(body)
	ensureMatrixSummaries(body)
}

func normalizeRatioReportSpec(body map[string]any) {
	normalizeReportBaseSpec(body, "ratio")
	ensureRatioDateGroup(body)
	ensureRatioExpressions(body)
	setShortcutDefault(body, "transversedatetypeone", "month")
	if dateField := firstShortcutValue(body, "transversegroupone", "dateFieldId", "dateField", "groupFieldId"); dateField != "" {
		setShortcutDefault(body, "datecon", dateField)
		if firstShortcutValue(body, "mainobjectcolumnid", "mainObjectColumnId") == "" && len(shortcutArray(body["fields"])) == 0 {
			body["mainobjectcolumnid"] = "totalrecord," + dateField
		}
	}
	if dateLocation := strings.ToLower(strings.TrimSpace(firstShortcutValue(body, "datelocation", "dateLocation"))); dateLocation != "" {
		body["datelocation"] = dateLocation
	}
}

func normalizeReportBaseSpec(body map[string]any, reportType string) {
	body["type"] = reportType
	body["reporttype"] = reportType
	setShortcutDefault(body, "islightning", "true")
	setShortcutDefault(body, "totalrecord", "1")
	setShortcutDefault(body, "scope", "user")
}

func setShortcutDefault(body map[string]any, key string, value any) {
	if strings.TrimSpace(fmt.Sprint(body[key])) == "" || fmt.Sprint(body[key]) == "<nil>" {
		body[key] = value
	}
}

func ensureMatrixGroups(body map[string]any) {
	groups := shortcutMap(body["groups"])
	if groups == nil {
		groups = map[string]any{}
		body["groups"] = groups
	}
	if _, ok := groups["rows"]; !ok {
		if rows := firstShortcutArray(body, "rows", "rowGroups", "transverseGroups"); len(rows) > 0 {
			groups["rows"] = rows
		}
	}
	if _, ok := groups["columns"]; !ok {
		if columns := firstShortcutArray(body, "columns", "columnGroups", "lengthwaysGroups"); len(columns) > 0 {
			groups["columns"] = columns
		}
	}
}

func ensureMatrixSummaries(body map[string]any) {
	if len(shortcutArray(body["summaries"])) > 0 || len(shortcutArray(body["gathers"])) > 0 {
		return
	}
	fields := firstShortcutArray(body, "summaryFields", "gatherFields")
	if len(fields) == 0 {
		return
	}
	summaries := make([]any, 0, len(fields))
	for _, field := range fields {
		if item, ok := field.(map[string]any); ok {
			summaries = append(summaries, item)
			continue
		}
		fieldID := strings.TrimSpace(fmt.Sprint(field))
		if fieldID != "" && fieldID != "<nil>" {
			summaries = append(summaries, map[string]any{"fieldId": fieldID, "method": "sum"})
		}
	}
	if len(summaries) > 0 {
		body["summaries"] = summaries
	}
}

func ensureRatioDateGroup(body map[string]any) {
	if firstShortcutValue(body, "transversegroupone") != "" {
		return
	}
	dateField := firstShortcutValue(body, "dateFieldId", "dateField", "groupFieldId")
	if dateField == "" {
		return
	}
	body["transversegroupone"] = dateField
}

func ensureRatioExpressions(body map[string]any) {
	if firstShortcutValue(body, "tbhbexpression", "tbhbExpression") != "" {
		return
	}
	expressions := firstShortcutArray(body, "ratioExpressions", "comparisonExpressions", "tbhbExpressions")
	if len(expressions) == 0 {
		return
	}
	payload := map[string]any{"data": expressions}
	encoded, err := json.Marshal(payload)
	if err == nil {
		body["tbhbexpression"] = string(encoded)
	}
}

func validateTabularReportSpec(body map[string]any) error {
	if !hasReportSource(body) {
		return fmt.Errorf("cloudcc reportTabular requires source object: provide objecta/mainObjectId/source.objects[0] or reporttypecustomid")
	}
	if !hasSelectedReportFields(body) {
		return fmt.Errorf("cloudcc reportTabular requires selected fields: provide fields[] or mainobjectcolumnid")
	}
	return nil
}

func validateSummaryReportSpec(body map[string]any) error {
	if !hasReportSource(body) {
		return fmt.Errorf("cloudcc reportSummary requires source object: provide objecta/mainObjectId/source.objects[0] or reporttypecustomid")
	}
	rowGroups := matrixGroupArray(body, "rows", "transversegroupone", "transversegrouptwo", "transversegroupthree")
	if len(rowGroups) == 0 {
		return fmt.Errorf("cloudcc reportSummary requires row groups: provide groups.rows, rows, or transversegroupone/two/three")
	}
	if len(rowGroups) > 3 {
		return fmt.Errorf("cloudcc reportSummary supports up to 3 row groups, matching main-svc transversegroupone/two/three")
	}
	if !hasSelectedReportFields(body) {
		return fmt.Errorf("cloudcc reportSummary requires selected fields: provide fields[] or mainobjectcolumnid")
	}
	if err := validateReportRelation(body, "reportSummary"); err != nil {
		return err
	}
	if !hasAnyMatrixSummary(body) {
		return fmt.Errorf("cloudcc reportSummary requires a statistic: provide summaries[], gathers[], gatherfieldname, or totalrecord")
	}
	return nil
}

func validateMatrixReportSpec(body map[string]any) error {
	rowGroups := matrixGroupArray(body, "rows", "transversegroupone", "transversegrouptwo", "transversegroupthree")
	columnGroups := matrixGroupArray(body, "columns", "lengthwaysgroupone", "lengthwaysgrouptwo")
	if len(rowGroups) == 0 || len(columnGroups) == 0 {
		return fmt.Errorf("cloudcc reportMatrix requires both row groups and column groups: provide groups.rows/groups.columns or rows/columns")
	}
	if len(rowGroups) > 3 {
		return fmt.Errorf("cloudcc reportMatrix supports up to 3 row groups, matching main-svc transversegroupone/two/three")
	}
	if len(columnGroups) > 2 {
		return fmt.Errorf("cloudcc reportMatrix supports up to 2 column groups, matching main-svc lengthwaysgroupone/two")
	}
	if !hasSelectedReportFields(body) {
		return fmt.Errorf("cloudcc reportMatrix requires selected fields: provide fields[] or mainobjectcolumnid")
	}
	if err := validateReportRelation(body, "reportMatrix"); err != nil {
		return err
	}
	if !hasAnyMatrixSummary(body) {
		return fmt.Errorf("cloudcc reportMatrix requires a statistic: provide summaries[], gathers[], gatherfieldname, or totalrecord")
	}
	if isShortcutTruthy(firstShortcutValue(body, "isshowchart", "showChart")) || isShortcutTruthy(firstShortcutValue(shortcutMap(body["options"]), "showChart")) {
		if firstShortcutValue(body, "dashboardtype", "dashboardType") == "" && firstShortcutValue(shortcutMap(body["chart"]), "type") == "" {
			return fmt.Errorf("cloudcc reportMatrix with isshowchart=true requires dashboardtype or chart.type")
		}
		if firstShortcutValue(body, "xcon", "xCon") == "" && firstShortcutValue(shortcutMap(body["chart"]), "x") == "" {
			return fmt.Errorf("cloudcc reportMatrix with isshowchart=true requires xcon or chart.x")
		}
		if firstShortcutValue(body, "ycon", "yCon") == "" && firstShortcutValue(shortcutMap(body["chart"]), "y") == "" {
			return fmt.Errorf("cloudcc reportMatrix with isshowchart=true requires ycon or chart.y")
		}
	}
	if hasShortcutDateRange(body) && firstShortcutValue(body, "datecon", "dateCon", "dateCondition") == "" &&
		firstShortcutValue(body, "dateFieldId", "dateField", "groupFieldId") == "" &&
		firstShortcutValue(shortcutMap(body["timeFrame"]), "dateFieldId", "fieldId") == "" &&
		firstShortcutValue(shortcutMap(body["dateRange"]), "dateFieldId") == "" {
		return fmt.Errorf("cloudcc reportMatrix with datarange/startdate/enddate requires datecon, dateCondition, or timeFrame.dateFieldId")
	}
	return nil
}

func validateRatioReportSpec(body map[string]any) error {
	if !hasReportSource(body) {
		return fmt.Errorf("cloudcc reportRatio requires source object: provide objecta/mainObjectId/source.objects[0] or reporttypecustomid")
	}
	if firstShortcutValue(body, "transversegroupone", "dateFieldId", "dateField", "groupFieldId") == "" {
		return fmt.Errorf("cloudcc reportRatio requires date group field: provide dateFieldId/dateField/groupFieldId or transversegroupone")
	}
	if !hasSelectedReportFields(body) {
		return fmt.Errorf("cloudcc reportRatio requires selected fields: provide fields[] or mainobjectcolumnid")
	}
	if err := validateReportRelation(body, "reportRatio"); err != nil {
		return err
	}
	dateLocation := strings.ToLower(strings.TrimSpace(firstShortcutValue(body, "datelocation", "dateLocation")))
	if dateLocation != "" && dateLocation != "first" && dateLocation != "end" {
		return fmt.Errorf("cloudcc reportRatio datelocation must be first or end")
	}
	dateType := firstShortcutValue(body, "transversedatetypeone", "transverseDateTypeOne")
	if dateType != "" && !isSupportedReportDateType(dateType) {
		return fmt.Errorf("cloudcc reportRatio transversedatetypeone must be day, week, month, quarter, year, FY, FQ, CY, or CQ")
	}
	return nil
}

func validateReportRelation(body map[string]any, resource string) error {
	if matrixReportTypeCustomID(body) != "" {
		return nil
	}
	for index := 1; index <= 3; index++ {
		if !hasReportObjectAt(body, index) {
			continue
		}
		label := reportRelationLabel(index)
		optionKey := "option" + label
		findIDKey := label + "findid"
		option := reportRelationFieldValue(body, index, optionKey, "option", "relationType")
		option = strings.ToLower(strings.TrimSpace(option))
		if option != "" {
			body[optionKey] = option
		}
		if option == "" {
			return fmt.Errorf("cloudcc %s source object %s requires %s or source.objects[%d].option (expected inner or outer)", resource, strings.ToUpper(label), optionKey, index)
		}
		if option != "inner" && option != "outer" {
			return fmt.Errorf("cloudcc %s %s/source.objects[%d].option must be inner or outer", resource, optionKey, index)
		}
		if reportRelationFieldValue(body, index, findIDKey, "findId", "findid", "referenceFieldId") == "" {
			return fmt.Errorf("cloudcc %s source object %s requires %s or source.objects[%d].findId", resource, strings.ToUpper(label), findIDKey, index)
		}
	}
	return nil
}

func hasReportObjectAt(body map[string]any, index int) bool {
	switch index {
	case 1:
		return hasSecondReportObject(body)
	case 2:
		return firstShortcutValue(body, "objectc", "cObjectId", "cobjectid") != "" ||
			sourceObjectValue(body, index, "objectId", "objId", "id", "object", "apiName") != ""
	case 3:
		return firstShortcutValue(body, "objectd", "dObjectId", "dobjectid") != "" ||
			sourceObjectValue(body, index, "objectId", "objId", "id", "object", "apiName") != ""
	default:
		return false
	}
}

func reportRelationLabel(index int) string {
	switch index {
	case 1:
		return "b"
	case 2:
		return "c"
	case 3:
		return "d"
	default:
		return ""
	}
}

func reportRelationFieldValue(body map[string]any, index int, directKey string, sourceKeys ...string) string {
	if value := firstShortcutValue(body, directKey); value != "" {
		return value
	}
	return sourceObjectValue(body, index, sourceKeys...)
}

func matrixReportTypeCustomID(body map[string]any) string {
	if value := firstShortcutValue(body, "reportTypeCustomId", "reporttypecustomid"); value != "" {
		return value
	}
	if source := shortcutMap(body["source"]); source != nil {
		return firstShortcutValue(source, "reportTypeCustomId", "reporttypecustomid")
	}
	return ""
}

func matrixGroupArray(body map[string]any, key string, directKeys ...string) []any {
	if groups := shortcutMap(body["groups"]); groups != nil {
		if values := shortcutArray(groups[key]); len(values) > 0 {
			return values
		}
	}
	result := make([]any, 0, len(directKeys))
	for _, key := range directKeys {
		if value := firstShortcutValue(body, key); value != "" {
			result = append(result, value)
		}
	}
	if len(result) > 0 {
		return result
	}
	return nil
}

func hasReportSource(body map[string]any) bool {
	return matrixReportTypeCustomID(body) != "" ||
		firstShortcutValue(body, "objecta", "mainObjectId", "objectId") != "" ||
		matrixRelationValue(body, 0, "objectId", "objId", "id", "object", "apiName") != ""
}

func hasSelectedReportFields(body map[string]any) bool {
	return firstShortcutValue(body, "mainobjectcolumnid", "mainObjectColumnId") != "" || len(shortcutArray(body["fields"])) > 0
}

func hasSecondReportObject(body map[string]any) bool {
	if firstShortcutValue(body, "objectb", "relatedObjectId") != "" {
		return true
	}
	return matrixRelationValue(body, 1, "objectId", "objId", "id", "object", "apiName") != ""
}

func matrixRelationValue(body map[string]any, index int, keys ...string) string {
	for _, key := range keys {
		if value := firstShortcutValue(body, key); value != "" {
			return value
		}
	}
	return sourceObjectValue(body, index, keys...)
}

func sourceObjectValue(body map[string]any, index int, keys ...string) string {
	source := shortcutMap(body["source"])
	if source == nil {
		return ""
	}
	objects := shortcutArray(source["objects"])
	if len(objects) <= index {
		return ""
	}
	if value, ok := objects[index].(string); ok {
		return strings.TrimSpace(value)
	}
	object := shortcutMap(objects[index])
	if object == nil {
		return ""
	}
	return firstShortcutValue(object, keys...)
}

func hasAnyMatrixSummary(body map[string]any) bool {
	return len(shortcutArray(body["summaries"])) > 0 ||
		len(shortcutArray(body["gathers"])) > 0 ||
		firstShortcutValue(body, "gatherfieldname", "gatherFieldName", "totalrecord") != ""
}

func hasShortcutDateRange(body map[string]any) bool {
	return firstShortcutValue(body,
		"datarange", "dataRange", "reportTimeType",
		"startdate", "startDate", "startdatestr", "startDateStr",
		"enddate", "endDate", "enddatestr", "endDateStr") != ""
}

func isShortcutTruthy(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	return normalized == "true" || normalized == "1" || normalized == "yes"
}

func isSupportedReportDateType(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "day", "week", "month", "quarter", "year", "fy", "fq", "cy", "cq":
		return true
	default:
		return false
	}
}

func firstShortcutArray(body map[string]any, keys ...string) []any {
	for _, key := range keys {
		if values := shortcutArray(body[key]); len(values) > 0 {
			return values
		}
	}
	return nil
}

func shortcutArray(value any) []any {
	switch items := value.(type) {
	case []any:
		return items
	case []map[string]any:
		result := make([]any, 0, len(items))
		for _, item := range items {
			result = append(result, item)
		}
		return result
	default:
		return nil
	}
}

func shortcutMap(value any) map[string]any {
	if item, ok := value.(map[string]any); ok {
		return item
	}
	return nil
}

func reportDeleteShortcutBody(args []string) (map[string]any, error) {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return nil, fmt.Errorf("cloudcc delete report <projectPath> <reportId> [confirmdelete]")
	}
	body := map[string]any{"id": strings.TrimSpace(args[0])}
	if len(args) > 1 && strings.TrimSpace(args[1]) != "" {
		body["confirmdelete"] = strings.TrimSpace(args[1])
	}
	return body, nil
}

func reportFolderListShortcutBody(args []string) (map[string]any, error) {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return map[string]any{}, nil
	}
	if body, err := parseObject(args[0], "cloudcc get reportFolder"); err == nil {
		return body, nil
	}
	return map[string]any{"searchKeyWord": strings.TrimSpace(args[0])}, nil
}

func reportFolderDetailShortcutBody(args []string) (map[string]any, error) {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return nil, fmt.Errorf("cloudcc detail reportFolder <projectPath> <folderId|encodedBodyJson>")
	}
	if body, err := parseObject(args[0], "cloudcc detail reportFolder"); err == nil {
		return body, nil
	}
	return map[string]any{"id": strings.TrimSpace(args[0])}, nil
}

func reportFolderCreateShortcutBody(args []string, requireID bool) (map[string]any, error) {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		if requireID {
			return nil, fmt.Errorf("cloudcc update reportFolder <projectPath> <folderId> <encodedOptionsJson>")
		}
		return nil, fmt.Errorf("cloudcc create reportFolder <projectPath> <name> [encodedOptionsJson]")
	}
	body := map[string]any{}
	if requireID {
		body["id"] = strings.TrimSpace(args[0])
		if len(args) > 1 && strings.TrimSpace(args[1]) != "" {
			options, err := parseObject(args[1], "cloudcc update reportFolder")
			if err != nil {
				return nil, err
			}
			for key, value := range options {
				body[key] = value
			}
		}
		return body, nil
	}
	if len(args) > 1 && strings.TrimSpace(args[1]) != "" {
		options, err := parseObject(args[1], "cloudcc create reportFolder")
		if err != nil {
			return nil, err
		}
		for key, value := range options {
			body[key] = value
		}
	}
	body["name"] = strings.TrimSpace(args[0])
	return body, nil
}

func reportFolderDeleteShortcutBody(args []string) (map[string]any, error) {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return nil, fmt.Errorf("cloudcc delete reportFolder <projectPath> <folderId>")
	}
	return map[string]any{"id": strings.TrimSpace(args[0])}, nil
}

func profileShortcutValue(args []string, required bool) (string, error) {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		if required {
			return "", fmt.Errorf("profile selector is required (id, name, or apiName)")
		}
		return "", nil
	}
	value := strings.TrimSpace(args[0])
	if !looksLikeJSONArg(value) {
		return value, nil
	}
	body, err := parseObject(value, "profile selector/filter")
	if err != nil {
		return "", err
	}
	resolved := firstShortcutValue(body, "selector", "id", "profileId", "apiName", "profilename", "profileName", "name", "filter")
	if resolved == "" && required {
		return "", fmt.Errorf("profile selector JSON must contain id, profileId, apiName, profilename, profileName, or name")
	}
	return resolved, nil
}

func profileListPath(filter string, selector string) string {
	values := url.Values{}
	if strings.TrimSpace(filter) != "" {
		values.Set("filter", strings.TrimSpace(filter))
	}
	if strings.TrimSpace(selector) != "" {
		values.Set("selector", strings.TrimSpace(selector))
	}
	if len(values) == 0 {
		return "/metadata/v1/profiles"
	}
	return "/metadata/v1/profiles?" + values.Encode()
}

func (c *client) resolveUniqueProfile(selector string) (map[string]any, error) {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return nil, fmt.Errorf("profile selector is required (id, name, or apiName)")
	}
	response, err := c.requestJSONMap(http.MethodGet, profileListPath("", selector), nil)
	if err != nil {
		return nil, err
	}
	rows, _ := response["profiles"].([]any)
	if len(rows) == 0 {
		return nil, fmt.Errorf("profile selector %q matched 0 profiles", selector)
	}
	if len(rows) > 1 {
		ids := make([]string, 0, len(rows))
		for _, row := range rows {
			if item, ok := row.(map[string]any); ok {
				ids = append(ids, strings.TrimSpace(fmt.Sprint(item["id"])))
			}
		}
		return nil, fmt.Errorf("profile selector %q matched %d profiles (%s); use a unique profile id or apiName", selector, len(rows), strings.Join(ids, ", "))
	}
	profile, ok := rows[0].(map[string]any)
	if !ok || strings.TrimSpace(fmt.Sprint(profile["id"])) == "" {
		return nil, fmt.Errorf("profile selector %q returned a row without id", selector)
	}
	return profile, nil
}

func isShortcutRead(action string) bool {
	switch strings.TrimSpace(action) {
	case "get", "detail", "getList", "newInfo", "editInfo", "validDelete":
		return true
	default:
		return false
	}
}

func shortcutProjectPath(args []string, cwd string) (string, []string) {
	if len(args) == 0 {
		return cwd, nil
	}
	first := strings.TrimSpace(args[0])
	if first == "" {
		return cwd, args[1:]
	}
	if looksLikeJSONArg(first) {
		return cwd, args
	}
	return first, args[1:]
}

func shortcutPlanSpec(action string, resource string, args []string) (map[string]any, string, error) {
	operation := shortcutOperation(action, resource)
	if resource == "object" {
		if operation == "create" {
			var accessable string
			var err error
			args, accessable, err = extractObjectCreateAccessableFlag(args)
			if err != nil {
				return nil, "", err
			}
			if len(args) > 0 && looksLikeJSONArg(args[0]) {
				body, err := parseObject(args[0], "cloudcc "+action+" object")
				if err == nil && accessable != "" {
					body["accessable"] = accessable
				}
				normalizeObjectShortcutSpec(body)
				return body, operation, err
			}
			spec := objectShortcutSpec(action, args)
			if accessable != "" {
				spec["accessable"] = accessable
			}
			return spec, operation, nil
		}
		if len(args) > 0 && looksLikeJSONArg(args[0]) {
			body, err := parseObject(args[0], "cloudcc "+action+" object")
			normalizeObjectShortcutSpec(body)
			return body, operation, err
		}
		return objectShortcutSpec(action, args), operation, nil
	}
	if resource == "validationRule" {
		return validationRuleShortcutSpec(action, args)
	}
	if resource == "workflow" || resource == "workflowRule" {
		return workflowShortcutSpec(action, args)
	}
	if resource == "fields" && strings.TrimSpace(action) == "delete" {
		if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
			return nil, "", fmt.Errorf("cloudcc delete fields <projectPath> <fieldId> [objectId] now creates a MetadataService delete plan")
		}
		field := map[string]any{"id": args[0]}
		if len(args) > 1 && strings.TrimSpace(args[1]) != "" {
			field["objectId"] = args[1]
		}
		return map[string]any{"fields": []any{field}}, operation, nil
	}
	body, _, err := shortcutBodySpec(action, resource, args)
	if err != nil {
		return nil, "", err
	}
	return body, operation, nil
}

func objectShortcutSpec(action string, args []string) map[string]any {
	switch strings.TrimSpace(action) {
	case "create":
		label := firstShortcutArg(args, 0)
		apiName := firstShortcutArg(args, 1)
		description := firstShortcutArg(args, 2)
		if len(args) == 2 {
			apiName = ""
			description = args[1]
		}
		spec := map[string]any{
			"label": label,
		}
		if apiName != "" {
			spec["apiName"] = apiName
			spec["name"] = apiName
			spec["schemetableName"] = apiName
		}
		if description != "" {
			spec["description"] = description
			spec["remark"] = description
		}
		normalizeObjectShortcutSpec(spec)
		return spec
	case "delete", "purge":
		return map[string]any{"id": firstShortcutArg(args, 0)}
	default:
		if len(args) > 0 && looksLikeJSONArg(args[0]) {
			if body, err := parseObject(args[0], "cloudcc "+action+" object"); err == nil {
				return body
			}
		}
		return map[string]any{"id": firstShortcutArg(args, 0)}
	}
}

func normalizeObjectShortcutSpec(spec map[string]any) {
	if spec == nil {
		return
	}
	apiName := firstShortcutValue(spec, "apiName", "name", "schemetableName")
	if apiName != "" {
		normalized := normalizeObjectAPIName(apiName, spec)
		spec["apiName"] = normalized
		spec["name"] = normalized
		spec["schemetableName"] = normalized
	}
	label := firstShortcutValue(spec, "label", "displayName")
	description := firstShortcutValue(spec, "description", "remark")
	if label != "" && (description == "" || !containsCJK(description)) {
		description = "用于管理" + label + "业务数据。"
		spec["description"] = description
		spec["remark"] = description
	}
}

func normalizeObjectAPIName(apiName string, spec map[string]any) string {
	apiName = strings.TrimSpace(apiName)
	if apiName == "" || shortcutBool(spec, "preserveApiPrefix", "keepApiPrefix") ||
		firstShortcutValue(spec, "apiPrefix", "namespacePrefix", "packagePrefix") != "" {
		return apiName
	}
	for _, prefix := range implicitObjectAPIPrefixes(spec) {
		prefix = strings.Trim(strings.ToLower(prefix), "_")
		if prefix == "" {
			continue
		}
		lowerAPIName := strings.ToLower(apiName)
		if strings.HasPrefix(lowerAPIName, prefix+"_") && len(apiName) > len(prefix)+1 {
			return strings.TrimSpace(apiName[len(prefix)+1:])
		}
	}
	return apiName
}

func implicitObjectAPIPrefixes(spec map[string]any) []string {
	prefixes := []string{"sun"}
	for _, key := range []string{"implicitApiPrefix", "customerPrefix", "projectPrefix", "tenantPrefix"} {
		value := firstShortcutValue(spec, key)
		if value != "" {
			prefixes = append(prefixes, value)
		}
	}
	return prefixes
}

func extractObjectCreateAccessableFlag(args []string) ([]string, string, error) {
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

func firstShortcutValue(spec map[string]any, keys ...string) string {
	for _, key := range keys {
		value := strings.TrimSpace(fmt.Sprint(spec[key]))
		if value != "" && value != "<nil>" {
			return value
		}
	}
	return ""
}

func shortcutBool(spec map[string]any, keys ...string) bool {
	for _, key := range keys {
		switch value := spec[key].(type) {
		case bool:
			if value {
				return true
			}
		case string:
			switch strings.ToLower(strings.TrimSpace(value)) {
			case "true", "1", "yes", "y":
				return true
			}
		}
	}
	return false
}

func containsCJK(value string) bool {
	for _, r := range value {
		if (r >= '\u4e00' && r <= '\u9fff') || (r >= '\u3400' && r <= '\u4dbf') {
			return true
		}
	}
	return false
}

func validationRuleShortcutSpec(action string, args []string) (map[string]any, string, error) {
	operation := shortcutOperation(action, "validationRule")
	if len(args) > 0 && looksLikeJSONArg(args[0]) {
		body, err := parseObject(args[0], "cloudcc "+action+" validationRule")
		return body, operation, err
	}
	switch strings.TrimSpace(action) {
	case "create":
		if len(args) < 4 {
			return nil, "", fmt.Errorf("cloudcc create validationRule <projectPath> <objectPrefix> <ruleName> <ruleContent> <errorMessage> now creates a MetadataService plan")
		}
		rule := map[string]any{
			"objectId":     args[0],
			"objectPrefix": args[0],
			"name":         args[1],
			"apiName":      args[1],
			"formula":      args[2],
			"errorMessage": args[3],
			"active":       false,
		}
		return map[string]any{"validationRules": []any{rule}}, operation, nil
	case "delete":
		if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
			return nil, "", fmt.Errorf("cloudcc delete validationRule <projectPath> <ruleId> now creates a MetadataService delete plan")
		}
		return map[string]any{"validationRules": []any{map[string]any{"id": args[0]}}}, operation, nil
	default:
		return shortcutBodySpec(action, "validationRule", args)
	}
}

func handleWorkflowReadShortcut(action string, projectPath string, args []string, stdout io.Writer, cwd string) error {
	action = strings.TrimSpace(action)
	c, _, err := newClient([]string{projectPath}, cwd)
	if err != nil {
		return err
	}
	if action == "detail" || action == "editInfo" || action == "validDelete" {
		if len(args) != 1 || strings.TrimSpace(args[0]) == "" {
			return fmt.Errorf("cloudcc %s workflow <projectPath> <workflow-id>", action)
		}
		return c.getJSON(stdout, "/metadata/v1/workflows/"+url.PathEscape(strings.TrimSpace(args[0])))
	}
	if len(args) > 1 {
		return fmt.Errorf("cloudcc %s workflow <projectPath> [object-id-or-filter]", action)
	}
	path := "/metadata/v1/workflows"
	if len(args) == 1 && strings.TrimSpace(args[0]) != "" {
		values := url.Values{}
		values.Set("filter", strings.TrimSpace(args[0]))
		path += "?" + values.Encode()
	}
	return c.getJSON(stdout, path)
}

func workflowShortcutSpec(action string, args []string) (map[string]any, string, error) {
	operation := shortcutOperation(action, "workflow")
	if len(args) > 0 && looksLikeJSONArg(args[0]) {
		body, err := parseObject(args[0], "cloudcc "+action+" workflow")
		return body, operation, err
	}
	switch strings.TrimSpace(action) {
	case "create":
		if len(args) < 3 {
			return nil, "", fmt.Errorf("cloudcc create workflow <projectPath> <targetObjectId> <name> <encodedOptionsJson> now creates a MetadataService plan")
		}
		options, err := parseObject(args[2], "cloudcc create workflow")
		if err != nil {
			return nil, "", err
		}
		options["targetobjectid"] = strings.TrimSpace(args[0])
		options["targetObjectId"] = strings.TrimSpace(args[0])
		options["name"] = strings.TrimSpace(args[1])
		return options, operation, nil
	case "update", "modify", "editSave", "save":
		if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
			return nil, "", fmt.Errorf("cloudcc %s workflow <projectPath> <workflowId|encodedWorkflowJson> [encodedOptionsJson] now creates a MetadataService plan", action)
		}
		if len(args) > 1 && looksLikeJSONArg(args[1]) {
			body, err := parseObject(args[1], "cloudcc "+action+" workflow")
			if err != nil {
				return nil, "", err
			}
			if firstShortcutValue(body, "id", "workflowId", "objid") == "" {
				body["id"] = strings.TrimSpace(args[0])
			}
			return body, operation, nil
		}
		return map[string]any{"id": strings.TrimSpace(args[0])}, operation, nil
	case "delete", "remove", "enable", "disable", "activate", "deactivate":
		if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
			return nil, "", fmt.Errorf("cloudcc %s workflow <projectPath> <workflowId> now creates a MetadataService plan", action)
		}
		return map[string]any{"id": strings.TrimSpace(args[0])}, operation, nil
	default:
		return shortcutBodySpec(action, "workflow", args)
	}
}

func shortcutBodySpec(action string, resource string, args []string) (map[string]any, string, error) {
	if len(args) > 0 && looksLikeJSONArg(args[0]) {
		body, err := parseObject(args[0], "cloudcc "+action+" "+resource)
		return body, shortcutOperation(action, resource), err
	}
	if len(args) > 1 && looksLikeJSONArg(args[1]) {
		body, err := parseObject(args[1], "cloudcc "+action+" "+resource)
		if err != nil {
			return nil, "", err
		}
		if _, ok := body["id"]; !ok && strings.TrimSpace(args[0]) != "" {
			body["id"] = args[0]
		}
		return body, shortcutOperation(action, resource), nil
	}
	if strings.TrimSpace(action) == "delete" {
		if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
			return nil, "", fmt.Errorf("cloudcc delete %s <projectPath> <id> now creates a MetadataService delete plan", resource)
		}
		body := map[string]any{"id": args[0]}
		if len(args) > 1 && strings.TrimSpace(args[1]) != "" {
			body["objectId"] = args[1]
		}
		if len(args) > 2 && strings.TrimSpace(args[2]) != "" {
			body["replaceId"] = args[2]
		}
		return body, shortcutOperation(action, resource), nil
	}
	return nil, "", fmt.Errorf("cloudcc %s %s legacy setup-svc shortcut is closed; provide a MetadataService spec JSON or use cloudcc plan msapi <projectPath> %s @spec.json", action, resource, lowCodeShortcutDomains[resource])
}

func shortcutOperation(action string, resource string) string {
	if resource == "object" && strings.TrimSpace(action) == "purge" {
		return "physical-purge"
	}
	if resource == "view" && (strings.TrimSpace(action) == "update" || strings.TrimSpace(action) == "editSave" || strings.TrimSpace(action) == "modify") {
		return "update"
	}
	switch strings.TrimSpace(action) {
	case "create":
		return "create"
	case "enable", "activate":
		return "activate"
	case "disable", "deactivate":
		return "deactivate"
	case "delete", "remove":
		return "delete"
	default:
		return "upsert"
	}
}

func looksLikeJSONArg(value string) bool {
	value = strings.TrimSpace(value)
	return strings.HasPrefix(value, "@") || strings.HasPrefix(value, "{") || strings.Contains(value, "%7B") || strings.Contains(value, "%7b")
}

func firstShortcutArg(args []string, index int) string {
	if index < 0 || index >= len(args) {
		return ""
	}
	return strings.TrimSpace(args[index])
}
