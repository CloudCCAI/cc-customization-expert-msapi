# CloudCC 简档 CLI 开发说明

## 读取契约

简档读取使用 MetadataService 专用端点：

```text
GET /metadata/v1/profiles?filter=<text>
GET /metadata/v1/profiles?selector=<id-or-name-or-apiName>
GET /metadata/v1/profiles/{id}
```

- `filter` 对 ID、名称、API 名称和描述做不区分大小写的包含匹配。
- `selector` 对 ID、名称和 API 名称做精确匹配，并返回全部精确匹配项。
- CLI 的 detail/delete 必须检查结果数量为 1；数量为 0 或大于 1 时 fail closed。
- `/metadata/v1/profiles/{id}` 只接受已经解析出的 ID，返回简档详情、用户引用计数、权限关联行计数和 `relations` 明细。

`detail profile` 的 `relations` 会展开：

- `tp_sys_profile_infoset`：应用、选项卡、对象、记录类型等可见性与默认标记。
- `tp_sys_profile_field`：对象字段可见/只读设置。
- `tp_sys_profile_layout`：简档、对象、记录类型到页面布局的映射。

验收记录类型可见性时，不要只看关联行数量；必须确认目标 profile 的 `recordtype` 行 `isenable=true`、`isdefault=true`，且 `tp_sys_profile_layout` 中存在对应 `recordtypeId + layoutId`。

## CLI 用法

```bash
cloudcc get profile <projectPath> [filter]
cloudcc detail profile <projectPath> <id|name|apiName>
cloudcc create profile <projectPath> <specJson|@file>
cloudcc update profile <projectPath> <specJson|@file>
cloudcc delete profile <projectPath> <id|name|apiName>
cloudcc apply msapi <projectPath> <planId>
```

## 创建契约

`profiles.create` 默认与 setup-web `/api/profile/newProfile` 和 setup-svc `ProfileService.copyProfile` 对齐，必须提供 `copyFromId`（兼容别名 `copyFromProfileId`）。创建计划会：

1. 新建 `tp_sys_profile` 和当前语言 `tp_sys_multi_lang`。
2. 复制来源简档除 `afe0000102`、`afe01002` 外的全部 `tp_sys_profile_infoset`。
3. 复制来源简档全部 `tp_sys_profile_field` 和 `tp_sys_profile_layout`。

这表示复制来源子图及其状态，不表示扫描权限定义表并生成租户所有权限。

```json
{
  "newProfileName": "销售经理简档",
  "copyFromId": "aaa000003",
  "type": "cloudcc"
}
```

显式空白创建使用 `blank: true`。它与 `copyFromId` 互斥，且不会产生任何默认权限关系：

```json
{
  "newProfileName": "空白集成简档",
  "blank": true,
  "type": "cloudcc"
}
```

没有 `copyFromId` 且没有 `blank: true`，或同时提供两者，计划阶段即失败。

## 更新契约

`profiles.update` 是 existing-row-only 操作：每个 `infosetUpdates`、`objectPermissions`、`tabPermissions`、`systemPermissions` 或 `loginRestrictions` 元素必须携带已有 `id`/`profileInfosetId`。服务端校验该行确实属于目标 profile，然后只生成 `UPDATE`。

允许修改的状态列为：

- `isenable` / `enabled`
- `appState`
- `tabState`
- `objOperateType` / `objectOperateType` / `operateType` / `crud`
- `assignDispatch`
- `ismobiletab` / `mobileTab`

关系身份和内容字段不可通过 update 修改：`profileId`、`category/infoCategory`、`relateId/relatedId`、信息集 `description` 均被拒绝。缺失行不会 upsert。

旧 setup-web 保存载荷中的 `appEnable`、`appState`、`tab`、`tabMoblieInfo`、`permission`、`objPermission`、`appMainpageid`、`aboutmeid`、`aboutmeremark`、`mobilecolleagueshowmanage`、`mailchimp`、`assignDispatch` 在 update 中会明确报错，不能再被静默忽略。先读取 detail 中的关系 ID，再提交结构化 existing-row update。

字段权限、布局分配、记录类型权限和权限定义不是本 update 的新增入口；使用相应领域命令维护。

历史 URI 编码 JSON 查询仍可作为单参数传入；CLI 会从 `selector`、`id`、`profileId`、`apiName`、`profilename`、`profileName`、`name` 或 `filter` 中提取明确值。新脚本建议直接传普通文本参数。

## 删除保护

MetadataService 对 profiles delete 执行两阶段保护：

1. 创建 plan 前验证目标 ID 存在、不是 `aaa000001`，且没有 `tp_sys_user.profile_id` 引用。
2. apply 的数据库事务内锁定目标简档行和对应用户引用范围，再次验证；失败时 operation 标记为 `FAILED`，不执行任何删除 mutation。

删除计划按以下顺序清理：

1. `tp_sys_profile_infoset`
2. `tp_sys_profile_field`
3. `tp_sys_profile_layout`
4. `tp_sys_multi_lang`
5. `tp_sys_profile`

不要绕开 CLI/MetadataService 直接拼接 SQL，也不要在重名情况下自动选择第一条记录。

## 验证建议

```bash
# 只读验证
cloudcc get profile .
cloudcc detail profile . <unique-id>

# 计划验证；不要立即 apply
cloudcc delete profile . <disposable-profile-id>

# 在明确授权的可丢弃窗口执行后审计
cloudcc apply msapi . <planId>
cloudcc changes msapi . <operationId>
cloudcc rollback-plan msapi . <operationId>
```

真实租户写入必须使用可丢弃简档，并先确认没有用户引用。系统管理员简档只应验证“被拒绝”，不得尝试绕过保护。
