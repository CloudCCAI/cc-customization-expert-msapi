package docs

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"path"
	"strings"
)

//go:embed assets
var embedded embed.FS

func Print(module string, kind string, out io.Writer) error {
	content, err := Read(module, kind)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, strings.TrimRight(content, "\r\n"))
	return err
}

func Read(module string, kind string) (string, error) {
	module = normalizeModule(module)
	requestedModule := module
	if module == "plugin" || module == "platform/plugin" {
		return "", fmt.Errorf("unsupported doc module: plugin; use platform/pagecomponent")
	}
	kind = normalizeKind(module, kind)
	if module == "" {
		return "", fmt.Errorf("doc requires module")
	}
	file := path.Join("assets", module, kind+".md")
	b, err := embedded.ReadFile(file)
	if err != nil {
		if _, statErr := fs.Stat(embedded, path.Join("assets", module)); statErr != nil {
			return "", fmt.Errorf("unsupported doc module: %s", module)
		}
		return "", fmt.Errorf("doc %s does not support %s", module, kind)
	}
	content := string(b)
	content = canonicalizeLayoutPaths(content)
	if requestedModule == "platform/pagecomponent" {
		prefix := "命名说明：CloudCC 自定义页面组件统一使用 pagecomponent 命令。\n\n"
		content = strings.ReplaceAll(content, "cloudcc create plugin", "cloudcc create pagecomponent")
		content = strings.ReplaceAll(content, "cloudcc publish plugin", "cloudcc publish pagecomponent")
		content = strings.ReplaceAll(content, "cloudcc get plugin", "cloudcc get pagecomponent")
		content = strings.ReplaceAll(content, "cloudcc detail plugin", "cloudcc detail pagecomponent")
		content = strings.ReplaceAll(content, "cloudcc pull plugin", "cloudcc pull pagecomponent")
		content = strings.ReplaceAll(content, "cloudcc delete plugin", "cloudcc delete pagecomponent")
		content = strings.ReplaceAll(content, "cloudcc doc plugin", "cloudcc doc platform/pagecomponent")
		content = strings.ReplaceAll(content, "pluginNameOrId", "pageComponentNameOrId")
		content = strings.ReplaceAll(content, "pluginName", "pageComponentName")
		content = strings.ReplaceAll(content, "pluginId", "pageComponentId")
		content = strings.ReplaceAll(content, "插件", "组件")
		return prefix + content, nil
	}
	return content, nil
}

func normalizeModule(module string) string {
	module = strings.TrimSpace(strings.ReplaceAll(module, "\\", "/"))
	if module == "report" {
		return "platform/report"
	}
	if module == "" {
		return ""
	}
	module = path.Clean(module)
	if module == "." || strings.HasPrefix(module, "/") || strings.HasPrefix(module, "../") || strings.Contains(module, "/../") {
		return ""
	}
	return module
}

func normalizeKind(module string, kind string) string {
	kind = strings.ToLower(strings.TrimSpace(kind))
	if kind == "" {
		if module == "platform/config" {
			return "devguide"
		}
		return "devguide"
	}
	return kind
}

func Modules() ([]string, error) {
	return modulesUnder("assets")
}

func modulesUnder(root string) ([]string, error) {
	entries, err := embedded.ReadDir(root)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		child := path.Join(root, entry.Name())
		files, err := embedded.ReadDir(child)
		if err != nil {
			return nil, err
		}
		hasMarkdown := false
		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".md") {
				hasMarkdown = true
				break
			}
		}
		if hasMarkdown {
			out = append(out, strings.TrimPrefix(child, "assets/"))
			continue
		}
		nested, err := modulesUnder(child)
		if err != nil {
			return nil, err
		}
		out = append(out, nested...)
	}
	return out, nil
}

func canonicalizeLayoutPaths(content string) string {
	replacements := [][2]string{
		{"plugins/", "frontend/pagecomponents/"},
		{"`plugins`", "`frontend/pagecomponents`"},
		{"`plugins/", "`frontend/pagecomponents/"},
		{"`classes/", "`backend/classes/"},
		{"`triggers/", "`backend/triggers/"},
		{"`schedule/", "`backend/schedule/"},
		{"<projectPath>/classes/", "<projectPath>/backend/classes/"},
		{"<projectPath>/triggers/", "<projectPath>/backend/triggers/"},
		{"<projectPath>/schedule/", "<projectPath>/backend/schedule/"},
	}
	for _, item := range replacements {
		content = strings.ReplaceAll(content, item[0], item[1])
	}
	return content
}
