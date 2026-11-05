# 工作流任务开发指南

`workflowTask` 对应 setup-web `/#/settings/action/workflow-tasks`，MSAPI Domain 为 `workflow-tasks`，最低 MetadataService `1.1.105`。任务定义与工作流绑定分离：本 Domain 管任务，`workflow` Domain 管关系。

所有写命令只生成 MetadataService 数据库计划，不调用 setup-service，也不自动 apply。

## 1. CLI 命令

```bash
cloudcc capabilities workflowTask <projectPath>
cloudcc get workflowTask <projectPath> [filter]
cloudcc detail workflowTask <projectPath> <actionId>
cloudcc candidates workflowTask <projectPath> [object]
cloudcc impact workflowTask <projectPath> <actionId>
cloudcc create workflowTask <projectPath> <@spec.json>
cloudcc update workflowTask <projectPath> <selector> <@patch.json>
cloudcc delete workflowTask <projectPath> <selector>
cloudcc apply msapi <projectPath> <planId>
```

## 2. 从候选项开始

```bash
cloudcc candidates workflowTask ./project Account
```

结果包含对象、对象字段、`users[]`、`roles[]`、`assigneeTypes`、`statuses` 和 `priorities`。
使用 `user` 或 `role` 负责人时，直接复制本次候选结果中的真实 `id`；MetadataService 会在计划阶段确认该
用户或角色仍存在。关系用户字段和到期日期字段必须来自目标对象；不要从其他环境复制数据库 ID 后直接使用。

## 3. 创建任务

`task.json`：

```json
{
  "targetObjectId": "Account",
  "targetObject": "客户",
  "subject": "客户跟进",
  "comments": "请在到期日前联系客户并记录结果",
  "status": "未开始",
  "priority": "中",
  "assignee": {
    "type": "owner",
    "ref": "Account:ownerid"
  },
  "dueDate": {
    "field": "triggerTime",
    "operator": "+",
    "days": 1
  },
  "notifyAssignee": true
}
```

```bash
cloudcc create workflowTask ./project @task.json
# 审阅 planId、tp_sys_actions_task、ownertype/ownid、到期日和 notifyOwner
# 记录计划返回的 target（acb...），下文记为 <taskId>
cloudcc apply msapi ./project <planId>
cloudcc detail workflowTask ./project <taskId>
```

## 4. 字段合同

| 字段 | 创建 | 更新 | 说明 |
| --- | --- | --- | --- |
| `targetObjectId` | 必填 | 可选 | 对象 ID/API 名/可唯一解析名称 |
| `targetObject` | 必填 | 可选 | 对象显示名 |
| `subject` | 必填 | 可选 | 任务主题，也是绑定关系显示名 |
| `comments` | 可选 | 可选 | 任务说明；默认空 |
| `status` | 可选 | 可选 | `未开始`、`进行中`、`已结束`；默认 `未开始` |
| `priority` | 可选 | 可选 | `高`、`中`、`低`；默认 `中` |
| `assignee` | 必填 | 可选 | 分配对象，见下一节 |
| `dueDate` | 必填 | 可选 | 相对日期规则 |
| `notifyAssignee` | 可选 | 可选 | 是否通知负责人；默认 `false` |

`tp_sys_actions_task` 没有 `API_NAME` 列，因此创建合同不接受 `apiName`。MetadataService 生成标准 `acb...` 动作 ID；创建计划的 `target`、apply readback 和列表/详情返回的 `id` 是唯一可靠选择器。不要用主题当选择器，主题允许重名。

## 5. 负责人类型

```json
{"type":"user","ref":"0052026ABCDEFghijk"}
{"type":"role","ref":"aaa2026ABCDEFghijk"}
{"type":"owner","ref":"Account:ownerid"}
{"type":"createdBy","ref":"Account:createbyid"}
{"type":"relatedUserField","ref":"field-related-user"}
```

- `user`：明确用户 ID；
- `role`：角色 ID；
- `owner`：触发记录所有者；
- `createdBy`：触发记录创建人；
- `relatedUserField`：目标对象上的用户关系字段。

