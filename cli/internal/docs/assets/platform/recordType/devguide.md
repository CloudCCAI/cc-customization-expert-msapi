# CloudCC 记录类型操作指南

---

## 1. 入口与列表页

进入路径：`对象管理 → 选择对象 → 记录类型` 标签页

列表页展示：
- **记录类型标签**：显示名称
- **描述**
- **是否启用**
- **最后修改人**

右上角操作按钮：**新建 / 排序 / 页面布局分配**

```bash
# 获取指定对象的记录类型列表
cloudcc getList recordType . <objid>
```

---

## 2. 新建前查询初始信息

新建前需先获取简档列表、现有记录类型列表、页面布局列表：

```bash
cloudcc newInfo recordType . <objid>
```

返回结构：

```json
{
  "existsRectypeList": [ { "id": "", "name": "--主--" } ],
  "profilemap": { "<profileId>": "简档名称" },
  "layouts": [ { "id": "<layoutId>", "name": "页面布局名称" } ],
  "profileRecordtypes": { "<profileId>": "" },
  "slist": []
}
```

---

## 3. 新建记录类型

```bash
cloudcc create recordType . <encodedBodyJson>
```

原始 JSON（`encodeURI` 编码前）：

```json
{
  "objid": "202646FC67ACF24D39sG",
  "name": "记录类型标签",
  "apiCode": "unique_api_name"
}
```

> 其余参数（简档列表、页面布局等）由系统自动调用 `newInfo` 接口获取并使用默认值：默认**不启用**（`isenable: ""`）、不继承现有记录类型、使用第一个页面布局、所有简档均启用且第一个简档设为默认。

| 字段       | 说明                                            |
|------------|-------------------------------------------------|
| `objid`    | 对象 ID                                         |
| `name`     | 记录类型标签（显示名称）                        |
| `apiCode`  | 唯一名称，创建后不可修改（对应 UI「唯一名称」） |

---

### 3.1 批量创建记录类型

一次要在同一个对象下创建多个记录类型时，使用 MetadataService plan/apply。文件顶层写 `objectId`、`objectApiName` 或 `objectPrefix` 指定目标对象，并在 `recordTypes[]` 中写每个记录类型。批量创建是对象级能力：同一个文件只能作用于一个对象，数组项不能覆盖到其它对象，也不能和其它 domain 混在同一次计划里提交。

示例 `record-types-batch.json`：

```json
{
  "objectPrefix": "b00",
  "onExisting": "createOnly",
  "recordTypes": [
    {
      "id": "rt_contract_standard",
      "name": "标准合同",
      "apiCode": "standard_contract",
      "isenable": "true"
    },
    {
      "id": "rt_contract_channel",
      "name": "渠道合同",
      "apiCode": "channel_contract",
      "description": "渠道业务专用记录类型"
    }
  ]
}
```

执行命令：

```bash
cloudcc plan msapi <projectPath> record-types @record-types-batch.json create
cloudcc apply msapi <projectPath> <planId> '{"async":true}'
cloudcc operation msapi <projectPath> <applyId>
cloudcc getList recordType <projectPath> <objid>
```

`recordType` / `record-types` 都可作为 `plan msapi` 的 domain 参数。批量计划会逐项检查同批重复、目标对象已有同 ID / API 名 / 名称记录类型、以及数组项是否声明了其它对象。`onExisting` 支持：

| 策略 | 行为 |
|------|------|
| `createOnly` | 默认策略；目标已存在时该项标记为 `FAILED_PRECHECK`，其它无关项继续生成步骤。 |
| `skipExisting` | 目标已存在时跳过该项，plan metadata 记录为 `SKIPPED`。 |

调用方应读取 plan metadata 中的 `batchItemResults`、`batchExecutableCount`、`batchPrecheckFailedCount`。`batchItemResults[].status` 可能是 `PLANNED`、`SKIPPED` 或 `FAILED_PRECHECK`；预检失败项不会生成 SQL 步骤。如果整批都没有可执行项，`apply` 会失败，避免提交空计划。

---

## 4. 编辑记录类型

**Step 1：获取回显数据**

```bash
cloudcc editInfo recordType . <encodedBodyJson>
# encodedBodyJson: { "id": "<recordTypeId>", "objid": "<objid>" }
```

**Step 2：修改后保存（只可编辑以下四个字段）**

| 可编辑字段    | 字段名        |
|--------------|---------------|
| 记录类型标签  | `name`        |
| 描述          | `description` |
| 唯一名称      | `apiCode`     |
| 启用          | `isenable`    |

```bash
cloudcc editSave recordType . <encodedBodyJson>
# encodedBodyJson: 将 editInfo 返回的 data 对象修改目标字段后提交
```

---

## 5. 删除记录类型

**Step 1：删除前校验**（检测启用状态 + 获取可替换记录类型列表）

```bash
cloudcc validDelete recordType . <encodedBodyJson>
# encodedBodyJson: { "id": "<recordTypeId>", "objid": "<objid>" }
```

> **规则**：调用此命令时会先检测记录类型是否处于启用状态：
> - 若当前为**启用**状态，命令将报错并提示先执行禁用操作，不会继续执行删除校验
> - 若当前为**禁用**状态，则正常返回可替换记录类型列表
>
> 请先调用 `cloudcc editSave` 将 `isenable` 设为 `"false"` 完成禁用，再重新执行此命令。

返回：

```json
{
  "obj": { "id": "2026667B898023FoKwCe", "name": "测试1" },
  "recordtypes": [
    { "id": "", "name": "--无--" },
    { "id": "2026A1C1DE314F24t6eZ", "name": "测试2" }
  ]
}
```

**Step 2：执行删除**

```bash
cloudcc delete recordType . <encodedBodyJson>
# encodedBodyJson: { "id": "<recordTypeId>", "objid": "<objid>", "replaceId": "<replaceId>" }
```

| 字段        | 说明                                        |
|-------------|---------------------------------------------|
| `id`        | 待删除记录类型 ID                           |
| `objid`     | 对象 ID                                     |
| `replaceId` | 替换记录类型 ID；不替换时传空字符串 `""`    |

> **注意**：删除后，原使用该记录类型的所有记录将替换为 `replaceId` 指定的记录类型。

---

## 6. 查看文档

```bash
cloudcc doc platform/recordType introduction   # 能力与适用场景说明
cloudcc doc platform/recordType devguide        # 本操作指南
```

---

## 7. API 参考

| 操作         | 接口                                  | 关键参数                       |
|--------------|---------------------------------------|--------------------------------|
| 获取列表     | `POST /api/recordType/getRecordTypeList` | `objid`                     |
| 新建初始信息 | `POST /api/recordType/newRecordType`  | `objid`                        |
| 新建保存     | `POST /api/recordType/saveRecordType` | `objid`, `obj`, `pslist`       |
| 编辑回显     | `POST /api/recordType/editRecordType` | `id`, `objid`                  |
| 编辑保存     | `POST /api/recordType/editSave`       | 完整记录类型对象               |
| 删除校验     | `POST /api/recordType/validDelete`    | `id`, `objid`                  |
| 删除         | `POST /api/recordType/deleteObj`      | `id`, `objid`, `replaceId`     |
