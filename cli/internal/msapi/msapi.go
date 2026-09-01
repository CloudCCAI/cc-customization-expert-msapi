package msapi

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"cloudcc-customization-expert-go/internal/config"
	"cloudcc-customization-expert-go/internal/jsonx"
)

const defaultServiceURL = "http://127.0.0.1:8087"

var metadataServicePromptReader io.Reader = os.Stdin
var metadataServicePromptWriter io.Writer = os.Stderr

type client struct {
	baseURL          string
	token            string
	cloudccUserToken string
	projectPath      string
	http             *http.Client
}

// Handle exposes the agent-facing MetadataService API through the normal
// CloudCC CLI shape: cloudcc <action> msapi ... or cloudcc <action> metadata ...
// The legacy setup/UI API commands remain available; this module is the
// explicit plan/apply/rollback path for metadata changes that need a ledger.
func Handle(action string, resource string, args []string, stdout io.Writer, cwd string) error {
	c, remaining, err := newClient(args, cwd)
	if err != nil {
		return err
	}
	switch action {
	case "capabilities", "capability":
		return c.getJSON(stdout, "/metadata/v1/capabilities")
	case "scan":
		return c.scan(stdout, remaining)
	case "resolve", "references":
		body, err := readObjectArg(remaining, 0, "cloudcc resolve "+resource+" <encodedReferencesJson>")
		if err != nil {
			return err
		}
		return c.writeJSON(stdout, http.MethodPost, "/metadata/v1/references:resolve", body)
	case "normalize", "validate", "plan":
		body, err := planRequest(resource, action, remaining)
		if err != nil {
			return err
		}
		path := "/metadata/v1/plans"
		if action == "normalize" {
			path = "/metadata/v1/intents:normalize"
		} else if action == "validate" {
			path = "/metadata/v1/intents:validate"
		}
		return c.writeJSON(stdout, http.MethodPost, path, body)
	case "apply":
		if len(remaining) > 0 && isSetupSvcLiveReplayPacketMode(remaining[0]) {
			return c.applySetupSvcLiveReplayPacket(stdout, remaining[1:])
		}
		if len(remaining) > 0 && isSetupSvcLiveReplayWorkspaceMode(remaining[0]) {
			return c.applySetupSvcLiveReplayWorkspace(stdout, remaining[1:])
		}
		if len(remaining) > 0 && isSetupSvcLiveReplayCaptureSourceWorkspaceMode(remaining[0]) {
			return c.applySetupSvcLiveReplayCaptureSourceWorkspace(stdout, remaining[1:])
		}
		if len(remaining) > 0 && isSetupSvcLiveReplayNormalizedDiffMode(remaining[0]) {
			return c.applySetupSvcLiveReplayNormalizedDiff(stdout, remaining[1:])
		}
		if len(remaining) > 0 && isSetupSvcLiveReplayManifestSyncMode(remaining[0]) {
			return c.applySetupSvcLiveReplayManifestSync(stdout, remaining[1:])
		}
		if len(remaining) > 0 && isSetupSvcLiveReplayEvidenceBundleMode(remaining[0]) {
			return c.applySetupSvcLiveReplayEvidenceBundle(stdout, remaining[1:])
		}
		if len(remaining) > 0 && isSetupSvcLiveReplayEvidenceImportMode(remaining[0]) {
			return c.applySetupSvcLiveReplayEvidenceImport(stdout, remaining[1:])
		}
		if len(remaining) > 0 && isSetupSvcLiveReplayQueryReadbackCaptureMode(remaining[0]) {
			return c.applySetupSvcLiveReplayQueryReadbackCapture(stdout, remaining[1:])
		}
		if len(remaining) > 0 && isSetupSvcLiveReplayMetadataServiceQueryScanCaptureMode(remaining[0]) {
			return c.applySetupSvcLiveReplayMetadataServiceQueryScanCapture(stdout, remaining[1:])
		}
		if len(remaining) > 0 && isSetupSvcLiveReplayMetadataServiceApplyCaptureMode(remaining[0]) {
			return c.applySetupSvcLiveReplayMetadataServiceApplyCapture(stdout, remaining[1:])
		}
		if len(remaining) > 0 && isSetupSvcLiveReplaySnapshotFromChangesMode(remaining[0]) {
			return c.applySetupSvcLiveReplaySnapshotFromChanges(stdout, remaining[1:])
		}
		if len(remaining) > 0 && isSetupSvcLiveReplayMatrixPromotionMode(remaining[0]) {
			return c.applySetupSvcLiveReplayMatrixPromotion(stdout, remaining[1:])
		}
		if len(remaining) < 1 || strings.TrimSpace(remaining[0]) == "" {
			return fmt.Errorf("cloudcc apply %s <planId> [encodedApplyRequest]", resource)
		}
		body := map[string]any{}
		if len(remaining) > 1 && strings.TrimSpace(remaining[1]) != "" {
			body, err = parseObject(remaining[1], "cloudcc apply "+resource)
			if err != nil {
				return err
			}
		}
		return c.writeJSON(stdout, http.MethodPost, "/metadata/v1/plans/"+url.PathEscape(remaining[0])+":apply", body)
	case "operation", "get":
		if len(remaining) < 1 || strings.TrimSpace(remaining[0]) == "" {
			return fmt.Errorf("cloudcc operation %s <operationId>", resource)
		}
		return c.getJSON(stdout, "/metadata/v1/operations/"+url.PathEscape(remaining[0]))
	case "changes":
		if len(remaining) < 1 || strings.TrimSpace(remaining[0]) == "" {
			return fmt.Errorf("cloudcc changes %s <operationId>", resource)
		}
		return c.getJSON(stdout, "/metadata/v1/operations/"+url.PathEscape(remaining[0])+"/changes")
	case "rollback-plan":
		if len(remaining) < 1 || strings.TrimSpace(remaining[0]) == "" {
			return fmt.Errorf("cloudcc rollback-plan %s <operationId>", resource)
		}
		return c.writeJSON(stdout, http.MethodPost, "/metadata/v1/operations/"+url.PathEscape(remaining[0])+":rollback-plan", map[string]any{})
	case "rollback":
		if len(remaining) < 1 || strings.TrimSpace(remaining[0]) == "" {
			return fmt.Errorf("cloudcc rollback %s <operationId>", resource)
		}
		return c.writeJSON(stdout, http.MethodPost, "/metadata/v1/operations/"+url.PathEscape(remaining[0])+":rollback", map[string]any{})
	case "mutate":
		body, err := mutationRequest(resource, remaining)
		if err != nil {
			return err
		}
		domain, _ := body["domain"].(string)
		delete(body, "domain")
		return c.writeJSON(stdout, http.MethodPost, "/metadata/v1/"+url.PathEscape(domain)+":mutate", body)
	case "draft-create":
		body, err := draftCreateRequest(remaining)
		if err != nil {
			return err
		}
		return c.writeJSON(stdout, http.MethodPost, "/metadata/v1/drafts", body)
	case "draft-update":
		if len(remaining) < 2 {
			return fmt.Errorf("cloudcc draft-update %s <draftId> <encodedPatchJson>", resource)
		}
		body, err := parseObject(remaining[1], "cloudcc draft-update "+resource)
		if err != nil {
			return err
		}
		return c.writeJSON(stdout, http.MethodPatch, "/metadata/v1/drafts/"+url.PathEscape(remaining[0]), body)
	case "draft-validate", "draft-plan":
		if len(remaining) < 1 || strings.TrimSpace(remaining[0]) == "" {
			return fmt.Errorf("cloudcc %s %s <draftId>", action, resource)
		}
		suffix := ":validate"
		if action == "draft-plan" {
			suffix = ":plan"
		}
		return c.writeJSON(stdout, http.MethodPost, "/metadata/v1/drafts/"+url.PathEscape(remaining[0])+suffix, map[string]any{})
	case "draft-delete":
		if len(remaining) < 1 || strings.TrimSpace(remaining[0]) == "" {
			return fmt.Errorf("cloudcc draft-delete %s <draftId>", resource)
		}
		return c.writeJSON(stdout, http.MethodDelete, "/metadata/v1/drafts/"+url.PathEscape(remaining[0]), nil)
	default:
		return fmt.Errorf("unsupported MetadataService command: cloudcc %s %s", action, resource)
	}
}

