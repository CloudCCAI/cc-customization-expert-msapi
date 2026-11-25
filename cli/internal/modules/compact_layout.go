package modules

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"slices"
	"strings"
	"unicode/utf16"

	"cloudcc-customization-expert-go/internal/config"
	"cloudcc-customization-expert-go/internal/httpclient"
	"cloudcc-customization-expert-go/internal/jsonx"
)

var compactID = regexp.MustCompile("^[A-Za-z0-9_]+$")

type compactUI struct {
	project string
	cfg     config.Config
	object  map[string]any
}

func handleCompactLayout(action string, args []string, stdout io.Writer, cwd string) error {
	project := firstArg(args, cwd)
	args = argsAfterProject(args)
	var spec map[string]any
	var selector string
	switch action {
	case "impact", "delete":
		return fmt.Errorf("compact_layout_delete_requires_msapi")
	case "create", "update", "assign":
		if len(args) != 1 {
			return fmt.Errorf("cloudcc %s compactLayout <projectPath> <specJson|@file>", action)
		}
		var err error
		spec, err = jsonx.ParseEncodedObject(args[0], action+" compactLayout")
		if err != nil {
			return err
		}
		keys := map[string][]string{
			"create": {"object", "apiName", "label", "fields"},
			"update": {"object", "selector", "apiName", "label", "fields"},
			"assign": {"object", "mainLayout", "recordTypeAssignments"},
		}
		if err = compactKeys(spec, keys[action]); err != nil {
			return err
		}
		selector = compactText(spec["object"])
	case "get", "capabilities", "assignments", "detail", "candidates":
		expected := 1
		if action == "detail" || action == "candidates" {
			expected = 2
		}
		if len(args) != expected || (expected == 2 && strings.TrimSpace(args[1]) == "") {
			return fmt.Errorf("invalid compactLayout %s arguments", action)
		}
		selector = strings.TrimSpace(args[0])
	default:
		return fmt.Errorf("unsupported compactLayout action %s", action)
	}
	if selector == "" {
		return fmt.Errorf("compactLayout requires a nonblank object selector")
	}
	mode, err := config.ProjectPlatformMode(project)
	if err != nil {
		return err
	}
	if mode != "lightning" {
		return fmt.Errorf("compactLayout is unavailable in horizontal UIAPI")
	}
	cfg, err := config.Load(project)
	if err != nil {
		return err
	}
	ui := compactUI{project: project, cfg: cfg}
	var objects bytes.Buffer
	if err := objectGet([]string{project}, &objects, cwd); err != nil {
		return err
	}
	var rows []map[string]any
	if err := json.Unmarshal(objects.Bytes(), &rows); err != nil {
		return err
	}
	object, err := compactResolve(rows, selector, "apiname", "schemetableName", "objprefix")
	if err != nil {
		return err
	}
	ui.object = map[string]any{"id": object["id"], "apiName": object["apiname"], "label": object["label"],
		"prefix": object["objprefix"]}
	result, err := ui.run(action, args, spec)
	if err != nil {
		return err
	}
	result["domain"], result["object"] = "compact-layouts", ui.object
	return printJSON(stdout, result)
}

func (u compactUI) call(action string, body map[string]any, write bool) (any, error) {
	id := compactText(u.object["id"])
	if !compactID.MatchString(id) {
		return nil, fmt.Errorf("compactLayout requires a platform object ID")
	}
	body["objid"] = id
	const path = "/api/keyMessageLayout/"
	var response map[string]any
	var err error
	if write {
		token := firstNonBlankString(strings.TrimSpace(os.Getenv("CLOUDCC_ACCESS_TOKEN")), config.String(u.cfg, "accessToken"))
		err = httpclient.New().PostClassOnce(postClassServiceBase(u.cfg, "setup")+path+action, body, token, &response)
	} else {
		err = postClassResponse(u.project, u.cfg, "setup", path+action, body, &response)
	}
	if err != nil {
		if write {
			return nil, fmt.Errorf("write_outcome_unknown: compactLayout %s transport failed", action)
		}
		return nil, err
	}
	if code := fmt.Sprint(response["returnCode"]); response["result"] == false ||
		(code != "<nil>" && code != "1") || compactText(response["returnInfo"]) != "" {
		if write && action != "save" {
			return nil, fmt.Errorf("write_outcome_unknown: compactLayout %s returned a platform error", action)
		}
		return nil, fmt.Errorf("compact_layout_setup_error: %s", compactText(response["returnInfo"]))
	}
	return response["data"], nil
}

