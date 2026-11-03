# CloudCC 公用小组 CLI 完整指南

## 1. 适用范围与安全模型

本指南适用于 MetadataService `1.1.98+` 和技能 `2.2.116+`。CLI 资源名为 `publicGroup`，服务端 Domain 为
`public-groups`。本能力复用 CloudCC 已有数据模型和 setup-service HTTP 合同，不修改 setup-web 或
setup-service 源码。

MSAPI 写入统一执行 plan → apply → detail/changes → rollback。plan 不修改租户；apply 在事务内重新检查
目标状态、嵌套图和删除引用，旧计划发生漂移时失败关闭。

## 2. 读取、唯一选择器与候选

```bash
cloudcc capabilities publicGroup .
cloudcc get publicGroup .
cloudcc get publicGroup . 销售
cloudcc detail publicGroup . afc-sales
cloudcc detail publicGroup . 华东销售组
cloudcc candidates publicGroup . user "" 张三
cloudcc candidates publicGroup . role "" 销售
cloudcc candidates publicGroup . roleAndSubordinates "" 销售
cloudcc candidates publicGroup . publicGroup 华东销售组 运营
```

列表 filter 可模糊匹配 ID 或名称。详情和写入 selector 按 ID 优先、完整名称次之；名称匹配必须唯一，
否则返回候选 ID 并停止。详情返回规范化成员，包括 `type`、`storageType`、`targetId`、`targetName` 和
`resolved`，并给出成员分类统计和引用当前小组的上级小组 ID。

查询嵌套小组候选时建议传目标小组 selector，服务端会排除会立即形成循环的候选。最终写计划仍会对完整
小组图做严格无环校验，不能把候选接口当作唯一校验。

## 3. 统一成员结构

所有写入使用 `members` 数组：

```json
{
  "members": [
    {"type": "user", "selector": "005-user-id"},
    {"type": "role", "selector": "华东销售经理"},
    {"type": "roleAndSubordinates", "selector": "渠道销售总监"},
    {"type": "publicGroup", "selector": "全国销售支持组"}
  ]
}
```

`selector` 可以是相应对象的 ID 或唯一完整名称。禁止提交 `type`（小组数据类型）、`source`、
`selecteditems`、`selectedItems`、`memberRowId`、`createById`、`createDate`、`lastModifyById`、
`lastModifyDate`，也不能在创建时指定小组 ID。这些字段由服务端管理。

## 4. 创建

`public-group-create.json`：

```json
{
  "name": "华东销售支持组",
  "useHierarchies": false,
  "members": [
    {"type": "user", "selector": "005-user-id"},
    {"type": "roleAndSubordinates", "selector": "华东销售经理"}
  ]
}
```

```bash
cloudcc create publicGroup . @public-group-create.json
cloudcc apply msapi . <planId>
cloudcc detail publicGroup . 华东销售支持组
```

MetadataService 生成 `afc` 小组 ID、`afd` 成员行 ID、固定 `type=3` 和审计字段。名称必须非空且唯一；
成员 selector 必须唯一命中现有对象。重复成员会按 `type + targetId` 去重。

## 5. 修改属性

```bash
cloudcc update publicGroup . 华东销售支持组 \
  '{"name":"华东大区销售支持组","useHierarchies":true}'
```

update 只修改显式提供的名称或层级属性，成员变更必须使用 assign。update 中出现 `members` 会返回
`public_group_members_require_assign`；无实际差异返回 `public_group_no_changes`。

## 6. 分配成员

add：

```json
{"mode":"add","members":[{"type":"user","selector":"005-user-2"}]}
```

remove：

```json
{"mode":"remove","members":[{"type":"role","selector":"华东销售经理"}]}
```

replace：

```json
{
  "mode":"replace",
  "members":[
    {"type":"user","selector":"005-user-2"},
    {"type":"publicGroup","selector":"区域运营组"}
  ]
}
```

```bash
cloudcc assign publicGroup . 华东大区销售支持组 @public-group-assignment.json
```

