# dataBulk 分片、流式与容量限制

## 服务端配置

- `MDS_BUSINESS_DATA_BULK_MAX_INLINE_RECORDS`：单个 JSON inline job 最大记录数，默认 200。
- `MDS_BUSINESS_DATA_BULK_BATCH_SIZE`：服务端每个数据库 DML 批次大小。
- `MDS_BUSINESS_DATA_BULK_WORKER_CONCURRENCY`：单实例 worker 并发数。
- `MDS_BUSINESS_DATA_BULK_LEASE_SECONDS`：worker 分片租约时长。

并发、数据库锁、磁盘、连接池和自动编号容量仍受目标部署环境约束。不要仅为提高速度盲目增大并发或 batch size。

## CLI 的三种提交方式

| 输入方式 | CLI 行为 | 适用场景 |
|---|---|---|
| JSON 文本或 `@file --format json` | CLI 读取并解析全部记录，再按 `--chunk-size` 或 write-schema 的 `maxInlineRecords` 提交一个或多个 inline job | 小批量、需要客户端分片 |
| `@file --format ndjson|csv`，不传 `--chunk-size` | 使用服务端流式端点，CLI 不把整个文件读入内存 | 大文件，推荐 |
| `@file --format ndjson|csv --chunk-size <n>` | CLI 读取完整文件并按指定大小提交 inline job | 明确需要客户端分片且本机内存足够 |

`--chunk-size` 不能绕过服务端限制。若指定值超过 `maxInlineRecords`，服务端仍可能返回 `bulk_inline_limit_exceeded`。稳妥做法是先执行 `schema`，再让 CLI 自动使用返回上限，或显式选择不大于该值的分片大小。

## 等待与结果文件

- 单个不超过快速提交阈值的 inline job，在未使用 `--wait` 和 `--output-dir` 时立即返回服务端接收结果。
- 输入超过单 job 上限并自动拆分时，CLI 会等待所有分片完成后输出聚合 summary。
- 流式 NDJSON/CSV 只有使用 `--wait` 或 `--output-dir` 时才等待终态。
- 为获得可靠的 success/failed 文件，始终组合使用 `--wait --output-dir <dir>`。

## 文件建议

- 大文件优先 NDJSON；它逐行定位错误最直接。
- CSV 第一行必须是字段 API 名，所有行列数一致；逗号和引号按标准 CSV 转义，流式 CSV 的单个字段不要包含实际换行符。
- JSON 适合小批量和自动化生成，不适合超大文件，因为 CLI 需要整体解析。
- 使用独立结果目录，避免不同对象、操作或批次互相覆盖同名 summary 文件。
