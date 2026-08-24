# CloudCC 全局选项列表操作指南

---

## 1. 入口与列表页

进入路径：`设置 → 开发者空间 → 全局选项列表值集`

列表页展示：
- **标签**：全局选项列表的显示名称
- **描述**：可选的说明文字

右上角操作按钮：
- **新建**：创建新的全局选项列表
- **字段依赖性**：查看引用关系
- **已删除全局值集**：查看已删除记录

---

## 2. 新建全局选项列表

点击「新建」弹出表单，填写以下字段：

| 字段             | 参数名            | 是否必填 | 说明                                    |
|------------------|-------------------|----------|-----------------------------------------|
| 标签             | label             | 是       | 显示名称，如"客户等级"                 |
| 名称             | name              | 是       | API 名称，如"khdj"                    |
| 描述             | description       | 否       | 对该选项列表用途的说明                  |
| 选项值列表       | ptext             | 否       | 每行一个值，行之间用 `\r\n` 分隔        |
| 按字母顺序排序   | isPicklistSorted  | 否       | `"1"` 表示启用，`"0"` 表示禁用          |
| 将第一个值作为默认值 | isFirstDefault | 否       | `"1"` 表示启用，`"0"` 表示禁用          |

### CLI 调用示例

参数使用 `encodeURI` 编码后传入（保留 `"` 和 `:` 等 JSON 必要字符）：

```bash
cloudcc create globalSelectList . '%7B%22label%22:%22%E5%AE%A2%E6%88%B7%E7%AD%89%E7%BA%A7%22,%22name%22:%22khdj%22,%22description%22:%22%E5%AE%A2%E6%88%B7%E7%AD%89%E7%BA%A7%E5%88%86%E7%B1%BB%22,%22ptext%22:%22%E6%99%AE%E9%80%9A%5Cr%5Cn%E9%93%B6%E7%89%8C%5Cr%5Cn%E9%87%91%E7%89%8C%22,%22isPicklistSorted%22:%220%22,%22isFirstDefault%22:%220%22%7D'
```

原始 JSON（`encodeURI` 编码前）：

```json
{
  "label": "客户等级",
  "name": "khdj",
  "description": "客户等级分类",
  "ptext": "普通\r\n银牌\r\n金牌",
  "isPicklistSorted": "0",
  "isFirstDefault": "0"
}
```

> **注意**：参数请使用 `encodeURI` 编码，而非 `encodeURIComponent`。`encodeURI` 不会转义 `"` 和 `:` 等 JSON 基础字符，服务端可正确解析。

---

## 3. 批量添加全局选项列表

一次要创建多个全局选项列表时，使用 MetadataService plan/apply，文件顶层写 `globalSelectLists` 数组。每个数组项就是一个普通全局选项列表 spec。

示例 `global-select-lists-batch.json`：

```json
{
  "globalSelectLists": [
    {
      "id": "gsl_customer_level",
      "name": "customer_level",
      "label": "客户等级",
      "description": "客户等级共享值集",
      "options": ["普通", "银牌", "金牌"]
    },
    {
      "id": "gsl_business_line",
      "name": "business_line",
      "label": "业务线",
      "options": [
        {"value": "industry", "localizedValue": "工业"},
        {"value": "service", "localizedValue": "服务"}
      ]
    }
  ]
}
```

执行命令：

```bash
cloudcc plan msapi <projectPath> global-select-lists @global-select-lists-batch.json create
cloudcc apply msapi <projectPath> <planId> '{"async":true}'
cloudcc operation msapi <projectPath> <applyId>
cloudcc get globalSelectList <projectPath>
```

批量全局选项列表创建可能会展开主表、选项和本地化选项行，推荐使用异步 apply。异步 apply 会立即返回 `applyId`，当前实现中 `applyId` 与 `operationId` 相同；后续用 `cloudcc operation msapi <projectPath> <applyId>` 轮询，直到状态为 `VERIFIED` / `APPLIED` / `FAILED`。如果状态仍是 `APPLYING`，不要重新提交同一个全局选项列表批量创建 plan。

批量计划会先拦截明显冲突，并按单个列表返回预检结果：

- 同一批内 `id` / `globalSelectId` 重复时，冲突列表标记为 `FAILED_PRECHECK`。
- 同一批内 `name` / `apiName` 重复时，冲突列表标记为 `FAILED_PRECHECK`。
- 同一列表内选项值重复时，该列表标记为 `FAILED_PRECHECK`。
- 目标环境已有同 `id` 或同 `name` 的列表时，按 `onExisting` 策略处理。

调用方应读取 plan metadata 中的 `batchItemResults`、`batchExecutableCount`、`batchPrecheckFailedCount`。`batchItemResults[].status` 可能是 `PLANNED`、`SKIPPED` 或 `FAILED_PRECHECK`；`FAILED_PRECHECK` 会带 `error` / `message`，且不会生成 SQL 步骤。如果整批列表都只剩 `FAILED_PRECHECK`，`apply` 会直接把 operation 标记为 `FAILED`。

预检失败是计划阶段的逐项结果；真正进入 `apply` 的列表仍按现有阶段批量写库，并在事务内执行。如果数据库约束、运行时副作用或并发检查失败，当前 operation 仍可能整体 `FAILED`。

