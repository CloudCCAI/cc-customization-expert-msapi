# dataIndex 故障排查

| 错误码 | 原因 | 处理方式 |
|---|---|---|
| `data_index_limit_exceeded` | 达到平台或数据库索引数量上限 | 执行 `cloudcc analyze dataIndex` 并审阅冗余建议 |
| `data_index_key_too_long` | 字段组合超过索引键字节限制 | 减少或调整字段；当前不支持字符串前缀索引 |
| `data_index_name_conflict` | 同名索引已有不同字段定义 | 省略名称让系统生成，或更换逻辑名称 |
| `data_index_redundant` | 已有长索引覆盖请求的最左前缀 | 查看现有索引，无需重复创建 |
| `data_index_permission_denied` | 数据库账户缺少 DDL 权限 | 由管理员补充最小权限后重试 |
| `data_index_lock_timeout` | DDL 无法获得锁 | 在低峰维护窗口重试 |
| `data_index_disk_full` | 表空间或磁盘不足 | 扩容或清理空间后重试 |
| `data_index_optimization_plan_stale` | 分析后索引发生变化 | 重新执行 `analyze dataIndex` |
| `data_index_recommendation_not_executable` | 建议只能人工审阅 | 不要强制执行；结合查询负载人工处理 |
| `data_index_protected` | 目标是系统、唯一、约束或非平台索引 | 保留该索引，必要时走独立 DBA 评审 |

任务失败时使用 `cloudcc status dataIndex <projectPath> <jobId>` 查看稳定错误码和安全说明。服务不会返回原始 SQL、连接信息或凭据。

