package projectlayout

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Issue struct {
	Code    string `json:"code"`
	Path    string `json:"path"`
	Message string `json:"message"`
}
type Report struct {
	Status     string         `json:"status"`
	ReadOnly   bool           `json:"readOnly"`
	Registered bool           `json:"registered"`
	ScriptRoot string         `json:"clientScripts,omitempty"`
	Entries    []Entry        `json:"entries"`
	Errors     []Issue        `json:"errors"`
	Warnings   []Issue        `json:"warnings"`
	Summary    map[string]int `json:"summary"`
}

func git(project string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", project}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
	b, err := cmd.Output()
	return string(b), err
}

func WriteDoctor(project string, out io.Writer) error {
	r, err := Doctor(project)
	if e := json.NewEncoder(out).Encode(r); e != nil {
		return e
	}
	return err
}

func Doctor(project string) (Report, error) {
	r := Report{Status: "passed", ReadOnly: true, Entries: []Entry{}, Errors: []Issue{}, Warnings: []Issue{}, Summary: map[string]int{}}
	add := func(hard bool, code, path, msg string) {
		i := Issue{code, path, msg}
		if hard {
			r.Errors = append(r.Errors, i)
		} else {
			r.Warnings = append(r.Warnings, i)
		}
	}
	finish := func() (Report, error) {
		r.Summary["errorCount"] = len(r.Errors)
		r.Summary["warningCount"] = len(r.Warnings)
		r.Summary["rootEntryCount"] = len(r.Entries)
		if len(r.Errors) > 0 {
			r.Status = "failed"
			return r, fmt.Errorf("project layout check found %d error(s)", len(r.Errors))
		}
		if len(r.Warnings) > 0 {
			r.Status = "passed_with_warnings"
		}
		return r, nil
	}
	abs, err := filepath.Abs(project)
	if err != nil {
		add(true, "project_unavailable", ".", "Cannot resolve project path")
		return finish()
	}
	abs, err = filepath.EvalSymlinks(abs)
	if err != nil {
		add(true, "project_unavailable", ".", "Project path does not exist or cannot be resolved")
		return finish()
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		add(true, "project_unavailable", ".", "Project path is not a readable directory")
		return finish()
	}
	reg, registered, err := Load(abs)
	r.Registered = registered
	if err != nil {
		add(true, "layout_registry_invalid", RegistryPath, err.Error())
		return finish()
	}
	if !registered {
		add(false, "layout_registry_missing", RegistryPath, "Generic advisory inventory only; project AGENTS and approved allowlist remain authoritative. Use init project-layout only when authorized to adopt a registry.")
	}
	allowed := map[string]Entry{}
	for _, e := range reg.Entries {
		allowed[e.Name] = e
	}
	tracked, staged := map[string]bool{}, map[string]bool{}
	gitRoot, gitErr := git(abs, "rev-parse", "--show-toplevel")
	gitReady := false
	if gitErr == nil {
		resolved, e := filepath.EvalSymlinks(strings.TrimSpace(gitRoot))
		gitReady = e == nil && resolved == abs
	}
	if gitReady {
		for _, item := range []struct {
			args []string
			dest map[string]bool
		}{{[]string{"ls-files", "-z"}, tracked}, {[]string{"diff", "--cached", "--name-only", "--diff-filter=ACMR", "-z"}, staged}} {
			text, e := git(abs, item.args...)
			if e != nil {
				add(true, "git_inventory_failed", ".git", "Cannot inspect tracked or staged paths")
				continue
			}
			for _, p := range strings.Split(text, "\x00") {
				if p != "" {
					item.dest[p] = true
				}
			}
		}
	} else {
		add(false, "git_inventory_unavailable", ".git", "No Git repository rooted at this project; tracking, staged paths and ignore rules were not verified.")
	}
	names := map[string]bool{}
	for _, e := range entries {
		names[e.Name()] = true
	}
	for p := range tracked {
		names[strings.Split(p, "/")[0]] = true
	}
	for p := range staged {
		names[strings.Split(p, "/")[0]] = true
	}
	keys := []string{}
	for name := range names {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	for _, name := range keys {
		if name == ".git" {
			if tracked[name] || staged[name] {
				add(true, "git_control_tracked", name, "Git control files must not be tracked")
			}
			if fi, e := os.Lstat(filepath.Join(abs, name)); e == nil && (!fi.IsDir() || fi.Mode()&os.ModeSymlink != 0) && !gitReady {
				add(true, "git_control_invalid", name, "Expected a Git-owned directory or linked-worktree control file")
			}
			continue
		}
		e, ok := allowed[name]
		if !ok {
			e = Entry{Name: name, Role: "unregistered"}
			add(registered, "root_entry_unregistered", name, "Classify purpose, owner/lifecycle in project rules and register this root entry before adoption.")
		}
		r.Entries = append(r.Entries, e)
		path, safeErr := SafePath(abs, name)
		if safeErr != nil {
			add(true, "path_unsafe", name, safeErr.Error())
			continue
		}
		fi, statErr := os.Stat(path)
		if statErr != nil && !os.IsNotExist(statErr) {
			add(true, "path_unreadable", name, "Cannot inspect root entry")
			continue
		}
		if ok && statErr == nil && ((e.Kind == "directory") != fi.IsDir()) {
			add(true, "root_kind_mismatch", name, "Entry type does not match registry")
		}
		if ok && e.Tracking == "local" {
			shared := false
			for p := range tracked {
				if p == name || strings.HasPrefix(p, name+"/") {
					shared = true
					break
				}
			}
			for p := range staged {
				if p == name || strings.HasPrefix(p, name+"/") {
					shared = true
					break
				}
			}
			if shared {
				add(true, "local_entry_tracked", name, "Local-only content is tracked or staged")
			}
			if statErr == nil && gitReady {
				if _, e := git(abs, "check-ignore", "-q", "--", name); e != nil {
					add(true, "local_entry_not_ignored", name, "Local-only entry must be ignored by Git")
				}
			}
		}
	}
	for _, e := range reg.Entries {
		if e.Required {
			p, safeErr := SafePath(abs, e.Name)
			if safeErr != nil {
				continue
			}
			if _, err := os.Stat(p); os.IsNotExist(err) {
				add(true, "required_root_missing", e.Name, "Required project root entry is missing")
			}
		}
	}
	e := checkScriptDirectory(abs)
	if e != nil {
		add(true, "script_path_invalid", ClientScriptsPath, e.Error())
	} else {
		r.ScriptRoot = ClientScriptsPath
	}
	if exists(abs, ".claw") && exists(abs, ".ai-dev") {
		add(false, "state_roots_overlap", ".claw|.ai-dev", "Declare one authoritative state root; do not update both")
	}
	if !exists(abs, reg.ReferenceMaterials+"/README.md") {
		add(false, "reference_index_missing", reg.ReferenceMaterials+"/README.md", "Source materials need a provenance/adoption index; reconcile project rules before creating or relocating it")
	}
	if exists(abs, "deployment") && exists(abs, "docs/specs/cloudcc") {
		add(false, "execution_inputs_overlap", "deployment|docs/specs/cloudcc", "Review execution inputs versus deployment packages and backups; keep one authoritative input path")
	}
	if exists(abs, "lib") && exists(abs, "backend/lib") {
		items, _ := os.ReadDir(filepath.Join(abs, "lib"))
		same := 0
		for _, item := range items {
			if !item.IsDir() && strings.HasSuffix(item.Name(), ".jar") {
				left, le := SafePath(abs, "lib/"+item.Name())
				right, re := SafePath(abs, "backend/lib/"+item.Name())
				if le != nil || re != nil {
					continue
				}
				a, ae := fileDigest(left)
				b, be := fileDigest(right)
				if ae == nil && be == nil && a == b {
					same++
				}
			}
		}
		if same > 0 {
			add(false, "duplicate_dependency_files", "lib|backend/lib", fmt.Sprintf("%d identical JAR pairs; inspect consumers before consolidation", same))
		}
	}
	for _, rel := range []string{".claw/current-status.md", ".claw/task-board.md", ".ai-dev/current-status.md", ".ai-dev/task-board.md"} {
		p, e := SafePath(abs, rel)
		if e == nil {
			if fi, e := os.Stat(p); e == nil && fi.Size() > 100*1024 {
				add(false, "state_index_large", rel, "State entry exceeds 100 KiB; move historical detail to task cards/archive")
			}
		}
	}
	delivery, e := SafePath(abs, "docs/delivery")
	if e == nil {
		roots, _ := os.ReadDir(delivery)
		for _, d := range roots {
			if !d.IsDir() {
				continue
			}
			base := "docs/delivery/" + d.Name()
			for _, rel := range requiredDeliveryPaths() {
				if !exists(abs, base+"/"+rel) {
					add(false, "delivery_entry_missing", base+"/"+rel, "Fixed delivery entry is missing or renamed; use a stable entry linking the authoritative content, and document partial-module scope")
				}
			}
			if exists(abs, base+"/03-detailed-design") {
				add(false, "legacy_design_directory", base+"/03-detailed-design", "Keep an explicit archive/replacement index; current module design belongs to 03-module-design or its referenced FEAT")
			}
			p, e := SafePath(abs, base+"/08-release-evidence")
			if e == nil {
				n := 0
				_ = filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
					if err != nil {
						return nil
					}
					if d.IsDir() && d.Name() == "snapshots" {
						return filepath.SkipDir
					}
					if !d.IsDir() {
						switch filepath.Ext(path) {
						case ".json", ".log", ".java":
							n++
						}
					}
					return nil
				})
				if n > 0 {
					add(false, "raw_evidence_in_delivery", base+"/08-release-evidence", fmt.Sprintf("%d JSON/log/Java files need review: release indexes should reference execution evidence; retain historical traceability", n))
				}
			}
		}
		if len(roots) > 1 {
			add(false, "delivery_scope_review", "docs/delivery", "Declare primary project versus module/independent delivery scope; test-governance currently requires one adopted testing root")
		}
	}
	checkOutputRegistration(abs, add)
	return finish()
}

