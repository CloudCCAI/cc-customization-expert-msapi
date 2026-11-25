# 关键信息布局

`compactLayout` 管理对象记录详情的重点字段和记录类型分配。每个布局可选择 1–10 个字段，展示顺序跟随配置。记录优先使用记录类型的显式分配，再使用对象主要布局，最后使用系统默认布局。

适用于客户详情首屏显示名称、级别和负责人，商机首屏显示阶段、金额和预计成交日期，或按记录类型为不同业务显示不同摘要。先选择能帮助用户辨识记录和判断当前状态的少量字段，再按需要增加专用布局。已有布局可以满足需求时可直接调整字段与顺序。

主要布局对没有显式记录类型映射的记录生效。系统默认布局由平台维护，不能删除。新增普通布局需要明确分配后才会影响相应记录。修改普通布局的标签、API 名或字段时，应先读取详情和当前分配，确认实际使用范围。

- MSAPI 需要 MetadataService 1.2.20+。支持查询、候选、创建、更新、分配、删除影响、删除及审计回滚。
- Lightning UIAPI 直接使用平台接口。支持查询、创建、更新和完整分配，并回读 setup 结果。删除和精确清除分配使用 MSAPI。
- 横纵版 UIAPI 不开放此 Domain。Universal 按项目实际执行渠道选择动作。

MSAPI 的写命令先返回计划，apply 后验证配置。Lightning UIAPI 写命令直接执行并回读，分配结果只能确认 setup 页面投影。选择操作渠道时通过项目 provider 配置确定，命令使用同一套 JSON 属性。

使用顺序为查询能力、读取当前布局和分配、读取候选、提交变更、执行计划和回读。删除前还要读取删除影响并处理主要布局或记录类型引用。完整参数、返回值、清空规则、文件示例和错误处理见 `cloudcc doc compactLayout devguide`。

```bash
cloudcc domain compactLayout
cloudcc capabilities compactLayout . Account
cloudcc doc compactLayout devguide
```
