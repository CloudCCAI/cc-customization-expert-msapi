# CloudCC 报表开发指南

## 1. 模块定位

`report` 与 `reportFolder` 用于管理 CloudCC 报表和报表文件夹元数据。当前 CLI 已收口到 MetadataService：

- 查询类命令调用 MetadataService 只读接口。
- 创建、更新、删除类命令只生成 MetadataService plan。
- 真正写入必须显式执行 `cloudcc apply msapi <projectPath> <planId>`。

可通过以下命令读取文档：

```bash
cloudcc doc platform/report introduction
cloudcc doc platform/report devguide
cloudcc doc report introduction
cloudcc doc report devguide
```

---

## 2. 当前支持的命令

```bash
cloudcc get report <projectPath> [folderId] [searchKeyWord] [page] [pageSize] [orderField] [orderType]
cloudcc get report <projectPath> <encodedBodyJson>
cloudcc detail report <projectPath> <reportId>
cloudcc detail report <projectPath> <encodedBodyJson>
cloudcc create report <projectPath> <encodedReportJson>
cloudcc update report <projectPath> <reportId> <encodedReportJson>
cloudcc update report <projectPath> <encodedReportJson>
cloudcc create reportTabular <projectPath> <encodedReportJson|@file>
cloudcc update reportTabular <projectPath> <reportId> <encodedReportJson|@file>
cloudcc create reportSummary <projectPath> <encodedReportJson|@file>
cloudcc update reportSummary <projectPath> <reportId> <encodedReportJson|@file>
cloudcc create reportMatrix <projectPath> <encodedMatrixReportJson|@file>
cloudcc update reportMatrix <projectPath> <reportId> <encodedMatrixReportJson|@file>
cloudcc update reportMatrix <projectPath> <encodedMatrixReportJson|@file>
cloudcc create reportRatio <projectPath> <encodedReportJson|@file>
cloudcc update reportRatio <projectPath> <reportId> <encodedReportJson|@file>
cloudcc delete report <projectPath> <reportId> [confirmdelete]

cloudcc get reportFolder <projectPath> [searchKeyWord]
cloudcc get reportFolder <projectPath> <encodedBodyJson>
cloudcc detail reportFolder <projectPath> <folderId>
cloudcc detail reportFolder <projectPath> <encodedBodyJson>
cloudcc create reportFolder <projectPath> <name> [encodedOptionsJson]
cloudcc update reportFolder <projectPath> <folderId> <encodedOptionsJson>
cloudcc delete reportFolder <projectPath> <folderId>
```

说明：

- `projectPath`：本地 CloudCC 项目路径，用于读取 `cloudcc-cli.config.json`。
- `encodedBodyJson` / `encodedReportJson`：URL 编码后的 JSON。CLI 也支持未编码 JSON，但跨 shell 推荐 URL 编码。
- `create/update/delete report` 和 `create/update/delete reportFolder` 返回的是 plan，不直接写库。

执行计划：

```bash
cloudcc apply msapi <projectPath> <planId>
```

### 2.1 类型化报表快捷命令

`report` 是通用入口，按传入 JSON 生成 `reports` domain plan。`reportTabular`、`reportSummary`、`reportMatrix`、`reportRatio` 是类型化入口，会先补齐 `reporttype`、`type`、`islightning`、`totalrecord`、`scope`，再按 main-svc `saveReport` 语义做客户端校验。

| 快捷命令 | 固定 `reporttype` | 必填核心参数 | 自动补全 | 适用场景 |
| --- | --- | --- | --- | --- |
| `reportTabular` | `Tabular` | `objecta`/`mainObjectId`/`source.objects[0]` 或 `reporttypecustomid`；`fields[]` 或 `mainobjectcolumnid` | `islightning=true`、`totalrecord=1`、`scope=user` | 列表式报表，只展示明细字段，不配置分组 |
| `reportSummary` | `Summary` | 数据源；展示字段；`groups.rows[]`/`rows[]` 或 `transversegroupone/two/three`；统计字段或 `totalrecord` | 同上；`summaryFields[]`/`gatherFields[]` 会转为 `summaries[]` | 汇总式报表，有行分组和统计 |
| `reportMatrix` | `Matrix` | 数据源；展示字段；行分组和列分组；统计字段或 `totalrecord` | 同上；行列分组会写入 `groups.rows/groups.columns`；应用时按 main-svc 规则落库交换横纵字段 | 矩阵式报表，同时有行分组和列分组 |
| `reportRatio` | `ratio` | 数据源；`dateFieldId`/`dateField`/`groupFieldId` 或 `transversegroupone` | 同上；默认 `transversedatetypeone=month`；用日期字段补 `datecon` 和 `mainobjectcolumnid=totalrecord,<dateField>`；`ratioExpressions[]` 会转为 `tbhbexpression` | 同比/环比报表 |

多对象报表如果没有直接传 `reporttypecustomid`，从第二个对象开始必须提供关联方式和关联字段：`optionb/c/d` 期待值为 `inner` 或 `outer`，`bfindid/cfindid/dfindid` 为真实关联字段 ID。`source.objects[n].option/findId` 与顶层别名等价。

---

## 3. MetadataService 接口映射

| CLI 命令 | MetadataService 行为 |
| --- | --- |
| `get report` | `POST /metadata/v1/reports:query` |
| `detail report` | `POST /metadata/v1/reports:detail` |
| `create report` | `POST /metadata/v1/plans`，domain=`reports`，operation=`create` |
| `update report` | `POST /metadata/v1/plans`，domain=`reports`，operation=`upsert` |
| `create/update reportTabular` | `POST /metadata/v1/plans`，domain=`reports`，operation=`create/upsert`，固定 `reporttype=Tabular` |
| `create/update reportSummary` | `POST /metadata/v1/plans`，domain=`reports`，operation=`create/upsert`，固定 `reporttype=Summary` |
| `create/update reportMatrix` | `POST /metadata/v1/plans`，domain=`reports`，operation=`create/upsert`，固定 `reporttype=Matrix` |
| `create/update reportRatio` | `POST /metadata/v1/plans`，domain=`reports`，operation=`create/upsert`，固定 `reporttype=ratio` |
| `delete report` | `POST /metadata/v1/plans`，domain=`reports`，operation=`delete` |
| `get reportFolder` | `POST /metadata/v1/report-folders:query` |
| `detail reportFolder` | `POST /metadata/v1/report-folders:detail` |
| `create reportFolder` | `POST /metadata/v1/plans`，domain=`reports`，operation=`folder-create` |
| `update reportFolder` | `POST /metadata/v1/plans`，domain=`reports`，operation=`folder-update` |
| `delete reportFolder` | `POST /metadata/v1/plans`，domain=`reports`，operation=`folder-delete` |

主要落库表：

| 表 | 用途 |
| --- | --- |
| `tp_sys_report` | 报表主表，保存名称、类型、文件夹、展示字段、筛选条件、分组、图表开关等 |
| `tp_sys_folder` | 报表文件夹 |
| `tp_sys_condition` | 报表筛选条件明细 |
| `tp_sys_report_fieldname` | 报表展示字段明细 |
| `tp_sys_reportgather` | 报表汇总字段 |
| `tp_sys_reportgroup` | 报表分组字段 |
| `tp_sys_report_object` | 显式报表对象配置；不是 `source.objects` 数据源定义 |
| `tp_sys_report_object_detail` | 显式报表对象关联明细 |
| `tp_sys_report_expression` | 报表行式公式/汇总公式 |
| `tp_sys_reporttypecustom` | 自定义报表类型/报表数据源 |
| `tp_sys_reporttypecustomfields` | 自定义报表类型字段 |

