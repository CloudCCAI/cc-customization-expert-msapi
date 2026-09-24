# dataIndex 优化说明

## 两阶段安全协议

```bash
cloudcc analyze dataIndex <projectPath> Account
cloudcc optimization-plan dataIndex <projectPath> <planId>
cloudcc optimize dataIndex <projectPath> <planId> --recommendations rec_xxx --confirm --wait
```

分析结果包含全部索引、快照指纹、有效期、容量和建议。用户明确确认前不会执行任何 DDL。

## 建议类型

- `DROP_DUPLICATE`：有序字段定义完全重复。只有目标索引由平台管理且不是唯一/系统/约束索引时可自动执行。
- `REVIEW_LEFT_PREFIX`：短索引是长索引的最左前缀。短索引可能更小、更快，因此仅提示人工结合查询负载判断。
- `REVIEW_INDEX_LIMIT`：索引数量达到平台或数据库有效上限，应先清理重复和冗余索引。

不做猜测性合并、字段重排或自动删除未使用索引。非平台管理索引永远只报告。执行前会重新读取索引并比对指纹，任何并发 DDL 都会使方案失败并要求重新分析。

