# 电子邮件模板 CLI 用户指南

本指南无需阅读 setup-web 或 setup-service 源码。它覆盖资源发现、模板与文件夹 CRUD、正文文件、合并字段、计划执行、验证、回滚、安全限制和排障。

## 1. 能力边界

- `emailTemplate` 只管理 `EMAIL` 类型，不管理短信模板。
- `emailTemplateFolder` 管理邮件模板文件夹。
- MSAPI 是完整且推荐的通道：读取、计划、关系库写入、变更记录和回滚均由 MetadataService 管理，不调用 setup-service。
- Lightning UIAPI 仅在 capability 明确广告时复用标准接口；缺少安全引用检查时删除失败关闭。
- Horizontal UIAPI 当前没有已验证标准契约，失败关闭。可显式选择 MSAPI。
- CLI 不修改 setup-web、setup-service；CLI 本身不直接写表，MSAPI 由 MetadataService 在计划、漂移和审计约束下写电子邮件模板元数据表。

先检查项目连接与能力：

```text
cloudcc capabilities emailTemplate <projectPath>
cloudcc get emailTemplateFolder <projectPath>
cloudcc get emailTemplate <projectPath>
```

`projectPath` 是含 CLI 项目配置的目录。MSAPI 电子邮件模板只需要访问令牌、MetadataService 地址及其关系库数据源；`setupSvc` 不是本 Domain 的依赖，也不要写入 spec。若同一项目还使用 CCChat、搜索布局、移动端设置或队列等委托型 Domain，CLI 会从当前环境的 `setupSvc` 读取地址并随请求传给 MetadataService，无需在 MetadataService 再配置一份租户地址。

## 2. 稳定资源名与 selector

只能使用 `emailTemplate` 和 `emailTemplateFolder`。没有 `template`、`mailTemplate` 等含糊别名。

- 模板读取/写入 selector：精确平台 ID 或唯一 `apiName`。
- 文件夹 selector：精确平台 ID 或唯一名称。
- 模板 label 只用于展示和过滤，不能作为写 selector。
- 名称/API 名重复或 selector 命中多项时，CLI 不猜测目标。

```text
cloudcc get emailTemplate <projectPath> escalation
cloudcc detail emailTemplate <projectPath> CaseEscalationNotice
cloudcc detail emailTemplateFolder <projectPath> 服务通知
```

## 3. 创建文件夹

`folder.json`：

```json
{
  "name": "服务通知",
  "accessMode": "readWrite",
  "visibility": {"scope": "all"}
}
```

```text
cloudcc create emailTemplateFolder <projectPath> @folder.json
```

命令只创建计划，输出中记录 `planId`。检查步骤、目标和风险后执行：

```text
cloudcc apply msapi <projectPath> <planId>
```

`accessMode` 为 `readOnly` 或 `readWrite`。自定义文件夹的 `visibility.scope` 当前可靠支持 `all`。`selected` 或 `ownerOnly` 需要目标租户提供可验证的 typed-subject/owner 契约；契约不可用时会失败关闭，不接受底层 `accessibleuserset` 字符串。

平台预置的公用、专用、联系人通知文件夹可读取，但不可改名、改权限或删除。

## 4. 创建模板

`template.json`：

```json
{
  "apiName": "CaseEscalationNotice",
  "label": "服务请求升级通知",
  "folder": "服务通知",
  "subject": "服务请求 {!Case.Name} 已升级",
  "body": {
    "format": "html",
    "contentFile": "templates/case-escalation-notice.html"
  },
  "enabled": true,
  "encoding": "GBK",
  "description": "服务请求升级邮件"
}
```

```text
cloudcc create emailTemplate <projectPath> @template.json
cloudcc apply msapi <projectPath> <planId>
```

必填字段为 `apiName`、`label`、`folder`、`subject`、`body.format` 和正文。`apiName` 必须以字母开头，只含字母、数字、下划线，并在租户内唯一。CLI 不使用平台随机 API 名 fallback。

正文有两种互斥写法：

```json
{"body":{"format":"text","content":"您好 {!Contact.Name}"}}
```

```json
{"body":{"format":"html","contentFile":"templates/message.html"}}
```

当 spec 使用 `@file` 时，相对 `contentFile` 按 spec 文件所在目录解析；否则按 CLI 当前目录解析。文件必须是 UTF-8，CLI 安全上限为 1 MiB。CLI 在发网前读取文件并替换成 `content`，服务端不会接收本地路径。HTML 原文不格式化、不清洗、不下载外部资源；计划只记录正文 SHA-256、UTF-8 字节数和 token 数，不在日志中输出完整正文。

## 5. 合并字段

支持形式：

```text
{!ObjectApi.FieldApi}
{!ObjectApi.LookupFieldApi.TerminalFieldApi}
```

后一种最多一层 lookup。修改 subject 或正文时，会检查括号、标识符和关系深度；语法不合法时不会静默删除。保留模板的历史未知 token 在只修改名称、描述或启用状态时原样保存。

