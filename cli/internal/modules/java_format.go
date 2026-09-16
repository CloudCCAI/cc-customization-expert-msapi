package modules

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const (
	javaFormatterVersion = "1.29.0"
	javaFormatterJarName = "google-java-format-1.29.0-all-deps.jar"
	javaFormatterSHA256  = "aeb4e1831d56011e7e06f3393c4e17340c7cca0fd7ba9076ab8dc0624759f7f0"
	formatStartMarker    = "// CLOUDCC_FORMAT_FRAGMENT_START"
	formatEndMarker      = "// CLOUDCC_FORMAT_FRAGMENT_END"
)

type javaFormatIssue struct {
	Line    int    `json:"line"`
	Rule    string `json:"rule"`
	Message string `json:"message"`
}

type javaFormatResult struct {
	Status           string            `json:"status"`
	Resource         string            `json:"resource"`
	Name             string            `json:"name,omitempty"`
	SourceFile       string            `json:"sourceFile"`
	Changed          bool              `json:"changed"`
	Written          bool              `json:"written"`
	Formatter        string            `json:"formatter"`
	FormatterVersion string            `json:"formatterVersion"`
	Style            string            `json:"style"`
	Issues           []javaFormatIssue `json:"issues,omitempty"`
	RepairCommand    string            `json:"repairCommand,omitempty"`
}

func handleJavaFormat(resource string, args []string, stdout io.Writer, cwd string) error {
	if resource == "highcode" {
		return checkHighCodeJavaFormat(args, stdout, cwd)
	}
	if len(args) == 0 || strings.HasPrefix(args[0], "--") {
		return fmt.Errorf("cloudcc format %s <name> [projectPath] [--check|--write]", resource)
	}
	namePath := args[0]
	projectPath := cwd
	projectProvided := false
	write := false
	for _, arg := range args[1:] {
		switch {
		case arg == "--write":
			write = true
		case arg == "--check":
		case strings.HasPrefix(arg, "--"):
			return fmt.Errorf("unknown Java format option: %s", arg)
		case !projectProvided:
			projectPath = arg
			projectProvided = true
		default:
			return fmt.Errorf("unexpected Java format argument: %s", arg)
		}
	}
	if abs, err := filepath.Abs(projectPath); err == nil {
		projectPath = abs
	}
	sourceFile, canonicalResource, name, err := javaResourceSourceFile(resource, namePath, projectPath)
	if err != nil {
		return err
	}
	result, err := formatJavaFile(sourceFile, canonicalResource, name, projectPath, write)
	if write {
		result.RepairCommand = ""
	} else {
		result.RepairCommand = javaFormatRepairCommand(canonicalResource, namePath, projectPath)
	}
	if writeErr := writeJSON(stdout, result); writeErr != nil {
		return writeErr
	}
	return err
}

func javaResourceSourceFile(resource string, namePath string, projectPath string) (string, string, string, error) {
	name := filepath.Base(namePath)
	switch resource {
	case "classes":
		return backendResourcePath(projectPath, "classes", namePath, name+".java"), "classes", name, nil
	case "trigger", "triggers":
		return backendResourcePath(projectPath, "triggers", namePath, name+".java"), "trigger", name, nil
	case "timer", "schedule":
		return backendResourcePath(projectPath, "schedule", namePath, name+".java"), "timer", name, nil
	default:
		return "", "", "", fmt.Errorf("Java formatting supports classes, trigger, timer, or highcode; got %s", resource)
	}
}

