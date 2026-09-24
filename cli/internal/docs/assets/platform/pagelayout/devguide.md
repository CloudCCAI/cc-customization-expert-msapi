# CloudCC 页面布局 CLI 命令说明

## 支持的命令

| 操作 | 说明 |
|------|------|
| `get` | 查询页面布局列表 |
| `create` | 创建/复制页面布局 |
| `assign` | 将已有页面布局分配给一个或多个简档，可选指定记录类型 |
| `delete` | 删除页面布局 |
| `detail` | 查询页面布局详情（支持 PC / mobile） |
| `update` | 保存布局编辑结果 |
| `detail/update pagelayout mobile` | 查询/保存移动页面布局 |
| `detail/update pagelayout row` | 查询/保存行式布局 |
| `detail/update pagelayout hover` | 查询/保存悬停布局 |
| `get/detail/create/update/enable/disable/delete pagelayout dynamic` | 管理动态页面布局规则 |
| `create/update/delete pagelayout dynamic-main-condition` | 管理动态布局主条件 |
| `create/update/delete pagelayout dynamic-second-condition` | 管理动态布局二级条件 |
| `create/update/delete pagelayout dynamic-action` | 管理动态布局触发动作 |

## 页面布局详情能力矩阵

页面布局详情页包含五类布局能力。CLI 按 setup-web/setup-svc 的真实能力边界拆分，不把所有能力混在一个不透明的 `pagelayout update` 中。

| 页面布局详情能力 | CLI kind | setup-svc 入口 | 当前 CLI/MSAPI 支持 |
|------------------|----------|----------------|---------------------|
| PC 页面布局 | 默认或 `pc` | `/api/modifyLayoutLightning/queryLayout`、`/saveLayout`、`/saveButtonLayout`、`/saveRelatedList` | 支持列表、详情、创建/复制、删除、sections 更新和独立布局分配；按钮/相关列表可通过 JSON spec 或 setup-svc 兼容入口处理 |
| 移动页面布局 | `mobile` | `/api/modifyLayoutLightning/queryLayout`、`/saveLayout`，请求体 `type=mobile` | 支持详情和 sections 保存；保存时传 PC 根布局 ID，服务端定位其 mobile 子布局 |
| 行式布局 | `row` | `/api/modifyLayoutLightning/queryMultiLayout`、`/saveMultiLayout` | 支持详情和完整替换保存；必填字段排在可选字段前 |
| 悬停布局 | `hover` | `/api/modifyLayoutLightning/queryMiniLayout`、`/saveMiniLayout` | 支持详情和字段完整替换保存；悬停相关列表 JSON 可传给 setup-svc，MetadataService 表级写入需等待 live parity 完成 |
| 动态页面布局 | `dynamic` | `/api/dynamicPageLayout/*` | 支持规则列表、详情、新增、编辑、启停、删除，以及主条件、二级条件、动作的计划写入 |

命令中的 `row` 也接受 `line` / `lineLayout` / `multiLayout` 别名；`hover` 也接受 `mini` / `miniLayout`；`dynamic` 也接受 `dynamicLayout`。

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

`create pagelayout` 只创建或复制页面布局，不分配简档。使用 MetadataService `1.1.62+` 时，最简单的命令会自动读取对象字段、详情按钮和入向查找/主详关系并设计布局：

```bash
cloudcc create pagelayout . 20267D1465464C5OB6m5 "课程表2"
```

该命令不需要调用方传字段、按钮或相关列表，也不会暗中选择“第一个布局”复制。计划的 `contentMode=auto`，并返回 `autoDesigned=true`、字段/按钮/相关列表计数；确认 plan 后仍需显式执行 `apply`。

内容来源按以下规则确定：

