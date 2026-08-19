package msapi

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"cloudcc-customization-expert-go/internal/config"
	"cloudcc-customization-expert-go/internal/httpclient"
)

type highCodeScanResult struct {
	Mode        string           `json:"mode"`
	Source      string           `json:"source"`
	Runtime     string           `json:"runtime"`
	ProjectPath string           `json:"projectPath"`
	Totals      highCodeTotals   `json:"totals"`
	Domains     []highCodeDomain `json:"domains"`
	Legacy      highCodeLegacy   `json:"legacy"`
}

type highCodeTotals struct {
	Domains           int `json:"domains"`
	Assets            int `json:"assets"`
	Configured        int `json:"configured"`
	Publishable       int `json:"publishable"`
	Files             int `json:"files"`
	Issues            int `json:"issues"`
	EmptyDomains      int `json:"emptyDomains"`
	MissingDomains    int `json:"missingDomains"`
	LegacyNodeScripts int `json:"legacyNodeScripts"`
}

type highCodeDomain struct {
	Domain      string          `json:"domain"`
	Path        string          `json:"path"`
	Exists      bool            `json:"exists"`
	Assets      int             `json:"assets"`
	Configured  int             `json:"configured"`
	Publishable int             `json:"publishable"`
	Files       int             `json:"files"`
	Issues      []string        `json:"issues,omitempty"`
	Items       []highCodeAsset `json:"items,omitempty"`
}

type highCodeAsset struct {
	Name        string   `json:"name"`
	Path        string   `json:"path"`
	SourceFiles []string `json:"sourceFiles,omitempty"`
	ConfigPath  string   `json:"configPath,omitempty"`
	BundlePath  string   `json:"bundlePath,omitempty"`
	Publishable bool     `json:"publishable"`
	Issues      []string `json:"issues,omitempty"`
}

type highCodeLegacy struct {
	NodeScripts []highCodeAsset `json:"nodeScripts,omitempty"`
}

type onlineHighCodeScanResult struct {
	Mode        string                 `json:"mode"`
	Source      string                 `json:"source"`
	Runtime     string                 `json:"runtime"`
	ProjectPath string                 `json:"projectPath"`
	Totals      onlineHighCodeTotals   `json:"totals"`
	Domains     []onlineHighCodeDomain `json:"domains"`
}

type onlineHighCodeTotals struct {
	Domains     int `json:"domains"`
	OnlineItems int `json:"onlineItems"`
	Passed      int `json:"passed"`
	Failed      int `json:"failed"`
	Unsupported int `json:"unsupported"`
	OutOfScope  int `json:"outOfScope"`
}

type onlineHighCodeDomain struct {
	Domain    string           `json:"domain"`
	Transport string           `json:"transport"`
	Endpoint  string           `json:"endpoint,omitempty"`
	Status    string           `json:"status"`
	Count     int              `json:"count"`
	Sample    []map[string]any `json:"sample,omitempty"`
	Issues    []string         `json:"issues,omitempty"`
}

type structuredHighCodeSpec struct {
	domain         string
	relPath        string
	sourceExts     []string
	sourceNames    []string
	requiresConfig bool
}

type onlineHighCodeSpec struct {
	domain     string
	transport  string
	base       string
	path       string
	paths      []string
	body       map[string]any
	header     map[string]any
	candidates []onlineHighCodeCandidate
}

type onlineHighCodeCandidate struct {
	transport string
	base      string
	path      string
	body      map[string]any
	header    map[string]any
}

