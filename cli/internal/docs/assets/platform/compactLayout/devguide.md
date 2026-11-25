# 关键信息布局使用指南

使用项目当前环境，先运行 `cloudcc doctor provider .`。查询目标对象能力、字段候选和当前配置后，再提交变更。

```text
cloudcc capabilities compactLayout <projectPath> <object>
cloudcc get compactLayout <projectPath> <object>
cloudcc detail compactLayout <projectPath> <object> <layout>
cloudcc candidates compactLayout <projectPath> <object> <fields|recordTypes|layouts>
cloudcc assignments compactLayout <projectPath> <object>
cloudcc impact compactLayout <projectPath> <object> <layout> delete
cloudcc <create|update|assign|delete> compactLayout <projectPath> <specJson|@file>
```

`object` 接受对象 ID、API 名或前缀。布局选择器接受目标对象内的真实 ID 或 API 名。歧义时报错。字段与记录类型可用回读的 ID 或 API 名。优先使用候选返回的 ID，尤其在 Lightning 接口未提供字段 API 名时。候选 `eligible=false` 不可用于新配置。

## 新建与更新

新建需提供对象、API 名、标签和有序字段。API 名和标签分别在对象内唯一。新建完成后再按需分配。新建参数如下。

```json
{"object":"Account","apiName":"SALES_HIGHLIGHTS","label":"销售关键信息","fields":["Name","OwnerId"]}
```

更新要求 `object`、`selector` 和至少一个变更项。提供 `fields` 时完整替换并按数组顺序保存。省略保留，空数组和 null 无效。必须为 1–10 个不同的可选字段。系统默认布局可更新标签和字段，保留系统名称与身份。Lightning 更新如需保留不再有效的旧字段，应重新选择一套有效字段再提交。

Lightning 保存接口无法可靠处理名称或标签中的半角单引号、反斜线和 NUL 字符。需要保存这些文本时使用 MSAPI。

```json
{"object":"Account","selector":"SALES_HIGHLIGHTS","label":"销售摘要","fields":["OwnerId","Name"]}
```

## 分配

```json
{"object":"Account","mainLayout":"SALES_HIGHLIGHTS","recordTypeAssignments":[{"recordType":"ENTERPRISE","layout":"SALES_HIGHLIGHTS"}]}
```

MSAPI 分配至少包含一个变更项。省略一项则保留，`mainLayout: null` 清除主要布局标记，`recordTypeAssignments: []` 清除全部显式分配。非空映射数组完整替换现有显式关系。每个记录类型只能出现一次。

Lightning UIAPI 要求非空主要布局以及全部实际记录类型的映射，无记录类型时使用空数组。分配读取的 `explicitAssignmentsKnown=false` 表示 setup 只能返回页面投影。写后 `setupProjection` 表示投影已验证，`unchangedProjection` 表示提交前投影已一致。两者均无法独立证明显式关系落库。需要精确配置或清除时使用 MSAPI。

## 删除与执行

先读取删除影响。系统默认、当前主要或仍被引用的布局不可删除。先调整分配，再创建删除计划。

```json
{"object":"Account","selector":"SALES_HIGHLIGHTS"}
```

MSAPI 快捷写动作生成计划。审阅变更后执行并确认实际结果。通用命令也支持同一份 JSON。

```bash
cloudcc plan msapi . compact-layouts @create.json create
cloudcc apply msapi . <planId>
cloudcc changes msapi . <operationId>
cloudcc detail compactLayout . Account SALES_HIGHLIGHTS
cloudcc rollback-plan msapi . <operationId>
cloudcc rollback msapi . <operationId>
```

状态漂移需要依据最新读回重新规划。Lightning UIAPI 直接提交并回读，`write_outcome_unknown` 时先查当前状态，再决定如何继续。MSAPI 账本和回滚仅适用于 MSAPI 操作。

## 参数逐项说明

