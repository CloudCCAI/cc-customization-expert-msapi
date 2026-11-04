# CloudCC 角色 CLI 完整指南

## 1. 适用范围

本指南适用于 MetadataService `1.1.100+` 和技能 `2.2.119+`。角色使用 CLI 一级资源名 `role`，服务端
Domain 为 `roles`。MSAPI 写入遵循 plan → apply → readback/changes → rollback，不直接修改租户。

## 2. 读取与发现

```bash
cloudcc capabilities role .
cloudcc get role .
cloudcc get role . 销售
cloudcc detail role . role-sales-manager
cloudcc detail role . 销售经理
cloudcc candidates role . createParent
cloudcc candidates role . moveParent role-sales-manager
cloudcc candidates role . assignmentFallback role-sales-manager
```

列表 filter 可以模糊匹配 ID、名称和父角色 ID。详情和所有写操作使用精确 selector：ID 优先，名称必须
唯一。详情包含规范化的 `role`、`parent`、`ancestors`、`children`、`assignment.userCount/userIds`；
兼容的 allowlist relations 仅用于诊断，不应作为写入模板复制。

`moveParent` 候选会排除目标自身和目标后代，并在 `exclusionReasons` 中说明原因。

## 3. 创建角色

`role-create.json`：

```json
{
  "name": "华东销售经理",
  "parent": {"selector": "全国销售总监"}
}
```

```bash
cloudcc create role . @role-create.json
cloudcc apply msapi . <planId>
cloudcc detail role . 华东销售经理
```

MetadataService 读取父角色完整祖先链，生成 self row、全部 ancestor rows、正确 gap/level 和两条配套角色组。
角色名称允许重复；后续写入应保存返回的目标 ID，只有唯一名称才能作为 selector。已有角色体系中完全
省略父角色返回 `role_parent_required`，父角色不存在返回 `role_not_found`。父链必须包含从 self 到根角色
的连续 closure；缺行、重复 ancestor、非法 gap 或 level
不一致会返回 `role_hierarchy_invalid`，不得绕过后继续创建。

创建计划会绑定父角色完整祖先链。plan 后父链发生变化时 apply 返回 `role_hierarchy_changed`，应重新读取
父角色并生成新计划，不能应用旧 closure。

禁止提交：`rowId`、`parentClosure`、`ancestors`、`closureRows`、`gap`、`currentLevel`、`columns`。

### 3.1 显式创建根角色

已有角色时，只有显式 `parent:null` 才创建新的根角色：

```json
{"name":"区域 CEO","parent":null}
```

省略 parent 仍失败，避免字段漏传时意外改变角色树。根角色不可 move 或 delete。

### 3.2 外部迁移身份与重试

```json
{
  "name":"销售经理",
  "sourceSystem":"salesforce",
  "externalKey":"00E-source-child",
  "parent":{"sourceSystem":"salesforce","externalKey":"00E-source-parent"}
}
```

MetadataService 用规范化 sourceSystem 和 externalKey 确定性生成目标角色 ID；计划 metadata 的
`externalIdentity.targetRoleId` 是源角色到目标角色的稳定映射。CLI 同时生成稳定 idempotency key，相同
请求重试返回原计划。父 externalKey 对应角色必须已经创建，因此批量迁移应按父先于子的拓扑顺序执行。

不同 externalKey 可以使用相同名称，不会按名称合并。若该外部身份的目标 ID 已存在但请求没有命中原
幂等计划，返回 `role_external_identity_exists`，其 `details.targetRoleId` 可用于恢复映射；不得改名后重试
制造第二个角色。

## 4. 重命名角色

```bash
cloudcc update role . 华东销售经理 '{"name":"华东大区销售经理"}'
```

update 只处理非结构属性并同步 closure rows 和配套角色组名称。相同名称返回 `role_no_changes`。任何
`parent/parentRoleId` 字段返回 `role_parent_change_requires_move`，父级调整必须使用 move。

## 5. 移动角色子树

```bash
cloudcc candidates role . moveParent 华东大区销售经理
cloudcc move role . 华东大区销售经理 渠道销售总监
```

move 会把目标角色及其全部后代作为一个子树重新计算 closure rows 和 level。计划会列出每个受影响角色
的旧 closure 删除和新 closure 写入。根角色、移到自身/后代、当前父级不变、现有层级循环分别返回
`role_system_managed`、`role_move_cycle`、`role_no_changes`、`role_hierarchy_cycle`。

move 风险等级为 destructive；SQL 删除步骤带预期行数，计划后 closure 数量变化会失败并回滚事务。
计划还保存完整受影响子树以及新父角色祖先链的层级指纹；apply 前发现父级、gap、level 或 closure 行发生变化时返回
`role_hierarchy_changed`，必须重新读取并生成新计划。

## 6. 用户分配

add：

