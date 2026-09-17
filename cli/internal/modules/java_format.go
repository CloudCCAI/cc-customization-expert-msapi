package modules

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	javaFormatterName    = "cloudcc-go-layout"
	javaFormatterVersion = "1"
	javaFormatterStyle   = "lightweight (4-space indentation)"
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
		Formatter:        javaFormatterName,
		FormatterVersion: javaFormatterVersion,
		Style:            javaFormatterStyle,
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
	result.Changed = !bytes.Equal(original, []byte(formatted))
	if result.Changed {
		result.Issues = javaFormatDiffIssues(string(original), formatted)
		if !write {
			return result, fmt.Errorf("%s source needs lightweight layout cleanup", resource)
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

func formatJavaSource(source string, _ string) (string, error) {
	source = string(normalizeJavaNewlines([]byte(source)))
	source = strings.TrimPrefix(source, "\uFEFF")
	return formatJavaLayout(source)
}

func ensureOneFinalNewline(value string) string {
	return strings.TrimRight(value, "\n") + "\n"
}

type javaLayoutState int

const (
	javaLayoutNormal javaLayoutState = iota
	javaLayoutString
	javaLayoutChar
	javaLayoutLineComment
	javaLayoutBlockComment
	javaLayoutTextBlock
)

func formatJavaLayout(source string) (string, error) {
	var lines []string
	var current strings.Builder
	state := javaLayoutNormal
	indent := 0
	parenDepth := 0
	escaped := false
	suppressPhysicalNewline := false
	currentRaw := false
	textBlockFirstLine := false

	emit := func(raw bool) bool {
		value := current.String()
		current.Reset()
		if raw {
			lines = append(lines, strings.TrimRight(value, "\r"))
			currentRaw = false
			return true
		}
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			currentRaw = false
			return false
		}
		leadingWidth := javaLeadingIndentWidth(value)
		expectedWidth := indent * 4
		if javaLayoutContinuation(trimmed, lines) && leadingWidth > expectedWidth {
			expectedWidth = leadingWidth
		}
		lines = append(lines, strings.Repeat(" ", expectedWidth)+trimmed)
		currentRaw = false
		return true
	}
	emitBlank := func() {
		if len(lines) == 0 || lines[len(lines)-1] != "" {
			lines = append(lines, "")
		}
	}

	for i := 0; i < len(source); i++ {
		ch := source[i]
		next := byte(0)
		if i+1 < len(source) {
			next = source[i+1]
		}

		switch state {
		case javaLayoutString:
			current.WriteByte(ch)
			if ch == '\n' {
				return "", fmt.Errorf("unterminated Java string literal")
			}
			if escaped {
				escaped = false
			} else if ch == '\\' {
				escaped = true
			} else if ch == '"' {
				state = javaLayoutNormal
			}
			continue
		case javaLayoutChar:
			current.WriteByte(ch)
			if ch == '\n' {
				return "", fmt.Errorf("unterminated Java character literal")
			}
			if escaped {
				escaped = false
			} else if ch == '\\' {
				escaped = true
			} else if ch == '\'' {
				state = javaLayoutNormal
			}
			continue
		case javaLayoutLineComment:
			if ch == '\n' {
				emit(currentRaw)
				state = javaLayoutNormal
				suppressPhysicalNewline = false
			} else {
				current.WriteByte(ch)
			}
			continue
		case javaLayoutBlockComment:
			if ch == '\n' {
				emit(currentRaw)
				suppressPhysicalNewline = false
				continue
			}
			current.WriteByte(ch)
			if ch == '*' && next == '/' {
				current.WriteByte(next)
				i++
				state = javaLayoutNormal
				continue
			}
			continue
		case javaLayoutTextBlock:
			if ch == '"' && i+2 < len(source) && source[i:i+3] == `"""` {
				current.WriteString(`"""`)
				i += 2
				state = javaLayoutNormal
				continue
			}
			if ch == '\n' {
				if textBlockFirstLine {
					emit(false)
					textBlockFirstLine = false
				} else {
					emit(true)
				}
				currentRaw = true
				suppressPhysicalNewline = false
			} else {
				current.WriteByte(ch)
			}
			continue
		}

		if ch == '/' && next == '/' {
			current.WriteString("//")
			i++
			state = javaLayoutLineComment
			continue
		}
		if ch == '/' && next == '*' {
			current.WriteString("/*")
			i++
			state = javaLayoutBlockComment
			continue
		}
		if ch == '"' && i+2 < len(source) && source[i:i+3] == `"""` {
			current.WriteString(`"""`)
			i += 2
			state = javaLayoutTextBlock
			textBlockFirstLine = true
			continue
		}

		switch ch {
		case '"':
			current.WriteByte(ch)
			state = javaLayoutString
		case '\'':
			current.WriteByte(ch)
			state = javaLayoutChar
		case '(':
			parenDepth++
			current.WriteByte(ch)
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
			current.WriteByte(ch)
		case '{':
			trimmed := strings.TrimRight(current.String(), " \t")
			current.Reset()
			current.WriteString(trimmed)
			if trimmed != "" && !strings.HasSuffix(trimmed, " ") {
				current.WriteByte(' ')
			}
			closing := i + 1
			for closing < len(source) && (source[closing] == ' ' || source[closing] == '\t') {
				closing++
			}
			if closing < len(source) && source[closing] == '}' {
				current.WriteString("{}")
				i = closing
				continue
			}
			current.WriteByte(ch)
			emit(currentRaw)
			indent++
			suppressPhysicalNewline = true
		case '}':
			if strings.TrimSpace(current.String()) != "" {
				emit(currentRaw)
			}
			if indent > 0 {
				indent--
			}
			current.WriteByte(ch)
			suppressPhysicalNewline = false
		case ';':
			current.WriteByte(ch)
			if parenDepth == 0 {
				emit(currentRaw)
				suppressPhysicalNewline = true
			}
		case '\n':
			if strings.TrimSpace(current.String()) != "" || currentRaw {
				emit(currentRaw)
				suppressPhysicalNewline = false
			} else if suppressPhysicalNewline {
				current.Reset()
				suppressPhysicalNewline = false
			} else {
				current.Reset()
				emitBlank()
			}
		default:
			if current.Len() == 1 && current.String() == "}" && isJavaIdentifierByte(ch) {
				current.WriteByte(' ')
			}
			current.WriteByte(ch)
		}
	}

	if state == javaLayoutString || state == javaLayoutChar || state == javaLayoutBlockComment || state == javaLayoutTextBlock {
		return "", fmt.Errorf("unterminated Java literal or block comment")
	}
	if strings.TrimSpace(current.String()) != "" || currentRaw {
		emit(currentRaw)
	}
	return ensureOneFinalNewline(strings.Join(lines, "\n")), nil
}

func javaLeadingIndentWidth(value string) int {
	width := 0
	for i := 0; i < len(value); i++ {
		switch value[i] {
		case ' ':
			width++
		case '\t':
			width += 4
		default:
			return width
		}
	}
	return width
}

func javaLayoutContinuation(current string, lines []string) bool {
	for _, prefix := range []string{".", "+", "&&", "||", "?", ":"} {
		if strings.HasPrefix(current, prefix) {
			return true
		}
	}
	if len(lines) == 0 {
		return false
	}
	previous := strings.TrimSpace(lines[len(lines)-1])
	for _, suffix := range []string{"=", "(", "[", ",", ".", "+", "-", "*", "/", "&&", "||", "?", ":"} {
		if strings.HasSuffix(previous, suffix) {
			return true
		}
	}
	return false
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
	return []javaFormatIssue{{Line: line, Rule: "readable_layout", Message: "source differs from the lightweight readable Java layout"}}
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

func formatJavaFileBeforePublish(sourceFile string, resource string, namePath string, projectPath string) (javaFormatResult, error) {
	name := filepath.Base(namePath)
	result, err := formatJavaFile(sourceFile, resource, name, projectPath, true)
	if err != nil {
		result.Status = "format_warning"
		result.RepairCommand = javaFormatRepairCommand(resource, namePath, projectPath)
		if len(result.Issues) == 0 {
			result.Issues = []javaFormatIssue{{Line: 1, Rule: "formatter_warning", Message: err.Error()}}
		}
		return result, nil
	}
	return result, nil
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
		"formatter": javaFormatterName, "formatterVersion": javaFormatterVersion,
		"style": javaFormatterStyle, "scannedFiles": len(files),
		"nonCanonicalFiles": nonCanonical, "files": items,
	}); err != nil {
		return err
	}
	if nonCanonical > 0 {
		return fmt.Errorf("%d high-code Java files need lightweight layout cleanup", nonCanonical)
	}
	return nil
}
