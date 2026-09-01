---
name: cc-customization-expert-msapi
version: 2.2.43-msapi
description: "CloudCC CRM/PaaS 实施与开发的 Go 离线技能。Strict MetadataService package: low-code operations require MetadataService."
---

# CloudCC CRM 实施专家技能 MSAPI v2.2.43-msapi

当前技能版本：`2.2.43-msapi`。分发名：`cc-customization-expert-msapi`。

## Provider 规则

- 低代码元数据统一使用稳定的 CloudCC CLI 词汇和共享能力矩阵；高代码资源继续使用 CloudCC 原有 resource/API 通道。
- 执行模式从 `CLOUDCC_EXECUTION_MODE`、当前环境 `executionMode` / `execution_mode`、包默认值依次解析；仅支持 `auto`、`msapi`、`uiapi`。
- 技能包根目录默认带 `cloudcc-cli.config.json`。首次初始化技能配置时先问用户公有云还是私有云；公有云使用默认 `metadataService.url=https://dc52.apis.cloudcc.cn/metadata`，私有云再提示用户提供 MetadataService URL 并更新该文件。
- 每次环境切换或首次写入前运行：`tools/bin/cloudcc doctor provider <projectPath>`。输出会明确所选 provider、原因、安全级别和目标 MetadataService/setup-svc 兼容性。
- `auto` 仅对已配置的 MetadataService 做 `GET /metadata/v1/capabilities` 只读探测；未配置时选择 UIAPI。已配置但不可用、认证失败或不兼容时失败关闭，绝不静默降级。
- MSAPI 原生提供服务端 plan/apply/changes/rollback；UIAPI 是直接 CloudCC UI/API 调用，不承诺服务端 ledger 或 rollback。不要把 UIAPI 补偿操作当作 MSAPI 回滚。

## 必须保留

