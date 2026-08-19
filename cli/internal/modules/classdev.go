package modules

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloudcc-customization-expert-go/internal/config"
)

const classTemplateRendererSource = `
import com.cloudcc.core.cls.fag.FagTemplate;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;

public final class CloudCCClassTemplateRenderer {
    public static void main(String[] args) throws Exception {
        if (args.length != 3) {
            throw new IllegalArgumentException("source, output and package arguments are required");
        }
        String source = Files.readString(Path.of(args[0]), StandardCharsets.UTF_8);
        String template = FagTemplate.getSource();
        String wrapped = template.replace("$triggerSrc", source)
            .replace("com.cloudcc.core.fag;", args[2] + ";");
        Files.writeString(Path.of(args[1]), wrapped, StandardCharsets.UTF_8);
    }
}
`

type classCompilerManifest struct {
	FormatVersion int                     `json:"formatVersion"`
	Compiler      string                  `json:"compiler"`
	JavaTarget    int                     `json:"javaTarget"`
	ArtifactCount int                     `json:"artifactCount"`
	Artifacts     []classCompilerArtifact `json:"artifacts"`
}

type classCompilerArtifact struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type classDevOptions struct {
	ProjectPath        string
	CompilerHome       string
	JavaHome           string
	ValidationEvidence string
}

type classDevEnvironment struct {
	CompilerHome      string   `json:"compilerHome"`
	ManifestPath      string   `json:"manifestPath"`
	ManifestSHA256    string   `json:"manifestSha256,omitempty"`
	ClassesRoot       string   `json:"classesRoot"`
	JavaHome          string   `json:"javaHome"`
	JavaVersion       string   `json:"javaVersion,omitempty"`
	Java              string   `json:"java"`
	Javac             string   `json:"javac"`
	CcegJar           string   `json:"ccegJar"`
	Classpath         []string `json:"-"`
	JarEntries        []string `json:"-"`
	ClasspathEntries  int      `json:"classpathEntries"`
	ArtifactCount     int      `json:"artifactCount"`
	IntegrityVerified bool     `json:"integrityVerified"`
	Missing           []string `json:"missing,omitempty"`
	Warnings          []string `json:"warnings,omitempty"`
	Ready             bool     `json:"ready"`
}

type classWorkflowGates struct {
	CompilerReady           bool `json:"compilerReady"`
	TargetGatewayConfigured bool `json:"targetGatewayConfigured"`
	PublishAuthConfigured   bool `json:"publishAuthConfigured"`
	PublishReady            bool `json:"publishReady"`
}

type classCompileDiagnostic struct {
	Kind          string `json:"kind"`
	Message       string `json:"message"`
	GeneratedLine int    `json:"generatedLine,omitempty"`
	SourceLine    int    `json:"sourceLine,omitempty"`
}

type classValidationResult struct {
	Status            string                   `json:"status"`
	Valid             bool                     `json:"valid"`
	ClassName         string                   `json:"className"`
	SourceFile        string                   `json:"sourceFile"`
	SourceSHA256      string                   `json:"sourceSha256"`
	CompilerHome      string                   `json:"compilerHome"`
	CompilerManifest  string                   `json:"compilerManifestSha256"`
	JavaVersion       string                   `json:"javaVersion"`
	TemplateClass     string                   `json:"templateClass"`
	GeneratedPackage  string                   `json:"generatedPackage"`
	ClasspathEntries  int                      `json:"classpathEntries"`
	Diagnostics       []classCompileDiagnostic `json:"diagnostics"`
	PolicyViolations  []string                 `json:"policyViolations,omitempty"`
	CompilationOutput string                   `json:"compilationOutput,omitempty"`
}

