# CloudCC 字段开发指南

## 1. 模块定位

`fields` 模块用于管理 CloudCC 对象字段，当前提供字段列表查询、创建、更新、upsert、删除能力。创建、更新、upsert 字段推荐使用 MetadataService spec；旧版位置参数命令只作为基础字段的兼容快捷入口。

可通过以下命令读取文档：

```bash
cloudcc doc platform/fields introduction
cloudcc doc platform/fields devguide
```

---

## 2. 当前支持的命令

```bash
cloudcc get fields <projectPath> <object-id-apiName-or-prefix>
cloudcc plan msapi <projectPath> fields @field.json create
cloudcc plan msapi <projectPath> fields @field.json update
cloudcc plan msapi <projectPath> fields @field.json upsert
cloudcc plan msapi <projectPath> fields @field.json delete
cloudcc apply msapi <projectPath> <planId> ['{"async":true}']
cloudcc operation msapi <projectPath> <operationId>
```

说明：

- `projectPath`：本地项目路径，用于读取 `cloudcc-cli.config.json`。
- `get fields`：按对象 ID、对象 API 名或对象前缀查询对象字段，返回标准字段和自定义字段。
- `plan msapi ... fields`：生成字段元数据计划，不直接写库；复核 plan 后再执行 `apply`。
- `apply msapi`：执行已生成的计划。批量字段、包含布局/权限/相关列表等展开步骤的计划推荐使用 `{"async":true}`。
- `operation msapi`：查询异步 apply 的执行状态和验证结果。

