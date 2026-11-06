package expressionsyntax

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"cloudcc-customization-expert-go/internal/config"
	"cloudcc-customization-expert-go/internal/httpclient"
)

const (
	DiscoveryPath      = "/api/validateRule/getFieldSuffix"
	RelationshipSuffix = "__r"
	RequestHeader      = "X-CloudCC-Expression-Field-Suffix"
	cacheSchemaVersion = "cloudcc-expression-syntax-cache/v1"
	defaultCacheTTL    = time.Hour
)

var suffixPattern = regexp.MustCompile(`^__[A-Za-z][A-Za-z0-9_]*$`)
var now = time.Now

type Syntax struct {
	FieldSuffix                string `json:"fieldSuffix"`
	RelationshipSuffix         string `json:"relationshipSuffix"`
	ResolvedAt                 int64  `json:"resolvedAt"`
	ExpiresAt                  int64  `json:"expiresAt"`
	MetadataAPINamesHaveSuffix bool   `json:"metadataApiNamesHaveSuffix"`
}

type cacheEntry struct {
	SchemaVersion string `json:"schemaVersion"`
	Syntax
}

type Fetcher func() (Syntax, error)

var globalCache = struct {
	sync.Mutex
	values map[string]Syntax
}{values: map[string]Syntax{}}

func ResolveUIAPI(projectPath string) (Syntax, error) {
	return resolve(projectPath, func(cfg config.Config) (Syntax, error) {
		setupSvc := strings.TrimRight(config.SetupServiceRequestTarget(cfg), "/")
		if setupSvc == "" {
			return Syntax{}, unavailable("the active target is incomplete")
		}
		token := strings.TrimSpace(config.String(cfg, "accessToken"))
		if token == "" {
			return Syntax{}, unavailable("the active target is not authenticated")
		}
		var response any
		if err := httpclient.New().PostClass(setupSvc+DiscoveryPath, map[string]any{}, token, &response); err != nil {
			return Syntax{}, unavailable("target recognition failed: " + err.Error())
		}
		return syntaxFromValue(response, now()), nil
	})
}

// Resolve is retained for callers that explicitly use the UIAPI discovery path.
func Resolve(projectPath string) (Syntax, error) { return ResolveUIAPI(projectPath) }

// ResolveMSAPI uses a MetadataService-backed fetcher on a cache miss so the
// client's normal target, authentication, and token-refresh behavior remains
// authoritative.
func ResolveMSAPI(projectPath string, fetcher Fetcher) (Syntax, error) {
	if fetcher == nil {
		return Syntax{}, unavailable("MetadataService target recognition is unavailable")
	}
	return resolve(projectPath, func(config.Config) (Syntax, error) { return fetcher() })
}

func resolve(projectPath string, fetch func(config.Config) (Syntax, error)) (Syntax, error) {
	cfg, err := config.Load(projectPath)
	if err != nil {
		return Syntax{}, err
	}
	key, err := targetFingerprint(cfg)
	if err != nil {
		return Syntax{}, err
	}
	current := now()
	globalCache.Lock()
	if cached, ok := globalCache.values[key]; ok && validAt(cached, current) {
		globalCache.Unlock()
		return cached, nil
	}
	globalCache.Unlock()
	if cached, ok := loadDiskCache(key, current); ok {
		globalCache.Lock()
		globalCache.values[key] = cached
		globalCache.Unlock()
		return cached, nil
	}

	resolved, err := fetch(cfg)
	if err != nil {
		return Syntax{}, err
	}
	resolved, err = normalizeSyntax(resolved, current)
	if err != nil {
		return Syntax{}, err
	}
	if err := saveDiskCache(key, resolved); err != nil {
		return Syntax{}, fmt.Errorf("expression_field_suffix_cache_failed: %w", err)
	}
	globalCache.Lock()
	globalCache.values[key] = resolved
	globalCache.Unlock()
	return resolved, nil
}

