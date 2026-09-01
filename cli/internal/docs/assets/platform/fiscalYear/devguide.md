# 会计年度 CLI 用户级与开发说明

会计年度属于低代码元数据能力。MSAPI/Universal 选择 MetadataService 时，`fiscalYear`、`fiscal-year`、`fiscal-years` 都映射到 canonical domain `fiscal-years`。该 domain 同时管理年度及其下级会计季度。

## 读取

```bash
cloudcc get fiscalYear <projectPath> [filter]
cloudcc detail fiscalYear <projectPath> <id-or-year-name>
```

读取调用 MetadataService 只读接口：

- `cloudcc get fiscalYear .` -> `GET /metadata/v1/fiscal-years`
- `cloudcc get fiscalYear . 2026` -> `GET /metadata/v1/fiscal-years?filter=2026`
- `cloudcc detail fiscalYear . 2026` -> `GET /metadata/v1/fiscal-years/2026`

返回的 `fiscalYears[]` 包含年度 ID、年度名称、开始日期、结束日期、描述、排序序号和审计字段。详情接口同时返回 `fiscalQuarters[]`、`relatedRows.tp_sys_fiscalyear`、`relatedRows.tp_sys_fiscalquarter` 与原始行关系，便于验收和差异比对。

## 创建年度计划

简写：

```bash
cloudcc create fiscalYear <projectPath> <year-name> <startDate> <endDate> [description]
```

示例：

```bash
cloudcc create fiscalYear . 2027 2027-01-01 2027-12-31 "FY 2027"
cloudcc apply msapi . <planId>
```

JSON：

```bash
cloudcc plan msapi . fiscal-years '{"id":"fy-2027","name":"2027","startDate":"2027-01-01","endDate":"2027-12-31","description":"FY 2027"}' create
cloudcc plan msapi . fiscal-years '{"id":"fy-2027","name":"2027","startDate":"2027-01-01","endDate":"2027-12-31","quarters":[{"id":"fq-2027-q1","name":"第一季度","startDate":"2027-01-01","endDate":"2027-03-31"}]}' create
cloudcc apply msapi . <planId>
```

年度字段：

| 字段 | 必填 | 说明 |
|------|------|------|
| `id` / `fiscalYearId` | 否 | 年度 ID；省略时由 MetadataService 生成 |
| `name` / `fiscalYearName` | 是 | 年度名称，必须是数字，如 `2027` |
| `startDate` / `fiscalyearStartdate` | 是 | 年度开始日期，格式 `yyyy-MM-dd` |
| `endDate` / `fiscalyearEnddate` | 是 | 年度结束日期，格式 `yyyy-MM-dd` |
| `description` / `fiscalYearDescription` | 否 | 描述 |
| `sequence` / `fiscalyearseq` | 否 | 排序序号；需要完全复刻线上顺序时显式传入 |
| `quarters[]` / `fiscalQuarters[]` / `objList[]` | 否 | 下级会计季度列表 |

## 创建或更新季度计划

简写：

```bash
cloudcc createQuarter fiscalYear <projectPath> <fiscal-year-id> <quarter-name> <startDate> <endDate> [description] [quarter-id]
```

示例：

```bash
cloudcc createQuarter fiscalYear . fy-2027 "第一季度" 2027-01-01 2027-03-31 "FY 2027 Q1" fq-2027-q1
cloudcc apply msapi . <planId>
```

JSON：

```bash
cloudcc plan msapi . fiscal-years '{"fiscalYearId":"fy-2027","name":"第二季度","startDate":"2027-04-01","endDate":"2027-06-30"}' saveFiscalQuarter
cloudcc apply msapi . <planId>
```

季度字段：

| 字段 | 必填 | 说明 |
|------|------|------|
| `id` / `quarterId` / `fiscalQuarterId` | 否 | 季度 ID；省略时由 MetadataService 生成 |
| `fiscalYearId` / `fiscalyearid` | 单独季度保存必填 | 所属年度 ID；嵌套在年度 spec 下时默认使用当前年度 ID |
| `name` / `quarterName` / `fiscalQuarterName` | 是 | 季度名称 |
| `startDate` / `fiscalQuarterStartdate` | 是 | 季度开始日期，`yyyy-MM-dd` |
| `endDate` / `fiscalQuarterEnddate` | 是 | 季度结束日期，`yyyy-MM-dd` |
| `description` / `fiscalQuarterDescription` | 否 | 描述 |
| `sequence` / `fiscalQuarterSeq` | 否 | 排序序号 |

## 删除季度计划

```bash
cloudcc deleteQuarter fiscalYear <projectPath> <quarter-id>
cloudcc plan msapi <projectPath> fiscal-years '{"quarterId":"fq-2027-q1"}' delete-quarter
cloudcc apply msapi <projectPath> <planId> '{"approval":"..."}'
```

删除季度会生成 `tp_sys_fiscalquarter` 删除步骤，plan metadata 标记 setup-service 来源 `/api/fiscalYear/delFiscalQuarter`。

## 删除年度计划

```bash
cloudcc delete fiscalYear <projectPath> <fiscal-year-id>
cloudcc apply msapi <projectPath> <planId> '{"approval":"..."}'
```

删除年度计划会先删除该年度下 `tp_sys_fiscalquarter` 记录，再删除 `tp_sys_fiscalyear` 记录。删除是 destructive plan，必须复核 plan 后显式 apply。是否允许删除已被业务数据引用的年度，应以目标租户实际规则和服务端错误为准。

## 校验规则

- 年度名称必须为数字。
- 年度 `endDate` 不能早于 `startDate`。
- plan 和 apply 都会基于 `tp_sys_fiscalyear` 检查年度日期区间重叠。
- 季度范围必须落在所属年度范围内。
- 同一年度下季度范围不能互相重叠，计划内多个季度和数据库已有季度都会被校验。
- 计划包含季度时 `quartersIncluded=true`；只更新年度主数据且无季度时为 `false`。

## setup-service 对照

| CLI/MetadataService | setup-service |
|---------------------|---------------|
| list | `/api/fiscalYear/queryFiscalYearList` |
| detail | `/api/fiscalYear/fiscalYearDetail` |
| create/update plan | `/api/fiscalYear/saveFiscalYear` |
| delete plan | `/api/fiscalYear/delFiscalYear` |
| quarter create/update plan | `/api/fiscalYear/saveFiscalQuarter` |
| quarter delete plan | `/api/fiscalYear/delFiscalQuarter` |