func newClient(args []string, cwd string) (*client, []string, error) {
	projectPath := cwd
	remaining := append([]string{}, args...)
	if len(remaining) > 0 && isProjectPath(remaining[0]) {
		projectPath = remaining[0]
		remaining = remaining[1:]
	}
	baseURL, err := configuredServiceURL(projectPath)
	if err != nil {
		return nil, nil, err
	}
	return &client{
		baseURL:          strings.TrimRight(baseURL, "/"),
		token:            accessToken(projectPath),
		cloudccUserToken: cloudccUserToken(projectPath),
		projectPath:      projectPath,
		http:             &http.Client{Timeout: 180 * time.Second},
	}, remaining, nil
}

func serviceURL(projectPath string) (string, error) {
	if value, err := configuredServiceURL(projectPath); err != nil {
		return "", err
	} else if value != "" {
		return value, nil
	}
	root, err := config.Root(projectPath)
	if err != nil {
		return "", fmt.Errorf("MetadataService/MSAPI URL is required before calling msapi; cannot read %s: %w. Run cloudcc create project <name|.>, add metadataService.url to the active env, or set CLOUDCC_METADATA_SERVICE_URL", filepath.Join(projectPath, "cloudcc-cli.config.json"), err)
	}
	use, _ := root["use"].(string)
	if strings.TrimSpace(use) == "" {
		return "", fmt.Errorf("MetadataService/MSAPI URL is required before calling msapi; cloudcc-cli.config.json missing active use env")
	}
	active, _ := root[use].(map[string]any)
	if active == nil {
		return "", fmt.Errorf("MetadataService/MSAPI URL is required before calling msapi; cloudcc-cli.config.json missing env %s", use)
	}
	return promptAndSaveMetadataServiceURL(projectPath, root, use, active)
}

func configuredServiceURL(projectPath string) (string, error) {
	if value := strings.TrimSpace(os.Getenv("CLOUDCC_METADATA_SERVICE_URL")); value != "" {
		return strings.TrimRight(value, "/"), validateMetadataServiceURL(value)
	}
	root, err := config.Root(projectPath)
	if err != nil {
		return "", nil
	}
	use, _ := root["use"].(string)
	active, _ := root[use].(map[string]any)
	if active == nil {
		return "", nil
	}
	if value := stringValue(active["metadataServiceUrl"]); value != "" {
		return strings.TrimRight(value, "/"), validateMetadataServiceURL(value)
	}
	if value := stringValue(active["metadata_service_url"]); value != "" {
		return strings.TrimRight(value, "/"), validateMetadataServiceURL(value)
	}
	if ms, _ := active["metadataService"].(map[string]any); ms != nil {
		if value := stringValue(ms["url"]); value != "" {
			return strings.TrimRight(value, "/"), validateMetadataServiceURL(value)
		}
	}
	return "", nil
}

func promptAndSaveMetadataServiceURL(projectPath string, root map[string]any, use string, active map[string]any) (string, error) {
	if metadataServicePromptReader == os.Stdin && !stdinIsInteractive() {
		return "", fmt.Errorf("MetadataService/MSAPI URL is required before calling msapi; add metadataService.url to %s env %q or set CLOUDCC_METADATA_SERVICE_URL", filepath.Join(projectPath, "cloudcc-cli.config.json"), use)
	}
	fmt.Fprint(metadataServicePromptWriter, "请输入 MetadataService/MSAPI 地址（例如 http://127.0.0.1:8087）：")
	reader := bufio.NewReader(metadataServicePromptReader)
	value, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("cannot read MetadataService/MSAPI URL: %w", err)
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("MetadataService/MSAPI URL is required before calling msapi; add metadataService.url to %s env %q or set CLOUDCC_METADATA_SERVICE_URL", filepath.Join(projectPath, "cloudcc-cli.config.json"), use)
	}
	if err := validateMetadataServiceURL(value); err != nil {
		return "", err
	}
	metadataService, _ := active["metadataService"].(map[string]any)
	if metadataService == nil {
		metadataService = map[string]any{}
		active["metadataService"] = metadataService
	}
	metadataService["url"] = value
	root[use] = active
	if err := jsonx.WriteObjectFile(filepath.Join(projectPath, "cloudcc-cli.config.json"), root); err != nil {
		return "", fmt.Errorf("cannot save MetadataService/MSAPI URL to cloudcc-cli.config.json: %w", err)
	}
	fmt.Fprintln(metadataServicePromptWriter, "已保存到当前环境的 metadataService.url。")
	return value, nil
}

