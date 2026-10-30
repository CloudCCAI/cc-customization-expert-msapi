# 查找字段筛选器 CLI 用户指南

查找筛选器限制关系字段弹窗中可选择的记录。本文从查询现状、发现候选、编写条件、生成和执行计划，一直覆盖到回读、停用、清空、删除及回滚。本文适用于 MetadataService `1.1.95+`；低版本不具备本文所述的完整操作符、类型校验和候选值契约。

## 0. 使用前检查和平台边界

```text
cloudcc --version
cloudcc doctor provider <projectPath>
cloudcc domain lookupFilter
cloudcc doc lookupFilter devguide
```

- 严格 MSAPI 包要求 MetadataService `1.1.95+`。
- Universal 包只有实际选择 MSAPI provider 时才能使用本资源。
- Lightning UIAPI 和 horizontal UIAPI 都不提供这套友好查找筛选器契约，会在发网前拒绝；不要把本文 JSON 直接改造成 setup-service 私有请求。
- `get/detail/candidates` 是只读操作；`create/update/delete` 只生成 MetadataService 计划，不会直接改租户。
- `object` 和 `field` 可使用目标环境能唯一解析的 ID、API 名等选择器；有歧义或不存在时失败，不采用第一条结果。

## 1. 命令速查

```text
cloudcc get lookupFilter <projectPath> <object>
cloudcc detail lookupFilter <projectPath> <object> <field>
cloudcc candidates lookupFilter <projectPath> <object> <field>
cloudcc create lookupFilter <projectPath> <@filter.json>
cloudcc update lookupFilter <projectPath> <@filter.json>
cloudcc delete lookupFilter <projectPath> <object> <field>
```

`lookup-filter`、`lookup-filters` 与 `lookupFilter` 等价。写命令只生成计划，检查计划后还要执行：

```text
cloudcc apply msapi <planId>
cloudcc detail lookupFilter . contact account_id
```

推荐先执行 `candidates`。它按字段返回 `eligible`、`supportedOperators`、`exclusionReason`；选项字段还会返回可写入的 `options[].key` 和可识别标签。

### 三个只读命令分别看什么

```text
cloudcc get lookupFilter . contact
cloudcc detail lookupFilter . contact account_id
cloudcc candidates lookupFilter . contact account_id
```

- `get`：列出对象上的关系字段、启用状态、逻辑表达式和条件数量。
- `detail`：读取一个关系字段的规范化规则；旧底层编码会尽可能转为友好操作符。
- `candidates`：读取源对象与查找对象的字段、逐字段操作符、排除原因以及选项键。新建或修改前应先执行。

## 2. 多值 in / notIn

一个字段命中多个状态中的任意一个，使用 `in` 和 `right.values` 数组，不要把多个值塞进 `right.value`：

```json
{
  "object": "approval_order",
  "field": "account_id",
  "lookupFilter": {
    "enabled": true,
    "logic": "1",
    "conditions": [
      {
        "left": {"scope": "lookupObject", "field": "approval_status"},
        "operator": "in",
        "right": {"values": ["审批通过", "草稿", "提交待审批", "审批拒绝"]}
      }
    ]
  }
}
```

`notIn` 写法相同，含义是“不属于数组中的任何值”。约束如下：

- `values` 必须是非空数组，最多 100 项；重复项会按原顺序去重。
- 单个值不能包含英文逗号 `,`，因为底层兼容存储使用逗号分隔。
- `in/notIn` 只开放给已验证的文本、选项和关系类字段。数字、日期、时间、复选框等字段会在计划阶段拒绝。
- 选项字段可传候选项的存储键或唯一标签；服务会归一化为存储键。标签不存在或匹配多个选项时会失败，不会猜测。
- 全局选项字段会自动解析其全局选项集；存在多个绑定时失败关闭。

## 3. 完整友好结构

```json
{
  "object": "contact",
  "field": "account_id",
  "lookupFilter": {
    "enabled": true,
    "logic": "1 AND (2 OR 3)",
    "conditions": [
      {
        "left": {"scope": "lookupObject", "field": "status"},
        "operator": "in",
        "right": {"values": ["Active", "Pending"]}
      },
      {
        "left": {"scope": "lookupObject", "field": "name"},
        "operator": "startsWith",
        "right": {"value": "Cloud"}
      },
      {
        "left": {"scope": "lookupObject", "field": "owner_id"},
        "operator": "equals",
        "right": {"scope": "sourceObject", "field": "owner_id"}
      }
    ]
  }
}
```

`scope` 只接受 `lookupObject`（被筛选的目标对象）和 `sourceObject`（持有关系字段的当前对象）。字段比较使用 `right.scope + right.field`；常量比较使用 `right.value`；集合比较使用 `right.values`，三种形式不要混用。

`logic` 中的编号从 1 开始，只能使用条件编号、`AND`、`OR`、`NOT` 和括号，并且必须覆盖全部条件。省略时默认用 `AND` 连接全部条件。最多 10 条条件。

## 4. 操作符和字段类型

最终可用项以 `cloudcc candidates lookupFilter ...` 的 `supportedOperators` 为准。

