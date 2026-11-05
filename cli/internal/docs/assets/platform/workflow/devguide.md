# 工作流开发指南

`workflow`（别名 `workflowRule`）对应 setup-web `/#/settings/workflowRule/workflow-rule`，MSAPI Domain 为 `workflows`，最低 MetadataService `1.1.105`。本 Domain 管规则、条件、启停、动作关系、时间触发器和周期触发器。

CLI 写操作不调用 setup-service。所有写命令先生成计划，必须审阅后显式 apply。

## 1. 命令总览

### 规则

```bash
cloudcc get workflow <projectPath> [filter]
cloudcc detail workflow <projectPath> <workflowId>
cloudcc create workflow <projectPath> <@workflow.json>
cloudcc update workflow <projectPath> <workflowId> <@patch.json>
cloudcc enable workflow <projectPath> <workflowId>
cloudcc disable workflow <projectPath> <workflowId>
cloudcc delete workflow <projectPath> <workflowId>
```

### 动作关系

```bash
cloudcc bindAction workflow <projectPath> <@bind.json>
cloudcc createAction workflow <projectPath> <@create-and-bind.json>
cloudcc unbindAction workflow <projectPath> <@unbind.json>
```

### 时间触发器

```bash
cloudcc createTimeTrigger workflow <projectPath> <@spec.json>
cloudcc updateTimeTrigger workflow <projectPath> <@patch.json>
cloudcc deleteTimeTrigger workflow <projectPath> <@delete.json>
```

### 周期触发器

```bash
cloudcc createRecurringTrigger workflow <projectPath> <@spec.json>
cloudcc updateRecurringTrigger workflow <projectPath> <@patch.json>
cloudcc deleteRecurringTrigger workflow <projectPath> <@delete.json>
```

### 计划执行和审计

```bash
cloudcc apply msapi <projectPath> <planId>
cloudcc operation msapi <projectPath> <operationId>
cloudcc changes msapi <projectPath> <operationId>
cloudcc rollback-plan msapi <projectPath> <operationId>
```

## 2. 创建规则

`workflow.json`：

```json
{
  "apiName": "account_status_rule",
  "name": "客户状态工作流",
  "description": "客户状态满足条件时触发",
  "targetObjectId": "obj-account",
  "targetObject": "客户",
  "evaluateRuleType": "U",
  "condtionOption": "R",
  "condition": {
    "filter": "1 AND 2",
    "items": [
      {"fieldId":"field-status","operator":"=","value":"ACTIVE"},
      {"fieldId":"field-level","operator":"!=","value":"INVALID"}
    ]
  }
}
```

```bash
cloudcc create workflow ./project @workflow.json
# 审阅 tp_sys_workflow 和 tp_sys_condition steps
cloudcc apply msapi ./project <planId>
cloudcc detail workflow ./project <workflowId>
```

创建时省略 `id`，MetadataService 会分配 `acd` ID；省略 `apiName` 时会生成稳定格式的 API 名。建议生产配置显式提供可读 API 名。

## 3. 规则字段与条件

| 字段 | 说明 |
| --- | --- |
| `id` / `workflowId` | 更新、启停、删除和子资源操作所需规则 ID |
| `apiName` | 规则 API 名 |
| `name` | 显示名 |
| `description` | 说明 |
| `targetObjectId` | 目标对象 ID；动作必须指向同一对象 |
| `targetObject` | 对象显示名 |
| `evaluateRuleType` | 平台值 `I`、`U`、`A` |
| `condtionOption` | 保留平台拼写；`R` 条件行，`F` 公式 |
| `functionCode` / `formula` | 公式条件文本 |
| `condition.items` / `conditions` | 条件行 |
| `condition.filter` / `bool_filter` | 条件逻辑，如 `1 AND (2 OR 3)` |

工作流更新采用条件集合三态语义：完全省略条件相关字段时保留现有条件；显式提交空 `conditions` / `condition.items` 时清空条件行；提交非空集合时先删除旧条件再插入完整新集合。切换公式/条件模式时应显式提交 `condtionOption` 及对应 `functionCode` 或条件集合。名称、说明等普通 patch 不会再隐式清空条件或把公式模式改回 `R`。

公式条件示例：

```json
{
  "id":"acd-account-status",
  "condtionOption":"F",
  "functionCode":"status__c = 'ACTIVE'"
}
```

