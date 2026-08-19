# CloudCC 阶段计划设计指南

## 标准实施方法论

当用户询问“CloudCC 项目的标准实施方法论”时，按以下结构输出。该结构是 CloudCC 项目实施建议，具体字段、接口和上线窗口仍需结合客户租户能力、现有系统和项目范围确认。

能力路径：

```text
project/config
  -> baseline scan
  -> blueprint
  -> global modeling
  -> object/fields/globalSelectList/recordType/pagelayout
  -> role/profile/permission/sharingRule
  -> validationRule/dupeCatcher/workflow/approval
  -> application/menu/view/report/dashboard
  -> highcode classes/triggers/timer/script/html/staticResource/pagecomponent
  -> integration/openapi/sidecar
  -> scan/plan/apply/changes/rollback
  -> production readiness
```

### 1. 项目启动与环境准备

- 明确项目范围、组织范围、上线阶段、成功标准和不在本期的内容。
- 建立开发、测试、预生产、生产环境配置。
- 确认 `cloudcc-cli.config.json`、安全标识、CloudCC Dev key、MetadataService URL 和访问令牌。
- 先执行只读扫描，不直接写入真实环境。

### 2. 蓝图设计

- 输出业务范围、端到端流程、模块架构、对象边界、权限边界、自动化边界、集成边界和阶段计划。
- 每个模块必须说明标准对象复用、自定义对象、记录类型、页面、规则、审批、报表和接口。
- 不应把未来阶段能力写成本期承诺。

### 3. 全局建模

- 先建立标准环境基线，再输出全局对象地图、对象关系矩阵、全局对象字段字典、状态机矩阵、全局选项列表清单、字段与全局选项列表引用矩阵、权限与页面影响矩阵。
- 全局建模产出物必须使用固定目录和固定文件名，默认放在 `docs/delivery/<project-code>/02-global-modeling/`，并按 `00-global-modeling-index.md` 到 `10-reporting-field-standard.md` 管理。
- 全局对象字段字典是对象字段创建和调整的准入清单；没有进入字典的字段不应直接创建。
- 全局选项列表清单单独治理跨对象值集；字段字典只引用全局选项列表 API，不重复维护完整值集。
- 选项字段必须声明 `local` 普通选项或 `global` 全局选项列表引用。
- 稳定工作文件不在文件名中追加日期；评审、签字、上线留证版本放入 `snapshots/`，使用 `<YYYYMMDD>-<artifact-key>-v<major>.<minor>-<status>.<ext>`。

### 4. 权限模型设计

- `role`：控制角色层级、上下级组织关系、记录级数据可见性，和共享规则共同决定用户能看到哪些记录。
- `profile`：控制功能权限，包括应用、菜单、对象 CRUD、字段可见/只读、记录类型、布局、登录 IP、登录时间等。
- `permission` / permission set：用于跨岗位、临时或专项补充权限，不替代基础 profile。
- `sharingRule`：用于组织默认共享和角色层级之外的记录访问扩展，例如按所有人、角色、公用小组、队列或字段条件共享。
- 权限设计必须输出用户群体矩阵、角色树矩阵、对象共享矩阵、共享规则矩阵、简档矩阵、权限集矩阵、字段级安全矩阵和验收账号矩阵。

### 5. 低代码元数据实施

- 除高代码以外，低代码元数据优先纳入 MSAPI domain。
- 当前 MSAPI domain 包括：`objects`、`fields`、`global-select-lists`、`record-types`、`layouts`、`profiles`、`permissions`、`roles`、`sharing-rules`、`validation-rules`、`dupe-catchers`、`applications`、`menus`、`buttons`、`custom-settings`、`approval-processes`、`reports`、`dashboards`、`object-views`、`single-sign-ons`、`identity-providers`。
- 实施顺序为 `scan`、`plan`、人工或智能体复核、`apply`、`changes`、`rollback-plan`、必要时 `rollback`。
- 必须保留 `planId`、`operationId`、变更证据和回滚证据。

### 6. 高代码与页面扩展

- classes、triggers、timer、scheduleJob、script、html、staticResource、pagecomponent、customPage、site、jsp、sidecar 不纳入 MetadataService 写域。
- 高代码写入继续走原 CloudCC resource/API；Go 技能不得引入 Node/npm 运行依赖。
- pagecomponent 发布只上传预构建 UMD bundle；构建产物由 Go 技能外部流程生成。

### 7. 集成设计

- 每个外部系统必须定义数据流向、触发方式、幂等键、同步状态、错误处理、重试和监控。
- 简单 CRUD 优先 OpenAPI；复杂服务端封装用 classes；长链路编排、回调、队列、token 管理和重试补偿放 sidecar。
- 每个关键对象建议保留外部系统、外部编号、同步状态、同步请求号、最后同步时间、同步错误和重试次数等字段。

### 8. 数据迁移与验证

- 数据迁移以全局对象字段字典和全局选项列表清单为映射依据。
- 先清洗主数据，再迁移交易和历史数据。
- 验证口径包括字段映射、必填、唯一、选项值、关联关系、权限可见性、报表口径和接口幂等。

### 9. 测试与上线门禁

