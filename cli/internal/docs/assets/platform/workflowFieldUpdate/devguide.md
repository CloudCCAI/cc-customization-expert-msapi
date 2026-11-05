# 工作流字段更新开发指南

本文是 `workflowFieldUpdate` 的完整 CLI 用户指南。对应 setup-web 路由 `/#/settings/action/workflow-field-updates`，MSAPI Domain 为 `workflow-field-updates`，最低 MetadataService 版本为 `1.1.105`。

MSAPI 只通过 MetadataService 直连元数据数据库，不调用 setup-service。`create`、`update`、`delete` 只生成计划，不会立即修改环境；必须显式执行 `apply msapi`。

## 1. 命令总览

| 目的 | 命令 |
| --- | --- |
| 查看能力 | `cloudcc capabilities workflowFieldUpdate <projectPath>` |
| 列表/搜索 | `cloudcc get workflowFieldUpdate <projectPath> [filter]` |
| 查看详情 | `cloudcc detail workflowFieldUpdate <projectPath> <id-or-apiName>` |
| 获取对象、字段、值模式候选 | `cloudcc candidates workflowFieldUpdate <projectPath> [object]` |
| 查看工作流引用 | `cloudcc impact workflowFieldUpdate <projectPath> <id-or-apiName>` |
| 创建计划 | `cloudcc create workflowFieldUpdate <projectPath> <@spec.json>` |
| 更新计划 | `cloudcc update workflowFieldUpdate <projectPath> <selector> <@patch.json>` |
| 删除计划 | `cloudcc delete workflowFieldUpdate <projectPath> <selector>` |
| 应用计划 | `cloudcc apply msapi <projectPath> <planId>` |
| 查询执行 | `cloudcc operation msapi <projectPath> <operationId>` |
| 查看变更 | `cloudcc changes msapi <projectPath> <operationId>` |
| 回滚计划 | `cloudcc rollback-plan msapi <projectPath> <operationId>` |

`selector` 可使用动作 ID 或唯一 API 名。短横线别名 `workflow-field-update` 也可用，但推荐文档中的 `workflowFieldUpdate`。

## 2. 首次使用的完整流程

### 2.1 检查连接和服务版本

```bash
cloudcc get config ./project
cloudcc doctor provider ./project
cloudcc capabilities workflowFieldUpdate ./project
```

活动环境需要配置 `metadataService.url`，且服务版本不低于 `1.1.105`。

### 2.2 查询候选字段

```bash
cloudcc candidates workflowFieldUpdate ./project Account
```

返回结果包含：

- `objects[]`：对象 `id`、`apiName`、`label`；
- `fields[]`：属于目标对象的字段 `id`、`apiName`、`label`、`type`；
- `excludedFields[]`：公式、累计汇总、自动编号、主详、多选及系统审计字段等不可更新候选和排除原因；
- `valueModes`：全部可用模式的并集；
- `valueModesByFieldType`：按字段类型划分的真实可用模式，提交前必须以它为准。

请只使用 `fields[]` 中的候选，不要从 `excludedFields[]` 取值，也不要猜测数据库 ID。提交字段 API 名时，MetadataService 会归一为目标环境的真实字段 ID；歧义或跨对象字段会在计划阶段失败。

### 2.3 编写创建文件

`field-update.json`：

```json
{
  "apiName": "afud_mark_contacted",
  "targetObjectId": "Account",
  "targetObject": "客户",
  "name": "标记已联系",
  "description": "工作流自动更新客户状态",
  "field": "field_status",
  "value": {
    "mode": "literal",
    "value": "CONTACTED"
  }
}
```

生成计划：

```bash
cloudcc create workflowFieldUpdate ./project @field-update.json
```

返回的 `planId` 不是执行结果。审阅计划中的：

- Domain 是否为 `workflow-field-updates`；
- 目标表是否为 `tp_sys_actions_fieldupdate`；
- `targetObjectId`、`updateField`、`operate`、`fieldValue` 是否正确；
- `setupServiceRequired` 是否为 `false`；
- 是否存在 warnings。

### 2.4 应用并回读

```bash
cloudcc apply msapi ./project <planId>
cloudcc operation msapi ./project <operationId>
cloudcc detail workflowFieldUpdate ./project afud_mark_contacted
```

只有 apply 成功且 detail 回读与预期一致，才算创建完成。

## 3. 字段说明

| 字段 | 创建 | 更新 | 说明 |
| --- | --- | --- | --- |
| `apiName` | 必填 | 可选 | 唯一 API 名；必须以 `afud_` 开头，最长 80，只允许字母、数字、下划线 |
| `targetObjectId` | 必填 | 可选 | 对象 ID、API 名或可唯一解析名称；服务端归一为对象 ID |
| `targetObject` | 必填 | 可选 | 对象显示名 |
| `name` | 必填 | 可选 | 动作显示名；修改后同步已有绑定关系的显示名 |
| `description` | 可选 | 可选 | 说明；创建省略时保存空字符串 |
| `field` | 必填 | 可选 | 待更新字段 ID 或 API 名，必须属于目标对象且出现在 `candidates.fields[]` |
| `value` | 必填 | 可选 | 值定义对象，见下一节 |
| `selector` | 禁止 | CLI 自动加入 | 更新/删除的现有动作选择器 |

未知字段、服务端管理字段和空必填值会在计划阶段被拒绝。

## 4. 值模式与字段类型矩阵

setup-web 会按字段类型显示不同控件，CLI 同样严格校验：

