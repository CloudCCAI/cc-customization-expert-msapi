package governance

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var unstableStandardName = regexp.MustCompile(`(?i)(^|[-_])v?\d+\.\d+([._-]|$)|(^|[-_])20\d{6}([._-]|$)`)

var requiredStandardFields = []string{
	"kind",
	"standard_id",
	"title",
	"status",
	"version",
	"owner_role",
	"effective_date",
	"review_trigger",
}

var supportedStandardStatuses = map[string]bool{
	"draft":      true,
	"active":     true,
	"deprecated": true,
	"archived":   true,
}

type Issue struct {
	Code    string `json:"code"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
}

type Standard struct {
	ID       string `json:"standardId"`
	Path     string `json:"path"`
	Status   string `json:"status"`
	Version  string `json:"version"`
	Indexed  bool   `json:"indexed"`
	GateRead bool   `json:"agentsReadGate"`
}

type DeliveryRoot struct {
	ProjectCode  string     `json:"projectCode"`
	Path         string     `json:"path"`
	StandardsDir string     `json:"standardsDir,omitempty"`
	IndexPath    string     `json:"standardIndexPath,omitempty"`
	ProcessIndex string     `json:"processIndexPath,omitempty"`
	Standards    []Standard `json:"standards"`
}

type Report struct {
	Status        string         `json:"status"`
	ProjectPath   string         `json:"projectPath"`
	ReadOnly      bool           `json:"readOnly"`
	DeliveryRoots []DeliveryRoot `json:"deliveryRoots"`
	Errors        []Issue        `json:"errors"`
	Warnings      []Issue        `json:"warnings"`
	Summary       map[string]int `json:"summary"`
}

func WriteDoctor(projectPath string, stdout io.Writer) error {
	report, err := Validate(projectPath)
	if encodeErr := json.NewEncoder(stdout).Encode(report); encodeErr != nil {
		return encodeErr
	}
	if err != nil {
		return err
	}
	return nil
}

func Validate(projectPath string) (Report, error) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return Report{}, err
	}
	report := Report{
		Status:      "passed",
		ProjectPath: absPath,
		ReadOnly:    true,
		Summary: map[string]int{
			"deliveryRootCount": 0,
			"standardCount":     0,
			"activeCount":       0,
			"errorCount":        0,
			"warningCount":      0,
		},
	}
	info, statErr := os.Stat(absPath)
	if statErr != nil {
		return report, fmt.Errorf("project-governance doctor cannot read project path %s: %w", absPath, statErr)
	}
	if !info.IsDir() {
		return report, fmt.Errorf("project-governance doctor requires a directory: %s", absPath)
	}

	deliveryBase := filepath.Join(absPath, "docs", "delivery")
	entries, readErr := os.ReadDir(deliveryBase)
	if os.IsNotExist(readErr) {
		report.Status = "not_adopted"
		return report, nil
	}
	if readErr != nil {
		return report, fmt.Errorf("project-governance doctor cannot read %s: %w", deliveryBase, readErr)
	}

	agentsPath := filepath.Join(absPath, "AGENTS.md")
	agentsContent, agentsErr := os.ReadFile(agentsPath)
	agentsText := string(agentsContent)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		governanceDir := filepath.Join(deliveryBase, entry.Name(), "00-governance")
		if info, err := os.Stat(governanceDir); err != nil || !info.IsDir() {
			continue
		}
		root := validateDeliveryRoot(absPath, entry.Name(), governanceDir, agentsText, agentsErr)
		report.DeliveryRoots = append(report.DeliveryRoots, root.deliveryRoot)
		report.Errors = append(report.Errors, root.errors...)
		report.Warnings = append(report.Warnings, root.warnings...)
	}

	sort.Slice(report.DeliveryRoots, func(i, j int) bool {
		return report.DeliveryRoots[i].ProjectCode < report.DeliveryRoots[j].ProjectCode
	})
	if len(report.DeliveryRoots) == 0 {
		report.Status = "not_adopted"
		return report, nil
	}
	for _, root := range report.DeliveryRoots {
		report.Summary["standardCount"] += len(root.Standards)
		for _, standard := range root.Standards {
			if standard.Status == "active" {
				report.Summary["activeCount"]++
			}
		}
	}
	report.Summary["deliveryRootCount"] = len(report.DeliveryRoots)
	report.Summary["errorCount"] = len(report.Errors)
	report.Summary["warningCount"] = len(report.Warnings)
	if len(report.Errors) > 0 {
		report.Status = "failed"
		return report, fmt.Errorf("project-governance doctor found %d error(s)", len(report.Errors))
	}
	return report, nil
}

type deliveryValidation struct {
	deliveryRoot DeliveryRoot
	errors       []Issue
	warnings     []Issue
}

type standardCandidate struct {
	path string
	meta map[string]string
}

func validateDeliveryRoot(projectPath string, projectCode string, governanceDir string, agentsText string, agentsErr error) deliveryValidation {
	result := deliveryValidation{deliveryRoot: DeliveryRoot{
		ProjectCode: projectCode,
		Path:        rel(projectPath, filepath.Dir(governanceDir)),
	}}
	standardsDir := filepath.Join(governanceDir, "standards")
	if info, err := os.Stat(standardsDir); err != nil || !info.IsDir() {
		result.warnings = append(result.warnings, Issue{
			Code:    "standards_not_adopted",
			Path:    rel(projectPath, standardsDir),
			Message: "delivery root has no standards directory",
		})
		return result
	}
	result.deliveryRoot.StandardsDir = rel(projectPath, standardsDir)

	markdownFiles, err := markdownFilesUnder(standardsDir)
	if err != nil {
		result.errors = append(result.errors, Issue{Code: "standards_unreadable", Path: rel(projectPath, standardsDir), Message: err.Error()})
		return result
	}

	indexPath := ""
	indexText := ""
	var candidates []standardCandidate
	seen := map[string]string{}
	for _, file := range markdownFiles {
		content, readErr := os.ReadFile(file)
		if readErr != nil {
			result.errors = append(result.errors, Issue{Code: "standard_unreadable", Path: rel(projectPath, file), Message: readErr.Error()})
			continue
		}
		meta := parseFrontMatter(string(content))
		if meta["kind"] == "project-standard-index" {
			if indexPath != "" {
				result.errors = append(result.errors, Issue{Code: "multiple_standard_indexes", Path: rel(projectPath, file), Message: "standards directory must have one project-standard-index"})
				continue
			}
			indexPath = file
			indexText = string(content)
			continue
		}
		if meta["kind"] != "project-standard" {
			continue
		}
		candidates = append(candidates, standardCandidate{path: file, meta: meta})
	}
	for _, candidate := range candidates {
		standard, standardIssues := validateStandard(projectPath, standardsDir, candidate.path, candidate.meta, indexText, agentsText)
		result.deliveryRoot.Standards = append(result.deliveryRoot.Standards, standard)
		result.errors = append(result.errors, standardIssues...)
		if previous, ok := seen[standard.ID]; ok && standard.ID != "" {
			result.errors = append(result.errors, Issue{Code: "duplicate_standard_id", Path: standard.Path, Message: fmt.Sprintf("standard ID %s is already used by %s", standard.ID, previous)})
		} else if standard.ID != "" {
			seen[standard.ID] = standard.Path
		}
	}
	if indexPath == "" {
		result.errors = append(result.errors, Issue{Code: "standard_index_missing", Path: result.deliveryRoot.StandardsDir, Message: "standards directory requires one Markdown file with kind: project-standard-index"})
	} else {
		result.deliveryRoot.IndexPath = rel(projectPath, indexPath)
		for i := range result.deliveryRoot.Standards {
			standard := &result.deliveryRoot.Standards[i]
			standard.Indexed = strings.Contains(indexText, standard.ID) && strings.Contains(indexText, filepath.Base(standard.Path))
			if standard.Status == "active" && !standard.Indexed {
				result.errors = append(result.errors, Issue{Code: "active_standard_not_indexed", Path: standard.Path, Message: fmt.Sprintf("active standard %s must be registered by ID and path in the standard index", standard.ID)})
			}
		}
	}
	if agentsErr != nil && activeCount(result.deliveryRoot.Standards) > 0 {
		result.errors = append(result.errors, Issue{Code: "agents_missing", Path: "AGENTS.md", Message: "active project standards require an AGENTS.md read gate"})
	}
	for i := range result.deliveryRoot.Standards {
		standard := &result.deliveryRoot.Standards[i]
		standard.GateRead = strings.Contains(agentsText, standard.ID) && strings.Contains(agentsText, standard.Path)
		if standard.Status == "active" && !standard.GateRead {
			result.errors = append(result.errors, Issue{Code: "active_standard_read_gate_missing", Path: standard.Path, Message: fmt.Sprintf("AGENTS.md must reference active standard %s and its stable path", standard.ID)})
		}
	}

	processIndex := findProcessIndex(filepath.Join(filepath.Dir(governanceDir), "01-blueprint", "processes"))
	if processIndex != "" {
		result.deliveryRoot.ProcessIndex = rel(projectPath, processIndex)
		if content, err := os.ReadFile(processIndex); err != nil {
			result.errors = append(result.errors, Issue{Code: "process_index_unreadable", Path: result.deliveryRoot.ProcessIndex, Message: err.Error()})
		} else {
			text := string(content)
			for _, required := range []struct {
				value string
				code  string
			}{
				{value: "standards/", code: "process_index_standard_reference_missing"},
				{value: "FEAT", code: "process_index_feat_reference_missing"},
				{value: ".claw/task-board.md", code: "process_index_task_reference_missing"},
			} {
				if !strings.Contains(text, required.value) {
					result.errors = append(result.errors, Issue{Code: required.code, Path: result.deliveryRoot.ProcessIndex, Message: fmt.Sprintf("process index must reference %s", required.value)})
				}
			}
		}
	}

	sort.Slice(result.deliveryRoot.Standards, func(i, j int) bool {
		return result.deliveryRoot.Standards[i].ID < result.deliveryRoot.Standards[j].ID
	})
	return result
}

func validateStandard(projectPath string, standardsDir string, file string, meta map[string]string, indexText string, agentsText string) (Standard, []Issue) {
	standard := Standard{
		ID:      meta["standard_id"],
		Path:    rel(projectPath, file),
		Status:  strings.ToLower(meta["status"]),
		Version: meta["version"],
	}
	var issues []Issue
	for _, field := range requiredStandardFields {
		if strings.TrimSpace(meta[field]) == "" {
			issues = append(issues, Issue{Code: "standard_metadata_missing", Path: standard.Path, Message: fmt.Sprintf("project standard requires front matter field %s", field)})
		}
	}
	if standard.Status != "" && !supportedStandardStatuses[standard.Status] {
		issues = append(issues, Issue{Code: "standard_status_invalid", Path: standard.Path, Message: fmt.Sprintf("unsupported project standard status %q", standard.Status)})
	}
	relToStandards, _ := filepath.Rel(standardsDir, file)
	inArchive := firstPathSegment(relToStandards) == "archive"
	if standard.Status == "active" && inArchive {
		issues = append(issues, Issue{Code: "active_standard_in_archive", Path: standard.Path, Message: "active standard cannot be stored under archive"})
	}
	if standard.Status == "archived" && !inArchive {
		issues = append(issues, Issue{Code: "archived_standard_outside_archive", Path: standard.Path, Message: "archived standard must be stored under archive"})
	}
	if unstableStandardName.MatchString(filepath.Base(file)) {
		issues = append(issues, Issue{Code: "unstable_standard_filename", Path: standard.Path, Message: "standard filename must not contain a date or version"})
	}
	standard.Indexed = standard.ID != "" && strings.Contains(indexText, standard.ID) && strings.Contains(indexText, filepath.Base(file))
	standard.GateRead = standard.ID != "" && strings.Contains(agentsText, standard.ID) && strings.Contains(agentsText, standard.Path)
	return standard, issues
}

func markdownFilesUnder(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
			files = append(files, path)
		}
		return nil
	})
	sort.Strings(files)
	return files, err
}

func parseFrontMatter(content string) map[string]string {
	meta := map[string]string{}
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return meta
	}
	for _, line := range lines[1:] {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			break
		}
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		key, value, ok := strings.Cut(trimmed, ":")
		if !ok {
			continue
		}
		meta[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"'`)
	}
	return meta
}

func findProcessIndex(processDir string) string {
	entries, err := os.ReadDir(processDir)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
			continue
		}
		path := filepath.Join(processDir, entry.Name())
		content, err := os.ReadFile(path)
		if err == nil && parseFrontMatter(string(content))["kind"] == "process-diagram-index" {
			return path
		}
	}
	return ""
}

func activeCount(standards []Standard) int {
	count := 0
	for _, standard := range standards {
		if standard.Status == "active" {
			count++
		}
	}
	return count
}

func firstPathSegment(path string) string {
	path = filepath.ToSlash(path)
	if index := strings.Index(path, "/"); index >= 0 {
		return path[:index]
	}
	return path
}

func rel(root string, path string) string {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(relative)
}