## 4. 查询详情和关系结构

```bash
cloudcc get workflow ./project 客户
cloudcc detail workflow ./project acd-account-status
```

详情包含：

- `workflow`：规则根记录；
- `relations.tp_sys_condition`：条件；
- `relations.tp_sys_actions_relation`：即时、时间、周期动作关系；
- `relations.tp_sys_workflowdepend`：时间触发器；
- `relations.tp_sys_schedular`：周期触发器；
- `relationCounts`：各表行数。

后续解绑需要从关系行读取 `relationId`；更新/删除触发器需要读取 `triggerId`。时间和周期触发器均沿用 setup-service 默认 `DBMan.insert(obj)` 的无业务前缀 20 位 ID，禁止按 `awt/aws` 等猜测前缀构造。

## 5. 绑定已有即时动作

先确认动作存在且目标对象一致：

```bash
cloudcc detail workflowTask ./project acb-task-id
cloudcc detail workflow ./project acd-account-status
```

`bind.json`：

```json
{
  "id":"acd-account-status",
  "scope":{"type":"immediate"},
  "action":{"domain":"workflow-tasks","ref":"acb-task-id"}
}
```

```bash
cloudcc bindAction workflow ./project @bind.json
cloudcc apply msapi ./project <planId>
cloudcc detail workflow ./project acd-account-status
```

`action.domain` 支持 `workflow-tasks`、`workflow-email-alerts`、`workflow-field-updates`。只有即时作用域允许绑定已有动作。相同父级、类型和动作的重复关系返回 `workflow_action_relation_duplicate`。

审阅关系计划时，`tp_sys_actions_relation.triggerType` 应与 setup-web 一致：即时为 `workflow`，时间为
`workflowtime`，周期为 `workflowrecurrency`；`relatedId` 分别指向工作流 ID、时间触发器 ID、周期触发器 ID。
CLI 的 `scope.type` 仍使用更易读的 `immediate`、`time`、`recurring`，不要在输入 JSON 中混用底层值。

`workflow-tasks` 与 `workflow-email-alerts` 的标准表没有 API 名列，`action.ref` 必须使用创建计划/readback 返回的真实 ID：任务为 `acb...`，邮件动作为无业务前缀的 20 位 ID。`workflow-field-updates` 具有 `UNIQUE_NAME`，可使用真实 ID 或 `afud_...` API 名。

## 6. 创建并绑定动作

`create-and-bind.json`：

```json
{
  "id":"acd-account-status",
  "scope":{"type":"immediate"},
  "action":{
    "domain":"workflow-email-alerts",
    "create":{
      "targetObjectId":"obj-account",
      "targetObject":"客户",
      "name":"客户通知",
      "template":"AccountNotice",
      "recipients":[{"type":"relatedUserField","ref":"ownerid"}]
    }
  }
}
```

```bash
cloudcc createAction workflow ./project @create-and-bind.json
```

同一计划先创建动作根记录，再创建 `tp_sys_actions_relation`，apply 时在同一 MetadataService 事务中执行。动作字段详见三个动作 Domain 文档。

## 7. 解绑

`unbind.json`：

```json
{"id":"acd-account-status","relationId":"abf-relation-id"}
```

```bash
cloudcc unbindAction workflow ./project @unbind.json
cloudcc apply msapi ./project <planId>
```

解绑只删除关系，不删除任务、邮件通知或字段更新定义。`relationId` 必须确实属于该工作流或其触发器，否则返回 not found。

## 8. 时间触发器

创建 `time-trigger.json`：

```json
{
  "id":"acd-account-status",
  "field":"field-contract-end",
  "operator":"-",
  "amount":1,
  "unit":"day"
}
```

- `field` 必须是工作流目标对象上的日期或日期时间字段，API 名会归一为真实字段 ID；
- `operator` 为 `+` 或 `-`；
- `amount` 为大于等于 0 的整数；
- `unit` 为 `day` 或 `hour`。

```bash
cloudcc createTimeTrigger workflow ./project @time-trigger.json
cloudcc apply msapi ./project <planId>
cloudcc detail workflow ./project acd-account-status
```

更新采用部分更新：

```json
{"id":"acd-account-status","triggerId":"time-trigger-id","amount":2}
```