func formatJavaFile(sourceFile string, resource string, name string, projectPath string, write bool) (javaFormatResult, error) {
	result := javaFormatResult{
		Status:           "format_check_failed",
		Resource:         resource,
		Name:             name,
		SourceFile:       sourceFile,
		Formatter:        "google-java-format",
		FormatterVersion: javaFormatterVersion,
		Style:            "AOSP (4-space indentation)",
	}
	original, err := os.ReadFile(sourceFile)
	if err != nil {
		return result, err
	}
	formatted, err := formatJavaSource(string(original), projectPath)
	if err != nil {
		result.Issues = []javaFormatIssue{{Line: 1, Rule: "formatter_error", Message: err.Error()}}
		return result, err
	}
	const maxFormatPasses = 5
	stable := false
	for pass := 1; pass <= maxFormatPasses; pass++ {
		next, nextErr := formatJavaSource(formatted, projectPath)
		if nextErr != nil {
			result.Issues = []javaFormatIssue{{Line: 1, Rule: "formatter_error", Message: nextErr.Error()}}
			return result, nextErr
		}
		if next == formatted {
			stable = true
			break
		}
		formatted = next
	}
	if !stable {
		result.Issues = []javaFormatIssue{{Line: 1, Rule: "formatter_nonconvergent", Message: "Java formatter did not converge to a stable result"}}
		return result, fmt.Errorf("Java formatter did not converge after %d passes", maxFormatPasses)
	}
	result.Changed = !bytes.Equal(original, []byte(formatted))
	if result.Changed {
		result.Issues = javaFormatDiffIssues(string(original), formatted)
		if !write {
			return result, fmt.Errorf("%s source is not canonically formatted", resource)
		}
		if err := os.WriteFile(sourceFile, []byte(formatted), 0o644); err != nil {
			return result, err
		}
		result.Written = true
		result.Status = "formatted"
		return result, nil
	}
	result.Status = "format_clean"
	return result, nil
}

func normalizeJavaNewlines(source []byte) []byte {
	source = bytes.ReplaceAll(source, []byte("\r\n"), []byte("\n"))
	source = bytes.ReplaceAll(source, []byte("\r"), []byte("\n"))
	return source
}

func formatJavaSource(source string, projectPath string) (string, error) {
	source = string(normalizeJavaNewlines([]byte(source)))
	source = strings.TrimPrefix(source, "\uFEFF")
	fragment := !hasTopLevelJavaType(source)
	input := source
	if fragment {
		input = wrapJavaFragment(source)
	}
	jar := discoverJavaFormatterJar(projectPath)
	if jar == "" {
		return "", fmt.Errorf("packaged Java formatter is missing; expected tools/java-formatter/%s", javaFormatterJarName)
	}
	if err := verifyJavaFormatterJar(jar); err != nil {
		return "", err
	}
	java := discoverJavaFormatterExecutable(projectPath)
	if java == "" {
		return "", fmt.Errorf("JDK 21 java executable is required for high-code formatting")
	}
	workDir, err := os.MkdirTemp("", "cloudcc-java-format-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(workDir)
	inputFile := filepath.Join(workDir, "CloudCCFormatInput.java")
	if err := os.WriteFile(inputFile, []byte(input), 0o600); err != nil {
		return "", err
	}
	cmd := exec.Command(java, "-jar", jar, "--aosp", "--skip-sorting-imports", "--skip-removing-unused-imports", "--skip-javadoc-formatting", inputFile)
	output, runErr := cmd.CombinedOutput()
	if runErr != nil {
		return "", fmt.Errorf("google-java-format failed: %s", strings.TrimSpace(string(output)))
	}
	formatted := string(normalizeJavaNewlines(output))
	if fragment {
		return unwrapJavaFragment(formatted)
	}
	return ensureOneFinalNewline(formatted), nil
}

func discoverJavaFormatterExecutable(projectPath string) string {
	opts := classDevOptions{ProjectPath: projectPath}
	fillClassDevOptionsFromProject(&opts)
	for _, home := range []string{opts.JavaHome, os.Getenv("CLOUDCC_JAVA_HOME"), os.Getenv("JAVA_HOME")} {
		home = strings.TrimSpace(home)
		if home == "" {
			continue
		}
		candidate := filepath.Join(home, "bin", executableName("java"))
		if isExecutableFile(candidate) {
			return candidate
		}
	}
	if found, err := exec.LookPath(executableName("java")); err == nil {
		return found
	}
	return ""
}

func hasTopLevelJavaType(source string) bool {
	for _, declaration := range javaNamedTypeDeclarations(source) {
		if declaration.depth == 0 {
			return true
		}
	}
	return false
}

func wrapJavaFragment(source string) string {
	var body strings.Builder
	for _, line := range strings.Split(strings.Trim(source, "\n"), "\n") {
		body.WriteString("        ")
		body.WriteString(line)
		body.WriteByte('\n')
	}
	return "final class CloudCCFormatFragment {\n    void execute() throws Exception {\n        " + formatStartMarker + "\n" + body.String() + "        " + formatEndMarker + "\n    }\n}\n"
}

func unwrapJavaFragment(formatted string) (string, error) {
	start := strings.Index(formatted, formatStartMarker)
	end := strings.Index(formatted, formatEndMarker)
	if start < 0 || end < 0 || end <= start {
		return "", fmt.Errorf("formatted Java fragment markers were not preserved")
	}
	start = strings.Index(formatted[start:], "\n") + start + 1
	fragment := formatted[start:end]
	lines := strings.Split(strings.TrimSuffix(fragment, "\n"), "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "        ") {
			lines[i] = strings.TrimPrefix(line, "        ")
		}
	}
	return ensureOneFinalNewline(strings.Join(lines, "\n")), nil
}

