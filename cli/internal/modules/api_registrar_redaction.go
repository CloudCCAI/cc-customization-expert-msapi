package modules

import (
	"encoding/json"
	"regexp"
	"strings"
)

const apiRegistrarRedactedValue = "<redacted>"

var apiRegistrarSecretTextPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(authorization\s*[:=]\s*)(bearer\s+)?[A-Za-z0-9._~+/=-]+`),
	regexp.MustCompile(`(?i)(access[_-]?token\s*[:=]\s*)[A-Za-z0-9._~+/=-]+`),
	regexp.MustCompile(`(?i)(refresh[_-]?token\s*[:=]\s*)[A-Za-z0-9._~+/=-]+`),
	regexp.MustCompile(`(?i)(id[_-]?token\s*[:=]\s*)[A-Za-z0-9._~+/=-]+`),
	regexp.MustCompile(`(?i)(token=)[^&\s"'<>]+`),
	regexp.MustCompile(`(?i)(secret=)[^&\s"'<>]+`),
	regexp.MustCompile(`(?i)(password=)[^&\s"'<>]+`),
}

func redactApiRegistrarLogValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			if isApiRegistrarSensitiveKey(key) {
				out[key] = apiRegistrarRedactedValue
				continue
			}
			out[key] = redactApiRegistrarLogValue(child)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, child := range typed {
			out[i] = redactApiRegistrarLogValue(child)
		}
		return out
	case string:
		return redactApiRegistrarLogString(typed)
	default:
		return value
	}
}

func isApiRegistrarSensitiveKey(key string) bool {
	normalized := strings.ToLower(strings.NewReplacer("_", "", "-", "", " ", "").Replace(strings.TrimSpace(key)))
	if normalized == "" {
		return false
	}
	sensitiveFragments := []string{
		"authorization",
		"cookie",
		"accesstoken",
		"refreshtoken",
		"idtoken",
		"token",
		"password",
		"passwd",
		"secret",
		"apikey",
		"appkey",
		"clientsecret",
		"opensecretkey",
		"safetymark",
	}
	for _, fragment := range sensitiveFragments {
		if strings.Contains(normalized, fragment) {
			return true
		}
	}
	return false
}

func redactApiRegistrarLogString(value string) string {
	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		var decoded any
		if err := json.Unmarshal([]byte(value), &decoded); err == nil {
			if encoded, err := json.Marshal(redactApiRegistrarLogValue(decoded)); err == nil {
				return string(encoded)
			}
		}
	}
	redacted := value
	for _, pattern := range apiRegistrarSecretTextPatterns {
		redacted = pattern.ReplaceAllString(redacted, "${1}"+apiRegistrarRedactedValue)
	}
	return redacted
}