| `contentMode` | 触发方式 | 行为 |
|---|---|---|
| `auto` | 没有 `sourceLayoutId`，且完全没有 `sections` / `buttons` / `layoutButtons` / `relatedLists` 或旧式 `nameFieldId` 等默认分区字段；也可显式填写 | 从对象元数据自动生成内容。显式 `contentMode=auto` 时，已经出现的数组优先，缺少的内容类别才自动补齐；显式空数组表示不要该类别。 |
| `explicit` | 提供任一内容数组且未写 `contentMode`，或显式填写 | 只使用调用方提交的布局内容，不自动补齐遗漏类别。 |
| `clone` | 提供 `sourceLayoutId`，或显式填写并同时提供源布局 | 完整复制指定源布局的部分、字段、按钮、相关列表及子布局；不会默认选择第一个布局。 |
| `blank` | 必须显式填写 | 创建真正的空白布局；不能同时提交内容数组或 `sourceLayoutId`。 |

自动设计规则：

- 基础短字段最先进入 `基本信息`，名称/编号、类型/状态、主关联、负责人优先，并在两列间交替平衡。
- 金额、地址、时间、状态等字段只有形成稳定主题时才建立对应业务部分；零散字段仍归入双列 `基本信息`。
- 长文本、富文本、说明和备注进入单列 `备注信息`。
- 创建人、修改人、所有人、记录类型等系统字段进入末尾双列 `系统信息`，并按只读处理。
- 详情按钮只从当前对象可见的 `detailBtn` 中选择，标准按钮优先，隐藏按钮和列表按钮不进入详情布局。
- 相关列表只从真实的入向查找/主详关系生成；显示列优先名称/编号、状态、金额/数量、日期、负责人，最多 7 列；列表按钮只从子对象可见 `listBtn` 中选择。

创建完成后，如需让简档使用该布局，再执行独立的 `assign pagelayout` 命令。创建成功不等于用户已经能看到布局；验收时分别检查布局内容和布局分配结果。

**参数说明：**

| 参数 | 必填 | 说明 |
|------|------|------|
| `projectPath` | 否 | 项目路径，`.` 表示当前目录 |
| `objId` | 是 | 对象 ID |
| `layoutName` | 是 | 新页面布局名称 |
| `sourceLayoutId` | 否 | 要复制的真实源布局 ID；只有传入时才进入 `clone` 模式，不传时默认自动设计 |
| `isCloneDynamic` | 否 | 是否复制动态布局规则，默认 `true` |

**示例：**

```bash
# 自动设计字段、按钮和相关列表
cloudcc create pagelayout . 20267D1465464C5OB6m5 "课程表2"

# 指定源布局 ID 进行复制
cloudcc create pagelayout . 20267D1465464C5OB6m5 "课程表2" add20261DA7347CZPAUz

# 不复制动态布局规则
cloudcc create pagelayout . 20267D1465464C5OB6m5 "课程表2" add20261DA7347CZPAUz false

```

自动模式也可以使用 MetadataService JSON，并在 plan 阶段查看设计结果：

```json
{
  "objectId": "20267D1465464C5OB6m5",
  "layoutName": "课程自动布局",
  "contentMode": "auto"
}
```

```bash
cloudcc plan msapi . layouts @layout-auto.json create
# 检查 contentMode、autoDesigned、sectionCount、autoDesignFieldCount、
# layoutButtonCount、relatedListCount 和 warnings 后再执行
cloudcc apply msapi . <planId>
cloudcc detail pagelayout . 20267D1465464C5OB6m5 <layoutId>
```

显式 `auto` 可按类别覆盖自动结果。例如保留自动字段和相关列表、但明确不要任何按钮：

```json
{
  "objectId": "20267D1465464C5OB6m5",
  "layoutName": "无按钮自动布局",
  "contentMode": "auto",
  "buttons": []
}
```

创建空白布局必须显式声明，不能依赖省略内容：

```json
{
  "objectId": "20267D1465464C5OB6m5",
  "layoutName": "课程空白布局",
  "contentMode": "blank"
}
```

完整手工设计使用 `contentMode=explicit` 和后文的完整页面布局 JSON。只要任一内容数组已经出现且没有显式写 `contentMode=auto`，系统就按 `explicit` 处理，不会猜测遗漏的按钮或相关列表。

复制现有布局时只需要指定源布局：

```json
{
  "id": "layout_course_sales",
  "objectId": "20267D1465464C5OB6m5",
  "layoutName": "课程销售布局",
  "contentMode": "clone",
  "sourceLayoutId": "add20261DA7347CZPAUz"
}
```

### 页面布局分配