func syntaxFromValue(value any, fetched time.Time) Syntax {
	resolvedAt := int64Value(value, "resolvedAt")
	expiresAt := int64Value(value, "expiresAt")
	if resolvedAt <= 0 {
		resolvedAt = fetched.UnixMilli()
	}
	if expiresAt <= resolvedAt {
		expiresAt = fetched.Add(defaultCacheTTL).UnixMilli()
	}
	return Syntax{FieldSuffix: suffixFromResponse(value), RelationshipSuffix: RelationshipSuffix,
		ResolvedAt: resolvedAt, ExpiresAt: expiresAt, MetadataAPINamesHaveSuffix: false}
}

func ParseRemote(value map[string]any) (Syntax, error) {
	resolvedAt := int64Value(value, "resolvedAt")
	expiresAt := int64Value(value, "expiresAt")
	if resolvedAt <= 0 || expiresAt <= resolvedAt {
		return Syntax{}, fmt.Errorf("expression_field_suffix_invalid: MetadataService returned invalid cache timestamps")
	}
	return normalizeSyntax(Syntax{
		FieldSuffix:        suffixFromResponse(value),
		RelationshipSuffix: RelationshipSuffix,
		ResolvedAt:         resolvedAt,
		ExpiresAt:          expiresAt,
	}, now())
}

func normalizeSyntax(value Syntax, current time.Time) (Syntax, error) {
	value.FieldSuffix = strings.TrimSpace(value.FieldSuffix)
	if !suffixPattern.MatchString(value.FieldSuffix) {
		return Syntax{}, fmt.Errorf("expression_field_suffix_invalid: target returned an invalid expression field suffix")
	}
	if strings.EqualFold(value.FieldSuffix, RelationshipSuffix) {
		return Syntax{}, fmt.Errorf("expression_field_suffix_invalid: target returned the reserved relationship suffix")
	}
	value.RelationshipSuffix = RelationshipSuffix
	value.MetadataAPINamesHaveSuffix = false
	if value.ResolvedAt <= 0 || value.ExpiresAt <= value.ResolvedAt {
		return Syntax{}, fmt.Errorf("expression_field_suffix_invalid: cache timestamps are invalid")
	}
	if !validAt(value, current) {
		return Syntax{}, unavailable("target recognition returned an expired value")
	}
	return value, nil
}

func suffixFromResponse(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case map[string]any:
		for _, key := range []string{"data", "fieldSuffix", "suffix", "value"} {
			if suffix := suffixFromResponse(typed[key]); suffix != "" {
				return suffix
			}
		}
	case []any:
		if len(typed) == 1 {
			return suffixFromResponse(typed[0])
		}
	}
	return ""
}

func int64Value(value any, key string) int64 {
	if object, ok := value.(map[string]any); ok {
		if nested, ok := object["data"].(map[string]any); ok {
			if result := int64Value(nested, key); result > 0 {
				return result
			}
		}
		switch typed := object[key].(type) {
		case float64:
			return int64(typed)
		case json.Number:
			result, _ := typed.Int64()
			return result
		case int64:
			return typed
		case int:
			return int64(typed)
		}
	}
	return 0
}

func targetFingerprint(cfg config.Config) (string, error) {
	target := strings.TrimRight(config.SetupServiceRequestTarget(cfg), "/")
	if target == "" {
		return "", unavailable("the active target is incomplete")
	}
	identity := firstNonBlank(config.String(cfg, "orgId"), config.String(cfg, "organizationId"),
		config.String(cfg, "tenantId"))
	if identity == "" {
		token := strings.TrimSpace(config.String(cfg, "accessToken"))
		if token == "" {
			return "", unavailable("the active target identity is unavailable")
		}
		identity = digest(token)
	}
	return digest(strings.ToLower(target) + "|" + identity), nil
}

func cacheRoot() (string, error) {
	if override := strings.TrimSpace(os.Getenv("CLOUDCC_EXPRESSION_SYNTAX_CACHE_DIR")); override != "" {
		return override, nil
	}
	root, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "cloudcc-cli", "expression-syntax"), nil
}

