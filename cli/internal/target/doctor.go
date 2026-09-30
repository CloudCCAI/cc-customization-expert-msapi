package target

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"

	"cloudcc-customization-expert-go/internal/config"
	"cloudcc-customization-expert-go/internal/horizontal"
)

type DoctorResult struct {
	PlatformMode     string             `json:"platformMode"`
	Status           string             `json:"status"`
	ExecutionMode    string             `json:"executionMode,omitempty"`
	SelectedProvider string             `json:"selectedProvider,omitempty"`
	MainAppOrigin    string             `json:"mainAppOrigin,omitempty"`
	AuthConfigured   bool               `json:"authConfigured"`
	Authenticated    bool               `json:"authenticated,omitempty"`
	UserID           string             `json:"userId,omitempty"`
	OrgID            string             `json:"orgId,omitempty"`
	Capabilities     []CapabilityStatus `json:"capabilities,omitempty"`
}

type CapabilityStatus struct {
	Domain       string `json:"domain"`
	Actions      string `json:"actions"`
	Availability string `json:"availability"`
	Reason       string `json:"reason,omitempty"`
}

func WriteDoctor(projectPath string, stdout io.Writer) error {
	cfg, err := config.Load(projectPath)
	if err != nil {
		return err
	}
	mode := config.PlatformMode(cfg)
	result := DoctorResult{
		PlatformMode:  mode,
		ExecutionMode: config.String(cfg, "executionMode"),
		Status:        "ok",
	}
	if mode != config.PlatformHorizontal {
		result.AuthConfigured = config.String(cfg, "accessToken") != "" || config.String(cfg, "CloudCCDev") != ""
		return json.NewEncoder(stdout).Encode(result)
	}

	result.SelectedProvider = selectedHorizontalProvider(cfg)
	metadataAvailability := "provider-adapter"
	metadataReason := "source-backed UIAPI batch supports object/application/record-type reads, object-view CRUD, and PC page-layout detail/save"
	dataExtensionAvailability := "requires-msapi"
	dataExtensionReason := "configure MetadataService and select msapi or auto"
	if hasMetadataService(cfg) {
		metadataAvailability = "partial"
		metadataReason = "shared domains and complete API-registrar metadata CRUD are available through MetadataService; report/dashboard writes remain closed"
		dataExtensionAvailability = "enabled"
		dataExtensionReason = "available through MetadataService"
	}
	result.Capabilities = []CapabilityStatus{
		{Domain: "openapi", Actions: "query,pageQuery,create,update,delete,upsert", Availability: "enabled"},
		{Domain: "metadata", Actions: "*", Availability: metadataAvailability, Reason: metadataReason},
		{Domain: "api-registrars", Actions: "uiapi-management", Availability: "unsupported", Reason: "main-app exposes no registered management controller; select MSAPI"},
		{Domain: "dataIndex,dataBulk", Actions: "*", Availability: dataExtensionAvailability, Reason: dataExtensionReason},
		{Domain: "highcode", Actions: "remote", Availability: "pending-evidence", Reason: "main-app high-code adapter is not enabled"},
	}
	result.MainAppOrigin = origin(config.String(cfg, "mainAppUrl"))
	result.AuthConfigured = strings.TrimSpace(config.String(cfg, "username")) != "" && config.String(cfg, "password") != ""
	if config.String(cfg, "password") == "CLOUDCC_PASSWORD" {
		return fmt.Errorf("horizontal auth.password is still the generated CLOUDCC_PASSWORD placeholder")
	}
	client, err := horizontal.New(config.String(cfg, "mainAppUrl"))
	if err != nil {
		return err
	}
	session, err := client.AcquireSession(context.Background(), projectPath, cfg)
	if err != nil {
		return err
	}
	result.UserID = mapText(session.UserInfo, "userId", "userid")
	result.OrgID = mapText(session.UserInfo, "orgId", "orgid")
	result.Authenticated, err = client.IsValid(context.Background(), session.Binding)
	if err != nil {
		return err
	}
	if !result.Authenticated {
		return fmt.Errorf("horizontal session validation failed after login")
	}
	return json.NewEncoder(stdout).Encode(result)
}

func selectedHorizontalProvider(cfg config.Config) string {
	mode := strings.ToLower(strings.TrimSpace(config.String(cfg, "executionMode")))
	if mode == "msapi" || mode == "uiapi" {
		return mode
	}
	if hasMetadataService(cfg) {
		return "auto"
	}
	return "uiapi"
}

func hasMetadataService(cfg config.Config) bool {
	if strings.TrimSpace(config.String(cfg, "metadataServiceUrl")) != "" {
		return true
	}
	metadata, _ := cfg["metadataService"].(map[string]any)
	return strings.TrimSpace(fmt.Sprint(metadata["url"])) != "" && fmt.Sprint(metadata["url"]) != "<nil>"
}

func origin(value string) string {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host
}

func mapText(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(fmt.Sprint(values[key])); value != "" && value != "<nil>" {
			return value
		}
	}
	return ""
}
