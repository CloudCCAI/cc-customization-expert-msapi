# CloudCC 队列 CLI 完整指南

## 1. 范围、版本与平台标准

本指南适用于 MetadataService `1.2.3+`、技能 `2.3.3+`。CLI 资源为 `queue`，服务端 Domain 为
`queues`。它管理可作为业务记录 owner 的普通队列，不管理 MQ、审批队列、Bulk job、公海池或记录转移。

MSAPI 的队列数据全部位于目标租户关系库。MetadataService 在单一事务中维护 `tp_sys_queue`、
`tp_sys_queue_tab`、`tp_sys_queue_member`、生成的 `tp_sys_view` 和 `tp_sys_view_field`，并在 apply 前后执行
聚合漂移、owner 引用和完整回读校验。Lightning UIAPI 仍调用 `/api/queueSetup/*`；两条 Provider 路径不混用。

## 2. Provider 能力边界

| Provider | 读取 | 写入 | 安全语义 |
|---|---|---|---|
| MSAPI | 完整 | create/update/assign/delete | plan-first、漂移检查、回读、changes、rollback |
| Lightning UIAPI | 完整 | 标准服务直写；assign 仅 replace | 每次写后 query 回读；无服务端变更台账 |
| Horizontal UIAPI | 无 | 无 | 失败关闭，使用 MSAPI |

MSAPI 是推荐路径。UIAPI 不会伪装成具备计划台账；Lightning 的 `add/remove` 因缺少可靠原子合并合同而
拒绝，避免并发覆盖。Universal provider 根据项目配置选路，不得从 horizontal 静默回退到 Lightning。

## 3. 读取和唯一选择器

```bash
cloudcc capabilities queue <projectPath>
cloudcc get queue <projectPath> [filter]
cloudcc detail queue <projectPath> <queueId|apiName>
```

`filter` 可匹配 ID、apiName 或 label。写 selector 严格按精确 ID、唯一 apiName 解析；label 不用于自动写
选择，避免重名误操作。详情中的规范字段为：

```json
{
  "queueId": "que-example",
  "apiName": "ServiceCaseQueue",
  "label": "服务请求队列",
  "email": "service@example.invalid",
  "sendEmailToMembers": true,
  "description": "服务请求统一入口",
  "supportedObjects": [{"id":"account","apiName":"account","label":"客户","prefix":"001"}],
  "members": [{"type":"roleAndSubordinates","id":"role-service","label":"服务经理"}]
}
```

ID、关系行 ID、审计字段、`marketseaid` 和生成视图字段由平台管理，不能作为写入字段提交。

## 4. 候选发现

```bash
cloudcc candidates queue . objects [keyword]
cloudcc candidates queue . members user [keyword]
cloudcc candidates queue . members role [keyword]
cloudcc candidates queue . members roleAndSubordinates [keyword]
```

创建和分配前必须使用候选。对象和成员 selector 可以使用候选 ID 或唯一 API/显示名称，最终都会被解析为
平台 ID。当前不支持 `publicGroup`；也不要提交 setup-service 数字类型 `1/2/3/4`。

## 5. 创建

`queue-create.json`：

```json
{
  "apiName": "ServiceCaseQueue",
  "label": "服务请求队列",
  "email": "service@example.invalid",
  "sendEmailToMembers": true,
  "description": "服务请求统一入口",
  "supportedObjects": ["Case"],
  "members": [
    {"type":"roleAndSubordinates","selector":"服务经理"},
    {"type":"user","selector":"usr-example"}
  ]
}
```

```bash
cloudcc create queue . @queue-create.json
cloudcc apply msapi . <planId>
```

规则：`apiName` 匹配 `^[A-Za-z][A-Za-z0-9_]{0,39}$`；label 非空；email 为空或合法；
`sendEmailToMembers` 必须是 JSON boolean；对象、成员各至少一个；重复自然键被拒绝；调用方不能指定 ID。
MetadataService 在计划中预留平台格式的稳定 queue ID，使写后回读和删除补偿能精确定位同一聚合。

## 6. 更新基础属性

`queue-update.json`：

```json
{
  "label": "服务请求一级队列",
  "email": "",
  "sendEmailToMembers": false,
  "description": ""
}
```

```bash
cloudcc update queue . ServiceCaseQueue @queue-update.json
```

允许 `apiName`、`label`、`email`、`sendEmailToMembers`、`description` 和可选 `supportedObjects`。省略表示
保留；显式传入 `supportedObjects` 表示完整替换，并仍要求至少一个对象。只有 email 和 description 可用
空字符串清空；`null` 被拒绝。成员增量继续使用 `assign`。相同状态返回 `queue_no_changes`，不会生成空计划。

## 7. 分配对象和成员

`queue-assign.json`：

```json
{
  "objects": {"mode":"replace","values":["Case","Lead"]},
  "members": {
    "mode":"add",
    "values":[{"type":"role","selector":"一线支持"}]
  }
}
```

```bash
cloudcc assign queue . ServiceCaseQueue @queue-assign.json
```

`objects`、`members` 可单独出现；mode 只能是 `add/remove/replace`。服务端先读取完整当前聚合，再合并为
完整 desired state 并编译最小关系差异，防止遗漏字段。最终对象或成员集合为空会分别返回
`queue_empty_objects`、`queue_empty_members`。MSAPI 支持三种 mode；Lightning UIAPI 仅允许 replace。
也就是说，Lightning 自动化必须显式使用 `mode=replace`，增量模式应切换到 MSAPI。