---

## 4. 报表参数总览

当前 MetadataService 支持两种写法：

1. 结构化写法：更适合 AI/CLI 生成，字段更清晰。
2. 直接字段写法：直接传 MetadataService 支持的报表落库字段。

两种写法可以混用；同一语义同时出现时，以更明确的字段为准。

### 4.1 结构化写法

```json
{
  "id": "rpt_sales_summary",
  "name": "销售汇总报表",
  "apiName": "sales_summary",
  "type": "Summary",
  "folderId": "folder_sales",
  "reportTypeCustomId": "rtc_sales",
  "source": {
    "reportTypeCustomId": "rtc_sales",
    "objects": [
      { "objectId": "Account", "recordTypeId": "" }
    ]
  },
  "fields": [
    { "fieldId": "field_name", "objectId": "Account", "fieldName": "客户名称", "location": 1 },
    { "fieldId": "field_amount", "objectId": "Opportunity", "fieldName": "金额", "location": 2 }
  ],
  "filters": {
    "logic": "1 AND 2",
    "items": [
      { "fieldId": "field_status", "op": "e", "value": "Active" },
      { "fieldId": "field_amount", "op": "g", "value": "1000" }
    ]
  },
  "groups": {
    "rows": [
      { "fieldId": "owner_id", "fieldName": "负责人", "sort": "asc", "dateType": "" }
    ],
    "columns": []
  },
  "summaries": [
    { "fieldId": "amount", "fieldName": "金额", "method": "sum" }
  ],
  "chart": {
    "type": "bar_0",
    "x": "totalrecord",
    "y": "owner_id",
    "summaryWay": "sum",
    "recordNum": "5"
  },
  "options": {
    "scope": "all",
    "showChart": true,
    "showDetail": true,
    "showTotal": true,
    "showSubtotal": true,
    "showCurrency": true,
    "currency": "CNY"
  }
}
```

DSL 字段说明：

| 字段 | 类型 | 必填 | 说明 | 对应落库字段 |
| --- | --- | --- | --- | --- |
| `id` | string | 更新必填 | 报表 ID；创建时可不传 | `tp_sys_report.id` |
| `name` | string | 是 | 报表名称 | `name` |
| `apiName` | string | 否 | 报表 API 名称 | `apiname` / `apiName` |
| `type` | string | 否 | 报表结构类型，见第 5 节 | `reporttype` |
| `folderId` / `reportFolderId` | string | 否 | 报表文件夹 ID；不传默认 `baf20200821lightning` | `reportfolderid` |
| `reportTypeCustomId` | string | 否 | 自定义报表类型/报表数据源 ID | `reporttypecustomid` |
| `source.reportTypeCustomId` | string | 否 | 数据源 ID，等同 `reportTypeCustomId` | `reporttypecustomid` |
| `source.objects[]` | array | 否 | 报表数据源对象列表；用于生成或更新 `tp_sys_reporttypecustom` 与 `tp_sys_reporttypecustomfields` | `tp_sys_reporttypecustom`、`tp_sys_reporttypecustomfields`、`tp_sys_report.mainobjectid` 等 |
| `objects[]` | array | 否 | 显式报表对象配置；仅需要写入 `tp_sys_report_object` / `tp_sys_report_object_detail` 时使用 | `tp_sys_report_object`、`tp_sys_report_object_detail` |
| `fields[]` | array | 否 | 明细展示字段列表；只代表报表明细列 | `tp_sys_report_fieldname`、`mainobjectcolumnid` 的一部分 |
| `filters.logic` | string | 否 | 条件组合表达式，例如 `1 AND 2` | `conditionVals.filter`、`tp_sys_condition.boolFilter` |
| `filters.items[]` | array | 否 | 条件列表 | `conditionVals.data`、`tp_sys_condition` |
| `groups.rows[]` | array | 否 | 行分组 | `transversegroup*`、`tp_sys_reportgroup` |
| `groups.columns[]` | array | 否 | 列分组 | `lengthwaysgroup*`、`tp_sys_reportgroup` |
| `summaries[]` | array | 否 | 汇总字段 | `gatherfieldname`、`tp_sys_reportgather` |
| `chart` | object | 否 | 图表配置 | `dashboardtype`、`xcon`、`ycon`、`summaryway` 等 |
| `options` | object | 否 | 显示、权限范围、币种等选项 | `scope`、`isshow*`、`currency` |

数据源说明：

- `source.objects[]` 描述报表取数对象，第一项会作为主对象，后续项可通过 `option`、`findId`、`referenceFieldId` 等字段描述关联对象。它不会直接写入 `tp_sys_report_object`。
- 当 payload 提供 `source.objects[]` 和 `fields[]`，但没有提供完整 `reportTypeCustom` 对象时，MetadataService 会自动生成 `tp_sys_reporttypecustom`，并从 `fields[]` 去重生成 `tp_sys_reporttypecustomfields`，再把生成或传入的 ID 写回 `tp_sys_report.reporttypecustomid`。
- 顶层 `objects[]` 是另一类扩展配置，只在需要显式维护 `tp_sys_report_object` 与 `tp_sys_report_object_detail` 时使用。普通报表数据源不要把 `source.objects[]` 复制到顶层 `objects[]`。

### 4.2 直接字段写法

如果已经明确 CloudCC 报表表字段或需要精确控制落库值，也可以直接传下列字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | string | 报表 ID；更新时必填 |
| `name` | string | 报表名称 |
| `reporttype` | string | 报表结构类型：`Tabular`、`Summary`、`Matrix`、`ratio`；也可由分组字段推断 |
| `reportfolderid` | string | 报表文件夹 ID |
| `description` | string | 描述 |
| `islightning` | string/boolean | Lightning 报表标记 |
| `scope` | string | 数据范围；常见 `user` 表示我的，`all` 表示全部 |
| `reporttypecustomid` | string | 报表数据源/自定义报表类型 ID |
| `mainobjectcolumnid` | string | main-svc 运行期字段集合，多个字段用逗号分隔；不是单纯展示字段列表，Matrix 下还应包含分组、筛选、日期、汇总、图表字段和 `totalrecord` |
| `dataSize` | string | 查询数据量 |
| `pageSize` | string | 分页大小或预览页大小 |
| `page` | string | 页码 |
| `totalrecord` | string | 是否统计记录数，常见 `1` |
| `conditionVals` | string | 筛选条件 JSON 字符串，见第 6 节 |
| `drillConditionVals` | string | 图表钻取条件 JSON 字符串，格式同 `conditionVals` |
| `detailConditions` | string | 明细钻取条件，矩阵/图表明细查询时常见 |
| `objecta` / `objectb` / `objectc` / `objectd` | string | A/B/C/D 数据源对象 ID |
| `optionb` / `optionc` / `optiond` | string | 对象关联方式，常见 `inner`、`outer` |
| `bfindid` / `cfindid` / `dfindid` | string | 对象间关联字段 ID |
| `datecon` | string | 日期筛选字段 ID |
| `startdatestr` / `enddatestr` | string | 日期筛选开始/结束日期，常用 `yyyy-MM-dd` |
| `datarange` | string | 快捷日期范围 |
| `currency` | string | 币种，例如 `CNY` |
| `isshowcurrency` | string/boolean | 是否显示币种 |
| `isshowchart` | string/boolean | 是否显示图表 |
| `isshowdetail` | string/boolean | 是否显示明细 |
| `isshowtotal` | string/boolean | 是否显示总计 |
| `isshowsubtotal` | string/boolean | 是否显示小计 |
| `expression` | string | 行式公式/汇总公式 JSON 字符串 |
| `summaryexpression` | string | 汇总公式 ID |
| `tbhbexpression` | string | 同比/环比配置 JSON 字符串 |
| `roleid` | string | 查询报表数据时使用的角色 ID |
| `orderFieldApi` / `orderFieldType` / `orderType` | string | 明细排序字段 API、字段类型、排序方式 |

