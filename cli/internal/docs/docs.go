package docs

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"path"
	"strings"
	"text/tabwriter"

	"cloudcc-customization-expert-go/internal/domaincatalog"
)

//go:embed assets
var embedded embed.FS

// Handle serves offline discovery and document reads without loading project configuration.
func Handle(args []string, out io.Writer) error {
	if len(args) == 0 || (len(args) == 1 && args[0] == "list") {
		return PrintIndex(out)
	}
	if len(args) > 2 || args[0] == "list" {
		return fmt.Errorf("usage: cloudcc doc [list | <module> [introduction|devguide]]")
	}
	kind := ""
	if len(args) == 2 {
		kind = args[1]
	}
	return Print(args[0], kind, out)
}

func PrintIndex(out io.Writer) error {
	modules, err := Modules()
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "MODULE\tPATH\tINTRODUCTION\tDEVGUIDE")
	for _, module := range modules {
		fmt.Fprintf(w, "%s\t%s\tcloudcc doc %s introduction\tcloudcc doc %s devguide\n",
			path.Base(module), module, path.Base(module), path.Base(module))
	}
	return w.Flush()
}

func Print(module string, kind string, out io.Writer) error {
	content, err := Read(module, kind)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, strings.TrimRight(content, "\r\n"))
	return err
}

func Read(module string, kind string) (string, error) {
	var err error
	module, err = normalizeModule(module)
	if err != nil {
		return "", err
	}
	requestedModule := module
	if module == "plugin" || module == "platform/plugin" {
		module = "platform/pagecomponent"
	}
	kind = normalizeKind(module, kind)
	if module == "" {
		return "", fmt.Errorf("doc requires module")
	}
	if strings.ContainsAny(kind, "/\\") || kind == "." || kind == ".." {
		return "", fmt.Errorf("unsupported doc type %q; use introduction or devguide", kind)
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
	if requestedModule == "platform/pagecomponent" || requestedModule == "plugin" || requestedModule == "platform/plugin" {
		prefix := "命名说明：CloudCC 自定义页面组件推荐使用 pagecomponent 命令；plugin/plugins 作为历史兼容别名。\n\n"
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

func normalizeModule(module string) (string, error) {
	module = strings.TrimSpace(strings.ReplaceAll(module, "\\", "/"))
	if module == "" || path.Clean(module) == "." || strings.HasPrefix(module, "/") ||
		strings.HasPrefix(module, "../") || strings.Contains(module, "/../") || strings.HasSuffix(module, "/..") {
		return "", fmt.Errorf("invalid doc module %q; use cloudcc doc list", module)
	}
	module = path.Clean(module)
	// Resource names and aliases share the catalog's documentation destinations.
	if !strings.Contains(module, "/") {
		if resource, ok := domaincatalog.FindResource(module); ok {
			if command := strings.Fields(resource.Documentation.Introduction); len(command) >= 3 {
				module = command[2]
			}
		} else if domain, ok := domaincatalog.Find(module); ok {
			if command := strings.Fields(domain.DocumentationCommand); len(command) >= 3 {
				module = command[2]
			}
		}
	}
	if strings.EqualFold(module, "fieldDependency") || strings.EqualFold(module, "field-dependency") ||
		strings.EqualFold(module, "field-dependencies") {
		return "platform/fieldDependency", nil
	}
	if strings.EqualFold(module, "lookupFilter") || strings.EqualFold(module, "lookup-filter") ||
		strings.EqualFold(module, "lookup-filters") {
		return "platform/lookupFilter", nil
	}
	if strings.EqualFold(module, "fieldTracking") || strings.EqualFold(module, "fieldTrack") ||
		strings.EqualFold(module, "field-tracking") {
		return "platform/fieldTracking", nil
	}
	if strings.EqualFold(module, "ccchat") || strings.EqualFold(module, "ccChat") ||
		strings.EqualFold(module, "ccchatSetting") || strings.EqualFold(module, "ccchatFeedTracking") ||
		strings.EqualFold(module, "feedTracking") {
		return "platform/ccchat", nil
	}
	if strings.EqualFold(module, "detailPageReport") || strings.EqualFold(module, "detailPageChart") ||
		strings.EqualFold(module, "detail-page-report") || strings.EqualFold(module, "detail-page-reports") {
		return "platform/detailPageReport", nil
	}
	if module == "report" {
		return "platform/report", nil
	}
	if module == "dashboard" {
		return "platform/dashboard", nil
	}
	if strings.EqualFold(module, "dataIndex") {
		return "platform/dataIndex", nil
	}
	if strings.EqualFold(module, "dataBulk") {
		return "platform/dataBulk", nil
	}
	if strings.EqualFold(module, "domains") {
		return "platform/domains", nil
	}
	if module == "platform/plugin" {
		return "platform/pagecomponent", nil
	}
	modules, err := Modules()
	if err != nil {
		return "", err
	}
	matched := ""
	for _, candidate := range modules {
		if strings.EqualFold(module, candidate) || (!strings.Contains(module, "/") && strings.EqualFold(module, path.Base(candidate))) {
			if matched != "" {
				return "", fmt.Errorf("ambiguous doc module %q; use a full path from cloudcc doc list", module)
			}
			matched = candidate
		}
	}
	if matched == "" {
		return "", fmt.Errorf("unsupported doc module: %s; use cloudcc doc list", module)
	}
	return matched, nil
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