MSAPI 的 add/remove/replace 都先读取当前集合并只生成必要差异。嵌套小组采用完整图循环检测，直接自引用、
间接回路都会返回 `public_group_cycle`。plan 后目标成员或任何嵌套边变化，apply 返回
`public_group_state_changed` 或 `public_group_graph_changed`，应重新读取并生成计划。

Lightning UIAPI 只有完整保存语义，因此 CLI 仅接受 `mode=replace`；add/remove 会在发请求前拒绝，避免把
局部列表误当成全量列表。需要增量差异时将项目 `executionMode` 设为 `msapi` 或 `auto` 且配置可用的
MetadataService。Horizontal UIAPI 当前无已维护公用小组合同，始终失败关闭。

## 7. 删除保护

```bash
cloudcc detail publicGroup . 华东大区销售支持组
cloudcc delete publicGroup . 华东大区销售支持组
cloudcc apply msapi . <planId>
```

删除前必须先解除：

- 其他公用小组通过 `publicGroup` 成员对它的引用；
- 共享规则中作为 owner 或 shared-with 目标的引用。

plan 和 apply 都执行检查；apply 还锁定相关状态并复核。存在引用时分别返回
`public_group_nested_reference` 或 `public_group_sharing_reference`，不会级联删除其他配置。

## 8. 计划、回读、changes 与回滚

```bash
cloudcc get msapi . <planId>
cloudcc apply msapi . <planId>
cloudcc operation msapi . <operationId>
cloudcc changes msapi . <operationId>
cloudcc rollback-plan msapi . <operationId>
cloudcc rollback msapi . <rollbackPlanId>
```

创建后确认 ID、名称和成员；update 后确认未提交字段保持不变；assign 后确认成员差异；删除后确认 detail
返回 404。回读异常时先查看 changes，不要重复 apply 同一旧计划。

UIAPI 是直接 setup-service 调用，不提供 MetadataService 的服务器 plan、changes 或 rollback 语义；成功响应后
仍应执行 `detail publicGroup` 回读，并按组织变更流程保留操作前快照。

## 9. 常见错误与处理

| 错误码或提示 | 含义 | 处理 |
|---|---|---|
| `public_group_selector_required` | 缺少 selector | 提供 ID 或唯一完整名称 |
| `public_group_selector_ambiguous` | 名称命中多个小组 | 使用返回的小组 ID |
| `public_group_not_found` | 小组不存在 | 重新列表并修正 selector |
| `public_group_name_exists` | 名称冲突 | 使用唯一名称 |
| `public_group_member_type_invalid` | 成员类型不在四类中 | 使用 user/role/roleAndSubordinates/publicGroup |
| `public_group_member_not_found` | 成员 selector 不存在 | 先用 candidates 查找 |
| `public_group_managed_field_unsupported` | 提交了服务端管理字段 | 删除 ID/type/source/selecteditems/审计字段 |
| `public_group_cycle` | 嵌套小组形成直接或间接循环 | 调整嵌套关系后重新规划 |
| `public_group_state_changed` | plan 后目标属性或成员变化 | 重新 detail 并生成计划 |
| `public_group_graph_changed` | plan 后嵌套图变化 | 重新读取候选并生成计划 |
| `public_group_nested_reference` | 仍被其他小组包含 | 先从上级小组移除 |
| `public_group_sharing_reference` | 仍被共享规则引用 | 先修改或删除共享规则 |
| `public_group_no_changes` | 请求与当前状态相同 | 无需执行 |
| `mode=replace only` | UIAPI 不支持安全增量成员操作 | 使用 replace 或切换 MSAPI |
| `public_group_horizontal_uiapi_unsupported` | horizontal 无维护合同 | 使用 MSAPI |

## 10. 建议操作清单

1. `capabilities` 确认后端版本和动作。
2. `get/detail/candidates` 获取 ID 与当前成员，避免凭名称猜测。
3. 使用统一 `members` 结构，不提交底层字段。
4. MSAPI 审阅 plan 的目标、SQL 步骤、风险和警告后再 apply。
5. apply 后执行 detail 和 changes；UIAPI 成功后也必须 detail 回读。
6. 删除前清理嵌套小组和共享规则引用。
7. 发生漂移或循环错误时重新读取并重新规划，不修改计划绕过检查。
