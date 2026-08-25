package projectoutputs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type InitResult struct {
	Status      string   `json:"status"`
	ProjectPath string   `json:"projectPath"`
	ProjectCode string   `json:"projectCode"`
	Created     []string `json:"created"`
	Skipped     []string `json:"skipped"`
	NextActions []string `json:"nextActions"`
}

func Init(projectPath string, projectCode string) (InitResult, error) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return InitResult{}, err
	}
	projectCode = strings.TrimSpace(projectCode)
	result := InitResult{Status: "initialized", ProjectPath: absPath, ProjectCode: projectCode}
	info, err := os.Stat(absPath)
	if err != nil || !info.IsDir() {
		return result, fmt.Errorf("project-outputs init requires an existing project directory: %s", absPath)
	}
	if !validID(projectCode) {
		return result, fmt.Errorf("projectCode must use letters, digits, dot, underscore, or hyphen")
	}

	files := outputScaffoldFiles(projectCode)
	for relPath, content := range files {
		output := filepath.Join(absPath, filepath.FromSlash(relPath))
		if _, statErr := os.Stat(output); statErr == nil {
			result.Skipped = append(result.Skipped, relPath)
			continue
		} else if !os.IsNotExist(statErr) {
			return result, statErr
		}
		if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
			return result, err
		}
		if err := os.WriteFile(output, []byte(content), 0o644); err != nil {
			return result, err
		}
		result.Created = append(result.Created, relPath)
	}
	sort.Strings(result.Created)
	sort.Strings(result.Skipped)
	result.NextActions = []string{
		"Record only contract-, customer-, or project-required outputs in outputs/output-manifest.json.",
		"Create each output directory only after the deliverable is confirmed; do not pre-create fixed document or tool categories.",
		"Keep credentials and raw customer data outside outputs, and record SHA-256 for approved or delivered local artifacts.",
		"Run cloudcc doctor project-outputs <projectPath> before handoff or release evidence freeze.",
	}
	return result, nil
}

func outputScaffoldFiles(projectCode string) map[string]string {
	manifest, _ := json.MarshalIndent(Manifest{
		SchemaVersion: SchemaVersion,
		ProjectCode:   projectCode,
		Outputs:       []OutputEntry{},
	}, "", "  ")
	return map[string]string{
		"outputs/README.md":            outputsReadme(),
		"outputs/00-output-index.md":   outputIndex(),
		"outputs/output-manifest.json": string(append(manifest, '\n')),
	}
}

func outputsReadme() string {
	return `# Project Outputs

This directory is the governed project delivery and handoff boundary. It may contain required documents, project-specific tools, sanitized data packages, deployment packages, training materials, integration packages, or other explicitly requested outputs.

Only README.md, 00-output-index.md, and output-manifest.json are fixed. Create an output directory only after the contract, customer, or project requirement is confirmed; there is no fixed content template.

## Boundaries

- docs/delivery contains implementation design, matrices, plans, testing, cutover, and release governance.
- test-assets contains reusable test definitions, fixtures, assertions, and automation.
- evidence contains real decisions, runs, logs, screenshots, readbacks, and approval evidence.
- outputs contains the actual items intended for customer or downstream-team delivery; it references evidence instead of copying it.
- dist, build, tmp, caches, and ordinary generated output are not project deliverables.

Do not ignore the entire outputs directory. Keep registries, manifests, source, configuration examples, and small governed artifacts in version control. Large artifacts may use a controlled artifact store when output-manifest.json records the stable external reference, version, and digest.

## Tool outputs

Project-specific operations, migration, validation, or integration tools must document purpose, inputs, outputs, dependencies, dry-run or precheck, idempotency, failure handling, backup, rollback, tests, version, and SHA-256. Reusable cross-project tools belong in a dedicated tools module or repository; outputs records only the project-adopted frozen package.

Never store passwords, access tokens, private keys, reusable secrets, raw customer data, or live environment credentials here.
`
}

func outputIndex() string {
	return `# Project Output Index

The machine source of truth is output-manifest.json. Keep this human-readable index aligned without duplicating full document, tool, evidence, or release content.

| Output ID | Kind | Title | Requirement source | Audience | Owner role | Format | Status | Working source | Release or snapshot | Approval or evidence |
|---|---|---|---|---|---|---|---|---|---|---|
`
}