func exists(root, rel string) bool {
	p, e := SafePath(root, rel)
	if e != nil {
		return false
	}
	_, e = os.Stat(p)
	return e == nil
}
func fileDigest(path string) ([32]byte, error) {
	var result [32]byte
	f, e := os.Open(path)
	if e != nil {
		return result, e
	}
	defer f.Close()
	h := sha256.New()
	if _, e = io.Copy(h, f); e != nil {
		return result, e
	}
	copy(result[:], h.Sum(nil))
	return result, nil
}

func checkOutputRegistration(root string, add func(bool, string, string, string)) {
	p, e := SafePath(root, "outputs/output-manifest.json")
	if e != nil {
		return
	}
	b, e := os.ReadFile(p)
	if e != nil {
		return
	}
	var m struct {
		Outputs []struct {
			WorkingPaths     []string `json:"workingPaths"`
			SnapshotPaths    []string `json:"snapshotPaths"`
			ReleaseArtifacts []struct {
				Path string `json:"path"`
			} `json:"releaseArtifacts"`
		} `json:"outputs"`
	}
	if json.Unmarshal(b, &m) != nil {
		add(false, "output_manifest_unreadable", "outputs/output-manifest.json", "Run doctor project-outputs for full manifest validation")
		return
	}
	refs := []string{}
	for _, o := range m.Outputs {
		refs = append(refs, o.WorkingPaths...)
		refs = append(refs, o.SnapshotPaths...)
		for _, a := range o.ReleaseArtifacts {
			refs = append(refs, a.Path)
		}
	}
	items, _ := os.ReadDir(filepath.Dir(p))
	for _, item := range items {
		name := item.Name()
		if name == "README.md" || name == "00-output-index.md" || name == "output-manifest.json" || name == ".DS_Store" {
			continue
		}
		rel := "outputs/" + name
		covered := false
		for _, ref := range refs {
			ref = strings.TrimSuffix(ref, "/")
			if ref == rel || strings.HasPrefix(ref, rel+"/") {
				covered = true
				break
			}
		}
		if !covered {
			add(false, "output_entry_unregistered", rel, "Top-level output is not referenced by workingPaths, snapshotPaths or releaseArtifacts; classify historical material before delivery")
		}
	}
}

