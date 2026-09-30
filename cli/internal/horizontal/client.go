package horizontal

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"cloudcc-customization-expert-go/internal/config"
)

const (
	DistributorPath          = "/distributor.action"
	LoginServiceName         = "clogin"
	BindingValidationService = "isValidWithBinding"
	maxResponseSize          = 16 << 20
)

type Session struct {
	Binding    string
	Token      string
	UserInfo   map[string]any
	ReturnCode string
}

type RemoteError struct {
	Operation string
	Code      string
	Message   string
}

func (e *RemoteError) Error() string {
	category := remoteErrorCategory(e.Code)
	if e.Code != "" {
		return fmt.Sprintf("horizontal %s failed [%s/%s]: %s", e.Operation, category, e.Code, e.Message)
	}
	return fmt.Sprintf("horizontal %s failed [%s]: %s", e.Operation, category, e.Message)
}

type Client struct {
	baseURL        string
	distributorURL string
	http           *http.Client
}

func New(mainAppURL string) (*Client, error) {
	return NewWithHTTPClient(mainAppURL, nil)
}

func NewWithHTTPClient(mainAppURL string, client *http.Client) (*Client, error) {
	base := strings.TrimRight(strings.TrimSpace(mainAppURL), "/")
	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("invalid horizontal mainAppUrl %q", mainAppURL)
	}
	if client == nil {
		jar, _ := cookiejar.New(nil)
		client = &http.Client{
			Timeout: 60 * time.Second,
			Jar:     jar,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // preserve existing CLI TLS behavior
			},
		}
	}
	return &Client{baseURL: base, distributorURL: base + DistributorPath, http: client}, nil
}

func (c *Client) Login(ctx context.Context, username string, password string, language string) (Session, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return Session{}, fmt.Errorf("horizontal login requires auth.username and auth.password")
	}
	if strings.TrimSpace(language) == "" {
		language = "zh"
	}
	response, err := c.Post(ctx, url.Values{
		"serviceName": {LoginServiceName},
		"userName":    {username},
		"password":    {password},
		"language":    {language},
	})
	if err != nil {
		return Session{}, err
	}
	if !succeeded(response) {
		return Session{}, remoteError("login", response, "login rejected")
	}
	binding := text(response["binding"])
	if binding == "" {
		return Session{}, &RemoteError{Operation: "login", Code: text(response["returnCode"]), Message: "response did not include binding"}
	}
	userInfo, _ := response["userInfo"].(map[string]any)
	return Session{
		Binding:    binding,
		Token:      text(response["token"]),
		UserInfo:   userInfo,
		ReturnCode: text(response["returnCode"]),
	}, nil
}

// AcquireSession reuses only a target-scoped cached binding that the server
// still considers valid. Passwords and cookies are never persisted.
func (c *Client) AcquireSession(ctx context.Context, projectPath string, cfg config.Config) (Session, error) {
	if cached, ok := config.LoadHorizontalSession(projectPath); ok {
		valid, err := c.IsValid(ctx, cached.Binding)
		if err == nil && valid {
			return Session{Binding: cached.Binding, Token: cached.Token, UserInfo: cached.UserInfo}, nil
		}
		_ = config.ClearCacheEntry(projectPath)
	}
	session, err := c.Login(ctx, config.String(cfg, "username"), config.String(cfg, "password"), config.String(cfg, "language"))
	if err != nil {
		return Session{}, err
	}
	if err := config.SaveHorizontalSession(projectPath, config.HorizontalSessionCache{
		Binding: session.Binding, Token: session.Token, UserInfo: cacheableUserInfo(session.UserInfo),
	}); err != nil {
		return Session{}, fmt.Errorf("horizontal session cache failed: %w", err)
	}
	return session, nil
}

func cacheableUserInfo(userInfo map[string]any) map[string]any {
	out := map[string]any{}
	for _, key := range []string{"userId", "userid", "orgId", "orgid", "profileId", "profileid", "roleId", "roleid", "language", "dbType"} {
		if value, exists := userInfo[key]; exists {
			out[key] = value
		}
	}
	return out
}

