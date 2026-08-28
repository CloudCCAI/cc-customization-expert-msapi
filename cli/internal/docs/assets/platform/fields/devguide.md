# CloudCC 字段开发指南

## 1. 模块定位

`fields` 模块用于管理 CloudCC 对象字段，当前提供字段列表查询、字段详情、创建、更新、删除能力。

可通过以下命令读取文档：

```bash
cloudcc doc platform/fields introduction
cloudcc doc platform/fields devguide
```

---

## 2. 当前支持的命令

```bash
cloudcc get fields <projectPath> <objPrefix>
cloudcc detail fields <projectPath> <fieldId> <fdtype> <objid>
cloudcc create fields <projectPath> <fieldType> <objid> <fieldLabel> <remark> [extraArg]
cloudcc update fields <projectPath> <fieldId> <fieldType> <objid> <fieldLabel> <remark> [extraArg]
cloudcc delete fields <projectPath> <fieldId> <objid>
```

说明：

- `projectPath`：本地项目路径，用于读取 `cloudcc-cli.config.js`
- `objPrefix`：对象前缀，用于查询字段
- `fieldType`：字段类型编码（例如 `S`、`N`、`D` 等）
- `objid`：对象 ID
- `fieldLabel`：字段显示名称
- `extraArg`：部分字段类型需要的额外参数，如 `ptext` 或 `lookupObj`
- `remark`：用于描述字段的业务功能（写入字段模板的 `obj.remark`，位置固定为 `argvs[6]`）
- `fieldId`：字段 ID（`update` / `delete` 必填；可通过 `get fields` 返回的 `id` 获取）

说明：

- 当前 CLI 参数以 `src/fields/buildFieldData.js`、`src/fields/create.js`、`src/fields/update.js`、`src/fields/get.js`、`src/fields/delete.js` 的实际实现为准
- 字段 API 名称在当前创建逻辑中默认由 CLI 自动生成，不要求单独传入

### 2.1 MetadataService 物理槽位规则

- `dataFieldRef`（如 `str_field1`、`date_field1`）是对象内唯一的物理存储槽位，不是可随意复用的业务标识。
- MetadataService 创建或 upsert 字段时会查询 `tp_sys_schemetable`，保留已有字段自己的映射，并按字段类型选择首个可用空槽；例如已占用 `str_field1`、`str_field3` 时，新文本类字段使用 `str_field2`。
- 普通字段计划应省略 `dataFieldRef`，由服务端分配。只有迁移或精确回放等确有需要的场景才显式提供，并且必须保证同一对象内唯一。
- plan 阶段会拒绝现有映射冲突和同一计划内的重复槽位；apply 阶段会在事务内重新锁定并检查，防止陈旧或并发计划覆盖其他字段。
- 收到 `field_slot_conflict` 时，重新执行目标环境 scan 并生成新 plan；不要修改数据库或绕过 MetadataService 守卫。

### 2.2 MetadataService 完整字段元数据

`cloudcc plan msapi <projectPath> fields @field.json create` 会按平台元数据保存规则展开字段元数据；对象计划中的 `fields[]` 也使用同一条展开链路。普通 CLI `create fields` 的位置参数只覆盖基础字段模板，自动编号、查找筛选和相关列表等高级配置应使用 MetadataService spec。

字段创建如果会进入页面布局，必须先按 `platform/pagelayout devguide` 的页面布局配置方法论判断落位。能读取对象布局详情时，优先在字段 spec 中显式提供 `layoutPlacements`，或在字段创建后通过 `pagelayout detail` / `pagelayout update` 调整 PC 和 mobile 布局。只有缺少布局上下文时才允许依赖 MetadataService 自动摆放，并在输出中标注为兜底。

#### 2.2.1 批量添加字段

MetadataService `fields` 计划支持在同一个 spec 中通过 `fields[]` 批量添加字段：

```json
{
  "objectId": "<objectId>",
  "objectApiName": "<objectApiName>",
  "onExisting": "createOnly",
  "fields": [
    {"apiName": "approval_status", "label": "审批状态", "type": "L", "options": [{"value": "待审批"}]},
    {"apiName": "approved_at", "label": "审批时间", "type": "F"},
    {"apiName": "archive_owner", "label": "档案负责人", "type": "Y", "lookupObjectId": "<userObjectId>"}
  ]
}
```

执行方式：

```bash
cloudcc plan msapi <projectPath> fields @fields-batch.json create
cloudcc apply msapi <projectPath> <planId> '{"async":true}'
cloudcc operation msapi <projectPath> <applyId>
```

批量字段创建可能会展开字段主表、语言、选项、权限、布局、筛选、相关列表和数据库视图刷新等步骤，推荐使用异步 apply。异步 apply 会立即返回 `applyId`，当前实现中 `applyId` 与 `operationId` 相同；后续用 `cloudcc operation msapi <projectPath> <applyId>` 轮询，直到状态为 `VERIFIED` / `APPLIED` / `FAILED`。如果状态仍是 `APPLYING`，不要重新提交同一个字段批量创建 plan。

批量策略：

- `createOnly`：默认策略。目标对象上已存在同 API 名字段，或显式 `id` 已存在时，该字段在 plan metadata 的 `batchItemResults` 中标记为 `FAILED_PRECHECK`，不影响其它无关字段继续生成步骤。
- `skipExisting`：已存在字段跳过，只为不存在的字段生成创建步骤；plan metadata 会返回 `batchCreateCount`、`batchSkipCount` 和 `batchFields` 摘要。
- `updateExisting`：已存在字段转为 `field.update`，不存在字段继续创建；适合补标签、权限、选项、布局等元数据。
- `upsertByApiName`：按 `objectId + apiName` 解析，存在则更新，不存在则创建；`upsert` 操作默认使用该策略。