func scanHighCodeProject(projectPath string) (highCodeScanResult, error) {
	if projectPath == "" {
		var err error
		projectPath, err = os.Getwd()
		if err != nil {
			return highCodeScanResult{}, err
		}
	}
	projectPath, err := filepath.Abs(projectPath)
	if err != nil {
		return highCodeScanResult{}, err
	}
	specs := []structuredHighCodeSpec{
		{domain: "classes", relPath: "backend/classes", sourceExts: []string{".java"}, requiresConfig: true},
		{domain: "triggers", relPath: "backend/triggers", sourceExts: []string{".java"}, requiresConfig: true},
		{domain: "timer", relPath: "backend/schedule", sourceExts: []string{".java"}, requiresConfig: true},
		{domain: "script", relPath: "script", sourceExts: []string{".js"}, requiresConfig: true},
		{domain: "html", relPath: "html", sourceExts: []string{".html"}, sourceNames: []string{"index.html"}, requiresConfig: true},
	}
	domains := make([]highCodeDomain, 0, len(specs)+3)
	for _, spec := range specs {
		domains = append(domains, scanStructuredHighCodeDomain(projectPath, spec))
	}
	domains = append(domains, scanPageComponentHighCodeDomain(projectPath))
	domains = append(domains, scanFileHighCodeDomain(projectPath, "staticResource", []string{
		"staticResource", "staticResources", "static-resources", "assets/staticResource",
	}, true))
	domains = append(domains, scanFileHighCodeDomain(projectPath, "sidecar", []string{"sidecar"}, false))

	legacy := scanLegacyNodeScripts(projectPath)
	result := highCodeScanResult{
		Mode:        "go-local-highcode-scan",
		Source:      "project:" + filepath.Base(projectPath),
		Runtime:     "go",
		ProjectPath: projectPath,
		Domains:     domains,
		Legacy:      legacy,
	}
	result.Totals.Domains = len(domains)
	for _, domain := range domains {
		result.Totals.Assets += domain.Assets
		result.Totals.Configured += domain.Configured
		result.Totals.Publishable += domain.Publishable
		result.Totals.Files += domain.Files
		result.Totals.Issues += len(domain.Issues)
		if domain.Exists && domain.Assets == 0 {
			result.Totals.EmptyDomains++
		}
		if !domain.Exists {
			result.Totals.MissingDomains++
		}
		for _, item := range domain.Items {
			result.Totals.Issues += len(item.Issues)
		}
	}
	result.Totals.LegacyNodeScripts = len(legacy.NodeScripts)
	result.Totals.Issues += len(legacy.NodeScripts)
	return result, nil
}

