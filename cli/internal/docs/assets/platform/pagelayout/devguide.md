# CloudCC 页面布局 CLI 命令说明

## 支持的命令

| 操作 | 说明 |
|------|------|
| `get` | 查询页面布局列表 |
| `create` | 创建/复制页面布局 |
| `delete` | 删除页面布局 |
| `detail` | 查询页面布局详情（支持 PC / mobile） |
| `update` | 保存布局编辑结果 |

## CLI 命令详解

### 查询页面布局列表

```bash
cloudcc get pagelayout <projectPath> <prefix>
```

**参数说明：**

| 参数 | 必填 | 说明 |
|------|------|------|
| `projectPath` | 否 | 项目路径，`.` 表示当前目录 |
| `prefix` | 是 | 对象前缀（如 001, b25） |

**示例：**

```bash
# 查询对象 b25 的页面布局列表
cloudcc get pagelayout . b25

# 查询客户对象（001）的页面布局列表
cloudcc get pagelayout . 001
```

### 创建页面布局

```bash
cloudcc create pagelayout <projectPath> <objId> <layoutName> [sourceLayoutId] [isCloneDynamic]
```

**参数说明：**

| 参数 | 必填 | 说明 |
|------|------|------|
| `projectPath` | 否 | 项目路径，`.` 表示当前目录 |
| `objId` | 是 | 对象 ID |
| `layoutName` | 是 | 新页面布局名称 |
| `sourceLayoutId` | 否 | 要复制的源布局 ID，不传则使用默认第一个 |
| `isCloneDynamic` | 否 | 是否复制动态布局规则，默认 `true` |

**示例：**

```bash
# 创建页面布局（自动使用默认布局作为模板）
cloudcc create pagelayout . 20267D1465464C5OB6m5 "课程表2"

# 指定源布局 ID 进行复制
cloudcc create pagelayout . 20267D1465464C5OB6m5 "课程表2" add20261DA7347CZPAUz

# 不复制动态布局规则
cloudcc create pagelayout . 20267D1465464C5OB6m5 "课程表2" add20261DA7347CZPAUz false
```

### 批量创建页面布局

一次要在同一个对象下创建多个页面布局时，使用 MetadataService plan/apply。文件顶层写 `objectId`、`objectApiName` 或 `objectPrefix` 指定目标对象，并在 `layouts[]` 中写每个布局。批量创建是对象级能力：同一个文件只能作用于一个对象，数组项不能覆盖到其它对象，也不能和其它 domain 混在同一次计划里提交。

示例 `layouts-batch.json`：

```json
{
  "objectId": "20267D1465464C5OB6m5",
  "onExisting": "createOnly",
  "layouts": [
    {
      "id": "layout_contract_default",
      "layoutName": "合同默认布局",
      "sections": [
        {
          "label": "基本信息",
          "fields": ["name", "ownerid"]
        }
      ]
    },
    {
      "targetLayoutId": "layout_contract_channel",
      "layoutName": "渠道合同布局",
      "sourceLayoutId": "layout_contract_default",
      "isCloneDynamic": "true"
    }
  ]
}
```

执行命令：

```bash
cloudcc plan msapi <projectPath> layouts @layouts-batch.json create
cloudcc apply msapi <projectPath> <planId> '{"async":true}'
cloudcc operation msapi <projectPath> <applyId>
cloudcc get pagelayout <projectPath> <prefix>
```

`pagelayout` / `layouts` 都可作为 `plan msapi` 的 domain 参数。批量计划会逐项检查同批重复、目标对象已有同 ID / API 名 / 名称布局、数组项是否声明了其它对象。复制布局时，`sourceLayoutId` / `cloneFromLayoutId` / 复制形态下的 `layoutId` 必须属于同一个根对象，跨对象源布局会标记为 `FAILED_PRECHECK`。

`onExisting` 支持：

| 策略 | 行为 |
|------|------|
| `createOnly` | 默认策略；目标已存在时该项标记为 `FAILED_PRECHECK`，其它无关项继续生成步骤。 |
| `skipExisting` | 目标已存在时跳过该项，plan metadata 记录为 `SKIPPED`。 |