对象范围变化时，MetadataService 保留未变化的关系和视图 ID，只为新增对象创建关系、视图和默认字段，
并完整删除已移除对象的生成视图子图。自动化仍应验证业务语义，不依赖新增资源的物理 ID。
如果被移除对象仍有 `ownerid=queueId` 的业务记录，计划或 apply 返回
`queue_object_remove_owner_reference`；应先通过 OpenAPI 明确转移 owner，再重新规划。

## 8. 删除与 owner 阻断

```bash
cloudcc impact queue . ServiceCaseQueue delete
cloudcc delete queue . ServiceCaseQueue
cloudcc apply msapi . <planId>
```

删除计划和 apply 都在租户关系库执行 `checkBeforeDeleting` 等价保护。只要任一支持对象仍有 `ownerid=queueId` 的
业务记录，就返回 `queue_delete_owner_reference`；不会隐式转移、删除或级联修改业务记录。请先通过标准
业务数据/OpenAPI 明确转移 owner，再重新运行 impact 和创建新计划。旧计划不得在转移后直接复用。

MSAPI 删除会在同一事务中删除生成视图子表、视图、对象/成员关系和队列根。删除为 destructive risk，
必须显式允许破坏性 apply；任何步骤或回读不一致都会回滚事务，不会留下部分删除。

## 9. 计划、漂移、回读与回滚

典型流程：

```bash
cloudcc create queue . @queue-create.json
# 审阅 planId、risk、beforeStateHash、desiredStateHash 和完整 desiredState
cloudcc apply msapi . <planId>
cloudcc changes msapi . <operationId>
cloudcc rollback-plan msapi . <operationId>
cloudcc rollback msapi . <operationId>
```

apply 前重新计算状态 hash。队列属性、对象或成员在 plan 后变化会返回 `queue_state_changed`，应重新
detail、重新合并意图并生成计划。所有 SQL 步骤与写后语义回读位于同一事务；回读还校验每个对象恰有一个
生成视图、访问成员集合以及默认字段顺序。任何不一致返回 `queue_readback_mismatch` 并回滚。

回滚继续由 MetadataService change ledger 管理，并受目标状态保护；创建、更新、分配和删除的每一条关系库
变化都有对应快照。旧版本生成的 setup-service Queue 计划必须重新规划，返回 `queue_plan_backend_obsolete`。

## 10. 错误码与处理

| 错误码 | 含义 | 处理 |
|---|---|---|
| `queue_selector_required` | 缺少 selector | 提供 queue ID 或唯一 apiName |
| `queue_not_found` | 队列不存在 | 重新 get，确认普通队列及环境 |
| `queue_selector_ambiguous` | selector 不唯一 | 使用 queue ID |
| `queue_spec_invalid` | 字段、类型、格式或 null 错误 | 按本文 JSON 修正 |
| `queue_object_not_candidate` | 对象不在标准候选中 | 先运行 objects candidates |
| `queue_member_type_unsupported` | 成员类型未支持 | 使用 user/role/roleAndSubordinates |
| `queue_member_not_candidate` | 成员不可用或不唯一 | 用 candidates 获取 ID |
| `queue_empty_objects` | 最终对象为空 | 至少保留一个对象 |
| `queue_empty_members` | 最终成员为空 | 至少保留一个成员 |
| `queue_no_changes` | 规范化后无差异 | 无需 apply |
| `queue_state_changed` | plan 后状态变化 | 重新读取并规划 |
| `queue_delete_owner_reference` | 队列仍拥有记录 | 先显式转移 owner |
| `queue_object_remove_owner_reference` | 待移除对象仍有队列 owner 记录 | 先显式转移该对象记录 owner |
| `queue_readback_mismatch` | 写后状态未收敛 | 不要盲重试；检查 changes/当前详情 |
| `queue_object_relation_invalid` | 历史对象关系缺少 ID/前缀 | 审计并显式修复，不按名称猜测 |
| `queue_plan_backend_obsolete` | 旧远程 Queue 计划 | 用当前服务重新生成计划 |
| `queue_horizontal_uiapi_unsupported` | horizontal 无维护合同 | 使用 MSAPI |

## 11. 排障清单

1. `cloudcc doctor target .`，确认 provider/platform 与 token。
2. `cloudcc capabilities queue .`，确认 MetadataService `1.2.3+` 且 `executionBackend=tenant-relational-db`。
3. 重新运行 get/detail/candidates，禁止复用旧候选或旧 plan。
4. 检查 JSON boolean、空字符串与 null，确认未提交 ID、关系行、视图或审计字段。
5. 删除失败先运行 impact；不要通过数据库手工改 owner 或直接删队列表。
6. readback mismatch 时先查 changes 和标准设置页，不立即重复执行同一写命令。
7. horizontal 项目明确切换 MSAPI；不要把 Lightning setup-svc 地址塞进 horizontal 配置绕过边界。

## 12. 自动化与最小权限建议

自动化应固定使用 queue ID 或唯一 apiName，保存 plan/operation ID，审阅 destructive risk，并把 candidates、
impact、apply、readback 分成可观测步骤。访问令牌仅授予读取队列、候选以及必要的队列管理权限；日志和
fixture 禁止保存 accessToken、Cookie、真实邮箱或客户数据。测试使用虚构 ID 和 `.invalid` 邮箱。
