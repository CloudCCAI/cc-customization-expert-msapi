# 电子邮件模板

`emailTemplate` 用于以稳定 API 名称管理平台设置页中的电子邮件模板，`emailTemplateFolder` 用于管理模板文件夹。CLI 复用平台标准 setup-service 能力；MSAPI 写操作先生成可审计计划，再由用户显式执行。

```text
cloudcc get emailTemplate <projectPath> [filter]
cloudcc detail emailTemplate <projectPath> <id-or-apiName>
cloudcc create emailTemplate <projectPath> <@spec.json>
cloudcc update emailTemplate <projectPath> <selector> <@patch.json>
cloudcc delete emailTemplate <projectPath> <selector>

cloudcc get emailTemplateFolder <projectPath> [filter]
cloudcc detail emailTemplateFolder <projectPath> <id-or-name>
cloudcc create emailTemplateFolder <projectPath> <@spec.json>
```

模板支持纯文本与 HTML、UTF-8 本地正文文件、合并字段、启停、移动文件夹和安全删除。系统模板、平台预置文件夹和存在已知引用的模板均只读或禁止删除。完整用法见：

```text
cloudcc doc emailTemplate devguide
```