字段名大小写说明：

- MetadataService 接受常见 camelCase 和小写字段，例如 `reportTypeCustomId` 与 `reporttypecustomid`。
- `conditionVals`、`drillConditionVals`、`detailConditions` 是当前服务支持的报表条件字段名。
- `folderId`、`reportFolderId`、`reportfolderid` 都会被识别为报表文件夹 ID。
- `source.objects[]` 与顶层 `objects[]` 含义不同：前者是数据源定义，后者是显式报表对象配置。

---

## 5. 报表类型

### 5.1 `reporttype` 与 `reporttypecustomid` 不要混用

| 字段 | 含义 |
| --- | --- |
| `reporttype` / `type` | 报表结构类型，决定列表式、汇总式、矩阵式、同环比 |
| `reporttypecustomid` / `reportTypeCustomId` | 报表数据源/自定义报表类型 ID，决定基于哪些对象和关联关系取数 |

### 5.2 `reporttype` 取值

| 取值 | 类型 | 说明 |
| --- | --- | --- |
| `Tabular` | 列表式报表 | 主要展示明细记录，可配置展示字段、筛选条件、排序和分页 |
| `Summary` | 汇总报表 | 在明细字段基础上增加行分组、汇总字段和统计方式 |
| `Matrix` | 矩阵式报表 | 同时配置行分组与列分组，适合交叉统计 |
| `ratio` | 同环比报表 | 用于同比、环比分析，配合 `tbhbexpression`、日期字段和日期位置 |

### 5.3 类型推断

如果没有显式传 `reporttype` 或 `type`，MetadataService 会按分组字段推断：

| 字段组合 | 推断类型 |
| --- | --- |
| 没有 `groups.rows`，也没有 `groups.columns`；或所有 `transversegroup*`、`lengthwaysgroup*` 为空 | `Tabular` |
| 有 `groups.rows` 或任意 `transversegroup*`，但没有列分组 | `Summary` |
| 有行分组，也有 `groups.columns` 或任意 `lengthwaysgroup*` | `Matrix` |
| 显式传 `ratio` | `ratio` |

矩阵报表说明：为对齐当前 CloudCC 主服务保存逻辑，`Matrix` 类型在最终落库时会把列分组写入 `transversegroup*`，把行分组写入 `lengthwaysgroup*`。使用结构化 DSL 时仍按业务语义填写 `groups.rows` 和 `groups.columns`；复核 plan 时看到横向/纵向字段交换属于正常现象。

---

## 6. 筛选条件 `filters` 与 `conditionVals`

### 6.1 结构化筛选条件

推荐使用：

```json
{
  "filters": {
    "logic": "(1 OR 2) AND 3",
    "items": [
      { "fieldId": "field_status", "op": "e", "value": "Active" },
      { "fieldId": "field_name", "op": "c", "value": "北京" },
      { "fieldId": "createdate", "op": "h", "value": "2026-01-01 00:00:00" }
    ]
  }
}
```

MetadataService 会生成：

- `tp_sys_condition` 条件明细行。
- plan 元数据中的条件数量、逻辑编号压缩结果等复核信息。

`conditionVals` 是兼容 main-svc `saveReport` 输入的条件 envelope，不再作为 `tp_sys_report` 主表字段写入；有效条件会物化为 `tp_sys_condition` 行。`filters.logic` 中的条件编号会按有效条件重新压缩，例如原始 `(1 OR 3)` 在第 2 条无效时会落为 `(1 OR 2)`。

如果 `filters.items[]` 中某一项缺少 `fieldId` 或 `op`，该条件不会写入 `tp_sys_condition`。

### 6.2 直接传 `conditionVals`

`conditionVals` 本身是字符串，字符串内容是一段 JSON。它用于兼容 main-svc 输入形状，MetadataService 会解析其中的 `data[]` 生成 `tp_sys_condition` 行，不会把这段字符串写回 `tp_sys_report.conditionVals`：

```json
{
  "conditionVals": "{\"data\":[{\"fieldId\":\"field_status\",\"op\":\"e\",\"val\":\"Active\"}],\"filter\":\"1\",\"mainObjId\":\"Account\",\"relatedObjIdC\":\"\",\"relatedObjIdD\":\"\"}"
}
```

内层 JSON 结构：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `data` | array | 是 | 条件数组。数组顺序从 1 开始编号，`filter` 中的 `1`、`2`、`3` 对应第 1、2、3 个条件 |
| `data[].fieldId` | string | 是 | 条件字段 ID |
| `data[].op` | string | 是 | 操作符代码 |
| `data[].val` | string | 是 | 条件值 |
| `filter` | string | 否 | 条件组合表达式，支持条件序号、`AND`、`OR`、括号 |
| `mainObjId` | string | 否 | 主对象 ID |
| `relatedObjId` | string | 否 | B 对象 ID |
| `relatedObjIdC` | string | 否 | C 对象 ID |
| `relatedObjIdD` | string | 否 | D 对象 ID |

空条件推荐写法：

```json
{
  "data": [],
  "filter": "",
  "relatedObjIdC": "",
  "relatedObjIdD": ""
}
```

`conditionVals` 的字符串内容必须是标准 JSON。不要使用尾随逗号，例如不要写成 `{"data":[],"filter":"","relatedObjIdC":"","relatedObjIdD":"",}`。

如果 `conditionVals.data[]` 中某一项缺少 `fieldId` 或 `op`，MetadataService 会跳过该条件行，并按有效条件重新压缩 `filter` 中的编号。建议在生成 JSON 前就移除无效条件，避免人工复核 plan 时产生误解。

### 6.3 操作符 `op`

| op | 含义 | 常用场景 |
| --- | --- | --- |
| `e` | 等于 | 文本、数字、日期、下拉、查找 |
| `n` | 不等于 | 文本、数字、日期、下拉、查找 |
| `s` | 开头是 | 文本 |
| `c` | 包含 | 文本、多选、部分公式文本；支持在 `value`/`val` 中用英文逗号传多个匹配值，例如 `北京,上海` |
| `k` | 不包含 | 文本、多选、部分公式文本；支持在 `value`/`val` 中用英文逗号传多个排除值，例如 `停用,作废` |
| `l` | 小于 | 数字、日期、日期时间 |
| `g` | 大于 | 数字、日期、日期时间 |
| `m` | 小于等于 | 数字、日期、日期时间，常用于范围结束 |
| `h` | 大于等于 | 数字、日期、日期时间，常用于范围开始 |
| `u` | 包括 | 多选或集合类字段 |
| `x` | 不包括 | 多选或集合类字段 |

多值说明：

- `c`/`k` 会按 main-svc 报表运行期语义处理逗号分隔值；MetadataService 只负责把 CSV 原样写入 `conditionVals.data[].val` 和 `tp_sys_condition.value`，不会在计划阶段拆成多条条件。
- `u`/`x` 也用于多选或集合类字段，值通常同样以英文逗号组织；使用前应确认目标字段的运行期报表查询支持该集合语义。
- 如果筛选值本身包含英文逗号，不要用 `c`/`k` 的 CSV 简写；应改用其它业务字段、转义策略或拆成更明确的条件组合，避免 main-svc 运行期按多值解析。

范围过滤建议用两条条件组合，例如开始日期用 `h`，结束日期用 `m`。