调用方应读取 plan metadata 中的 `batchItemResults`、`batchExecutableCount`、`batchPrecheckFailedCount`。`batchItemResults[].status` 可能是 `PLANNED`、`SKIPPED` 或 `FAILED_PRECHECK`；预检失败项不会生成 SQL 步骤。如果整批都没有可执行项，`apply` 会失败，避免提交空计划。

### 删除页面布局

```bash
cloudcc delete pagelayout <projectPath> <layoutId>
```

**参数说明：**

| 参数 | 必填 | 说明 |
|------|------|------|
| `projectPath` | 否 | 项目路径，`.` 表示当前目录 |
| `layoutId` | 是 | 要删除的页面布局 ID |

**示例：**

```bash
# 删除指定页面布局
cloudcc delete pagelayout . add202610BD89F09XyGT
```

## 查询布局详情

```bash
cloudcc detail pagelayout <projectPath> <objId> <layoutId> [type]
```

说明：

- `objId`：对象标识，例如客户对象为 `account`
- `layoutId`：布局 ID，从查询布局列表获取
- `type`：可选，`mobile` 表示移动端布局，默认查询 PC 布局

示例：

```bash
cloudcc detail pagelayout . account add100000001328m7xZh
cloudcc detail pagelayout . account add100000001328m7xZh mobile
```

## 常用布局结构

`detail` 返回结果通常在 `data` 下包含布局基础信息和 `sections`。不同环境返回的字段会有差异，编辑布局时应以当前 `detail` 返回为准。

常见层级：

```text
data
└── sections[]
    ├── sectionId
    ├── sectionName
    ├── labelKey
    ├── showDetailHeader
    ├── showEditHeader
    └── columns[][]
```

### Section（分组）

一个 `section` 对应详情页上的一个字段分组，例如“基本信息”“联系人信息”。常用字段：

| 字段 | 说明 |
|------|------|
| `sectionId` / `sectionid` | 分组 ID，保存时必须保留 |
| `sectionName` | 分组名称 |
| `labelKey` | 分组显示文案 |
| `showDetailHeader` | 详情页是否显示分组标题 |
| `showEditHeader` | 编辑页是否显示分组标题 |
| `columns` | 字段列结构，通常是二维数组 |

### Columns（列）

`columns` 通常是二维数组：第一层表示列，第二层表示该列中的字段或组件项。双列布局常见结构如下：

```json
{
  "sections": [
    {
      "sectionId": "adf201596491538bIl0N",
      "sectionName": "基本信息",
      "labelKey": "基本信息",
      "showDetailHeader": true,
      "showEditHeader": true,
      "columns": [
        [
          { "fieldId": "name", "label": "名称" }
        ],
        [
          { "fieldId": "ownerid", "label": "所有人" }
        ]
      ]
    }
  ]
}
```

### 保存前建议

- 从 `detail` 的 `data.sections` 复制现有结构，尽量只调整需要变更的分组、列或字段顺序。
- 保留每个 section 的 `sectionId`，否则 `update` 会拒绝提交。
- 不要手工保留运行时控制字段；CLI 保存前会移除 `sortOrder`、`categoriesAllowed`、`canChangeColumns`、`canDeleteSection`。
- 如果需要移动字段，优先在同一个 `columns` 二维数组内调整字段对象的位置，避免重写整份布局。

## 字段语义驱动的布局摆放方法论

创建或补齐自定义字段时，技能应先主动设计布局落位；MetadataService 的自动摆放只能作为兜底。只要能读取到对象布局详情，就在字段计划中显式给出 `layoutPlacements`，或通过 `pagelayout update` 同步调整 PC / mobile 布局。

### 输入信息

- 先读取对象字段全集和全局对象字段字典，确认新增字段与已有字段的业务关系、是否复用标准字段、是否属于同一业务分组。
- 读取目标对象页面布局：`cloudcc get pagelayout . <prefix>` 后，对主要 PC 布局和 mobile 布局执行 `cloudcc detail pagelayout . <objId> <layoutId> [mobile]`。
- 结合字段 `label`、`apiName`、`type`、`remark`、帮助文本、必填/只读状态、引用对象、选项含义和对象场景判断位置。

