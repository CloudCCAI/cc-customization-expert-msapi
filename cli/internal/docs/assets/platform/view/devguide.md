# CloudCC 对象列表视图开发指南

本指南适用于 MetadataService `1.2.8+`、CLI Skill `2.3.8+` 的 Object View Contract 2。列表视图指对象记录列表中的“全部、我的、队列、最近记录”等视图，不是 setup-svc 名称相近但合同不完整的旧 `/api/view/**` 接口。

## 1. 预检和命令

```bash
cloudcc capabilities view <projectPath> <object>
cloudcc list-v2 view <projectPath> <object>
cloudcc detail-v2 view <projectPath> <viewId> <object>
cloudcc candidates view <projectPath> <object> fields
cloudcc candidates view <projectPath> <object> sortFields
cloudcc candidates view <projectPath> <object> conditionOperators <fieldId>
cloudcc candidates view <projectPath> <object> queues
cloudcc candidates view <projectPath> <object> visibilityMembers
cloudcc candidates view <projectPath> <object> buttons
```

`object` 可用对象 ID、API 名或前缀。读取结果包含 `contractVersion: 2`、`provider`、规范化 `object` 和 revision/fingerprint。`msapi` 由 MetadataService 直接读取、计划和事务写入元数据库；`uiapi` 调用项目 `apiSvc`，由 api-service 转发 main-api。旧 `cloudcc get/detail view` 仍是 Contract 1，不要混用 spec。

`conditionOperators` 必须带一个当前对象中可筛选的 `fieldId`，因为运算符由字段类型决定；其他候选 section 不接受额外字段参数。

## 2. 创建属性

| 属性 | 必填/默认 | 说明 |
| --- | --- | --- |
| `viewContractVersion` | 必填，固定 `2` | 启用 Contract 2。 |
| `objectId` | 创建必填 | 目标对象 ID。 |
| `id` | 可选，自动生成 | 1–80 位字母、数字、`_`、`-`；不能为虚拟 ID `0..4`。 |
| `label` | 创建必填 | 非空显示名称。 |
| `apiName` | 可选，自动生成 | 字母开头，仅字母、数字、`_`，对象内不区分大小写唯一。 |
| `filterScope` | 默认 `{"type":"all"}` | `all`、`mine` 或带 `queueId` 的 `queue`。 |
| `visibility` | 默认 `{"type":"private"}` | `private`、`allUsers` 或带 typed IDs 的 `members`。 |
| `fields` | 创建必填且非空 | 有序字段 ID；只用 `fields` 中 `eligible=true` 的项。 |
| `conditions` | 可选 | `expression` 加有序 `items`。 |
| `sort` | 可选 | `fieldId` 和 `asc`/`desc`。 |
| `buttonIds` | 可选，默认无 | 有序按钮 ID。 |
| `mobileEnabled` | 默认 `false` | 是否应用于移动端。 |

最小创建 spec：

```json
{
  "viewContractVersion": 2,
  "objectId": "account_object_id",
  "label": "全部客户",
  "fields": ["account_name_field_id"]
}
```

```bash
cloudcc plan msapi <projectPath> object-views @view.json create
cloudcc apply msapi <projectPath> <planId>
cloudcc detail-v2 view <projectPath> <savedViewId> <object>
```

## 3. 候选合同

候选是写入 spec 的权威 ID 来源。字段候选结合对象归属、字段状态和当前认证用户的显式字段可见性返回：

```json
{
  "id": "field_id",
  "apiName": "name",
  "label": "客户名称",
  "type": "Z",
  "effectiveType": "N",
  "displayable": true,
  "sortable": true,
  "filterable": true,
  "eligible": true,
  "exclusionReason": ""
}
```

- `fields` 和 `sortFields` 返回字段及能力标志，只提交目标 section 中 `eligible=true` 的项。
- 条件字段也从 `fields` 响应选择，但判断标准是 `filterable=true`，而不是显示列使用的 `eligible`；选定后再按 fieldId 查询运算符。
- 公式字段 `type=Z` 按 `effectiveType` 选择运算符。
- `exclusionReason` 可为 `field_inactive`、`field_not_visible`、`field_not_displayable`、`field_not_sortable`。
- 聚合字段 `A`、地理位置 `LT` 不可作显示/排序列；`LT` 仍可按 `inRange` 筛选。
- `queues`、`buttons` 返回 `id`、`label`、`apiName`；`visibilityMembers` 返回 `id`、`selectorId`、`type`、`label`。

