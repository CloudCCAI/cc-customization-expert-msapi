# 字段追踪 CLI 使用指南

本文说明如何使用 `fieldTracking` 查询候选字段、审阅当前配置、完整替换或清空追踪字段，并在 MSAPI 模式下审计和回滚变更。

## 使用前检查

```text
cloudcc --version
cloudcc doctor provider <projectPath>
cloudcc domain fieldTracking
```

- MSAPI 要求 MetadataService `1.1.85` 或更高版本。写命令只生成计划，必须显式 apply 才会生效。
- Lightning UIAPI 直接提交到平台配置接口，不生成 MetadataService ledger 或 rollback。
- horizontal UIAPI 当前不可用，会在发网前明确拒绝；horizontal 项目请使用 MSAPI。

## 命令总览

```text
cloudcc get fieldTracking <projectPath> <object>
cloudcc candidates fieldTracking <projectPath> <object>
cloudcc update fieldTracking <projectPath> <@tracking.json>
cloudcc clear fieldTracking <projectPath> <object>
```

`object` 可传对象 ID、API 名或对象前缀。文档和命令也接受 `fieldTrack`、`field-tracking` 别名，示例统一使用 `fieldTracking`。

## 完整操作流程

### 1. 查询当前配置

```text
cloudcc get fieldTracking . account
```

重点检查以下返回字段：

- `selectedCount`：当前已选择的用户可管理字段数，不包含系统管理字段。
- `limit`：兼容字段；MSAPI 返回 `null`，因为不执行数量限制。
- `quotaAuthoritative`：兼容字段；MSAPI 返回 `false`。
- `quotaValidation`：MSAPI 返回 `not-enforced`，表示没有读取或校验 `FIELD_TRACK / RES010`。
- `fields[].selected`：字段当前是否启用追踪。
- `fields[].systemManaged`：是否由平台管理；为 `true` 时不要写入 `trackedFields`。
- `fields[].trackingMode`：`old-and-new` 记录新旧值，`change-only` 只记录字段发生过变化。

### 2. 查询可配置候选字段

```text
cloudcc candidates fieldTracking . account
```

只使用候选结果中返回的字段 ID 或 `apiName`。候选结果已经排除不可配置字段；不要从其他对象复制字段 ID，也不要自行构造字段 ID。

### 3. 准备完整替换文件

新建 `field-tracking.json`：

```json
{
  "object": "account",
  "trackedFields": [
    "industry",
    "description"
  ]
}
```

`trackedFields` 必须显式提供，并表示变更后的完整用户字段集合。它不是“在当前配置上追加这些字段”。如果当前启用的用户字段需要继续保留，也必须继续写在数组中。

该文件只改变已有字段的追踪开关，不会创建字段。MSAPI 不限制数组中的候选字段数量，也不校验公有云 `FIELD_TRACK / RES010` 配额。

### 4. 生成、审阅并执行 MSAPI 计划

```text
cloudcc update fieldTracking . @field-tracking.json
cloudcc apply msapi . <planId>
cloudcc operation msapi . <operationId>
cloudcc changes msapi . <operationId>
cloudcc get fieldTracking . account
```

MSAPI 模式下，`update` 返回的 JSON 是待执行计划。先审阅对象、变更步骤、完整替换提示和返回的 `planId`，确认无误后再 apply。`apply` 返回 `operationId`；随后用 `operation` 查看状态、用 `changes` 查看实际变更，并再次 get 回读确认。

Lightning UIAPI 模式下，`update` 会直接提交，不返回可 apply 的 MetadataService `planId`。

### 5. 更新字段集合

修改 `field-tracking.json` 后重新执行：

```text
cloudcc update fieldTracking . @field-tracking.json
cloudcc apply msapi . <planId>
cloudcc changes msapi . <operationId>
cloudcc get fieldTracking . account
```

如果计划生成后对象字段、当前选择、系统管理字段或候选资格发生变化，apply 会拒绝旧计划。此时重新运行 get 和 candidates，更新文件并生成新计划，不要反复执行旧 `planId`。

### 6. 清空用户可管理字段

```text
cloudcc clear fieldTracking . account
cloudcc apply msapi . <planId>
cloudcc changes msapi . <operationId>
cloudcc get fieldTracking . account
```

也可以提交 `"trackedFields": []`。两种方式都会保留平台管理的系统字段，不会删除历史记录数据。

### 7. 回滚 MSAPI 操作

```text
cloudcc rollback-plan msapi . <operationId>
cloudcc rollback msapi . <operationId>
cloudcc get fieldTracking . account
```

先审阅 rollback-plan，再执行 rollback，并通过 get 回读。Lightning UIAPI 没有 MetadataService `operationId`，因此不适用此回滚流程。

## 参数和安全规则

- `trackedFields` 只能是数组；更新时不可省略。
- 字段选择器支持字段 ID 或 API 名，但必须唯一命中当前对象的候选字段。
- 数组中不可重复选择同一字段，也不可包含其他对象的字段。
- `systemManaged=true` 的字段由平台保留，禁止写入 `trackedFields`。
- MSAPI 不限制追踪字段数量，也不读取或校验公有云 `FIELD_TRACK / RES010`；Lightning UIAPI 仍由 setup-service 决定是否接受请求。
- 长文本、富文本和图片等字段可能返回 `trackingMode=change-only`；这表示只记录变化事件，不提供旧值和新值对比。
- `clear` 只关闭用户可管理字段；它不是删除历史记录命令。

## 常见问题

- “字段不是候选字段”：重新运行 candidates，使用当前对象真实返回的字段 ID 或 API 名。
- “系统管理字段不能提交”：从 `trackedFields` 删除 `systemManaged=true` 的字段；平台会自行保留它。
- “计划已发生漂移”：配置或候选资格在 plan 后发生变化，重新查询并生成新计划。
- Lightning UIAPI 返回额度错误：这是 setup-service 的平台侧行为；若该项目允许使用 MSAPI，可切换 Provider 后重新生成计划，不能复用 UIAPI 的失败请求作为已生效依据。
- “horizontal UIAPI 不支持”：为项目配置 MetadataService 并选择 MSAPI Provider。
- apply 后结果与预期不一致：使用 operation 和 changes 检查真实执行结果，再通过 get 回读；不要只根据 update 的计划内容判断已生效。

能力概览使用：

```text
cloudcc doc fieldTracking introduction
```