- 先运行 `tools/bin/cloudcc --version`，并使用技能包内置 `tools/bin/cloudcc`，不在线安装 CLI，不依赖全局 Node/npm。
- 方案设计前读取 `platform/overview`、`platform/capabilityMap`、`platform/standardCapabilities`；低代码变更先读取目标模块 `devguide`。
- 初始化或治理交付目录、创建跨模块长期规范、创建或评审业务/系统流程图前读取 `methodology/projectGovernance`，并遵守项目 `AGENTS.md` 指向的项目标准；项目标准只定义方法和质量门禁，不复制 FEAT、ADR、DevOps 或任务板事实。
- 初始化或治理项目最终文档、项目专用工具、数据/部署/培训/集成包等产出物前读取 `methodology/projectOutputs`；项目根 `outputs/` 只固定 README、索引和 manifest，不固定内容模板，approved/delivered 本地交付件必须记录 SHA-256，禁止凭据和真实客户数据。
- 初始化或治理端到端测试资产、分析变更影响、建议测试范围、记录人工范围决策或测试运行证据前读取 `methodology/testGovernance`；建议保持 advisory/non-blocking，人类 decision 是范围事实源，未执行测试不得标记通过或替代 UAT。
- 用户提出业务需求、功能需求、需求设计、方案设计或高代码需求时，默认先进入需求设计/方案设计模式：先识别业务目标、对象/字段、页面、流程、权限、数据质量、集成和批处理边界，再匹配平台标准低代码能力。总体原则是优先使用平台标准元数据解决需求问题；即便用户把需求归类为“写类”“写触发器”“写定时类”等高代码需求，也必须先告知哪些低代码能力可以实现或部分实现该需求，并给出低代码优先的推荐路径。只有低代码能力无法覆盖、需要外部集成/复杂编排/动态跨对象计算/历史数据治理等场景时，才进入高代码设计和源码生成。
- 任何业务需求或高代码需求都必须先做平台标准元数据能力匹配：字段类型、对象/字段配置、页面布局、验证规则、查重过滤器、工作流/审批、共享/权限、公式/汇总、自动编号、查找筛选、相关列表等低代码能力能满足时，默认必须优先使用这些平台标准元数据实现，并走 MetadataService scan/plan/apply/rollback 或对应低代码快捷命令；不得因为用户提到“写类”“写触发器”“写定时类”就直接生成 Java 代码。若用户把可低代码实现的事项归类为高代码需求，必须明确提示“该需求优先建议用 <具体低代码能力> 实现”，再说明是否仍需要少量高代码补充。
- 全局对象字段字典必须作为元数据处置决策表处理：先分类 `创建字段`、`复用标准字段`、`复用现有自定义字段`、`不建字段`、`迁移定位键`、`源编码映射`、`仅crosswalk`、`系统字段`、`全局选项集` 或 `待确认`，再按最终设计明确、结构化列明确、英文源字段 snake_case 规范化、中文拼音兜底、扫描匹配、人工确认的顺序确定 API。`待定`、空 API、占位 API、`待确认` 行以及非字段处置行不得进入 MSAPI fields plan。
- 高代码写入（classes、triggers、timer、script、html、staticResource、pagecomponent、customPage 等）不进入 MetadataService 元数据写域，继续走 CloudCC 原资源 API。
- 接口注册器属于低代码配置元数据：create/update/delete/query 走 MetadataService，debug/logs/logDetail 保持 setup-svc 实时运行态通道。高代码调用外部 HTTP 前读取 `platform/apiRegistrar devguide`，源码只使用已调试且为 `ACTIVE` 的 `apiCode`，不得硬编码 URL 或误用软件包标识 `apiName`。从技能 `2.2.33` 开始，接口注册器运行态调试/日志和高代码远程调用调整要求 MetadataService `1.1.41` 或更高版本，并建议 setup-svc `19.7.R8` 或更高版本；setup-svc 分支版本只做提醒，不按字符串直接阻断。
- 从技能 `2.2.39` 开始，会计年度 `fiscal-years` 纳入年度和下级会计季度：年度详情返回 `fiscalQuarters[]`，年度 spec 可嵌套 `quarters[]`，也可用 `createQuarter/deleteQuarter fiscalYear` 快捷命令；区域层级 `areas` 仅对齐 setup-web 使用的 `/api/area/queryTree`、`/api/area/saveArea`、`/api/area/DeleteArea`。用户管理 CLI 改用 setup-svc `/api/usermange/*`，删除语义为停用用户。
- 从技能 `2.2.40` 开始，CloudCC `accessToken` 自动刷新如果在 `/api/cauth/token` 失败，会立即返回接口失败原因并提示检查当前环境的 `cloudcc-cli.config.json` 配置，不再继续请求到只剩通用缺 token 错误。
- 从技能 `2.2.41` 开始，`cloudcc get/getList view` 统一作为对象视图列表查询，可传对象 ID/API 名/前缀或 JSON filter；`detail/editInfo view` 才按 viewId 查详情。字段文档明确 `P`、`c`、`N`、`LT` 的 create/update/upsert 精度规则为 `length + decimalPlaces <= 18`，历史非法字段需要先修复字段定义，CLI 不自动缩短字段。
- 从技能 `2.2.42` 开始，页面布局创建可显式携带 `assignments[]` 或 `--profile/--record-type` 同步写入 `tp_sys_profile_layout`，也可用 `cloudcc assign pagelayout` 单独分配布局；未传分配参数时仍保持 setup-svc 的“只创建/复制布局，不自动分配记录类型布局”语义。
- 从技能 `2.2.43` 开始，币种管理 `currencies` 纳入 MetadataService 低代码域：支持币种列表/详情/可新增币种/高级汇率读取，固定币种新增/修改/启停/汇率维护，高级多币种开关，dated rate 新增/修改/删除，以及要求显式 `rates[]` 的公司本位币变更计划。
- 高代码发布前先读 `cloudcc doc platform/classes|triggers|timer devguide` 或 `platform/almRelease devguide`；从技能 `2.2.7` 开始，classes/triggers/timer 的 publish 建议 setup-svc `19.3.R20` 或更高版本，不要求 MetadataService 版本门槛。classes 固定执行本地编译、目标 setup-svc validate、最后 save；triggers/timer 执行目标 setup-svc validate、最后 save；并把 validate 失败详情返回调用方。
- 从技能 `2.2.38` 开始，classes/triggers/timer 创建时默认发送 setup-svc 自定义代码 `version=3`；更新时先读取目标 detail，优先沿用线上记录的 version，线上 version 为空按旧版 `2` 处理，再 validate/save，并在保存后把线上 ID/version 写回本地 `config.json`，避免旧本地配置把线上版本 3 降级。
- 能力矩阵在 `capability-matrix.json`。`adapter` 表示已有 UIAPI 适配通道；`requires_adapter` 表示该 UIAPI 操作当前必须失败关闭，不能改走 MetadataService。

## 常用命令

```bash
tools/bin/cloudcc --version
tools/bin/cloudcc doctor provider /path/to/project
tools/bin/cloudcc doctor project-governance /path/to/project
tools/bin/cloudcc init project-outputs /path/to/project demo-crm
tools/bin/cloudcc doctor project-outputs /path/to/project
tools/bin/cloudcc init test-governance /path/to/project demo-crm
tools/bin/cloudcc advise testing /path/to/project @change.json
tools/bin/cloudcc decide testing /path/to/project @decision.json
tools/bin/cloudcc record testing /path/to/project @run.json
tools/bin/cloudcc doctor test-governance /path/to/project
tools/bin/cloudcc doc platform/overview introduction
tools/bin/cloudcc doc platform/apiRegistrar devguide
tools/bin/cloudcc doc methodology/projectGovernance devguide
tools/bin/cloudcc doc methodology/projectOutputs devguide
tools/bin/cloudcc doc methodology/testGovernance devguide
tools/bin/cloudcc doc methodology/deliveryPlan devguide
tools/bin/cloudcc create project demo-cloudcc
tools/bin/cloudcc create object /path/to/project '<provider-specific object input>'
tools/bin/cloudcc plan msapi /path/to/project objects @object.json create
tools/bin/cloudcc apply msapi /path/to/project <planId>
tools/bin/cloudcc publish classes ExampleClass /path/to/project
```

MSAPI 的 `plan/apply/changes/rollback`、setup-svc parity replay、用户管理和报告/字段/会计年度/区域/币种等详细命令使用 `cloudcc --help` 与内置 `platform/*` 文档。UIAPI 模式下仅使用已适配的低代码资源命令；未适配域会返回明确错误。
