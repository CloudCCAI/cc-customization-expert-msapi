# 详情页报表分配介绍

`detailPageReport` 用于把已有报表展示在业务记录详情页。它是独立元数据 Domain，归属键是“对象 + 记录类型”，不是页面布局 ID。

## 能力范围

- 同一对象的主记录类型和每个自定义记录类型可以分别配置。
- 每个分配包含 1–10 个按顺序展示的报表图表。
- 图表可以展示全部报表数据，也可以按当前详情记录过滤。
- `report` 管理报表定义；`detailPageReport` 只管理已有报表的详情页展示分配。
- MSAPI 支持 plan、changes、apply 和 rollback；Lightning UIAPI 直接调用 setup-service；horizontal UIAPI 当前失败关闭。

## 快速查看

```text
cloudcc domain detailPageReport
cloudcc get detailPageReport <projectPath> [object] [recordType]
cloudcc candidates detailPageReport <projectPath> <object> [recordType]
```

完整命令、参数、JSON 示例、创建/更新/删除流程和常见问题，请查看：

```text
cloudcc doc detailPageReport devguide
```
