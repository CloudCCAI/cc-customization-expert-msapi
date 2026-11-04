# CloudCC Lightning 仪表板开发指南

## 1. 命令与执行模型

```bash
cloudcc get dashboard <projectPath> [filter]
cloudcc getList dashboard <projectPath> '{"folderId":"folder-id","lightning":true}'
cloudcc detail dashboard <projectPath> <dashboardId>
cloudcc runtime dashboard <projectPath> '{"userId":"005...","roleId":"...","profileId":"...","folderId":"..."}'

cloudcc create dashboard <projectPath> @dashboard.json
cloudcc update dashboard <projectPath> <dashboardId> @dashboard-root.json
cloudcc save dashboard <projectPath> <dashboardId> @dashboard-root.json
cloudcc modify dashboard <projectPath> <dashboardId> @dashboard-root.json
cloudcc replace dashboard <projectPath> <dashboardId> @dashboard.json
cloudcc delete dashboard <projectPath> <dashboardId>

cloudcc apply msapi <projectPath> <planId>
cloudcc doc dashboard devguide
```

查询命令直接调用 MetadataService 只读接口。创建、更新和替换由 MetadataService 原生写入关系库；删除生成
平台委托计划。复核 plan 后必须显式执行 `apply msapi` 才会产生变更。

## 2. `update` 与 `replace` 的区别

- `update`、`save`、`modify`、`editSave`：只更新仪表板主记录的可编辑字段，不改组件和筛选器。省略 `lightning`、`width`、`height` 时不会写入默认值。
- `replace`、`upsert`：聚合写入。只有显式提供 `components`（或兼容别名 `reports`）时才替换组件集合。
- 聚合写入中，集合字段遵循三态：省略表示保留；空数组表示清空；非空数组表示完整替换。

因此，只改名称或描述应使用 `update`；需要增删组件或筛选器时使用 `replace`。

## 3. 创建/替换 JSON

```json
{
  "id": "dashboard-id",
  "name": "销售管理-经营分析",
  "description": "销售核心指标",
  "lightning": true,
  "width": 12,
  "height": 8,
  "folderId": "existing-folder-id",
  "folder": {
    "id": "new-folder-id",
    "name": "销售管理",
    "folderType": "lightningdashboard",
    "viewType": "1",
    "purview": "1",
    "accessibleUsers": []
  },
  "components": [
    {
      "id": "component-id",
      "name": "本月订单额",
      "reportId": "report-id",
      "dashboardType": "number",
      "x": 0,
      "y": 0,
      "width": 4,
      "height": 3,
      "recordNum": 10,
      "unit": "integral",
      "showMainPage": true,
      "showValue": true,
      "showPercent": false,
      "showTotal": true,
      "filters": [
        {
          "id": "filter-id",
          "objectId": "account",
          "fieldId": "status-field-id",
          "type": "e"
        }
      ]
    }
  ]
}
```

`folderId` 引用已有文件夹；`folder` 用于在同一计划中创建或更新文件夹。Lightning 仪表板的新文件夹会规范为 `folderType=lightningdashboard`。未传私有文件夹设置时，`viewType` 默认 `1`、`purview` 默认 `1`，所有者为当前执行人。

## 4. 参数说明

### 4.1 主记录

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `id` | string | 更新/替换/删除必填 | 仪表板 ID；创建可省略，由服务生成 |
| `name` / `label` | string | 创建必填 | 名称 |
| `description` | string | 否 | 描述 |
| `folderId` / `dashboardFolderId` | string | 建议 | 已有仪表板文件夹 ID |
| `lightning` | boolean/string | 否 | Lightning 标记；创建默认 `true`，更新省略时保留原值 |
| `width` / `height` | integer | 否 | 主记录尺寸；省略时不覆盖已有值 |
| `managed` | string | 否 | 托管标记；普通根记录更新不修改该字段 |
| `components` / `reports` | array | 否 | 组件完整集合；最多 15 个；仅 `replace/upsert` 处理 |

