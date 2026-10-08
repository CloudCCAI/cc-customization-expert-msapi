package horizontalaction

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"cloudcc-customization-expert-go/internal/config"
	"cloudcc-customization-expert-go/internal/horizontal"
)

const maxResponseSize = 16 << 20

var responseQuerySecretPattern = regexp.MustCompile(`(?i)(binding|accessToken|pluginToken|token|password)=([^&\s"'<>]+)`)
var responseInputSecretPattern = regexp.MustCompile(`(?i)(name=["'](?:binding|accessToken|pluginToken|token|password)["'][^>]*value=["'])([^"']*)(["'])`)

type Receipt struct {
	Operation        string `json:"operation"`
	Resource         string `json:"resource"`
	Status           string `json:"status"`
	HTTPStatus       int    `json:"httpStatus"`
	Location         string `json:"location,omitempty"`
	Content          string `json:"content,omitempty"`
	OutputPath       string `json:"outputPath,omitempty"`
	IDResolution     string `json:"idResolution,omitempty"`
	ID               string `json:"id,omitempty"`
	ConfigUpdated    *bool  `json:"configUpdated,omitempty"`
	ResolutionSource string `json:"resolutionSource,omitempty"`
	Verification     string `json:"verification,omitempty"`
	rawContent       string
	rawLocation      string
}

func (c *Client) Download(ctx context.Context, projectPath string, cfg config.Config, resource, operation, path string, query url.Values, outputPath string) (receipt Receipt, err error) {
	session, err := c.session.AcquireActionSession(ctx, projectPath, cfg)
	if err != nil {
		return Receipt{}, err
	}
	endpoint, err := c.actionURL(path, session.Binding)
	if err != nil {
		return Receipt{}, err
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return Receipt{}, err
	}
	q := u.Query()
	for key, values := range query {
		for _, value := range values {
			q.Add(key, value)
		}
	}
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Receipt{}, err
	}
	req.Header.Set("Referer", c.baseURL+"/")
	res, err := c.http.Do(req)
	if err != nil {
		return Receipt{}, fmt.Errorf("horizontal %s %s Action request failed: %w", resource, operation, err)
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 400 {
		return Receipt{}, fmt.Errorf("horizontal %s %s Action returned HTTP %d", resource, operation, res.StatusCode)
	}
	absOutput, err := filepath.Abs(outputPath)
	if err != nil {
		return Receipt{}, err
	}
	if err := os.MkdirAll(filepath.Dir(absOutput), 0755); err != nil {
		return Receipt{}, err
	}
	temp, err := os.CreateTemp(filepath.Dir(absOutput), ".cloudcc-static-resource-*")
	if err != nil {
		return Receipt{}, err
	}
	tempPath := temp.Name()
	defer func() {
		_ = temp.Close()
		if err != nil {
			_ = os.Remove(tempPath)
		}
	}()
	written, copyErr := io.Copy(temp, io.LimitReader(res.Body, (5<<20)+1))
	if copyErr != nil {
		return Receipt{}, copyErr
	}
	if written > 5<<20 {
		return Receipt{}, fmt.Errorf("horizontal staticResource download exceeds 5 MiB")
	}
	if err = temp.Close(); err != nil {
		return Receipt{}, err
	}
	if _, statErr := os.Stat(absOutput); statErr == nil {
		return Receipt{}, fmt.Errorf("refusing to overwrite existing staticResource output %s", absOutput)
	} else if !os.IsNotExist(statErr) {
		return Receipt{}, statErr
	}
	if err = os.Rename(tempPath, absOutput); err != nil {
		return Receipt{}, err
	}
	return Receipt{Operation: operation, Resource: resource, Status: "received", HTTPStatus: res.StatusCode, OutputPath: absOutput}, nil
}

type Client struct {
	baseURL string
	http    *http.Client
	session *horizontal.Client
	binding string
}

func New(mainAppURL string) (*Client, error) {
	base := strings.TrimRight(strings.TrimSpace(mainAppURL), "/")
	u, err := url.Parse(base)
	if err != nil || u.Scheme == "" || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, fmt.Errorf("invalid horizontal mainAppUrl %q", mainAppURL)
	}
	jar, _ := cookiejar.New(nil)
	hc := &http.Client{
		Timeout:       60 * time.Second,
		Jar:           jar,
		Transport:     &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}, // preserve CLI behavior
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}
	sessionClient, err := horizontal.NewWithHTTPClient(base, hc)
	if err != nil {
		return nil, err
	}
	return &Client{baseURL: base, http: hc, session: sessionClient}, nil
}