批量添加不是简单把多个命令拼在一起。plan 阶段会先解析所有字段目标，检查同一对象内重复 API 名、重复显式字段 ID、重复显式 `dataFieldRef`，并查询目标环境已有字段。可在执行前判断的问题会按字段写入 `batchItemResults`：`PLANNED` 表示已生成步骤，`SKIPPED` 表示被策略跳过，`FAILED_PRECHECK` 表示该字段没有生成 SQL 步骤且会带 `error` / `message`。同时读取 `batchExecutableCount` 和 `batchPrecheckFailedCount` 可以判断本次批量还有多少字段会进入 apply。如果整批字段都只剩 `FAILED_PRECHECK`，`apply` 会直接把 operation 标记为 `FAILED`。

apply 阶段会优先批量执行字段主表行，再按既有顺序执行语言、选项、权限、布局、筛选和相关列表等展开步骤。字段主表写入仍使用批处理和事务保护；如果执行期 SQL、并发槽位校验或运行时副作用失败，operation 可能整体 `FAILED`。逐项结果只用于执行前预检，不表示执行期会把同一个事务拆成逐字段提交。

物理槽位仍然由服务端按对象维度统一分配：同一个 `fields[]` 中多个文本字段会依次占用 `str_field1`、`str_field2` 等首个可用槽位；已有字段的槽位会先被保留，不会被新字段抢占；预检失败的字段不计入可执行字段数量，也不会占用同对象字段数量上限。除迁移回放外不要手写 `dataFieldRef`；如果必须手写，必须保证同一对象内不重复，否则该字段会在 plan 阶段成为 `FAILED_PRECHECK`，或在并发 apply 校验中触发 `field_slot_conflict`。

如果一个批量 spec 中的字段分属不同对象，可以在每个字段上覆盖 `objectId`；未覆盖时继承根级 `objectId`。查找字段、主详字段、自动编号、公式、累计汇总、地址和地理定位等高级字段仍使用下文相同字段定义，批量只是提交载体，不会降低单字段校验要求。

自动编号字段必须同时生成字段行和 `tp_sys_autonum` 配置，不能只把字段类型设为 `V`：

```json
{
  "objectId": "<sourceObjectId>",
  "apiName": "serial_no",
  "label": "流水号",
  "type": "V",
  "showFormat": "NO-{0000}",
  "beginIndex": "1"
}
```

新建自定义对象且名称字段类型为自动编号时，对象 spec 可直接提供 `showFormat`、`beginIndex`；MetadataService 会为默认名称字段生成对应配置。已有对象若历史上缺少 `tp_sys_autonum`，应先审计并补齐数据，不能依赖编辑页容忍空结果。

查找字段筛选支持 `conditionVals` 条件配置；计划会设置主字段的 `isenableFilter` / `filterType` / `filterLogic`，并替换该字段的 `tp_sys_lookup_filter_condition` 条件行：

```json
{
  "objectId": "<sourceObjectId>",
  "apiName": "archive_definition_id",
  "label": "档案定义",
  "type": "Y",
  "lookupObjectId": "obj_archive_definition",
  "isenableFilter": true,
  "filterType": "1l,2r",
  "conditionVals": {
    "filter": "1 AND 2",
    "data": [
      {"leftvalue":"obj_archive_definition-<statusFieldId>","op":"e","valfld":"value","val":"Active","rightvalue":""},
      {"leftvalue":"obj_archive_definition-<ownerFieldId>","op":"e","valfld":"field","val":"","rightvalue":"<sourceObjectId>-<ownerFieldId>"}
    ]
  }
}
```

其中 `valfld=value` 使用 `val` 常量；`valfld=field` 使用 `rightvalue=<objectId>-<fieldId>`。重新发布带筛选的字段会先删除该字段的旧条件，再按顺序写入新条件，避免已删除条件继续生效。

查找关系（`Y`）和主详关系（`M`）未显式传 `relatedLists` 时，MetadataService 会按 `lookupObjectId` 查询被关联对象的所有布局并创建相关列表；`childrelationName` 是相关列表标签，`mainlayoutIds` 决定桌面/移动布局是否显示，默认列使用来源对象的名称字段，并同步中文、英文、日文标签。例如：

```json
{
  "objectId": "<sourceObjectId>",
  "apiName": "archive_definition_id",
  "label": "档案定义",
  "type": "Y",
  "lookupObjectId": "obj_archive_definition",
  "childrelationName": "来源记录",
  "mainlayoutIds": "<targetRootLayoutId>"
}
```

`lookupObjectId` / `lookupObj` 必须传被关联对象在 `tp_sys_object.ID` 中的对象 ID。不要传对象 API 名（`SCHEMETABLE_NAME`）、物理表名（`DATATABLE_NAME`）或显示名称。标准对象示例中 `account` 可能同时是对象 ID 和 API 名，容易造成误解；自定义对象通常不同，必须先解析出真实对象 ID 再创建查找字段。

对象默认搜索布局相关的字段、按钮和 lookup-layout ID 必须保持在平台 `rel_id` 的 20 字符边界内。MetadataService 生成的默认 ID 使用短 ID；不要在 spec 中覆盖为超长 ID，否则旧版 `to_mlang` 查询可能报 `Data too long for column 'rel_id'`。

### 2.3 MetadataService 字段类型矩阵

