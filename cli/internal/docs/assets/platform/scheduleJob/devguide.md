# CloudCC 定时作业（scheduleJob）开发指南

本指南说明 `scheduleJob` 模块在 cloudcc-cli
中的能力边界、命令与参数，以及二次开发注意事项。

---

## 一、模块定位

`scheduleJob` 只负责“作业管理”相关能力：

- 保存作业（新建/编辑）：`create`
- 查询作业列表：`get`
- 查询作业详情：`detail`
- 删除作业：`delete`
- 文档查看：`doc`

setup-svc 的真实资源名是 `schedulAbleprg`，不是 `scheduleJob`。CLI 路由如下：

- `get`：`/api/schedulAbleprg/list`
- `detail`：`/api/schedulAbleprg/edit`
- `create` / `update`：`/api/schedulAbleprg/save`
- `delete`：`/api/schedulAbleprg/delete`
- `getList`：`/api/lookup/getLookupData`，默认 `prefix=ccp`

---

## 二、CLI 命令说明

### 1) 保存作业（新建/编辑）

```bash
cloudcc create scheduleJob <projectPath> <encodedBodyJson>
```

参数说明：

| 参数名称 | 是否必须 | 备注 |
| :--- | :--- | :--- |
| `projectPath` | 否 | 项目路径，默认当前目录 |
| `encodedBodyJson` | 是 | URI 编码后的保存参数 JSON |

新建定时作业前提条件（按你提供的前端文档口径）：

- 必填：`name`、`prgid`、`frequency`、`startdate`、`enddate`、`executetime`
- 新建时：`id = ""`（编辑时传作业 id）
- `frequency` 仅支持：`weekly` / `monthly`
- 条件必填：
  - 周频（`frequency = "weekly"`）：`weeks` 必填（逗号分隔字符串）
  - 月频（`frequency = "monthly"`）：`monthtype` 必填，且
    - `monthtype = "day"` 时 `days` 必填（`1~31` 或 `last`）
    - `monthtype = "week"` 时 `weeknum` 与 `mweek` 必填
- `startdate` / `enddate` 格式必须为 `yyyy-MM-dd`
- `executetime` 取值 `"0"`~`"23"`
- `prgid`：新建前必须先查定时作业能够使用的定时类列表：
  - 先执行命令：`cloudcc getList scheduleJob [projectPath]`获取id

参数组装规则（提交保存前必须按此归一化）：

- 当 `frequency = "monthly"`：
  - 固定清空周频字段：`weeks = ""`
  - 若 `monthtype = "week"`：`days = ""`，保留 `weeknum`、`mweek`
  - 若 `monthtype = "day"`：保留 `days`，并清空 `weeknum = ""`、`mweek = ""`
- 当 `frequency = "weekly"`：
  - 若调用方内部是数组，先转为逗号串：`weeks = weeks.join(",")`
  - 固定清空月频字段：`monthtype = ""`、`days = ""`、`weeknum = ""`、`mweek = ""`
- 日期边界校验：
  - `startdate` / `enddate` 均需 `yyyy-MM-dd`
  - `enddate` 年份需 `< 2038`（与前端行为保持一致）

示例（周频）：

```json
{
  "id": "",
  "name": "每周同步作业",
  "prgid": "ccp_job_001",
  "frequency": "weekly",
  "weeks": "Mon,Wed,Fri",
  "monthtype": "",
  "days": "",
  "weeknum": "",
  "mweek": "",
  "startdate": "2026-03-25",
  "enddate": "2026-12-31",
  "executetime": "9"
}
```

示例（每天上午 10:00 执行一次，已实际验证可创建成功）：

1. 先查询可用 `prgid`

```bash
cloudcc getList scheduleJob "/path/to/project"
```

返回示例：

```json
[
  {
    "sortfieldsql": 1776329019000,
    "name": "ScheduleRetryTimer",
    "id": "ccp202689F3015BpWah8"
  }
]
```

2. 再创建定时作业

```bash
cloudcc create scheduleJob "/path/to/project" '{"id":"","name":"daily-10am-job","prgid":"ccp202689F3015BpWah8","frequency":"weekly","weeks":"Mon,Tue,Wed,Thu,Fri,Sat,Sun","monthtype":"","days":"","weeknum":"","mweek":"","startdate":"2026-04-16","enddate":"2037-12-31","executetime":"10"}'
```

3. 创建后查询结果

```bash
cloudcc get scheduleJob "/path/to/project"
```

结果示例：

```json
[
  {
    "prgname": "ScheduleRetryTimer",
    "name": "daily-10am-job",
    "id": "2026F8F8AC2E294x4wsk",
    "startdate": "2026-04-16",
    "enddate": "2037-12-31",
    "executetime": "上午 10:00",
    "prgid": "ccp202689F3015BpWah8"
  }
]
```

注意：

- 当前平台口径下，“每天执行一次”应使用 `frequency: "weekly"` 加 `weeks: "Mon,Tue,Wed,Thu,Fri,Sat,Sun"`。
- `executetime` 传小时值字符串即可，例如上午 10 点传 `"10"`。
- `getList scheduleJob` 返回的是可用于创建作业的 `prgid` 列表，不是已创建的定时作业列表。

---

### 2) 查询定时作业列表

```bash
cloudcc get scheduleJob <projectPath> [encodedCondJson]
```

参数说明：

| 参数              | 必填 | 说明                      |
| ----------------- | ---- | ------------------------- |
| `projectPath`     | 否   | 项目路径，默认当前目录    |
| `encodedCondJson` | 否   | URI 编码后的查询条件 JSON |

---

### 3) 查询作业详情

```bash
cloudcc detail scheduleJob <jobId> [projectPath]
```

参数说明：

| 参数          | 必填 | 说明                   |
| ------------- | ---- | ---------------------- |
| `jobId`       | 是   | 定时作业 ID            |
| `projectPath` | 否   | 项目路径，默认当前目录 |

---

### 4) 查询定时作业能够使用的定时类列表

```bash
cloudcc getList scheduleJob [projectPath] 
```

参数说明：

| 参数 | 必填 | 说明 |
| ------------- | ---- | ---------------------- |
| `projectPath` | 否   | 项目路径，默认当前目录 |

---

### 5) 删除作业

```bash
cloudcc delete scheduleJob <jobId> [projectPath]
```

参数说明：

| 参数          | 必填 | 说明                   |
| ------------- | ---- | ---------------------- |
| `jobId`       | 是   | 待删除作业 ID          |
| `projectPath` | 否   | 项目路径，默认当前目录 |

---

### 6) 查看文档

```bash
cloudcc doc platform/scheduleJob <introduction|devguide>
```

---

## 三、关联文档

- `cloudcc doc platform/timer introduction`
- `cloudcc doc platform/timer devguide`

---