func requiredDeliveryPaths() []string {
	groups := []struct {
		dir   string
		files []string
	}{
		{"00-governance", []string{"00-delivery-index", "01-project-scope", "02-environment-config", "03-risk-issue-decision-log", "standards/00-standard-index"}},
		{"01-blueprint", []string{"00-blueprint-index", "01-business-scope", "02-end-to-end-process", "03-module-architecture", "04-phase-roadmap", "processes/00-process-index"}},
		{"02-global-modeling", []string{"00-global-modeling-index", "01-standard-environment-baseline", "02-global-object-map", "03-object-relationship-matrix", "04-global-object-field-dictionary", "05-state-machine-matrix", "06-global-select-list-catalog", "07-field-select-list-reference-matrix", "08-permission-page-impact-matrix", "09-integration-field-standard", "10-reporting-field-standard"}},
		{"03-module-design", []string{"00-module-design-index"}},
		{"04-security", []string{"00-security-index", "01-user-group-matrix", "02-role-tree-matrix", "03-profile-matrix", "04-permission-set-matrix", "05-sharing-rule-matrix", "06-field-security-matrix"}},
		{"05-integration", []string{"00-integration-index"}},
		{"06-data-migration", []string{"00-data-migration-index", "01-source-target-mapping", "02-data-cleansing-rules", "03-migration-validation-report"}},
		{"07-testing-cutover", []string{"00-testing-cutover-index", "01-uat-scenario-matrix", "02-permission-acceptance-matrix", "03-cutover-runbook", "04-rollback-runbook", "05-test-impact-matrix", "06-test-data-catalog", "07-environment-account-matrix", "08-requirement-test-traceability"}},
		{"08-release-evidence", []string{"00-release-evidence-index", "01-msapi-plan-apply-changes", "02-production-readiness-gate", "03-post-release-verification"}},
	}
	paths := []string{}
	for _, g := range groups {
		for _, f := range g.files {
			paths = append(paths, g.dir+"/"+f+".md")
		}
	}
	return paths
}