已有页面布局需要补做、改配或增加记录类型分配时，使用独立的 `assign pagelayout`。该操作只写布局分配，不重写 sections、按钮或相关列表：

```bash
cloudcc assign pagelayout <projectPath> <objectId|apiName|prefix> <layoutId> --profile <profileId> [--record-type <recordTypeId>]
```

`--profile` 可重复传入。`--record-type` 不是必填；省略时分配到对象的主类型，传入时分配到该记录类型。

示例：

```bash
cloudcc assign pagelayout . 20267D1465464C5OB6m5 layout_course_sales --profile aaa000001 --record-type rt_course_domestic

# 分配到主类型
cloudcc assign pagelayout . 20267D1465464C5OB6m5 layout_course_main --profile aaa000001 --profile aaa000002
```

批量或复杂分配建议使用 MetadataService JSON：

```json
{
  "objectId": "20267D1465464C5OB6m5",
  "layoutId": "layout_course_sales",
  "assignments": [
    {
      "profileId": "aaa000001",
      "recordTypeId": "rt_course_domestic"
    },
    {
      "profileId": "aaa000002",
      "recordTypeId": "rt_course_domestic"
    }
  ]
}
```

执行：

```bash
cloudcc plan msapi . layouts @layout-assignments.json assign
cloudcc apply msapi . <planId>
```

同一“简档 + 对象 + 记录类型”是一个分配范围；再次分配时应把该范围的 `layoutId` 更新为目标布局，而不是为不同布局保留多条冲突分配。验收时执行 `detail pagelayout` 并检查 `content.assignments[]`，确认 `profileId`、`objectId`、`recordTypeId`、`layoutId` 分别对应目标简档、对象、记录类型和页面布局。不要只看页面布局是否创建成功。

## 完整页面布局 JSON

下面示例在一次 MetadataService 计划中创建布局字段、详情页按钮、相关列表、相关列表显示列和相关列表按钮：

```json
{
  "id": "layout_contract_sales",
  "objectId": "obj_contract",
  "layoutName": "合同销售布局",
  "contentMode": "explicit",
  "apiName": "contract_sales_layout",
  "sections": [
    {
      "id": "section_contract_basic",
      "name": "基本信息",
      "showDetailHeader": true,
      "showEditHeader": true,
      "columns": [
        [
          {"fieldId": "field_contract_name", "required": true, "readonly": false},
          {"fieldId": "field_contract_customer"}
        ],
        [
          {"fieldId": "field_contract_status"},
          {"fieldId": "field_contract_amount"}
        ]
      ]
    }
  ],
  "buttons": [
    {"buttonId": "button_contract_submit", "seq": 1},
    {"buttonId": "button_contract_clone", "seq": 2}
  ],
  "relatedLists": [
    {
      "id": "related_contract_payments",
      "name": "回款明细",
      "objectId": "obj_payment",
      "fieldId": "field_payment_contract",
      "seq": 1,
      "show": true,
      "orderField": "field_payment_date",
      "orderDir": "desc",
      "fields": [
        {"fieldId": "field_payment_name", "seq": 1},
        {"fieldId": "field_payment_status", "seq": 2},
        {"fieldId": "field_payment_amount", "seq": 3},
        {"fieldId": "field_payment_date", "seq": 4},
        {"fieldId": "ownerid", "seq": 5}
      ],
      "buttons": [
        {"buttonId": "button_payment_new", "seq": 1}
      ]
    }
  ]
}
```

执行：

```bash
cloudcc plan msapi . layouts @contract-sales-layout.json create
cloudcc apply msapi . <planId>
cloudcc detail pagelayout . obj_contract layout_contract_sales
```

### 完整 JSON 字段说明

