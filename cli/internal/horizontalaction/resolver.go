package horizontalaction

import (
	"context"
	"fmt"
	"html"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"cloudcc-customization-expert-go/internal/config"
)

type PublishIdentity struct {
	Name        string
	APIName     string
	Label       string
	ContentType string
}

type PublishRequest struct {
	Resource string
	Path     string
	Form     url.Values
	IDField  string
	Identity PublishIdentity
	FilePath string
}

var resolutionRetryDelays = []time.Duration{0, 200 * time.Millisecond, 500 * time.Millisecond, time.Second, 2 * time.Second}

func ExecuteResolvedPublish(ctx context.Context, projectPath string, cfg config.Config, request PublishRequest) (Receipt, error) {
	if request.Resource == "staticResource" {
		info, statErr := os.Stat(request.FilePath)
		if statErr != nil {
			return Receipt{}, statErr
		}
		if info.Size() > 5<<20 {
			return Receipt{}, fmt.Errorf("horizontal staticResource file exceeds 5 MiB: %d bytes", info.Size())
		}
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(request.FilePath), "."))
		allowed := map[string]bool{"png": true, "jpg": true, "gif": true, "js": true, "css": true, "zip": true}
		if !allowed[ext] {
			return Receipt{}, fmt.Errorf("horizontal staticResource supports png, jpg, gif, js, css, or zip; got %q", ext)
		}
		request.Form.Set("contentType", ext)
	}
	client, err := New(config.String(cfg, "mainAppUrl"))
	if err != nil {
		return Receipt{}, err
	}
	form := cloneValues(request.Form)
	resolvedBeforeSubmit := ""
	if strings.TrimSpace(form.Get(request.IDField)) == "" {
		matches, lookupErr := lookupExact(ctx, client, projectPath, cfg, request)
		if lookupErr != nil {
			return Receipt{}, fmt.Errorf("horizontal %s pre-publish id lookup failed: %w", request.Resource, lookupErr)
		}
		switch len(matches) {
		case 0:
		case 1:
			if verifyID(ctx, client, projectPath, cfg, request, matches[0]) {
				resolvedBeforeSubmit = matches[0]
				form.Set(request.IDField, matches[0])
			}
		default:
			return Receipt{}, fmt.Errorf("horizontal %s identity is ambiguous: %d exact matches", request.Resource, len(matches))
		}
	}
	var receipt Receipt
	if request.Resource == "staticResource" {
		receipt, err = client.MultipartFile(ctx, projectPath, cfg, request.Resource, "publish", request.Path, form, "file", request.FilePath, true)
	} else {
		receipt, err = client.Form(ctx, projectPath, cfg, request.Resource, "publish", request.Path, form, true)
	}
	if err != nil {
		return Receipt{}, err
	}
	if existingID := strings.TrimSpace(form.Get(request.IDField)); receipt.ID == "" && existingID != "" {
		receipt.ID = existingID
		receipt.ResolutionSource = "existing_config"
	}
	if resolvedBeforeSubmit != "" {
		receipt.ID = resolvedBeforeSubmit
		receipt.ResolutionSource = "preflight_recovery"
	}
	if receipt.ID == "" {
		if matches := exactMatches(request.Resource, receipt.rawContent, request.Identity); len(matches) == 1 {
			receipt.ID = matches[0]
			receipt.ResolutionSource = "save_body"
		}
	}
	if receipt.ID == "" {
		for _, delay := range resolutionRetryDelays {
			if delay > 0 {
				select {
				case <-ctx.Done():
					return receipt, ctx.Err()
				case <-time.After(delay):
				}
			}
			matches, lookupErr := lookupExact(ctx, client, projectPath, cfg, request)
			if lookupErr != nil {
				continue
			}
			if len(matches) == 1 {
				receipt.ID = matches[0]
				receipt.ResolutionSource = "secondary_query"
				break
			}
			if len(matches) > 1 {
				break
			}
		}
	}
	if receipt.ID != "" && verifyID(ctx, client, projectPath, cfg, request, receipt.ID) {
		receipt.IDResolution = "resolved"
		receipt.Verification = "detail"
	} else {
		receipt.ID = ""
		receipt.IDResolution = "unresolved"
		receipt.ResolutionSource = ""
		receipt.Verification = ""
	}
	updated := false
	receipt.ConfigUpdated = &updated
	return receipt, nil
}

func lookupExact(ctx context.Context, client *Client, projectPath string, cfg config.Config, request PublishRequest) ([]string, error) {
	form := url.Values{}
	switch request.Resource {
	case "customComponent":
		form.Set("m", "index")
	case "staticResource":
		form.Set("m", "list")
	default:
		form.Set("m", "list")
		form.Set("sname", firstNonBlank(request.Identity.APIName, request.Identity.Name, request.Identity.Label))
		form.Set("shownum", "2000")
		form.Set("showpage", "1")
	}
	receipt, err := client.Form(ctx, projectPath, cfg, request.Resource, "resolve-id", request.Path, form, false)
	if err != nil {
		return nil, err
	}
	return exactMatches(request.Resource, receipt.rawContent, request.Identity), nil
}