### 分组选择

分组优先级：

1. 用户或字段计划明确指定的 `layoutId` / `sectionId`。
2. 与字段语义最匹配的业务分组。
3. 对象主信息分组，例如“基本信息”“详细信息”“业务信息”。

不要把普通业务字段放入系统信息、审计信息、历史信息等系统分组。常见语义映射：

| 字段语义 | 优先分组 |
|----------|----------|
| 金额、价格、成本、预算、收入、费用、折扣、税额 | 价格信息、财务信息、商务信息、报价信息 |
| 地址、地区、省市区、坐标、门店、仓库、位置 | 地址信息、区域信息、物流信息 |
| 客户、联系人、电话、邮箱、微信、负责人、供应商 | 客户信息、联系人信息、主体信息 |
| 日期、时间、周期、截止、开始、结束、交付、计划 | 时间信息、计划信息、进度信息 |
| 状态、阶段、类型、分类、等级、优先级、来源 | 状态信息、分类信息、流程信息 |
| 备注、描述、原因、说明、附件说明 | 备注信息、补充信息 |

如果已有分组同时命中多个语义，优先选择当前对象业务主流程更强的分组。例如订单对象中的“收货地址”优先放物流/地址分组，而不是客户基本信息；报价对象中的“预估成本”优先放报价/财务分组，而不是基本信息。

### 行列与顺序

- 双列布局中保持左右列大体均衡，优先放入较短的一列；不要因为新增字段导致某一列明显过长。
- 金额、状态、日期、负责人、客户等高频字段靠近同类字段，尽量放在用户阅读路径的上半区。
- 长文本、富文本、地址、图片、文件、JSON/复杂结构字段使用整行或单列位置，避免挤入窄列。
- 同一业务组内保持“主字段 -> 状态/日期 -> 金额/数量 -> 说明”的自然阅读顺序；明细解释字段靠近它解释的主字段。
- 移动端布局更强调高频查看和编辑，优先放核心字段，低频说明类字段靠后。

### 字段计划输出

当创建字段并且布局元数据可用时，字段计划应包含落位意图：

```json
{
  "apiName": "estimated_cost",
  "label": "预估成本",
  "type": "C",
  "layoutPlacements": [
    {
      "layoutId": "目标PC布局ID",
      "sectionId": "价格信息分组ID",
      "rowIndex": 2,
      "colIndex": 2
    },
    {
      "layoutId": "目标移动布局ID",
      "sectionId": "价格信息分组ID",
      "rowIndex": 2,
      "colIndex": 1
    }
  ]
}
```

如果当前接口只能先创建字段再更新布局，先完成字段创建，再用 `pagelayout detail` 的原始 `sections` 结构插入字段项并执行 `pagelayout update`。输出方案时说明字段为什么放在该分组、PC 与 mobile 是否一致，以及哪些布局没有足够信息需要后续人工确认。

## 页面布局配置方法论

本方法论用于 CloudCC PC 页面布局设计、字段落位、相关列表配置与智能体评审。移动端布局必须单独适配，不能直接照搬 PC 布局。历史统计、一次性扫描证据和本地分析路径不进入通用技能正文；智能体只使用下面的可复用规则。

### 布局命名

- 单一业务场景只有一个布局时，布局名默认使用对象中文标签。
- 多布局只在角色、场景、字段集合、只读性、相关列表或分配规则确有差异时创建。
- 多布局命名使用 `角色/场景 + 动作/阶段`，例如 `销售录入`、`财务审核`、`项目经理交付处理`。
- 禁止使用 `test`、`测试布局`、`SYSTEM`、`新布局` 等无法说明长期业务用途的名称。
- 新增布局前必须说明目标角色、与现有布局的差异、分配依据以及何时可废弃；任一答案缺失时，优先调整现有布局。

### 部分分组

页面默认结构为：

1. `基本信息`：记录身份、类型、状态、主关联和负责人。
2. 业务主题部分：按用户任务、生命周期、共同维护责任或业务主题分组。
3. `备注信息` 或 `详细描述`：长文本、说明、原因和补充材料。
4. `系统信息`：创建、修改、所有人、记录类型等平台或审计字段，始终位于最后。

