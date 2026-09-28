# dataBulk 完整使用指南

`dataBulk` 是一级 Data Extension Domain，用于按逻辑对象和字段 API 名提交、查询和管理业务数据批量作业。它直接写入业务数据表，不走元数据 `plan/apply`。

## 1. 使用前准备

- 使用 MSAPI 包，或让 Universal 包明确选择 MSAPI provider；UIAPI 不支持该 Domain。
- MetadataService 版本不低于 `1.1.71`，并启用 `MDS_BUSINESS_DATA_BULK_ENABLED=true`。
- 使用技能包内置 CLI，并先确认 provider：

```bash
cloudcc --version
cloudcc doctor provider <projectPath>
cloudcc domain dataBulk
```

`[projectPath]` 是可选项目目录。在项目根目录执行时可以省略；在其他目录执行时应显式传入。

所需 scope：

- `data:bulk:read`：读取 write-schema、作业状态和逐行结果。
- `data:bulk:write`：提交、恢复、重试和取消作业。
- `data:bulk:delete`：提交 `DELETE_BY_ID`。
- 跨组织管理仅在明确授权时使用 `data:bulk:admin`。

## 2. 完整命令参考

```bash
cloudcc schema dataBulk [projectPath] <object>

cloudcc submit dataBulk [projectPath] <object> <operation> <recordsJson|@file> \
  [--format json|ndjson|csv] \
  [--external-key-field <apiName>] \
  [--chunk-size <n>] [--wait] [--poll-interval-ms <n>] \
  [--output-dir <dir>]

cloudcc status  dataBulk [projectPath] <jobId>
cloudcc results dataBulk [projectPath] <jobId>
cloudcc resume  dataBulk [projectPath] <jobId>
cloudcc retry   dataBulk [projectPath] <jobId>
cloudcc cancel  dataBulk [projectPath] <jobId>
```

参数说明：

- `<object>`：标准或自定义对象 API 名。
- `<operation>`：`INSERT`、`UPDATE_BY_ID`、`UPSERT_BY_ID`、`UPSERT_BY_EXTERNAL_KEY` 或 `DELETE_BY_ID`；连字符写法会规范化为下划线。
- `<recordsJson|@file>`：非空 JSON 数组、包含 `records[]` 的 JSON 对象，或使用 `@` 引用的 JSON/NDJSON/CSV 文件。
- `--format`：默认 `json`；文件扩展名不会自动决定格式。
- `--external-key-field`：仅 `UPSERT_BY_EXTERNAL_KEY` 必填，值为字段 API 名。
- `--chunk-size`：正整数。把输入加载到 CLI 后按该大小提交多个 inline job；不能绕过服务端 `maxInlineRecords`。
- `--wait`：轮询全部 job 至终态；生产导入建议使用。
- `--poll-interval-ms`：正整数轮询间隔，默认 1000 毫秒。
- `--output-dir`：写出 summary/success/failed JSON；应与 `--wait` 一起使用，确保结果文件是终态数据。

## 3. 先读取 write-schema

```bash
cloudcc schema dataBulk <projectPath> Account
```

提交前必须确认：

- 对象是否正确，以及对象的自动编号状态。
- 每个输入字段是否标记为 `DIRECT_WRITABLE`。
- 操作要求的 `id` 或外部键字段是否可用。
- `maxInlineRecords`，用于 JSON/客户端分片。

不要猜测字段 API 名，也不要提交系统管理、只读、公式、审计、逻辑删除或自动编号字段。

## 4. 输入文件格式

### JSON

顶层可以是数组：

```json
[
  {"name": "Account A", "external_id": "A-001"},
  {"name": "Account B", "external_id": "A-002"}
]
```

也可以是包含 `records[]` 的对象：

```json
{"records":[{"name":"Account A"},{"name":"Account B"}]}
```

推荐使用文件，避免不同 shell 的引号差异：

```bash
cloudcc submit dataBulk <projectPath> Account INSERT @accounts.json \
  --format json --wait --output-dir ./bulk-results
```

JSON 会由 CLI 读取并按 `--chunk-size` 或 write-schema 的 `maxInlineRecords` 分片。

### NDJSON

每个非空行必须是一个 JSON 对象：

```ndjson
{"name":"Account A","external_id":"A-001"}
{"name":"Account B","external_id":"A-002"}
```

大文件不要传 `--chunk-size`，CLI 会使用服务端流式端点，不把整个文件读入内存：

