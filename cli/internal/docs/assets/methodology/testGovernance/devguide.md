# CloudCC 建议式端到端测试治理指南

## 1. 基本原则

开发阶段默认采用双层机制：

1. 系统读取变更资源、项目阶段、影响目录和场景目录，生成非阻断建议。
2. 人类确认执行建议范围、缩减、扩大或跳过。
3. decision 保存 AI 建议、建议哈希、选择范围、责任角色、时间和原因。
4. 测试执行后保存 run manifest；跳过也保存 `risk_accepted` 事实。
5. 发布证据只引用选中的 decision/run，不把技术检查误写成业务 UAT。

AI 不得因为建议为 full-core 就自动执行全量测试，也不得因为人类选择 skip 就把功能标记为 passed。

## 2. 项目阶段建议

| 阶段 | 默认建议起点 | 人类可以做什么 |
|---|---|---|
| prototype / rapid-development | smoke | 跳过、冒烟或扩大 |
| feature-development | feature-closure | 选择当前功能闭环或缩减 |
| stabilization / SIT | affected-chain | 选择上下游影响链或全量核心链 |
| UAT / release-candidate / production | full-core | 业务负责人确认范围、豁免和 Go/No-Go |

资源类型会提高建议范围：权限、共享、迁移、公共组件和平台升级通常建议 full-core；触发器、类、工作流、审批和接口通常建议 affected-chain；页面布局和标签通常从 smoke 开始。项目 impact-map 可以覆盖通用起点。

## 3. 初始化

先读取：

```bash
cloudcc doc methodology/projectGovernance devguide
cloudcc doc methodology/testGovernance devguide
```

显式初始化：

```bash
cloudcc init test-governance <projectPath> <projectCode>
```

初始化器：

- 只创建缺失文件和目录，不覆盖项目已有资产。
- 不创建或修改 `.claw`。
- 不修改 `AGENTS.md`。
- 生成 `STD-TEST-001` draft；人工评审后才可改为 active，并在 AGENTS 中加入标准 ID 和稳定路径读取门禁。
- 不调用 CloudCC、MetadataService 或外部系统。

### 3.1 治理与项目文档

```text
docs/delivery/<project-code>/
├── 00-governance/standards/
│   ├── 00-standard-index.md
│   └── 10-test-governance-standard.md
├── 07-testing-cutover/
│   ├── 00-testing-cutover-index.md
│   ├── 01-uat-scenario-matrix.md
│   ├── 02-permission-acceptance-matrix.md
│   ├── 03-cutover-runbook.md
│   ├── 04-rollback-runbook.md
│   ├── 05-test-impact-matrix.md
│   ├── 06-test-data-catalog.md
│   ├── 07-environment-account-matrix.md
│   ├── 08-requirement-test-traceability.md
│   └── snapshots/
└── 08-release-evidence/
    ├── 00-release-evidence-index.md
    ├── 01-msapi-plan-apply-changes.md
    ├── 02-production-readiness-gate.md
    ├── 03-post-release-verification.md
    └── snapshots/
```

- 标准目录回答“项目长期怎样治理测试”。
- `07-testing-cutover` 回答“当前项目具体测什么、谁验收、怎样切换和回滚”。
- `08-release-evidence` 回答“本次候选发布采用了哪些证据”。
- 项目正式验收报告或客户交付包位于根 `outputs/`，只引用这里的 UAT 和 evidence，不复制真实运行证据。
- snapshots 只保存评审、签字和发布冻结版本；稳定工作文件不带日期。

### 3.2 可复用机器资产

```text
test-assets/
├── schemas/
├── catalog/
│   ├── impact-map.json
│   └── scenario-index.json
├── scenarios/
├── suites/
├── fixtures/
├── assertions/
└── automation/
```

- schemas 约束输入结构。
- catalog 把对象、字段、触发器、类、工作流、审批、接口和模块映射到场景/测试包。
- scenarios 保存稳定场景 ID 的可执行定义。
- suites 只引用场景 ID，不复制用例正文。
- fixtures 只保存生成式或批准脱敏的数据模板、初始化和清理逻辑。
- assertions 保存状态、关联、权限、审批、接口幂等、报表和读回断言。
- automation 保存 API/UI/runner，不保存某次运行日志。

### 3.3 决策和运行证据

```text
evidence/testing/
├── decisions/<changeSetId>/
│   ├── recommendation.json
│   └── human-decision.json
└── runs/<runId>/
    ├── manifest.json
    ├── summary.md
    ├── readbacks/
    ├── screenshots/
    ├── logs/
    └── defect-links.md
```

decision 和 run 目录不可覆盖；修订或重跑必须使用新的 ID。不要存储 `.env`、`.pem`、`.key`、credential 文件或 JSON 中的 password/token/secret/privateKey 等非空值。

