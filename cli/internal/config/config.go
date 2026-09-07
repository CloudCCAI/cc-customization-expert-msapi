package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloudcc-customization-expert-go/internal/httpclient"
	"cloudcc-customization-expert-go/internal/jsonx"
)

const defaultBaseURL = "https://developer.apis.cloudcc.cn"

type Config map[string]any

func Load(projectPath string) (Config, error) {
	if projectPath == "" {
		var err error
		projectPath, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}
	if _, err := os.Stat(filepath.Join(projectPath, "cloudcc-cli.config.json")); err == nil {
		if cached, err := loadCache(projectPath); err == nil && cached != nil {
			return cached, nil
		}
		cfg, err := loadJSONConfig(projectPath)
		if err == nil && cfg != nil {
			if old, oldErr := loadOldPackage(projectPath); oldErr == nil && old != nil {
				mergeMissingConfig(cfg, old)
			}
			return resolveDevConsoleConfig(projectPath, cfg, false)
		}
		if old, oldErr := loadOldPackage(projectPath); oldErr == nil && old != nil {
			return resolveDevConsoleConfig(projectPath, old, false)
		}
	} else if old, err := loadOldPackage(projectPath); err == nil && old != nil {
		return resolveDevConsoleConfig(projectPath, old, false)
	}
	if _, err := os.Stat(filepath.Join(projectPath, "cloudcc-cli.config.js")); err == nil {
		return nil, fmt.Errorf("cloudcc-cli.config.js is not executable by the Go CLI; migrate to cloudcc-cli.config.json")
	}
	return nil, fmt.Errorf("no valid cloudcc-cli config found in %s", projectPath)
}

func RefreshAccessToken(projectPath string) (Config, error) {
	if projectPath == "" {
		var err error
		projectPath, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}
	_ = ClearCacheEntry(projectPath)
	if cfg, err := loadJSONConfig(projectPath); err == nil && cfg != nil {
		if old, oldErr := loadOldPackage(projectPath); oldErr == nil && old != nil {
			mergeMissingConfig(cfg, old)
		}
		return resolveDevConsoleConfig(projectPath, cfg, true)
	}
	if old, err := loadOldPackage(projectPath); err == nil && old != nil {
		return resolveDevConsoleConfig(projectPath, old, true)
	}
	return nil, fmt.Errorf("CloudCC accessToken refresh failed before /api/cauth/token: no refreshable CloudCC credentials found; check cloudcc-cli.config.json active env for CloudCCDev or username/safetyMark/clientId/openSecretKey/orgId/apiSvc")
}

func Use(projectPath string, env string) error {
	if env == "" {
		return fmt.Errorf("env is required")
	}
	file := filepath.Join(projectPath, "cloudcc-cli.config.json")
	root, err := jsonx.ReadObjectFile(file)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", file, err)
	}
	root["use"] = env
	return jsonx.WriteObjectFile(file, root)
}

func Root(projectPath string) (map[string]any, error) {
	file := filepath.Join(projectPath, "cloudcc-cli.config.json")
	return jsonx.ReadObjectFile(file)
}

func loadOldPackage(projectPath string) (Config, error) {
	file := filepath.Join(projectPath, "package.json")
	b, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var root map[string]any
	if err := json.Unmarshal(b, &root); err != nil {
		return nil, err
	}
	if cfg, ok := root["devConsoleConfig"].(map[string]any); ok {
		return Config(cfg), nil
	}
	return nil, nil
}

func loadJSONConfig(projectPath string) (Config, error) {
	root, err := Root(projectPath)
	if err != nil {
		return nil, err
	}
	use, _ := root["use"].(string)
	if use == "" {
		return nil, fmt.Errorf("cloudcc-cli.config.json missing use")
	}
	cfg, ok := root[use].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("cloudcc-cli.config.json missing env %s", use)
	}
	out := map[string]any{}
	for k, v := range cfg {
		out[k] = v
	}
	if cloudDev, _ := out["CloudCCDev"].(string); cloudDev != "" {
		decoded, err := decodeCloudCCDev(cloudDev)
		if err != nil {
			out["CloudCCDevDecodeError"] = err.Error()
			return Config(out), nil
		}
		for k, v := range decoded {
			if _, exists := out[k]; !exists {
				out[k] = v
			}
		}
		out["CloudCCDev"] = ""
	}
	return Config(out), nil
}