MetadataService 支持平台开放的全部字段编码。编码区分大小写：`c` 是币种，`C` 是累计汇总；`ENC` / `ENCD` 可作为输入别名，保存时会规范为平台字段编码 `enc` / `encd`。

字段 spec 可传 `labels`、`translations`、`multiLang` 或 `multiLanguages` 数组；每项使用 `country|locale|language|lang`、`label|labelName|name|value` 和 `isDefault|isdefault`。只要提供显式数组，MetadataService 就逐条保留全部语言和默认标志，不自动缩减为 zh/en。`remark`/`description` 未提供时保持空值。全局选项字段必须使用目标环境真实回读的 `globalSelectId`，本地选项为空但存在明确全局关联时不会触发本地值必填错误。

| 字段族 | 类型 | 物理槽位与关键展开 |
| --- | --- | --- |
| 标量 | `S U P c N T B H E SCORE` | `str_fieldN`；同步长度、精度、默认值、重复/掩码等属性 |
| 日期时间 | `D F` | 分别使用 `date_fieldN`、`time_fieldN` |
| 大文本 | `X MR` | `longtext_fieldN` |
| CLOB | `J A` | `clob_fieldN` |
| 媒体文件 | `IMG FL` | `str_fieldN`；同步上传数量、图片表达式/水印、文件重复数量 |
| 选项 | `L Q` | `str_fieldN`；展开 `tp_sys_code`、多语言值、全局选项关系和记录类型依赖 |
| 关系 | `Y M` | `lookup_fieldN`；展开筛选、相关列表，apply 后幂等同步物理列索引；`M` 还更新对象主详层级 |
| 计算 | `Z C` | 固定 `datafieldRef=none`；持久化公式/汇总定义及条件引用 |
| 生成/加密 | `V enc encd` | `V` 展开 `tp_sys_autonum`；加密字段同步掩码和简档加密可见性 |
| 复合 | `AD LT` | 父字段为 `none`；分别展开 6 个地址子字段、2 个经纬度子字段及其语言/权限关系 |

公式字段不能只传展示公式。MetadataService 不会把未经审核的表达式自动拼成 SQL，因此 `formulaText`、`formulaType` 和 `executeExpression` 必须配套提供：

```json
{
  "objectId": "<objectId>",
  "apiName": "net_amount",
  "label": "净额",
  "type": "Z",
  "formulaType": "N",
  "formulaText": "amount-discount",
  "executeExpression": "t0.str_field1-t0.str_field2",
  "formulaReferences": [
    {"relationId":"<amountFieldId>","relationType":"field"},
    {"relationId":"<discountFieldId>","relationType":"field"}
  ]
}
```

累计汇总字段必须使用 MetadataService spec 创建，不要使用位置参数版 `cloudcc create fields ...`。CLI/MetadataService 会按平台元数据规则补齐保存所需数据：`datafieldRef=none`、`decimalPlaces`、`summaryfieldtype`、`displayThousands`、`isReplicable` 和 `executeExpression` 会根据汇总方法、子对象和汇总字段自动派生；调用方通常不需要也不应该自己拼 `executeExpression`。

必填字段：

- `objectId`：主对象 ID。
- `apiName` / `label`：累计汇总字段 API 名和显示名；显式 `id` 可选，但长度不能超过 20。
- `type`: 固定为 `C`。
- `childtype` / `expressionType`：汇总方法，常用 `COUNT`、`SUM`、`MIN`、`MAX`。
- `summarizedObj` / `childid`：`<子对象ID>:<子对象API名>:<指向主对象的主详/查找字段API名>`。不要使用物理表名或物理列名；MetadataService 会在能识别旧输入时自动归一化为平台字段编辑页可显示的 API 形态。
- `aggregateField` / `fieldid`：非 `COUNT` 必填，格式为 `<被汇总字段ID>:<字段类型>:<被汇总字段API名>`；公式字段可使用 `<字段ID>:Z:<字段API名>:<公式返回类型>`。不要使用 `DATAFIELD_REF` 物理列。

支持的 SUM 汇总字段类型包括数字 `N`、百分比 `P`、币种 `c`、评分 `SCORE`，以及返回这些类型的公式字段。`COUNT` 不需要 `aggregateField`。

单个累计汇总字段创建示例：

```json
{
  "objectId": "<masterObjectId>",
  "apiName": "line_total",
  "label": "明细合计",
  "type": "C",
  "childtype": "SUM",
  "summarizedObj": "<detailObjectId>:<detailObjectApiName>:<masterRelationFieldApiName>",
  "aggregateField": "<amountFieldId>:N:<amountFieldApiName>"
}
```

执行方式：

```bash
cloudcc plan msapi <projectPath> fields @rollup-field.json create
cloudcc apply msapi <projectPath> <planId>
```

也可以传兼容旧版字段别名；MetadataService 会兼容 `objid`、`fdtype`、`childid`、`childtype`、`fieldid`、`isaggfilter` 和嵌套 `obj`：

```json
{
  "objid": "<masterObjectId>",
  "fdtype": "C",
  "childid": "<detailObjectId>:<detailObjectApiName>:<masterRelationFieldApiName>",
  "childtype": "SUM",
  "fieldid": "<amountFieldId>:N:<amountFieldApiName>",
  "obj": {
    "apiname": "line_total",
    "nameLabel": "明细合计"
  }
}
```

批量创建累计汇总字段时，顶层写 `fields[]`；每一项和单字段 spec 相同。推荐异步 apply，并读取 `batchItemResults` 判断每个字段是否进入执行：