| 节点 | 关键字段 | 说明 |
|------|----------|------|
| `sections[]` | `id/name/columns[][]` | 页面字段分组；`columns` 第一层是列，第二层是该列字段。 |
| `sections[].columns[][]` | `fieldId/required/readonly` | `fieldId` 必须来自目标对象字段回读；不要使用标签代替字段 ID。 |
| `buttons[]` | `buttonId/seq` | 挂载已经存在的详情页按钮；按钮定义本身先通过 `buttons` domain 创建。 |
| `relatedLists[]` | `name/objectId/fieldId/relatedListType/seq/show` | `objectId` 是子对象；普通业务列表的 `fieldId` 是子对象上指向当前父对象的查找/主详关系字段；系统列表按下文固定参数填写。 |
| `relatedLists[].fields[]` | `fieldId/seq/fieldStyle` | 相关列表显示列，字段来自子对象；首列应是可点击名称/编号字段。 |
| `relatedLists[].buttons[]` | `buttonId/seq` | 挂载已经存在且适用于该相关列表的按钮。 |

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
cloudcc detail pagelayout <projectPath> mobile <objId> <layoutId>
cloudcc detail pagelayout <projectPath> row <prefix> <layoutId>
cloudcc detail pagelayout <projectPath> hover <layoutId>
```

说明：

- `objId`：对象标识，例如客户对象为 `account`
- `layoutId`：布局 ID，从查询布局列表获取
- `type`：可选，`mobile` 表示移动端布局，默认查询 PC 布局

示例：

```bash
cloudcc detail pagelayout . account add100000001328m7xZh
cloudcc detail pagelayout . account add100000001328m7xZh mobile
cloudcc detail pagelayout . mobile account add100000001328m7xZh
cloudcc detail pagelayout . row account add100000001328m7xZh
cloudcc detail pagelayout . hover add100000001328m7xZh
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

- 从 `detail` 的 `content.sections` 复制现有结构，尽量只调整需要变更的分组、列或字段顺序。
- 保留每个 section 的 `sectionId`，否则 `update` 会拒绝提交。
- 不要手工保留运行时控制字段；CLI 保存前会移除 `sortOrder`、`categoriesAllowed`、`canChangeColumns`、`canDeleteSection`。
- 如果需要移动字段，优先在同一个 `columns` 二维数组内调整字段对象的位置，避免重写整份布局。

## 字段语义驱动的布局摆放方法论

创建或补齐自定义字段时，技能应先主动设计布局落位；MetadataService 的自动摆放只能作为兜底。只要能读取到对象布局详情，就在字段计划中显式给出 `layoutPlacements`，或通过 `pagelayout update` 同步调整 PC / mobile 布局。

对象创建 spec 内嵌 `fields[]` 时，MetadataService 会把业务字段落入本次计划生成的默认 PC/mobile 布局：短字段按 PC 双列平衡，长文本按 PC 整行，mobile 单列。字段单独创建时，仍会读取当前对象已有的 PC/mobile 布局并自动摆放；如果目标布局还不存在，应改用对象内完整编排或显式 `layoutPlacements`，不要把普通 warning 当成交付完成。

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

相关列表由两个层次组成：

1. 查找关系或主详关系字段定义“子记录如何关联父记录”。
2. 页面布局的 `relatedLists[]` 定义“在哪个父对象布局展示、展示哪些子对象字段、按什么顺序以及提供哪些按钮”。

不要只提交相关列表配置而没有真实关系字段。以“合同—回款明细”为例，`field_payment_contract` 必须是回款对象上指向合同对象的查找或主详字段；相关列表中的 `fields[]` 也必须是回款对象字段。

只有当用户在父记录详情页需要浏览、创建或追踪子记录时，才挂载相关列表。以下情况不应挂载：只是存在技术关系但没有父记录内操作场景、与已有列表重复、子对象数据量过大且页面列表无法提供有效筛选或摘要、仅供技术集成使用且没有业务可读价值。

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

#### 平台系统相关列表

CLI/MSAPI JSON 只使用本节列出的参数名，不需要、也不应填写数据库表名或数据库列名。标准写法统一使用 `objectId`、`fieldId`、`relatedListType`、`show` 和 `seq`；不要把内部存储名称混入 spec。

平台系统相关列表不是普通“子对象 + 关系字段”列表，必须使用平台约定的固定标识：

