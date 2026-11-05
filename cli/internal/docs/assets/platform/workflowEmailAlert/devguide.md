# 工作流电子邮件通知开发指南

`workflowEmailAlert` 对应 setup-web `/#/settings/action/workflow-alerts`，MSAPI Domain 为 `workflow-email-alerts`，最低 MetadataService `1.1.105`。邮件动作与工作流绑定分离；本 Domain 管定义，`workflow` 管关系。

MSAPI 使用 MetadataService 直连元数据数据库，不调用 setup-service。写命令仅返回计划，必须显式 apply。

## 1. 命令总览

```bash
cloudcc capabilities workflowEmailAlert <projectPath>
cloudcc get workflowEmailAlert <projectPath> [filter]
cloudcc detail workflowEmailAlert <projectPath> <actionId>
cloudcc candidates workflowEmailAlert <projectPath> [object]
cloudcc impact workflowEmailAlert <projectPath> <actionId>
cloudcc create workflowEmailAlert <projectPath> <@spec.json>
cloudcc update workflowEmailAlert <projectPath> <selector> <@patch.json>
cloudcc delete workflowEmailAlert <projectPath> <selector>
cloudcc apply msapi <projectPath> <planId>
```

## 2. 候选项和前置条件

```bash
cloudcc candidates workflowEmailAlert ./project Account
```

返回：

- `objects[]`：可用对象；
- `fields[]`：对象字段，用于关系用户字段或电子邮件字段收件人；
- `templates[]`：类型为 EMAIL 的模板 ID、API 名、名称；
- `users[]`：可作为明确收件人的用户 ID、名称和登录名；
- `roles[]`：可作为角色或角色及下级收件人的角色 ID、名称；
- `recipientTypes`：支持的收件人类型。

创建前必须先有可用邮件模板。`template` 可以提交模板 ID、API 名或可唯一解析的名称，服务端保存模板 ID。
`user`、`role`、`roleAndSubordinates` 的 `ref` 应直接取本次候选结果；不存在的 ID 会在计划阶段失败。

## 3. 创建邮件通知

`email-alert.json`：

```json
{
  "targetObjectId": "Account",
  "targetObject": "客户",
  "name": "客户状态通知",
  "description": "客户状态通知",
  "template": "AccountStatusNotice",
  "recipients": [
    {"type":"relatedUserField","ref":"ownerid"},
    {"type":"emailField","ref":"field_contact_email"}
  ],
  "additionalEmails": ["audit@example.com"]
}
```

```bash
cloudcc create workflowEmailAlert ./project @email-alert.json
# 审阅 tp_sys_actions_email、emailTemplateId、recipientSet、addtionalEmail
# 记录计划返回的无业务前缀 20 位 target，下文记为 <emailAlertId>
cloudcc apply msapi ./project <planId>
cloudcc detail workflowEmailAlert ./project <emailAlertId>
```

`additionalEmails` 可省略，默认空列表。

## 4. 字段合同

| 字段 | 创建 | 更新 | 说明 |
| --- | --- | --- | --- |
| `targetObjectId` | 必填 | 可选 | 对象 ID/API 名/可唯一解析名称 |
| `targetObject` | 必填 | 可选 | 对象显示名 |
| `name` | 必填（推荐） | 可选 | 动作显示名；修改会同步绑定关系显示名 |
| `description` | `name` 省略时必填 | 可选 | 平台兼容别名，与 `name` 指向同一显示名字段；两者同时提交时必须相同 |
| `template` | 必填 | 可选 | EMAIL 模板 ID/API 名/唯一名称 |
| `recipients` | 必填 | 可选 | 1 到 5 个收件人的数组 |
| `additionalEmails` | 可选 | 可选 | 额外邮箱，最多 5 个唯一地址 |

`tp_sys_actions_email` 没有 `API_NAME` 列，因此创建合同不接受 `apiName`。与 setup-service 的 `DBMan.insert(obj)` 一致，MetadataService 生成以年份开头、无业务前缀的 20 位动作 ID；创建计划的 `target`、apply readback 和列表/详情返回的 `id` 是详情、更新、影响、删除及绑定的选择器。名称允许重名，不能作为选择器。

当前 setup-web 标准表单没有发件人配置，setup-service 的当前保存逻辑也不写 `FROM_EMAIL`。因此 CLI 不接受 `from`，避免产生“数据库有值但运行时不使用”的伪配置。邮件实际发件身份由目标环境邮件系统决定。

## 5. 收件人类型

```json
{"type":"user","ref":"0052026ABCDEFghijk"}
{"type":"role","ref":"aaa2026ABCDEFghijk"}
{"type":"roleAndSubordinates","ref":"aaa2026ABCDEFghijk"}
{"type":"relatedUserField","ref":"ownerid"}
{"type":"emailField","ref":"field_contact_email"}
```

