# dataBulk：业务数据批量作业

`dataBulk` 是一级 Data Extension Domain，专门用于大批量数据导入、初始化和离线批处理。它由 MetadataService 提供物理映射、作业账本和批量 DML，但不属于元数据 Domain，也不走通用 metadata `plan/apply`。

`dataBulk` 不适用于日常业务场景。标准 CRM 对象和自定义对象的常规查询、新增、修改、删除和 upsert 都使用 `openapi`。

## 文档入口

```bash
cloudcc doc dataBulk                  # 完整操作指南
cloudcc doc dataBulk introduction     # 能力概览
cloudcc doc dataBulk operations       # 五种操作的数据规则
cloudcc doc dataBulk limits           # 分片、流式与容量限制
cloudcc doc dataBulk troubleshooting  # 状态、错误与恢复
```

## 标准流程

1. `schema` 读取对象的 `DIRECT_WRITABLE` 字段、自动编号状态和 `maxInlineRecords`。
2. 准备 JSON、NDJSON 或 CSV，只提交允许写入的逻辑字段 API 名。
3. `submit` 提交并建议使用 `--wait --output-dir` 保存终态结果。
4. `status` 查看作业，`results` 查看逐行结果。
5. 中断作业使用 `resume`；`FAILED`/`PARTIAL_SUCCESS` 使用 `retry`；未终态作业可 `cancel`。

```bash
cloudcc schema dataBulk <projectPath> Account
cloudcc submit dataBulk <projectPath> Account INSERT @accounts.ndjson \
  --format ndjson --wait --output-dir ./bulk-results
```

## 可用条件

- MSAPI provider。
- MetadataService `1.1.71` 或更高版本。
- 服务端启用 `MDS_BUSINESS_DATA_BULK_ENABLED=true`。
- 读取、写入和删除分别具备 `data:bulk:read`、`data:bulk:write`、`data:bulk:delete` scope。

UIAPI 包会在读取凭据和发起网络请求前拒绝；Universal 包必须先选择 MSAPI provider。dataBulk 直接写业务数据，不执行验证规则、触发器、查重过滤器、共享规则或工作流。
