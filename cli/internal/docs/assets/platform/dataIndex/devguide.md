# dataIndex 完整使用指南

`dataIndex` 是一级 Data Extension Domain，用于查看、规划、创建、分析和安全优化业务数据索引。所有命令只接受对象和字段 API 名，不接受物理表名、物理列名或 SQL。

## 1. 使用前准备

- 使用 MSAPI 包，或让 Universal 包明确选择 MSAPI provider；UIAPI 不支持该 Domain。
- MetadataService 版本不低于 `1.1.71`，并启用 `MDS_BUSINESS_DATA_INDEX_ENABLED=true`。
- 使用技能包内置 CLI，并先确认 provider：

```bash
cloudcc --version
cloudcc doctor provider <projectPath>
cloudcc domain dataIndex
```

`[projectPath]` 表示可选项目目录。在 CloudCC 项目根目录执行时可以省略；在其他目录执行时应显式传入，避免使用错误环境。

所需 scope：

- `data:index:read`：查看索引、规划、分析、读取优化方案和任务状态。
- `data:index:write`：创建索引。
- `data:index:optimize`：执行已确认的优化建议。
- 跨组织管理仅在明确授权时使用 `data:index:admin`。

## 2. 完整命令参考

| 目的 | 命令 | 是否修改数据库 |
|---|---|---|
| 查看对象全部索引 | `cloudcc get dataIndex [projectPath] <object>` | 否 |
| 预检创建方案 | `cloudcc plan dataIndex [projectPath] <object> --fields <f1,f2> [--name <name>]` | 否 |
| 创建索引 | `cloudcc create dataIndex [projectPath] <object> --fields <f1,f2> [--name <name>] --confirm [--wait] [--poll-interval-ms <n>]` | 是 |
| 分析可优化项 | `cloudcc analyze dataIndex [projectPath] <object>` | 否 |
| 读取优化方案 | `cloudcc optimization-plan dataIndex [projectPath] <planId>` | 否 |
| 执行指定建议 | `cloudcc optimize dataIndex [projectPath] <planId> --recommendations <id1,id2> --confirm [--wait] [--poll-interval-ms <n>]` | 是 |
| 执行全部安全建议 | `cloudcc optimize dataIndex [projectPath] <planId> --all-executable --confirm [--wait] [--poll-interval-ms <n>]` | 是 |
| 查询创建或优化任务 | `cloudcc status dataIndex [projectPath] <jobId>` | 否 |

参数说明：

- `--fields`：按索引顺序传入一至四个字段 API 名，使用英文逗号分隔。
- `--name`：可选逻辑索引名；省略时由服务端生成符合目标数据库长度限制的名称。
- `--confirm`：创建和优化的强制确认开关，缺少时 CLI 在网络写请求前失败。
- `--wait`：持续轮询到 `SUCCEEDED` 或 `FAILED`；省略时立即返回 `jobId`。
- `--poll-interval-ms`：配合 `--wait` 设置正整数轮询间隔，默认 1000 毫秒。
- `--recommendations` 与 `--all-executable` 二选一；前者用于精确执行已审阅的建议 ID。

## 3. 创建索引的完整流程

以下示例为 `Account` 的 `ownerId,createdDate` 创建复合索引：

```bash
# 1. 查看现状
cloudcc get dataIndex <projectPath> Account

# 2. 只生成创建方案，不执行 DDL
cloudcc plan dataIndex <projectPath> Account \
  --fields ownerId,createdDate

# 3. 审阅方案后确认创建，并等待终态
cloudcc create dataIndex <projectPath> Account \
  --fields ownerId,createdDate --confirm --wait

# 4. 如未使用 --wait，使用 create 返回的 jobId 查询
cloudcc status dataIndex <projectPath> <jobId>

# 5. 再次回读实际索引
cloudcc get dataIndex <projectPath> Account
```

`plan` 会返回解析后的对象、字段顺序、拟用名称、数据库类型、已有覆盖关系、当前数量、平台/数据库有效上限、剩余容量、执行模式和锁风险。出现以下结果时不要直接创建：

- `existingCoverage=LEFT_PREFIX`：现有长索引已经覆盖请求字段前缀，创建会以 `data_index_redundant` 拒绝。
- `remainingCapacity=0`：已达到有效上限，应先执行优化分析。
- 大表或高写入表的锁风险较高：应安排维护窗口。

完全相同的有序字段索引已经存在时，创建按幂等成功处理，结果通常为 `ALREADY_EXISTS`。

## 4. 创建规则

- 复合索引遵循最左前缀原则；常用等值过滤字段通常放在前面，再考虑范围和排序字段。
- 字段顺序有意义：`ownerId,createdDate` 与 `createdDate,ownerId` 不是同一索引。
- 不支持唯一、表达式、函数、部分、位图、包含列和字符串前缀长度索引。
- LOB、长文本、公式、地址等复合存储类型或没有单一物理列的字段不能创建索引。
- 名称只能使用数据库安全标识符；若不确定，应省略 `--name`。
- 创建前会检查名称冲突、字段合法性、重复/覆盖关系、当前数量、平台上限和数据库硬限制。
- 默认平台上限为每表 32 个，可由 `MDS_BUSINESS_DATA_INDEX_MAX_PER_TABLE` 调整；实际有效上限取平台与数据库限制中的较小值。
- 标准索引 DDL 可能持有元数据锁并占用额外磁盘空间，大表必须评估维护窗口与回退方案。

## 5. 优化索引的完整流程

```bash
# 1. 只分析，不执行 DDL；保存返回的 planId
cloudcc analyze dataIndex <projectPath> Account

# 2. 读取并审阅方案、expiresAt 和 recommendations
cloudcc optimization-plan dataIndex <projectPath> <planId>

# 3A. 精确执行选中的安全建议
cloudcc optimize dataIndex <projectPath> <planId> \
  --recommendations <recommendationId1,recommendationId2> --confirm --wait

# 3B. 或执行方案中全部 executable=true 的建议
cloudcc optimize dataIndex <projectPath> <planId> \
  --all-executable --confirm --wait

# 4. 未等待时，用 optimize 返回的 dio... jobId 查询
cloudcc status dataIndex <projectPath> <jobId>
```

`analyze` 只生成带索引快照指纹和有效期的方案。只有平台管理、普通、非唯一、非系统保护且结构完全重复的索引才可能标记为 `executable=true`。左前缀覆盖、未使用、非平台管理、唯一、主键、约束支撑和系统索引只提供人工审阅建议。

方案过期或分析后索引结构发生变化时，执行会失败；重新运行 `analyze`，不要绕过指纹检查。

## 6. 任务结果与失败处理

- 创建任务 ID 通常以 `dix` 开头，优化任务 ID 通常以 `dio` 开头；`status` 会自动选择正确的状态端点。
- `--wait` 返回 `SUCCEEDED` 或 `FAILED` 的最终任务 JSON。自动化脚本还应检查 `status`、`outcome`、`errorCode` 和 `errorMessage`，不要只看进程退出码。
- 遇到容量、锁、键长、权限或空间错误时，先按稳定错误码修复原因，再重新执行 `plan` 或 `analyze`。

## 7. 回滚与删除边界

CLI 不提供任意删除索引命令。优化只会删除方案中可执行的安全重复索引，并在执行前重新验证快照。若需要恢复被优化删除的平台索引，使用原对象和字段 API 名重新执行 `plan` 与 `create`；不要手写物理 DDL。

更多专题文档：

```bash
cloudcc doc dataIndex optimization
cloudcc doc dataIndex database-limits
cloudcc doc dataIndex troubleshooting
```
