# CCChat 设置与微贴跟踪

`ccchat` 是一个全局 Metadata Domain，包含两个子资源：

- `settings`：组织级 CCChat 开关、微贴编辑、群组归档、同事邀请和邮箱域。
- `feedTracking`：对象级微贴跟踪开关与最多 20 个跟踪字段。

它与 `fieldTracking` 完全不同：`fieldTracking` 管理 SQL 字段历史，`ccchat feedTracking` 管理 CCChat 微贴配置。

MSAPI 使用 MetadataService 进行 plan/apply/changes/rollback 审计，并由 setup-service 权威执行 Mongo、People 初始化和 Redis 刷新。CLI 和 MetadataService 不连接租户 Mongo。Lightning UIAPI 直接使用既有 setup-service 接口；horizontal UIAPI 失败关闭。

数据归档不在本能力范围内。`allowGroupArchived` 是群组归档业务开关，不是 CCChat 数据归档。

快速开始：

```powershell
cloudcc get ccchat . settings
cloudcc getList ccchat . feedTracking
cloudcc candidates ccchat . feedTracking Account
cloudcc doc ccchat devguide
```
