package testgovernance

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
	Warnings    []string `json:"warnings"`
	NextActions []string `json:"nextActions"`
}

func Init(projectPath string, projectCode string) (InitResult, error) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return InitResult{}, err
	}
	result := InitResult{Status: "initialized", ProjectPath: absPath, ProjectCode: strings.TrimSpace(projectCode)}
	info, err := os.Stat(absPath)
	if err != nil || !info.IsDir() {
		return result, fmt.Errorf("test-governance init requires an existing project directory: %s", absPath)
	}
	if !validID(projectCode) {
		return result, fmt.Errorf("projectCode must use letters, digits, dot, underscore, or hyphen")
	}

	deliveryRoot := filepath.ToSlash(filepath.Join("docs", "delivery", projectCode))
	files := scaffoldFiles(projectCode)
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
	for _, relDir := range scaffoldDirs(projectCode) {
		if err := os.MkdirAll(filepath.Join(absPath, filepath.FromSlash(relDir)), 0o755); err != nil {
			return result, err
		}
	}
	indexPath := filepath.Join(absPath, filepath.FromSlash(deliveryRoot+"/00-governance/standards/00-standard-index.md"))
	if _, err := os.Stat(indexPath); err == nil && !containsFile(result.Created, deliveryRoot+"/00-governance/standards/00-standard-index.md") {
		result.Warnings = append(result.Warnings, "existing standard index was not modified; register STD-TEST-001 manually before activation")
	}
	result.NextActions = []string{
		"Review the draft STD-TEST-001 testing standard and project-specific matrices.",
		"Activate the standard only after human review, then add its stable path to AGENTS.md.",
		"Populate test-assets/catalog before relying on advisory scenario selection.",
		"Run cloudcc doctor test-governance <projectPath> after project-specific content is added.",
	}
	sort.Strings(result.Created)
	sort.Strings(result.Skipped)
	return result, nil
}

func scaffoldDirs(projectCode string) []string {
	root := filepath.ToSlash(filepath.Join("docs", "delivery", projectCode))
	return []string{
		root + "/00-governance/standards/archive",
		root + "/07-testing-cutover/snapshots",
		root + "/08-release-evidence/snapshots",
		"test-assets/schemas",
		"test-assets/catalog",
		"test-assets/scenarios",
		"test-assets/suites",
		"test-assets/fixtures/templates",
		"test-assets/fixtures/seeds",
		"test-assets/fixtures/cleanup",
		"test-assets/assertions",
		"test-assets/automation",
		"evidence/testing/decisions",
		"evidence/testing/runs",
	}
}

func scaffoldFiles(projectCode string) map[string]string {
	deliveryRoot := filepath.ToSlash(filepath.Join("docs", "delivery", projectCode))
	impactMap, _ := json.MarshalIndent(ImpactMap{SchemaVersion: ImpactMapSchemaVersion, Resources: []ImpactEntry{}}, "", "  ")
	scenarioIndex, _ := json.MarshalIndent(ScenarioIndex{SchemaVersion: ScenarioIndexSchemaVersion, Scenarios: []ScenarioEntry{}}, "", "  ")
	return map[string]string{
		deliveryRoot + "/00-governance/standards/00-standard-index.md":           standardIndexTemplate(),
		deliveryRoot + "/00-governance/standards/10-test-governance-standard.md": testingStandardTemplate(),
		deliveryRoot + "/07-testing-cutover/00-testing-cutover-index.md":         testingIndexTemplate(),
		deliveryRoot + "/07-testing-cutover/01-uat-scenario-matrix.md":           matrixTemplate("UAT Scenario Matrix", "业务端到端场景、角色、前置条件、主流程、反向流程、预期状态和场景资产路径。"),
		deliveryRoot + "/07-testing-cutover/02-permission-acceptance-matrix.md":  matrixTemplate("Permission Acceptance Matrix", "应用、菜单、对象、字段、记录五层权限的正向和反向验收。"),
		deliveryRoot + "/07-testing-cutover/03-cutover-runbook.md":               runbookTemplate("Cutover Runbook", "上线顺序、负责人、时间窗、检查点和停止条件。"),
		deliveryRoot + "/07-testing-cutover/04-rollback-runbook.md":              runbookTemplate("Rollback Runbook", "MSAPI 回滚、高代码旧版恢复、数据恢复和回滚后验证。"),
		deliveryRoot + "/07-testing-cutover/05-test-impact-matrix.md":            matrixTemplate("Test Impact Matrix", "资源到模块、场景、角色、自动化、接口和报表的影响关系；机器版本位于 test-assets/catalog。"),
		deliveryRoot + "/07-testing-cutover/06-test-data-catalog.md":             matrixTemplate("Test Data Catalog", "生成式或脱敏测试数据语义、创建、唯一标识、保留和清理规则；不保存凭据或真实客户数据。"),
		deliveryRoot + "/07-testing-cutover/07-environment-account-matrix.md":    matrixTemplate("Environment And Account Matrix", "环境能力、测试角色、简档、权限集和限制；不保存口令、Token 或密钥。"),
		deliveryRoot + "/07-testing-cutover/08-requirement-test-traceability.md": matrixTemplate("Requirement Test Traceability", "需求、设计、资源、场景、运行、缺陷、UAT 和发布版本的追踪关系。"),
		deliveryRoot + "/08-release-evidence/00-release-evidence-index.md":       releaseTemplate("Release Evidence Index", "只引用本次发布采用的 decision/run 和发布证据，不复制完整运行证据。"),
		deliveryRoot + "/08-release-evidence/01-msapi-plan-apply-changes.md":     releaseTemplate("MSAPI Plan Apply Changes", "记录本次发布相关 planId、operationId、changes、readback 和 rollback 证据引用。"),
		deliveryRoot + "/08-release-evidence/02-production-readiness-gate.md":    releaseTemplate("Production Readiness Gate", "汇总测试范围决策、缺测风险、UAT、回滚、版本指纹和 Go/No-Go。"),
		deliveryRoot + "/08-release-evidence/03-post-release-verification.md":    releaseTemplate("Post Release Verification", "记录上线后技术检查、受控业务读回、未完成验收和后续动作。"),
		"test-assets/README.md":                    testAssetsReadme(),
		"test-assets/catalog/impact-map.json":      string(append(impactMap, '\n')),
		"test-assets/catalog/scenario-index.json":  string(append(scenarioIndex, '\n')),
		"test-assets/schemas/change.schema.json":   schemaTemplate(ChangeSchemaVersion, []string{"schemaVersion", "changeSetId", "phase", "resources"}),
		"test-assets/schemas/decision.schema.json": schemaTemplate(DecisionSchemaVersion, []string{"schemaVersion", "recommendation", "selectedScope", "confirmedBy", "decidedByRole", "decidedAt"}),
		"test-assets/schemas/run.schema.json":      schemaTemplate(RunSchemaVersion, []string{"schemaVersion", "runId", "changeSetId", "sourceRevision", "environment", "status", "startedAt", "completedAt", "scenarioResults"}),
	}
}