func (c *Client) Form(ctx context.Context, projectPath string, cfg config.Config, resource, operation, path string, values url.Values, mutating bool) (Receipt, error) {
	binding, err := c.ensureSession(ctx, projectPath, cfg)
	if err != nil {
		return Receipt{}, err
	}
	values = cloneValues(values)
	endpoint, err := c.actionURL(path, binding)
	if err != nil {
		return Receipt{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return Receipt{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	return c.do(req, resource, operation, mutating)
}

func (c *Client) MultipartFile(ctx context.Context, projectPath string, cfg config.Config, resource, operation, path string, fields url.Values, fieldName, filePath string, mutating bool) (Receipt, error) {
	binding, err := c.ensureSession(ctx, projectPath, cfg)
	if err != nil {
		return Receipt{}, err
	}
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	for key, values := range fields {
		for _, value := range values {
			if err := w.WriteField(key, value); err != nil {
				return Receipt{}, err
			}
		}
	}
	if strings.TrimSpace(filePath) != "" {
		f, err := os.Open(filePath)
		if err != nil {
			return Receipt{}, err
		}
		defer f.Close()
		part, err := w.CreateFormFile(fieldName, filepath.Base(filePath))
		if err != nil {
			return Receipt{}, err
		}
		if _, err := io.Copy(part, f); err != nil {
			return Receipt{}, err
		}
	}
	if err := w.Close(); err != nil {
		return Receipt{}, err
	}
	endpoint, err := c.actionURL(path, binding)
	if err != nil {
		return Receipt{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return Receipt{}, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	return c.do(req, resource, operation, mutating)
}

func (c *Client) ensureSession(ctx context.Context, projectPath string, cfg config.Config) (string, error) {
	if c.binding != "" {
		return c.binding, nil
	}
	session, err := c.session.AcquireActionSession(ctx, projectPath, cfg)
	if err != nil {
		return "", err
	}
	c.binding = session.Binding
	return c.binding, nil
}

func (c *Client) actionURL(path, binding string) (string, error) {
	if !strings.HasPrefix(path, "/") || strings.Contains(path, "://") {
		return "", fmt.Errorf("invalid horizontal Action path %q", path)
	}
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("binding", binding)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (c *Client) do(req *http.Request, resource, operation string, mutating bool) (Receipt, error) {
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/json;q=0.9,*/*;q=0.8")
	req.Header.Set("Referer", c.baseURL+"/")
	res, err := c.http.Do(req)
	if err != nil {
		return Receipt{}, fmt.Errorf("horizontal %s %s Action request failed: %w", resource, operation, err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, maxResponseSize+1))
	if err != nil {
		return Receipt{}, fmt.Errorf("horizontal %s %s Action response failed: %w", resource, operation, err)
	}
	if len(body) > maxResponseSize {
		return Receipt{}, fmt.Errorf("horizontal %s %s Action response exceeds %d bytes", resource, operation, maxResponseSize)
	}
	if res.StatusCode < 200 || res.StatusCode >= 400 {
		return Receipt{}, fmt.Errorf("horizontal %s %s Action returned HTTP %d", resource, operation, res.StatusCode)
	}
	receipt := Receipt{Operation: operation, Resource: resource, Status: "submitted", HTTPStatus: res.StatusCode, rawContent: string(body)}
	if id := strings.TrimSpace(res.Header.Get("X-CloudCC-Resource-Id")); id != "" {
		receipt.ID = id
		receipt.ResolutionSource = "response_header"
	}
	if !mutating {
		receipt.Status = "received"
		receipt.Content = sanitizeResponseContent(string(body))
	}
	if location := strings.TrimSpace(res.Header.Get("Location")); location != "" {
		if parsed, err := url.Parse(location); err == nil && !parsed.IsAbs() {
			receipt.rawLocation = location
			if receipt.ID == "" {
				receipt.ID = firstNonBlank(parsed.Query().Get("resourceId"), parsed.Query().Get("id"))
				if receipt.ID != "" {
					receipt.ResolutionSource = "redirect_location"
				}
			}
			parsed.RawQuery = ""
			receipt.Location = parsed.String()
		}
	}
	return receipt, nil
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func sanitizeResponseContent(content string) string {
	content = responseQuerySecretPattern.ReplaceAllString(content, `${1}=[REDACTED]`)
	content = responseInputSecretPattern.ReplaceAllString(content, `${1}[REDACTED]${3}`)
	return content
}

func cloneValues(in url.Values) url.Values {
	out := url.Values{}
	for key, values := range in {
		out[key] = append([]string(nil), values...)
	}
	return out
}
