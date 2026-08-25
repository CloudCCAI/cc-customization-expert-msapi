package docs

import (
	"strings"
	"testing"
)

func TestReadEmbeddedDoc(t *testing.T) {
	content, err := Read("platform/object", "introduction")
	if err != nil {
		t.Fatal(err)
	}
	if content == "" {
		t.Fatal("object introduction doc is empty")
	}
}

func TestConfigDefaultsToDevGuide(t *testing.T) {
	content, err := Read("platform/config", "")
	if err != nil {
		t.Fatal(err)
	}
	if content == "" {
		t.Fatal("config devguide doc is empty")
	}
}

func TestPageComponentDocAliasesPluginDoc(t *testing.T) {
	content, err := Read("platform/pagecomponent", "devguide")
	if err != nil {
		t.Fatal(err)
	}
	if !containsAll(content, "pagecomponent", "cloudcc create pagecomponent") {
		t.Fatalf("pagecomponent doc did not expose canonical naming:\n%s", content[:min(len(content), 300)])
	}
}

func TestPluginDocIsNotPublic(t *testing.T) {
	if _, err := Read("plugin", "devguide"); err == nil {
		t.Fatal("plugin doc should not be exposed as a public module")
	}
	if _, err := Read("platform/plugin", "devguide"); err == nil {
		t.Fatal("platform/plugin doc should not be exposed as a public module")
	}
}

func TestFlatDocModulesAreNotPublic(t *testing.T) {
	if _, err := Read("object", "introduction"); err == nil {
		t.Fatal("flat object doc should not be exposed after two-level docs layout")
	}
}

func TestPlatformKnowledgeDocsAreEmbedded(t *testing.T) {
	for _, module := range []string{
		"platform/overview",
		"platform/capabilityMap",
		"platform/standardCapabilities",
		"platform/security",
		"platform/automation",
		"platform/dataModeling",
		"platform/integrationArchitecture",
		"platform/integrationPatterns",
		"platform/lowcodeHighcode",
		"platform/mobileCapabilities",
		"platform/almRelease",
	} {
		content, err := Read(module, "introduction")
		if err != nil {
			t.Fatalf("%s introduction should be embedded: %v", module, err)
		}
		if !strings.Contains(content, "CloudCC") {
			t.Fatalf("%s introduction should mention CloudCC", module)
		}
		devguide, err := Read(module, "devguide")
		if err != nil {
			t.Fatalf("%s devguide should be embedded: %v", module, err)
		}
		if devguide == "" {
			t.Fatalf("%s devguide is empty", module)
		}
	}
}

func TestHighCodePublishValidationGateDocs(t *testing.T) {
	for _, tc := range []struct {
		module string
		want   []string
	}{
		{
			module: "platform/classes",
			want: []string{
				"cloudcc publish classes <name> [projectPath] [--validation-evidence <file>]",
				"本地编译验证",
				"POST /api/ccfag/validate",
				"POST /api/ccfag/save",
				"19.3.R20",
				"服务端实际读取并编译的是 `source`",
				"%2B",
			},
		},
		{
			module: "platform/triggers",
			want: []string{
				"cloudcc publish trigger <objectApi/TriggerName> [projectPath]",
				"POST /api/trigger/validate",
				"POST /api/triggerSetup/saveTrigger",
				"19.3.R20",
				"`apiname`",
				"`triggerTime`",
				"`version`",
				"直接传原始 `triggerSource`",
			},
		},
		{
			module: "platform/timer",
			want: []string{
				"cloudcc publish timer <name> [projectPath]",
				"POST /api/ccPeak/validate",
				"POST /api/ccPeak/save",
				"19.3.R20",
				"服务端实际读取并编译的是 `source`",
				"%2B",
			},
		},
		{
			module: "platform/almRelease",
			want: []string{
				"类、触发器和定时类不是直接 save",
				"目标 setup-svc 最低版本要求为 `19.3.R20`",
				"triggers 和 timer 不做本地包装编译",
				"data.errors",
				"data.warnings",
				"trigger 的 validate 会读取 `triggerSource`、`apiname`、`triggerTime`、`version`",
				"URLDecoder-compatible",
			},
		},
	} {
		content, err := Read(tc.module, "devguide")
		if err != nil {
			t.Fatalf("%s devguide should be embedded: %v", tc.module, err)
		}
		if !containsAll(content, tc.want...) {
			t.Fatalf("%s devguide should document high-code publish validation gate", tc.module)
		}
	}
}

func TestMethodologyAndPlaybookDocsAreEmbedded(t *testing.T) {
	for _, module := range []string{
		"methodology/projectGovernance",
		"methodology/projectOutputs",
		"methodology/testGovernance",
		"methodology/blueprint",
		"methodology/globalModeling",
		"methodology/moduleDesign",
		"methodology/integrationDesign",
		"methodology/deliveryPlan",
		"playbooks/manufacturingCrm",
		"playbooks/serviceWorkOrder",
		"playbooks/miniProgramPortal",
	} {
		content, err := Read(module, "introduction")
		if err != nil {
			t.Fatalf("%s introduction should be embedded: %v", module, err)
		}
		if !strings.Contains(content, "CloudCC") {
			t.Fatalf("%s introduction should mention CloudCC", module)
		}
		devguide, err := Read(module, "devguide")
		if err != nil {
			t.Fatalf("%s devguide should be embedded: %v", module, err)
		}
		if devguide == "" {
			t.Fatalf("%s devguide is empty", module)
		}
	}
}