func stdinIsInteractive() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func validateMetadataServiceURL(value string) error {
	u, err := url.Parse(value)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("MetadataService/MSAPI URL must be an absolute http(s) URL, got %q", value)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("MetadataService/MSAPI URL must use http or https, got %q", value)
	}
	return nil
}

func accessToken(projectPath string) string {
	if value := strings.TrimSpace(os.Getenv("CLOUDCC_METADATA_SERVICE_ACCESS_TOKEN")); value != "" {
		return trimBearer(value)
	}
	if root, err := config.Root(projectPath); err == nil {
		use, _ := root["use"].(string)
		active, _ := root[use].(map[string]any)
		if value := tokenFromMap(active); value != "" {
			return trimBearer(value)
		}
	}
	cfg, err := config.Load(projectPath)
	if err != nil {
		return ""
	}
	return trimBearer(firstString(
		config.String(cfg, "metadataServiceAccessToken"),
		config.String(cfg, "accessToken"),
		config.String(cfg, "token"),
	))
}

func cloudccUserToken(projectPath string) string {
	if root, err := config.Root(projectPath); err == nil {
		use, _ := root["use"].(string)
		active, _ := root[use].(map[string]any)
		if value := firstString(stringValue(active["accessToken"]), stringValue(active["token"])); value != "" {
			return trimBearer(value)
		}
	}
	cfg, err := config.Load(projectPath)
	if err != nil {
		return ""
	}
	return trimBearer(firstString(
		config.String(cfg, "accessToken"),
		config.String(cfg, "token"),
	))
}

func tokenFromMap(active map[string]any) string {
	if active == nil {
		return ""
	}
	if ms, _ := active["metadataService"].(map[string]any); ms != nil {
		if value := stringValue(ms["accessToken"]); value != "" {
			return value
		}
		if value := stringValue(ms["token"]); value != "" {
			return value
		}
	}
	return firstString(
		stringValue(active["metadataServiceAccessToken"]),
		stringValue(active["accessToken"]),
		stringValue(active["token"]),
	)
}

func trimBearer(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(strings.ToLower(value), "bearer ") {
		return strings.TrimSpace(value[7:])
	}
	return value
}

func isProjectPath(value string) bool {
	if value == "." {
		return true
	}
	if strings.HasPrefix(value, "@") {
		return false
	}
	if strings.HasPrefix(strings.TrimSpace(value), "{") {
		return false
	}
	if strings.Contains(value, "%7B") || strings.Contains(value, "%7b") {
		return false
	}
	info, err := os.Stat(value)
	if err != nil || !info.IsDir() {
		return false
	}
	if _, err := os.Stat(filepath.Join(value, "cloudcc-cli.config.json")); err == nil {
		return true
	}
	return false
}

func planRequest(resource string, action string, args []string) (map[string]any, error) {
	if len(args) == 1 {
		body, err := parseObject(args[0], "cloudcc "+action+" "+resource)
		if err != nil {
			return nil, err
		}
		if err := rejectHighCodeMetadataBody(body, action); err != nil {
			return nil, err
		}
		return body, nil
	}
	if len(args) < 2 {
		return nil, fmt.Errorf("cloudcc %s %s <domain> <encodedSpecJson> [operation]", action, resource)
	}
	if err := rejectHighCodeMetadataDomain(args[0], action); err != nil {
		return nil, err
	}
	spec, err := parseObject(args[1], "cloudcc "+action+" "+resource)
	if err != nil {
		return nil, err
	}
	operation := "upsert"
	if len(args) > 2 && strings.TrimSpace(args[2]) != "" {
		operation = args[2]
	}
	return map[string]any{
		"domain":    normalizeDomain(args[0]),
		"operation": operation,
		"mode":      operation,
		"spec":      spec,
		"context":   defaultContext(),
	}, nil
}

func mutationRequest(resource string, args []string) (map[string]any, error) {
	if len(args) == 1 {
		body, err := parseObject(args[0], "cloudcc mutate "+resource)
		if err != nil {
			return nil, err
		}
		if err := rejectHighCodeMetadataBody(body, "mutate"); err != nil {
			return nil, err
		}
		return body, nil
	}
	if len(args) < 3 {
		return nil, fmt.Errorf("cloudcc mutate %s <domain> <mutation> <encodedPatchOrSpecJson>", resource)
	}
	if err := rejectHighCodeMetadataDomain(args[0], "mutate"); err != nil {
		return nil, err
	}
	patch, err := parseObject(args[2], "cloudcc mutate "+resource)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"domain":   normalizeDomain(args[0]),
		"mutation": args[1],
		"patch":    patch,
		"context":  defaultContext(),
	}, nil
}

func draftCreateRequest(args []string) (map[string]any, error) {
	if len(args) == 1 {
		body, err := parseObject(args[0], "cloudcc draft-create msapi")
		if err != nil {
			return nil, err
		}
		if err := rejectHighCodeMetadataBody(body, "draft-create"); err != nil {
			return nil, err
		}
		return body, nil
	}
	if len(args) < 3 {
		return nil, fmt.Errorf("cloudcc draft-create msapi <domain> <operation> <encodedDraftJson>")
	}
	if err := rejectHighCodeMetadataDomain(args[0], "draft-create"); err != nil {
		return nil, err
	}
	draft, err := parseObject(args[2], "cloudcc draft-create msapi")
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"domain":    normalizeDomain(args[0]),
		"operation": args[1],
		"draft":     draft,
		"context":   defaultContext(),
	}, nil
}

