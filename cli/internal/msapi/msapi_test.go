package msapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestClientTimeoutAllowsLongMetadataApplyCapture(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:8087"}}}`)
	client, _, err := newClient(nil, tmp)
	if err != nil {
		t.Fatal(err)
	}
	if client.http.Timeout < 180*time.Second {
		t.Fatalf("expected MetadataService client timeout to cover long object apply captures, got %s", client.http.Timeout)
	}
}

func TestCapabilitiesUsesMetadataServiceURLFromEnv(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method %s", r.Method)
		}
		if r.URL.Path != "/metadata/v1/capabilities" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"service":"cc-metadata-service","domains":["objects"]}`))
	}))
	defer server.Close()
	t.Setenv("CLOUDCC_METADATA_SERVICE_URL", server.URL)

	var stdout bytes.Buffer
	if err := Handle("capabilities", "msapi", nil, &stdout, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "cc-metadata-service") {
		t.Fatalf("expected capabilities JSON, got %s", stdout.String())
	}
}

func TestCapabilitiesPromptsForMetadataServiceURLAndPersistsToActiveEnv(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metadata/v1/capabilities" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"service":"cc-metadata-service","domains":["objects"]}`))
	}))
	defer server.Close()
	tmp := t.TempDir()
	writeTestFile(t, filepath.Join(tmp, "cloudcc-cli.config.json"), `{
  "use": "dev",
  "dev": {
    "safetyMark": "safe",
    "CloudCCDev": "encoded-dev-key"
  }
}`)
	oldReader := metadataServicePromptReader
	oldWriter := metadataServicePromptWriter
	metadataServicePromptReader = strings.NewReader(server.URL + "\n")
	var prompt bytes.Buffer
	metadataServicePromptWriter = &prompt
	defer func() {
		metadataServicePromptReader = oldReader
		metadataServicePromptWriter = oldWriter
	}()

	var stdout bytes.Buffer
	if err := Handle("capabilities", "msapi", []string{tmp}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "cc-metadata-service") {
		t.Fatalf("expected capabilities JSON, got %s", stdout.String())
	}
	root := readTestObjectFile(t, filepath.Join(tmp, "cloudcc-cli.config.json"))
	dev := root["dev"].(map[string]any)
	metadataService := dev["metadataService"].(map[string]any)
	if metadataService["url"] != server.URL {
		t.Fatalf("expected prompted URL persisted, got %#v", metadataService)
	}
	if !strings.Contains(prompt.String(), "metadataService.url") {
		t.Fatalf("expected save prompt, got %s", prompt.String())
	}
}

func TestMetadataServiceURLMissingAndEmptyPromptDoesNotFallbackToDefault(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, filepath.Join(tmp, "cloudcc-cli.config.json"), `{"use":"dev","dev":{"safetyMark":"safe"}}`)
	oldReader := metadataServicePromptReader
	oldWriter := metadataServicePromptWriter
	metadataServicePromptReader = strings.NewReader("")
	metadataServicePromptWriter = io.Discard
	defer func() {
		metadataServicePromptReader = oldReader
		metadataServicePromptWriter = oldWriter
	}()

	var stdout bytes.Buffer
	err := Handle("capabilities", "msapi", []string{tmp}, &stdout, tmp)
	if err == nil {
		t.Fatal("expected missing MetadataService URL error")
	}
	if !strings.Contains(err.Error(), "metadataService.url") || strings.Contains(err.Error(), defaultServiceURL) {
		t.Fatalf("expected explicit missing config error without default fallback, got %v", err)
	}
}

func TestSetupSvcLiveReplayEnvironmentReportsReachabilityAndRedactsSecrets(t *testing.T) {
	t.Setenv("MDS_RUNTIME_MODE", "self-hosted")
	t.Setenv("MDS_SERVER_PORT", "18087")
	t.Setenv("MDS_JDBC_URL", "jdbc:mysql://secret-db-host:3306/secret_metadata")
	t.Setenv("MDS_DB_USERNAME", "secret-db-user")
	t.Setenv("MDS_DB_PASSWORD", "secret-db-password")
	t.Setenv("MDS_DB_DRIVER", "com.mysql.cj.jdbc.Driver")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/actuator/health" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"status":"UP"}`))
	}))
	defer server.Close()
	tmp := t.TempDir()
	writeTestFile(t, filepath.Join(tmp, "cloudcc-cli.config.json"), `{
  "use": "dev",
  "dev": {
    "setupSvc": "https://example.cloudcc.cn/setup?token=setup-secret",
    "apiSvc": "https://example.cloudcc.cn/apisvc?token=api-secret",
    "accessToken": "bus-secret",
    "metadataService": {
      "url": "`+server.URL+`?token=metadata-secret",
      "token": "metadata-token-secret"
    }
  }
}`)
	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-environment"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	output := stdout.String()
	for _, secret := range []string{"setup-secret", "api-secret", "bus-secret", "metadata-secret", "metadata-token-secret", "secret-db-host", "secret_metadata", "secret-db-user", "secret-db-password"} {
		if strings.Contains(output, secret) {
			t.Fatalf("environment output leaked secret %q: %s", secret, output)
		}
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["mode"] != "setup-svc-live-replay-environment" {
		t.Fatalf("unexpected mode: %#v", result["mode"])
	}
	metadataService := result["metadataService"].(map[string]any)
	if metadataService["status"] != "reachable" || metadataService["reachable"] != true {
		t.Fatalf("expected reachable metadata service, got %#v", metadataService)
	}
	config := result["config"].(map[string]any)
	if config["accessTokenConfigured"] != true || config["metadataServiceTokenSource"] != "metadataService" {
		t.Fatalf("expected redacted token presence, got %#v", config)
	}
	if strings.Contains(config["setupSvcEndpoint"].(string), "?") || strings.Contains(config["metadataServiceEndpoint"].(string), "?") {
		t.Fatalf("expected endpoints without query strings, got %#v", config)
	}
	datasource := result["metadataServiceDatasource"].(map[string]any)
	if datasource["status"] != "ready" ||
		datasource["readyForRealDatasource"] != true ||
		datasource["jdbcUrlSource"] != "env:MDS_JDBC_URL" ||
		datasource["passwordSource"] != "env:MDS_DB_PASSWORD" ||
		datasource["jdbcUrlLooksDefaultH2"] != false {
		t.Fatalf("expected redacted real datasource readiness, got %#v", datasource)
	}
	if strings.Contains(datasource["startCommandHint"].(string), "secret") {
		t.Fatalf("start command hint leaked datasource secret: %#v", datasource["startCommandHint"])
	}
	gates := completionAuditGatesByName(result)
	gateDatasource := gates["metadata_service_datasource"]["metadataServiceDatasource"].(map[string]any)
	if gateDatasource["status"] != "ready" ||
		gateDatasource["readyForRealDatasource"] != true ||
		gateDatasource["jdbcUrlSource"] != "env:MDS_JDBC_URL" {
		t.Fatalf("expected environment datasource gate to carry redacted readiness, got %#v", gates["metadata_service_datasource"])
	}
}

func TestSetupSvcLiveReplayEnvironmentBlocksWhenMetadataServiceUnreachable(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, filepath.Join(tmp, "cloudcc-cli.config.json"), `{
  "use": "dev",
  "dev": {
    "setupSvc": "https://example.cloudcc.cn/setup",
    "apiSvc": "https://example.cloudcc.cn/apisvc",
    "metadataService": {"url": "http://127.0.0.1:1"}
  }
}`)
	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-environment"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked_metadata_service_unreachable" {
		t.Fatalf("expected blocked_metadata_service_unreachable, got %s", stdout.String())
	}
	metadataService := result["metadataService"].(map[string]any)
	if metadataService["reachable"] != false || metadataService["status"] != "unreachable" {
		t.Fatalf("expected unreachable metadata service, got %#v", metadataService)
	}
	datasource := result["metadataServiceDatasource"].(map[string]any)
	if datasource["status"] != "missing_real_datasource" || datasource["readyForRealDatasource"] != false {
		t.Fatalf("expected missing datasource readiness, got %#v", datasource)
	}
	missing, ok := datasource["missing"].([]any)
	if !ok ||
		!containsAnyString(missing, "MDS_JDBC_URL") ||
		!containsAnyString(missing, "MDS_DB_USERNAME") ||
		!containsAnyString(missing, "MDS_DB_PASSWORD") ||
		!containsAnyString(missing, "MDS_DB_DRIVER") {
		t.Fatalf("expected missing MDS datasource vars, got %#v", datasource["missing"])
	}
	if !strings.Contains(stdout.String(), "metadata_service_datasource") {
		t.Fatalf("expected datasource readiness gate, got %s", stdout.String())
	}
	gates := completionAuditGatesByName(result)
	gateDatasource := gates["metadata_service_datasource"]["metadataServiceDatasource"].(map[string]any)
	if gateDatasource["status"] != "missing_real_datasource" ||
		gateDatasource["readyForRealDatasource"] != false ||
		!containsAnyString(gateDatasource["missing"].([]any), "MDS_JDBC_URL") {
		t.Fatalf("expected environment datasource gate to carry missing readiness, got %#v", gates["metadata_service_datasource"])
	}
}

func TestSetupSvcLiveReplayEnvironmentMirrorsCompletionRepairQueues(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, filepath.Join(tmp, "cloudcc-cli.config.json"), `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:1"}}}`)

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-workspace", "--execute", "--approval", setupSvcParityEvidenceWorkspaceApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-environment"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	audit := result["completionAudit"].(map[string]any)
	queues, _ := audit["repairQueues"].([]any)
	if int(audit["failedEvidenceTotal"].(float64)) == 0 ||
		int(audit["repairQueueCount"].(float64)) != len(queues) ||
		!containsRepairQueueWithPositiveCount(queues, "setup-svc", "tableSnapshots") ||
		!containsRepairQueueWithPositiveCount(queues, "metadata-service", "runtimeEffectChecks") {
		t.Fatalf("expected environment audit to mirror completion repair queues, got %#v", audit)
	}
}

func TestPlanBuildsRequestForDomainShortcut(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method %s", r.Method)
		}
		if r.URL.Path != "/metadata/v1/plans" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"planId":"plan_test","operationId":"oper_test","status":"PLANNED"}`))
	}))
	defer server.Close()
	t.Setenv("CLOUDCC_METADATA_SERVICE_URL", server.URL)

	var stdout bytes.Buffer
	if err := Handle("plan", "metadata", []string{"object", `{"id":"obj_test","label":"测试对象"}`}, &stdout, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if received["domain"] != "objects" {
		t.Fatalf("expected objects domain, got %#v", received["domain"])
	}
	if received["operation"] != "upsert" {
		t.Fatalf("expected default upsert operation, got %#v", received["operation"])
	}
	spec := received["spec"].(map[string]any)
	if spec["id"] != "obj_test" {
		t.Fatalf("expected spec to be forwarded, got %#v", spec)
	}
	if !strings.Contains(stdout.String(), "plan_test") {
		t.Fatalf("expected plan response, got %s", stdout.String())
	}
}

func TestProjectConfigCanProvideMetadataServiceURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"service":"configured"}`))
	}))
	defer server.Close()
	tmp := t.TempDir()
	config := `{"use":"dev","dev":{"metadataService":{"url":"` + server.URL + `"}}}`
	if err := os.WriteFile(tmp+"/cloudcc-cli.config.json", []byte(config), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	if err := Handle("capabilities", "msapi", []string{tmp}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "configured") {
		t.Fatalf("expected configured service response, got %s", stdout.String())
	}
}

func TestMetadataServiceTokenHeadersUseEnvOverride(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("accessToken"); got != "msapi-test-token" {
			t.Fatalf("expected accessToken header, got %q", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer msapi-test-token" {
			t.Fatalf("expected bearer Authorization header, got %q", got)
		}
		_, _ = w.Write([]byte(`{"service":"secured"}`))
	}))
	defer server.Close()
	t.Setenv("CLOUDCC_METADATA_SERVICE_URL", server.URL)
	t.Setenv("CLOUDCC_METADATA_SERVICE_ACCESS_TOKEN", "Bearer msapi-test-token")

	var stdout bytes.Buffer
	if err := Handle("capabilities", "msapi", nil, &stdout, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "secured") {
		t.Fatalf("expected secured service response, got %s", stdout.String())
	}
}

func TestProjectConfigCanProvideMetadataServiceToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("accessToken"); got != "configured-token" {
			t.Fatalf("expected configured token, got %q", got)
		}
		if got := r.Header.Get("X-CloudCC-User-AccessToken"); got != "cloudcc-user-token" {
			t.Fatalf("expected CloudCC user token header, got %q", got)
		}
		_, _ = w.Write([]byte(`{"service":"configured-token"}`))
	}))
	defer server.Close()
	tmp := t.TempDir()
	config := `{"use":"dev","dev":{"accessToken":"cloudcc-user-token","metadataService":{"url":"` + server.URL + `","token":"configured-token"}}}`
	if err := os.WriteFile(tmp+"/cloudcc-cli.config.json", []byte(config), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	if err := Handle("capabilities", "msapi", []string{tmp}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "configured-token") {
		t.Fatalf("expected configured token response, got %s", stdout.String())
	}
}

func TestMetadataServiceInvalidTokenClearsCacheAndRetries(t *testing.T) {
	var metadataCalls int
	var tokenCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/cauth/token":
			tokenCalls++
			if r.Method != http.MethodPost {
				t.Fatalf("unexpected token method %s", r.Method)
			}
			_, _ = w.Write([]byte(`{"result":true,"data":{"accessToken":"fresh-token"}}`))
		case "/metadata/v1/capabilities":
			metadataCalls++
			switch r.Header.Get("accessToken") {
			case "stale-token":
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"status":401,"error":"invalid_token","message":"CloudCC accessToken validation failed."}`))
			case "fresh-token":
				_, _ = w.Write([]byte(`{"service":"refreshed"}`))
			default:
				t.Fatalf("unexpected metadata accessToken %q", r.Header.Get("accessToken"))
			}
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	tmp := t.TempDir()
	cfg := `{"use":"dev","dev":{"metadataService":{"url":"` + server.URL + `"},"apiSvc":"` + server.URL + `","setupSvc":"` + server.URL + `","username":"dev@example.com","safetyMark":"mark","clientId":"client","openSecretKey":"secret","orgId":"org"}}`
	if err := os.WriteFile(tmp+"/cloudcc-cli.config.json", []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}
	cache := fmt.Sprintf(`{"mark":{"accessToken":"stale-token","apiSvc":%q,"setupSvc":%q,"username":"dev@example.com","safetyMark":"mark","clientId":"client","openSecretKey":"secret","orgId":"org","timestamp":%d}}`, server.URL, server.URL, time.Now().UnixMilli())
	if err := os.WriteFile(tmp+"/.cloudcc-cache.json", []byte(cache), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	if err := Handle("capabilities", "msapi", []string{tmp}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "refreshed") {
		t.Fatalf("expected refreshed response, got %s", stdout.String())
	}
	if metadataCalls != 2 {
		t.Fatalf("expected metadata retry, got %d calls", metadataCalls)
	}
	if tokenCalls != 1 {
		t.Fatalf("expected one token refresh, got %d calls", tokenCalls)
	}
	cacheBytes, err := os.ReadFile(tmp + "/.cloudcc-cache.json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cacheBytes), "fresh-token") || strings.Contains(string(cacheBytes), "stale-token") {
		t.Fatalf("expected cache to contain only refreshed token, got %s", string(cacheBytes))
	}
}

func TestMetadataServiceInvalidRefreshedTokenDoesNotStayCached(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/cauth/token":
			_, _ = w.Write([]byte(`{"result":true,"data":{"accessToken":"still-invalid-token"}}`))
		case "/metadata/v1/capabilities":
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"status":401,"error":"invalid_token","message":"CloudCC accessToken validation failed."}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	tmp := t.TempDir()
	cfg := `{"use":"dev","dev":{"metadataService":{"url":"` + server.URL + `"},"apiSvc":"` + server.URL + `","setupSvc":"` + server.URL + `","username":"dev@example.com","safetyMark":"mark","clientId":"client","openSecretKey":"secret","orgId":"org"}}`
	if err := os.WriteFile(tmp+"/cloudcc-cli.config.json", []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}
	cache := fmt.Sprintf(`{"mark":{"accessToken":"stale-token","apiSvc":%q,"setupSvc":%q,"username":"dev@example.com","safetyMark":"mark","clientId":"client","openSecretKey":"secret","orgId":"org","timestamp":%d}}`, server.URL, server.URL, time.Now().UnixMilli())
	if err := os.WriteFile(tmp+"/.cloudcc-cache.json", []byte(cache), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	err := Handle("capabilities", "msapi", []string{tmp}, &stdout, tmp)
	if err == nil || !strings.Contains(err.Error(), "invalid_token") {
		t.Fatalf("expected invalid_token error, got %v", err)
	}
	cacheBytes, err := os.ReadFile(tmp + "/.cloudcc-cache.json")
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if strings.Contains(string(cacheBytes), "stale-token") || strings.Contains(string(cacheBytes), "still-invalid-token") {
		t.Fatalf("expected invalid tokens to be removed from cache, got %s", string(cacheBytes))
	}
}

func TestScanSummaryCallsMetadataServiceScanner(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method %s", r.Method)
		}
		if r.URL.Path != "/metadata/v1/scans/summary" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"service":"cc-metadata-service","mode":"read-only"}`))
	}))
	defer server.Close()
	t.Setenv("CLOUDCC_METADATA_SERVICE_URL", server.URL)

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", nil, &stdout, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "read-only") {
		t.Fatalf("expected scanner summary response, got %s", stdout.String())
	}
}

func TestScanStandardCatalogCallsMetadataServiceScanner(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method %s", r.Method)
		}
		if r.URL.Path != "/metadata/v1/scans/standard-catalog" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"service":"cc-metadata-service","mode":"read-only-standard-catalog"}`))
	}))
	defer server.Close()
	t.Setenv("CLOUDCC_METADATA_SERVICE_URL", server.URL)

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{"standard-catalog"}, &stdout, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "read-only-standard-catalog") {
		t.Fatalf("expected scanner standard catalog response, got %s", stdout.String())
	}
}

func TestScanCompareAcceptsRequestFile(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method %s", r.Method)
		}
		if r.URL.Path != "/metadata/v1/scans:compare" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"mode":"read-only-compare","totals":{"checks":1}}`))
	}))
	defer server.Close()
	t.Setenv("CLOUDCC_METADATA_SERVICE_URL", server.URL)

	tmp := t.TempDir()
	requestFile := tmp + "/scan-request.json"
	if err := os.WriteFile(requestFile, []byte(`{"source":"unit","checks":[{"label":"objects"}]}`), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{"@" + requestFile}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	if received["source"] != "unit" {
		t.Fatalf("expected request file to be forwarded, got %#v", received)
	}
	if !strings.Contains(stdout.String(), "read-only-compare") {
		t.Fatalf("expected compare response, got %s", stdout.String())
	}
}

func TestProjectScanBuildsCompareRequest(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metadata/v1/scans:compare" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"mode":"read-only-compare","totals":{"checks":10}}`))
	}))
	defer server.Close()

	tmp := t.TempDir()
	if err := os.MkdirAll(tmp+"/config", 0755); err != nil {
		t.Fatal(err)
	}
	config := `{"use":"dev","dev":{"metadataService":{"url":"` + server.URL + `"}}}`
	if err := os.WriteFile(tmp+"/cloudcc-cli.config.json", []byte(config), 0644); err != nil {
		t.Fatal(err)
	}
	metadataPlan := `{
	  "objects": [
	    {"key":"account","label":"客户","existingApiName":"Account","fields":[{"label":"CRM临时客户编号"}]},
	    {"key":"contract","label":"合同","apiName":"contract","fieldSets":["common"]}
	  ],
	  "fieldSets": {"common":[{"label":"审批状态"}]}
	}`
	if err := os.WriteFile(tmp+"/config/cloudcc-metadata-plan.json", []byte(metadataPlan), 0644); err != nil {
		t.Fatal(err)
	}
	businessPlan := `{
	  "recordTypes": [{"objectKey":"contract","items":[{"name":"标准合同","apiCode":"standard_contract"}]}],
	  "validationRules": [{"objectKey":"contract","name":"禁止负数金额"}]
	}`
	if err := os.WriteFile(tmp+"/config/cloudcc-business-config-plan.json", []byte(businessPlan), 0644); err != nil {
		t.Fatal(err)
	}
	navigationPlan := `{
	  "application": {"label":"CRM工作台","apiName":"crm_workbench"},
	  "groups": [{"key":"sales","label":"销售","objects":["account","contract"]}]
	}`
	if err := os.WriteFile(tmp+"/config/cloudcc-navigation-plan.json", []byte(navigationPlan), 0644); err != nil {
		t.Fatal(err)
	}
	fieldAPIMap := `{
	  "objects": [{
	    "key": "contract",
	    "objectId": "obj_contract",
	    "plannedApiByLabel": {
	      "审批状态": {"apiName":"spzt_custom_l_field"},
	      "价格": {"apiName":"jg_custom_n_field"}
	    }
	  }]
	}`
	if err := os.WriteFile(tmp+"/config/cloudcc-field-api-map.json", []byte(fieldAPIMap), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "project"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	if received["source"] != "project:"+tmp[strings.LastIndex(tmp, "/")+1:] {
		t.Fatalf("unexpected source %#v", received["source"])
	}
	checks, ok := received["checks"].([]any)
	if !ok || len(checks) != 11 {
		t.Fatalf("expected 11 project checks, got %#v", received["checks"])
	}
	objectLabelCheck := checks[0].(map[string]any)
	if objectLabelCheck["missingStatus"] != "informational_gap" {
		t.Fatalf("expected object labels to be informational, got %#v", objectLabelCheck)
	}
	fieldCheck := checks[3].(map[string]any)
	if fieldCheck["label"] != "project object-scoped field API names" {
		t.Fatalf("expected object-scoped field check, got %#v", fieldCheck["label"])
	}
	records := fieldCheck["records"].([]any)
	if len(records) != 2 {
		t.Fatalf("expected two object-scoped field records, got %#v", records)
	}
	recordColumns := fieldCheck["recordColumns"].([]any)
	if len(recordColumns) != 2 {
		t.Fatalf("expected two record columns, got %#v", recordColumns)
	}
	validationRuleCheck := checks[6].(map[string]any)
	if _, ok := validationRuleCheck["missingStatus"]; ok {
		t.Fatalf("expected validation rules to stay blocking, got %#v", validationRuleCheck)
	}
	menuLabelCheck := checks[9].(map[string]any)
	if menuLabelCheck["missingStatus"] != "informational_gap" {
		t.Fatalf("expected menu labels to be informational, got %#v", menuLabelCheck)
	}
	if !strings.Contains(stdout.String(), "read-only-compare") {
		t.Fatalf("expected compare response, got %s", stdout.String())
	}
}

func TestFieldMapScanBuildsReadOnlyRequest(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method %s", r.Method)
		}
		if r.URL.Path != "/metadata/v1/scans/field-map" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"mode":"read-only-field-map","summary":{"objectsRequested":2}}`))
	}))
	defer server.Close()

	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"`+server.URL+`"}}}`)
	writeTestFile(t, tmp+"/config/cloudcc-metadata-plan.json", `{
	  "objects": [
	    {
	      "key":"account",
	      "phase":"p1",
	      "action":"reuse",
	      "label":"客户",
	      "existingApiName":"Account",
	      "prefix":"001",
	      "fieldSets":["common"],
	      "fields":[{"label":"CRM临时客户编号","type":"S","remark":"临时编号"}]
	    },
	    {
	      "key":"contract",
	      "label":"合同",
	      "apiName":"Contract",
	      "fields":[{"label":"有效开始日期","type":"D"}]
	    }
	  ],
	  "fieldSets": {
	    "common": [{"label":"主数据同步状态","type":"L","remark":"同步状态"}]
	  }
	}`)

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "field-map"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "read-only-field-map") {
		t.Fatalf("expected field-map response, got %s", stdout.String())
	}
	if received["source"] != "field-map:"+tmp[strings.LastIndex(tmp, "/")+1:] {
		t.Fatalf("unexpected source %#v", received["source"])
	}
	objects, ok := received["objects"].([]any)
	if !ok || len(objects) != 2 {
		t.Fatalf("expected two objects, got %#v", received["objects"])
	}
	first := objects[0].(map[string]any)
	if first["prefix"] != "001" || first["existingApiName"] != "Account" {
		t.Fatalf("expected object identity fields, got %#v", first)
	}
	fields := first["fields"].([]any)
	if len(fields) != 2 {
		t.Fatalf("expected expanded fieldSet plus object field, got %#v", fields)
	}
	if fields[0].(map[string]any)["source"] != "fieldSet:common" {
		t.Fatalf("expected fieldSet source, got %#v", fields[0])
	}
	if fields[1].(map[string]any)["label"] != "CRM临时客户编号" {
		t.Fatalf("expected object field, got %#v", fields[1])
	}
}

func TestFieldMapScanFallsBackWhenMetadataPlanMissing(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method %s", r.Method)
		}
		if r.URL.Path != "/metadata/v1/scans/field-map" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"mode":"read-only-field-map","summary":{"objectsRequested":0,"fallback":"tenant-catalog"}}`))
	}))
	defer server.Close()

	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"`+server.URL+`"}}}`)

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "field-map"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "read-only-field-map") {
		t.Fatalf("expected field-map response, got %s", stdout.String())
	}
	if received["planAvailable"] != false {
		t.Fatalf("expected missing plan fallback marker, got %#v", received["planAvailable"])
	}
	objects, ok := received["objects"].([]any)
	if !ok || len(objects) != 0 {
		t.Fatalf("expected empty planned objects for fallback, got %#v", received["objects"])
	}
}

func TestSetupSvcLiveReplayReadinessIsReadOnlyAndCoversAllDomains(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	writeTestFile(t, tmp+"/package.json", `{"devConsoleConfig":{"accessToken":"token","setupSvc":"https://setup.example.invalid","apiSvc":"https://api.example.invalid"}}`)

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-readiness"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["readOnly"] != true || result["execute"] != false || result["approvalRequired"] != true {
		t.Fatalf("expected read-only approval-gated readiness, got %#v", result)
	}
	if result["approvalPhrase"] != setupSvcParityReplayApproval {
		t.Fatalf("unexpected approval phrase %#v", result["approvalPhrase"])
	}
	if result["status"] != "ready_for_approved_live_replay" {
		t.Fatalf("expected ready status, got %#v", result["status"])
	}
	totals := result["totals"].(map[string]any)
	if int(totals["domains"].(float64)) != len(setupSvcLiveReplayDomains()) {
		t.Fatalf("expected all domains in readiness totals, got %#v", totals)
	}
	domains := result["domains"].([]any)
	if len(domains) != 21 {
		t.Fatalf("expected 21 covered domains, got %d", len(domains))
	}
	seen := map[string]bool{}
	for _, raw := range domains {
		domain := raw.(map[string]any)
		seen[domain["domain"].(string)] = true
		if domain["status"] != "covered_pending_live_replay" || domain["approvalRequired"] != true {
			t.Fatalf("expected covered pending status for domain, got %#v", domain)
		}
		if len(domain["setupSvcEvidence"].([]any)) == 0 || len(domain["metadataServiceEvidence"].([]any)) == 0 || len(domain["queryEvidence"].([]any)) == 0 {
			t.Fatalf("expected evidence requirements for domain, got %#v", domain)
		}
	}
	for _, required := range []string{"objects", "fields", "layouts", "permissions", "approval-processes", "reports", "dashboards", "object-views"} {
		if !seen[required] {
			t.Fatalf("expected domain %s in readiness output", required)
		}
	}
}

func TestSetupSvcLiveReplayCoverageAuditCoversCRUDQAndVariants(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	writeSetupSvcLiveReplayParityMatrix(t, tmp, nil)

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-coverage"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "passed" || result["readOnly"] != true {
		t.Fatalf("expected passed read-only coverage audit, got %#v", result)
	}
	totals := result["totals"].(map[string]any)
	if int(totals["domains"].(float64)) != 21 ||
		int(totals["operations"].(float64)) != setupSvcLiveReplayOperationCount() ||
		int(totals["canonicalCrudQueryDomains"].(float64)) != 21 ||
		int(totals["queryOperations"].(float64)) != 21 ||
		int(totals["writeOperations"].(float64)) != 70 ||
		int(totals["variantOperations"].(float64)) != 7 {
		t.Fatalf("unexpected coverage totals: %#v", totals)
	}
	if int(totals["runtimeEffects"].(float64)) != 90 ||
		int(totals["queryReadbackExpectations"].(float64)) != 70 {
		t.Fatalf("expected runtime/readback expectation coverage totals, got %#v", totals)
	}
	families := result["operationFamilies"].(map[string]any)
	if len(families["create"].([]any)) != 21 || len(families["update"].([]any)) != 21 || len(families["query"].([]any)) != 21 {
		t.Fatalf("expected canonical operation family coverage, got %#v", families)
	}
	if !containsStringItem(families["variants"].([]any), "objects/physical-purge") ||
		!containsStringItem(families["variants"].([]any), "permissions/assign") ||
		!containsStringItem(families["variants"].([]any), "permissions/remove") ||
		!containsStringItem(families["variants"].([]any), "roles/assign") ||
		!containsStringItem(families["variants"].([]any), "reports/folder-create") ||
		!containsStringItem(families["variants"].([]any), "reports/folder-update") ||
		!containsStringItem(families["variants"].([]any), "reports/folder-delete") {
		t.Fatalf("expected variant operations to be visible, got %#v", families["variants"])
	}
	testEvidence := result["testEvidenceContract"].(map[string]any)
	if testEvidence["status"] != "passed" ||
		int(testEvidence["domains"].(float64)) != 21 ||
		int(testEvidence["operations"].(float64)) != setupSvcLiveReplayOperationCount() ||
		int(totals["testEvidenceDomains"].(float64)) != 21 ||
		int(totals["testEvidenceOperations"].(float64)) != setupSvcLiveReplayOperationCount() {
		t.Fatalf("expected complete replay test evidence coverage, got contract=%#v totals=%#v", testEvidence, totals)
	}
	if testEvidence["testSourceStatus"] != "not_found" {
		t.Fatalf("temporary coverage audit should not require Java source tree, got %#v", testEvidence)
	}
}

func TestSetupSvcLiveReplayCoverageAuditBlocksMatrixGaps(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	writeSetupSvcLiveReplayParityMatrix(t, tmp, nil)
	matrixPath := filepath.Join(tmp, "cc-metadata-service/src/test/resources/parity/msapi-setup-svc-parity-matrix.json")
	rewriteSetupSvcLiveReplayParityMatrix(t, matrixPath, func(matrix map[string]any) {
		firstDomain := matrix["domains"].([]any)[0].(map[string]any)
		delete(firstDomain, "queryIncluded")
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-coverage"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("expected blocked coverage audit, got %#v", result)
	}
	if !containsStringItem(result["blockingIssues"].([]any), "parityMatrix: objects: missing queryIncluded") ||
		!containsStringItem(result["blockingIssues"].([]any), "objects: queryIncluded must be true") {
		t.Fatalf("expected queryIncluded blockers, got %#v", result["blockingIssues"])
	}
}

func TestSetupSvcLiveReplayCoverageAuditBlocksMatrixRuntimeAndReadbackGaps(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	writeSetupSvcLiveReplayParityMatrix(t, tmp, nil)
	matrixPath := filepath.Join(tmp, "cc-metadata-service/src/test/resources/parity/msapi-setup-svc-parity-matrix.json")
	rewriteSetupSvcLiveReplayParityMatrix(t, matrixPath, func(matrix map[string]any) {
		firstDomain := matrix["domains"].([]any)[0].(map[string]any)
		delete(firstDomain, "runtimeEffects")
		delete(firstDomain, "queryReadbackExpectations")
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-coverage"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("expected blocked coverage audit, got %#v", result)
	}
	if !containsStringItem(result["blockingIssues"].([]any), "parityMatrix: objects: missing runtimeEffects") ||
		!containsStringItem(result["blockingIssues"].([]any), "parityMatrix: objects: missing queryReadbackExpectations") ||
		!containsStringItem(result["blockingIssues"].([]any), "objects: missing runtimeEffects; missing queryReadbackExpectations") {
		t.Fatalf("expected runtime/readback matrix blockers, got %#v", result["blockingIssues"])
	}
}

func TestSetupSvcLiveReplayCoverageAuditBlocksTestEvidenceGaps(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	writeSetupSvcLiveReplayParityMatrix(t, tmp, nil)
	evidencePath := filepath.Join(tmp, "cc-metadata-service/src/test/resources/parity/msapi-parity-test-evidence.json")
	rewriteSetupSvcLiveReplayTestEvidence(t, evidencePath, func(evidence map[string]any) {
		firstDomain := evidence["domains"].([]any)[0].(map[string]any)
		firstDomain["operationEvidence"] = firstDomain["operationEvidence"].([]any)[:4]
	})
	t.Setenv("CLOUDCC_MSAPI_PARITY_TEST_EVIDENCE_FILE", evidencePath)

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-coverage"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("expected blocked coverage audit, got %#v", result)
	}
	if !containsStringItem(result["blockingIssues"].([]any), "objects: missing replay test evidence for operations query") {
		t.Fatalf("expected missing operation test evidence blocker, got %#v", result["blockingIssues"])
	}
}

func TestSetupSvcLiveReplayCoverageAuditBlocksTestEvidenceSourceDrift(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	writeSetupSvcLiveReplayParityMatrix(t, tmp, nil)
	sourceRoot := filepath.Join(tmp, "cc-metadata-service/src/test/java/com/cloudcc/metadata/parity")
	writeTestFile(t, filepath.Join(sourceRoot, "GeneratedParityReplayTest.java"), `
package com.cloudcc.metadata.parity;

class GeneratedParityReplayTest {
    void unrelatedMethod() {
    }
}
`)

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-coverage"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("expected blocked coverage audit, got %#v", result)
	}
	testEvidence := result["testEvidenceContract"].(map[string]any)
	if testEvidence["testSourceStatus"] != "blocked" || int(testEvidence["testSourceChecks"].(float64)) != setupSvcLiveReplayOperationCount() {
		t.Fatalf("expected blocked test source checks for every operation, got %#v", testEvidence)
	}
	if !containsStringItem(result["blockingIssues"].([]any), "testEvidence: objects/create: missing replay test method GeneratedParityReplayTest.generatedParityReplayCoversMatrixOperation") {
		t.Fatalf("expected missing source method blocker, got %#v", result["blockingIssues"])
	}
}

func TestSetupSvcLiveReplayTestEvidenceFindsSourceBesideExternalEvidence(t *testing.T) {
	project := t.TempDir()
	repo := t.TempDir()
	writeSetupSvcLiveReplayParityMatrix(t, repo, nil)
	writeGeneratedSetupSvcLiveReplayTestSource(t, repo)
	t.Setenv("CLOUDCC_MSAPI_PARITY_TEST_EVIDENCE_FILE", filepath.Join(repo, "test-fixtures/msapi/parity/msapi-parity-test-evidence.json"))

	evidence := setupSvcLiveReplayTestEvidenceStatus(project)
	if evidence.Status != "passed" ||
		evidence.TestSourceStatus != "passed" ||
		evidence.TestSourceChecks != setupSvcLiveReplayOperationCount() ||
		!strings.HasPrefix(evidence.TestSourcePath, repo) {
		t.Fatalf("expected source checks to resolve beside external evidence, got %#v", evidence)
	}
}

func TestSetupSvcLiveReplayPacketBuildsPendingManifestTemplate(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-packet"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["readOnly"] != true || result["execute"] != false || result["approvalRequired"] != true {
		t.Fatalf("expected read-only non-executing packet, got %#v", result)
	}
	if result["status"] != "ready_for_manual_evidence_collection" {
		t.Fatalf("unexpected packet status %#v", result["status"])
	}
	totals := result["totals"].(map[string]any)
	if int(totals["domains"].(float64)) != len(setupSvcLiveReplayDomains()) || int(totals["operations"].(float64)) != setupSvcLiveReplayOperationCount() {
		t.Fatalf("expected full domain/operation packet totals, got %#v", totals)
	}
	if int(totals["queryOperations"].(float64)) != len(setupSvcLiveReplayDomains()) {
		t.Fatalf("expected one query operation per domain, got %#v", totals)
	}
	manifest := result["manifestTemplate"].(map[string]any)
	if manifest["status"] != "pending" {
		t.Fatalf("manifest template should remain pending, got %#v", manifest)
	}
	if result["contractVersion"] != setupSvcLiveReplayContractVersion || result["contractFingerprint"] != setupSvcLiveReplayExpectedContractFingerprint() {
		t.Fatalf("packet contract identity mismatch, got version=%#v fingerprint=%#v", result["contractVersion"], result["contractFingerprint"])
	}
	if manifest["contractVersion"] != setupSvcLiveReplayContractVersion || manifest["contractFingerprint"] != result["contractFingerprint"] {
		t.Fatalf("manifest template must inherit packet contract identity, got %#v", manifest)
	}
	domains := manifest["domains"].([]any)
	if len(domains) != len(setupSvcLiveReplayDomains()) {
		t.Fatalf("expected manifest template for all domains, got %d", len(domains))
	}
	packetDomains := result["domains"].([]any)
	firstPacketDomain := packetDomains[0].(map[string]any)
	requiredTables := firstPacketDomain["requiredTables"].([]any)
	if !containsStringItem(requiredTables, "tp_sys_object") || !containsStringItem(requiredTables, "tp_sys_schemetable") {
		t.Fatalf("packet domain must carry required metadata tables, got %#v", firstPacketDomain)
	}
	runtimeEffects := firstPacketDomain["runtimeEffects"].([]any)
	if !containsStringItem(runtimeEffects, "datatable-prefix-allocation") || !containsStringItem(runtimeEffects, "standard-object-default-fields") {
		t.Fatalf("packet domain must carry runtime effect contract, got %#v", firstPacketDomain)
	}
	queryReadbackExpectations := firstPacketDomain["queryReadbackExpectations"].([]any)
	if !containsStringItem(queryReadbackExpectations, "object-identity-prefix-datatable-readback") || !containsStringItem(queryReadbackExpectations, "standard-custom-field-readback") {
		t.Fatalf("packet domain must carry query/readback expectation contract, got %#v", firstPacketDomain)
	}
	firstOperation := domains[0].(map[string]any)["operations"].([]any)[0].(map[string]any)
	if firstOperation["setupSvcEvidenceStatus"] != "pending" ||
		firstOperation["metadataServiceEvidenceStatus"] != "pending" ||
		firstOperation["queryEvidenceStatus"] != "pending" ||
		firstOperation["normalizedDiffStatus"] != "pending" {
		t.Fatalf("manifest evidence statuses must start pending, got %#v", firstOperation)
	}
}

func TestSetupSvcLiveReplayRequiredTablesMatchBundledParityMatrix(t *testing.T) {
	matrixPath := filepath.Join("..", "..", "..", "test-fixtures", "msapi", "parity", "msapi-setup-svc-parity-matrix.json")
	body, err := os.ReadFile(matrixPath)
	if err != nil {
		t.Fatal(err)
	}
	var matrix struct {
		Domains []struct {
			Domain         string   `json:"domain"`
			RequiredTables []string `json:"requiredTables"`
		} `json:"domains"`
	}
	if err := json.Unmarshal(body, &matrix); err != nil {
		t.Fatal(err)
	}
	expected := map[string][]string{}
	for _, domain := range matrix.Domains {
		expected[normalizeDomain(domain.Domain)] = domain.RequiredTables
	}
	for _, domain := range setupSvcLiveReplayDomains() {
		matrixTables, ok := expected[normalizeDomain(domain.Domain)]
		if !ok {
			t.Fatalf("live replay domain %s missing from bundled parity matrix", domain.Domain)
		}
		if strings.Join(domain.RequiredTables, "\x00") != strings.Join(matrixTables, "\x00") {
			t.Fatalf("requiredTables for %s drifted from bundled parity matrix\nlive replay: %#v\nmatrix: %#v", domain.Domain, domain.RequiredTables, matrixTables)
		}
		delete(expected, normalizeDomain(domain.Domain))
	}
	if len(expected) > 0 {
		t.Fatalf("bundled parity matrix has domains missing from live replay contract: %#v", expected)
	}
}

func TestSetupSvcLiveReplayReadinessBlocksIncompleteParityMatrix(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	writeTestFile(t, tmp+"/package.json", `{"devConsoleConfig":{"accessToken":"token","setupSvc":"https://setup.example.invalid","apiSvc":"https://api.example.invalid"}}`)
	writeSetupSvcLiveReplayParityMatrix(t, tmp, nil)
	matrixPath := filepath.Join(tmp, "cc-metadata-service/src/test/resources/parity/msapi-setup-svc-parity-matrix.json")
	rewriteSetupSvcLiveReplayParityMatrix(t, matrixPath, func(matrix map[string]any) {
		firstDomain := matrix["domains"].([]any)[0].(map[string]any)
		firstDomain["setupSvcReferences"] = []any{}
		delete(firstDomain, "queryIncluded")
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-readiness"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked_parity_matrix_contract" {
		t.Fatalf("incomplete parity matrix must block readiness, got %#v", result)
	}
	matrixContract := result["matrixContract"].(map[string]any)
	if matrixContract["status"] != "blocked" {
		t.Fatalf("expected blocked matrix contract, got %#v", matrixContract)
	}
	if !containsStringItem(result["blockingIssues"].([]any), "parityMatrix: objects: missing setupSvcReferences") ||
		!containsStringItem(result["blockingIssues"].([]any), "parityMatrix: objects: missing queryIncluded") {
		t.Fatalf("expected matrix reference/queryIncluded blockers, got %#v", result["blockingIssues"])
	}
}

func TestSetupSvcLiveReplayPacketBlocksParityMatrixRequiredTableDrift(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	writeSetupSvcLiveReplayParityMatrix(t, tmp, nil)
	matrixPath := filepath.Join(tmp, "cc-metadata-service/src/test/resources/parity/msapi-setup-svc-parity-matrix.json")
	rewriteSetupSvcLiveReplayParityMatrix(t, matrixPath, func(matrix map[string]any) {
		firstDomain := matrix["domains"].([]any)[0].(map[string]any)
		firstDomain["requiredTables"] = []any{"tp_sys_schemetable"}
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-packet"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked_parity_matrix_contract" {
		t.Fatalf("requiredTables drift must block packet generation, got %#v", result)
	}
	if !containsStringItem(result["blockingIssues"].([]any), "parityMatrix: objects: missing requiredTables tp_sys_object") {
		t.Fatalf("expected missing requiredTables blocker, got %#v", result["blockingIssues"])
	}
}

func TestSetupSvcLiveReplayPacketBlocksRuntimeReadbackExpectationDrift(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	writeSetupSvcLiveReplayParityMatrix(t, tmp, nil)
	matrixPath := filepath.Join(tmp, "cc-metadata-service/src/test/resources/parity/msapi-setup-svc-parity-matrix.json")
	rewriteSetupSvcLiveReplayParityMatrix(t, matrixPath, func(matrix map[string]any) {
		firstDomain := matrix["domains"].([]any)[0].(map[string]any)
		firstDomain["runtimeEffects"] = []any{"unexpected-runtime-effect"}
		firstDomain["queryReadbackExpectations"] = []any{"unexpected-readback-expectation"}
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-packet"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked_parity_matrix_contract" {
		t.Fatalf("runtime/readback expectation drift must block packet generation, got %#v", result)
	}
	if !containsStringItem(result["blockingIssues"].([]any), "parityMatrix: objects: missing runtimeEffects datatable-prefix-allocation") ||
		!containsStringItem(result["blockingIssues"].([]any), "parityMatrix: objects: unexpected queryReadbackExpectations unexpected-readback-expectation") {
		t.Fatalf("expected runtime/readback expectation blockers, got %#v", result["blockingIssues"])
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksParityMatrixQueryDrift(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	writeSetupSvcLiveReplayParityMatrix(t, tmp, nil)
	matrixPath := filepath.Join(tmp, "cc-metadata-service/src/test/resources/parity/msapi-setup-svc-parity-matrix.json")
	rewriteSetupSvcLiveReplayParityMatrix(t, matrixPath, func(matrix map[string]any) {
		firstDomain := matrix["domains"].([]any)[0].(map[string]any)
		firstDomain["queryIncluded"] = false
	})
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("matrix query drift must block evidence verification, got %#v", result)
	}
	if !containsStringItem(result["blockingIssues"].([]any), "parityMatrix: objects: queryIncluded must be true") {
		t.Fatalf("expected queryIncluded matrix blocker, got %#v", result["blockingIssues"])
	}
}

func TestSetupSvcLiveReplayPacketApplyDryRunValidatesCompletePacket(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	packetPath := filepath.Join(tmp, "setup-svc-live-replay-packet.json")
	packet := buildSetupSvcLiveReplayPacket(tmp)
	b, err := json.Marshal(packet)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, packetPath, string(b))

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-packet", "@" + packetPath, "--dry-run"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "dry_run_ready" || result["readOnly"] != true || result["execute"] != false {
		t.Fatalf("expected dry-run ready apply result, got %#v", result)
	}
	totals := result["totals"].(map[string]any)
	if int(totals["domains"].(float64)) != len(setupSvcLiveReplayDomains()) || int(totals["operations"].(float64)) != setupSvcLiveReplayOperationCount() {
		t.Fatalf("expected complete packet totals, got %#v", totals)
	}
	if _, ok := result["blockingIssues"]; ok {
		t.Fatalf("complete packet should not have blocking issues, got %#v", result["blockingIssues"])
	}
}

func TestSetupSvcLiveReplayWorkspaceDryRunDoesNotWriteFiles(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-workspace", "--dry-run"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "dry_run_ready" || result["readOnly"] != true || result["execute"] != false {
		t.Fatalf("expected dry-run workspace result, got %#v", result)
	}
	totals := result["totals"].(map[string]any)
	if int(totals["domains"].(float64)) != len(setupSvcLiveReplayDomains()) || int(totals["artifactFiles"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount() {
		t.Fatalf("expected complete workspace totals, got %#v", totals)
	}
	if _, err := os.Stat(filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")); !os.IsNotExist(err) {
		t.Fatalf("dry-run must not write manifest, stat err=%v", err)
	}
}

func TestSetupSvcLiveReplayWorkspaceExecuteRequiresApproval(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)

	var stdout bytes.Buffer
	err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-workspace", "--execute", "--approval", "WRONG"}, &stdout, tmp)
	if err == nil || !strings.Contains(err.Error(), setupSvcParityEvidenceWorkspaceApproval) {
		t.Fatalf("expected workspace approval error, got %v", err)
	}
}

func TestSetupSvcLiveReplayWorkspaceExecuteWritesPendingTemplatesThatVerifierBlocks(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	writeSetupSvcLiveReplayParityMatrix(t, tmp, nil)

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-workspace", "--execute", "--approval", setupSvcParityEvidenceWorkspaceApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "applied" || result["readOnly"] != false {
		t.Fatalf("expected workspace applied, got %#v", result)
	}
	totals := result["totals"].(map[string]any)
	if int(totals["writtenFiles"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount()+1 {
		t.Fatalf("expected manifest plus artifact templates written, got %#v", totals)
	}
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf("expected manifest template written: %v", err)
	}
	artifactPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/objects/create/setup-svc.json")
	var artifact map[string]any
	if payload, err := os.ReadFile(artifactPath); err != nil {
		t.Fatalf("expected artifact template: %v", err)
	} else if err := json.Unmarshal(payload, &artifact); err != nil {
		t.Fatal(err)
	}
	if artifact["status"] != "pending" || artifact["artifactType"] != "setup-svc" || artifact["domain"] != "objects" || artifact["operation"] != "create" {
		t.Fatalf("expected pending setup-svc artifact template, got %#v", artifact)
	}
	runtimeChecks := artifact["runtimeEffectChecks"].([]any)
	if len(runtimeChecks) == 0 || runtimeChecks[0].(map[string]any)["status"] != "pending" ||
		runtimeChecks[0].(map[string]any)["name"] != "datatable-prefix-allocation" {
		t.Fatalf("expected pending runtime effect checks in setup-svc template, got %#v", runtimeChecks)
	}
	snapshotShape := artifact["requiredSnapshotShape"].(map[string]any)
	if !containsStringItem(snapshotShape["runtimeEffectChecksRequired"].([]any), "datatable-prefix-allocation") ||
		!containsStringItem(snapshotShape["rowEvidenceKeys"].([]any), "rows") ||
		!containsStringItem(snapshotShape["columnEvidenceKeys"].([]any), "columns") {
		t.Fatalf("expected setup-svc template to declare snapshot evidence requirements, got %#v", snapshotShape)
	}
	checklist := artifact["artifactReplacementChecklist"].([]any)
	if !containsStringFragment(checklist, "Keep project") || !containsStringFragment(checklist, "rowCount-only") {
		t.Fatalf("expected artifact replacement checklist to reject weak placeholders, got %#v", checklist)
	}
	queryArtifactPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/objects/create/query-readback.json")
	var queryArtifact map[string]any
	if payload, err := os.ReadFile(queryArtifactPath); err != nil {
		t.Fatalf("expected query-readback artifact template: %v", err)
	} else if err := json.Unmarshal(payload, &queryArtifact); err != nil {
		t.Fatal(err)
	}
	readbackChecks := queryArtifact["readbackChecks"].(map[string]any)
	expectationChecks := readbackChecks["readbackExpectationChecks"].([]any)
	if len(expectationChecks) == 0 || expectationChecks[0].(map[string]any)["status"] != "pending" ||
		expectationChecks[0].(map[string]any)["name"] != "object-identity-prefix-datatable-readback" {
		t.Fatalf("expected pending readback expectation checks in query template, got %#v", expectationChecks)
	}
	readbackShape := queryArtifact["requiredReadbackShape"].(map[string]any)
	if !containsStringItem(readbackShape["readbackExpectationChecksRequired"].([]any), "object-identity-prefix-datatable-readback") ||
		!containsStringItem(readbackShape["requiredNumericZeroCleanCounters"].([]any), "missingRelationships") ||
		!containsStringFragment(readbackShape["rejectedEvidencePatterns"].([]any), "rowCount-only") {
		t.Fatalf("expected query template to declare strict readback requirements, got %#v", readbackShape)
	}
	normalizedDiffPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/objects/create/normalized-diff.json")
	var diffArtifact map[string]any
	if payload, err := os.ReadFile(normalizedDiffPath); err != nil {
		t.Fatalf("expected normalized-diff artifact template: %v", err)
	} else if err := json.Unmarshal(payload, &diffArtifact); err != nil {
		t.Fatal(err)
	}
	diffShape := diffArtifact["requiredDiffShape"].(map[string]any)
	if !containsStringItem(diffShape["requiredNumericZeroCleanCounters"].([]any), "differences") ||
		!containsStringItem(diffShape["nestedCleanNodes"].([]any), "normalizedDiff") {
		t.Fatalf("expected diff template to declare numeric-zero requirements, got %#v", diffShape)
	}
	cleanupPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/objects/create/cleanup.json")
	var cleanupArtifact map[string]any
	if payload, err := os.ReadFile(cleanupPath); err != nil {
		t.Fatalf("expected cleanup artifact template: %v", err)
	} else if err := json.Unmarshal(payload, &cleanupArtifact); err != nil {
		t.Fatal(err)
	}
	cleanupShape := cleanupArtifact["requiredCleanupShape"].(map[string]any)
	if !containsStringItem(cleanupShape["requiredNumericZeroResidualCounters"].([]any), "orphan") ||
		!containsStringItem(cleanupShape["residualEvidenceKeys"].([]any), "removed") {
		t.Fatalf("expected cleanup template to declare residual proof requirements, got %#v", cleanupShape)
	}

	stdout.Reset()
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var evidence map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &evidence); err != nil {
		t.Fatal(err)
	}
	if evidence["status"] != "blocked" {
		t.Fatalf("pending workspace evidence must not pass verifier, got %#v", evidence)
	}
	if !containsStringItem(evidence["blockingIssues"].([]any), "manifest: status not passed pending") {
		t.Fatalf("expected pending manifest blocker, got %#v", evidence["blockingIssues"])
	}
}

func TestSetupSvcLiveReplayWorkspaceTemplatesUseOperationScopedContracts(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	writeSetupSvcLiveReplayParityMatrix(t, tmp, nil)

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-workspace", "--execute", "--approval", setupSvcParityEvidenceWorkspaceApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	toStrings := func(raw []any) []string {
		out := make([]string, 0, len(raw))
		for _, item := range raw {
			out = append(out, item.(string))
		}
		return out
	}
	domainsByName := map[string]setupSvcLiveReplayDomain{}
	for _, domain := range setupSvcLiveReplayDomains() {
		domainsByName[domain.Domain] = domain
	}

	idpDelete := readTestJSONMap(t, filepath.Join(tmp, "outputs/setup-svc-live-replay/identity-providers/delete/metadata-service.json"))
	idpRequiredTables := toStrings(idpDelete["requiredTables"].([]any))
	idpExpectedTables := setupSvcLiveReplayRequiredTablesForOperation(domainsByName["identity-providers"], "delete")
	if !reflect.DeepEqual(idpRequiredTables, idpExpectedTables) || !reflect.DeepEqual(idpRequiredTables, []string{"tp_sys_idp_sps"}) {
		t.Fatalf("identity-provider delete template must use operation-scoped tables\nexpected=%#v\nactual=%#v", idpExpectedTables, idpRequiredTables)
	}
	idpSnapshotShape := idpDelete["requiredSnapshotShape"].(map[string]any)
	if !reflect.DeepEqual(toStrings(idpSnapshotShape["requiredTables"].([]any)), idpExpectedTables) {
		t.Fatalf("identity-provider delete requiredSnapshotShape must match operation tables, got %#v", idpSnapshotShape)
	}
	idpSnapshots := idpDelete["tableSnapshots"].(map[string]any)
	if len(idpSnapshots) != 1 {
		t.Fatalf("identity-provider delete template should allocate only one table snapshot, got %#v", idpSnapshots)
	}
	if _, ok := idpSnapshots["tp_sys_idp_config"]; ok {
		t.Fatalf("identity-provider delete template must not require config rows, got %#v", idpSnapshots)
	}

	objectQuery := readTestJSONMap(t, filepath.Join(tmp, "outputs/setup-svc-live-replay/objects/query/metadata-service.json"))
	objectQueryEffects := toStrings(objectQuery["runtimeEffects"].([]any))
	if len(objectQueryEffects) != 0 {
		t.Fatalf("objects/query template must not require write runtime effects, got %#v", objectQueryEffects)
	}
	if checks := objectQuery["runtimeEffectChecks"].([]any); len(checks) != 0 {
		t.Fatalf("objects/query template must not create runtimeEffectChecks, got %#v", checks)
	}
	querySnapshotShape := objectQuery["requiredSnapshotShape"].(map[string]any)
	if effects := toStrings(querySnapshotShape["runtimeEffectChecksRequired"].([]any)); len(effects) != 0 {
		t.Fatalf("objects/query requiredSnapshotShape must not require runtime effects, got %#v", effects)
	}

	objectCreate := readTestJSONMap(t, filepath.Join(tmp, "outputs/setup-svc-live-replay/objects/create/metadata-service.json"))
	createEffects := toStrings(objectCreate["runtimeEffects"].([]any))
	expectedCreateEffects := setupSvcLiveReplayRuntimeEffectsForOperation("objects", "create")
	if !reflect.DeepEqual(createEffects, expectedCreateEffects) {
		t.Fatalf("objects/create template should keep operation-scoped write effects\nexpected=%#v\nactual=%#v", expectedCreateEffects, createEffects)
	}
}

func TestSetupSvcLiveReplayWorkspaceExecutePreservesCollectedEvidence(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	writeSetupSvcLiveReplayParityMatrix(t, tmp, nil)

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-workspace", "--execute", "--approval", setupSvcParityEvidenceWorkspaceApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	artifactPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/objects/create/setup-svc.json")
	payload, err := os.ReadFile(artifactPath)
	if err != nil {
		t.Fatal(err)
	}
	var artifact map[string]any
	if err := json.Unmarshal(payload, &artifact); err != nil {
		t.Fatal(err)
	}
	artifact["status"] = "passed"
	artifact["tableSnapshots"] = map[string]any{
		"tp_sys_object": map[string]any{
			"columns": []string{"id", "schemetable_name"},
			"rows": []map[string]string{{
				"id":               "obj-replay-evidence",
				"schemetable_name": "contract",
			}},
		},
	}
	updatedPayload, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifactPath, append(updatedPayload, '\n'), 0644); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-workspace", "--execute", "--approval", setupSvcParityEvidenceWorkspaceApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked_workspace_write" {
		t.Fatalf("expected workspace write blocker for collected evidence, got %#v", result)
	}
	if !containsStringFragment(result["blockingIssues"].([]any), "objects/create/setup-svc.json") ||
		!containsStringFragment(result["blockingIssues"].([]any), "status is passed") {
		t.Fatalf("expected blocker to identify existing passed evidence, got %#v", result["blockingIssues"])
	}
	payload, err = os.ReadFile(artifactPath)
	if err != nil {
		t.Fatal(err)
	}
	var preserved map[string]any
	if err := json.Unmarshal(payload, &preserved); err != nil {
		t.Fatal(err)
	}
	if preserved["status"] != "passed" {
		t.Fatalf("expected passed evidence status to be preserved, got %#v", preserved["status"])
	}
	snapshots := preserved["tableSnapshots"].(map[string]any)
	objectSnapshot := snapshots["tp_sys_object"].(map[string]any)
	rows := objectSnapshot["rows"].([]any)
	if rows[0].(map[string]any)["id"] != "obj-replay-evidence" {
		t.Fatalf("expected collected row evidence to be preserved, got %#v", rows)
	}
}

func TestSetupSvcLiveReplayNormalizedDiffExecuteRequiresApproval(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)

	var stdout bytes.Buffer
	err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-normalized-diff", "--execute", "--approval", "WRONG"}, &stdout, tmp)
	if err == nil || !strings.Contains(err.Error(), setupSvcParityNormalizedDiffApproval) {
		t.Fatalf("expected normalized diff approval error, got %v", err)
	}
}

func TestSetupSvcLiveReplayNormalizedDiffDryRunDoesNotWriteFiles(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	diffPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/objects/create/normalized-diff.json")
	before, err := os.ReadFile(diffPath)
	if err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-normalized-diff", "--dry-run"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "dry_run_ready" || result["readOnly"] != true {
		t.Fatalf("expected dry-run normalized diff result, got %#v", result)
	}
	totals := result["totals"].(map[string]any)
	if int(totals["cleanOperations"].(float64)) != setupSvcLiveReplayOperationCount() || int(totals["writtenFiles"].(float64)) != 0 {
		t.Fatalf("expected all operations clean and no writes, got %#v", totals)
	}
	after, err := os.ReadFile(diffPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatalf("dry-run must not rewrite normalized diff artifact")
	}
}

func TestSetupSvcLiveReplayNormalizedDiffExecuteWritesCleanDiffAndManifestStatus(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	markSetupSvcLiveReplayManifestOperationStatus(t, manifestPath, "objects", "create", "normalizedDiffStatus", "pending")

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-normalized-diff", "--execute", "--approval", setupSvcParityNormalizedDiffApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "applied" || result["readOnly"] != false {
		t.Fatalf("expected normalized diff applied, got %#v", result)
	}
	totals := result["totals"].(map[string]any)
	if int(totals["writtenFiles"].(float64)) != setupSvcLiveReplayOperationCount()+1 || int(totals["dirtyOperations"].(float64)) != 0 {
		t.Fatalf("expected all diff artifacts plus manifest written, got %#v", totals)
	}
	diff := readTestJSONMap(t, filepath.Join(tmp, "outputs/setup-svc-live-replay/objects/create/normalized-diff.json"))
	if diff["status"] != "passed" || diff["artifactType"] != "normalized-diff" {
		t.Fatalf("expected generated passed normalized diff artifact, got %#v", diff)
	}
	diffTotals := diff["totals"].(map[string]any)
	if int(diffTotals["differences"].(float64)) != 0 || int(diffTotals["failed"].(float64)) != 0 {
		t.Fatalf("expected clean diff totals, got %#v", diffTotals)
	}
	manifest := readTestJSONMap(t, manifestPath)
	status := setupSvcLiveReplayManifestOperationField(t, manifest, "objects", "create", "normalizedDiffStatus")
	if status != "passed" {
		t.Fatalf("expected manifest normalizedDiffStatus updated to passed, got %s", status)
	}
}

func TestSetupSvcLiveReplayNormalizedDiffDetectsSnapshotDifferences(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	snapshots := setupSvcLiveReplayTestTableSnapshots("objects")
	snapshots[0]["rows"] = []map[string]any{{"id": "metadata-service-different-row"}}
	writeSetupSvcLiveReplayArtifact(t, tmp, "objects", "create", "outputs/setup-svc-live-replay/objects/create/metadata-service.json", map[string]any{
		"status":         "passed",
		"tableSnapshots": snapshots,
	})

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-normalized-diff", "--dry-run"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "diffs_failed" {
		t.Fatalf("expected dirty normalized diff status, got %#v", result)
	}
	totals := result["totals"].(map[string]any)
	if int(totals["dirtyOperations"].(float64)) != 1 {
		t.Fatalf("expected one dirty operation, got %#v", totals)
	}
	first := result["domains"].([]any)[0].(map[string]any)["operations"].([]any)[0].(map[string]any)
	if first["status"] != "dirty" || int(first["differences"].(float64)) == 0 {
		t.Fatalf("expected objects/create dirty operation with differences, got %#v", first)
	}
}

func TestSetupSvcLiveReplayManifestSyncExecuteRequiresApproval(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)

	var stdout bytes.Buffer
	err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-manifest-sync", "--execute", "--approval", "WRONG"}, &stdout, tmp)
	if err == nil || !strings.Contains(err.Error(), setupSvcParityManifestSyncApproval) {
		t.Fatalf("expected manifest sync approval error, got %v", err)
	}
}

func TestSetupSvcLiveReplayManifestSyncDryRunDoesNotWriteManifest(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	markSetupSvcLiveReplayManifestOperationStatus(t, manifestPath, "objects", "create", "setupSvcEvidenceStatus", "pending")
	before, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-manifest-sync", "--dry-run"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "passed" || result["readOnly"] != true || result["execute"] != false {
		t.Fatalf("expected dry-run manifest sync to derive passed status, got %#v", result)
	}
	totals := result["totals"].(map[string]any)
	if int(totals["updatedOperations"].(float64)) != 1 || int(totals["writtenFiles"].(float64)) != 0 {
		t.Fatalf("expected one derived operation update and no writes, got %#v", totals)
	}
	for _, field := range []string{"artifactFiles", "passedArtifacts", "pendingArtifacts", "failedArtifacts", "passedOperations", "pendingOperations", "failedOperations", "updatedOperations", "writtenFiles"} {
		if result[field] != totals[field] {
			t.Fatalf("expected top-level %s to mirror totals, got top=%#v totals=%#v", field, result[field], totals[field])
		}
	}
	after, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatalf("dry-run must not rewrite manifest")
	}
}

func TestSetupSvcLiveReplayManifestSyncExecuteUpdatesStatusesAndVerifierPasses(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	writeSetupSvcLiveReplayParityMatrix(t, tmp, nil)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	rewriteSetupSvcLiveReplayManifest(t, manifestPath, func(manifest map[string]any) {
		manifest["status"] = "pending"
		for _, rawDomain := range manifest["domains"].([]any) {
			domain := rawDomain.(map[string]any)
			for _, rawOperation := range domain["operations"].([]any) {
				operation := rawOperation.(map[string]any)
				operation["setupSvcEvidenceStatus"] = "pending"
				operation["metadataServiceEvidenceStatus"] = "pending"
				operation["queryEvidenceStatus"] = "pending"
				operation["normalizedDiffStatus"] = "pending"
				if _, ok := operation["cleanupStatus"]; ok {
					operation["cleanupStatus"] = "pending"
				}
			}
		}
	})

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-manifest-sync", "--execute", "--approval", setupSvcParityManifestSyncApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "passed" || result["readOnly"] != false || result["approved"] != true {
		t.Fatalf("expected manifest sync applied passed result, got %#v", result)
	}
	totals := result["totals"].(map[string]any)
	if int(totals["passedOperations"].(float64)) != setupSvcLiveReplayOperationCount() || int(totals["updatedOperations"].(float64)) != setupSvcLiveReplayOperationCount() || int(totals["writtenFiles"].(float64)) != 1 {
		t.Fatalf("expected all operations synced and manifest written, got %#v", totals)
	}
	manifest := readTestJSONMap(t, manifestPath)
	if manifest["status"] != "passed" {
		t.Fatalf("expected manifest status passed, got %#v", manifest["status"])
	}
	if status := setupSvcLiveReplayManifestOperationField(t, manifest, "objects", "create", "setupSvcEvidenceStatus"); status != "passed" {
		t.Fatalf("expected objects/create setupSvcEvidenceStatus passed, got %s", status)
	}

	stdout.Reset()
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var evidence map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &evidence); err != nil {
		t.Fatal(err)
	}
	if evidence["status"] != "passed" {
		t.Fatalf("synced manifest should pass strict evidence verifier, got %#v", evidence)
	}
}

func TestSetupSvcLiveReplayManifestSyncMarksInvalidArtifactFailed(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeTestFile(t, filepath.Join(tmp, "outputs/setup-svc-live-replay/objects/create/query-readback.json"), `{"status":"passed"`)

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-manifest-sync", "--execute", "--approval", setupSvcParityManifestSyncApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "failed_evidence" {
		t.Fatalf("expected failed evidence sync result, got %#v", result)
	}
	totals := result["totals"].(map[string]any)
	if int(totals["failedOperations"].(float64)) != 1 || int(totals["failedArtifacts"].(float64)) != 1 {
		t.Fatalf("expected one failed artifact/operation, got %#v", totals)
	}
	manifest := readTestJSONMap(t, manifestPath)
	if manifest["status"] != "failed" {
		t.Fatalf("expected failed manifest status, got %#v", manifest["status"])
	}
	if status := setupSvcLiveReplayManifestOperationField(t, manifest, "objects", "create", "queryEvidenceStatus"); status != "failed" {
		t.Fatalf("expected queryEvidenceStatus failed, got %s", status)
	}
}

func TestSetupSvcLiveReplayManifestSyncMarksMissingRuntimeReadbackEvidenceFailed(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayMissingRuntimeReadbackEvidenceArtifacts(t, tmp, manifestPath)

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-manifest-sync", "--execute", "--approval", setupSvcParityManifestSyncApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "failed_evidence" {
		t.Fatalf("expected failed evidence sync result, got %#v", result)
	}
	totals := result["totals"].(map[string]any)
	if int(totals["failedOperations"].(float64)) != 1 || int(totals["failedArtifacts"].(float64)) != 2 {
		t.Fatalf("expected one failed operation with two failed artifacts, got %#v", totals)
	}
	first := result["domains"].([]any)[0].(map[string]any)["operations"].([]any)[0].(map[string]any)
	artifactStatuses := first["artifactStatuses"].([]any)
	if !containsStringItem(artifactStatuses[0].(map[string]any)["issues"].([]any), "runtimeEffectsMissingEvidence=datatable-prefix-allocation") ||
		!containsStringItem(artifactStatuses[2].(map[string]any)["issues"].([]any), "queryReadbackExpectationsMissingEvidence=object-identity-prefix-datatable-readback") {
		t.Fatalf("expected runtime/readback missing evidence issues, got %#v", artifactStatuses)
	}
	manifest := readTestJSONMap(t, manifestPath)
	if status := setupSvcLiveReplayManifestOperationField(t, manifest, "objects", "create", "setupSvcEvidenceStatus"); status != "failed" {
		t.Fatalf("expected setupSvcEvidenceStatus failed, got %s", status)
	}
	if status := setupSvcLiveReplayManifestOperationField(t, manifest, "objects", "create", "queryEvidenceStatus"); status != "failed" {
		t.Fatalf("expected queryEvidenceStatus failed, got %s", status)
	}
}

func TestSetupSvcLiveReplayEvidenceImportValidatesAndWritesBatch(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	packet := buildSetupSvcLiveReplayPacket(tmp)
	if result, err := buildSetupSvcLiveReplayWorkspaceApplyResult(tmp, packet, true, setupSvcParityEvidenceWorkspaceApproval); err != nil || result.Status != "applied" {
		t.Fatalf("expected workspace prepared, result=%#v err=%v", result, err)
	}
	queryFile := filepath.Join("outputs", "setup-svc-live-replay", "objects", "create", "query-readback.json")
	queryPath := filepath.Join(tmp, queryFile)
	before, err := os.ReadFile(queryPath)
	if err != nil {
		t.Fatal(err)
	}
	artifact := map[string]any{
		"status":              "passed",
		"contractVersion":     setupSvcLiveReplayContractVersion,
		"contractFingerprint": setupSvcLiveReplayExpectedContractFingerprint(),
		"project":             tmp,
		"domain":              "objects",
		"operation":           "create",
		"artifactType":        "query-readback",
		"readbackChecks": map[string]any{
			"requiredFields":            []string{"id"},
			"requiredRelationships":     []string{"metadata-table-links"},
			"relationshipChecks":        []map[string]any{{"name": "metadata-table-links", "status": "passed", "source": "tp_sys_object", "target": "tp_sys_schemetable", "field": "id"}},
			"readbackExpectationChecks": setupSvcLiveReplayPassedExpectationChecks(setupSvcLiveReplayQueryReadbackExpectations("objects")),
			"missingFields":             0,
			"missingRelationships":      0,
			"mismatchedFields":          0,
			"brokenRelationships":       0,
			"unreadableRelationships":   0,
		},
		"readbackTables": setupSvcLiveReplayTestReadbackTables("objects"),
	}
	importBody, err := json.Marshal(map[string]any{
		"manifestPath": filepath.Join("outputs", "setup-svc-live-replay", "manifest.json"),
		"artifacts": []map[string]any{{
			"path":     queryFile,
			"artifact": artifact,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-evidence-import", string(importBody), "--dry-run"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var dryRun map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &dryRun); err != nil {
		t.Fatal(err)
	}
	if dryRun["status"] != "dry_run_ready" || dryRun["readOnly"] != true {
		t.Fatalf("expected dry-run import to pass without writes, got %#v", dryRun)
	}
	totals := dryRun["totals"].(map[string]any)
	if int(totals["passed"].(float64)) != 1 || int(totals["writtenFiles"].(float64)) != 0 {
		t.Fatalf("expected one validated artifact and no writes, got %#v", totals)
	}
	if int(dryRun["artifactCount"].(float64)) != 1 ||
		int(dryRun["passedArtifacts"].(float64)) != 1 ||
		int(dryRun["failedArtifacts"].(float64)) != 0 ||
		int(dryRun["skippedDuplicateRecords"].(float64)) != 0 ||
		int(dryRun["writtenFiles"].(float64)) != 0 {
		t.Fatalf("expected top-level import totals to mirror nested totals, got %#v", dryRun)
	}
	afterDryRun, err := os.ReadFile(queryPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(afterDryRun) {
		t.Fatalf("dry-run must not rewrite imported evidence artifact")
	}

	stdout.Reset()
	err = Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-evidence-import", string(importBody), "--execute", "--approval", "WRONG"}, &stdout, tmp)
	if err == nil || !strings.Contains(err.Error(), setupSvcParityEvidenceImportApproval) {
		t.Fatalf("expected evidence import approval error, got %v", err)
	}

	stdout.Reset()
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-evidence-import", string(importBody), "--execute", "--approval", setupSvcParityEvidenceImportApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var applied map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &applied); err != nil {
		t.Fatal(err)
	}
	if applied["status"] != "applied" || applied["approved"] != true {
		t.Fatalf("expected evidence import applied, got %#v", applied)
	}
	if int(applied["artifactCount"].(float64)) != 1 || int(applied["writtenFiles"].(float64)) != 1 {
		t.Fatalf("expected top-level applied totals to include written file count, got %#v", applied)
	}
	written := readTestJSONMap(t, queryPath)
	if written["status"] != "passed" || written["artifactType"] != "query-readback" {
		t.Fatalf("expected imported query-readback artifact, got %#v", written)
	}

	stdout.Reset()
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-manifest-sync", "--execute", "--approval", setupSvcParityManifestSyncApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	manifest := readTestJSONMap(t, filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json"))
	if status := setupSvcLiveReplayManifestOperationField(t, manifest, "objects", "create", "queryEvidenceStatus"); status != "passed" {
		t.Fatalf("expected imported query evidence to sync as passed, got %s", status)
	}
}

func TestSetupSvcLiveReplayEvidenceImportReadsSourceFiles(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	packet := buildSetupSvcLiveReplayPacket(tmp)
	if result, err := buildSetupSvcLiveReplayWorkspaceApplyResult(tmp, packet, true, setupSvcParityEvidenceWorkspaceApproval); err != nil || result.Status != "applied" {
		t.Fatalf("expected workspace prepared, result=%#v err=%v", result, err)
	}
	queryFile := filepath.Join("outputs", "setup-svc-live-replay", "objects", "create", "query-readback.json")
	queryPath := filepath.Join(tmp, queryFile)
	artifact := map[string]any{
		"status":              "passed",
		"contractVersion":     setupSvcLiveReplayContractVersion,
		"contractFingerprint": setupSvcLiveReplayExpectedContractFingerprint(),
		"project":             tmp,
		"domain":              "objects",
		"operation":           "create",
		"artifactType":        "query-readback",
		"readbackChecks": map[string]any{
			"requiredFields":            []string{"id"},
			"requiredRelationships":     []string{"metadata-table-links"},
			"relationshipChecks":        []map[string]any{{"name": "metadata-table-links", "status": "passed", "source": "tp_sys_object", "target": "tp_sys_schemetable", "field": "id"}},
			"readbackExpectationChecks": setupSvcLiveReplayPassedExpectationChecks(setupSvcLiveReplayQueryReadbackExpectations("objects")),
			"missingFields":             0,
			"missingRelationships":      0,
			"mismatchedFields":          0,
			"brokenRelationships":       0,
			"unreadableRelationships":   0,
		},
		"readbackTables": setupSvcLiveReplayTestReadbackTables("objects"),
	}
	artifactBody, err := json.Marshal(artifact)
	if err != nil {
		t.Fatal(err)
	}
	sourceFile := filepath.Join("captures", "objects-create-query-readback.json")
	writeTestFile(t, filepath.Join(tmp, sourceFile), string(artifactBody))
	importBody, err := json.Marshal(map[string]any{
		"manifestPath": filepath.Join("outputs", "setup-svc-live-replay", "manifest.json"),
		"artifactReplacementRecords": []map[string]any{{
			"targetPath":   queryFile,
			"sourcePath":   sourceFile,
			"domain":       "objects",
			"operation":    "create",
			"artifactType": "query-readback",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-evidence-import", string(importBody), "--dry-run"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var dryRun map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &dryRun); err != nil {
		t.Fatal(err)
	}
	if dryRun["status"] != "dry_run_ready" {
		t.Fatalf("expected source-file dry-run import to pass, got %#v", dryRun)
	}
	record := dryRun["artifacts"].([]any)[0].(map[string]any)
	if record["sourcePath"] == "" || record["status"] != "ready" {
		t.Fatalf("expected sourcePath to be reported with ready status, got %#v", record)
	}

	stdout.Reset()
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-evidence-import", string(importBody), "--execute", "--approval", setupSvcParityEvidenceImportApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var applied map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &applied); err != nil {
		t.Fatal(err)
	}
	if applied["status"] != "applied" {
		t.Fatalf("expected source-file import applied, got %#v", applied)
	}
	written := readTestJSONMap(t, queryPath)
	if written["status"] != "passed" || written["artifactType"] != "query-readback" {
		t.Fatalf("expected source-file imported query-readback artifact, got %#v", written)
	}

	stdout.Reset()
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-manifest-sync", "--execute", "--approval", setupSvcParityManifestSyncApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	manifest := readTestJSONMap(t, filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json"))
	if status := setupSvcLiveReplayManifestOperationField(t, manifest, "objects", "create", "queryEvidenceStatus"); status != "passed" {
		t.Fatalf("expected source-file query evidence to sync as passed, got %s", status)
	}
}

func TestSetupSvcLiveReplayEvidenceImportReadsNestedWorklistSourceFiles(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	packet := buildSetupSvcLiveReplayPacket(tmp)
	if result, err := buildSetupSvcLiveReplayWorkspaceApplyResult(tmp, packet, true, setupSvcParityEvidenceWorkspaceApproval); err != nil || result.Status != "applied" {
		t.Fatalf("expected workspace prepared, result=%#v err=%v", result, err)
	}

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{
		tmp,
		"setup-svc-live-replay-worklist",
		"--artifact-type", "query-readback",
		"--evidence-section", "readbackTables",
		"--section-status", "missing",
		"--limit", "1",
		"--batch-index", "0",
	}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var worklist map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &worklist); err != nil {
		t.Fatal(err)
	}
	queue := worklist["queues"].([]any)[0].(map[string]any)
	batch := queue["batches"].([]any)[0].(map[string]any)
	operatorBatch := batch["operatorBatch"].(map[string]any)
	replacementRecord := operatorBatch["artifactReplacementRecords"].([]any)[0].(map[string]any)
	domain := replacementRecord["domain"].(string)
	operation := replacementRecord["operation"].(string)
	artifactType := replacementRecord["artifactType"].(string)
	sourceFile := filepath.Join("captures", domain+"-"+operation+"-"+artifactType+".json")
	artifact := map[string]any{
		"status":              "passed",
		"contractVersion":     setupSvcLiveReplayContractVersion,
		"contractFingerprint": setupSvcLiveReplayExpectedContractFingerprint(),
		"project":             tmp,
		"domain":              domain,
		"operation":           operation,
		"artifactType":        artifactType,
		"readbackChecks": map[string]any{
			"requiredFields":            []string{"id"},
			"requiredRelationships":     []string{"metadata-table-links"},
			"relationshipChecks":        []map[string]any{{"name": "metadata-table-links", "status": "passed", "source": "tp_sys_object", "target": "tp_sys_schemetable", "field": "id"}},
			"readbackExpectationChecks": setupSvcLiveReplayPassedExpectationChecks(setupSvcLiveReplayQueryReadbackExpectations(domain)),
			"missingFields":             0,
			"missingRelationships":      0,
			"mismatchedFields":          0,
			"brokenRelationships":       0,
			"unreadableRelationships":   0,
		},
		"readbackTables": setupSvcLiveReplayTestReadbackTables(domain),
	}
	artifactBody, err := json.Marshal(artifact)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(tmp, sourceFile), string(artifactBody))
	worklist["manifestPath"] = filepath.Join("outputs", "setup-svc-live-replay", "manifest.json")
	worklist["sourceRoot"] = "captures"
	replacementRecord["sourceFile"] = filepath.Base(sourceFile)
	operatorBatch["artifactReplacementRecords"] = append(operatorBatch["artifactReplacementRecords"].([]any), replacementRecord)
	importBody, err := json.Marshal(worklist)
	if err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-evidence-import", string(importBody), "--dry-run"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var dryRun map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &dryRun); err != nil {
		t.Fatal(err)
	}
	if dryRun["status"] != "dry_run_ready" {
		t.Fatalf("expected nested worklist source-file import dry-run to pass, got %#v", dryRun)
	}
	totals := dryRun["totals"].(map[string]any)
	if int(totals["passed"].(float64)) != 1 ||
		int(totals["artifacts"].(float64)) != 1 ||
		int(totals["skippedDuplicateRecords"].(float64)) != 1 {
		t.Fatalf("expected duplicate nested replacement records to import once and skip duplicate, got %#v", totals)
	}

	stdout.Reset()
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-evidence-import", string(importBody), "--execute", "--approval", setupSvcParityEvidenceImportApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var applied map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &applied); err != nil {
		t.Fatal(err)
	}
	if applied["status"] != "applied" {
		t.Fatalf("expected nested worklist source-file import applied, got %#v", applied)
	}

	stdout.Reset()
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-manifest-sync", "--execute", "--approval", setupSvcParityManifestSyncApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	manifest := readTestJSONMap(t, filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json"))
	if status := setupSvcLiveReplayManifestOperationField(t, manifest, domain, operation, "queryEvidenceStatus"); status != "passed" {
		t.Fatalf("expected nested worklist query evidence to sync as passed, got %s", status)
	}
}

func TestSetupSvcLiveReplayEvidenceImportAutoMatchesWorklistSourceRoot(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	packet := buildSetupSvcLiveReplayPacket(tmp)
	if result, err := buildSetupSvcLiveReplayWorkspaceApplyResult(tmp, packet, true, setupSvcParityEvidenceWorkspaceApproval); err != nil || result.Status != "applied" {
		t.Fatalf("expected workspace prepared, result=%#v err=%v", result, err)
	}

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{
		tmp,
		"setup-svc-live-replay-worklist",
		"--artifact-type", "query-readback",
		"--evidence-section", "readbackTables",
		"--section-status", "missing",
		"--limit", "1",
		"--batch-index", "0",
	}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var worklist map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &worklist); err != nil {
		t.Fatal(err)
	}
	queue := worklist["queues"].([]any)[0].(map[string]any)
	batch := queue["batches"].([]any)[0].(map[string]any)
	operatorBatch := batch["operatorBatch"].(map[string]any)
	replacementRecord := operatorBatch["artifactReplacementRecords"].([]any)[0].(map[string]any)
	domain := replacementRecord["domain"].(string)
	operation := replacementRecord["operation"].(string)
	artifactType := replacementRecord["artifactType"].(string)
	targetPath := replacementRecord["path"].(string)
	artifact := map[string]any{
		"status":              "passed",
		"contractVersion":     setupSvcLiveReplayContractVersion,
		"contractFingerprint": setupSvcLiveReplayExpectedContractFingerprint(),
		"project":             tmp,
		"domain":              domain,
		"operation":           operation,
		"artifactType":        artifactType,
		"readbackChecks": map[string]any{
			"requiredFields":            []string{"id"},
			"requiredRelationships":     []string{"metadata-table-links"},
			"relationshipChecks":        []map[string]any{{"name": "metadata-table-links", "status": "passed", "source": "tp_sys_object", "target": "tp_sys_schemetable", "field": "id"}},
			"readbackExpectationChecks": setupSvcLiveReplayPassedExpectationChecks(setupSvcLiveReplayQueryReadbackExpectations(domain)),
			"missingFields":             0,
			"missingRelationships":      0,
			"mismatchedFields":          0,
			"brokenRelationships":       0,
			"unreadableRelationships":   0,
		},
		"readbackTables": setupSvcLiveReplayTestReadbackTables(domain),
	}
	artifactBody, err := json.Marshal(artifact)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(tmp, "captures", targetPath), string(artifactBody))
	worklist["manifestPath"] = filepath.Join("outputs", "setup-svc-live-replay", "manifest.json")
	worklist["sourceRoot"] = "captures"
	importBody, err := json.Marshal(worklist)
	if err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-evidence-import", string(importBody), "--dry-run"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var dryRun map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &dryRun); err != nil {
		t.Fatal(err)
	}
	if dryRun["status"] != "dry_run_ready" {
		t.Fatalf("expected mirrored sourceRoot worklist import dry-run to pass, got %#v", dryRun)
	}
	record := dryRun["artifacts"].([]any)[0].(map[string]any)
	if !strings.Contains(record["sourcePath"].(string), filepath.Join("captures", targetPath)) {
		t.Fatalf("expected mirrored sourcePath to be reported, got %#v", record)
	}

	stdout.Reset()
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-evidence-import", string(importBody), "--execute", "--approval", setupSvcParityEvidenceImportApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var applied map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &applied); err != nil {
		t.Fatal(err)
	}
	if applied["status"] != "applied" {
		t.Fatalf("expected mirrored sourceRoot worklist import applied, got %#v", applied)
	}
	written := readTestJSONMap(t, filepath.Join(tmp, targetPath))
	if written["status"] != "passed" || written["artifactType"] != artifactType {
		t.Fatalf("expected mirrored sourceRoot imported artifact, got %#v", written)
	}
}

func TestSetupSvcLiveReplayEvidenceImportBlocksMissingPayload(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	packet := buildSetupSvcLiveReplayPacket(tmp)
	if result, err := buildSetupSvcLiveReplayWorkspaceApplyResult(tmp, packet, true, setupSvcParityEvidenceWorkspaceApproval); err != nil || result.Status != "applied" {
		t.Fatalf("expected workspace prepared, result=%#v err=%v", result, err)
	}
	queryFile := filepath.Join("outputs", "setup-svc-live-replay", "objects", "create", "query-readback.json")
	queryPath := filepath.Join(tmp, queryFile)
	before, err := os.ReadFile(queryPath)
	if err != nil {
		t.Fatal(err)
	}
	importBody, err := json.Marshal(map[string]any{
		"manifestPath": filepath.Join("outputs", "setup-svc-live-replay", "manifest.json"),
		"operatorPacket": map[string]any{
			"suggestedWorklistPath": "/tmp/worklist-query-readback-batch-0.json",
			"saveWorklistCommand":   "cloudcc scan msapi " + tmp + " setup-svc-live-replay-worklist --batch-index 0 > /tmp/worklist-query-readback-batch-0.json",
			"dryRunImportCommand":   "cloudcc apply msapi " + tmp + " setup-svc-live-replay-evidence-import @/tmp/worklist-query-readback-batch-0.json --dry-run",
			"executeImportCommand":  "cloudcc apply msapi " + tmp + " setup-svc-live-replay-evidence-import @/tmp/worklist-query-readback-batch-0.json --execute --approval " + setupSvcParityEvidenceImportApproval,
		},
		"artifactReplacementRecords": []map[string]any{{
			"targetPath":          queryFile,
			"domain":              "objects",
			"operation":           "create",
			"artifactType":        "query-readback",
			"suggestedSourcePath": filepath.Join("captures", queryFile),
			"missingEvidenceSections": []string{
				"readbackTables",
				"cleanCounters",
			},
			"captureTask": map[string]any{
				"sourceSystem":            "metadata-service",
				"postCaptureCheckCommand": "cloudcc scan msapi " + tmp + " setup-svc-live-replay-capture-plan --domain objects --operation create --artifact-type query-readback --source-readiness complete --limit 1",
			},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-evidence-import", string(importBody), "--execute", "--approval", setupSvcParityEvidenceImportApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("expected missing payload import to block, got %#v", result)
	}
	totals := result["totals"].(map[string]any)
	if int(totals["failed"].(float64)) != 1 || int(totals["writtenFiles"].(float64)) != 0 {
		t.Fatalf("expected one failed artifact and no writes, got %#v", totals)
	}
	if int(result["artifactCount"].(float64)) != 1 ||
		int(result["failedArtifacts"].(float64)) != 1 ||
		int(result["writtenFiles"].(float64)) != 0 {
		t.Fatalf("expected blocked import top-level totals to mirror nested totals, got %#v", result)
	}
	nextCommands := result["nextCommands"].(map[string]any)
	if nextCommands["suggestedWorklistPath"] != "/tmp/worklist-query-readback-batch-0.json" ||
		!strings.Contains(nextCommands["saveCurrentWorklist"].(string), "--batch-index 0 > /tmp/worklist-query-readback-batch-0.json") ||
		!strings.Contains(nextCommands["dryRunCurrentImport"].(string), "setup-svc-live-replay-evidence-import @/tmp/worklist-query-readback-batch-0.json --dry-run") ||
		!strings.Contains(nextCommands["executeCurrentImport"].(string), setupSvcParityEvidenceImportApproval) {
		t.Fatalf("expected blocked import to echo current worklist/import commands, got %#v", nextCommands)
	}
	repairSummary := result["repairSummary"].(map[string]any)
	if int(repairSummary["failedArtifacts"].(float64)) != 1 {
		t.Fatalf("expected one repair failed artifact, got %#v", repairSummary)
	}
	if !containsMapCount(repairSummary["issueCounts"].([]any), "missing artifact payload or sourcePath", 1) ||
		!containsSectionCount(repairSummary["missingEvidenceSections"].([]any), "readbackTables", 1) ||
		!containsSectionCount(repairSummary["missingEvidenceSections"].([]any), "cleanCounters", 1) ||
		!containsMapCount(repairSummary["artifactTypes"].([]any), "query-readback", 1) {
		t.Fatalf("expected repair summary issue/section/type counts, got %#v", repairSummary)
	}
	repairQueues := repairSummary["repairQueues"].([]any)
	if !containsRepairQueue(repairQueues, "query-readback", "readbackTables") ||
		!containsRepairQueue(repairQueues, "query-readback", "cleanCounters") {
		t.Fatalf("expected repair queues for missing evidence sections, got %#v", repairQueues)
	}
	sourceFiles := repairSummary["sourceFiles"].([]any)
	if len(sourceFiles) != 1 {
		t.Fatalf("expected one repair source file, got %#v", sourceFiles)
	}
	sourceFile := sourceFiles[0].(map[string]any)
	if sourceFile["targetPath"] != queryFile ||
		sourceFile["artifactType"] != "query-readback" ||
		!containsStringItem(sourceFile["missingEvidenceSections"].([]any), "readbackTables") {
		t.Fatalf("expected repair source file details, got %#v", sourceFile)
	}
	captureTask := sourceFile["captureTask"].(map[string]any)
	if captureTask["sourceSystem"] != "metadata-service" ||
		!strings.Contains(captureTask["postCaptureCheckCommand"].(string), "setup-svc-live-replay-capture-plan") {
		t.Fatalf("expected repair source capture task, got %#v", captureTask)
	}
	after, err := os.ReadFile(queryPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatalf("blocked import must not rewrite evidence artifact")
	}
}

func TestSetupSvcLiveReplayEvidenceBundleExecuteRequiresApproval(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	writeSetupSvcLiveReplayParityMatrix(t, tmp, nil)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)

	var stdout bytes.Buffer
	err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-evidence-bundle", "--execute", "--approval", "WRONG"}, &stdout, tmp)
	if err == nil || !strings.Contains(err.Error(), setupSvcParityEvidenceBundleApproval) {
		t.Fatalf("expected evidence bundle approval error, got %v", err)
	}
}

func TestSetupSvcLiveReplayEvidenceBundleDryRunDoesNotWriteFile(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	writeSetupSvcLiveReplayParityMatrix(t, tmp, nil)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	bundlePath := filepath.Join(tmp, "outputs/setup-svc-live-replay/evidence-bundle.json")

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-evidence-bundle", "--dry-run"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "dry_run_ready" || result["readOnly"] != true || result["evidenceStatus"] != "passed" {
		t.Fatalf("expected dry-run evidence bundle ready, got %#v", result)
	}
	totals := result["totals"].(map[string]any)
	wantFiles := setupSvcLiveReplayExpectedArtifactFileCount() + 1
	if int(totals["evidenceFiles"].(float64)) != wantFiles || int(totals["artifactFiles"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount() || int(totals["writtenFiles"].(float64)) != 0 {
		t.Fatalf("unexpected bundle totals: %#v", totals)
	}
	if _, err := os.Stat(bundlePath); !os.IsNotExist(err) {
		t.Fatalf("dry-run must not write bundle, stat err=%v", err)
	}
}

func TestSetupSvcLiveReplayEvidenceBundleExecuteWritesChecksums(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	writeSetupSvcLiveReplayParityMatrix(t, tmp, nil)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-evidence-bundle", "--execute", "--approval", setupSvcParityEvidenceBundleApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "applied" || result["readOnly"] != false || result["approved"] != true {
		t.Fatalf("expected applied evidence bundle, got %#v", result)
	}
	bundle := readTestJSONMap(t, filepath.Join(tmp, "outputs/setup-svc-live-replay/evidence-bundle.json"))
	if bundle["status"] != "applied" || bundle["evidenceStatus"] != "passed" {
		t.Fatalf("unexpected bundle header: %#v", bundle)
	}
	totals := bundle["totals"].(map[string]any)
	if int(totals["writtenFiles"].(float64)) != 1 || int(totals["verifiedOperations"].(float64)) != setupSvcLiveReplayOperationCount() {
		t.Fatalf("expected written bundle with verified operation totals, got %#v", totals)
	}
	files := bundle["files"].([]any)
	if len(files) != setupSvcLiveReplayExpectedArtifactFileCount()+1 {
		t.Fatalf("unexpected bundle file count %d", len(files))
	}
	foundManifest := false
	foundArtifact := false
	for _, raw := range files {
		file := raw.(map[string]any)
		if file["path"] == "outputs/setup-svc-live-replay/manifest.json" && file["artifactType"] == "manifest" && strings.HasPrefix(file["sha256"].(string), "sha256:") {
			foundManifest = true
		}
		if file["path"] == "outputs/setup-svc-live-replay/objects/create/setup-svc.json" && file["artifactType"] == "setup-svc" && file["domain"] == "objects" && file["operation"] == "create" && strings.HasPrefix(file["sha256"].(string), "sha256:") {
			foundArtifact = true
		}
	}
	if !foundManifest || !foundArtifact {
		t.Fatalf("expected manifest and objects/create setup-svc checksum entries, found manifest=%v artifact=%v", foundManifest, foundArtifact)
	}
}

func TestSetupSvcLiveReplayEvidenceBundleBlocksPendingEvidence(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	writeSetupSvcLiveReplayParityMatrix(t, tmp, nil)
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-workspace", "--execute", "--approval", setupSvcParityEvidenceWorkspaceApproval}, &bytes.Buffer{}, tmp); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-evidence-bundle", "--dry-run"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked_evidence_not_passed" || result["evidenceStatus"] != "blocked" {
		t.Fatalf("pending evidence must block bundle, got %#v", result)
	}
	if !containsStringItem(result["blockingIssues"].([]any), "evidence: blocked") {
		t.Fatalf("expected evidence blocked issue, got %#v", result["blockingIssues"])
	}
}

func TestSetupSvcLiveReplayEvidenceBundleBlocksMissingRuntimeReadbackEvidence(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	writeSetupSvcLiveReplayParityMatrix(t, tmp, nil)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayMissingRuntimeReadbackEvidenceArtifacts(t, tmp, manifestPath)

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-evidence-bundle", "--dry-run"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked_evidence_not_passed" || result["evidenceStatus"] != "blocked" {
		t.Fatalf("missing runtime/readback evidence must block bundle, got %#v", result)
	}
	issues := result["blockingIssues"].([]any)
	if !containsStringItem(issues, "evidence: blocked") ||
		!containsStringFragment(issues, "runtimeEffectsMissingEvidence=datatable-prefix-allocation") ||
		!containsStringFragment(issues, "queryReadbackExpectationsMissingEvidence=object-identity-prefix-datatable-readback") {
		t.Fatalf("expected strict evidence blockers in bundle result, got %#v", issues)
	}
}

func TestSetupSvcLiveReplayEvidenceBundleScanReportsMissingBundle(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	writeSetupSvcLiveReplayParityMatrix(t, tmp, nil)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence-bundle"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "missing" || result["readOnly"] != true {
		t.Fatalf("expected missing read-only bundle scan, got %#v", result)
	}
	if !containsStringItem(result["blockingIssues"].([]any), "evidenceBundle: missing") {
		t.Fatalf("expected missing bundle issue, got %#v", result["blockingIssues"])
	}
}

func TestSetupSvcLiveReplayEvidenceBundleScanPassesForCurrentBundle(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	writeSetupSvcLiveReplayParityMatrix(t, tmp, nil)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeCurrentSetupSvcLiveReplayEvidenceBundle(t, tmp)

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence-bundle"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "passed" || result["readOnly"] != true {
		t.Fatalf("expected passed read-only bundle scan, got %#v", result)
	}
	bundle := result["bundle"].(map[string]any)
	if bundle["status"] != "passed" || bundle["evidenceStatus"] != "passed" {
		t.Fatalf("expected passed nested bundle verification, got %#v", bundle)
	}
}

func TestSetupSvcLiveReplayEvidenceBundleScanDetectsStaleBundle(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	writeSetupSvcLiveReplayParityMatrix(t, tmp, nil)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeCurrentSetupSvcLiveReplayEvidenceBundle(t, tmp)
	artifactPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/objects/create/setup-svc.json")
	artifact := readTestJSONMap(t, artifactPath)
	artifact["checksumOnlyChange"] = true
	body, err := json.Marshal(artifact)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, artifactPath, string(body))

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence-bundle"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "stale" || result["readOnly"] != true {
		t.Fatalf("expected stale read-only bundle scan, got %#v", result)
	}
	if !containsStringItem(result["blockingIssues"].([]any), "evidenceBundle: stale file outputs/setup-svc-live-replay/objects/create/setup-svc.json") {
		t.Fatalf("expected stale bundle issue, got %#v", result["blockingIssues"])
	}
}

func TestSetupSvcLiveReplayPacketApplyDryRunBlocksUnexpectedDomain(t *testing.T) {
	tmp := t.TempDir()
	packet := buildSetupSvcLiveReplayPacket(tmp)
	packet.Domains = append(packet.Domains, setupSvcLiveReplayPacketDomain{Domain: "unsupported-domain"})

	result := runSetupSvcLiveReplayPacketDryRun(t, tmp, packet)
	if result["status"] != "blocked_incomplete_packet" {
		t.Fatalf("unexpected domain must block packet, got %#v", result)
	}
	issues := result["blockingIssues"].([]any)
	if !containsStringFragment(issues, "packet: unexpected domain unsupported-domain") {
		t.Fatalf("expected unexpected domain issue, got %#v", issues)
	}
}

func TestSetupSvcLiveReplayPacketApplyDryRunBlocksDuplicateDomain(t *testing.T) {
	tmp := t.TempDir()
	packet := buildSetupSvcLiveReplayPacket(tmp)
	packet.Domains = append(packet.Domains, packet.Domains[0])

	result := runSetupSvcLiveReplayPacketDryRun(t, tmp, packet)
	if result["status"] != "blocked_incomplete_packet" {
		t.Fatalf("duplicate domain must block packet, got %#v", result)
	}
	issues := result["blockingIssues"].([]any)
	if !containsStringFragment(issues, "packet: duplicate domain objects") {
		t.Fatalf("expected duplicate domain issue, got %#v", issues)
	}
}

func TestSetupSvcLiveReplayPacketApplyDryRunBlocksUnexpectedOperation(t *testing.T) {
	tmp := t.TempDir()
	packet := buildSetupSvcLiveReplayPacket(tmp)
	packet.Domains[0].Operations = append(packet.Domains[0].Operations, setupSvcLiveReplayPacketOperation{
		Operation:        "unsupported-op",
		Status:           "pending",
		ReadOnly:         false,
		RequiredEvidence: setupSvcLiveReplayRequiredEvidence("create"),
		EvidenceFiles:    setupSvcLiveReplayEvidenceFiles("objects", "unsupported-op", true),
	})

	result := runSetupSvcLiveReplayPacketDryRun(t, tmp, packet)
	if result["status"] != "blocked_incomplete_packet" {
		t.Fatalf("unexpected operation must block packet, got %#v", result)
	}
	issues := result["blockingIssues"].([]any)
	if !containsStringFragment(issues, "objects/unsupported-op: unexpected operation") {
		t.Fatalf("expected unexpected operation issue, got %#v", issues)
	}
}

func TestSetupSvcLiveReplayPacketApplyDryRunBlocksDuplicateOperation(t *testing.T) {
	tmp := t.TempDir()
	packet := buildSetupSvcLiveReplayPacket(tmp)
	packet.Domains[0].Operations = append(packet.Domains[0].Operations, packet.Domains[0].Operations[0])

	result := runSetupSvcLiveReplayPacketDryRun(t, tmp, packet)
	if result["status"] != "blocked_incomplete_packet" {
		t.Fatalf("duplicate operation must block packet, got %#v", result)
	}
	issues := result["blockingIssues"].([]any)
	if !containsStringFragment(issues, "objects/create: duplicate operation") {
		t.Fatalf("expected duplicate operation issue, got %#v", issues)
	}
}

func TestSetupSvcLiveReplayPacketApplyDryRunBlocksTamperedEnvelope(t *testing.T) {
	tmp := t.TempDir()
	packet := buildSetupSvcLiveReplayPacket(tmp)
	packet.Project = filepath.Join(tmp, "other-project")
	packet.Totals.Operations--

	result := runSetupSvcLiveReplayPacketDryRun(t, tmp, packet)
	if result["status"] != "blocked_incomplete_packet" {
		t.Fatalf("tampered packet envelope must block packet, got %#v", result)
	}
	issues := result["blockingIssues"].([]any)
	if !containsStringFragment(issues, "packet: project mismatch") {
		t.Fatalf("expected project mismatch issue, got %#v", issues)
	}
	if !containsStringFragment(issues, "packet: totals.operations mismatch") {
		t.Fatalf("expected totals.operations mismatch issue, got %#v", issues)
	}
}

func TestSetupSvcLiveReplayPacketApplyDryRunBlocksTamperedManifestTemplate(t *testing.T) {
	tmp := t.TempDir()
	packet := buildSetupSvcLiveReplayPacket(tmp)
	packet.ManifestTemplate.Project = filepath.Join(tmp, "other-project")
	duplicate := filepath.Join("outputs", "setup-svc-live-replay", "objects", "create", "setup-svc.json")
	packet.ManifestTemplate.Domains[0].Operations[0].EvidenceFiles = append(packet.ManifestTemplate.Domains[0].Operations[0].EvidenceFiles, duplicate)

	result := runSetupSvcLiveReplayPacketDryRun(t, tmp, packet)
	if result["status"] != "blocked_incomplete_packet" {
		t.Fatalf("tampered manifest template must block packet, got %#v", result)
	}
	issues := result["blockingIssues"].([]any)
	if !containsStringFragment(issues, "manifestTemplate: project mismatch") {
		t.Fatalf("expected manifestTemplate project mismatch issue, got %#v", issues)
	}
	if !containsStringFragment(issues, "manifestTemplate: objects/create: duplicate evidenceFiles outputs/setup-svc-live-replay/objects/create/setup-svc.json") {
		t.Fatalf("expected manifestTemplate duplicate evidenceFiles issue, got %#v", issues)
	}
}

func runSetupSvcLiveReplayPacketDryRun(t *testing.T, tmp string, packet setupSvcLiveReplayPacket) map[string]any {
	t.Helper()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	packetPath := filepath.Join(tmp, "setup-svc-live-replay-packet.json")
	b, err := json.Marshal(packet)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, packetPath, string(b))

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-packet", "@" + packetPath, "--dry-run"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func setupSvcLiveReplayExpectedArtifactFileCount() int {
	total := 0
	for _, domain := range setupSvcLiveReplayDomains() {
		for _, operation := range domain.Operations {
			total += len(setupSvcLiveReplayEvidenceFiles(domain.Domain, operation, operation != "query"))
		}
	}
	return total
}

func setupSvcLiveReplayWriteOperationCount() int {
	total := 0
	for _, domain := range setupSvcLiveReplayDomains() {
		for _, operation := range domain.Operations {
			if operation != "query" {
				total++
			}
		}
	}
	return total
}

func setupSvcLiveReplayExpectedMissingSectionRecordCount() int {
	// setup-svc(2) + metadata-service(3) + query-readback(6) +
	// normalized-diff(2) are required for every operation; cleanup(2) is
	// required only for writes.
	return 13*setupSvcLiveReplayOperationCount() + 2*setupSvcLiveReplayWriteOperationCount()
}

func TestSetupSvcLiveReplayPacketApplyDryRunBlocksTamperedRequiredTables(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	packetPath := filepath.Join(tmp, "setup-svc-live-replay-packet.json")
	packet := buildSetupSvcLiveReplayPacket(tmp)
	packet.Domains[0].RequiredTables = []string{"tp_sys_object", "tp_sys_extra_weak_table"}
	b, err := json.Marshal(packet)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, packetPath, string(b))

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-packet", "@" + packetPath, "--dry-run"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked_incomplete_packet" {
		t.Fatalf("tampered requiredTables must block packet, got %#v", result)
	}
	totals := result["totals"].(map[string]any)
	if int(totals["failedOperations"].(float64)) == 0 {
		t.Fatalf("expected failed operations for invalid requiredTables, got %#v", totals)
	}
	issues := result["blockingIssues"].([]any)
	if !containsStringFragment(issues, "missing requiredTables tp_sys_datatablestate") {
		t.Fatalf("expected missing requiredTables issue, got %#v", issues)
	}
	if !containsStringFragment(issues, "unexpected requiredTables tp_sys_extra_weak_table") {
		t.Fatalf("expected unexpected requiredTables issue, got %#v", issues)
	}
}

func TestSetupSvcLiveReplayPacketApplyDryRunBlocksTamperedRequiredEvidence(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	packetPath := filepath.Join(tmp, "setup-svc-live-replay-packet.json")
	packet := buildSetupSvcLiveReplayPacket(tmp)
	packet.Domains[0].Operations[0].RequiredEvidence = []string{
		"setupSvcEvidenceStatus",
		"metadataServiceEvidenceStatus",
		"queryEvidenceStatus",
		"normalizedDiffStatus",
	}
	b, err := json.Marshal(packet)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, packetPath, string(b))

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-packet", "@" + packetPath, "--dry-run"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked_incomplete_packet" {
		t.Fatalf("tampered requiredEvidence must block packet, got %#v", result)
	}
	totals := result["totals"].(map[string]any)
	if int(totals["failedOperations"].(float64)) == 0 {
		t.Fatalf("expected failed operation for invalid requiredEvidence, got %#v", totals)
	}
	if !containsStringFragment(result["blockingIssues"].([]any), "missing requiredEvidence cleanupStatus") {
		t.Fatalf("expected missing cleanupStatus issue, got %#v", result["blockingIssues"])
	}
}

func TestSetupSvcLiveReplayPacketApplyDryRunBlocksTamperedEvidenceFiles(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	packetPath := filepath.Join(tmp, "setup-svc-live-replay-packet.json")
	packet := buildSetupSvcLiveReplayPacket(tmp)
	packet.Domains[0].Operations[0].EvidenceFiles = []string{
		filepath.Join("outputs", "setup-svc-live-replay", "objects", "create", "setup-svc.json"),
		filepath.Join("outputs", "setup-svc-live-replay", "objects", "create", "metadata-service.json"),
		filepath.Join("outputs", "setup-svc-live-replay", "objects", "create", "normalized-diff.json"),
		filepath.Join("outputs", "setup-svc-live-replay", "objects", "create", "cleanup.json"),
		filepath.Join("outputs", "setup-svc-live-replay", "objects", "create", "extra.json"),
	}
	b, err := json.Marshal(packet)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, packetPath, string(b))

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-packet", "@" + packetPath, "--dry-run"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked_incomplete_packet" {
		t.Fatalf("tampered evidenceFiles must block packet, got %#v", result)
	}
	totals := result["totals"].(map[string]any)
	if int(totals["failedOperations"].(float64)) == 0 {
		t.Fatalf("expected failed operation for invalid evidenceFiles, got %#v", totals)
	}
	issues := result["blockingIssues"].([]any)
	if !containsStringFragment(issues, "missing evidenceFiles outputs/setup-svc-live-replay/objects/create/query-readback.json") {
		t.Fatalf("expected missing query-readback artifact issue, got %#v", issues)
	}
	if !containsStringFragment(issues, "unexpected evidenceFiles outputs/setup-svc-live-replay/objects/create/extra.json") {
		t.Fatalf("expected unexpected extra artifact issue, got %#v", issues)
	}
}

func TestSetupSvcLiveReplayPacketApplyDryRunBlocksDuplicateEvidenceFiles(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	packetPath := filepath.Join(tmp, "setup-svc-live-replay-packet.json")
	packet := buildSetupSvcLiveReplayPacket(tmp)
	duplicate := filepath.Join("outputs", "setup-svc-live-replay", "objects", "create", "setup-svc.json")
	packet.Domains[0].Operations[0].EvidenceFiles = append(packet.Domains[0].Operations[0].EvidenceFiles, duplicate)
	b, err := json.Marshal(packet)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, packetPath, string(b))

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-packet", "@" + packetPath, "--dry-run"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked_incomplete_packet" {
		t.Fatalf("duplicate evidenceFiles must block packet, got %#v", result)
	}
	issues := result["blockingIssues"].([]any)
	if !containsStringFragment(issues, "duplicate evidenceFiles outputs/setup-svc-live-replay/objects/create/setup-svc.json") {
		t.Fatalf("expected duplicate evidenceFiles issue, got %#v", issues)
	}
}

func TestSetupSvcLiveReplayPacketApplyExecuteRequiresApproval(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	packetPath := filepath.Join(tmp, "setup-svc-live-replay-packet.json")
	packet := buildSetupSvcLiveReplayPacket(tmp)
	b, err := json.Marshal(packet)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, packetPath, string(b))

	var stdout bytes.Buffer
	err = Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-packet", "@" + packetPath, "--execute", "--approval", "WRONG"}, &stdout, tmp)
	if err == nil || !strings.Contains(err.Error(), setupSvcParityReplayApproval) {
		t.Fatalf("expected approval failure, got %v", err)
	}
	if strings.TrimSpace(stdout.String()) != "" {
		t.Fatalf("approval failure should not emit success JSON, got %s", stdout.String())
	}
}

func TestSetupSvcLiveReplayEvidencePassesCompleteManifest(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "passed" || result["readOnly"] != true {
		t.Fatalf("expected passed read-only evidence result, got %#v", result)
	}
	totals := result["totals"].(map[string]any)
	if int(totals["verifiedDomains"].(float64)) != len(setupSvcLiveReplayDomains()) {
		t.Fatalf("expected all domains verified, got %#v", totals)
	}
	if int(totals["failedOperations"].(float64)) != 0 || int(totals["missingOperations"].(float64)) != 0 {
		t.Fatalf("expected no operation gaps, got %#v", totals)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksIncompleteManifest(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, false)

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("expected blocked evidence result, got %#v", result)
	}
	totals := result["totals"].(map[string]any)
	if int(totals["verifiedDomains"].(float64)) != 0 {
		t.Fatalf("incomplete manifest must not verify domains, got %#v", totals)
	}
	if int(totals["missingDomains"].(float64)) == 0 || int(totals["failedOperations"].(float64)) == 0 {
		t.Fatalf("expected missing domains and failed evidence, got %#v", totals)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksMissingArtifactFiles(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifestWithoutArtifacts(t, manifestPath)

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("passed statuses without artifact files must be blocked, got %#v", result)
	}
	totals := result["totals"].(map[string]any)
	if int(totals["verifiedDomains"].(float64)) != 0 || int(totals["failedOperations"].(float64)) == 0 {
		t.Fatalf("expected failed artifact evidence and no verified domains, got %#v", totals)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	failedEvidence := failed["failedEvidence"].([]any)
	found := false
	for _, item := range failedEvidence {
		if strings.HasPrefix(item.(string), "evidenceFileUnreadable:") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected unreadable evidence file issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksFailedArtifactStatus(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeTestFile(t, filepath.Join(tmp, "outputs/setup-svc-live-replay/objects/create/setup-svc.json"), `{"status":"failed"}`)

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("failed artifact status must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "evidenceFileStatusNotPassed:") {
		t.Fatalf("expected failed artifact status issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksCleanOnlyArtifactStatus(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeSetupSvcLiveReplayArtifact(t, tmp, "objects", "create", "outputs/setup-svc-live-replay/objects/create/setup-svc.json", map[string]any{
		"clean": true,
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("clean-only artifact status must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "evidenceFileStatusNotPassed:") {
		t.Fatalf("expected clean-only artifact status issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksBooleanOnlyArtifactStatus(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeSetupSvcLiveReplayArtifact(t, tmp, "objects", "create", "outputs/setup-svc-live-replay/objects/create/setup-svc.json", map[string]any{
		"passed": true,
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("boolean-only artifact status must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "evidenceFileStatusNotPassed:") {
		t.Fatalf("expected boolean-only artifact status issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksDirtyNormalizedDiffArtifact(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeSetupSvcLiveReplayArtifact(t, tmp, "objects", "create", "outputs/setup-svc-live-replay/objects/create/normalized-diff.json", map[string]any{"status": "passed", "totals": map[string]any{"missingRows": 1}})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("dirty normalized diff must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "normalizedDiffNotClean:") {
		t.Fatalf("expected dirty normalized diff issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksUnexpectedNormalizedDiffArtifact(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeSetupSvcLiveReplayArtifact(t, tmp, "objects", "create", "outputs/setup-svc-live-replay/objects/create/normalized-diff.json", map[string]any{
		"status": "passed",
		"totals": map[string]any{"unexpectedRows": 1},
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("unexpected normalized diff must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "normalizedDiffNotClean:") {
		t.Fatalf("expected unexpected normalized diff issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksNestedDirtyNormalizedDiffArtifact(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeSetupSvcLiveReplayArtifact(t, tmp, "objects", "create", "outputs/setup-svc-live-replay/objects/create/normalized-diff.json", map[string]any{
		"status": "passed",
		"totals": map[string]any{"missingRows": 0, "mismatchedValues": 0, "differences": 0},
		"diff":   map[string]any{"missingRows": 1},
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("nested dirty normalized diff must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "normalizedDiffNotClean:") {
		t.Fatalf("expected nested dirty normalized diff issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksNormalizedDiffWithoutCleanCounters(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	payload, err := json.Marshal(map[string]any{
		"status":              "passed",
		"project":             tmp,
		"contractVersion":     setupSvcLiveReplayContractVersion,
		"contractFingerprint": setupSvcLiveReplayExpectedContractFingerprint(),
		"domain":              "objects",
		"operation":           "create",
		"artifactType":        "normalized-diff",
	})
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(tmp, "outputs/setup-svc-live-replay/objects/create/normalized-diff.json"), string(payload))

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("normalized diff without clean counters must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "normalizedDiffMissingCleanEvidence:") {
		t.Fatalf("expected missing clean diff evidence issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksNormalizedDiffWithOnlyCleanFlag(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeSetupSvcLiveReplayArtifact(t, tmp, "objects", "create", "outputs/setup-svc-live-replay/objects/create/normalized-diff.json", map[string]any{
		"status": "passed",
		"clean":  true,
		"totals": map[string]any{},
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("normalized diff with only clean flag must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "normalizedDiffMissingCleanEvidence:") {
		t.Fatalf("expected missing clean diff counters issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksNormalizedDiffStringCleanCounters(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeSetupSvcLiveReplayArtifact(t, tmp, "objects", "create", "outputs/setup-svc-live-replay/objects/create/normalized-diff.json", map[string]any{
		"status": "passed",
		"totals": map[string]any{
			"missingRows":      "0",
			"mismatchedValues": "clean",
			"differences":      0,
		},
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("normalized diff string counters must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "normalizedDiffNotClean:") {
		t.Fatalf("expected invalid clean diff counter issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksQueryReadbackWithoutStructureProof(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	payload, err := json.Marshal(map[string]any{
		"status":              "passed",
		"project":             tmp,
		"contractVersion":     setupSvcLiveReplayContractVersion,
		"contractFingerprint": setupSvcLiveReplayExpectedContractFingerprint(),
		"domain":              "objects",
		"operation":           "create",
		"artifactType":        "query-readback",
	})
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(tmp, "outputs/setup-svc-live-replay/objects/create/query-readback.json"), string(payload))

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("query readback without structure proof must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "queryReadbackMissingStructureEvidence:") {
		t.Fatalf("expected missing query readback structure issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksQueryReadbackWithoutExpectationEvidence(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeSetupSvcLiveReplayArtifact(t, tmp, "objects", "create", "outputs/setup-svc-live-replay/objects/create/query-readback.json", map[string]any{
		"status":                    "passed",
		"queryReadbackExpectations": setupSvcLiveReplayQueryReadbackExpectations("objects"),
		"queryShape":                map[string]any{"fields": []string{"id"}},
		"readbackShape":             map[string]any{"fields": []string{"id"}},
		"readbackChecks": map[string]any{
			"requiredFields":          []string{"id"},
			"requiredRelationships":   []string{"metadata-table-links"},
			"relationshipChecks":      []map[string]any{{"name": "metadata-table-links", "status": "passed", "source": "tp_sys_object", "target": "tp_sys_schemetable", "field": "id"}},
			"missingFields":           0,
			"missingRelationships":    0,
			"mismatchedFields":        0,
			"brokenRelationships":     0,
			"unreadableRelationships": 0,
		},
		"readbackTables": setupSvcLiveReplayTestReadbackTables("objects"),
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("query readback without expectation evidence must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "queryReadbackExpectationsMissingEvidence=object-identity-prefix-datatable-readback") {
		t.Fatalf("expected query/readback expectation evidence blocker, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksQueryReadbackWithOnlyCleanCountersAsStructure(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeSetupSvcLiveReplayArtifact(t, tmp, "objects", "create", "outputs/setup-svc-live-replay/objects/create/query-readback.json", map[string]any{
		"status": "passed",
		"readbackChecks": map[string]any{
			"requiredRelationships":   []string{"metadata-table-links"},
			"relationshipChecks":      []map[string]any{{"name": "metadata-table-links", "status": "passed"}},
			"missingFields":           0,
			"missingRelationships":    0,
			"mismatchedFields":        0,
			"brokenRelationships":     0,
			"unreadableRelationships": 0,
		},
		"readbackTables": setupSvcLiveReplayTestReadbackTables("objects"),
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("query readback clean counters must not count as structure proof, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "queryReadbackMissingStructureEvidence:") {
		t.Fatalf("expected missing query readback structure issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksQueryReadbackWithoutCleanCounters(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeSetupSvcLiveReplayArtifact(t, tmp, "objects", "create", "outputs/setup-svc-live-replay/objects/create/query-readback.json", map[string]any{
		"status": "passed",
		"readbackChecks": map[string]any{
			"requiredFields": []string{"id"},
		},
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("query readback without clean counters must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "queryReadbackMissingCleanCounters:") {
		t.Fatalf("expected missing readback clean counters issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksQueryReadbackWithoutRelationshipEvidence(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeSetupSvcLiveReplayArtifact(t, tmp, "objects", "create", "outputs/setup-svc-live-replay/objects/create/query-readback.json", map[string]any{
		"status": "passed",
		"readbackChecks": map[string]any{
			"requiredFields":          []string{"id"},
			"missingFields":           0,
			"missingRelationships":    0,
			"mismatchedFields":        0,
			"brokenRelationships":     0,
			"unreadableRelationships": 0,
		},
		"readbackTables": setupSvcLiveReplayTestReadbackTables("objects"),
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("query readback without relationship evidence must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "queryReadbackMissingRelationshipEvidence:") {
		t.Fatalf("expected missing readback relationship evidence issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksQueryReadbackWithOnlyRequiredRelationships(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeSetupSvcLiveReplayArtifact(t, tmp, "objects", "create", "outputs/setup-svc-live-replay/objects/create/query-readback.json", map[string]any{
		"status": "passed",
		"readbackChecks": map[string]any{
			"requiredFields":        []string{"id"},
			"requiredRelationships": []string{"metadata-table-links"},
			"missingFields":         0,
			"missingRelationships":  0,
			"mismatchedFields":      0,
			"brokenRelationships":   0,
		},
		"readbackTables": setupSvcLiveReplayTestReadbackTables("objects"),
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("query readback with only requiredRelationships must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "queryReadbackMissingRelationshipEvidence:") {
		t.Fatalf("expected missing relationship evidence issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksUnnamedPassedQueryReadbackRelationshipCheck(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeSetupSvcLiveReplayArtifact(t, tmp, "objects", "create", "outputs/setup-svc-live-replay/objects/create/query-readback.json", map[string]any{
		"status": "passed",
		"readbackChecks": map[string]any{
			"requiredFields":          []string{"id"},
			"requiredRelationships":   []string{"metadata-table-links"},
			"relationshipChecks":      []map[string]any{{"status": "passed"}},
			"missingFields":           0,
			"missingRelationships":    0,
			"mismatchedFields":        0,
			"brokenRelationships":     0,
			"unreadableRelationships": 0,
		},
		"readbackTables": setupSvcLiveReplayTestReadbackTables("objects"),
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("unnamed passed relationship check must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "queryReadbackMissingRelationshipEvidence:") {
		t.Fatalf("expected missing relationship evidence issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksQueryReadbackRelationshipWithoutField(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeSetupSvcLiveReplayArtifact(t, tmp, "objects", "create", "outputs/setup-svc-live-replay/objects/create/query-readback.json", map[string]any{
		"status": "passed",
		"readbackChecks": map[string]any{
			"requiredFields":          []string{"id"},
			"requiredRelationships":   []string{"metadata-table-links"},
			"relationshipChecks":      []map[string]any{{"sourceTable": "tp_sys_object", "targetTable": "tp_sys_schemetable"}},
			"missingFields":           0,
			"missingRelationships":    0,
			"mismatchedFields":        0,
			"brokenRelationships":     0,
			"unreadableRelationships": 0,
		},
		"readbackTables": setupSvcLiveReplayTestReadbackTables("objects"),
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("relationship evidence without field must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "queryReadbackMissingRelationshipEvidence:") {
		t.Fatalf("expected missing relationship evidence issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksFailedQueryReadbackRelationshipCheck(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeSetupSvcLiveReplayArtifact(t, tmp, "objects", "create", "outputs/setup-svc-live-replay/objects/create/query-readback.json", map[string]any{
		"status": "passed",
		"readbackChecks": map[string]any{
			"requiredFields":          []string{"id"},
			"requiredRelationships":   []string{"metadata-table-links"},
			"relationshipChecks":      []map[string]any{{"name": "metadata-table-links", "status": "failed"}},
			"missingFields":           0,
			"missingRelationships":    0,
			"mismatchedFields":        0,
			"brokenRelationships":     0,
			"unreadableRelationships": 0,
		},
		"readbackTables": setupSvcLiveReplayTestReadbackTables("objects"),
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("failed relationship check must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "queryReadbackRelationshipEvidenceNotPassed:") {
		t.Fatalf("expected failed relationship evidence issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksQueryReadbackStringCleanCounters(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeSetupSvcLiveReplayArtifact(t, tmp, "objects", "create", "outputs/setup-svc-live-replay/objects/create/query-readback.json", map[string]any{
		"status": "passed",
		"readbackChecks": map[string]any{
			"requiredFields":          []string{"id"},
			"missingFields":           "0",
			"missingRelationships":    false,
			"mismatchedFields":        0,
			"brokenRelationships":     0,
			"unreadableRelationships": 0,
		},
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("query readback string/bool counters must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "queryReadbackStructureIncomplete:") {
		t.Fatalf("expected invalid readback clean counter issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksQueryReadbackWithoutRequiredTableCoverage(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeSetupSvcLiveReplayArtifact(t, tmp, "objects", "create", "outputs/setup-svc-live-replay/objects/create/query-readback.json", map[string]any{
		"status": "passed",
		"readbackChecks": map[string]any{
			"requiredFields":          []string{"id"},
			"missingFields":           0,
			"missingRelationships":    0,
			"mismatchedFields":        0,
			"brokenRelationships":     0,
			"unreadableRelationships": 0,
		},
		"readbackTables": []any{},
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("query readback without required table coverage must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "queryReadbackMissingTableCoverage:") {
		t.Fatalf("expected missing readback table coverage issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksQueryReadbackWithOnlyRequiredTablesList(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeSetupSvcLiveReplayArtifact(t, tmp, "objects", "create", "outputs/setup-svc-live-replay/objects/create/query-readback.json", map[string]any{
		"status":         "passed",
		"requiredTables": []string{"tp_sys_object", "tp_sys_datatablestate", "tp_sys_schemetable"},
		"readbackTables": []string{"tp_sys_object", "tp_sys_datatablestate", "tp_sys_schemetable"},
		"readbackChecks": map[string]any{
			"requiredFields":          []string{"id"},
			"missingFields":           0,
			"missingRelationships":    0,
			"mismatchedFields":        0,
			"brokenRelationships":     0,
			"unreadableRelationships": 0,
		},
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("query readback with only requiredTables list must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "queryReadbackMissingTableCoverage:") {
		t.Fatalf("expected missing readback table coverage issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksQueryReadbackWithoutFieldEvidence(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeSetupSvcLiveReplayArtifact(t, tmp, "objects", "create", "outputs/setup-svc-live-replay/objects/create/query-readback.json", map[string]any{
		"status": "passed",
		"readbackChecks": map[string]any{
			"requiredFields":          []string{"id"},
			"missingFields":           0,
			"missingRelationships":    0,
			"mismatchedFields":        0,
			"brokenRelationships":     0,
			"unreadableRelationships": 0,
		},
		"readbackTables": []map[string]any{
			{"table": "tp_sys_object", "rowCount": 1},
			{"table": "tp_sys_datatablestate", "rowCount": 1},
			{"table": "tp_sys_schemetable", "rowCount": 1},
			{"table": "tp_sys_multi_lang", "rowCount": 1},
			{"table": "tp_sys_profile_infoset", "rowCount": 1},
			{"table": "tp_sys_profile_field", "rowCount": 1},
			{"table": "tp_sys_layout", "rowCount": 1},
			{"table": "tp_sys_view", "rowCount": 1},
		},
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("query readback without field evidence must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "queryReadbackMissingFieldEvidence:") {
		t.Fatalf("expected missing readback field evidence issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksQueryReadbackWithoutRowEvidence(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeSetupSvcLiveReplayArtifact(t, tmp, "objects", "create", "outputs/setup-svc-live-replay/objects/create/query-readback.json", map[string]any{
		"status": "passed",
		"readbackChecks": map[string]any{
			"requiredFields":          []string{"id"},
			"missingFields":           0,
			"missingRelationships":    0,
			"mismatchedFields":        0,
			"brokenRelationships":     0,
			"unreadableRelationships": 0,
		},
		"readbackTables": []map[string]any{
			{"table": "tp_sys_object", "columns": []string{"id"}},
			{"table": "tp_sys_datatablestate", "columns": []string{"id"}},
			{"table": "tp_sys_schemetable", "columns": []string{"id"}},
			{"table": "tp_sys_multi_lang", "columns": []string{"id"}},
			{"table": "tp_sys_profile_infoset", "columns": []string{"id"}},
			{"table": "tp_sys_profile_field", "columns": []string{"id"}},
			{"table": "tp_sys_layout", "columns": []string{"id"}},
			{"table": "tp_sys_view", "columns": []string{"id"}},
		},
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("query readback without row evidence must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "queryReadbackMissingRowEvidence:") {
		t.Fatalf("expected missing readback row evidence issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksQueryReadbackWithRowCountAndColumnsOnly(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeSetupSvcLiveReplayArtifact(t, tmp, "objects", "create", "outputs/setup-svc-live-replay/objects/create/query-readback.json", map[string]any{
		"status": "passed",
		"readbackChecks": map[string]any{
			"requiredFields":          []string{"id"},
			"requiredRelationships":   []string{"metadata-table-links"},
			"relationshipChecks":      []map[string]any{{"name": "metadata-table-links", "status": "passed"}},
			"missingFields":           0,
			"missingRelationships":    0,
			"mismatchedFields":        0,
			"brokenRelationships":     0,
			"unreadableRelationships": 0,
		},
		"readbackTables": []map[string]any{
			{"table": "tp_sys_object", "rowCount": 1, "columns": []string{"id"}, "requiredFields": []string{"id"}, "requiredRelationships": []string{"metadata-table-links"}},
			{"table": "tp_sys_datatablestate", "rowCount": 1, "columns": []string{"id"}, "requiredFields": []string{"id"}, "requiredRelationships": []string{"metadata-table-links"}},
			{"table": "tp_sys_schemetable", "rowCount": 1, "columns": []string{"id"}, "requiredFields": []string{"id"}, "requiredRelationships": []string{"metadata-table-links"}},
		},
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("query readback with only rowCount and columns must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "queryReadbackMissingRowEvidence:") {
		t.Fatalf("expected missing readback row evidence issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksQueryReadbackWithEmptyTableDetails(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeSetupSvcLiveReplayArtifact(t, tmp, "objects", "create", "outputs/setup-svc-live-replay/objects/create/query-readback.json", map[string]any{
		"status": "passed",
		"readbackChecks": map[string]any{
			"requiredFields":          []string{"id"},
			"missingFields":           0,
			"missingRelationships":    0,
			"mismatchedFields":        0,
			"brokenRelationships":     0,
			"unreadableRelationships": 0,
		},
		"readbackTables": []map[string]any{
			{"table": "tp_sys_object", "rowCount": 0, "columns": []string{}},
			{"table": "tp_sys_datatablestate", "rowCount": 0, "columns": []string{}},
			{"table": "tp_sys_schemetable", "rowCount": 0, "columns": []string{}},
		},
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("query readback with empty table details must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "queryReadbackMissingTableCoverage:") {
		t.Fatalf("expected missing readback table coverage issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksQueryReadbackWithEmptyKeyedTableDetails(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeSetupSvcLiveReplayArtifact(t, tmp, "objects", "create", "outputs/setup-svc-live-replay/objects/create/query-readback.json", map[string]any{
		"status": "passed",
		"readbackChecks": map[string]any{
			"requiredFields":          []string{"id"},
			"missingFields":           0,
			"missingRelationships":    0,
			"mismatchedFields":        0,
			"brokenRelationships":     0,
			"unreadableRelationships": 0,
		},
		"readbackTables": map[string]any{
			"tp_sys_object":         map[string]any{"rowCount": 0, "columns": []any{}},
			"tp_sys_datatablestate": map[string]any{"rowCount": 0, "columns": []any{}},
			"tp_sys_schemetable":    map[string]any{"rowCount": 0, "columns": []any{}},
		},
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("query readback with empty keyed table details must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "queryReadbackMissingTableCoverage:") {
		t.Fatalf("expected missing readback table coverage issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksQueryReadbackWithScalarKeyedTableDetails(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeSetupSvcLiveReplayArtifact(t, tmp, "objects", "create", "outputs/setup-svc-live-replay/objects/create/query-readback.json", map[string]any{
		"status": "passed",
		"readbackChecks": map[string]any{
			"requiredFields":          []string{"id"},
			"requiredRelationships":   []string{"metadata-table-links"},
			"relationshipChecks":      []map[string]any{{"name": "metadata-table-links", "status": "passed"}},
			"missingFields":           0,
			"missingRelationships":    0,
			"mismatchedFields":        0,
			"brokenRelationships":     0,
			"unreadableRelationships": 0,
		},
		"readbackTables": map[string]any{
			"tp_sys_object":         "present",
			"tp_sys_datatablestate": "present",
			"tp_sys_schemetable":    "present",
		},
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("query readback with scalar keyed table details must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "queryReadbackMissingTableCoverage:") {
		t.Fatalf("expected missing readback table coverage issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksSetupSvcSnapshotWithoutRequiredTables(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	payload, err := json.Marshal(map[string]any{
		"status":              "passed",
		"project":             tmp,
		"contractVersion":     setupSvcLiveReplayContractVersion,
		"contractFingerprint": setupSvcLiveReplayExpectedContractFingerprint(),
		"domain":              "objects",
		"operation":           "create",
		"artifactType":        "setup-svc",
	})
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(tmp, "outputs/setup-svc-live-replay/objects/create/setup-svc.json"), string(payload))

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("setup-svc snapshot without table evidence must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "setupSvcSnapshotMissingTableEvidence:") {
		t.Fatalf("expected missing setup-svc table snapshot issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksSnapshotWithoutRuntimeEffectEvidence(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeSetupSvcLiveReplayArtifact(t, tmp, "objects", "create", "outputs/setup-svc-live-replay/objects/create/setup-svc.json", map[string]any{
		"status":              "passed",
		"runtimeEffects":      setupSvcLiveReplayRuntimeEffects("objects"),
		"runtimeEffectChecks": []map[string]any{},
		"tableSnapshots":      setupSvcLiveReplayTestTableSnapshots("objects"),
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("snapshot without runtime effect evidence must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "runtimeEffectsMissingEvidence=datatable-prefix-allocation") {
		t.Fatalf("expected runtime effect evidence blocker, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksSnapshotWithOnlyRequiredTables(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	payload, err := json.Marshal(map[string]any{
		"status":              "passed",
		"project":             tmp,
		"contractVersion":     setupSvcLiveReplayContractVersion,
		"contractFingerprint": setupSvcLiveReplayExpectedContractFingerprint(),
		"domain":              "objects",
		"operation":           "create",
		"artifactType":        "setup-svc",
		"requiredTables":      []string{"tp_sys_object", "tp_sys_schemafield", "tp_sys_page_layout"},
	})
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(tmp, "outputs/setup-svc-live-replay/objects/create/setup-svc.json"), string(payload))

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("snapshot with only requiredTables must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "setupSvcSnapshotMissingTableEvidence:") {
		t.Fatalf("expected missing table snapshot evidence issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksSnapshotWithEmptyTableDetails(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeSetupSvcLiveReplayArtifact(t, tmp, "objects", "create", "outputs/setup-svc-live-replay/objects/create/setup-svc.json", map[string]any{
		"status": "passed",
		"tableSnapshots": []map[string]any{
			{"table": "tp_sys_object", "rowCount": 0, "rows": []any{}},
			{"table": "tp_sys_datatablestate", "rowCount": 0, "rows": []any{}},
			{"table": "tp_sys_schemetable", "rowCount": 0, "rows": []any{}},
		},
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("setup-svc snapshot with empty table details must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "setupSvcSnapshotMissingTableEvidence:") {
		t.Fatalf("expected missing table snapshot evidence issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksSnapshotWithEmptyKeyedTableDetails(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeSetupSvcLiveReplayArtifact(t, tmp, "objects", "create", "outputs/setup-svc-live-replay/objects/create/setup-svc.json", map[string]any{
		"status": "passed",
		"tableSnapshots": map[string]any{
			"tp_sys_object":         map[string]any{"rowCount": 0, "rows": []any{}},
			"tp_sys_datatablestate": map[string]any{"rowCount": 0, "rows": []any{}},
			"tp_sys_schemetable":    map[string]any{"rowCount": 0, "rows": []any{}},
		},
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("setup-svc snapshot with empty keyed table details must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "setupSvcSnapshotMissingTableEvidence:") {
		t.Fatalf("expected missing table snapshot evidence issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksSnapshotWithScalarKeyedTableDetails(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeSetupSvcLiveReplayArtifact(t, tmp, "objects", "create", "outputs/setup-svc-live-replay/objects/create/setup-svc.json", map[string]any{
		"status": "passed",
		"tableSnapshots": map[string]any{
			"tp_sys_object":         "present",
			"tp_sys_datatablestate": "present",
			"tp_sys_schemetable":    "present",
		},
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("setup-svc snapshot with scalar keyed table details must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "setupSvcSnapshotMissingTableEvidence:") {
		t.Fatalf("expected missing table snapshot evidence issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksSnapshotWithOnlyRowCount(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeSetupSvcLiveReplayArtifact(t, tmp, "objects", "create", "outputs/setup-svc-live-replay/objects/create/setup-svc.json", map[string]any{
		"status": "passed",
		"tableSnapshots": []map[string]any{
			{"table": "tp_sys_object", "rowCount": 1},
			{"table": "tp_sys_datatablestate", "rowCount": 1},
			{"table": "tp_sys_schemetable", "rowCount": 1},
		},
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("setup-svc snapshot with only rowCount must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "setupSvcSnapshotMissingTableEvidence:") {
		t.Fatalf("expected missing row/column snapshot evidence issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksSnapshotWithOnlyColumns(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeSetupSvcLiveReplayArtifact(t, tmp, "objects", "create", "outputs/setup-svc-live-replay/objects/create/setup-svc.json", map[string]any{
		"status": "passed",
		"tableSnapshots": []map[string]any{
			{"table": "tp_sys_object", "columns": []string{"id"}},
			{"table": "tp_sys_datatablestate", "columns": []string{"id"}},
			{"table": "tp_sys_schemetable", "columns": []string{"id"}},
		},
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("setup-svc snapshot with only columns must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "setupSvcSnapshotMissingTableEvidence:") {
		t.Fatalf("expected missing row/column snapshot evidence issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksSetupSvcMetadataServiceSnapshotTableMismatch(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	snapshots := setupSvcLiveReplayTestTableSnapshots("objects")
	snapshots = append(snapshots, map[string]any{
		"table":       "tp_sys_only_setup_svc",
		"rowCount":    1,
		"columns":     []string{"id"},
		"primaryKeys": []string{"id"},
		"rows":        []map[string]any{{"id": "tp_sys_only_setup_svc-replay-id"}},
	})
	writeSetupSvcLiveReplayArtifact(t, tmp, "objects", "create", "outputs/setup-svc-live-replay/objects/create/setup-svc.json", map[string]any{
		"status":         "passed",
		"tableSnapshots": snapshots,
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("snapshot table mismatch must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "snapshotTableSetMismatch:metadataServiceMissingTables=tp_sys_only_setup_svc") {
		t.Fatalf("expected metadata-service missing table mismatch issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksMetadataServiceSetupSvcSnapshotTableMismatch(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	snapshots := setupSvcLiveReplayTestTableSnapshots("objects")
	snapshots = append(snapshots, map[string]any{
		"table":       "tp_sys_only_metadata_service",
		"rowCount":    1,
		"columns":     []string{"id"},
		"primaryKeys": []string{"id"},
		"rows":        []map[string]any{{"id": "tp_sys_only_metadata_service-replay-id"}},
	})
	writeSetupSvcLiveReplayArtifact(t, tmp, "objects", "create", "outputs/setup-svc-live-replay/objects/create/metadata-service.json", map[string]any{
		"status":         "passed",
		"tableSnapshots": snapshots,
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("snapshot table mismatch must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "snapshotTableSetMismatch:setupSvcMissingTables=tp_sys_only_metadata_service") {
		t.Fatalf("expected setup-svc missing table mismatch issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksMetadataServiceWithoutDatasourceProof(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeSetupSvcLiveReplayArtifact(t, tmp, "objects", "create", "outputs/setup-svc-live-replay/objects/create/metadata-service.json", map[string]any{
		"status":                    "passed",
		"metadataServiceDatasource": nil,
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("metadata-service evidence without datasource proof must block, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "metadataServiceDatasourceMissingEvidence:") {
		t.Fatalf("expected datasource proof issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksCleanupWithoutResidualProof(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	payload, err := json.Marshal(map[string]any{
		"status":              "passed",
		"project":             tmp,
		"contractVersion":     setupSvcLiveReplayContractVersion,
		"contractFingerprint": setupSvcLiveReplayExpectedContractFingerprint(),
		"domain":              "objects",
		"operation":           "create",
		"artifactType":        "cleanup",
	})
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(tmp, "outputs/setup-svc-live-replay/objects/create/cleanup.json"), string(payload))

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("cleanup without residual proof must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "cleanupMissingResidualEvidence:") {
		t.Fatalf("expected missing cleanup residual evidence issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksCleanupWithoutResidualCounters(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeSetupSvcLiveReplayArtifact(t, tmp, "objects", "create", "outputs/setup-svc-live-replay/objects/create/cleanup.json", map[string]any{
		"status": "passed",
		"cleanupChecks": map[string]any{
			"deletedRows": 1,
		},
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("cleanup without residual counters must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "cleanupMissingResidualCounters:") {
		t.Fatalf("expected missing cleanup residual counters issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksCleanupWithResidualRows(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeSetupSvcLiveReplayArtifact(t, tmp, "objects", "create", "outputs/setup-svc-live-replay/objects/create/cleanup.json", map[string]any{
		"status": "passed",
		"cleanupChecks": map[string]any{
			"deletedRows":   1,
			"remainingRows": 1,
			"residualRows":  0,
			"orphanRows":    0,
			"errors":        0,
			"failures":      0,
		},
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("cleanup with residual rows must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "cleanupResidualMetadataRemaining:") {
		t.Fatalf("expected cleanup residual metadata issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksCleanupStringCleanCounters(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeSetupSvcLiveReplayArtifact(t, tmp, "objects", "create", "outputs/setup-svc-live-replay/objects/create/cleanup.json", map[string]any{
		"status": "passed",
		"cleanupChecks": map[string]any{
			"deletedRows":   1,
			"remainingRows": "0",
			"residualRows":  false,
			"orphanRows":    0,
			"errors":        0,
			"failures":      0,
		},
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("cleanup string/bool counters must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "cleanupResidualMetadataRemaining:") {
		t.Fatalf("expected invalid cleanup residual counter issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksMismatchedArtifactIdentity(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeSetupSvcLiveReplayArtifact(t, tmp, "fields", "create", "outputs/setup-svc-live-replay/objects/create/setup-svc.json", map[string]any{"status": "passed"})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("mismatched artifact identity must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "evidenceFileDomainMismatch:") {
		t.Fatalf("expected mismatched domain artifact issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksMismatchedArtifactContractFingerprint(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeSetupSvcLiveReplayArtifact(t, tmp, "objects", "create", "outputs/setup-svc-live-replay/objects/create/setup-svc.json", map[string]any{
		"status":              "passed",
		"contractFingerprint": "sha256:wrong",
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("mismatched artifact contract fingerprint must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "evidenceFileContractFingerprintMismatch:") {
		t.Fatalf("expected mismatched artifact contract fingerprint issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksMismatchedArtifactProject(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeSetupSvcLiveReplayArtifact(t, tmp, "objects", "create", "outputs/setup-svc-live-replay/objects/create/setup-svc.json", map[string]any{
		"status":  "passed",
		"project": filepath.Join(tmp, "other-project"),
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("mismatched artifact project must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "evidenceFileProjectMismatch:") {
		t.Fatalf("expected mismatched artifact project issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceAcceptsProjectAbsoluteEvidencePaths(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	rewriteSetupSvcLiveReplayManifestOperation(t, manifestPath, "objects", "create", func(operation map[string]any) {
		files := []any{}
		for _, file := range operation["evidenceFiles"].([]any) {
			files = append(files, filepath.Join(tmp, file.(string)))
		}
		operation["evidenceFiles"] = files
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "passed" {
		t.Fatalf("project-absolute canonical evidence files should pass, got %#v", result)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksLooseEvidenceFilePathContract(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	rewriteSetupSvcLiveReplayManifestOperation(t, manifestPath, "objects", "create", func(operation map[string]any) {
		files := operation["evidenceFiles"].([]any)
		files[0] = "setup-svc.json"
		operation["evidenceFiles"] = files
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("basename-only evidence file path must be blocked, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["missingEvidence"].([]any), "evidenceFiles:outputs/setup-svc-live-replay/objects/create/setup-svc.json") {
		t.Fatalf("expected missing canonical evidence path, got %#v", failed)
	}
	if !containsStringItem(failed["failedEvidence"].([]any), "unexpectedEvidenceFile:setup-svc.json") {
		t.Fatalf("expected unexpected loose evidence path, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksDuplicateEvidenceFiles(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	rewriteSetupSvcLiveReplayManifestOperation(t, manifestPath, "objects", "create", func(operation map[string]any) {
		files := operation["evidenceFiles"].([]any)
		operation["evidenceFiles"] = append(files, files[0])
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("duplicate evidence file path must be blocked, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if !containsStringItem(failed["failedEvidence"].([]any), "duplicateEvidenceFile:outputs/setup-svc-live-replay/objects/create/setup-svc.json") {
		t.Fatalf("expected duplicate evidence file issue, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksMissingContractFingerprint(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	rewriteSetupSvcLiveReplayManifest(t, manifestPath, func(manifest map[string]any) {
		delete(manifest, "contractFingerprint")
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("missing contract fingerprint must block evidence, got %#v", result)
	}
	if !containsStringItem(result["blockingIssues"].([]any), "manifest: missing contractFingerprint") {
		t.Fatalf("expected missing contract fingerprint issue, got %#v", result["blockingIssues"])
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksMissingManifestMode(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	rewriteSetupSvcLiveReplayManifest(t, manifestPath, func(manifest map[string]any) {
		delete(manifest, "mode")
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("missing manifest mode must block evidence, got %#v", result)
	}
	if !containsStringItem(result["blockingIssues"].([]any), "manifest: missing mode") {
		t.Fatalf("expected missing mode issue, got %#v", result["blockingIssues"])
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksUnexpectedManifestMode(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	rewriteSetupSvcLiveReplayManifest(t, manifestPath, func(manifest map[string]any) {
		manifest["mode"] = "setup-svc-live-replay-packet"
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("unexpected manifest mode must block evidence, got %#v", result)
	}
	if !containsStringItem(result["blockingIssues"].([]any), "manifest: unexpected mode setup-svc-live-replay-packet") {
		t.Fatalf("expected unexpected mode issue, got %#v", result["blockingIssues"])
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksMissingManifestStatus(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	rewriteSetupSvcLiveReplayManifest(t, manifestPath, func(manifest map[string]any) {
		delete(manifest, "status")
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("missing manifest status must block evidence, got %#v", result)
	}
	if !containsStringItem(result["blockingIssues"].([]any), "manifest: missing status") {
		t.Fatalf("expected missing status issue, got %#v", result["blockingIssues"])
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksPendingManifestStatus(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	rewriteSetupSvcLiveReplayManifest(t, manifestPath, func(manifest map[string]any) {
		manifest["status"] = "pending"
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("pending manifest status must block evidence, got %#v", result)
	}
	if !containsStringItem(result["blockingIssues"].([]any), "manifest: status not passed pending") {
		t.Fatalf("expected pending status issue, got %#v", result["blockingIssues"])
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksCleanManifestStatus(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	rewriteSetupSvcLiveReplayManifest(t, manifestPath, func(manifest map[string]any) {
		manifest["status"] = "clean"
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("clean manifest status must block evidence, got %#v", result)
	}
	if !containsStringItem(result["blockingIssues"].([]any), "manifest: status not passed clean") {
		t.Fatalf("expected clean status issue, got %#v", result["blockingIssues"])
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksMismatchedManifestProject(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	rewriteSetupSvcLiveReplayManifest(t, manifestPath, func(manifest map[string]any) {
		manifest["project"] = filepath.Join(tmp, "other-project")
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("mismatched manifest project must block evidence, got %#v", result)
	}
	if !containsStringItem(result["blockingIssues"].([]any), "manifest: project mismatch") {
		t.Fatalf("expected manifest project mismatch issue, got %#v", result["blockingIssues"])
	}
}

func TestSetupSvcLiveReplayGapsReportsMissingManifest(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-gaps"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "missing_manifest" || result["readOnly"] != true {
		t.Fatalf("missing manifest should produce read-only gap report, got %#v", result)
	}
	totals := result["totals"].(map[string]any)
	if int(totals["missingOperations"].(float64)) != setupSvcLiveReplayOperationCount() {
		t.Fatalf("expected every operation missing, got %#v", totals)
	}
	plan := result["collectionPlan"].(map[string]any)
	if plan["status"] != "collect_missing_artifacts" ||
		int(plan["totalArtifacts"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount() ||
		int(plan["missingArtifacts"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount() ||
		int(plan["queryReadbackArtifacts"].(float64)) != setupSvcLiveReplayOperationCount() {
		t.Fatalf("expected missing manifest collection plan to cover every artifact and query readback file, got %#v", plan)
	}
	if !containsStringItem(result["blockingIssues"].([]any), "manifest: missing live replay evidence "+filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")) {
		t.Fatalf("expected missing manifest blocker, got %#v", result["blockingIssues"])
	}
}

func TestSetupSvcLiveReplayGapsReportsPendingWorkspaceArtifacts(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-workspace", "--execute", "--approval", setupSvcParityEvidenceWorkspaceApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-gaps"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "pending_evidence" {
		t.Fatalf("placeholder workspace should be pending evidence, got %#v", result)
	}
	totals := result["totals"].(map[string]any)
	if int(totals["pendingOperations"].(float64)) != setupSvcLiveReplayOperationCount() {
		t.Fatalf("expected every operation pending, got %#v", totals)
	}
	plan := result["collectionPlan"].(map[string]any)
	if plan["status"] != "replace_pending_artifacts" ||
		int(plan["pendingArtifacts"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount() ||
		int(plan["queryReadbackArtifacts"].(float64)) != setupSvcLiveReplayOperationCount() {
		t.Fatalf("expected pending workspace collection plan to cover every artifact and query readback file, got %#v", plan)
	}
	runbook := plan["runbook"].([]any)
	if len(runbook) < 5 {
		t.Fatalf("expected collection plan runbook to guide evidence replacement through audit, got %#v", runbook)
	}
	collectStep := runbook[0].(map[string]any)
	if collectStep["step"] != "collect_or_replace_evidence" ||
		collectStep["status"] != "replace_pending_artifacts" ||
		!containsStringFragment(collectStep["commands"].([]any), "--evidence-section tableSnapshots") ||
		!containsStringFragment(collectStep["commands"].([]any), "--offset 0 --limit 25") {
		t.Fatalf("expected runbook to start with bounded evidence section batch commands, got %#v", collectStep)
	}
	var syncStep, bundleStep map[string]any
	for _, item := range runbook {
		step := item.(map[string]any)
		switch step["step"] {
		case "sync_manifest_status":
			syncStep = step
		case "write_evidence_bundle":
			bundleStep = step
		}
	}
	if syncStep == nil || !containsStringFragment(syncStep["commands"].([]any), "setup-svc-live-replay-manifest-sync") ||
		!containsStringFragment(syncStep["commands"].([]any), setupSvcParityManifestSyncApproval) {
		t.Fatalf("expected runbook to include approval-gated manifest sync command, got %#v", runbook)
	}
	if bundleStep == nil || !containsStringFragment(bundleStep["commands"].([]any), "setup-svc-live-replay-evidence-bundle") ||
		!containsStringFragment(bundleStep["commands"].([]any), setupSvcParityEvidenceBundleApproval) {
		t.Fatalf("expected runbook to include approval-gated evidence bundle command, got %#v", runbook)
	}
	nextCommands := result["nextCommands"].(map[string]any)
	if !strings.Contains(nextCommands["syncManifest"].(string), "setup-svc-live-replay-manifest-sync") ||
		!strings.Contains(nextCommands["writeBundle"].(string), "setup-svc-live-replay-evidence-bundle") {
		t.Fatalf("expected gap nextCommands to expose syncManifest and writeBundle, got %#v", nextCommands)
	}
	if int(plan["nextArtifactLimit"].(float64)) != 25 ||
		int(plan["omittedNextArtifacts"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount()-25 {
		t.Fatalf("expected bounded next artifact list to report omitted artifacts, got %#v", plan)
	}
	artifactTypes := plan["artifactTypes"].([]any)
	var queryType map[string]any
	for _, item := range artifactTypes {
		candidate := item.(map[string]any)
		if candidate["artifactType"] == "query-readback" {
			queryType = candidate
			break
		}
	}
	if queryType == nil || int(queryType["pending"].(float64)) != setupSvcLiveReplayOperationCount() {
		t.Fatalf("expected query-readback pending artifact type summary, got %#v", artifactTypes)
	}
	evidenceSections := plan["evidenceSections"].([]any)
	if !evidenceSectionSummaryHasCount(evidenceSections, "setup-svc", "status", setupSvcLiveReplayOperationCount(), setupSvcLiveReplayOperationCount(), 0) ||
		!evidenceSectionSummaryHasCount(evidenceSections, "setup-svc", "tableSnapshots", setupSvcLiveReplayOperationCount(), 0, setupSvcLiveReplayOperationCount()) ||
		!evidenceSectionSummaryHasCount(evidenceSections, "query-readback", "status", setupSvcLiveReplayOperationCount(), setupSvcLiveReplayOperationCount(), 0) ||
		!evidenceSectionSummaryHasCount(evidenceSections, "query-readback", "readbackTables", setupSvcLiveReplayOperationCount(), 0, setupSvcLiveReplayOperationCount()) {
		t.Fatalf("expected collection plan to summarize present identity sections and missing proof sections, got %#v", evidenceSections)
	}
	readbackTablesSummary := evidenceSectionSummary(evidenceSections, "query-readback", "readbackTables")
	if readbackTablesSummary["nextAction"] != "collect_missing_evidence_section" ||
		!strings.Contains(readbackTablesSummary["queueCommand"].(string), "--artifact-type query-readback") ||
		!strings.Contains(readbackTablesSummary["queueCommand"].(string), "--evidence-section readbackTables") ||
		!strings.Contains(readbackTablesSummary["queueCommand"].(string), "--section-status missing --offset 0 --limit 25") {
		t.Fatalf("expected missing readbackTables summary to expose a section-specific queue command, got %#v", readbackTablesSummary)
	}
	statusSummary := evidenceSectionSummary(evidenceSections, "query-readback", "status")
	if _, ok := statusSummary["queueCommand"]; ok {
		t.Fatalf("present identity section should not expose a missing-section queue command, got %#v", statusSummary)
	}
	missingSectionQueues := plan["missingSectionQueues"].([]any)
	readbackTablesQueue := evidenceSectionQueue(missingSectionQueues, "query-readback", "readbackTables")
	if readbackTablesQueue == nil ||
		int(readbackTablesQueue["missing"].(float64)) != setupSvcLiveReplayOperationCount() ||
		readbackTablesQueue["requiredShapeKey"] != "requiredReadbackShape" ||
		readbackTablesQueue["manifestStatusField"] != "queryEvidenceStatus" ||
		int(readbackTablesQueue["pageSize"].(float64)) != 25 ||
		int(readbackTablesQueue["batchCount"].(float64)) != (setupSvcLiveReplayOperationCount()+24)/25 ||
		len(readbackTablesQueue["batchCommands"].([]any)) != (setupSvcLiveReplayOperationCount()+24)/25 ||
		!containsStringFragment(readbackTablesQueue["batchCommands"].([]any), "--offset 75 --limit 25") ||
		!strings.Contains(readbackTablesQueue["queueCommand"].(string), "--artifact-type query-readback") ||
		!strings.Contains(readbackTablesQueue["queueCommand"].(string), "--evidence-section readbackTables") ||
		!strings.Contains(readbackTablesQueue["queueCommand"].(string), "--section-status missing --offset 0 --limit 25") {
		t.Fatalf("expected missingSectionQueues to expose the query-readback readbackTables work queue, got %#v", missingSectionQueues)
	}
	if _, ok := readbackTablesQueue["omittedBatchCommands"]; ok {
		t.Fatalf("expected small missingSectionQueues batch list to include every batch command, got %#v", readbackTablesQueue)
	}
	if evidenceSectionQueue(missingSectionQueues, "query-readback", "status") != nil {
		t.Fatalf("present identity section should not be included in missingSectionQueues, got %#v", missingSectionQueues)
	}
	if firstQueue := missingSectionQueues[0].(map[string]any); int(firstQueue["missing"].(float64)) < int(readbackTablesQueue["missing"].(float64)) {
		t.Fatalf("expected missingSectionQueues to be prioritized by missing count, got %#v", missingSectionQueues)
	}
	nextArtifacts := plan["nextArtifacts"].([]any)
	if len(nextArtifacts) == 0 {
		t.Fatalf("expected bounded next artifact actions, got %#v", plan)
	}
	firstAction := nextArtifacts[0].(map[string]any)
	if firstAction["requiredShapeKey"] != "requiredSnapshotShape" || firstAction["status"] != "pending" ||
		firstAction["nextAction"] != "replace_pending_setup_svc_artifact" ||
		firstAction["manifestStatusField"] != "setupSvcEvidenceStatus" {
		t.Fatalf("expected next artifact to name required shape and pending status, got %#v", firstAction)
	}
	if !containsStringItem(firstAction["requiredTables"].([]any), "tp_sys_object") ||
		!containsStringItem(firstAction["runtimeEffects"].([]any), "datatable-prefix-allocation") {
		t.Fatalf("expected setup-svc collection action to include required tables and runtime effects, got %#v", firstAction)
	}
	if !containsStringFragment(firstAction["replacementChecklist"].([]any), "tableSnapshots") ||
		!containsStringFragment(firstAction["replacementChecklist"].([]any), "runtimeEffectChecks") {
		t.Fatalf("expected setup-svc collection action to include snapshot/runtime replacement checklist, got %#v", firstAction)
	}
	if !containsStringItem(firstAction["requiredEvidenceSections"].([]any), "tableSnapshots") ||
		!containsStringItem(firstAction["requiredEvidenceSections"].([]any), "runtimeEffectChecks") {
		t.Fatalf("expected setup-svc collection action to include machine-readable evidence sections, got %#v", firstAction)
	}
	if !evidenceSectionHasStatus(firstAction["evidenceSectionStatuses"].([]any), "status", "present") ||
		!evidenceSectionHasStatus(firstAction["evidenceSectionStatuses"].([]any), "project", "present") ||
		!evidenceSectionHasStatus(firstAction["evidenceSectionStatuses"].([]any), "tableSnapshots", "missing") ||
		!evidenceSectionHasStatus(firstAction["evidenceSectionStatuses"].([]any), "runtimeEffectChecks", "missing") {
		t.Fatalf("expected pending setup-svc action to report present identity sections and missing snapshot sections, got %#v", firstAction)
	}
	var queryAction map[string]any
	for _, item := range nextArtifacts {
		candidate := item.(map[string]any)
		if candidate["artifactType"] == "query-readback" {
			queryAction = candidate
			break
		}
	}
	if queryAction == nil {
		t.Fatalf("expected bounded next artifacts to include query-readback, got %#v", nextArtifacts)
	}
	if queryAction["requiredShapeKey"] != "requiredReadbackShape" ||
		queryAction["nextAction"] != "replace_pending_query_readback_artifact" ||
		queryAction["manifestStatusField"] != "queryEvidenceStatus" ||
		!containsStringItem(queryAction["requiredTables"].([]any), "tp_sys_object") ||
		!containsStringItem(queryAction["queryReadbackExpectations"].([]any), "object-identity-prefix-datatable-readback") {
		t.Fatalf("expected query-readback collection action to expose readback contract, got %#v", queryAction)
	}
	if !containsStringFragment(queryAction["replacementChecklist"].([]any), "readback table coverage") ||
		!containsStringFragment(queryAction["replacementChecklist"].([]any), "relationshipChecks") ||
		!containsStringFragment(queryAction["replacementChecklist"].([]any), "numeric zero") {
		t.Fatalf("expected query-readback collection action to include readback replacement checklist, got %#v", queryAction)
	}
	if !containsStringItem(queryAction["requiredEvidenceSections"].([]any), "readbackTables") ||
		!containsStringItem(queryAction["requiredEvidenceSections"].([]any), "relationshipChecks") ||
		!containsStringItem(queryAction["requiredEvidenceSections"].([]any), "readbackExpectationChecks") ||
		!containsStringItem(queryAction["requiredEvidenceSections"].([]any), "cleanCounters") {
		t.Fatalf("expected query-readback collection action to expose required evidence sections, got %#v", queryAction)
	}
	if !evidenceSectionHasStatus(queryAction["evidenceSectionStatuses"].([]any), "status", "present") ||
		!evidenceSectionHasStatus(queryAction["evidenceSectionStatuses"].([]any), "readbackTables", "missing") ||
		!evidenceSectionHasStatus(queryAction["evidenceSectionStatuses"].([]any), "relationshipChecks", "missing") ||
		!evidenceSectionHasStatus(queryAction["evidenceSectionStatuses"].([]any), "cleanCounters", "missing") {
		t.Fatalf("expected pending query-readback action to report present status and missing readback evidence sections, got %#v", queryAction)
	}
	firstDomain := result["domains"].([]any)[0].(map[string]any)
	firstOperation := firstDomain["operations"].([]any)[0].(map[string]any)
	if firstOperation["status"] != "pending_evidence" || firstOperation["nextAction"] != "collect_setup_svc_snapshot" {
		t.Fatalf("expected first operation to request setup-svc evidence, got %#v", firstOperation)
	}
}

func TestSetupSvcLiveReplayWorklistExpandsMissingSectionBatches(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-workspace", "--execute", "--approval", setupSvcParityEvidenceWorkspaceApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-worklist"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["mode"] != "setup-svc-live-replay-worklist" || result["readOnly"] != true || result["status"] != "pending_evidence" {
		t.Fatalf("expected read-only pending worklist, got %#v", result)
	}
	if result["sourceRoot"] != "captures" || result["captureRoot"] != filepath.Join(tmp, "captures") {
		t.Fatalf("expected worklist to expose mirrored capture root, got sourceRoot=%#v captureRoot=%#v", result["sourceRoot"], result["captureRoot"])
	}
	totals := result["totals"].(map[string]any)
	if int(totals["queues"].(float64)) < 10 ||
		int(totals["sourceFilesPresent"].(float64)) != 0 ||
		int(totals["sourceFilesMissing"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount() ||
		int(totals["sourceFilesComplete"].(float64)) != 0 ||
		int(totals["sourceFilesIncomplete"].(float64)) != 0 ||
		int(totals["queryReadbackQueues"].(float64)) != len(setupSvcLiveReplayRequiredEvidenceSections("query-readback"))-7 ||
		int(totals["queryReadbackArtifacts"].(float64)) != setupSvcLiveReplayOperationCount()*(len(setupSvcLiveReplayRequiredEvidenceSections("query-readback"))-7) {
		t.Fatalf("expected worklist totals to include expanded query-readback batches, got %#v", totals)
	}
	if int(result["sourceFilesPresent"].(float64)) != int(totals["sourceFilesPresent"].(float64)) ||
		int(result["sourceFiles"].(float64)) != int(totals["sourceFilesPresent"].(float64))+int(totals["sourceFilesMissing"].(float64)) ||
		int(result["targetFiles"].(float64)) != int(totals["uniqueArtifactFiles"].(float64)) ||
		int(result["sourceFilesMissing"].(float64)) != int(totals["sourceFilesMissing"].(float64)) ||
		int(result["sourceFilesComplete"].(float64)) != int(totals["sourceFilesComplete"].(float64)) ||
		int(result["sourceFilesIncomplete"].(float64)) != int(totals["sourceFilesIncomplete"].(float64)) {
		t.Fatalf("expected worklist top-level source counters to mirror totals for reconciliation, got result=%#v totals=%#v", result, totals)
	}
	for _, mirror := range []struct {
		top   string
		total string
	}{
		{top: "queuesCount", total: "queues"},
		{top: "batches", total: "batches"},
		{top: "artifacts", total: "artifacts"},
		{top: "uniqueArtifactFiles", total: "uniqueArtifactFiles"},
		{top: "duplicateArtifactRecords", total: "duplicateArtifactRecords"},
		{top: "missingSections", total: "missingSections"},
		{top: "queryReadbackQueues", total: "queryReadbackQueues"},
		{top: "queryReadbackArtifacts", total: "queryReadbackArtifacts"},
		{top: "omittedBatches", total: "omittedBatches"},
	} {
		if result[mirror.top] != totals[mirror.total] {
			t.Fatalf("expected worklist top-level %s to mirror totals.%s, got top=%#v totals=%#v", mirror.top, mirror.total, result[mirror.top], totals[mirror.total])
		}
	}
	queues := result["queues"].([]any)
	batchSaveCommands := result["batchSaveCommands"].([]any)
	operatorPacket := result["operatorPacket"].(map[string]any)
	if int(operatorPacket["sourceFiles"].(float64)) != int(result["sourceFiles"].(float64)) ||
		int(operatorPacket["targetFiles"].(float64)) != int(result["targetFiles"].(float64)) ||
		int(operatorPacket["uniqueArtifactFiles"].(float64)) != int(totals["uniqueArtifactFiles"].(float64)) {
		t.Fatalf("expected worklist operatorPacket source/target counters to mirror top-level totals, got operatorPacket=%#v result=%#v totals=%#v", operatorPacket, result, totals)
	}
	operatorBatchSaveCommands := operatorPacket["batchSaveCommands"].([]any)
	if len(batchSaveCommands) != int(totals["batches"].(float64)) ||
		len(operatorBatchSaveCommands) != len(batchSaveCommands) ||
		!containsBatchSaveCommand(batchSaveCommands, "query-readback", "readbackTables", 0, "worklist-query-readback-readbacktables-missing-batch-0") {
		t.Fatalf("expected worklist to expose top-level/operator batch save command index, got top=%#v packet=%#v totals=%#v", batchSaveCommands, operatorBatchSaveCommands, totals)
	}
	readbackTablesQueue := map[string]any(nil)
	for _, item := range queues {
		queue := item.(map[string]any)
		if queue["artifactType"] == "query-readback" && queue["section"] == "readbackTables" {
			readbackTablesQueue = queue
			break
		}
	}
	if readbackTablesQueue == nil ||
		readbackTablesQueue["requiredShapeKey"] != "requiredReadbackShape" ||
		readbackTablesQueue["manifestStatusField"] != "queryEvidenceStatus" ||
		int(readbackTablesQueue["missing"].(float64)) != setupSvcLiveReplayOperationCount() {
		t.Fatalf("expected query-readback readbackTables queue in worklist, got %#v", queues)
	}
	batches := readbackTablesQueue["batches"].([]any)
	if len(batches) != (setupSvcLiveReplayOperationCount()+24)/25 {
		t.Fatalf("expected every readbackTables batch expanded, got %#v", batches)
	}
	firstBatch := batches[0].(map[string]any)
	if firstBatch["command"] == "" ||
		!strings.Contains(firstBatch["command"].(string), "setup-svc-live-replay-worklist") ||
		strings.Contains(firstBatch["command"].(string), "setup-svc-live-replay-gaps") ||
		!strings.Contains(firstBatch["command"].(string), "--artifact-type query-readback") ||
		!strings.Contains(firstBatch["command"].(string), "--evidence-section readbackTables") ||
		!strings.Contains(firstBatch["command"].(string), "--batch-index 0") ||
		!strings.Contains(firstBatch["suggestedWorklistPath"].(string), "worklist-query-readback-readbacktables-missing-batch-0") ||
		!strings.Contains(firstBatch["saveWorklistCommand"].(string), " > ") ||
		!strings.Contains(firstBatch["saveWorklistCommand"].(string), firstBatch["suggestedWorklistPath"].(string)) ||
		!strings.Contains(firstBatch["dryRunImportCommand"].(string), "setup-svc-live-replay-evidence-import") ||
		!strings.Contains(firstBatch["dryRunImportCommand"].(string), firstBatch["suggestedWorklistPath"].(string)) ||
		int(firstBatch["count"].(float64)) != 25 {
		t.Fatalf("expected first worklist batch to expose copy-ready command and 25 artifacts, got %#v", firstBatch)
	}
	artifacts := firstBatch["artifacts"].([]any)
	firstArtifact := artifacts[0].(map[string]any)
	if firstArtifact["artifactType"] != "query-readback" ||
		firstArtifact["requiredShapeKey"] != "requiredReadbackShape" ||
		firstArtifact["manifestStatusField"] != "queryEvidenceStatus" ||
		!containsStringItem(firstArtifact["requiredEvidenceSections"].([]any), "readbackTables") ||
		!containsStringItem(firstArtifact["requiredTables"].([]any), "tp_sys_object") ||
		!containsStringItem(firstArtifact["queryReadbackExpectations"].([]any), "object-identity-prefix-datatable-readback") ||
		!containsStringFragment(firstArtifact["replacementChecklist"].([]any), "readback table coverage") {
		t.Fatalf("expected expanded worklist artifact to carry query readback contract, got %#v", firstArtifact)
	}
	if !evidenceSectionHasStatus(firstArtifact["evidenceSectionStatuses"].([]any), "readbackTables", "missing") ||
		!evidenceSectionHasStatus(firstArtifact["evidenceSectionStatuses"].([]any), "relationshipChecks", "missing") {
		t.Fatalf("expected expanded worklist artifact to keep section statuses, got %#v", firstArtifact)
	}
}

func TestSetupSvcLiveReplaySourceChecklistDedupesWorklistRecords(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-workspace", "--execute", "--approval", setupSvcParityEvidenceWorkspaceApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-capture-source-workspace", "--execute", "--approval", setupSvcParityCaptureSourceWorkspaceApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-source-checklist", "--source-readiness", "incomplete"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["mode"] != "setup-svc-live-replay-source-checklist" || result["readOnly"] != true {
		t.Fatalf("expected read-only source checklist, got %#v", result)
	}
	filters := result["filters"].(map[string]any)
	if filters["sourceReadiness"] != "incomplete" {
		t.Fatalf("expected source checklist to preserve top-level source-readiness filter, got %#v", filters)
	}
	totals := result["totals"].(map[string]any)
	if int(totals["inputWorklists"].(float64)) != 1 ||
		int(totals["worklistBatches"].(float64)) != 58 ||
		int(totals["replacementRecords"].(float64)) != setupSvcLiveReplayExpectedMissingSectionRecordCount() ||
		int(totals["uniqueSourceFiles"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount() ||
		int(totals["uniqueTargetFiles"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount() {
		t.Fatalf("expected source checklist to collapse expanded worklist records to unique source files, got %#v", totals)
	}
	if int(result["sourceFiles"].(float64)) != int(totals["uniqueSourceFiles"].(float64)) ||
		int(result["targetFiles"].(float64)) != int(totals["uniqueTargetFiles"].(float64)) ||
		int(result["replacementRecords"].(float64)) != int(totals["replacementRecords"].(float64)) ||
		int(result["worklistBatches"].(float64)) != int(totals["worklistBatches"].(float64)) {
		t.Fatalf("expected source checklist top-level counters to mirror totals, got result=%#v totals=%#v", result, totals)
	}
	artifactCounts := result["artifactTypeCounts"].([]any)
	if !sourceChecklistHasArtifactCount(artifactCounts, "setup-svc", setupSvcLiveReplayOperationCount(), 2*setupSvcLiveReplayOperationCount()) ||
		!sourceChecklistHasArtifactCount(artifactCounts, "metadata-service", setupSvcLiveReplayOperationCount(), 3*setupSvcLiveReplayOperationCount()) ||
		!sourceChecklistHasArtifactCount(artifactCounts, "query-readback", setupSvcLiveReplayOperationCount(), 6*setupSvcLiveReplayOperationCount()) ||
		!sourceChecklistHasArtifactCount(artifactCounts, "normalized-diff", setupSvcLiveReplayOperationCount(), 2*setupSvcLiveReplayOperationCount()) ||
		!sourceChecklistHasArtifactCount(artifactCounts, "cleanup", setupSvcLiveReplayWriteOperationCount(), 2*setupSvcLiveReplayWriteOperationCount()) {
		t.Fatalf("expected artifact counts to preserve source and section backlog counts, got %#v", artifactCounts)
	}
	readinessCounts := result["sourceReadinessCounts"].([]any)
	if len(readinessCounts) != 1 ||
		readinessCounts[0].(map[string]any)["sourceReadiness"] != "incomplete" ||
		int(readinessCounts[0].(map[string]any)["records"].(float64)) != setupSvcLiveReplayExpectedMissingSectionRecordCount() ||
		int(readinessCounts[0].(map[string]any)["sourceFiles"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount() {
		t.Fatalf("expected incomplete source readiness summary after capture workspace initialization, got %#v", readinessCounts)
	}
	missingSectionCounts := result["missingEvidenceSectionCounts"].([]any)
	if !sourceChecklistHasMissingSectionCount(missingSectionCounts, "tableSnapshots", 2*setupSvcLiveReplayOperationCount(), "metadata-service", "setup-svc") ||
		!sourceChecklistHasMissingSectionCount(missingSectionCounts, "runtimeEffectChecks", 2*setupSvcLiveReplayOperationCount(), "metadata-service", "setup-svc") ||
		!sourceChecklistHasMissingSectionCount(missingSectionCounts, "metadataServiceDatasource", setupSvcLiveReplayOperationCount(), "metadata-service") ||
		!sourceChecklistHasMissingSectionCount(missingSectionCounts, "readbackTables", setupSvcLiveReplayOperationCount(), "query-readback") ||
		!sourceChecklistHasMissingSectionCount(missingSectionCounts, "diffCounters", setupSvcLiveReplayOperationCount(), "normalized-diff") ||
		!sourceChecklistHasMissingSectionCount(missingSectionCounts, "residualCounters", setupSvcLiveReplayWriteOperationCount(), "cleanup") {
		t.Fatalf("expected missing evidence section counts to summarize source-file backlog, got %#v", missingSectionCounts)
	}
	nextQueueCommands := result["nextSourceQueueCommands"].([]any)
	if int(result["nextQueueCount"].(float64)) != len(nextQueueCommands) ||
		int(result["repairQueueCount"].(float64)) != len(nextQueueCommands) ||
		int(result["missingSectionKinds"].(float64)) != len(missingSectionCounts) {
		t.Fatalf("expected source checklist top-level queue counters to mirror queues, got result=%#v", result)
	}
	if !sourceChecklistHasNextQueueCommand(nextQueueCommands, "", "tableSnapshots", 2*setupSvcLiveReplayOperationCount(), "") ||
		!sourceChecklistHasNextQueueCommand(nextQueueCommands, "metadata-service", "metadataServiceDatasource", setupSvcLiveReplayOperationCount(), "") ||
		!sourceChecklistHasNextQueueCommand(nextQueueCommands, "query-readback", "readbackTables", setupSvcLiveReplayOperationCount(), "") ||
		!sourceChecklistHasNextQueueCommand(nextQueueCommands, "cleanup", "residualCounters", setupSvcLiveReplayWriteOperationCount(), "") {
		t.Fatalf("expected next source queue commands to summarize actionable source queues, got %#v", nextQueueCommands)
	}
	if !sourceChecklistHasNextQueuePagination(nextQueueCommands, "", "tableSnapshots", 2*setupSvcLiveReplayOperationCount(), 0, 25, 25, 8, 2*setupSvcLiveReplayOperationCount()-25) ||
		!sourceChecklistHasNextQueuePagination(nextQueueCommands, "cleanup", "residualCounters", setupSvcLiveReplayWriteOperationCount(), 0, 25, 25, 3, setupSvcLiveReplayWriteOperationCount()-25) {
		t.Fatalf("expected next source queue commands to expose first-page pagination metadata, got %#v", nextQueueCommands)
	}
	if !sourceChecklistHasNextPageCommands(nextQueueCommands, "", "tableSnapshots", 25) ||
		!sourceChecklistHasNextPageCommands(nextQueueCommands, "cleanup", "residualCounters", 25) {
		t.Fatalf("expected next source queue commands to expose next-page commands, got %#v", nextQueueCommands)
	}
	if !sourceChecklistHasAllPageSaveCommands(nextQueueCommands, "", "tableSnapshots", 8, 0, 175) ||
		!sourceChecklistHasAllPageSaveCommands(nextQueueCommands, "cleanup", "residualCounters", 3, 0, 50) {
		t.Fatalf("expected next source queue commands to expose all-page save commands, got %#v", nextQueueCommands)
	}
	if !sourceChecklistHasPageCommandSummary(result, 8, 0, 175) {
		t.Fatalf("expected source checklist to expose flattened all-page save commands, got %#v", result)
	}
	if !sourceChecklistHasPageSaveScript(result, 0, 175) {
		t.Fatalf("expected source checklist to expose a page save script, got %#v", result)
	}
	if !sourceChecklistHasSavePageScriptCommand(result, ".pageSaveScript", "setup-svc-live-replay-source-checklist") {
		t.Fatalf("expected source checklist to expose a saveable page script command, got %#v", result)
	}
	operatorPacket := result["operatorPacket"].(map[string]any)
	if int(operatorPacket["sourceFiles"].(float64)) != int(result["sourceFiles"].(float64)) ||
		int(operatorPacket["targetFiles"].(float64)) != int(result["targetFiles"].(float64)) ||
		int(operatorPacket["replacementRecords"].(float64)) != int(result["replacementRecords"].(float64)) ||
		int(operatorPacket["nextQueueCount"].(float64)) != len(nextQueueCommands) ||
		int(operatorPacket["repairQueueCount"].(float64)) != len(nextQueueCommands) {
		t.Fatalf("expected source checklist operatorPacket counters to mirror top-level counters, got operatorPacket=%#v result=%#v", operatorPacket, result)
	}
	sources := result["sources"].([]any)
	if len(sources) != setupSvcLiveReplayExpectedArtifactFileCount() {
		t.Fatalf("expected one source checklist entry per artifact, got %d", len(sources))
	}
	querySource := map[string]any(nil)
	for _, item := range sources {
		source := item.(map[string]any)
		if source["artifactType"] == "query-readback" {
			querySource = source
			break
		}
	}
	if querySource == nil ||
		!containsStringItem(querySource["missingEvidenceSections"].([]any), "readbackTables") ||
		!containsStringItem(querySource["missingEvidenceSections"].([]any), "relationshipChecks") ||
		!containsStringItem(querySource["missingEvidenceSections"].([]any), "readbackExpectationChecks") ||
		!containsStringItem(querySource["missingEvidenceSections"].([]any), "cleanCounters") ||
		len(querySource["worklistFiles"].([]any)) < 4 {
		t.Fatalf("expected query-readback source entry to merge all missing section queues, got %#v", querySource)
	}
	operatorPacket = result["operatorPacket"].(map[string]any)
	if int(operatorPacket["sourceFiles"].(float64)) != int(totals["uniqueSourceFiles"].(float64)) ||
		int(operatorPacket["targetFiles"].(float64)) != int(totals["uniqueTargetFiles"].(float64)) ||
		int(operatorPacket["replacementRecords"].(float64)) != int(totals["replacementRecords"].(float64)) ||
		int(operatorPacket["worklistQueues"].(float64)) != int(totals["worklistQueues"].(float64)) ||
		int(operatorPacket["worklistBatches"].(float64)) != int(totals["worklistBatches"].(float64)) ||
		int(operatorPacket["missingSectionKinds"].(float64)) != len(missingSectionCounts) ||
		int(operatorPacket["nextQueueCount"].(float64)) != len(nextQueueCommands) ||
		int(operatorPacket["repairQueueCount"].(float64)) != len(nextQueueCommands) {
		t.Fatalf("expected source checklist operator packet to mirror machine-readable totals, got %#v totals=%#v", operatorPacket, totals)
	}
	if !strings.Contains(operatorPacket["saveChecklistCommand"].(string), "setup-svc-live-replay-source-checklist") ||
		!strings.Contains(operatorPacket["suggestedChecklistPath"].(string), "source-capture-checklist") ||
		operatorPacket["sourceRoot"] != "captures" ||
		operatorPacket["captureRoot"] != filepath.Join(tmp, "captures") {
		t.Fatalf("expected source checklist operator packet to expose save command and capture root, got %#v", operatorPacket)
	}
	operatorNextQueueCommands := operatorPacket["nextSourceQueueCommands"].([]any)
	if !sourceChecklistHasNextQueueCommand(operatorNextQueueCommands, "query-readback", "readbackTables", setupSvcLiveReplayOperationCount(), "") {
		t.Fatalf("expected operator packet to mirror next source queue commands, got %#v", operatorNextQueueCommands)
	}
	if !sourceChecklistHasPageCommandSummary(operatorPacket, 8, 0, 175) {
		t.Fatalf("expected operator packet to mirror flattened all-page save commands, got %#v", operatorPacket)
	}
	if !sourceChecklistHasPageSaveScript(operatorPacket, 0, 175) {
		t.Fatalf("expected operator packet to mirror page save script, got %#v", operatorPacket)
	}
	if !sourceChecklistHasSavePageScriptCommand(operatorPacket, ".pageSaveScript", "setup-svc-live-replay-source-checklist") {
		t.Fatalf("expected operator packet to mirror saveable page script command, got %#v", operatorPacket)
	}

	stdout.Reset()
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-source-checklist", "--artifact-type", "query-readback", "--evidence-section", "readbackTables", "--section-status", "missing", "--source-readiness", "incomplete"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	result = map[string]any{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	filters = result["filters"].(map[string]any)
	if filters["artifactType"] != "query-readback" ||
		filters["evidenceSection"] != "readbackTables" ||
		filters["sectionStatus"] != "missing" ||
		filters["sourceReadiness"] != "incomplete" {
		t.Fatalf("expected source checklist to preserve top-level repairQueue filters, got %#v", filters)
	}
	totals = result["totals"].(map[string]any)
	if int(totals["replacementRecords"].(float64)) != setupSvcLiveReplayOperationCount() ||
		int(totals["uniqueSourceFiles"].(float64)) != setupSvcLiveReplayOperationCount() {
		t.Fatalf("expected filtered query-readback/readbackTables checklist to expose all operation records and sources, got %#v", totals)
	}
	missingSectionCounts = result["missingEvidenceSectionCounts"].([]any)
	if !sourceChecklistHasMissingSectionCount(missingSectionCounts, "readbackTables", setupSvcLiveReplayOperationCount(), "query-readback") ||
		!sourceChecklistHasMissingSectionCount(missingSectionCounts, "relationshipChecks", setupSvcLiveReplayOperationCount(), "query-readback") ||
		!sourceChecklistHasMissingSectionCount(missingSectionCounts, "cleanCounters", setupSvcLiveReplayOperationCount(), "query-readback") {
		t.Fatalf("expected filtered checklist to preserve all source-level missing section counts, got %#v", missingSectionCounts)
	}
	nextQueueCommands = result["nextSourceQueueCommands"].([]any)
	if !sourceChecklistHasNextQueueCommand(nextQueueCommands, "query-readback", "readbackTables", setupSvcLiveReplayOperationCount(), "incomplete") ||
		!sourceChecklistHasNextQueueCommand(nextQueueCommands, "query-readback", "relationshipChecks", setupSvcLiveReplayOperationCount(), "incomplete") ||
		!sourceChecklistHasNextQueueCommand(nextQueueCommands, "query-readback", "cleanCounters", setupSvcLiveReplayOperationCount(), "incomplete") {
		t.Fatalf("expected filtered checklist to emit source queue commands with repair filters, got %#v", nextQueueCommands)
	}
	if !sourceChecklistHasNextQueuePagination(nextQueueCommands, "query-readback", "readbackTables", setupSvcLiveReplayOperationCount(), 0, 25, 25, 4, setupSvcLiveReplayOperationCount()-25) {
		t.Fatalf("expected filtered checklist to emit pagination metadata, got %#v", nextQueueCommands)
	}
	if !sourceChecklistHasNextPageCommands(nextQueueCommands, "query-readback", "readbackTables", 25) {
		t.Fatalf("expected filtered checklist to emit next-page commands, got %#v", nextQueueCommands)
	}
	if !sourceChecklistHasAllPageSaveCommands(nextQueueCommands, "query-readback", "readbackTables", 4, 0, 75) {
		t.Fatalf("expected filtered checklist to emit all-page save commands, got %#v", nextQueueCommands)
	}
}

func TestSetupSvcLiveReplaySourceExecutionPacketGroupsCaptureBatches(t *testing.T) {
	t.Setenv("MDS_RUNTIME_MODE", "self-hosted")
	t.Setenv("MDS_SERVER_PORT", "18087")
	t.Setenv("MDS_JDBC_URL", "jdbc:mysql://source-execution-db-host:3306/source_execution_metadata")
	t.Setenv("MDS_DB_USERNAME", "source-execution-user")
	t.Setenv("MDS_DB_PASSWORD", "source-execution-password")
	t.Setenv("MDS_DB_DRIVER", "com.mysql.cj.jdbc.Driver")
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-workspace", "--execute", "--approval", setupSvcParityEvidenceWorkspaceApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-capture-source-workspace", "--execute", "--approval", setupSvcParityCaptureSourceWorkspaceApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-source-execution-packet", "--source-readiness", "incomplete"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["mode"] != "setup-svc-live-replay-source-execution-packet" || result["readOnly"] != true || result["generatedFrom"] != "setup-svc-live-replay-source-checklist" {
		t.Fatalf("expected read-only source execution packet, got %#v", result)
	}
	datasource := result["metadataServiceDatasource"].(map[string]any)
	if datasource["status"] != "ready" || datasource["readyForRealDatasource"] != true || datasource["jdbcUrlSource"] != "env:MDS_JDBC_URL" {
		t.Fatalf("expected source-execution datasource readiness, got %#v", datasource)
	}
	if strings.Contains(stdout.String(), "source-execution-db-host") ||
		strings.Contains(stdout.String(), "source_execution_metadata") ||
		strings.Contains(stdout.String(), "source-execution-user") ||
		strings.Contains(stdout.String(), "source-execution-password") {
		t.Fatalf("source-execution packet leaked datasource secret values: %s", stdout.String())
	}
	totals := result["totals"].(map[string]any)
	if int(totals["sourceFiles"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount() ||
		int(totals["targetFiles"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount() ||
		int(totals["incompleteSourceFiles"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount() ||
		int(totals["artifactTypes"].(float64)) != 5 ||
		int(totals["domainOperations"].(float64)) != setupSvcLiveReplayOperationCount() ||
		int(totals["evidenceSections"].(float64)) != 13 ||
		int(totals["captureGroups"].(float64)) != 6 ||
		int(totals["groupedSourceFiles"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount() ||
		int(totals["groupedTargetFiles"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount() {
		t.Fatalf("expected execution packet totals to match source checklist backlog, got %#v", totals)
	}
	if int(result["sourceFiles"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount() ||
		int(result["targetFiles"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount() ||
		int(result["sourceFilesIncomplete"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount() ||
		int(result["captureGroups"].(float64)) != 6 ||
		int(result["artifactTypes"].(float64)) != 5 ||
		int(result["domainOperations"].(float64)) != setupSvcLiveReplayOperationCount() ||
		int(result["evidenceSectionCount"].(float64)) != 13 {
		t.Fatalf("expected execution packet to mirror automation-friendly top-level totals, got %#v", result)
	}
	operatorPacket := result["operatorPacket"].(map[string]any)
	if int(operatorPacket["sourceFiles"].(float64)) != int(totals["sourceFiles"].(float64)) ||
		int(operatorPacket["targetFiles"].(float64)) != int(totals["targetFiles"].(float64)) ||
		int(operatorPacket["incompleteSourceFiles"].(float64)) != int(totals["incompleteSourceFiles"].(float64)) ||
		int(operatorPacket["artifactTypes"].(float64)) != int(totals["artifactTypes"].(float64)) ||
		int(operatorPacket["domainOperations"].(float64)) != int(totals["domainOperations"].(float64)) ||
		int(operatorPacket["evidenceSectionCount"].(float64)) != int(totals["evidenceSections"].(float64)) ||
		int(operatorPacket["captureGroups"].(float64)) != int(totals["captureGroups"].(float64)) ||
		int(operatorPacket["operatorBatchCount"].(float64)) != 6 ||
		int(operatorPacket["runbookStepCount"].(float64)) != 6 ||
		int(operatorPacket["batchSaveCommandCount"].(float64)) != 6 ||
		int(operatorPacket["importBatchSaveCommandCount"].(float64)) != 6 ||
		operatorPacket["sourceRoot"] != "captures" ||
		operatorPacket["captureRoot"] != filepath.Join(tmp, "captures") ||
		!strings.Contains(operatorPacket["completionAuditCommand"].(string), "setup-svc-live-replay-completion-audit") ||
		operatorPacket["metadataServiceDatasource"].(map[string]any)["readyForRealDatasource"] != true {
		t.Fatalf("expected source execution operatorPacket to mirror script-friendly totals and commands, got %#v totals=%#v", operatorPacket, totals)
	}
	groups := result["captureModeGroups"].([]any)
	groupAlias := result["groups"].([]any)
	if len(groupAlias) != len(groups) || !sourceExecutionHasArtifactOrder(groupAlias, []string{"setup-svc", "metadata-service", "metadata-service", "query-readback", "normalized-diff", "cleanup"}) {
		t.Fatalf("expected groups alias to mirror captureModeGroups, got groups=%#v captureModeGroups=%#v", groupAlias, groups)
	}
	if !sourceExecutionHasArtifactOrder(groups, []string{"setup-svc", "metadata-service", "metadata-service", "query-readback", "normalized-diff", "cleanup"}) {
		t.Fatalf("expected execution groups to follow replay dependency order, got %#v", groups)
	}
	if !sourceExecutionHasGroup(groups, "setup-svc", "setup-svc", "manual_or_scripted_snapshot_capture", setupSvcLiveReplayOperationCount(), setupSvcLiveReplayOperationCount(), setupSvcLiveReplayOperationCount(), "tableSnapshots", "runtimeEffectChecks") ||
		!sourceExecutionHasGroup(groups, "metadata-service", "metadata-service", "msapi_plan_apply_snapshot_capture", setupSvcLiveReplayWriteOperationCount(), setupSvcLiveReplayWriteOperationCount(), setupSvcLiveReplayWriteOperationCount(), "tableSnapshots", "runtimeEffectChecks") ||
		!sourceExecutionHasGroup(groups, "metadata-service", "metadata-service", "msapi_scan_snapshot_capture", 21, 21, 21, "tableSnapshots", "runtimeEffectChecks") ||
		!sourceExecutionHasGroup(groups, "query-readback", "msapi-query-readback", "msapi_query_readback_capture", setupSvcLiveReplayOperationCount(), setupSvcLiveReplayOperationCount(), setupSvcLiveReplayOperationCount(), "readbackTables", "cleanCounters") ||
		!sourceExecutionHasGroup(groups, "normalized-diff", "local-normalized-diff", "approval_gated_generated_diff", setupSvcLiveReplayOperationCount(), setupSvcLiveReplayOperationCount(), setupSvcLiveReplayOperationCount(), "diffCounters", "nestedCleanCounters") ||
		!sourceExecutionHasGroup(groups, "cleanup", "cleanup-verifier", "cleanup_residual_capture", setupSvcLiveReplayWriteOperationCount(), setupSvcLiveReplayWriteOperationCount(), setupSvcLiveReplayWriteOperationCount(), "residualCounters", "deletedOrRemovedEvidence") {
		t.Fatalf("expected execution packet to expose all artifact capture groups, got %#v", groups)
	}
	operatorBatches := result["operatorBatches"].([]any)
	if !sourceExecutionOperatorBatchHasCommands(operatorBatches, "query-readback", "setup-svc-live-replay-evidence-import", "setup-svc-live-replay-completion-audit") {
		t.Fatalf("expected operator batches to include import and completion commands, got %#v", operatorBatches)
	}
	if !sourceExecutionOperatorBatchesHaveTargetFiles(operatorBatches) {
		t.Fatalf("expected operator batches to mirror targetFiles for batch-local reconciliation, got %#v", operatorBatches)
	}
	if !sourceExecutionMetadataBatchesHaveDatasourceReadiness(operatorBatches) {
		t.Fatalf("expected metadata-service operator batches to expose datasource readiness, got %#v", operatorBatches)
	}
	if !sourceExecutionHasBatchSaveScript(result, 6, "query-readback-msapi_query_readback_capture-source-capture-batch-readiness-incomplete.json") {
		t.Fatalf("expected source execution packet to expose saveable batch script, got %#v", result)
	}
	if !sourceExecutionHasImportBatchSaveScript(result, 6, "query-readback-msapi_query_readback_capture-source-capture-batch-readiness-complete.json") {
		t.Fatalf("expected source execution packet to expose saveable complete import batch script, got %#v", result)
	}
	if !sourceExecutionHasRunbookMarkdown(result, 6, "query-readback") {
		t.Fatalf("expected source execution packet to expose saveable markdown runbook, got %#v", result)
	}
	if !sourceExecutionBatchSaveCommandsInOrder(result, []string{"setup-svc", "metadata-service", "metadata-service", "query-readback", "normalized-diff", "cleanup"}) {
		t.Fatalf("expected batch save commands to follow replay dependency order, got %#v", result["batchSaveCommands"])
	}
	if !sourceExecutionRunbookHasDependencyOrder(result, []string{"setup-svc", "metadata-service", "metadata-service", "query-readback", "normalized-diff", "cleanup"}) {
		t.Fatalf("expected execution runbook to expose dependency gates in replay order, got %#v", result["executionRunbook"])
	}

	stdout.Reset()
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-source-execution-packet", "--artifact-type", "query-readback", "--source-readiness", "incomplete"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	result = map[string]any{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	totals = result["totals"].(map[string]any)
	if int(totals["sourceFiles"].(float64)) != setupSvcLiveReplayOperationCount() ||
		int(totals["artifactTypes"].(float64)) != 1 ||
		int(totals["captureGroups"].(float64)) != 1 ||
		int(totals["evidenceSections"].(float64)) != 6 {
		t.Fatalf("expected filtered query-readback execution packet, got %#v", totals)
	}
	groups = result["captureModeGroups"].([]any)
	if len(groups) != 1 || !sourceExecutionHasGroup(groups, "query-readback", "msapi-query-readback", "msapi_query_readback_capture", setupSvcLiveReplayOperationCount(), setupSvcLiveReplayOperationCount(), setupSvcLiveReplayOperationCount(), "readbackExpectationChecks", "relationshipChecks") {
		t.Fatalf("expected filtered query-readback group to preserve all query evidence sections, got %#v", groups)
	}
	if result["artifactType"] != "query-readback" ||
		result["sourceSystem"] != "msapi-query-readback" ||
		result["captureMode"] != "msapi_query_readback_capture" ||
		int(result["sourceFiles"].(float64)) != setupSvcLiveReplayOperationCount() ||
		int(result["targetFiles"].(float64)) != setupSvcLiveReplayOperationCount() ||
		int(result["domainOperations"].(float64)) != setupSvcLiveReplayOperationCount() ||
		len(result["items"].([]any)) != setupSvcLiveReplayOperationCount() {
		t.Fatalf("expected filtered packet to mirror the single capture batch at top level, got %#v", result)
	}
	operatorBatch := result["operatorBatch"].(map[string]any)
	if operatorBatch["artifactType"] != "query-readback" ||
		int(operatorBatch["sourceFiles"].(float64)) != setupSvcLiveReplayOperationCount() ||
		int(operatorBatch["targetFiles"].(float64)) != setupSvcLiveReplayOperationCount() ||
		!strings.Contains(operatorBatch["dryRunImportCommand"].(string), "setup-svc-live-replay-evidence-import") ||
		!strings.Contains(operatorBatch["dryRunImportCommand"].(string), "query-readback-msapi_query_readback_capture-source-capture-batch-readiness-complete.json") ||
		strings.Contains(operatorBatch["dryRunImportCommand"].(string), "readiness-incomplete") ||
		!strings.Contains(operatorBatch["saveImportBatchCommand"].(string), "--source-readiness complete") ||
		!strings.Contains(operatorBatch["completionAuditCommand"].(string), "setup-svc-live-replay-completion-audit") {
		t.Fatalf("expected filtered packet to mirror single operator batch commands, got %#v", operatorBatch)
	}
	if !sourceExecutionHasBatchSaveScript(result, 1, "query-readback-msapi_query_readback_capture-source-capture-batch-readiness-incomplete.json") {
		t.Fatalf("expected filtered query-readback packet to expose one save command, got %#v", result)
	}
	if !sourceExecutionHasImportBatchSaveScript(result, 1, "query-readback-msapi_query_readback_capture-source-capture-batch-readiness-complete.json") {
		t.Fatalf("expected filtered query-readback packet to expose one complete import save command, got %#v", result)
	}
	if !sourceExecutionHasRunbookMarkdown(result, 1, "query-readback") {
		t.Fatalf("expected filtered query-readback packet to expose a focused markdown runbook, got %#v", result)
	}
	if !sourceExecutionRunbookHasDependencyOrder(result, []string{"query-readback"}) {
		t.Fatalf("expected filtered query-readback packet to expose a single runbook step, got %#v", result["executionRunbook"])
	}

	stdout.Reset()
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-source-execution-packet", "--artifact-type", "metadata-service", "--capture-mode", "msapi_scan_snapshot_capture", "--source-readiness", "incomplete"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	result = map[string]any{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	totals = result["totals"].(map[string]any)
	if int(totals["sourceFiles"].(float64)) != 21 ||
		int(totals["artifactTypes"].(float64)) != 1 ||
		int(totals["captureGroups"].(float64)) != 1 ||
		int(totals["evidenceSections"].(float64)) != 3 {
		t.Fatalf("expected filtered metadata-service query scan execution packet, got %#v", totals)
	}
	groups = result["captureModeGroups"].([]any)
	if len(groups) != 1 || !sourceExecutionHasGroup(groups, "metadata-service", "metadata-service", "msapi_scan_snapshot_capture", 21, 21, 21, "tableSnapshots", "runtimeEffectChecks", "metadataServiceDatasource") {
		t.Fatalf("expected filtered metadata-service scan group, got %#v", groups)
	}
	if result["artifactType"] != "metadata-service" ||
		result["sourceSystem"] != "metadata-service" ||
		result["captureMode"] != "msapi_scan_snapshot_capture" ||
		int(result["sourceFiles"].(float64)) != 21 ||
		int(result["targetFiles"].(float64)) != 21 ||
		int(result["domainOperations"].(float64)) != 21 ||
		len(result["items"].([]any)) != 21 {
		t.Fatalf("expected filtered metadata-service scan packet to mirror one capture batch, got %#v", result)
	}
	operatorBatch = result["operatorBatch"].(map[string]any)
	if operatorBatch["artifactType"] != "metadata-service" ||
		operatorBatch["captureMode"] != "msapi_scan_snapshot_capture" ||
		int(operatorBatch["sourceFiles"].(float64)) != 21 ||
		int(operatorBatch["targetFiles"].(float64)) != 21 ||
		!strings.Contains(operatorBatch["saveBatchCommand"].(string), "--capture-mode msapi_scan_snapshot_capture") ||
		!strings.Contains(operatorBatch["dryRunImportCommand"].(string), "metadata-service-msapi_scan_snapshot_capture-source-capture-batch-readiness-complete.json") ||
		!strings.Contains(operatorBatch["nextAction"].(string), "query scan capture") {
		t.Fatalf("expected metadata-service scan operator batch to expose mode-specific commands, got %#v", operatorBatch)
	}
	if operatorBatch["metadataServiceDatasource"].(map[string]any)["readyForRealDatasource"] != true {
		t.Fatalf("expected filtered metadata-service batch to expose datasource readiness, got %#v", operatorBatch)
	}
	if !sourceExecutionHasBatchSaveScript(result, 1, "metadata-service-msapi_scan_snapshot_capture-source-capture-batch-readiness-incomplete.json") {
		t.Fatalf("expected filtered metadata-service scan packet to expose one save command, got %#v", result)
	}
	if !sourceExecutionHasImportBatchSaveScript(result, 1, "metadata-service-msapi_scan_snapshot_capture-source-capture-batch-readiness-complete.json") {
		t.Fatalf("expected filtered metadata-service scan packet to expose one complete import save command, got %#v", result)
	}
	if !sourceExecutionRunbookHasDependencyOrder(result, []string{"metadata-service"}) {
		t.Fatalf("expected filtered metadata-service scan packet to expose a single runbook step, got %#v", result["executionRunbook"])
	}
	runbook := result["executionRunbook"].([]any)
	if runbook[0].(map[string]any)["metadataServiceDatasource"].(map[string]any)["readyForRealDatasource"] != true {
		t.Fatalf("expected filtered metadata-service runbook to expose datasource readiness, got %#v", runbook)
	}
}

func TestSetupSvcLiveReplaySourceHealthSummarizesRepairQueues(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-workspace", "--execute", "--approval", setupSvcParityEvidenceWorkspaceApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-capture-source-workspace", "--execute", "--approval", setupSvcParityCaptureSourceWorkspaceApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-source-health", "--source-readiness", "incomplete"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["mode"] != "setup-svc-live-replay-source-health" || result["readOnly"] != true || result["generatedFrom"] != "setup-svc-live-replay-source-checklist" {
		t.Fatalf("expected read-only source health report, got %#v", result)
	}
	if result["status"] != "pending_source_repair" {
		t.Fatalf("expected incomplete sources to require repair, got %#v", result["status"])
	}
	totals := result["totals"].(map[string]any)
	if int(totals["sourceFiles"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount() ||
		int(totals["targetFiles"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount() ||
		int(totals["artifactTypes"].(float64)) != 5 ||
		int(totals["domainOperations"].(float64)) != setupSvcLiveReplayOperationCount() ||
		int(totals["completeSourceFiles"].(float64)) != 0 ||
		int(totals["incompleteSourceFiles"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount() ||
		int(totals["repairRequiredSourceFiles"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount() ||
		int(totals["missingSectionInstances"].(float64)) != setupSvcLiveReplayExpectedMissingSectionRecordCount() ||
		totals["canImportCompleteSources"].(bool) {
		t.Fatalf("expected source health totals to summarize incomplete capture sources, got %#v", totals)
	}
	if int(result["sourceFiles"].(float64)) != int(totals["sourceFiles"].(float64)) ||
		int(result["targetFiles"].(float64)) != int(totals["targetFiles"].(float64)) ||
		int(result["sourceFilesPresent"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount() ||
		int(result["sourceFilesComplete"].(float64)) != int(totals["completeSourceFiles"].(float64)) ||
		int(result["sourceFilesIncomplete"].(float64)) != int(totals["incompleteSourceFiles"].(float64)) ||
		int(result["sourceFilesMissing"].(float64)) != int(totals["missingSourceFiles"].(float64)) ||
		int(result["completeSourceFiles"].(float64)) != int(totals["completeSourceFiles"].(float64)) ||
		int(result["incompleteSourceFiles"].(float64)) != int(totals["incompleteSourceFiles"].(float64)) ||
		int(result["missingSourceFiles"].(float64)) != int(totals["missingSourceFiles"].(float64)) {
		t.Fatalf("expected source health top-level source counters to mirror totals, got result=%#v totals=%#v", result, totals)
	}
	artifactTypes := result["artifactTypes"].([]any)
	if !sourceHealthHasArtifactType(artifactTypes, "setup-svc", setupSvcLiveReplayOperationCount(), 2*setupSvcLiveReplayOperationCount(), "tableSnapshots") ||
		!sourceHealthHasArtifactType(artifactTypes, "metadata-service", setupSvcLiveReplayOperationCount(), 3*setupSvcLiveReplayOperationCount(), "metadataServiceDatasource") ||
		!sourceHealthHasArtifactType(artifactTypes, "query-readback", setupSvcLiveReplayOperationCount(), 6*setupSvcLiveReplayOperationCount(), "readbackTables") ||
		!sourceHealthHasArtifactType(artifactTypes, "normalized-diff", setupSvcLiveReplayOperationCount(), 2*setupSvcLiveReplayOperationCount(), "diffCounters") ||
		!sourceHealthHasArtifactType(artifactTypes, "cleanup", setupSvcLiveReplayWriteOperationCount(), 2*setupSvcLiveReplayWriteOperationCount(), "residualCounters") {
		t.Fatalf("expected source health artifact summaries, got %#v", artifactTypes)
	}
	missingSections := result["missingSections"].([]any)
	if !sourceHealthHasMissingSection(missingSections, "metadata-service", "metadataServiceDatasource", setupSvcLiveReplayOperationCount()) ||
		!sourceHealthHasMissingSection(missingSections, "query-readback", "readbackTables", setupSvcLiveReplayOperationCount()) ||
		!sourceHealthHasMissingSection(missingSections, "cleanup", "residualCounters", setupSvcLiveReplayWriteOperationCount()) {
		t.Fatalf("expected source health missing sections with queue commands, got %#v", missingSections)
	}
	missingEvidenceSectionCounts := result["missingEvidenceSectionCounts"].([]any)
	if !sourceHealthHasMissingSection(missingEvidenceSectionCounts, "metadata-service", "metadataServiceDatasource", setupSvcLiveReplayOperationCount()) ||
		!sourceHealthHasMissingSection(missingEvidenceSectionCounts, "query-readback", "readbackTables", setupSvcLiveReplayOperationCount()) ||
		!sourceHealthHasMissingSection(missingEvidenceSectionCounts, "cleanup", "residualCounters", setupSvcLiveReplayWriteOperationCount()) {
		t.Fatalf("expected source health top-level missingEvidenceSectionCounts mirror, got %#v", missingEvidenceSectionCounts)
	}
	repairQueues := result["repairQueues"].([]any)
	if !sourceChecklistHasNextQueueCommand(repairQueues, "metadata-service", "metadataServiceDatasource", setupSvcLiveReplayOperationCount(), "incomplete") ||
		!sourceChecklistHasNextQueueCommand(repairQueues, "query-readback", "readbackTables", setupSvcLiveReplayOperationCount(), "incomplete") ||
		!sourceChecklistHasNextQueueCommand(repairQueues, "cleanup", "residualCounters", setupSvcLiveReplayWriteOperationCount(), "incomplete") {
		t.Fatalf("expected source health to mirror repair queues, got %#v", repairQueues)
	}
	for _, entry := range repairQueues {
		queue := entry.(map[string]any)
		if strings.TrimSpace(queue["artifactType"].(string)) == "" {
			t.Fatalf("expected source health repair queues to stay scoped by artifact type, got %#v", repairQueues)
		}
		if int(queue["count"].(float64)) != int(queue["sourceFiles"].(float64)) ||
			int(queue["targetFiles"].(float64)) != int(queue["sourceFiles"].(float64)) ||
			!strings.Contains(queue["command"].(string), "setup-svc-live-replay-source-checklist") {
			t.Fatalf("expected source health repair queues to expose automation-friendly count, targetFiles, and command aliases, got %#v", queue)
		}
		sourceExecutionCommand, ok := queue["sourceExecutionCommand"].(string)
		if !ok || !strings.Contains(sourceExecutionCommand, "setup-svc-live-replay-source-execution-packet") ||
			!strings.Contains(sourceExecutionCommand, "--artifact-type "+queue["artifactType"].(string)) ||
			!strings.Contains(sourceExecutionCommand, "--evidence-section "+queue["evidenceSection"].(string)) ||
			!strings.Contains(sourceExecutionCommand, "--source-readiness incomplete") {
			t.Fatalf("expected source health repair queues to expose scoped source-execution commands, got %#v", queue)
		}
		saveSourceExecutionCommand, ok := queue["saveSourceExecutionPacketCommand"].(string)
		if !ok || !strings.Contains(saveSourceExecutionCommand, sourceExecutionCommand) ||
			!strings.Contains(saveSourceExecutionCommand, " > ") ||
			!strings.Contains(saveSourceExecutionCommand, setupSvcLiveReplayRepairQueueSlug(queue["evidenceSection"].(string))) {
			t.Fatalf("expected source health repair queues to expose non-colliding source-execution save commands, got %#v", queue)
		}
	}
	operatorPacket := result["operatorPacket"].(map[string]any)
	if !strings.Contains(operatorPacket["sourceExecutionCommand"].(string), "setup-svc-live-replay-source-execution-packet") ||
		!strings.Contains(operatorPacket["sourceExecutionCommand"].(string), "--source-readiness incomplete") ||
		!strings.Contains(operatorPacket["completeSourceChecklistCommand"].(string), "--source-readiness complete") ||
		!strings.Contains(operatorPacket["completionAuditCommand"].(string), "setup-svc-live-replay-completion-audit") {
		t.Fatalf("expected source health operator packet to expose execution and audit commands, got %#v", operatorPacket)
	}
	operatorQueues := operatorPacket["repairQueues"].([]any)
	if !sourceChecklistHasNextQueueCommand(operatorQueues, "query-readback", "readbackTables", setupSvcLiveReplayOperationCount(), "incomplete") {
		t.Fatalf("expected operator packet to mirror source health repair queues, got %#v", operatorQueues)
	}
	for _, entry := range operatorQueues {
		queue := entry.(map[string]any)
		if int(queue["count"].(float64)) != int(queue["sourceFiles"].(float64)) ||
			int(queue["targetFiles"].(float64)) != int(queue["sourceFiles"].(float64)) ||
			!strings.Contains(queue["command"].(string), "setup-svc-live-replay-source-checklist") {
			t.Fatalf("expected operator packet repair queues to expose count, targetFiles, and command aliases, got %#v", queue)
		}
		if !strings.Contains(queue["saveSourceExecutionPacketCommand"].(string), "setup-svc-live-replay-source-execution-packet") {
			t.Fatalf("expected operator packet repair queues to expose source-execution save commands, got %#v", queue)
		}
	}

	stdout.Reset()
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-source-health", "--artifact-type", "query-readback", "--evidence-section", "readbackTables", "--section-status", "missing", "--source-readiness", "incomplete"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	result = map[string]any{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	totals = result["totals"].(map[string]any)
	if int(totals["sourceFiles"].(float64)) != setupSvcLiveReplayOperationCount() ||
		int(totals["artifactTypes"].(float64)) != 1 ||
		int(totals["missingSectionInstances"].(float64)) != setupSvcLiveReplayOperationCount()*6 {
		t.Fatalf("expected filtered source health to summarize query-readback missing sections, got %#v", totals)
	}
}

func TestSetupSvcLiveReplayEvidenceImportConsumesSourceExecutionItems(t *testing.T) {
	tmp := t.TempDir()
	packet := map[string]any{
		"mode":       "setup-svc-live-replay-source-execution-packet",
		"sourceRoot": "captures",
		"items": []any{map[string]any{
			"domain":       "applications",
			"operation":    "query",
			"artifactType": "metadata-service",
			"targetPath":   "outputs/setup-svc-live-replay/applications/query/metadata-service.json",
			"sourcePath":   "captures/outputs/setup-svc-live-replay/applications/query/metadata-service.json",
		}},
	}
	records := setupSvcLiveReplayEvidenceImportRecords(packet)
	if len(records) != 1 || records[0]["targetPath"] != "outputs/setup-svc-live-replay/applications/query/metadata-service.json" {
		t.Fatalf("expected source-execution items to be import records, got %#v", records)
	}
	sourcePath := setupSvcLiveReplayEvidenceImportRecordSourcePath(tmp, "captures", records[0])
	resolved := setupSvcLiveReplayEvidenceImportSourcePath(tmp, "captures", sourcePath)
	expected := filepath.Join(tmp, "captures", "outputs", "setup-svc-live-replay", "applications", "query", "metadata-service.json")
	if resolved != expected {
		t.Fatalf("expected sourceRoot not to be doubled for source-execution items, got %s want %s", resolved, expected)
	}
	body, err := json.Marshal(setupSvcLiveReplaySourceExecutionPacketResult{EvidenceSectionCount: 0})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"evidenceSectionCount":0`) {
		t.Fatalf("expected source-execution numeric totals to emit explicit zero values, got %s", string(body))
	}
}

func TestSetupSvcLiveReplaySourceValidateUsesEvidenceImportDryRun(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-workspace", "--execute", "--approval", setupSvcParityEvidenceWorkspaceApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-capture-source-workspace", "--execute", "--approval", setupSvcParityCaptureSourceWorkspaceApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-source-validate"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["mode"] != "setup-svc-live-replay-source-validate" || result["readOnly"] != true ||
		result["status"] != "no_sources" {
		t.Fatalf("expected source-validate to default to complete candidates without writes, got %#v", result)
	}
	filters := result["filters"].(map[string]any)
	if filters["sourceReadiness"] != "complete" {
		t.Fatalf("expected source-validate default filter to be complete, got %#v", filters)
	}
	totals := result["totals"].(map[string]any)
	if int(totals["sourceFiles"].(float64)) != 0 || int(totals["artifacts"].(float64)) != 0 {
		t.Fatalf("expected no complete source candidates in fresh workspace, got %#v", totals)
	}
	if int(result["sourceFiles"].(float64)) != int(totals["sourceFiles"].(float64)) ||
		int(result["artifactCount"].(float64)) != int(totals["artifacts"].(float64)) ||
		int(result["readyArtifacts"].(float64)) != int(totals["readyArtifacts"].(float64)) ||
		int(result["failedArtifacts"].(float64)) != int(totals["failedArtifacts"].(float64)) ||
		int(result["skippedDuplicateRecords"].(float64)) != int(totals["skippedDuplicateRecords"].(float64)) {
		t.Fatalf("expected source-validate top-level counters to mirror totals for empty candidates, got result=%#v totals=%#v", result, totals)
	}

	stdout.Reset()
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-source-validate", "--artifact-type", "query-readback", "--source-readiness", "incomplete"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	result = map[string]any{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked_source_validation" || result["generatedFrom"] != "setup-svc-live-replay-source-checklist" {
		t.Fatalf("expected source-validate to block incomplete sources through import dry-run, got %#v", result)
	}
	totals = result["totals"].(map[string]any)
	if int(totals["sourceFiles"].(float64)) != setupSvcLiveReplayOperationCount() ||
		int(totals["artifacts"].(float64)) != setupSvcLiveReplayOperationCount() ||
		int(totals["failedArtifacts"].(float64)) != setupSvcLiveReplayOperationCount() ||
		int(totals["readyArtifacts"].(float64)) != 0 {
		t.Fatalf("expected query-readback source validation failures for every operation, got %#v", totals)
	}
	if int(result["sourceFiles"].(float64)) != int(totals["sourceFiles"].(float64)) ||
		int(result["artifactCount"].(float64)) != int(totals["artifacts"].(float64)) ||
		int(result["readyArtifacts"].(float64)) != int(totals["readyArtifacts"].(float64)) ||
		int(result["failedArtifacts"].(float64)) != int(totals["failedArtifacts"].(float64)) ||
		int(result["skippedDuplicateRecords"].(float64)) != int(totals["skippedDuplicateRecords"].(float64)) {
		t.Fatalf("expected source-validate top-level counters to mirror totals for failed candidates, got result=%#v totals=%#v", result, totals)
	}
	importDryRun := result["importDryRun"].(map[string]any)
	if importDryRun["status"] != "blocked" ||
		int(importDryRun["failedArtifacts"].(float64)) != setupSvcLiveReplayOperationCount() ||
		int(importDryRun["writtenFiles"].(float64)) != 0 {
		t.Fatalf("expected source-validate to reuse blocked evidence-import dry-run, got %#v", importDryRun)
	}
	repairSummary := result["repairSummary"].(map[string]any)
	if int(repairSummary["failedArtifacts"].(float64)) != setupSvcLiveReplayOperationCount() ||
		int(repairSummary["repairQueueCount"].(float64)) != len(repairSummary["repairQueues"].([]any)) ||
		!evidenceImportHasIssueCount(repairSummary["issueCounts"].([]any), "queryReadbackMissingTableCoverage", setupSvcLiveReplayOperationCount()) ||
		!evidenceImportHasIssueCount(repairSummary["issueCounts"].([]any), "queryReadbackMissingRelationshipEvidence", setupSvcLiveReplayOperationCount()) {
		t.Fatalf("expected source-validate repair summary to expose strict query-readback issue families, got %#v", repairSummary)
	}
	operatorPacket := result["operatorPacket"].(map[string]any)
	if int(operatorPacket["sourceFiles"].(float64)) != int(totals["sourceFiles"].(float64)) ||
		int(operatorPacket["artifactCount"].(float64)) != int(totals["artifacts"].(float64)) ||
		int(operatorPacket["readyArtifacts"].(float64)) != int(totals["readyArtifacts"].(float64)) ||
		int(operatorPacket["failedArtifacts"].(float64)) != int(totals["failedArtifacts"].(float64)) ||
		int(operatorPacket["skippedDuplicateRecords"].(float64)) != int(totals["skippedDuplicateRecords"].(float64)) ||
		int(operatorPacket["repairQueueCount"].(float64)) != len(repairSummary["repairQueues"].([]any)) ||
		int(operatorPacket["issueKinds"].(float64)) != len(repairSummary["issueCounts"].([]any)) {
		t.Fatalf("expected source-validate operator packet to mirror validation counters, got %#v totals=%#v repairSummary=%#v", operatorPacket, totals, repairSummary)
	}
	operatorQueues := operatorPacket["repairQueues"].([]any)
	if !containsRepairQueueSourceChecklist(operatorQueues, "query-readback", "readbackTables") ||
		!containsRepairQueueSourceChecklist(operatorQueues, "query-readback", "relationshipChecks") {
		t.Fatalf("expected source-validate operator packet to mirror repair queues, got %#v", operatorPacket)
	}
	if !strings.Contains(operatorPacket["sourceChecklistCommand"].(string), "setup-svc-live-replay-source-checklist") ||
		!strings.Contains(operatorPacket["sourceHealthCommand"].(string), "setup-svc-live-replay-source-health") ||
		!strings.Contains(operatorPacket["dryRunImportCommand"].(string), "setup-svc-live-replay-evidence-import") ||
		!strings.Contains(operatorPacket["completionAuditCommand"].(string), "setup-svc-live-replay-completion-audit") {
		t.Fatalf("expected source-validate operator packet to expose pre-import runbook commands, got %#v", operatorPacket)
	}

	stdout.Reset()
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-source-validate", "--artifact-type", "metadata-service", "--evidence-section", "metadataServiceDatasource", "--source-readiness", "incomplete"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	result = map[string]any{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	repairSummary = result["repairSummary"].(map[string]any)
	repairQueues := repairSummary["repairQueues"].([]any)
	operatorPacket = result["operatorPacket"].(map[string]any)
	operatorQueues = operatorPacket["repairQueues"].([]any)
	if result["status"] != "blocked_source_validation" ||
		int(repairSummary["repairQueueCount"].(float64)) != len(repairQueues) ||
		!evidenceImportHasIssueCount(repairSummary["issueCounts"].([]any), "metadataServiceDatasourceMissingEvidence", setupSvcLiveReplayOperationCount()) ||
		!containsSectionCount(repairSummary["missingEvidenceSections"].([]any), "metadataServiceDatasource", setupSvcLiveReplayOperationCount()) ||
		!containsRepairQueueWithPositiveCount(repairQueues, "metadata-service", "metadataServiceDatasource") ||
		!containsRepairQueueSourceChecklist(repairQueues, "metadata-service", "metadataServiceDatasource") ||
		!containsRepairQueueWithPositiveCount(operatorQueues, "metadata-service", "metadataServiceDatasource") ||
		!containsRepairQueueSourceChecklist(operatorQueues, "metadata-service", "metadataServiceDatasource") ||
		int(operatorPacket["repairQueueCount"].(float64)) != len(repairQueues) {
		t.Fatalf("expected source-validate repair summary to route datasource proof failures to metadata-service/metadataServiceDatasource, got result=%#v repairSummary=%#v operatorPacket=%#v", result, repairSummary, operatorPacket)
	}
}

func TestSetupSvcLiveReplayQueryReadbackCapturePlan(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-workspace", "--execute", "--approval", setupSvcParityEvidenceWorkspaceApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-capture-source-workspace", "--execute", "--approval", setupSvcParityCaptureSourceWorkspaceApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-query-readback-capture-plan", "--source-readiness", "incomplete"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["mode"] != "setup-svc-live-replay-query-readback-capture-plan" || result["readOnly"] != true || result["generatedFrom"] != "setup-svc-live-replay-source-checklist" {
		t.Fatalf("expected read-only query-readback capture plan, got %#v", result)
	}
	totals := result["totals"].(map[string]any)
	if int(totals["queryReadbackSources"].(float64)) != setupSvcLiveReplayOperationCount() ||
		int(result["queryReadbackSources"].(float64)) != setupSvcLiveReplayOperationCount() ||
		int(result["totalQueryReadbackSources"].(float64)) != setupSvcLiveReplayOperationCount() ||
		int(result["returnedQueryReadbackSources"].(float64)) != setupSvcLiveReplayOperationCount() ||
		int(totals["totalQueryReadbackSources"].(float64)) != setupSvcLiveReplayOperationCount() ||
		int(totals["returnedQueryReadbackSources"].(float64)) != setupSvcLiveReplayOperationCount() ||
		int(totals["domainOperations"].(float64)) != setupSvcLiveReplayOperationCount() ||
		int(totals["requiredTables"].(float64)) == 0 ||
		int(totals["expectations"].(float64)) == 0 {
		t.Fatalf("expected query-readback capture totals for every operation, got %#v", totals)
	}
	requests := result["captureRequests"].([]any)
	if len(requests) != setupSvcLiveReplayOperationCount() {
		t.Fatalf("expected one capture request per operation, got %d", len(requests))
	}
	first := requests[0].(map[string]any)
	if first["artifactType"] != nil {
		t.Fatalf("capture request should not expose a top-level artifactType outside the artifact shape, got %#v", first)
	}
	shape := first["captureArtifactShape"].(map[string]any)
	if shape["status"] != "pending_capture" ||
		shape["artifactType"] != "query-readback" ||
		shape["contractVersion"] == "" ||
		len(shape["readbackTables"].([]any)) == 0 ||
		len(shape["readbackExpectationChecks"].([]any)) == 0 {
		t.Fatalf("expected pending query-readback artifact shape with required sections, got %#v", shape)
	}
	cleanCounters := shape["cleanCounters"].(map[string]any)
	if _, ok := cleanCounters["missingFields"]; !ok {
		t.Fatalf("expected artifact shape to declare clean counter keys, got %#v", cleanCounters)
	}
	commands := first["recommendedReadbackCommands"].([]any)
	if !containsStringFragment(commands, "standard-catalog") || !containsStringFragment(commands, "field-map") ||
		!strings.Contains(first["scannerCommand"].(string), "standard-catalog") {
		t.Fatalf("expected readback capture request to include scanner commands, got %#v", commands)
	}
	if !strings.Contains(first["completeWorklistCommand"].(string), "--source-readiness complete") ||
		!strings.Contains(first["dryRunImportCommand"].(string), "setup-svc-live-replay-evidence-import") {
		t.Fatalf("expected capture request to include complete worklist/import commands, got %#v", first)
	}
	operatorPacket := result["operatorPacket"].(map[string]any)
	if !strings.Contains(operatorPacket["savePlanCommand"].(string), "setup-svc-live-replay-query-readback-capture-plan") ||
		!strings.Contains(operatorPacket["recommendedBatchCommand"].(string), "--artifact-type query-readback") ||
		!strings.Contains(operatorPacket["completionAuditCommand"].(string), "setup-svc-live-replay-completion-audit") {
		t.Fatalf("expected operator packet to expose capture-plan batch commands, got %#v", operatorPacket)
	}

	stdout.Reset()
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-query-readback-capture-plan", "--domain", "objects", "--operation", "create", "--source-readiness", "incomplete"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	result = map[string]any{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	totals = result["totals"].(map[string]any)
	if int(totals["queryReadbackSources"].(float64)) != 1 || int(totals["domainOperations"].(float64)) != 1 {
		t.Fatalf("expected filtered query-readback capture plan for one operation, got %#v", totals)
	}

	stdout.Reset()
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-query-readback-capture-plan", "--source-readiness", "incomplete", "--offset", "0", "--limit", "2"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	result = map[string]any{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	totals = result["totals"].(map[string]any)
	requests = result["captureRequests"].([]any)
	if len(requests) != 2 ||
		int(result["queryReadbackSources"].(float64)) != 2 ||
		int(result["returnedQueryReadbackSources"].(float64)) != 2 ||
		int(result["totalQueryReadbackSources"].(float64)) != setupSvcLiveReplayOperationCount() ||
		int(result["omittedQueryReadbackSources"].(float64)) != setupSvcLiveReplayOperationCount()-2 ||
		int(totals["queryReadbackSources"].(float64)) != 2 ||
		int(totals["returnedQueryReadbackSources"].(float64)) != 2 ||
		int(totals["totalQueryReadbackSources"].(float64)) != setupSvcLiveReplayOperationCount() ||
		int(totals["omittedQueryReadbackSources"].(float64)) != setupSvcLiveReplayOperationCount()-2 ||
		int(totals["limit"].(float64)) != 2 {
		t.Fatalf("expected explicit paging to bound query-readback capture requests, totals=%#v requests=%d", totals, len(requests))
	}
}

func TestSetupSvcLiveReplayQueryReadbackCaptureWritesVerifiableSources(t *testing.T) {
	tmp := t.TempDir()
	var standardCalls, fieldMapCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/metadata/v1/scans/standard-catalog":
			if r.Method != http.MethodGet {
				t.Fatalf("unexpected standard-catalog method %s", r.Method)
			}
			standardCalls++
			_, _ = w.Write([]byte(`{"service":"cc-metadata-service","mode":"read-only-standard-catalog","objects":[{"apiName":"sample"}]}`))
		case "/metadata/v1/scans/field-map":
			if r.Method != http.MethodPost {
				t.Fatalf("unexpected field-map method %s", r.Method)
			}
			fieldMapCalls++
			_, _ = w.Write([]byte(`{"service":"cc-metadata-service","mode":"read-only-field-map","objects":[{"apiName":"sample","fields":[{"apiName":"name"}]}]}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"`+server.URL+`"}}}`)

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-workspace", "--execute", "--approval", setupSvcParityEvidenceWorkspaceApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-capture-source-workspace", "--artifact-type", "query-readback", "--domain", "objects", "--operation", "create", "--execute", "--approval", setupSvcParityCaptureSourceWorkspaceApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-query-readback-capture", "--domain", "objects", "--operation", "create"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "dry_run_ready" || result["readOnly"] != true ||
		int(result["passedArtifacts"].(float64)) != 1 || int(result["writtenFiles"].(float64)) != 0 {
		t.Fatalf("expected dry-run ready without writes, got %#v", result)
	}
	sourcePath := filepath.Join(tmp, "captures", "outputs", "setup-svc-live-replay", "objects", "create", "query-readback.json")
	if _, err := os.Stat(sourcePath); err != nil {
		t.Fatalf("expected existing incomplete source template after workspace prep: %v", err)
	}

	stdout.Reset()
	err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-query-readback-capture", "--domain", "objects", "--operation", "create", "--execute", "--approval", "WRONG"}, &stdout, tmp)
	if err == nil || !strings.Contains(err.Error(), setupSvcParityQueryReadbackCaptureApproval) {
		t.Fatalf("expected approval gate error, got %v", err)
	}

	stdout.Reset()
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-query-readback-capture", "--domain", "objects", "--operation", "create", "--execute", "--approval", setupSvcParityQueryReadbackCaptureApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	result = map[string]any{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "applied" || result["readOnly"] != false ||
		int(result["passedArtifacts"].(float64)) != 1 || int(result["writtenFiles"].(float64)) != 1 ||
		standardCalls == 0 || fieldMapCalls == 0 {
		t.Fatalf("expected approved capture to write one source and call scanners, got result=%#v standardCalls=%d fieldMapCalls=%d", result, standardCalls, fieldMapCalls)
	}
	artifact := readTestJSONMap(t, sourcePath)
	if artifact["status"] != "passed" || artifact["artifactType"] != "query-readback" ||
		len(artifact["readbackTables"].([]any)) == 0 ||
		len(artifact["relationshipChecks"].([]any)) == 0 ||
		len(artifact["readbackExpectationChecks"].([]any)) == 0 {
		t.Fatalf("expected complete query-readback source artifact, got %#v", artifact)
	}

	stdout.Reset()
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-source-validate", "--artifact-type", "query-readback", "--domain", "objects", "--operation", "create", "--source-readiness", "complete"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	result = map[string]any{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	totals := result["totals"].(map[string]any)
	if result["status"] != "ready_for_import_dry_run" ||
		int(totals["readyArtifacts"].(float64)) != 1 ||
		int(totals["failedArtifacts"].(float64)) != 0 {
		t.Fatalf("expected source-validate to accept captured query-readback source, got %#v", result)
	}
}

func TestSetupSvcLiveReplaySnapshotFromChangesWritesVerifiableSource(t *testing.T) {
	setReadyMetadataServiceDatasourceEnv(t, "snapshot-writes")
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:8087"}}}`)
	var objectDomain setupSvcLiveReplayDomain
	for _, domain := range setupSvcLiveReplayDomains() {
		if domain.Domain == "objects" {
			objectDomain = domain
			break
		}
	}
	changes := []any{}
	for _, table := range objectDomain.RequiredTables {
		changes = append(changes, map[string]any{
			"tableName":    table,
			"mutationType": "UPSERT",
			"targetId":     table + "_row",
			"status":       "applied",
			"after": []any{map[string]any{
				"id":     table + "_row",
				"source": "metadata-service-apply",
			}},
		})
	}
	for _, effect := range objectDomain.RuntimeEffects {
		changes = append(changes, map[string]any{
			"tableName":    effect,
			"mutationType": "SIDE_EFFECT",
			"targetId":     "obj_snapshot",
			"status":       "applied",
			"after": map[string]any{
				"effectType": effect,
				"status":     "applied",
			},
		})
	}
	packet := map[string]any{
		"manifestPath": filepath.Join(tmp, "outputs", "setup-svc-live-replay", "manifest.json"),
		"artifacts": []any{map[string]any{
			"domain":              "objects",
			"operation":           "create",
			"artifactType":        "metadata-service",
			"suggestedSourcePath": "captures/outputs/setup-svc-live-replay/objects/create/metadata-service.json",
			"path":                "outputs/setup-svc-live-replay/objects/create/metadata-service.json",
			"changes":             changes,
		}},
	}
	body, err := json.Marshal(packet)
	if err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-snapshot-from-changes", string(body)}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "dry_run_ready" ||
		int(result["passedArtifacts"].(float64)) != 1 ||
		int(result["writtenFiles"].(float64)) != 0 {
		t.Fatalf("expected dry-run ready snapshot hydration, got %#v", result)
	}

	stdout.Reset()
	err = Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-snapshot-from-changes", string(body), "--execute", "--approval", "WRONG"}, &stdout, tmp)
	if err == nil || !strings.Contains(err.Error(), setupSvcParitySnapshotFromChangesApproval) {
		t.Fatalf("expected snapshot hydration approval gate, got %v", err)
	}

	stdout.Reset()
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-snapshot-from-changes", string(body), "--execute", "--approval", setupSvcParitySnapshotFromChangesApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	result = map[string]any{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "applied" ||
		int(result["passedArtifacts"].(float64)) != 1 ||
		int(result["writtenFiles"].(float64)) != 1 {
		t.Fatalf("expected approved snapshot hydration to write one source, got %#v", result)
	}
	artifact := readTestJSONMap(t, filepath.Join(tmp, "captures", "outputs", "setup-svc-live-replay", "objects", "create", "metadata-service.json"))
	if artifact["status"] != "passed" || artifact["artifactType"] != "metadata-service" ||
		len(artifact["runtimeEffectChecks"].([]any)) != len(setupSvcLiveReplayRuntimeEffectsForOperation(objectDomain.Domain, "create")) {
		t.Fatalf("expected complete metadata-service snapshot artifact, got %#v", artifact)
	}
	snapshots := artifact["tableSnapshots"].(map[string]any)
	if len(snapshots) != len(objectDomain.RequiredTables) {
		t.Fatalf("expected all required table snapshots, got %d want %d", len(snapshots), len(objectDomain.RequiredTables))
	}
}

func TestSetupSvcLiveReplayOperationRequiredTablesStayWithinDomainContract(t *testing.T) {
	for _, domain := range setupSvcLiveReplayDomains() {
		domainTables := map[string]bool{}
		for _, table := range domain.RequiredTables {
			normalized := strings.ToLower(strings.TrimSpace(table))
			if normalized == "" {
				t.Fatalf("%s has blank domain required table in %#v", domain.Domain, domain.RequiredTables)
			}
			if domainTables[normalized] {
				t.Fatalf("%s has duplicate domain required table %s in %#v", domain.Domain, table, domain.RequiredTables)
			}
			domainTables[normalized] = true
		}
		for _, operation := range domain.Operations {
			operationTables := setupSvcLiveReplayRequiredTablesForOperation(domain, operation)
			if len(operationTables) == 0 {
				t.Fatalf("%s/%s has no operation required tables", domain.Domain, operation)
			}
			seen := map[string]bool{}
			for _, table := range operationTables {
				normalized := strings.ToLower(strings.TrimSpace(table))
				if normalized == "" {
					t.Fatalf("%s/%s has blank operation required table in %#v", domain.Domain, operation, operationTables)
				}
				if seen[normalized] {
					t.Fatalf("%s/%s has duplicate operation required table %s in %#v", domain.Domain, operation, table, operationTables)
				}
				if !domainTables[normalized] {
					t.Fatalf("%s/%s required table %s is outside domain requiredTables %#v", domain.Domain, operation, table, domain.RequiredTables)
				}
				seen[normalized] = true
			}
		}
	}
}

func TestSetupSvcLiveReplayOperationRuntimeAndReadbackContractsStayWithinDomainContract(t *testing.T) {
	for _, domain := range setupSvcLiveReplayDomains() {
		domainEffects := map[string]bool{}
		for _, effect := range domain.RuntimeEffects {
			normalized := strings.ToLower(strings.TrimSpace(effect))
			if normalized == "" {
				t.Fatalf("%s has blank domain runtime effect in %#v", domain.Domain, domain.RuntimeEffects)
			}
			if domainEffects[normalized] {
				t.Fatalf("%s has duplicate domain runtime effect %s in %#v", domain.Domain, effect, domain.RuntimeEffects)
			}
			domainEffects[normalized] = true
		}
		domainExpectations := map[string]bool{}
		for _, expectation := range domain.QueryReadbackExpectations {
			normalized := strings.ToLower(strings.TrimSpace(expectation))
			if normalized == "" {
				t.Fatalf("%s has blank domain query/readback expectation in %#v", domain.Domain, domain.QueryReadbackExpectations)
			}
			if domainExpectations[normalized] {
				t.Fatalf("%s has duplicate domain query/readback expectation %s in %#v", domain.Domain, expectation, domain.QueryReadbackExpectations)
			}
			domainExpectations[normalized] = true
		}
		for _, operation := range domain.Operations {
			operationEffects := setupSvcLiveReplayRuntimeEffectsForOperation(domain.Domain, operation)
			if strings.EqualFold(strings.TrimSpace(operation), "query") {
				if len(operationEffects) != 0 {
					t.Fatalf("%s/query should not require write runtime effects, got %#v", domain.Domain, operationEffects)
				}
			} else if domain.Domain == "global-select-lists" && operation == "delete" {
				if len(operationEffects) != 0 {
					t.Fatalf("global-select-lists/delete should not require runtime effects, got %#v", operationEffects)
				}
			} else if len(operationEffects) == 0 {
				t.Fatalf("%s/%s has no operation runtime effects", domain.Domain, operation)
			}
			seenEffects := map[string]bool{}
			for _, effect := range operationEffects {
				normalized := strings.ToLower(strings.TrimSpace(effect))
				if normalized == "" {
					t.Fatalf("%s/%s has blank operation runtime effect in %#v", domain.Domain, operation, operationEffects)
				}
				if seenEffects[normalized] {
					t.Fatalf("%s/%s has duplicate operation runtime effect %s in %#v", domain.Domain, operation, effect, operationEffects)
				}
				if !domainEffects[normalized] {
					t.Fatalf("%s/%s runtime effect %s is outside domain runtimeEffects %#v", domain.Domain, operation, effect, domain.RuntimeEffects)
				}
				seenEffects[normalized] = true
			}
			if len(domain.QueryReadbackExpectations) == 0 {
				t.Fatalf("%s/%s has no query/readback expectations", domain.Domain, operation)
			}
			for _, expectation := range domain.QueryReadbackExpectations {
				normalized := strings.ToLower(strings.TrimSpace(expectation))
				if !domainExpectations[normalized] {
					t.Fatalf("%s/%s query/readback expectation %s is outside domain expectations %#v", domain.Domain, operation, expectation, domain.QueryReadbackExpectations)
				}
			}
		}
	}
}

func TestSetupSvcLiveReplaySnapshotFromChangesKeepsZeroRowCleanupEvidence(t *testing.T) {
	setReadyMetadataServiceDatasourceEnv(t, "snapshot-cleanup")
	applicationDomain := setupSvcLiveReplayDomain{}
	for _, domain := range setupSvcLiveReplayDomains() {
		if domain.Domain == "applications" {
			applicationDomain = domain
			break
		}
	}
	requiredTables := setupSvcLiveReplayRequiredTablesForOperation(applicationDomain, "delete")
	for _, table := range requiredTables {
		if table == "tp_sys_tab" {
			t.Fatalf("application delete should not require non-owned tab definitions, got %#v", requiredTables)
		}
	}
	identityProviderDomain := setupSvcLiveReplayDomain{}
	for _, domain := range setupSvcLiveReplayDomains() {
		if domain.Domain == "identity-providers" {
			identityProviderDomain = domain
			break
		}
	}
	identityProviderDeleteTables := setupSvcLiveReplayRequiredTablesForOperation(identityProviderDomain, "delete")
	if !reflect.DeepEqual(identityProviderDeleteTables, []string{"tp_sys_idp_sps"}) {
		t.Fatalf("identity-provider delete should require only setup-svc SAML SP rows, got %#v", identityProviderDeleteTables)
	}
	idpClient := &client{projectPath: t.TempDir()}
	idpItem := map[string]any{
		"domain":       "identity-providers",
		"operation":    "delete",
		"artifactType": "metadata-service",
		"operationId":  "oper-idp-delete",
		"changes": []any{
			map[string]any{
				"tableName":    "tp_sys_idp_sps",
				"mutationType": "DELETE",
				"status":       "applied",
				"before": []any{
					map[string]any{"id": "idpsp_price", "app": "price_app"},
				},
				"after": []any{},
			},
			map[string]any{
				"tableName":    "idp-delete-cleanup",
				"mutationType": "SIDE_EFFECT",
				"status":       "applied",
				"after":        map[string]any{"effectType": "idp-delete-cleanup"},
			},
		},
	}
	idpArtifact, _, idpIssues := idpClient.setupSvcLiveReplaySnapshotArtifactFromChanges(0, idpItem,
		map[string]setupSvcLiveReplayDomain{"identity-providers": identityProviderDomain}, nil)
	if len(idpIssues) > 0 {
		t.Fatalf("expected identity-provider delete snapshot hydration without config table changes, got %#v", idpIssues)
	}
	if failures := verifySetupSvcLiveReplayEvidenceArtifact(idpClient.projectPath,
		"outputs/setup-svc-live-replay/identity-providers/delete/metadata-service.json",
		identityProviderDomain, "delete", idpArtifact); len(failures) > 0 {
		t.Fatalf("expected identity-provider delete artifact to verify with only tp_sys_idp_sps, got %#v artifact=%#v", failures, idpArtifact)
	}
	changes := []any{
		map[string]any{
			"tableName":    "tp_sys_app",
			"mutationType": "DELETE",
			"status":       "applied",
			"before":       []any{},
			"after":        []any{},
		},
		map[string]any{
			"tableName":    "tp_sys_app_tab",
			"mutationType": "DELETE",
			"status":       "applied",
			"before":       []any{},
			"after":        []any{},
		},
		map[string]any{
			"tableName":    "tp_sys_profile_infoset",
			"mutationType": "DELETE",
			"status":       "applied",
			"before":       []any{},
			"after":        []any{},
		},
		map[string]any{
			"tableName":    "tp_sys_multi_lang",
			"mutationType": "DELETE",
			"status":       "applied",
			"before":       []any{},
			"after":        []any{},
		},
	}
	snapshots := setupSvcLiveReplayTableSnapshotsFromChanges(changes, requiredTables)
	if len(snapshots) != len(requiredTables) {
		t.Fatalf("expected zero-row delete changes to preserve table evidence, got %#v required %#v", snapshots, requiredTables)
	}
	failures := setupSvcLiveReplaySnapshotTableFailures(map[string]any{"tableSnapshots": snapshots}, requiredTables, "metadataServiceSnapshot")
	if len(failures) > 0 {
		t.Fatalf("zero-row cleanup table evidence should verify, got %#v", failures)
	}
	checks := setupSvcLiveReplayRuntimeEffectChecksFromChanges(changes, setupSvcLiveReplayRuntimeEffectsForOperation(applicationDomain.Domain, "delete"))
	if len(checks) != 1 || checks[0]["status"] != "passed" {
		t.Fatalf("expected cleanup runtime effect to be inferred from delete changes, got %#v", checks)
	}
	domainsByName := map[string]setupSvcLiveReplayDomain{}
	for _, domain := range setupSvcLiveReplayDomains() {
		domainsByName[domain.Domain] = domain
	}
	deleteTableExpectations := map[string][]string{
		"dashboards":          {"tp_sys_dashboard", "tp_sys_dashboard_report", "tp_sys_dashboard_condition", "tp_sys_recent_items", "tp_sys_dashboard_snapshot", "tp_sys_snapshot_refress"},
		"global-select-lists": {"tp_sys_global_select"},
		"record-types":        {"tp_sys_recordtype", "tp_sys_profile_infoset", "tp_sys_profile_layout", "tp_sys_field_dependency", "tp_sys_multi_lang"},
		"reports":             {"tp_sys_report", "tp_sys_condition", "tp_sys_report_expression", "tp_sys_report_fieldname", "tp_sys_report_object", "tp_sys_report_object_detail", "tp_sys_reportgather", "tp_sys_reportgroup", "tp_sys_recent_items"},
		"roles":               {"tp_sys_role", "tp_sys_group"},
	}
	reportsDomain := domainsByName["reports"]
	for _, operation := range []string{"folder-create", "folder-update", "folder-delete"} {
		actualTables := setupSvcLiveReplayRequiredTablesForOperation(reportsDomain, operation)
		if !reflect.DeepEqual(actualTables, []string{"tp_sys_folder"}) {
			t.Fatalf("reports/%s should require only folder rows, got %#v", operation, actualTables)
		}
	}
	for domainName, expectedTables := range deleteTableExpectations {
		domain := domainsByName[domainName]
		actualTables := setupSvcLiveReplayRequiredTablesForOperation(domain, "delete")
		if !reflect.DeepEqual(actualTables, expectedTables) {
			t.Fatalf("%s delete required tables mismatch\nexpected=%#v\nactual=%#v", domainName, expectedTables, actualTables)
		}
		var deleteChanges []any
		for _, table := range expectedTables {
			mutation := "DELETE"
			before := []any{map[string]any{"id": table + "_row"}}
			after := []any{}
			if domainName == "global-select-lists" {
				mutation = "UPDATE"
				before = []any{map[string]any{"id": table + "_row", "isdeleted": "0"}}
				after = []any{map[string]any{"id": table + "_row", "isdeleted": "1"}}
			}
			deleteChanges = append(deleteChanges, map[string]any{
				"tableName":    table,
				"mutationType": mutation,
				"status":       "applied",
				"before":       before,
				"after":        after,
			})
		}
		item := map[string]any{
			"domain":       domainName,
			"operation":    "delete",
			"artifactType": "metadata-service",
			"operationId":  "oper-" + domainName + "-delete",
			"changes":      deleteChanges,
		}
		artifact, _, issues := idpClient.setupSvcLiveReplaySnapshotArtifactFromChanges(0, item,
			map[string]setupSvcLiveReplayDomain{domainName: domain}, nil)
		if len(issues) > 0 {
			t.Fatalf("expected %s delete snapshot hydration without non-delete domain tables, got %#v", domainName, issues)
		}
		if failures := verifySetupSvcLiveReplayEvidenceArtifact(idpClient.projectPath,
			"outputs/setup-svc-live-replay/"+domainName+"/delete/metadata-service.json",
			domain, "delete", artifact); len(failures) > 0 {
			t.Fatalf("expected %s delete artifact to verify with operation-scoped tables, got %#v artifact=%#v", domainName, failures, artifact)
		}
	}
}

func TestSetupSvcLiveReplaySnapshotFromChangesFetchesOperationChanges(t *testing.T) {
	setReadyMetadataServiceDatasourceEnv(t, "snapshot-fetch")
	tmp := t.TempDir()
	var objectDomain setupSvcLiveReplayDomain
	for _, domain := range setupSvcLiveReplayDomains() {
		if domain.Domain == "objects" {
			objectDomain = domain
			break
		}
	}
	changes := []any{}
	for _, table := range objectDomain.RequiredTables {
		changes = append(changes, map[string]any{
			"tableName":    table,
			"mutationType": "UPSERT",
			"status":       "applied",
			"after": []any{map[string]any{
				"id":     table + "_row",
				"source": "metadata-service-changes-endpoint",
			}},
		})
	}
	for _, effect := range objectDomain.RuntimeEffects {
		changes = append(changes, map[string]any{
			"tableName":    effect,
			"mutationType": "SIDE_EFFECT",
			"targetId":     "obj_snapshot",
			"status":       "applied",
			"after": map[string]any{
				"effectType": effect,
			},
		})
	}
	var changesCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metadata/v1/operations/op_snapshot/changes" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		changesCalls++
		body, err := json.Marshal(changes)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write(body)
	}))
	defer server.Close()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"`+server.URL+`"}}}`)
	packet := map[string]any{
		"artifacts": []any{map[string]any{
			"domain":              "objects",
			"operation":           "create",
			"artifactType":        "metadata-service",
			"operationId":         "op_snapshot",
			"suggestedSourcePath": "captures/outputs/setup-svc-live-replay/objects/create/metadata-service.json",
			"path":                "outputs/setup-svc-live-replay/objects/create/metadata-service.json",
		}},
	}
	body, err := json.Marshal(packet)
	if err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-snapshot-from-changes", string(body), "--execute", "--approval", setupSvcParitySnapshotFromChangesApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "applied" || int(result["passedArtifacts"].(float64)) != 1 ||
		int(result["writtenFiles"].(float64)) != 1 || changesCalls != 1 {
		t.Fatalf("expected operationId fetch to hydrate one source, result=%#v changesCalls=%d", result, changesCalls)
	}
	artifact := readTestJSONMap(t, filepath.Join(tmp, "captures", "outputs", "setup-svc-live-replay", "objects", "create", "metadata-service.json"))
	if artifact["operationId"] != "op_snapshot" || len(artifact["changes"].([]any)) != len(changes) {
		t.Fatalf("expected artifact to preserve fetched changes and operation id, got %#v", artifact)
	}
}

func TestSetupSvcLiveReplaySnapshotFromChangesUsesOperationResultMap(t *testing.T) {
	setReadyMetadataServiceDatasourceEnv(t, "snapshot-map")
	tmp := t.TempDir()
	var objectDomain setupSvcLiveReplayDomain
	for _, domain := range setupSvcLiveReplayDomains() {
		if domain.Domain == "objects" {
			objectDomain = domain
			break
		}
	}
	changes := setupSvcLiveReplayTestChangesForDomain(objectDomain)
	var changesCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metadata/v1/operations/op_from_map/changes" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		changesCalls++
		body, err := json.Marshal(map[string]any{"changes": changes})
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write(body)
	}))
	defer server.Close()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"`+server.URL+`"}}}`)
	packet := map[string]any{
		"operationResults": []any{map[string]any{
			"domain":       "objects",
			"operation":    "create",
			"artifactType": "metadata-service",
			"operationId":  "op_from_map",
		}},
		"artifactReplacementRecords": []any{map[string]any{
			"domain":              "objects",
			"operation":           "create",
			"artifactType":        "metadata-service",
			"suggestedSourcePath": "captures/outputs/setup-svc-live-replay/objects/create/metadata-service.json",
			"path":                "outputs/setup-svc-live-replay/objects/create/metadata-service.json",
		}},
	}
	body, err := json.Marshal(packet)
	if err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-snapshot-from-changes", string(body), "--execute", "--approval", setupSvcParitySnapshotFromChangesApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "applied" || int(result["writtenFiles"].(float64)) != 1 || changesCalls != 1 {
		t.Fatalf("expected operationResults map to hydrate source, result=%#v changesCalls=%d", result, changesCalls)
	}
	artifact := readTestJSONMap(t, filepath.Join(tmp, "captures", "outputs", "setup-svc-live-replay", "objects", "create", "metadata-service.json"))
	if artifact["operationId"] != "op_from_map" {
		t.Fatalf("expected mapped operation id to be preserved, got %#v", artifact)
	}
}

func TestSetupSvcLiveReplaySnapshotFromChangesUsesSourceOperationID(t *testing.T) {
	setReadyMetadataServiceDatasourceEnv(t, "snapshot-source")
	tmp := t.TempDir()
	var objectDomain setupSvcLiveReplayDomain
	for _, domain := range setupSvcLiveReplayDomains() {
		if domain.Domain == "objects" {
			objectDomain = domain
			break
		}
	}
	changes := setupSvcLiveReplayTestChangesForDomain(objectDomain)
	var changesCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metadata/v1/operations/op_from_source/changes" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		changesCalls++
		body, err := json.Marshal(changes)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write(body)
	}))
	defer server.Close()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"`+server.URL+`"}}}`)
	writeTestFile(t, filepath.Join(tmp, "captures", "outputs", "setup-svc-live-replay", "objects", "create", "metadata-service.json"), `{
  "domain": "objects",
  "operation": "create",
  "artifactType": "metadata-service",
  "operationId": "op_from_source"
}`)
	packet := map[string]any{
		"artifactReplacementRecords": []any{map[string]any{
			"domain":              "objects",
			"operation":           "create",
			"artifactType":        "metadata-service",
			"suggestedSourcePath": "captures/outputs/setup-svc-live-replay/objects/create/metadata-service.json",
			"path":                "outputs/setup-svc-live-replay/objects/create/metadata-service.json",
		}},
	}
	body, err := json.Marshal(packet)
	if err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-snapshot-from-changes", string(body), "--execute", "--approval", setupSvcParitySnapshotFromChangesApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "applied" || int(result["writtenFiles"].(float64)) != 1 || changesCalls != 1 {
		t.Fatalf("expected source operationId to hydrate source, result=%#v changesCalls=%d", result, changesCalls)
	}
	artifact := readTestJSONMap(t, filepath.Join(tmp, "captures", "outputs", "setup-svc-live-replay", "objects", "create", "metadata-service.json"))
	if artifact["operationId"] != "op_from_source" {
		t.Fatalf("expected source operation id to be preserved, got %#v", artifact)
	}
}

func TestSetupSvcLiveReplaySnapshotFromChangesUsesOperationResultFile(t *testing.T) {
	setReadyMetadataServiceDatasourceEnv(t, "snapshot-file")
	tmp := t.TempDir()
	var objectDomain setupSvcLiveReplayDomain
	for _, domain := range setupSvcLiveReplayDomains() {
		if domain.Domain == "objects" {
			objectDomain = domain
			break
		}
	}
	changes := setupSvcLiveReplayTestChangesForDomain(objectDomain)
	var changesCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metadata/v1/operations/op_from_file/changes" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		changesCalls++
		body, err := json.Marshal(map[string]any{"changes": changes})
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write(body)
	}))
	defer server.Close()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"`+server.URL+`"}}}`)
	resultPath := filepath.Join(tmp, "outputs", "setup-svc-live-replay", "apply-results", "objects-create.json")
	writeTestFile(t, resultPath, `{
  "domain": "objects",
  "operation": "create",
  "artifactType": "metadata-service",
  "operationId": "op_from_file"
}`)
	packet := map[string]any{
		"operationResultsFile": resultPath,
		"artifactReplacementRecords": []any{map[string]any{
			"domain":              "objects",
			"operation":           "create",
			"artifactType":        "metadata-service",
			"suggestedSourcePath": "captures/outputs/setup-svc-live-replay/objects/create/metadata-service.json",
			"path":                "outputs/setup-svc-live-replay/objects/create/metadata-service.json",
		}},
	}
	body, err := json.Marshal(packet)
	if err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-snapshot-from-changes", string(body), "--execute", "--approval", setupSvcParitySnapshotFromChangesApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "applied" || int(result["writtenFiles"].(float64)) != 1 || changesCalls != 1 {
		t.Fatalf("expected operation result file to hydrate source, result=%#v changesCalls=%d", result, changesCalls)
	}
	artifact := readTestJSONMap(t, filepath.Join(tmp, "captures", "outputs", "setup-svc-live-replay", "objects", "create", "metadata-service.json"))
	if artifact["operationId"] != "op_from_file" {
		t.Fatalf("expected file operation id to be preserved, got %#v", artifact)
	}
}

func TestSetupSvcLiveReplayMetadataServiceApplyCaptureWritesOperationResult(t *testing.T) {
	t.Setenv("MDS_RUNTIME_MODE", "self-hosted")
	t.Setenv("MDS_SERVER_PORT", "18087")
	t.Setenv("MDS_JDBC_URL", "jdbc:mysql://apply-capture-db-host:3306/apply_capture_metadata")
	t.Setenv("MDS_DB_USERNAME", "apply-capture-user")
	t.Setenv("MDS_DB_PASSWORD", "apply-capture-password")
	t.Setenv("MDS_DB_DRIVER", "com.mysql.cj.jdbc.Driver")
	tmp := t.TempDir()
	var applyCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/metadata/v1/plans/plan_capture:apply" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		applyCalls++
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["verifyAfterApply"] != true {
			t.Fatalf("expected verifyAfterApply apply request, got %#v", body)
		}
		_, _ = w.Write([]byte(`{"operationId":"op_capture","planId":"plan_capture","status":"APPLIED"}`))
	}))
	defer server.Close()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"`+server.URL+`"}}}`)
	packet := map[string]any{
		"applyResultsDir": "outputs/setup-svc-live-replay/apply-results-test",
		"artifactReplacementRecords": []any{map[string]any{
			"domain":       "objects",
			"operation":    "create",
			"artifactType": "metadata-service",
			"planId":       "plan_capture",
			"path":         "outputs/setup-svc-live-replay/objects/create/metadata-service.json",
		}},
	}
	body, err := json.Marshal(packet)
	if err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-metadata-service-apply-capture", string(body)}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "dry_run_ready" || int(result["writtenFiles"].(float64)) != 0 || applyCalls != 0 {
		t.Fatalf("expected dry-run to avoid apply calls, result=%#v calls=%d", result, applyCalls)
	}
	datasource := result["metadataServiceDatasource"].(map[string]any)
	if datasource["status"] != "ready" || datasource["readyForRealDatasource"] != true || datasource["jdbcUrlSource"] != "env:MDS_JDBC_URL" {
		t.Fatalf("expected apply-capture datasource readiness, got %#v", datasource)
	}
	if strings.Contains(stdout.String(), "apply-capture-db-host") ||
		strings.Contains(stdout.String(), "apply-capture-user") ||
		strings.Contains(stdout.String(), "apply-capture-password") {
		t.Fatalf("apply-capture leaked datasource secret values: %s", stdout.String())
	}
	stdout.Reset()
	err = Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-metadata-service-apply-capture", string(body), "--execute", "--approval", "WRONG"}, &stdout, tmp)
	if err == nil || !strings.Contains(err.Error(), setupSvcParityMetadataServiceApplyCaptureApproval) {
		t.Fatalf("expected apply-capture approval gate, got %v", err)
	}
	stdout.Reset()
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-metadata-service-apply-capture", string(body), "--execute", "--approval", setupSvcParityMetadataServiceApplyCaptureApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	result = map[string]any{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "applied" || int(result["writtenFiles"].(float64)) != 1 || applyCalls != 1 {
		t.Fatalf("expected approved apply capture to write result, result=%#v calls=%d", result, applyCalls)
	}
	artifacts := result["artifacts"].([]any)
	resultPath := artifacts[0].(map[string]any)["resultPath"].(string)
	captured := readTestJSONMap(t, filepath.Join(tmp, resultPath))
	if captured["operationId"] != "op_capture" || captured["domain"] != "objects" || captured["operation"] != "create" || captured["artifactType"] != "metadata-service" {
		t.Fatalf("expected captured apply result to include replay identity, got %#v", captured)
	}
	capturedDatasource := captured["metadataServiceDatasource"].(map[string]any)
	if capturedDatasource["readyForRealDatasource"] != true || capturedDatasource["status"] != "ready" {
		t.Fatalf("expected captured apply result to include ready datasource proof, got %#v", capturedDatasource)
	}
}

func TestSetupSvcLiveReplayMetadataServiceApplyCaptureBuildsReplayPlanRequest(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:8087"}}}`)
	packet := map[string]any{
		"artifacts": []any{
			map[string]any{
				"domain":       "objects",
				"operation":    "create",
				"artifactType": "metadata-service",
				"path":         "outputs/setup-svc-live-replay/objects/create/metadata-service.json",
			},
			map[string]any{
				"domain":       "objects",
				"operation":    "query",
				"artifactType": "metadata-service",
				"path":         "outputs/setup-svc-live-replay/objects/query/metadata-service.json",
			},
		},
	}
	body, err := json.Marshal(packet)
	if err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-metadata-service-apply-capture", string(body)}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "dry_run_ready" || int(result["passedArtifacts"].(float64)) != 1 || int(result["failedArtifacts"].(float64)) != 0 {
		t.Fatalf("expected generated create planRequest and skipped query, got %#v", result)
	}
	artifacts := result["artifacts"].([]any)
	if artifacts[0].(map[string]any)["status"] != "ready" || artifacts[1].(map[string]any)["status"] != "skipped" {
		t.Fatalf("expected ready create and skipped query, got %#v", artifacts)
	}
}

func TestSetupSvcLiveReplayMetadataServiceApplyCaptureExecuteBlocksMissingDatasource(t *testing.T) {
	t.Setenv("MDS_JDBC_URL", "")
	t.Setenv("MDS_DB_USERNAME", "")
	t.Setenv("MDS_DB_PASSWORD", "")
	t.Setenv("MDS_DB_DRIVER", "")
	tmp := t.TempDir()
	var planCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		planCalls++
		t.Fatalf("execute must not call MetadataService when real datasource is missing: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"`+server.URL+`"}}}`)
	packet := map[string]any{
		"artifacts": []any{map[string]any{
			"domain":       "objects",
			"operation":    "create",
			"artifactType": "metadata-service",
			"path":         "outputs/setup-svc-live-replay/objects/create/metadata-service.json",
		}},
	}
	body, err := json.Marshal(packet)
	if err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-metadata-service-apply-capture", string(body), "--execute", "--approval", setupSvcParityMetadataServiceApplyCaptureApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked_metadata_service_datasource" || result["approved"] != true || planCalls != 0 {
		t.Fatalf("expected approved apply capture to block before service calls, result=%#v planCalls=%d", result, planCalls)
	}
	if !containsStringFragment(result["blockingIssues"].([]any), "metadataServiceDatasource: missing MDS_JDBC_URL") ||
		!containsStringFragment(result["blockingIssues"].([]any), "metadataServiceDatasource: default H2 datasource is not valid live replay evidence") {
		t.Fatalf("expected datasource blocking issues, got %#v", result["blockingIssues"])
	}
}

func TestSetupSvcLiveReplayMetadataServiceQueryScanCaptureWritesVerifiableSources(t *testing.T) {
	t.Setenv("MDS_JDBC_URL", "")
	t.Setenv("MDS_DB_USERNAME", "")
	t.Setenv("MDS_DB_PASSWORD", "")
	t.Setenv("MDS_DB_DRIVER", "")
	tmp := t.TempDir()
	var standardCalls, fieldMapCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/metadata/v1/scans/standard-catalog":
			if r.Method != http.MethodGet {
				t.Fatalf("unexpected standard-catalog method %s", r.Method)
			}
			standardCalls++
			_, _ = w.Write([]byte(`{"service":"cc-metadata-service","mode":"read-only-standard-catalog","objects":[{"apiName":"sample"}]}`))
		case "/metadata/v1/scans/field-map":
			if r.Method != http.MethodPost {
				t.Fatalf("unexpected field-map method %s", r.Method)
			}
			fieldMapCalls++
			_, _ = w.Write([]byte(`{"service":"cc-metadata-service","mode":"read-only-field-map","objects":[{"apiName":"sample","fields":[{"apiName":"name"}]}]}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"`+server.URL+`"}}}`)
	packet := map[string]any{
		"artifactReplacementRecords": []any{map[string]any{
			"domain":       "objects",
			"operation":    "query",
			"artifactType": "metadata-service",
			"path":         "outputs/setup-svc-live-replay/objects/query/metadata-service.json",
			"sourcePath":   "captures/outputs/setup-svc-live-replay/objects/query/metadata-service.json",
			"captureTask": map[string]any{
				"captureMode": "msapi_scan_snapshot_capture",
				"scanCommand": "cloudcc scan msapi /tmp/project standard-catalog",
				"scanRequest": map[string]any{"domain": "objects", "operation": "query"},
			},
		}},
	}
	body, err := json.Marshal(packet)
	if err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-metadata-service-query-scan-capture", string(body)}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" ||
		int(result["failedArtifacts"].(float64)) != 1 ||
		int(result["writtenFiles"].(float64)) != 0 ||
		standardCalls != 1 || fieldMapCalls != 1 {
		t.Fatalf("expected missing datasource proof to block dry-run scan capture evidence, result=%#v standardCalls=%d fieldMapCalls=%d", result, standardCalls, fieldMapCalls)
	}
	datasource := result["metadataServiceDatasource"].(map[string]any)
	if datasource["status"] != "missing_real_datasource" || datasource["readyForRealDatasource"] != false {
		t.Fatalf("expected query-scan datasource readiness to expose missing datasource, got %#v", datasource)
	}
	missing := datasource["missing"].([]any)
	if !containsAnyString(missing, "MDS_JDBC_URL") ||
		!containsAnyString(missing, "MDS_DB_USERNAME") ||
		!containsAnyString(missing, "MDS_DB_PASSWORD") ||
		!containsAnyString(missing, "MDS_DB_DRIVER") {
		t.Fatalf("expected missing datasource variables, got %#v", missing)
	}
	stdout.Reset()
	err = Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-metadata-service-query-scan-capture", string(body), "--execute", "--approval", "WRONG"}, &stdout, tmp)
	if err == nil || !strings.Contains(err.Error(), setupSvcParityMetadataServiceQueryScanCaptureApproval) {
		t.Fatalf("expected query scan approval gate, got %v", err)
	}
	stdout.Reset()
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-metadata-service-query-scan-capture", string(body), "--execute", "--approval", setupSvcParityMetadataServiceQueryScanCaptureApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	result = map[string]any{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked_metadata_service_datasource" ||
		int(result["writtenFiles"].(float64)) != 0 ||
		standardCalls != 1 || fieldMapCalls != 1 {
		t.Fatalf("expected missing datasource to block approved query scan before service calls, result=%#v standardCalls=%d fieldMapCalls=%d", result, standardCalls, fieldMapCalls)
	}
	if !containsStringFragment(result["blockingIssues"].([]any), "metadataServiceDatasource: missing MDS_JDBC_URL") {
		t.Fatalf("expected datasource blocking issue, got %#v", result["blockingIssues"])
	}
	t.Setenv("MDS_JDBC_URL", "jdbc:mysql://query-scan-db-host:3306/query_scan_metadata")
	t.Setenv("MDS_DB_USERNAME", "query-scan-user")
	t.Setenv("MDS_DB_PASSWORD", "query-scan-password")
	t.Setenv("MDS_DB_DRIVER", "com.mysql.cj.jdbc.Driver")
	stdout.Reset()
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-metadata-service-query-scan-capture", string(body)}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	result = map[string]any{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "dry_run_ready" || int(result["passedArtifacts"].(float64)) != 1 {
		t.Fatalf("expected ready datasource dry-run scan capture, got %#v", result)
	}
	stdout.Reset()
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-metadata-service-query-scan-capture", string(body), "--execute", "--approval", setupSvcParityMetadataServiceQueryScanCaptureApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	result = map[string]any{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "applied" || int(result["writtenFiles"].(float64)) != 1 || int(result["passedArtifacts"].(float64)) != 1 {
		t.Fatalf("expected approved query scan capture to write one source, got %#v", result)
	}
	artifact := readTestJSONMap(t, filepath.Join(tmp, "captures", "outputs", "setup-svc-live-replay", "objects", "query", "metadata-service.json"))
	if artifact["status"] != "passed" || artifact["artifactType"] != "metadata-service" || artifact["sourceKind"] != "metadata-service-query-scan" {
		t.Fatalf("expected metadata-service query scan artifact, got %#v", artifact)
	}
	artifactDatasource := artifact["metadataServiceDatasource"].(map[string]any)
	if artifactDatasource["readyForRealDatasource"] != true || artifactDatasource["status"] != "ready" {
		t.Fatalf("expected query scan artifact to include ready datasource proof, got %#v", artifactDatasource)
	}
	if len(artifact["tableSnapshots"].(map[string]any)) == 0 || len(artifact["runtimeEffectChecks"].([]any)) != 0 || int(artifact["missingRuntimeEffects"].(float64)) != 0 {
		t.Fatalf("expected query scan artifact to include table snapshots and no write runtime effects, got %#v", artifact)
	}
}

func TestSetupSvcLiveReplayMetadataServiceApplyCaptureCreatesPlanFromReplayRequest(t *testing.T) {
	t.Setenv("MDS_JDBC_URL", "jdbc:mysql://apply-plan-db-host:3306/apply_plan_metadata")
	t.Setenv("MDS_DB_USERNAME", "apply-plan-user")
	t.Setenv("MDS_DB_PASSWORD", "apply-plan-password")
	t.Setenv("MDS_DB_DRIVER", "com.mysql.cj.jdbc.Driver")
	tmp := t.TempDir()
	var planCalls int
	var applyCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/metadata/v1/plans":
			planCalls++
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["domain"] != "objects" || body["operation"] != "create" {
				t.Fatalf("expected generated objects/create planRequest, got %#v", body)
			}
			spec := body["spec"].(map[string]any)
			if spec["apiName"] != "cc_replay_object" || spec["label"] != "回放对象" {
				t.Fatalf("expected replay-safe object spec, got %#v", spec)
			}
			_, _ = w.Write([]byte(`{"planId":"plan_generated"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/metadata/v1/plans/plan_generated:apply":
			applyCalls++
			_, _ = w.Write([]byte(`{"operationId":"op_generated","status":"APPLIED"}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"`+server.URL+`"}}}`)
	packet := map[string]any{
		"applyResultsDir": "outputs/setup-svc-live-replay/apply-results-generated",
		"artifacts": []any{map[string]any{
			"domain":       "objects",
			"operation":    "create",
			"artifactType": "metadata-service",
			"path":         "outputs/setup-svc-live-replay/objects/create/metadata-service.json",
		}},
	}
	body, err := json.Marshal(packet)
	if err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-metadata-service-apply-capture", string(body), "--execute", "--approval", setupSvcParityMetadataServiceApplyCaptureApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "applied" || int(result["writtenFiles"].(float64)) != 1 || planCalls != 1 || applyCalls != 1 {
		t.Fatalf("expected generated plan/apply capture, result=%#v planCalls=%d applyCalls=%d", result, planCalls, applyCalls)
	}
}

func TestSetupSvcLiveReplayMetadataServiceApplyCaptureApprovesPhysicalPurge(t *testing.T) {
	t.Setenv("MDS_JDBC_URL", "jdbc:mysql://physical-purge-db-host:3306/physical_purge_metadata")
	t.Setenv("MDS_DB_USERNAME", "physical-purge-user")
	t.Setenv("MDS_DB_PASSWORD", "physical-purge-password")
	t.Setenv("MDS_DB_DRIVER", "com.mysql.cj.jdbc.Driver")
	tmp := t.TempDir()
	var applyCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/metadata/v1/plans":
			_, _ = w.Write([]byte(`{"planId":"plan_physical"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/metadata/v1/plans/plan_physical:apply":
			applyCalls++
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["approval"] != "CLOUDCC_OBJECT_PHYSICAL_DELETE_APPROVED" {
				t.Fatalf("expected physical purge approval in apply request, got %#v", body)
			}
			_, _ = w.Write([]byte(`{"operationId":"op_physical","status":"APPLIED"}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"`+server.URL+`"}}}`)
	packet := map[string]any{
		"artifacts": []any{map[string]any{
			"domain":       "objects",
			"operation":    "physical-purge",
			"artifactType": "metadata-service",
			"path":         "outputs/setup-svc-live-replay/objects/physical-purge/metadata-service.json",
		}},
	}
	body, err := json.Marshal(packet)
	if err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-metadata-service-apply-capture", string(body), "--execute", "--approval", setupSvcParityMetadataServiceApplyCaptureApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	if applyCalls != 1 {
		t.Fatalf("expected one physical purge apply call, got %d", applyCalls)
	}
}

func TestSetupSvcLiveReplayMetadataServiceReplayPlanRequestsCoverAllWriteOperations(t *testing.T) {
	for _, domain := range setupSvcLiveReplayDomains() {
		for _, operation := range domain.Operations {
			if operation == "query" {
				if scanRequest := setupSvcLiveReplayMetadataServiceScanRequest(domain.Domain); len(scanRequest) == 0 {
					t.Fatalf("expected scan request for %s/query", domain.Domain)
				}
				continue
			}
			request := setupSvcLiveReplayMetadataServicePlanRequest(domain.Domain, operation)
			if len(request) == 0 {
				t.Fatalf("missing replay planRequest for %s/%s", domain.Domain, operation)
			}
			if request["domain"] != domain.Domain || request["operation"] != operation {
				t.Fatalf("unexpected planRequest identity for %s/%s: %#v", domain.Domain, operation, request)
			}
			if spec, ok := request["spec"].(map[string]any); !ok || len(spec) == 0 {
				t.Fatalf("missing spec for %s/%s: %#v", domain.Domain, operation, request)
			} else if operation == "create" || operation == "update" {
				switch domain.Domain {
				case "approval-processes", "sharing-rules":
					if len(testAnySlice(spec["conditions"])) == 0 {
						t.Fatalf("expected %s/%s replay spec to include conditions for tp_sys_condition evidence: %#v", domain.Domain, operation, spec)
					}
				case "reports":
					if len(testAnySlice(spec["conditions"])) == 0 || len(testAnySlice(spec["gathers"])) == 0 || len(testAnySlice(spec["groups"])) == 0 {
						t.Fatalf("expected reports/%s replay spec to include conditions, gathers and groups: %#v", operation, spec)
					}
				}
			} else if domain.Domain == "reports" && strings.HasPrefix(operation, "folder-") {
				if spec["id"] != "folrptccr" {
					t.Fatalf("expected reports/%s replay spec to target compacted report folder, got %#v", operation, spec)
				}
				if operation != "folder-delete" && spec["folderType"] != "report" {
					t.Fatalf("expected reports/%s replay spec to carry report folder type, got %#v", operation, spec)
				}
			}
			context := request["context"].(map[string]any)
			if context["actorId"] != "cloudcc" {
				t.Fatalf("expected replay plan request to carry safe actorId for %s/%s: %#v", domain.Domain, operation, context)
			}
			assertReplayPlanRequestIDsAreShort(t, domain.Domain, operation, request)
		}
	}
}

func testAnySlice(value any) []any {
	switch typed := value.(type) {
	case []any:
		return typed
	default:
		return nil
	}
}

func assertReplayPlanRequestIDsAreShort(t *testing.T, domain string, operation string, value any) {
	t.Helper()
	var walk func(path string, current any)
	walk = func(path string, current any) {
		switch typed := current.(type) {
		case map[string]any:
			for key, nested := range typed {
				walk(path+"."+key, nested)
			}
		case []any:
			for i, nested := range typed {
				walk(fmt.Sprintf("%s[%d]", path, i), nested)
			}
		case string:
			last := path
			if dot := strings.LastIndex(last, "."); dot >= 0 {
				last = last[dot+1:]
			}
			lower := strings.ToLower(last)
			if (lower == "id" || strings.HasSuffix(lower, "id") || strings.HasSuffix(lower, "ids")) &&
				strings.Contains(typed, "ccr") &&
				len(typed) > 18 {
				t.Fatalf("replay fixture ID too long for %s/%s at %s: %q (%d)", domain, operation, path, typed, len(typed))
			}
		}
	}
	walk("request", value)
}

func TestSetupSvcLiveReplaySnapshotFromChangesBlocksMissingChanges(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:8087"}}}`)
	packet := map[string]any{
		"artifacts": []any{map[string]any{
			"domain":       "objects",
			"operation":    "create",
			"artifactType": "metadata-service",
			"path":         "outputs/setup-svc-live-replay/objects/create/metadata-service.json",
		}},
	}
	body, err := json.Marshal(packet)
	if err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-snapshot-from-changes", string(body)}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" || int(result["failedArtifacts"].(float64)) != 1 {
		t.Fatalf("expected missing changes to block snapshot hydration, got %#v", result)
	}
	if !containsStringFragment(result["blockingIssues"].([]any), "missingOperationChanges") {
		t.Fatalf("expected missingOperationChanges issue, got %#v", result["blockingIssues"])
	}
}

func setupSvcLiveReplayTestChangesForDomain(domain setupSvcLiveReplayDomain) []any {
	changes := []any{}
	for _, table := range domain.RequiredTables {
		changes = append(changes, map[string]any{
			"tableName":    table,
			"mutationType": "UPSERT",
			"status":       "applied",
			"after": []any{map[string]any{
				"id":     table + "_row",
				"source": "metadata-service-changes-endpoint",
			}},
		})
	}
	for _, effect := range domain.RuntimeEffects {
		changes = append(changes, map[string]any{
			"tableName":    effect,
			"mutationType": "SIDE_EFFECT",
			"targetId":     "obj_snapshot",
			"status":       "applied",
			"after": map[string]any{
				"effectType": effect,
			},
		})
	}
	return changes
}

func TestSetupSvcLiveReplayCapturePlanListsUniqueArtifactsAndSourceStatus(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-capture-plan"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["mode"] != "setup-svc-live-replay-capture-plan" || result["readOnly"] != true || result["status"] != "missing" {
		t.Fatalf("expected read-only missing capture plan, got %#v", result)
	}
	if result["sourceRoot"] != "captures" || result["captureRoot"] != filepath.Join(tmp, "captures") {
		t.Fatalf("expected capture plan to expose mirrored capture root, got sourceRoot=%#v captureRoot=%#v", result["sourceRoot"], result["captureRoot"])
	}
	totals := result["totals"].(map[string]any)
	if int(totals["artifactFiles"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount() ||
		int(totals["filteredArtifactFiles"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount() ||
		int(totals["sourceFilesPresent"].(float64)) != 0 ||
		int(totals["sourceFilesMissing"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount() ||
		int(totals["sourceFilesComplete"].(float64)) != 0 ||
		int(totals["sourceFilesIncomplete"].(float64)) != 0 ||
		int(totals["queryReadbackArtifacts"].(float64)) != setupSvcLiveReplayOperationCount() ||
		int(totals["setupSvcArtifacts"].(float64)) != setupSvcLiveReplayOperationCount() ||
		int(totals["metadataServiceArtifacts"].(float64)) != setupSvcLiveReplayOperationCount() {
		t.Fatalf("expected capture plan totals to describe unique canonical artifacts, got %#v", totals)
	}
	if int(result["totalNextArtifacts"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount() ||
		int(result["nextArtifactOffset"].(float64)) != 0 ||
		int(result["nextArtifactLimit"].(float64)) != 25 ||
		int(result["omittedNextArtifacts"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount()-25 {
		t.Fatalf("expected default capture plan page to include 25 artifacts, got %#v", result)
	}
	artifacts := result["artifacts"].([]any)
	if len(artifacts) != 25 {
		t.Fatalf("expected default capture plan page size 25, got %d", len(artifacts))
	}
	firstArtifact := artifacts[0].(map[string]any)
	if firstArtifact["artifactType"] != "setup-svc" ||
		firstArtifact["suggestedSourceExists"] != false ||
		firstArtifact["sourceReadiness"] != "missing" ||
		firstArtifact["requiredShapeKey"] != "requiredSnapshotShape" ||
		firstArtifact["manifestStatusField"] != "setupSvcEvidenceStatus" ||
		!strings.HasPrefix(firstArtifact["suggestedSourcePath"].(string), filepath.Join("captures", "outputs", "setup-svc-live-replay")) ||
		!containsStringItem(firstArtifact["requiredEvidenceSections"].([]any), "tableSnapshots") ||
		!containsStringItem(firstArtifact["missingEvidenceSections"].([]any), "tableSnapshots") ||
		len(firstArtifact["requiredTables"].([]any)) == 0 ||
		len(firstArtifact["runtimeEffects"].([]any)) == 0 ||
		!containsStringFragment(firstArtifact["checklist"].([]any), "tableSnapshots") {
		t.Fatalf("expected capture plan artifact to carry strict setup-svc evidence contract, got %#v", firstArtifact)
	}
	firstCaptureTask := firstArtifact["captureTask"].(map[string]any)
	if firstCaptureTask["sourceSystem"] != "setup-svc" ||
		firstCaptureTask["captureMode"] != "manual_or_scripted_snapshot_capture" ||
		firstCaptureTask["targetPath"] != firstArtifact["path"] ||
		firstCaptureTask["suggestedSourcePath"] != firstArtifact["suggestedSourcePath"] ||
		firstCaptureTask["requiredShapeKey"] != "requiredSnapshotShape" ||
		firstCaptureTask["manifestStatusField"] != "setupSvcEvidenceStatus" ||
		!strings.Contains(firstCaptureTask["manualAction"].(string), "legacy setup-svc") ||
		!strings.Contains(firstCaptureTask["postCaptureCheckCommand"].(string), "--source-readiness complete") ||
		!strings.Contains(firstCaptureTask["postCaptureImportHint"].(string), "evidence-import --dry-run") ||
		!containsStringItem(firstCaptureTask["requiredEvidenceSections"].([]any), "tableSnapshots") ||
		!containsStringFragment(firstCaptureTask["stopConditions"].([]any), "Do not import incomplete captures") {
		t.Fatalf("expected setup-svc capture task to guide real source collection and readiness checks, got %#v", firstCaptureTask)
	}
	pageCommands := result["pageCommands"].(map[string]any)
	if !strings.Contains(pageCommands["currentPage"].(string), "setup-svc-live-replay-capture-plan") ||
		!strings.Contains(pageCommands["nextPage"].(string), "--offset 25 --limit 25") {
		t.Fatalf("expected capture plan page commands, got %#v", pageCommands)
	}
	operatorPacket := result["operatorPacket"].(map[string]any)
	if operatorPacket["sourceRoot"] != "captures" ||
		int(operatorPacket["sourceFilesPresent"].(float64)) != 0 ||
		int(operatorPacket["sourceFilesMissing"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount() ||
		int(operatorPacket["sourceFilesComplete"].(float64)) != 0 ||
		int(operatorPacket["sourceFilesIncomplete"].(float64)) != 0 ||
		!strings.Contains(operatorPacket["saveWorklistCommand"].(string), "setup-svc-live-replay-worklist") ||
		!strings.Contains(operatorPacket["dryRunImportCommand"].(string), "setup-svc-live-replay-evidence-import @") ||
		!containsStringFragment(operatorPacket["postReplacementCommands"].([]any), "setup-svc-live-replay-manifest-sync") {
		t.Fatalf("expected capture plan operator packet to guide import through existing worklist gates, got %#v", operatorPacket)
	}

	queryArgs := []string{tmp, "setup-svc-live-replay-capture-plan", "--artifact-type", "query-readback", "--limit", "5"}
	stdout.Reset()
	if err := Handle("scan", "msapi", queryArgs, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var queryOnly map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &queryOnly); err != nil {
		t.Fatal(err)
	}
	queryTotals := queryOnly["totals"].(map[string]any)
	if int(queryOnly["totalNextArtifacts"].(float64)) != setupSvcLiveReplayOperationCount() ||
		int(queryTotals["filteredArtifactFiles"].(float64)) != setupSvcLiveReplayOperationCount() ||
		int(queryTotals["queryReadbackArtifacts"].(float64)) != setupSvcLiveReplayOperationCount() {
		t.Fatalf("expected query-readback capture plan to list every query/readback artifact, got %#v", queryOnly)
	}
	queryArtifact := queryOnly["artifacts"].([]any)[0].(map[string]any)
	if queryArtifact["artifactType"] != "query-readback" ||
		queryArtifact["requiredShapeKey"] != "requiredReadbackShape" ||
		queryArtifact["manifestStatusField"] != "queryEvidenceStatus" ||
		!containsStringItem(queryArtifact["requiredEvidenceSections"].([]any), "readbackTables") ||
		!containsStringFragment(queryArtifact["checklist"].([]any), "readback table coverage") {
		t.Fatalf("expected query-readback capture artifact to include readback contract, got %#v", queryArtifact)
	}
	queryCaptureTask := queryArtifact["captureTask"].(map[string]any)
	if queryCaptureTask["sourceSystem"] != "msapi-query-readback" ||
		queryCaptureTask["captureMode"] != "msapi_query_readback_capture" ||
		!strings.Contains(queryCaptureTask["manualAction"].(string), "query/readback") ||
		!containsStringItem(queryCaptureTask["requiredEvidenceSections"].([]any), "readbackExpectationChecks") ||
		len(queryCaptureTask["queryReadbackExpectations"].([]any)) == 0 ||
		!strings.Contains(queryCaptureTask["postCaptureCheckCommand"].(string), "--artifact-type query-readback") {
		t.Fatalf("expected query-readback capture task to preserve query evidence contract, got %#v", queryCaptureTask)
	}

	normalizedArgs := []string{tmp, "setup-svc-live-replay-capture-plan", "--artifact-type", "normalized-diff", "--limit", "1"}
	stdout.Reset()
	if err := Handle("scan", "msapi", normalizedArgs, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var normalizedOnly map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &normalizedOnly); err != nil {
		t.Fatal(err)
	}
	normalizedArtifact := normalizedOnly["artifacts"].([]any)[0].(map[string]any)
	normalizedCaptureTask := normalizedArtifact["captureTask"].(map[string]any)
	if normalizedCaptureTask["sourceSystem"] != "local-normalized-diff" ||
		normalizedCaptureTask["captureMode"] != "approval_gated_generated_diff" ||
		!strings.Contains(normalizedCaptureTask["captureCommand"].(string), "setup-svc-live-replay-normalized-diff") ||
		!strings.Contains(normalizedCaptureTask["captureCommand"].(string), setupSvcParityNormalizedDiffApproval) ||
		!containsStringItem(normalizedCaptureTask["requiredEvidenceSections"].([]any), "diffCounters") {
		t.Fatalf("expected normalized diff capture task to expose approval-gated generation command, got %#v", normalizedCaptureTask)
	}

	sourcePath := filepath.Join(tmp, firstArtifact["suggestedSourcePath"].(string))
	sourceJSON := fmt.Sprintf(`{
  "status": "passed",
  "project": %q,
  "contractVersion": %q,
  "contractFingerprint": %q,
  "domain": %q,
  "operation": %q,
  "artifactType": %q,
  "tableSnapshots": {"tp_sys_object": {"columns": ["ID"], "rows": [{"ID": "obj_1"}]}},
  "runtimeEffectChecks": [{"name": "metadata-allocation", "status": "passed"}]
}`, tmp, setupSvcLiveReplayContractVersion, setupSvcLiveReplayExpectedContractFingerprint(), firstArtifact["domain"], firstArtifact["operation"], firstArtifact["artifactType"])
	writeTestFile(t, sourcePath, sourceJSON)

	stdout.Reset()
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-capture-plan", "--source-status", "present"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var presentOnly map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &presentOnly); err != nil {
		t.Fatal(err)
	}
	presentFilters := presentOnly["filters"].(map[string]any)
	if presentFilters["sourceStatus"] != "present" {
		t.Fatalf("expected present source-status filter, got %#v", presentFilters)
	}
	presentTotals := presentOnly["totals"].(map[string]any)
	if int(presentOnly["totalNextArtifacts"].(float64)) != 1 ||
		int(presentTotals["sourceFilesPresent"].(float64)) != 1 ||
		int(presentTotals["sourceFilesMissing"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount()-1 ||
		int(presentTotals["sourceFilesComplete"].(float64)) != 1 ||
		int(presentTotals["sourceFilesIncomplete"].(float64)) != 0 ||
		int(presentTotals["filteredArtifactFiles"].(float64)) != 1 ||
		int(presentTotals["sourceEvidenceMissing"].(float64)) != 0 {
		t.Fatalf("expected present capture plan to keep one complete source file, got %#v", presentOnly)
	}
	presentArtifact := presentOnly["artifacts"].([]any)[0].(map[string]any)
	presentPacket := presentOnly["operatorPacket"].(map[string]any)
	missingSections, _ := presentArtifact["missingEvidenceSections"].([]any)
	if presentArtifact["suggestedSourceExists"] != true ||
		presentArtifact["sourceReadiness"] != "complete" ||
		len(missingSections) != 0 ||
		!evidenceSectionHasStatus(presentArtifact["sourceEvidenceSections"].([]any), "tableSnapshots", "present") ||
		!evidenceSectionHasStatus(presentArtifact["sourceEvidenceSections"].([]any), "runtimeEffectChecks", "present") {
		t.Fatalf("expected present source artifact to expose complete section statuses, got %#v", presentArtifact)
	}
	if int(presentPacket["sourceFilesPresent"].(float64)) != 1 ||
		int(presentPacket["sourceFilesMissing"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount()-1 ||
		int(presentPacket["sourceFilesComplete"].(float64)) != 1 ||
		int(presentPacket["sourceFilesIncomplete"].(float64)) != 0 {
		t.Fatalf("expected capture plan operator packet to mirror source completeness counters, got %#v", presentPacket)
	}
	presentSourceSections := presentOnly["sourceEvidenceSections"].([]any)
	presentPacketSections := presentPacket["sourceEvidenceSections"].([]any)
	if len(presentSourceSections) == 0 ||
		len(presentPacketSections) != len(presentSourceSections) ||
		!evidenceSectionSummaryHasCount(presentSourceSections, "setup-svc", "tableSnapshots", 1, 1, 0) ||
		!evidenceSectionSummaryHasCount(presentSourceSections, "setup-svc", "runtimeEffectChecks", 1, 1, 0) {
		t.Fatalf("expected capture plan to summarize complete source evidence sections, got top=%#v packet=%#v", presentSourceSections, presentPacketSections)
	}
	if queues, ok := presentOnly["sourceMissingSectionQueues"].([]any); ok && len(queues) != 0 {
		t.Fatalf("complete source capture plan should not expose missing section queues, got %#v", queues)
	}

	stdout.Reset()
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-capture-plan", "--source-readiness", "complete"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var completeOnly map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &completeOnly); err != nil {
		t.Fatal(err)
	}
	completeFilters := completeOnly["filters"].(map[string]any)
	completePageCommands := completeOnly["pageCommands"].(map[string]any)
	if completeFilters["sourceReadiness"] != "complete" ||
		int(completeOnly["totalNextArtifacts"].(float64)) != 1 ||
		completeOnly["artifacts"].([]any)[0].(map[string]any)["sourceReadiness"] != "complete" ||
		!strings.Contains(completePageCommands["currentPage"].(string), "--source-readiness complete") {
		t.Fatalf("expected complete source-readiness filter to isolate importable artifact, got %#v", completeOnly)
	}

	stdout.Reset()
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-capture-plan", "--source-status", "missing", "--offset", "25", "--limit", "25"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var missingPage map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &missingPage); err != nil {
		t.Fatal(err)
	}
	missingPageCommands := missingPage["pageCommands"].(map[string]any)
	if int(missingPage["totalNextArtifacts"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount()-1 ||
		int(missingPage["omittedNextArtifacts"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount()-1-50 ||
		!strings.Contains(missingPageCommands["currentPage"].(string), "--source-status missing") ||
		!strings.Contains(missingPageCommands["previousPage"].(string), "--offset 0 --limit 25") ||
		!strings.Contains(missingPageCommands["nextPage"].(string), "--offset 50 --limit 25") {
		t.Fatalf("expected missing source capture plan to page remaining artifacts, got %#v", missingPage)
	}

	secondArtifact := artifacts[1].(map[string]any)
	partialSourcePath := filepath.Join(tmp, secondArtifact["suggestedSourcePath"].(string))
	partialSourceJSON := fmt.Sprintf(`{
  "status": "passed",
  "project": %q,
  "contractVersion": %q,
  "contractFingerprint": %q,
  "domain": %q,
  "operation": %q,
  "artifactType": %q
}`, tmp, setupSvcLiveReplayContractVersion, setupSvcLiveReplayExpectedContractFingerprint(), secondArtifact["domain"], secondArtifact["operation"], secondArtifact["artifactType"])
	writeTestFile(t, partialSourcePath, partialSourceJSON)

	stdout.Reset()
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-capture-plan", "--source-readiness", "incomplete"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var incompleteOnly map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &incompleteOnly); err != nil {
		t.Fatal(err)
	}
	incompleteTotals := incompleteOnly["totals"].(map[string]any)
	incompleteArtifact := incompleteOnly["artifacts"].([]any)[0].(map[string]any)
	if int(incompleteOnly["totalNextArtifacts"].(float64)) != 1 ||
		int(incompleteTotals["sourceFilesPresent"].(float64)) != 2 ||
		int(incompleteTotals["sourceFilesMissing"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount()-2 ||
		int(incompleteTotals["sourceFilesComplete"].(float64)) != 1 ||
		int(incompleteTotals["sourceFilesIncomplete"].(float64)) != 1 ||
		int(incompleteTotals["filteredArtifactFiles"].(float64)) != 1 ||
		incompleteArtifact["sourceReadiness"] != "incomplete" ||
		len(incompleteArtifact["missingEvidenceSections"].([]any)) == 0 {
		t.Fatalf("expected incomplete source-readiness filter to isolate partial source file, got %#v", incompleteOnly)
	}
	incompleteSourceSections := incompleteOnly["sourceEvidenceSections"].([]any)
	incompletePacketSections := incompleteOnly["operatorPacket"].(map[string]any)["sourceEvidenceSections"].([]any)
	incompleteTableSnapshots := evidenceSectionSummary(incompleteSourceSections, "metadata-service", "tableSnapshots")
	if len(incompleteSourceSections) == 0 ||
		len(incompletePacketSections) != len(incompleteSourceSections) ||
		incompleteTableSnapshots == nil ||
		int(incompleteTableSnapshots["total"].(float64)) != 1 ||
		int(incompleteTableSnapshots["present"].(float64)) != 0 ||
		int(incompleteTableSnapshots["missing"].(float64)) != 1 ||
		!strings.Contains(incompleteTableSnapshots["queueCommand"].(string), "--source-readiness incomplete") ||
		!strings.Contains(incompleteTableSnapshots["queueCommand"].(string), "--artifact-type metadata-service") ||
		!strings.Contains(incompleteTableSnapshots["queueCommand"].(string), "--evidence-section tableSnapshots") {
		t.Fatalf("expected capture plan source section backlog to preserve source-readiness filter, got top=%#v packet=%#v", incompleteSourceSections, incompletePacketSections)
	}
	incompleteQueues := incompleteOnly["sourceMissingSectionQueues"].([]any)
	incompletePacketQueues := incompleteOnly["operatorPacket"].(map[string]any)["sourceMissingSectionQueues"].([]any)
	incompleteTableQueue := evidenceSectionQueue(incompleteQueues, "metadata-service", "tableSnapshots")
	if len(incompleteQueues) == 0 ||
		len(incompletePacketQueues) != len(incompleteQueues) ||
		incompleteTableQueue == nil ||
		int(incompleteTableQueue["missing"].(float64)) != 1 ||
		int(incompleteTableQueue["pageSize"].(float64)) != 25 ||
		int(incompleteTableQueue["batchCount"].(float64)) != 1 ||
		!strings.Contains(incompleteTableQueue["queueCommand"].(string), "--source-readiness incomplete") ||
		!containsStringFragment(incompleteTableQueue["batchCommands"].([]any), "--source-readiness incomplete") {
		t.Fatalf("expected capture plan source missing section queues with batched source filters, got top=%#v packet=%#v", incompleteQueues, incompletePacketQueues)
	}
	if !strings.Contains(incompleteTableQueue["queueCommand"].(string), "setup-svc-live-replay-capture-plan") ||
		strings.Contains(incompleteTableQueue["queueCommand"].(string), "setup-svc-live-replay-gaps") {
		t.Fatalf("expected source missing section queue to use capture-plan, got %#v", incompleteTableQueue)
	}

	stdout.Reset()
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-capture-plan", "--artifact-type", "metadata-service", "--evidence-section", "tableSnapshots", "--section-status", "missing", "--source-readiness", "incomplete"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var incompleteSectionOnly map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &incompleteSectionOnly); err != nil {
		t.Fatal(err)
	}
	incompleteSectionFilters := incompleteSectionOnly["filters"].(map[string]any)
	incompleteSectionCommands := incompleteSectionOnly["pageCommands"].(map[string]any)
	if incompleteSectionFilters["evidenceSection"] != "tableSnapshots" ||
		incompleteSectionFilters["sectionStatus"] != "missing" ||
		int(incompleteSectionOnly["totalNextArtifacts"].(float64)) != 1 ||
		incompleteSectionOnly["artifacts"].([]any)[0].(map[string]any)["artifactType"] != "metadata-service" ||
		!strings.Contains(incompleteSectionCommands["currentPage"].(string), "--evidence-section tableSnapshots") ||
		!strings.Contains(incompleteSectionCommands["currentPage"].(string), "--section-status missing") ||
		!strings.Contains(incompleteSectionCommands["currentPage"].(string), "--source-readiness incomplete") {
		t.Fatalf("expected capture plan to filter by source evidence section/status, got %#v", incompleteSectionOnly)
	}

	stdout.Reset()
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-capture-plan", "--source-readiness", "missing", "--offset", "25", "--limit", "25"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var missingReadinessPage map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &missingReadinessPage); err != nil {
		t.Fatal(err)
	}
	missingReadinessCommands := missingReadinessPage["pageCommands"].(map[string]any)
	if int(missingReadinessPage["totalNextArtifacts"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount()-2 ||
		int(missingReadinessPage["omittedNextArtifacts"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount()-2-50 ||
		!strings.Contains(missingReadinessCommands["currentPage"].(string), "--source-readiness missing") ||
		!strings.Contains(missingReadinessCommands["previousPage"].(string), "--offset 0 --limit 25") ||
		!strings.Contains(missingReadinessCommands["nextPage"].(string), "--offset 50 --limit 25") {
		t.Fatalf("expected missing source-readiness capture plan to page only absent artifacts, got %#v", missingReadinessPage)
	}
}

func TestSetupSvcLiveReplayCaptureSourceWorkspaceInitializesMissingTemplates(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)

	args := []string{tmp, "setup-svc-live-replay-capture-source-workspace", "--artifact-type", "query-readback", "--limit", "2"}
	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", append(args, "--dry-run"), &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var dryRun map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &dryRun); err != nil {
		t.Fatal(err)
	}
	if dryRun["mode"] != "setup-svc-live-replay-capture-source-workspace" ||
		dryRun["readOnly"] != false ||
		dryRun["status"] != "dry_run_ready" ||
		dryRun["approved"] != false {
		t.Fatalf("expected dry-run capture source workspace result, got %#v", dryRun)
	}
	dryTotals := dryRun["totals"].(map[string]any)
	if int(dryTotals["filteredArtifactFiles"].(float64)) != setupSvcLiveReplayOperationCount() ||
		int(dryTotals["plannedFiles"].(float64)) != 2 ||
		int(dryTotals["writtenFiles"].(float64)) != 0 ||
		int(dryTotals["sourceFilesMissing"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount() {
		t.Fatalf("expected dry-run to plan two missing query-readback source files without writes, got %#v", dryTotals)
	}
	sampleFiles := dryRun["sampleFiles"].([]any)
	if len(sampleFiles) != 2 || !strings.HasPrefix(sampleFiles[0].(string), filepath.Join("captures", "outputs", "setup-svc-live-replay")) {
		t.Fatalf("expected dry-run sample capture source paths, got %#v", sampleFiles)
	}
	if _, err := os.Stat(filepath.Join(tmp, sampleFiles[0].(string))); !os.IsNotExist(err) {
		t.Fatalf("dry-run must not write first sample file, stat err=%v", err)
	}
	nextCommands := dryRun["nextCommands"].(map[string]any)
	if !strings.Contains(nextCommands["prepareCaptureSources"].(string), setupSvcParityCaptureSourceWorkspaceApproval) ||
		!strings.Contains(nextCommands["completeWorklist"].(string), "--source-readiness complete") ||
		!strings.Contains(nextCommands["dryRunImport"].(string), "setup-svc-live-replay-evidence-import") {
		t.Fatalf("expected capture source workspace next commands, got %#v", nextCommands)
	}

	stdout.Reset()
	err := Handle("apply", "msapi", append(args, "--execute", "--approval", "WRONG"), &stdout, tmp)
	if err == nil || !strings.Contains(err.Error(), setupSvcParityCaptureSourceWorkspaceApproval) {
		t.Fatalf("expected wrong approval to be rejected, err=%v", err)
	}
	if _, statErr := os.Stat(filepath.Join(tmp, sampleFiles[0].(string))); !os.IsNotExist(statErr) {
		t.Fatalf("wrong approval must not write first sample file, stat err=%v", statErr)
	}

	stdout.Reset()
	if err := Handle("apply", "msapi", append(args, "--execute", "--approval", setupSvcParityCaptureSourceWorkspaceApproval), &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var applied map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &applied); err != nil {
		t.Fatal(err)
	}
	appliedTotals := applied["totals"].(map[string]any)
	if applied["status"] != "applied" ||
		applied["approved"] != true ||
		int(appliedTotals["plannedFiles"].(float64)) != 2 ||
		int(appliedTotals["writtenFiles"].(float64)) != 2 ||
		int(appliedTotals["skippedExistingFiles"].(float64)) != 0 {
		t.Fatalf("expected approved execution to write two capture source templates, got %#v", applied)
	}
	firstSourcePath := filepath.Join(tmp, sampleFiles[0].(string))
	payload, err := os.ReadFile(firstSourcePath)
	if err != nil {
		t.Fatal(err)
	}
	var sourceTemplate map[string]any
	if err := json.Unmarshal(payload, &sourceTemplate); err != nil {
		t.Fatal(err)
	}
	if sourceTemplate["status"] != "pending" ||
		sourceTemplate["artifactType"] != "query-readback" ||
		sourceTemplate["sourceTemplateStatus"] != "incomplete" ||
		sourceTemplate["requiredShapeKey"] != "requiredReadbackShape" ||
		sourceTemplate["manifestStatusField"] != "queryEvidenceStatus" ||
		!containsStringItem(sourceTemplate["requiredEvidenceSections"].([]any), "readbackTables") ||
		!strings.Contains(sourceTemplate["targetEvidencePath"].(string), filepath.Join("outputs", "setup-svc-live-replay")) {
		t.Fatalf("expected pending query-readback source template with target evidence path, got %#v", sourceTemplate)
	}
	captureTask := sourceTemplate["captureTask"].(map[string]any)
	if captureTask["sourceSystem"] != "msapi-query-readback" ||
		captureTask["captureMode"] != "msapi_query_readback_capture" ||
		captureTask["requiredShapeKey"] != "requiredReadbackShape" ||
		!containsStringItem(captureTask["requiredEvidenceSections"].([]any), "readbackTables") ||
		!strings.Contains(captureTask["postCaptureCheckCommand"].(string), "--source-readiness complete") {
		t.Fatalf("expected source template to embed actionable query-readback captureTask, got %#v", captureTask)
	}

	stdout.Reset()
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-capture-plan", "--artifact-type", "query-readback", "--source-readiness", "incomplete"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var incompletePlan map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &incompletePlan); err != nil {
		t.Fatal(err)
	}
	if int(incompletePlan["totalNextArtifacts"].(float64)) != 2 {
		t.Fatalf("expected generated source templates to remain incomplete until real evidence is captured, got %#v", incompletePlan)
	}

	delete(sourceTemplate, "requiredShapeKey")
	delete(sourceTemplate, "manifestStatusField")
	delete(sourceTemplate, "requiredEvidenceSections")
	sourceTemplate["contractFingerprint"] = "sha256:old-template-contract"
	legacyBody, err := json.Marshal(sourceTemplate)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, firstSourcePath, string(legacyBody))

	stdout.Reset()
	presentArgs := []string{tmp, "setup-svc-live-replay-capture-source-workspace", "--artifact-type", "query-readback", "--source-status", "present", "--limit", "2"}
	if err := Handle("apply", "msapi", append(presentArgs, "--execute", "--approval", setupSvcParityCaptureSourceWorkspaceApproval), &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var refreshed map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &refreshed); err != nil {
		t.Fatal(err)
	}
	refreshedTotals := refreshed["totals"].(map[string]any)
	if refreshed["status"] != "applied" ||
		int(refreshedTotals["writtenFiles"].(float64)) != 0 ||
		int(refreshedTotals["refreshedExistingFiles"].(float64)) != 2 ||
		int(refreshedTotals["skippedExistingFiles"].(float64)) != 0 ||
		int(refreshedTotals["plannedFiles"].(float64)) != 2 {
		t.Fatalf("expected present-source execution to refresh pending templates onto the current contract without overwriting evidence, got %#v", refreshed)
	}
	refreshedPayload, err := os.ReadFile(firstSourcePath)
	if err != nil {
		t.Fatal(err)
	}
	var refreshedTemplate map[string]any
	if err := json.Unmarshal(refreshedPayload, &refreshedTemplate); err != nil {
		t.Fatal(err)
	}
	if refreshedTemplate["requiredShapeKey"] != "requiredReadbackShape" ||
		refreshedTemplate["manifestStatusField"] != "queryEvidenceStatus" ||
		refreshedTemplate["contractFingerprint"] != buildSetupSvcLiveReplayPacket(tmp).ContractFingerprint ||
		!containsStringItem(refreshedTemplate["requiredEvidenceSections"].([]any), "readbackTables") {
		t.Fatalf("expected legacy pending template refresh to restore top-level guide fields and contract identity, got %#v", refreshedTemplate)
	}

	stdout.Reset()
	if err := Handle("apply", "msapi", append(presentArgs, "--execute", "--approval", setupSvcParityCaptureSourceWorkspaceApproval), &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var repeated map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &repeated); err != nil {
		t.Fatal(err)
	}
	repeatedTotals := repeated["totals"].(map[string]any)
	if repeated["status"] != "applied" ||
		int(repeatedTotals["writtenFiles"].(float64)) != 0 ||
		int(repeatedTotals["refreshedExistingFiles"].(float64)) != 2 ||
		int(repeatedTotals["skippedExistingFiles"].(float64)) != 0 ||
		int(repeatedTotals["plannedFiles"].(float64)) != 2 {
		t.Fatalf("expected repeated present-source execution to safely refresh pending capture templates without overwriting evidence, got %#v", repeated)
	}
}

func TestSetupSvcLiveReplayWorklistFiltersSingleBatch(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-workspace", "--execute", "--approval", setupSvcParityEvidenceWorkspaceApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	worklistArgs := []string{
		tmp,
		"setup-svc-live-replay-worklist",
		"--artifact-type", "query-readback",
		"--evidence-section", "readbackTables",
		"--section-status", "missing",
		"--limit", "10",
		"--batch-index", "2",
	}
	stdout.Reset()
	if err := Handle("scan", "msapi", worklistArgs, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if int(result["batchIndex"].(float64)) != 2 || int(result["batchLimit"].(float64)) != 1 {
		t.Fatalf("expected worklist to expose selected batch index and effective limit, got %#v", result)
	}
	operatorPacket := result["operatorPacket"].(map[string]any)
	if operatorPacket["replacementStatusTarget"] != "passed|verified|success" ||
		operatorPacket["sourceRoot"] != "captures" ||
		operatorPacket["captureRoot"] != filepath.Join(tmp, "captures") ||
		int(operatorPacket["artifactReplacementCount"].(float64)) != 10 ||
		int(operatorPacket["sourceFilesPresent"].(float64)) != 0 ||
		int(operatorPacket["sourceFilesMissing"].(float64)) != 10 ||
		!strings.Contains(operatorPacket["suggestedWorklistPath"].(string), "worklist-query-readback-readbacktables-missing-batch-2") ||
		!strings.Contains(operatorPacket["saveWorklistCommand"].(string), "setup-svc-live-replay-worklist") ||
		!strings.Contains(operatorPacket["saveWorklistCommand"].(string), "--batch-index 2") ||
		!strings.Contains(operatorPacket["saveWorklistCommand"].(string), ">") ||
		!strings.Contains(operatorPacket["dryRunImportCommand"].(string), "setup-svc-live-replay-evidence-import @") ||
		!strings.Contains(operatorPacket["dryRunImportCommand"].(string), "--dry-run") ||
		!strings.Contains(operatorPacket["executeImportCommand"].(string), setupSvcParityEvidenceImportApproval) ||
		!containsStringFragment(operatorPacket["postReplacementCommands"].([]any), "setup-svc-live-replay-manifest-sync") ||
		!containsStringFragment(operatorPacket["postReplacementCommands"].([]any), "setup-svc-live-replay-evidence") ||
		!containsStringFragment(operatorPacket["postReplacementCommands"].([]any), "setup-svc-live-replay-evidence-bundle") ||
		!containsStringFragment(operatorPacket["stopConditions"].([]any), "Do not mark any artifact passed") {
		t.Fatalf("expected worklist operator packet to guide replacement through downstream gates, got %#v", operatorPacket)
	}
	batchSaveCommands := result["batchSaveCommands"].([]any)
	operatorBatchSaveCommands := operatorPacket["batchSaveCommands"].([]any)
	if len(batchSaveCommands) != 1 ||
		len(operatorBatchSaveCommands) != 1 ||
		!containsBatchSaveCommand(batchSaveCommands, "query-readback", "readbackTables", 2, "worklist-query-readback-readbacktables-missing-batch-2") {
		t.Fatalf("expected filtered worklist to expose only the selected batch save command, got top=%#v packet=%#v", batchSaveCommands, operatorBatchSaveCommands)
	}
	totals := result["totals"].(map[string]any)
	if int(totals["queues"].(float64)) != 1 ||
		int(totals["batches"].(float64)) != 1 ||
		int(totals["artifacts"].(float64)) != 10 ||
		int(totals["sourceFilesPresent"].(float64)) != 0 ||
		int(totals["sourceFilesMissing"].(float64)) != 10 ||
		int(totals["queryReadbackQueues"].(float64)) != 1 ||
		int(totals["queryReadbackArtifacts"].(float64)) != 10 ||
		int(totals["omittedBatches"].(float64)) != (setupSvcLiveReplayOperationCount()+9)/10-1 {
		t.Fatalf("expected single filtered query-readback batch totals, got %#v", totals)
	}
	queues := result["queues"].([]any)
	if len(queues) != 1 {
		t.Fatalf("expected one filtered queue, got %#v", queues)
	}
	queue := queues[0].(map[string]any)
	if queue["artifactType"] != "query-readback" ||
		queue["section"] != "readbackTables" ||
		queue["requiredShapeKey"] != "requiredReadbackShape" ||
		queue["manifestStatusField"] != "queryEvidenceStatus" {
		t.Fatalf("expected only query-readback readbackTables queue, got %#v", queue)
	}
	if !strings.Contains(queue["queueCommand"].(string), "setup-svc-live-replay-worklist") ||
		strings.Contains(queue["queueCommand"].(string), "setup-svc-live-replay-gaps") ||
		!strings.Contains(queue["queueCommand"].(string), "--artifact-type query-readback") ||
		!strings.Contains(queue["queueCommand"].(string), "--evidence-section readbackTables") ||
		!strings.Contains(queue["queueCommand"].(string), "--section-status missing") {
		t.Fatalf("expected worklist queue command to stay executable as worklist command, got %#v", queue["queueCommand"])
	}
	batches := queue["batches"].([]any)
	if len(batches) != 1 {
		t.Fatalf("expected one selected batch, got %#v", batches)
	}
	batch := batches[0].(map[string]any)
	if int(batch["batchIndex"].(float64)) != 2 ||
		int(batch["offset"].(float64)) != 20 ||
		int(batch["limit"].(float64)) != 10 ||
		int(batch["count"].(float64)) != 10 ||
		!strings.Contains(batch["command"].(string), "setup-svc-live-replay-worklist") ||
		strings.Contains(batch["command"].(string), "setup-svc-live-replay-gaps") ||
		!strings.Contains(batch["command"].(string), "--batch-index 2") ||
		!strings.Contains(batch["saveWorklistCommand"].(string), " > ") ||
		!strings.Contains(batch["saveWorklistCommand"].(string), "worklist-query-readback-readbacktables-missing-batch-2") ||
		!strings.Contains(batch["executeImportCommand"].(string), setupSvcParityEvidenceImportApproval) {
		t.Fatalf("expected selected batch index 2 with offset 20, got %#v", batch)
	}
	operatorBatch := batch["operatorBatch"].(map[string]any)
	if int(operatorBatch["batchIndex"].(float64)) != 2 ||
		operatorBatch["artifactType"] != "query-readback" ||
		operatorBatch["evidenceSection"] != "readbackTables" ||
		operatorBatch["replacementStatusTarget"] != "passed|verified|success" ||
		!containsStringFragment(operatorBatch["postReplacementCommands"].([]any), "setup-svc-live-replay-manifest-sync") ||
		!containsStringFragment(operatorBatch["postReplacementCommands"].([]any), "setup-svc-live-replay-completion-audit") {
		t.Fatalf("expected selected batch operator instructions to preserve query-readback scope and downstream gates, got %#v", operatorBatch)
	}
	replacementRecords := operatorBatch["artifactReplacementRecords"].([]any)
	if len(replacementRecords) != 10 {
		t.Fatalf("expected one replacement record per artifact, got %#v", replacementRecords)
	}
	firstRecord := replacementRecords[0].(map[string]any)
	if firstRecord["artifactType"] != "query-readback" ||
		!strings.HasPrefix(firstRecord["suggestedSourcePath"].(string), filepath.Join("captures", "outputs", "setup-svc-live-replay")) ||
		firstRecord["suggestedSourceExists"] != false ||
		firstRecord["sourceReadiness"] != "missing" ||
		firstRecord["requiredShapeKey"] != "requiredReadbackShape" ||
		firstRecord["manifestStatusField"] != "queryEvidenceStatus" ||
		firstRecord["replacementStatusTarget"] != "passed|verified|success" ||
		!containsStringItem(firstRecord["requiredEvidenceSections"].([]any), "readbackTables") ||
		!containsStringItem(firstRecord["missingEvidenceSections"].([]any), "readbackTables") ||
		!containsStringItem(firstRecord["missingEvidenceSections"].([]any), "relationshipChecks") ||
		len(firstRecord["requiredTables"].([]any)) == 0 ||
		len(firstRecord["queryReadbackExpectations"].([]any)) == 0 ||
		!containsStringFragment(firstRecord["checklist"].([]any), "readback table coverage") {
		t.Fatalf("expected replacement record to carry strict query-readback evidence contract, got %#v", firstRecord)
	}
	firstRecordCaptureTask := firstRecord["captureTask"].(map[string]any)
	if firstRecordCaptureTask["sourceSystem"] != "msapi-query-readback" ||
		firstRecordCaptureTask["captureMode"] != "msapi_query_readback_capture" ||
		firstRecordCaptureTask["targetPath"] != firstRecord["path"] ||
		firstRecordCaptureTask["suggestedSourcePath"] != firstRecord["suggestedSourcePath"] ||
		firstRecordCaptureTask["manifestStatusField"] != "queryEvidenceStatus" ||
		!strings.Contains(firstRecordCaptureTask["manualAction"].(string), "query/readback") ||
		!containsStringItem(firstRecordCaptureTask["requiredEvidenceSections"].([]any), "readbackTables") ||
		len(firstRecordCaptureTask["queryReadbackExpectations"].([]any)) == 0 ||
		!strings.Contains(firstRecordCaptureTask["postCaptureCheckCommand"].(string), "--source-readiness complete") {
		t.Fatalf("expected replacement record capture task to preserve query-readback collection instructions, got %#v", firstRecordCaptureTask)
	}
	writeTestFile(t, filepath.Join(tmp, firstRecord["suggestedSourcePath"].(string)), `{"status":"passed"}`)
	stdout.Reset()
	if err := Handle("scan", "msapi", worklistArgs, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var refreshed map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &refreshed); err != nil {
		t.Fatal(err)
	}
	refreshedTotals := refreshed["totals"].(map[string]any)
	if int(refreshedTotals["sourceFilesPresent"].(float64)) != 1 ||
		int(refreshedTotals["sourceFilesMissing"].(float64)) != 9 ||
		int(refreshedTotals["sourceFilesComplete"].(float64)) != 0 ||
		int(refreshedTotals["sourceFilesIncomplete"].(float64)) != 1 {
		t.Fatalf("expected one mirrored source file present after writing capture, got %#v", refreshedTotals)
	}
	refreshedPacket := refreshed["operatorPacket"].(map[string]any)
	if int(refreshedPacket["sourceFilesPresent"].(float64)) != 1 ||
		int(refreshedPacket["sourceFilesMissing"].(float64)) != 9 ||
		int(refreshedPacket["sourceFilesComplete"].(float64)) != 0 ||
		int(refreshedPacket["sourceFilesIncomplete"].(float64)) != 1 {
		t.Fatalf("expected operator packet source file counters to refresh, got %#v", refreshedPacket)
	}
	refreshedRecord := refreshed["queues"].([]any)[0].(map[string]any)["batches"].([]any)[0].(map[string]any)["operatorBatch"].(map[string]any)["artifactReplacementRecords"].([]any)[0].(map[string]any)
	if refreshedRecord["suggestedSourceExists"] != true || refreshedRecord["sourceReadiness"] != "incomplete" {
		t.Fatalf("expected replacement record to detect mirrored capture file, got %#v", refreshedRecord)
	}
	completeArgs := append(append([]string{}, worklistArgs...), "--source-readiness", "complete")
	stdout.Reset()
	if err := Handle("scan", "msapi", completeArgs, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var noComplete map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &noComplete); err != nil {
		t.Fatal(err)
	}
	noCompleteTotals := noComplete["totals"].(map[string]any)
	noCompletePacket := noComplete["operatorPacket"].(map[string]any)
	noCompleteQueue := noComplete["queues"].([]any)[0].(map[string]any)
	noCompleteBatchCount := 0
	if batches, ok := noCompleteQueue["batches"].([]any); ok {
		noCompleteBatchCount = len(batches)
	}
	if int(noCompleteTotals["artifacts"].(float64)) != 0 ||
		int(noCompleteTotals["batches"].(float64)) != 0 ||
		int(noCompleteTotals["sourceFilesComplete"].(float64)) != 0 ||
		noCompleteBatchCount != 0 ||
		noComplete["batchSaveCommands"] != nil ||
		noCompletePacket["batchSaveCommands"] != nil ||
		!strings.Contains(noCompletePacket["saveWorklistCommand"].(string), "--source-readiness complete") {
		t.Fatalf("expected incomplete present source to stay out of complete worklist, got %#v", noComplete)
	}
	secondRecord := replacementRecords[1].(map[string]any)
	completeSourceJSON := fmt.Sprintf(`{
  "status": "passed",
  "project": %q,
  "contractVersion": %q,
  "contractFingerprint": %q,
  "domain": %q,
  "operation": %q,
  "artifactType": %q,
  "queryShape": {"fields": ["ID"]},
  "readbackShape": {"fields": ["ID"]},
  "readbackTables": {"tp_sys_object": {"columns": ["ID"], "rows": [{"ID": "obj_1"}]}},
  "relationshipChecks": [{"name": "object-readback", "status": "passed", "source": "tp_sys_object", "target": "tp_sys_object", "field": "ID"}],
  "readbackExpectationChecks": [{"name": "query-readback-structure", "status": "passed"}],
  "missing": 0,
  "mismatched": 0,
  "broken": 0,
  "errors": 0
}`, tmp, setupSvcLiveReplayContractVersion, setupSvcLiveReplayExpectedContractFingerprint(), secondRecord["domain"], secondRecord["operation"], secondRecord["artifactType"])
	writeTestFile(t, filepath.Join(tmp, secondRecord["suggestedSourcePath"].(string)), completeSourceJSON)

	stdout.Reset()
	if err := Handle("scan", "msapi", completeArgs, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var completeOnly map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &completeOnly); err != nil {
		t.Fatal(err)
	}
	completeFilters := completeOnly["filters"].(map[string]any)
	completeTotals := completeOnly["totals"].(map[string]any)
	completePacket := completeOnly["operatorPacket"].(map[string]any)
	completeRecords := completeOnly["queues"].([]any)[0].(map[string]any)["batches"].([]any)[0].(map[string]any)["operatorBatch"].(map[string]any)["artifactReplacementRecords"].([]any)
	if completeFilters["sourceReadiness"] != "complete" ||
		int(completeTotals["artifacts"].(float64)) != 1 ||
		int(completeTotals["batches"].(float64)) != 1 ||
		int(completeTotals["sourceFilesPresent"].(float64)) != 1 ||
		int(completeTotals["sourceFilesComplete"].(float64)) != 1 ||
		int(completeTotals["sourceFilesIncomplete"].(float64)) != 0 ||
		int(completePacket["artifactReplacementCount"].(float64)) != 1 ||
		int(completePacket["sourceFilesComplete"].(float64)) != 1 ||
		len(completeOnly["batchSaveCommands"].([]any)) != 1 ||
		len(completePacket["batchSaveCommands"].([]any)) != 1 ||
		!strings.Contains(completePacket["suggestedWorklistPath"].(string), "readiness-complete") ||
		!strings.Contains(completePacket["saveWorklistCommand"].(string), "--source-readiness complete") ||
		len(completeRecords) != 1 ||
		completeRecords[0].(map[string]any)["sourceReadiness"] != "complete" {
		t.Fatalf("expected source-readiness complete worklist to keep only structurally complete capture, got %#v", completeOnly)
	}
	incompleteArgs := append(append([]string{}, worklistArgs...), "--source-readiness", "incomplete")
	stdout.Reset()
	if err := Handle("scan", "msapi", incompleteArgs, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var incompleteOnly map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &incompleteOnly); err != nil {
		t.Fatal(err)
	}
	incompleteTotals := incompleteOnly["totals"].(map[string]any)
	if int(incompleteTotals["artifacts"].(float64)) != 1 ||
		int(incompleteTotals["sourceFilesIncomplete"].(float64)) != 1 ||
		incompleteOnly["queues"].([]any)[0].(map[string]any)["batches"].([]any)[0].(map[string]any)["operatorBatch"].(map[string]any)["artifactReplacementRecords"].([]any)[0].(map[string]any)["sourceReadiness"] != "incomplete" {
		t.Fatalf("expected source-readiness incomplete worklist to isolate partial capture, got %#v", incompleteOnly)
	}
	presentArgs := append(append([]string{}, worklistArgs...), "--source-status", "present")
	stdout.Reset()
	if err := Handle("scan", "msapi", presentArgs, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var presentOnly map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &presentOnly); err != nil {
		t.Fatal(err)
	}
	presentFilters := presentOnly["filters"].(map[string]any)
	if presentFilters["sourceStatus"] != "present" {
		t.Fatalf("expected present source-status filter, got %#v", presentFilters)
	}
	presentTotals := presentOnly["totals"].(map[string]any)
	if int(presentTotals["artifacts"].(float64)) != 2 ||
		int(presentTotals["sourceFilesPresent"].(float64)) != 2 ||
		int(presentTotals["sourceFilesMissing"].(float64)) != 0 ||
		int(presentTotals["sourceFilesComplete"].(float64)) != 1 ||
		int(presentTotals["sourceFilesIncomplete"].(float64)) != 1 {
		t.Fatalf("expected source-status present worklist to keep only captured records, got %#v", presentTotals)
	}
	presentPacket := presentOnly["operatorPacket"].(map[string]any)
	if int(presentPacket["artifactReplacementCount"].(float64)) != 2 ||
		!strings.Contains(presentPacket["suggestedWorklistPath"].(string), "source-present") ||
		!strings.Contains(presentPacket["saveWorklistCommand"].(string), "--source-status present") {
		t.Fatalf("expected present worklist operator packet to keep importable source filter, got %#v", presentPacket)
	}
	presentRecords := presentOnly["queues"].([]any)[0].(map[string]any)["batches"].([]any)[0].(map[string]any)["operatorBatch"].(map[string]any)["artifactReplacementRecords"].([]any)
	if len(presentRecords) != 2 ||
		presentRecords[0].(map[string]any)["suggestedSourceExists"] != true ||
		presentRecords[1].(map[string]any)["suggestedSourceExists"] != true {
		t.Fatalf("expected one present source replacement record, got %#v", presentRecords)
	}
	missingArgs := append(append([]string{}, worklistArgs...), "--source-status", "missing")
	stdout.Reset()
	if err := Handle("scan", "msapi", missingArgs, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var missingOnly map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &missingOnly); err != nil {
		t.Fatal(err)
	}
	missingTotals := missingOnly["totals"].(map[string]any)
	if int(missingTotals["artifacts"].(float64)) != 8 || int(missingTotals["sourceFilesPresent"].(float64)) != 0 || int(missingTotals["sourceFilesMissing"].(float64)) != 8 {
		t.Fatalf("expected source-status missing worklist to keep uncaptured records, got %#v", missingTotals)
	}
	for _, item := range batch["artifacts"].([]any) {
		artifact := item.(map[string]any)
		if artifact["artifactType"] != "query-readback" ||
			!evidenceSectionHasStatus(artifact["evidenceSectionStatuses"].([]any), "readbackTables", "missing") {
			t.Fatalf("expected filtered artifacts to stay within query-readback readbackTables work, got %#v", artifact)
		}
	}
}

func TestSetupSvcLiveReplayGapsPagesFilteredCollectionPlanArtifacts(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-workspace", "--execute", "--approval", setupSvcParityEvidenceWorkspaceApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-gaps", "--artifact-type", "query-readback", "--offset", "1", "--limit", "3"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	plan := result["collectionPlan"].(map[string]any)
	if int(plan["nextArtifactOffset"].(float64)) != 1 ||
		int(plan["nextArtifactLimit"].(float64)) != 3 ||
		int(plan["totalNextArtifacts"].(float64)) != setupSvcLiveReplayOperationCount() ||
		int(plan["omittedNextArtifacts"].(float64)) != setupSvcLiveReplayOperationCount()-4 {
		t.Fatalf("expected paged query-readback collection plan, got %#v", plan)
	}
	filters := plan["filters"].(map[string]any)
	if filters["artifactType"] != "query-readback" {
		t.Fatalf("expected query-readback filter in collection plan, got %#v", filters)
	}
	pageCommands := plan["pageCommands"].(map[string]any)
	currentPage := pageCommands["currentPage"].(string)
	nextPage := pageCommands["nextPage"].(string)
	previousPage := pageCommands["previousPage"].(string)
	if !strings.Contains(currentPage, "--artifact-type query-readback --offset 1 --limit 3") ||
		!strings.Contains(nextPage, "--artifact-type query-readback --offset 4 --limit 3") ||
		!strings.Contains(previousPage, "--artifact-type query-readback --offset 0 --limit 3") {
		t.Fatalf("expected collection page commands to preserve filters and page offsets, got %#v", pageCommands)
	}
	nextArtifacts := plan["nextArtifacts"].([]any)
	if len(nextArtifacts) != 3 {
		t.Fatalf("expected exactly three paged artifacts, got %#v", nextArtifacts)
	}
	for _, item := range nextArtifacts {
		action := item.(map[string]any)
		if action["artifactType"] != "query-readback" || action["requiredShapeKey"] != "requiredReadbackShape" ||
			action["manifestStatusField"] != "queryEvidenceStatus" ||
			!containsStringItem(action["requiredEvidenceSections"].([]any), "readbackExpectationChecks") ||
			!evidenceSectionHasStatus(action["evidenceSectionStatuses"].([]any), "readbackExpectationChecks", "missing") ||
			!containsStringFragment(action["replacementChecklist"].([]any), "readback table coverage") {
			t.Fatalf("expected paged actions to stay within query-readback contract, got %#v", action)
		}
	}
}

func TestSetupSvcLiveReplayGapsFiltersByEvidenceSectionStatus(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-workspace", "--execute", "--approval", setupSvcParityEvidenceWorkspaceApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-gaps", "--artifact-type", "query-readback", "--evidence-section", "readbackTables", "--section-status", "missing", "--offset", "1", "--limit", "3"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	plan := result["collectionPlan"].(map[string]any)
	if int(plan["totalNextArtifacts"].(float64)) != setupSvcLiveReplayOperationCount() ||
		int(plan["omittedNextArtifacts"].(float64)) != setupSvcLiveReplayOperationCount()-4 {
		t.Fatalf("expected readbackTables missing filter to match every query-readback artifact, got %#v", plan)
	}
	filters := plan["filters"].(map[string]any)
	if filters["artifactType"] != "query-readback" ||
		filters["evidenceSection"] != "readbackTables" ||
		filters["sectionStatus"] != "missing" {
		t.Fatalf("expected evidence section filters in collection plan, got %#v", filters)
	}
	readbackTablesQueue := evidenceSectionQueue(plan["missingSectionQueues"].([]any), "query-readback", "readbackTables")
	if readbackTablesQueue == nil ||
		int(readbackTablesQueue["pageSize"].(float64)) != 3 ||
		int(readbackTablesQueue["batchCount"].(float64)) != (setupSvcLiveReplayOperationCount()+2)/3 ||
		len(readbackTablesQueue["batchCommands"].([]any)) != 10 ||
		int(readbackTablesQueue["omittedBatchCommands"].(float64)) != (setupSvcLiveReplayOperationCount()+2)/3-10 ||
		!containsStringFragment(readbackTablesQueue["batchCommands"].([]any), "--offset 27 --limit 3") ||
		readbackTablesQueue["requiredShapeKey"] != "requiredReadbackShape" ||
		readbackTablesQueue["manifestStatusField"] != "queryEvidenceStatus" {
		t.Fatalf("expected filtered missingSectionQueues to preserve batch size and readback shape metadata, got %#v", plan["missingSectionQueues"])
	}
	pageCommands := plan["pageCommands"].(map[string]any)
	for _, key := range []string{"currentPage", "nextPage", "previousPage"} {
		command := pageCommands[key].(string)
		if !strings.Contains(command, "--artifact-type query-readback") ||
			!strings.Contains(command, "--evidence-section readbackTables") ||
			!strings.Contains(command, "--section-status missing") {
			t.Fatalf("expected page command %s to preserve evidence section filters, got %s", key, command)
		}
	}
	nextArtifacts := plan["nextArtifacts"].([]any)
	if len(nextArtifacts) != 3 {
		t.Fatalf("expected three filtered next artifacts, got %#v", nextArtifacts)
	}
	for _, item := range nextArtifacts {
		action := item.(map[string]any)
		if action["artifactType"] != "query-readback" ||
			!evidenceSectionHasStatus(action["evidenceSectionStatuses"].([]any), "readbackTables", "missing") {
			t.Fatalf("expected filtered action to require missing readbackTables proof, got %#v", action)
		}
	}
}

func TestSetupSvcLiveReplayGapsReportMissingRuntimeReadbackEvidence(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayMissingRuntimeReadbackEvidenceArtifacts(t, tmp, manifestPath)

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-gaps"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("gap scan should block missing runtime/readback evidence, got %#v", result)
	}
	totals := result["totals"].(map[string]any)
	if int(totals["failedOperations"].(float64)) != 1 || int(totals["failedArtifacts"].(float64)) != 2 {
		t.Fatalf("expected one failed operation with two failed artifact issues, got %#v", totals)
	}
	firstOperation := result["domains"].([]any)[0].(map[string]any)["operations"].([]any)[0].(map[string]any)
	if firstOperation["status"] != "failed_evidence" || firstOperation["nextAction"] != "repair_failed_evidence" {
		t.Fatalf("expected failed evidence gap operation, got %#v", firstOperation)
	}
	if !containsStringItem(firstOperation["failedEvidence"].([]any), "runtimeEffectsMissingEvidence=datatable-prefix-allocation") ||
		!containsStringItem(firstOperation["failedEvidence"].([]any), "queryReadbackExpectationsMissingEvidence=object-identity-prefix-datatable-readback") {
		t.Fatalf("expected runtime/readback failed evidence items, got %#v", firstOperation["failedEvidence"])
	}
}

func TestSetupSvcLiveReplayGapsReportsReadyForNormalizedDiff(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	markSetupSvcLiveReplayManifestOperationStatus(t, manifestPath, "objects", "create", "normalizedDiffStatus", "pending")
	if err := os.Remove(filepath.Join(tmp, "outputs/setup-svc-live-replay/objects/create/normalized-diff.json")); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-gaps"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "ready_for_normalized_diff" {
		t.Fatalf("source snapshots with pending diff should be ready for normalized diff, got %#v", result)
	}
	totals := result["totals"].(map[string]any)
	if int(totals["readyForDiffOperations"].(float64)) != 1 {
		t.Fatalf("expected one operation ready for diff, got %#v", totals)
	}
	firstDomain := result["domains"].([]any)[0].(map[string]any)
	firstOperation := firstDomain["operations"].([]any)[0].(map[string]any)
	if firstOperation["status"] != "ready_for_normalized_diff" || firstOperation["nextAction"] != "generate_normalized_diff" {
		t.Fatalf("expected objects/create to request normalized diff, got %#v", firstOperation)
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksUnexpectedManifestDomain(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	rewriteSetupSvcLiveReplayManifest(t, manifestPath, func(manifest map[string]any) {
		manifest["domains"] = append(manifest["domains"].([]any), map[string]any{
			"domain":     "unsupported-domain",
			"operations": []any{},
		})
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("unexpected manifest domain must block evidence, got %#v", result)
	}
	if !containsStringItem(result["blockingIssues"].([]any), "manifest: unexpected domain unsupported-domain") {
		t.Fatalf("expected unexpected domain issue, got %#v", result["blockingIssues"])
	}
}

func TestSetupSvcLiveReplayEvidenceBlocksUnexpectedManifestOperation(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	rewriteSetupSvcLiveReplayManifest(t, manifestPath, func(manifest map[string]any) {
		domain := manifest["domains"].([]any)[0].(map[string]any)
		domain["operations"] = append(domain["operations"].([]any), map[string]any{
			"operation": "unsupported-operation",
		})
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-evidence"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("unexpected manifest operation must block evidence, got %#v", result)
	}
	domain := result["domains"].([]any)[0].(map[string]any)
	failed := domain["failedOperations"].([]any)[0].(map[string]any)
	if failed["operation"] != "unsupported-operation" || !containsStringItem(failed["failedEvidence"].([]any), "unexpectedOperation") {
		t.Fatalf("expected unexpected operation failure, got %#v", failed)
	}
}

func TestSetupSvcLiveReplayPromotionBlocksGlobalManifestContractFailure(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	rewriteSetupSvcLiveReplayManifest(t, manifestPath, func(manifest map[string]any) {
		manifest["contractFingerprint"] = "sha256:wrong"
	})

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-promotion"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("global manifest contract failure must block promotion, got %#v", result)
	}
	totals := result["totals"].(map[string]any)
	if int(totals["promotableDomains"].(float64)) != 0 || int(totals["blockedDomains"].(float64)) != len(setupSvcLiveReplayDomains()) {
		t.Fatalf("expected no promotable domains under global block, got %#v", totals)
	}
	if updates, ok := result["matrixUpdates"].([]any); ok && len(updates) > 0 {
		t.Fatalf("blocked manifest contract should not emit matrix updates, got %#v", updates)
	}
}

func TestSetupSvcLiveReplayPromotionBlocksMissingRuntimeReadbackEvidence(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayMissingRuntimeReadbackEvidenceArtifacts(t, tmp, manifestPath)

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-promotion"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("missing runtime/readback evidence must block promotion, got %#v", result)
	}
	totals := result["totals"].(map[string]any)
	if int(totals["promotableDomains"].(float64)) != 0 || int(totals["failedOperations"].(float64)) != 1 {
		t.Fatalf("expected one failed evidence operation and no promotable domains, got %#v", totals)
	}
	if updates, ok := result["matrixUpdates"].([]any); ok && len(updates) > 0 {
		t.Fatalf("failed evidence must not emit matrix updates, got %#v", updates)
	}
	issues := result["blockingIssues"].([]any)
	if !containsStringFragment(issues, "runtimeEffectsMissingEvidence=datatable-prefix-allocation") ||
		!containsStringFragment(issues, "queryReadbackExpectationsMissingEvidence=object-identity-prefix-datatable-readback") {
		t.Fatalf("expected runtime/readback blockers in promotion result, got %#v", issues)
	}
}

func TestSetupSvcLiveReplayPromotionPromotesOnlyVerifiedDomains(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-promotion"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "passed" || result["readOnly"] != true {
		t.Fatalf("expected passed read-only promotion result, got %#v", result)
	}
	totals := result["totals"].(map[string]any)
	if int(totals["promotableDomains"].(float64)) != len(setupSvcLiveReplayDomains()) {
		t.Fatalf("expected all domains promotable, got %#v", totals)
	}
	if int(totals["blockedDomains"].(float64)) != 0 || int(totals["failedOperations"].(float64)) != 0 {
		t.Fatalf("expected no blocked promotion items, got %#v", totals)
	}
	matrixUpdates := result["matrixUpdates"].([]any)
	if len(matrixUpdates) != len(setupSvcLiveReplayDomains()) {
		t.Fatalf("expected one matrix update per domain, got %d", len(matrixUpdates))
	}
	first := matrixUpdates[0].(map[string]any)
	if first["fromStatus"] != "covered" || first["toStatus"] != "verified" {
		t.Fatalf("expected covered->verified matrix update, got %#v", first)
	}
}

func TestSetupSvcLiveReplayPromotionRespectsParityMatrixStatus(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	writeSetupSvcLiveReplayParityMatrix(t, tmp, map[string]string{"objects": "partial"})
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-promotion"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "partial" {
		t.Fatalf("expected partial promotion result when matrix still has partial domains, got %#v", result)
	}
	totals := result["totals"].(map[string]any)
	if int(totals["promotableDomains"].(float64)) != len(setupSvcLiveReplayDomains())-1 || int(totals["blockedDomains"].(float64)) != 1 {
		t.Fatalf("expected exactly one matrix-status blocked domain, got %#v", totals)
	}
	if !containsStringItem(result["blockingIssues"].([]any), "objects: matrix status partial cannot promote to verified") {
		t.Fatalf("expected matrix status blocking issue, got %#v", result["blockingIssues"])
	}
	for _, raw := range result["matrixUpdates"].([]any) {
		update := raw.(map[string]any)
		if update["domain"] == "objects" {
			t.Fatalf("partial matrix domain must not emit verified update, got %#v", update)
		}
	}
	foundObjects := false
	for _, raw := range result["domains"].([]any) {
		domain := raw.(map[string]any)
		if domain["domain"] != "objects" {
			continue
		}
		foundObjects = true
		if domain["currentMatrixStatus"] != "partial" || domain["recommendedStatus"] != "partial" || domain["canPromote"] != false {
			t.Fatalf("expected objects to remain partial and non-promotable, got %#v", domain)
		}
	}
	if !foundObjects {
		t.Fatalf("objects domain missing from promotion result: %#v", result["domains"])
	}
}

func TestSetupSvcLiveReplayPromotionUsesBundledSkillMatrix(t *testing.T) {
	tmp := t.TempDir()
	projectPath := filepath.Join(tmp, "project")
	skillRoot := filepath.Join(tmp, "skill")
	otherCWD := filepath.Join(tmp, "other-cwd")
	writeTestFile(t, filepath.Join(projectPath, "cloudcc-cli.config.json"), `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	writeSetupSvcLiveReplayParityMatrix(t, skillRoot, map[string]string{"objects": "partial"})
	manifestPath := filepath.Join(projectPath, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	if err := os.MkdirAll(otherCWD, 0755); err != nil {
		t.Fatal(err)
	}
	oldCWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(otherCWD); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldCWD)
	})
	t.Setenv("CLOUDCC_MSAPI_SKILL_ROOT", skillRoot)

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{projectPath, "setup-svc-live-replay-promotion"}, &stdout, projectPath); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "partial" {
		t.Fatalf("expected bundled skill matrix to block one partial domain, got %#v", result)
	}
	if !containsStringItem(result["blockingIssues"].([]any), "objects: matrix status partial cannot promote to verified") {
		t.Fatalf("expected bundled matrix status blocking issue, got %#v", result["blockingIssues"])
	}
}

func TestSetupSvcLiveReplayPromotionBlocksIncompleteManifest(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, false)

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-promotion"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked" {
		t.Fatalf("expected blocked promotion result, got %#v", result)
	}
	totals := result["totals"].(map[string]any)
	if int(totals["promotableDomains"].(float64)) != 0 {
		t.Fatalf("incomplete manifest must not produce promotable domains, got %#v", totals)
	}
	if int(totals["blockedDomains"].(float64)) == 0 || int(totals["missingOperations"].(float64)) == 0 {
		t.Fatalf("expected blocked/missing promotion totals, got %#v", totals)
	}
	if updates, ok := result["matrixUpdates"].([]any); ok && len(updates) > 0 {
		t.Fatalf("blocked manifest should not emit matrix updates, got %#v", updates)
	}
}

func TestSetupSvcLiveReplayMatrixPromotionDryRunDoesNotModifyMatrix(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	writeSetupSvcLiveReplayParityMatrix(t, tmp, nil)
	matrixPath := filepath.Join(tmp, "cc-metadata-service/src/test/resources/parity/msapi-setup-svc-parity-matrix.json")
	before, err := os.ReadFile(matrixPath)
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeCurrentSetupSvcLiveReplayEvidenceBundle(t, tmp)

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-promotion", "--dry-run"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "dry_run_ready" || result["readOnly"] != true {
		t.Fatalf("expected dry-run ready matrix promotion, got %#v", result)
	}
	totals := result["totals"].(map[string]any)
	if int(totals["candidateUpdates"].(float64)) != len(setupSvcLiveReplayDomains()) || int(totals["appliedUpdates"].(float64)) != 0 {
		t.Fatalf("expected candidate updates without writes, got %#v", totals)
	}
	after, err := os.ReadFile(matrixPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatalf("dry-run must not modify matrix")
	}
}

func TestSetupSvcLiveReplayMatrixPromotionBlocksMissingEvidenceBundle(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	writeSetupSvcLiveReplayParityMatrix(t, tmp, nil)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-promotion", "--dry-run"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked_evidence_bundle" {
		t.Fatalf("missing bundle must block matrix promotion, got %#v", result)
	}
	totals := result["totals"].(map[string]any)
	if int(totals["candidateUpdates"].(float64)) != len(setupSvcLiveReplayDomains()) ||
		int(totals["blockedUpdates"].(float64)) != len(setupSvcLiveReplayDomains()) {
		t.Fatalf("expected candidate updates blocked by missing bundle, got %#v", totals)
	}
	if !containsStringFragment(result["blockingIssues"].([]any), "evidenceBundle: missing") {
		t.Fatalf("expected missing evidenceBundle blocker, got %#v", result["blockingIssues"])
	}
}

func TestSetupSvcLiveReplayMatrixPromotionExecuteRequiresApproval(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	writeSetupSvcLiveReplayParityMatrix(t, tmp, nil)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)

	var stdout bytes.Buffer
	err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-promotion", "--execute", "--approval", "WRONG"}, &stdout, tmp)
	if err == nil || !strings.Contains(err.Error(), setupSvcParityMatrixPromotionApproval) {
		t.Fatalf("expected matrix promotion approval error, got %v", err)
	}
}

func TestSetupSvcLiveReplayMatrixPromotionExecuteUpdatesMatrixAndCompletionPasses(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	writeSetupSvcLiveReplayParityMatrix(t, tmp, nil)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeCurrentSetupSvcLiveReplayEvidenceBundle(t, tmp)

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-promotion", "--execute", "--approval", setupSvcParityMatrixPromotionApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var applyResult map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &applyResult); err != nil {
		t.Fatal(err)
	}
	if applyResult["status"] != "applied" || applyResult["readOnly"] != false {
		t.Fatalf("expected matrix promotion applied, got %#v", applyResult)
	}
	totals := applyResult["totals"].(map[string]any)
	if int(totals["appliedUpdates"].(float64)) != len(setupSvcLiveReplayDomains()) {
		t.Fatalf("expected every domain updated, got %#v", totals)
	}
	statuses := setupSvcLiveReplayMatrixStatuses(tmp)
	for _, domain := range setupSvcLiveReplayDomains() {
		if statuses[normalizeDomain(domain.Domain)] != "verified" {
			t.Fatalf("expected %s verified after matrix promotion, got %s", domain.Domain, statuses[normalizeDomain(domain.Domain)])
		}
	}

	stdout.Reset()
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-completion-audit"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var completion map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &completion); err != nil {
		t.Fatal(err)
	}
	if completion["status"] != "passed" {
		t.Fatalf("completion audit should pass after approved matrix promotion, got %#v", completion)
	}
}

func TestSetupSvcLiveReplayMatrixPromotionBlocksIncompleteManifest(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	writeSetupSvcLiveReplayParityMatrix(t, tmp, nil)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, false)

	var stdout bytes.Buffer
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-promotion", "--dry-run"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked_promotion_audit" {
		t.Fatalf("incomplete manifest must block matrix promotion, got %#v", result)
	}
	totals := result["totals"].(map[string]any)
	if int(totals["candidateUpdates"].(float64)) != 0 {
		t.Fatalf("incomplete manifest should not produce matrix updates, got %#v", totals)
	}
}

func TestSetupSvcLiveReplayCompletionAuditBlocksMissingManifest(t *testing.T) {
	t.Setenv("MDS_JDBC_URL", "")
	t.Setenv("MDS_DB_USERNAME", "")
	t.Setenv("MDS_DB_PASSWORD", "")
	t.Setenv("MDS_DB_DRIVER", "")
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	writeSetupSvcLiveReplayParityMatrix(t, tmp, nil)

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-completion-audit"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked_missing_live_replay_evidence" || result["readOnly"] != true {
		t.Fatalf("missing manifest must block completion audit, got %#v", result)
	}
	if !containsStringItem(result["blockingIssues"].([]any), "manifest: missing live replay evidence "+filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")) {
		t.Fatalf("expected missing manifest blocker, got %#v", result["blockingIssues"])
	}
	gates := completionAuditGatesByName(result)
	if gates["test_evidence_contract"]["status"] != "passed" || gates["test_evidence_contract"]["blocking"] == true {
		t.Fatalf("test evidence gate should pass before waiting for live evidence, got %#v", gates["test_evidence_contract"])
	}
	if gates["matrix_contract"]["status"] != "passed" || gates["live_replay_evidence"]["status"] != "missing" {
		t.Fatalf("expected matrix pass and evidence missing gates, got %#v", gates)
	}
	datasource := result["metadataServiceDatasource"].(map[string]any)
	if datasource["status"] != "missing_real_datasource" || datasource["readyForRealDatasource"] != false {
		t.Fatalf("completion audit should expose missing real datasource, got %#v", datasource)
	}
	missing := datasource["missing"].([]any)
	if !containsAnyString(missing, "MDS_JDBC_URL") ||
		!containsAnyString(missing, "MDS_DB_USERNAME") ||
		!containsAnyString(missing, "MDS_DB_PASSWORD") ||
		!containsAnyString(missing, "MDS_DB_DRIVER") {
		t.Fatalf("expected missing datasource variables in completion audit, got %#v", missing)
	}
	totals := result["totals"].(map[string]any)
	if int(totals["blockedDomains"].(float64)) != len(setupSvcLiveReplayDomains()) {
		t.Fatalf("missing evidence should block every domain, got %#v", totals)
	}
}

func TestSetupSvcLiveReplayCompletionAuditRequiresMatrixStatusUpdate(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	writeSetupSvcLiveReplayParityMatrix(t, tmp, nil)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-completion-audit"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked_evidence_bundle" {
		t.Fatalf("passed evidence without bundle must block completion audit, got %#v", result)
	}
	gates := completionAuditGatesByName(result)
	if gates["evidence_bundle"]["status"] != "missing" || gates["evidence_bundle"]["blocking"] != true {
		t.Fatalf("missing bundle gate must block completion, got %#v", gates["evidence_bundle"])
	}
	commands := result["nextCommands"].([]any)
	if !containsStringFragment(commands, "setup-svc-live-replay-evidence-bundle") ||
		!containsStringFragment(commands, "--dry-run") ||
		!containsStringFragment(commands, setupSvcParityEvidenceBundleApproval) ||
		!containsStringFragment(commands, "setup-svc-live-replay-promotion") ||
		!containsStringFragment(commands, "setup-svc-live-replay-completion-audit") {
		t.Fatalf("missing bundle completion audit should expose bundle and follow-up commands, got %#v", commands)
	}

	writeCurrentSetupSvcLiveReplayEvidenceBundle(t, tmp)
	stdout.Reset()
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-completion-audit"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "ready_for_matrix_status_update" {
		t.Fatalf("passed evidence with current bundle and covered matrix must be ready for matrix update, got %#v", result)
	}
	totals := result["totals"].(map[string]any)
	if int(totals["evidenceVerifiedDomains"].(float64)) != len(setupSvcLiveReplayDomains()) ||
		int(totals["promotableDomains"].(float64)) != len(setupSvcLiveReplayDomains()) ||
		int(totals["matrixNonVerifiedDomains"].(float64)) != len(setupSvcLiveReplayDomains()) {
		t.Fatalf("expected verified evidence, promotable covered matrix domains, got %#v", totals)
	}
	gates = completionAuditGatesByName(result)
	if gates["matrix_status"]["status"] != "pending_update" || gates["matrix_status"]["blocking"] != true {
		t.Fatalf("covered matrix must block completion until status update, got %#v", gates["matrix_status"])
	}
	if gates["evidence_bundle"]["status"] != "passed" {
		t.Fatalf("current bundle gate should pass, got %#v", gates["evidence_bundle"])
	}
	commands = result["nextCommands"].([]any)
	if !containsStringFragment(commands, "setup-svc-live-replay-promotion") ||
		!containsStringFragment(commands, "--dry-run") ||
		!containsStringFragment(commands, setupSvcParityMatrixPromotionApproval) ||
		!containsStringFragment(commands, "setup-svc-live-replay-completion-audit") ||
		containsStringFragment(commands, setupSvcParityEvidenceBundleApproval) {
		t.Fatalf("ready matrix update audit should expose promotion commands without bundle rewrite, got %#v", commands)
	}
	firstDomain := result["domains"].([]any)[0].(map[string]any)
	if firstDomain["completionStatus"] != "ready_for_matrix_update" || firstDomain["canPromote"] != true {
		t.Fatalf("expected domain ready for matrix update, got %#v", firstDomain)
	}
}

func TestSetupSvcLiveReplayCompletionAuditReportsMissingRuntimeReadbackEvidence(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	writeSetupSvcLiveReplayParityMatrix(t, tmp, nil)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayMissingRuntimeReadbackEvidenceArtifacts(t, tmp, manifestPath)

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-completion-audit"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked_live_replay_evidence" {
		t.Fatalf("missing runtime/readback evidence must block completion audit, got %#v", result)
	}
	gates := completionAuditGatesByName(result)
	if gates["live_replay_evidence"]["status"] != "blocked" || gates["live_replay_evidence"]["blocking"] != true {
		t.Fatalf("expected blocking live evidence gate, got %#v", gates["live_replay_evidence"])
	}
	commands := result["nextCommands"].([]any)
	if !containsStringFragment(commands, "--evidence-section") ||
		!containsStringFragment(commands, "setup-svc-live-replay-manifest-sync") ||
		!containsStringFragment(commands, "setup-svc-live-replay-evidence-bundle") ||
		containsStringFragment(commands, "setup-svc-live-replay-workspace") {
		t.Fatalf("expected completion audit to expose evidence collection runbook commands, got %#v", commands)
	}
	issues := result["blockingIssues"].([]any)
	if !containsStringFragment(issues, "objects/create: runtimeEffectsMissingEvidence=datatable-prefix-allocation") ||
		!containsStringFragment(issues, "objects/create: queryReadbackExpectationsMissingEvidence=object-identity-prefix-datatable-readback") {
		t.Fatalf("expected runtime/readback blockers in completion audit, got %#v", issues)
	}
	topFailedEvidence := result["failedEvidence"].([]any)
	if !containsStringFragment(topFailedEvidence, "objects/create: runtimeEffectsMissingEvidence=datatable-prefix-allocation") ||
		!containsStringFragment(topFailedEvidence, "objects/create: queryReadbackExpectationsMissingEvidence=object-identity-prefix-datatable-readback") {
		t.Fatalf("expected top-level failedEvidence details for machine repair queues, got %#v", topFailedEvidence)
	}
	failedEvidenceSummary := result["failedEvidenceSummary"].(map[string]any)
	if int(failedEvidenceSummary["total"].(float64)) != len(topFailedEvidence) ||
		!containsMapCount(failedEvidenceSummary["issueCounts"].([]any), "runtimeEffectsMissingEvidence", 1) ||
		!containsMapCount(failedEvidenceSummary["issueCounts"].([]any), "queryReadbackExpectationsMissingEvidence", 1) ||
		!containsDomainOperationCount(failedEvidenceSummary["domainOperationCounts"].([]any), "objects", "create", 2) {
		t.Fatalf("expected completion audit failedEvidenceSummary for repair automation, got %#v", failedEvidenceSummary)
	}
	repairQueues := failedEvidenceSummary["repairQueues"].([]any)
	if int(failedEvidenceSummary["repairQueueCount"].(float64)) != len(repairQueues) {
		t.Fatalf("expected failedEvidenceSummary repairQueueCount mirror, got %#v", failedEvidenceSummary)
	}
	repairPlan := result["repairPlan"].(map[string]any)
	if int(repairPlan["repairQueueCount"].(float64)) != len(repairQueues) ||
		int(repairPlan["totalSourceFiles"].(float64)) != len(topFailedEvidence) ||
		int(repairPlan["totalTargetFiles"].(float64)) != len(topFailedEvidence) ||
		repairPlan["primarySourceSystem"] == "" ||
		repairPlan["primaryEvidenceSection"] == "" ||
		!strings.Contains(repairPlan["primaryCommand"].(string), "setup-svc-live-replay-source-execution-packet") ||
		len(repairPlan["nextRepairCommands"].([]any)) != len(repairQueues) ||
		!containsStringFragment(repairPlan["nextRepairCommands"].([]any), "setup-svc-live-replay-source-execution-packet") ||
		len(repairPlan["postRepairCommands"].([]any)) != 6 ||
		!containsStringWithFragments(repairPlan["postRepairCommands"].([]any), "setup-svc-live-replay-source-validate", "--source-readiness complete") ||
		!containsStringWithFragments(repairPlan["postRepairCommands"].([]any), "setup-svc-live-replay-worklist", "--source-readiness complete", "--batch-index 0", " > ") ||
		!containsStringWithFragments(repairPlan["postRepairCommands"].([]any), "setup-svc-live-replay-evidence-import @", "--dry-run") ||
		!containsStringWithFragments(repairPlan["postRepairCommands"].([]any), "setup-svc-live-replay-evidence-import @", "--execute", setupSvcParityEvidenceImportApproval) ||
		!containsStringWithFragments(repairPlan["postRepairCommands"].([]any), "setup-svc-live-replay-manifest-sync", "--dry-run") ||
		!containsStringFragment(repairPlan["postRepairCommands"].([]any), "setup-svc-live-replay-completion-audit") ||
		!strings.Contains(repairPlan["nextRepairScript"].(string), "set -euo pipefail") ||
		!strings.Contains(repairPlan["nextRepairScript"].(string), "setup-svc-live-replay-source-execution-packet") ||
		!strings.Contains(repairPlan["nextRepairScriptPath"].(string), "repair-plan-next-repair-commands.sh") ||
		!strings.Contains(repairPlan["saveNextRepairScriptCommand"].(string), ".repairPlan.nextRepairScript") ||
		len(repairPlan["groups"].([]any)) != len(repairQueues) ||
		len(repairPlan["domainOperations"].([]any)) == 0 {
		t.Fatalf("expected completion repairPlan to summarize repair queues, got %#v queues=%#v", repairPlan, repairQueues)
	}
	objectsCreateRepairPlan := repairPlanDomainOperation(repairPlan["domainOperations"].([]any), "objects", "create")
	if objectsCreateRepairPlan == nil ||
		int(objectsCreateRepairPlan["failedEvidenceCount"].(float64)) != 2 ||
		objectsCreateRepairPlan["primaryRepairQueue"] == "" ||
		!containsStringFragment(objectsCreateRepairPlan["repairQueues"].([]any), "setup-svc/runtimeEffectChecks") ||
		!containsStringFragment(objectsCreateRepairPlan["repairQueues"].([]any), "query-readback/readbackExpectationChecks") {
		t.Fatalf("expected repairPlan domainOperations to map objects/create to concrete repair queues, got %#v", repairPlan)
	}
	if int(result["failedEvidenceTotal"].(float64)) != len(topFailedEvidence) ||
		int(result["repairQueueCount"].(float64)) != len(repairQueues) {
		t.Fatalf("expected top-level failed evidence and repair queue mirrors, got %#v", result)
	}
	totals := result["totals"].(map[string]any)
	if int(totals["failedEvidenceTotal"].(float64)) != len(topFailedEvidence) ||
		int(totals["repairQueueCount"].(float64)) != len(repairQueues) {
		t.Fatalf("expected totals failed evidence and repair queue mirrors, got %#v", totals)
	}
	gates = completionAuditGatesByName(result)
	matrixSummary := gates["matrix_contract"]["summary"].(map[string]any)
	if matrixSummary["status"] != "passed" ||
		int(matrixSummary["matrixCoveredDomains"].(float64)) != len(setupSvcLiveReplayDomains()) {
		t.Fatalf("expected completion matrix gate summary, got %#v", gates["matrix_contract"])
	}
	testSummary := gates["test_evidence_contract"]["summary"].(map[string]any)
	if testSummary["status"] != "passed" ||
		int(testSummary["operations"].(float64)) != setupSvcLiveReplayOperationCount() {
		t.Fatalf("expected completion test evidence gate summary, got %#v", gates["test_evidence_contract"])
	}
	liveSummary := gates["live_replay_evidence"]["summary"].(map[string]any)
	if int(liveSummary["failedEvidenceTotal"].(float64)) != len(topFailedEvidence) ||
		int(liveSummary["repairQueueCount"].(float64)) != len(repairQueues) {
		t.Fatalf("expected completion live evidence gate summary, got %#v", gates["live_replay_evidence"])
	}
	promotionSummary := gates["promotion_audit"]["summary"].(map[string]any)
	if promotionSummary["status"] == "" {
		t.Fatalf("expected completion promotion gate summary, got %#v", gates["promotion_audit"])
	}
	gateSummaries := result["gateSummaries"].(map[string]any)
	topLiveSummary := gateSummaries["live_replay_evidence"].(map[string]any)
	if int(topLiveSummary["failedEvidenceTotal"].(float64)) != len(topFailedEvidence) ||
		int(topLiveSummary["repairQueueCount"].(float64)) != len(repairQueues) {
		t.Fatalf("expected top-level completion gateSummaries mirror, got %#v", gateSummaries)
	}
	topMatrixSummary := gateSummaries["matrix_contract"].(map[string]any)
	if int(topMatrixSummary["matrixCoveredDomains"].(float64)) != len(setupSvcLiveReplayDomains()) {
		t.Fatalf("expected top-level matrix gate summary mirror, got %#v", gateSummaries)
	}
	gateStatuses := result["gateStatuses"].(map[string]any)
	liveStatus := gateStatuses["live_replay_evidence"].(map[string]any)
	if liveStatus["status"] != "blocked" || liveStatus["blocking"] != true {
		t.Fatalf("expected top-level completion gateStatuses mirror, got %#v", gateStatuses)
	}
	matrixStatus := gateStatuses["matrix_contract"].(map[string]any)
	if matrixStatus["status"] != "passed" || matrixStatus["blocking"] == true {
		t.Fatalf("expected top-level matrix gate status mirror, got %#v", gateStatuses)
	}
	operatorPacket := result["operatorPacket"].(map[string]any)
	operatorLiveStatus := operatorPacket["gateStatuses"].(map[string]any)["live_replay_evidence"].(map[string]any)
	operatorLiveSummary := operatorPacket["gateSummaries"].(map[string]any)["live_replay_evidence"].(map[string]any)
	if operatorPacket["status"] != result["status"] ||
		operatorLiveStatus["status"] != "blocked" ||
		operatorLiveStatus["blocking"] != true ||
		int(operatorLiveSummary["failedEvidenceTotal"].(float64)) != len(topFailedEvidence) ||
		len(operatorPacket["failedEvidence"].([]any)) != len(topFailedEvidence) ||
		!containsStringFragment(operatorPacket["failedEvidence"].([]any), "objects/create: queryReadbackExpectationsMissingEvidence=object-identity-prefix-datatable-readback") ||
		int(operatorPacket["failedEvidenceTotal"].(float64)) != len(topFailedEvidence) ||
		int(operatorPacket["repairQueueCount"].(float64)) != len(repairQueues) ||
		int(operatorPacket["repairPlan"].(map[string]any)["repairQueueCount"].(float64)) != len(repairQueues) ||
		len(operatorPacket["repairQueues"].([]any)) != len(repairQueues) ||
		int(operatorPacket["blockedDomains"].(float64)) == 0 ||
		len(operatorPacket["domains"].([]any)) != len(result["domains"].([]any)) ||
		!containsStringFragment(operatorPacket["blockingIssues"].([]any), "manifest: blocked") ||
		!containsStringFragment(operatorPacket["blockingIssues"].([]any), "objects/create: runtimeEffectsMissingEvidence=datatable-prefix-allocation") ||
		!containsStringFragment(operatorPacket["nextCommands"].([]any), "setup-svc-live-replay-evidence-bundle") {
		t.Fatalf("expected completion operatorPacket to mirror final gate machine state, got %#v", operatorPacket)
	}
	operatorRepairQueues := operatorPacket["repairQueues"].([]any)
	if !containsRepairQueue(operatorRepairQueues, "setup-svc", "runtimeEffectChecks") ||
		!containsRepairQueue(operatorRepairQueues, "query-readback", "readbackExpectationChecks") {
		t.Fatalf("expected completion operatorPacket to carry repair queues, got %#v", operatorPacket)
	}
	operatorRepairPlan := operatorPacket["repairPlan"].(map[string]any)
	if int(operatorRepairPlan["totalSourceFiles"].(float64)) != int(repairPlan["totalSourceFiles"].(float64)) ||
		len(operatorRepairPlan["groups"].([]any)) != len(repairQueues) ||
		len(operatorRepairPlan["nextRepairCommands"].([]any)) != len(repairPlan["nextRepairCommands"].([]any)) ||
		len(operatorRepairPlan["postRepairCommands"].([]any)) != len(repairPlan["postRepairCommands"].([]any)) ||
		operatorRepairPlan["nextRepairScriptPath"] != repairPlan["nextRepairScriptPath"] ||
		len(operatorRepairPlan["domainOperations"].([]any)) != len(repairPlan["domainOperations"].([]any)) {
		t.Fatalf("expected completion operatorPacket repairPlan to mirror top-level repairPlan, got operator=%#v top=%#v", operatorRepairPlan, repairPlan)
	}
	operatorFailedSummary := operatorPacket["failedEvidenceSummary"].(map[string]any)
	if int(operatorFailedSummary["total"].(float64)) != len(topFailedEvidence) ||
		int(operatorFailedSummary["repairQueueCount"].(float64)) != len(repairQueues) ||
		len(operatorFailedSummary["repairQueues"].([]any)) != len(repairQueues) ||
		!containsMapCount(operatorFailedSummary["issueCounts"].([]any), "runtimeEffectsMissingEvidence", 1) {
		t.Fatalf("expected completion operatorPacket to carry failed evidence summary, got %#v", operatorPacket)
	}
	if !containsRepairQueue(repairQueues, "setup-svc", "runtimeEffectChecks") ||
		!containsRepairQueue(repairQueues, "query-readback", "readbackExpectationChecks") {
		t.Fatalf("expected completion audit repair queues for failed evidence sections, got %#v", repairQueues)
	}
	if !containsRepairQueueSourceChecklist(repairQueues, "setup-svc", "runtimeEffectChecks") ||
		!containsRepairQueueSourceChecklist(repairQueues, "query-readback", "readbackExpectationChecks") {
		t.Fatalf("expected completion audit repair queues to include source-checklist commands, got %#v", repairQueues)
	}
	firstDomain := result["domains"].([]any)[0].(map[string]any)
	operatorFirstDomain := operatorPacket["domains"].([]any)[0].(map[string]any)
	if operatorFirstDomain["domain"] != firstDomain["domain"] ||
		operatorFirstDomain["completionStatus"] != firstDomain["completionStatus"] ||
		len(operatorFirstDomain["failedEvidence"].([]any)) != len(firstDomain["failedEvidence"].([]any)) ||
		int(operatorFirstDomain["repairPlan"].(map[string]any)["repairQueueCount"].(float64)) != int(firstDomain["repairPlan"].(map[string]any)["repairQueueCount"].(float64)) {
		t.Fatalf("expected completion operatorPacket domains to mirror top-level domain state, got operator=%#v top=%#v", operatorFirstDomain, firstDomain)
	}
	if firstDomain["completionStatus"] != "blocked_failed_evidence" {
		t.Fatalf("expected objects domain blocked by failed evidence, got %#v", firstDomain)
	}
	failedEvidence := firstDomain["failedEvidence"].([]any)
	if !containsStringFragment(failedEvidence, "objects/create: runtimeEffectsMissingEvidence=datatable-prefix-allocation") ||
		!containsStringFragment(failedEvidence, "objects/create: queryReadbackExpectationsMissingEvidence=object-identity-prefix-datatable-readback") {
		t.Fatalf("expected domain failedEvidence details, got %#v", failedEvidence)
	}
	domainRepairPlan := firstDomain["repairPlan"].(map[string]any)
	if int(domainRepairPlan["repairQueueCount"].(float64)) != 2 ||
		int(domainRepairPlan["totalSourceFiles"].(float64)) != len(failedEvidence) ||
		len(domainRepairPlan["nextRepairCommands"].([]any)) != 2 ||
		!containsStringFragment(domainRepairPlan["nextRepairCommands"].([]any), "setup-svc-live-replay-source-execution-packet") ||
		len(domainRepairPlan["postRepairCommands"].([]any)) != 6 ||
		!containsStringWithFragments(domainRepairPlan["postRepairCommands"].([]any), "setup-svc-live-replay-source-validate", "--domain objects", "--source-readiness complete") ||
		!containsStringWithFragments(domainRepairPlan["postRepairCommands"].([]any), "setup-svc-live-replay-worklist", "--domain objects", "--source-readiness complete", "--batch-index 0", " > ") ||
		!containsStringWithFragments(domainRepairPlan["postRepairCommands"].([]any), "setup-svc-live-replay-evidence-import @", "--dry-run") ||
		!containsStringWithFragments(domainRepairPlan["postRepairCommands"].([]any), "setup-svc-live-replay-manifest-sync", "--dry-run") ||
		!containsStringFragment(domainRepairPlan["postRepairCommands"].([]any), "setup-svc-live-replay-completion-audit") ||
		!strings.Contains(domainRepairPlan["nextRepairScript"].(string), "set -euo pipefail") ||
		!strings.Contains(domainRepairPlan["nextRepairScriptPath"].(string), "repair-plan-objects-next-repair-commands.sh") ||
		!strings.Contains(domainRepairPlan["saveNextRepairScriptCommand"].(string), "select(.domain==") ||
		!strings.Contains(domainRepairPlan["saveNextRepairScriptCommand"].(string), ".repairPlan.nextRepairScript") ||
		len(domainRepairPlan["domainOperations"].([]any)) != 1 {
		t.Fatalf("expected domain repairPlan to summarize only domain-local failed evidence, got %#v", domainRepairPlan)
	}
	domainObjectsCreateRepairPlan := repairPlanDomainOperation(domainRepairPlan["domainOperations"].([]any), "objects", "create")
	if domainObjectsCreateRepairPlan == nil ||
		int(domainObjectsCreateRepairPlan["failedEvidenceCount"].(float64)) != len(failedEvidence) ||
		!containsStringFragment(domainObjectsCreateRepairPlan["repairQueues"].([]any), "setup-svc/runtimeEffectChecks") ||
		!containsStringFragment(domainObjectsCreateRepairPlan["repairQueues"].([]any), "query-readback/readbackExpectationChecks") {
		t.Fatalf("expected domain repairPlan to map objects/create to its local repair queues, got %#v", domainRepairPlan)
	}
}

func TestSetupSvcLiveReplayCompletionFailedEvidenceSummaryDedupesRepairQueueArtifacts(t *testing.T) {
	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	failed := []string{
		"objects/create: evidenceFileStatusNotPassed:outputs/setup-svc-live-replay/objects/create/setup-svc.json",
		"objects/create: runtimeEffectsMissingEvidence=datatable-prefix-allocation:outputs/setup-svc-live-replay/objects/create/setup-svc.json",
		"objects/create: metadataServiceDatasourceMissingEvidence:outputs/setup-svc-live-replay/objects/create/metadata-service.json",
		"objects/create: metadataServiceDatasourceNotReady:outputs/setup-svc-live-replay/objects/create/metadata-service.json",
	}

	summary := buildSetupSvcLiveReplayCompletionFailedEvidenceSummary(tmp, manifestPath, failed)
	if summary.Total != 4 ||
		!containsRepairQueue(anySliceFromRepairQueues(summary.RepairQueues), "setup-svc", "runtimeEffectChecks") ||
		!containsRepairQueue(anySliceFromRepairQueues(summary.RepairQueues), "setup-svc", "tableSnapshots") ||
		!containsRepairQueue(anySliceFromRepairQueues(summary.RepairQueues), "metadata-service", "metadataServiceDatasource") {
		t.Fatalf("expected repair queues to count each artifact/section once, got %#v", summary)
	}
}

func TestSetupSvcLiveReplayCompletionAuditPassesAfterVerifiedMatrix(t *testing.T) {
	t.Setenv("MDS_RUNTIME_MODE", "self-hosted")
	t.Setenv("MDS_SERVER_PORT", "18087")
	t.Setenv("MDS_JDBC_URL", "jdbc:mysql://completion-db-host:3306/completion_metadata")
	t.Setenv("MDS_DB_USERNAME", "completion-user")
	t.Setenv("MDS_DB_PASSWORD", "completion-password")
	t.Setenv("MDS_DB_DRIVER", "com.mysql.cj.jdbc.Driver")
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	statuses := map[string]string{}
	for _, domain := range setupSvcLiveReplayDomains() {
		statuses[domain.Domain] = "verified"
	}
	writeSetupSvcLiveReplayParityMatrix(t, tmp, statuses)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeCurrentSetupSvcLiveReplayEvidenceBundle(t, tmp)

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-completion-audit"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "passed" {
		t.Fatalf("verified matrix and passed evidence should pass completion audit, got %#v", result)
	}
	datasource := result["metadataServiceDatasource"].(map[string]any)
	if datasource["status"] != "ready" || datasource["readyForRealDatasource"] != true || datasource["jdbcUrlSource"] != "env:MDS_JDBC_URL" {
		t.Fatalf("completion audit should expose ready datasource without values, got %#v", datasource)
	}
	if strings.Contains(stdout.String(), "completion-db-host") ||
		strings.Contains(stdout.String(), "completion-user") ||
		strings.Contains(stdout.String(), "completion-password") {
		t.Fatalf("completion audit leaked datasource secret values: %s", stdout.String())
	}
	totals := result["totals"].(map[string]any)
	if int(totals["completedDomains"].(float64)) != len(setupSvcLiveReplayDomains()) ||
		int(totals["matrixNonVerifiedDomains"].(float64)) != 0 ||
		int(totals["blockedDomains"].(float64)) != 0 {
		t.Fatalf("expected all domains complete, got %#v", totals)
	}
	gates := completionAuditGatesByName(result)
	for _, name := range []string{"matrix_contract", "test_evidence_contract", "live_replay_evidence", "promotion_audit", "evidence_bundle", "matrix_status"} {
		if gates[name]["status"] != "passed" || gates[name]["blocking"] == true {
			t.Fatalf("expected gate %s passed/non-blocking, got %#v", name, gates[name])
		}
	}
}

func TestSetupSvcLiveReplayCompletionAuditBlocksTestEvidenceSourceDrift(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:18087"}}}`)
	statuses := map[string]string{}
	for _, domain := range setupSvcLiveReplayDomains() {
		statuses[domain.Domain] = "verified"
	}
	writeSetupSvcLiveReplayParityMatrix(t, tmp, statuses)
	sourceRoot := filepath.Join(tmp, "cc-metadata-service/src/test/java/com/cloudcc/metadata/parity")
	writeTestFile(t, filepath.Join(sourceRoot, "GeneratedParityReplayTest.java"), `
package com.cloudcc.metadata.parity;

class GeneratedParityReplayTest {
    void unrelatedMethod() {
    }
}
`)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeCurrentSetupSvcLiveReplayEvidenceBundle(t, tmp)

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-completion-audit"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked_test_evidence_contract" {
		t.Fatalf("test evidence source drift must block completion audit, got %#v", result)
	}
	gates := completionAuditGatesByName(result)
	if gates["test_evidence_contract"]["status"] != "blocked" || gates["test_evidence_contract"]["blocking"] != true {
		t.Fatalf("expected blocking test evidence gate, got %#v", gates["test_evidence_contract"])
	}
	testEvidence := result["testEvidenceContract"].(map[string]any)
	if testEvidence["testSourceStatus"] != "blocked" || int(testEvidence["testSourceChecks"].(float64)) != setupSvcLiveReplayOperationCount() {
		t.Fatalf("expected blocked source checks for every operation, got %#v", testEvidence)
	}
	if !containsStringItem(result["blockingIssues"].([]any), "testEvidence: objects/create: missing replay test method GeneratedParityReplayTest.generatedParityReplayCoversMatrixOperation") {
		t.Fatalf("expected source method blocker, got %#v", result["blockingIssues"])
	}
}

func TestSetupSvcLiveReplayPreflightReadyBeforeApprovedReplay(t *testing.T) {
	t.Setenv("MDS_RUNTIME_MODE", "self-hosted")
	t.Setenv("MDS_SERVER_PORT", "18087")
	t.Setenv("MDS_JDBC_URL", "jdbc:mysql://preflight-db-host:3306/preflight_metadata")
	t.Setenv("MDS_DB_USERNAME", "preflight-user")
	t.Setenv("MDS_DB_PASSWORD", "preflight-password")
	t.Setenv("MDS_DB_DRIVER", "com.mysql.cj.jdbc.Driver")
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{
	  "use":"dev",
	  "dev":{
	    "metadataService":{"url":"http://127.0.0.1:18087"},
	    "setupSvc":"http://127.0.0.1:18080/setup",
	    "apiSvc":"http://127.0.0.1:18080/api",
	    "accessToken":"unit-token"
	  }
	}`)
	writeSetupSvcLiveReplayParityMatrix(t, tmp, nil)
	writeGeneratedSetupSvcLiveReplayTestSource(t, tmp)

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-preflight"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "ready_for_approved_live_replay" || result["readOnly"] != true {
		t.Fatalf("expected ready read-only preflight before approved replay, got %#v", result)
	}
	totals := result["totals"].(map[string]any)
	if int(totals["domains"].(float64)) != len(setupSvcLiveReplayDomains()) ||
		int(totals["operations"].(float64)) != setupSvcLiveReplayOperationCount() {
		t.Fatalf("expected full packet totals, got %#v", totals)
	}
	gates := completionAuditGatesByName(result)
	for _, name := range []string{"readiness", "metadata_service_datasource", "packet_dry_run", "coverage", "gaps", "evidence_bundle", "completion_audit"} {
		if gates[name] == nil {
			t.Fatalf("expected preflight gate %s, got %#v", name, gates)
		}
	}
	if gates["readiness"]["status"] != "ready_for_approved_live_replay" || gates["readiness"]["blocking"] == true {
		t.Fatalf("expected non-blocking readiness gate, got %#v", gates["readiness"])
	}
	if gates["metadata_service_datasource"]["status"] != "ready" || gates["metadata_service_datasource"]["blocking"] == true {
		t.Fatalf("expected non-blocking ready datasource gate, got %#v", gates["metadata_service_datasource"])
	}
	datasource := result["metadataServiceDatasource"].(map[string]any)
	if datasource["readyForRealDatasource"] != true || datasource["jdbcUrlSource"] != "env:MDS_JDBC_URL" {
		t.Fatalf("expected top-level datasource readiness, got %#v", datasource)
	}
	gateDatasource := gates["metadata_service_datasource"]["metadataServiceDatasource"].(map[string]any)
	if gateDatasource["readyForRealDatasource"] != true ||
		gateDatasource["jdbcUrlSource"] != "env:MDS_JDBC_URL" ||
		gateDatasource["jdbcUrlLooksDefaultH2"] != false {
		t.Fatalf("expected preflight datasource gate to carry redacted readiness, got %#v", gates["metadata_service_datasource"])
	}
	if strings.Contains(stdout.String(), "preflight-db-host") ||
		strings.Contains(stdout.String(), "preflight-user") ||
		strings.Contains(stdout.String(), "preflight-password") {
		t.Fatalf("preflight leaked datasource secret values: %s", stdout.String())
	}
	if gates["packet_dry_run"]["status"] != "dry_run_ready" || gates["packet_dry_run"]["blocking"] == true {
		t.Fatalf("expected non-blocking packet dry-run gate, got %#v", gates["packet_dry_run"])
	}
	if gates["coverage"]["status"] != "passed" || gates["coverage"]["blocking"] == true {
		t.Fatalf("expected non-blocking coverage gate, got %#v", gates["coverage"])
	}
	if gates["gaps"]["status"] != "missing_manifest" || gates["gaps"]["blocking"] == true {
		t.Fatalf("expected missing manifest to remain non-blocking before live replay, got %#v", gates["gaps"])
	}
	if gates["evidence_bundle"]["status"] != "missing" || gates["evidence_bundle"]["blocking"] == true {
		t.Fatalf("expected missing bundle to remain non-blocking before live replay, got %#v", gates["evidence_bundle"])
	}
	if gates["completion_audit"]["status"] != "blocked_missing_live_replay_evidence" || gates["completion_audit"]["blocking"] == true {
		t.Fatalf("expected completion audit live-evidence block to be informational in preflight, got %#v", gates["completion_audit"])
	}
	commands := result["nextCommands"].([]any)
	if !containsStringFragment(commands, "setup-svc-live-replay-packet") ||
		!containsStringFragment(commands, "setup-svc-live-replay-workspace") {
		t.Fatalf("expected packet and workspace next commands, got %#v", commands)
	}
}

func TestSetupSvcLiveReplayPreflightBlocksCoverageContractDrift(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{
	  "use":"dev",
	  "dev":{
	    "metadataService":{"url":"http://127.0.0.1:18087"},
	    "setupSvc":"http://127.0.0.1:18080/setup",
	    "apiSvc":"http://127.0.0.1:18080/api",
	    "accessToken":"unit-token"
	  }
	}`)
	writeSetupSvcLiveReplayParityMatrix(t, tmp, nil)
	matrixPath := filepath.Join(tmp, "cc-metadata-service/src/test/resources/parity/msapi-setup-svc-parity-matrix.json")
	rewriteSetupSvcLiveReplayParityMatrix(t, matrixPath, func(matrix map[string]any) {
		domains := matrix["domains"].([]any)
		first := domains[0].(map[string]any)
		first["runtimeEffects"] = []any{}
	})
	writeGeneratedSetupSvcLiveReplayTestSource(t, tmp)

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-preflight"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked_preflight" {
		t.Fatalf("matrix contract drift must block preflight, got %#v", result)
	}
	gates := completionAuditGatesByName(result)
	if gates["coverage"]["status"] != "blocked" || gates["coverage"]["blocking"] != true {
		t.Fatalf("expected blocking coverage gate, got %#v", gates["coverage"])
	}
	if !containsStringFragment(result["blockingIssues"].([]any), "coverage: blocked") {
		t.Fatalf("expected coverage blocking issue, got %#v", result["blockingIssues"])
	}
}

func TestSetupSvcLiveReplayPreflightKeepsSourceExecutionCommandsWhenBlocked(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{
	  "use":"dev",
	  "dev":{
	    "metadataService":{"url":"http://127.0.0.1:18087"},
	    "setupSvc":"http://127.0.0.1:18080/setup",
	    "apiSvc":"http://127.0.0.1:18080/api",
	    "accessToken":"unit-token"
	  }
	}`)
	writeSetupSvcLiveReplayParityMatrix(t, tmp, nil)
	writeGeneratedSetupSvcLiveReplayTestSource(t, tmp)
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-workspace", "--execute", "--approval", setupSvcParityEvidenceWorkspaceApproval}, &bytes.Buffer{}, tmp); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-capture-source-workspace", manifestPath, "--execute", "--approval", setupSvcParityCaptureSourceWorkspaceApproval}, &bytes.Buffer{}, tmp); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(tmp, "outputs/setup-svc-live-replay/evidence-bundle.json"), `{`)

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-preflight"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked_preflight" {
		t.Fatalf("invalid bundle should block preflight, got %#v", result)
	}
	if !containsStringFragment(result["blockingIssues"].([]any), "evidence_bundle: blocked") {
		t.Fatalf("expected evidence bundle blocking issue, got %#v", result["blockingIssues"])
	}
	captureSources := result["captureSources"].(map[string]any)
	if !preflightCaptureSourcesHasSourceExecutionCommands(captureSources) {
		t.Fatalf("expected blocked preflight captureSources to expose source execution packet/script save commands, got %#v", captureSources)
	}
	commands := result["nextCommands"].([]any)
	if !containsStringWithFragments(commands, "setup-svc-live-replay-source-execution-packet", "--source-readiness incomplete", "source-capture-execution-packet-readiness-incomplete.json") ||
		!containsStringWithFragments(commands, "jq -r '.batchSaveScript'", "source-capture-execution-packet-readiness-incomplete.sh") ||
		!containsStringWithFragments(commands, "jq -r '.importBatchSaveScript'", "source-capture-import-batches-readiness-complete.sh") {
		t.Fatalf("expected blocked preflight nextCommands to include source execution packet/script save commands, got %#v", commands)
	}
}

func TestSetupSvcLiveReplayPreflightAllowsPendingWorkspaceEvidence(t *testing.T) {
	t.Setenv("MDS_JDBC_URL", "")
	t.Setenv("MDS_DB_USERNAME", "")
	t.Setenv("MDS_DB_PASSWORD", "")
	t.Setenv("MDS_DB_DRIVER", "")
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{
	  "use":"dev",
	  "dev":{
	    "metadataService":{"url":"http://127.0.0.1:18087"},
	    "setupSvc":"http://127.0.0.1:18080/setup",
	    "apiSvc":"http://127.0.0.1:18080/api",
	    "accessToken":"unit-token"
	  }
	}`)
	writeSetupSvcLiveReplayParityMatrix(t, tmp, nil)
	writeGeneratedSetupSvcLiveReplayTestSource(t, tmp)
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-workspace", "--execute", "--approval", setupSvcParityEvidenceWorkspaceApproval}, &bytes.Buffer{}, tmp); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-preflight"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "evidence_collection_in_progress" {
		t.Fatalf("pending evidence workspace should be in progress, not blocked, got %#v", result)
	}
	gates := completionAuditGatesByName(result)
	if gates["gaps"]["status"] != "pending_evidence" || gates["gaps"]["blocking"] == true {
		t.Fatalf("expected pending gaps to be non-blocking, got %#v", gates["gaps"])
	}
	if gates["completion_audit"]["status"] != "blocked_live_replay_evidence" || gates["completion_audit"]["blocking"] == true {
		t.Fatalf("expected pending live evidence completion status to be non-blocking in preflight, got %#v", gates["completion_audit"])
	}
	if gates["metadata_service_datasource"]["status"] != "missing_real_datasource" || gates["metadata_service_datasource"]["blocking"] == true {
		t.Fatalf("expected missing datasource to be visible but non-blocking during evidence collection, got %#v", gates["metadata_service_datasource"])
	}
	datasource := result["metadataServiceDatasource"].(map[string]any)
	if datasource["readyForRealDatasource"] != false {
		t.Fatalf("expected preflight to expose missing real datasource, got %#v", datasource)
	}
	gateDatasource := gates["metadata_service_datasource"]["metadataServiceDatasource"].(map[string]any)
	if gateDatasource["status"] != "missing_real_datasource" ||
		gateDatasource["readyForRealDatasource"] != false ||
		!containsAnyString(gateDatasource["missing"].([]any), "MDS_JDBC_URL") {
		t.Fatalf("expected preflight datasource gate to carry missing readiness, got %#v", gates["metadata_service_datasource"])
	}
	missing := datasource["missing"].([]any)
	if !containsAnyString(missing, "MDS_JDBC_URL") ||
		!containsAnyString(missing, "MDS_DB_USERNAME") ||
		!containsAnyString(missing, "MDS_DB_PASSWORD") ||
		!containsAnyString(missing, "MDS_DB_DRIVER") {
		t.Fatalf("expected missing datasource variables, got %#v", missing)
	}
	captureSources := result["captureSources"].(map[string]any)
	if captureSources["status"] != "missing" ||
		int(captureSources["artifactFiles"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount() ||
		int(captureSources["sourceFiles"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount() ||
		int(captureSources["sourceFilesPresent"].(float64)) != 0 ||
		int(captureSources["sourceFilesMissing"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount() ||
		int(captureSources["sourceFilesComplete"].(float64)) != 0 ||
		int(captureSources["sourceFilesIncomplete"].(float64)) != 0 ||
		int(captureSources["sourceTemplatesMissingGuideFields"].(float64)) != 0 ||
		!strings.Contains(captureSources["missingWorklistCommand"].(string), "--source-status missing") ||
		!strings.Contains(captureSources["missingSourceChecklistCommand"].(string), "setup-svc-live-replay-source-checklist") ||
		!strings.Contains(captureSources["missingSourceChecklistCommand"].(string), "--source-status missing") ||
		!strings.Contains(captureSources["captureSourceWorkspaceDryRunCommand"].(string), "setup-svc-live-replay-capture-source-workspace") ||
		!strings.Contains(captureSources["captureSourceWorkspaceDryRunCommand"].(string), "--dry-run") ||
		!strings.Contains(captureSources["captureSourceWorkspaceExecuteCommand"].(string), setupSvcParityCaptureSourceWorkspaceApproval) ||
		!strings.Contains(captureSources["captureSourceWorkspaceExecuteCommand"].(string), "--source-status missing") {
		t.Fatalf("expected missing capture source summary with initializer commands, got %#v", captureSources)
	}
	if _, ok := captureSources["completeWorklistCommand"]; ok {
		t.Fatalf("missing sources should not advertise complete import worklist yet, got %#v", captureSources)
	}
	if _, ok := captureSources["saveSourceExecutionPacketCommand"]; ok {
		t.Fatalf("missing sources should not advertise source execution packet commands yet, got %#v", captureSources)
	}
	if gates["capture_sources"]["status"] != "missing" || gates["capture_sources"]["blocking"] == true {
		t.Fatalf("expected non-blocking capture source gate, got %#v", gates["capture_sources"])
	}
	commands := result["nextCommands"].([]any)
	if !containsStringFragment(commands, "setup-svc-live-replay-capture-source-workspace") ||
		!containsStringFragment(commands, setupSvcParityCaptureSourceWorkspaceApproval) ||
		!containsStringFragment(commands, "setup-svc-live-replay-worklist") ||
		!containsStringFragment(commands, "setup-svc-live-replay-source-checklist") ||
		!containsStringFragment(commands, "--source-status missing") ||
		!containsStringFragment(commands, "--evidence-section tableSnapshots") ||
		!containsStringFragment(commands, "setup-svc-live-replay-manifest-sync") ||
		!containsStringFragment(commands, "setup-svc-live-replay-evidence-bundle") ||
		containsStringFragment(commands, "setup-svc-live-replay-workspace") {
		t.Fatalf("expected in-progress preflight to expose evidence collection runbook commands, got %#v", commands)
	}

	firstCapture := setupSvcLiveReplayWorklistSuggestedSourcePath(setupSvcLiveReplayEvidenceFiles("objects", "create", true)[0])
	writeTestFile(t, filepath.Join(tmp, firstCapture), `{"status":"passed"}`)
	stdout.Reset()
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-preflight"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	captureSources = result["captureSources"].(map[string]any)
	if captureSources["status"] != "partial" ||
		int(captureSources["sourceFiles"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount() ||
		int(captureSources["sourceFilesPresent"].(float64)) != 1 ||
		int(captureSources["sourceFilesMissing"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount()-1 ||
		int(captureSources["sourceFilesComplete"].(float64)) != 0 ||
		int(captureSources["sourceFilesIncomplete"].(float64)) != 1 ||
		int(captureSources["sourceTemplatesMissingGuideFields"].(float64)) != 0 {
		t.Fatalf("expected partial capture source summary after one mirrored capture, got %#v", captureSources)
	}
	if !strings.Contains(captureSources["incompleteWorklistCommand"].(string), "--source-readiness incomplete") ||
		!strings.Contains(captureSources["incompleteSourceChecklistCommand"].(string), "--source-readiness incomplete") ||
		!strings.Contains(captureSources["presentWorklistCommand"].(string), "--source-status present") ||
		!strings.Contains(captureSources["presentSourceChecklistCommand"].(string), "--source-status present") {
		t.Fatalf("expected partial capture sources to expose incomplete and present review worklists, got %#v", captureSources)
	}

	stdout.Reset()
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-capture-source-workspace", manifestPath, "--execute", "--approval", setupSvcParityCaptureSourceWorkspaceApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-preflight"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	captureSources = result["captureSources"].(map[string]any)
	if captureSources["status"] != "partial" ||
		int(captureSources["sourceFiles"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount() ||
		int(captureSources["sourceFilesPresent"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount() ||
		int(captureSources["sourceFilesMissing"].(float64)) != 0 ||
		int(captureSources["sourceFilesComplete"].(float64)) != 0 ||
		int(captureSources["sourceFilesIncomplete"].(float64)) != setupSvcLiveReplayExpectedArtifactFileCount() ||
		int(captureSources["sourceTemplatesMissingGuideFields"].(float64)) != 0 {
		t.Fatalf("expected all source templates present but incomplete after workspace initialization, got %#v", captureSources)
	}
	if _, ok := captureSources["captureSourceWorkspaceDryRunCommand"]; ok {
		t.Fatalf("no missing source templates should suppress capture-source initializer commands, got %#v", captureSources)
	}
	captureSourceMissingSectionCounts := captureSources["missingEvidenceSectionCounts"].([]any)
	if !sourceChecklistHasMissingSectionCount(captureSourceMissingSectionCounts, "tableSnapshots", 2*setupSvcLiveReplayOperationCount(), "metadata-service", "setup-svc") ||
		!sourceChecklistHasMissingSectionCount(captureSourceMissingSectionCounts, "runtimeEffectChecks", 2*setupSvcLiveReplayOperationCount(), "metadata-service", "setup-svc") ||
		!sourceChecklistHasMissingSectionCount(captureSourceMissingSectionCounts, "readbackTables", setupSvcLiveReplayOperationCount(), "query-readback") ||
		!sourceChecklistHasMissingSectionCount(captureSourceMissingSectionCounts, "diffCounters", setupSvcLiveReplayOperationCount(), "normalized-diff") ||
		!sourceChecklistHasMissingSectionCount(captureSourceMissingSectionCounts, "residualCounters", setupSvcLiveReplayWriteOperationCount(), "cleanup") {
		t.Fatalf("expected preflight captureSources to expose source-level missing section counts, got %#v", captureSourceMissingSectionCounts)
	}
	captureSourceNextQueueCommands := captureSources["nextSourceQueueCommands"].([]any)
	if !sourceChecklistHasNextQueueCommand(captureSourceNextQueueCommands, "", "tableSnapshots", 2*setupSvcLiveReplayOperationCount(), "") ||
		!sourceChecklistHasNextQueueCommand(captureSourceNextQueueCommands, "query-readback", "readbackTables", setupSvcLiveReplayOperationCount(), "") ||
		!sourceChecklistHasNextQueueCommand(captureSourceNextQueueCommands, "cleanup", "residualCounters", setupSvcLiveReplayWriteOperationCount(), "") {
		t.Fatalf("expected preflight captureSources to expose actionable source queue commands, got %#v", captureSourceNextQueueCommands)
	}
	if !sourceChecklistHasNextPageCommands(captureSourceNextQueueCommands, "", "tableSnapshots", 25) ||
		!sourceChecklistHasNextPageCommands(captureSourceNextQueueCommands, "cleanup", "residualCounters", 25) {
		t.Fatalf("expected preflight captureSources to expose source queue next-page commands, got %#v", captureSourceNextQueueCommands)
	}
	if !sourceChecklistHasAllPageSaveCommands(captureSourceNextQueueCommands, "", "tableSnapshots", 8, 0, 175) ||
		!sourceChecklistHasAllPageSaveCommands(captureSourceNextQueueCommands, "cleanup", "residualCounters", 3, 0, 50) {
		t.Fatalf("expected preflight captureSources to expose source queue all-page save commands, got %#v", captureSourceNextQueueCommands)
	}
	if !sourceChecklistHasPageCommandSummary(captureSources, 8, 0, 175) {
		t.Fatalf("expected preflight captureSources to expose flattened all-page save commands, got %#v", captureSources)
	}
	if !sourceChecklistHasPageSaveScript(captureSources, 0, 175) {
		t.Fatalf("expected preflight captureSources to expose page save script, got %#v", captureSources)
	}
	if !sourceChecklistHasSavePageScriptCommand(captureSources, ".captureSources.pageSaveScript", "setup-svc-live-replay-preflight") {
		t.Fatalf("expected preflight captureSources to expose saveable page script command, got %#v", captureSources)
	}
	if !preflightCaptureSourcesHasSourceExecutionCommands(captureSources) {
		t.Fatalf("expected preflight captureSources to expose source execution packet/script save commands, got %#v", captureSources)
	}
	commands = result["nextCommands"].([]any)
	if containsStringFragment(commands, "setup-svc-live-replay-capture-source-workspace") ||
		containsStringFragment(commands, "--source-status missing") ||
		containsStringFragment(commands, "setup-svc-live-replay-evidence-import") ||
		!containsStringFragment(commands, "--source-readiness incomplete") ||
		!containsStringFragment(commands, "--source-status present") ||
		!containsStringWithFragments(commands, "setup-svc-live-replay-source-checklist", "--source-readiness incomplete", " > ") ||
		!containsStringWithFragments(commands, "setup-svc-live-replay-source-checklist", "--source-status present", " > ") {
		t.Fatalf("expected complete source-template coverage to route to incomplete review, got %#v", commands)
	}
	if !containsStringWithFragments(commands, "setup-svc-live-replay-worklist", "--source-readiness incomplete", "--batch-index 0", " > ") ||
		containsStringWithFragments(commands, "setup-svc-live-replay-gaps", "--source-readiness incomplete") {
		t.Fatalf("expected incomplete source review to expose saveable worklist batch commands, got %#v", commands)
	}
	if !containsStringWithFragments(commands, "setup-svc-live-replay-source-execution-packet", "--source-readiness incomplete", "source-capture-execution-packet-readiness-incomplete.json") ||
		!containsStringWithFragments(commands, "jq -r '.batchSaveScript'", "source-capture-execution-packet-readiness-incomplete.sh") ||
		!containsStringWithFragments(commands, "jq -r '.importBatchSaveScript'", "source-capture-import-batches-readiness-complete.sh") {
		t.Fatalf("expected incomplete source review to expose source execution packet/script save commands, got %#v", commands)
	}
	if containsSourceExecutionCommandWithPagination(commands) {
		t.Fatalf("source execution packet save commands must cover all incomplete sources without pagination, got %#v", commands)
	}

	stdout.Reset()
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-worklist", manifestPath, "--source-readiness", "incomplete", "--offset", "0", "--limit", "3"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var incompleteWorklist map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &incompleteWorklist); err != nil {
		t.Fatal(err)
	}
	incompleteWorklistTotals := incompleteWorklist["totals"].(map[string]any)
	incompleteWorklistPacket := incompleteWorklist["operatorPacket"].(map[string]any)
	incompleteSourceSections := incompleteWorklist["sourceEvidenceSections"].([]any)
	incompletePacketSections := incompleteWorklistPacket["sourceEvidenceSections"].([]any)
	if int(incompleteWorklistTotals["sourceFilesPresent"].(float64)) <= 0 ||
		int(incompleteWorklistTotals["uniqueArtifactFiles"].(float64)) != int(incompleteWorklistTotals["sourceFilesPresent"].(float64)) ||
		int(incompleteWorklistTotals["duplicateArtifactRecords"].(float64)) != int(incompleteWorklistTotals["artifacts"].(float64))-int(incompleteWorklistTotals["uniqueArtifactFiles"].(float64)) ||
		int(incompleteWorklistTotals["sourceFilesMissing"].(float64)) != 0 ||
		int(incompleteWorklistTotals["sourceFilesComplete"].(float64)) != 0 ||
		int(incompleteWorklistTotals["sourceFilesIncomplete"].(float64)) != int(incompleteWorklistTotals["sourceFilesPresent"].(float64)) ||
		int(incompleteWorklistPacket["uniqueArtifactFiles"].(float64)) != int(incompleteWorklistTotals["uniqueArtifactFiles"].(float64)) ||
		int(incompleteWorklistPacket["duplicateArtifactRecords"].(float64)) != int(incompleteWorklistTotals["duplicateArtifactRecords"].(float64)) ||
		int(incompleteWorklistPacket["sourceFilesPresent"].(float64)) != int(incompleteWorklistTotals["sourceFilesPresent"].(float64)) ||
		int(incompleteWorklistPacket["sourceFilesIncomplete"].(float64)) != int(incompleteWorklistTotals["sourceFilesIncomplete"].(float64)) {
		t.Fatalf("expected worklist source counters to count unique mirrored source files, got totals=%#v packet=%#v", incompleteWorklistTotals, incompleteWorklistPacket)
	}
	incompleteTableSummary := evidenceSectionSummary(incompleteSourceSections, "setup-svc", "tableSnapshots")
	if len(incompleteSourceSections) == 0 ||
		len(incompletePacketSections) != len(incompleteSourceSections) ||
		incompleteTableSummary == nil ||
		int(incompleteTableSummary["missing"].(float64)) == 0 ||
		!strings.Contains(incompleteTableSummary["queueCommand"].(string), "setup-svc-live-replay-worklist") ||
		strings.Contains(incompleteTableSummary["queueCommand"].(string), "setup-svc-live-replay-gaps") ||
		!strings.Contains(incompleteTableSummary["queueCommand"].(string), "--source-readiness incomplete") {
		t.Fatalf("expected worklist to aggregate missing source evidence sections, got top=%#v packet=%#v", incompleteSourceSections, incompletePacketSections)
	}
	incompleteQueues := incompleteWorklist["queues"].([]any)
	if len(incompleteQueues) == 0 {
		t.Fatalf("expected incomplete worklist queues")
	}
	incompleteQueue := incompleteQueues[0].(map[string]any)
	incompleteBatches := incompleteQueue["batches"].([]any)
	if !strings.Contains(incompleteQueue["queueCommand"].(string), "setup-svc-live-replay-worklist") ||
		strings.Contains(incompleteQueue["queueCommand"].(string), "setup-svc-live-replay-gaps") ||
		!strings.Contains(incompleteQueue["queueCommand"].(string), "--source-readiness incomplete") ||
		len(incompleteBatches) == 0 ||
		!strings.Contains(incompleteBatches[0].(map[string]any)["command"].(string), "setup-svc-live-replay-worklist") ||
		strings.Contains(incompleteBatches[0].(map[string]any)["command"].(string), "setup-svc-live-replay-gaps") ||
		!strings.Contains(incompleteBatches[0].(map[string]any)["command"].(string), "--source-readiness incomplete") ||
		!strings.Contains(incompleteBatches[0].(map[string]any)["command"].(string), "--batch-index") ||
		!strings.Contains(incompleteBatches[0].(map[string]any)["saveWorklistCommand"].(string), " > ") ||
		!strings.Contains(incompleteBatches[0].(map[string]any)["dryRunImportCommand"].(string), "setup-svc-live-replay-evidence-import") {
		t.Fatalf("expected incomplete worklist queue and batch commands to preserve source filters as executable worklist commands, got queue=%#v", incompleteQueue)
	}
	if int(incompleteWorklistTotals["artifacts"].(float64)) <= int(incompleteWorklistTotals["sourceFilesPresent"].(float64)) {
		t.Fatalf("test must cover expanded records greater than unique source files, got %#v", incompleteWorklistTotals)
	}

	completeSetupSvcSource := setupSvcLiveReplayWorklistSuggestedSourcePath(setupSvcLiveReplayEvidenceFiles("objects", "create", true)[0])
	writeSetupSvcLiveReplayArtifact(t, tmp, "objects", "create", completeSetupSvcSource, map[string]any{"status": "passed"})
	stdout.Reset()
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-worklist", manifestPath, "--artifact-type", "setup-svc", "--source-readiness", "complete", "--offset", "0", "--limit", "5"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var completeSourceWorklist map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &completeSourceWorklist); err != nil {
		t.Fatal(err)
	}
	completeTotals := completeSourceWorklist["totals"].(map[string]any)
	completeSections := completeSourceWorklist["sourceEvidenceSections"].([]any)
	completeTableSnapshots := evidenceSectionSummary(completeSections, "setup-svc", "tableSnapshots")
	completeRuntimeChecks := evidenceSectionSummary(completeSections, "setup-svc", "runtimeEffectChecks")
	if int(completeTotals["sourceFilesComplete"].(float64)) != 1 ||
		int(completeTotals["sourceFilesIncomplete"].(float64)) != 0 ||
		completeTableSnapshots == nil ||
		int(completeTableSnapshots["present"].(float64)) != 1 ||
		int(completeTableSnapshots["missing"].(float64)) != 0 ||
		completeRuntimeChecks == nil ||
		int(completeRuntimeChecks["present"].(float64)) != 1 ||
		int(completeRuntimeChecks["missing"].(float64)) != 0 {
		t.Fatalf("expected source-complete worklist section summaries to come from mirrored capture source, got totals=%#v sections=%#v", completeTotals, completeSections)
	}
	completeQueues := completeSourceWorklist["queues"].([]any)
	if len(completeQueues) == 0 {
		t.Fatalf("expected source-complete worklist to keep target pending queue records")
	}
	completeBatch := completeQueues[0].(map[string]any)["batches"].([]any)[0].(map[string]any)
	completeRecord := completeBatch["operatorBatch"].(map[string]any)["artifactReplacementRecords"].([]any)[0].(map[string]any)
	completeMissingSections, _ := completeRecord["missingEvidenceSections"].([]any)
	if completeRecord["sourceReadiness"] != "complete" ||
		!evidenceSectionHasStatus(completeRecord["sourceEvidenceSections"].([]any), "tableSnapshots", "present") ||
		!evidenceSectionHasStatus(completeRecord["sourceEvidenceSections"].([]any), "runtimeEffectChecks", "present") ||
		containsStringItem(completeMissingSections, "tableSnapshots") ||
		containsStringItem(completeMissingSections, "runtimeEffectChecks") {
		t.Fatalf("expected replacement record missing sections to be based on mirrored source rather than pending target artifact, got %#v", completeRecord)
	}

	legacyTemplatePath := filepath.Join(tmp, setupSvcLiveReplayWorklistSuggestedSourcePath(setupSvcLiveReplayEvidenceFiles("fields", "create", true)[0]))
	legacyBody, err := os.ReadFile(legacyTemplatePath)
	if err != nil {
		t.Fatal(err)
	}
	var legacyTemplate map[string]any
	if err := json.Unmarshal(legacyBody, &legacyTemplate); err != nil {
		t.Fatal(err)
	}
	delete(legacyTemplate, "requiredShapeKey")
	delete(legacyTemplate, "manifestStatusField")
	delete(legacyTemplate, "requiredEvidenceSections")
	legacyBytes, err := json.MarshalIndent(legacyTemplate, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, legacyTemplatePath, string(legacyBytes))

	stdout.Reset()
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-preflight"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	captureSources = result["captureSources"].(map[string]any)
	if int(captureSources["sourceTemplatesMissingGuideFields"].(float64)) != 1 ||
		!strings.Contains(captureSources["captureSourceWorkspaceRefreshCommand"].(string), "--source-status present") ||
		!strings.Contains(captureSources["captureSourceWorkspaceRefreshCommand"].(string), setupSvcParityCaptureSourceWorkspaceApproval) {
		t.Fatalf("expected legacy pending source template to expose a safe refresh command, got %#v", captureSources)
	}
	commands = result["nextCommands"].([]any)
	if !containsStringFragment(commands, "setup-svc-live-replay-capture-source-workspace") ||
		!containsStringFragment(commands, "--source-status present") ||
		!containsStringFragment(commands, setupSvcParityCaptureSourceWorkspaceApproval) {
		t.Fatalf("expected preflight nextCommands to include capture-source refresh command, got %#v", commands)
	}

	stdout.Reset()
	if err := Handle("apply", "msapi", []string{tmp, "setup-svc-live-replay-capture-source-workspace", manifestPath, "--source-status", "present", "--execute", "--approval", setupSvcParityCaptureSourceWorkspaceApproval}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-preflight"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	captureSources = result["captureSources"].(map[string]any)
	if int(captureSources["sourceTemplatesMissingGuideFields"].(float64)) != 0 {
		t.Fatalf("expected source template guide refresh to clear missing guide fields, got %#v", captureSources)
	}
}

func TestSetupSvcLiveReplayPreflightForwardsBundleAndMatrixCompletionCommands(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{
	  "use":"dev",
	  "dev":{
	    "metadataService":{"url":"http://127.0.0.1:18087"},
	    "setupSvc":"http://127.0.0.1:18080/setup",
	    "apiSvc":"http://127.0.0.1:18080/api",
	    "accessToken":"unit-token"
	  }
	}`)
	writeSetupSvcLiveReplayParityMatrix(t, tmp, nil)
	writeGeneratedSetupSvcLiveReplayTestSource(t, tmp)
	manifestPath := filepath.Join(tmp, "outputs/setup-svc-live-replay/manifest.json")
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-preflight"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "blocked_evidence_bundle" {
		t.Fatalf("passed evidence without bundle should route preflight to bundle stage, got %#v", result)
	}
	gates := completionAuditGatesByName(result)
	if gates["completion_audit"]["status"] != "blocked_evidence_bundle" || gates["completion_audit"]["blocking"] == true {
		t.Fatalf("expected bundle completion gate to remain actionable, got %#v", gates["completion_audit"])
	}
	commands := result["nextCommands"].([]any)
	if !containsStringFragment(commands, "setup-svc-live-replay-evidence-bundle") ||
		!containsStringFragment(commands, setupSvcParityEvidenceBundleApproval) ||
		containsStringFragment(commands, "setup-svc-live-replay-workspace") {
		t.Fatalf("expected preflight to forward bundle nextCommands, got %#v", commands)
	}

	writeCurrentSetupSvcLiveReplayEvidenceBundle(t, tmp)
	stdout.Reset()
	if err := Handle("scan", "msapi", []string{tmp, "setup-svc-live-replay-preflight"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "ready_for_matrix_status_update" {
		t.Fatalf("passed evidence and bundle should route preflight to matrix promotion, got %#v", result)
	}
	gates = completionAuditGatesByName(result)
	if gates["completion_audit"]["status"] != "ready_for_matrix_status_update" || gates["completion_audit"]["blocking"] == true {
		t.Fatalf("expected matrix update completion gate to remain actionable, got %#v", gates["completion_audit"])
	}
	commands = result["nextCommands"].([]any)
	if !containsStringFragment(commands, "setup-svc-live-replay-promotion") ||
		!containsStringFragment(commands, setupSvcParityMatrixPromotionApproval) ||
		containsStringFragment(commands, setupSvcParityEvidenceBundleApproval) ||
		containsStringFragment(commands, "setup-svc-live-replay-workspace") {
		t.Fatalf("expected preflight to forward matrix promotion nextCommands, got %#v", commands)
	}
}

func completionAuditGatesByName(result map[string]any) map[string]map[string]any {
	gates := map[string]map[string]any{}
	for _, raw := range result["gates"].([]any) {
		item := raw.(map[string]any)
		gates[item["name"].(string)] = item
	}
	return gates
}

func setupSvcLiveReplayOperationCount() int {
	total := 0
	for _, domain := range setupSvcLiveReplayDomains() {
		total += len(domain.Operations)
	}
	return total
}

func TestHighCodeScanReportsLocalAssetsAndLegacyScripts(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"metadataService":{"url":"http://127.0.0.1:1"}}}`)
	writeTestFile(t, tmp+"/backend/classes/PriceService/PriceService.java", `public class PriceService {}`)
	writeTestFile(t, tmp+"/backend/classes/PriceService/config.json", `{"name":"PriceService"}`)
	writeTestFile(t, tmp+"/backend/triggers/account/AccountBefore/AccountBefore.java", `public class AccountBefore {}`)
	writeTestFile(t, tmp+"/backend/schedule/DailyJob/DailyJob.java", `public class DailyJob {}`)
	writeTestFile(t, tmp+"/backend/schedule/DailyJob/config.json", `{"name":"DailyJob"}`)
	writeTestFile(t, tmp+"/script/account/ClientScript/ClientScript.js", `function main($CCDK, obj) {}`)
	writeTestFile(t, tmp+"/script/account/ClientScript/config.json", `{"name":"ClientScript"}`)
	writeTestFile(t, tmp+"/html/Workbench/index.html", `<!doctype html>`)
	writeTestFile(t, tmp+"/html/Workbench/config.json", `{"apiName":"Workbench"}`)
	writeTestFile(t, tmp+"/frontend/pagecomponents/cc-demo/cc-demo.vue", `<template><div /></template>`)
	writeTestFile(t, tmp+"/frontend/pagecomponents/cc-demo/config.json", `{"component":"component-cc-demo","compName":"cc-demo"}`)
	writeTestFile(t, tmp+"/frontend/build/component-cc-demo.umd.min.js", `window.customElements.define("component-cc-demo", class extends HTMLElement {});`)
	writeTestFile(t, tmp+"/staticResource/logo.png", `png`)
	writeTestFile(t, tmp+"/sidecar/main.go", `package main`)
	writeTestFile(t, tmp+"/scripts/apply-cloudcc-metadata.js", `console.log("legacy")`)

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "highcode"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("scan output should be JSON: %v\n%s", err, stdout.String())
	}
	if result["mode"] != "go-local-highcode-scan" {
		t.Fatalf("unexpected mode %#v", result["mode"])
	}
	totals := result["totals"].(map[string]any)
	if got := int(totals["legacyNodeScripts"].(float64)); got != 1 {
		t.Fatalf("expected one legacy Node script, got %d", got)
	}
	if got := int(totals["publishable"].(float64)); got < 6 {
		t.Fatalf("expected publishable local assets, got %d", got)
	}
	if !strings.Contains(stdout.String(), "not_a_go_skill_validation_path") {
		t.Fatalf("expected legacy script warning, got %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "config_missing") {
		t.Fatalf("expected trigger config issue, got %s", stdout.String())
	}
}

func TestHighCodeScanWorksForEmptyProject(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{}}`)
	for _, dir := range []string{
		"backend/classes",
		"backend/triggers",
		"backend/schedule",
		"frontend/pagecomponents",
		"sidecar",
	} {
		if err := os.MkdirAll(tmp+"/"+dir, 0755); err != nil {
			t.Fatal(err)
		}
		writeTestFile(t, tmp+"/"+dir+"/.gitkeep", "")
	}
	writeTestFile(t, tmp+"/scripts/apply-cloudcc-navigation.js", `console.log("legacy")`)

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "local"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	totals := result["totals"].(map[string]any)
	if got := int(totals["assets"].(float64)); got != 0 {
		t.Fatalf("expected no high-code assets, got %d", got)
	}
	if got := int(totals["legacyNodeScripts"].(float64)); got != 1 {
		t.Fatalf("expected legacy script to be reported, got %d", got)
	}
	if !strings.Contains(stdout.String(), `"empty"`) {
		t.Fatalf("expected empty domain evidence, got %s", stdout.String())
	}
}

func TestOnlineHighCodeScanCallsReadOnlyListEndpoints(t *testing.T) {
	seen := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method %s", r.Method)
		}
		seen[r.URL.Path]++
		switch r.URL.Path {
		case "/setup/api/ccfag/list":
			_, _ = w.Write([]byte(`{"returnCode":"1","result":true,"data":{"list":[{"id":"cls1","name":"PriceService"}]}}`))
		case "/setup/api/triggerSetup/getTriggerByCondition":
			_, _ = w.Write([]byte(`{"returnCode":"1","result":true,"data":{"list":[{"id":"trg1","name":"AccountBefore"}]}}`))
		case "/setup/api/ccPeak/list":
			_, _ = w.Write([]byte(`{"returnCode":"1","result":true,"data":{"list":[{"id":"tmr1","name":"DailyJob"}]}}`))
		case "/setup/api/staticResource/list", "/setup/api/staticresource/list":
			http.Error(w, "NoResourceFound", http.StatusNotFound)
		case "/setup/api/staticResources/queryList":
			_, _ = w.Write([]byte(`{"returnCode":"1","result":true,"data":{"list":[{"id":"res1","name":"logo"}]}}`))
		case "/devconsole/script/pageClientScript":
			_, _ = w.Write([]byte(`{"data":{"list":[{"id":"scr1","scriptName":"ClientScript"}]}}`))
		case "/devconsole/custom/pc/1.0/post/pageCustomComp":
			_, _ = w.Write([]byte(`{"returnCode":"200","data":{"list":[{"id":"pc1","compUniName":"component-demo"}]}}`))
		case "/devconsole/htmlComponent/pageHtmlComponent":
			_, _ = w.Write([]byte(`{"returnCode":"20-000-OK","data":{"list":[{"id":"html1","htmlLabel":"Workbench","apiName":"workbench"}]}}`))
		case "/devconsole/custom/pc/1.0/post/pageCustomPage":
			_, _ = w.Write([]byte(`{"returnCode":"200","data":{"list":[{"id":"page1","pageLabel":"Workbench","pageApi":"workbench_page"}]}}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	tmp := t.TempDir()
	writeTestFile(t, tmp+"/cloudcc-cli.config.json", `{"use":"dev","dev":{"baseUrl":"`+server.URL+`","setupSvc":"`+server.URL+`/setup","apiSvc":"`+server.URL+`/api","accessToken":"token","pluginToken":"plugin","version":"private"}}`)

	var stdout bytes.Buffer
	if err := Handle("scan", "msapi", []string{tmp, "online-highcode"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		"/setup/api/ccfag/list",
		"/setup/api/triggerSetup/getTriggerByCondition",
		"/setup/api/ccPeak/list",
		"/setup/api/staticResource/list",
		"/setup/api/staticresource/list",
		"/setup/api/staticResources/queryList",
		"/devconsole/script/pageClientScript",
		"/devconsole/custom/pc/1.0/post/pageCustomComp",
		"/devconsole/htmlComponent/pageHtmlComponent",
		"/devconsole/custom/pc/1.0/post/pageCustomPage",
	} {
		if seen[path] != 1 {
			t.Fatalf("expected %s once, saw %d", path, seen[path])
		}
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["mode"] != "go-online-highcode-scan" {
		t.Fatalf("unexpected mode %#v", result["mode"])
	}
	totals := result["totals"].(map[string]any)
	if got := int(totals["onlineItems"].(float64)); got != 8 {
		t.Fatalf("expected eight online items, got %d: %s", got, stdout.String())
	}
	if got := int(totals["unsupported"].(float64)); got != 0 {
		t.Fatalf("expected no unsupported CloudCC online endpoints, got %d: %s", got, stdout.String())
	}
	if got := int(totals["outOfScope"].(float64)); got != 1 {
		t.Fatalf("expected only sidecar out of scope, got %d: %s", got, stdout.String())
	}
	if !strings.Contains(stdout.String(), `"/api/triggerSetup/getTriggerByCondition"`) {
		t.Fatalf("expected canonical trigger metadata endpoint evidence, got %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"/api/staticResources/queryList"`) {
		t.Fatalf("expected staticResource fallback endpoint evidence, got %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"domain": "html"`) || !strings.Contains(stdout.String(), `"domain": "customPage"`) {
		t.Fatalf("expected html and customPage online scanner coverage, got %s", stdout.String())
	}
}

func writeSetupSvcLiveReplayManifest(t *testing.T, path string, complete bool) {
	t.Helper()
	projectPath := setupSvcLiveReplayProjectPathFromManifest(path)
	domains := []map[string]any{}
	for _, expected := range setupSvcLiveReplayDomains() {
		if !complete && expected.Domain != "objects" {
			continue
		}
		operations := []map[string]any{}
		for _, operation := range expected.Operations {
			item := map[string]any{
				"operation":                     operation,
				"setupSvcEvidenceStatus":        "passed",
				"metadataServiceEvidenceStatus": "passed",
				"queryEvidenceStatus":           "passed",
				"normalizedDiffStatus":          "passed",
			}
			if operation != "query" {
				item["cleanupStatus"] = "passed"
			}
			files := setupSvcLiveReplayEvidenceFiles(expected.Domain, operation, operation != "query")
			item["evidenceFiles"] = files
			for _, file := range files {
				writeSetupSvcLiveReplayArtifact(t, projectPath, expected.Domain, operation, file, map[string]any{"status": "passed"})
			}
			if !complete && operation == "create" {
				item["normalizedDiffStatus"] = "failed"
				delete(item, "cleanupStatus")
			}
			operations = append(operations, item)
		}
		domains = append(domains, map[string]any{
			"domain":     expected.Domain,
			"operations": operations,
		})
	}
	body, err := json.Marshal(map[string]any{
		"mode":                "setup-svc-live-replay-evidence",
		"status":              "passed",
		"project":             projectPath,
		"contractVersion":     setupSvcLiveReplayContractVersion,
		"contractFingerprint": setupSvcLiveReplayExpectedContractFingerprint(),
		"domains":             domains,
	})
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, path, string(body))
}

func writeSetupSvcLiveReplayMissingRuntimeReadbackEvidenceArtifacts(t *testing.T, projectPath string, manifestPath string) {
	t.Helper()
	writeSetupSvcLiveReplayManifest(t, manifestPath, true)
	writeSetupSvcLiveReplayArtifact(t, projectPath, "objects", "create", "outputs/setup-svc-live-replay/objects/create/setup-svc.json", map[string]any{
		"status":              "passed",
		"runtimeEffects":      setupSvcLiveReplayRuntimeEffects("objects"),
		"runtimeEffectChecks": []map[string]any{},
		"tableSnapshots":      setupSvcLiveReplayTestTableSnapshots("objects"),
	})
	writeSetupSvcLiveReplayArtifact(t, projectPath, "objects", "create", "outputs/setup-svc-live-replay/objects/create/query-readback.json", map[string]any{
		"status":                    "passed",
		"queryReadbackExpectations": setupSvcLiveReplayQueryReadbackExpectations("objects"),
		"queryShape":                map[string]any{"fields": []string{"id"}},
		"readbackShape":             map[string]any{"fields": []string{"id"}},
		"readbackChecks": map[string]any{
			"requiredFields":          []string{"id"},
			"requiredRelationships":   []string{"metadata-table-links"},
			"relationshipChecks":      []map[string]any{{"name": "metadata-table-links", "status": "passed", "source": "tp_sys_object", "target": "tp_sys_schemetable", "field": "id"}},
			"missingFields":           0,
			"missingRelationships":    0,
			"mismatchedFields":        0,
			"brokenRelationships":     0,
			"unreadableRelationships": 0,
		},
		"readbackTables": setupSvcLiveReplayTestReadbackTables("objects"),
	})
}

func writeSetupSvcLiveReplayManifestWithoutArtifacts(t *testing.T, path string) {
	t.Helper()
	projectPath := setupSvcLiveReplayProjectPathFromManifest(path)
	domains := []map[string]any{}
	for _, expected := range setupSvcLiveReplayDomains() {
		operations := []map[string]any{}
		for _, operation := range expected.Operations {
			item := map[string]any{
				"operation":                     operation,
				"setupSvcEvidenceStatus":        "passed",
				"metadataServiceEvidenceStatus": "passed",
				"queryEvidenceStatus":           "passed",
				"normalizedDiffStatus":          "passed",
			}
			if operation != "query" {
				item["cleanupStatus"] = "passed"
			}
			item["evidenceFiles"] = setupSvcLiveReplayEvidenceFiles(expected.Domain, operation, operation != "query")
			operations = append(operations, item)
		}
		domains = append(domains, map[string]any{
			"domain":     expected.Domain,
			"operations": operations,
		})
	}
	body, err := json.Marshal(map[string]any{
		"mode":                "setup-svc-live-replay-evidence",
		"status":              "passed",
		"project":             projectPath,
		"contractVersion":     setupSvcLiveReplayContractVersion,
		"contractFingerprint": setupSvcLiveReplayExpectedContractFingerprint(),
		"domains":             domains,
	})
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, path, string(body))
}

func writeCurrentSetupSvcLiveReplayEvidenceBundle(t *testing.T, projectPath string) {
	t.Helper()
	result, err := buildSetupSvcLiveReplayEvidenceBundleApplyResult(projectPath, "", true, setupSvcParityEvidenceBundleApproval)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "applied" {
		t.Fatalf("expected applied evidence bundle, got %#v", result)
	}
}

func rewriteSetupSvcLiveReplayManifest(t *testing.T, path string, rewrite func(map[string]any)) {
	t.Helper()
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(payload, &manifest); err != nil {
		t.Fatal(err)
	}
	rewrite(manifest)
	body, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, path, string(body))
}

func writeSetupSvcLiveReplayParityMatrix(t *testing.T, projectPath string, statusOverrides map[string]string) {
	t.Helper()
	domains := []map[string]any{}
	evidenceDomains := []map[string]any{}
	for _, expected := range setupSvcLiveReplayDomains() {
		status := "covered"
		if override := strings.TrimSpace(statusOverrides[expected.Domain]); override != "" {
			status = override
		}
		domains = append(domains, map[string]any{
			"domain":                    expected.Domain,
			"setupSvcReferences":        []string{"test/setup-svc/" + expected.Domain},
			"operations":                expected.Operations,
			"queryIncluded":             true,
			"status":                    status,
			"requiredTables":            expected.RequiredTables,
			"runtimeEffects":            setupSvcLiveReplayTestRuntimeEffects(expected.Domain),
			"queryReadbackExpectations": setupSvcLiveReplayTestQueryReadbackExpectations(expected.Domain),
		})
		operationEvidence := []map[string]any{}
		for _, operation := range expected.Operations {
			operationEvidence = append(operationEvidence, map[string]any{
				"operation":  operation,
				"testClass":  "GeneratedParityReplayTest",
				"testMethod": "generatedParityReplayCoversMatrixOperation",
			})
		}
		evidenceDomains = append(evidenceDomains, map[string]any{
			"domain":            expected.Domain,
			"operationEvidence": operationEvidence,
		})
	}
	body, err := json.Marshal(map[string]any{
		"version": 1,
		"domains": domains,
	})
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(projectPath, "cc-metadata-service/src/test/resources/parity/msapi-setup-svc-parity-matrix.json"), string(body))
	writeTestFile(t, filepath.Join(projectPath, "test-fixtures/msapi/parity/msapi-setup-svc-parity-matrix.json"), string(body))
	evidenceBody, err := json.Marshal(map[string]any{
		"version": 1,
		"domains": evidenceDomains,
	})
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(projectPath, "cc-metadata-service/src/test/resources/parity/msapi-parity-test-evidence.json"), string(evidenceBody))
	writeTestFile(t, filepath.Join(projectPath, "test-fixtures/msapi/parity/msapi-parity-test-evidence.json"), string(evidenceBody))
}

func writeGeneratedSetupSvcLiveReplayTestSource(t *testing.T, projectPath string) {
	t.Helper()
	writeTestFile(t, filepath.Join(projectPath, "cc-metadata-service/src/test/java/com/cloudcc/metadata/parity/GeneratedParityReplayTest.java"), `
package com.cloudcc.metadata.parity;

class GeneratedParityReplayTest {
    void generatedParityReplayCoversMatrixOperation() {
    }
}
`)
}

func setupSvcLiveReplayTestRuntimeEffects(domain string) []string {
	return setupSvcLiveReplayRuntimeEffects(domain)
}

func setupSvcLiveReplayTestQueryReadbackExpectations(domain string) []string {
	return setupSvcLiveReplayQueryReadbackExpectations(domain)
}

func rewriteSetupSvcLiveReplayParityMatrix(t *testing.T, path string, rewrite func(map[string]any)) {
	t.Helper()
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var matrix map[string]any
	if err := json.Unmarshal(payload, &matrix); err != nil {
		t.Fatal(err)
	}
	rewrite(matrix)
	body, err := json.Marshal(matrix)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, path, string(body))
}

func rewriteSetupSvcLiveReplayTestEvidence(t *testing.T, path string, rewrite func(map[string]any)) {
	t.Helper()
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var evidence map[string]any
	if err := json.Unmarshal(payload, &evidence); err != nil {
		t.Fatal(err)
	}
	rewrite(evidence)
	body, err := json.Marshal(evidence)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, path, string(body))
}

func rewriteSetupSvcLiveReplayManifestOperation(t *testing.T, path string, domainName string, operationName string, rewrite func(map[string]any)) {
	t.Helper()
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(payload, &manifest); err != nil {
		t.Fatal(err)
	}
	for _, rawDomain := range manifest["domains"].([]any) {
		domain := rawDomain.(map[string]any)
		if normalizeDomain(domain["domain"].(string)) != normalizeDomain(domainName) {
			continue
		}
		for _, rawOperation := range domain["operations"].([]any) {
			operation := rawOperation.(map[string]any)
			if strings.EqualFold(operation["operation"].(string), operationName) {
				rewrite(operation)
			}
		}
	}
	body, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, path, string(body))
}

func markSetupSvcLiveReplayManifestOperationStatus(t *testing.T, path string, domainName string, operationName string, field string, value string) {
	t.Helper()
	rewriteSetupSvcLiveReplayManifestOperation(t, path, domainName, operationName, func(operation map[string]any) {
		operation[field] = value
	})
}

func setupSvcLiveReplayManifestOperationField(t *testing.T, manifest map[string]any, domainName string, operationName string, field string) string {
	t.Helper()
	for _, rawDomain := range manifest["domains"].([]any) {
		domain := rawDomain.(map[string]any)
		if normalizeDomain(domain["domain"].(string)) != normalizeDomain(domainName) {
			continue
		}
		for _, rawOperation := range domain["operations"].([]any) {
			operation := rawOperation.(map[string]any)
			if strings.EqualFold(operation["operation"].(string), operationName) {
				value, _ := operation[field].(string)
				return value
			}
		}
	}
	return ""
}

func readTestJSONMap(t *testing.T, path string) map[string]any {
	t.Helper()
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(payload, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func setReadyMetadataServiceDatasourceEnv(t *testing.T, label string) {
	t.Helper()
	normalized := strings.NewReplacer("_", "-", " ", "-").Replace(strings.ToLower(strings.TrimSpace(label)))
	if normalized == "" {
		normalized = "test"
	}
	t.Setenv("MDS_RUNTIME_MODE", "self-hosted")
	t.Setenv("MDS_SERVER_PORT", "18087")
	t.Setenv("MDS_JDBC_URL", "jdbc:mysql://"+normalized+"-db-host:3306/"+strings.ReplaceAll(normalized, "-", "_")+"_metadata")
	t.Setenv("MDS_DB_USERNAME", normalized+"-user")
	t.Setenv("MDS_DB_PASSWORD", normalized+"-password")
	t.Setenv("MDS_DB_DRIVER", "com.mysql.cj.jdbc.Driver")
}

func setupSvcLiveReplayProjectPathFromManifest(path string) string {
	return strings.TrimSuffix(path, filepath.Join("outputs", "setup-svc-live-replay", "manifest.json"))
}

func writeSetupSvcLiveReplayArtifact(t *testing.T, projectPath string, domain string, operation string, file string, payload map[string]any) {
	t.Helper()
	if payload == nil {
		payload = map[string]any{}
	}
	if _, ok := payload["contractVersion"]; !ok {
		payload["contractVersion"] = setupSvcLiveReplayContractVersion
	}
	if _, ok := payload["contractFingerprint"]; !ok {
		payload["contractFingerprint"] = setupSvcLiveReplayExpectedContractFingerprint()
	}
	if _, ok := payload["project"]; !ok {
		payload["project"] = projectPath
	}
	payload["domain"] = domain
	payload["operation"] = operation
	payload["artifactType"] = setupSvcLiveReplayArtifactType(file)
	if setupSvcLiveReplayArtifactType(file) == "normalized-diff" {
		if _, ok := payload["totals"]; !ok {
			payload["totals"] = map[string]any{"missingRows": 0, "mismatchedValues": 0, "differences": 0}
		}
	}
	if setupSvcLiveReplayArtifactType(file) == "query-readback" {
		if _, ok := payload["readbackChecks"]; !ok {
			payload["readbackChecks"] = map[string]any{
				"requiredFields":            []string{"id"},
				"requiredRelationships":     []string{"metadata-table-links"},
				"relationshipChecks":        []map[string]any{{"name": "metadata-table-links", "status": "passed"}},
				"readbackExpectationChecks": setupSvcLiveReplayPassedExpectationChecks(setupSvcLiveReplayQueryReadbackExpectations(domain)),
				"missingFields":             0,
				"missingRelationships":      0,
				"mismatchedFields":          0,
				"brokenRelationships":       0,
				"unreadableRelationships":   0,
			}
		}
		if _, ok := payload["readbackTables"]; !ok {
			payload["readbackTables"] = setupSvcLiveReplayTestReadbackTables(domain)
		}
	}
	if artifactType := setupSvcLiveReplayArtifactType(file); artifactType == "setup-svc" || artifactType == "metadata-service" {
		if _, ok := payload["tableSnapshots"]; !ok {
			payload["tableSnapshots"] = setupSvcLiveReplayTestTableSnapshots(domain)
		}
		if _, ok := payload["runtimeEffectChecks"]; !ok {
			payload["runtimeEffectChecks"] = setupSvcLiveReplayPassedExpectationChecks(setupSvcLiveReplayRuntimeEffects(domain))
		}
		if artifactType == "metadata-service" {
			if _, ok := payload["metadataServiceDatasource"]; !ok {
				payload["metadataServiceDatasource"] = map[string]any{
					"status":                 "ready",
					"readyForRealDatasource": true,
					"jdbcUrlConfigured":      true,
					"usernameConfigured":     true,
					"passwordConfigured":     true,
					"driverConfigured":       true,
					"jdbcUrlLooksDefaultH2":  false,
					"redaction":              "test fixture reports datasource readiness without secret values",
				}
			}
		}
	}
	if setupSvcLiveReplayArtifactType(file) == "cleanup" {
		if _, ok := payload["cleanupChecks"]; !ok {
			payload["cleanupChecks"] = map[string]any{
				"deletedRows":   1,
				"remainingRows": 0,
				"residualRows":  0,
				"orphanRows":    0,
				"errors":        0,
				"failures":      0,
			}
		}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(projectPath, file), string(body))
}

func setupSvcLiveReplayPassedExpectationChecks(expectations []string) []map[string]any {
	checks := make([]map[string]any, 0, len(expectations))
	for _, expectation := range expectations {
		checks = append(checks, map[string]any{
			"name":   expectation,
			"status": "passed",
			"evidence": map[string]any{
				"checked": true,
			},
		})
	}
	return checks
}

func setupSvcLiveReplayTestTableSnapshots(domain string) []map[string]any {
	for _, expected := range setupSvcLiveReplayDomains() {
		if normalizeDomain(expected.Domain) != normalizeDomain(domain) {
			continue
		}
		snapshots := make([]map[string]any, 0, len(expected.RequiredTables))
		for _, table := range expected.RequiredTables {
			snapshots = append(snapshots, map[string]any{
				"table":       table,
				"rowCount":    1,
				"columns":     []string{"id"},
				"primaryKeys": []string{"id"},
				"rows":        []map[string]any{{"id": table + "-replay-id"}},
			})
		}
		return snapshots
	}
	return nil
}

func setupSvcLiveReplayTestReadbackTables(domain string) []map[string]any {
	for _, expected := range setupSvcLiveReplayDomains() {
		if normalizeDomain(expected.Domain) != normalizeDomain(domain) {
			continue
		}
		tables := make([]map[string]any, 0, len(expected.RequiredTables))
		for _, table := range expected.RequiredTables {
			tables = append(tables, map[string]any{
				"table":                 table,
				"rowCount":              1,
				"requiredFields":        []string{"id"},
				"columns":               []string{"id"},
				"requiredRelationships": []string{"metadata-table-links"},
				"rows":                  []map[string]any{{"id": table + "-readback-id"}},
			})
		}
		return tables
	}
	return nil
}

func containsStringItem(items []any, prefix string) bool {
	for _, item := range items {
		if strings.HasPrefix(item.(string), prefix) {
			return true
		}
	}
	return false
}

func containsStringFragment(items []any, fragment string) bool {
	for _, item := range items {
		if strings.Contains(item.(string), fragment) {
			return true
		}
	}
	return false
}

func containsAnyString(values []any, want string) bool {
	for _, raw := range values {
		if strings.Contains(fmt.Sprint(raw), want) {
			return true
		}
	}
	return false
}

func containsMapCount(items []any, name string, count int) bool {
	for _, item := range items {
		entry := item.(map[string]any)
		if entry["name"] == name && int(entry["count"].(float64)) == count {
			return true
		}
	}
	return false
}

func containsSectionCount(items []any, section string, count int) bool {
	for _, item := range items {
		entry := item.(map[string]any)
		if entry["section"] == section && int(entry["count"].(float64)) == count {
			return true
		}
	}
	return false
}

func containsDomainOperationCount(items []any, domain string, operation string, count int) bool {
	for _, item := range items {
		entry := item.(map[string]any)
		if entry["domain"] == domain && entry["operation"] == operation && int(entry["count"].(float64)) == count {
			return true
		}
	}
	return false
}

func repairPlanDomainOperation(items []any, domain string, operation string) map[string]any {
	for _, item := range items {
		entry := item.(map[string]any)
		if entry["domain"] == domain && entry["operation"] == operation {
			return entry
		}
	}
	return nil
}

func containsRepairQueue(items []any, artifactType string, section string) bool {
	for _, item := range items {
		entry := item.(map[string]any)
		if entry["artifactType"] != artifactType || entry["evidenceSection"] != section || int(entry["count"].(float64)) != 1 {
			continue
		}
		if int(entry["sourceFiles"].(float64)) != int(entry["count"].(float64)) ||
			int(entry["targetFiles"].(float64)) != int(entry["count"].(float64)) {
			continue
		}
		if !strings.Contains(entry["capturePlanCommand"].(string), "setup-svc-live-replay-capture-plan") ||
			!strings.Contains(entry["worklistCommand"].(string), "setup-svc-live-replay-worklist") ||
			!strings.Contains(entry["saveWorklistCommand"].(string), " > ") ||
			!containsRepairQueueSourceChecklist([]any{item}, artifactType, section) ||
			!containsRepairQueueSourceExecution([]any{item}, artifactType, section) {
			continue
		}
		return true
	}
	return false
}

func containsRepairQueueWithPositiveCount(items []any, artifactType string, section string) bool {
	for _, item := range items {
		entry := item.(map[string]any)
		if entry["artifactType"] == artifactType &&
			entry["evidenceSection"] == section &&
			int(entry["count"].(float64)) > 0 &&
			int(entry["sourceFiles"].(float64)) == int(entry["count"].(float64)) &&
			int(entry["targetFiles"].(float64)) == int(entry["count"].(float64)) {
			return true
		}
	}
	return false
}

func containsRepairQueueSourceChecklist(items []any, artifactType string, section string) bool {
	for _, item := range items {
		entry := item.(map[string]any)
		if entry["artifactType"] != artifactType || entry["evidenceSection"] != section {
			continue
		}
		sourceCommand, ok := entry["sourceChecklistCommand"].(string)
		if !ok || !strings.Contains(sourceCommand, "setup-svc-live-replay-source-checklist") ||
			!strings.Contains(sourceCommand, "--artifact-type "+artifactType) ||
			!strings.Contains(sourceCommand, "--evidence-section "+section) ||
			!strings.Contains(sourceCommand, "--source-readiness incomplete") {
			continue
		}
		saveCommand, ok := entry["saveSourceChecklistCommand"].(string)
		if !ok || !strings.Contains(saveCommand, sourceCommand) || !strings.Contains(saveCommand, " > ") {
			continue
		}
		return true
	}
	return false
}

func containsRepairQueueSourceExecution(items []any, artifactType string, section string) bool {
	for _, item := range items {
		entry := item.(map[string]any)
		if entry["artifactType"] != artifactType || entry["evidenceSection"] != section {
			continue
		}
		sourceCommand, ok := entry["sourceExecutionCommand"].(string)
		if !ok || !strings.Contains(sourceCommand, "setup-svc-live-replay-source-execution-packet") ||
			!strings.Contains(sourceCommand, "--artifact-type "+artifactType) ||
			!strings.Contains(sourceCommand, "--evidence-section "+section) ||
			!strings.Contains(sourceCommand, "--source-readiness incomplete") {
			continue
		}
		saveCommand, ok := entry["saveSourceExecutionPacketCommand"].(string)
		if !ok || !strings.Contains(saveCommand, sourceCommand) || !strings.Contains(saveCommand, " > ") {
			continue
		}
		return true
	}
	return false
}

func anySliceFromRepairQueues(items []setupSvcLiveReplayEvidenceImportRepairQueue) []any {
	result := make([]any, 0, len(items))
	for _, item := range items {
		result = append(result, map[string]any{
			"artifactType":                     item.ArtifactType,
			"evidenceSection":                  item.EvidenceSection,
			"count":                            float64(item.Count),
			"sourceFiles":                      float64(item.SourceFiles),
			"targetFiles":                      float64(item.TargetFiles),
			"capturePlanCommand":               item.CapturePlanCommand,
			"worklistCommand":                  item.WorklistCommand,
			"saveWorklistCommand":              item.SaveWorklistCommand,
			"sourceChecklistCommand":           item.SourceChecklistCommand,
			"saveSourceChecklistCommand":       item.SaveSourceChecklistCommand,
			"sourceExecutionCommand":           item.SourceExecutionCommand,
			"saveSourceExecutionPacketCommand": item.SaveSourceExecutionCommand,
		})
	}
	return result
}

func containsBatchSaveCommand(items []any, artifactType string, section string, batchIndex int, pathFragment string) bool {
	for _, item := range items {
		entry := item.(map[string]any)
		if entry["artifactType"] != artifactType ||
			entry["evidenceSection"] != section ||
			int(entry["batchIndex"].(float64)) != batchIndex {
			continue
		}
		if !strings.Contains(entry["suggestedWorklistPath"].(string), pathFragment) ||
			!strings.Contains(entry["saveWorklistCommand"].(string), "setup-svc-live-replay-worklist") ||
			!strings.Contains(entry["saveWorklistCommand"].(string), " > ") ||
			!strings.Contains(entry["dryRunImportCommand"].(string), "setup-svc-live-replay-evidence-import @") ||
			!strings.Contains(entry["executeImportCommand"].(string), setupSvcParityEvidenceImportApproval) {
			continue
		}
		return true
	}
	return false
}

func sourceChecklistHasArtifactCount(items []any, artifactType string, sourceFiles int, records int) bool {
	for _, item := range items {
		entry := item.(map[string]any)
		if entry["artifactType"] == artifactType &&
			int(entry["sourceFiles"].(float64)) == sourceFiles &&
			int(entry["records"].(float64)) == records {
			return true
		}
	}
	return false
}

func sourceChecklistHasMissingSectionCount(items []any, evidenceSection string, sourceFiles int, artifactTypes ...string) bool {
	for _, item := range items {
		entry := item.(map[string]any)
		if entry["evidenceSection"] != evidenceSection ||
			int(entry["sourceFiles"].(float64)) != sourceFiles {
			continue
		}
		values := entry["artifactTypes"].([]any)
		for _, artifactType := range artifactTypes {
			if !containsStringItem(values, artifactType) {
				return false
			}
		}
		return true
	}
	return false
}

func sourceHealthHasArtifactType(items []any, artifactType string, sourceFiles int, missingSectionInstances int, topSection string) bool {
	for _, item := range items {
		entry := item.(map[string]any)
		if entry["artifactType"] != artifactType ||
			int(entry["sourceFiles"].(float64)) != sourceFiles ||
			int(entry["missingSectionInstances"].(float64)) != missingSectionInstances {
			continue
		}
		topSections := entry["topMissingSections"].([]any)
		return containsStringItem(topSections, topSection)
	}
	return false
}

func sourceHealthHasMissingSection(items []any, artifactType string, evidenceSection string, sourceFiles int) bool {
	for _, item := range items {
		entry := item.(map[string]any)
		if entry["artifactType"] == artifactType &&
			entry["evidenceSection"] == evidenceSection &&
			int(entry["sourceFiles"].(float64)) == sourceFiles &&
			int(entry["targetFiles"].(float64)) == sourceFiles &&
			strings.Contains(entry["queueCommand"].(string), "setup-svc-live-replay-source-checklist") {
			return true
		}
	}
	return false
}

func evidenceImportHasIssueCount(items []any, name string, count int) bool {
	for _, item := range items {
		entry := item.(map[string]any)
		if entry["name"] == name && int(entry["count"].(float64)) == count {
			return true
		}
	}
	return false
}

func sourceExecutionHasGroup(items []any, artifactType string, sourceSystem string, captureMode string, sourceFiles int, targetFiles int, domainOperations int, evidenceSections ...string) bool {
	for _, item := range items {
		entry := item.(map[string]any)
		if entry["artifactType"] != artifactType ||
			entry["sourceSystem"] != sourceSystem ||
			entry["captureMode"] != captureMode ||
			int(entry["sourceFiles"].(float64)) != sourceFiles ||
			int(entry["targetFiles"].(float64)) != targetFiles ||
			int(entry["domainOperations"].(float64)) != domainOperations {
			continue
		}
		values := entry["evidenceSections"].([]any)
		for _, section := range evidenceSections {
			if !containsStringItem(values, section) {
				return false
			}
		}
		if !strings.Contains(entry["suggestedBatchPath"].(string), artifactType) ||
			!strings.Contains(entry["suggestedBatchPath"].(string), "source-capture-batch") ||
			!strings.Contains(entry["saveBatchCommand"].(string), "setup-svc-live-replay-source-execution-packet") ||
			!strings.Contains(entry["saveBatchCommand"].(string), "--capture-mode "+captureMode) ||
			!strings.Contains(entry["postCaptureCheckCommand"].(string), "setup-svc-live-replay-source-checklist") {
			return false
		}
		return true
	}
	return false
}

func sourceExecutionHasArtifactOrder(items []any, expected []string) bool {
	if len(items) != len(expected) {
		return false
	}
	for i, item := range items {
		entry := item.(map[string]any)
		if entry["artifactType"] != expected[i] {
			return false
		}
	}
	return true
}

func sourceExecutionBatchSaveCommandsInOrder(result map[string]any, expected []string) bool {
	commands, ok := result["batchSaveCommands"].([]any)
	if !ok || len(commands) != len(expected) {
		return false
	}
	for i, expectedArtifact := range expected {
		command, ok := commands[i].(string)
		if !ok || !strings.Contains(command, expectedArtifact) || !strings.Contains(command, "source-capture-batch") {
			return false
		}
	}
	return true
}

func sourceExecutionRunbookHasDependencyOrder(result map[string]any, expected []string) bool {
	runbook, ok := result["executionRunbook"].([]any)
	if !ok || len(runbook) != len(expected) {
		return false
	}
	completed := []string{}
	for i, expectedArtifact := range expected {
		step := runbook[i].(map[string]any)
		if step["artifactType"] != expectedArtifact || int(step["order"].(float64)) != i+1 {
			return false
		}
		dependsOn := []any{}
		if raw, ok := step["dependsOn"].([]any); ok {
			dependsOn = raw
		}
		if len(dependsOn) != len(completed) {
			return false
		}
		for j, dependency := range completed {
			if dependsOn[j] != dependency {
				return false
			}
		}
		combined := strings.Join([]string{
			step["gate"].(string),
			step["saveBatchCommand"].(string),
			step["saveImportBatchCommand"].(string),
			step["postCaptureCheckCommand"].(string),
			step["dryRunImportCommand"].(string),
			step["approvedImportCommand"].(string),
			step["completionAuditCommand"].(string),
		}, "\n")
		for _, fragment := range []string{"setup-svc-live-replay-source-execution-packet", "setup-svc-live-replay-source-checklist", "setup-svc-live-replay-evidence-import", "setup-svc-live-replay-completion-audit"} {
			if !strings.Contains(combined, fragment) {
				return false
			}
		}
		importPath, ok := step["suggestedImportBatchPath"].(string)
		if !ok || !strings.Contains(importPath, expectedArtifact) || !strings.Contains(importPath, "source-capture-batch-readiness-complete.json") {
			return false
		}
		batchPath, ok := step["batchPath"].(string)
		if !ok || !strings.Contains(batchPath, expectedArtifact) || !strings.Contains(batchPath, "source-capture-batch-readiness-incomplete.json") {
			return false
		}
		suggestedBatchPath, ok := step["suggestedBatchPath"].(string)
		if !ok || suggestedBatchPath != batchPath {
			return false
		}
		if strings.Contains(step["dryRunImportCommand"].(string), "readiness-incomplete") ||
			!strings.Contains(step["dryRunImportCommand"].(string), "readiness-complete") {
			return false
		}
		if !sourceExecutionRunbookStepHasCaptureCommand(step, expectedArtifact) {
			return false
		}
		if sections, ok := step["evidenceSections"].([]any); !ok || len(sections) == 0 {
			return false
		}
		completed = append(completed, expectedArtifact)
	}
	return true
}

func sourceExecutionRunbookStepHasCaptureCommand(step map[string]any, artifactType string) bool {
	captureMode, _ := step["captureMode"].(string)
	dryRun, dryRunOK := step["dryRunCaptureCommand"].(string)
	execute, executeOK := step["executeCaptureCommand"].(string)
	switch {
	case artifactType == "metadata-service" && captureMode == "msapi_plan_apply_snapshot_capture":
		return dryRunOK && executeOK &&
			strings.Contains(dryRun, "setup-svc-live-replay-metadata-service-apply-capture") &&
			strings.Contains(dryRun, "--dry-run") &&
			strings.Contains(execute, "CLOUDCC_SETUP_SVC_PARITY_METADATA_SERVICE_APPLY_CAPTURE_APPROVED")
	case artifactType == "metadata-service" && captureMode == "msapi_scan_snapshot_capture":
		return dryRunOK && executeOK &&
			strings.Contains(dryRun, "setup-svc-live-replay-metadata-service-query-scan-capture") &&
			strings.Contains(dryRun, "--dry-run") &&
			strings.Contains(execute, "CLOUDCC_SETUP_SVC_PARITY_METADATA_SERVICE_QUERY_SCAN_CAPTURE_APPROVED")
	case artifactType == "query-readback":
		return dryRunOK && executeOK &&
			strings.Contains(dryRun, "setup-svc-live-replay-query-readback-capture") &&
			strings.Contains(dryRun, "--dry-run") &&
			strings.Contains(execute, "CLOUDCC_SETUP_SVC_PARITY_QUERY_READBACK_CAPTURE_APPROVED")
	case artifactType == "normalized-diff":
		return dryRunOK && executeOK &&
			strings.Contains(dryRun, "setup-svc-live-replay-normalized-diff") &&
			strings.Contains(dryRun, "--dry-run") &&
			strings.Contains(execute, "CLOUDCC_SETUP_SVC_PARITY_NORMALIZED_DIFF_APPROVED")
	default:
		required, ok := step["manualCaptureRequired"].(bool)
		return ok && required && !dryRunOK && !executeOK
	}
}

func sourceExecutionOperatorBatchHasCommands(items []any, artifactType string, fragments ...string) bool {
	for _, item := range items {
		entry := item.(map[string]any)
		if entry["artifactType"] != artifactType {
			continue
		}
		combined := strings.Join([]string{
			entry["saveBatchCommand"].(string),
			entry["saveImportBatchCommand"].(string),
			entry["dryRunImportCommand"].(string),
			entry["approvedImportCommand"].(string),
			entry["completionAuditCommand"].(string),
		}, "\n")
		for _, fragment := range fragments {
			if !strings.Contains(combined, fragment) {
				return false
			}
		}
		return true
	}
	return false
}

func sourceExecutionOperatorBatchesHaveTargetFiles(items []any) bool {
	for _, item := range items {
		entry := item.(map[string]any)
		sourceFiles, sourceOK := entry["sourceFiles"].(float64)
		targetFiles, targetOK := entry["targetFiles"].(float64)
		if !sourceOK || !targetOK || int(sourceFiles) <= 0 || int(targetFiles) != int(sourceFiles) {
			return false
		}
		batchPath, batchOK := entry["batchPath"].(string)
		suggestedBatchPath, suggestedOK := entry["suggestedBatchPath"].(string)
		if !batchOK || !suggestedOK || batchPath == "" || batchPath != suggestedBatchPath {
			return false
		}
	}
	return len(items) > 0
}

func sourceExecutionMetadataBatchesHaveDatasourceReadiness(items []any) bool {
	metadataBatches := 0
	for _, item := range items {
		entry := item.(map[string]any)
		datasource, hasDatasource := entry["metadataServiceDatasource"].(map[string]any)
		if entry["artifactType"] != "metadata-service" {
			if hasDatasource {
				return false
			}
			continue
		}
		metadataBatches++
		if !hasDatasource || datasource["readyForRealDatasource"] != true || datasource["jdbcUrlSource"] != "env:MDS_JDBC_URL" {
			return false
		}
	}
	return metadataBatches == 2
}

func sourceExecutionHasBatchSaveScript(result map[string]any, commandCount int, pathFragment string) bool {
	commands, ok := result["batchSaveCommands"].([]any)
	if !ok || len(commands) != commandCount {
		return false
	}
	if !containsStringWithFragments(commands, "setup-svc-live-replay-source-execution-packet", pathFragment, " > ") {
		return false
	}
	script, ok := result["batchSaveScript"].(string)
	if !ok ||
		!strings.Contains(script, "# Generated by setup-svc-live-replay-source-execution-packet") ||
		!strings.Contains(script, "mkdir -p ") ||
		!strings.Contains(script, pathFragment) ||
		!strings.Contains(script, "setup-svc-live-replay-source-execution-packet") {
		return false
	}
	scriptPath, ok := result["batchSaveScriptPath"].(string)
	if !ok || !strings.HasSuffix(scriptPath, ".sh") {
		return false
	}
	saveCommand, ok := result["saveBatchSaveScriptCommand"].(string)
	return ok &&
		strings.Contains(saveCommand, "setup-svc-live-replay-source-execution-packet") &&
		strings.Contains(saveCommand, "jq -r '.batchSaveScript'") &&
		strings.Contains(saveCommand, "chmod +x") &&
		strings.Contains(saveCommand, scriptPath)
}

func sourceExecutionHasImportBatchSaveScript(result map[string]any, commandCount int, pathFragment string) bool {
	commands, ok := result["importBatchSaveCommands"].([]any)
	if !ok || len(commands) != commandCount {
		return false
	}
	if !containsStringWithFragments(commands, "setup-svc-live-replay-source-execution-packet", pathFragment, "--source-readiness complete", " > ") {
		return false
	}
	for _, raw := range commands {
		command, ok := raw.(string)
		if !ok || strings.Contains(command, "readiness-incomplete") {
			return false
		}
	}
	script, ok := result["importBatchSaveScript"].(string)
	if !ok ||
		!strings.Contains(script, "# Generated by setup-svc-live-replay-source-execution-packet") ||
		!strings.Contains(script, "mkdir -p ") ||
		!strings.Contains(script, pathFragment) ||
		!strings.Contains(script, "--source-readiness complete") ||
		!strings.Contains(script, "mktemp ") ||
		!strings.Contains(script, "jq '(.items // []) | length'") ||
		!strings.Contains(script, "SKIP no complete sources") ||
		!strings.Contains(script, "mv \"$tmp\"") ||
		strings.Contains(script, "readiness-incomplete") {
		return false
	}
	scriptPath, ok := result["importBatchSaveScriptPath"].(string)
	if !ok || !strings.HasSuffix(scriptPath, ".sh") || !strings.Contains(scriptPath, "source-capture-import-batches") {
		return false
	}
	saveCommand, ok := result["saveImportBatchSaveScriptCommand"].(string)
	return ok &&
		strings.Contains(saveCommand, "setup-svc-live-replay-source-execution-packet") &&
		strings.Contains(saveCommand, "jq -r '.importBatchSaveScript'") &&
		strings.Contains(saveCommand, "chmod +x") &&
		strings.Contains(saveCommand, scriptPath)
}

func sourceExecutionHasRunbookMarkdown(result map[string]any, stepCount int, artifactType string) bool {
	markdown, ok := result["runbookMarkdown"].(string)
	if !ok ||
		!strings.Contains(markdown, "# setup-svc live replay source capture runbook") ||
		!strings.Contains(markdown, "## Dependency Order") ||
		!strings.Contains(markdown, "## Capture Steps") ||
		!strings.Contains(markdown, "## Stop Conditions") ||
		!strings.Contains(markdown, "Approved capture") ||
		!strings.Contains(markdown, "setup-svc-live-replay-evidence-import") ||
		!strings.Contains(markdown, "setup-svc-live-replay-completion-audit") ||
		!strings.Contains(markdown, artifactType) ||
		strings.Count(markdown, "### ") != stepCount {
		return false
	}
	path, ok := result["runbookMarkdownPath"].(string)
	if !ok || !strings.HasSuffix(path, "-runbook.md") {
		return false
	}
	saveCommand, ok := result["saveRunbookMarkdownCommand"].(string)
	return ok &&
		strings.Contains(saveCommand, "setup-svc-live-replay-source-execution-packet") &&
		strings.Contains(saveCommand, "jq -r '.runbookMarkdown'") &&
		strings.Contains(saveCommand, path)
}

func sourceChecklistHasNextQueueCommand(items []any, artifactType string, evidenceSection string, sourceFiles int, sourceReadiness string) bool {
	for _, item := range items {
		entry := item.(map[string]any)
		if entry["evidenceSection"] != evidenceSection ||
			int(entry["sourceFiles"].(float64)) != sourceFiles {
			continue
		}
		if artifactType == "" {
			if entry["artifactType"] != nil && entry["artifactType"] != "" {
				continue
			}
		} else if entry["artifactType"] != artifactType {
			continue
		}
		if sourceReadiness != "" && entry["sourceReadiness"] != sourceReadiness {
			continue
		}
		worklistCommand := entry["worklistCommand"].(string)
		sourceChecklistCommand := entry["sourceChecklistCommand"].(string)
		if !strings.Contains(worklistCommand, "setup-svc-live-replay-worklist") ||
			!strings.Contains(sourceChecklistCommand, "setup-svc-live-replay-source-checklist") ||
			!strings.Contains(entry["saveWorklistCommand"].(string), " > ") ||
			!strings.Contains(entry["saveSourceChecklistCommand"].(string), " > ") ||
			!strings.Contains(worklistCommand, "--evidence-section "+evidenceSection) ||
			!strings.Contains(sourceChecklistCommand, "--section-status missing") {
			continue
		}
		if artifactType != "" && !strings.Contains(worklistCommand, "--artifact-type "+artifactType) {
			continue
		}
		if sourceReadiness != "" && !strings.Contains(sourceChecklistCommand, "--source-readiness "+sourceReadiness) {
			continue
		}
		return true
	}
	return false
}

func sourceChecklistHasNextQueuePagination(items []any, artifactType string, evidenceSection string, sourceFiles int, offset int, limit int, pageSize int, pageCount int, omittedSourceFiles int) bool {
	for _, item := range items {
		entry := item.(map[string]any)
		if entry["evidenceSection"] != evidenceSection ||
			int(entry["sourceFiles"].(float64)) != sourceFiles {
			continue
		}
		if artifactType == "" {
			if entry["artifactType"] != nil && entry["artifactType"] != "" {
				continue
			}
		} else if entry["artifactType"] != artifactType {
			continue
		}
		return int(entry["offset"].(float64)) == offset &&
			int(entry["limit"].(float64)) == limit &&
			int(entry["pageSize"].(float64)) == pageSize &&
			int(entry["pageCount"].(float64)) == pageCount &&
			int(entry["omittedSourceFiles"].(float64)) == omittedSourceFiles
	}
	return false
}

func sourceChecklistHasNextPageCommands(items []any, artifactType string, evidenceSection string, nextOffset int) bool {
	for _, item := range items {
		entry := item.(map[string]any)
		if entry["evidenceSection"] != evidenceSection {
			continue
		}
		if artifactType == "" {
			if entry["artifactType"] != nil && entry["artifactType"] != "" {
				continue
			}
		} else if entry["artifactType"] != artifactType {
			continue
		}
		nextWorklist, ok := entry["nextPageWorklistCommand"].(string)
		if !ok {
			continue
		}
		saveNextWorklist, ok := entry["saveNextPageWorklistCommand"].(string)
		if !ok {
			continue
		}
		nextChecklist, ok := entry["nextPageSourceChecklistCommand"].(string)
		if !ok {
			continue
		}
		saveNextChecklist, ok := entry["saveNextPageSourceChecklistCommand"].(string)
		if !ok {
			continue
		}
		offsetFragment := fmt.Sprintf("--offset %d", nextOffset)
		if strings.Contains(nextWorklist, "setup-svc-live-replay-worklist") &&
			strings.Contains(nextWorklist, offsetFragment) &&
			strings.Contains(nextChecklist, "setup-svc-live-replay-source-checklist") &&
			strings.Contains(nextChecklist, offsetFragment) &&
			strings.Contains(saveNextWorklist, " > ") &&
			strings.Contains(saveNextChecklist, " > ") {
			return true
		}
	}
	return false
}

func sourceChecklistHasAllPageSaveCommands(items []any, artifactType string, evidenceSection string, pageCount int, firstOffset int, lastOffset int) bool {
	for _, item := range items {
		entry := item.(map[string]any)
		if entry["evidenceSection"] != evidenceSection {
			continue
		}
		if artifactType == "" {
			if entry["artifactType"] != nil && entry["artifactType"] != "" {
				continue
			}
		} else if entry["artifactType"] != artifactType {
			continue
		}
		worklistCommands, ok := entry["pageWorklistSaveCommands"].([]any)
		if !ok || len(worklistCommands) != pageCount {
			continue
		}
		checklistCommands, ok := entry["pageSourceChecklistSaveCommands"].([]any)
		if !ok || len(checklistCommands) != pageCount {
			continue
		}
		firstOffsetFragment := fmt.Sprintf("--offset %d", firstOffset)
		lastOffsetFragment := fmt.Sprintf("--offset %d", lastOffset)
		if containsStringWithFragments(worklistCommands, "setup-svc-live-replay-worklist", firstOffsetFragment, " > ") &&
			containsStringWithFragments(worklistCommands, "setup-svc-live-replay-worklist", lastOffsetFragment, " > ") &&
			containsStringWithFragments(checklistCommands, "setup-svc-live-replay-source-checklist", firstOffsetFragment, " > ") &&
			containsStringWithFragments(checklistCommands, "setup-svc-live-replay-source-checklist", lastOffsetFragment, " > ") {
			return true
		}
	}
	return false
}

func sourceChecklistHasPageCommandSummary(entry map[string]any, minPages int, firstOffset int, lastOffset int) bool {
	worklistCommands, ok := entry["pageWorklistSaveCommands"].([]any)
	if !ok || len(worklistCommands) < minPages {
		return false
	}
	checklistCommands, ok := entry["pageSourceChecklistSaveCommands"].([]any)
	if !ok || len(checklistCommands) < minPages {
		return false
	}
	firstOffsetFragment := fmt.Sprintf("--offset %d", firstOffset)
	lastOffsetFragment := fmt.Sprintf("--offset %d", lastOffset)
	return containsStringWithFragments(worklistCommands, "setup-svc-live-replay-worklist", firstOffsetFragment, " > ") &&
		containsStringWithFragments(worklistCommands, "setup-svc-live-replay-worklist", lastOffsetFragment, " > ") &&
		containsStringWithFragments(checklistCommands, "setup-svc-live-replay-source-checklist", firstOffsetFragment, " > ") &&
		containsStringWithFragments(checklistCommands, "setup-svc-live-replay-source-checklist", lastOffsetFragment, " > ")
}

func sourceChecklistHasPageSaveScript(entry map[string]any, firstOffset int, lastOffset int) bool {
	script, ok := entry["pageSaveScript"].(string)
	if !ok {
		return false
	}
	return strings.Contains(script, "#!/usr/bin/env bash") &&
		strings.Contains(script, "set -euo pipefail") &&
		strings.Contains(script, "setup-svc-live-replay-worklist") &&
		strings.Contains(script, "setup-svc-live-replay-source-checklist") &&
		strings.Contains(script, fmt.Sprintf("--offset %d", firstOffset)) &&
		strings.Contains(script, fmt.Sprintf("--offset %d", lastOffset)) &&
		strings.Contains(script, " > ")
}

func sourceChecklistHasSavePageScriptCommand(entry map[string]any, jqPath string, commandFragment string) bool {
	path, ok := entry["pageSaveScriptPath"].(string)
	if !ok || !strings.HasSuffix(path, ".sh") {
		return false
	}
	command, ok := entry["savePageSaveScriptCommand"].(string)
	if !ok {
		return false
	}
	return strings.Contains(command, commandFragment) &&
		strings.Contains(command, "jq -r '"+jqPath+"'") &&
		strings.Contains(command, " > ") &&
		strings.Contains(command, path) &&
		strings.Contains(command, "chmod +x")
}

func preflightCaptureSourcesHasSourceExecutionCommands(entry map[string]any) bool {
	packetPath, ok := entry["sourceExecutionPacketPath"].(string)
	if !ok || !strings.HasSuffix(packetPath, "source-capture-execution-packet-readiness-incomplete.json") {
		return false
	}
	savePacket, ok := entry["saveSourceExecutionPacketCommand"].(string)
	if !ok ||
		!strings.Contains(savePacket, "setup-svc-live-replay-source-execution-packet") ||
		!strings.Contains(savePacket, "--source-readiness incomplete") ||
		!strings.Contains(savePacket, packetPath) ||
		commandHasPagination(savePacket) {
		return false
	}
	batchScriptPath, ok := entry["sourceExecutionBatchScriptPath"].(string)
	if !ok || !strings.HasSuffix(batchScriptPath, "source-capture-execution-packet-readiness-incomplete.sh") {
		return false
	}
	saveBatchScript, ok := entry["saveSourceExecutionBatchScriptCommand"].(string)
	if !ok ||
		!strings.Contains(saveBatchScript, "jq -r '.batchSaveScript'") ||
		!strings.Contains(saveBatchScript, batchScriptPath) ||
		!strings.Contains(saveBatchScript, "chmod +x") ||
		commandHasPagination(saveBatchScript) {
		return false
	}
	importScriptPath, ok := entry["sourceExecutionImportScriptPath"].(string)
	if !ok || !strings.HasSuffix(importScriptPath, "source-capture-import-batches-readiness-complete.sh") {
		return false
	}
	saveImportScript, ok := entry["saveSourceExecutionImportScriptCommand"].(string)
	return ok &&
		strings.Contains(saveImportScript, "jq -r '.importBatchSaveScript'") &&
		strings.Contains(saveImportScript, importScriptPath) &&
		strings.Contains(saveImportScript, "chmod +x") &&
		!commandHasPagination(saveImportScript)
}

func commandHasPagination(command string) bool {
	return strings.Contains(command, "--offset") || strings.Contains(command, "--limit")
}

func containsSourceExecutionCommandWithPagination(items []any) bool {
	for _, item := range items {
		command, ok := item.(string)
		if !ok || !strings.Contains(command, "setup-svc-live-replay-source-execution-packet") {
			continue
		}
		if commandHasPagination(command) {
			return true
		}
	}
	return false
}

func containsStringWithFragments(items []any, fragments ...string) bool {
	for _, item := range items {
		value := item.(string)
		matched := true
		for _, fragment := range fragments {
			if !strings.Contains(value, fragment) {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

func evidenceSectionHasStatus(items []any, section string, status string) bool {
	for _, item := range items {
		sectionStatus := item.(map[string]any)
		if sectionStatus["section"] == section && sectionStatus["status"] == status {
			return true
		}
	}
	return false
}

func evidenceSectionSummaryHasCount(items []any, artifactType string, section string, total int, present int, missing int) bool {
	for _, item := range items {
		summary := item.(map[string]any)
		if summary["artifactType"] != artifactType || summary["section"] != section {
			continue
		}
		return int(summary["total"].(float64)) == total &&
			int(summary["present"].(float64)) == present &&
			int(summary["missing"].(float64)) == missing
	}
	return false
}

func evidenceSectionSummary(items []any, artifactType string, section string) map[string]any {
	for _, item := range items {
		summary := item.(map[string]any)
		if summary["artifactType"] == artifactType && summary["section"] == section {
			return summary
		}
	}
	return nil
}

func evidenceSectionQueue(items []any, artifactType string, section string) map[string]any {
	for _, item := range items {
		queue := item.(map[string]any)
		if queue["artifactType"] == artifactType && queue["section"] == section {
			return queue
		}
	}
	return nil
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func readTestObjectFile(t *testing.T, path string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestDraftUpdateRequiresPatch(t *testing.T) {
	var stdout bytes.Buffer
	err := Handle("draft-update", "msapi", []string{"draft_1"}, &stdout, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "draft-update") {
		t.Fatalf("expected draft-update usage error, got %v", err)
	}
}

func TestMetadataDomainAliases(t *testing.T) {
	cases := map[string]string{
		"object":           "objects",
		"fields":           "fields",
		"recordType":       "record-types",
		"pagelayout":       "layouts",
		"permission":       "permissions",
		"sharingRule":      "sharing-rules",
		"globalSelectList": "global-select-lists",
		"validationRule":   "validation-rules",
		"menu":             "menus",
		"buttonLink":       "buttons",
		"role":             "roles",
		"customSetting":    "custom-settings",
		"dupeCatcher":      "dupe-catchers",
		"singleSignOn":     "single-sign-ons",
		"identityProvider": "identity-providers",
		"approval":         "approval-processes",
		"view":             "object-views",
	}
	for input, want := range cases {
		if got := normalizeDomain(input); got != want {
			t.Fatalf("normalizeDomain(%q) = %q, want %q", input, got, want)
		}
		if !IsMetadataDomain(input) {
			t.Fatalf("expected %q to be recognized as a metadata domain", input)
		}
	}
	if IsMetadataDomainAction("get") {
		t.Fatal("get should remain on the legacy/module command path")
	}
	if !IsMetadataDomainAction("plan") {
		t.Fatal("plan should be routed to MetadataService")
	}
}

func TestHighCodeDomainsAreNotMetadataServiceWriteDomains(t *testing.T) {
	highCodeDomains := []string{
		"classes",
		"triggers",
		"timer",
		"schedule",
		"scheduleJob",
		"script",
		"html",
		"staticResource",
		"pagecomponent",
		"customPage",
		"sidecar",
	}
	for _, domain := range highCodeDomains {
		if IsMetadataDomain(domain) {
			t.Fatalf("%q must stay outside MetadataService domain routing", domain)
		}
		if !isHighCodeResourceDomain(domain) {
			t.Fatalf("%q should be recognized as a high-code resource boundary", domain)
		}
	}
}

func TestHighCodeMetadataServicePlanInputsAreRejectedBeforeHTTP(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		t.Fatalf("MetadataService must not be called for high-code domain %s", r.URL.Path)
	}))
	defer server.Close()
	t.Setenv("CLOUDCC_METADATA_SERVICE_URL", server.URL)

	cases := []struct {
		name   string
		action string
		args   []string
	}{
		{name: "plan shortcut", action: "plan", args: []string{"classes", `{"name":"DemoClass"}`}},
		{name: "plan raw body", action: "plan", args: []string{`{"domain":"pagecomponent","operation":"upsert","spec":{}}`}},
		{name: "validate shortcut", action: "validate", args: []string{"triggers", `{"name":"BeforeSave"}`}},
		{name: "mutate shortcut", action: "mutate", args: []string{"script", "update", `{"name":"ClientScript"}`}},
		{name: "mutate raw body", action: "mutate", args: []string{`{"domain":"html","mutation":"update","patch":{}}`}},
		{name: "draft shortcut", action: "draft-create", args: []string{"staticResource", "upsert", `{"name":"logo"}`}},
		{name: "draft raw body", action: "draft-create", args: []string{`{"domain":"customPage","operation":"upsert","draft":{}}`}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout bytes.Buffer
			err := Handle(tc.action, "msapi", tc.args, &stdout, t.TempDir())
			if err == nil {
				t.Fatal("expected high-code MetadataService boundary error")
			}
			if !strings.Contains(err.Error(), "high-code writes stay on existing CloudCC resource/API paths") {
				t.Fatalf("expected high-code boundary error, got %v", err)
			}
		})
	}
	if calls != 0 {
		t.Fatalf("MetadataService should not be called for rejected high-code domains, got %d calls", calls)
	}
}
