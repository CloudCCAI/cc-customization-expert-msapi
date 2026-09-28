// Package domaincatalog exposes the embedded first-class Domain catalog used
// by help, discovery, provider gates, documentation, and release generation.
package domaincatalog

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"cloudcc-customization-expert-go/internal/edition"
)

//go:embed domain-catalog.json
var catalogJSON []byte

type Catalog struct {
	SchemaVersion string     `json:"schemaVersion"`
	Categories    []Category `json:"categories"`
	Domains       []Domain   `json:"domains"`
}

type Category struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

type Availability struct {
	MSAPI     string `json:"msapi"`
	UIAPI     string `json:"uiapi"`
	Universal string `json:"universal"`
}

type Domain struct {
	Name                  string       `json:"name"`
	Label                 string       `json:"label"`
	Level                 string       `json:"level"`
	Category              string       `json:"category"`
	Backend               string       `json:"backend"`
	Transport             string       `json:"transport"`
	Availability          Availability `json:"availability"`
	MinimumBackendVersion string       `json:"minimumBackendVersion,omitempty"`
	DocumentationCommand  string       `json:"documentationCommand"`
	Actions               []string     `json:"actions"`
}

type DomainView struct {
	Domain
	Package             string `json:"package"`
	PackageAvailability string `json:"packageAvailability"`
}

func Load() (Catalog, error) {
	var catalog Catalog
	if err := json.Unmarshal(catalogJSON, &catalog); err != nil {
		return Catalog{}, fmt.Errorf("embedded Domain catalog is invalid: %w", err)
	}
	if catalog.SchemaVersion == "" || len(catalog.Categories) == 0 || len(catalog.Domains) == 0 {
		return Catalog{}, fmt.Errorf("embedded Domain catalog is incomplete")
	}
	categories := make(map[string]struct{}, len(catalog.Categories))
	for _, category := range catalog.Categories {
		name := strings.TrimSpace(category.Name)
		if name == "" || strings.TrimSpace(category.Label) == "" || strings.TrimSpace(category.Description) == "" {
			return Catalog{}, fmt.Errorf("embedded Domain catalog contains an incomplete category")
		}
		if _, duplicate := categories[name]; duplicate {
			return Catalog{}, fmt.Errorf("embedded Domain catalog contains duplicate category %q", name)
		}
		categories[name] = struct{}{}
	}
	domains := make(map[string]struct{}, len(catalog.Domains))
	for _, domain := range catalog.Domains {
		key := strings.ToLower(strings.TrimSpace(domain.Name))
		if key == "" || strings.TrimSpace(domain.Backend) == "" || strings.TrimSpace(domain.Transport) == "" ||
			strings.TrimSpace(domain.DocumentationCommand) == "" || len(domain.Actions) == 0 {
			return Catalog{}, fmt.Errorf("embedded Domain catalog contains an incomplete Domain %q", domain.Name)
		}
		if _, ok := categories[domain.Category]; !ok {
			return Catalog{}, fmt.Errorf("embedded Domain catalog Domain %q has unknown category %q", domain.Name, domain.Category)
		}
		if _, duplicate := domains[key]; duplicate {
			return Catalog{}, fmt.Errorf("embedded Domain catalog contains duplicate Domain %q", domain.Name)
		}
		domains[key] = struct{}{}
		if domain.Availability.MSAPI == "" || domain.Availability.UIAPI == "" || domain.Availability.Universal == "" {
			return Catalog{}, fmt.Errorf("embedded Domain catalog Domain %q has incomplete package availability", domain.Name)
		}
	}
	return catalog, nil
}

func Find(name string) (Domain, bool) {
	catalog, err := Load()
	if err != nil {
		return Domain{}, false
	}
	for _, domain := range catalog.Domains {
		if strings.EqualFold(strings.TrimSpace(name), domain.Name) {
			return domain, true
		}
	}
	return Domain{}, false
}

func PackageKind() string {
	name := strings.ToLower(strings.TrimSpace(edition.PackageName))
	switch {
	case strings.HasSuffix(name, "-uiapi"):
		return "uiapi"
	case strings.HasSuffix(name, "-universal"):
		return "universal"
	default:
		return "msapi"
	}
}

func AvailabilityFor(domain Domain, packageKind string) string {
	switch strings.ToLower(strings.TrimSpace(packageKind)) {
	case "uiapi":
		return domain.Availability.UIAPI
	case "universal":
		return domain.Availability.Universal
	default:
		return domain.Availability.MSAPI
	}
}

func View(domain Domain) DomainView {
	kind := PackageKind()
	return DomainView{Domain: domain, Package: edition.PackageName,
		PackageAvailability: AvailabilityFor(domain, kind)}
}

func WriteList(writer io.Writer) error {
	catalog, err := Load()
	if err != nil {
		return err
	}
	views := make([]DomainView, 0, len(catalog.Domains))
	for _, domain := range catalog.Domains {
		views = append(views, View(domain))
	}
	return writeJSON(writer, map[string]any{
		"schemaVersion": catalog.SchemaVersion,
		"package":       edition.PackageName,
		"categories":    catalog.Categories,
		"domains":       views,
	})
}

func WriteDetail(writer io.Writer, name string) error {
	domain, ok := Find(name)
	if !ok {
		return fmt.Errorf("unknown Domain %q; run cloudcc domains", name)
	}
	return writeJSON(writer, View(domain))
}

func WriteHelp(writer io.Writer) error {
	catalog, err := Load()
	if err != nil {
		return err
	}
	fmt.Fprintln(writer, "Domain discovery:")
	fmt.Fprintln(writer, "  cloudcc domains")
	fmt.Fprintln(writer, "  cloudcc domain <name>")
	for _, category := range catalog.Categories {
		var items []string
		for _, domain := range catalog.Domains {
			if domain.Category != category.Name {
				continue
			}
			availability := AvailabilityFor(domain, PackageKind())
			label := domain.Name
			if availability != "enabled" && availability != "provider-adapter" && availability != "provider-selected" {
				label += " [" + availability + "]"
			}
			items = append(items, label)
		}
		if len(items) > 0 {
			fmt.Fprintf(writer, "  %s: %s\n", category.Label, strings.Join(items, ", "))
		}
	}
	return nil
}

func writeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
