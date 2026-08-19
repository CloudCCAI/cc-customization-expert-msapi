package modules

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClassSourcePolicyMatchesSetupSvcGuards(t *testing.T) {
	cases := []struct {
		name   string
		source string
		want   string
	}{
		{name: "valid", source: `public class Demo { public void run() {} }`},
		{name: "dbman", source: `public class Demo { DBMan db; }`, want: "DBMan"},
		{name: "system-exit", source: `public class Demo { void run() { System.exit(0); } }`, want: "System.exit"},
		{name: "while-true", source: `public class Demo { void run() { while (true) {} } }`, want: "unbounded"},
		{name: "cached", source: `import com.cloudcc.core.service.CCCached; public class Demo {}`, want: "blocked platform import"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			violations := classSourcePolicyViolations(tc.source)
			if tc.want == "" && len(violations) != 0 {
				t.Fatalf("expected no policy violations, got %v", violations)
			}
			if tc.want != "" && !strings.Contains(strings.Join(violations, " "), tc.want) {
				t.Fatalf("expected violation containing %q, got %v", tc.want, violations)
			}
		})
	}
}

func TestClassRuntimeContractRequiresUserInfoConstructor(t *testing.T) {
	if !hasUserInfoConstructor(`public class Demo { public Demo(UserInfo userInfo) {} }`, "Demo") {
		t.Fatal("expected UserInfo constructor to satisfy runtime contract")
	}
	if hasUserInfoConstructor(`public class Demo { public Demo() {} }`, "Demo") {
		t.Fatal("expected no-arg constructor to fail runtime contract")
	}
}

func TestLookupClassIDAcceptsPlatformSuccessCodeAndExactName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":true,"returnCode":"1","data":{"list":[{"id":"class-001","name":"DemoClass"},{"id":"class-002","name":"DemoClassOld"}]}}`))
	}))
	defer server.Close()
	id, err := lookupClassID(server.URL, "token", "DemoClass")
	if err != nil || id != "class-001" {
		t.Fatalf("expected exact existing class id, id=%q err=%v", id, err)
	}
}

func TestParseJavacDiagnosticsMapsGeneratedLinesToSource(t *testing.T) {
	generated := "/tmp/Demo.java"
	diagnostics := parseJavacDiagnostics(generated+`:25: error: cannot find symbol
`, generated, 22, 10)
	if len(diagnostics) != 1 || diagnostics[0].SourceLine != 4 || diagnostics[0].GeneratedLine != 25 {
		t.Fatalf("unexpected mapped diagnostics: %#v", diagnostics)
	}
}

func TestStandaloneCompilerBundleIsCompleteWithoutPlatformCheckout(t *testing.T) {
	t.Setenv("CLOUDCC_PLATFORM_HOME", "/path/that/must/not/be-used")
	t.Setenv("CLOUDCC_CLASS_COMPILER_HOME", "")
	env := discoverClassDevEnvironment(classDevOptions{ProjectPath: filepath.Clean("../../..")})
	if !env.Ready {
		t.Fatalf("expected packaged compiler bundle to be ready, missing=%v", env.Missing)
	}
	if env.ArtifactCount != 30 || !env.IntegrityVerified {
		t.Fatalf("unexpected compiler bundle result: count=%d integrity=%v", env.ArtifactCount, env.IntegrityVerified)
	}
	if strings.Contains(env.CompilerHome, "cloudccone") || strings.Contains(strings.Join(env.Classpath, "\n"), ".m2") {
		t.Fatalf("standalone compiler leaked a platform checkout or Maven cache: home=%s", env.CompilerHome)
	}
}

func TestClassPublishBaseURLDerivesPublicGateway(t *testing.T) {
	cases := map[string]string{
		"https://example.cloudcc.cn/ccdomaingateway":        "https://example.cloudcc.cn/ccdomaingateway/setup",
		"https://example.cloudcc.cn/ccdomaingateway/apisvc": "https://example.cloudcc.cn/ccdomaingateway/setup",
		"https://example.cloudcc.cn":                        "https://example.cloudcc.cn/ccdomaingateway/setup",
	}
	for base, expected := range cases {
		actual, err := classPublishBaseURL(map[string]any{"baseUrl": base})
		if err != nil || actual != expected {
			t.Fatalf("base=%s expected=%s actual=%s err=%v", base, expected, actual, err)
		}
	}
}

