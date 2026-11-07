# CloudCC 查重过滤器（Dupe Catcher）

查重过滤器对应设置页 `/#/settings/dupeCatcher`，详情页是
`/#/settings/dupeCatcher/detail?id=<filterId>`。CLI 一级 Domain 名称是 `dupeCatcher`，服务端名称是
`dupe-catchers`。

一个过滤器聚合由三部分组成：

- 过滤器主体：目标对象、启用状态，以及新增/更新时提示或阻止保存的策略。
- 条件：限定新记录和已有记录进入查重的范围；条件本身不定义怎样算重复。
- 规则：定义参与比较的字段和匹配方式；多条规则按 **AND** 组合。

因此运行语义是“新旧记录均满足条件，且每条规则都匹配，才命中重复”。没有规则的过滤器不会产生有效
重复命中，建议先以未启用状态创建，添加并回读规则后再启用。

MetadataService `1.2.6+` 提供 Contract 2：规范化详情、对象/规则字段/条件字段候选项、规则详情及
独立的规则增改删计划。Lightning UIAPI 映射 setup-service 标准接口；Horizontal UIAPI 没有已验证合同，
失败关闭。写命令在 MSAPI 下只创建计划，必须显式 apply。

完整生命周期、JSON 和错误说明：

```bash
cloudcc doc dupeCatcher devguide
```
