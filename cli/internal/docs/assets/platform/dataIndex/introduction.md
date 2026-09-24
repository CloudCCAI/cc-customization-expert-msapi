# dataIndex：业务数据索引生命周期管理

`dataIndex` 是 CloudCC Skill 中与对象、字段、布局同等级的一级 Domain，用于查看、规划、创建、分析和安全优化业务数据索引。

第一步查看完整用户文档：

```bash
cloudcc doc dataIndex
cloudcc doc dataIndex optimization
cloudcc doc dataIndex database-limits
cloudcc doc dataIndex troubleshooting
```

该 Domain 只接受对象和字段 API 名，不接受物理表名、物理列名或 SQL。创建和优化都会先读取实际数据库索引并在异步执行后进行元数据回读。

核心流程：

1. `get` 查看对象当前全部索引。
2. `plan` 预检拟创建索引，不执行 DDL。
3. `create --confirm` 创建普通非唯一索引。
4. `analyze` 生成优化方案，绝不执行 DDL。
5. 用户审阅后使用 `optimize --confirm` 执行选中的安全建议。
6. `status` 查询创建或优化任务。

要求 MetadataService `1.1.71`，并由服务端显式启用 `MDS_BUSINESS_DATA_INDEX_ENABLED=true`。