func scanOnlineHighCodeProject(projectPath string) (onlineHighCodeScanResult, error) {
	if projectPath == "" {
		var err error
		projectPath, err = os.Getwd()
		if err != nil {
			return onlineHighCodeScanResult{}, err
		}
	}
	projectPath, err := filepath.Abs(projectPath)
	if err != nil {
		return onlineHighCodeScanResult{}, err
	}
	cfg, err := config.Load(projectPath)
	if err != nil {
		return onlineHighCodeScanResult{}, err
	}
	specs := []onlineHighCodeSpec{
		{domain: "classes", transport: "setup", base: "setup", path: "/api/ccfag/list", body: map[string]any{"shownum": 2000, "showpage": 1, "sname": ""}},
		{domain: "triggers", transport: "setup", base: "setup", path: "/api/triggerSetup/getTriggerByCondition", body: map[string]any{"shownum": "2000", "showpage": "1", "sname": "", "objId": "", "fid": "", "rptcond": "lastmodifydate", "rptorder": "desc"}},
		{domain: "timer", transport: "setup", base: "setup", path: "/api/ccPeak/list", body: map[string]any{"shownum": 2000, "showpage": 1, "sname": ""}},
		{domain: "script", transport: "devconsole-envelope", base: "base", path: "/devconsole/script/pageClientScript", body: map[string]any{"pageSize": 2000, "pageNo": 1, "condition": map[string]any{"scriptName": "", "pageLabel": "", "objName": ""}}},
		{domain: "staticResource", candidates: []onlineHighCodeCandidate{
			{transport: "setup", base: "setup", path: "/api/staticResource/list", body: map[string]any{}},
			{transport: "setup", base: "setup", path: "/api/staticresource/list", body: map[string]any{}},
			{transport: "setup", base: "setup", path: "/api/staticResources/queryList", body: map[string]any{"fileAPI": "", "contentType": "", "keyWord": ""}},
			{transport: "devconsole-token", base: "base", path: highCodeDevDispatch(cfg) + "/staticResource/pageStaticResource", body: map[string]any{"pageNo": 1, "pageSize": 2000, "condition": map[string]any{"label": "", "fileAPI": ""}}},
			{transport: "devconsole-token", base: "base", path: highCodeDevDispatch(cfg) + "/staticResource/listStaticResource", body: map[string]any{}},
		}},
		{domain: "pagecomponent", transport: "devconsole-envelope", base: "base", path: highCodeDevDispatch(cfg) + "/custom/pc/1.0/post/pageCustomComp", body: map[string]any{"pageNo": 1, "pageSize": 2000, "condition": map[string]any{"compName": "", "dtBegin": "", "dtEnd": ""}}, header: highCodePageComponentHeader(cfg)},
		{domain: "html", transport: "devconsole-token", base: "base", path: highCodeDevDispatch(cfg) + "/htmlComponent/pageHtmlComponent", body: map[string]any{"pageNo": 1, "pageSize": 2000, "condition": map[string]any{"htmlLabel": "", "apiName": ""}}},
		{domain: "customPage", transport: "devconsole-envelope", base: "base", path: highCodeDevDispatch(cfg) + "/custom/pc/1.0/post/pageCustomPage", body: map[string]any{"pageNo": 1, "pageSize": 2000, "condition": map[string]any{"pageLabel": "", "pageApi": ""}}, header: highCodePageComponentHeader(cfg)},
	}
	result := onlineHighCodeScanResult{
		Mode:        "go-online-highcode-scan",
		Source:      "project:" + filepath.Base(projectPath),
		Runtime:     "go",
		ProjectPath: projectPath,
	}
	for _, spec := range specs {
		result.Domains = append(result.Domains, scanOnlineHighCodeDomain(cfg, spec))
	}
	result.Domains = append(result.Domains,
		onlineHighCodeDomain{Domain: "sidecar", Transport: "external", Status: "out_of_scope", Issues: []string{"sidecar is external deployment runtime, not CloudCC platform metadata; verify it with a deployment manifest or runtime monitor outside online-highcode"}},
	)
	result.Totals.Domains = len(result.Domains)
	for _, domain := range result.Domains {
		result.Totals.OnlineItems += domain.Count
		switch domain.Status {
		case "passed":
			result.Totals.Passed++
		case "unsupported_endpoint":
			result.Totals.Unsupported++
		case "out_of_scope":
			result.Totals.OutOfScope++
		default:
			result.Totals.Failed++
		}
	}
	return result, nil
}

func scanOnlineHighCodeDomain(cfg config.Config, spec onlineHighCodeSpec) onlineHighCodeDomain {
	candidates := onlineHighCodeCandidates(spec)
	domain := onlineHighCodeDomain{
		Domain:    spec.domain,
		Transport: onlineHighCodeCandidateTransports(candidates),
		Endpoint:  onlineHighCodeCandidatePaths(candidates),
		Status:    "passed",
	}
	var res map[string]any
	var errors []string
	for _, candidate := range candidates {
		res = map[string]any{}
		err := callOnlineHighCodePath(cfg, candidate, &res)
		if err == nil {
			if msg := onlineHighCodeResponseError(res); msg != "" {
				errors = append(errors, candidate.path+": "+msg)
				continue
			}
			domain.Endpoint = candidate.path
			domain.Transport = candidate.transport
			items := onlineHighCodeItems(res)
			domain.Count = len(items)
			domain.Sample = onlineHighCodeSample(items, 20)
			return domain
		}
		errors = append(errors, candidate.path+": "+err.Error())
	}
	domain.Status = "endpoint_unavailable"
	domain.Issues = append(domain.Issues, errors...)
	return domain
}

