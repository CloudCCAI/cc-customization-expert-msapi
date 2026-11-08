# CloudCC 对象列表视图介绍

对象列表视图定义业务对象列表的展示字段、记录筛选、可见范围、排序和自定义按钮。本 Domain 操作的是“视图定义”；选中视图后的业务记录、表头运行态和记录数量不在本 Domain 内。

## 视图类型

- `0`：我最近查看的；`1`：我最近创建的；`2`：我最近修改的；`3`：最新记录；`4`：数据归档。
- `0..4` 是平台合成的虚拟视图，没有可编辑的持久化定义。它们可以成为用户默认视图，但不能创建、修改、复制或删除。
- “全部”“我的”、系统视图和默认视图通常是持久化视图。它们不是虚拟视图，不能仅因系统、标准或默认标签而判定为只读；实际编辑/删除能力由当前用户权限、所有权和引用状态决定。

“我的”记录范围表示按业务记录 owner 筛选，不等于视图只对自己可见。视图可见范围另由 `private`、`allUsers` 或 `members` 控制；`queue` 也是记录范围，不是共享范围。

## Contract 与 Provider

- Contract 1 的 `get/detail view` 和通用 plan/apply 保留原有低层兼容形状。
- Contract 2 必须显式使用 `list-v2/detail-v2` 或在计划 spec 中提交 `viewContractVersion: 2`；旧服务不支持时失败关闭。
- MSAPI 由 MetadataService 直接事务维护关系型视图聚合、用户默认值和顺序，并绑定已验证的 CloudCC 当前用户；虚拟 ID `3`（最新记录）和 `4`（数据归档）的出现状态通过当前项目 `apiSvc` 做只读权威解析，无法解析时明确省略并标记 `unresolved`，不会信任调用方布尔值。
- Lightning UIAPI 直接调用项目 `apiSvc`，由 api-service 转发 main-api 的权威 `/api/view/**` 实现。setup-service 的同路径接口是较旧子集，不是完整能力基线。
- UIAPI 没有 MetadataService ledger/原子 rollback 保证；Universal 在动作开始前固定 provider，Horizontal 没有等价接口时失败关闭。

使用 `cloudcc doc view devguide` 查看命令、更新集合语义和安全规则。