func (c *Client) IsValid(ctx context.Context, binding string) (bool, error) {
	if strings.TrimSpace(binding) == "" {
		return false, nil
	}
	response, err := c.Post(ctx, url.Values{
		"serviceName": {BindingValidationService},
		"binding":     {binding},
	})
	if err != nil {
		return false, err
	}
	if !succeeded(response) {
		return false, remoteError("binding validation", response, "validation rejected")
	}
	data, _ := response["data"].(map[string]any)
	valid, _ := data["isValid"].(bool)
	return valid, nil
}

func (c *Client) Post(ctx context.Context, form url.Values) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.distributorURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	req.Header.Set("Accept", "application/json")
	response, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("horizontal distributor request failed: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseSize+1))
	if err != nil {
		return nil, fmt.Errorf("horizontal distributor response read failed: %w", err)
	}
	if len(body) > maxResponseSize {
		return nil, fmt.Errorf("horizontal distributor response exceeds %d bytes", maxResponseSize)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("horizontal distributor returned HTTP %d", response.StatusCode)
	}
	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("ACTION_REDIRECT_LOGIN: horizontal distributor returned non-JSON response")
	}
	return result, nil
}

// PostJSON invokes an existing main-app JSON endpoint with the authenticated
// binding. It is intentionally separate from the distributor protocol so UIAPI
// adapters cannot accidentally send REST payloads through the login channel.
func (c *Client) PostJSON(ctx context.Context, path string, binding string, query url.Values, body map[string]any) (map[string]any, error) {
	path = strings.TrimSpace(path)
	if path == "" || !strings.HasPrefix(path, "/") {
		return nil, fmt.Errorf("invalid horizontal main-app API path %q", path)
	}
	if strings.TrimSpace(binding) == "" {
		return nil, fmt.Errorf("horizontal main-app API request requires binding")
	}
	if query == nil {
		query = url.Values{}
	}
	query = cloneValues(query)
	query.Set("binding", binding)
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("horizontal main-app request encode failed: %w", err)
	}
	endpoint := c.baseURL + path
	if encoded := query.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(payload)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("binding", binding)
	response, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("horizontal main-app API request failed: %w", err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, maxResponseSize+1))
	if err != nil {
		return nil, fmt.Errorf("horizontal main-app API response read failed: %w", err)
	}
	if len(responseBody) > maxResponseSize {
		return nil, fmt.Errorf("horizontal main-app API response exceeds %d bytes", maxResponseSize)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("horizontal main-app API returned HTTP %d", response.StatusCode)
	}
	var result map[string]any
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("horizontal main-app API returned non-JSON response")
	}
	return result, nil
}

func cloneValues(values url.Values) url.Values {
	cloned := url.Values{}
	for key, list := range values {
		cloned[key] = append([]string(nil), list...)
	}
	return cloned
}

func remoteErrorCategory(code string) string {
	switch strings.TrimSpace(code) {
	case "-2":
		return "AUTH_BINDING_EXPIRED"
	case "10054":
		return "AUTH_MFA_REQUIRED"
	case "10060":
		return "AUTH_ENCRYPTION_REQUIRED"
	default:
		return "ACTION_VALIDATION_FAILED"
	}
}

func succeeded(response map[string]any) bool {
	if result, ok := response["result"].(bool); ok {
		return result
	}
	return text(response["returnCode"]) == "1"
}

func remoteError(operation string, response map[string]any, fallback string) error {
	message := firstText(response["returnInfo"], response["message"], response["msg"])
	if message == "" {
		message = fallback
	}
	return &RemoteError{Operation: operation, Code: text(response["returnCode"]), Message: message}
}

func firstText(values ...any) string {
	for _, value := range values {
		if candidate := text(value); candidate != "" && candidate != "<nil>" {
			return candidate
		}
	}
	return ""
}

func text(value any) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}
