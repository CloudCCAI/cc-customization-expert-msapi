package openapi

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	if action == "uploadAttachment" {
		return uploadAttachment(args, stdout, cwd)
	}
	if action == "submitApproval" {
		return submitApproval(args, stdout, cwd)
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

func uploadAttachment(args []string, stdout io.Writer, cwd string) error {
	if len(args) < 3 {
		return fmt.Errorf("usage: cloudcc uploadAttachment openapi <projectPath> <recordId> <filePath> [optionsJson|@file]")
	}
	projectPath := args[0]
	if projectPath == "" {
		projectPath = cwd
	}
	recordID := strings.TrimSpace(args[1])
	if recordID == "" {
		return fmt.Errorf("Upload Attachment OpenAPI Failed: recordId is required")
	}
	filePath, err := filepath.Abs(args[2])
	if err != nil {
		return fmt.Errorf("Upload Attachment OpenAPI Failed: cannot resolve file path: %w", err)
	}
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("Upload Attachment OpenAPI Failed: cannot read %s: %w", filePath, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("Upload Attachment OpenAPI Failed: attachment path is not a regular file: %s", filePath)
	}

	options := map[string]any{}
	if len(args) > 3 && strings.TrimSpace(args[3]) != "" {
		options, err = jsonx.ParseEncodedObject(args[3], "Upload Attachment OpenAPI options")
		if err != nil {
			return err
		}
	}
	allowed := map[string]bool{"fileName": true, "groupid": true, "libid": true, "parentid": true, "isFromEmail": true, "sourceForm": true}
	for key := range options {
		if !allowed[key] {
			return fmt.Errorf("Upload Attachment OpenAPI Failed: unsupported option %q", key)
		}
	}
	fileName := stringOption(options, "fileName", filepath.Base(filePath))
	fileType := strings.TrimPrefix(strings.ToLower(filepath.Ext(fileName)), ".")
	if strings.TrimSpace(fileName) == "" || fileName == "." || fileType == "" {
		return fmt.Errorf("Upload Attachment OpenAPI Failed: fileName must include a suffix")
	}

	apiSvc, accessToken, err := loadEndpoint(projectPath)
	if err != nil {
		return err
	}
	fields := map[string]string{
		"fileName":    fileName,
		"groupid":     stringOption(options, "groupid", ""),
		"libid":       stringOption(options, "libid", ""),
		"parentid":    stringOption(options, "parentid", ""),
		"isFromEmail": stringOption(options, "isFromEmail", ""),
		"recordId":    recordID,
	}
	if sourceForm := stringOption(options, "sourceForm", ""); sourceForm != "" {
		fields["sourceForm"] = sourceForm
	}

	var uploadResponse map[string]any
	if err := httpclient.New().PostMultipartFile(apiSvc+"/api/file/upload", filePath, fileName, fields, accessToken, &uploadResponse); err != nil {
		return fmt.Errorf("Upload Attachment OpenAPI Failed: %w", err)
	}
	if !responseSucceeded(uploadResponse) {
		return responseError("Upload Attachment", uploadResponse, "upload failed")
	}
	uploadData, ok := uploadResponse["data"].(map[string]any)
	if !ok {
		return fmt.Errorf("Upload Attachment OpenAPI Failed: upload response data is missing")
	}
	fileContentID := first(fmt.Sprint(uploadData["fileContentId"]), fmt.Sprint(uploadData["fileContentID"]))
	fileInfoID := first(fmt.Sprint(uploadData["fileinfoid"]), fmt.Sprint(uploadData["fileInfoId"]))
	if fileContentID == "" || fileInfoID == "" {
		return fmt.Errorf("Upload Attachment OpenAPI Failed: upload response is missing fileContentId or fileinfoid")
	}
	bindItem := map[string]any{
		"recordid":      recordID,
		"name":          first(fmt.Sprint(uploadData["name"]), fileName),
		"type":          first(fmt.Sprint(uploadData["type"]), fileType),
		"fileContentId": fileContentID,
		"fileinfoid":    fileInfoID,
		"filesize":      first(fmt.Sprint(uploadData["filesize"]), fmt.Sprint(info.Size())),
	}
	var bindResponse map[string]any
	if err := httpclient.New().PostClass(apiSvc+"/api/file/bind", []map[string]any{bindItem}, accessToken, &bindResponse); err != nil {
		return fmt.Errorf("Bind Attachment OpenAPI Failed after upload (fileContentId=%s, fileinfoid=%s): %w", fileContentID, fileInfoID, err)
	}
	if !responseSucceeded(bindResponse) {
		return fmt.Errorf("%w; uploaded file remains unbound (fileContentId=%s, fileinfoid=%s)", responseError("Bind Attachment", bindResponse, "bind failed"), fileContentID, fileInfoID)
	}
	result := map[string]any{
		"result":     true,
		"action":     "uploadAttachment",
		"recordId":   recordID,
		"upload":     uploadData,
		"attachment": bindResponse["data"],
	}
	b, _ := json.Marshal(result)
	fmt.Fprintln(stdout, string(b))
	return nil
}

func submitApproval(args []string, stdout io.Writer, cwd string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: cloudcc submitApproval openapi <projectPath> <bodyJson|@file>")
	}
	projectPath := args[0]
	if projectPath == "" {
		projectPath = cwd
	}
	body, err := jsonx.ParseEncodedObject(args[1], "Submit Approval OpenAPI")
	if err != nil {
		return err
	}
	allowed := map[string]bool{"relatedId": true, "fprId": true, "appPath": true, "comments": true}
	for key := range body {
		if !allowed[key] {
			return fmt.Errorf("Submit Approval OpenAPI Failed: unsupported field %q", key)
		}
	}
	relatedID, ok := body["relatedId"].(string)
	if !ok || strings.TrimSpace(relatedID) == "" {
		return fmt.Errorf("Submit Approval OpenAPI Failed: relatedId is required")
	}
	for _, key := range []string{"fprId", "appPath", "comments"} {
		if value, exists := body[key]; exists {
			if _, ok := value.(string); !ok {
				return fmt.Errorf("Submit Approval OpenAPI Failed: %s must be a string", key)
			}
		}
	}
	apiSvc, accessToken, err := loadEndpoint(projectPath)
	if err != nil {
		return err
	}
	var response map[string]any
	if err := httpclient.New().PostClass(apiSvc+"/api/approval/submitApproval", body, accessToken, &response); err != nil {
		return fmt.Errorf("Submit Approval OpenAPI Failed: %w", err)
	}
	if !responseSucceeded(response) {
		code := fmt.Sprint(response["returnCode"])
		if code == "Manual" {
			return fmt.Errorf("Submit Approval OpenAPI Failed [Manual]: the approval process requires an explicit next approver; retry with fprId")
		}
		return responseError("Submit Approval", response, "submission failed")
	}
	b, _ := json.Marshal(response)
	fmt.Fprintln(stdout, string(b))
	return nil
}