```json
{
  "filters": {
    "logic": "1 AND 2",
    "items": [
      { "fieldId": "closeDate", "op": "h", "value": "2026-01-01 00:00:00" },
      { "fieldId": "closeDate", "op": "m", "value": "2026-12-31 23:59:59" }
    ]
  }
}
```

---

## 7. 展示字段

### 7.1 结构化展示字段

```json
{
  "fields": [
    { "fieldId": "field_name", "fieldName": "客户名称", "objectId": "Account", "location": 1 },
    { "fieldId": "field_amount", "fieldName": "金额", "objectId": "Opportunity", "location": 2 }
  ]
}
```

字段说明：

| 字段 | 说明 |
| --- | --- |
| `fieldId` / `id` / `field` | 字段 ID |
| `fieldName` / `label` / `name` | 字段显示名 |
| `objectId` / `objId` | 所属对象 ID |
| `location` / `seq` / `sort` | 展示顺序 |
| `fieldApi` / `apiName` | 字段 API 名称 |

### 7.2 直接传展示字段

也可以直接传 `mainobjectcolumnid`：

```json
{
  "mainobjectcolumnid": "field_name,field_amount,totalrecord"
}
```

说明：

- 多个字段 ID 用逗号分隔。
- `totalrecord` 常用于统计记录数。
- 如果同时传 `fields[]` 和 `mainobjectcolumnid`，MetadataService 会尽量保留 `mainobjectcolumnid` 并生成字段明细行。

---

## 8. 分组与汇总

### 8.1 行分组

结构化写法：

```json
{
  "groups": {
    "rows": [
      { "fieldId": "owner_id", "fieldName": "负责人", "sort": "asc", "dateType": "" },
      { "fieldId": "createdate", "fieldName": "创建日期", "sort": "desc", "dateType": "month" }
    ]
  }
}
```

直接字段：

| 字段 | 说明 |
| --- | --- |
| `transversegroupone` / `transversegrouptwo` / `transversegroupthree` | 第 1/2/3 行分组字段 |
| `transversesorttypeone` / `transversesorttypetwo` / `transversesorttypethree` | 行分组排序，`asc` 或 `desc` |
| `transversedatetypeone` / `transversedatetypetwo` / `transversedatetypethree` | 行分组日期粒度 |

注意：直接字段写法中的 `transversegroup*` 表示提交给 MetadataService 的行分组语义；如果最终推断为 `Matrix`，计划中的主表落库值会按主服务规则与 `lengthwaysgroup*` 交换。

### 8.2 列分组

结构化写法：

```json
{
  "groups": {
    "columns": [
      { "fieldId": "product_id", "fieldName": "产品", "sort": "asc", "dateType": "" }
    ]
  }
}
```

直接字段：

| 字段 | 说明 |
| --- | --- |
| `lengthwaysgroupone` / `lengthwaysgrouptwo` | 第 1/2 列分组字段 |
| `lengthwayssorttypeone` / `lengthwayssorttypetwo` | 列分组排序，`asc` 或 `desc` |
| `lengthwaysdatetypeone` / `lengthwaysdatetypetwo` | 列分组日期粒度 |

注意：直接字段写法中的 `lengthwaysgroup*` 表示提交给 MetadataService 的列分组语义；如果最终推断为 `Matrix`，计划中的主表落库值会按主服务规则与 `transversegroup*` 交换。

### 8.3 日期分组粒度

| 取值 | 说明 |
| --- | --- |
| `day` | 日 |
| `month` | 日历月 |
| `year` | 日历年 |
| `FY` | 财年 |
| `FQ` | 财年季度 |
| `CY` | 自定义年 |
| `CQ` | 自定义季度 |

### 8.4 汇总字段

结构化写法：

```json
{
  "summaries": [
    { "fieldId": "amount", "fieldName": "金额", "method": "sum" },
    { "fieldId": "score", "fieldName": "评分", "method": "avg" }
  ]
}
```

直接字段：

| 字段 | 说明 |
| --- | --- |
| `gatherfieldname` | 求和字段 ID 列表 |
| `avgfield` | 平均值字段 |
| `maxfield` | 最大值字段 |
| `minfield` | 最小值字段 |
| `uniquefield` | 唯一值统计字段 |
| `summaryexpression` | 汇总公式 ID |

常见 `method`：

| 取值 | 说明 |
| --- | --- |
| `sum` | 求和 |
| `avg` | 平均 |
| `max` | 最大值 |
| `min` | 最小值 |
| `unique` | 唯一值统计 |

---

## 9. 图表字段

结构化写法：

```json
{
  "chart": {
    "type": "bar_0",
    "x": "totalrecord",
    "y": "owner_id",
    "xGather": "",
    "summaryWay": "sum",
    "recordNum": "5",
    "sortCondition": "value",
    "sortType": "desc"
  }
}
```

直接字段：

| 字段 | 说明 |
| --- | --- |
| `dashboardtype` | 图表类型 |
| `xcon` | X 轴统计值 |
| `xgather` | 第二分组 |
| `ycon` | Y 轴第一分组 |
| `dashboardsortcondition` | 图表排序条件，常见 `gather`、`value`、`label` |
| `orderbyfield` | 排序字段 |
| `orderbyfieldtype` | 排序字段类型 |
| `dashboardsorttype` | 排序方式，`asc` 或 `desc` |
| `unit` | 显示单位 |
| `summaryway` | 统计类型 |
| `recordnum` | 图表条目数 |
| `isshowchart` | 是否显示图表 |
| `isshowpercent` | 是否显示百分比 |
| `isshowvalue` | 是否显示数值 |
| `dashboardid` / `reportid` / `objectid` / `viewid` | 仪表板、报表、对象、视图上下文 ID |
| `dashboardx` / `dashboardy` / `width` / `height` | 仪表板布局位置和尺寸 |
| `min` | 图表最小值 |
| `isurl` | URL 跳转配置开关 |
| `linkageid` | 联动 ID |
| `isagreement` | 协议/联动类开关 |

常见图表类型：

| 取值 | 说明 |
| --- | --- |
| `column_0` / `column_1` | 柱状图 |
| `bar_0` / `bar_1` | 条形图 |
| `line_0` / `line_1` | 折线图 |
| `pie` | 饼图 |
| `donut` | 环形图 |
| `column_duidie` | 堆叠柱状图 |
| `bar_duidie` | 堆叠条形图 |
| `area` | 面积图 |
| `funnel` | 漏斗图 |
| `gauge` | 仪表盘图 |

---

## 10. 公式与同环比

### 10.1 公式 `expression`

`expression` 是 JSON 字符串，用于保存行式公式和汇总公式：

```json
{
  "expression": "{\"data\":[{\"id\":\"\",\"apiname\":\"汇总公式\",\"expression\":\"Account__c.amount__c:SUM+Account__c.cost__c:SUM\",\"type\":\"Z\",\"description\":\"描述\",\"decimalplaces\":\"2\"},{\"id\":\"\",\"apiname\":\"行式公式\",\"expression\":\"Account__c.amount__c+Account__c.cost__c\",\"type\":\"L\",\"description\":\"描述\",\"decimalplaces\":\"2\"}]}"
}
```

| 字段 | 说明 |
| --- | --- |
| `type=Z` | 汇总公式 |
| `type=L` | 行式公式 |
| `apiname` | 公式名称 |
| `expression` | 公式表达式 |
| `decimalplaces` | 小数位 |

如果 `tp_sys_report.expression` 传的是逗号分隔的表达式 ID，计划会额外更新这些 `tp_sys_report_expression.reportid`，把公式归属到当前报表；如果使用 `expressions[]` 结构化数组，计划会直接 upsert 对应公式行。

### 10.2 同环比 `tbhbexpression`

同环比报表建议显式传：