func (u compactUI) layouts() ([]map[string]any, error) {
	data, err := u.call("queryList", map[string]any{}, false)
	if err != nil {
		return nil, err
	}
	rows, err := compactRows(data)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		row["apiName"] = row["name"]
		row["objectId"] = u.object["id"]
		row["isMain"] = compactFlag(row["ismain"])
		row["isSystemDefault"] = compactFlag(row["isdefault"])
		row["lastModifiedAt"] = row["lastmodifydate"]
		row["lastModifiedById"] = row["lastmodifybyid"]
	}
	return rows, nil
}

func (u compactUI) edit(id string) (map[string]any, error) {
	if id != "" && !compactID.MatchString(id) {
		return nil, fmt.Errorf("invalid compactLayout ID")
	}
	data, err := u.call("editPage", map[string]any{"id": id}, false)
	if err != nil {
		return nil, err
	}
	result, ok := data.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid compactLayout edit response")
	}
	if id != "" {
		root, _ := result["compactLayout"].(map[string]any)
		if compactText(root["id"]) != id || compactText(root["object_id"]) != compactText(u.object["id"]) {
			return nil, fmt.Errorf("compactLayout edit target mismatch")
		}
	}
	return result, nil
}

func (u compactUI) run(action string, args []string, spec map[string]any) (map[string]any, error) {
	switch action {
	case "capabilities":
		if _, err := u.layouts(); err != nil {
			return nil, err
		}
		return map[string]any{"available": true, "executionBackend": "setup-service",
			"actions":     []string{"get", "detail", "capabilities", "candidates", "assignments", "create", "update", "assign"},
			"fieldLimits": map[string]int{"min": 1, "max": 10}, "assignmentModes": []string{"full-setup-projection"},
			"reasons": []string{"delete_requires_msapi", "explicit_assignment_readback_unavailable"}}, nil
	case "candidates":
		var candidates []map[string]any
		var err error
		switch args[1] {
		case "fields":
			candidates, err = u.fieldCandidates()
		case "layouts", "recordTypes":
			var layouts, records []map[string]any
			layouts, records, err = u.assignmentState()
			candidates = layouts
			if args[1] == "recordTypes" {
				candidates = records
			}
		default:
			return nil, fmt.Errorf("compactLayout candidate kind must be fields, recordTypes, or layouts")
		}
		if err != nil {
			return nil, err
		}
		return map[string]any{"kind": args[1], "candidates": candidates, "count": len(candidates)}, nil
	case "assignments", "assign":
		return u.assign(action, spec)
	}
	layouts, err := u.layouts()
	if err != nil {
		return nil, err
	}
	if action == "get" {
		return map[string]any{"layouts": layouts, "count": len(layouts)}, nil
	}
	var row map[string]any
	if action == "detail" || action == "update" {
		selector := compactText(spec["selector"])
		if action == "detail" {
			selector = args[1]
		}
		row, err = compactResolve(layouts, selector, "apiName")
		if err != nil {
			return nil, err
		}
	}
	if action == "detail" {
		return u.detail(row)
	}
	return u.save(action, spec, row, layouts)
}

func (u compactUI) detail(row map[string]any) (map[string]any, error) {
	data, err := u.edit(compactText(row["id"]))
	if err != nil {
		return nil, err
	}
	fields, err := compactRows(data["selectedFieldList"])
	if err != nil {
		return nil, err
	}
	root := data["compactLayout"].(map[string]any)
	row["apiName"], row["label"] = root["name"], root["label"]
	row["fields"] = fields
	return map[string]any{"layout": row, "diagnostics": []string{}}, nil
}