func (c *client) scan(stdout io.Writer, args []string) error {
	if len(args) == 0 || strings.EqualFold(strings.TrimSpace(args[0]), "summary") {
		return c.getJSON(stdout, "/metadata/v1/scans/summary")
	}
	mode := strings.TrimSpace(args[0])
	if strings.HasPrefix(mode, "@") || strings.HasPrefix(mode, "{") || strings.Contains(mode, "%7B") || strings.Contains(mode, "%7b") {
		body, err := parseObject(mode, "cloudcc scan msapi")
		if err != nil {
			return err
		}
		return c.writeJSON(stdout, http.MethodPost, "/metadata/v1/scans:compare", body)
	}
	if strings.EqualFold(mode, "compare") && len(args) > 1 {
		body, err := parseObject(args[1], "cloudcc scan msapi compare")
		if err != nil {
			return err
		}
		return c.writeJSON(stdout, http.MethodPost, "/metadata/v1/scans:compare", body)
	}
	switch strings.ToLower(mode) {
	case "standard-catalog", "standard-capability-catalog", "standard-objects", "catalog":
		return c.getJSON(stdout, "/metadata/v1/scans/standard-catalog")
	case "highcode", "local", "project-local":
		result, err := scanHighCodeProject(c.projectPath)
		if err != nil {
			return err
		}
		b, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, string(b))
		return nil
	case "setup-svc-live-replay-readiness", "setup-svc-parity-replay-readiness", "live-setup-svc-replay-readiness", "parity-live-replay-readiness":
		result := setupSvcLiveReplayReadinessResult(c.projectPath, c.baseURL)
		b, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, string(b))
		return nil
	case "setup-svc-live-replay-environment", "setup-svc-live-replay-env", "setup-svc-parity-replay-environment", "parity-live-replay-environment":
		result := buildSetupSvcLiveReplayEnvironmentResult(c.projectPath, c.baseURL)
		b, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, string(b))
		return nil
	case "setup-svc-live-replay-packet", "setup-svc-parity-replay-packet", "live-setup-svc-replay-packet", "parity-live-replay-packet":
		result := buildSetupSvcLiveReplayPacket(c.projectPath)
		b, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, string(b))
		return nil
	case "setup-svc-live-replay-coverage", "setup-svc-live-replay-coverage-audit", "setup-svc-parity-replay-coverage", "parity-live-replay-coverage":
		result := buildSetupSvcLiveReplayCoverageResult(c.projectPath)
		b, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, string(b))
		return nil
	case "setup-svc-live-replay-preflight", "setup-svc-live-replay-preflight-audit", "setup-svc-parity-replay-preflight", "parity-live-replay-preflight":
		result := buildSetupSvcLiveReplayPreflightResult(c.projectPath, c.baseURL)
		b, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, string(b))
		return nil
	case "setup-svc-live-replay-evidence", "setup-svc-live-replay-verify", "setup-svc-parity-replay-evidence", "parity-live-replay-evidence":
		manifestArg := ""
		if len(args) > 1 {
			manifestArg = args[1]
		}
		result, err := buildSetupSvcLiveReplayEvidenceResult(c.projectPath, manifestArg)
		if err != nil {
			return err
		}
		b, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, string(b))
		return nil
	case "setup-svc-live-replay-gaps", "setup-svc-live-replay-gap", "setup-svc-live-replay-status", "setup-svc-parity-replay-gaps", "parity-live-replay-gaps":
		result, err := buildSetupSvcLiveReplayGapResult(c.projectPath, "", args[1:]...)
		if err != nil {
			return err
		}
		b, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, string(b))
		return nil
	case "setup-svc-live-replay-capture-plan", "setup-svc-live-replay-captures", "setup-svc-parity-replay-capture-plan", "parity-live-replay-capture-plan":
		result, err := buildSetupSvcLiveReplayCapturePlanResult(c.projectPath, "", args[1:]...)
		if err != nil {
			return err
		}
		b, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, string(b))
		return nil
	case "setup-svc-live-replay-worklist", "setup-svc-live-replay-evidence-worklist", "setup-svc-parity-replay-worklist", "parity-live-replay-worklist":
		result, err := buildSetupSvcLiveReplayWorklistResult(c.projectPath, "", args[1:]...)
		if err != nil {
			return err
		}
		b, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, string(b))
		return nil
	case "setup-svc-live-replay-source-checklist", "setup-svc-live-replay-capture-checklist", "setup-svc-live-replay-source-captures", "setup-svc-parity-replay-source-checklist", "parity-live-replay-source-checklist":
		result, err := buildSetupSvcLiveReplaySourceChecklistResult(c.projectPath, "", args[1:]...)
		if err != nil {
			return err
		}
		b, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, string(b))
		return nil
	case "setup-svc-live-replay-source-health", "setup-svc-live-replay-capture-health", "setup-svc-parity-replay-source-health", "parity-live-replay-source-health":
		result, err := buildSetupSvcLiveReplaySourceHealthResult(c.projectPath, "", args[1:]...)
		if err != nil {
			return err
		}
		b, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, string(b))
		return nil
	case "setup-svc-live-replay-source-validate", "setup-svc-live-replay-capture-validate", "setup-svc-parity-replay-source-validate", "parity-live-replay-source-validate":
		result, err := buildSetupSvcLiveReplaySourceValidateResult(c.projectPath, "", args[1:]...)
		if err != nil {
			return err
		}
		b, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, string(b))
		return nil
	case "setup-svc-live-replay-source-execution-packet", "setup-svc-live-replay-source-capture-execution-packet", "setup-svc-live-replay-execution-packet", "setup-svc-parity-replay-source-execution-packet", "parity-live-replay-source-execution-packet":
		result, err := buildSetupSvcLiveReplaySourceExecutionPacketResult(c.projectPath, "", args[1:]...)
		if err != nil {
			return err
		}
		b, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, string(b))
		return nil
	case "setup-svc-live-replay-query-readback-capture-plan", "setup-svc-live-replay-query-capture-plan", "setup-svc-parity-replay-query-readback-capture-plan", "parity-live-replay-query-readback-capture-plan":
		result, err := buildSetupSvcLiveReplayQueryReadbackCapturePlanResult(c.projectPath, "", args[1:]...)
		if err != nil {
			return err
		}
		b, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, string(b))
		return nil
	case "setup-svc-live-replay-evidence-bundle", "setup-svc-live-replay-bundle", "setup-svc-live-replay-checksum", "setup-svc-parity-replay-evidence-bundle", "parity-live-replay-evidence-bundle":
		return c.scanSetupSvcLiveReplayEvidenceBundle(stdout, args[1:])
	case "setup-svc-live-replay-promotion", "setup-svc-live-replay-audit", "setup-svc-live-replay-matrix", "setup-svc-parity-replay-promotion", "parity-live-replay-promotion":
		manifestArg := ""
		if len(args) > 1 {
			manifestArg = args[1]
		}
		result, err := buildSetupSvcLiveReplayPromotionResult(c.projectPath, manifestArg)
		if err != nil {
			return err
		}
		b, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, string(b))
		return nil
	case "setup-svc-live-replay-completion-audit", "setup-svc-live-replay-completion", "setup-svc-live-replay-final-audit", "setup-svc-parity-replay-completion-audit", "parity-live-replay-completion-audit":
		manifestArg := ""
		if len(args) > 1 {
			manifestArg = args[1]
		}
		result := buildSetupSvcLiveReplayCompletionAuditResult(c.projectPath, manifestArg)
		b, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, string(b))
		return nil
	case "online-highcode", "remote-highcode", "deployments":
		result, err := scanOnlineHighCodeProject(c.projectPath)
		if err != nil {
			return err
		}
		b, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, string(b))
		return nil
	case "field-api-map", "field-map":
		body, err := projectFieldMapRequest(c.projectPath)
		if err != nil {
			return err
		}
		return c.writeJSON(stdout, http.MethodPost, "/metadata/v1/scans/field-map", body)
	case "project":
		body, err := projectScanRequest(c.projectPath)
		if err != nil {
			return err
		}
		return c.writeJSON(stdout, http.MethodPost, "/metadata/v1/scans:compare", body)
	default:
		return fmt.Errorf("cloudcc scan msapi [summary|standard-catalog|project|field-map|highcode|online-highcode|setup-svc-live-replay-readiness|setup-svc-live-replay-environment|setup-svc-live-replay-packet|setup-svc-live-replay-coverage|setup-svc-live-replay-preflight|setup-svc-live-replay-evidence|setup-svc-live-replay-gaps|setup-svc-live-replay-capture-plan|setup-svc-live-replay-worklist|setup-svc-live-replay-source-checklist|setup-svc-live-replay-source-health|setup-svc-live-replay-source-validate|setup-svc-live-replay-source-execution-packet|setup-svc-live-replay-query-readback-capture-plan|setup-svc-live-replay-evidence-bundle|setup-svc-live-replay-promotion|setup-svc-live-replay-completion-audit|local|@compareRequest.json]")
	}
}

