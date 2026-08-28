package compatibility

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"cloudcc-customization-expert-go/internal/config"
	"cloudcc-customization-expert-go/internal/version"
)

const (
	CapabilityHighCodePublishValidationGate = "highcode.publishValidationGate"
	CapabilityAPIRegistrarRemoteRuntime     = "setupSvc.apiRegistrar.remoteRuntime"
)

type Requirement struct {
	Capability                    string   `json:"capability"`
	Label                         string   `json:"label"`
	IntroducedInSkillVersion      string   `json:"introducedInSkillVersion"`
	MinimumMetadataServiceVersion string   `json:"minimumMetadataServiceVersion,omitempty"`
	MinimumSetupSvcVersion        string   `json:"minimumSetupSvcVersion,omitempty"`
	Domains                       []string `json:"domains"`
	Operations                    []string `json:"operations"`
	Reason                        string   `json:"reason"`
}

type Check struct {
	Capability              string `json:"capability"`
	Label                   string `json:"label"`
	Status                  string `json:"status"`
	CurrentSkillVersion     string `json:"currentSkillVersion"`
	RequiredSkillVersion    string `json:"requiredSkillVersion"`
	CurrentMetadataService  string `json:"currentMetadataServiceVersion,omitempty"`
	RequiredMetadataService string `json:"requiredMetadataServiceVersion,omitempty"`
	CurrentSetupSvc         string `json:"currentSetupSvcVersion,omitempty"`
	RequiredSetupSvc        string `json:"requiredSetupSvcVersion,omitempty"`
	Message                 string `json:"message"`
	Reason                  string `json:"reason,omitempty"`
}

type Report struct {
	Status string  `json:"status"`
	Checks []Check `json:"checks"`
}

func Requirements() []Requirement {
	return []Requirement{
		{
			Capability:               CapabilityHighCodePublishValidationGate,
			Label:                    "High-code publish remote validation for classes, triggers, and timers",
			IntroducedInSkillVersion: "2.2.7-msapi",
			MinimumSetupSvcVersion:   "19.3.R20",
			Domains:                  []string{"classes", "triggers", "timer"},
			Operations:               []string{"publish", "validate"},
			Reason:                   "High-code publish depends on the CLI and target setup-svc validate endpoints /api/ccfag/validate, /api/trigger/validate, and /api/ccPeak/validate; setup-svc version checks are advisory because customer branches may carry the capability under non-R version labels.",
		},
		{
			Capability:                    CapabilityAPIRegistrarRemoteRuntime,
			Label:                         "API registrar runtime debug/logs and high-code remote invocation adjustments",
			IntroducedInSkillVersion:      "2.2.33-msapi",
			MinimumMetadataServiceVersion: "1.1.41",
			MinimumSetupSvcVersion:        "19.7.R8",
			Domains:                       []string{"api-registrars", "classes", "triggers", "timer"},
			Operations:                    []string{"debug", "logs", "logDetail", "remote-call"},
			Reason:                        "Requires MetadataService API registrar metadata contracts and reminds operators to confirm setup-svc runtime debug/log and remote invocation contract support.",
		},
	}
}

