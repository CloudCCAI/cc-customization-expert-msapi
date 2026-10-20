# 字段追踪

`fieldTracking` 管理对象字段的历史追踪配置，与字段依赖无关。

## 能力范围

- 查看对象当前启用追踪的字段。
- 查看可配置候选字段及其追踪模式。
- 使用完整字段集合替换当前用户可管理的追踪配置。
- 清空用户可管理字段，同时保留平台管理的系统字段。
- MSAPI 支持计划审阅、执行记录、变更审计和回滚。

`trackedFields` 使用完整替换语义，而不是增量添加。数组中遗漏的用户可管理字段会被关闭；空数组等价于 `clear`。字段选择器可使用候选结果返回的字段 ID 或 API 名，`systemManaged=true` 的字段不要写入输入。

这里配置的是已有字段是否参与历史追踪，不会创建新字段。MSAPI 不限制用户可管理追踪字段的数量，也不读取或校验公有云 `FIELD_TRACK / RES010` 配额；字段是否可追踪仍以 candidates 返回的资格结果为准。

## 平台边界

- MSAPI：写命令先返回待审阅的 `planId`，显式执行后返回 `operationId`，支持 changes 和 rollback。
- Lightning UIAPI：写命令直接提交，不生成 MetadataService 计划或回滚记录；setup-service 自身的限制和错误由平台决定并原样返回。
- horizontal UIAPI：当前不可用；horizontal 项目应选择 MSAPI。

完整命令、JSON 示例、回读和排错流程请使用：

```text
cloudcc doc fieldTracking devguide
```
