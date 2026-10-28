# 查找字段筛选器 CLI 用户指南

## 命令

```text
cloudcc get lookupFilter <projectPath> <object>
cloudcc detail lookupFilter <projectPath> <object> <field>
cloudcc candidates lookupFilter <projectPath> <object> <field>
cloudcc create lookupFilter <projectPath> <@filter.json>
cloudcc update lookupFilter <projectPath> <@filter.json>
cloudcc delete lookupFilter <projectPath> <object> <field>
```

别名 `lookup-filter`、`lookup-filters` 等价。需要 MetadataService `1.1.92` 或更高版本。

## 独立创建或更新

`lookup-filter.json`：

```json
{
  "object": "contact",
  "field": "account_id",
  "lookupFilter": {
    "enabled": true,
    "logic": "1 AND 2",
    "conditions": [
      {
        "left": {"scope": "lookupObject", "field": "status"},
        "operator": "equals",
        "right": {"value": "Active"}
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

```text
cloudcc update lookupFilter . @lookup-filter.json
cloudcc apply msapi <planId>
cloudcc detail lookupFilter . contact account_id
```

`scope` 只接受 `lookupObject` 和 `sourceObject`。支持的操作符为 `equals`、`notEquals`、`contains`、`notContains`、`greaterThan`、`lessThan`、`greaterOrEqual`、`lessOrEqual`。规则最多 10 条；字段名或对象名必须在目标环境唯一解析，无法解析时计划会失败，不会猜测。

## 与字段创建或更新一起提交

在关系字段 spec 中直接加入同一个 `lookupFilter`：

```json
{
  "object": "contact",
  "apiName": "account_id",
  "type": "Y",
  "lookupObjectId": "account",
  "lookupFilter": {
    "enabled": true,
    "logic": "1",
    "conditions": [
      {
        "left": {"scope": "lookupObject", "field": "status"},
        "operator": "equals",
        "right": {"value": "Active"}
      }
    ]
  }
}
```

```text
cloudcc create fields . @field.json
cloudcc apply msapi <planId>
```

省略 `lookupFilter` 表示保留现有规则；显式 `null` 会被拒绝。`{"enabled":false}` 只停用并保留条件，`{"clear":true}` 会停用并删除条件。独立命令 `cloudcc delete lookupFilter ...` 等价于清空规则。

旧版 `conditionVals` 结构仍可读取和写入，但新脚本应优先使用上述友好结构。不要在同一次请求中同时传入内容冲突的 `lookupFilter` 和 `conditionVals`。