func (u compactUI) fieldCandidates() ([]map[string]any, error) {
	result, err := u.edit("")
	if err != nil {
		return nil, err
	}
	fields, err := compactRows(result["unselectedFieldList"])
	if err != nil {
		return nil, err
	}
	// editPage lists layout-eligible IDs, while queryField supplies API names.
	if prefix := compactText(u.object["prefix"]); prefix != "" {
		var raw map[string]any
		if err := postClassResponse(u.project, u.cfg, "setup", "/api/fieldSetup/queryField",
			map[string]any{"prefix": prefix}, &raw); err != nil {
			return nil, err
		}
		metadata := map[string]map[string]any{}
		compactCollectFields(raw["data"], metadata)
		usable := make([]map[string]any, 0, len(fields))
		for _, row := range fields {
			row["eligible"] = true
			if meta := metadata[compactText(row["id"])]; meta != nil {
				row["apiName"] = firstAny(meta["apiName"], meta["apiname"], meta["schemefieldName"], meta["schemefieldname"], meta["schemefield_name"])
				row["type"] = firstAny(meta["fieldType"], meta["schemefieldType"], meta["schemefieldtype"], meta["schemefield_type"])
				state := compactText(firstAny(meta["fieldState"], meta["field_state"]))
				if compactFlag(firstAny(meta["isDeleted"], meta["is_deleted"])) || (state != "" && state != "enable") {
					row["eligible"], row["exclusionReason"] = false, "inactive_field"
				}
			}
			usable = append(usable, row)
		}
		return usable, nil
	}
	for _, row := range fields {
		row["eligible"] = true
	}
	return fields, nil
}

func compactCollectFields(data any, result map[string]map[string]any) {
	switch value := data.(type) {
	case []any:
		for _, item := range value {
			compactCollectFields(item, result)
		}
	case map[string]any:
		if id := compactText(value["id"]); id != "" &&
			firstAny(value["apiName"], value["apiname"], value["schemefieldName"], value["schemefieldname"], value["schemefield_name"]) != nil {
			result[id] = value
		}
		for _, item := range value {
			compactCollectFields(item, result)
		}
	}
}

func (u compactUI) save(action string, spec, row map[string]any, layouts []map[string]any) (map[string]any, error) {
	if action == "update" {
		if _, err := u.detail(row); err != nil {
			return nil, err
		}
	}
	id := compactText(row["id"])
	name, label := compactText(row["apiName"]), compactText(row["label"])
	if v, ok := spec["apiName"]; ok {
		name = compactText(v)
	}
	if v, ok := spec["label"]; ok {
		label = compactText(v)
	}
	if name == "" || label == "" || len(utf16.Encode([]rune(name))) > 255 || len(utf16.Encode([]rune(label))) > 255 {
		return nil, fmt.Errorf("compactLayout requires apiName and label of 1–255 characters")
	}
	if strings.ContainsAny(name+label, "'\\\x00") {
		return nil, fmt.Errorf("Lightning compactLayout save does not support apostrophes, backslashes, or NUL in names and labels; use MSAPI")
	}
	if row["isSystemDefault"] == true && name != row["apiName"] {
		return nil, fmt.Errorf("the system layout name cannot change")
	}
	for _, current := range layouts {
		if current["id"] != id && (strings.EqualFold(compactText(current["apiName"]), name) ||
			strings.EqualFold(compactText(current["label"]), label)) {
			return nil, fmt.Errorf("compact_layout_duplicate_name")
		}
	}
	var requested []any
	if v, supplied := spec["fields"]; supplied {
		requested, _ = v.([]any)
	} else if action == "update" && (spec["label"] != nil || spec["apiName"] != nil) {
		for _, field := range row["fields"].([]map[string]any) {
			requested = append(requested, field["id"])
		}
	}
	if len(requested) < 1 || len(requested) > 10 {
		return nil, fmt.Errorf("compactLayout requires fields with 1–10 eligible selectors, or a valid current field list")
	}
	candidates, err := u.fieldCandidates()
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, selection := range requested {
		field, err := compactResolve(candidates, compactText(selection), "apiName")
		fieldID := compactText(field["id"])
		if err != nil || field["eligible"] != true || !compactID.MatchString(fieldID) || slices.Contains(ids, fieldID) {
			return nil, fmt.Errorf("compactLayout field is unknown, ineligible, or duplicated: %s", compactText(selection))
		}
		ids = append(ids, fieldID)
	}
	body := map[string]any{"selectedFieldIds": strings.Join(ids, ","), "compactLayout": map[string]any{"id": id, "label": label, "name": name}}
	if _, err := u.call("save", body, true); err != nil {
		return nil, err
	}
	actual, err := u.layouts()
	if err != nil {
		return nil, fmt.Errorf("write_outcome_unknown: compactLayout list readback failed")
	}
	if id == "" {
		id = name
	}
	row, err = compactResolve(actual, id, "apiName")
	if err != nil || row["apiName"] != name || row["label"] != label {
		return nil, fmt.Errorf("write_outcome_unknown: compactLayout identity or label did not match")
	}
	result, err := u.detail(row)
	if err != nil {
		return nil, fmt.Errorf("write_outcome_unknown: compactLayout detail readback failed")
	}
	if row["apiName"] != name || row["label"] != label {
		return nil, fmt.Errorf("write_outcome_unknown: compactLayout detail name or label did not match")
	}
	var observed []string
	for _, field := range row["fields"].([]map[string]any) {
		observed = append(observed, compactText(field["id"]))
	}
	if !slices.Equal(ids, observed) {
		return nil, fmt.Errorf("write_outcome_unknown: compactLayout ordered fields did not match")
	}
	result["verification"] = "setupReadback"
	return result, nil
}

