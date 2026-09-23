package msapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

type validationRuleFormulaResult struct {
	Valid            bool                              `json:"valid"`
	Compiler         string                            `json:"compiler"`
	CompilerManifest string                            `json:"compilerManifest"`
	Rules            []validationRuleFormulaRuleResult `json:"rules"`
}

type validationRuleFormulaRuleResult struct {
	Index          int    `json:"index"`
	ObjectSelector string `json:"objectSelector"`
	FormulaSHA256  string `json:"formulaSha256"`
	Status         string `json:"status"`
}

type validationRuleFormulaTarget struct {
	Index          int
	RuleID         string
	ObjectSelector string
	Formula        string
}

type validationRuleField struct {
	ID           string
	APIName      string
	SchemeName   string
	Type         string
	LookupObject string
}

type validationCompilerBundle struct {
	Home           string
	Java           string
	Javac          string
	ClassesRoot    string
	Classpath      []string
	ManifestSHA256 string
}

var validationRuleFieldArgumentPattern = regexp.MustCompile(`(?i)\b(PRIORVALUE|ISCHANGED|BEGINS|CONTAINS|LEFT|RIGHT|SUBSTITUTE|numberCompare|ISNULL)\s*\(\s*((?:[A-Za-z_][A-Za-z0-9_]*__r\.)*[A-Za-z_][A-Za-z0-9_]*)\b`)
var validationRuleIdentifierPattern = regexp.MustCompile(`\b(?:[A-Za-z_][A-Za-z0-9_]*__r\.)*[A-Za-z_][A-Za-z0-9_]*\b`)
var validationRuleUserVariablePattern = regexp.MustCompile(`\$User\.[A-Za-z_][A-Za-z0-9_]*`)

var validationRuleBuiltInUserVariables = map[string]bool{
	"id": true, "name": true, "roleId": true, "roleName": true,
	"profileId": true, "profileName": true, "department": true,
	"title": true, "email": true, "phone": true, "mobilePhone": true,
}

const validationRuleCompilerSource = `
import com.cloudcc.core.cls.CCInvoker;
import com.cloudcc.core.cls.ExpressionFunction;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;

public class ValidationRuleCompiler {
    public static void main(String[] args) throws Exception {
        String expression = Files.readString(Path.of(args[0]), StandardCharsets.UTF_8).trim();
        ExpressionFunction compiled = new CCInvoker().newFunction(expression);
        if (compiled == null) {
            throw new IllegalArgumentException("cceg rejected the validation-rule expression");
        }
    }
}
`

// The standalone compiler bundle intentionally omits the server logging backend.
// CCInvoker only needs this API surface while performing an offline compilation.
const validationRuleLog4jStubSource = `
package org.apache.log4j;

public class Logger {
    public static Logger getLogger(Class<?> ignored) { return new Logger(); }
    public void error(Object ignored) {}
    public void info(Object ignored) {}
    public void warn(Object ignored) {}
    public void debug(Object ignored) {}
}
`

const validationRuleConfigCenterStubSource = `
package com.cloudcc.configcenter.service;

import java.util.HashMap;
import java.util.Map;

public class CCConfigCenter {
    public static Map<String, String> load(String ignored) { return new HashMap<>(); }
}
`