离线语法通过不代表真实邮件一定能渲染。上线前应使用目标对象的真实记录完成主题、HTML、空值、多语言和权限 UAT。

## 6. 更新、移动和启停

更新使用 patch；未出现的字段保持不变。`patch.json`：

```json
{
  "label": "服务请求升级提醒",
  "folder": "服务通知",
  "enabled": false,
  "description": "维护窗口暂时停用"
}
```

```text
cloudcc update emailTemplate <projectPath> CaseEscalationNotice @patch.json
cloudcc apply msapi <projectPath> <planId>
```

更新正文时仍使用完整 `body`，并指定 `format` 与 `content`/`contentFile`。`description` 可显式清空；subject 与正文不可为空。MetadataService 先读取 before-state，将 patch 合并成完整 desired-state，再生成受 schema 校验和行数断言保护的 SQL 计划，避免省略字段被清空。

文件夹更新：

```json
{"name":"服务通知（生产）","accessMode":"readOnly","visibility":{"scope":"all"}}
```

```text
cloudcc update emailTemplateFolder <projectPath> 服务通知 @folder-patch.json
```

## 7. 删除前影响检查

删除前必须先查看影响：

```text
cloudcc impact emailTemplate <projectPath> CaseEscalationNotice delete
cloudcc impact emailTemplateFolder <projectPath> 服务通知 delete
```

模板删除会检查平台 managed/system 标志以及已确认的消费者引用，包括动作邮件和审批邮件引用。发现引用即拒绝，必须先在相应标准功能中解除或迁移引用。

文件夹非空时默认拒绝：

```text
cloudcc delete emailTemplateFolder <projectPath> 服务通知
```

即使指定 `--cascade-templates`，当前版本仍要求先逐个模板完成独立引用检查并删除，然后删除空文件夹。这是故意的失败关闭策略，避免隐式级联绕过模板引用保护。

模板删除流程：

```text
cloudcc delete emailTemplate <projectPath> CaseEscalationNotice
cloudcc apply msapi <projectPath> <planId>
```

## 8. 计划、漂移、变更与回滚

所有 MSAPI 写命令仅生成计划。计划包含 before/desired hash、目标关系表、风险等级和正文摘要。应用前若模板或文件夹已被其他管理员修改，应用返回 state drift，必须重新读取并生成新计划；不要强行复用旧计划。

应用后通过通用变更命令查看 ledger，并用生成的 rollback 信息回滚。创建回滚使用计划中的平台 ID；删除后重建可能得到新平台 ID，因此引用恢复必须按变更记录复核。回滚同样执行漂移检查、事务行数断言和写后回读。

如果关系库执行结果不确定，MetadataService 会基于 ledger 和读回状态判断：

- 已是 desired：按成功收敛；
- 仍是 before：最多安全重试；
- 介于两者或无法读取：标记结果不确定并给出补偿信息，不盲目重复写。

## 9. 常见错误

- `email_template_spec_invalid`：字段、正文、format、合并 token 或未知只读字段非法。
- `email_template_api_code_duplicate`：`apiName` 已存在；改用唯一稳定名称。
- `email_template_not_found`：ID/API 名不存在，或登录租户不正确。
- `email_template_delete_referenced`：模板为系统/托管模板或仍有引用。
- `email_template_folder_not_empty`：先逐个检查并删除/移动模板。
- `email_template_folder_subject_resolution_unavailable`：租户没有安全 typed-subject 候选契约；不要提交底层权限串。
- `email_template_plan_state_drifted`：计划后对象发生变化；重新 detail、更新 spec、重新 plan。
- `setup_service_target_required`：只会出现在委托型 Domain；当前 CLI 环境缺少 `setupSvc`。电子邮件模板不应返回此错误。
- `setup_service_target_changed`：委托型 Domain 的 plan 与 apply 使用了不同 `setupSvc`；重新确认当前环境并生成计划。
- `metadata_table_missing` / `metadata_column_missing`：MetadataService 连接的数据源不是目标租户兼容 schema，先修正数据源，不要改走 setup-service 规避。
- `cloudcc_access_token_required`：当前命令没有可转发的 CloudCC access token。

HTTP 200 但平台业务结果为失败仍会被识别为错误；配额、权限和唯一性拒绝不会当成成功。

## 10. 上线检查清单

1. `capabilities` 确认当前 provider、平台轴和版本。
2. `get/detail` 确认 selector 唯一且目标不是 system/managed。
3. 文件夹存在并符合访问范围；不要依赖隐藏的底层权限串。
4. HTML/纯文本、UTF-8、正文大小和外部资源均已检查。
5. 合并 token 使用目标租户真实对象/字段，并用真实记录 UAT。
6. 保存 plan JSON/planId，复核风险、before/desired hash 和正文摘要。
7. 显式 apply，随后 detail 验证 apiName、folder、subject、format、enabled 和正文 hash。
8. 删除前运行 impact，并迁移全部引用；文件夹先清空。
9. 保存 change/rollback 证据；涉及删除重建时记录新旧平台 ID 映射。
10. 凭据、真实客户正文和租户数据不得进入通用 skill 包或发布归档。
