# dataIndex 优化说明

## 两阶段安全协议

```bash
# 分析只生成方案
cloudcc analyze dataIndex [projectPath] <object>

# 审阅方案
cloudcc optimization-plan dataIndex [projectPath] <planId>

# 精确执行选中的可执行建议
cloudcc optimize dataIndex [projectPath] <planId> \
  --recommendations <id1,id2> --confirm --wait

# 或执行方案中全部 executable=true 的建议
cloudcc optimize dataIndex [projectPath] <planId> \
  --all-executable --confirm --wait
```

`analyze` 返回 `planId`、对象、数据库类型、当前全部索引、快照指纹、当前/有效上限、剩余容量、`recommendations[]`、`createdAt` 和 `expiresAt`。用户明确确认前不会执行任何 DDL。

每条建议至少应审阅：

- `recommendationId`：传给 `--recommendations` 的 ID。
- `type`：建议类型。
- `executable`：是否允许 CLI 自动执行。
- 目标索引及作为判断依据的索引。
- 原因与安全说明。

## 建议类型

- `DROP_DUPLICATE`：有序字段定义完全重复。只有待删除索引由平台管理，且不是唯一、系统或约束支撑索引时才可执行。
- `REVIEW_LEFT_PREFIX`：短索引是长索引的最左前缀。短索引可能更小、更快，因此只提示结合查询负载人工判断。
- `REVIEW_INDEX_LIMIT`：索引数量达到平台或数据库有效上限，应先处理安全重复项，再评估其他冗余项。

不会自动执行猜测性合并、字段重排或“未使用索引”删除。非平台管理索引、唯一索引、主键、系统索引和约束支撑索引永远只报告。

## 方案失效与并发保护

- 方案超过 `expiresAt` 后返回 `data_index_optimization_plan_expired`，必须重新分析。
- 执行前会重新读取全部索引并比对快照指纹；任何并发 DDL 都会使旧方案失效。
- 只能执行当前方案内且 `executable=true` 的建议；人工审阅项返回 `data_index_recommendation_not_executable`。
- `--recommendations` 适合生产环境精确选择；`--all-executable` 适合确认方案中所有安全重复项都应处理的场景。

优化任务 ID 通常以 `dio` 开头。省略 `--wait` 时保存返回的 `jobId`，随后执行：

```bash
cloudcc status dataIndex [projectPath] <jobId>
cloudcc get dataIndex [projectPath] <object>
```