```json
{ "reporttype": "ratio" }
```

`tbhbexpression` 示例：

```json
{
  "tbhbexpression": "{\"data\":[{\"groupFieldId\":\"createdate\",\"type\":\"TB\",\"expression\":\"amount:SUM\"},{\"groupFieldId\":\"createdate\",\"type\":\"HB\",\"expression\":\"amount:SUM\"}]}"
}
```

| 字段 | 说明 |
| --- | --- |
| `type=TB` | 同比 |
| `type=HB` | 环比 |
| `groupFieldId` | 分组字段 |
| `expression` | 汇总表达式，例如 `amount:SUM` |
| `datelocation` | 日期位置，常见取值 `first`、`end` |

---

## 11. 各类报表示例

### 11.1 列表式报表 Tabular

```json
{
  "name": "客户明细报表",
  "reporttype": "Tabular",
  "reportfolderid": "baf20200821lightning",
  "islightning": "true",
  "scope": "all",
  "objecta": "accountObjectId",
  "mainobjectcolumnid": "fieldId1,fieldId2,totalrecord",
  "totalrecord": "1",
  "dataSize": "50",
  "pageSize": "1",
  "isshowdetail": "true",
  "isshowchart": "false",
  "conditionVals": "{\"data\":[],\"filter\":\"\",\"relatedObjIdC\":\"\",\"relatedObjIdD\":\"\"}"
}
```

### 11.2 汇总报表 Summary

```json
{
  "name": "销售额汇总报表",
  "type": "Summary",
  "folderId": "baf20200821lightning",
  "source": {
    "reportTypeCustomId": "rtc_sales",
    "objects": [{ "objectId": "Opportunity" }]
  },
  "fields": [
    { "fieldId": "owner_id", "fieldName": "负责人", "location": 1 },
    { "fieldId": "amount", "fieldName": "金额", "location": 2 }
  ],
  "filters": {
    "logic": "1",
    "items": [
      { "fieldId": "stage", "op": "n", "value": "Closed Lost" }
    ]
  },
  "groups": {
    "rows": [{ "fieldId": "owner_id", "fieldName": "负责人", "sort": "asc" }]
  },
  "summaries": [
    { "fieldId": "amount", "fieldName": "金额", "method": "sum" }
  ],
  "options": {
    "scope": "all",
    "showTotal": true,
    "showSubtotal": true,
    "showDetail": true
  }
}
```

### 11.3 矩阵式报表 Matrix

```json
{
  "name": "地区产品矩阵报表",
  "type": "Matrix",
  "folderId": "baf20200821lightning",
  "reportTypeCustomId": "rtc_sales",
  "fields": [
    { "fieldId": "region", "fieldName": "区域", "location": 1 },
    { "fieldId": "product", "fieldName": "产品", "location": 2 },
    { "fieldId": "amount", "fieldName": "金额", "location": 3 }
  ],
  "groups": {
    "rows": [{ "fieldId": "region", "fieldName": "区域", "sort": "asc" }],
    "columns": [{ "fieldId": "product", "fieldName": "产品", "sort": "asc" }]
  },
  "summaries": [
    { "fieldId": "amount", "fieldName": "金额", "method": "sum" }
  ],
  "chart": {
    "type": "bar_0",
    "x": "totalrecord",
    "y": "product",
    "summaryWay": "sum",
    "recordNum": "5"
  },
  "options": {
    "showChart": true,
    "showDetail": true,
    "showTotal": true,
    "showSubtotal": true
  }
}
```

### 11.4 同环比报表 ratio

```json
{
  "name": "客户月度同环比报表",
  "reporttype": "ratio",
  "reportfolderid": "baf20200821lightning",
  "islightning": "true",
  "scope": "all",
  "objecta": "account",
  "mainobjectcolumnid": "dateFieldId,totalrecord",
  "totalrecord": "1",
  "datecon": "",
  "startdatestr": "2026-01-01",
  "enddatestr": "2026-07-31",
  "transversegroupone": "dateFieldId",
  "transversedatetypeone": "month",
  "gatherfieldname": "totalrecord",
  "datelocation": "first",
  "orderFieldApi": "t0createdate",
  "orderFieldType": "F",
  "orderType": "desc",
  "conditionVals": "{\"data\":[],\"filter\":\"\",\"relatedObjIdC\":\"\",\"relatedObjIdD\":\"\"}",
  "drillConditionVals": "{\"data\":[],\"filter\":\"\",\"relatedObjIdC\":\"\",\"relatedObjIdD\":\"\"}",
  "tbhbexpression": ""
}
```

---

## 12. 报表文件夹

### 12.1 查询文件夹

```bash
cloudcc get reportFolder <projectPath> [searchKeyWord]
cloudcc get reportFolder <projectPath> <encodedBodyJson>
```

请求字段：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `searchKeyWord` | string | 否 | 文件夹关键字 |
| `page` | integer | 否 | 页码 |
| `pageSize` | integer | 否 | 每页条数 |
| `orderField` | string | 否 | 排序字段 |
| `orderType` | string | 否 | 排序方式，`asc` 或 `desc` |

查询时 `foldertype=report` 和 `foldertype=lightning` 都会作为报表文件夹返回。

### 12.2 创建文件夹

```bash
cloudcc create reportFolder <projectPath> <name> [encodedOptionsJson]
```

参数：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `name` | string | 是 | 文件夹名称 |
| `id` | string | 否 | 文件夹 ID；新增通常不传 |
| `folderType` / `foldertype` | string | 否 | 文件夹类型；新建默认 `lightning` |
| `viewType` / `viewtype` | string | 否 | 访问范围 |
| `purview` | string | 否 | 文件夹读写权限 |
| `type` | string | 否 | 搜索类型 |
| `findName` | string | 否 | 搜索关键字 |
| `folderUser` | string | 否 | 指定可访问用户、角色、组等 ID 列表 |
| `folderReport` | string | 否 | 逗号分隔的报表 ID 列表 |
| `reportIds` | array | 否 | 推荐写法，报表 ID 数组 |

示例：

```bash
cloudcc create reportFolder . "销售报表" "%7B%22viewType%22%3A%222%22%2C%22purview%22%3A%221%22%2C%22reportIds%22%3A%5B%22rpt_sales%22%5D%7D"
```

### 12.3 更新文件夹

```bash
cloudcc update reportFolder <projectPath> <folderId> <encodedOptionsJson>
```

更新文件夹元数据：

```json
{
  "name": "销售报表",
  "viewType": "2",
  "purview": "1"
}
```

更新文件夹内报表：

```json
{
  "name": "销售报表",
  "reportIds": ["rpt_a", "rpt_b"]
}
```

成员变更规则：

- 传 `reportIds` 或 `folderReport`：表示显式替换文件夹内报表。更新时会先把原文件夹内报表移回 `baf20200821lightning`，再把新列表分配到当前文件夹。
- 不传 `reportIds` / `folderReport`：只更新文件夹名称、权限等元数据，不改变报表归属。
- 传空数组 `reportIds: []`：表示清空该文件夹，将现有报表移回 `baf20200821lightning`。

### 12.4 删除文件夹

```bash
cloudcc delete reportFolder <projectPath> <folderId>
```

删除前建议先查询：

```bash
cloudcc get report . <folderId>
cloudcc detail reportFolder . <folderId>
```

如果文件夹内仍有报表，应先决定是否迁移到其他文件夹或未归档公用文件夹。

---

## 13. 查询报表

### 13.1 查询列表

```bash
cloudcc get report <projectPath> [folderId] [searchKeyWord] [page] [pageSize] [orderField] [orderType]
cloudcc get report <projectPath> <encodedBodyJson>
```

