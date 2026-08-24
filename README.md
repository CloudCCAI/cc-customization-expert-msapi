# cc-customization-expert-msapi v2.2.18-msapi

CloudCC CRM/PaaS 离线 Go 技能，发布目标：`MSAPI`。

## Runtime

```bash
tools/bin/cloudcc --version
tools/bin/cloudcc doctor provider /path/to/project
```

Strict MetadataService package: low-code operations require MetadataService.

该包由 `cc-customization-expert-go` 的共享核心生成。低代码能力及 provider 状态见 `capability-matrix.json`；高代码资源继续复用 CloudCC 原 resource/API 通道。

技能包根目录包含 `cloudcc-cli.config.json`。公有云默认使用 `https://dc52.apis.cloudcc.cn/metadata`；私有云初始化时将当前环境的 `metadataService.url` 改为用户提供的私有云 MetadataService 地址。

批量创建对象、字段和全局选项列表时，plan metadata 会返回 `batchItemResults`、`batchExecutableCount` 和 `batchPrecheckFailedCount`。调用方可以在 apply 前区分 `PLANNED`、`SKIPPED`、`FAILED_PRECHECK` 单项结果；实际 apply 仍保持 SQL 批处理和事务保护。

全局对象字段字典按元数据处置决策表治理：优先采用最终设计明确 API，其次结构化 API 列、英文源字段 snake_case 规范化，再用中文拼音兜底；迁移定位键、源编码映射、仅 crosswalk、系统字段和全局选项集必须先分流，不能进入 MSAPI fields plan。

调用方通过 `cloudcc doc platform/classes|triggers|timer devguide` 或 `cloudcc doc platform/almRelease devguide` 认识高代码发布命令；这些文档说明了 classes 本地编译、setup-svc validate、save 的顺序，以及 triggers/timer 远程 validate 后 save、失败返回和源码编码规则。从技能 `2.2.7` 开始，高代码发布要求目标 setup-svc `19.3.R20` 或更高版本。

## Package Purity

本包是生成产物，不包含 `.git`、`.claw`、测试夹具、项目证据、凭据或本地绝对路径。请勿直接修改 `dist/`；修改源代码或模板后使用仓库发布脚本重新生成。

