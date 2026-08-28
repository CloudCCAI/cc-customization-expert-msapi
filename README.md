# cc-customization-expert-msapi v2.2.37-msapi

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

CloudCC 高代码主动调用外部 HTTP 服务时，先读取 `cloudcc doc platform/apiRegistrar devguide`。接口注册器的配置 CRUD 属于 MetadataService 域，调试和日志属于 setup-svc 实时运行态；业务源码只引用调试成功且为 `ACTIVE` 的 `apiCode`。从技能 `2.2.33` 开始，接口注册器运行态调试/日志和高代码远程调用调整要求 MetadataService `1.1.41` 或更高版本，并建议 setup-svc `19.7.R8` 或更高版本。

## Package Purity

本包是生成产物，不包含 `.git`、`.claw`、测试夹具、项目证据、凭据或本地绝对路径。请勿直接修改 `dist/`；修改源代码或模板后使用仓库发布脚本重新生成。