| 字段类别 | 可用友好操作符 |
|---|---|
| 文本、选项、关系及已验证的文本类字段 | `equals`、`notEquals`、`lessThan`、`greaterThan`、`lessOrEqual`、`greaterOrEqual`、`contains`、`notContains`、`startsWith`、`in`、`notIn`、`isNull`、`isNotNull` |
| 数字、币种、百分比、日期、日期时间、时间、公式 | `equals`、`notEquals`、`lessThan`、`greaterThan`、`lessOrEqual`、`greaterOrEqual`、`isNull`、`isNotNull` |
| 复选框 | `equals`、`notEquals`、`isNull`、`isNotNull` |
| 图片、文件、富文本等未验证类型 | 不可用；候选接口返回排除原因 |

空值判断不传值：

```json
{
  "left": {"scope": "lookupObject", "field": "closed_date"},
  "operator": "isNull",
  "right": {}
}
```

数值会在计划阶段校验并规范化；复选框接受 `true/false` 或 `1/0`。字段对字段比较要求两侧类型兼容，`in/notIn` 和空值操作符不能用于字段对字段比较。

友好名称与底层编码由服务统一转换。特别注意：`lessOrEqual` 对应运行时 `m`，`greaterOrEqual` 对应运行时 `h`。新脚本应始终使用友好名称，不要手写底层编码。

## 5. 创建、更新、停用、清空和删除

```text
cloudcc update lookupFilter . @lookup-filter.json
cloudcc apply msapi . <planId>
cloudcc operation msapi . <operationId>
cloudcc changes msapi . <operationId>
cloudcc detail lookupFilter . contact account_id
```

`create` 和 `update` 都会返回待执行计划。先检查目标对象、关系字段、`conditionCount`、`enabled`、删除/替换步骤和风险级别，再 apply。apply 返回 `operationId`；用 `operation` 查看状态、`changes` 查看实际变更，最后必须用 `detail` 回读。计划生成后若关系字段目标发生变化，apply 的漂移保护会拒绝陈旧计划，此时重新查询并生成计划。

也可以通过 `fields` 资源，在 `Y`、`M` 或 `MR` 关系字段创建/更新 spec 中嵌入同一 `lookupFilter`：

```json
{
  "object": "contact",
  "apiName": "account_id",
  "type": "Y",
  "lookupObjectId": "account",
  "lookupFilter": {
    "enabled": true,
    "conditions": [{
      "left": {"scope": "lookupObject", "field": "status"},
      "operator": "equals",
      "right": {"value": "Active"}
    }]
  }
}
```

| 输入 | 结果 |
|---|---|
| 省略 `lookupFilter` | 保留已有规则 |
| `{"enabled": false}` | 停用但保留已有条件 |
| `{"clear": true}` | 停用并删除条件 |
| `cloudcc delete lookupFilter ...` | 等价于清空 |
| `lookupFilter: null` | 拒绝，避免误清空 |
| `enabled: true` 且显式空条件 | 拒绝，避免启用无效规则 |

停用：

```json
{"object":"contact","field":"account_id","lookupFilter":{"enabled":false}}
```

清空：

```json
{"object":"contact","field":"account_id","lookupFilter":{"clear":true}}
```

独立删除：

```text
cloudcc delete lookupFilter . contact account_id
cloudcc apply msapi . <planId>
cloudcc changes msapi . <operationId>
cloudcc detail lookupFilter . contact account_id
```

这里的 delete 是“停用并删除全部筛选条件”，不会删除关系字段本身。

## 6. 审计和回滚

每次 apply 后保留 `operationId`。需要撤销时：

```text
cloudcc rollback-plan msapi . <operationId>
cloudcc rollback msapi . <operationId>
cloudcc operation msapi . <operationId>
cloudcc detail lookupFilter . contact account_id
```

先审阅 `rollback-plan`，确认它只恢复预期字段状态和条件，再执行 rollback，并以 `detail` 回读为准。回滚的是一次已记录的 MetadataService 操作，不是任意时间点恢复；如果之后已有其他修改，应先判断回滚计划是否仍安全。

MetadataService 还提供只读 `GET /metadata/v1/lookup-filters/operator-audit?limit=100`，用于从操作账本中列出可以明确证明的历史 `m/h` 反转计划。它不会猜测所有旧租户行，也不会自动修复；报告项必须逐条结合当前配置和业务含义人工确认。

## 7. 旧 conditionVals 兼容

旧版 `conditionVals` 仍可输入，文本 JSON 和对象形式都会经过与友好结构相同的字段、操作符、类型、选项值和逻辑校验。`u/x/s/m/h` 等旧编码会归一化后再生成计划。

新脚本应使用 `lookupFilter.conditions`。同一请求同时提供 `lookupFilter` 和 `conditionVals` 时，两者必须等价，否则失败。

## 8. 自关联和排错

当关系字段的源对象与查找对象相同时，旧数据只凭对象 ID 无法可靠区分两侧。新写入必须使用能被运行时安全表达的 `lookupObject` 范围；歧义的 `sourceObject` 比较会失败关闭。读取旧规则时，服务会结合持久化的 `filterType` 恢复范围。

常见排错顺序：

1. 运行 `candidates`，确认字段 `eligible=true`，并从 `supportedOperators` 选择操作符。
2. 选项字段从 `options` 复制 `key`，或使用唯一标签。
3. 检查 `logic` 是否覆盖 1 到条件总数的全部编号。
4. 计划成功后先审阅，再 `apply`，最后用 `detail` 回读。
5. 看到 `referenced_field_missing` 警告时，说明历史条件引用字段已不存在，应修正规则。
6. apply 失败时先看 `operation` 和 `changes`，不要盲目重复提交旧计划。
