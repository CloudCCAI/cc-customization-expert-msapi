# CloudCC 自定义设置操作指南

查看本文档：

```bash
cloudcc doc platform/customSetting devguide
```

自定义设置由主对象、字段、默认语言标签和布局组成。写操作先生成 MetadataService 计划，不会在 `plan` 阶段直接修改数据库；审查计划后再显式执行 `apply`。

## 查询

```bash
cloudcc get customSetting <projectPath> [selector]
cloudcc detail customSetting <projectPath> <settingId>
```

列表用于定位自定义设置，详情用于确认主对象及字段。字段出现在详情页的必要条件不仅是字段主记录存在，还必须有匹配页面语言的 `tp_sys_multi_lang` 字段标签，并被放入该自定义设置的布局。

## 新建自定义设置

将规格保存为 `custom-setting-create.json`：

```json
{
  "tpSysObjectVO": {
    "id": "cs_invoice_limit",
    "schemetableName": "invoice_limit",
    "label": "开票额度设置",
    "dataType": "L",
    "accessable": "2"
  },
  "fields": [
    {
      "id": "cs_field_limit",
      "apiName": "limit_amount",
      "nameLabel": "额度",
      "type": "N"
    }
  ]
}
```

可以使用快捷命令，也可以显式使用 MSAPI 工作流：

```bash
cloudcc create customSetting <projectPath> @custom-setting-create.json

cloudcc plan msapi <projectPath> custom-settings @custom-setting-create.json create
cloudcc apply msapi <projectPath> <planId>
```

新字段至少应提供 `apiName`、`nameLabel` 和 `type`；建议显式提供稳定的 `id`。`nameLabel` 会生成默认语言字段标签，不能只写字段主表而省略标签。

## 给已有自定义设置新增字段

`update` 是局部更新。只提交 `id` 和 `fields` 时，不会重写主对象名称、物理表名、前缀或其他未提交属性。

```json
{
  "id": "cs_invoice_limit",
  "fields": [
    {
      "id": "cs_field_warning",
      "apiName": "warning_text",
      "nameLabel": "预警说明",
      "type": "S",
      "schemefieldLength": "255"
    }
  ]
}
```

```bash
cloudcc update customSetting <projectPath> <settingId> @custom-setting-update.json

cloudcc plan msapi <projectPath> custom-settings @custom-setting-update.json update
cloudcc apply msapi <projectPath> <planId>
```

CLI 会将 `update`、`modify`、`editSave` 和 `save` 的自定义设置快捷入口映射为 MSAPI `update`，不会误用整体 `upsert` 语义。字段按以下顺序识别：

1. 优先使用字段 `id`。
2. 没有命中时，使用当前自定义设置下的 `apiName` 匹配。
3. 均未命中时创建字段，并创建当前语言的默认标签。
4. ID 与 API 名分别命中不同记录，或存在重复标签时，计划会失败并要求先清理歧义。

## 修改已有字段

```json
{
  "id": "cs_invoice_limit",
  "layoutPolicy": "preserve",
  "fields": [
    {
      "id": "cs_field_limit",
      "apiName": "limit_amount",
      "nameLabel": "单笔额度",
      "type": "N"
    }
  ]
}
```

已有字段和已有多语言标签使用严格更新：目标不存在或命中数量不符合预期时，应用阶段会失败，不会把“更新 0 行”当成成功。标签存在时保留其原 ID；标签不存在时插入新行。

## 布局策略

`layoutPolicy` 控制字段如何进入详情布局：

- `all-profiles`：默认值。为所有现有简档补齐布局关联，并将当前自定义设置的全部有效自定义字段补入布局。
- `selected-profiles`：只处理 `profileIds` 中列出的简档；`profileIds` 不能为空。
- `preserve`：不修改布局，适合只改字段标签或其他不影响布局的属性。

布局同步采用非破坏式补齐，不会先删除所有简档布局。若数据库中存在多个根布局或多个候选区块，计划会失败并要求显式消除歧义。

## 审查、应用与回滚

```bash
cloudcc operation msapi <projectPath> <operationId>
cloudcc changes msapi <projectPath> <operationId>
cloudcc apply msapi <projectPath> <planId>
cloudcc rollback-plan msapi <projectPath> <operationId>
cloudcc rollback msapi <projectPath> <operationId>
```

应用前重点检查：

- 新字段应包含 `custom-setting-field.insert` 和 `custom-setting-language.insert`。
- 已有字段应包含 `custom-setting-field.update`；已有标签应按实际标签 ID 更新。
- 仅新增字段时，不应出现 `custom-setting-object.update`。
- 非 `preserve` 策略应包含目标简档的布局关联和全部有效字段的区块关联。
- `country` 应与页面实际语言一致；默认中文通常为 `zh`，不要仅依据操作端的 locale 猜测。

应用后再次执行 `cloudcc detail customSetting`，并在实际页面语言下确认字段标签和布局均可见。若字段主记录存在但页面不显示，优先检查对应 `relationId + type=field + country` 的 `tp_sys_multi_lang` 标签，以及布局区块关联，而不是重复创建字段。

## 删除

```bash
cloudcc delete customSetting <projectPath> <settingId>
```

删除同样先生成计划，但执行边界与查询、创建、更新不同：MSAPI 会把整个删除操作委托给当前 CLI 环境的
`setupSvc` 标准接口 `/api/customsetting/deleteobj`。标准服务根据租户 `dbIndex` 重命名非空 Mongo 值集合，
自动计算 `_C_DEL<n>` 历史后缀，软删除关系库元数据并刷新平台缓存。MetadataService 不直接连接租户 Mongo，
spec 中也不要提供 `setupSvc`、Mongo 地址或 `deleteSuffix`。

项目环境至少需要：

```json
{
  "setupSvc": "https://tenant.example.com/setup",
  "metadataService": {"url": "https://metadata.example.com"}
}
```

计划只保存目标地址指纹，不保存地址和令牌。生成计划后切换环境或修改 `setupSvc`，apply 会返回
`setup_service_target_changed`，必须在正确环境重新生成计划。

删除不会自动重试，也不能由 CLI 自动回滚。若返回 `uncertain_remote_outcome`，表示网络失败发生时远端状态
不能安全判定；先执行 `detail customSetting` 并检查平台页面，禁止直接重复 apply。恢复需要按审批流程同时处理
Mongo 集合、对象元数据和缓存。

应用前确认业务代码、配置数据和布局不存在仍需保留的依赖，并检查计划只有一个 `PLATFORM_DELETE` 步骤，
其 `platformDeleteMutation.targetService` 为 `SETUP`。

常见错误：

- `setup_service_target_required`：当前环境缺少合法 `setupSvc`。
- `setup_service_target_changed`：plan 与 apply 的活动环境或地址不同。
- `platform_delete_plan_state_drifted`：生成计划后自定义设置发生变化，重新生成计划。
- `platform_delete_rejected`：平台标准服务拒绝删除，按返回信息处理依赖或权限问题。
- `uncertain_remote_outcome`：不要自动重试，先做只读核对和人工恢复判断。
