# 详情页报表分配 CLI 使用指南

本文说明如何使用 `detailPageReport` 完成查询、候选报表检查、创建、更新、删除、审阅、执行和回滚。

## 使用前检查

```text
cloudcc --version
cloudcc doctor provider <projectPath>
cloudcc domain detailPageReport
```

- MSAPI 要求 MetadataService `1.1.83` 或更高版本，写操作先生成计划，再显式 apply，可查看 changes 并 rollback。
- Lightning UIAPI 直接调用 setup-service，不提供 MetadataService ledger/rollback。
- horizontal UIAPI 尚无稳定契约，会在发网前明确拒绝；horizontal 项目请使用 MSAPI。
- `report` 管理报表定义；`detailPageReport` 只管理已有报表在详情页的展示分配。

## 命令总览

```text
cloudcc get detailPageReport <projectPath> [object] [recordType]
cloudcc detail detailPageReport <projectPath> <assignmentId>
cloudcc candidates detailPageReport <projectPath> <object> [recordType]
cloudcc create detailPageReport <projectPath> <@spec.json>
cloudcc update detailPageReport <projectPath> <assignmentId> <@spec.json>
cloudcc delete detailPageReport <projectPath> <assignmentId>
```

`object` 可传对象 ID、API 名或前缀。`recordType` 可传记录类型 ID、API 名、旧 ID，或 `main`/`master`/`default`；省略时表示主记录类型。

## 完整操作流程

### 1. 查询已有分配

```text
cloudcc get detailPageReport . account main
cloudcc detail detailPageReport . <assignmentId>
```

列表和详情返回分配 `id`、对象、记录类型及按 `sort` 排序的 `charts[]`。后续更新和删除使用真实回读的 `id`。

### 2. 查询可用报表

```text
cloudcc candidates detailPageReport . account main
```

候选结果中的 `currentRecordFilterSupported=true` 表示该报表的数据源包含目标对象，可以使用 `filterMode=currentRecord`。

### 3. 准备创建文件

新建 `detail-page-report.json`：

```json
{
  "object": "account",
  "recordType": "main",
  "charts": [
    {
      "reportId": "sales_report",
      "title": "当前客户销售趋势",
      "filterMode": "currentRecord",
      "showRefresh": true
    },
    {
      "reportId": "all_accounts",
      "title": "全部客户",
      "filterMode": "none",
      "showRefresh": false
    }
  ]
}
```

`charts[]` 必须包含 1–10 项；同一报表不能重复；标题最多 50 个字符；`filterMode` 只能是 `none` 或 `currentRecord`。

### 4. 创建、审阅并执行 MSAPI 计划

```text
cloudcc create detailPageReport . @detail-page-report.json
cloudcc apply msapi . <planId>
cloudcc operation msapi . <operationId>
cloudcc changes msapi . <operationId>
cloudcc get detailPageReport . account main
```

MSAPI 模式下，`create` 返回的 JSON 已包含待执行计划，先审阅其中的步骤和目标，再使用返回的 `planId` 执行 `apply`。`apply` 返回 `operationId`；随后用 `operation` 查看执行状态、用 `changes` 查看实际变更，并通过 get/detail 回读确认。Lightning UIAPI 模式直接执行写入，不返回可 apply 的 MetadataService 计划。

### 5. 更新完整图表集合

更新使用完整 `charts[]` 替换语义：保留项也必须继续写在文件中，数组顺序就是详情页展示顺序。

```text
cloudcc update detailPageReport . <assignmentId> @detail-page-report.json
cloudcc apply msapi . <planId>
cloudcc changes msapi . <operationId>
cloudcc detail detailPageReport . <assignmentId>
```

已有分配不能通过 update 改绑到其他对象或记录类型。

### 6. 删除与回滚

```text
cloudcc delete detailPageReport . <assignmentId>
cloudcc apply msapi . <planId>
cloudcc changes msapi . <operationId>
cloudcc rollback msapi . <operationId>
```

删除只移除详情页分配及其图表项，不删除报表定义。MSAPI 模式下先审阅 delete 返回的计划，再使用其 `planId` 执行 apply；`changes` 和 `rollback` 均使用 apply 返回的真实 `operationId`。`rollback` 不适用于 Lightning UIAPI。

## 参数和校验规则

- 同一对象 + 记录类型只能有一个分配。
- 主记录类型使用 `main`、`master`、`default` 或省略记录类型。
- `charts[]` 是完整替换集合，必须显式提供 1–10 项，且同一报表不可重复。
- `title` 最多 50 个字符。
- `filterMode=currentRecord` 时，报表数据源必须包含目标对象。
- 报表必须已存在且未删除。
- 已有分配不可通过 update 改绑到另一对象或记录类型。

## 常见问题

- “当前记录过滤要求报表数据源包含目标对象”：先运行 candidates，选择支持当前记录过滤的报表，或把 `filterMode` 改为 `none`。
- “对象 + 记录类型已存在分配”：先 get 查询并使用 update，不要再次 create。
- “报表不存在或已删除”：使用 candidates 返回的真实 `reportId`。
- “horizontal UIAPI 不支持”：为项目配置 MetadataService 并选择 MSAPI provider。
- 旧 `report` spec 的顶层 `objects[]` 不再接受；请改用独立的 `detailPageReport` 文件。

能力概览使用：

```text
cloudcc doc detailPageReport introduction
```