func onlineHighCodeCandidates(spec onlineHighCodeSpec) []onlineHighCodeCandidate {
	if len(spec.candidates) > 0 {
		return spec.candidates
	}
	paths := spec.paths
	if len(paths) == 0 {
		paths = []string{spec.path}
	}
	candidates := make([]onlineHighCodeCandidate, 0, len(paths))
	for _, path := range paths {
		candidates = append(candidates, onlineHighCodeCandidate{
			transport: spec.transport,
			base:      spec.base,
			path:      path,
			body:      spec.body,
			header:    spec.header,
		})
	}
	return candidates
}

func onlineHighCodeCandidatePaths(candidates []onlineHighCodeCandidate) string {
	paths := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		paths = append(paths, candidate.path)
	}
	return strings.Join(paths, ",")
}

func onlineHighCodeCandidateTransports(candidates []onlineHighCodeCandidate) string {
	seen := map[string]bool{}
	transports := []string{}
	for _, candidate := range candidates {
		if candidate.transport != "" && !seen[candidate.transport] {
			seen[candidate.transport] = true
			transports = append(transports, candidate.transport)
		}
	}
	return strings.Join(transports, ",")
}

func callOnlineHighCodePath(cfg config.Config, candidate onlineHighCodeCandidate, res *map[string]any) error {
	switch candidate.transport {
	case "setup":
		return httpclient.New().PostClass(strings.TrimRight(config.String(cfg, "setupSvc"), "/")+candidate.path, candidate.body, config.String(cfg, "accessToken"), res)
	case "devconsole-envelope":
		header := candidate.header
		if header == nil {
			header = map[string]any(cfg)
		}
		return httpclient.New().PostEnvelope(highCodeBaseURL(cfg)+candidate.path, candidate.body, header, res)
	case "devconsole-token":
		return httpclient.New().PostRaw(highCodeBaseURL(cfg)+candidate.path, candidate.body, map[string]string{"accessToken": firstString(config.String(cfg, "accessToken"), config.String(cfg, "pluginToken"))}, res)
	default:
		return fmt.Errorf("unsupported transport %s", candidate.transport)
	}
}

func scanStructuredHighCodeDomain(projectPath string, spec structuredHighCodeSpec) highCodeDomain {
	base := filepath.Join(projectPath, filepath.FromSlash(spec.relPath))
	domain := highCodeDomain{Domain: spec.domain, Path: spec.relPath}
	if info, err := os.Stat(base); err != nil || !info.IsDir() {
		domain.Issues = append(domain.Issues, "directory_missing")
		return domain
	}
	domain.Exists = true
	byDir := map[string]*highCodeAsset{}
	_ = filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			domain.Issues = append(domain.Issues, "walk_error:"+err.Error())
			return nil
		}
		if shouldSkipHighCodePath(path, d) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		domain.Files++
		relFromBase := slashRel(base, path)
		dirRel := filepath.ToSlash(filepath.Dir(relFromBase))
		item := highCodeAssetForDir(byDir, projectPath, filepath.Dir(path), dirRel, path)
		if strings.EqualFold(d.Name(), "config.json") {
			item.ConfigPath = slashRel(projectPath, path)
			return nil
		}
		if matchesHighCodeSource(d.Name(), spec.sourceExts, spec.sourceNames) {
			item.SourceFiles = append(item.SourceFiles, slashRel(projectPath, path))
		}
		return nil
	})
	for _, item := range sortedHighCodeAssets(byDir) {
		if len(item.SourceFiles) == 0 && item.ConfigPath == "" {
			continue
		}
		if len(item.SourceFiles) == 0 {
			item.Issues = append(item.Issues, "source_missing")
		}
		if spec.requiresConfig && item.ConfigPath == "" {
			item.Issues = append(item.Issues, "config_missing")
		}
		item.Publishable = len(item.SourceFiles) > 0 && (!spec.requiresConfig || item.ConfigPath != "")
		domain.Items = append(domain.Items, item)
	}
	finishHighCodeDomain(&domain)
	return domain
}

