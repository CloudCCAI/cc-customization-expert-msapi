# 字段依赖开发指南

## 自定义映射

```json
{
  "object": "contact",
  "controlField": "customer_type",
  "dependentField": "contact_level",
  "mode": "custom",
  "mappings": [
    {"controlValue": "enterprise", "allowedValues": ["key", "normal"]},
    {"controlValue": "individual", "allowedValues": ["normal"]}
  ]
}
```

`create`、`update` 和 `upsert` 都按目标字段对执行幂等替换。`mappings` 必须显式提供；空数组表示清空该字段对的自定义映射。CLI 先创建 MetadataService 计划，不会直接写库。

## 使用全局依赖

```json
{
  "object": "contact",
  "controlField": "customer_type",
  "dependentField": "contact_level",
  "mode": "global"
}
```

全局模式要求两个字段分别绑定唯一的全局选项列表，并且这两个全局列表之间已有全局字段依赖。切换到全局模式会在计划中清理该对象字段对的自定义映射，再写入全局引用配置。

## 安全约束

- 字段选择器支持字段 ID 或 API 名称；对象选择器支持 ID、API 名称或前缀。
- 控制字段仅允许 `B`/`L`，依赖字段仅允许 `L`/`Q`。
- 保存前检测直接和间接循环；值选择器必须在目标字段的选项中唯一命中。
- 记录类型使用的 `CONTROL_FIELD_TYPE=R` 映射不属于本能力，不会被字段依赖查询或替换。
- apply 会记录 before/after 快照，可按 MetadataService 操作回滚。
