package templates

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

//go:embed assets
var assets embed.FS

type projectFile struct {
	asset string
	path  string
	text  bool
}

const DefaultMetadataServiceURL = "https://dc52.apis.cloudcc.cn/metadata"

type ProjectOptions struct {
	PlatformMode       string
	ExecutionMode      string
	MainAppURL         string
	Username           string
	Language           string
	MetadataServiceURL string
}

var projectFiles = []projectFile{
	{asset: "assets/cloudcc-cli.config.json", path: "cloudcc-cli.config.json", text: true},
	{asset: "assets/gitignore", path: ".gitignore", text: true},
	{asset: "assets/frontend-readme.md", path: "frontend/README.md", text: true},
	{asset: "assets/lib/ccopenapi-0.1.3.jar", path: "backend/lib/ccopenapi-0.1.3.jar"},
	{asset: "assets/lib/fastjson-1.2.83.jar", path: "backend/lib/fastjson-1.2.83.jar"},
	{asset: "assets/lib/reflections-0.9.12.jar", path: "backend/lib/reflections-0.9.12.jar"},
}

var projectDirs = []string{
	"frontend/pagecomponents",
	"frontend/build",
	"backend/classes",
	"backend/triggers",
	"backend/schedule",
	"backend/lib",
	"sidecar",
}

func WriteProject(target string, projectName string) error {
	return WriteProjectWithOptions(target, projectName, ProjectOptions{PlatformMode: "lightning"})
}

func WriteProjectWithOptions(target string, projectName string, options ProjectOptions) error {
	options, err := normalizeProjectOptions(options)
	if err != nil {
		return err
	}
	target, err = filepath.Abs(target)
	if err != nil {
		return err
	}
	if err := ensureWritableProjectTarget(target); err != nil {
		return err
	}
	if err := os.MkdirAll(target, 0755); err != nil {
		return err
	}
	for _, dir := range projectDirs {
		if err := os.MkdirAll(filepath.Join(target, filepath.FromSlash(dir)), 0755); err != nil {
			return err
		}
	}
	replacements := map[string]string{
		"{{PROJECT_NAME}}":         sanitizePackageName(projectName),
		"{{METADATA_SERVICE_URL}}": DefaultMetadataServiceURL,
	}
	for _, file := range projectFiles {
		data, err := assets.ReadFile(file.asset)
		if err != nil {
			return fmt.Errorf("read embedded template %s: %w", file.asset, err)
		}
		if file.text {
			content := string(data)
			for old, newValue := range replacements {
				content = strings.ReplaceAll(content, old, newValue)
			}
			data = []byte(content)
		}
		output := filepath.Join(target, filepath.FromSlash(file.path))
		if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(output, data, 0644); err != nil {
			return err
		}
	}
	if options.PlatformMode == "horizontal" {
		root := map[string]any{
			"use": "dev",
			"dev": map[string]any{
				"platformMode":  "horizontal",
				"executionMode": options.ExecutionMode,
				"endpoints":     map[string]any{"mainAppUrl": options.MainAppURL},
				"auth": map[string]any{
					"username": options.Username,
					"password": "CLOUDCC_PASSWORD",
					"language": options.Language,
				},
				"metadataService": map[string]any{"url": options.MetadataServiceURL},
			},
		}
		data, err := json.MarshalIndent(root, "", "  ")
		if err != nil {
			return err
		}
		data = append(data, '\n')
		if err := os.WriteFile(filepath.Join(target, "cloudcc-cli.config.json"), data, 0644); err != nil {
			return err
		}
	}
	return nil
}

func normalizeProjectOptions(options ProjectOptions) (ProjectOptions, error) {
	options.PlatformMode = strings.ToLower(strings.TrimSpace(options.PlatformMode))
	if options.PlatformMode == "" {
		options.PlatformMode = "lightning"
	}
	if options.PlatformMode != "lightning" && options.PlatformMode != "horizontal" {
		return options, fmt.Errorf("unsupported platform %q; use lightning or horizontal", options.PlatformMode)
	}
	if options.PlatformMode == "lightning" {
		return options, nil
	}
	options.ExecutionMode = strings.ToLower(strings.TrimSpace(options.ExecutionMode))
	if options.ExecutionMode == "" {
		options.ExecutionMode = "auto"
	}
	if options.ExecutionMode != "auto" && options.ExecutionMode != "uiapi" && options.ExecutionMode != "msapi" {
		return options, fmt.Errorf("horizontal project execution mode must be auto, msapi, or uiapi")
	}
	if strings.TrimSpace(options.MainAppURL) == "" {
		options.MainAppURL = "https://tenant.example.com"
	}
	if strings.TrimSpace(options.Username) == "" {
		options.Username = "user@example.com"
	}
	if strings.TrimSpace(options.Language) == "" {
		options.Language = "zh"
	}
	if strings.TrimSpace(options.MetadataServiceURL) == "" {
		options.MetadataServiceURL = "https://metadata.example.com"
	}
	for label, value := range map[string]string{"main-app URL": options.MainAppURL, "MetadataService URL": options.MetadataServiceURL} {
		parsed, err := url.Parse(strings.TrimSpace(value))
		if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return options, fmt.Errorf("invalid horizontal %s %q", label, value)
		}
	}
	return options, nil
}

func ensureWritableProjectTarget(target string) error {
	entries, err := os.ReadDir(target)
	if err == nil {
		if len(entries) > 0 {
			return fmt.Errorf("target directory is not empty: %s", target)
		}
		return nil
	}
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func sanitizePackageName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = regexp.MustCompile(`[^a-z0-9._-]+`).ReplaceAllString(name, "-")
	name = strings.Trim(name, ".-_")
	if name == "" {
		return "cloudcc-project"
	}
	return name
}