func projectFieldMapRequest(projectPath string) (map[string]any, error) {
	configDir := filepath.Join(projectPath, "config")
	planPath := filepath.Join(configDir, "cloudcc-metadata-plan.json")
	metadataPlan, err := readJSONFile(planPath)
	planAvailable := true
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		metadataPlan = map[string]any{}
		planAvailable = false
	}
	objects := []map[string]any{}
	for _, object := range mapList(metadataPlan["objects"]) {
		key := firstMapString(object, "key")
		label := firstMapString(object, "label", "name")
		if key == "" && label == "" {
			continue
		}
		item := map[string]any{
			"key":              key,
			"phase":            firstMapString(object, "phase"),
			"action":           firstMapString(object, "action"),
			"label":            label,
			"apiName":          firstMapString(object, "apiName"),
			"generatedApiName": firstMapString(object, "generatedApiName"),
			"existingApiName":  firstMapString(object, "existingApiName"),
			"prefix":           firstMapString(object, "prefix", "keyPrefix"),
			"fields":           expandedFieldSpecs(object, metadataPlan),
		}
		objects = append(objects, item)
	}
	if len(objects) == 0 && planAvailable {
		return nil, fmt.Errorf("field-map scan requires config/cloudcc-metadata-plan.json objects")
	}
	return map[string]any{
		"source":         "field-map:" + filepath.Base(projectPath),
		"objects":        objects,
		"planAvailable":  planAvailable,
		"planSourcePath": filepath.ToSlash(planPath),
	}, nil
}

