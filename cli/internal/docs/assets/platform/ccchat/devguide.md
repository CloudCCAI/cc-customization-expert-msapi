# CCChat 设置与微贴跟踪完整操作指南

## 能力边界

本 Domain 只覆盖 Setup 中的 CCChat 全局设置和微贴跟踪。它不管理数据归档、发帖、评论、点赞、关注、群组或个人邮件提醒，也不会修改现有 `fieldTracking` 字段历史跟踪。

MSAPI 写操作只生成计划，必须显式 apply。MetadataService 记录 before/after、漂移校验与 rollback 数据，再调用 setup-service；Mongo 地址、凭据、accessToken、Cookie 不进入 plan 或审计账本。首次把 CCChat 从关闭改为启用可能初始化 People 文档，rollback 只恢复设置，不删除 People。

## 正式命令

```text
cloudcc get        ccchat <projectPath> settings
cloudcc update     ccchat <projectPath> settings <encodedSpecJson|@file>
cloudcc getList    ccchat <projectPath> feedTracking
cloudcc get        ccchat <projectPath> feedTracking <object>
cloudcc candidates ccchat <projectPath> feedTracking <object>
cloudcc update     ccchat <projectPath> feedTracking <encodedSpecJson|@file>
cloudcc clear      ccchat <projectPath> feedTracking <object>
```

`projectPath` 用于读取 provider 和认证配置；可以传 `.`。`object` 在 MSAPI 下支持对象 ID、API Name 或 prefix。JSON 可直接传入，也可使用 UTF-8 `@file`。

兼容入口为 `ccchatSetting`、`ccchatFeedTracking` 和 `feedTracking`，均归一化为唯一的 `ccchat` Domain。`fieldTracking` 不属于本 Domain。

## 全局设置

四个布尔字段必须全部显式提供，而且必须是真正的 JSON boolean。

```json
{
  "enabled": true,
  "allowFeedEdit": true,
  "allowGroupArchived": false,
  "allowInvite": true,
  "emailDomains": ["example.com", "example.org"]
}
```

`allowInvite=true` 时至少提供一个合法域名。`allowInvite=false` 时必须省略 `emailDomains`；服务端保留既有域名，避免完整保存误清空。

```powershell
cloudcc get ccchat . settings
cloudcc update ccchat . settings @settings.json
```

## 微贴跟踪

`trackedFields` 是完整替换集合，支持字段 ID 或 API Name。服务端从 setup-service 候选字段派生 `fieldId/api/fieldType/datafield`，调用方不能提交或伪造这些物理字段。最多 20 个字段。

```json
{
  "object": "Account",
  "enabled": true,
  "trackedFields": ["name", "status__c"]
}
```

```powershell
cloudcc getList ccchat . feedTracking
cloudcc get ccchat . feedTracking Account
cloudcc candidates ccchat . feedTracking Account
cloudcc update ccchat . feedTracking @tracking.json
cloudcc clear ccchat . feedTracking Account
```

`clear` 等价于关闭该对象并把字段集合替换为空。普通 update 即使 `enabled=false`，仍保存显式提供的完整字段集合。

## MSAPI 审计生命周期

```powershell
cloudcc plan msapi . ccchat @settings.json update-settings
cloudcc plan msapi . ccchat @tracking.json update-feed-tracking
cloudcc plan msapi . ccchat '{"object":"Account"}' clear-feed-tracking
cloudcc apply msapi . <planId>
cloudcc operation msapi . <operationId>
cloudcc changes msapi . <operationId>
cloudcc rollback-plan msapi . <operationId>
cloudcc rollback msapi . <operationId>
```

apply 会先重新读取 setup-service 状态并比较计划中的 SHA-256；状态变化时报 `ccchat_plan_state_drifted`，需要重新生成计划。写入后必须回读一致才算成功。远程结果不确定时先回读，不盲目重复写入。

## Provider 行为

- MSAPI：需要 MetadataService 1.1.87+，提供完整审计生命周期；setup-service 仍是权威执行者。
- Lightning UIAPI：直接调用 `/api/chatterGroup/*` 和 `/api/feedTrack/*`，保存后回读；没有 MetadataService ledger/rollback。
- Universal auto：按项目 provider 选择；已选择 MSAPI 时不因依赖错误静默回落。
- Horizontal UIAPI：在认证和网络调用前返回 `ccchat_horizontal_uiapi_unsupported`。

## 常见错误

- `ccchat_settings_invalid`：布尔字段缺失/类型错误，或邀请域名规则不满足。
- `ccchat_object_not_found` / `ccchat_object_ambiguous`：对象选择器无法唯一解析。
- `ccchat_field_not_candidate` / `ccchat_field_ambiguous`：字段不在 setup-service 候选集合或不唯一。
- `ccchat_field_limit_exceeded`：超过 20 个字段。
- `ccchat_plan_state_drifted`：plan 后配置已变化。
- `ccchat_readback_mismatch`：保存后的权威状态与计划不一致。
- `ccchat_setup_service_unavailable`：MetadataService 未配置或无法访问 setup-service。
- `ccchat_setup_response_invalid`：setup-service 虽返回成功 envelope，但缺少必要设置、对象身份、字段数组或物理字段映射；CLI/MetadataService 会失败关闭，不把缺失值默认为关闭或空集合。

检查顺序：`cloudcc doctor platform <projectPath>`、`cloudcc doctor provider <projectPath>`、`cloudcc domain ccchat`，再执行只读命令。不要配置 Mongo 地址作为补救；Mongo 路由属于 setup-service。
