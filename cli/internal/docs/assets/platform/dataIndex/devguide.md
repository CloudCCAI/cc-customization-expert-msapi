# dataIndex 使用指南

`dataIndex` 是一级 Skill Domain，不是 Object 文档中的附属能力。

## 能力入口

```bash
cloudcc doc dataIndex
cloudcc get dataIndex <projectPath> Account
cloudcc plan dataIndex <projectPath> Account --fields ownerId,createdDate
cloudcc create dataIndex <projectPath> Account --fields ownerId,createdDate --confirm --wait
cloudcc analyze dataIndex <projectPath> Account
cloudcc optimization-plan dataIndex <projectPath> <planId>
cloudcc optimize dataIndex <projectPath> <planId> --all-executable --confirm --wait
cloudcc status dataIndex <projectPath> <jobId>
```

## 创建规则

- `--fields` 必须按索引顺序提供一至四个字段 API 名。
- 复合索引遵循最左前缀原则；优先把常用等值过滤字段放在前面，再考虑范围、排序字段。
- 同一有序字段定义已经存在时按幂等成功处理；现有更长索引已覆盖请求前缀时拒绝冗余创建。
- 不支持唯一、表达式、函数、部分、位图、包含列和字符串前缀长度索引。
- LOB、长文本、公式、复合存储或没有单一物理列的字段不能创建索引。
- 创建前检查名称、字段、重复/覆盖关系、当前索引数、平台上限和数据库硬限制。
- 默认每表平台上限为 32，可通过 `MDS_BUSINESS_DATA_INDEX_MAX_PER_TABLE` 调整；MySQL InnoDB 还受 64 个二级索引硬限制。
- 大表 DDL 可能持有元数据锁，应该安排维护窗口。

## 优化规则

`analyze` 只生成带有效期和索引指纹的方案，不执行删除。自动优化只允许删除平台管理、普通、非唯一、非系统保护且结构完全重复的索引。

左前缀覆盖、未使用、非平台管理、唯一、主键、约束支撑和系统索引只提供审阅建议，不自动修改。索引结构在分析后发生变化时，旧方案失效，必须重新分析。

## 权限

- `data:index:read`：列表、计划、分析和状态读取。
- `data:index:write`：创建索引。
- `data:index:optimize`：确认后执行优化。
- `data:*` 和现有数据管理员权限继续按服务端兼容策略处理。

## 回滚

创建出的索引不会因为关闭功能自动删除。优化删除只针对平台管理的安全重复索引；如需恢复，使用原对象/字段 API 名重新执行 `plan` 和 `create`，不要手写物理 DDL。