func CheckProject(projectPath string, capabilities ...string) Report {
	cfg, cfgErr := config.Load(projectPath)
	metadataVersion, metadataSource := resolveMetadataServiceVersion(projectPath, cfg)
	setupVersion, setupSource := resolveSetupSvcVersion(cfg)
	checks := make([]Check, 0, len(capabilities))
	for _, capability := range capabilities {
		req, ok := requirementByCapability(capability)
		if !ok {
			continue
		}
		check := Check{
			Capability:           req.Capability,
			Label:                req.Label,
			Status:               "passed",
			CurrentSkillVersion:  version.Current(),
			RequiredSkillVersion: req.IntroducedInSkillVersion,
			Reason:               req.Reason,
		}
		if req.MinimumMetadataServiceVersion != "" {
			check.CurrentMetadataService = metadataVersion
			check.RequiredMetadataService = req.MinimumMetadataServiceVersion
			if metadataVersion == "" {
				check.Status = "unknown"
				if cfgErr != nil {
					check.Message = "MetadataService version could not be checked because CloudCC CLI config is not ready: " + cfgErr.Error()
				} else {
					check.Message = "MetadataService version is not available; configure metadataService.version/metadataServiceVersion or expose /ccmonitor/sck."
				}
			} else if CompareMetadataServiceVersion(metadataVersion, req.MinimumMetadataServiceVersion) < 0 {
				check.Status = "blocked"
				check.Message = fmt.Sprintf("MetadataService version %s is lower than required %s.", metadataVersion, req.MinimumMetadataServiceVersion)
			} else {
				check.Message = fmt.Sprintf("MetadataService version %s from %s satisfies %s.", metadataVersion, metadataSource, req.MinimumMetadataServiceVersion)
			}
		}
		if req.MinimumSetupSvcVersion != "" {
			check.CurrentSetupSvc = setupVersion
			check.RequiredSetupSvc = req.MinimumSetupSvcVersion
			if setupVersion == "" {
				check.Status = mergeStatus(check.Status, "unknown")
				if cfgErr != nil {
					check.Message = "setup-svc version could not be checked because CloudCC CLI config is not ready: " + cfgErr.Error()
				} else {
					check.Message = "setup-svc version is not available; configure setupSvc.version/setupSvcVersion or expose /ccmonitor/sck."
				}
			} else if CompareSetupSvcVersion(setupVersion, req.MinimumSetupSvcVersion) < 0 {
				check.Status = mergeStatus(check.Status, "warning")
				check.Message = fmt.Sprintf("setup-svc version %s is lower than the recommended baseline %s; customer branches may differ, so confirm the target setup-svc supports this capability or switch environments.", setupVersion, req.MinimumSetupSvcVersion)
			} else if check.Message == "" {
				check.Message = fmt.Sprintf("setup-svc version %s from %s satisfies %s.", setupVersion, setupSource, req.MinimumSetupSvcVersion)
			}
		}
		checks = append(checks, check)
	}
	return Report{Status: aggregateStatus(checks), Checks: checks}
}

func CheckAll(projectPath string) Report {
	capabilities := make([]string, 0, len(Requirements()))
	for _, req := range Requirements() {
		capabilities = append(capabilities, req.Capability)
	}
	return CheckProject(projectPath, capabilities...)
}

func RequireProject(projectPath string, capability string) error {
	report := CheckProject(projectPath, capability)
	for _, check := range report.Checks {
		metadataBlocked := check.RequiredMetadataService != "" &&
			(check.CurrentMetadataService == "" || CompareMetadataServiceVersion(check.CurrentMetadataService, check.RequiredMetadataService) < 0)
		if !metadataBlocked {
			continue
		}
		return fmt.Errorf("blocked_incompatible_target_version: capability %s requires MetadataService %s+; current MetadataService=%s. setup-svc recommended baseline is %s+ and current setup-svc is %s; customer branches may differ, so setup-svc is advisory only. %s",
			check.Capability,
			valueOrDash(check.RequiredMetadataService),
			valueOrDash(check.CurrentMetadataService),
			valueOrDash(check.RequiredSetupSvc),
			valueOrDash(check.CurrentSetupSvc),
			check.Message)
	}
	return nil
}

func CompareMetadataServiceVersion(actual string, minimum string) int {
	return compareNumericVersion(extractMetadataServiceVersion(actual), extractMetadataServiceVersion(minimum))
}

func CompareSetupSvcVersion(actual string, minimum string) int {
	return compareNumericVersion(parseSetupSvcVersion(actual), parseSetupSvcVersion(minimum))
}

func requirementByCapability(capability string) (Requirement, bool) {
	for _, req := range Requirements() {
		if req.Capability == capability {
			return req, true
		}
	}
	return Requirement{}, false
}

func resolveMetadataServiceVersion(projectPath string, cfg config.Config) (string, string) {
	if value := strings.TrimSpace(os.Getenv("CLOUDCC_METADATA_SERVICE_VERSION")); value != "" {
		return value, "CLOUDCC_METADATA_SERVICE_VERSION"
	}
	if cfg != nil {
		if value := firstString(cfg["metadataService.version"], cfg["metadataServiceVersion"], cfg["metadata_service_version"]); value != "" {
			return value, "project config"
		}
		if metadataService, _ := cfg["metadataService"].(map[string]any); metadataService != nil {
			if value := firstString(metadataService["version"]); value != "" {
				return value, "project config"
			}
		}
	}
	base, err := configuredMetadataServiceURL(projectPath, cfg)
	if err != nil || base == "" {
		return "", ""
	}
	if value := fetchVersion(strings.TrimRight(base, "/") + "/ccmonitor/sck"); value != "" {
		return value, "/ccmonitor/sck"
	}
	if value := fetchVersion(strings.TrimRight(base, "/") + "/metadata/v1/capabilities"); value != "" {
		return value, "/metadata/v1/capabilities"
	}
	return "", ""
}

