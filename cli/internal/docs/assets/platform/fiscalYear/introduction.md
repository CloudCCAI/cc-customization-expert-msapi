# CloudCC 会计年度

会计年度（Fiscal Year）对应 setup-web 路由 `/settings/companyInformation/companyFiscalYear`，用于维护公司级年度期间及下级会计季度。

## 能力边界

- 读：`GET /metadata/v1/fiscal-years`、`GET /metadata/v1/fiscal-years/{id}`
- 写：通过 MetadataService `fiscal-years` domain 生成 plan，再显式 apply
- setup-service 对照：`/api/fiscalYear/queryFiscalYearList`、`/api/fiscalYear/fiscalYearDetail`、`/api/fiscalYear/saveFiscalYear`、`/api/fiscalYear/delFiscalYear`、`/api/fiscalYear/saveFiscalQuarter`、`/api/fiscalYear/delFiscalQuarter`
- 季度归属：会计季度是会计年度下级记录，写入 `tp_sys_fiscalquarter`

## 常用 CLI

```bash
cloudcc get fiscalYear .
cloudcc detail fiscalYear . 2026
cloudcc create fiscalYear . 2027 2027-01-01 2027-12-31 "FY 2027"
cloudcc createQuarter fiscalYear . fy-2027 "第一季度" 2027-01-01 2027-03-31
cloudcc deleteQuarter fiscalYear . fq-2027-q1
cloudcc apply msapi . <planId>
```

会计年度写入会做年度日期重叠校验。季度写入会校验季度日期落在所属年度内，且同一年度下季度互不重叠；plan metadata 中会标记 `quartersIncluded=true`。