请求字段：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `folderId` | string | 否 | 报表文件夹 ID；空表示不按文件夹筛选 |
| `searchKeyWord` | string | 否 | 关键字 |
| `page` | integer | 否 | 页码，默认 `1` |
| `pageSize` | integer | 否 | 每页条数，默认 `20` |
| `orderField` | string | 否 | 排序字段 |
| `orderType` | string | 否 | 排序方式，`asc` 或 `desc` |

示例：

```bash
cloudcc get report . "" "销售" 1 20 name asc
cloudcc get report . "%7B%22folderId%22%3A%22baf20200821lightning%22%2C%22searchKeyWord%22%3A%22%E9%94%80%E5%94%AE%22%2C%22page%22%3A1%2C%22pageSize%22%3A20%7D"
```

### 13.2 查询详情

```bash
cloudcc detail report <projectPath> <reportId>
cloudcc detail report <projectPath> <encodedBodyJson>
```

请求字段：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `id` | string | 是 | 报表 ID |

---

## 14. 创建、更新、删除报表

### 14.1 创建

```bash
cloudcc create report <projectPath> <encodedReportJson>
```

创建时如果 body 中包含 `id`，CLI/MetadataService 会按新增语义处理；建议创建时不传 `id`。

### 14.2 更新

```bash
cloudcc update report <projectPath> <reportId> <encodedReportJson>
cloudcc update report <projectPath> <encodedReportJson>
```

说明：

- 第一种写法会把 `<reportId>` 写入 body 的 `id`。
- 第二种写法要求 JSON 自身包含 `id`。

### 14.3 删除

```bash
cloudcc delete report <projectPath> <reportId> [confirmdelete]
```

请求字段：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `id` | string | 是 | 报表 ID |
| `confirmdelete` | string | 否 | 确认删除标记 |

---

## 15. 编码示例

PowerShell 推荐写法：

```powershell
$body = @{
  name = "客户明细报表"
  type = "Tabular"
  folderId = "baf20200821lightning"
  fields = @(
    @{ fieldId = "field_name"; fieldName = "客户名称"; location = 1 }
  )
  filters = @{
    logic = "1"
    items = @(
      @{ fieldId = "field_status"; op = "e"; value = "Active" }
    )
  }
  options = @{
    scope = "all"
    showDetail = $true
  }
} | ConvertTo-Json -Depth 10 -Compress

$encoded = [uri]::EscapeDataString($body)
cloudcc create report . $encoded
```

直接传 `conditionVals` 写法：

```powershell
$condition = '{"data":[{"fieldId":"field_status","op":"e","val":"Active"}],"filter":"1","relatedObjIdC":"","relatedObjIdD":""}'
$body = @{
  name = "客户过滤报表"
  reporttype = "Tabular"
  reportfolderid = "baf20200821lightning"
  objecta = "accountObjectId"
  mainobjectcolumnid = "field_name,totalrecord"
  conditionVals = $condition
} | ConvertTo-Json -Compress

$encoded = [uri]::EscapeDataString($body)
cloudcc create report . $encoded
```

---

## 16. 矩阵式报表完整创建指南

矩阵式报表建议优先使用专用快捷命令：

```bash
cloudcc create reportMatrix <projectPath> @matrix-report.json
cloudcc update reportMatrix <projectPath> <reportId> @matrix-report.json
```

`reportMatrix` 会把业务语义参数补全为 `reports` domain 的 `create/upsert` plan，并在客户端提前校验矩阵必需项。它不会直接写库；仍然需要复核 plan 后执行：

```bash
cloudcc apply msapi <projectPath> <planId>
```

### 16.1 最小可用矩阵报表

```json
{
  "name": "客户机会矩阵报表",
  "source": {
    "objects": [
      { "objectId": "account" },
      { "objectId": "opportunity" }
    ]
  },
  "fields": [
    { "fieldId": "field_account_name", "objectId": "account", "fieldName": "客户名称", "location": 1 },
    { "fieldId": "field_amount", "objectId": "opportunity", "fieldName": "金额", "location": 2 }
  ],
  "rows": [
    { "fieldId": "field_owner", "fieldName": "负责人", "sort": "asc" }
  ],
  "columns": [
    { "fieldId": "field_close_date", "fieldName": "预计成交日期", "sort": "asc", "dateType": "month" }
  ],
  "summaryFields": ["field_amount"],
  "optionb": "inner",
  "bfindid": "field_account_lookup",
  "scope": "all",
  "startdatestr": "2026-01-01",
  "enddatestr": "2026-12-31"
}
```

### 16.2 自动补全和校验

| 参数 | 自动值 / 校验 | 说明 |
| --- | --- | --- |
| `type` / `reporttype` | 自动写为 `Matrix` | 固定创建矩阵式报表 |
| `islightning` | 默认 `true` | 使用 Lightning 报表模型 |
| `totalrecord` | 默认 `1` | 默认统计记录数；可同时配置金额等汇总字段 |
| `scope` | 默认 `user` | 不显式传时默认“我的”数据范围 |
| `rows` / `groups.rows` | 必须有，最多 3 个 | 行分组业务语义 |
| `columns` / `groups.columns` | 必须有，最多 2 个 | 列分组业务语义 |
| `fields[]` / `mainobjectcolumnid` | 必须有其一 | MetadataService 不能凭空猜测展示字段 |
| `summaries[]` / `summaryFields[]` / `gathers[]` / `gatherfieldname` / `totalrecord` | 必须有其一 | 统计字段；`totalrecord=1` 表示统计记录数 |
| `isshowchart=true` | 必须同时提供 `dashboardtype`、`xcon`、`ycon`，或 `chart.type`、`chart.x`、`chart.y` | CLI 会提前拦截缺少图表轴/类型的 Matrix payload |
| `datarange` / `startdatestr` / `enddatestr` | 出现日期范围时必须提供 `datecon` / `dateCondition` / `timeFrame.dateFieldId` / `dateFieldId` | 避免保存后 UI 无法还原时间范围使用的日期字段 |
| `optionb` | 有 B 对象且不复用已有 `reportTypeCustomId` 时必填；支持 `inner`、`outer` | A-B 关联方式 |
| `bfindid` | 有 B 对象且不复用已有 `reportTypeCustomId` 时必填 | A-B 关联字段 ID |

矩阵报表最终落库会兼容 main-svc 保存模型：用户按业务语义填写 `rows` 和 `columns`，计划里会把列分组写入 `tp_sys_report.transversegroup*`，把行分组写入 `tp_sys_report.lengthwaysgroup*`。MetadataService 会自动生成 `mainobjectcolumnid` 运行字段闭包，来源包括 `fields[]`、行列分组、筛选、日期字段、汇总字段、图表字段和 `totalrecord`；plan 元数据会展示闭包来源和 Matrix 行列落库映射。

## 17. 参数取值速查表

### 17.1 报表主参数

| 参数 | 类型 | 必填 | 支持值 / 格式 | 作用 |
| --- | --- | --- | --- | --- |
| `id` | string | 更新必填 | 报表 ID | 定位要更新或删除的报表；`create report` 和 `create reportMatrix` 会移除传入的 `id` |
| `name` | string | 创建必填 | 任意非空名称 | 报表显示名称 |
| `apiName` / `apiname` | string | 否 | 唯一 API 名 | 报表 API 名 |
| `type` / `reporttype` | string | 否 | `Tabular`、`Summary`、`Matrix`、`ratio` | 报表结构类型 |
| `folderId` / `reportFolderId` / `reportfolderid` | string | 否 | 文件夹 ID；默认 `baf20200821lightning` | 报表归属文件夹 |
| `description` | string | 否 | 任意文本 | 报表说明 |
| `islightning` / `lightning` | string/boolean | 否 | `true`、`false`；默认 `true` | 是否 Lightning 报表 |
| `scope` / `options.scope` | string | 否 | `user`、`all`；默认 `user` | `user` 表示我的数据，`all` 表示全部可见数据 |
| `reportTypeCustomId` / `reporttypecustomid` | string | 否 | 数据源 ID | 复用已有报表数据源，或指定要 upsert 的数据源 ID |