func resolveSetupSvcVersion(cfg config.Config) (string, string) {
	if value := strings.TrimSpace(os.Getenv("CLOUDCC_SETUP_SVC_VERSION")); value != "" {
		return value, "CLOUDCC_SETUP_SVC_VERSION"
	}
	if cfg == nil {
		return "", ""
	}
	if value := firstString(cfg["setupSvc.version"], cfg["setupSvcVersion"], cfg["setup_svc_version"]); value != "" {
		return value, "project config"
	}
	if setupSvc, _ := cfg["setupSvc"].(map[string]any); setupSvc != nil {
		if value := firstString(setupSvc["version"]); value != "" {
			return value, "project config"
		}
	}
	base := config.String(cfg, "setupSvc")
	if strings.TrimSpace(base) == "" {
		return "", ""
	}
	if value := fetchVersion(strings.TrimRight(base, "/") + "/ccmonitor/sck"); value != "" {
		return value, "/ccmonitor/sck"
	}
	return "", ""
}

func configuredMetadataServiceURL(projectPath string, cfg config.Config) (string, error) {
	if value := strings.TrimSpace(os.Getenv("CLOUDCC_METADATA_SERVICE_URL")); value != "" {
		return value, nil
	}
	if cfg != nil {
		if value := firstString(cfg["metadataService.url"], cfg["metadataServiceUrl"], cfg["metadata_service_url"]); value != "" {
			return value, nil
		}
		if metadataService, _ := cfg["metadataService"].(map[string]any); metadataService != nil {
			if value := firstString(metadataService["url"]); value != "" {
				return value, nil
			}
		}
	}
	root, err := config.Root(projectPath)
	if err != nil {
		return "", err
	}
	use, _ := root["use"].(string)
	active, _ := root[use].(map[string]any)
	if active == nil {
		return "", nil
	}
	if value := firstString(active["metadataService.url"], active["metadataServiceUrl"], active["metadata_service_url"]); value != "" {
		return value, nil
	}
	if metadataService, _ := active["metadataService"].(map[string]any); metadataService != nil {
		if value := firstString(metadataService["url"]); value != "" {
			return value, nil
		}
	}
	return "", nil
}

func fetchVersion(url string) string {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return ""
	}
	resp, err := (&http.Client{Timeout: 3 * time.Second}).Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return ""
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return ""
	}
	return versionFromMap(body)
}

func versionFromMap(body map[string]any) string {
	for _, key := range []string{"codeVersion", "serviceVersion", "setupSvcVersion", "metadataServiceVersion", "version", "buildVersion"} {
		if value := firstString(body[key]); value != "" {
			return value
		}
	}
	if data, _ := body["data"].(map[string]any); data != nil {
		return versionFromMap(data)
	}
	return ""
}

var numericVersionRE = regexp.MustCompile(`\d+`)

func extractMetadataServiceVersion(value string) []int {
	return parseInts(value)
}

func parseSetupSvcVersion(value string) []int {
	return parseInts(value)
}

func parseInts(value string) []int {
	matches := numericVersionRE.FindAllString(value, -1)
	out := make([]int, 0, len(matches))
	for _, match := range matches {
		n, _ := strconv.Atoi(match)
		out = append(out, n)
	}
	return out
}

func compareNumericVersion(actual []int, minimum []int) int {
	max := len(actual)
	if len(minimum) > max {
		max = len(minimum)
	}
	for i := 0; i < max; i++ {
		a, m := 0, 0
		if i < len(actual) {
			a = actual[i]
		}
		if i < len(minimum) {
			m = minimum[i]
		}
		if a > m {
			return 1
		}
		if a < m {
			return -1
		}
	}
	return 0
}

func aggregateStatus(checks []Check) string {
	status := "passed"
	for _, check := range checks {
		status = mergeStatus(status, check.Status)
	}
	return status
}

func mergeStatus(current string, next string) string {
	if current == "blocked" || next == "blocked" {
		return "blocked"
	}
	if current == "warning" || next == "warning" {
		return "warning"
	}
	if current == "unknown" || next == "unknown" {
		return "unknown"
	}
	return "passed"
}

func firstString(values ...any) string {
	for _, value := range values {
		if text := strings.TrimSpace(fmt.Sprint(value)); text != "" && text != "<nil>" {
			return text
		}
	}
	return ""
}

func valueOrDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}
