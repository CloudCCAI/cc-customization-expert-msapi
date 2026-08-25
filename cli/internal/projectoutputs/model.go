package projectoutputs

import (
	"regexp"
	"strings"
)

const SchemaVersion = "cloudcc-project-outputs/v1"

var safeIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
var sha256Pattern = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)

var allowedKinds = map[string]bool{
	"document":            true,
	"tool":                true,
	"data-package":        true,
	"deployment-package":  true,
	"training-package":    true,
	"integration-package": true,
	"other":               true,
}

var allowedStatuses = map[string]bool{
	"planned":   true,
	"draft":     true,
	"review":    true,
	"approved":  true,
	"delivered": true,
	"retired":   true,
}

type ReleaseArtifact struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256,omitempty"`
}

type OutputEntry struct {
	OutputID          string            `json:"outputId"`
	Kind              string            `json:"kind"`
	Title             string            `json:"title"`
	Status            string            `json:"status"`
	OwnerRole         string            `json:"ownerRole,omitempty"`
	RequirementSource string            `json:"requirementSource,omitempty"`
	Audience          []string          `json:"audience,omitempty"`
	Formats           []string          `json:"formats,omitempty"`
	WorkingPaths      []string          `json:"workingPaths,omitempty"`
	ReleaseArtifacts  []ReleaseArtifact `json:"releaseArtifacts,omitempty"`
	SnapshotPaths     []string          `json:"snapshotPaths,omitempty"`
	EvidenceRefs      []string          `json:"evidenceRefs,omitempty"`
	ExternalRefs      []string          `json:"externalRefs,omitempty"`
}

type Manifest struct {
	SchemaVersion string        `json:"schemaVersion"`
	ProjectCode   string        `json:"projectCode"`
	Outputs       []OutputEntry `json:"outputs"`
}

func validID(value string) bool {
	return safeIDPattern.MatchString(strings.TrimSpace(value))
}

func normalized(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