func (c *client) validateValidationRuleRequest(body map[string]any) (*validationRuleFormulaResult, error) {
	if normalizeDomain(firstMapString(body, "domain")) != "validation-rules" {
		return nil, nil
	}
	operation := strings.ToLower(firstMapString(body, "operation", "mode"))
	if operation == "delete" || operation == "remove" {
		return nil, nil
	}
	spec, _ := body["spec"].(map[string]any)
	if spec == nil {
		spec = body
	}
	targets, err := validationRuleFormulaTargets(spec, operation)
	if err != nil || len(targets) == 0 {
		return nil, err
	}
	bundle, err := discoverValidationCompilerBundle(c.projectPath)
	if err != nil {
		return nil, err
	}
	result := &validationRuleFormulaResult{
		Valid: true, Compiler: "packaged-cceg", CompilerManifest: bundle.ManifestSHA256,
		Rules: make([]validationRuleFormulaRuleResult, 0, len(targets)),
	}
	var userFields map[string]validationRuleField
	for _, target := range targets {
		if target.ObjectSelector == "" {
			target.ObjectSelector, err = c.validationRuleObjectSelector(target.RuleID)
			if err != nil {
				return nil, fmt.Errorf("validation rule %d target object: %w", target.Index, err)
			}
		}
		fields, err := c.validationRuleFields(target.ObjectSelector)
		if err != nil {
			return nil, fmt.Errorf("validation rule %d field metadata: %w", target.Index, err)
		}
		fields, err = c.validationRuleRelationshipFields(target.Formula, fields)
		if err != nil {
			return nil, fmt.Errorf("validation rule %d relationship metadata: %w", target.Index, err)
		}
		if validationRuleUsesDynamicUserVariable(target.Formula) && userFields == nil {
			userFields, err = c.validationRuleFields("user")
			if err != nil {
				return nil, fmt.Errorf("validation rule user field metadata: %w", err)
			}
		}
		expression, err := normalizeValidationRuleFormula(target.Formula, fields, userFields)
		if err != nil {
			return nil, fmt.Errorf("validation rule %d formula is invalid: %w", target.Index, err)
		}
		if err := compileValidationRuleFormula(bundle, expression); err != nil {
			return nil, fmt.Errorf("validation rule %d formula compilation failed: %w", target.Index, err)
		}
		digest := sha256.Sum256([]byte(target.Formula))
		result.Rules = append(result.Rules, validationRuleFormulaRuleResult{
			Index: target.Index, ObjectSelector: target.ObjectSelector,
			FormulaSHA256: hex.EncodeToString(digest[:]), Status: "passed",
		})
	}
	return result, nil
}

func validationRuleFormulaTargets(spec map[string]any, operation string) ([]validationRuleFormulaTarget, error) {
	items := mapList(spec["validationRules"])
	if len(items) == 0 {
		if wrapped, ok := spec["validate"].(map[string]any); ok {
			items = []map[string]any{wrapped}
		} else {
			items = []map[string]any{spec}
		}
	}
	targets := make([]validationRuleFormulaTarget, 0, len(items))
	rootObject := firstMapString(spec, "objectId", "objId", "objid", "objectApiName", "objectPrefix", "prefix")
	for index, item := range items {
		ruleID := firstMapString(item, "id", "ruleId", "validationRuleId")
		formula := firstMapString(item, "formula", "functionCode", "expression", "ruleContent", "formulaText")
		if formula == "" {
			if operation == "create" {
				return nil, fmt.Errorf("validation rule %d requires formula/functionCode/expression", index)
			}
			continue
		}
		selector := firstMapString(item, "objectId", "objId", "objid", "objectApiName", "objectPrefix", "prefix")
		if selector == "" {
			selector = rootObject
		}
		if selector == "" && operation == "create" {
			return nil, fmt.Errorf("validation rule %d requires objectId, objectApiName, or objectPrefix", index)
		}
		if selector == "" && ruleID == "" {
			return nil, fmt.Errorf("validation rule %d requires an object selector or rule id for formula validation", index)
		}
		targets = append(targets, validationRuleFormulaTarget{Index: index, RuleID: ruleID, ObjectSelector: selector, Formula: formula})
	}
	return targets, nil
}

func (c *client) validationRuleObjectSelector(ruleID string) (string, error) {
	response, err := c.requestJSONMap(http.MethodGet, "/metadata/v1/validation-rules/"+url.PathEscape(ruleID), nil)
	if err != nil {
		return "", err
	}
	rule, _ := response["validationRule"].(map[string]any)
	selector := firstMapString(rule, "objectId", "objId", "objid", "objectApiName", "objectPrefix", "prefix")
	if selector == "" {
		return "", fmt.Errorf("validation rule %q detail does not include objectId", ruleID)
	}
	return selector, nil
}