写命令使用 UTF-8 JSON 文件，`@create.json` 指向当前工作目录中的文件。项目路径 `.` 指向包含 `cloudcc-cli.config.json` 的项目。示例中的字段和记录类型名称须替换成目标环境候选实际返回的值。

| 参数 | 使用动作 | 类型与要求 |
|---|---|---|
| `object` | 所有写动作 | 必填字符串，目标对象 ID、API 名或前缀 |
| `selector` | update、delete | 必填字符串，目标对象内布局 ID 或 API 名 |
| `apiName` | create、update | create 必填，update 可选，去除首尾空白后 1–255 字符 |
| `label` | create、update | create 必填，update 可选，去除首尾空白后 1–255 字符 |
| `fields` | create、update | create 必填，有序字符串数组，1–10 项，通过 ID/API 名解析后仍须唯一 |
| `mainLayout` | assign | 布局 ID/API 名，MSAPI 允许 null，UIAPI 要求非空 |
| `recordTypeAssignments` | assign | 对象数组，每项仅含 `recordType` 和 `layout` 两个必填字符串 |

`apiName` 和 `label` 分别在同一对象内唯一。选择器先精确匹配 ID，再匹配 API 名，匹配多个目标会失败。显示标签用于展示，引用配置时使用 ID 或 API 名。未知属性会被拒绝。根 ID、字段关系 ID、记录类型关系 ID、序号、审计信息和系统默认标记由服务端维护。

| 属性 | 省略 | null | 空数组 | 非空值 |
|---|---|---|---|---|
| update.fields | 保留原集合 | 拒绝 | 拒绝 | 完整替换，数组顺序即显示顺序 |
| assign.mainLayout，MSAPI | 保留主要标记 | 清除主要标记 | 类型错误 | 设置唯一主要布局 |
| assign.recordTypeAssignments，MSAPI | 保留显式分配 | 拒绝 | 清除全部显式分配 | 完整替换，未包含的旧分配删除 |

MSAPI 的纯名称或标签更新可保留历史失效字段。显式提交 `fields` 时要求整套字段通过当前候选校验。候选会排除平台不支持的字段类型、已删除或停用字段、子字段及特定系统字段。应根据候选的排除原因调整选择。

## 读取结果怎么用

`get` 返回 `layouts[]` 和 `count`。每个布局的 `id` 可直接用于后续详情或更新。`isSystemDefault` 表示平台系统默认布局，`isMain` 表示当前主要布局。普通布局刚创建时两个值均为 false。

`detail` 返回 `layout`，包括根信息和 `fields[]`。MSAPI 字段项包含 `id/apiName/label/seq/eligible/exclusionReason`，`seq` 从 0 开始。Lightning 字段项保留 setup 返回的 ID、标签和序号。MSAPI 的 `revision` 为不透明状态指纹，由计划自动记录。无需将它复制进写入 JSON。

`assignments` 区分实际存储与运行时回退。例如 MSAPI 返回如下内容。

```json
{
  "mainLayoutId": "layout-sales",
  "systemDefaultLayoutId": "layout-system",
  "explicitAssignmentsKnown": true,
  "recordTypeAssignments": [],
  "effectiveAssignments": [
    {"recordTypeId":"record-enterprise","layoutId":"layout-sales","source":"main"}
  ]
}
```

这里记录类型没有显式映射，但会使用主要布局。`source` 可为 `recordType`、`main`、`systemDefault` 或 `none`。无记录类型的对象可使用 `mainLayoutId` 和 `systemDefaultLayoutId` 判断对象回退。没有配置时 ID 为 null，读取不会创建布局。

Lightning 分配响应包含 `projection: "setup"`、`explicitAssignmentsKnown: false`、`recordTypeAssignments: null`，有效项的 `source` 为 `unknown`。不要将这种回读当作显式关系清单。MSAPI 的 `diagnostics` 出现重复主要布局、重复记录类型分配或无效引用时，应先处理配置异常。

## 从创建到生效的完整流程

先查看项目当前执行渠道，再盘点布局和候选。