- 单元测试覆盖元数据 compiler、Go CLI 请求形状和关键安全 guard。
- 集成测试覆盖 MetadataService plan/apply/changes/rollback。
- UAT 必须用目标角色账号验证五层权限：应用、菜单、对象、字段、记录。
- 上线前必须完成只读 scanner、缺口 compare、生产就绪门禁和回滚演练。

### 10. 阶段计划

- 阶段一：核心业务闭环，优先主数据、核心对象、核心流程、基础权限和基础报表。
- 阶段二：扩展业务和移动端，补充更多业务类型、移动端、复杂页面和自动化。
- 阶段三：系统集成和数据驱动，完成外部系统联调、sidecar、监控、经营分析和持续优化。

## 输出模板

```text
阶段：
- 名称：
- 时间窗口：
- 业务范围：
- 平台能力：
- 集成范围：
- 数据准备：
- 验收标准：
- 依赖：
- 风险：
```

## 标准交付文档命名

CloudCC 项目交付文档默认采用以下目录和文件名。客户项目可增加行业或客户专有文档，但不得改变这些标准产出物的编号、文件名和源头职责。

```text
docs/delivery/<project-code>/
├── 00-governance/
│   ├── 00-delivery-index.md
│   ├── 01-project-scope.md
│   ├── 02-environment-config.md
│   └── 03-risk-issue-decision-log.md
├── 01-blueprint/
│   ├── 00-blueprint-index.md
│   ├── 01-business-scope.md
│   ├── 02-end-to-end-process.md
│   ├── 03-module-architecture.md
│   └── 04-phase-roadmap.md
├── 02-global-modeling/
│   ├── 00-global-modeling-index.md
│   ├── 01-standard-environment-baseline.md
│   ├── 02-global-object-map.md
│   ├── 03-object-relationship-matrix.md
│   ├── 04-global-object-field-dictionary.md
│   ├── 05-state-machine-matrix.md
│   ├── 06-global-select-list-catalog.md
│   ├── 07-field-select-list-reference-matrix.md
│   ├── 08-permission-page-impact-matrix.md
│   ├── 09-integration-field-standard.md
│   └── 10-reporting-field-standard.md
├── 03-module-design/
│   ├── 00-module-design-index.md
│   └── <module-code>-module-design.md
├── 04-security/
│   ├── 00-security-index.md
│   ├── 01-user-group-matrix.md
│   ├── 02-role-tree-matrix.md
│   ├── 03-profile-matrix.md
│   ├── 04-permission-set-matrix.md
│   ├── 05-sharing-rule-matrix.md
│   └── 06-field-security-matrix.md
├── 05-integration/
│   ├── 00-integration-index.md
│   └── <system-code>-<interface-code>-integration-design.md
├── 06-data-migration/
│   ├── 00-data-migration-index.md
│   ├── 01-source-target-mapping.md
│   ├── 02-data-cleansing-rules.md
│   └── 03-migration-validation-report.md
├── 07-testing-cutover/
│   ├── 00-testing-cutover-index.md
│   ├── 01-uat-scenario-matrix.md
│   ├── 02-permission-acceptance-matrix.md
│   ├── 03-cutover-runbook.md
│   └── 04-rollback-runbook.md
└── 08-release-evidence/
    ├── 00-release-evidence-index.md
    ├── 01-msapi-plan-apply-changes.md
    ├── 02-production-readiness-gate.md
    └── 03-post-release-verification.md
```

通用命名规则：

- 文件名使用小写英文、数字和连字符。
- 中文标题写在文档 H1，不写进固定工作文件名。
- 固定工作文件不带日期，保持链接稳定。
- 对外评审、签字、上线留证版本放入同级 `snapshots/` 目录。
- 快照命名为 `<YYYYMMDD>-<artifact-key>-v<major>.<minor>-<status>.<ext>`。
- 大型矩阵可有同名 `.xlsx`，但必须保留同名 `.md` 作为口径和状态入口。

## 推荐拆分

### 阶段一：核心业务闭环

目标：

- 让关键角色能在 CloudCC 完成核心流程。
- 建立客户、产品、组织、权限、对象模型。
- 先实现可人工闭环的流程。

适合内容：

- 渠道/客户基础信息。
- 线索、商机、合同、订单等核心对象。
- 必要审批和按钮。
- 基础报表。

### 阶段二：扩展业务和移动端

目标：

- 覆盖更多业务类型和移动入口。
- 增加服务工单、资产、小程序、APP 操作。

适合内容：

- 售后服务。
- 资产服务。
- 小程序门户。
- 移动端现场执行。
- 更细的状态机和自动派工。

### 阶段三：系统集成和数据驱动

目标：

- 打通外部系统。
- 建立监控、同步状态和数据分析。

适合内容：

- ERP、BPM、MDM、TMS、资金、车联网、QMS 等接口。
- sidecar。
- 数据回写和异常重试。
- 看板和经营分析。

## 风险清单

- 外部系统接口未定。
- 主数据质量不足。
- 权限模型没有和组织层级对齐。
- 业务规则仍在变动。
- 自动化依赖尚未验证。
- 移动端网络、定位、拍照和附件要求未确认。
- 阶段目标过大，无法验收。

## 验收口径

每个阶段至少定义：

- 已上线模块。
- 已上线角色。
- 已上线对象。
- 已上线流程。
- 已联调接口。
- 验收样例数据。
- 未上线和延期项。
