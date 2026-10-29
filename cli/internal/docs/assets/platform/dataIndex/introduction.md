# dataIndex：业务数据索引生命周期管理

`dataIndex` 是 CloudCC Skill 的一级 Data Extension Domain，只用于查看、规划、创建、分析和安全优化数据库索引。它不操作业务记录，也不配置对象、字段、布局等平台元数据。

删除计划会在人工批准前显示目标表数据量和谨慎阈值。默认超过 1,000,000 行返回 `CAUTION_LARGE_TABLE`；该提示要求人工重点评估性能影响，但不会绕过或扩大主键、唯一索引的硬保护范围。

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
6. 使用目标 `indexRef` 创建 `plan-delete`，向调用方返回风险、数量、challenge 和 fingerprint，不执行 DDL。
7. 由具备独立批准权限的人工操作方使用 `approve-delete` 发起第二次请求；主键和唯一索引始终阻断。
8. `status` 查询创建或删除任务，`delete-events` 查询追加式审计事件，并使用 `get` 回读最终索引。

该 Domain 只接受逻辑对象/字段和列表返回的稳定 `indexRef`，不接受物理表名、物理列名或 SQL。删除必须经过不可变预览和独立人工批准，并在异步执行后回读验证。

## 可用条件

- MSAPI provider。
- 删除能力要求 MetadataService `1.1.93` 或更高版本。
- 服务端启用 `MDS_BUSINESS_DATA_INDEX_ENABLED=true`。
- 读取、创建、删除计划和人工批准分别具备 `data:index:read`、`data:index:write`、`data:index:delete:plan`、`data:index:delete:approve` scope。

UIAPI 包会在读取凭据和发起网络请求前明确拒绝；Universal 包必须先选择 MSAPI provider。开始前运行：

```bash
cloudcc doctor provider <projectPath>
cloudcc domain dataIndex
cloudcc doc dataIndex
```
