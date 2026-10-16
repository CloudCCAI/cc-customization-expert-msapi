# 字段依赖

`fieldDependency` 根据控制字段的当前值限制依赖字段的可选值。例如，客户类型为“企业”时只允许选择企业联系人级别。

## 能力范围

- 查询对象已有的字段依赖和单个字段对详情。
- 查询可作为控制字段或依赖字段的候选字段。
- 创建或完整替换自定义值映射。
- 清空自定义映射但保留字段对配置，或删除整个字段对配置。
- 复用两个全局选项列表之间已经存在的全局依赖。
- MSAPI 支持计划审阅、执行记录、变更审计和回滚。

控制字段可以是复选框或单选列表，依赖字段可以是单选列表或多选列表。两个字段必须属于同一对象，不能相同，也不能形成直接或间接循环。

## 平台边界

- MSAPI：写命令先返回待审阅的 `planId`；显式 apply 后返回 `operationId`，支持 changes 和 rollback。
- Lightning UIAPI：写命令直接提交到平台，不生成 MetadataService 计划或回滚记录。
- horizontal UIAPI：当前不可用；horizontal 项目应选择 MSAPI。

完整命令、JSON 示例、自定义/全局模式、回读和排错流程请使用：

```text
cloudcc doc fieldDependency devguide
```