### 4.2 组件

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `id` | string | 否 | 组件 ID；省略时生成；同一请求内不可重复 |
| `name` / `label` | string | 否 | 组件名称 |
| `reportId` / `report` | string | 报表组件必填 | 引用的报表 ID |
| `dashboardType` / `type` | string | 是 | 图表类型，如 `number`、`bar_0`、`line_0`、`pie` |
| `x` / `y` | integer | 是 | 左上角坐标，必须大于等于 0 |
| `width` / `height` | integer | 是 | 组件尺寸，必须大于 0 |
| `xCondition` / `yCondition` / `xGather` | string | 否 | 图表统计轴与汇总字段 |
| `sortCondition` / `sortType` | string | 否 | 排序字段和方向 |
| `recordNum` | integer | 否 | 显示条数，默认 10 |
| `filters` | array | 否 | 组件筛选条件 |

筛选器常用字段为 `id`、`objectId`、`fieldId`、`type`。组件 ID 和筛选器 ID 在同一请求中都必须唯一。

## 5. 常用操作

只改主记录：

```json
{
  "name": "销售经营分析（新版）",
  "description": "更新说明"
}
```

```bash
cloudcc update dashboard . dashboard-id @dashboard-root.json
```

清空所有组件：

```json
{ "components": [] }
```

```bash
cloudcc replace dashboard . dashboard-id @dashboard.json
```

保留所有组件：在 `replace` JSON 中完全省略 `components` 和 `reports`。不要传空数组。

## 6. 删除与完整性

`delete dashboard` 只生成一个 `PLATFORM_DELETE` 步骤，整个操作委托给当前 CLI 环境的 `apiSvc` 标准接口
`/api/dashboard/deleteDashboard`。标准服务负责主页组件引用保护、仪表板与组件关系清理、租户 Mongo
`ChartSnapshot` 清理和配额更新。MetadataService 不再先删除 SQL，也不直接连接租户 Mongo。

项目环境至少需要：

```json
{
  "apiSvc": "https://tenant.example.com/apisvc",
  "metadataService": {"url": "https://metadata.example.com"}
}
```

CLI 通过 `X-CloudCC-API-Service-Base-URL` 请求级传递目标；MetadataService 不配置第二份 `apiSvc`。计划只保存
地址指纹。计划与 apply 的 `apiSvc` 不一致时会 fail closed，并要求重新生成计划。

删除不会自动重试，也不能自动回滚。平台接口内部包含关系库、Mongo 和配额等连续副作用，HTTP 超时后重复调用
可能重复扣减配额。若返回 `uncertain_remote_outcome`，先执行 `detail dashboard`、检查实际仪表板页面及平台配额，
再按审批流程决定恢复或修复，禁止直接重复 apply。

删除后不可再通过详情接口读取该仪表板；执行前应始终复核 destructive plan。

常见错误：

- `api_service_target_required`：当前环境缺少合法 `apiSvc`。
- `api_service_target_changed`：plan 与 apply 使用了不同环境或地址。
- `platform_delete_plan_state_drifted`：仪表板在计划后发生变化，重新生成计划。
- `platform_delete_rejected`：通常是主页组件仍在引用或平台权限拒绝。
- `uncertain_remote_outcome`：不要自动重试，先进行只读状态和配额核对。

## 7. 验证

1. `detail dashboard`：核对主记录字段和 `relationCounts`。
2. `runtime dashboard`：带目标用户、角色、简档和文件夹上下文验证 `visible` 与 `visibilityReason`。
3. 浏览器 UAT：进入实际文件夹验证布局、图表、筛选和权限。

`recentDashboard` 为空只表示该用户尚未打开仪表板，不表示仪表板创建失败。创建/替换不会为了“最近使用”列表而写入运行期最近访问记录。