MetadataService spec 是当前字段创建的主入口。调用方应在 JSON 中显式声明稳定的 `apiName`、`label`、`type` 和所属对象；不要依赖 CLI 自动生成字段 API 名。位置参数版 `cloudcc create fields ...` 属于兼容旧通道的基础字段快捷写法，见 [5. 兼容位置参数入口](#5-兼容位置参数入口)。

### 2.1 字段 ID 和选项 ID 规则

- 创建字段时通常不要传 `id`。字段物理 ID 由 MetadataService 按平台兼容格式生成。显式自定义字段 ID 必须匹配 `^ffe[A-Za-z0-9]{17}$`：以 `ffe` 开头、总长 20 位、只能包含 ASCII 字母和数字，不能带下划线。
- `apiName` 是调用方需要设计和传入的业务稳定标识；`id` 是平台物理元数据 ID。不要把 `apiName`、`f_ci_` + `apiName`、对象前缀、年份、随机串或示例值拼成字段 `id`。
- 只有更新、删除、精确迁移回放，或引用已经存在字段时，才使用字段 `id`；该值必须来自 `cloudcc get fields`、MetadataService scan/readback、创建回执或平台详情页回读。
- 本地选项列表 `options[]` 中也不要自造 `id` 或 `code`。普通创建只传 `value`、排序、默认值、本地化等业务字段；MetadataService 会生成或复用 `tp_sys_code` 行 ID。
- 对同一字段，本地选项的自然键是 `(codetype=字段ID, codevalue=选项值, LANG, RENDER)`。重复提交相同值、语言和 render 不会靠新 ID 变成合法新选项；目标库若已经存在多条相同自然键，需要先治理重复数据再重新 plan。

### 2.2 新增选项：先读、去重、提交、回读

适用于单选 `L`、多选 `Q` 的本地选项及其引用的全局选项列表。

1. **先读**：读取目标环境字段及完整已有选项；字段引用全局列表时，先确认真实 `globalSelectId` 并读取该列表的已有值，不能只看本地 spec 或凭名称猜测。
2. **去重和复用**：将待新增值与已有值、待新增值彼此比较；本地选项按上述自然键（字段、值、语言、render）匹配，全局列表按其列表范围和接口自然键匹配。已有选项直接复用真实 ID/编码，不创建同义替代项或自行生成 ID。全部已存在时无需写入。
3. **提交**：只新增缺失值，保留已有选项和引用。按接口语义构造请求：增量接口只传缺失值，全量接口提交保留已有值的去重合并结果；全局列表的 `optionsMode` 见 `cloudcc doc globalSelectList devguide`，不得为追加选项而使用破坏性替换。MSAPI 先审阅 plan，再 apply。
4. **回读核验**：提交完成后重新读取同一字段或列表，逐项确认新增值存在、自然键无重复、已有值及 ID/编码未丢失或改变。异步 apply 先等待完成再回读；回执成功不能代替回读。读取不完整或核验不一致时停止并排查，不宣告完成，也不盲目重提。

#### 审计和修复历史字段 ID

早期版本创建的自定义字段可能存在错误前缀、非 20 位长度、下划线或其他非法字符。先执行只读审计：

```bash
cloudcc audit fields <projectPath> [object-id-apiName-or-prefix] [fieldId]
```

审计返回 `violations` 和逐表 `referenceCounts`，不会创建 operation 或修改租户数据。修复时先生成计划，再显式 apply：

```bash
cloudcc repair fields <projectPath> <oldFieldId> [newFieldId|auto] [objectId] [expectedApiName]
cloudcc apply msapi <projectPath> <planId>
```

修复会保留对象、API 名、字段类型和 `DATAFIELD_REF`，在一个 MetadataService 事务中迁移已知引用并刷新视图/缓存。每个租户应顺序执行，先选择一个低引用字段做 canary 并回读业务记录。数据库外的代码、导出配置和集成参数必须按 old/new 映射同步更新。禁止用脚本直接 `UPDATE tp_sys_*`。

### 2.2 MetadataService 物理槽位规则

- `dataFieldRef`（如 `str_field1`、`date_field1`）是对象内唯一的物理存储槽位，不是可随意复用的业务标识。
- MetadataService 创建或 upsert 字段时会查询 `tp_sys_schemetable`，保留已有字段自己的映射，并按字段类型选择首个可用空槽；例如已占用 `str_field1`、`str_field3` 时，新文本类字段使用 `str_field2`。
- 普通字段计划应省略 `dataFieldRef`，由服务端分配。只有迁移或精确回放等确有需要的场景才显式提供，并且必须保证同一对象内唯一。
- plan 阶段会拒绝现有映射冲突和同一计划内的重复槽位；apply 阶段会在事务内重新锁定并检查，防止陈旧或并发计划覆盖其他字段。
- 收到 `field_slot_conflict` 时，重新执行目标环境 scan 并生成新 plan；不要修改数据库或绕过 MetadataService 守卫。

### 2.3 MetadataService 完整字段元数据

`cloudcc plan msapi <projectPath> fields @field.json create` 会按平台元数据保存规则展开字段元数据；对象计划中的 `fields[]` 也使用同一条展开链路。自动编号、查找筛选、相关列表、公式、累计汇总、地址、地理定位、字段权限和布局落位等完整字段能力都应使用 MetadataService spec。

字段创建如果会进入页面布局，必须先按 `platform/pagelayout devguide` 的页面布局配置方法论判断落位。能读取对象布局详情时，优先在字段 spec 中显式提供 `layoutPlacements`，或在字段创建后通过 `pagelayout detail` / `pagelayout update` 调整 PC 和 mobile 布局。只有缺少布局上下文时才允许依赖 MetadataService 自动摆放，并在输出中标注为兜底。

单字段创建示例：

```json
{
  "objectApiName": "account",
  "apiName": "approval_status",
  "label": "审批状态",
  "type": "L",
  "remark": "记录客户审批状态",
  "options": [
    {"value": "待审批"},
    {"value": "已通过"},
    {"value": "已驳回"}
  ],
  "profileFieldJson": [
    {"profileId": "<otherProfileId>", "visible": "true", "readonly": "false"}
  ],
  "layoutPlacements": [
    {"layoutId": "<layoutId>", "sectionId": "<sectionId>", "rowIndex": 1, "colIndex": 1}
  ]
}
```

常用字段：

| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `objectId` / `objectApiName` / `object` / `prefix` | 是 | 字段所属对象。推荐传 `objectApiName` 或真实回读的 `objectId`；同传时必须指向同一对象。 |
| `apiName` / `apiname` | 创建和 upsert 推荐必填 | 字段 API 名。用于自然键匹配、公式引用、自动化脚本和后续治理，应由调用方显式定义。 |
| `label` / `fieldLabel` / `nameLabel` | 是 | 字段显示名称。 |
| `type` / `fieldType` / `fdtype` / `schemefieldType` | 是 | 字段类型编码，区分大小写；`c` 是币种，`C` 是累计汇总。 |
| `remark` / `description` | 否 | 字段业务说明；未传时保持空值。 |
| `helps` / `helpText` | 否 | 字段帮助说明。 |
| `length` / `schemefieldLength` | 否 | 长度或整数位数，含义随字段类型变化。 |
| `decimalPlaces` | 否 | 小数位数；`P`、`c`、`N`、`LT` 受 `length + decimalPlaces <= 18` 约束。 |
| `defaultValue` | 否 | 默认值；图片/文件字段中也用于上传数量。 |
| `dataFieldRef` | 通常不要传 | 物理存储槽位。普通创建由 MetadataService 分配；只有迁移或精确回放时才显式传。 |
| `options` / `ptext` | 选项字段需要 | 本地选项值。`options[]` 是推荐结构；`ptext` 是兼容换行字符串；普通创建不要在选项中自造 `id` / `code`。 |
| `globalSelectId` / `useGlobalSelect` | 全局选项字段需要 | 目标环境真实回读的全局选项列表 ID。 |
| `lookupObjectId` / `lookupObj` | 关系字段需要 | 被关联对象的真实 `tp_sys_object.ID`，不是对象 API 名或显示名称。 |
| `conditionVals` | 否 | 查找筛选或累计汇总筛选的条件行。 |
| `formulaText` / `formulaType` | 公式字段需要 | 用户级公式表达式和返回类型；不要手写 `executeExpression`。 |
| `showFormat` / `beginIndex` | 自动编号字段需要 | 自动编号显示格式和起始序号。 |
| `summarizedObj` / `childtype` / `aggregateField` | 累计汇总需要 | 子对象、汇总方式和被汇总字段。 |
| `profileFieldJson` | 否 | 字段级简档权限，必须使用真实回读的简档 ID。MetadataService `1.1.94+` 创建字段时始终补齐并校正系统管理员 `aaa000001` 为可见、非只读，因此调用方无需传管理员行。其他简档按输入保留。 |
| `layoutPlacements` | 否 | 字段落位到页面布局的显式配置。 |

#### 2.3.1 数字精度上限

MetadataService 与平台字段编辑页保持同一条字段精度规则：`P`（百分比）、`c`（币种）、`N`（数字）和 `LT`（地理定位）的 `schemefieldLength` / `length` 与 `decimalPlaces` 必须为非负整数，且 `length + decimalPlaces <= 18`。

该规则在 create、update、upsert 以及对象计划内嵌字段展开时都会执行。历史上通过旧版 CLI 或其他链路创建的字段如果不满足该规则，后续 upsert 字段或把字段放入布局时也会被 MetadataService 拦截；CLI 不会自动缩短字段长度或修改小数位数。收到 `invalid_field_precision` 时，请先按目标租户的字段定义治理流程修复该字段精度，再重新生成 plan。

#### 2.3.2 批量添加字段

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

批量 spec 也可以在根级提供 `profileFieldJson`、`profileFields` 或 `fieldPermissions` 作为公共权限。字段项自身存在非空权限集合时以字段项为准；字段项省略权限时继承根级集合。MetadataService `1.1.94+` 随后为每个新字段独立生成权限行 ID，并统一补齐系统管理员 `aaa000001` 的 `visible=1`、`readonly=0`，不会复用根级权限行 ID。`AD`/`LT` 复合字段的父字段和全部物理子字段也执行相同规则。普通 update 若省略权限集合则不改变现有权限；显式更新管理员权限时仍会强制校正可见与只读值。`encryptvisible` 是独立的加密字段权限，不因系统管理员身份自动开启。

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

从 MetadataService `1.1.92` 开始，关系字段（`Y` / `M` / `MR`）可在创建或更新字段时直接使用友好的 `lookupFilter`。它与独立的 `lookupFilter` Domain 共用同一套解析和校验逻辑：

```json
{
  "objectId": "<sourceObjectId>",
  "apiName": "archive_definition_id",
  "label": "档案定义",
  "type": "Y",
  "lookupObjectId": "obj_archive_definition",
  "lookupFilter": {
    "enabled": true,
    "logic": "1 AND 2",
    "conditions": [
      {"left":{"scope":"lookupObject","field":"status"},"operator":"equals","right":{"value":"Active"}},
      {"left":{"scope":"lookupObject","field":"owner_id"},"operator":"equals","right":{"scope":"sourceObject","field":"owner_id"}}
    ]
  }
}
```

省略 `lookupFilter` 会保留现有规则；`{"enabled":false}` 只停用规则并保留条件；`{"clear":true}` 会停用并删除条件；显式 `null` 会被拒绝。最多支持 10 条条件。旧版 `conditionVals` 仍兼容，但新脚本应使用友好结构。查询、候选字段和独立维护流程见 `cloudcc doc lookupFilter devguide`。

查找关系（`Y`）和主详关系（`M`）未显式传 `relatedLists` 时，MetadataService 会按 `lookupObjectId` 查询被关联对象的所有布局并创建相关列表；此时 `childrelationName` 必填，它会原样作为相关列表标签，不能依赖对象名称回退。`mainlayoutIds` 决定桌面/移动布局是否显示，默认列使用来源对象的名称字段，并同步中文、英文、日文标签。例如：

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

从 MetadataService `1.1.69` 开始，字段自动生成的反向相关列表按平台元数据规则保存：`relatedlist_type=custom`，新建相关列表、默认列和多语言行使用 `aee`、`afa`、`bcb` 前缀。再次计划历史字段时，服务端按布局、来源对象和关系字段自然键复用已有行并修正错误类型，不会通过放宽读取条件来掩盖不一致数据。

对象默认搜索布局相关的字段、按钮和 lookup-layout ID 必须保持在平台 `rel_id` 的 20 字符边界内。MetadataService 生成的默认 ID 使用短 ID；不要在 spec 中覆盖为超长 ID，否则旧版 `to_mlang` 查询可能报 `Data too long for column 'rel_id'`。

### 2.4 MetadataService 字段类型矩阵

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

公式字段创建时，调用方只传用户级公式定义：`formulaText` 使用字段 API 名或公式编辑器插入的字段引用，`formulaType` 表示公式返回类型。`executeExpression` 是平台内部执行表达式，由 MetadataService 根据目标环境元数据生成并写入；普通 CLI 调用不要提供这个参数，也不要把 API 名公式或物理列名手写进去。

从 MetadataService `1.1.91` 开始，公式字段计划在入库前同时校验公式返回类型、关系路径类型和末端字段类型。引用对象、字段或关系目标不存在，或者字段类型不在公开契约内时，plan 阶段都会失败并返回可定位的错误。候选字段和计划校验使用同一套规则，避免文档、交互选择与实际编译不一致。

```json
{
  "objectId": "<objectId>",
  "apiName": "net_amount",
  "label": "净额",
  "type": "Z",
  "formulaType": "N",
  "formulaText": "amount__c-discount__c",
  "formulaReferences": [
    {"relationId":"<amountFieldId>","relationType":"field"},
    {"relationId":"<discountFieldId>","relationType":"field"}
  ]
}
```

#### 公式字段表达式、运算符和函数

公式字段（`type: "Z"`）创建支持字段公式函数和目标数据库函数。CLI 写入前应使用目标环境的字段公式校验能力确认表达式可编译；保存后的实际计算还会受目标环境的数据类型、数据库版本、数据库函数和字段引用解析约束，因此目标环境校验通过才是最终准入标准。当前文档以 MySQL 函数语法为例；如果目标环境使用其他数据库，应以该数据库及目标环境公式校验器实际支持的语法为准。

字段创建时三个公式相关字段要分工明确：

- `formulaText`：用户级公式表达式，使用字段 API 名或公式编辑器插入的字段引用，例如 `amount__c-discount__c`。
- `formulaType`：公式字段返回类型，只支持文本 `S`、币种 `c`、日期 `D`、日期时间 `F`、数字 `N`、百分比 `P`、URL `U`、选项列表 `L`。
- `executeExpression`：平台内部执行表达式。CLI 新建或修改公式字段时禁止传入；MetadataService 会根据 `formulaText`、`formulaType` 和目标对象字段元数据自动生成。历史字段仅查询或执行不修改公式定义的更新时，服务会保留已有内部表达式。
- `formulaReferences`：普通字段引用关系，写入 `tp_sys_field_reference`；不要把它当成跨对象公式依赖。
- `formulaRelevances` / `relevances`：跨对象公式依赖，写入 `tp_sys_relevance`。MetadataService 会从 `formulaText` 中的跨对象引用自动派生；只有需要补充已有历史依赖或特殊依赖时才显式提供，每项至少包含被引用对象 `relobjid` 和被引用字段 `relfieldid`。同对象公式通常不需要提供。

字段引用写法：

- 同对象字段使用 API 名，例如 `amount__c`。
- 关联对象字段使用关系 API 名加目标字段 API 名，例如 `account__r.amount__c`。
- 当前用户字段可使用 `$User.<field>`；如果目标环境没有对应用户字段，plan 会返回 `formula_user_field_not_found`。

#### 公式字段可引用类型与跨对象路径

公式是否能选择某字段，取决于字段在公式中的角色，而不是对象上是否存在该字段。

| 角色 | 支持的类型编码 | 规则 |
| --- | --- | --- |
| 公式返回类型 | `S c D F N P U L` | 新建或修改公式时，`formulaType` 只能取这些值。 |
| 末端操作数字段 | `S U P c N D F L Q B X E H V IMG SCORE MR` | 可以在同对象公式中引用，也可以作为跨对象关系路径的最后一段。 |
| 关系路径字段 | `Y M` | 只用于继续遍历关联对象，不能作为路径末端值。 |
| 不可作为新公式操作数 | `Z C J A FL enc encd AD LT T` | 包括公式、累计汇总、长文本/富文本、文件、加密、复合地址/定位及内部类型。未知类型也拒绝。 |

复合字段展开的子字段、已删除或已禁用字段、当前正在编辑的公式字段、没有可用物理映射的普通字段，也不会进入可选列表。已有 `Z/C` 字段可以正常查询；若更新时没有修改公式定义，服务会保留历史值，但新建公式或修改 `formulaText/formulaType` 时不能继续引用 `Z/C`。

先查询目标环境候选，再编写公式：

```bash
# 当前对象可作为末端操作数或关系入口的字段
cloudcc candidates fields <projectPath> <object>

# 沿关系路径查看目标对象候选；多级路径使用点号连接
cloudcc candidates fields <projectPath> <object> account__r.region__r

# 编辑已有公式时排除自身；排错时同时查看被排除字段及原因
cloudcc candidates fields <projectPath> <object> --current-field-id <fieldId> --include-unsupported
```

返回结果中的 `selectable=true` 表示可作为公式末端操作数，`traversable=true` 表示可继续进入关系对象；`supported=false` 的记录会给出 `reason`。默认不返回不支持项，只有 `--include-unsupported` 才返回。

跨对象公式只能沿已经存在的查找关系或主详关系遍历。中间关系字段把 `__c` 写成 `__r`，最后一段保留目标字段 API 名。例如：

```text
account__r.annual_revenue__c
account__r.region__r.region_code__c
```

关系路径的每一段必须是 `Y` 或 `M`，最后一段必须属于末端操作数类型。目标公式字段 `Z` 和累计汇总字段 `C` 即使可以在字段查询中看到，也不能插入新公式。不要手写 `executeExpression`、物理表名、物理列名或 `dataFieldRef` 来绕过类型校验。

常见失败原因：

| 错误码 | 说明 |
| --- | --- |
| `formula_object_not_found` | 公式所属对象或被引用对象不存在。 |
| `formula_field_not_found` | 公式引用了某对象上不存在的字段，错误信息会包含公式字段、缺失字段和对象。 |
| `formula_lookup_object_missing` | 公式使用了关系引用，但关系字段没有 lookup 目标对象。 |
| `formula_relationship_type_unsupported` | 关系路径中间段不是 `Y` 或 `M`，不能继续遍历。 |
| `formula_user_field_not_found` | `$User.<field>` 引用了当前环境不存在的用户字段。 |
| `formula_reference_type_unsupported` | 末端字段类型不在公开操作数类型集合内，例如 `Z`、`C`、`J`、`A`。 |
| `formula_result_type_unsupported` | `formulaType` 不属于 `S/c/D/F/N/P/U/L`。 |
| `formula_physical_mapping_missing` | 普通末端字段没有可用于公式编译的物理映射。 |
| `formula_self_reference_unsupported` | 公式直接或通过解析结果引用了自身。 |
| `formula_composite_child_unsupported` | 引用了地址、地理定位等复合字段的内部子字段。 |
| `formula_field_inactive` | 引用了已删除或已禁用字段。 |
| `formula_sql_unsafe` | 公式包含 SQL 语句、注释或多语句分隔符；公式入口只允许单个表达式。 |
| `formula_function_unsupported` | 目标数据库已知不支持这种函数用法，例如 MySQL 的 `decode(...)` 或数值掩码 `to_char(...)`。 |
| `formula_sql_invalid` | 编译后的表达式未通过目标数据库只读预检，常见原因是函数不存在、参数不匹配或 SQL 语法不属于当前数据库。 |

公式编辑器提供的运算符如下；是否能通过保存，以目标环境字段公式校验结果为准：

| 运算符 | 说明 |
| --- | --- |
| `+` | 加法；字符串场景需以目标环境校验结果为准。 |
| `-` | 减法。 |
| `*` | 乘法。 |
| `/` | 除法。 |
| `(` / `)` | 分组。 |
| `==` | 等于；字段公式保存时会按平台执行表达式转换为目标环境可识别的等值判断。 |
| `!=` | 不等于。 |
| `<` / `>` | 小于 / 大于。 |
| `<=` / `>=` | 小于等于 / 大于等于。 |
| `&&` | 逻辑与。 |
| `||` | 逻辑或。 |
| `&` | 公式编辑器展示为串联；不同目标环境可能按不同表达式引擎处理，必须以目标字段公式校验通过为准。 |
| `^` | 公式编辑器展示为乘方；不同目标环境可能按不同表达式引擎处理，必须以目标字段公式校验通过为准，不要默认等同于数据库 `POWER`。 |

公式字段没有独立的 JavaScript 函数解释器。平台会把字段引用编译成 SQL，并使用目标数据库执行只读限行查询做语法校验；其他表达式工具中的同名方法不属于字段公式创建链路。因此，函数是否支持由“平台转换 + 当前数据库函数/存储函数 + 当前账号执行权限”共同决定。

#### 平台直接转换的表达式

| 表达式 | 实际行为 |
| --- | --- |
| `HYPERLINK(url, friendly_name, target)` | 平台只提取第一个 URL 参数；MySQL/MariaDB 编译为 `concat(...)`，其他数据库使用 `||`。显示文本和 `target` 不进入执行表达式，不要把它理解成 HTML 超链接渲染函数。 |
| `DOMAIN` | 当前源码把它替换为字面量 `'DOMAIN'`，不是自动解析当前租户域名；需要真实域名时应显式传入经过配置管理的值。 |
| `==` | 保存时转换为 SQL 等号 `=`。 |

当前平台直接转换的函数只有 `HYPERLINK` 和 `DOMAIN`。`IMAGE(...)`、`VALUE(...)`、`TODAY()`、`ADDDAYS(...)` 等名称不会被平台公式编译器转换。

#### MySQL 原生表达式和函数

以下是 MySQL 目标常用写法，不是跨数据库抽象。参数、返回类型、SQL mode 和 `NULL` 行为必须以 MetadataService 计划预检及目标环境字段公式校验为准。

| 类型 | 可用写法示例 |
| --- | --- |
| 条件 | `CASE WHEN condition THEN value ELSE value END`、`IF(condition, true_value, false_value)`、`IFNULL(value, fallback)`、`COALESCE(...)` |
| 日期时间 | `CURDATE()`、`NOW()`、`DATE(...)`、`DATE_ADD(date, INTERVAL n DAY)`、`DATEDIFF(date1, date2)`、`YEAR(date)`、`MONTH(date)`、`DAY(date)` |
| 字符串 | `CONCAT(...)`、`CHAR_LENGTH(text)`、`LEFT(text, n)`、`RIGHT(text, n)`、`TRIM(text)`、`LOCATE(substr, text)` |
| 数值 | `ABS(x)`、`CEILING(x)`、`FLOOR(x)`、`MOD(x, y)`、`ROUND(x, n)`、`SQRT(x)`、`POWER(x, y)`、`LOG(x)`、`LN(x)`、`EXP(x)`、`LEAST(...)`、`GREATEST(...)` |
| 类型转换/格式化 | `CAST(value AS DECIMAL(...))`、`CAST(value AS CHAR)`、`FORMAT(number, decimals)` |
| 布尔/空值 | `AND`、`OR`、`NOT`、`IS NULL`、`IS NOT NULL`；选项值比较直接使用 `field__c = 'value'` |

MySQL 目标明确不接受 `decode(...)`，请改为 `CASE WHEN ... END`。数值格式化不要使用 Oracle 风格的 `to_char(number, '999...')`，请使用 `FORMAT(...)` 或显式 `CAST(...)`。

#### MySQL 中安装的 Oracle 兼容存储函数

CloudCC 的部分 MySQL 部署会安装下列兼容函数。它们属于“该数据库实例已安装时”的公式支持范围；MetadataService 不负责安装它们，也不把它们硬编码成所有环境必有能力。创建计划时会在当前目标数据库执行只读预检，函数不存在、签名不匹配或无执行权限都会返回 `formula_sql_invalid`。

| 函数 | 用途与边界 |
| --- | --- |
| `to_char(value, format)` | Oracle 风格日期/日期时间转文本；MySQL 数值掩码仍按上面的限制使用 `FORMAT`。 |
| `to_date(text, format)` | Oracle 风格文本转日期，具体格式串以目标库函数定义为准。 |
| `to_number(text)` | Oracle 风格文本转数字；可作为旧文档 `VALUE(text)` 的迁移替代，但必须先通过目标库预检。 |
| `trunc(value[, format])` | Oracle 风格截断；日期和数字重载/参数由目标库函数定义决定。 |

截图中显示的 `concat` 在 MySQL 本身也有同名原生函数，HYPERLINK 的 MySQL 编译会直接使用 `concat(...)`。

下列函数是 CloudCC 业务元数据辅助函数，不是通用 Oracle 兼容函数：

| 函数 | 说明 |
| --- | --- |
| `to_mcode(...)` | 平台代码值/显示值辅助转换；签名依部署版本而定。 |
| `to_mlang(...)` | 平台多语言标签读取；平台查询中常见 `to_mlang(relation_id, language)`。 |
| `to_mlang_type(...)` | 带类型的多语言辅助读取；签名依部署版本而定。 |

这三个函数在目标 MySQL 中存在且预检通过时可以被数据库调用，但不建议把它们当作普通业务公式的稳定跨版本 API；使用前应确认目标版本函数签名、返回类型和运行账号权限。

#### 旧函数名迁移建议

| 不应再直接承诺的写法 | MySQL 建议替代 |
| --- | --- |
| `VALUE(text)` | 已安装兼容函数时用 `to_number(text)`；否则用 `CAST(text AS DECIMAL(...))`。 |
| `TODAY()` | `CURDATE()` |
| `ADDDAYS(date, n)` | `DATE_ADD(date, INTERVAL n DAY)` |
| `DAYS(date1, date2)` | `DATEDIFF(date1, date2)` |
| `LEN(text)` | `CHAR_LENGTH(text)` |
| `AND(a, b)` / `OR(a, b)` | `a AND b` / `a OR b` |
| `ISPICKVAL(field, text)` | `field = text` |
| `ISNULL(value)` | `value IS NULL`；需要默认值时用 `IFNULL(value, fallback)`。 |
| `IMAGE(...)` | 无平台公式转换；使用图片字段/组件能力，不要作为普通公式函数。 |

数据库函数不是跨数据库的稳定抽象。Oracle、Kingbase 或其他目标必须使用各自语法并通过该目标库预检；不要仅因某个 MySQL 租户安装了兼容函数，就推断其他租户也支持。

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

创建前会执行平台字段约束：每对象最多 1 个主详字段、2 个自动编号、10 个文件字段、25 个图片字段；文本/加密长度、数字精度、上传数量、多选可见行数、重复选项和值域也会在 plan 阶段直接拒绝。从 MetadataService `1.1.88` 开始，MSAPI 存储字段容量以目标对象实际物理列为准，按数字序选择最小空槽，不再为 `J` 设置固定条数。字段删除会同步清理选项、筛选、相关列表、依赖、引用、自动编号、全局选项、语言、权限以及复合子字段元数据。

---

## 3. 查询字段详情（detail）

`cloudcc get fields` 是当前 MSAPI 字段读取主入口。旧 provider 下仍保留 `detail fields`，用于拉取与平台「字段编辑」页一致的数据（简档列表、对象列表、`fieldObj` 等）；在严格 MSAPI 模式下，如果需要字段详情，应先通过 `get fields` 按对象读取字段，再从返回内容中定位字段 ID、API 名、类型、权限和布局相关信息。

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
cloudcc get fields <projectPath> <object-id-apiName-or-prefix>
```
入参：
- `object-id-apiName-or-prefix`：对象 ID、对象 API 名或对象前缀。

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

## 5. 兼容位置参数入口

本节描述旧 provider/基础字段快捷写法。严格 MSAPI 模式下，低代码字段写入优先使用 [2.2 MetadataService 完整字段元数据](#22-metadataservice-完整字段元数据) 的 JSON spec；如果使用 `cloudcc create fields <projectPath> @field.json` 或 `cloudcc update fields <projectPath> @field.json`，CLI 会把 JSON 转成 MetadataService plan。不要用位置参数创建公式、自动编号、累计汇总、查找筛选、相关列表、地址/地理定位子字段、字段权限或布局落位等高级配置。

### 5.1 基础字段快捷命令

```bash
cloudcc create fields <projectPath> <fieldType> <objid> <fieldLabel> <remark> [extraArg]
```

#### 公用参数（所有字段类型通用）

- `projectPath`：本地项目路径，用于读取 `cloudcc-cli.config.json`
- `fieldType`：字段类型编码（例如 `S`、`U`、`L`、`Y` 等）
- `objid`：对象 ID
- `fieldLabel`：字段显示名称
- `remark`：字段业务功能描述（写入字段模板的 `obj.remark`；参数位置固定为 `argvs[6]`）
- `helps`：帮助说明，固定`argvs[7]`
- `defaultValue`：默认值，固定`argvs[8]`
- `extraArg`：仅部分字段类型需要的额外入参；其含义与参数位置随 `fieldType` 变化（通常从 `argvs[9]` 开始）

#### 兼容字段特殊入参（按 `fieldType`）

以下均假定 **`create fields`** 的 argv 下标（**`update fields`** 因多一个 `fieldId`，`remark` 及之后的参数整体 **`+1`**，见 [5.2](#52-兼容更新字段)）。

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
  - 大写 **`C`** 是累计汇总字段类型，不是币种；累计汇总应使用 MetadataService spec，不应和小写 **`c`** 混用
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
  - 与 **`X`** 类似：多行文本；MetadataService spec 的 **`schemefieldLength`**（可选）为**最大字符数**，须为 **1～131072** 的整数；不传仍默认 **`32000`**
  - **Provider 边界**：上述 131072 上限要求 MetadataService **`1.1.86`** 或更高版本；其他字段创建入口可能仍使用 32000 的长度限制
  - **`placeholder`**（可选）：输入区**提示信息**；**`argvs[10]`**
  - **槽位容量**：MetadataService **`1.1.88`** 起，MSAPI 不限制 `J` 的固定数量；`J` 与 `A` 共享目标对象实际存在的 **`clob_fieldN`** 槽位，并选择数字序最小的空槽。真实槽位全部占用时返回 **`field_slot_exhausted`**；其他字段创建入口可能使用不同容量规则
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

### 5.2 兼容更新字段

旧 provider 更新时必须在请求体中设置 **`obj.id`** 为平台上的字段 ID，且 **不会** 自动绑定页面布局（不写入 `layoutIds`）。当前 MSAPI 写入请使用 `cloudcc plan msapi <projectPath> fields @field.json update` 或 `upsert`。

```bash
cloudcc update fields <projectPath> <fieldId> <fieldType> <objid> <fieldLabel> <remark> [extraArg]
```

- `fieldId`：平台字段 ID（`argvs[3]`），对应保存时的 `fieldData.obj.id`
- 其余参数与 `create fields` 含义相同，但整体下标相对 **create** 后移一位：`fieldType` 为 `argvs[4]`，`objid` 为 `argvs[5]`，`fieldLabel` 为 `argvs[6]`，`remark` 为 `argvs[7]`；**`helps`** 为 **`argvs[8]`**、**`defaultValue`** 为 **`argvs[9]`**；各类型专属参数在 **create** 中为 **`argvs[9]`** 起，在 **update** 中为 **`argvs[10]`** 起（相对 **create** 一律 **`+1`**）

### 5.3 兼容基础字段类型说明

根据 CloudCC 官方关于“对象-字段”的说明，平台层面支持文本、URL、百分比、币种、数字、文本区、长文本、富文本、电话、电子邮件、日期、日期/时间、评分、选项列表、图片、查找关系、主详信息关系、公式、自动编号、累计汇总、查找多选、复选框等类型。

位置参数快捷入口只适合下列基础字段类型；完整字段编码和高级元数据能力以 [2.4 MetadataService 字段类型矩阵](#24-metadataservice-字段类型矩阵) 为准。

**说明**：字段类型编码大小写有语义差异，必须原样保留：小写 **`c`** 是币种，大写 **`C`** 是累计汇总。下列类型在兼容请求的 **`obj`** 中带有固定的 **`schemefieldLength`** 默认值（与平台「最大长度」类属性对齐；未通过 CLI 覆盖时使用）：**`U`** `2000`，**`D`** / **`T`** `20`，**`F`** `30`，**`B`** `10`，**`H`** `15`，**`E`** `254`，**`IMG`** `255`，**`AD`** `500`；**`S`**、**`N`**、**`P`**、**`c`**、**`X`**、**`J`**、**`SCORE`** 等仍由各自字段逻辑或上文「兼容字段特殊入参」定义。

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
| `J` | 文本区（长） | MetadataService spec 显式长度最多 131072 字符、缺省仍为 32000；MSAPI `1.1.88+` 与 `A` 共享实际 `clob_fieldN` 槽位且无固定 `J` 条数限制；Lightning UIAPI 可能仍受平台长度和数量限制，见 **「J（文本区-长）」** |
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
- 最终保存到平台时，对应的 `fdtype` 分别是 `enc`、`encd`
- 字段类型大小写不可归一化：小写 `c` 是币种，大写 `C` 是累计汇总
- 这两类字段更适合身份证号、银行卡号、敏感联系方式等场景

### 5.4 高级字段使用 MetadataService spec

公式（`Z`）、自动编号（`V`）、累计汇总（`C`）、查找筛选、相关列表、复合地址（`AD`）和地理定位（`LT`）都属于当前 CLI 可通过 MetadataService spec 创建的能力。调用方不要按位置参数猜测这些字段的展开参数，应直接编写 JSON spec 并执行：

```bash
cloudcc plan msapi <projectPath> fields @field.json create
cloudcc apply msapi <projectPath> <planId> '{"async":true}'
```

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
- 项目根目录存在可用的 `cloudcc-cli.config.json`
- 当前环境密钥配置正确
- 已确认对象 API 名称、字段 API 名称、字段类型和命名规则

---

## 8. 常见实践建议

- 优先先做对象建模，再补字段明细，避免频繁返工
- 字段命名规范化（显示名称清晰、API 名称可维护）
- 对核心字段（金额、状态、日期）提前统一格式约定
- 删除字段前先做依赖排查，确保不影响线上逻辑
