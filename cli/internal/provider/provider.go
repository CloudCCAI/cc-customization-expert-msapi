// Package provider resolves the low-code execution provider without changing
// the stable CloudCC CLI command surface.
package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloudcc-customization-expert-go/internal/config"
	"cloudcc-customization-expert-go/internal/edition"
)

const (
	ModeAuto  = "auto"
	ModeMSAPI = "msapi"
	ModeUIAPI = "uiapi"
)

type Selection struct {
	PackageName   string `json:"packageName"`
	RequestedMode string `json:"requestedMode"`
	SelectedMode  string `json:"selectedMode"`
	StrictMode    string `json:"strictMode,omitempty"`
	Reason        string `json:"reason"`
	Endpoint      string `json:"metadataServiceEndpoint,omitempty"`
	SafetyLevel   string `json:"safetyLevel"`
}

// ResolveForArgs uses the legacy shortcut convention: the first non-JSON
// value is the project path, even when it has not been created yet.
func ResolveForArgs(args []string, cwd string) (Selection, error) {
	projectPath := cwd
	if len(args) > 0 {
		first := strings.TrimSpace(args[0])
		if first != "" && !looksLikeJSONArg(first) {
			projectPath = first
		}
	}
	return Resolve(projectPath)
}

func Resolve(projectPath string) (Selection, error) {
	requested, endpoint, configured, err := requestedModeAndEndpoint(projectPath)
	if err != nil {
		return Selection{}, err
	}
	selection := Selection{
		PackageName:   edition.PackageName,
		RequestedMode: requested,
		StrictMode:    edition.StrictExecutionMode,
		Endpoint:      endpoint,
	}

	if strict := strings.TrimSpace(edition.StrictExecutionMode); strict != "" {
		if requested != ModeAuto && requested != strict {
			return Selection{}, fmt.Errorf("%s is a strict %s distribution; executionMode=%s is not allowed", edition.PackageName, strict, requested)
		}
		selection.SelectedMode = strict
		selection.Reason = "strict_distribution"
		selection.SafetyLevel = safetyLevel(strict)
		if strict == ModeMSAPI && !configured {
			return Selection{}, missingEndpointError(projectPath)
		}
		return selection, nil
	}

	switch requested {
	case ModeUIAPI:
		selection.SelectedMode = ModeUIAPI
		selection.Reason = "explicit_execution_mode"
		selection.SafetyLevel = safetyLevel(ModeUIAPI)
		return selection, nil
	case ModeMSAPI:
		if !configured {
			return Selection{}, missingEndpointError(projectPath)
		}
		selection.SelectedMode = ModeMSAPI
		selection.Reason = "explicit_execution_mode"
		selection.SafetyLevel = safetyLevel(ModeMSAPI)
		return selection, nil
	case ModeAuto:
		if !configured {
			selection.SelectedMode = ModeUIAPI
			selection.Reason = "metadata_service_not_configured"
			selection.SafetyLevel = safetyLevel(ModeUIAPI)
			return selection, nil
		}
		if err := probeMetadataService(endpoint); err != nil {
			return Selection{}, fmt.Errorf("MetadataService is configured for auto mode but cannot be used; refusing silent UIAPI fallback: %w", err)
		}
		selection.SelectedMode = ModeMSAPI
		selection.Reason = "metadata_service_capability_probe_passed"
		selection.SafetyLevel = safetyLevel(ModeMSAPI)
		return selection, nil
	default:
		return Selection{}, fmt.Errorf("unsupported executionMode %q; use auto, msapi, or uiapi", requested)
	}
}

func WriteDoctor(projectPath string, stdout io.Writer) error {
	selection, err := Resolve(projectPath)
	if err != nil {
		return err
	}
	return json.NewEncoder(stdout).Encode(selection)
}

// RequireMSAPI protects explicit `cloudcc ... msapi` commands in a strict
// UIAPI package and prevents an auto/UIAPI Universal session from bypassing
// the selected provider through a raw MetadataService command.
func RequireMSAPI(projectPath string) error {
	selection, err := Resolve(projectPath)
	if err != nil {
		return err
	}
	if selection.SelectedMode != ModeMSAPI {
		return fmt.Errorf("MetadataService command is unavailable with selected provider %s; set executionMode=msapi only when a compatible MetadataService is configured", selection.SelectedMode)
	}
	return nil
}

func requestedModeAndEndpoint(projectPath string) (mode string, endpoint string, configured bool, err error) {
	mode = strings.ToLower(strings.TrimSpace(os.Getenv("CLOUDCC_EXECUTION_MODE")))
	if mode == "" {
		mode = strings.TrimSpace(edition.DefaultExecutionMode)
	}
	endpoint = strings.TrimSpace(os.Getenv("CLOUDCC_METADATA_SERVICE_URL"))
	if endpoint != "" {
		if err := validateEndpoint(endpoint); err != nil {
			return "", "", false, err
		}
		return mode, strings.TrimRight(endpoint, "/"), true, nil
	}
	root, rootErr := config.Root(projectPath)
	if rootErr != nil {
		if os.IsNotExist(rootErr) {
			return mode, "", false, nil
		}
		return mode, "", false, nil
	}
	use, _ := root["use"].(string)
	active, _ := root[use].(map[string]any)
	if active == nil {
		return mode, "", false, nil
	}
	if envMode := firstString(active["executionMode"], active["execution_mode"]); envMode != "" && strings.TrimSpace(os.Getenv("CLOUDCC_EXECUTION_MODE")) == "" {
		mode = strings.ToLower(envMode)
	}
	endpoint = firstString(active["metadataServiceUrl"], active["metadata_service_url"])
	if endpoint == "" {
		if metadataService, _ := active["metadataService"].(map[string]any); metadataService != nil {
			endpoint = firstString(metadataService["url"])
		}
	}
	if endpoint == "" {
		return mode, "", false, nil
	}
	if err := validateEndpoint(endpoint); err != nil {
		return "", "", false, err
	}
	return mode, strings.TrimRight(endpoint, "/"), true, nil
}

func probeMetadataService(endpoint string) error {
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(endpoint, "/")+"/metadata/v1/capabilities", nil)
	if err != nil {
		return err
	}
	response, err := (&http.Client{Timeout: 3 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("capability probe returned HTTP %d", response.StatusCode)
	}
	return nil
}

func missingEndpointError(projectPath string) error {
	return fmt.Errorf("MetadataService/MSAPI URL is required for executionMode=msapi; add metadataService.url to %s or set CLOUDCC_METADATA_SERVICE_URL", filepath.Join(projectPath, "cloudcc-cli.config.json"))
}

func validateEndpoint(value string) error {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("invalid MetadataService URL %q", value)
	}
	return nil
}

func firstString(values ...any) string {
	for _, value := range values {
		if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
			return strings.TrimSpace(text)
		}
	}
	return ""
}

func looksLikeJSONArg(value string) bool {
	value = strings.TrimSpace(value)
	return strings.HasPrefix(value, "{") || strings.HasPrefix(value, "[") || strings.HasPrefix(value, "@") || strings.Contains(value, "%7B") || strings.Contains(value, "%7b")
}

func safetyLevel(mode string) string {
	if mode == ModeMSAPI {
		return "native_plan_apply_rollback"
	}
	return "direct_uiapi_no_server_rollback"
}