func scanPageComponentHighCodeDomain(projectPath string) highCodeDomain {
	const relPath = "frontend/pagecomponents"
	base := filepath.Join(projectPath, filepath.FromSlash(relPath))
	domain := highCodeDomain{Domain: "pagecomponent", Path: relPath}
	if info, err := os.Stat(base); err != nil || !info.IsDir() {
		domain.Issues = append(domain.Issues, "directory_missing")
		return domain
	}
	domain.Exists = true
	activeConfig := activeProjectConfig(projectPath)
	byDir := map[string]*highCodeAsset{}
	localConfigs := map[string]map[string]any{}
	_ = filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			domain.Issues = append(domain.Issues, "walk_error:"+err.Error())
			return nil
		}
		if shouldSkipHighCodePath(path, d) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		domain.Files++
		relFromBase := slashRel(base, path)
		dirRel := filepath.ToSlash(filepath.Dir(relFromBase))
		item := highCodeAssetForDir(byDir, projectPath, filepath.Dir(path), dirRel, path)
		if strings.EqualFold(d.Name(), "config.json") {
			item.ConfigPath = slashRel(projectPath, path)
			if cfg, err := readHighCodeJSON(path); err == nil {
				localConfigs[item.Name] = cfg
			} else {
				item.Issues = append(item.Issues, "config_invalid_json")
			}
			return nil
		}
		if strings.EqualFold(filepath.Ext(d.Name()), ".vue") {
			item.SourceFiles = append(item.SourceFiles, slashRel(projectPath, path))
		}
		return nil
	})
	for _, item := range sortedHighCodeAssets(byDir) {
		if len(item.SourceFiles) == 0 && item.ConfigPath == "" {
			continue
		}
		if len(item.SourceFiles) == 0 {
			item.Issues = append(item.Issues, "source_missing")
		}
		if item.ConfigPath == "" {
			item.Issues = append(item.Issues, "config_missing")
		}
		localConfig := localConfigs[item.Name]
		component := firstString(stringValue(localConfig["component"]), "component-"+item.Name)
		if bundle := firstExistingHighCodePath(pageComponentHighCodeBundleCandidates(projectPath, item.Name, component, localConfig, activeConfig)); bundle != "" {
			item.BundlePath = slashRel(projectPath, bundle)
		} else {
			item.Issues = append(item.Issues, "prebuilt_bundle_missing")
		}
		item.Publishable = len(item.SourceFiles) > 0 && item.ConfigPath != "" && item.BundlePath != ""
		domain.Items = append(domain.Items, item)
	}
	finishHighCodeDomain(&domain)
	return domain
}

func scanFileHighCodeDomain(projectPath string, domainName string, relPaths []string, publishable bool) highCodeDomain {
	domain := highCodeDomain{Domain: domainName, Path: strings.Join(relPaths, ",")}
	for _, relPath := range relPaths {
		base := filepath.Join(projectPath, filepath.FromSlash(relPath))
		if info, err := os.Stat(base); err == nil && info.IsDir() {
			domain.Exists = true
			domain.Path = relPath
			_ = filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					domain.Issues = append(domain.Issues, "walk_error:"+err.Error())
					return nil
				}
				if shouldSkipHighCodePath(path, d) {
					if d.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
				if d.IsDir() {
					return nil
				}
				domain.Files++
				item := highCodeAsset{
					Name:        strings.TrimSuffix(d.Name(), filepath.Ext(d.Name())),
					Path:        slashRel(projectPath, path),
					SourceFiles: []string{slashRel(projectPath, path)},
					Publishable: publishable,
				}
				domain.Items = append(domain.Items, item)
				return nil
			})
		}
	}
	if !domain.Exists {
		domain.Issues = append(domain.Issues, "directory_missing")
		return domain
	}
	finishHighCodeDomain(&domain)
	return domain
}

