# dataIndex 完整使用指南

`dataIndex` 是一级 Data Extension Domain，用于查看、规划、创建、分析和安全优化业务数据索引。所有命令只接受对象和字段 API 名，不接受物理表名、物理列名或 SQL。

## 1. 使用前准备

- 使用 MSAPI 包，或让 Universal 包明确选择 MSAPI provider；UIAPI 不支持该 Domain。
- MetadataService 版本不低于 `1.1.93`，并启用 `MDS_BUSINESS_DATA_INDEX_ENABLED=true`。
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
- `data:index:delete:plan`：生成索引删除计划，不执行 DDL。
- `data:index:delete:approve`：人工批准并启动删除任务；只应授予交互式人工操作账号。
- 跨组织管理仅在明确授权时使用 `data:index:admin`。

## 2. 完整命令参考

| 目的 | 命令 | 是否修改数据库 |
|---|---|---|
| 查看对象全部索引 | `cloudcc get dataIndex [projectPath] <object>` | 否 |
| 预检创建方案 | `cloudcc plan dataIndex [projectPath] <object> --fields <f1,f2> [--name <name>]` | 否 |
| 创建索引 | `cloudcc create dataIndex [projectPath] <object> --fields <f1,f2> [--name <name>] --confirm [--wait] [--poll-interval-ms <n>]` | 是 |
| 分析可优化项 | `cloudcc analyze dataIndex [projectPath] <object>` | 否 |
| 读取优化方案 | `cloudcc optimization-plan dataIndex [projectPath] <planId>` | 否 |
| 生成删除计划 | `cloudcc plan-delete dataIndex [projectPath] <object> --indexes <indexRef,...> --reason <text> [--ticket <id>]` | 否 |
| 读取删除计划 | `cloudcc delete-plan dataIndex [projectPath] <planId>` | 否 |
| 人工批准删除 | `cloudcc approve-delete dataIndex [projectPath] <planId> --challenge <value> --fingerprint <value> [--wait] [--poll-interval-ms <n>]` | 是 |
| 查询删除审计事件 | `cloudcc delete-events dataIndex [projectPath] <planId>` | 否 |
| 查询创建或删除任务 | `cloudcc status dataIndex [projectPath] <jobId>` | 否 |

参数说明：

- `--fields`：按索引顺序传入一至四个字段 API 名，使用英文逗号分隔。
- `--name`：可选逻辑索引名；省略时由服务端生成符合目标数据库长度限制的名称。
- `--confirm`：仅用于创建索引；删除不接受一步 `--confirm`。
- `--wait`：持续轮询到 `SUCCEEDED` 或 `FAILED`；省略时立即返回 `jobId`。
- `--poll-interval-ms`：配合 `--wait` 设置正整数轮询间隔，默认 1000 毫秒。
- `--indexes` 使用 `get`/`analyze` 返回的稳定 `indexRef`，不得传物理表名、列名或 SQL。
- `--challenge` 与 `--fingerprint` 必须逐字来自已人工审阅的删除计划。

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

## 5. 删除索引的完整流程

```bash
# 1. 查看 indexRef、唯一性、主键和风险标签
cloudcc get dataIndex <projectPath> Account

# 2. 生成不可变计划，不执行 DDL
cloudcc plan-delete dataIndex <projectPath> Account \
  --indexes <indexRef1,indexRef2> --reason "已审阅的历史索引清理" --ticket CHANGE-001

# 3. 人工读取并审阅 targets、beforeCount、predictedAfterCount、表数据量、challenge、fingerprint 和 expiresAt
cloudcc delete-plan dataIndex <projectPath> <planId>

# 4. 使用具备 approve scope 的人工凭据发起独立批准
cloudcc approve-delete dataIndex <projectPath> <planId> \
  --challenge <approvalChallenge> --fingerprint <fingerprint> --wait

# 5. 查询任务和追加式审计事件
cloudcc status dataIndex <projectPath> <jobId>
cloudcc delete-events dataIndex <projectPath> <planId>
```

主键和唯一索引始终为 `BLOCKED`。`sys_*`、`index_*`、`idx_createdate_*`、`idx_c<数字>_ownerid`、未纳管和平台默认索引只增加风险标签，不再永久阻断非唯一索引。删除计划可以同时返回可删除与阻断项，只对 `deletable=true` 项计算预计剩余数量。

删除计划同时返回 `tableRowCount`、`tableRowCountExact`、`rowCountWarningThreshold`、`rowCountWarning`、`rowCountWarningCode` 和提示文本。默认谨慎阈值为 1,000,000 行，可通过 `MDS_BUSINESS_DATA_INDEX_DELETE_ROW_WARNING_THRESHOLD` 调整。服务最多读取“阈值 + 1”行：未超阈值时数量精确；超过阈值时数量表示“至少这么多行”，并返回 `CAUTION_LARGE_TABLE`，人工应评估查询计划、写入开销和维护窗口后再批准。若计划时未告警、批准时数据量已经跨过阈值，批准返回 `data_index_delete_row_count_changed`，必须重新生成并审阅计划。警告不替代主键/唯一索引硬阻断，也不自动拒绝已经审阅过大表警告的审批。

计划请求永远不执行 DDL。批准必须是第二个请求，要求独立 `data:index:delete:approve` scope、一次性 challenge 和指纹；方案过期、已使用或索引结构变化时必须重新计划并重新人工批准。旧 `optimize --confirm` 不再执行删除。

## 6. 任务结果与失败处理

- 创建任务 ID 通常以 `dix` 开头，删除任务 ID 以 `dij` 开头；`status` 会自动选择正确的状态端点。
- 删除终态为 `VERIFIED`、`PARTIAL` 或 `FAILED`。DDL 不承诺批量事务原子性，必须检查每个 item、实际剩余数量、前后指纹和回读事件。
- 遇到容量、锁、键长、权限或空间错误时，先按稳定错误码修复原因，再重新执行 `plan` 或 `analyze`。

## 7. 回滚与删除边界

除主键和唯一索引外，其他索引可进入人工确认删除流程。服务保存删除前定义和逐项结果，但数据库 DDL 通常自动提交，不宣称失败后自动原子回滚；需要恢复时按保存定义重新 `plan`/`create`，并对恢复操作重新确认。不要手写物理 DDL。

更多专题文档：

```bash
cloudcc doc dataIndex optimization
cloudcc doc dataIndex database-limits
cloudcc doc dataIndex troubleshooting
```
