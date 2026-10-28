# 查找字段筛选器

`lookupFilter` 用于限制关系字段弹出的可选记录。它适用于查找关系 `Y`、主从关系 `M` 和反向主从关系 `MR`；与字段依赖 `fieldDependency` 不同，它筛选的是关联对象记录，不是选项列表值。

CLI 支持两种等价入口：

- 独立资源 `lookupFilter`：适合查询、单独维护和删除规则；
- `fields` 的 `lookupFilter` 属性：适合创建或更新关系字段时一次提交。

写操作在 MSAPI 下只创建计划，必须显式执行 `cloudcc apply msapi <planId>`。setup-web 的界面位于字段创建完成后的字段详情页：`Lookup Filter / 查找字段关联规则 -> Edit`，因此创建字段弹窗里看不到它。

完整示例见：

```text
cloudcc doc lookupFilter devguide
```