| 系统相关列表 | `objectId` | `fieldId` | `relatedListType` | 默认 `show` |
|---|---|---|---|---|
| 未处理活动 | `activity` | `none` | `openActivities` | `true` |
| 活动历史 | `activity` | `none` | `activityHistory` | `true` |
| 审批历史 | `fff000abe` | `none` | `approvalHistory` | `false` |
| 备注和附件 | `attachement` | `none` | `attachement` | `false` |
| 字段跟踪 | `track` | `none` | `track` | `false` |
| 邮件 | `emailobject` | `none` | `emailobject` | `false` |

对于能够由 `objectId` 唯一识别的审批历史、备注和附件、字段跟踪、邮件，省略 `fieldId`、`relatedListType` 和 `show` 时，MetadataService 会按上表补齐平台默认值。`activity` 同时对应“未处理活动”和“活动历史”，因此必须显式传 `relatedListType=openActivities` 或 `relatedListType=activityHistory`。如果系统列表的 `objectId`、`fieldId`、`relatedListType` 相互冲突，计划阶段会直接报错，不会把它当成普通 `object` 类型入库。为了让配置意图清晰，完整配置或跨环境迁移时仍建议显式填写上表参数。

例如，把“审批历史”加入布局并保持平台默认隐藏：

```json
{
  "contentMode": "explicit",
  "relatedLists": [
    {
      "name": "审批历史",
      "objectId": "fff000abe",
      "fieldId": "none",
      "relatedListType": "approvalHistory",
      "show": false,
      "seq": 1
    }
  ]
}
```

需要直接显示时才把 `show` 改为 `true`。审批历史由平台渲染，不需要配置普通业务相关列表使用的 `fields[]` 或 `buttons[]`。`attachement` 是平台约定的固定拼写，调用时不要自行改成 `attachment`。

#### 页面布局详情回读字段

```bash
cloudcc detail pagelayout . <objectId|apiName|prefix> <layoutId|apiName|name>
```

详情响应中的 `content` 是面向 CLI/MSAPI 用户的规范化布局内容，也是更新前应读取和复用的部分。它与创建/更新 spec 使用同一套参数名：

| 回读路径 | 返回字段 |
|---|---|
| `content` | `id/objectId/layoutName/apiName/sections/buttons/relatedLists/assignments` |
| `content.sections[]` | `id/name/seq/showDetailHeader/showEditHeader/show/columns` |
| `content.sections[].columns[][]` | `id/fieldId/required/readonly/seq/rowIndex/colIndex` |
| `content.buttons[]` | `rowId/buttonId/seq` |
| `content.relatedLists[]` | `id/name/objectId/fieldId/relatedListType/show/seq/fields/buttons` |
| `content.relatedLists[].fields[]` | `rowId/fieldId/seq/fieldStyle` |
| `content.relatedLists[].buttons[]` | `rowId/buttonId/seq` |
| `content.assignments[]` | `id/profileId/objectId/recordTypeId/layoutId` |

审批历史的规范化回读示例：

```json
{
  "content": {
    "relatedLists": [
      {
        "id": "<真实相关列表ID>",
        "name": "审批历史",
        "objectId": "fff000abe",
        "fieldId": "none",
        "relatedListType": "approvalHistory",
        "show": false,
        "seq": 1,
        "fields": [],
        "buttons": []
      }
    ]
  }
}
```

`show` 回读为 JSON 布尔值，不是 `0/1` 字符串。更新布局时只从 `content` 复制 `sections`、`buttons`、`relatedLists` 等用户参数；响应中的其他诊断信息不是 CLI spec，不要复制到 plan 文件。

#### 创建关系字段时显式生成相关列表

创建查找关系（`Y`）或主详关系（`M`）字段时，可以在字段 spec 中显式为父对象的一个或多个布局生成相关列表：

```json
{
  "objectId": "obj_payment",
  "apiName": "contract_id",
  "label": "合同",
  "type": "Y",
  "lookupObjectId": "obj_contract",
  "childrelationName": "回款明细",
  "relatedLists": [
    {
      "id": "related_contract_payments",
      "layoutId": "layout_contract_sales",
      "name": "回款明细",
      "objectId": "obj_payment",
      "seq": 1,
      "show": true,
      "orderField": "field_payment_date",
      "orderDir": "desc",
      "fields": [
        {"fieldId": "field_payment_name", "seq": 1},
        {"fieldId": "field_payment_status", "seq": 2},
        {"fieldId": "field_payment_amount", "seq": 3},
        {"fieldId": "field_payment_date", "seq": 4},
        {"fieldId": "ownerid", "seq": 5}
      ]
    }
  ]
}
```

