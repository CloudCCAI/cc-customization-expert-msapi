# CloudCC 简档使用总结

简档（Profile）定义用户的应用、菜单、对象、字段、记录类型、布局、系统功能和登录策略等权限。简档属于低代码元数据，CLI 通过 MetadataService 查询和计划执行，不再用 `standard-catalog` 猜测简档数据。

## 可用命令

| 操作 | 命令 | 行为 |
|------|------|------|
| 列表 | `cloudcc get profile <projectPath> [filter]` | 只读查询 `tp_sys_profile`，可按 ID、名称、API 名称或描述过滤 |
| 详情 | `cloudcc detail profile <projectPath> <id\|name\|apiName>` | 先唯一解析选择器，再按 ID 读取详情和引用计数 |
| 创建/更新 | `cloudcc create\|update profile <projectPath> <specJson\|@file>` | 校验来源或既有权限关系后，只创建 MetadataService plan |
| 删除 | `cloudcc delete profile <projectPath> <id\|name\|apiName>` | 唯一解析后创建受保护的 delete plan |
| 执行 | `cloudcc apply msapi <projectPath> <planId>` | 显式执行计划并记录 changes/rollback 证据 |

## 查询示例

```bash
# 所有简档
cloudcc get profile .

# 模糊过滤；参数会原样进行 URL query-component 编码
cloudcc get profile . "销售 + 服务"

# 使用 ID、名称或 API 名称读取唯一简档
cloudcc detail profile . aaa202672F656B7VfEjL
cloudcc detail profile . "销售经理简档"
cloudcc detail profile . sales_manager_profile
```

详情响应包含规范化的简档字段、`userReferenceCount`，以及 infoset、字段权限和布局分配的关联行计数。

## 创建简档

标准创建必须提供 `copyFromId`，语义与设置后台一致：新简档复制来源简档现有的全部信息集、字段权限和布局分配，同时保留每条权限的启用或关闭状态。它不是从权限定义表重新生成“系统全部权限”；来源简档缺少的关系不会凭空补齐，软件包菜单 `afe0000102`、`afe01002` 也不会复制。

```json
{
  "newProfileName": "销售经理简档",
  "copyFromId": "aaa000003",
  "type": "cloudcc"
}
```

```bash
cloudcc create profile . @profile-create.json
```

只有确实需要空白权限基线时才使用 `blank: true`。`blank: true` 和 `copyFromId` 互斥；两者都不提供会直接失败。

```json
{
  "newProfileName": "集成专用空白简档",
  "blank": true,
  "type": "cloudcc"
}
```

空白创建不会自动生成应用、菜单、对象、字段或布局权限，执行前应重点检查计划步骤。

## 更新简档权限

更新只允许修改目标简档下已经存在的信息集。先用 `detail profile` 取得 `relations.infosets` 中的现有 `id`，再按 ID 提交状态变化：

```json
{
  "id": "aaa202672F656B7VfEjL",
  "name": "销售经理简档",
  "systemPermissions": [
    {
      "id": "bba-existing-permission-row",
      "enabled": true
    }
  ],
  "objectPermissions": [
    {
      "id": "bba-existing-object-row",
      "objectOperateType": "1,1,1,1,0,0"
    }
  ]
}
```

```bash
cloudcc detail profile . aaa202672F656B7VfEjL
cloudcc update profile . @profile-update.json
```

可修改的状态字段包括 `enabled/isenable`、`appState`、`tabState`、`objectOperateType/objOperateType/crud`、`assignDispatch` 和 `mobileTab/ismobiletab`。以下请求会失败关闭：

- 信息集没有 `id`，或该 ID 不属于目标简档。
- 尝试修改 `profileId`、`infoCategory/category`、`relateId/relatedId` 或信息集 `description`。
- update 使用旧 UI 字符串字段，如 `permission`、`objPermission`、`appEnable`、`tab`。
- update 携带仅供创建使用的 `fieldPermissions`、`layoutAssignments`、`recordTypePermissions` 或 `permissionDefinitions`。

缺失的权限关系不会由 update 自动插入；应使用对应权限、字段、记录类型或布局领域的专用命令处理。

## 安全删除

```bash
# 只生成计划，不直接删除
cloudcc delete profile . sales_manager_profile

# 审核 planId 后显式执行
cloudcc apply msapi . <planId>
```

删除遵守以下硬性保护：

- ID、名称或 API 名称必须恰好匹配一条；零匹配或重名均停止，不取第一条。
- 系统管理员简档 `aaa000001` 永远不能删除。
- `tp_sys_user.profile_id` 仍有任何用户引用时不能删除。
- plan 创建时检查一次，apply 事务内对简档行和用户引用行加锁后再次检查，覆盖计划创建后的引用变化。
- 真正执行后通过 `cloudcc changes msapi <projectPath> <operationId>` 审计删除步骤；需要恢复时先生成 rollback plan。

## 与高代码源码编码修复的关系

简档能力与 classes/triggers/timer 的 Java 源码发布相互独立。当前版本继续保留 URLDecoder 兼容编码：Java 中的 `+`、`++`、`+=` 会编码为 `%2B`，不会被服务端误解为空格。

*文档版本：2.2.66 | 最后更新：2026-09-15*