### 17.2 数据源对象参数

| 参数 | 类型 | 必填 | 支持值 / 格式 | 作用 |
| --- | --- | --- | --- | --- |
| `source.objects[0].objectId` / `objecta` | string | 是 | 对象 ID 或 API 名 | 主对象 A |
| `source.objects[1].objectId` / `objectb` / `relatedObjectId` | string | 否 | 对象 ID 或 API 名 | 第二对象 B |
| `source.objects[2].objectId` / `objectc` | string | 否 | 对象 ID 或 API 名 | 第三对象 C |
| `source.objects[3].objectId` / `objectd` | string | 否 | 对象 ID 或 API 名 | 第四对象 D |
| `source.objects[1].option` / `optionb` | string | B 对象新建数据源时必填 | `inner`、`outer` | A-B 关联方式 |
| `source.objects[2].option` / `optionc` | string | C 对象新建数据源时必填 | `inner`、`outer` | B-C 关联方式 |
| `source.objects[3].option` / `optiond` | string | D 对象新建数据源时必填 | `inner`、`outer` | C-D 关联方式 |
| `source.objects[1].findId` / `bfindid` | string | B 对象新建数据源时必填 | 关联字段 ID | A-B 关联字段 |
| `source.objects[2].findId` / `cfindid` | string | C 对象新建数据源时必填 | 关联字段 ID | B-C 关联字段 |
| `source.objects[3].findId` / `dfindid` | string | D 对象新建数据源时必填 | 关联字段 ID | C-D 关联字段 |
| `treeType` | string | 否 | `line`、`scatter` | 数据源关系树类型 |
| `objectRelation` | string/object | 否 | 常见 `all`、`one`、`none`，或对象结构 | 数据源对象关系规则 |

### 17.3 展示字段、分组和汇总

| 参数 | 类型 | 必填 | 支持值 / 格式 | 作用 |
| --- | --- | --- | --- | --- |
| `fields[]` | array | 推荐 | 字段对象数组 | 明细展示字段；会生成字段明细和数据源字段 |
| `fields[].fieldId` / `field` | string | 是 | 字段 ID | 明细展示字段 ID |
| `fields[].objectId` / `objId` | string | 否 | 对象 ID/API 名 | 字段所属对象，用于推断 `OBJTYPE=A/B/C/D` |
| `fields[].fieldName` / `label` / `name` | string | 否 | 字段显示名 | 报表字段标签 |
| `fields[].location` | number/string | 否 | 正整数；默认按数组顺序 | 展示顺序 |
| `mainobjectcolumnid` / `mainObjectColumnId` | string | `fields[]` 缺失时必填 | 字段 ID CSV，例如 `fieldA,fieldB,totalrecord` | main-svc 运行期字段集合；Matrix 下必须覆盖展示、分组、过滤、日期、统计和图表依赖字段。MetadataService 会基于结构化参数自动补全闭包 |
| `groups.rows[]` / `rows[]` | array | Summary/Matrix 必填 | 最多 3 个 | 行分组业务语义 |
| `groups.columns[]` / `columns[]` | array | Matrix 必填 | 最多 2 个 | 列分组业务语义 |
| `groups.*[].sort` | string | 否 | `asc`、`desc`；默认 `asc` | 分组排序 |
| `groups.*[].dateType` | string | 日期字段建议填 | `day`、`month`、`year`、`FY`、`FQ`、`CY`、`CQ` | 日期分组粒度 |
| `summaries[].method` / `type` / `summaryType` | string | 否 | `sum`、`avg`、`max`、`min`、`unique`；默认 `sum` | 汇总方式 |
| `summaryFields[]` / `gatherFields[]` | array | 否 | 字段 ID 字符串数组或对象数组 | `reportMatrix` 快捷写法，会转成 `summaries[]` |
| `totalrecord` | string/boolean | 否 | `1` 表示统计记录数；空/`0` 表示不启用 | 是否统计记录数 |

### 17.4 筛选、日期、图表和显示

| 参数 | 类型 | 必填 | 支持值 / 格式 | 作用 |
| --- | --- | --- | --- | --- |
| `filters.logic` | string | 否 | `1 AND 2`、`(1 OR 2) AND 3` | 条件组合表达式 |
| `filters.items[].op` / `operator` | string | 条件必填 | 常见 `e`、`n`、`c`、`k`、`g`、`l`、`m`、`h` | 条件操作符，具体含义以租户主服务实现为准 |
| `conditionVals` | string | 否 | JSON 字符串 | 直接传 main-svc 条件结构 |
| `mainObjId` / `relatedObjId` / `relatedObjIdC` / `relatedObjIdD` | string | 多对象建议填 | 对象 ID/API 名 | 条件所属对象上下文 |
| `datecon` / `dateCondition` / `timeFrame.dateFieldId` / `dateFieldId` | string | 使用日期范围时必填 | 日期字段 ID | 报表日期筛选字段；有 `datarange/startdatestr/enddatestr` 时用于让 UI 还原时间范围 |
| `startDate` / `startdate` / `startdatestr` | string | 否 | `yyyy-MM-dd` 或 `yyyy-MM-dd HH:mm:ss` | 开始日期；`startdatestr` 会写入主表日期 |
| `endDate` / `enddate` / `enddatestr` | string | 否 | `yyyy-MM-dd` 或 `yyyy-MM-dd HH:mm:ss` | 结束日期；`enddatestr` 会写入主表日期 |
| `datarange` / `dataRange` / `reportTimeType` | string | 否 | 常见 `cury`，其他快捷值以租户主服务为准 | 快捷日期范围；不传或传空字符串表示不使用快捷日期范围，显式传 `cury` 才由 main-svc 按当前年解释 |
| `chart.type` / `dashboardtype` | string | 否 | `none`、`column_0`、`column_1`、`bar_0`、`bar_1`、`line_0`、`line_1`、`pie`、`donut`、`column_duidie`、`bar_duidie`、`area`、`funnel`、`gauge` | 图表类型 |
| `chart.x` / `xcon` | string | `isshowchart=true` 时必填 | 字段 ID 或 `totalrecord` | 图表统计值 |
| `chart.y` / `ycon` | string | `isshowchart=true` 时必填 | 分组字段 ID | 图表第一分组 |
| `chart.xGather` / `xgather` | string | 否 | 字段 ID | 图表统计辅助字段 |
| `chart.summaryWay` / `summaryway` | string | 否 | `sum`、`avg`、`max`、`min`、`unique`；图表默认 `sum` | 图表统计方式 |
| `chart.recordNum` / `recordnum` | string/number | 否 | 正整数；图表默认 `5` | 图表条目数 |
| `chart.unit` / `unit` | string | 否 | 常见 `integral`；图表默认 `integral` | 数值单位/格式 |
| `chart.sortType` / `dashboardsorttype` | string | 否 | `asc`、`desc` | 图表排序方向 |
| `chart.sortCondition` / `dashboardsortcondition` | string | 否 | `gather`、`value`、`label`；图表默认 `value` | 图表排序依据 |
| `options.showChart` / `isshowchart` | string/boolean | 否 | `true`、`false`；默认 `false` | 是否显示图表 |
| `options.showDetail` / `isshowdetail` | string/boolean | 否 | `true`、`false`；默认 `false` | 是否显示明细 |
| `options.showSubtotal` / `isshowsubtotal` | string/boolean | 否 | `true`、`false`；默认 `true` | 是否显示小计 |
| `options.showTotal` / `isshowtotal` | string/boolean | 否 | `true`、`false`；默认 `true` | 是否显示总计 |
| `options.showCurrency` / `isshowcurrency` | string/boolean | 否 | `true`、`false`；默认 `false` | 是否显示币种 |
| `currency` / `options.currency` | string | 否 | `CNY` 等币种代码；默认 `CNY` | 币种 |