执行：

```bash
cloudcc plan msapi . fields @payment-contract-field.json create
cloudcc apply msapi . <planId>
```

字段 spec 中每个 `relatedLists[]` 项的 `layoutId` 是父对象布局 ID；根级 `objectId` 是子对象 ID；`lookupObjectId` 是父对象 ID。显式写 `"relatedLists": []` 表示不要自动生成相关列表。完全省略 `relatedLists` 时，MetadataService 会按 `lookupObjectId` 查找父对象布局并依据 `mainlayoutIds` 自动生成默认相关列表。

#### 在已有布局中调整相关列表

布局更新采用显式完整替换语义：

- 省略 `relatedLists`：保留现有相关列表。
- 提供非空 `relatedLists[]`：以提交数组完整替换该布局的相关列表、显示列和列表按钮。
- 提供 `"relatedLists": []`：清空该布局的全部相关列表。

因此更新前必须先执行 `detail pagelayout`，从 `content.sections` 回读完整分区，并把 `content.relatedLists` 中希望保留的相关列表全部放回 JSON：

```json
{
  "id": "layout_contract_sales",
  "sections": [
    {
      "sectionId": "section_contract_basic",
      "sectionName": "基本信息",
      "columns": [
        [{"fieldId": "field_contract_name"}],
        [{"fieldId": "field_contract_status"}]
      ]
    }
  ],
  "relatedLists": [
    {
      "id": "related_contract_payments",
      "name": "回款明细",
      "objectId": "obj_payment",
      "fieldId": "field_payment_contract",
      "seq": 1,
      "show": true,
      "fields": [
        "field_payment_name",
        "field_payment_status",
        "field_payment_amount",
        "field_payment_date",
        "ownerid"
      ],
      "buttons": ["button_payment_new"]
    }
  ]
}
```

```bash
cloudcc plan msapi . layouts @contract-layout-update.json update
cloudcc apply msapi . <planId>
cloudcc detail pagelayout . obj_contract layout_contract_sales
```

