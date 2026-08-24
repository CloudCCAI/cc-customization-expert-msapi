---
name: cc-customization-expert-msapi
version: 2.2.18-msapi
description: CloudCC CRM/PaaS 实施与开发的 Go 离线技能。Strict MetadataService package: low-code operations require MetadataService.
---

# CloudCC CRM 实施专家技能 MSAPI v2.2.18-msapi

当前技能版本：`2.2.18-msapi`。分发名：`cc-customization-expert-msapi`。

## Provider 规则

- 低代码元数据统一使用稳定的 CloudCC CLI 词汇和共享能力矩阵；高代码资源继续使用 CloudCC 原有 resource/API 通道。
- 执行模式从 `CLOUDCC_EXECUTION_MODE`、当前环境 `executionMode` / `execution_mode`、包默认值依次解析；仅支持 `auto`、`msapi`、`uiapi`。
- 技能包根目录默认带 `cloudcc-cli.config.json`。首次初始化技能配置时先问用户公有云还是私有云；公有云使用默认 `metadataService.url=https://dc52.apis.cloudcc.cn/metadata`，私有云再提示用户提供 MetadataService URL 并更新该文件。
- 每次环境切换或首次写入前运行：`tools/bin/cloudcc doctor provider <projectPath>`。输出会明确所选 provider、原因和安全级别。
- `auto` 仅对已配置的 MetadataService 做 `GET /metadata/v1/capabilities` 只读探测；未配置时选择 UIAPI。已配置但不可用、认证失败或不兼容时失败关闭，绝不静默降级。
- MSAPI 原生提供服务端 plan/apply/changes/rollback；UIAPI 是直接 CloudCC UI/API 调用，不承诺服务端 ledger 或 rollback。不要把 UIAPI 补偿操作当作 MSAPI 回滚。

## 必须保留

- 先运行 `tools/bin/cloudcc --version`，并使用技能包内置 `tools/bin/cloudcc`，不在线安装 CLI，不依赖全局 Node/npm。
- 方案设计前读取 `platform/overview`、`platform/capabilityMap`、`platform/standardCapabilities`；低代码变更先读取目标模块 `devguide`。
- 全局对象字段字典必须作为元数据处置决策表处理：先分类 `创建字段`、`复用标准字段`、`复用现有自定义字段`、`不建字段`、`迁移定位键`、`源编码映射`、`仅crosswalk`、`系统字段`、`全局选项集` 或 `待确认`，再按最终设计明确、结构化列明确、英文源字段 snake_case 规范化、中文拼音兜底、扫描匹配、人工确认的顺序确定 API。`待定`、空 API、占位 API、`待确认` 行以及非字段处置行不得进入 MSAPI fields plan。
- 高代码写入（classes、triggers、timer、script、html、staticResource、pagecomponent、customPage 等）不进入 MetadataService 元数据写域，继续走 CloudCC 原资源 API。
- 高代码发布前先读 `cloudcc doc platform/classes|triggers|timer devguide` 或 `platform/almRelease devguide`；从技能 `2.2.7` 开始，classes/triggers/timer 的 publish 要求目标 setup-svc `19.3.R20` 或更高版本。classes 固定执行本地编译、目标 setup-svc validate、最后 save；triggers/timer 执行目标 setup-svc validate、最后 save；并把 validate 失败详情返回调用方。
- 能力矩阵在 `capability-matrix.json`。`adapter` 表示已有 UIAPI 适配通道；`requires_adapter` 表示该 UIAPI 操作当前必须失败关闭，不能改走 MetadataService。

## 常用命令

```bash
tools/bin/cloudcc --version
tools/bin/cloudcc doctor provider /path/to/project
tools/bin/cloudcc doc platform/overview introduction
tools/bin/cloudcc doc methodology/deliveryPlan devguide
tools/bin/cloudcc create project demo-cloudcc
tools/bin/cloudcc create object /path/to/project '<provider-specific object input>'
tools/bin/cloudcc plan msapi /path/to/project objects @object.json create
tools/bin/cloudcc apply msapi /path/to/project <planId>
tools/bin/cloudcc publish classes ExampleClass /path/to/project
```

MSAPI 的 `plan/apply/changes/rollback`、setup-svc parity replay 和报告/字段等详细命令使用 `cloudcc --help` 与内置 `platform/*` 文档。UIAPI 模式下仅使用已适配的低代码资源命令；未适配域会返回明确错误。