```json
{
  "objectId": "<masterObjectId>",
  "onExisting": "createOnly",
  "fields": [
    {
      "apiName": "line_count",
      "label": "明细数量",
      "type": "C",
      "childtype": "COUNT",
      "summarizedObj": "<detailObjectId>:<detailObjectApiName>:<masterRelationFieldApiName>"
    },
    {
      "apiName": "line_total",
      "label": "明细合计",
      "type": "C",
      "childtype": "SUM",
      "summarizedObj": "<detailObjectId>:<detailObjectApiName>:<masterRelationFieldApiName>",
      "aggregateField": "<amountFieldId>:N:<amountFieldApiName>"
    }
  ]
}
```

```bash
cloudcc plan msapi <projectPath> fields @rollup-fields-batch.json create
cloudcc apply msapi <projectPath> <planId> '{"async":true}'
cloudcc operation msapi <projectPath> <applyId>
```

筛选条件会先按 `relatedId=<summaryFieldId>` 清理旧行，再写入 `tp_sys_condition`。过滤型累计汇总需要同时提供两类信息：`conditionVals` / `summaryConditions` 用于平台字段编辑页显示和编辑条件行；`aggCondition` 或已审核的完整 `executeExpression` 用于生成实际执行 SQL。当前 MetadataService 不会自动把任意条件树完整编译成 SQL；为了避免生成漏筛选的 `executeExpression`，启用筛选时必须同时提供 `conditionVals` 和 `aggCondition`，或提供 `conditionVals` 加完整 `executeExpression`：

```json
{
  "objectId": "<masterObjectId>",
  "apiName": "active_line_total",
  "label": "生效明细合计",
  "type": "C",
  "childtype": "SUM",
  "summarizedObj": "<detailObjectId>:<detailObjectApiName>:<masterRelationFieldApiName>",
  "aggregateField": "<amountFieldId>:N:<amountFieldApiName>",
  "isAggfilter": true,
  "aggCondition": "status__c='Active'",
  "conditionVals": {
    "filter": "1",
    "data": [{"fieldId":"<statusFieldId>","op":"e","val":"Active","seq":1}]
  }
}
```

使用 MetadataService `1.1.36` 及以上时，筛选型累计汇总会按平台字段编辑页可显示的形态保存：字段定义使用 `isAggfilter=true`、`aggCondition`、`aggConditiondis` 表达累计汇总筛选；条件行写入 `MAIN_OBJ_ID`、`FIELD_ID`、`OPERATOR`、`VALUE`、`BOOL_FILTER`。字段编辑页依赖这些条件行还原筛选 UI，因此不要只提供 `aggCondition` 而省略 `conditionVals`。

如果过滤场景没有 `conditionVals` 条件行，plan 会返回 `summary_filter_conditions_required`，避免生成平台字段编辑页无法显示/编辑筛选条件的字段；如果有条件行但没有 `aggCondition` 且没有 `executeExpression`，plan 会返回 `summary_filter_sql_required`，不会生成一个可能漏数据的累计汇总字段。

主详字段会更新明细对象的 `accessable`、`is_master`、`parentobjid`，将主对象标记为 `master`，并向已有后代对象传播新的父路径：

```json
{
  "objectId": "<detailObjectId>",
  "apiName": "master_id",
  "label": "主记录",
  "type": "M",
  "lookupObj": "<masterObjectId>",
  "childrelationName": "明细"
}
```

地址和地理定位只需要声明父字段；子字段由服务端确定性生成并独立分配槽位。地址国家子字段还会自动关联 `country` 全局选项。不要自行把复合父字段映射到实体列：

```json
{
  "objectId": "<objectId>",
  "fields": [
    {"id":"<addressFieldId>","apiName":"billing_address","label":"账单地址","type":"AD"},
    {"id":"<locationFieldId>","apiName":"warehouse_location","label":"仓库位置","type":"LT","displayType":"1"}
  ]
}
```

创建前会执行平台字段约束：每对象最多 1 个主详字段、2 个自动编号、10 个长文本 `J`、10 个文件字段、25 个图片字段；文本/加密长度、数字精度、上传数量、多选可见行数、重复选项和值域也会在 plan 阶段直接拒绝。字段删除会同步清理选项、筛选、相关列表、依赖、引用、自动编号、全局选项、语言、权限以及复合子字段元数据。

---

## 3. 查询字段详情（detail）

用于拉取与平台「字段编辑」页一致的数据（简档列表、对象列表、`fieldObj` 等），对应接口 **`POST /api/fieldSetup/editField`**。

### 3.1 基本命令

```bash
cloudcc detail fields <projectPath> <fieldId> <fdtype> <objid>
```

- `fieldId`：字段 ID（与 `get fields` 返回的 `id` 一致）
- `fdtype`：字段类型编码（例如 `P`、`S`、`U`，与 `fdtype` / `schemefieldType` 一致）
- `objid`：所属对象 ID

请求体示例：

```json
{
  "fieldId": "ffe2026A3B035C748w95",
  "fdtype": "P",
  "objid": "2026BEECB242636G72cD"
}
```

成功时 CLI 将打印接口返回的 JSON（含 `result`、`data` 等，具体结构以平台为准）。

---

## 4. 查询字段列表（get）

### 4.1 基本命令

```bash
cloudcc get fields <projectPath> <objPrefix>
```
入参：
- `objPrefix`：对象前缀

返回结果包含：

- `obj`：对象基础信息
- `stdFields`：标准字段列表
- `cusFields`：自定义字段列表

每个字段通常包含：

