package horizontalui

import (
	"fmt"
	"io"
	"strings"
)

var aliases = map[string]string{
	"field": "fields", "fieldId": "fields",
	"pageLayout": "pagelayout", "layout": "pagelayout",
	"record-type": "recordType", "record-type-list": "recordType",
	"currencies": "currency", "companyCurrency": "currency", "company-currency": "currency",
	"fiscal-year": "fiscalYear", "fiscalYears": "fiscalYear", "fiscal-years": "fiscalYear",
	"object-view": "view", "object-views": "view",
	"apiRegister": "apiRegistrar", "api-registrar": "apiRegistrar", "api-register": "apiRegistrar",
}

// HandleLowCode fails closed until a maintained main-app Struts Action contract
// is registered for the requested operation. Horizontal compatibility must not
// call legacy Jersey Resource endpoints: those resources have not been
// maintained as a complete, stable setup API and may be absent or incomplete in
// a target deployment. Explicit horizontal projects never fall back to
// Lightning setup-svc/api-svc routes.
func HandleLowCode(action string, resource string, _ []string, _ io.Writer, _ string) error {
	resource = normalizeResource(resource)
	if resource == "apiRegistrar" {
		return fmt.Errorf("horizontal UIAPI does not expose a maintained API-registrar management contract; use the MSAPI provider")
	}
	return fmt.Errorf("cloudcc %s %s is not supported by horizontal UIAPI: legacy main-app Jersey Resource endpoints are intentionally disabled; a maintained Struts Action adapter is required", action, resource)
}

func normalizeResource(resource string) string {
	resource = strings.TrimSpace(resource)
	if canonical, ok := aliases[resource]; ok {
		return canonical
	}
	return resource
}