```bash
cloudcc doctor provider .
cloudcc capabilities compactLayout . Account
cloudcc get compactLayout . Account
cloudcc candidates compactLayout . Account fields
cloudcc candidates compactLayout . Account recordTypes
cloudcc assignments compactLayout . Account
```

将新建示例保存成 `create.json`，字段使用候选实际返回的 ID。MSAPI 执行如下命令。每次使用该次返回的真实 planId 和 operationId。

```bash
cloudcc create compactLayout . @create.json
cloudcc apply msapi . <createPlanId>
cloudcc operation msapi . <createOperationId>
cloudcc detail compactLayout . Account SALES_HIGHLIGHTS
```

将分配示例保存成 `assign.json`。MSAPI 可只提交要显式指定的记录类型，须把希望保留的其他显式分配也放入数组。UIAPI 必须包含全部记录类型。

```bash
cloudcc assign compactLayout . @assign.json
cloudcc apply msapi . <assignPlanId>
cloudcc assignments compactLayout . Account
```

Lightning UIAPI 的 create、update、assign 会直接写入并回读，命令成功后查看结果即可。不要将 UIAPI 返回的布局 ID 用作 MSAPI planId。

修改字段顺序时，先读取详情，把希望保留的全部字段按新顺序写入 `update.json`，再执行 `cloudcc update compactLayout . @update.json`。MSAPI 继续 apply，UIAPI 直接完成写入与回读。

## 撤销分配和删除的完整流程

以下文件 `unassign.json` 会清除对象主要标记和全部显式分配。若只需解除一个目标布局，请先读取全部分配，构造保留其他关系的新数组，并根据业务需要指定其他主要布局。

```json
{"object":"Account","mainLayout":null,"recordTypeAssignments":[]}
```

```bash
cloudcc assign compactLayout . @unassign.json
cloudcc apply msapi . <unassignPlanId>
cloudcc impact compactLayout . Account SALES_HIGHLIGHTS delete
cloudcc delete compactLayout . @delete.json
cloudcc apply msapi . <deletePlanId>
cloudcc get compactLayout . Account
```

影响结果的 `blocked` 为 true 时，查看 `blockers` 和 `recordTypeIds`。系统默认布局始终受保护，当前主要或仍被记录类型引用的布局需先解除引用。plan 后引用有变化时 apply 会拒绝，需重读后重新规划。

## 常见失败与处理

| 结果或错误 | 含义与处理 |
|---|---|
| `compact_layout_no_changes` | MSAPI 请求没有产生变化，检查现有配置即可 |
| `compact_layout_duplicate_name` | 对象内名称或标签重复，复用现有布局或使用另一个值 |
| `compact_layout_field_ineligible` | 字段缺失、跨对象或被排除，重查字段候选 |
| `compact_layout_selector_ambiguous` | API 名匹配多个目标，使用回读的真实 ID |
| `compact_layout_invalid_references` | 当前聚合有重复或失效关系，读取 diagnostics 并治理后重试 |
| `compact_layout_state_changed` | 计划后的配置已变化，重新读取并创建计划 |
| `compact_layout_candidate_changed` | 计划引用的字段或记录类型已失效，重查候选并重新规划 |
| `compact_layout_delete_blocked` | 删除引用或系统保护阻断，检查 impact |
| `compact_layout_readback_mismatch` | MSAPI 事务回读不匹配，事务已回滚，检查 operation 和服务端日志 |
| `write_outcome_unknown` | UIAPI 提交后结果不确定，按原 ID/API 名查询并比较，确认后再决定是否重试 |
| 动作不可用或 schema unavailable | 检查所选 provider、服务版本、capabilities 和目标环境元数据结构 |

MSAPI rollback 会检查原操作写后的聚合指纹。后续修改造成漂移时会拒绝覆盖当前配置。涉及多个已执行操作时，按相反顺序回滚，并在每次回滚后回读确认。角色或字段权限会影响业务页面实际可见字段，布局配置成功后仍应使用目标用户核对记录详情页面。