func verifyID(ctx context.Context, client *Client, projectPath string, cfg config.Config, request PublishRequest, id string) bool {
	form := url.Values{}
	switch request.Resource {
	case "customComponent":
		form.Set("m", "queryById")
		form.Set("id", id)
	case "staticResource":
		form.Set("m", "detail")
		form.Set("resourceId", id)
	default:
		form.Set("m", "detail")
		form.Set("id", id)
	}
	receipt, err := client.Form(ctx, projectPath, cfg, request.Resource, "verify-id", request.Path, form, false)
	if err != nil {
		return false
	}
	plain := searchableContent(receipt.rawContent)
	for _, expected := range []string{request.Identity.APIName, request.Identity.Name, request.Identity.Label} {
		if expected != "" && !containsToken(plain, expected) {
			return false
		}
	}
	if request.Resource == "staticResource" && request.Identity.ContentType != "" && !containsToken(plain, request.Identity.ContentType) {
		return false
	}
	return id != ""
}

var rowPattern = regexp.MustCompile(`(?is)<tr\b[^>]*\bclass=["'][^"']*\bdataRow\b[^"']*["'][^>]*>(.*?)</tr>`)
var anchorPattern = regexp.MustCompile(`(?is)<a\b([^>]*)>(.*?)</a>`)
var cellPattern = regexp.MustCompile(`(?is)<td\b[^>]*>(.*?)</td>`)
var attributePattern = regexp.MustCompile(`(?is)([a-zA-Z_:][-a-zA-Z0-9_:.]*)\s*=\s*["']([^"']*)["']`)
var tagPattern = regexp.MustCompile(`(?is)<[^>]+>`)
var customIDPattern = regexp.MustCompile(`(?is)submitLink\(\s*['"]([^'"]+)['"]\s*,\s*['"]queryById['"]`)
var staticIDPattern = regexp.MustCompile(`(?is)[?&]resourceId=([^&"']+)`)

func exactMatches(resource, content string, identity PublishIdentity) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, match := range rowPattern.FindAllStringSubmatch(content, -1) {
		row := match[1]
		cells := []string{}
		for _, cell := range cellPattern.FindAllStringSubmatch(row, -1) {
			cells = append(cells, normalizedText(cell[1]))
		}
		id := ""
		switch resource {
		case "customComponent":
			if m := customIDPattern.FindStringSubmatch(row); len(m) > 1 {
				id = html.UnescapeString(m[1])
			}
		case "staticResource":
			if m := staticIDPattern.FindStringSubmatch(html.UnescapeString(row)); len(m) > 1 {
				id, _ = url.QueryUnescape(html.UnescapeString(m[1]))
			}
		default:
			for _, a := range anchorPattern.FindAllStringSubmatch(row, -1) {
				attrs := parseAttributes(a[1])
				if hasClass(attrs["class"], "xxx") {
					id = attrs["id"]
					break
				}
			}
		}
		if id == "" {
			continue
		}
		if identity.APIName != "" && !hasExactCell(cells, identity.APIName) {
			continue
		}
		if identity.Name != "" && !hasExactCell(cells, identity.Name) {
			continue
		}
		if identity.Label != "" && !hasExactCell(cells, identity.Label) {
			continue
		}
		if resource == "staticResource" && identity.ContentType != "" && !hasExactCell(cells, identity.ContentType) {
			continue
		}
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

func parseAttributes(source string) map[string]string {
	out := map[string]string{}
	for _, match := range attributePattern.FindAllStringSubmatch(source, -1) {
		out[strings.ToLower(match[1])] = html.UnescapeString(match[2])
	}
	return out
}

func hasClass(classes, want string) bool {
	for _, value := range strings.Fields(classes) {
		if value == want {
			return true
		}
	}
	return false
}
func hasExactCell(cells []string, want string) bool {
	for _, value := range cells {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(want)) {
			return true
		}
	}
	return false
}
func normalizedText(source string) string {
	return strings.Join(strings.Fields(html.UnescapeString(tagPattern.ReplaceAllString(source, " "))), " ")
}
func searchableContent(source string) string {
	parts := []string{normalizedText(source)}
	for _, tag := range regexp.MustCompile(`(?is)<(?:input|textarea)\b([^>]*)>`).FindAllStringSubmatch(source, -1) {
		attrs := parseAttributes(tag[1])
		if value := attrs["value"]; value != "" {
			parts = append(parts, value)
		}
	}
	return strings.Join(parts, " ")
}
func containsToken(content, expected string) bool {
	return strings.Contains(strings.ToLower(content), strings.ToLower(strings.TrimSpace(expected)))
}