分组选择优先级：

1. 复用同对象现有布局中语义相同且命名清楚的部分。
2. 按用户完成任务时的共同阅读、共同填写和共同变更关系分组。
3. 按业务实体或主题分组，例如客户、合同、金额、回款、地址、交付。
4. 按业务阶段分组，例如申请、评审、执行、确认；阶段部分只放该阶段真实使用的字段。
5. 无法形成稳定业务主题时才使用 `补充信息`，不得把它当作长期垃圾桶。

部分名称使用简短稳定的业务名词，推荐 2-12 个汉字。禁止空名称、`新建部分`、`信息`、`其他` 等无语义名称；禁止在部分名中放 HTML、颜色、字号、链接或长操作说明；禁止把普通业务字段放入 `系统信息`，也不要把审计字段散落到业务部分。

部分容量默认控制在 4-10 个短字段。11-20 个字段时应检查是否可按主题、阶段或填写角色拆分；超过 20 个字段必须说明无法拆分的业务理由；超过 30 个字段默认不通过评审，除非有显式批准和使用理由。空部分默认删除，除非动态渲染组件有明确用途。

### 字段排列

同一布局优先遵循以下阅读顺序：

1. 记录名称、编号或主题。
2. 记录类型、业务分类、阶段和状态。
3. 客户、联系人、合同、项目等主关联。
4. 负责人、协作人或责任组织。
5. 计划开始/结束、发生日期、截止日期等关键时间。
6. 数量、金额、币种、折扣、汇总指标。
7. 原因、描述、备注和补充材料。
8. 创建、修改等系统审计信息。

短文本、日期、数值、选择、查找关系字段默认使用双列；双列左右字段数差尽量不超过 1，整行字段例外。长文本、富文本、完整地址、图片、文件、JSON 或需要横向比较的字段使用单列或整行。

同行字段必须有阅读关系，优先配对开始/结束、计划/实际、金额/币种、数量/单价、类型/状态、阶段/完成度、省市区/详细地址、创建人及创建时间/修改人及修改时间。不得为填满页面强行配对无关字段，关键字段可以单独占据一行。

同一业务组内保持自然顺序：主字段 -> 状态/日期 -> 金额/数量 -> 说明。字段位置由业务语义和用户任务决定，不按字段创建时间排序。

`系统信息` 始终位于最后，优先按行展示创建人/创建时间与最后修改人/最后修改时间，再展示所有人、记录类型、币种等确需用户查看的平台字段。其余仅用于内部计算或集成的字段默认不显示，除非有明确排障或业务读数需求。

### 相关列表

只有当用户在父记录详情页需要浏览、创建或追踪子记录时，才挂载相关列表。以下情况不应挂载：只是暴露数据库关系但没有父记录内操作场景、与已有列表重复、子对象数据量过大且页面列表无法提供有效筛选或摘要、仅供技术集成使用且没有业务可读价值。

相关列表名称使用子记录集合的业务称谓，例如 `联系人`、`销售订单`、`回款明细`，不要使用对象 API 名或关系字段名。

相关列表顺序按当前页面核心任务排序：

1. 核心交易或明细列表，例如订单产品、合同行、回款明细。
2. 支撑业务关系，例如联系人、项目成员、关联项目。
3. 执行与协作记录，例如任务、未处理活动、活动历史。
4. 审批、字段跟踪等治理记录。
5. 文件、备注和附件。

如果审批进度是该对象的首要任务，可将批准历史前移。所有可见列表的 `seq` 应唯一、连续，并在调整后读回确认。

自定义业务相关列表默认展示 5-7 列；8-10 列仅用于确需横向判断的列表；超过 10 列必须拆减或说明理由。移动端必须重新选取核心列，不直接复制 PC 列集合。

相关列表字段顺序：

1. 可点击的名称、编号或主题，作为强制身份列。
2. 类型、状态、阶段或优先级。
3. 与父记录判断最相关的金额、数量或关键指标。
4. 关键日期。
5. 负责人或所有人。
6. 短备注，仅在列表中确有辨识价值时放在最后。

