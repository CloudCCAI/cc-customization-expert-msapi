# CloudCC 自定义对象开发指南

## 1. 模块定位

`object` 模块用于管理 CloudCC 自定义对象。当前 CLI 快捷命令已收口到 MetadataService：查询走 MetadataService scan，创建、删除和物理删除只生成 MetadataService plan，真正执行必须显式 `apply msapi <planId>`。

可通过以下命令读取文档：

```bash
cloudcc doc platform/object introduction
cloudcc doc platform/object devguide
```

---

## 2. 当前支持的命令

```bash
cloudcc get object <projectPath> [type]
cloudcc create object <projectPath> <label> <业务功能描述>
cloudcc create object <projectPath> <label> <nameLabel> <业务功能描述>
cloudcc create object <projectPath> <label> <nameLabel> <businessDescription> --accessable <0|1|2>
cloudcc delete object <projectPath> <objid>
cloudcc purge object <projectPath> <objid>
```

说明：

- `projectPath`：本地项目路径，用于读取 `cloudcc-cli.config.js`
- `label`：对象中文名称或展示名称
- `nameLabel`：对象 API 名称；仅在上一种四参数形式中传入；不传（三参数形式）时基于 `label` 自动生成
- `业务功能描述`（**必填**，放在最后）：拼接到默认 remark「用于管理与「label」相关的业务数据」之后，整体写入 `obj.remark`
- `objid`：对象 ID
- `create/delete/purge object`：快捷命令只创建 MetadataService plan；如需执行，继续使用 `cloudcc apply msapi <projectPath> <planId>`

---

## 3. 查询对象

### 3.1 查询全部对象

```bash
cloudcc get object <projectPath>
```

返回 MetadataService `standard-catalog` 扫描结果，其中包含标准对象、自定义对象、字段和能力信号。

### 3.2 按类型查询

```bash
cloudcc get object <projectPath> standard
cloudcc get object <projectPath> custom
cloudcc get object <projectPath> deleted
```

常见含义：

- `standard`：标准对象
- `custom`：自定义对象
- `deleted`：保留兼容参数；CLI 不再调用 setup-svc 回收站接口，而是返回 MetadataService 标准目录扫描结果。需要精确确认删除状态时，以 `standard-catalog` 中对象状态或专用 compare 结果为准。

---

## 4. 创建对象

### 4.1 基本命令

```bash
cloudcc create object <projectPath> <label> "记录客户拜访与跟进计划"
cloudcc create object <projectPath> <label> <nameLabel> "主数据：客户主档"
cloudcc create object <projectPath> <label> <nameLabel> "合同管理" --accessable 0
```

### 4.2 `--accessable` 默认访问权限

`--accessable` 用于在创建对象时设置 CloudCC 对象默认访问权限，会写入对象创建 spec/body 的 `accessable` 字段。

取值只支持以下三种：

| 值 | 含义 | 适用场景 |
| --- | --- | --- |
| `0` | 专用 | 只有记录所有者、上级角色或共享规则授权用户可访问 |
| `1` | 公用只读 | 组织内用户默认可读，不默认可编辑 |
| `2` | 公用读/写 | 组织内用户默认可读写 |

不传 `--accessable` 时，CLI 不显式写入该字段，继续使用 CloudCC/MetadataService 当前默认行为。普通对象创建不支持 `3=从父`；从父访问权限应由主详关系创建流程设置，不应通过普通对象创建参数传入。

示例：

```bash
cloudcc create object ./project 合同 contract "合同管理" --accessable 0
cloudcc create object ./project 知识库 knowledge "知识库对象" --accessable 1
cloudcc create object ./project 公告 notice "公告对象" --accessable 2
```

### 4.3 批量创建对象

一次要创建多个对象时，使用 `plan msapi ... objects @文件 create`，文件顶层写 `objects` 数组。每个数组项就是一个普通对象创建 spec。

示例 `objects-batch.json`：

```json
{
  "objects": [
    {
      "id": "obj_contract",
      "label": "合同",
      "apiName": "contract",
      "nameLabel": "合同编号",
      "description": "用于管理合同主数据",
      "accessable": 0
    },
    {
      "id": "obj_invoice",
      "label": "发票",
      "apiName": "invoice",
      "nameLabel": "发票编号",
      "description": "用于管理发票主数据",
      "accessable": 1
    }
  ]
}
```