## 4. 记录范围和可见性

```json
{"filterScope":{"type":"all"}}
```

```json
{"filterScope":{"type":"mine"}}
```

```json
{"filterScope":{"type":"queue","queueId":"queue_candidate_id"}}
```

`queueId` 必须来自目标对象 `queues` 候选。可见性为 `private`、`allUsers` 或 `members`：

```json
{
  "visibility": {
    "type": "members",
    "memberIds": ["User:user_id", "Role:role_id", "RoleAndSub:role_id", "Group:group_id"]
  }
}
```

`members` 至少一个且不可重复。必须使用 `visibilityMembers` 返回的完整 typed ID；允许前缀为 `User:`、`Role:`、`RoleAndSub:`、`Group:`，不可填裸数据库 ID。`private`/`allUsers` 使用空 `memberIds`。

## 5. 字段、排序和按钮

`fields` 顺序就是列顺序。创建时必须非空；更新时省略表示保留，传 `[]` 非法。排序示例：

```json
{"sort":{"fieldId":"created_at_field_id","direction":"desc"}}
```

排序字段必须为 `sortFields` 中 `eligible=true`，`direction` 只能是 `asc` 或 `desc`。`buttonIds` 只接受目标对象候选；更新时 `buttonIds: []` 表示明确清空。

## 6. 条件、运算符和值

条件项必须有唯一正整数 `sequence`、字段 ID、友好运算符 ID和值：

```json
{
  "conditions": {
    "expression": "1 AND (2 OR 3)",
    "items": [
      {"sequence":1,"fieldId":"name_field_id","operator":"startsWith","value":"A"},
      {"sequence":2,"fieldId":"amount_field_id","operator":"greaterOrEqual","value":1000},
      {"sequence":3,"fieldId":"active_field_id","operator":"eq","value":true}
    ]
  }
}
```

两个及以上条件必须显式提供 `expression` 并恰好引用全部 sequence；只允许数字、括号、`AND`、`OR`。一个条件可用空 expression；零条件用 `{"expression":"","items":[]}`。每个字段都应先执行：

```bash
cloudcc candidates view <projectPath> <object> conditionOperators <fieldId>
```

| 字段 `effectiveType` | 友好运算符 | 值 |
| --- | --- | --- |
| 布尔 `B` | `eq`, `ne` | `true`/`false` |
| 数字/日期/时间 `P,c,D,F,T,N,SCORE` | `eq`, `ne`, `lt`, `gt`, `lessOrEqual`, `greaterOrEqual` | 类型相符的标量 |
| 地理位置 `LT` | `inRange` | `{"value":"中心或起点","endValue":"范围或终点"}` |
| 文本/选择/关系等 `U,C,ct,Y,MR,AD,H,E,R,encd,S,X,A,J,L,Q,M,V` | 上述六个比较运算符，加 `contains`, `notContains`, `startsWith` | 字符串或相应标量 |

候选同时返回 `storageCode` 和 `valueSchema`。存储码只用于诊断/回读：`eq=e`、`ne=n`、`lt=l`、`gt=g`、`lessOrEqual=m`、`greaterOrEqual=h`、`contains=c`、`notContains=k`、`startsWith=s`、`inRange=o`。新 spec 始终写友好运算符。
选择列表应提交选项存储 key，关系字段应提交目标记录 ID；日期、时间和数字应使用目标字段可接受的规范标量，不能用页面显示标签代替存储值。

## 7. 完整示例

“我的客户”、成员共享、多条件：

```json
{
  "viewContractVersion": 2,
  "objectId": "account_object_id",
  "label": "我的重点客户",
  "apiName": "my_priority_accounts",
  "filterScope": {"type":"mine"},
  "visibility": {"type":"members","memberIds":["RoleAndSub:sales_role_id"]},
  "fields": ["name_field_id","phone_field_id","owner_field_id"],
  "conditions": {
    "expression": "1 AND 2",
    "items": [
      {"sequence":1,"fieldId":"level_field_id","operator":"eq","value":"A"},
      {"sequence":2,"fieldId":"amount_field_id","operator":"greaterOrEqual","value":100000}
    ]
  },
  "sort": {"fieldId":"created_at_field_id","direction":"desc"},
  "buttonIds": ["export_button_id"],
  "mobileEnabled": true
}
```

队列视图：

