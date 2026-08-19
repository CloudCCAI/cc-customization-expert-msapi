# CloudCC 报表能力介绍

## 1. 什么是报表

CloudCC 报表是面向业务数据分析、经营监控和管理决策的低代码分析能力。

它允许用户基于 CloudCC 对象、字段、关联关系和记录数据，配置查询字段、筛选条件、分组维度、汇总方式和图表展示，把分散在 CRM、服务、项目、审批等业务对象中的数据整理成可读、可统计、可追踪的分析视图。

报表不是高代码资源，也不是本地文件发布能力。当前 CLI 中的 `report` 和 `reportFolder` 属于低代码元数据：读取通过 MetadataService 只读接口完成，创建、更新、删除通过 MetadataService 计划生成，执行必须显式 `apply msapi <planId>`。

---

## 2. 报表能做什么

报表通常用于：

- 查询业务记录明细，例如客户列表、商机明细、工单明细。
- 按维度汇总统计，例如按负责人统计销售额，按区域统计客户数。
- 交叉分析数据，例如按区域和产品形成矩阵报表。
- 做同环比分析，例如按月份观察新增客户、销售额或服务量变化。
- 用图表呈现统计结果，例如柱状图、条形图、折线图、饼图、漏斗图、仪表盘图。
- 通过报表文件夹组织报表，并控制报表的可见范围。

报表经常和这些模块联动：

- `object`：报表的数据来源。
- `fields`：报表展示、过滤、分组和汇总的字段来源。
- `profile` / `role` / `permission` / `sharingRule`：报表可见性和数据范围的权限基础。
- `dashboard`：报表图表可作为仪表板展示内容。
- `view`：部分列表视图和报表在业务表达上会相互参考，但落库表和用途不同。

---

## 3. 当前 CLI 能力边界

当前 Go 版 CLI 支持：

- 查询报表列表，支持文件夹筛选、关键字、分页和排序。
- 查询报表详情。
- 创建报表计划。
- 创建和更新矩阵式报表计划时，可使用 `reportMatrix` 专用快捷命令做必填项校验和参数补全。
- 更新报表计划。
- 删除报表计划。
- 查询报表文件夹列表和详情。
- 创建、更新、删除报表文件夹计划。
- 使用中文内置文档查看报表参数、报表类型、筛选条件、分组、汇总、图表和文件夹语义。

当前 CLI 不做：

- 不通过其他写入通道保存报表。
- 不绕过 MetadataService 直接写数据库。
- 不执行报表数据运行期查询或导出。
- 不下载报表文件。
- 不封装定时订阅等运行期功能。

---

### 3.1 类型化报表快捷命令

当前 CLI 支持四个类型化报表写入入口：

- `reportTabular`：列表式报表，固定 `reporttype=Tabular`，要求数据源和展示字段。
- `reportSummary`：汇总式报表，固定 `reporttype=Summary`，要求数据源、展示字段、行分组和统计字段或 `totalrecord`。
- `reportMatrix`：矩阵式报表，固定 `reporttype=Matrix`，要求数据源、展示字段、行分组、列分组和统计字段或 `totalrecord`。
- `reportRatio`：同比/环比报表，固定 `reporttype=ratio`，要求数据源和日期分组字段，`datelocation` 期待值为 `first` 或 `end`。

所有类型化命令只生成 MetadataService plan，不直接写库；执行仍需 `cloudcc apply msapi <projectPath> <planId>`。

## 4. 关键概念

### 4.1 `reporttype`

`reporttype` 是报表结构类型，表示报表是列表式、汇总式、矩阵式还是同环比报表。

常见取值：

- `Tabular`：列表式报表。
- `Summary`：汇总报表。
- `Matrix`：矩阵式报表。
- `ratio`：同比/环比报表。

如果没有显式传 `reporttype`，MetadataService 会根据分组字段推断：

- 没有行分组、没有列分组：`Tabular`。
- 有行分组、没有列分组：`Summary`。
- 有行分组、也有列分组：`Matrix`。
- 显式传 `ratio`：同环比报表。

