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
	Resources     []Resource `json:"resources"`
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
	Kind                  string       `json:"kind,omitempty"`
	Category              string       `json:"category"`
	Backend               string       `json:"backend"`
	Transport             string       `json:"transport"`
	Availability          Availability `json:"availability"`
	MinimumBackendVersion string       `json:"minimumBackendVersion,omitempty"`
	DocumentationCommand  string       `json:"documentationCommand"`
	Actions               []string     `json:"actions"`
	ActionSemantics       string       `json:"actionSemantics,omitempty"`
	ResourceNames         []string     `json:"resources,omitempty"`
	Routes                []Route      `json:"routes,omitempty"`
}

type DomainView struct {
	Domain
	Package              string            `json:"package"`
	PackageAvailability  string            `json:"packageAvailability"`
	ResourceCount        int               `json:"resourceCount,omitempty"`
	Resources            []ResourceSummary `json:"resourceDetails,omitempty"`
	EffectiveRoute       *Route            `json:"effectiveRoute,omitempty"`
	EffectiveRouteReason string            `json:"effectiveRouteReason,omitempty"`
}

type Route struct {
	Provider     string `json:"provider"`
	Backend      string `json:"backend"`
	Transport    string `json:"transport"`
	Availability string `json:"availability"`
}

type Documentation struct {
	Introduction string `json:"introduction"`
	Devguide     string `json:"devguide"`
}

type Resource struct {
	Name                  string        `json:"name"`
	Label                 string        `json:"label"`
	Kind                  string        `json:"kind"`
	Parent                string        `json:"parent"`
	Category              string        `json:"category"`
	Aliases               []string      `json:"aliases,omitempty"`
	Availability          Availability  `json:"availability"`
	MinimumBackendVersion string        `json:"minimumBackendVersion,omitempty"`
	Routes                []Route       `json:"routes"`
	Actions               []string      `json:"actions"`
	Commands              []string      `json:"commands"`
	Documentation         Documentation `json:"documentation"`
}

type ResourceSummary struct {
	Name                 string   `json:"name"`
	Label                string   `json:"label"`
	Aliases              []string `json:"aliases,omitempty"`
	Actions              []string `json:"actions"`
	DocumentationCommand string   `json:"documentationCommand"`
}

type ResourceView struct {
	Resource
	Package              string `json:"package"`
	PackageAvailability  string `json:"packageAvailability"`
	EffectiveRoute       *Route `json:"effectiveRoute,omitempty"`
	EffectiveRouteReason string `json:"effectiveRouteReason,omitempty"`
}

type ListOptions struct {
	Category string
	Format   string
}

