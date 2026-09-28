# dataIndex：业务数据索引生命周期管理

`dataIndex` 是 CloudCC Skill 的一级 Data Extension Domain，用于查看、规划、创建、分析和安全优化业务数据索引。它由 MetadataService 承载，但不属于对象、字段、布局等元数据 Domain。

## 文档入口

```bash
cloudcc doc dataIndex                  # 完整操作指南
cloudcc doc dataIndex introduction     # 能力概览
cloudcc doc dataIndex optimization     # 优化协议与建议类型
cloudcc doc dataIndex database-limits  # 数据库与容量限制
cloudcc doc dataIndex troubleshooting  # 错误码与处理方式
```

## 核心安全流程

1. `get` 查看对象当前全部索引。
2. `plan` 预检拟创建索引，不执行 DDL。
3. 审阅方案后使用 `create --confirm` 创建普通非唯一索引。
4. `analyze` 生成带快照指纹和有效期的优化方案，不执行 DDL。
5. 使用 `optimization-plan` 审阅每条建议及其 `executable` 状态。
6. 用户选择建议后使用 `optimize --confirm` 执行。
7. `status` 查询创建或优化任务，并使用 `get` 回读最终索引。

该 Domain 只接受对象和字段 API 名，不接受物理表名、物理列名或 SQL。创建和优化都会在执行前读取真实索引，并在异步执行后回读验证。

## 可用条件

- MSAPI provider。
- MetadataService `1.1.71` 或更高版本。
- 服务端启用 `MDS_BUSINESS_DATA_INDEX_ENABLED=true`。
- 读取、创建和优化分别具备 `data:index:read`、`data:index:write`、`data:index:optimize` scope。

UIAPI 包会在读取凭据和发起网络请求前明确拒绝；Universal 包必须先选择 MSAPI provider。开始前运行：

```bash
cloudcc doctor provider <projectPath>
cloudcc domain dataIndex
cloudcc doc dataIndex
```
