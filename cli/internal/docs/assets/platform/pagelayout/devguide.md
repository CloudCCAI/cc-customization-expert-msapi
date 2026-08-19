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
