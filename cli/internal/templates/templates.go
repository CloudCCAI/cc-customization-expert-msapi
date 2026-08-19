package templates

import (
	"embed"
	"fmt"
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
	target, err := filepath.Abs(target)
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
		"{{PROJECT_NAME}}": sanitizePackageName(projectName),
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
	return nil
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