func ensureOneFinalNewline(value string) string {
	return strings.TrimRight(value, "\n") + "\n"
}

func discoverJavaFormatterJar(projectPath string) string {
	if explicit := strings.TrimSpace(os.Getenv("CLOUDCC_JAVA_FORMATTER_JAR")); isExecutableFile(explicit) {
		return explicit
	}
	if executable, err := os.Executable(); err == nil {
		if resolved, resolveErr := filepath.EvalSymlinks(executable); resolveErr == nil {
			executable = resolved
		}
		candidate := filepath.Join(filepath.Dir(executable), "..", "java-formatter", javaFormatterJarName)
		if isExecutableFile(candidate) {
			return candidate
		}
	}
	for _, start := range []string{projectPath, currentWorkingDirectory()} {
		current, err := filepath.Abs(start)
		if err != nil {
			continue
		}
		for {
			for _, candidate := range []string{
				filepath.Join(current, "tools", "java-formatter", javaFormatterJarName),
				filepath.Join(current, "cc-customization-expert-go", "tools", "java-formatter", javaFormatterJarName),
			} {
				if isExecutableFile(candidate) {
					return candidate
				}
			}
			parent := filepath.Dir(current)
			if parent == current {
				break
			}
			current = parent
		}
	}
	return ""
}

func verifyJavaFormatterJar(path string) error {
	contents, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("cannot read packaged Java formatter: %w", err)
	}
	digest := sha256.Sum256(contents)
	if hex.EncodeToString(digest[:]) != javaFormatterSHA256 {
		return fmt.Errorf("packaged Java formatter SHA-256 mismatch")
	}
	return nil
}

func javaFormatDiffIssues(original string, formatted string) []javaFormatIssue {
	originalLines := strings.Split(string(normalizeJavaNewlines([]byte(original))), "\n")
	issues := make([]javaFormatIssue, 0)
	for index, text := range originalLines {
		leading := text[:len(text)-len(strings.TrimLeft(text, " \t"))]
		switch {
		case strings.Contains(leading, "\t"):
			issues = append(issues, javaFormatIssue{Line: index + 1, Rule: "tab_indentation", Message: "use four spaces for indentation; tabs are not allowed"})
		case strings.TrimRight(text, " \t") != text:
			issues = append(issues, javaFormatIssue{Line: index + 1, Rule: "trailing_whitespace", Message: "remove trailing whitespace"})
		case ordinaryJavaSemicolonCount(text) > 1:
			issues = append(issues, javaFormatIssue{Line: index + 1, Rule: "multiple_statements_on_line", Message: "ordinary Java statements must occupy separate lines"})
		}
		if len(issues) == 20 {
			return issues
		}
	}
	if len(issues) > 0 {
		return issues
	}
	formattedLines := strings.Split(formatted, "\n")
	limit := len(originalLines)
	if len(formattedLines) < limit {
		limit = len(formattedLines)
	}
	first := 0
	for first < limit && originalLines[first] == formattedLines[first] {
		first++
	}
	line := first + 1
	return []javaFormatIssue{{Line: line, Rule: "canonical_layout", Message: "source differs from the canonical AOSP Java format"}}
}