func projectScanRequest(projectPath string) (map[string]any, error) {
	configDir := filepath.Join(projectPath, "config")
	metadataPlan, err := readJSONFile(filepath.Join(configDir, "cloudcc-metadata-plan.json"))
	if err != nil {
		return nil, err
	}
	businessPlan, err := readJSONFile(filepath.Join(configDir, "cloudcc-business-config-plan.json"))
	if err != nil {
		return nil, err
	}
	navigationPlan, err := readJSONFile(filepath.Join(configDir, "cloudcc-navigation-plan.json"))
	if err != nil {
		return nil, err
	}
	fieldAPIMap, _ := readJSONFile(filepath.Join(configDir, "cloudcc-field-api-map.json"))

	objectLabels := []string{}
	objectAPINames := []string{}
	fieldLabels := []string{}
	objectLabelsByKey := map[string]string{}
	objectAPINamesByKey := map[string]string{}
	for _, object := range mapList(metadataPlan["objects"]) {
		key := firstMapString(object, "key")
		label := firstMapString(object, "label", "name")
		apiName := firstMapString(object, "apiName", "generatedApiName", "existingApiName")
		if key != "" && label != "" {
			objectLabelsByKey[key] = label
		}
		if key != "" && apiName != "" {
			objectAPINamesByKey[key] = apiName
		}
		objectLabels = appendIfPresent(objectLabels, label)
		objectAPINames = appendIfPresent(objectAPINames, apiName)
		fieldLabels = append(fieldLabels, expandedFieldLabels(object, metadataPlan)...)
	}

	recordTypeNames := []string{}
	layoutNames := []string{}
	for _, group := range mapList(businessPlan["recordTypes"]) {
		objectKey := firstMapString(group, "objectKey")
		objectLabel := objectLabelsByKey[objectKey]
		for _, item := range mapList(group["items"]) {
			name := firstMapString(item, "name", "label")
			recordTypeNames = appendIfPresent(recordTypeNames, name)
			if objectLabel != "" && name != "" {
				layoutNames = append(layoutNames, objectLabel+"-"+name+"布局")
			}
		}
	}

	validationRuleNames := []string{}
	for _, rule := range mapList(businessPlan["validationRules"]) {
		validationRuleNames = appendIfPresent(validationRuleNames, firstMapString(rule, "name", "label"))
	}

	applicationLabels := []string{}
	applicationAPINames := []string{}
	if application, ok := navigationPlan["application"].(map[string]any); ok {
		applicationLabels = appendIfPresent(applicationLabels, firstMapString(application, "label", "name"))
		applicationAPINames = appendIfPresent(applicationAPINames, firstMapString(application, "apiName", "appName"))
	}

	menuLabels := []string{}
	menuObjectAPINames := []string{}
	for _, group := range mapList(navigationPlan["groups"]) {
		for _, objectKey := range stringList(group["objects"]) {
			menuLabels = appendIfPresent(menuLabels, objectLabelsByKey[objectKey])
			menuObjectAPINames = appendIfPresent(menuObjectAPINames, objectAPINamesByKey[objectKey])
		}
	}

	checks := []map[string]any{}
	checks = addInformationalScanCheck(checks, "project object labels", "objects", "tp_sys_object",
		[]string{"LABEL", "OBJLABEL", "NAME", "OBJNAME"}, objectLabels)
	checks = addScanCheck(checks, "project object API names", "objects", "tp_sys_object",
		[]string{"SCHEMETABLE_NAME", "OBJNAME", "APINAME", "NAME"}, objectAPINames)
	checks = addScanCheck(checks, "project planned field labels", "fields", "tp_sys_schemetable",
		[]string{"SCHEMEFIELD_NAME", "FIELDNAME", "LABEL", "NAME"}, fieldLabels)
	checks = addRecordScanCheck(checks, "project object-scoped field API names", "fields", "tp_sys_schemetable",
		[]map[string]any{
			{"key": "objectId", "matchColumns": []string{"SCHEMETABLE_ID", "OBJ_ID", "OBJECT_ID"}},
			{"key": "fieldApiName", "matchColumns": []string{"APINAME", "api_name"}},
		},
		objectScopedFieldAPIRecords(fieldAPIMap))
	checks = addScanCheck(checks, "project record type names", "record-types", "tp_sys_recordtype",
		[]string{"NAME", "LABEL", "RECORDTYPE_NAME", "RECORDTYPENAME", "APICODE", "APINAME"}, recordTypeNames)
	checks = addScanCheck(checks, "project page layout names", "layouts", "tp_sys_layout",
		[]string{"NAME", "LABEL", "LAYOUTNAME"}, layoutNames)
	checks = addScanCheck(checks, "project validation rule names", "validation-rules", "tp_sys_validaterule",
		[]string{"NAME", "LABEL", "RULE_NAME", "VALIDATE_NAME"}, validationRuleNames)
	checks = addScanCheck(checks, "project application labels", "applications", "tp_sys_app",
		[]string{"APP_LABEL", "APPLABEL", "LABEL", "NAME"}, applicationLabels)
	checks = addScanCheck(checks, "project application API names", "applications", "tp_sys_app",
		[]string{"APP_NAME", "APPNAME", "APINAME", "NAME"}, applicationAPINames)
	checks = addInformationalScanCheck(checks, "project menu labels", "applications", "tp_sys_tab",
		[]string{"TAB_LABEL", "TABLABEL", "LABEL", "NAME"}, menuLabels)
	checks = addScanCheck(checks, "project menu object API names", "applications", "tp_sys_tab",
		[]string{"TAB_NAME", "api_name", "NAME", "OBJ_ID"}, menuObjectAPINames)

	return map[string]any{
		"source":           "project:" + filepath.Base(projectPath),
		"maxMissingValues": 200,
		"checks":           checks,
	}, nil
}

func readJSONFile(path string) (map[string]any, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("%s: JSON file is invalid: %w", path, err)
	}
	return out, nil
}

func expandedFieldLabels(object map[string]any, metadataPlan map[string]any) []string {
	labels := []string{}
	fieldSets, _ := metadataPlan["fieldSets"].(map[string]any)
	for _, fieldSetKey := range stringList(object["fieldSets"]) {
		for _, field := range mapList(fieldSets[fieldSetKey]) {
			labels = appendIfPresent(labels, firstMapString(field, "label", "fieldname", "name"))
		}
	}
	for _, field := range mapList(object["fields"]) {
		labels = appendIfPresent(labels, firstMapString(field, "label", "fieldname", "name"))
	}
	return labels
}

func expandedFieldSpecs(object map[string]any, metadataPlan map[string]any) []map[string]any {
	fields := []map[string]any{}
	fieldSets, _ := metadataPlan["fieldSets"].(map[string]any)
	for _, fieldSetKey := range stringList(object["fieldSets"]) {
		for _, field := range mapList(fieldSets[fieldSetKey]) {
			fields = append(fields, fieldSpec(field, "fieldSet:"+fieldSetKey))
		}
	}
	for _, field := range mapList(object["fields"]) {
		fields = append(fields, fieldSpec(field, "object.fields"))
	}
	return fields
}

func fieldSpec(field map[string]any, source string) map[string]any {
	return map[string]any{
		"label":  firstMapString(field, "label", "fieldname", "name"),
		"type":   firstMapString(field, "type", "fieldType", "schemefieldType"),
		"source": source,
		"remark": firstMapString(field, "remark", "description"),
	}
}

func addScanCheck(checks []map[string]any, label string, domain string, table string, matchColumns []string, values []string) []map[string]any {
	values = nonEmptyStrings(values)
	if len(values) == 0 {
		return checks
	}
	return append(checks, map[string]any{
		"label":        label,
		"domain":       domain,
		"table":        table,
		"matchColumns": matchColumns,
		"values":       values,
	})
}

func addInformationalScanCheck(checks []map[string]any, label string, domain string, table string, matchColumns []string, values []string) []map[string]any {
	values = nonEmptyStrings(values)
	if len(values) == 0 {
		return checks
	}
	return append(checks, map[string]any{
		"label":         label,
		"domain":        domain,
		"table":         table,
		"matchColumns":  matchColumns,
		"values":        values,
		"missingStatus": "informational_gap",
	})
}