func handleClassDev(action string, args []string, stdout io.Writer, _ io.Writer, cwd string) error {
	switch action {
	case "doctor", "prepare":
		opts, err := parseClassDevOptions(args, cwd, false)
		if err != nil {
			return err
		}
		env := discoverClassDevEnvironment(opts)
		gates := discoverClassWorkflowGates(opts, env)
		status := "blocked"
		if env.Ready {
			status = "compile_ready"
		}
		if gates.PublishReady {
			status = "publish_ready"
		}
		if err := writeJSON(stdout, map[string]any{
			"status":      status,
			"environment": env,
			"gates":       gates,
			"standalone": map[string]any{
				"platformSourceRequired": false,
				"setupSvcRequired":       false,
				"mainSvcRequired":        false,
				"mavenCacheRequired":     false,
			},
			"nextCommands": []string{
				"cloudcc validate classes <ClassName> <projectPath>",
				"cloudcc publish classes <ClassName> <projectPath>",
			},
		}); err != nil {
			return err
		}
		if !env.Ready {
			return fmt.Errorf("standalone CloudCC class compiler is not ready: %s", strings.Join(env.Missing, "; "))
		}
		if !gates.PublishReady {
			return fmt.Errorf("compiler is ready, but target CloudCC gateway or publish authentication is not configured")
		}
		return nil
	case "validate":
		name, opts, err := parseNamedClassDevOptions(args, cwd)
		if err != nil {
			return err
		}
		result, err := validateClass(name, opts)
		if writeErr := writeJSON(stdout, result); writeErr != nil {
			return writeErr
		}
		return err
	case "test":
		return fmt.Errorf("test classes is not part of the standalone compiler/publisher; it would introduce a main-svc runtime dependency")
	default:
		return fmt.Errorf("unsupported classes development action: %s", action)
	}
}

func parseNamedClassDevOptions(args []string, cwd string) (string, classDevOptions, error) {
	if len(args) == 0 || strings.HasPrefix(args[0], "--") {
		return "", classDevOptions{}, fmt.Errorf("class name is required")
	}
	name := filepath.Base(args[0])
	opts, err := parseClassDevOptions(args[1:], cwd, true)
	return name, opts, err
}

func parseClassDevOptions(args []string, cwd string, allowProjectPositional bool) (classDevOptions, error) {
	opts := classDevOptions{ProjectPath: cwd}
	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		switch {
		case arg == "--compiler-home" && i+1 < len(args):
			i++
			opts.CompilerHome = args[i]
		case strings.HasPrefix(arg, "--compiler-home="):
			opts.CompilerHome = strings.TrimPrefix(arg, "--compiler-home=")
		case arg == "--java-home" && i+1 < len(args):
			i++
			opts.JavaHome = args[i]
		case strings.HasPrefix(arg, "--java-home="):
			opts.JavaHome = strings.TrimPrefix(arg, "--java-home=")
		case arg == "--validation-evidence" && i+1 < len(args):
			i++
			opts.ValidationEvidence = args[i]
		case strings.HasPrefix(arg, "--validation-evidence="):
			opts.ValidationEvidence = strings.TrimPrefix(arg, "--validation-evidence=")
		case arg == "--project" && i+1 < len(args):
			i++
			opts.ProjectPath = args[i]
		case strings.HasPrefix(arg, "--project="):
			opts.ProjectPath = strings.TrimPrefix(arg, "--project=")
		case !strings.HasPrefix(arg, "--") && (allowProjectPositional || opts.ProjectPath == cwd):
			opts.ProjectPath = arg
		default:
			return classDevOptions{}, fmt.Errorf("unknown classes option: %s", arg)
		}
	}
	if abs, err := filepath.Abs(opts.ProjectPath); err == nil {
		opts.ProjectPath = abs
	}
	fillClassDevOptionsFromProject(&opts)
	return opts, nil
}

func fillClassDevOptionsFromProject(opts *classDevOptions) {
	root, err := config.Root(opts.ProjectPath)
	if err != nil {
		return
	}
	use := strings.TrimSpace(fmt.Sprint(root["use"]))
	active, _ := root[use].(map[string]any)
	classDev, _ := active["classDev"].(map[string]any)
	if opts.CompilerHome == "" {
		opts.CompilerHome = mapString(classDev, "compilerHome")
	}
	if opts.JavaHome == "" {
		opts.JavaHome = mapString(classDev, "javaHome")
	}
}

func discoverClassWorkflowGates(opts classDevOptions, env classDevEnvironment) classWorkflowGates {
	gates := classWorkflowGates{CompilerReady: env.Ready}
	root, err := config.Root(opts.ProjectPath)
	if err == nil {
		use := strings.TrimSpace(fmt.Sprint(root["use"]))
		active, _ := root[use].(map[string]any)
		gates.TargetGatewayConfigured = nonBlankMapValue(active, "classPublishUrl") || nonBlankMapValue(active, "baseUrl") || nonBlankMapValue(active, "CloudCCDev") || nonBlankMapValue(active, "setupSvc") || nonBlankMapValue(active, "apiSvc")
		gates.PublishAuthConfigured = nonBlankMapValue(active, "accessToken") || nonBlankMapValue(active, "CloudCCDev") || (nonBlankMapValue(active, "username") && nonBlankMapValue(active, "safetyMark") && nonBlankMapValue(active, "clientId") && nonBlankMapValue(active, "openSecretKey") && nonBlankMapValue(active, "orgId"))
	}
	if strings.TrimSpace(os.Getenv("CLOUDCC_CLASS_PUBLISH_URL")) != "" {
		gates.TargetGatewayConfigured = true
	}
	if strings.TrimSpace(os.Getenv("CLOUDCC_ACCESS_TOKEN")) != "" {
		gates.PublishAuthConfigured = true
	}
	gates.PublishReady = gates.CompilerReady && gates.TargetGatewayConfigured && gates.PublishAuthConfigured
	return gates
}

