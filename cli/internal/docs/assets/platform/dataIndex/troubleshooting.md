# dataIndex 故障排查

| 错误码或现象 | 原因 | 处理方式 |
|---|---|---|
| Domain unavailable / requires MSAPI | 当前为 UIAPI，或 Universal 选择了 UIAPI | 运行 `cloudcc doctor provider <projectPath>`，改用 MSAPI 或配置兼容的 MetadataService |
| 功能端点不存在或被关闭 | MetadataService 版本过低，或未启用索引能力 | 确认版本不低于 `1.1.71`，并设置 `MDS_BUSINESS_DATA_INDEX_ENABLED=true` |
| `data_index_invalid_request` | 字段为空、超过四个、重复或输入不符合逻辑 API 约束 | 只传一至四个不同字段 API 名 |
| `data_index_field_not_supported` | 字段是 LOB、长文本、公式、复合存储或没有单一物理列 | 调整索引字段，不能通过物理列名绕过 |
| `data_index_limit_exceeded` | 达到平台或数据库索引数量上限 | 执行 `cloudcc analyze dataIndex <projectPath> <object>` 并审阅冗余建议 |
| `data_index_key_too_long` | 字段组合超过索引键字节限制 | 减少或调整字段；当前不支持字符串前缀索引 |
| `data_index_name_conflict` | 同名索引已有不同字段定义 | 省略 `--name` 让系统生成，或更换名称 |
| `data_index_redundant` | 已有长索引覆盖请求的最左前缀 | 使用 `get` 查看现有索引，不要重复创建 |
| `data_index_permission_denied` | 数据库账户缺少 DDL 权限 | 由管理员补充最小权限后重试 |
| `data_index_lock_timeout` | DDL 无法获得元数据锁 | 在低峰维护窗口重新 `plan` 后执行 |
| `data_index_disk_full` | 表空间或磁盘不足 | 扩容或清理空间后重试 |
| `data_index_worker_rejected` | 索引任务执行队列已满 | 稍后重试；不要并发重复提交相同创建请求 |
| `data_index_optimization_plan_expired` | 优化方案超过有效期 | 重新执行 `analyze` |
| `data_index_optimization_plan_stale` | 分析后索引结构发生变化 | 重新执行 `analyze`，再审阅新方案 |
| `data_index_recommendation_not_found` | 建议 ID 不属于当前方案 | 从 `optimization-plan` 的 `recommendations[]` 复制 ID |
| `data_index_recommendation_not_executable` | 建议只能人工审阅 | 不要强制执行；结合查询负载走独立 DBA 评审 |
| `data_index_protected` | 目标是系统、唯一、约束或非平台索引 | 保留该索引，必要时走独立 DBA 评审 |

排查顺序：

```bash
cloudcc doctor provider <projectPath>
cloudcc domain dataIndex
cloudcc get dataIndex <projectPath> <object>
cloudcc status dataIndex <projectPath> <jobId>
```

任务失败时检查 `status`、`errorCode` 和 `errorMessage`。服务不会返回原始 SQL、连接信息或凭据。修复原因后应重新运行 `plan` 或 `analyze`，不要直接重放物理 DDL。
