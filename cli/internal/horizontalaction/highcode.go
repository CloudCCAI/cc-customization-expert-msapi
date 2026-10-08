package horizontalaction

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"cloudcc-customization-expert-go/internal/config"
)

func ExecuteForm(ctx context.Context, projectPath string, cfg config.Config, resource, operation string, form url.Values, mutating bool) (Receipt, error) {
	paths := map[string]string{"classes": "/ccfag.action", "trigger": "/trigger.action", "timer": "/ccpeak.action", "visualPage": "/page.action", "visualPageAccess": "/saveAccessAuthority.action", "visualPageProfileEnable": "/savetenableVP.action", "customComponent": "/customComponent.action", "staticResource": "/staticResource.action"}
	path, ok := paths[resource]
	if !ok {
		return Receipt{}, fmt.Errorf("unsupported horizontal high-code resource %q", resource)
	}
	client, err := New(config.String(cfg, "mainAppUrl"))
	if err != nil {
		return Receipt{}, err
	}
	return client.Form(ctx, projectPath, cfg, resource, operation, path, form, mutating)
}

func ExecuteStaticResourceDownload(ctx context.Context, projectPath string, cfg config.Config, resourceID, outputPath string) (Receipt, error) {
	client, err := New(config.String(cfg, "mainAppUrl"))
	if err != nil {
		return Receipt{}, err
	}
	return client.Download(ctx, projectPath, cfg, "staticResource", "pull", "/staticResource.action", url.Values{"m": {"getResource"}, "resourceId": {resourceID}}, outputPath)
}

func ExecuteStaticResource(ctx context.Context, projectPath string, cfg config.Config, operation string, fields url.Values, filePath string, mutating bool) (Receipt, error) {
	client, err := New(config.String(cfg, "mainAppUrl"))
	if err != nil {
		return Receipt{}, err
	}
	if filePath != "" {
		info, err := os.Stat(filePath)
		if err != nil {
			return Receipt{}, err
		}
		if info.Size() > 5<<20 {
			return Receipt{}, fmt.Errorf("horizontal staticResource file exceeds 5 MiB: %d bytes", info.Size())
		}
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filePath), "."))
		allowed := map[string]bool{"png": true, "jpg": true, "gif": true, "js": true, "css": true, "zip": true}
		if !allowed[ext] {
			return Receipt{}, fmt.Errorf("horizontal staticResource supports png, jpg, gif, js, css, or zip; got %q", ext)
		}
		fields.Set("contentType", ext)
	}
	return client.MultipartFile(ctx, projectPath, cfg, "staticResource", operation, "/staticResource.action", fields, "file", filePath, mutating)
}