func scanLegacyNodeScripts(projectPath string) highCodeLegacy {
	base := filepath.Join(projectPath, "scripts")
	var legacy highCodeLegacy
	if info, err := os.Stat(base); err != nil || !info.IsDir() {
		return legacy
	}
	_ = filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if shouldSkipHighCodePath(path, d) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() || !strings.EqualFold(filepath.Ext(d.Name()), ".js") {
			return nil
		}
		legacy.NodeScripts = append(legacy.NodeScripts, highCodeAsset{
			Name: strings.TrimSuffix(d.Name(), filepath.Ext(d.Name())),
			Path: slashRel(projectPath, path),
			Issues: []string{
				"legacy_node_implementation_script",
				"not_a_go_skill_validation_path",
			},
		})
		return nil
	})
	sort.Slice(legacy.NodeScripts, func(i, j int) bool {
		return legacy.NodeScripts[i].Path < legacy.NodeScripts[j].Path
	})
	return legacy
}

func finishHighCodeDomain(domain *highCodeDomain) {
	sort.Slice(domain.Items, func(i, j int) bool { return domain.Items[i].Path < domain.Items[j].Path })
	domain.Assets = len(domain.Items)
	for _, item := range domain.Items {
		if item.ConfigPath != "" {
			domain.Configured++
		}
		if item.Publishable {
			domain.Publishable++
		}
	}
	if domain.Exists && domain.Assets == 0 {
		domain.Issues = append(domain.Issues, "empty")
	}
}

func highCodeAssetForDir(byDir map[string]*highCodeAsset, projectPath string, absDir string, dirRel string, filePath string) *highCodeAsset {
	dirRel = strings.Trim(dirRel, "/")
	if dirRel == "." {
		dirRel = ""
	}
	key := filepath.ToSlash(absDir)
	if item := byDir[key]; item != nil {
		return item
	}
	name := filepath.Base(absDir)
	if dirRel == "" {
		name = strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	}
	item := &highCodeAsset{
		Name: name,
		Path: slashRel(projectPath, absDir),
	}
	byDir[key] = item
	return item
}

func sortedHighCodeAssets(byDir map[string]*highCodeAsset) []highCodeAsset {
	keys := make([]string, 0, len(byDir))
	for key := range byDir {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]highCodeAsset, 0, len(keys))
	for _, key := range keys {
		item := *byDir[key]
		sort.Strings(item.SourceFiles)
		out = append(out, item)
	}
	return out
}

func matchesHighCodeSource(name string, exts []string, names []string) bool {
	for _, exact := range names {
		if strings.EqualFold(name, exact) {
			return true
		}
	}
	ext := strings.ToLower(filepath.Ext(name))
	for _, allowed := range exts {
		if ext == strings.ToLower(allowed) {
			return true
		}
	}
	return false
}

func shouldSkipHighCodePath(path string, d fs.DirEntry) bool {
	name := d.Name()
	if name == ".git" || name == "dist" {
		return true
	}
	if strings.HasPrefix(name, ".") && name != "." && name != ".." {
		if d.IsDir() {
			return true
		}
		switch name {
		case ".gitkeep", ".DS_Store":
			return true
		}
	}
	if !d.IsDir() {
		switch name {
		case ".gitkeep", ".DS_Store":
			return true
		}
	}
	return false
}

func readHighCodeJSON(path string) (map[string]any, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("%s: invalid JSON: %w", path, err)
	}
	return out, nil
}

func activeProjectConfig(projectPath string) map[string]any {
	root, err := config.Root(projectPath)
	if err != nil {
		return nil
	}
	use := stringValue(root["use"])
	active, _ := root[use].(map[string]any)
	return active
}