func (c *client) validationRuleFields(objectSelector string) (map[string]validationRuleField, error) {
	response, err := c.requestJSONMap(http.MethodGet, "/metadata/v1/fields?object="+url.QueryEscape(objectSelector), nil)
	if err != nil {
		return nil, err
	}
	if found, exists := response["found"].(bool); exists && !found {
		return nil, fmt.Errorf("object selector %q was not found", objectSelector)
	}
	fields := map[string]validationRuleField{}
	for _, key := range []string{"standardFields", "customFields"} {
		for _, item := range mapList(response[key]) {
			apiName := firstMapString(item, "apiName", "apiname")
			schemeName := firstMapString(item, "schemefieldName", "schemefield_name")
			if apiName == "" {
				apiName = schemeName
			}
			if schemeName == "" {
				schemeName = apiName
			}
			fieldType := firstMapString(item, "schemefieldType", "schemefield_type", "type")
			if apiName == "" {
				continue
			}
			field := validationRuleField{
				ID: firstMapString(item, "id", "fieldId", "field_id"), APIName: apiName, SchemeName: schemeName, Type: fieldType,
				LookupObject: firstMapString(item, "lookupObj", "lookup_obj", "referenceTo", "reference_to"),
			}
			for _, alias := range append(validationRuleFieldAliases(apiName), validationRuleFieldAliases(schemeName)...) {
				fields[strings.ToLower(alias)] = field
			}
		}
	}
	return fields, nil
}

func (c *client) validationRuleRelationshipFields(formula string, rootFields map[string]validationRuleField) (map[string]validationRuleField, error) {
	result := make(map[string]validationRuleField, len(rootFields))
	for key, field := range rootFields {
		result[key] = field
	}
	tokens := map[string]bool{}
	rewriteValidationRuleCode(formula, func(code string) string {
		for _, token := range validationRuleIdentifierPattern.FindAllString(code, -1) {
			if strings.Contains(token, "__r.") {
				tokens[token] = true
			}
		}
		return code
	})
	cache := map[string]map[string]validationRuleField{}
	for token := range tokens {
		parts := strings.Split(token, "__r.")
		current := rootFields
		for index, part := range parts {
			field, ok := validationRuleLookupField(current, part)
			if !ok {
				return nil, fmt.Errorf("field path %s cannot resolve segment %s", token, part)
			}
			if index == len(parts)-1 {
				result[strings.ToLower(token)] = field
				break
			}
			if strings.TrimSpace(field.LookupObject) == "" {
				return nil, fmt.Errorf("field path %s segment %s is not a lookup field", token, part)
			}
			next := cache[field.LookupObject]
			if next == nil {
				var err error
				next, err = c.validationRuleFields(field.LookupObject)
				if err != nil {
					return nil, fmt.Errorf("field path %s lookup %s: %w", token, field.LookupObject, err)
				}
				cache[field.LookupObject] = next
			}
			current = next
		}
	}
	return result, nil
}

func validationRuleFieldAliases(apiName string) []string {
	trimmed := strings.TrimSpace(apiName)
	base := strings.TrimSuffix(trimmed, "__c")
	return []string{trimmed, base, base + "__c"}
}

func validationRuleUsesDynamicUserVariable(formula string) bool {
	dynamic := false
	rewriteValidationRuleCode(formula, func(code string) string {
		for _, variable := range validationRuleUserVariablePattern.FindAllString(code, -1) {
			if !validationRuleBuiltInUserVariables[strings.TrimPrefix(variable, "$User.")] {
				dynamic = true
			}
		}
		return code
	})
	return dynamic
}

