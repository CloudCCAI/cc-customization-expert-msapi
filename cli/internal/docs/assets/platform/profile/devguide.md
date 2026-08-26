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
