# CloudCC Skill 一级 Domain 分类

一级 Domain 表示 CLI 中可直接发现和使用的能力入口，不表示该能力一定属于元数据。使用以下命令查看当前发行包的完整分类与可用性：

```bash
cloudcc domains
cloudcc domain dataIndex
cloudcc domain dataBulk
cloudcc domain openapi
```

Domain 分为五类：

- `metadata`：对象、字段、布局、简档、工作流等平台配置和模型控制面。
- `data-extension`：使用元数据但直接作用于业务数据或物理数据结构的扩展能力，例如 `dataIndex`、`dataBulk`。
- `integration`：外部系统和平台接口能力，例如直接调用 `api-svc` 的 `openapi`。
- `high-code`：类、触发器、定时类、页面组件等平台开发资产。
- `governance`：项目交付物和测试治理等本地能力。

`backend` 只说明能力由哪个服务承载，不决定能力类别。`dataIndex` 和 `dataBulk` 虽由 MetadataService 提供，仍属于 Data Extension；它们不属于元数据，也不是 Metadata Domain。

## 用户需求路由

| 用户要做什么 | 使用 Domain |
|---|---|
| 对标准 CRM 对象或自定义对象执行常规业务数据 CRUD | `openapi` |
| 大批量数据导入、初始化或离线批处理 | `dataBulk` |
| 查看、创建或优化数据库索引 | `dataIndex` |
| 使用平台二开能力配置对象、字段、页面、权限或流程 | 对应元数据 Domain |

常规业务数据操作不要路由到 `dataBulk`。`openapi` 才是标准对象和自定义对象日常查询、新增、修改、删除及 upsert 的入口。
