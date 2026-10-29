# dataIndex 优化说明

## 分析与人工确认删除协议

```bash
# 分析只生成方案
cloudcc analyze dataIndex [projectPath] <object>

# 审阅方案
cloudcc optimization-plan dataIndex [projectPath] <planId>

# 使用建议目标在 get/analyze 返回中的 indexRef 生成删除计划
cloudcc plan-delete dataIndex [projectPath] <object> \
  --indexes <indexRef,...> --reason <text>

# 人工审阅后使用独立批准凭据执行
cloudcc approve-delete dataIndex [projectPath] <deletePlanId> \
  --challenge <approvalChallenge> --fingerprint <fingerprint> --wait
```

`analyze` 仍只提供建议。任何建议若需要删除，都必须转换为独立删除计划；旧 `optimize --confirm` 不再执行 DDL。删除计划返回目标、阻断原因、风险、删除前数量、预计剩余数量、表数据量、快照指纹、challenge 和有效期。表数据量默认以 1,000,000 行为谨慎阈值，超过时返回 `CAUTION_LARGE_TABLE`；为避免提示逻辑扫描整张超大表，数量最多统计到阈值 + 1，`tableRowCountExact=false` 表示“至少该数量”。

每条建议至少应审阅：

- `recommendationId`：传给 `--recommendations` 的 ID。
- `type`：建议类型。
- `executable`：是否允许 CLI 自动执行。
- 目标索引及作为判断依据的索引。
- 原因与安全说明。

## 建议类型

- `DROP_DUPLICATE`：有序字段定义完全重复，仅作为选择建议；实际删除仍需独立计划和人工批准。
- `REVIEW_LEFT_PREFIX`：短索引是长索引的最左前缀。短索引可能更小、更快，因此只提示结合查询负载人工判断。
- `REVIEW_INDEX_LIMIT`：索引数量达到平台或数据库有效上限，应先处理安全重复项，再评估其他冗余项。

主键和唯一索引永远阻断。其他索引不再因名称前缀或未纳管而永久阻断，但相关风险必须进入删除计划和审计日志。

## 方案失效与并发保护

- 方案超过 `expiresAt` 后返回 `data_index_optimization_plan_expired`，必须重新分析。
- 执行前会重新读取全部索引并比对快照指纹；任何并发 DDL 都会使旧方案失效。
- 删除批准只能执行当前删除计划中 `deletable=true` 的目标；批准前后都会重新检查主键、唯一性和指纹。
- challenge、fingerprint、approve scope、计划单次使用状态缺一不可。

优化任务 ID 通常以 `dio` 开头。省略 `--wait` 时保存返回的 `jobId`，随后执行：

```bash
cloudcc status dataIndex [projectPath] <jobId>
cloudcc get dataIndex [projectPath] <object>
```