- `fieldname`：字段显示名称
- `apiname`：字段 API 名称
- `schemefieldType`：字段类型编码
- `id`：字段 ID

---

## 5. 创建字段

### 5.1 基本命令

```bash
cloudcc create fields <projectPath> <fieldType> <objid> <fieldLabel> <remark> [extraArg]
```

#### 公用参数（所有字段类型通用）

- `projectPath`：本地项目路径
- `fieldType`：字段类型编码（例如 `S`、`U`、`L`、`Y` 等）
- `objid`：对象 ID
- `fieldLabel`：字段显示名称
- `remark`：字段业务功能描述（写入字段模板的 `obj.remark`；参数位置固定为 `argvs[6]`）
- `helps`：帮助说明，固定`argvs[7]`
- `defaultValue`：默认值，固定`argvs[8]`
- `extraArg`：仅部分字段类型需要的额外入参；其含义与参数位置随 `fieldType` 变化（通常从 `argvs[9]` 开始）

#### 字段特殊入参（按 `fieldType`）

以下均假定 **`create fields`** 的 argv 下标（**`update fields`** 因多一个 `fieldId`，`remark` 及之后的参数整体 **`+1`**，见 [5.2](#52-更新字段)）。

**约定**：**`helps`**、**`defaultValue`** 见上文公用参数（**`argvs[7]`**、**`argvs[8]`**）；本节只写各类型在 **`argvs[9]`** 起的**专属参数**。若不需要默认值，**`argvs[8]`** 仍请传 **`''`** 占位，避免与专属参数错位。

- `S`（文本）
  - `schemefieldLength`（可选）：字段长度；**`argvs[9]`**；不传默认 `255`
  - `isrepeat`（可选）：是否允许重复值；**`argvs[10]`**；`true` / `false`；不传默认 `true`
  - `placeholder`（可选）：输入框提示文案（`obj.placeholder`）；**`argvs[11]`**（与「空字符串占位」含义不同）
  - `casesensitive`（可选）：是否区分大小写；**`argvs[12]`**；仅当 `isrepeat=false` 时生效；空则默认 `false`
  - **空字符串占位**：**`argvs[9]`～`argvs[12]`** 顺序固定；若要向更后面的槽位传值，中间未用槽位须 **`''`**
- `U`（URL）
  - `edittype`（可选）：链接打开方式，`_blank` 或 `_self`；**`argvs[9]`**；不传默认 `_blank`；非 `_self` 按实现回退为 `_blank`
- `P`（百分比）
  - `schemefieldLength`（可选）：小数点**左侧**（整数部分）位数；**`argvs[9]`**；不传默认 `10`
  - `decimalPlaces`（可选）：小数点**右侧**位数；**`argvs[10]`**；不传默认 `2`
  - **约束**：两者须为非负整数，且**之和不能大于 18**
- `c`（币种）
  - 专属参数与 **`P`** 相同（**`argvs[9]`**、**`argvs[10]`**），约束相同
- `C`（累计汇总）
  - 大写 **`C`** 是累计汇总字段类型，不是币种；累计汇总通常由平台高级配置维护，不应和小写 **`c`** 混用
- `N`（数字）
  - `schemefieldLength`（可选）：小数点**左侧**（整数部分）位数；**`argvs[9]`**；不传默认 `10`
  - `decimalPlaces`（可选）：小数点**右侧**位数；**`argvs[10]`**；不传默认 `0`
  - `isrepeat`（可选）：是否允许重复值；**`argvs[11]`**；`true` / `false`；不传默认 `true`（允许重复）
  - `displayThousands`（可选）：平台字段 **`obj.displayThousands`**；**`argvs[12]`**；**`"1"`** 表示不允许重复，**`"0"`** 表示允许重复；不传默认 **`"0"`**
  - **约束**：`schemefieldLength` 与 `decimalPlaces` 须为非负整数，且**之和不能大于 18**（与 `P`/`c` 精度规则一致）
- `IMG`（图片）
  - **`defaultValue`**（**`argvs[8]`**）：**可上传图片数量**（写入 `obj.defaultValue`）；须为 **1～100** 的整数；空则默认 **`3`**
  - **`formulaType`**（可选）：录入方式 **`url`** 或 **`input`**（顶层 `formulaType`）；**`argvs[9]`**；不传默认 **`input`**
  - **`watermarkstatus`**（可选）：是否支持水印拍照；**`argvs[10]`**；**`"0"`** 不支持，**`"1"`** 支持；不传默认 **`"0"`**
- `FL`（文件）
  - **`defaultValue`**（**`argvs[8]`**）：**可上传文件数量**（须为 **1～100** 的整数；写入 **`obj.defaultValue`**，并与 **`obj.isrepeat`** 设为**同一数值**）；空则默认 **`1`**
- `ENC`（加密文本-存储加密）、`ENCD`（加密文本-显示加密）
  - 二者共用下列参数（**`argvs[9]`**～**`argvs[12]`**）：
  - **`schemefieldLength`**（可选）：文本最大长度，写入 **`obj.schemefieldLength`**；**`argvs[9]`**；须为 **1～255** 的整数；不传默认 **`255`**
  - **`masktype`**（可选）：掩码类型；**`argvs[10]`**；**`all`** 掩码全部；**`4`** 掩码后四位；**`card`** 卡号格式；**`custom`** 自定义；不传默认 **`all`**
  - **`encrypttype`**（可选）：**`masktype`=`custom`** 时生效，自定义掩码规则；**`argvs[11]`**；参考 **`"{AAAA}{****}{AAAA}"`**；空则使用默认示例串
  - **`maskcharacter`**（可选）：掩码字符；**`argvs[12]`**；**`*`** 或 **`X`**；不传默认 **`*`**
- `LT`（地理定位）
  - **`schemefieldLength`**（可选）：小数点**左侧**位数，写入 **`obj.schemefieldLength`**；**`argvs[9]`**；须为非负整数；不传默认 **`8`**（与默认 **`decimalPlaces`** 之和为 **18**，满足平台总位数上限）
  - **`decimalPlaces`**（可选）：小数点**右侧**位数，写入 **`obj.decimalPlaces`**；**`argvs[10]`**；须为非负整数；不传默认 **`10`**
  - **约束**：两者须为非负整数，且**之和不能大于 18**（与 **`P`** / **`c`** / **`N`** 精度规则一致）
  - **`displayType`**（可选）：纬度与经度显示方式；**`argvs[11]`**；**`"1"`** 数字展示，**`"2"`** 度、分、秒；不传默认 **`"1"`**（写入顶层 **`displayType`**）
- **仅公用 `helps` / `defaultValue`、无 `argvs[9]` 起专属参数的类型**：`D`、`F`、`T`、`H`、`A`、`AD` 等（**不含** **`IMG`** / **`FL`** / **`ENC`** / **`ENCD`** / **`LT`** 等已单列的类型；见上文各节）
- `J`（文本区-长）
  - 与 **`X`** 类似：多行文本；**`schemefieldLength`**（可选）为**最大字符数**；**`argvs[9]`**；须为 **1～32000** 的整数；不传默认 **`32000`**
  - **`placeholder`**（可选）：输入区**提示信息**；**`argvs[10]`**
  - **平台说明**：每个对象**最多 10 个** **`J`** 类型字段；条数由平台校验，**CLI 创建时不统计**，请在对象上自行控制数量
- `X`（文本区）
  - `schemefieldLength`（可选）：多行文本**最大字符数**；**`argvs[9]`**；须为 **1～4000** 的整数；不传默认 **`4000`**（与平台常见上限一致）
  - `placeholder`（可选）：输入区**提示信息**；**`argvs[10]`**
- `SCORE`（评分）
  - `schemefieldLength`（可选）：**最大评分**，决定**星星图标显示个数**；**`argvs[9]`**；须为 **1～100** 的整数；不传默认 **`10`**
- `E`（电子邮件）
  - `isrepeat`（可选）：是否允许重复；**`argvs[9]`**；**`"true"`** 允许重复，**`"false"`** 不允许重复；不传默认 **`"true"`**
- `B`（复选框）
  - **`defaultValue`**（**`argvs[8]`**）：**仅** **`"0"`** 或 **`"1"`** —— **`"0"`** 表示默认**未选中**，**`"1"`** 表示默认**选中**；空或其它值按 **`"0"`** 处理
- `L`（选项列表-单选）
  - `ptext`（**`useGlobalSelect`=`"0"` 时必填**）：选项内容；**`argvs[9]`**；**格式**：多个选项用 **`\r\n`** 连接（勿用仅 `\n` 作为分隔）。**`useGlobalSelect`=`"1"`**（全局列表）时可为空字符串
  - **`useGlobalSelect`**（可选）：是否使用全局选项列表；**`argvs[10]`**；**`"0"`** 不使用，**`"1"`** 使用；不传默认 **`"0"`**
  - **`edittype`**（可选）：选择样式，写入 **`obj.edittype`**；**`argvs[11]`**；**`radio`** 单选框列表，**`select`** 单选下拉；不传默认 **`select`**
  - **`isPicklistSorted`**（可选）：按字母顺序而非输入顺序排序；**`argvs[12]`**；**`"0"`** 否，**`"1"`** 是；不传默认 **`"0"`**
  - **`defPl`**（可选）：将第一个值作为默认值；**`argvs[13]`**；**`"0"`** 否，**`"1"`** 是；不传默认 **`"0"`**
  - **`globalSelectId`**：全局选项列表 id；**`argvs[14]`**。**`useGlobalSelect`=`"1"`** 时必填；为 **`"0"`** 时传 **`''`** 占位即可
- `Q`（选项列表-多选）
  - 与 **`L`** 相同的前缀参数：**`ptext`**、**`useGlobalSelect`**、**`edittype`**、**`isPicklistSorted`**、**`defPl`**、**`globalSelectId`**（**`argvs[9]`**～**`argvs[14]`**），含义与取值规则见上文 **「L（选项列表-单选）」**（**`useGlobalSelect`=`"0"`** 时 **`ptext`** 必填；**`"1"`** 时 **`globalSelectId`** 必填）
  - **`visibleLines`**（可选）：下拉最多展示几行选项，写入 **`obj.visibleLines`**；**`argvs[15]`**；须为 **1～100** 的整数；不传默认 **`4`**
  - **`showalloptions`**（可选）：是否显示所有选项，写入 **`obj.showalloptions`**；**`argvs[16]`**；**`"0"`** 否，**`"1"`** 是；不传默认 **`"0"`**
- `Y`（查找关系）
  - `lookupObj`（必填）：被关联对象的 `tp_sys_object.ID`，不能传对象 API 名、物理表名或显示名称；**`argvs[9]`**
  - **`lookupObjDefaultField`**（可选）：搜索辅助字段，写入 **`obj.lookupObjDefaultField`**；搜索时用于展示（如关联对象上用于显示的字段 API 名）；**`argvs[10]`**；不传或空字符串表示不指定
- `MR`（查找多选）
  - `lookupObj`（必填）：被关联对象的 `tp_sys_object.ID`；**`argvs[9]`**
- `M`（主详信息关系）
  - `lookupObj`（必填）：被关联对象的 `tp_sys_object.ID`；**`argvs[9]`**

### 5.2 更新字段

更新时必须在请求体中设置 **`obj.id`** 为平台上的字段 ID，且 **不会** 自动绑定页面布局（不写入 `layoutIds`）。

```bash
cloudcc update fields <projectPath> <fieldId> <fieldType> <objid> <fieldLabel> <remark> [extraArg]
```

- `fieldId`：平台字段 ID（`argvs[3]`），对应保存时的 `fieldData.obj.id`
- 其余参数与 `create fields` 含义相同，但整体下标相对 **create** 后移一位：`fieldType` 为 `argvs[4]`，`objid` 为 `argvs[5]`，`fieldLabel` 为 `argvs[6]`，`remark` 为 `argvs[7]`；**`helps`** 为 **`argvs[8]`**、**`defaultValue`** 为 **`argvs[9]`**；各类型专属参数在 **create** 中为 **`argvs[9]`** 起，在 **update** 中为 **`argvs[10]`** 起（相对 **create** 一律 **`+1`**）

### 5.3 支持的字段类型说明

根据 CloudCC 官方关于“对象-字段”的说明，平台层面支持文本、URL、百分比、币种、数字、文本区、长文本、富文本、电话、电子邮件、日期、日期/时间、评分、选项列表、图片、查找关系、主详信息关系、公式、自动编号、累计汇总、查找多选、复选框等类型。

当前 `src/fields/fields` 中已实现的 CLI 字段类型如下。

**说明**：字段类型编码大小写有语义差异，必须原样保留：小写 **`c`** 是币种，大写 **`C`** 是累计汇总。下列类型在模板 **`obj`** 中带有固定的 **`schemefieldLength`** 默认值（与平台「最大长度」类属性对齐；未通过 CLI 覆盖时使用）：**`U`** `2000`，**`D`** / **`T`** `20`，**`F`** `30`，**`B`** `10`，**`H`** `15`，**`E`** `254`，**`IMG`** `255`，**`AD`** `500`；**`S`**、**`N`**、**`P`**、**`c`**、**`X`**、**`J`**、**`SCORE`** 等仍由各自字段逻辑或上文「字段特殊入参」定义。

#### 基础输入类

| CLI 类型编码 | 类型名称 | 说明 |
| --- | --- | --- |
| `S` | 文本 | 单行文本，默认长度 255（可通过 `schemefieldLength` 调整） |
| `U` | URL | 输入有效网址，点击后可打开链接 |
| `P` | 百分比 | 输入百分比数字；可通过 `schemefieldLength` / `decimalPlaces` 配置整数位与小数位 |
| `c` | 币种 | 金额/币种场景；可通过 `schemefieldLength` / `decimalPlaces` 配置整数位与小数位（与 `P` 参数位一致） |
| `N` | 数字 | 输入数值，适用于数量、金额基数等 |
| `D` | 日期 | 仅日期 |
| `T` | 时间 | 仅时间 |
| `F` | 日期/时间 | 日期和时间组合 |
| `B` | 复选框 | 默认值仅 `"0"` / `"1"`（未选中 / 选中），见上文 **「B（复选框）」** 专属说明 |
| `H` | 电话 | 电话号码 |
| `E` | 电子邮件 | 邮箱地址；可选 `isrepeat`（`argvs[9]`），见上文 **「E（电子邮件）」** |
| `SCORE` | 评分 | 最大评分 1～100（`schemefieldLength`，星星个数），见 **「SCORE（评分）」** |

#### 文本区与内容类

| CLI 类型编码 | 类型名称 | 说明 |
| --- | --- | --- |
| `X` | 文本区 | 多行文本，最多 4000 字符（`schemefieldLength`），见 **「X（文本区）」** |
| `J` | 文本区（长） | 多行长文本，最多 32000 字符；每对象最多 10 个 `J` 字段（平台限制），见 **「J（文本区-长）」** |
| `A` | 文本区（富文本） | 支持富文本内容、图文描述 |
| `IMG` | 图片 | 上传数量见 **`defaultValue`**（`argvs[8]`），可选 **`formulaType`** / **`watermarkstatus`**（`argvs[9]` / `argvs[10]`），见上文 **「IMG（图片）」** |
| `FL` | 文件 | 可上传文件数量见 **`defaultValue`**（`argvs[8]`），与 **`isrepeat`** 同步为同一数值，见上文 **「FL（文件）」** |

#### 选择类

| CLI 类型编码 | 类型名称 | 说明 |
| --- | --- | --- |
| `L` | 选项列表 | 单选；**`ptext`** / **`useGlobalSelect`** / **`edittype`**（radio·select）等见上文 **「L（选项列表-单选）」** |
| `Q` | 选项列表（多选） | 与 **`L`** 类似的全局/排序等 + **`visibleLines`** / **`showalloptions`**，见 **「Q（选项列表-多选）」** |

**`L`**、**`Q`** 在 **`ptext`** 之后还可带可选参数：**`L`** 为 **`argvs[10..14]`**；**`Q`** 为 **`argvs[10..16]`**（在 **`L`** 的基础上多 **`visibleLines`**、**`showalloptions`**），详见上文 **「L」** / **「Q」** 小节。

```bash
cloudcc create fields <projectPath> L <objid> <fieldLabel> <remark> [helps] [defaultValue] <ptext> [useGlobalSelect] [edittype] [isPicklistSorted] [defPl] [globalSelectId]
cloudcc create fields <projectPath> Q <objid> <fieldLabel> <remark> [helps] [defaultValue] <ptext> [useGlobalSelect] [edittype] [isPicklistSorted] [defPl] [globalSelectId] [visibleLines] [showalloptions]
```

其中 **`helps`**、**`defaultValue`** 在 **`argvs[7]`**、**`argvs[8]`**，**`ptext`** 在 **`argvs[9]`**（若不需要帮助或默认值，可用 **`''`** 占位）。**`L` / `Q`** 在 **`useGlobalSelect`=`"0"`** 时 **`ptext`** 必填；**`useGlobalSelect`=`"1"`** 时 **`ptext`** 可为 **`''`**，但 **`globalSelectId`**（**`argvs[14]`**）必填。**`L` / `Q` 的 `ptext` 格式要求**（本地列表）：多个选项之间必须使用 **`\r\n`（CRLF）** 换行连接，逻辑上等价于字符串 `"a\r\nb\r\nc"`（一行一个选项）。在 Shell 中可用 `$'a\r\nb\r\nc'` 等形式传入真实回车换行序列。若要向 **`argvs[10]`**（**`L`/`Q`** 专属参数）及之后某槽位传值而保留更前面槽位的默认，中间未用的槽位须用 **`''`** 占位（与 **`S`** 等类型相同约定）。

平台层面的选项列表能力常用于：

- 状态
- 阶段
- 分类
- 来源
- 优先级

#### 关系类

| CLI 类型编码 | 类型名称 | 说明 |
| --- | --- | --- |
| `Y` | 查找关系 | 关联另一个对象；可选 **`lookupObjDefaultField`**（`argvs[10]`），见 **「Y（查找关系）」** |
| `MR` | 查找多选 | 多选关联对象 |
| `M` | 主详信息关系 | 主从/主详关系字段 |

这三类字段创建时都需要额外传入目标对象参数：

```bash
cloudcc create fields <projectPath> Y <objid> <fieldLabel> <remark> [helps] [defaultValue] <lookupObj> [lookupObjDefaultField]
cloudcc create fields <projectPath> MR <objid> <fieldLabel> <remark> [helps] [defaultValue] <lookupObj>
cloudcc create fields <projectPath> M <objid> <fieldLabel> <remark> [helps] [defaultValue] <lookupObj>
```

其中 **`helps`**、**`defaultValue`** 在 **`argvs[7]`**、**`argvs[8]`**，**`lookupObj`** 在 **`argvs[9]`**。`lookupObj` 表示被关联对象的 `tp_sys_object.ID`。如果调用方只有对象 API 名，应先通过对象查询/引用解析拿到 ID；直接传 API 名会导致运行时详情页查询无法把 `lookup#...` 占位符替换为真实查找 SQL。**仅 `Y`** 还可选 **`lookupObjDefaultField`**（**`argvs[10]`**；不需要时可传 **`''`**）。

这类字段常用于：

- 客户与联系人的关联
- 项目与客户的关联
- 主记录与明细记录的主详结构
- 多对象之间的一对多或多对多表达

#### 地址与定位类

| CLI 类型编码 | 类型名称 | 说明 |
| --- | --- | --- |
| `AD` | 地址 | 地址复合字段 |
| `LT` | 地理定位 | **`schemefieldLength`** / **`decimalPlaces`** / **`displayType`**（`argvs[9..11]`），见 **「LT」** |

#### 安全与加密类

| CLI 类型编码 | 类型名称 | 说明 |
| --- | --- | --- |
| `ENC` | 加密文本（存储加密） | 与 **`ENCD`** 相同的 **`argvs[9..12]`**（长度·掩码），见 **「ENC / ENCD」** |
| `ENCD` | 加密文本（显示加密） | 与 **`ENC`** 相同的 **`argvs[9..12]`**，见 **「ENC / ENCD」** |

说明：

- CLI 中使用的类型编码是 `ENC`、`ENCD`
- 模板内部提交到接口时，对应的 `fdtype` 分别是 `enc`、`encd`
- 字段类型大小写不可归一化：小写 `c` 是币种，大写 `C` 是累计汇总
- 这两类字段更适合身份证号、银行卡号、敏感联系方式等场景

### 5.4 当前 CLI 未覆盖但平台支持的字段能力

根据官方文档，平台层面还支持以下字段能力，但当前 `src/fields/fields` 目录下尚未看到对应 CLI 创建模板：

- 公式
- 自动编号
- 累计汇总（`C`，当前不作为普通币种字段创建）

如果后续希望通过 CLI 直接创建这些字段类型，需要继续补充对应的模板实现。

---

## 6. 删除字段

### 6.1 基本命令

```bash
cloudcc delete fields <projectPath> <fieldId> <objid>
```

删除前建议：

- 先确认字段未被页面、公式、触发器、类或脚本依赖
- 先记录字段 API 名称与 ID，防止误删
- 在测试环境验证后再在正式环境执行

---

## 7. 开发前检查

- 已完成 `cloudcc doc platform/project devguide` 中的环境准备
- 项目根目录存在可用的 `cloudcc-cli.config.js`
- 当前环境密钥配置正确
- 已确认对象 API 名称、字段类型和命名规则

---

## 8. 常见实践建议

- 优先先做对象建模，再补字段明细，避免频繁返工
- 字段命名规范化（显示名称清晰、API 名称可维护）
- 对核心字段（金额、状态、日期）提前统一格式约定
- 删除字段前先做依赖排查，确保不影响线上逻辑
