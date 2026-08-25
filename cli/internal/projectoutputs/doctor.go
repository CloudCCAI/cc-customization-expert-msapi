package projectoutputs

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type DoctorIssue struct {
	Code    string `json:"code"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
}

type DoctorReport struct {
	Status      string         `json:"status"`
	ProjectPath string         `json:"projectPath"`
	ProjectCode string         `json:"projectCode,omitempty"`
	ReadOnly    bool           `json:"readOnly"`
	Errors      []DoctorIssue  `json:"errors"`
	Warnings    []DoctorIssue  `json:"warnings"`
	Summary     map[string]int `json:"summary"`
}

func WriteDoctor(projectPath string, stdout io.Writer) error {
	report, err := Doctor(projectPath)
	if encodeErr := json.NewEncoder(stdout).Encode(report); encodeErr != nil {
		return encodeErr
	}
	return err
}

func Doctor(projectPath string) (DoctorReport, error) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return DoctorReport{}, err
	}
	report := DoctorReport{
		Status:      "passed",
		ProjectPath: absPath,
		ReadOnly:    true,
		Summary: map[string]int{
			"outputCount":   0,
			"artifactCount": 0,
			"errorCount":    0,
			"warningCount":  0,
		},
	}
	outputsRoot := filepath.Join(absPath, "outputs")
	if _, err := os.Stat(outputsRoot); os.IsNotExist(err) {
		report.Status = "not_adopted"
		return report, nil
	} else if err != nil {
		return report, err
	}

	for _, rel := range []string{"outputs/README.md", "outputs/00-output-index.md", "outputs/output-manifest.json"} {
		if _, err := os.Stat(filepath.Join(absPath, filepath.FromSlash(rel))); err != nil {
			report.Errors = append(report.Errors, DoctorIssue{Code: "required_output_asset_missing", Path: rel, Message: "required project outputs governance file is missing"})
		}
	}

	manifestPath := filepath.Join(outputsRoot, "output-manifest.json")
	manifest, manifestErr := readManifest(manifestPath)
	if manifestErr != nil {
		report.Errors = append(report.Errors, DoctorIssue{Code: "output_manifest_invalid", Path: "outputs/output-manifest.json", Message: manifestErr.Error()})
	} else {
		report.ProjectCode = manifest.ProjectCode
		validateManifest(absPath, manifest, &report)
	}
	scanSensitiveOutputs(outputsRoot, &report)

	report.Summary["errorCount"] = len(report.Errors)
	report.Summary["warningCount"] = len(report.Warnings)
	if len(report.Errors) > 0 {
		report.Status = "failed"
		return report, fmt.Errorf("project outputs governance validation failed with %d error(s)", len(report.Errors))
	}
	return report, nil
}

func readManifest(path string) (Manifest, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	var manifest Manifest
	if err := json.Unmarshal(b, &manifest); err != nil {
		return Manifest{}, err
	}
	if manifest.SchemaVersion != SchemaVersion {
		return manifest, fmt.Errorf("schemaVersion must be %s", SchemaVersion)
	}
	if !validID(manifest.ProjectCode) {
		return manifest, fmt.Errorf("projectCode must use letters, digits, dot, underscore, or hyphen")
	}
	return manifest, nil
}

func validateManifest(projectRoot string, manifest Manifest, report *DoctorReport) {
	report.Summary["outputCount"] = len(manifest.Outputs)
	seen := map[string]bool{}
	for index, output := range manifest.Outputs {
		basePath := fmt.Sprintf("outputs/output-manifest.json#/outputs/%d", index)
		if !validID(output.OutputID) {
			report.Errors = append(report.Errors, DoctorIssue{Code: "output_id_invalid", Path: basePath, Message: "outputId must use letters, digits, dot, underscore, or hyphen"})
		} else if seen[output.OutputID] {
			report.Errors = append(report.Errors, DoctorIssue{Code: "output_id_duplicate", Path: basePath, Message: "outputId must be unique"})
		}
		seen[output.OutputID] = true
		kind := normalized(output.Kind)
		if !allowedKinds[kind] {
			report.Errors = append(report.Errors, DoctorIssue{Code: "output_kind_invalid", Path: basePath, Message: "kind is not supported"})
		}
		status := normalized(output.Status)
		if !allowedStatuses[status] {
			report.Errors = append(report.Errors, DoctorIssue{Code: "output_status_invalid", Path: basePath, Message: "status is not supported"})
		}
		if strings.TrimSpace(output.Title) == "" {
			report.Errors = append(report.Errors, DoctorIssue{Code: "output_title_missing", Path: basePath, Message: "title is required"})
		}

		for _, path := range append(append([]string{}, output.WorkingPaths...), append(output.SnapshotPaths, output.EvidenceRefs...)...) {
			validateReferencePath(projectRoot, path, status, basePath, report)
		}
		for _, artifact := range output.ReleaseArtifacts {
			report.Summary["artifactCount"]++
			if !validateReferencePath(projectRoot, artifact.Path, status, basePath, report) {
				continue
			}
			artifactPath := filepath.Join(projectRoot, filepath.FromSlash(filepath.ToSlash(filepath.Clean(artifact.Path))))
			if status == "approved" || status == "delivered" {
				if !sha256Pattern.MatchString(strings.TrimSpace(artifact.SHA256)) {
					report.Errors = append(report.Errors, DoctorIssue{Code: "release_sha256_required", Path: artifact.Path, Message: "approved or delivered local artifacts require a SHA-256 digest"})
					continue
				}
			}
			if strings.TrimSpace(artifact.SHA256) != "" {
				actual, err := fileSHA256(artifactPath)
				if err == nil && !strings.EqualFold(actual, artifact.SHA256) {
					report.Errors = append(report.Errors, DoctorIssue{Code: "release_sha256_mismatch", Path: artifact.Path, Message: "release artifact SHA-256 does not match the manifest"})
				}
			}
		}
		if (status == "approved" || status == "delivered") && len(output.ReleaseArtifacts) == 0 && len(output.ExternalRefs) == 0 {
			report.Errors = append(report.Errors, DoctorIssue{Code: "approved_output_release_missing", Path: basePath, Message: "approved or delivered output requires a local release artifact or external reference"})
		}
	}
}

func validateReferencePath(projectRoot string, value string, status string, manifestPath string, report *DoctorReport) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		report.Errors = append(report.Errors, DoctorIssue{Code: "output_path_empty", Path: manifestPath, Message: "local reference path cannot be empty"})
		return false
	}
	clean := filepath.Clean(filepath.FromSlash(value))
	if filepath.IsAbs(clean) || filepath.VolumeName(clean) != "" || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		report.Errors = append(report.Errors, DoctorIssue{Code: "output_path_unsafe", Path: value, Message: "local reference must be a project-relative path without traversal"})
		return false
	}
	fullPath := filepath.Join(projectRoot, clean)
	if _, err := os.Stat(fullPath); err != nil {
		issue := DoctorIssue{Code: "output_reference_missing", Path: filepath.ToSlash(value), Message: "referenced local output path does not exist"}
		if status == "approved" || status == "delivered" {
			report.Errors = append(report.Errors, issue)
		} else {
			report.Warnings = append(report.Warnings, issue)
		}
		return false
	}
	resolvedRoot, rootErr := filepath.EvalSymlinks(projectRoot)
	resolvedPath, pathErr := filepath.EvalSymlinks(fullPath)
	if rootErr != nil || pathErr != nil || !pathWithinRoot(resolvedRoot, resolvedPath) {
		report.Errors = append(report.Errors, DoctorIssue{Code: "output_path_symlink_escape", Path: filepath.ToSlash(value), Message: "local reference must remain inside the project after resolving symbolic links"})
		return false
	}
	return true
}

func pathWithinRoot(root string, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func fileSHA256(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func scanSensitiveOutputs(root string, report *DoctorReport) {
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		name := strings.ToLower(entry.Name())
		rel, _ := filepath.Rel(filepath.Dir(root), path)
		rel = filepath.ToSlash(rel)
		if name == ".env" || strings.HasSuffix(name, ".pem") || strings.HasSuffix(name, ".key") || strings.Contains(name, "credential") {
			report.Errors = append(report.Errors, DoctorIssue{Code: "sensitive_output_file_forbidden", Path: rel, Message: "outputs must not contain credential or private-key files"})
			return nil
		}
		if strings.HasSuffix(name, ".json") {
			b, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			var value any
			if json.Unmarshal(b, &value) == nil && containsSecretValue(value) {
				report.Errors = append(report.Errors, DoctorIssue{Code: "sensitive_output_json_forbidden", Path: rel, Message: "outputs JSON contains a non-empty secret-like field"})
			}
		}
		return nil
	})
}

func containsSecretValue(value any) bool {
	switch current := value.(type) {
	case map[string]any:
		for key, child := range current {
			normalizedKey := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(key, "_", ""), "-", ""))
			switch normalizedKey {
			case "password", "token", "accesstoken", "secret", "privatekey", "opensecretkey":
				if text, ok := child.(string); ok && strings.TrimSpace(text) != "" {
					return true
				}
			}
			if containsSecretValue(child) {
				return true
			}
		}
	case []any:
		for _, child := range current {
			if containsSecretValue(child) {
				return true
			}
		}
	}
	return false
}