func mergeMissingConfig(dst Config, src Config) {
	for key, value := range src {
		if stringValue(dst[key]) == "" {
			dst[key] = value
		}
	}
}

func decodeCloudCCDev(value string) (map[string]any, error) {
	b, err := base64.StdEncoding.DecodeString(strings.TrimSpace(value))
	if err != nil {
		return nil, fmt.Errorf("CloudCCDev could not be decoded as base64 JSON: %w", err)
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("CloudCCDev is not JSON: %w", err)
	}
	if base, _ := out["baseUrl"].(string); base != "" && !strings.Contains(base, "ccdomaingateway") {
		out["baseUrl"] = strings.TrimRight(base, "/") + "/ccdomaingateway"
	}
	return out, nil
}

func loadCache(projectPath string) (Config, error) {
	root, err := Root(projectPath)
	if err != nil {
		return nil, err
	}
	use, _ := root["use"].(string)
	active, _ := root[use].(map[string]any)
	if active == nil {
		return nil, nil
	}
	key := stringValue(active["safetyMark"])
	if key == "" {
		key = stringValue(active["secretKey"])
	}
	if key == "" {
		return nil, nil
	}
	cache, err := readCache(projectPath)
	if err != nil {
		return nil, nil
	}
	entry, _ := cache[key].(map[string]any)
	if entry == nil {
		return nil, nil
	}
	ts, ok := entry["timestamp"].(float64)
	if !ok {
		return nil, nil
	}
	if time.Since(time.UnixMilli(int64(ts))) > time.Hour {
		return nil, nil
	}
	if tokenNearExpiry(stringValue(entry["accessToken"]), 5*time.Minute) {
		return nil, nil
	}
	return Config(entry), nil
}

func resolveDevConsoleConfig(projectPath string, cfg Config, forceAccessTokenRefresh bool) (Config, error) {
	if cfg["apiSvc"] == nil || cfg["setupSvc"] == nil {
		if err := addBaseURLs(cfg); err != nil {
			return nil, err
		}
	}
	client := httpclient.New()
	if err := addBusToken(client, cfg, forceAccessTokenRefresh); err != nil {
		return nil, err
	}
	if err := addSecretKey(client, cfg); err != nil {
		return nil, err
	}
	if stringValue(cfg["version"]) != "private" {
		if err := addPluginToken(client, cfg); err != nil {
			return nil, err
		}
	}
	cfg["timestamp"] = float64(time.Now().UnixMilli())
	_ = writeCacheEntry(projectPath, cfg)
	return cfg, nil
}

func addBaseURLs(cfg Config) error {
	baseURL := strings.TrimRight(stringValue(cfg["baseUrl"]), "/")
	apiPrefix := stringValue(cfg["apiSvcPrefix"])
	if apiPrefix == "" {
		apiPrefix = "/apisvc"
	}
	setupPrefix := stringValue(cfg["setupSvcPrefix"])
	if setupPrefix == "" {
		setupPrefix = "/setup"
	}
	if baseURL != "" {
		cfg["apiSvc"] = baseURL + apiPrefix
		cfg["setupSvc"] = baseURL + setupPrefix
		return nil
	}
	orgID := stringValue(cfg["orgId"])
	if orgID == "" {
		return nil
	}
	resp, err := http.Get(defaultBaseURL + "/oauth/apidomain?scope=cloudccCRM&orgId=" + url.QueryEscape(orgID))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return err
	}
	if result, _ := body["result"].(bool); result {
		apiSvc := stringValue(body["orgapi_address"])
		cfg["apiSvc"] = apiSvc
		if u, err := url.Parse(apiSvc); err == nil {
			cfg["setupSvc"] = u.Scheme + "://" + u.Host + setupPrefix
		}
	}
	return nil
}

