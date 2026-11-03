# CloudCC 公用小组 Domain

`publicGroup` 是 CLI 一级资源，MetadataService 规范 Domain 为 `public-groups`。它用于查询、创建、修改、
分配成员和受保护删除公用小组。

```bash
cloudcc get publicGroup .
cloudcc detail publicGroup . <小组ID或唯一名称>
cloudcc capabilities publicGroup .
cloudcc candidates publicGroup . user
cloudcc candidates publicGroup . publicGroup <目标小组>
```

成员统一使用四种易读类型：`user`、`role`、`roleAndSubordinates`、`publicGroup`。不要直接提交数据库的
`User`、`Role`、`RoleAndSub`、`Group`，也不要提交 `selecteditems`、成员行 ID、审计字段或小组类型。

MSAPI 写命令只生成计划，审阅后必须显式 apply：

```bash
cloudcc apply msapi . <planId>
cloudcc changes msapi . <operationId>
cloudcc rollback-plan msapi . <operationId>
```

Lightning UIAPI 使用现有 setup-service 合同，支持查询和完整替换成员；增量 `add/remove`、循环校验、漂移
保护、changes 和 rollback 需要 MSAPI。Horizontal UIAPI 没有已维护合同，会失败关闭并提示改用 MSAPI。

完整 JSON、provider 差异、删除保护、错误码和排障流程见：

```bash
cloudcc doc publicGroup devguide
```