func nonBlankMapValue(values map[string]any, key string) bool {
	return strings.TrimSpace(mapString(values, key)) != ""
}

func mapString(values map[string]any, key string) string {
	if values == nil || values[key] == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(values[key]))
}

func discoverClassDevEnvironment(opts classDevOptions) classDevEnvironment {
	env := classDevEnvironment{}
	env.CompilerHome = discoverClassCompilerHome(opts)
	if env.CompilerHome != "" {
		env.CompilerHome, _ = filepath.Abs(env.CompilerHome)
		env.ManifestPath = filepath.Join(env.CompilerHome, "manifest.json")
		env.ClassesRoot = filepath.Join(env.CompilerHome, "WEB-INF", "classes")
	}
	env.JavaHome = firstNonEmptyPath(opts.JavaHome, os.Getenv("CLOUDCC_JAVA_HOME"), os.Getenv("JAVA_HOME"))
	if env.JavaHome == "" && runtime.GOOS == "darwin" {
		env.JavaHome = firstExistingPath(
			"/opt/homebrew/opt/openjdk@21/libexec/openjdk.jdk/Contents/Home",
			"/usr/local/opt/openjdk@21/libexec/openjdk.jdk/Contents/Home",
		)
	}
	if env.JavaHome != "" {
		env.Java = filepath.Join(env.JavaHome, "bin", executableName("java"))
		env.Javac = filepath.Join(env.JavaHome, "bin", executableName("javac"))
	}
	if !isExecutableFile(env.Java) {
		if found, err := exec.LookPath("java"); err == nil {
			env.Java = found
		}
	}
	if !isExecutableFile(env.Javac) {
		if found, err := exec.LookPath("javac"); err == nil {
			env.Javac = found
		}
	}
	if env.CompilerHome == "" || !isDir(env.CompilerHome) {
		env.Missing = append(env.Missing, "packaged class compiler bundle (assets/class-compiler)")
	} else {
		manifest, classpath, manifestDigest, problems := verifyClassCompilerBundle(env.CompilerHome)
		env.ManifestSHA256 = manifestDigest
		env.ArtifactCount = len(manifest.Artifacts)
		env.JarEntries = classpath
		env.Classpath = append([]string{env.ClassesRoot}, classpath...)
		env.ClasspathEntries = len(env.Classpath)
		env.CcegJar = filepath.Join(env.CompilerHome, "WEB-INF", "lib", "cceg.jar")
		if len(problems) > 0 {
			env.Missing = append(env.Missing, problems...)
		} else {
			env.IntegrityVerified = true
		}
	}
	if !isDir(env.ClassesRoot) {
		env.Missing = append(env.Missing, "compiler WEB-INF/classes root")
	}
	if !isExecutableFile(env.Java) {
		env.Missing = append(env.Missing, "JDK 21 java executable")
	}
	if !isExecutableFile(env.Javac) {
		env.Missing = append(env.Missing, "JDK 21 javac executable")
	}
	if env.CcegJar == "" || !isExecutableFile(env.CcegJar) {
		env.Missing = append(env.Missing, "bundled WEB-INF/lib/cceg.jar")
	}
	if isExecutableFile(env.Java) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		output, err := exec.CommandContext(ctx, env.Java, "-version").CombinedOutput()
		env.JavaVersion = firstLine(string(output))
		if err != nil {
			env.Missing = append(env.Missing, "runnable JDK: "+err.Error())
		} else if !regexp.MustCompile(`(?:version\s+"21|openjdk\s+21)`).MatchString(strings.ToLower(string(output))) {
			env.Missing = append(env.Missing, "JDK 21 required; discovered "+env.JavaVersion)
		}
	}
	env.Ready = len(env.Missing) == 0 && env.IntegrityVerified
	return env
}