func normalizeValidationRuleFormula(formula string, fields map[string]validationRuleField, userFields map[string]validationRuleField) (string, error) {
	expression := strings.TrimSpace(formula)
	if expression == "" {
		return "", fmt.Errorf("formula is empty")
	}
	var variableError error
	expression = rewriteValidationRuleCode(expression, func(code string) string {
		return validationRuleUserVariablePattern.ReplaceAllStringFunc(code, func(value string) string {
			name := strings.TrimPrefix(value, "$User.")
			if !validationRuleBuiltInUserVariables[name] {
				field, ok := userFields[strings.ToLower(name)]
				if !ok || field.SchemeName != name {
					variableError = fmt.Errorf("user field %s does not exist; run cloudcc get fields <projectPath> ccuser and use the exact schemefieldName", value)
					return value
				}
				return validationRuleRepresentativeValue(field)
			}
			return `"sample-user-value"`
		})
	})
	if variableError != nil {
		return "", variableError
	}
	var fieldError error
	expression = rewriteValidationRuleCode(expression, func(code string) string {
		return validationRuleFieldArgumentPattern.ReplaceAllStringFunc(code, func(match string) string {
			parts := validationRuleFieldArgumentPattern.FindStringSubmatch(match)
			field, ok := validationRuleLookupField(fields, parts[2])
			if !ok {
				fieldError = fmt.Errorf("field %s does not exist on the target object", parts[2])
				return match
			}
			return strings.Replace(match, parts[2], `"`+field.SchemeName+`"`, 1)
		})
	})
	if fieldError != nil {
		return "", fieldError
	}
	expression = rewriteValidationRuleCode(expression, func(code string) string {
		return replaceValidationRuleIdentifiers(code, fields, &fieldError)
	})
	if fieldError != nil {
		return "", fieldError
	}
	return expression, nil
}

func validationRuleLookupField(fields map[string]validationRuleField, token string) (validationRuleField, bool) {
	field, ok := fields[strings.ToLower(token)]
	if !ok {
		field, ok = fields[strings.ToLower(strings.TrimSuffix(token, "__c"))]
	}
	return field, ok
}

func replaceValidationRuleIdentifiers(code string, fields map[string]validationRuleField, fieldError *error) string {
	matches := validationRuleIdentifierPattern.FindAllStringIndex(code, -1)
	if len(matches) == 0 {
		return code
	}
	var out strings.Builder
	start := 0
	for _, match := range matches {
		out.WriteString(code[start:match[0]])
		token := code[match[0]:match[1]]
		field, ok := validationRuleLookupField(fields, token)
		if ok && !validationRuleIdentifierIsFunction(code, match[1]) {
			out.WriteString(validationRuleRepresentativeValue(field))
		} else {
			if !ok && strings.HasSuffix(strings.ToLower(token), "__c") && *fieldError == nil {
				*fieldError = fmt.Errorf("field %s does not exist on the target object", token)
			}
			out.WriteString(token)
		}
		start = match[1]
	}
	out.WriteString(code[start:])
	return out.String()
}

func validationRuleIdentifierIsFunction(code string, end int) bool {
	for end < len(code) && (code[end] == ' ' || code[end] == '\t' || code[end] == '\r' || code[end] == '\n') {
		end++
	}
	return end < len(code) && code[end] == '('
}

func rewriteValidationRuleCode(expression string, rewrite func(string) string) string {
	var out strings.Builder
	codeStart := 0
	for index := 0; index < len(expression); {
		if expression[index] != '"' && expression[index] != '\'' {
			index++
			continue
		}
		out.WriteString(rewrite(expression[codeStart:index]))
		quote := expression[index]
		literalStart := index
		index++
		for index < len(expression) {
			if expression[index] == '\\' {
				index += 2
				continue
			}
			if expression[index] == quote {
				index++
				break
			}
			index++
		}
		if index > len(expression) {
			index = len(expression)
		}
		out.WriteString(expression[literalStart:index])
		codeStart = index
	}
	out.WriteString(rewrite(expression[codeStart:]))
	return out.String()
}

