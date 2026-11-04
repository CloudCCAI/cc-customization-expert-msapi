// Package projectlayout defines offline project path ownership, never tenant configuration.
package projectlayout

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const RegistryPath = "docs/project-layout.json"
const SchemaVersion = "cloudcc-project-layout/v2"
const ClientScriptsPath = "frontend/scripts"

type Entry struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Role     string `json:"role"`
	Tracking string `json:"tracking"`
	Required bool   `json:"required,omitempty"`
}

type Registry struct {
	SchemaVersion      string  `json:"schemaVersion"`
	ReferenceMaterials string  `json:"referenceMaterials"`
	Entries            []Entry `json:"rootEntries"`
}

func DefaultRegistry() Registry {
	r := Registry{SchemaVersion: SchemaVersion, ReferenceMaterials: "references"}
	for _, v := range []struct{ name, role string }{
		{".claw", "project-state"}, {".ai-dev", "legacy-project-state"}, {"docs", "design-and-governance"},
		{"references", "source-materials"}, {"backend", "server-code"}, {"frontend", "frontend-code"},
		{"sidecar", "external-services"},
		{"html", "html-resources"}, {"staticResource", "static-resources"}, {"visualpage", "horizontal-pages"},
		{"customComponent", "horizontal-components"}, {"tools", "project-tools"},
		{"test-assets", "reusable-tests"}, {"evidence", "execution-evidence"}, {"outputs", "deliverables"},
		{".agents", "project-skills"},
	} {
		r.Entries = append(r.Entries, Entry{Name: v.name, Kind: "directory", Role: v.role, Tracking: "tracked"})
	}
	for _, name := range []string{"README.md", "AGENTS.md", ".gitignore", "cloudcc-cli.config.example.json"} {
		r.Entries = append(r.Entries, Entry{Name: name, Kind: "file", Role: "project-entry", Tracking: "tracked"})
	}
	for _, name := range []string{"cloudcc-cli.config.json", ".cloudcc-cache.json", ".DS_Store"} {
		r.Entries = append(r.Entries, Entry{Name: name, Kind: "file", Role: "local-configuration", Tracking: "local"})
	}
	for _, name := range []string{".claw-local", "captures"} {
		r.Entries = append(r.Entries, Entry{Name: name, Kind: "directory", Role: "local-work", Tracking: "local"})
	}
	return r
}