func loadDiskCache(key string, current time.Time) (Syntax, bool) {
	root, err := cacheRoot()
	if err != nil {
		return Syntax{}, false
	}
	content, err := os.ReadFile(filepath.Join(root, key+".json"))
	if err != nil {
		return Syntax{}, false
	}
	var entry cacheEntry
	if json.Unmarshal(content, &entry) != nil || entry.SchemaVersion != cacheSchemaVersion {
		return Syntax{}, false
	}
	value, err := normalizeSyntax(entry.Syntax, current)
	return value, err == nil
}

func saveDiskCache(key string, value Syntax) error {
	root, err := cacheRoot()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return err
	}
	encoded, err := json.Marshal(cacheEntry{SchemaVersion: cacheSchemaVersion, Syntax: value})
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(root, ".expression-syntax-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(encoded); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	finalPath := filepath.Join(root, key+".json")
	if err := os.Rename(temporaryPath, finalPath); err != nil {
		_ = os.Remove(finalPath)
		if err := os.Rename(temporaryPath, finalPath); err != nil {
			return err
		}
	}
	return os.Chmod(finalPath, 0o600)
}

func validAt(value Syntax, current time.Time) bool { return value.ExpiresAt > current.UnixMilli() }

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func unavailable(message string) error {
	return fmt.Errorf("expression_field_suffix_unavailable: %s", message)
}

// RequiresTargetSyntax is intentionally limited to formula fields, validation
// rules, and workflow formula conditions.
func RequiresTargetSyntax(action string, resource string, args []string, cwd string) bool {
	action = strings.ToLower(strings.TrimSpace(action))
	resource = normalizeResource(resource)
	values := decodedArguments(args, cwd)
	if resource == "fields" && (action == "delete" || action == "remove" || action == "impact") {
		return true
	}
	if !isExpressionAction(action) {
		return false
	}
	switch resource {
	case "validationrule":
		return containsExpressionKey(values) || positionalValidationFormula(args)
	case "fields":
		return containsFormulaField(values)
	case "workflow":
		return containsWorkflowFormula(values)
	default:
		return false
	}
}

func positionalValidationFormula(args []string) bool {
	if len(args) < 5 {
		return false
	}
	for _, arg := range args[1:] {
		trimmed := strings.TrimSpace(arg)
		if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "@") {
			return false
		}
	}
	return strings.TrimSpace(args[3]) != ""
}

func RequiresPlanBody(body map[string]any) bool {
	domain := normalizeResource(strings.TrimSpace(fmt.Sprint(body["domain"])))
	operation := strings.ToLower(strings.TrimSpace(fmt.Sprint(body["operation"])))
	if operation == "" || operation == "<nil>" {
		operation = strings.ToLower(strings.TrimSpace(fmt.Sprint(body["mutation"])))
	}
	spec := body["spec"]
	if spec == nil {
		spec = body["patch"]
	}
	switch domain {
	case "validationrule":
		return containsExpressionKey(spec)
	case "fields":
		return operation == "delete" || operation == "remove" || operation == "impact" || containsFormulaField(spec)
	case "workflow":
		return containsWorkflowFormula(spec)
	default:
		return false
	}
}

func ProjectPath(args []string, cwd string) string {
	if len(args) == 0 {
		return cwd
	}
	first := strings.TrimSpace(args[0])
	if first == "" || strings.HasPrefix(first, "{") || strings.HasPrefix(first, "[") || strings.HasPrefix(first, "@") {
		return cwd
	}
	if decoded, err := url.QueryUnescape(first); err == nil && (strings.HasPrefix(decoded, "{") || strings.HasPrefix(decoded, "[")) {
		return cwd
	}
	return first
}

func isExpressionAction(action string) bool {
	switch action {
	case "create", "update", "upsert", "save", "plan", "validate", "normalize", "mutate":
		return true
	default:
		return false
	}
}