- `user`：明确用户；
- `role`：角色成员；
- `roleAndSubordinates`：角色及下级；
- `relatedUserField`：目标对象用户关系字段；
- `emailField`：目标对象电子邮件字段。

每项都必须有非空字符串 `ref`。`user`、`role` 和 `roleAndSubordinates` 必须使用目标环境回读的字母数字物理 ID；`relatedUserField` 必须引用目标对象上的 lookup 字段，`emailField` 必须引用目标对象上的电子邮件字段；服务端会把字段 API 名归一为真实字段 ID。`recipients` 必须为 1 到 5 项，与 setup-web 的选择上限一致，CLI 保留数组顺序。

计划审阅时，`recipientSet` 必须是平台 `{type,id,name}` 格式，例如
`[{"type":"relateuser","id":"ownerid","name":"所有人"}]`；键为 `id` 而不是 `ref`，类型分别编译为
`user`、`role`、`roleandsub`、`relateuser`、`emailfield`。详情回读会再转换为上述 CLI
友好类型。若计划仍出现 `{type,ref}`，不要 apply。

## 6. 额外邮箱规则

```json
"additionalEmails": ["AUDIT@example.com", "ops@example.com"]
```

服务端会去除首尾空格、转为小写、按首次出现顺序去重并验证邮箱格式。去重后最多 5 个；超过限制或格式非法会在计划阶段失败。

## 7. 查询和部分更新

```bash
cloudcc get workflowEmailAlert ./project 状态
cloudcc detail workflowEmailAlert ./project <emailAlertId>
```

`email-alert-patch.json`：

```json
{
  "name": "重点客户状态通知",
  "template": "VipAccountStatusNotice",
  "recipients": [
    {"type":"relatedUserField","ref":"ownerid"},
    {"type":"role","ref":"aaa2026ABCDEFghijk"}
  ],
  "additionalEmails": []
}
```

```bash
cloudcc update workflowEmailAlert ./project <emailAlertId> @email-alert-patch.json
cloudcc apply msapi ./project <planId>
cloudcc detail workflowEmailAlert ./project <emailAlertId>
```

更新为部分更新；数组字段一旦提交即整体替换。动作被引用时不能改变目标对象；未被引用时若改变 `targetObjectId`，必须同时提交 `targetObject` 和完整 `recipients`，以重新校验字段型收件人。名称变化会同步绑定关系的 `actionName`。

## 8. 绑定到工作流

绑定已有邮件动作只能用于即时作用域：

```json
{"id":"acd_account_rule","scope":{"type":"immediate"},"action":{"domain":"workflow-email-alerts","ref":"email-action-id"}}
```

```bash
cloudcc bindAction workflow ./project @bind-email.json
```

时间或周期作用域必须创建并绑定新动作：

```json
{
  "id":"acd_account_rule",
  "scope":{"type":"recurring","triggerId":"recurring-trigger-id"},
  "action":{"domain":"workflow-email-alerts","create":{
    "targetObjectId":"Account","targetObject":"客户",
    "name":"每日客户通知","template":"DailyAccountNotice",
    "recipients":[{"type":"relatedUserField","ref":"ownerid"}]
  }}
}
```

```bash
cloudcc createAction workflow ./project @create-recurring-email.json
```

## 9. 引用、解绑和删除

```bash
cloudcc impact workflowEmailAlert ./project <emailAlertId>
```

使用返回的 `relationId` 解绑：

```json
{"id":"acd_account_rule","relationId":"abf-relation-id"}
```

```bash
cloudcc unbindAction workflow ./project @unbind.json
cloudcc apply msapi ./project <planId>
cloudcc impact workflowEmailAlert ./project <emailAlertId>
cloudcc delete workflowEmailAlert ./project <emailAlertId>
```

存在引用时删除返回 `workflow_action_delete_referenced`，不会自动解绑或删除工作流。

## 10. 常见错误与上线检查

| 错误码 | 处理 |
| --- | --- |
| `workflow_email_template_invalid` | 先用 `candidates` 确认 EMAIL 模板 |
| `workflow_action_spec_invalid` | 检查是否误传 `apiName`、收件人、邮箱和必填字段 |
| `workflow_action_target_change_referenced` | 先解绑再换对象 |
| `workflow_action_delete_referenced` | 先 impact、解绑并复查 |
| `workflow_action_managed` | 平台托管动作只读 |

上线前确认模板在目标环境有效、类型化收件人和额外地址分别不超过 5 个、字段型收件人属于目标对象、计划不含 setup-service、apply 后已回读，并通过测试工作流验证实际邮件投递。