## 18. 四类报表最小参数与期待值

### 18.1 列表式 Tabular

最小 JSON：

```json
{
  "name": "客户列表",
  "objecta": "account",
  "fields": [{ "fieldId": "field_account_name", "objectId": "account" }]
}
```

| 参数 | 必填 | 期待值 | 作用 |
| --- | --- | --- | --- |
| `reporttype` / `type` | 类型化命令自动补 | `Tabular` | 列表式报表 |
| `objecta` / `mainObjectId` / `source.objects[0].objectId` | 是，除非传 `reporttypecustomid` | 对象 API 名或对象 ID | 主对象 |
| `fields[]` / `mainobjectcolumnid` | 是 | 字段 ID 数组或 CSV | 明细展示字段 |

### 18.2 汇总式 Summary

最小 JSON：

```json
{
  "name": "客户负责人汇总",
  "objecta": "account",
  "fields": [{ "fieldId": "field_account_name" }],
  "rows": [{ "fieldId": "owner_id", "sort": "asc" }],
  "summaryFields": ["amount"]
}
```

| 参数 | 必填 | 期待值 | 作用 |
| --- | --- | --- | --- |
| `reporttype` / `type` | 类型化命令自动补 | `Summary` | 汇总式报表 |
| `groups.rows[]` / `rows[]` | 是 | 最多 3 个；每项含 `fieldId` | 行分组，等价 `transversegroupone/two/three` |
| `groups.rows[].sort` | 否 | `asc`、`desc`；默认 `asc` | 分组排序 |
| `groups.rows[].dateType` | 日期字段建议 | `day`、`week`、`month`、`quarter`、`year`、`FY`、`FQ`、`CY`、`CQ`；默认 `month` | 日期粒度 |
| `summaries[]` / `summaryFields[]` / `gatherFields[]` / `totalrecord` | 是 | `summaries[].method` 支持 `sum`、`avg`、`max`、`min`、`unique`；`totalrecord=1` 表示统计记录数 | 汇总统计 |

### 18.3 矩阵式 Matrix

最小 JSON：

```json
{
  "name": "客户月份矩阵",
  "objecta": "account",
  "fields": [{ "fieldId": "field_account_name" }],
  "rows": [{ "fieldId": "owner_id" }],
  "columns": [{ "fieldId": "createdate", "dateType": "month" }],
  "summaryFields": ["amount"]
}
```

| 参数 | 必填 | 期待值 | 作用 |
| --- | --- | --- | --- |
| `reporttype` / `type` | 类型化命令自动补 | `Matrix` | 矩阵式报表 |
| `groups.rows[]` / `rows[]` | 是 | 最多 3 个 | 业务语义上的行分组 |
| `groups.columns[]` / `columns[]` | 是 | 最多 2 个 | 业务语义上的列分组 |
| `summaryFields[]` / `summaries[]` / `totalrecord` | 是 | 同 Summary | 统计值 |

注意：main-svc 保存 Matrix 时会把业务语义上的列分组写入 `tp_sys_report.transversegroupone/two`，把业务语义上的行分组写入 `lengthwaysgroupone/two`。MetadataService 已按这个落库规则交换，不需要调用方手动交换。

### 18.4 同比/环比 ratio

最小 JSON：

```json
{
  "name": "客户同比环比",
  "objecta": "account",
  "dateFieldId": "createdate",
  "datelocation": "first",
  "ratioExpressions": [
    { "type": "TB", "fieldId": "totalrecord", "dateFieldId": "createdate" }
  ]
}
```

| 参数 | 必填 | 期待值 | 作用 |
| --- | --- | --- | --- |
| `reporttype` / `type` | 类型化命令自动补 | `ratio` | 同比/环比报表 |
| `dateFieldId` / `dateField` / `groupFieldId` / `transversegroupone` | 是 | 日期字段 ID | 时间分组字段 |
| `transversedatetypeone` | 否 | `day`、`week`、`month`、`quarter`、`year`、`FY`、`FQ`、`CY`、`CQ`；默认 `month` | 时间粒度 |
| `datelocation` | 否 | `first` 或 `end` | 对比日期定位；main-svc 期待值 |
| `ratioExpressions[]` / `comparisonExpressions[]` / `tbhbExpressions[]` | 建议 | 对象数组；CLI/MetadataService 会包装为 `{ "data": [...] }` 字符串并写入 `tbhbexpression` | 同比/环比表达式 |
| `mainobjectcolumnid` | 否 | 默认补 `totalrecord,<dateField>` | 展示/统计字段 |

### 18.5 通用期待值

| 参数 | 期待值 |
| --- | --- |
| `optionb` / `optionc` / `optiond` | `inner` 或 `outer` |
| `scope` | 常用 `user`、`all`；类型化命令默认 `user` |
| `islightning` | `true` 或 `false`；类型化命令默认 `true` |
| `totalrecord` | `1` 表示统计记录数；空或 `0` 表示不启用 |
| `isshowchart` / `isshowdetail` / `isshowsubtotal` / `isshowtotal` / `isshowcurrency` | `true` 或 `false` |

## 19. 开发前检查

- 已确认 MetadataService 地址和 token 配置可用。
- 已查询目标租户的对象、字段、报表文件夹和权限。
- 已确认 `reporttype` 与分组字段组合。
- 已确认 `reporttypecustomid` 是否需要，以及它不是 `reporttype`。
- 已确认报表文件夹 ID；缺省时接受默认 `baf20200821lightning`。
- 已将复杂筛选条件整理为 `filters.items` 或合法 `conditionVals` 字符串。
- 已检查图表字段、汇总字段、日期分组字段是否引用真实字段 ID。
- 已先生成 plan 并复核步骤，再执行 `apply msapi`。

---

## 19. 常见问题

### 19.1 为什么新建文件夹默认写 `lightning`

当前服务把 `foldertype=report` 和 `foldertype=lightning` 都视为报表文件夹。新建文件夹默认写 `lightning`，用于和当前 Lightning 报表文件夹语义保持一致。

### 19.2 不传报表文件夹会怎样

不传 `folderId`、`reportFolderId` 或 `reportfolderid` 时，报表默认进入 CloudCC 未归档公用报表文件夹：

```text
baf20200821lightning
```

### 19.3 更新文件夹名称会不会移动报表

不会。只有显式传 `reportIds` 或 `folderReport` 时才会调整文件夹内报表。

### 19.4 `conditionVals` 为什么是字符串

`conditionVals` 兼容 main-svc `saveReport` 的输入 envelope，因此外层报表 JSON 中 `conditionVals` 是 string，string 内容才是条件 JSON。MetadataService 会解析 `conditionVals.data[]` 或 `filters.items[]` 并生成 `tp_sys_condition` 行；当前 main-svc parity 不要求把这段字符串写入 `tp_sys_report` 主表。

### 19.5 应该优先用结构化写法还是直接字段写法

新建 CLI 自动化建议优先用结构化写法，因为结构更清晰。需要精确控制落库字段时，可以使用直接字段写法。