### 4.2 `reporttypecustomid`

`reporttypecustomid` 是报表数据源或自定义报表类型 ID，不是报表结构类型。

它表示报表基于哪组对象、对象关联关系和可选字段取数。不要把它和 `reporttype` 混用。
创建或更新报表时，如果没有现成的 `reporttypecustomid`，可以通过 `source.objects` 描述主对象和关联对象，并通过 `fields` 描述可选字段；MetadataService 会生成对应的数据源模型并把 ID 写回报表主表。

注意 `source.objects` 与顶层 `objects` 含义不同：`source.objects` 是报表数据源定义，顶层 `objects` 只用于显式维护报表对象扩展配置。

### 4.3 `reportfolderid`

`reportfolderid` 是报表所属文件夹 ID。

如果创建或更新报表时没有传 `folderId`、`reportFolderId` 或 `reportfolderid`，MetadataService 会默认使用 CloudCC 未归档公用报表文件夹：

```text
baf20200821lightning
```

### 4.4 报表文件夹类型

当前服务读取报表文件夹时识别两类 `foldertype`：

- `report`：报表文件夹类型。
- `lightning`：Lightning 报表文件夹类型。

当前 CLI 和 MetadataService 查询报表文件夹时会同时识别 `report` 与 `lightning`。新建报表文件夹默认写 `foldertype=lightning`。

---

## 5. 推荐使用方式

开发或实施报表前，建议按这个顺序确认：

1. 使用 `cloudcc scan msapi <projectPath> standard-catalog` 或报表详情确认对象、字段和已有报表文件夹。
2. 明确报表类型：列表式、汇总式、矩阵式或同环比。
3. 明确数据源：是否需要 `reporttypecustomid`，涉及哪些对象和关联字段。
4. 明确展示字段：写入 `mainobjectcolumnid` 或 DSL 的 `fields`。
5. 明确筛选条件：使用 `filters.items`，或直接传 `conditionVals` JSON 字符串。
6. 明确分组与汇总：使用 `groups.rows`、`groups.columns`、`summaries`，或直接传 `transversegroup*`、`lengthwaysgroup*`、`gatherfieldname`。
7. 明确图表：使用 `chart`，或直接传 `dashboardtype`、`xcon`、`ycon`、`summaryway`。
8. 先 `create/update report` 生成 plan，复核计划步骤后再 `apply msapi <planId>`。

---

## 6. 相关命令

```bash
cloudcc get report <projectPath> [folderId] [searchKeyWord] [page] [pageSize] [orderField] [orderType]
cloudcc detail report <projectPath> <reportId>
cloudcc create report <projectPath> <encodedReportJson>
cloudcc update report <projectPath> <reportId> <encodedReportJson>
cloudcc create reportTabular <projectPath> <encodedReportJson|@file>
cloudcc update reportTabular <projectPath> <reportId> <encodedReportJson|@file>
cloudcc create reportSummary <projectPath> <encodedReportJson|@file>
cloudcc update reportSummary <projectPath> <reportId> <encodedReportJson|@file>
cloudcc create reportMatrix <projectPath> <encodedMatrixReportJson|@file>
cloudcc update reportMatrix <projectPath> <reportId> <encodedMatrixReportJson|@file>
cloudcc create reportRatio <projectPath> <encodedReportJson|@file>
cloudcc update reportRatio <projectPath> <reportId> <encodedReportJson|@file>
cloudcc delete report <projectPath> <reportId> [confirmdelete]

cloudcc get reportFolder <projectPath> [searchKeyWord]
cloudcc detail reportFolder <projectPath> <folderId>
cloudcc create reportFolder <projectPath> <name> [encodedOptionsJson]
cloudcc update reportFolder <projectPath> <folderId> <encodedOptionsJson>
cloudcc delete reportFolder <projectPath> <folderId>

cloudcc doc platform/report introduction
cloudcc doc platform/report devguide
cloudcc doc report introduction
cloudcc doc report devguide
```

`cloudcc doc report ...` 是 `cloudcc doc platform/report ...` 的短别名。
