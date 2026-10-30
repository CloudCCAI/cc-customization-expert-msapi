# 查找字段筛选器

`lookupFilter` 限制 `Y`、`M`、`MR` 关系字段弹窗中的候选记录。它筛选关联对象记录，不是字段依赖的选项联动。

CLI 提供两种入口：独立 `lookupFilter` 资源用于查询和维护；关系字段 spec 内的 `lookupFilter` 用于随字段一起创建或更新。MSAPI 写操作只生成计划，必须审阅后显式 `cloudcc apply msapi <planId>`。

MetadataService `1.1.95+` 支持类型感知的操作符候选、`startsWith`、`in/notIn` 多值集合、`isNull/isNotNull`、选项标签到存储键归一化、旧 `conditionVals` 统一校验及可靠回读。配置前先运行：

```text
cloudcc candidates lookupFilter <projectPath> <object> <field>
cloudcc doc lookupFilter devguide
```

其中“字段等于审批通过、草稿、提交待审批、审批拒绝中的任意一个”应使用 `operator: "in"` 和 `right.values` 数组，完整可复制示例见用户指南。