- `evidence/testing/decisions` 是 AI 建议与人工范围选择的事实源。
- `evidence/testing/runs` 是实际执行 manifest、读回、截图、日志和缺陷引用的事实源。

## 4. 生成测试建议

变更输入契约：

```json
{
  "schemaVersion": "cloudcc-test-change/v1",
  "changeSetId": "CHG-001",
  "phase": "feature-development",
  "description": "Adjust opportunity update behavior",
  "resources": [
    {
      "kind": "trigger",
      "name": "OpportunityTrigger",
      "module": "opportunity",
      "events": ["update"],
      "shared": false
    }
  ]
}
```

执行：

```bash
cloudcc advise testing <projectPath> @change.json > recommendation.json
```

输出包含：

- `advisory: true`
- `blocking: false`
- `recommendedScope`
- `riskLevel` / `riskTags`
- `affectedModules`
- `scenarioIds` / `suiteIds`
- `estimatedMinutes`
- `catalogCoverage`
- `reasons`
- `humanDecisionRequired: true`
- `recommendationHash`

catalog 未建立或资源未匹配时，输出 `catalogCoverage: missing` 和 `catalog-gap`，仍保持非阻断；人类必须复核影响范围，不能把“没有匹配到用例”理解成“不需要测试”。

## 5. 保存人工决定

decision 输入必须内嵌完整 recommendation：

```json
{
  "schemaVersion": "cloudcc-test-decision/v1",
  "recommendation": {},
  "selectedScope": "smoke",
  "confirmedBy": "human",
  "decidedByRole": "project-manager",
  "decidedAt": "2026-01-01T08:00:00Z",
  "reason": "Feature is still changing; expand before UAT."
}
```

执行：

```bash
cloudcc decide testing <projectPath> @decision.json
```

规则：

- `selectedScope` 仅支持 skip、smoke、feature-closure、affected-chain、full-core。
- skip 或偏离推荐范围时 reason 必填。
- `confirmedBy` 必须为 human。
- CLI 重算 recommendationHash，拒绝建议漂移。
- skip 产生 `verificationState: risk_accepted`，不产生 passed。

## 6. 记录测试运行

```json
{
  "schemaVersion": "cloudcc-test-run/v1",
  "runId": "TEST-RUN-001",
  "changeSetId": "CHG-001",
  "sourceRevision": "commit-or-package-digest",
  "environment": "SIT",
  "environmentFingerprint": "environment-fingerprint",
  "testAssetRevision": "test-asset-revision",
  "status": "passed",
  "startedAt": "2026-01-01T08:10:00Z",
  "completedAt": "2026-01-01T08:20:00Z",
  "scenarioResults": [
    {
      "scenarioId": "E2E-OPP-001",
      "status": "passed",
      "evidence": "readbacks/opportunity.json"
    }
  ],
  "businessAcceptanceStatus": "pending"
}
```

执行：

```bash
cloudcc record testing <projectPath> @run.json
```

规则：

- 必须先存在同 changeSetId 的有效人工 decision。
- skip decision 只能记录 skipped/unverified，且不能包含伪造的场景执行结果。
- passed 且业务验收 pending 时状态为 `pending_uat`；不能仅凭技术测试声明 `verified`。
- 业务 accepted/rejected 必须提供 `businessAcceptanceEvidence` 引用。
- 外部 runner 可以把日志、截图、读回和缺陷链接写入 CLI 创建的 run 目录。

## 7. 只读检查

```bash
cloudcc doctor test-governance <projectPath>
```

doctor 检查：

- 固定 `07-testing-cutover`、`08-release-evidence` 和测试资产目录。
- impact-map/scenario-index schema、场景 ID 唯一性和场景文件引用。
- decision 建议哈希、人工确认和 ID 一致性。
- run manifest、decision 引用、状态和不可伪造边界。
- 敏感文件名和 JSON 敏感字段风险。

doctor 是只读结构检查。passed 不证明测试内容正确、用例已执行、CloudCC 目标租户能力成立或业务已经验收。

## 8. CloudCC 专项闭环

- 低代码：保留 scan → plan → human review → apply → changes → readback → rollback evidence。
- 高代码：保留 compile/validate → publish → source readback → business assertion；validate 成功不等于业务成功。
- 权限：目标角色验证应用、菜单、对象、字段、记录五层正反向结果。
- 集成：验证请求、响应、幂等、重试、补偿、同步状态和最终业务数据。
- 真实更新：保留业务记录 ID、版本/修订、关联请求号和写后读回；凭据和私钥不进入证据。

## 9. 项目状态和发布证据

- `.claw/test-report.md` 只保存最近验证摘要、runId、失败/未执行边界和下一步，不复制完整日志。
- `08-release-evidence` 只链接本次候选采用的 decision/run、UAT、回滚和上线后验证。
- 技术检查、SIT、UAT、生产授权和上线后业务验证必须分开报告。