func Load() (Catalog, error) {
	var catalog Catalog
	if err := json.Unmarshal(catalogJSON, &catalog); err != nil {
		return Catalog{}, fmt.Errorf("embedded Domain catalog is invalid: %w", err)
	}
	if catalog.SchemaVersion == "" || len(catalog.Categories) == 0 || len(catalog.Domains) == 0 || len(catalog.Resources) == 0 {
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
	resources := make(map[string]struct{}, len(catalog.Resources))
	aliases := make(map[string]string)
	for _, resource := range catalog.Resources {
		key := strings.ToLower(strings.TrimSpace(resource.Name))
		if key == "" || resource.Kind != "resource" || strings.TrimSpace(resource.Parent) == "" ||
			strings.TrimSpace(resource.Category) == "" || len(resource.Routes) == 0 || len(resource.Actions) == 0 ||
			len(resource.Commands) == 0 || strings.TrimSpace(resource.Documentation.Introduction) == "" ||
			strings.TrimSpace(resource.Documentation.Devguide) == "" {
			return Catalog{}, fmt.Errorf("embedded Domain catalog contains an incomplete resource %q", resource.Name)
		}
		if _, duplicate := resources[key]; duplicate {
			return Catalog{}, fmt.Errorf("embedded Domain catalog contains duplicate resource %q", resource.Name)
		}
		if _, conflict := domains[key]; conflict {
			return Catalog{}, fmt.Errorf("embedded Domain catalog resource %q conflicts with a top-level Domain", resource.Name)
		}
		resources[key] = struct{}{}
		for _, alias := range append([]string{resource.Name}, resource.Aliases...) {
			aliasKey := strings.ToLower(strings.TrimSpace(alias))
			if aliasKey == "" {
				return Catalog{}, fmt.Errorf("embedded Domain catalog resource %q has an empty alias", resource.Name)
			}
			if existing, duplicate := aliases[aliasKey]; duplicate && existing != resource.Name {
				return Catalog{}, fmt.Errorf("embedded Domain catalog alias %q is shared by %q and %q", alias, existing, resource.Name)
			}
			aliases[aliasKey] = resource.Name
		}
	}
	for _, domain := range catalog.Domains {
		for _, resourceName := range domain.ResourceNames {
			resource, ok := findResourceIn(catalog, resourceName)
			if !ok || resource.Parent != domain.Name {
				return Catalog{}, fmt.Errorf("embedded Domain catalog Domain %q references invalid child resource %q", domain.Name, resourceName)
			}
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

func FindResource(name string) (Resource, bool) {
	catalog, err := Load()
	if err != nil {
		return Resource{}, false
	}
	wanted := strings.TrimSpace(name)
	for _, resource := range catalog.Resources {
		if strings.EqualFold(wanted, resource.Name) {
			return resource, true
		}
		for _, alias := range resource.Aliases {
			if strings.EqualFold(wanted, alias) {
				return resource, true
			}
		}
	}
	return Resource{}, false
}

func findResourceIn(catalog Catalog, name string) (Resource, bool) {
	for _, resource := range catalog.Resources {
		if strings.EqualFold(strings.TrimSpace(name), resource.Name) {
			return resource, true
		}
	}
	return Resource{}, false
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
	view := DomainView{Domain: domain, Package: edition.PackageName,
		PackageAvailability: AvailabilityFor(domain, kind)}
	catalog, _ := Load()
	for _, name := range domain.ResourceNames {
		resource, ok := findResourceIn(catalog, name)
		if !ok {
			continue
		}
		view.Resources = append(view.Resources, ResourceSummary{
			Name: resource.Name, Label: resource.Label, Aliases: resource.Aliases,
			Actions: resource.Actions, DocumentationCommand: resource.Documentation.Introduction,
		})
	}
	view.ResourceCount = len(view.Resources)
	view.EffectiveRoute, view.EffectiveRouteReason = effectiveRoute(domain.Routes, kind)
	return view
}

func ResourceDetail(resource Resource) ResourceView {
	kind := PackageKind()
	view := ResourceView{Resource: resource, Package: edition.PackageName,
		PackageAvailability: availabilityForResource(resource, kind)}
	view.EffectiveRoute, view.EffectiveRouteReason = effectiveRoute(resource.Routes, kind)
	return view
}

func availabilityForResource(resource Resource, packageKind string) string {
	switch strings.ToLower(strings.TrimSpace(packageKind)) {
	case "uiapi":
		return resource.Availability.UIAPI
	case "universal":
		return resource.Availability.Universal
	default:
		return resource.Availability.MSAPI
	}
}

func effectiveRoute(routes []Route, packageKind string) (*Route, string) {
	provider := strings.ToLower(strings.TrimSpace(packageKind))
	for _, route := range routes {
		if route.Provider == "all" {
			copy := route
			return &copy, ""
		}
	}
	if provider == "universal" {
		return nil, "provider selection requires project context"
	}
	for _, route := range routes {
		if route.Provider == provider {
			copy := route
			return &copy, ""
		}
	}
	return nil, "no route is available for this package"
}

func WriteList(writer io.Writer) error {
	return WriteListWithOptions(writer, ListOptions{Format: "json"})
}

func WriteListWithOptions(writer io.Writer, options ListOptions) error {
	catalog, err := Load()
	if err != nil {
		return err
	}
	if category := strings.TrimSpace(options.Category); category != "" {
		known := false
		for _, item := range catalog.Categories {
			if strings.EqualFold(category, item.Name) {
				known = true
				break
			}
		}
		if !known {
			return fmt.Errorf("unknown Domain category %q", category)
		}
	}
	views := make([]DomainView, 0, len(catalog.Domains))
	for _, domain := range catalog.Domains {
		if strings.TrimSpace(options.Category) != "" && !strings.EqualFold(strings.TrimSpace(options.Category), domain.Category) {
			continue
		}
		views = append(views, View(domain))
	}
	if strings.EqualFold(strings.TrimSpace(options.Format), "table") {
		fmt.Fprintln(writer, "NAME\tCATEGORY\tRESOURCES\tAVAILABILITY")
		for _, view := range views {
			fmt.Fprintf(writer, "%s\t%s\t%d\t%s\n", view.Name, view.Category, view.ResourceCount, view.PackageAvailability)
		}
		return nil
	}
	return writeJSON(writer, map[string]any{
		"schemaVersion": catalog.SchemaVersion,
		"package":       edition.PackageName,
		"categories":    catalog.Categories,
		"domains":       views,
	})
}

func WriteDetail(writer io.Writer, name string) error {
	return WriteDetailWithFormat(writer, name, "json")
}

func WriteDetailWithFormat(writer io.Writer, name string, format string) error {
	domain, ok := Find(name)
	if ok {
		view := View(domain)
		if strings.EqualFold(strings.TrimSpace(format), "table") {
			fmt.Fprintln(writer, "NAME\tKIND\tCATEGORY\tRESOURCES\tAVAILABILITY")
			fmt.Fprintf(writer, "%s\t%s\t%s\t%d\t%s\n", view.Name, view.Kind, view.Category, view.ResourceCount, view.PackageAvailability)
			for _, resource := range view.Resources {
				fmt.Fprintf(writer, "  %s\tresource\t%s\t-\t%s\n", resource.Name, view.Category, view.PackageAvailability)
			}
			return nil
		}
		return writeJSON(writer, view)
	}
	resource, ok := FindResource(name)
	if ok {
		view := ResourceDetail(resource)
		if strings.EqualFold(strings.TrimSpace(format), "table") {
			fmt.Fprintln(writer, "NAME\tPARENT\tCATEGORY\tAVAILABILITY\tACTIONS")
			fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n", view.Name, view.Parent, view.Category, view.PackageAvailability, strings.Join(view.Actions, ","))
			fmt.Fprintln(writer, "COMMANDS")
			for _, command := range view.Commands {
				fmt.Fprintln(writer, command)
			}
			return nil
		}
		return writeJSON(writer, view)
	}
	return fmt.Errorf("unknown Domain or resource %q; run cloudcc domains", name)
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
	fmt.Fprintln(writer, "  Use cloudcc domain <group-or-resource> for child resources, commands, routes, and documentation.")
	return nil
}

func writeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