避免展示父记录回查字段、内部 ID、长文本、图片、公式明细、重复含义字段和对判断无帮助的审计字段。批准历史、字段跟踪等平台系统列表允许空字段配置；自定义业务列表没有显式字段配置时必须复核是否漏配。

### 智能体执行流程

智能体处理新增字段、布局调整或相关列表需求时必须执行：

1. 读取对象、全部现有 PC 布局、mobile 布局、布局分配场景、字段元数据和相关列表。
2. 确定目标角色、记录阶段，以及页面需要回答的三个主要业务问题。
3. 优先复用同对象的语义部分和相邻字段；禁止默认追加到底部。
4. 按“同任务共同变更”原则选择部分；没有合适部分时才创建语义明确的新部分。
5. 安排位置：身份优先、业务过程居中、说明靠后、系统信息最后；成对字段同行，长字段整行。
6. 配置相关列表：只挂载有父记录内使用场景的列表；身份列第一，自定义业务列表默认 5-7 列。
7. 执行静态校验：检查命名、重复、字段遗漏、列平衡、部分容量、列表序号和系统字段归属。
8. 走 MetadataService 变更链：真实修改必须 plan、人工或智能体审阅、apply 和读回验证，不得旁路写库。
9. 输出方案或计划时说明每个新增字段/列表的布局、部分、行列、排序和业务理由；缺少上下文时标注为待确认或兜底。

### 自动评审门禁

| 检查项 | 通过条件 | 严重度 |
|---|---|---|
| 布局用途 | 名称能映射到明确角色/场景，且无重复布局 | 必须 |
| 首屏身份 | 名称、编号或主题位于首个可见部分前部 | 必须 |
| 系统信息 | 位于最后，普通业务字段未被当作审计字段堆入 | 必须 |
| 部分名称 | 非空、无 HTML/URL、无临时词，表达稳定业务主题 | 必须 |
| 部分容量 | 默认不超过 10 个短字段；超过 20 有理由；超过 30 需显式批准 | 必须 |
| 双列平衡 | 左右字段数基本平衡，整行字段例外 | 建议 |
| 字段相邻关系 | 同行字段语义相关，阶段和时间顺序自然 | 必须 |
| 相关列表必要性 | 每个列表都有父记录内的用户任务 | 必须 |
| 相关列表首列 | 自定义业务列表以可点击身份字段为第一列 | 必须 |
| 自定义列表列数 | 默认 5-7，超过 10 有明确理由 | 必须 |
| 系统列表空字段 | 仅平台渲染类型可接受；自定义列表需复核 | 必须 |
| PC/mobile | 分别验证，不以 PC 成功替代移动端验收 | 必须 |
| 回读 | 序号、部分、字段位置和显示列与计划一致 | 必须 |

## 更新布局

```bash
cloudcc update pagelayout <projectPath> <layoutId> <encodedLayoutJSON>
```

说明：

- `encodedLayoutJSON` 需要是 URL 编码后的 JSON，且必须包含 `sections` 字段（通常从 `detail` 返回的 `data.sections` 构造）
- CLI 提交前会清理每个 section 上的 `sortOrder`、`categoriesAllowed`、`canChangeColumns`、`canDeleteSection`
- 最终提交体使用 `{ "layoutId": "...", "layoutJson": "<string>" }` 并调用 `saveLayout`

示例（仅示意）：

```bash
cloudcc update pagelayout . add100000001328m7xZh '%7B%22sections%22%3A%5B%7B%22sectionId%22%3A%22adf201596491538bIl0N%22%2C%22sectionName%22%3A%22%E5%9F%BA%E6%9C%AC%E4%BF%A1%E6%81%AF%22%2C%22labelKey%22%3A%22%E5%9F%BA%E6%9C%AC%E4%BF%A1%E6%81%AF%22%2C%22showDetailHeader%22%3Atrue%2C%22showEditHeader%22%3Atrue%2C%22columns%22%3A%5B%5B%5D%5D%7D%5D%7D'
```