func addRecordScanCheck(checks []map[string]any, label string, domain string, table string, recordColumns []map[string]any, records []map[string]any) []map[string]any {
	if len(recordColumns) == 0 || len(records) == 0 {
		return checks
	}
	return append(checks, map[string]any{
		"label":         label,
		"domain":        domain,
		"table":         table,
		"recordColumns": recordColumns,
		"records":       records,
	})
}

func objectScopedFieldAPIRecords(fieldAPIMap map[string]any) []map[string]any {
	records := []map[string]any{}
	for _, object := range mapList(fieldAPIMap["objects"]) {
		objectID := firstMapString(object, "objectId", "id")
		if objectID == "" {
			continue
		}
		planned, _ := object["plannedApiByLabel"].(map[string]any)
		labels := make([]string, 0, len(planned))
		for label := range planned {
			labels = append(labels, label)
		}
		sort.Strings(labels)
		for _, label := range labels {
			field, _ := planned[label].(map[string]any)
			apiName := firstMapString(field, "apiName", "api_name")
			if apiName == "" {
				continue
			}
			records = append(records, map[string]any{
				"objectId":     objectID,
				"fieldApiName": apiName,
				"fieldLabel":   label,
			})
		}
	}
	return records
}

func mapList(value any) []map[string]any {
	switch items := value.(type) {
	case []map[string]any:
		return items
	case []any:
		out := make([]map[string]any, 0, len(items))
		for _, item := range items {
			if m, ok := item.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	default:
		return nil
	}
}

func stringList(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = appendIfPresent(out, stringValue(item))
	}
	return out
}

func firstMapString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := stringValue(values[key]); value != "" {
			return value
		}
	}
	return ""
}

func appendIfPresent(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" || value == "<nil>" {
		return values
	}
	return append(values, value)
}

func nonEmptyStrings(values []string) []string {
	out := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || value == "<nil>" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func readObjectArg(args []string, index int, usage string) (map[string]any, error) {
	if len(args) <= index || strings.TrimSpace(args[index]) == "" {
		return nil, fmt.Errorf(usage)
	}
	return parseObject(args[index], usage)
}

func parseObject(value string, label string) (map[string]any, error) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "@") {
		b, err := os.ReadFile(strings.TrimPrefix(value, "@"))
		if err != nil {
			return nil, err
		}
		var out map[string]any
		if err := json.Unmarshal(b, &out); err != nil {
			return nil, fmt.Errorf("%s: JSON file is invalid: %w", label, err)
		}
		return out, nil
	}
	if strings.HasPrefix(value, "{") {
		var out map[string]any
		if err := json.Unmarshal([]byte(value), &out); err != nil {
			return nil, fmt.Errorf("%s: JSON is invalid: %w", label, err)
		}
		return out, nil
	}
	return jsonx.ParseEncodedObject(value, label)
}

func (c *client) getJSON(stdout io.Writer, path string) error {
	return c.writeJSON(stdout, http.MethodGet, path, nil)
}

func (c *client) writeJSON(stdout io.Writer, method string, path string, body any) error {
	var payload []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		payload = b
	}
	resBody, statusCode, err := c.doJSON(method, path, payload)
	if err != nil {
		return err
	}
	if statusCode < 200 || statusCode >= 300 {
		if c.refreshTokenAfterInvalidToken(statusCode, resBody) {
			resBody, statusCode, err = c.doJSON(method, path, payload)
			if err != nil {
				return err
			}
		}
		if statusCode < 200 || statusCode >= 300 {
			c.clearCacheIfInvalidToken(statusCode, resBody)
			return fmt.Errorf("metadata service http %d: %s", statusCode, string(resBody))
		}
	}
	if len(strings.TrimSpace(string(resBody))) == 0 {
		fmt.Fprintln(stdout, "{}")
		return nil
	}
	var pretty bytes.Buffer
	if json.Indent(&pretty, resBody, "", "  ") == nil {
		fmt.Fprintln(stdout, pretty.String())
		return nil
	}
	fmt.Fprintln(stdout, string(resBody))
	return nil
}