| `fields[].type` | 允许模式 | 落库语义 |
| --- | --- | --- |
| `L` / `R`（选项类） | `literal`、`previous`、`next` | 固定选项 `T`、上一项 `P`、下一项 `N` |
| `Y`（lookup） | `literal` | `value` 必须是目标环境回读的记录 ID，`OPERATE` 为空 |
| `B`（布尔） | `literal` | `value` 只能是 `true` / `false`，`OPERATE` 为空 |
| `SCORE`（评分） | `literal` | `value` 为 1..100 的整数，`OPERATE` 为空 |
| 其他可更新类型 | `clear`、`formula` | 清空 `X` 或公式 `F` |

不在矩阵中的组合在计划阶段失败。例如普通文本字段不能用 `literal`，应使用平台公式表达一个常量；布尔字段不能使用 `clear`；`previous/next` 不能用于非选项字段。

### 4.1 固定值 `literal`

```json
"value": {"mode":"literal","value":"CONTACTED"}
```

只适用于 `L/R/Y/B/SCORE`。选项字段提交目标环境真实选项键；lookup 提交真实记录 ID；布尔提交 `true/false`；评分提交 1..100。CLI 不把显示名转换成存储值。

### 4.2 清空 `clear`

```json
"value": {"mode":"clear"}
```

只适用于矩阵中的“其他可更新类型”。不提交 `value.value`，计划会将动作值设为空。

### 4.3 公式 `formula`

```json
"value": {"mode":"formula","value":"amount__c * 1.13"}
```

只适用于矩阵中的“其他可更新类型”。公式必须是非空字符串，按平台字段更新公式表达式保存。MetadataService 不调用 setup-service 的公式校验接口；创建前必须在目标环境用标准公式编辑器验证字段 API 名、类型和函数，不要提交 SQL。

### 4.4 上一个值 `previous`

```json
"value": {"mode":"previous"}
```

只适用于 `L/R` 选项类字段，使用平台的“上一个值”语义，不保存 `value.value`。

### 4.5 下一个值 `next`

```json
"value": {"mode":"next"}
```

只适用于 `L/R` 选项类字段，使用平台的“下一个值”语义，不保存 `value.value`。

## 5. 查询、更新和引用分析

列表支持模糊过滤：

```bash
cloudcc get workflowFieldUpdate ./project 已联系
cloudcc detail workflowFieldUpdate ./project afud_mark_contacted
```

部分更新文件 `field-update-patch.json`：

```json
{
  "name": "标记客户已联系",
  "description": "由客户状态工作流执行",
  "value": {"mode":"literal","value":"CONTACTED"}
}
```

```bash
cloudcc update workflowFieldUpdate ./project afud_mark_contacted @field-update-patch.json
cloudcc apply msapi ./project <planId>
cloudcc detail workflowFieldUpdate ./project afud_mark_contacted
```

更新采用部分更新语义，未提交字段保持不变。如果动作已有工作流引用，不能直接改变 `targetObjectId`；必须先解绑全部关系。未被引用时若改变 `targetObjectId`，必须同时提交 `targetObject`、`field` 和 `value`，并按新字段类型重新校验，避免跨对象保留旧字段。

查看引用：

```bash
cloudcc impact workflowFieldUpdate ./project afud_mark_contacted
```

返回 `relationId`、`relatedId`、`triggerType` 和 `actionType`。`relatedId` 可能指向工作流本身、时间触发器或周期触发器。

## 6. 绑定到工作流

字段更新定义不自行保存工作流关系。即时绑定由 `workflow` Domain 完成：

```json
{
  "id": "acd_account_status",
  "scope": {"type":"immediate"},
  "action": {
    "domain": "workflow-field-updates",
    "ref": "afud_mark_contacted"
  }
}
```

```bash
cloudcc bindAction workflow ./project @bind-field-update.json
cloudcc apply msapi ./project <planId>
cloudcc detail workflow ./project acd_account_status
```

时间触发器不允许绑定已有动作，但可用 `createAction workflow` 创建并绑定新的字段更新。周期触发器不支持字段更新，无论已有还是新建都会被拒绝。

## 7. 安全删除

```bash
cloudcc impact workflowFieldUpdate ./project afud_mark_contacted
cloudcc delete workflowFieldUpdate ./project afud_mark_contacted
```

只要存在引用，删除计划就会返回 `workflow_action_delete_referenced`。CLI 不会自动解绑，也不会级联删除工作流。正确顺序是：

1. 用 `impact` 找到 `relationId`；
2. 用 `cloudcc unbindAction workflow ...` 生成解绑计划并 apply；
3. 再次执行 `impact`，确认 `count=0`；
4. 生成删除计划、审阅并 apply；
5. detail 应返回 not found。

## 8. 常见错误

| 错误码 | 原因与处理 |
| --- | --- |
| `workflow_action_object_invalid` | 对象无法唯一解析；先运行 `candidates` |
| `workflow_action_spec_invalid` | 必填字段、字段归属、API 名前缀或值模式错误 |
| `workflow_action_api_name_duplicate` | API 名已存在；改名或执行 update |
| `workflow_action_target_change_referenced` | 被引用动作试图换对象；先解绑 |
| `workflow_action_delete_referenced` | 删除前仍有工作流引用 |
| `workflow_action_managed` | 平台托管动作只读 |
| `workflow_recurring_field_update_unsupported` | 周期工作流不能使用字段更新 |

## 9. 上线检查清单

- 已用 `candidates` 从目标环境选择对象和字段；
- API 名以 `afud_` 开头且唯一；
- 值模式符合 `valueModesByFieldType`，固定值使用目标环境真实选项键/记录 ID/数据格式；
- 公式已在目标环境验证；
- 已审阅 plan，不含 setup-service 调用；
- apply 后已 detail 回读；
- 绑定后已读取 workflow 详情确认作用域；
- 删除前 `impact` 为零。
