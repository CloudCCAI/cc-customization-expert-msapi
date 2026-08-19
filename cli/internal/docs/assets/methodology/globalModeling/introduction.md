# CloudCC 全局对象字段模型治理

## 定位

本模块用于指导 AI Agent 在 CloudCC PaaS 低代码加高代码实施中做全局对象字段建模。它不是单个对象或单个模块的设计说明，而是跨应用、跨模块统一规划对象、字段、关系、状态、选项列表、权限、页面、集成字段和报表口径。

事实依据：

- CloudCC 平台支持对象、字段、记录类型、全局选项列表、页面布局、权限、角色、共享规则、验证规则、触发器、类、OpenAPI、pagecomponent 等能力。
- 当前技能已有 `platform/dataModeling`、`platform/object`、`platform/fields`、`platform/globalSelectList`、`platform/recordType`、`platform/pagelayout` 和 `methodology/moduleDesign`。
- 全局选项列表的功能定义、入口、字段依赖性、已删除值集、CLI 创建/查询/详情/删除能力，以 `platform/globalSelectList introduction` 和 `platform/globalSelectList devguide` 为准。
- 当前项目约定：前端页面组件在 `frontend/`，后端 class、trigger、schedule 在 `backend/`，外挂中间程序在 `sidecar/`。

方法论说明：

- 下文的命名规范、字段字典和治理模板属于 CloudCC 项目实施建议，应以客户项目命名规范和平台实际限制为准。
- 使用标准初始化的 CloudCC 环境作为基线时，应先确认标准对象、标准字段、标准选项列表和权限结构，再决定是否扩展自定义对象字段。
- 不应把安全标识、Dev Key、Open Secret Key、Token 或生产地址写入模型文档、仓库或技能资料。

## 为什么需要全局建模

业务应用如果只按页面或模块局部建模，容易出现：

- 同一业务实体在不同模块重复建对象。
- 同一含义字段在不同对象上命名不一致。
- 业务状态、审批状态、同步状态混在一个字段里。
- 选项值散落在多个对象，后期无法统一维护。
- 集成字段没有统一命名，接口排障困难。
- 报表口径依赖临时计算，跨模块指标不一致。

全局对象字段模型治理的目标是让 CloudCC 应用具备统一的数据骨架。

## 核心产物

一个完整的全局建模方案应至少包含：

```text
1. 全局对象地图
2. 对象关系矩阵
3. 全局对象字段字典
4. 状态机矩阵
5. 全局选项列表清单
6. 命名规范
7. 权限和共享影响
8. 页面布局影响
9. 集成字段规范
10. 报表口径字段
11. 数据导入和迁移策略
12. 待确认事项
```

## 对象分层

建议把对象先分层，再进入单对象字段设计：

```text
主数据对象：客户、联系人、产品、资产、组织、服务商、服务网点
过程对象：线索、商机、服务请求、工单、审批申请、变更申请
交易对象：合同、订单、交货、开票、收款、索赔、结算
明细对象：订单明细、费用明细、配件明细、执行记录、审批明细
日志对象：接口日志、同步日志、操作历史、异常记录、回调记录
配置对象：规则、策略、映射、字典、计费标准、工单生成配置
```

## 字段处置与默认命名原则

当前技能推荐的默认命名规范：

- 对象标签和字段标签默认使用中文，直接表达业务含义。
- 字段字典不是简单的字段名翻译表，而是元数据处置决策表。每一行必须先判断字段处置：`创建字段`、`复用标准字段`、`复用现有自定义字段`、`不建字段`、`迁移定位键`、`源编码映射`、`仅crosswalk`、`系统字段`、`全局选项集` 或 `待确认`。
- 字段 API 优先采用最终设计、字段说明或映射表中已经明确的 API；其次采用结构化列中的明确 API；再对清晰英文业务字段按 snake_case 规范化；只有没有明确英文/API 设计时，才使用中文拼音方式兜底。
- 两个字的名称使用全拼。
- 三个字及以上的名称使用每个字拼音首字母。
- API 名尽量短、稳定、可读。
- 不添加无业务意义的后缀，如 `_custom_c_field`、`_field`、`_obj`。
- 只有在避免冲突或表达明确分组时才添加必要限定词。
- `Status_ID`、`Company ID`、`Legacy_ID`、`Source_Record_ID` 等源系统技术字段不能仅因存在英文名称就自动建字段；应先判断是否仅用于迁移定位、源编码映射或 crosswalk。
- `待定` 不是合格的字段 API 状态。确实缺少证据时使用 `待确认` 并写明原因；`待确认` 和任何占位 API 都不得进入 MSAPI plan。

示例：

| 标签 | 推荐 API | 不推荐 |
|---|---|---|
| 目标金额 | `mbje` | `mbje_custom_c_field` |
| 客户 | `kehu` | `kh_obj` |
| 服务工单 | `fwgd` | `fwgd_custom_object` |
| 同步状态 | `tbzt` | `sync_status_custom_field` |
| 外部编号 | `wbbh` | `external_id_custom_c_field` |
| Deposit Amount | `deposit_amount` | `depositamount` |
| CRF No | `crf_no` | `crf_no_custom_field` |

## 全局选项列表原则

平台功能事实：

- 全局选项列表是跨对象共享的值集机制。
- 定义一次后，多个对象的选项列表字段可复用同一套值集。
- 修改全局选项列表后，所有引用它的字段同步更新。
- 平台提供字段依赖性入口，可查看哪些对象字段引用了该全局选项列表。
- 地址字段相关的省州、县区、国家、市是典型内置全局选项列表。

适合做全局选项列表的内容：

- 国家、省、市、地区等地域属性。
- 行业、客户类型、证件类型等全局主数据属性。
- 通用业务类型、优先级、风险等级、重要程度等通用属性。
- 跨对象复用的状态、阶段、分类。
- 需要与外部系统做编码映射的选项。

不适合做全局选项列表的内容：

- 只在单一对象内使用，且不会复用的临时选项。
- 受某个局部流程强约束的页面展示值。
- 需要频繁按记录动态变化的数据。
- 应该建成独立对象维护的主数据，如产品、服务网点、工程师。

字段实现提示：

- 真实系统中创建对象字段必须以全局对象字段字典为依据。
- 全局对象字段字典不直接包含完整全局选项列表值集；选项字段必须声明 `local` 或 `global` 值集来源。全局选项集和值集本体进入 `06-global-select-list-catalog.md`，字段字典只保留引用关系。
- 创建或调整选项列表字段前，先读取 `platform/fields devguide`。
- 字段使用全局选项列表时，需要关注 `useGlobalSelect` 和 `globalSelectId`。
- 全局选项列表自身的名称也遵循全局 API 命名规范，例如 `客户等级` 推荐 `khdj`。如果客户项目已有明确英文 API 命名规范，以项目规范为准。

## 相关文档

```bash
cloudcc doc platform/dataModeling devguide
cloudcc doc platform/object devguide
cloudcc doc platform/fields devguide
cloudcc doc platform/globalSelectList introduction
cloudcc doc platform/globalSelectList devguide
cloudcc doc platform/recordType devguide
cloudcc doc platform/pagelayout devguide
cloudcc doc methodology/moduleDesign devguide
cloudcc doc methodology/integrationDesign devguide
```
