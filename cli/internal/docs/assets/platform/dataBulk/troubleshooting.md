# dataBulk 故障处理

| 错误码或现象 | 原因 | 处理 |
|---|---|---|
| Domain unavailable / requires MSAPI | 当前是 UIAPI，或 Universal 选择了 UIAPI | 运行 `cloudcc doctor provider <projectPath>`，改用 MSAPI 或配置兼容 MetadataService |
| 功能端点不存在或被关闭 | MetadataService 版本过低，或未启用 Bulk | 确认版本不低于 `1.1.71`，并设置 `MDS_BUSINESS_DATA_BULK_ENABLED=true` |
| `bulk_inline_limit_exceeded` | inline job 超过服务端上限 | 读取 schema，减小 `--chunk-size`；大文件改用不带 `--chunk-size` 的 NDJSON/CSV 流式上传 |
| 字段不可写 | 字段不在 `DIRECT_WRITABLE` 集合 | 删除该字段，或改用能执行平台业务规则的 OpenAPI |
| `externalKeyField is required` | 外部键 upsert 未指定字段 | 增加 `--external-key-field <apiName>`，并确保每行包含该字段 |
| `bulk_external_key_ambiguous` | 目标库中同一外部键匹配多条有效记录 | 先治理重复数据，不要盲目重试 |
| CSV 解析失败 | 表头为空/重复、列数不一致或引号不合法 | 修复 CSV；表头必须使用字段 API 名 |
| NDJSON 指定行失败 | 该行不是合法 JSON 对象 | 根据错误行号修复，空行可以保留 |
| `FAILED` | 作业或全部行失败 | 先查看 `status` 和 `results`；修复根因后执行 `retry` 或提交修正文件 |
| `PARTIAL_SUCCESS` | 部分行成功、部分行失败 | 保存成功记录 ID，只修复失败行；可使用 `retry` 重试原失败行 |
| `bulk_job_not_retryable` | 作业不在 `FAILED`/`PARTIAL_SUCCESS` | 先查询状态，不要对运行中或已成功作业重试 |
| `bulk_job_not_resumable` | 作业已取消，或失败作业错误地使用 resume | `CANCELED` 不可恢复；`FAILED` 使用 `retry` |
| 作业中断且未终态 | worker 或实例中断 | 查询状态；确认非失败/非取消后执行 `resume` |
| 重复数据风险 | 未使用稳定外部键、真实 ID 或未核对已成功行 | 先检查 `results`，避免重复提交完整文件 |

推荐排查顺序：

```bash
cloudcc doctor provider <projectPath>
cloudcc schema dataBulk <projectPath> <object>
cloudcc status dataBulk <projectPath> <jobId>
cloudcc results dataBulk <projectPath> <jobId>
```

生产执行前必须确认目标租户、对象、操作、输入行数、删除语义、scope 和恢复方案。终端输出与结果文件不得包含凭据。dataBulk 不提供业务事务级回滚；已成功行不会因为后续行失败、取消或重试而自动撤销。
