# CloudCC 视图开发指南

## 1. 当前支持命令

```bash
cloudcc get view <projectPath> [objectId|objectApiName|encodedJsonFilter]
cloudcc detail view <projectPath> <viewId>
cloudcc update view fieldSetupAllViews <projectPath> <objId> <fieldIds(逗号或JSON数组)> [allViewId]
cloudcc doc platform/view introduction
cloudcc doc platform/view devguide
```

## 2. 参数说明

- `projectPath`：项目路径（用于读取配置）
- `objectId|objectApiName`：对象 ID、对象 API 名或对象前缀，用于查询该对象下的所有对象视图；大小写不敏感
- `encodedJsonFilter`：可选 JSON 查询体，支持 `objectId`、`objId`、`object`、`objectApiName`、`apiName`、`keyword`、`name`、`label`、`page`、`pageSize`
- `viewId`：视图 ID，仅用于 `cloudcc detail view ...`
- `objId`：对象 ID，供 `fieldSetupAllViews` 更新全部视图字段时使用
- `fieldIds`：字段 ID 列表，支持：
  - 逗号分隔字符串（例如 `ffe2024D0B157CENv70t,ffe202593C0FF31GbnZM`）
  - JSON 数组（可先 `encodeURI`）
- `allViewId`：可选；不传时 CLI 会按对象查询视图列表，按 `label=全部` 自动解析视图 ID

## 3. 使用示例

```bash
# 查询 account 对象下所有对象视图
cloudcc get view . account

# JSON 查询体写法；适合同时传分页或关键字
cloudcc get view . '{"objectId":"account","pageSize":100}'

# 查询某个视图详情
cloudcc detail view . aec2026CC9BCAA1YbIgV

# 不传 viewId：自动查 label=全部 的视图
cloudcc update view fieldSetupAllViews . 2024CE595708124vAsZS "ffe2024D0B157CENv70t,ffe202593C0FF31GbnZM,ffe202543798E16dFu7g"

# 显式传入 viewId：按指定视图更新
cloudcc update view fieldSetupAllViews . 2024CE595708124vAsZS "%5B%22ffe2024D0B157CENv70t%22%2C%22ffe202593C0FF31GbnZM%22%5D" aec2026CC9BCAA1YbIgV
```

## 4. 语义约定

- `get view` / `getList view` 是列表查询语义；带一个对象参数时查询该对象下所有对象视图，不会把该参数当作视图 ID。
- `detail view` / `editInfo view` 是详情查询语义；必须传入视图 ID。
- 查询对象下所有对象视图推荐写法：`cloudcc get view . account`；如果需要分页、关键字或显式 JSON 查询体，可写 `cloudcc get view . '{"objectId":"account"}'`。