```json
{
  "viewContractVersion": 2,
  "objectId": "case_object_id",
  "label": "一线支持队列",
  "filterScope": {"type":"queue","queueId":"support_queue_id"},
  "visibility": {"type":"allUsers","memberIds":[]},
  "fields": ["case_number_field_id","subject_field_id","status_field_id"],
  "conditions": {"expression":"","items":[]},
  "sort": {"fieldId":"created_at_field_id","direction":"asc"},
  "buttonIds": []
}
```

地理位置条件：

```json
{
  "viewContractVersion": 2,
  "objectId": "site_object_id",
  "label": "服务范围内站点",
  "fields": ["site_name_field_id"],
  "conditions": {
    "expression": "",
    "items": [
      {
        "sequence": 1,
        "fieldId": "location_field_id",
        "operator": "inRange",
        "value": {"value":"31.2304,121.4737","endValue":"10km"}
      }
    ]
  }
}
```

## 8. 更新、复制、删除、默认和顺序

更新是 patch：属性省略表示保留；集合显式提供表示完整替换。`conditions.items: []`、`buttonIds: []` 表示清空，`fields: []` 非法。更新/删除必须带 `detail-v2` 的最新 `revision`：

```json
{
  "viewContractVersion": 2,
  "id": "stored_view_id",
  "objectId": "account_object_id",
  "revision": "revision_from_detail_v2",
  "label": "重点客户（新版）",
  "buttonIds": [],
  "conditions": {"expression":"","items":[]}
}
```

复制默认产生 private 新视图：

```json
{
  "viewContractVersion": 2,
  "sourceViewId": "stored_view_id",
  "newLabel": "重点客户副本",
  "apiName": "priority_accounts_copy"
}
```

```bash
cloudcc plan msapi <projectPath> object-views @view-update.json update
cloudcc plan msapi <projectPath> object-views @view-copy.json copy
cloudcc plan msapi <projectPath> object-views @view-delete.json delete
cloudcc set-default view <projectPath> <objectId> <viewId>
cloudcc reorder view <projectPath> <objectId> "<viewId1>;<viewId2>;0;1;2"
cloudcc reorder-public view <projectPath> <objectId> "<viewId1>;<viewId2>;0;1;2"
```

删除前按影响分析处理 dashboard、默认值和顺序引用。默认/顺序是用户偏好，不属于 update spec。`reorder-public` 需要管理权限。虚拟 ID `0..4` 永远只读；除虚拟视图外，满足权限的已存储视图均可编辑。动态虚拟 ID `3`、`4` 的出现状态由 api-service 解析，状态不可达时 MSAPI 顺序替换失败关闭。

## 9. Provider 差异

MSAPI 在计划事务中替换 owned rows，支持 `buttonIds: []`。UIAPI 的所有列表视图操作调用 `apiSvc`；当前 main-api `saveView` 在 `customButtons` 为空时不会清旧按钮，所以 UIAPI 对该操作失败关闭。UIAPI 创建/更新/复制必须详情回读，删除/默认/顺序必须列表或偏好回读；保存 ID 缺失或语义不一致返回 `view_remote_result_unknown`。

## 10. 常见错误

| 错误码 | 处理 |
| --- | --- |
| `view_field_not_found` | fieldId 不属于目标对象；重读候选。 |
| `view_field_not_filterable` | 字段对当前用户不可筛选或类型不支持。 |
| `view_field_invalid` | 显示字段重复、不属于对象或 `eligible=false`。 |
| `view_sort_invalid` | 排序字段不可排序或方向非法。 |
| `view_condition_invalid` | sequence、expression、运算符或值错误。 |
| `view_queue_invalid` | queueId 不属于目标对象。 |
| `view_visibility_member_invalid` | typed ID 格式、存在性或唯一性错误。 |
| `view_api_name_invalid` / `view_api_name_duplicate` | 修正格式或重名。 |
| `view_revision_required` / `view_revision_conflict` | 重读详情并使用最新 revision。 |
| `view_candidate_guard_unavailable` | 目标库缺少验证结构；停止写入并检查迁移。 |

## 11. 审计、回滚和上线检查

```bash
cloudcc changes msapi <projectPath> <operationId>
cloudcc rollback msapi <projectPath> <operationId>
cloudcc doc view introduction
cloudcc doc object-views devguide
```

上线前确认：候选来自目标对象和当前用户；每个条件字段单独查询运算符；计划无虚拟视图写入；apply 后用 `detail-v2` 校验列顺序、范围、可见性、条件、排序、按钮和移动端标志；保留 operationId 供审计/回滚。
