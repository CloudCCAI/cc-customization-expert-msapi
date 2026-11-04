# CloudCC 队列 CLI

`queue` 是业务记录所有人队列，对应设置页 `/#/settings/usermanage/queue`，不是消息队列、审批队列、
Bulk job 或公海池。CLI 复用平台 `/api/queueSetup/*` 标准能力，不修改 setup-web、setup-service，也不裸写
队列表来模拟标准行为；因此创建、修改对象范围和删除时，平台生成队列视图的副作用仍由标准服务负责。

常用读取命令：

```bash
cloudcc capabilities queue .
cloudcc get queue .
cloudcc get queue . 服务
cloudcc detail queue . ServiceCaseQueue
cloudcc candidates queue . objects Case
cloudcc candidates queue . members user 张三
cloudcc impact queue . ServiceCaseQueue delete
```

成员只支持 `user`、`role`、`roleAndSubordinates`。公用小组成员在当前版本失败关闭，不能用枚举值 `1`
绕过，因为运行时权限和通知展开合同尚未得到平台证据。

MSAPI 写命令只生成计划，必须审阅并显式 apply：

```bash
cloudcc create queue . @queue-create.json
cloudcc apply msapi . <planId>
cloudcc changes msapi . <operationId>
cloudcc rollback-plan msapi . <operationId>
```

Lightning UIAPI 直接调用标准服务并回读；其 `assign` 仅支持 `replace`。Horizontal UIAPI 没有已验证合同，
会返回 `queue_horizontal_uiapi_unsupported`。完整输入、分配、删除保护、错误码和排障见：

```bash
cloudcc doc queue devguide
```