func validationRuleRepresentativeValue(field validationRuleField) string {
	fieldType := strings.TrimSpace(field.Type)
	switch fieldType {
	case "P", "N", "SCORE", "c", "C", "Z":
		return "1"
	case "B":
		if field.ID == "ffe201212041212DdimA" {
			return "1"
		}
		return "true"
	case "D":
		return `new java.text.SimpleDateFormat("yyyy-MM-dd").parse("2099-12-31")`
	case "F":
		return `new java.text.SimpleDateFormat("yyyy-MM-dd HH:mm:ss").parse("2099-12-31 00:00:00")`
	default:
		return `"sample"`
	}
}

func discoverValidationCompilerBundle(projectPath string) (validationCompilerBundle, error) {
	home := ""
	var executableCompilerHome string
	if executable, err := os.Executable(); err == nil {
		if resolved, resolveErr := filepath.EvalSymlinks(executable); resolveErr == nil {
			executable = resolved
		}
		executableCompilerHome = filepath.Clean(filepath.Join(filepath.Dir(executable), "..", "..", "assets", "class-compiler"))
	}
	workingDirectory, _ := os.Getwd()
	for _, candidate := range []string{os.Getenv("CLOUDCC_CLASS_COMPILER_HOME"), executableCompilerHome, findValidationCompilerHome(projectPath), findValidationCompilerHome(workingDirectory)} {
		if validationCompilerHome(candidate) {
			home = candidate
			break
		}
	}
	if home == "" {
		return validationCompilerBundle{}, fmt.Errorf("packaged expression compiler was not found; expected assets/class-compiler")
	}
	absHome, _ := filepath.Abs(home)
	javaPath, javacPath := "", ""
	javaHome := firstNonBlank(os.Getenv("CLOUDCC_JAVA_HOME"), os.Getenv("JAVA_HOME"))
	if javaHome != "" {
		javaPath = filepath.Join(javaHome, "bin", validationExecutableName("java"))
		javacPath = filepath.Join(javaHome, "bin", validationExecutableName("javac"))
	}
	if !regularFile(javaPath) {
		javaPath, _ = exec.LookPath("java")
	}
	if !regularFile(javacPath) {
		javacPath, _ = exec.LookPath("javac")
	}
	if !regularFile(javaPath) || !regularFile(javacPath) {
		return validationCompilerBundle{}, fmt.Errorf("JDK java and javac are required for validation-rule formula compilation")
	}
	manifestPath := filepath.Join(absHome, "manifest.json")
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return validationCompilerBundle{}, fmt.Errorf("read compiler manifest: %w", err)
	}
	var manifest struct {
		Artifacts []struct {
			Path   string `json:"path"`
			Size   int64  `json:"size"`
			SHA256 string `json:"sha256"`
		} `json:"artifacts"`
	}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return validationCompilerBundle{}, fmt.Errorf("parse compiler manifest: %w", err)
	}
	classpath := []string{filepath.Join(absHome, "WEB-INF", "classes")}
	for _, artifact := range manifest.Artifacts {
		path := filepath.Join(absHome, filepath.FromSlash(artifact.Path))
		contents, err := os.ReadFile(path)
		if err != nil {
			return validationCompilerBundle{}, fmt.Errorf("compiler artifact %s: %w", artifact.Path, err)
		}
		digest := sha256.Sum256(contents)
		if int64(len(contents)) != artifact.Size || !strings.EqualFold(hex.EncodeToString(digest[:]), artifact.SHA256) {
			return validationCompilerBundle{}, fmt.Errorf("compiler artifact integrity mismatch: %s", artifact.Path)
		}
		if strings.HasSuffix(strings.ToLower(path), ".jar") {
			classpath = append(classpath, path)
		}
	}
	sort.Strings(classpath[1:])
	manifestDigest := sha256.Sum256(manifestBytes)
	return validationCompilerBundle{Home: absHome, Java: javaPath, Javac: javacPath,
		ClassesRoot: classpath[0], Classpath: classpath, ManifestSHA256: hex.EncodeToString(manifestDigest[:])}, nil
}

