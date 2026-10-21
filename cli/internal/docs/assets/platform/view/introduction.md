# CloudCC 对象视图介绍

对象视图控制业务对象列表的展示列、列顺序、筛选条件和排序，供用户按业务场景查看记录。

## 当前能力

- 按对象 ID、API 名或对象前缀查询视图列表，以及按视图 ID 读取详情。
- MSAPI 的 `object-views` Domain 支持视图创建、更新和删除计划，包括字段、条件及受支持的图表/看板配置。
- `fieldSetupAllViews` 可批量同步对象视图的展示字段，具体执行通道和参数见指南。

`get/getList view` 是列表查询，`detail/editInfo view` 才按视图 ID 查询。MSAPI 写入先生成计划，审核后显式 apply；视图删除可能影响列表按钮、图表、看板和用户默认视图，应先核对引用。

使用 `cloudcc doc view devguide` 查看参数、JSON 示例和计划执行方法；也可用 `cloudcc doc object-views devguide`。