func (u compactUI) assignmentState() ([]map[string]any, []map[string]any, error) {
	data, err := u.call("editRecordtypeLayout", map[string]any{}, false)
	if err != nil {
		return nil, nil, err
	}
	response, ok := data.(map[string]any)
	if !ok {
		return nil, nil, fmt.Errorf("invalid compactLayout assignment response")
	}
	layouts, err := compactRows(response["list"])
	if err != nil {
		return nil, nil, err
	}
	records, err := compactRows(response["recordtypeLayouts"])
	if err != nil {
		return nil, nil, err
	}
	for _, row := range layouts {
		row["apiName"], row["eligible"] = row["name"], true
	}
	for _, row := range records {
		row["id"], row["apiName"], row["label"], row["eligible"] = row["recordtypeid"], row["rname"], row["recordtypelabel"], true
	}
	return layouts, records, nil
}

func (u compactUI) assign(action string, spec map[string]any) (map[string]any, error) {
	layouts, records, err := u.assignmentState()
	if err != nil {
		return nil, err
	}
	if action == "assign" {
		main, err := compactResolve(layouts, compactText(spec["mainLayout"]), "apiName")
		if err != nil {
			return nil, fmt.Errorf("Lightning compactLayout assign requires an explicit mainLayout: %w", err)
		}
		mainID := compactText(main["id"])
		items, err := compactRows(spec["recordTypeAssignments"])
		_, supplied := spec["recordTypeAssignments"].([]any)
		if !compactID.MatchString(mainID) || !supplied || err != nil || len(items) != len(records) {
			return nil, fmt.Errorf("Lightning compactLayout assign requires mappings for all record types")
		}
		expected := map[string]string{}
		var pairs []string
		for _, item := range items {
			if err := compactKeys(item, []string{"recordType", "layout"}); err != nil {
				return nil, err
			}
			record, err := compactResolve(records, compactText(item["recordType"]), "apiName")
			if err != nil {
				return nil, err
			}
			layout, err := compactResolve(layouts, compactText(item["layout"]), "apiName")
			if err != nil {
				return nil, err
			}
			recordID, layoutID := compactText(record["id"]), compactText(layout["id"])
			if !compactID.MatchString(recordID) || !compactID.MatchString(layoutID) || expected[recordID] != "" {
				return nil, fmt.Errorf("invalid or repeated compactLayout assignment")
			}
			expected[recordID] = layoutID
			pairs = append(pairs, recordID+","+layoutID)
		}
		if compactProjectionMatches(layouts, records, mainID, expected) {
			result := compactProjection(layouts, records)
			result["verification"] = "unchangedProjection"
			return result, nil
		}
		_, err = u.call("saveRecordLayout", map[string]any{"mainId": mainID, "saveContent": strings.Join(pairs, ";")}, true)
		if err != nil {
			return nil, err
		}
		layouts, records, err = u.assignmentState()
		if err != nil || !compactProjectionMatches(layouts, records, mainID, expected) {
			return nil, fmt.Errorf("write_outcome_unknown: compactLayout assignment projection did not match")
		}
		result := compactProjection(layouts, records)
		result["verification"] = "setupProjection"
		return result, nil
	}
	return compactProjection(layouts, records), nil
}

