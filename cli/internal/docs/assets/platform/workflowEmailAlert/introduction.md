# 工作流电子邮件通知 Domain

`workflowEmailAlert` 是工作流电子邮件通知的一级 CLI Domain，MetadataService 规范名为 `workflow-email-alerts`，对应 setup-web `/#/settings/action/workflow-alerts`。它管理可复用的邮件动作定义，不管理电子邮件模板正文，也不直接发送测试邮件。

## 适用场景

- 工作流触发后用现有 EMAIL 模板通知用户、角色、角色及下级；
- 从目标记录的关系用户字段或电子邮件字段动态取得收件人；
- 配置最多 5 个类型化收件人和最多 5 个额外邮箱；
- 审计邮件动作与即时、时间、周期工作流之间的引用关系。

## 能力与边界

该 Domain 支持能力查询、列表、详情、对象/字段/模板/收件人候选、引用影响、创建、部分更新和受保护删除。MSAPI 要求 MetadataService `1.1.105+`，由 MetadataService 直接读写 `tp_sys_actions_email`，不调用 setup-service。写命令只生成计划，必须显式 apply。

邮件模板必须预先存在；模板正文、文件夹与访问控制由 `emailTemplate` / `emailTemplateFolder` 管理。邮件动作保存模板引用与收件人规则。当前 setup-web 表单不提供发件人设置，CLI 同样不接受 `from`，避免写入运行时不会消费的伪配置。平台将 `name` / `description` 映射到同一个动作显示名字段，它们不是两个可独立维护的说明字段。

动作与工作流的关系归 `workflow` Domain：已有邮件动作只能绑定即时作用域，时间和周期作用域只能创建并绑定新动作。解绑和删除工作流不会删除邮件动作根记录；仍被引用的动作不能删除或跨对象修改。

## 建议工作流

```bash
cloudcc candidates workflowEmailAlert . Account
cloudcc create workflowEmailAlert . @email-alert.json
# 记录创建计划返回的 20 位 target，它是邮件动作 ID
cloudcc apply msapi . <planId>
cloudcc detail workflowEmailAlert . <email-action-id>
cloudcc bindAction workflow . @bind-email.json
```

邮件动作表没有 API 名列，setup-service 对该表使用无业务前缀的 20 位标准 ID。创建、查询和绑定使用 MetadataService 返回的真实 ID，不要编造 `acc...` 或 `aemail_*`。CLI 的友好收件人 `{type,ref}` 会在计划中严格编译成平台运行时需要的 `{type,id,name}` 与底层类型值。

配置完成后应在测试工作流中验证实际邮件投递；MetadataService 的元数据回读只能证明配置已保存，不能替代邮件网关和模板渲染测试。

完整命令、字段、五类收件人、额外邮箱、绑定/解绑和错误处理见：

```bash
cloudcc doc workflowEmailAlert devguide
```