### 3.1 已存在列表处理策略

根字段 `onExisting` 控制目标环境已有同名或同 ID 列表时的行为：

| 策略 | 行为 |
|------|------|
| `createOnly` | 默认策略；目标已存在时该列表标记为 `FAILED_PRECHECK`，其它无关列表可继续生成步骤。 |
| `skipExisting` | 目标已存在时跳过该列表，plan metadata 会记录 skipped。 |
| `updateExisting` | 目标已存在时更新 `tp_sys_global_select` 主表；默认不改选项。 |
| `upsertByApiName` | 按 `name` / `apiName` 解析目标；`upsert` 操作默认使用该策略。 |

示例：

```json
{
  "onExisting": "skipExisting",
  "globalSelectLists": [
    {"name": "customer_level", "label": "客户等级"},
    {"name": "customer_status", "label": "客户状态", "options": ["正常", "冻结"]}
  ]
}
```

### 3.2 给已有列表批量添加选项

对已有列表处理选项时，必须显式声明 `optionsMode`。如果不声明，`updateExisting` 只更新列表主表，不会重建、追加或覆盖已有 `tp_sys_code` 选项链。

| 模式 | 行为 |
|------|------|
| `none` | 不处理选项，只处理列表主表。 |
| `append` | 只追加不存在的选项值；已存在值跳过。 |
| `merge` | 已存在值复用原选项 ID 更新标签、排序、启用状态等；不存在值新增。 |
| `replace` | 先删除该列表下所有 `tp_sys_code` 选项，再写入请求选项；必须 `allowDestructive=true`。 |

追加选项示例：

```json
{
  "onExisting": "updateExisting",
  "optionsMode": "append",
  "globalSelectLists": [
    {
      "name": "customer_level",
      "label": "客户等级",
      "options": ["钻石", "战略客户"]
    }
  ]
}
```

合并选项示例：

```json
{
  "onExisting": "updateExisting",
  "optionsMode": "merge",
  "globalSelectLists": [
    {
      "name": "customer_level",
      "label": "客户等级",
      "options": [
        {"value": "gold", "localizedValue": "金牌客户", "active": true},
        {"value": "platinum", "localizedValue": "铂金客户", "active": true}
      ]
    }
  ]
}
```

替换选项属于破坏性操作，创建 plan 时必须允许 destructive。只在确认目标列表的旧选项可以全部重建时使用：

```json
{
  "onExisting": "updateExisting",
  "optionsMode": "replace",
  "globalSelectLists": [
    {
      "name": "customer_level",
      "label": "客户等级",
      "options": ["普通", "银牌", "金牌"]
    }
  ]
}
```

批量全局选项列表能力要求 MetadataService `1.1.24` 或更高版本。旧版本只支持把 `globalSelectLists[]` 顺序展开为普通 upsert，不支持目标环境预检、`onExisting` 策略或 `optionsMode`。

---

## 4. 查询全局选项列表

```bash
# 获取全部列表（默认分页 pageSize=10000）
cloudcc get globalSelectList

# 指定项目路径
cloudcc get globalSelectList /path/to/project
```

返回结果为 `globalSelectList` 数组，每条记录包含：

```json
{
  "id": "20266882A3E97ACaV8PT",
  "label": "Test",
  "name": "ttt",
  "description": "",
  "datatype": "",
  "isDeleted": "0",
  "createdate": 1774454400000,
  "lastmodifydate": 1774454400000
}
```

---

## 5. 查看详情

```bash
cloudcc detail globalSelectList . <id>
```

- `<id>`：全局选项列表的 ID，可从列表接口返回值中获取

返回结构：

```json
{
  "globalSelect": {
    "id": "202652A011E5B4CAEpjg",
    "label": "test2",
    "name": "test2",
    "description": "描述",
    "isDeleted": "0",
    "createdate": 1774454400000,
    "lastmodifydate": 1774454400000
  },
  "enabledList": [
    {
      "id": "bba2026DF6280BEeSivr",
      "codevalue": "值1",
      "sortorder": "0",
      "isdefaultvalue": "1",
      "isactive": "1"
    }
  ],
  "disabledList": [],
  "useList": []
}
```

| 字段          | 说明                                   |
|---------------|----------------------------------------|
| `globalSelect`  | 全局选项列表基本信息                   |
| `enabledList`   | 启用中的选项值列表                     |
| `disabledList`  | 已禁用的选项值列表                     |
| `useList`       | 引用该值集的字段列表                   |

---

## 6. 删除全局选项列表

```bash
cloudcc delete globalSelectList . <id>
```

删除逻辑分两步自动处理：
1. **软删除**：先以 `deleteFromDisk=false` 请求，将记录标记为已删除（可在「已删除全局值集」中找回）
2. **彻底删除**：若软删除失败（记录已处于软删除状态），自动以 `deleteFromDisk=true` 重试，从磁盘彻底删除

> **注意**：删除前请确认该选项列表没有被字段引用，或已做好引用迁移，否则可能影响现有数据。

---

## 7. 查看文档

```bash
# 查看能力介绍
cloudcc doc platform/globalSelectList introduction

# 查看操作指南（本文）
cloudcc doc platform/globalSelectList devguide
```

