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

## 3. 查询全局选项列表

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

## 4. 查看详情

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

## 5. 删除全局选项列表

```bash
cloudcc delete globalSelectList . <id>
```

删除逻辑分两步自动处理：
1. **软删除**：先以 `deleteFromDisk=false` 请求，将记录标记为已删除（可在「已删除全局值集」中找回）
2. **彻底删除**：若软删除失败（记录已处于软删除状态），自动以 `deleteFromDisk=true` 重试，从磁盘彻底删除

> **注意**：删除前请确认该选项列表没有被字段引用，或已做好引用迁移，否则可能影响现有数据。

---

## 6. 查看文档

```bash
# 查看能力介绍
cloudcc doc platform/globalSelectList introduction

# 查看操作指南（本文）
cloudcc doc platform/globalSelectList devguide
```