func addBusToken(client *httpclient.Client, cfg Config, force bool) error {
	if stringValue(cfg["accessToken"]) != "" && !force {
		return nil
	}
	if stringValue(cfg["username"]) == "" || stringValue(cfg["safetyMark"]) == "" || stringValue(cfg["clientId"]) == "" || stringValue(cfg["openSecretKey"]) == "" || stringValue(cfg["orgId"]) == "" {
		if force {
			return fmt.Errorf("CloudCC accessToken refresh failed before /api/cauth/token: refreshable credentials are missing; check cloudcc-cli.config.json active env for CloudCCDev or username/safetyMark/clientId/openSecretKey/orgId")
		}
		return nil
	}
	apiSvc := strings.TrimRight(stringValue(cfg["apiSvc"]), "/")
	if apiSvc == "" {
		return fmt.Errorf("CloudCC accessToken refresh failed before /api/cauth/token: apiSvc is missing; check cloudcc-cli.config.json active env for CloudCCDev/baseUrl/orgId/apiSvc configuration")
	}
	body := map[string]any{
		"username":   cfg["username"],
		"safetyMark": cfg["safetyMark"],
		"clientId":   cfg["clientId"],
		"secretKey":  cfg["openSecretKey"],
		"orgId":      cfg["orgId"],
	}
	var res map[string]any
	if err := client.PostRaw(apiSvc+"/api/cauth/token", body, nil, &res); err != nil {
		return fmt.Errorf("CloudCC accessToken refresh failed at /api/cauth/token: %w; check cloudcc-cli.config.json active env for username/safetyMark/clientId/openSecretKey/orgId/apiSvc", err)
	}
	if ok, _ := res["result"].(bool); ok || numberOrString(res["returnCode"], "") == "1" {
		if data, _ := res["data"].(map[string]any); data != nil {
			if token := stringValue(data["accessToken"]); token != "" {
				cfg["accessToken"] = token
				return nil
			}
		}
	}
	return fmt.Errorf("CloudCC accessToken refresh failed at /api/cauth/token: %s; check cloudcc-cli.config.json active env for username/safetyMark/clientId/openSecretKey/orgId/apiSvc", cloudccResponseMessage(res))
}

func addSecretKey(_ *httpclient.Client, cfg Config) error {
	if stringValue(cfg["secretKey"]) != "" || stringValue(cfg["username"]) == "" {
		return nil
	}
	form := "username=" + url.QueryEscape(stringValue(cfg["username"]))
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(baseURL(cfg), "/")+"/sysconfig/auth/secretkey/get", strings.NewReader(form))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var res map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return err
	}
	if data, _ := res["data"].(map[string]any); data != nil {
		cfg["secretKey"] = data["secretKey"]
	}
	return nil
}

func addPluginToken(client *httpclient.Client, cfg Config) error {
	if stringValue(cfg["pluginToken"]) != "" || stringValue(cfg["username"]) == "" || stringValue(cfg["secretKey"]) == "" {
		return nil
	}
	var res map[string]any
	if err := client.PostEnvelope(strings.TrimRight(baseURL(cfg), "/")+"/sysconfig/auth/pc/1.0/post/tokenInfo", map[string]any{
		"username":  cfg["username"],
		"secretKey": cfg["secretKey"],
	}, nil, &res); err != nil {
		return err
	}
	if code := fmt.Sprint(res["returnCode"]); code == "200" {
		if data, _ := res["data"].(map[string]any); data != nil {
			cfg["pluginToken"] = data["accessToken"]
		}
	}
	return nil
}

func readCache(projectPath string) (map[string]any, error) {
	file := filepath.Join(projectPath, ".cloudcc-cache.json")
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return map[string]any{}, nil
	}
	return jsonx.ReadObjectFile(file)
}