func TestProjectOutputsDocumentsDynamicDocumentsAndTools(t *testing.T) {
	content, err := Read("methodology/projectOutputs", "devguide")
	if err != nil {
		t.Fatal(err)
	}
	if !containsAll(
		content,
		"cloudcc init project-outputs <projectPath> <projectCode>",
		"cloudcc doctor project-outputs <projectPath>",
		"cloudcc-project-outputs/v1",
		"output-manifest.json",
		"document",
		"tool",
		"data-package",
		"deployment-package",
		"training-package",
		"integration-package",
		"SHA-256",
		"不预建 documents、tools、operations、migration、training 等目录",
	) {
		t.Fatalf("project outputs doc should define a dynamic document/tool delivery boundary and machine manifest")
	}
}

func TestTestGovernanceDocumentsAdvisoryHumanConfirmedLifecycle(t *testing.T) {
	content, err := Read("methodology/testGovernance", "devguide")
	if err != nil {
		t.Fatal(err)
	}
	if !containsAll(
		content,
		"cloudcc init test-governance <projectPath> <projectCode>",
		"cloudcc advise testing <projectPath> @change.json",
		"cloudcc decide testing <projectPath> @decision.json",
		"cloudcc record testing <projectPath> @run.json",
		"cloudcc doctor test-governance <projectPath>",
		"advisory: true",
		"blocking: false",
		"risk_accepted",
		"pending_uat",
		"evidence/testing/decisions",
		"evidence/testing/runs",
	) {
		t.Fatalf("test governance doc should define advisory recommendations, human decisions, run evidence, and truthful acceptance states")
	}
}

func TestProjectGovernanceSeparatesStandardsFromProjectFacts(t *testing.T) {
	content, err := Read("methodology/projectGovernance", "devguide")
	if err != nil {
		t.Fatal(err)
	}
	if !containsAll(
		content,
		"00-governance/standards",
		"kind: project-standard",
		"standard_id: STD-001",
		"`AGENTS.md` 只写短门禁",
		"标准不得独立维护",
		"01-blueprint/processes",
		"cloudcc doctor project-governance <projectPath>",
	) {
		t.Fatalf("project governance doc should define standard registry, read gates, fact separation, process artifacts, and read-only validation")
	}
}

func TestGlobalModelingDocIncludesDeliveryArtifactNaming(t *testing.T) {
	content, err := Read("methodology/globalModeling", "devguide")
	if err != nil {
		t.Fatal(err)
	}
	if !containsAll(
		content,
		"02-global-modeling",
		"04-global-object-field-dictionary.md",
		"06-global-select-list-catalog.md",
		"<YYYYMMDD>-<artifact-key>-v<major>.<minor>-<status>.<ext>",
	) {
		t.Fatalf("global modeling doc should include fixed delivery artifact naming standard")
	}
}

func TestGlobalModelingRequiresStandardCapabilityCatalog(t *testing.T) {
	content, err := Read("methodology/globalModeling", "devguide")
	if err != nil {
		t.Fatal(err)
	}
	if !containsAll(
		content,
		"cloudcc scan msapi <projectPath> standard-catalog",
		"商务云",
		"CPQ",
		"客户小组",
		"不要在未确认标准能力前直接创建自定义对象和字段",
	) {
		t.Fatalf("global modeling doc should require standard capability catalog scan")
	}
}

func TestGlobalModelingRequiresFieldDispositionGovernance(t *testing.T) {
	content, err := Read("methodology/globalModeling", "devguide")
	if err != nil {
		t.Fatal(err)
	}
	if !containsAll(
		content,
		"元数据处置决策表",
		"字段处置",
		"API 来源",
		"英文源字段规范化",
		"仅crosswalk",
		"MSAPI plan 前置门禁",
		"API 名=待定",
	) {
		t.Fatalf("global modeling doc should require field disposition and API source governance")
	}
}

func TestStandardCapabilitiesDocIncludesCloudApplications(t *testing.T) {
	content, err := Read("platform/standardCapabilities", "introduction")
	if err != nil {
		t.Fatal(err)
	}
	if !containsAll(
		content,
		"商务云",
		"CPQ",
		"现场服务云",
		"客户服务云",
		"项目云",
		"伙伴云",
		"利润云",
		"客户小组",
		"联系人角色",
		"业务机会产品",
		"业务机会小组",
		"联系方式",
		"字段级扫描要求",
		"动态发现机制",
		"TABLE_TYPE = '2'",
	) {
		t.Fatalf("standard capabilities doc should include CloudCC standard application capability catalog")
	}
}

func TestStandardCapabilitiesDocRequiresRelationshipAndLineItemObjects(t *testing.T) {
	content, err := Read("platform/standardCapabilities", "devguide")
	if err != nil {
		t.Fatal(err)
	}
	if !containsAll(
		content,
		"ContactRole",
		"OpportunityProduct",
		"OpportunityTeam",
		"联系方式",
		"TABLE_TYPE = '2'",
		"字段级",
		"只扫描主对象或菜单对象",
		"动态发现",
	) {
		t.Fatalf("standard capabilities devguide should require relationship and line-item standard objects")
	}
}

func TestModulesListsTwoLevelDocs(t *testing.T) {
	modules, err := Modules()
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(modules, "\n")
	for _, module := range []string{
		"platform/object",
		"methodology/blueprint",
		"methodology/globalModeling",
		"playbooks/serviceWorkOrder",
	} {
		if !strings.Contains(joined, module) {
			t.Fatalf("Modules() should include %s; got:\n%s", module, joined)
		}
	}
}

func containsAll(content string, values ...string) bool {
	for _, value := range values {
		if !strings.Contains(content, value) {
			return false
		}
	}
	return true
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