执行命令：

```bash
cloudcc plan msapi <projectPath> objects @objects-batch.json create
cloudcc apply msapi <projectPath> <planId>
cloudcc get object <projectPath> custom
```

批量创建时建议不要在 spec 中手工填写 `prefix`、`objPrefix` 或 `datatableName`。MetadataService 会在 `apply` 阶段按租户级锁为每个对象分配 CloudCC 兼容的唯一 `PREFIX` 和 `DATATABLE_NAME`，同一批内不会复用同一个前缀或物理表。

批量计划会先拦截明显冲突：

- 同一批内对象 API 名重复会失败。
- 同一批内显式 `datatableName` 重复会失败。
- 同一批内显式 `prefix`/`objPrefix` 重复会失败。

`apply` 完成后，以 `get object` 或对象详情回读的真实 `id`、`prefix`、`datatableName` 为准。调用方如果把对象拆成多批执行，也应在每批 apply 后回读结果，不要用 plan 前的临时推断值作为最终前缀。

### 4.4 创建过程

创建对象时，快捷命令会：

1. 将命令参数转换为 `objects` domain spec。
2. 调用 MetadataService `/metadata/v1/plans` 创建 `create` plan。
3. 返回 `planId`、`operationId`、风险、步骤和告警。
4. 不直接写库；执行必须由 `cloudcc apply msapi <projectPath> <planId>` 完成。

---

## 5. 删除对象

### 5.1 逻辑删除

```bash
cloudcc delete object <projectPath> <objid>
```

`delete object` 创建 MetadataService `objects delete` 计划，不再调用 setup-svc `/api/customObject/deleteLogic`。执行该计划后，对象会按 MetadataService 对象删除编译器的逻辑进入已删除状态。

### 5.2 查询已逻辑删除对象

```bash
cloudcc get object <projectPath> deleted
```

该命令用于读取 MetadataService 标准目录扫描结果。物理删除前仍必须确认对象 ID、名称、是否自定义对象、是否已逻辑删除以及依赖关系。

### 5.3 物理删除

```bash
cloudcc purge object <projectPath> <objid>
```

`purge object` 创建 MetadataService `objects physical-purge` 计划，不再调用 setup-svc `/api/customObject/deletePhysics`。该快捷命令只产出计划，不执行物理删除。

### 5.4 MetadataService 计划化物理删除

生产实施或项目清理使用 MetadataService 的计划/执行/证据链：

```bash
cloudcc plan msapi <projectPath> objects '{"id":"<objid>"}' physical-purge
cloudcc apply msapi <projectPath> <planId> '{"approval":"CLOUDCC_OBJECT_PHYSICAL_DELETE_APPROVED"}'
cloudcc changes msapi <projectPath> <operationId>
cloudcc rollback-plan msapi <projectPath> <operationId>
```

`physical-purge` 由 MetadataService 原生执行，保留 setup-svc `CustomObjectDeleteService.delObject` 的清理语义，并增加 preflight、逐表影响行数、孤儿已删除对象清理、执行后验证和 `changes` 证据。该操作会记录 `custom-object-physical-purge` side-effect 证据，并明确 `rollbackSupported=false`。

删除前建议：

- 先确认对象未被页面、菜单、字段或自动化逻辑依赖
- 先保留对象 ID 与对象名称，避免误删
- 先执行 `cloudcc get object <projectPath> deleted` 或 `cloudcc scan msapi <projectPath> standard-catalog` 保存对象目录证据
- 物理删除不可通过 `deleteLogicCancel` 恢复，只应在确认不再需要该自定义对象时执行

---

## 6. 开发前检查

- 已完成 `cloudcc doc platform/project devguide` 中的环境准备
- 项目根目录存在可用的 `cloudcc-cli.config.js`
- 当前环境密钥配置正确
- 已确认对象名称、API 名称、角色权限需求

---

## 7. 占位说明

这份文档当前为初版开发指南，后续可继续补充：

- 对象建模规范
- 命名建议
- 角色权限说明
- 与字段、菜单、应用的联动关系