func ClearCacheEntry(projectPath string) error {
	key, err := activeCacheKey(projectPath)
	if err != nil {
		return err
	}
	file := filepath.Join(projectPath, ".cloudcc-cache.json")
	if key == "" {
		if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	cache, err := readCache(projectPath)
	if err != nil {
		return err
	}
	delete(cache, key)
	if len(cache) == 0 {
		if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	return jsonx.WriteObjectFile(file, cache)
}

func writeCacheEntry(projectPath string, cfg Config) error {
	key := cacheKey(cfg)
	if key == "" {
		return nil
	}
	cache, _ := readCache(projectPath)
	cache[key] = map[string]any(cfg)
	return jsonx.WriteObjectFile(filepath.Join(projectPath, ".cloudcc-cache.json"), cache)
}

func activeCacheKey(projectPath string) (string, error) {
	root, err := Root(projectPath)
	if err != nil {
		return "", err
	}
	use, _ := root["use"].(string)
	active, _ := root[use].(map[string]any)
	if active == nil {
		return "", nil
	}
	return cacheKey(Config(active)), nil
}

func cacheKey(cfg Config) string {
	key := stringValue(cfg["safetyMark"])
	if key == "" {
		key = stringValue(cfg["secretKey"])
	}
	return key
}

func baseURL(cfg Config) string {
	if v := stringValue(cfg["baseUrl"]); v != "" {
		return v
	}
	return defaultBaseURL
}

func stringValue(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case fmt.Stringer:
		return x.String()
	case nil:
		return ""
	default:
		return fmt.Sprint(x)
	}
}

func String(cfg Config, key string) string {
	return stringValue(cfg[key])
}

func AccessTokenErrorMessage(value any) string {
	return accessTokenErrorMessage(value, 0)
}

func cloudccResponseMessage(res map[string]any) string {
	for _, key := range []string{"returnInfo", "message", "msg", "error", "returnMsg"} {
		if value := stringValue(res[key]); strings.TrimSpace(value) != "" && value != "<nil>" {
			return strings.TrimSpace(value)
		}
	}
	code := strings.TrimSpace(numberOrString(res["returnCode"], ""))
	if code != "" && code != "<nil>" {
		return "returnCode=" + code
	}
	return "response did not include data.accessToken"
}

func numberOrString(v any, fallback string) string {
	switch x := v.(type) {
	case nil:
		return fallback
	case string:
		return x
	case float64:
		return fmt.Sprintf("%.0f", x)
	default:
		return fmt.Sprint(x)
	}
}

func accessTokenErrorMessage(value any, depth int) string {
	if value == nil || depth > 6 {
		return ""
	}
	switch v := value.(type) {
	case error:
		return accessTokenErrorMessage(v.Error(), depth+1)
	case string:
		text := strings.TrimSpace(v)
		if text == "" || text == "<nil>" {
			return ""
		}
		lower := strings.ToLower(text)
		if strings.Contains(text, "accessToken无效") ||
			strings.Contains(text, "token无效") ||
			strings.Contains(text, "令牌无效") ||
			strings.Contains(text, "登录过期") ||
			strings.Contains(text, "认证失败") ||
			strings.Contains(lower, "invalid_token") ||
			strings.Contains(lower, "expired_token") ||
			strings.Contains(lower, "token expired") ||
			strings.Contains(lower, "access token has expired") ||
			strings.Contains(lower, "access token validation failed") ||
			strings.Contains(lower, "accesstoken invalid") ||
			strings.Contains(lower, "invalid accesstoken") {
			return text
		}
	case map[string]any:
		for _, key := range []string{"error", "returnInfo", "message", "msg", "returnMsg", "code", "returnCode"} {
			if msg := accessTokenErrorMessage(v[key], depth+1); msg != "" {
				return msg
			}
		}
		for _, child := range v {
			if msg := accessTokenErrorMessage(child, depth+1); msg != "" {
				return msg
			}
		}
	case []any:
		for _, child := range v {
			if msg := accessTokenErrorMessage(child, depth+1); msg != "" {
				return msg
			}
		}
	}
	return ""
}

func tokenNearExpiry(token string, skew time.Duration) bool {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 3 {
		return false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return false
	}
	exp, ok := numericClaim(claims["exp"])
	if !ok || exp <= 0 {
		return false
	}
	return time.Now().Add(skew).Unix() >= exp
}

func numericClaim(value any) (int64, bool) {
	switch v := value.(type) {
	case float64:
		return int64(v), true
	case int64:
		return v, true
	case int:
		return int64(v), true
	case json.Number:
		n, err := v.Int64()
		return n, err == nil
	case string:
		var n int64
		if _, err := fmt.Sscan(strings.TrimSpace(v), &n); err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}