```bash
cloudcc submit dataBulk <projectPath> Account INSERT @accounts.ndjson \
  --format ndjson --wait --output-dir ./bulk-results
```

### CSV

第一行必须是字段 API 名；每行列数必须一致，字段值按 CSV 文本传入：

```csv
name,external_id
Account A,A-001
Account B,A-002
```

大文件流式上传：

```bash
cloudcc submit dataBulk <projectPath> Account UPSERT_BY_EXTERNAL_KEY @accounts.csv \
  --format csv --external-key-field external_id --wait \
  --output-dir ./bulk-results
```

对 NDJSON/CSV 指定 `--chunk-size` 会改为“CLI 读取完整文件后分片提交 inline job”，不再是文件流式上传。只有确实需要客户端分片并确认本机内存足够时才这样做。

## 5. 五种操作

| 操作 | 输入规则 | 常见用途 |
|---|---|---|
| `INSERT` | 至少一个可写业务字段；不得传 `id` | 批量新增 |
| `UPDATE_BY_ID` | 每行必须包含真实回读 `id` 和至少一个可写字段 | 更新已有记录 |
| `UPSERT_BY_ID` | 有 `id` 时更新，无 `id` 时按新增字段创建；不得构造伪 ID | ID 已知的混合新增/更新 |
| `UPSERT_BY_EXTERNAL_KEY` | 必须提供 `--external-key-field`，且每行包含该字段；不得传 `id` | 外部系统幂等同步 |
| `DELETE_BY_ID` | 每行只能包含真实 `id`；需要 `data:bulk:delete` | 逻辑删除 |

详细规则与示例：

```bash
cloudcc doc dataBulk operations
```

## 6. 标准生产流程

```bash
# 1. 确认 provider 和 schema
cloudcc doctor provider <projectPath>
cloudcc schema dataBulk <projectPath> Account

# 2. 先用少量数据执行并等待终态
cloudcc submit dataBulk <projectPath> Account INSERT @accounts-smoke.ndjson \
  --format ndjson --wait --output-dir ./bulk-results/smoke

# 3. 核对 summary、逐行结果和目标租户数据
cloudcc status dataBulk <projectPath> <jobId>
cloudcc results dataBulk <projectPath> <jobId>

# 4. 提交正式大文件
cloudcc submit dataBulk <projectPath> Account INSERT @accounts.ndjson \
  --format ndjson --wait --output-dir ./bulk-results/production
```

`--output-dir` 生成：

- `<object>_<operation>_summary.json`
- `<object>_<operation>_success.json`
- `<object>_<operation>_failed.json`

分片提交时 summary 的 `jobs[]` 给出每个 `jobId`、状态和绝对行范围；逐行结果增加 `jobId` 与 `absoluteRowNumber`，用于定位原文件。

自动化脚本必须检查 `jobs[].status`、`successfulRecords`、`failedRecords` 和结果数量，不能只依赖 CLI 退出码。

## 7. 作业状态与管理动作

终态包括 `SUCCEEDED`、`PARTIAL_SUCCESS`、`FAILED` 和 `CANCELED`。

- `status`：随时读取汇总状态。
- `results`：读取逐行 `rowNumber`、`recordId`、`status`、错误码和错误信息。
- `resume`：重新调度未终态、未取消且未失败的中断作业；成功或部分成功时按幂等返回当前状态。
- `retry`：只适用于 `FAILED` 或 `PARTIAL_SUCCESS`，重新处理失败行，不重做成功行。
- `cancel`：请求取消尚未终态的作业；已完成行不会回滚，`CANCELED` 作业不能恢复。

```bash
cloudcc status dataBulk <projectPath> <jobId>
cloudcc results dataBulk <projectPath> <jobId>
cloudcc retry dataBulk <projectPath> <jobId>
```

## 8. 重要业务边界

dataBulk 不执行验证规则、触发器、查重过滤器、共享规则或工作流，也不提供 MetadataService 元数据回滚。需要这些平台运行时语义的数据写入应使用 OpenAPI。

写入前应完成业务校验、去重、权限审批、目标租户确认和备份。部分成功时先用 `results` 获取成功记录 ID，再修复失败行并执行 `retry` 或提交单独的补偿作业；禁止把重试当作回滚。

更多文档：

```bash
cloudcc doc dataBulk operations
cloudcc doc dataBulk limits
cloudcc doc dataBulk troubleshooting
```