func TestStandalonePublishUsesGatewayListSaveDetail(t *testing.T) {
	const source = `public class DemoClass {
    private final UserInfo userInfo;
    public DemoClass(UserInfo userInfo) { this.userInfo = userInfo; }
    public String execute(String value) { return value + "-ok"; }
}`
	paths := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths[r.URL.Path]++
		if r.Header.Get("accessToken") != "token" {
			t.Errorf("expected accessToken header, got %q", r.Header.Get("accessToken"))
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/ccfag/list":
			_, _ = w.Write([]byte(`{"result":true,"returnCode":"1","data":{"list":[]}}`))
		case "/api/ccfag/validate":
			_, _ = w.Write([]byte(`{"result":true,"returnCode":"1","data":{"valid":true,"message":"Compilation succeeded.","errors":[],"warnings":[]}}`))
		case "/api/ccfag/save":
			_, _ = w.Write([]byte(`{"result":true,"returnCode":"1","data":"class-001"}`))
		case "/api/ccfag/detail":
			_ = json.NewEncoder(w).Encode(map[string]any{"result": true, "returnCode": "1", "data": map[string]any{"id": "class-001", "source": source}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	project := t.TempDir()
	writeClassDevTestFile(t, filepath.Join(project, "cloudcc-cli.config.json"), `{"use":"dev","dev":{"accessToken":"token","classPublishUrl":"`+server.URL+`","version":"private"}}`)
	writeClassDevTestFile(t, filepath.Join(project, "backend", "classes", "DemoClass", "DemoClass.java"), "// @SOURCE_CONTENT_START\n"+source+"\n// @SOURCE_CONTENT_END\n")
	writeClassDevTestFile(t, filepath.Join(project, "backend", "classes", "DemoClass", "config.json"), `{"version":"2"}`)
	var stdout, stderr bytes.Buffer
	if err := publishClassResource([]string{"DemoClass", project}, &stdout, &stderr, project); err != nil {
		t.Fatalf("standalone publish failed: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
	}
	for _, path := range []string{"/api/ccfag/list", "/api/ccfag/validate", "/api/ccfag/save", "/api/ccfag/detail"} {
		if paths[path] != 1 {
			t.Fatalf("expected one gateway request to %s, got %d", path, paths[path])
		}
	}
	if !strings.Contains(stdout.String(), `"status": "published_and_verified"`) || !strings.Contains(stdout.String(), `"publishGateway": "`+server.URL+`"`) {
		t.Fatalf("unexpected publish output: %s", stdout.String())
	}
}

func TestPublishClassRejectsStaleValidationEvidenceBeforeHTTP(t *testing.T) {
	project := t.TempDir()
	writeClassDevTestFile(t, filepath.Join(project, "cloudcc-cli.config.json"), `{"use":"dev","dev":{"accessToken":"token","apiSvc":"http://127.0.0.1:1","setupSvc":"http://127.0.0.1:1"}}`)
	writeClassDevTestFile(t, filepath.Join(project, "backend", "classes", "DemoClass", "DemoClass.java"), "// @SOURCE_CONTENT_START\npublic class DemoClass {}\n// @SOURCE_CONTENT_END\n")
	writeClassDevTestFile(t, filepath.Join(project, "backend", "classes", "DemoClass", "config.json"), `{"version":"2"}`)
	evidencePath := filepath.Join(project, "stale-validation.json")
	writeClassDevTestFile(t, evidencePath, `{"status":"passed","valid":true,"className":"DemoClass","sourceSha256":"stale","diagnostics":[]}`)
	var stdout, stderr bytes.Buffer
	err := publishClassResource([]string{"DemoClass", "--validation-evidence", evidencePath}, &stdout, &stderr, project)
	if err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("expected stale validation evidence rejection, got %v", err)
	}
}

func writeClassDevTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