// SafePath rejects lexical escape and existing symlinks pointing outside the project,
// including a missing leaf under an existing symlinked parent.
func SafePath(project, relative string) (string, error) {
	if relative == "" || strings.ContainsAny(relative, "\\:\x00\r\n") || filepath.IsAbs(relative) || strings.HasPrefix(relative, "/") {
		return "", fmt.Errorf("unsafe project-relative path")
	}
	for _, p := range strings.Split(relative, "/") {
		if p == "" || p == "." || p == ".." {
			return "", fmt.Errorf("unsafe project-relative path")
		}
	}
	root, err := filepath.Abs(project)
	if err != nil {
		return "", err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	target := filepath.Join(root, filepath.FromSlash(relative))
	probe := target
	for {
		_, err = os.Lstat(probe)
		if err == nil {
			break
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		probe = filepath.Dir(probe)
	}
	resolved, err := filepath.EvalSymlinks(probe)
	if err != nil {
		return "", fmt.Errorf("unresolvable project path: %w", err)
	}
	rel, err := filepath.Rel(root, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("project path escapes through symlink")
	}
	return target, nil
}

func Load(project string) (Registry, bool, error) {
	r := DefaultRegistry()
	path, err := SafePath(project, RegistryPath)
	if err != nil {
		return r, false, err
	}
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return r, false, nil
	}
	if err != nil {
		return r, false, err
	}
	var loaded Registry
	d := json.NewDecoder(strings.NewReader(string(b)))
	d.DisallowUnknownFields()
	if err := d.Decode(&loaded); err != nil {
		return r, true, fmt.Errorf("invalid layout registry: %w", err)
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return r, true, fmt.Errorf("layout registry must contain one JSON object")
	}
	if loaded.SchemaVersion != SchemaVersion {
		return r, true, fmt.Errorf("unsupported project layout schema")
	}
	if _, err := SafePath(project, loaded.ReferenceMaterials); err != nil {
		return r, true, fmt.Errorf("invalid referenceMaterials: %w", err)
	}
	seen := map[string]bool{}
	for _, e := range loaded.Entries {
		if strings.Contains(e.Name, "/") || e.Name == ".git" {
			return r, true, fmt.Errorf("root entry must be one name; .git is reserved")
		}
		// Validate syntax without probing the entry: doctor reports individual symlink errors.
		if e.Name == "" || e.Name == "." || e.Name == ".." || strings.ContainsAny(e.Name, "\\:\x00\r\n") {
			return r, true, fmt.Errorf("invalid root entry name")
		}
		key := strings.ToLower(e.Name)
		if seen[key] {
			return r, true, fmt.Errorf("duplicate root entry: %s", e.Name)
		}
		seen[key] = true
		if (e.Kind != "file" && e.Kind != "directory") || strings.TrimSpace(e.Role) == "" || (e.Tracking != "local" && e.Tracking != "tracked") {
			return r, true, fmt.Errorf("invalid root entry kind, role, or tracking: %s", e.Name)
		}
	}
	refRoot := strings.Split(loaded.ReferenceMaterials, "/")[0]
	refAllowed := false
	for _, e := range loaded.Entries {
		if e.Name == refRoot && e.Kind == "directory" && e.Tracking == "tracked" {
			refAllowed = true
		}
	}
	if !refAllowed {
		return r, true, fmt.Errorf("referenceMaterials must belong to a registered shared directory")
	}
	return loaded, true, nil
}

// ScriptRoot returns the single canonical frontend source directory.
func ScriptRoot(project string) (string, error) {
	r, registered, err := Load(project)
	if err != nil {
		return "", err
	}
	if registered {
		allowed := false
		for _, e := range r.Entries {
			if e.Name == "frontend" && e.Kind == "directory" && e.Tracking == "tracked" {
				allowed = true
			}
		}
		if !allowed {
			return "", fmt.Errorf("frontend must be registered as a shared source directory")
		}
	}
	if err := checkScriptDirectory(project); err != nil {
		return "", err
	}
	return ClientScriptsPath, nil
}

// Checking a layout does not require optional capability directories to exist.
func checkScriptDirectory(project string) error {
	path, err := SafePath(project, ClientScriptsPath)
	if err != nil {
		return err
	}
	info, err := os.Stat(path)
	if err == nil && !info.IsDir() {
		return fmt.Errorf("%s must be a directory", ClientScriptsPath)
	}
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func ScriptPath(project, selector string) (string, error) {
	if len(strings.Split(selector, "/")) != 2 {
		return "", fmt.Errorf("script selector must be <object>/<name>")
	}
	root, err := ScriptRoot(project)
	if err != nil {
		return "", err
	}
	return SafePath(project, root+"/"+selector)
}

func Handle(action string, args []string, out io.Writer, cwd string) error {
	if len(args) > 1 {
		return fmt.Errorf("usage: cloudcc %s project-layout [projectPath]", action)
	}
	project := cwd
	if len(args) == 1 {
		project = args[0]
		if !filepath.IsAbs(project) {
			project = filepath.Join(cwd, project)
		}
	}
	if action == "doctor" {
		return WriteDoctor(project, out)
	}
	if action != "init" {
		return fmt.Errorf("unsupported project-layout action")
	}
	path, err := SafePath(project, RegistryPath)
	if err != nil {
		return err
	}
	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("layout registry already exists; review it without overwriting")
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	e := json.NewEncoder(f)
	e.SetIndent("", "  ")
	writeErr := e.Encode(DefaultRegistry())
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	return json.NewEncoder(out).Encode(map[string]any{"status": "created", "path": RegistryPath, "reviewRequired": true, "message": "Review rootEntries and project rules before adopting; existing directories were not auto-registered or moved."})
}