func normalizeResource(value string) string {
	value = strings.ToLower(strings.NewReplacer("-", "", "_", "").Replace(strings.TrimSpace(value)))
	switch value {
	case "field":
		return "fields"
	case "validationrules", "validaterule":
		return "validationrule"
	case "workflows", "workflowrule":
		return "workflow"
	default:
		return value
	}
}

func decodedArguments(args []string, cwd string) []any {
	values := make([]any, 0, len(args))
	for _, arg := range args {
		raw := strings.TrimSpace(arg)
		if strings.HasPrefix(raw, "@") && len(raw) > 1 {
			path := strings.TrimSpace(strings.TrimPrefix(raw, "@"))
			if !filepath.IsAbs(path) {
				path = filepath.Join(cwd, path)
			}
			if content, err := os.ReadFile(path); err == nil {
				raw = string(content)
			}
		} else if decoded, err := url.QueryUnescape(raw); err == nil {
			raw = decoded
		}
		var value any
		if json.Unmarshal([]byte(raw), &value) == nil {
			values = append(values, value)
		}
	}
	return values
}

func containsExpressionKey(value any) bool {
	switch typed := value.(type) {
	case []any:
		for _, child := range typed {
			if containsExpressionKey(child) {
				return true
			}
		}
	case map[string]any:
		for key, child := range typed {
			switch strings.ToLower(key) {
			case "formula", "functioncode", "rulecontent", "expression":
				if strings.TrimSpace(fmt.Sprint(child)) != "" {
					return true
				}
			}
			if containsExpressionKey(child) {
				return true
			}
		}
	}
	return false
}

func containsFormulaField(value any) bool {
	switch typed := value.(type) {
	case []any:
		for _, child := range typed {
			if containsFormulaField(child) {
				return true
			}
		}
	case map[string]any:
		formulaType := false
		formulaText := false
		for key, child := range typed {
			lower := strings.ToLower(key)
			if (lower == "type" || lower == "fieldtype" || lower == "fdtype") && strings.EqualFold(strings.TrimSpace(fmt.Sprint(child)), "Z") {
				formulaType = true
			}
			if (lower == "formulatext" || lower == "formula") && strings.TrimSpace(fmt.Sprint(child)) != "" {
				formulaText = true
			}
		}
		if formulaType || formulaText {
			return true
		}
		for _, child := range typed {
			if containsFormulaField(child) {
				return true
			}
		}
	}
	return false
}

func containsWorkflowFormula(value any) bool {
	switch typed := value.(type) {
	case []any:
		for _, child := range typed {
			if containsWorkflowFormula(child) {
				return true
			}
		}
	case map[string]any:
		formulaMode := false
		formulaValue := false
		for key, child := range typed {
			lower := strings.ToLower(key)
			if (lower == "conditionoption" || lower == "condtionoption" || lower == "mode") &&
				(strings.EqualFold(strings.TrimSpace(fmt.Sprint(child)), "F") || strings.EqualFold(strings.TrimSpace(fmt.Sprint(child)), "formula")) {
				formulaMode = true
			}
			if (lower == "functioncode" || lower == "formula" || lower == "expression") && strings.TrimSpace(fmt.Sprint(child)) != "" {
				formulaValue = true
			}
		}
		if formulaMode || formulaValue {
			return true
		}
		for _, child := range typed {
			if containsWorkflowFormula(child) {
				return true
			}
		}
	}
	return false
}

func ResetCacheForTests() {
	globalCache.Lock()
	globalCache.values = map[string]Syntax{}
	globalCache.Unlock()
}

// Invalidate removes the cache entry for the active target after MetadataService
// reports that the target configuration changed.
func Invalidate(projectPath string) {
	cfg, err := config.Load(projectPath)
	if err != nil {
		return
	}
	key, err := targetFingerprint(cfg)
	if err != nil {
		return
	}
	globalCache.Lock()
	delete(globalCache.values, key)
	globalCache.Unlock()
	if root, rootErr := cacheRoot(); rootErr == nil {
		_ = os.Remove(filepath.Join(root, key+".json"))
	}
}
