# CloudCC 简档使用总结

简档（Profile）定义用户的应用、菜单、对象、字段、记录类型、布局、系统功能和登录策略等权限。简档属于低代码元数据，CLI 通过 MetadataService 查询和计划执行，不再用 `standard-catalog` 猜测简档数据。

## 可用命令

| 操作 | 命令 | 行为 |
|------|------|------|
| 列表 | `cloudcc get profile <projectPath> [filter]` | 只读查询 `tp_sys_profile`，可按 ID、名称、API 名称或描述过滤 |
| 详情 | `cloudcc detail profile <projectPath> <id\|name\|apiName>` | 先唯一解析选择器，再按 ID 读取详情和引用计数 |
| 创建/更新 | `cloudcc create\|update profile <projectPath> <specJson\|@file>` | 只创建 MetadataService plan |
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

*文档版本：2.1.279 | 最后更新：2026-07-14*