func containsFile(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func standardIndexTemplate() string {
	return `---
kind: project-standard-index
title: Project Standards
status: active
---

# Project Standards

| Standard ID | Standard | Status | Owner role | Read trigger |
|---|---|---|---|---|
| STD-TEST-001 | [Advisory Test Governance](10-test-governance-standard.md) | draft | qa-agent | Before changing project-wide testing rules or approving a release test scope |
`
}

func testingStandardTemplate() string {
	return `---
kind: project-standard
standard_id: STD-TEST-001
title: Advisory End-to-End Test Governance
status: draft
version: 1.0
owner_role: qa-agent
effective_date: pending-human-review
review_trigger: Project phase, risk appetite, release process, or test tooling changes
---

# Advisory End-to-End Test Governance

## Principle

The system detects changes and recommends a test scope. A human confirms skip, smoke, feature-closure, affected-chain, or full-core. Recommendations do not block development and do not replace UAT or Go/No-Go.

## Truthful states

- verified: confirmed scope executed and passed.
- partially_verified: only part of the confirmed scope passed.
- unverified: no current test evidence.
- risk_accepted: a human explicitly chose skip or accepted a reduced scope.
- pending_uat: technical checks passed while target-role business acceptance is pending.

## Activation

Keep this standard in draft until reviewed. When activated, update the standard index and add an AGENTS.md read gate containing STD-TEST-001 and this stable path. Project-specific scenarios, roles, data, decisions, and evidence belong outside this standard.
`
}

func testingIndexTemplate() string {
	return `# Testing And Cutover Index

This is the stable project entry for test design and cutover preparation. Link current scenario, permission, impact, data, environment, traceability, decision, run, defect, UAT, and release evidence; do not copy full execution logs here.

## Current status

- Phase: pending
- Target release: pending
- Latest decision: pending
- Latest run: pending
- Business UAT: pending
- Open risks: pending
`
}

func matrixTemplate(title string, purpose string) string {
	return fmt.Sprintf("# %s\n\n%s\n\n| ID | Scope | Owner role | Status | Source or evidence |\n|---|---|---|---|---|\n", title, purpose)
}

func runbookTemplate(title string, purpose string) string {
	return fmt.Sprintf("# %s\n\n%s\n\n## Preconditions\n\n- Pending\n\n## Steps\n\n1. Pending\n\n## Stop or rollback conditions\n\n- Pending\n", title, purpose)
}

func releaseTemplate(title string, purpose string) string {
	return fmt.Sprintf("# %s\n\n%s\n\n## Candidate identity\n\n- Source revision: pending\n- Package or image digest: pending\n- Environment fingerprint: pending\n\n## Evidence references\n\n- Pending\n", title, purpose)
}

func testAssetsReadme() string {
	return `# Test Assets

Reusable machine assets only. Store schemas, impact catalogs, scenario definitions, suites, generated or approved sanitized fixtures, reusable assertions, and automation runners here.

Do not store customer credentials, reusable secrets, raw customer data, one-off run logs, screenshots, decisions, or release evidence. Decisions and immutable run evidence belong under evidence/testing.
`
}

func schemaTemplate(schemaVersion string, required []string) string {
	payload := map[string]any{
		"$schema":  "https://json-schema.org/draft/2020-12/schema",
		"title":    schemaVersion,
		"type":     "object",
		"required": required,
		"properties": map[string]any{
			"schemaVersion": map[string]any{"const": schemaVersion},
		},
		"additionalProperties": true,
	}
	b, _ := json.MarshalIndent(payload, "", "  ")
	return string(append(b, '\n'))
}