删除：

```json
{"id":"acd-account-status","triggerId":"time-trigger-id"}
```

删除时间触发器会先删除它拥有的动作关系，再删除触发器；动作定义保留。

时间作用域不能绑定已有动作。创建新动作时 scope 为：

```json
{"type":"time","triggerId":"time-trigger-id"}
```

## 9. 周期触发器

`recurring-trigger.json`：

```json
{
  "id":"acd-account-status",
  "startDate":"2026-11-06",
  "endDate":"2027-11-05",
  "startTimeHour":"09",
  "startTimeMinute":"00",
  "frequency":"weekly",
  "frequencyType":"w1",
  "intervals":1,
  "onWeekday":"1,3,5"
}
```

创建必须提供 `startDate`、`endDate`（`YYYY-MM-DD`）、`startTimeHour`（0-23）、`startTimeMinute`（`0/15/30/45`）、`frequency` 和 `frequencyType`；`intervals` 为正整数，默认 1。枚举与 setup-web 一致：

- `daily` 对应 `d1`（每天）或 `d2`（每隔 N 天）；
- `weekly` 对应 `w1`，`onWeekday` 是 `0` 到 `6` 的逗号列表；
- `monthly` 对应 `m1` 或 `m2`。`m1` 需要 `onDay=1..31`；`m2` 需要 `onWhichWeek=1..4` 和 `onWeekday=0..6`。

结束日期不得早于开始日期。可用更新字段为上述字段；更新只提交需要修改的字段，MetadataService 会把补丁与
当前调度合并后再校验，因此单独修改 `frequency` 或 `frequencyType` 也必须与保留值匹配。若切换频率类型，
应在同一补丁中提交匹配的 `frequency`、`frequencyType` 以及该类型要求的周/月字段。

更新必须包含 `triggerId` 和至少一个调度字段：

```json
{"id":"acd-account-status","triggerId":"recurring-trigger-id","startTimeHour":"10","startTimeMinute":"30"}
```

删除只需工作流 ID 和触发器 ID。删除时先清理关系，再删调度记录，动作定义保留。

周期 scope 为 `{"type":"recurring","triggerId":"recurring-trigger-id"}`。周期作用域允许新建任务和邮件通知，不允许字段更新。

## 10. 启用和停用

```bash
cloudcc enable workflow ./project acd-account-status
cloudcc apply msapi ./project <planId>
cloudcc disable workflow ./project acd-account-status
```

启用前 MetadataService 校验三类已知动作仍存在、动作目标对象与工作流一致，并拒绝周期字段更新。apply 还要求工作流更新恰好影响一行，避免对不存在规则静默成功。

## 11. 删除规则

```bash
cloudcc detail workflow ./project acd-account-status
cloudcc delete workflow ./project acd-account-status
cloudcc apply msapi ./project <planId>
```

清理顺序：时间/周期触发器动作关系、条件、即时关系、时间触发器、周期触发器、工作流根记录。任务、邮件通知、字段更新根记录全部保留，之后可用各动作 Domain 的 `impact` 检查是否还有引用并决定是否删除。

## 12. 常见错误

| 错误码 | 含义 |
| --- | --- |
| `workflow_not_found` | 工作流不存在 |
| `workflow_trigger_not_found` | triggerId 不属于该工作流 |
| `workflow_action_relation_not_found` | relationId 不属于该工作流图 |
| `workflow_action_relation_duplicate` | 重复绑定 |
| `workflow_existing_action_scope_unsupported` | 时间/周期作用域绑定已有动作 |
| `workflow_recurring_field_update_unsupported` | 周期作用域使用字段更新 |
| `workflow_action_target_mismatch` | 动作和规则目标对象不同 |
| `workflow_spec_invalid` | scope、触发器字段或值不合法 |

## 13. 上线检查清单

- 先创建规则和动作，apply 后分别 detail 回读；
- 从 workflow 详情获取真实 relationId/triggerId；
- 即时才使用 `bindAction`，时间/周期使用 `createAction`；
- 周期不使用字段更新；
- 启用前检查条件、触发器和每条动作关系；
- 审阅计划确认 `executionBackend=metadata-db`、`setupServiceRequired=false`；
- 删除前保存详情证据，删除后确认动作根仍按设计保留。
