# CloudCC 区域

区域（Area）对应 setup-web 路由 `/settings/companyInformation/hierarchicalStructure`，用于维护公司区域层级结构。当前 CLI/MSAPI 能力只覆盖区域树、保存区域和删除区域。

## 能力边界

- 读：`GET /metadata/v1/areas/tree`、`GET /metadata/v1/areas/{id}`
- 写：通过 MetadataService `areas` domain 生成 plan，再显式 apply
- setup-service 对照：`/api/area/queryTree`、`/api/area/saveArea`、`/api/area/DeleteArea`
- 不包含：`forecastsHierarchy`、区域用户分配、预测配额、访问权限辅助页签

## 常用 CLI

```bash
cloudcc get area .
cloudcc detail area . area-east
cloudcc create area . "华东" area-root area-east
cloudcc delete area . area-east
cloudcc apply msapi . <planId>
```

区域保存会维护 `tp_sys_area` 闭包行和 `tp_sys_group` 的 `area`、`areaAndSub` 两类组。删除会先校验区域是否存在子区域闭包行，存在子区域时拒绝生成/执行删除计划。
