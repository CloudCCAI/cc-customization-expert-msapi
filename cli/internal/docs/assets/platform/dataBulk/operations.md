# dataBulk 操作规则与示例

执行所有操作前先运行：

```bash
cloudcc schema dataBulk [projectPath] <object>
```

只使用 write-schema 中 `classification=DIRECT_WRITABLE` 的字段。系统字段、创建/修改审计字段、逻辑删除字段、公式和自动编号由服务端管理。

## INSERT

- 输入不得包含 `id`。
- 每行至少包含一个可写业务字段，并满足数据库必填约束。

```bash
cloudcc submit dataBulk <projectPath> Account INSERT @accounts.ndjson \
  --format ndjson --wait --output-dir ./bulk-results/insert
```

## UPDATE_BY_ID

- 每行必须包含目标租户真实回读的 `id`。
- 除 `id` 外至少包含一个待更新的 `DIRECT_WRITABLE` 字段。
- 不存在、已删除或不属于当前租户的 ID 会逐行失败。

```ndjson
{"id":"<recordId1>","name":"Updated A"}
{"id":"<recordId2>","name":"Updated B"}
```

```bash
cloudcc submit dataBulk <projectPath> Account UPDATE_BY_ID @accounts-update.ndjson \
  --format ndjson --wait --output-dir ./bulk-results/update
```

## UPSERT_BY_ID

- 行内有真实 `id` 时更新该记录。
- 行内没有 `id` 时按新增记录处理。
- 不得构造、猜测或复用其他租户的 ID。

```bash
cloudcc submit dataBulk <projectPath> Account UPSERT_BY_ID @accounts-upsert-id.ndjson \
  --format ndjson --wait --output-dir ./bulk-results/upsert-id
```

## UPSERT_BY_EXTERNAL_KEY

- 必须传 `--external-key-field <apiName>`，每行必须包含非空外部键值。
- 外部键字段必须为 write-schema 允许写入的字段。
- 输入不得包含 `id`。
- 同一批次的外部键应唯一；目标库匹配多条有效记录时返回 `bulk_external_key_ambiguous`。

```bash
cloudcc submit dataBulk <projectPath> Account UPSERT_BY_EXTERNAL_KEY @accounts.csv \
  --format csv --external-key-field external_id --wait \
  --output-dir ./bulk-results/upsert-external
```

## DELETE_BY_ID

- 每行只能包含 `id`，不得附带其他字段。
- `id` 必须来自目标租户真实回读。
- 需要 `data:bulk:delete` scope。
- 执行平台逻辑删除，不代表可自动恢复。

```ndjson
{"id":"<recordId1>"}
{"id":"<recordId2>"}
```

```bash
cloudcc submit dataBulk <projectPath> Account DELETE_BY_ID @accounts-delete.ndjson \
  --format ndjson --wait --output-dir ./bulk-results/delete
```

## 结果核对

每次提交后至少核对：

- summary 中所有 `jobs[].status` 是否为预期终态。
- `totalRecords = successfulRecords + failedRecords`。
- success/failed 文件和 `results` 中的行数、`recordId`、错误码是否一致。
- 分片任务的 `absoluteRowNumber` 是否能对应原始输入行。

失败行修复后，可以对 `FAILED` 或 `PARTIAL_SUCCESS` 作业执行 `retry`；该动作只重试失败行。若要修改失败行内容，应生成新的只含失败数据的文件并提交新作业。
