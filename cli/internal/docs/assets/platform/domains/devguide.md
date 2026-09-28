# 一级 Domain 开发与路由规则

## 三个独立维度

每个 Domain 必须同时声明：

1. `category`：metadata、data-extension、integration、high-code 或 governance。
2. `backend` / `transport`：metadata-service、api-svc、setup-svc/devconsole 或 local，以及 msapi、direct 或 local。
3. `availability`：MSAPI、UIAPI、Universal 三种发行包中的可用状态。

不得根据服务名称推断能力类别，也不得仅靠帮助文本限制运行时访问。`dataIndex` 和 `dataBulk` 不属于元数据；MetadataService 只是它们当前的承载后端。

## Provider 规则

- `dataIndex`、`dataBulk`：MSAPI 启用；UIAPI 不可用；Universal 只有实际选择 MSAPI 时可用。
- `openapi`：标准 CRM 对象和自定义对象的常规业务数据 CRUD；三个发行包均可用，直接调用平台 `api-svc`，不经过 MetadataService。
- `dataBulk`：只用于大批量数据导入、初始化和离线批处理，不作为日常业务操作入口。
- `dataIndex`：只用于数据库索引生命周期和优化。
- 对象、字段、页面、权限和流程等平台二开配置使用对应元数据 Domain。
- UIAPI 对 MSAPI-only Domain 的拒绝发生在凭据读取和业务 HTTP 请求之前。

## 发现与文档

`cloudcc domains` 返回当前包的分类、后端和有效可用性；`cloudcc domain <name>` 返回一个 Domain 的动作、版本要求和文档入口。Domain 行为以嵌入 CLI 的 Domain Catalog 为事实源，帮助、发行包配置和运行时门禁必须保持一致。