func loadEndpoint(projectPath string) (string, string, error) {
	cfg, err := config.Load(projectPath)
	if err != nil {
		return "", "", err
	}
	apiSvc := strings.TrimRight(first(config.String(cfg, "apiSvc"), config.String(cfg, "apisvc")), "/")
	accessToken := first(config.String(cfg, "accessToken"), config.String(cfg, "token"))
	if apiSvc == "" || accessToken == "" {
		return "", "", fmt.Errorf("OpenAPI Failed: apiSvc or accessToken is missing in resolved config")
	}
	return apiSvc, accessToken, nil
}

func responseSucceeded(response map[string]any) bool {
	return response["result"] == true || fmt.Sprint(response["returnCode"]) == "1"
}

func responseError(label string, response map[string]any, fallback string) error {
	code := first(fmt.Sprint(response["returnCode"]), fmt.Sprint(response["code"]))
	message := first(fmt.Sprint(response["returnInfo"]), fmt.Sprint(response["message"]), fallback)
	if code != "" {
		return fmt.Errorf("%s OpenAPI Failed [%s]: %s", label, code, message)
	}
	return fmt.Errorf("%s OpenAPI Failed: %s", label, message)
}

func stringOption(options map[string]any, key string, fallback string) string {
	value, exists := options[key]
	if !exists || value == nil {
		return fallback
	}
	if text, ok := value.(string); ok {
		return text
	}
	return fmt.Sprint(value)
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