```json
{"mode":"add","users":["user-001","user-002"]}
```

remove：

```json
{"mode":"remove","users":["user-001"],"removedUserFallbackRole":{"selector":"普通员工"}}
```

replace：

```json
{"mode":"replace","users":["user-002","user-003"],"removedUserFallbackRole":{"selector":"普通员工"}}
```

```bash
cloudcc assign role . 华东大区销售经理 @role-assignment.json
```

`add` 只增加差异；`remove` 只迁移当前属于目标角色的指定用户；`replace` 读取当前集合后计算增删。
发生移除时 fallback 必填且不能等于目标角色。空差异返回 `role_no_changes`。

计划会记录相关用户的原始角色集合，并在 apply 事务中按稳定顺序锁定用户行后复核；任一用户在 plan 后被其他操作重新分配时返回
`role_assignment_changed`。目标角色和 fallback 角色也会在事务内重新校验并锁定；角色已删除时返回
`role_assignment_role_changed`。不要重复应用旧计划，应重新读取用户分配后重新规划。不存在的用户返回
`role_assignment_user_not_found`，不会等到 SQL 阶段才静默忽略。

## 7. 删除角色

```bash
cloudcc detail role . <selector>
cloudcc delete role . <selector>
cloudcc apply msapi . <planId>
```

计划和 apply 都会检查角色存在性、根角色、后代和已分配用户，错误分别为 `role_not_found`、
`role_system_managed`、`role_has_descendants`、`role_in_use`。先移动下级角色、迁移用户，再重新规划删除。

## 8. 审计、回读与回滚

```bash
cloudcc get msapi . <planId>
cloudcc apply msapi . <planId>
cloudcc operation msapi . <operationId>
cloudcc changes msapi . <operationId>
cloudcc rollback-plan msapi . <operationId>
cloudcc rollback msapi . <rollbackPlanId>
```

创建后回读父级和 ancestors；重命名后确认层级未变；move 后回读目标及后代；assign 后确认
`assignment.userIds`；删除后确认 detail 返回 404。回读异常时先检查 changes，不要重复 apply。

## 9. 常见错误

| 错误码 | 含义 | 处理 |
|---|---|---|
| `role_selector_required` | 缺少 selector | 提供 ID 或唯一完整名称 |
| `role_selector_ambiguous` | 名称命中多个角色 | 使用返回的目标角色 ID |
| `role_external_identity_exists` | externalKey 已对应一个目标角色 | 使用 `details.targetRoleId` 恢复映射 |
| `role_external_key_invalid` | externalKey 为空或过长 | 提供 1..256 字符稳定源 ID |
| `role_source_system_invalid` | sourceSystem 格式非法 | 使用小写字母、数字、点、下划线或连字符 |
| `role_managed_field_unsupported` | 提交了服务端管理字段 | 删除 closure/gap/level/rowId/columns |
| `role_parent_change_requires_move` | update 尝试改父级 | 改用 `move role` |
| `role_move_cycle` | move 形成循环 | 选择 eligible 父级 |
| `role_hierarchy_invalid` | 当前 closure 缺失、重复或层级不连续 | 先修复角色层级，不要生成写计划 |
| `role_hierarchy_changed` | create/move 计划后相关层级发生变化 | 重新读取并生成计划 |
| `role_state_changed` | rename 计划后目标角色发生变化 | 重新读取并生成 update 计划 |
| `role_assignment_fallback_required` | 移除用户缺少 fallback | 提供有效 fallback 角色 |
| `role_assignment_user_not_found` | 用户 ID 不存在 | 修正用户列表后重新规划 |
| `role_assignment_changed` | plan 后用户角色发生变化 | 重新读取分配并重新规划 |
| `role_assignment_role_changed` | 目标或 fallback 角色已删除 | 选择有效角色并重新规划 |
| `role_operation_unsupported` | operation 不是角色生命周期动作 | 使用 catalog 中公开的命令 |
| `role_no_changes` | 与当前状态相同 | 无需执行或重新读取 |

## 10. 安全边界

- 不修改 setup-service/setup-web。
- 不要求用户理解或提交 `tp_sys_role` closure table 字段。
- 不把模糊 filter 直接用于写入。
- 所有写入先审阅 plan，再显式 apply，完成后必须回读。

## 11. normalize / validate 自动化语义

```bash
cloudcc normalize msapi . roles @role-create.json create
cloudcc validate msapi . roles @role-create.json create
```

两者均执行角色业务校验，并返回 `phase`、`valid`、`errors`、结构化 `issues[]` 和 `normalizedSpec`。
每个 issue 至少含稳定 `code` 与 `message`，可能附带 `details`。当 `valid:false` 时 CLI 先把 JSON 写到
stdout，再以非零退出码结束；Shell/CI 必须检查退出码。`plan` 才会持久化计划，normalize/validate 不写
角色或计划账本。