func compileValidationRuleFormula(bundle validationCompilerBundle, expression string) error {
	workDir, err := os.MkdirTemp("", "cloudcc-validation-rule-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workDir)
	compilerFile := filepath.Join(workDir, "ValidationRuleCompiler.java")
	expressionFile := filepath.Join(workDir, "expression.txt")
	loggerDir := filepath.Join(workDir, "org", "apache", "log4j")
	loggerFile := filepath.Join(loggerDir, "Logger.java")
	configCenterDir := filepath.Join(workDir, "com", "cloudcc", "configcenter", "service")
	configCenterFile := filepath.Join(configCenterDir, "CCConfigCenter.java")
	classesDir := filepath.Join(workDir, "classes")
	if err := os.MkdirAll(classesDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(loggerDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(configCenterDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(compilerFile, []byte(validationRuleCompilerSource), 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(loggerFile, []byte(validationRuleLog4jStubSource), 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(configCenterFile, []byte(validationRuleConfigCenterStubSource), 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(expressionFile, []byte(expression), 0o600); err != nil {
		return err
	}
	for _, name := range []string{"cceg.jar", "cloudcc.jar"} {
		var source string
		for _, entry := range bundle.Classpath {
			if strings.EqualFold(filepath.Base(entry), name) {
				source = entry
				break
			}
		}
		if source == "" {
			return fmt.Errorf("packaged compiler is missing %s", name)
		}
		contents, err := os.ReadFile(source)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(classesDir, name), contents, 0o600); err != nil {
			return err
		}
	}
	classpath := strings.Join(bundle.Classpath, string(os.PathListSeparator))
	if output, err := runValidationCompilerCommand(bundle.Javac, "-J-Dfile.encoding=UTF-8", "-J-Duser.language=en", "-J-Duser.country=US", "-encoding", "UTF-8", "-classpath", classpath, "-d", classesDir, loggerFile, configCenterFile, compilerFile); err != nil {
		return fmt.Errorf("cannot compile cceg validation runner: %s", strings.TrimSpace(output))
	}
	runtimeClasspath := strings.Join(append([]string{classesDir}, bundle.Classpath...), string(os.PathListSeparator))
	if output, err := runValidationCompilerCommand(bundle.Java,
		"-Dfile.encoding=UTF-8", "-Dstdout.encoding=UTF-8", "-Dstderr.encoding=UTF-8",
		"-Dsun.stdout.encoding=UTF-8", "-Dsun.stderr.encoding=UTF-8",
		"-Duser.language=en", "-Duser.country=US",
		"-classpath", runtimeClasspath, "ValidationRuleCompiler", expressionFile); err != nil {
		detail := strings.TrimSpace(output)
		if detail == "" {
			detail = err.Error()
		}
		return fmt.Errorf("cceg rejected expression: %s", detail)
	}
	return nil
}

func runValidationCompilerCommand(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return string(output), fmt.Errorf("compiler timed out")
	}
	return string(output), err
}

func findValidationCompilerHome(start string) string {
	current, err := filepath.Abs(strings.TrimSpace(start))
	if err != nil || current == "" {
		return ""
	}
	for {
		for _, candidate := range []string{filepath.Join(current, "assets", "class-compiler"), filepath.Join(current, "cc-customization-expert-go", "assets", "class-compiler")} {
			if validationCompilerHome(candidate) {
				return candidate
			}
		}
		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}
		current = parent
	}
}

func validationCompilerHome(path string) bool {
	return regularFile(filepath.Join(strings.TrimSpace(path), "manifest.json")) && regularFile(filepath.Join(strings.TrimSpace(path), "WEB-INF", "lib", "cceg.jar"))
}

func regularFile(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func validationExecutableName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
