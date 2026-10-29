# dataIndex 故障排查

| 错误码或现象 | 原因 | 处理方式 |
|---|---|---|
| Domain unavailable / requires MSAPI | 当前为 UIAPI，或 Universal 选择了 UIAPI | 运行 `cloudcc doctor provider <projectPath>`，改用 MSAPI 或配置兼容的 MetadataService |
| 功能端点不存在或被关闭 | MetadataService 版本过低，或未启用索引能力 | 删除要求版本不低于 `1.1.93`，并设置 `MDS_BUSINESS_DATA_INDEX_ENABLED=true` |
| `data_index_invalid_request` | 字段为空、超过四个、重复或输入不符合逻辑 API 约束 | 只传一至四个不同字段 API 名 |
| `data_index_field_not_supported` | 字段是 LOB、长文本、公式、复合存储或没有单一物理列 | 调整索引字段，不能通过物理列名绕过 |
| `data_index_limit_exceeded` | 达到平台或数据库索引数量上限 | 执行 `cloudcc analyze dataIndex <projectPath> <object>` 并审阅冗余建议 |
| `CAUTION_LARGE_TABLE` | 删除计划检测到表数据量超过谨慎阈值 | 查看 `tableRowCountExact`；为 false 时按“至少该数量”理解，并在批准前评估查询、写入与维护窗口影响 |
| `data_index_delete_row_count_changed` | 计划后、批准前数据量跨过谨慎阈值 | 重新执行 `plan-delete`，审阅大表警告后再批准 |
| `data_index_key_too_long` | 字段组合超过索引键字节限制 | 减少或调整字段；当前不支持字符串前缀索引 |
| `data_index_name_conflict` | 同名索引已有不同字段定义 | 省略 `--name` 让系统生成，或更换名称 |
| `data_index_redundant` | 已有长索引覆盖请求的最左前缀 | 使用 `get` 查看现有索引，不要重复创建 |
| `data_index_permission_denied` | 数据库账户缺少 DDL 权限 | 由管理员补充最小权限后重试 |
| `data_index_lock_timeout` | DDL 无法获得元数据锁 | 在低峰维护窗口重新 `plan` 后执行 |
| `data_index_disk_full` | 表空间或磁盘不足 | 扩容或清理空间后重试 |
| `data_index_worker_rejected` | 索引任务执行队列已满 | 稍后重试；不要并发重复提交相同创建请求 |
| `data_index_delete_plan_expired` | 删除计划超过有效期 | 重新执行 `plan-delete` 并重新人工批准 |
| `data_index_delete_plan_stale` | 计划或批准后索引结构发生变化 | 重新生成、审阅并批准计划 |
| `data_index_delete_challenge_mismatch` | challenge 不是当前计划返回值 | 重新读取计划，由人工逐字确认 |
| `data_index_delete_fingerprint_mismatch` | 批准的指纹不是已审阅快照 | 停止批准并重新核对计划 |
| `data_index_delete_plan_already_used` | 计划已经批准或重放 | 查询原 job；需要重试时创建新计划 |
| `data_index_delete_protected` | 目标在执行前成为主键或唯一索引 | 禁止删除并重新生成计划 |

排查顺序：

```bash
cloudcc doctor provider <projectPath>
cloudcc domain dataIndex
cloudcc get dataIndex <projectPath> <object>
cloudcc status dataIndex <projectPath> <jobId>
```

任务失败时检查 `status`、`errorCode` 和 `errorMessage`。服务不会返回原始 SQL、连接信息或凭据。修复原因后应重新运行 `plan` 或 `analyze`，不要直接重放物理 DDL。