func pageComponentHighCodeBundleCandidates(projectPath string, name string, component string, localConfig map[string]any, activeConfig map[string]any) []string {
	rawValues := []any{
		localConfig["bundlePath"],
		localConfig["prebuiltBundlePath"],
		localConfig["prebuiltBundle"],
		localConfig["jsBundlePath"],
		activeConfig["pagecomponentBundlePath"],
		activeConfig["prebuiltBundlePath"],
	}
	componentDir := filepath.Join(projectPath, "frontend", "pagecomponents", name)
	candidates := []string{}
	seen := map[string]bool{}
	addCandidate := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" || value == "<nil>" {
			return
		}
		if !filepath.IsAbs(value) {
			for _, base := range []string{projectPath, componentDir} {
				abs := filepath.Clean(filepath.Join(base, filepath.FromSlash(value)))
				if !seen[abs] {
					seen[abs] = true
					candidates = append(candidates, abs)
				}
			}
			return
		}
		abs := filepath.Clean(value)
		if !seen[abs] {
			seen[abs] = true
			candidates = append(candidates, abs)
		}
	}
	for _, raw := range rawValues {
		addCandidate(stringValue(raw))
	}
	for _, path := range []string{
		filepath.Join(projectPath, "frontend", "build", component+".umd.min.js"),
		filepath.Join(projectPath, "frontend", "build", component+".umd.js"),
		filepath.Join(componentDir, "build", component+".umd.min.js"),
		filepath.Join(componentDir, "build", component+".umd.js"),
		filepath.Join(componentDir, "build", name+".umd.min.js"),
		filepath.Join(componentDir, "build", name+".umd.js"),
	} {
		addCandidate(path)
	}
	return candidates
}

func firstExistingHighCodePath(paths []string) string {
	for _, path := range paths {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path
		}
	}
	return ""
}

func onlineHighCodeResponseError(value map[string]any) string {
	if result, ok := value["result"].(bool); ok && !result {
		return firstString(stringValue(value["returnInfo"]), stringValue(value["msg"]), stringValue(value["errormsg"]), "business response returned result=false")
	}
	code := firstString(stringValue(value["returnCode"]), stringValue(value["code"]))
	if code == "" {
		return ""
	}
	if code == "1" || code == "200" || code == "000-000" || strings.Contains(code, "-000-") || strings.EqualFold(code, "success") {
		return ""
	}
	return firstString(stringValue(value["returnInfo"]), stringValue(value["msg"]), stringValue(value["errormsg"]), "business response returned code "+code)
}

func onlineHighCodeItems(value any) []map[string]any {
	switch v := value.(type) {
	case []any:
		out := make([]map[string]any, 0, len(v))
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	case map[string]any:
		for _, key := range []string{"list", "rows", "records", "items", "data"} {
			if items := onlineHighCodeItems(v[key]); len(items) > 0 {
				return items
			}
		}
		if data, ok := v["data"].(map[string]any); ok {
			for _, key := range []string{"list", "rows", "records", "items"} {
				if items := onlineHighCodeItems(data[key]); len(items) > 0 {
					return items
				}
			}
		}
	}
	return nil
}

func onlineHighCodeSample(items []map[string]any, limit int) []map[string]any {
	if len(items) == 0 {
		return nil
	}
	if limit > len(items) {
		limit = len(items)
	}
	keys := []string{"id", "name", "label", "apiname", "apiName", "compName", "component", "compUniName", "htmlLabel", "scriptName", "objName", "isactive", "isActive"}
	out := make([]map[string]any, 0, limit)
	for _, item := range items[:limit] {
		row := map[string]any{}
		for _, key := range keys {
			if value := item[key]; value != nil && stringValue(value) != "" && stringValue(value) != "<nil>" {
				row[key] = value
			}
		}
		if len(row) == 0 {
			row["keys"] = sortedOnlineHighCodeKeys(item)
		}
		out = append(out, row)
	}
	return out
}

func sortedOnlineHighCodeKeys(item map[string]any) []string {
	keys := make([]string, 0, len(item))
	for key := range item {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func highCodeBaseURL(cfg config.Config) string {
	if v := config.String(cfg, "baseUrl"); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "https://developer.apis.cloudcc.cn"
}

func highCodeDevDispatch(cfg config.Config) string {
	if v := config.String(cfg, "devSvcDispatch"); v != "" {
		return v
	}
	return "/devconsole"
}

func highCodePageComponentHeader(cfg config.Config) map[string]any {
	return map[string]any{
		"appType":     "lightning-setup",
		"appVersion":  "0.0.1",
		"accessToken": firstString(config.String(cfg, "accessToken"), config.String(cfg, "pluginToken")),
		"source":      "lightning-setup",
		"version":     "public",
	}
}

func slashRel(base string, path string) string {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}