func (c *client) requestJSONMap(method string, path string, body any) (map[string]any, error) {
	var payload []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		payload = b
	}
	resBody, statusCode, err := c.doJSON(method, path, payload)
	if err != nil {
		return nil, err
	}
	if statusCode < 200 || statusCode >= 300 {
		if c.refreshTokenAfterInvalidToken(statusCode, resBody) {
			resBody, statusCode, err = c.doJSON(method, path, payload)
			if err != nil {
				return nil, err
			}
		}
		if statusCode < 200 || statusCode >= 300 {
			c.clearCacheIfInvalidToken(statusCode, resBody)
			return nil, fmt.Errorf("metadata service http %d: %s", statusCode, string(resBody))
		}
	}
	if len(strings.TrimSpace(string(resBody))) == 0 {
		return map[string]any{}, nil
	}
	var out map[string]any
	if err := json.Unmarshal(resBody, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *client) doJSON(method string, path string, payload []byte) ([]byte, int, error) {
	if strings.TrimSpace(c.baseURL) == "" {
		baseURL, err := serviceURL(c.projectPath)
		if err != nil {
			return nil, 0, err
		}
		c.baseURL = strings.TrimRight(baseURL, "/")
	}
	var reader io.Reader
	if payload != nil {
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequest(method, c.baseURL+path, reader)
	if err != nil {
		return nil, 0, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	}
	if c.token != "" {
		req.Header.Set("accessToken", c.token)
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if c.cloudccUserToken != "" {
		req.Header.Set("X-CloudCC-User-AccessToken", c.cloudccUserToken)
	}
	res, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, 0, err
	}
	return resBody, res.StatusCode, nil
}

func (c *client) refreshTokenAfterInvalidToken(statusCode int, resBody []byte) bool {
	if statusCode != http.StatusUnauthorized || !isInvalidTokenResponse(resBody) {
		return false
	}
	if strings.TrimSpace(os.Getenv("CLOUDCC_METADATA_SERVICE_ACCESS_TOKEN")) != "" {
		return false
	}
	if err := config.ClearCacheEntry(c.projectPath); err != nil {
		return false
	}
	cfg, err := config.Load(c.projectPath)
	if err != nil {
		return false
	}
	token := trimBearer(firstString(
		config.String(cfg, "metadataServiceAccessToken"),
		config.String(cfg, "accessToken"),
		config.String(cfg, "token"),
	))
	if token == "" {
		return false
	}
	c.token = token
	return true
}

func (c *client) clearCacheIfInvalidToken(statusCode int, resBody []byte) {
	if statusCode != http.StatusUnauthorized || !isInvalidTokenResponse(resBody) {
		return
	}
	if strings.TrimSpace(os.Getenv("CLOUDCC_METADATA_SERVICE_ACCESS_TOKEN")) != "" {
		return
	}
	_ = config.ClearCacheEntry(c.projectPath)
}

func isInvalidTokenResponse(resBody []byte) bool {
	var body map[string]any
	if err := json.Unmarshal(resBody, &body); err == nil {
		if strings.EqualFold(stringValue(body["error"]), "invalid_token") {
			return true
		}
		if strings.Contains(strings.ToLower(stringValue(body["message"])), "invalid_token") {
			return true
		}
	}
	return strings.Contains(strings.ToLower(string(resBody)), "invalid_token")
}

func IsMetadataDomainAction(action string) bool {
	switch strings.TrimSpace(action) {
	case "normalize", "validate", "plan", "mutate", "draft-create":
		return true
	default:
		return false
	}
}

func IsMetadataDomain(value string) bool {
	return normalizeDomain(value) != strings.TrimSpace(value) || isCanonicalDomain(value)
}

func rejectHighCodeMetadataBody(body map[string]any, action string) error {
	if body == nil {
		return nil
	}
	if domain := stringValue(body["domain"]); domain != "" {
		return rejectHighCodeMetadataDomain(domain, action)
	}
	return nil
}

func rejectHighCodeMetadataDomain(domain string, action string) error {
	if !isHighCodeResourceDomain(domain) {
		return nil
	}
	return fmt.Errorf("high-code writes stay on existing CloudCC resource/API paths; %q is not a MetadataService domain for cloudcc %s msapi", strings.TrimSpace(domain), action)
}

func isHighCodeResourceDomain(value string) bool {
	key := strings.ToLower(strings.TrimSpace(value))
	key = strings.ReplaceAll(key, "-", "")
	key = strings.ReplaceAll(key, "_", "")
	key = strings.ReplaceAll(key, " ", "")
	switch key {
	case "class", "classes", "trigger", "triggers", "timer", "schedule", "schedulejob", "schedulejobs", "script", "clientscript",
		"html", "htmlcomponent", "staticresource", "staticresources", "pagecomponent", "pagecomponents",
		"custompage", "custompages", "sidecar":
		return true
	default:
		return false
	}
}

func isCanonicalDomain(value string) bool {
	switch strings.TrimSpace(value) {
	case "objects", "fields", "global-select-lists", "record-types", "layouts", "profiles", "permissions",
		"sharing-rules", "validation-rules", "applications", "menus", "buttons", "roles", "custom-settings", "dupe-catchers",
		"single-sign-ons", "identity-providers", "approval-processes", "reports", "dashboards", "object-views", "api-registrars",
		"fiscal-years", "areas":
		return true
	default:
		return false
	}
}

func normalizeDomain(value string) string {
	switch strings.TrimSpace(value) {
	case "object":
		return "objects"
	case "field":
		return "fields"
	case "recordType", "record-type", "record_type":
		return "record-types"
	case "layout", "pagelayout", "page-layout":
		return "layouts"
	case "profile":
		return "profiles"
	case "permission", "system-permission", "systempermission":
		return "permissions"
	case "sharingRule", "sharing-rule", "sharingrule", "sharing-rules", "sharingRules", "sharingrules":
		return "sharing-rules"
	case "approval", "approval-process":
		return "approval-processes"
	case "report":
		return "reports"
	case "dashboard":
		return "dashboards"
	case "view", "object-view":
		return "object-views"
	case "globalSelectList", "global-select-list", "global-select", "globalselectlist", "globalselect", "global-select-lists":
		return "global-select-lists"
	case "validationRule", "validation-rule", "validationrule", "validation-rules":
		return "validation-rules"
	case "application", "app":
		return "applications"
	case "menu", "menus", "tab", "tabs":
		return "menus"
	case "button", "button-link", "buttonLink", "buttonlink":
		return "buttons"
	case "role":
		return "roles"
	case "customSetting", "custom-setting", "customsetting", "custom-settings":
		return "custom-settings"
	case "dupeCatcher", "dupe-catcher", "dupecatcher", "duplication", "dupe-catchers":
		return "dupe-catchers"
	case "singleSignOn", "single-sign-on", "singleSignOns", "singlesignon", "sso", "single-sign-ons":
		return "single-sign-ons"
	case "identityProvider", "identity-provider", "identityprovider", "idp", "identity-providers":
		return "identity-providers"
	case "apiRegistrar", "apiRegister", "api-registrar", "api-register", "apiregistrar", "apiregister", "api-registrars":
		return "api-registrars"
	case "fiscalYear", "fiscal-year", "fiscalyear", "fiscalyears", "companyFiscalYear", "company-fiscal-year", "fiscal-years":
		return "fiscal-years"
	case "area", "areas", "region", "hierarchicalStructure", "hierarchical-structure":
		return "areas"
	default:
		return strings.TrimSpace(value)
	}
}

func defaultContext() map[string]any {
	return map[string]any{
		"source": "cli",
		"locale": "zh-CN",
	}
}

func stringValue(v any) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

func firstString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" && strings.TrimSpace(value) != "<nil>" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func stableName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var out strings.Builder
	lastUnderscore := false
	for _, r := range value {
		isAlphaNum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlphaNum {
			out.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore && out.Len() > 0 {
			out.WriteByte('_')
			lastUnderscore = true
		}
	}
	result := strings.Trim(out.String(), "_")
	if result == "" {
		return "rule"
	}
	return result
}

func boolValue(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "true", "1", "yes", "y":
			return true
		default:
			return false
		}
	case float64:
		return v != 0
	case int:
		return v != 0
	default:
		return false
	}
}