`assignee.type` 和 `assignee.ref` 始终必填。`user` / `role` 必须提交当前 `candidates` 返回的目标环境
物理 ID；不存在的 ID 会在生成计划时被拒绝。`owner` / `createdBy` 使用
`objectApiName:fieldApiName` 路径；`relatedUserField` 必须引用目标对象上的 lookup 字段，字段 API 名会
归一为真实字段 ID。更新负责人时必须同时提交完整 `assignee` 对象，不能只更新其中一个属性。

## 6. 到期日期

以工作流触发时间后一天为到期日：

```json
{"field":"triggerTime","operator":"+","days":1}
```

以目标对象日期字段前三天为到期日：

```json
{"field":"field_contract_end","operator":"-","days":3}
```

`operator` 只能是 `+` 或 `-`，`days` 必须是大于等于 0 的 JSON 整数。除 `triggerTime` 外，字段必须是属于 `targetObjectId` 的日期或日期时间字段，并会被归一为真实字段 ID。

## 7. 查询和部分更新

```bash
cloudcc get workflowTask ./project 跟进
cloudcc detail workflowTask ./project <taskId>
```

`task-patch.json`：

```json
{
  "subject": "重点客户跟进",
  "priority": "高",
  "dueDate": {"field":"triggerTime","operator":"+","days":0},
  "notifyAssignee": true
}
```

```bash
cloudcc update workflowTask ./project <taskId> @task-patch.json
cloudcc apply msapi ./project <planId>
cloudcc detail workflowTask ./project <taskId>
```

未提交字段保持不变。主题变化会同步所有 `tp_sys_actions_relation.actionName`。动作已被引用时，不允许改变目标对象；未被引用时若改变 `targetObjectId`，必须同时提交 `targetObject`、完整 `assignee` 和完整 `dueDate`，避免保留旧对象的字段引用。

## 8. 绑定、创建并绑定、解绑

绑定已有任务到即时动作：

```json
{"id":"acd_account_rule","scope":{"type":"immediate"},"action":{"domain":"workflow-tasks","ref":"acb-action-id"}}
```

```bash
cloudcc bindAction workflow ./project @bind-task.json
```

时间或周期作用域不能绑定已有任务，必须创建并绑定：

```json
{
  "id":"acd_account_rule",
  "scope":{"type":"time","triggerId":"time-trigger-id"},
  "action":{"domain":"workflow-tasks","create":{
    "targetObjectId":"Account","targetObject":"客户",
    "subject":"到期客户跟进","assignee":{"type":"owner","ref":"Account:ownerid"},
    "dueDate":{"field":"triggerTime","operator":"+","days":0}
  }}
}
```

```bash
cloudcc createAction workflow ./project @create-time-task.json
```

解绑使用工作流详情或 `impact workflowTask` 返回的 `relationId`：

```json
{"id":"acd_account_rule","relationId":"abf-relation-id"}
```

```bash
cloudcc unbindAction workflow ./project @unbind.json
```

## 9. 删除保护

```bash
cloudcc impact workflowTask ./project <taskId>
cloudcc delete workflowTask ./project <taskId>
```

存在任何即时、时间或周期引用时，删除返回 `workflow_action_delete_referenced`。CLI 不会隐式解绑。先通过 workflow 逐个解绑，确认 `impact` 的 `count=0`，再生成和应用删除计划。

## 10. 常见错误和检查清单

- `workflow_action_spec_invalid`：检查是否误传 `apiName`、枚举、负责人、到期字段和天数。
- `workflow_action_target_change_referenced`：先解绑再换对象。
- `workflow_action_delete_referenced`：先运行 `impact` 并解绑。
- `workflow_action_managed`：平台托管动作不可写。
- `workflow_existing_action_scope_unsupported`：已有动作只能绑定即时作用域。

上线前确认：候选项来自目标环境、计划表为 `tp_sys_actions_task`、apply 后已回读、绑定后的 workflow 详情正确、删除前引用数为零。
