# 工作流字段更新 Domain

`workflowFieldUpdate` 是工作流字段更新动作的一级 CLI Domain，MetadataService 规范名为 `workflow-field-updates`，对应 setup-web `/#/settings/action/workflow-field-updates`。它描述“工作流执行时怎样修改目标记录的某个字段”，不直接更新业务记录。

## 适用场景

- 工作流满足条件后把字段设为固定值或清空；
- 保存平台字段更新公式；
- 使用平台的上一个值/下一个值语义；
- 复用、迁移、审计字段更新定义并检查引用影响。

## 能力与边界

该 Domain 支持能力查询、列表、详情、对象/字段/值模式候选、引用影响、创建、部分更新和受保护删除。MSAPI 要求 MetadataService `1.1.105+`，由 MetadataService 直接读写 `tp_sys_actions_fieldupdate`，不调用 setup-service。所有写命令为 plan-only。

字段必须属于目标对象。五种值模式是能力并集，不是每种字段都能使用：`L/R` 支持固定/上一个/下一个，`Y/B/SCORE` 支持类型化固定值，其他可更新字段支持清空/公式。必须先读取 `candidates.fields[].type` 与 `valueModesByFieldType`；固定值的数据格式、选项键和公式可用性取决于目标环境，并应在测试工作流中验证。

工作流关系归 `workflow` Domain：

- 已有字段更新只能绑定即时作用域；
- 时间作用域可创建并绑定新的字段更新；
- 周期作用域不支持字段更新；
- 解绑和工作流删除只清理关系，保留字段更新定义；
- 存在引用时不能删除或改变目标对象。

## 建议工作流

```bash
cloudcc candidates workflowFieldUpdate . Account
cloudcc create workflowFieldUpdate . @field-update.json
cloudcc apply msapi . <planId>
cloudcc detail workflowFieldUpdate . afud_mark_contacted
cloudcc bindAction workflow . @bind-field-update.json
```

完整命令、五种值模式、字段合同、引用保护、绑定限制和验收清单见：

```bash
cloudcc doc workflowFieldUpdate devguide
```
