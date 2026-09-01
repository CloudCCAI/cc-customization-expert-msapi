# 区域 CLI 用户级与开发说明

区域属于低代码元数据能力。MSAPI/Universal 选择 MetadataService 时，`area`、`areas`、`hierarchicalStructure` 都映射到 canonical domain `areas`。

## 读取区域树

```bash
cloudcc get area <projectPath> [filter]
cloudcc detail area <projectPath> <area-id-or-name>
```

读取调用 MetadataService 只读接口：

- `cloudcc get area .` -> `GET /metadata/v1/areas/tree`
- `cloudcc get area . 华东` -> `GET /metadata/v1/areas/tree?filter=华东`
- `cloudcc detail area . area-east` -> `GET /metadata/v1/areas/area-east`

树接口按 setup-service `queryTree` 的闭包表口径返回根节点和直接父子边：`currentlevel=0` 或 `gap=1`。详情接口返回该区域的完整 `tp_sys_area` 闭包行、相关 `tp_sys_group`、以及这些组下的 `tp_sys_groupmember`。

## 创建区域计划

简写：

```bash
cloudcc create area <projectPath> <name> [parentAreaId] [areaId]
```

示例：

```bash
cloudcc create area . "华北" area-root area-north
cloudcc apply msapi . <planId>
```

JSON：

```bash
cloudcc plan msapi . areas '{"areaId":"area-north","name":"华北","parentId":"area-root","currentlevel":1}' create
cloudcc apply msapi . <planId>
```

支持字段：

| 字段 | 必填 | 说明 |
|------|------|------|
| `areaId` / `areaid` | 否 | 区域逻辑 ID；省略时由 MetadataService 生成 |
| `id` / `rowId` | 否 | `tp_sys_area` self row ID；通常省略 |
| `name` / `areaName` | 是 | 区域名称 |
| `parentId` / `parentid` | 否 | 直接上级区域 ID |
| `currentlevel` / `currentLevel` | 否 | 层级，默认 `0` |
| `apiName` / `apiname` | 否 | API 名称 |
| `description` | 否 | 描述 |
| `parentClosure[]` / `ancestors[]` / `closureRows[]` | 否 | 完整祖先闭包行；省略时只生成直接父级闭包行 |

创建计划会：

- 写入 `tp_sys_area` self row：`areaId = parentId`、`gap=0`。
- 根据 `parentClosure[]` 或 `parentId` 写入祖先闭包行。
- 写入 `tp_sys_group` 两类区域组：`type=area` 和 `type=areaAndSub`。
- plan metadata 标记 setup-service 来源 `/api/area/saveArea`。

## 更新区域计划

```bash
cloudcc update area <projectPath> '{"areaId":"area-east","name":"华东大区"}'
cloudcc apply msapi . <planId>
```

当前阶段更新覆盖区域名称，并同步更新相关区域组名称。区域移动父节点在 setup-service 中会重建子树闭包；如果需要迁移父节点，建议先读 `detail area`，准备完整闭包计划并人工复核。

## 删除区域计划

```bash
cloudcc delete area <projectPath> <area-id>
cloudcc apply msapi . <planId>
```

删除计划会按 setup-service `DeleteArea`/`removeArea` 语义处理：

- 删除该区域相关组下的 `tp_sys_groupmember`。
- 删除该区域的 `tp_sys_group`。
- 删除该区域全部 `tp_sys_area` 闭包行。

plan 和 apply 都会先检查目标区域是否存在，且 `parentId=<areaId>` 的闭包行数量不能大于 1；存在子区域时返回 `area_has_children`。

## setup-service 对照

| CLI/MetadataService | setup-service |
|---------------------|---------------|
| tree | `/api/area/queryTree` |
| create/update plan | `/api/area/saveArea` |
| delete plan | `/api/area/DeleteArea` |

`forecastsHierarchy`、用户分配、预测配额、访问权限页签是区域周边业务，不纳入当前 `areas` domain。