func discoverClassCompilerHome(opts classDevOptions) string {
	for _, candidate := range []string{opts.CompilerHome, os.Getenv("CLOUDCC_CLASS_COMPILER_HOME")} {
		if isDir(candidate) {
			return candidate
		}
	}
	if executable, err := os.Executable(); err == nil {
		if resolved, resolveErr := filepath.EvalSymlinks(executable); resolveErr == nil {
			executable = resolved
		}
		candidate := filepath.Clean(filepath.Join(filepath.Dir(executable), "..", "..", "assets", "class-compiler"))
		if isDir(candidate) {
			return candidate
		}
	}
	for _, start := range []string{opts.ProjectPath, currentWorkingDirectory()} {
		if candidate := findClassCompilerHome(start); candidate != "" {
			return candidate
		}
	}
	return ""
}

func findClassCompilerHome(start string) string {
	current, err := filepath.Abs(start)
	if err != nil {
		return ""
	}
	for {
		for _, candidate := range []string{
			filepath.Join(current, "assets", "class-compiler"),
			filepath.Join(current, "cc-customization-expert-go", "assets", "class-compiler"),
		} {
			if isDir(candidate) {
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

func verifyClassCompilerBundle(home string) (classCompilerManifest, []string, string, []string) {
	manifestPath := filepath.Join(home, "manifest.json")
	b, err := os.ReadFile(manifestPath)
	if err != nil {
		return classCompilerManifest{}, nil, "", []string{"compiler manifest.json: " + err.Error()}
	}
	manifestDigest := sourceDigest(string(b))
	var manifest classCompilerManifest
	if err := json.Unmarshal(b, &manifest); err != nil {
		return manifest, nil, manifestDigest, []string{"invalid compiler manifest.json: " + err.Error()}
	}
	var problems []string
	if manifest.FormatVersion != 1 {
		problems = append(problems, fmt.Sprintf("unsupported compiler manifest format %d", manifest.FormatVersion))
	}
	if manifest.JavaTarget != 21 {
		problems = append(problems, fmt.Sprintf("compiler manifest Java target is %d, expected 21", manifest.JavaTarget))
	}
	if manifest.ArtifactCount != len(manifest.Artifacts) || len(manifest.Artifacts) == 0 {
		problems = append(problems, "compiler manifest artifact count mismatch")
	}
	seen := map[string]bool{}
	var classpath []string
	for _, artifact := range manifest.Artifacts {
		clean := filepath.Clean(filepath.FromSlash(artifact.Path))
		if filepath.IsAbs(clean) || clean == "." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || seen[clean] {
			problems = append(problems, "invalid compiler artifact path: "+artifact.Path)
			continue
		}
		seen[clean] = true
		path := filepath.Join(home, clean)
		info, err := os.Lstat(path)
		if err != nil {
			problems = append(problems, "missing compiler artifact: "+artifact.Path)
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			problems = append(problems, "compiler artifact must be packaged, not symlinked: "+artifact.Path)
			continue
		}
		if info.Size() != artifact.Size {
			problems = append(problems, "compiler artifact size mismatch: "+artifact.Path)
			continue
		}
		contents, readErr := os.ReadFile(path)
		if readErr != nil {
			problems = append(problems, "cannot read compiler artifact: "+artifact.Path)
			continue
		}
		digest := sha256.Sum256(contents)
		if !strings.EqualFold(hex.EncodeToString(digest[:]), artifact.SHA256) {
			problems = append(problems, "compiler artifact SHA-256 mismatch: "+artifact.Path)
			continue
		}
		classpath = append(classpath, path)
	}
	sort.Strings(classpath)
	return manifest, classpath, manifestDigest, problems
}

func classPublishBaseURL(cfg config.Config) (string, error) {
	explicit := firstNonBlankString(strings.TrimSpace(os.Getenv("CLOUDCC_CLASS_PUBLISH_URL")), config.String(cfg, "classPublishUrl"), config.String(cfg, "setupSvc"))
	if explicit != "" {
		return validateClassPublishURL(explicit)
	}
	for _, candidate := range []string{config.String(cfg, "baseUrl"), config.String(cfg, "apiSvc")} {
		if strings.TrimSpace(candidate) == "" {
			continue
		}
		u, err := url.Parse(strings.TrimSpace(candidate))
		if err != nil || u.Scheme == "" || u.Host == "" {
			continue
		}
		path := strings.TrimRight(u.Path, "/")
		switch {
		case strings.HasSuffix(path, "/ccdomaingateway/setup") || strings.HasSuffix(path, "/setup"):
		case strings.HasSuffix(path, "/ccdomaingateway/apisvc"):
			path = strings.TrimSuffix(path, "/apisvc") + "/setup"
		case strings.HasSuffix(path, "/ccdomaingateway"):
			path += "/setup"
		case strings.HasSuffix(path, "/apisvc"):
			path = strings.TrimSuffix(path, "/apisvc") + "/setup"
		case path == "":
			path = "/ccdomaingateway/setup"
		default:
			path += "/ccdomaingateway/setup"
		}
		u.Path = path
		u.RawPath = ""
		u.RawQuery = ""
		u.Fragment = ""
		return validateClassPublishURL(u.String())
	}
	return "", fmt.Errorf("target CloudCC class publish gateway is missing; configure CloudCCDev/baseUrl or classPublishUrl")
}

func validateClassPublishURL(value string) (string, error) {
	u, err := url.Parse(strings.TrimRight(strings.TrimSpace(value), "/"))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", fmt.Errorf("invalid CloudCC class publish gateway URL: %s", value)
	}
	return strings.TrimRight(u.String(), "/"), nil
}

func validateClass(name string, opts classDevOptions) (classValidationResult, error) {
	sourceFile := filepath.Join(opts.ProjectPath, "backend", "classes", name, name+".java")
	source, err := readMarkedSource(sourceFile)
	result := classValidationResult{
		Status:           "failed",
		ClassName:        name,
		SourceFile:       sourceFile,
		TemplateClass:    "com.cloudcc.core.cls.fag.FagTemplate",
		GeneratedPackage: "com.cloudcc.core.fag.CLOCAL",
		Diagnostics:      []classCompileDiagnostic{},
	}
	if err != nil {
		return result, err
	}
	source = strings.TrimSpace(source)
	result.SourceSHA256 = sourceDigest(source)
	result.PolicyViolations = classSourcePolicyViolations(source)
	if !hasUserInfoConstructor(source, name) {
		result.PolicyViolations = append(result.PolicyViolations, "public "+name+"(UserInfo userInfo) constructor is required by the CloudCC class invoker")
	}
	if len(result.PolicyViolations) > 0 {
		return result, fmt.Errorf("class source violates CloudCC policy: %s", strings.Join(result.PolicyViolations, "; "))
	}
	env := discoverClassDevEnvironment(opts)
	result.CompilerHome = env.CompilerHome
	result.CompilerManifest = env.ManifestSHA256
	result.JavaVersion = env.JavaVersion
	result.ClasspathEntries = len(env.Classpath)
	if !env.Ready {
		return result, fmt.Errorf("standalone CloudCC class compiler is not ready: %s", strings.Join(env.Missing, "; "))
	}
	workDir, err := os.MkdirTemp("", "cloudcc-class-validate-*")
	if err != nil {
		return result, err
	}
	defer os.RemoveAll(workDir)
	rendererFile := filepath.Join(workDir, "CloudCCClassTemplateRenderer.java")
	inputFile := filepath.Join(workDir, "source.java.fragment")
	generatedFile := filepath.Join(workDir, name+".java")
	classesDir := filepath.Join(workDir, "classes")
	if err := os.MkdirAll(classesDir, 0o755); err != nil {
		return result, err
	}
	if err := os.WriteFile(rendererFile, []byte(classTemplateRendererSource), 0o600); err != nil {
		return result, err
	}
	if err := os.WriteFile(inputFile, []byte(source), 0o600); err != nil {
		return result, err
	}
	classpath := strings.Join(env.Classpath, string(os.PathListSeparator))
	if output, runErr := runCommand(env.Javac, "-encoding", "UTF-8", "-classpath", classpath, "-d", classesDir, rendererFile); runErr != nil {
		result.CompilationOutput = strings.TrimSpace(output)
		return result, fmt.Errorf("cannot compile packaged CloudCC template renderer: %w", runErr)
	}
	runtimeEntries := append([]string{env.ClassesRoot, classesDir}, env.JarEntries...)
	if output, runErr := runCommand(env.Java, "-classpath", strings.Join(runtimeEntries, string(os.PathListSeparator)), "CloudCCClassTemplateRenderer", inputFile, generatedFile, result.GeneratedPackage); runErr != nil {
		result.CompilationOutput = strings.TrimSpace(output)
		return result, fmt.Errorf("cannot render packaged CloudCC FagTemplate: %w", runErr)
	}
	generated, err := os.ReadFile(generatedFile)
	if err != nil {
		return result, err
	}
	sourceOffset := bytes.Index(generated, []byte(source))
	if sourceOffset < 0 {
		return result, fmt.Errorf("rendered FagTemplate does not contain the original class source")
	}
	sourceStartLine := 1 + strings.Count(string(generated[:sourceOffset]), "\n")
	compilerEntries := append([]string{classesDir, env.ClassesRoot}, env.JarEntries...)
	output, compileErr := runCommand(env.Javac, "-encoding", "UTF-8", "-XDuseUnsharedTable", "-classpath", strings.Join(compilerEntries, string(os.PathListSeparator)), "-d", classesDir, generatedFile)
	result.CompilationOutput = strings.TrimSpace(output)
	result.Diagnostics = parseJavacDiagnostics(output, generatedFile, sourceStartLine, strings.Count(source, "\n")+1)
	if compileErr != nil {
		if len(result.Diagnostics) == 0 {
			result.Diagnostics = append(result.Diagnostics, classCompileDiagnostic{Kind: "error", Message: strings.TrimSpace(output)})
		}
		return result, fmt.Errorf("CloudCC class compilation failed")
	}
	result.Valid = true
	result.Status = "passed"
	result.CompilationOutput = ""
	return result, nil
}

func classSourcePolicyViolations(source string) []string {
	var violations []string
	blockedImports := []string{
		"com.cloudcc.core.service.CCCached",
		"com.cloudcc.database.redis.RedisDB",
		"com.cloudcc.database.redis.*",
		"com.cloudcc.core.service.*",
	}
	for _, blocked := range blockedImports {
		if strings.Contains(source, blocked) {
			violations = append(violations, "blocked platform import: "+blocked)
		}
	}
	if strings.Contains(source, "DBMan") {
		violations = append(violations, "DBMan is rejected by the CloudCC class template")
	}
	if strings.Contains(source, "System.exit(") {
		violations = append(violations, "System.exit is not allowed")
	}
	if regexp.MustCompile(`(?s)for\s*\(.*;\s*(?:true)?\s*;.*\)`).MatchString(source) || regexp.MustCompile(`while\s*\(\s*true\s*\)`).MatchString(source) {
		violations = append(violations, "unbounded for/while loop is rejected by CloudCC")
	}
	return violations
}

func hasUserInfoConstructor(source string, className string) bool {
	pattern := `(?m)public\s+` + regexp.QuoteMeta(className) + `\s*\(\s*(?:com\.g3cloud\.common\.)?UserInfo\s+[A-Za-z_$][A-Za-z0-9_$]*\s*\)`
	return regexp.MustCompile(pattern).MatchString(source)
}

func parseJavacDiagnostics(output string, generatedFile string, sourceStartLine int, sourceLineCount int) []classCompileDiagnostic {
	pattern := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(generatedFile) + `:(\d+):\s*(error|warning):\s*(.+)$`)
	matches := pattern.FindAllStringSubmatch(output, -1)
	result := make([]classCompileDiagnostic, 0, len(matches))
	for _, match := range matches {
		line, _ := strconv.Atoi(match[1])
		diagnostic := classCompileDiagnostic{Kind: match[2], Message: strings.TrimSpace(match[3]), GeneratedLine: line}
		if line >= sourceStartLine && line < sourceStartLine+sourceLineCount {
			diagnostic.SourceLine = line - sourceStartLine + 1
		}
		result = append(result, diagnostic)
	}
	return result
}

func sourceDigest(source string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(strings.ReplaceAll(source, "\r\n", "\n"))))
	return hex.EncodeToString(sum[:])
}

func runCommand(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(os.Environ(), "LC_ALL=C", "LANG=C")
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return string(output), fmt.Errorf("command timed out: %s", name)
	}
	return string(output), err
}

func writeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func firstExistingPath(paths ...string) string {
	for _, path := range paths {
		if path != "" {
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}
	return ""
}

func firstNonEmptyPath(paths ...string) string {
	for _, path := range paths {
		if strings.TrimSpace(path) != "" {
			return strings.TrimSpace(path)
		}
	}
	return ""
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func isExecutableFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func executableName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func firstLine(value string) string {
	if idx := strings.IndexByte(value, '\n'); idx >= 0 {
		return strings.TrimSpace(value[:idx])
	}
	return strings.TrimSpace(value)
}

func currentWorkingDirectory() string {
	cwd, _ := os.Getwd()
	return cwd
}