func compactProjectionMatches(layouts, records []map[string]any, main string, expected map[string]string) bool {
	if len(records) != len(expected) {
		return false
	}
	mainCount := 0
	for _, row := range layouts {
		if compactFlag(row["ismain"]) {
			if row["id"] != main {
				return false
			}
			mainCount++
		}
	}
	seen := map[string]bool{}
	for _, row := range records {
		id := compactText(row["id"])
		if seen[id] || expected[id] == "" || expected[id] != compactText(row["layoutid"]) {
			return false
		}
		seen[id] = true
	}
	return mainCount == 1
}

func compactProjection(layouts, records []map[string]any) map[string]any {
	var main, fallback []string
	for _, row := range layouts {
		if compactFlag(row["ismain"]) {
			main = append(main, compactText(row["id"]))
		}
		if compactFlag(row["isdefault"]) {
			fallback = append(fallback, compactText(row["id"]))
		}
	}
	effective := make([]map[string]any, 0, len(records))
	for _, row := range records {
		effective = append(effective, map[string]any{"recordTypeId": row["id"], "layoutId": row["layoutid"], "source": "unknown"})
	}
	result := map[string]any{"projection": "setup", "explicitAssignmentsKnown": false, "recordTypeAssignments": nil,
		"mainLayoutId": nil, "systemDefaultLayoutId": nil, "effectiveAssignments": effective}
	if len(main) == 1 {
		result["mainLayoutId"] = main[0]
	}
	if len(fallback) == 1 {
		result["systemDefaultLayoutId"] = fallback[0]
	}
	return result
}

func compactKeys(obj map[string]any, keys []string) error {
	for key := range obj {
		if !slices.Contains(keys, key) {
			return fmt.Errorf("unknown compactLayout property %q", key)
		}
	}
	return nil
}

func compactRows(data any) ([]map[string]any, error) {
	rows, ok := data.([]any)
	if !ok {
		return nil, fmt.Errorf("compactLayout response or input must contain an array")
	}
	result := make([]map[string]any, 0, len(rows))
	for _, value := range rows {
		row, ok := value.(map[string]any)
		if !ok || row == nil {
			return nil, fmt.Errorf("compactLayout expected an object row")
		}
		result = append(result, row)
	}
	return result, nil
}

func compactResolve(rows []map[string]any, selector string, keys ...string) (map[string]any, error) {
	if selector == "" {
		return nil, fmt.Errorf("compactLayout requires a nonblank selector")
	}
	for _, fields := range [][]string{{"id"}, keys} {
		var found map[string]any
		for _, row := range rows {
			for _, key := range fields {
				if strings.EqualFold(selector, compactText(row[key])) {
					if found != nil {
						return nil, fmt.Errorf("compactLayout selector is ambiguous: %s", selector)
					}
					found = row
					break
				}
			}
		}
		if found != nil {
			return found, nil
		}
	}
	return nil, fmt.Errorf("compactLayout selector not found: %s", selector)
}

func compactText(v any) string {
	value, _ := v.(string)
	return strings.TrimSpace(value)
}

func compactFlag(v any) bool { return fmt.Sprint(v) == "1" || v == true }