回读时至少核对 `content.relatedLists[]` 中的名称、`seq`、`objectId`、`fieldId`、`relatedListType`、`show`、显示列顺序和按钮顺序，并确认 PC/mobile 所属布局。不要只验证相关列表根配置存在。

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
cloudcc update pagelayout <projectPath> mobile <layoutId> <encodedLayoutJSON>
cloudcc update pagelayout <projectPath> row <layoutId> <requiredFieldIds> <optionalFieldIds>
cloudcc update pagelayout <projectPath> hover <layoutId> <fieldIds> [miniRelationlistJSON]
```

说明：

- `encodedLayoutJSON` 需要是 URL 编码后的 JSON，且必须包含 `sections` 字段（通常从 `detail` 返回的 `content.sections` 构造）
- CLI 提交前会清理每个 section 上的 `sortOrder`、`categoriesAllowed`、`canChangeColumns`、`canDeleteSection`
- 最终提交体使用 `{ "layoutId": "...", "layoutJson": "<string>" }` 并调用 `saveLayout`
- MetadataService JSON 更新同样要求完整 `sections[]`；`buttons` / `relatedLists` 省略时保留，显式数组时完整替换，空数组表示清空

示例（仅示意）：

```bash
cloudcc update pagelayout . add100000001328m7xZh '%7B%22sections%22%3A%5B%7B%22sectionId%22%3A%22adf201596491538bIl0N%22%2C%22sectionName%22%3A%22%E5%9F%BA%E6%9C%AC%E4%BF%A1%E6%81%AF%22%2C%22labelKey%22%3A%22%E5%9F%BA%E6%9C%AC%E4%BF%A1%E6%81%AF%22%2C%22showDetailHeader%22%3Atrue%2C%22showEditHeader%22%3Atrue%2C%22columns%22%3A%5B%5B%5D%5D%7D%5D%7D'
cloudcc update pagelayout . mobile add100000001328m7xZh '%7B%22sections%22%3A%5B%5D%7D'
cloudcc update pagelayout . row add100000001328m7xZh 'name,status' 'phone,email'
cloudcc update pagelayout . hover add100000001328m7xZh 'name,phone,email'
```

### 移动页面布局

移动页面布局保存沿用页面布局 sections 结构，但请求体带 `type=mobile`。命令中的 `<layoutId>` 传 PC 根布局 ID；setup-svc 会按 `parentid=<layoutId>` 定位移动端子布局。MetadataService 计划同样会先解析 mobile 子布局，再对该子布局执行 section/field replacement。

### 行式布局

行式布局是列表行/摘要行中显示的字段集合。保存时会以本次参数按顺序替换目标行式布局的既有字段集合：

| 参数 | 说明 |
|------|------|
| `requiredFieldIds` | 逗号分隔的必填字段 ID/API，保存后 `required=1`，顺序排在前面 |
| `optionalFieldIds` | 逗号分隔的可选字段 ID/API，保存后 `required=0`，顺序排在必填字段后 |

### 悬停布局

悬停布局是 lookup、引用字段或详情悬停卡片中展示的简要字段集合。保存时会以本次参数替换目标悬停布局的既有字段集合：

| 参数 | 说明 |
|------|------|
| `fieldIds` | 逗号分隔的悬停字段 ID/API，按传入顺序保存 |
| `miniRelationlistJSON` | 可选；setup-svc 兼容模式会原样传给 `/saveMiniLayout`，MetadataService 表级写入暂不自动展开相关列表 JSON |

## 动态页面布局

动态页面布局规则决定字段或分组在 PC/mobile 页面中的条件显示行为。规则本体、主条件、二级条件、触发动作是独立能力。

```bash
cloudcc get pagelayout <projectPath> dynamic <layoutId>
cloudcc detail pagelayout <projectPath> dynamic <dynamicLayoutId>
cloudcc create pagelayout <projectPath> dynamic <layoutId> <encodedRuleJSON>
cloudcc update pagelayout <projectPath> dynamic <encodedRuleJSON>
cloudcc enable pagelayout <projectPath> dynamic <dynamicLayoutId> [encodedRuleJSON]
cloudcc disable pagelayout <projectPath> dynamic <dynamicLayoutId> [encodedRuleJSON]
cloudcc delete pagelayout <projectPath> dynamic <dynamicLayoutId>
```

规则 JSON 常用字段：`id` / `dynamicLayoutId`、`layoutId`、`name`、`description` / `descreption`、`pcOrMobile` / `pc_or_mobile`、`isActive`、`mainCondition`、`mainConditions[]`、`secondConditions[]`、`actions[]`。

主条件：

```bash
cloudcc create pagelayout <projectPath> dynamic-main-condition <encodedConditionJSON>
cloudcc update pagelayout <projectPath> dynamic-main-condition <encodedConditionJSON>
cloudcc delete pagelayout <projectPath> dynamic-main-condition <conditionId>
```

主条件 JSON 常用字段：`id`、`dynamicId`、`fieldId`、`operator`、`value`、`seq`。

二级条件：

```bash
cloudcc create pagelayout <projectPath> dynamic-second-condition <encodedConditionJSON>
cloudcc update pagelayout <projectPath> dynamic-second-condition <encodedConditionJSON>
cloudcc delete pagelayout <projectPath> dynamic-second-condition <secondConditionId>
```

二级条件 JSON 常用字段：`id`、`mainConditionId`、`label`、`seq`、`fields[]`。`fields[]` 中每项使用 `fieldId`、`operator`、`value`、`BoolFilter`、`seq`，并作为该二级条件的下属条件项保存。

触发动作：

```bash
cloudcc create pagelayout <projectPath> dynamic-action <encodedActionJSON>
cloudcc update pagelayout <projectPath> dynamic-action <encodedActionJSON>
cloudcc delete pagelayout <projectPath> dynamic-action <actionId>
```

动作 JSON 常用字段：`id`、`mainConditionId`、`secondConditionId`、`type`、`fieldId`、`sectionId`、`seq`。`showsection` / `hidesection` 使用 `sectionId`，其它字段类动作使用 `fieldId`。
