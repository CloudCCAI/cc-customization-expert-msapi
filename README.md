# cc-customization-expert-msapi v2.2.44-msapi

CloudCC CRM/PaaS 离线 Go 技能，发布目标：`MSAPI`。

## Runtime

```bash
tools/bin/cloudcc --version
tools/bin/cloudcc doctor provider /path/to/project
```

Strict MetadataService package: low-code operations require MetadataService.

该包由 `cc-customization-expert-go` 的共享核心生成。低代码能力及 provider 状态见 `capability-matrix.json`；高代码资源继续复用 CloudCC 原 resource/API 通道。

项目交付目录和跨模块长期标准按 `cloudcc doc methodology/projectGovernance devguide` 治理；可用 `tools/bin/cloudcc doctor project-governance <projectPath>` 只读检查标准索引、元数据、AGENTS 读取门禁和流程图入口。通用包不内置任何客户项目标准正文。

项目最终文档、项目专用工具、数据/部署/培训/集成包按 `cloudcc doc methodology/projectOutputs devguide` 治理；`init project-outputs` 只创建根 `outputs/` 的 README、索引和 manifest，具体产出目录按项目要求动态创建，`doctor project-outputs` 只读检查路径、状态、SHA-256 和敏感内容风险。

需求、方案设计和高代码需求都先做平台标准元数据能力匹配。字段、对象、布局、验证规则、查重过滤器、工作流/审批、共享/权限、公式/汇总、自动编号、查找筛选、相关列表等低代码能力能满足时，优先用平台元数据实现；即便用户把事项归类为高代码，也应先说明可用低代码能力，再判断是否需要自定义类、触发器或定时类补充。

端到端测试按 `cloudcc doc methodology/testGovernance devguide` 采用建议层＋人工确认层：`init test-governance` 只创建缺失的中性骨架，`advise testing` 输出非阻断建议，`decide testing` 保存人工范围决定，`record testing` 保存运行 manifest，`doctor test-governance` 只读检查结构、哈希、引用和敏感文件风险。通用包不内置客户测试用例、决策、运行证据或 UAT 结论。

技能包根目录包含 `cloudcc-cli.config.json`。公有云默认使用 `https://dc52.apis.cloudcc.cn/metadata`；私有云初始化时将当前环境的 `metadataService.url` 改为用户提供的私有云 MetadataService 地址。

批量创建对象、字段和全局选项列表时，plan metadata 会返回 `batchItemResults`、`batchExecutableCount` 和 `batchPrecheckFailedCount`。调用方可以在 apply 前区分 `PLANNED`、`SKIPPED`、`FAILED_PRECHECK` 单项结果；实际 apply 仍保持 SQL 批处理和事务保护。

全局对象字段字典按元数据处置决策表治理：优先采用最终设计明确 API，其次结构化 API 列、英文源字段 snake_case 规范化，再用中文拼音兜底；迁移定位键、源编码映射、仅 crosswalk、系统字段和全局选项集必须先分流，不能进入 MSAPI fields plan。

调用方通过 `cloudcc doc platform/classes|triggers|timer devguide` 或 `cloudcc doc platform/almRelease devguide` 认识高代码发布命令；这些文档说明了 classes 本地编译、setup-svc validate、save 的顺序，以及 triggers/timer 远程 validate 后 save、失败返回和源码编码规则。从技能 `2.2.7` 开始，高代码发布建议 setup-svc `19.3.R20` 或更高版本，不要求 MetadataService 版本门槛；setup-svc 分支版本只做提醒，不按字符串直接阻断。

从技能 `2.2.38` 开始，classes/triggers/timer 创建默认按 setup-svc 新版自定义代码语义发送 `version=3`；更新会先读取目标 detail，优先沿用线上记录的 version，线上 version 为空按旧版 `2` 处理，保存后再把线上 ID/version 写回本地 `config.json`。

从技能 `2.2.39` 开始，会计年度和区域层级进入 MetadataService 低代码域：`fiscal-years` 管理年度及下级会计季度，详情返回 `fiscalQuarters[]`，年度 spec 可嵌套 `quarters[]`，并提供 `createQuarter/deleteQuarter fiscalYear` 快捷命令；`areas` 仅按 setup-web `/api/area/queryTree`、`/api/area/saveArea`、`/api/area/DeleteArea` 建模。用户管理 CLI 改用 setup-svc `/api/usermange/*`，删除/disable/deactivate 统一为停用用户。

从技能 `2.2.40` 开始，CloudCC `accessToken` 自动刷新如果在 `/api/cauth/token` 失败，会立即返回接口失败原因并提示检查当前环境的 `cloudcc-cli.config.json` 配置，不再继续请求到只剩通用缺 token 错误。

从技能 `2.2.41` 开始，`cloudcc get/getList view` 统一作为对象视图列表查询，可传对象 ID/API 名/前缀或 JSON filter；`detail/editInfo view` 才按 viewId 查详情。字段文档明确 `P`、`c`、`N`、`LT` 的 create/update/upsert 精度规则为 `length + decimalPlaces <= 18`，历史非法字段需要先修复字段定义，CLI 不自动缩短字段。

从技能 `2.2.42` 开始，页面布局创建可显式携带 `assignments[]` 或 `--profile/--record-type` 同步写入 `tp_sys_profile_layout`，也可用 `cloudcc assign pagelayout` 单独分配布局；未传分配参数时仍保持 setup-svc 的“只创建/复制布局，不自动分配记录类型布局”语义。

从技能 `2.2.43` 开始，币种管理进入 MetadataService 低代码域：`currencies` 支持币种列表、详情、可新增币种、高级汇率读取，固定币种新增/修改/启停/汇率维护，高级多币种开关，dated rate 新增/修改/删除，以及要求显式重算 `rates[]` 的公司本位币变更计划。

从技能 `2.2.44` 开始，验证规则 CLI 用户级文档按 setup-service `validateFunction` 实际函数补充运算符和函数说明，示例使用服务端实际存在的 `ISNULL`，不把 `ISBLANK` 或前端面板中未确认的 `PRECISE*` 函数作为验证规则能力承诺。

CloudCC 高代码主动调用外部 HTTP 服务时，先读取 `cloudcc doc platform/apiRegistrar devguide`。接口注册器的配置 CRUD 属于 MetadataService 域，调试和日志属于 setup-svc 实时运行态；业务源码只引用调试成功且为 `ACTIVE` 的 `apiCode`。从技能 `2.2.33` 开始，接口注册器运行态调试/日志和高代码远程调用调整要求 MetadataService `1.1.41` 或更高版本，并建议 setup-svc `19.7.R8` 或更高版本。

## Package Purity

本包是生成产物，不包含 `.git`、`.claw`、测试夹具、项目证据、凭据或本地绝对路径。请勿直接修改 `dist/`；修改源代码或模板后使用仓库发布脚本重新生成。