func ordinaryJavaSemicolonCount(line string) int {
	count, parens := 0, 0
	inString, inChar, escaped := false, false, false
	for i := 0; i < len(line); i++ {
		ch := line[i]
		if escaped {
			escaped = false
			continue
		}
		if (inString || inChar) && ch == '\\' {
			escaped = true
			continue
		}
		if inString {
			if ch == '"' {
				inString = false
			}
			continue
		}
		if inChar {
			if ch == '\'' {
				inChar = false
			}
			continue
		}
		if i+1 < len(line) && line[i:i+2] == "//" {
			break
		}
		switch ch {
		case '"':
			inString = true
		case '\'':
			inChar = true
		case '(':
			parens++
		case ')':
			if parens > 0 {
				parens--
			}
		case ';':
			if parens == 0 {
				count++
			}
		}
	}
	return count
}

func javaFormatRepairCommand(resource string, namePath string, projectPath string) string {
	return fmt.Sprintf("cloudcc format %s %s %s --write", resource, namePath, projectPath)
}

func requireJavaFileFormatted(sourceFile string, resource string, namePath string, projectPath string) (javaFormatResult, error) {
	name := filepath.Base(namePath)
	result, err := formatJavaFile(sourceFile, resource, name, projectPath, false)
	result.RepairCommand = javaFormatRepairCommand(resource, namePath, projectPath)
	if err != nil {
		result.Status = "blocked_local_format"
	}
	return result, err
}

func formatJavaFileBeforePublish(sourceFile string, resource string, namePath string, projectPath string) (javaFormatResult, error) {
	name := filepath.Base(namePath)
	result, err := formatJavaFile(sourceFile, resource, name, projectPath, true)
	if err != nil {
		result.Status = "blocked_local_format"
		result.RepairCommand = javaFormatRepairCommand(resource, namePath, projectPath)
	}
	return result, err
}

func checkHighCodeJavaFormat(args []string, stdout io.Writer, cwd string) error {
	projectPath := cwd
	projectProvided := false
	for _, arg := range args {
		switch {
		case arg == "--check":
		case arg == "--write":
			return fmt.Errorf("project-wide highcode format is read-only; format individual resources with --write")
		case strings.HasPrefix(arg, "--"):
			return fmt.Errorf("unknown Java format option: %s", arg)
		case !projectProvided:
			projectPath = arg
			projectProvided = true
		default:
			return fmt.Errorf("unexpected Java format argument: %s", arg)
		}
	}
	if abs, err := filepath.Abs(projectPath); err == nil {
		projectPath = abs
	}
	type scanItem struct {
		Path    string            `json:"path"`
		Changed bool              `json:"changed"`
		Issues  []javaFormatIssue `json:"issues,omitempty"`
	}
	var files []string
	for _, root := range []string{"classes", "triggers", "schedule"} {
		base := backendResourcePath(projectPath, root)
		_ = filepath.WalkDir(base, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr == nil && !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".java") && !strings.HasSuffix(entry.Name(), "Test.java") {
				files = append(files, path)
			}
			return nil
		})
	}
	sort.Strings(files)
	items := make([]scanItem, 0, len(files))
	nonCanonical := 0
	for _, file := range files {
		original, readErr := os.ReadFile(file)
		if readErr != nil {
			return readErr
		}
		formatted, formatErr := formatJavaSource(string(original), projectPath)
		item := scanItem{Path: file}
		if formatErr != nil {
			item.Changed = true
			item.Issues = []javaFormatIssue{{Line: 1, Rule: "formatter_error", Message: formatErr.Error()}}
		} else if !bytes.Equal(original, []byte(formatted)) {
			item.Changed = true
			item.Issues = javaFormatDiffIssues(string(original), formatted)
		}
		if item.Changed {
			nonCanonical++
		}
		items = append(items, item)
	}
	status := "format_clean"
	if nonCanonical > 0 {
		status = "format_check_failed"
	}
	if err := writeJSON(stdout, map[string]any{
		"status": status, "resource": "highcode", "projectPath": projectPath,
		"formatter": "google-java-format", "formatterVersion": javaFormatterVersion,
		"style": "AOSP (4-space indentation)", "scannedFiles": len(files),
		"nonCanonicalFiles": nonCanonical, "files": items,
	}); err != nil {
		return err
	}
	if nonCanonical > 0 {
		return fmt.Errorf("%d high-code Java files are not canonically formatted", nonCanonical)
	}
	return nil
}
