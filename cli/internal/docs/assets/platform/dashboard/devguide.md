# Lightning Dashboard Dev Guide

## Commands

```bash
tools/bin/cloudcc get dashboard <projectPath> [filter]
tools/bin/cloudcc getList dashboard <projectPath> '{"folderId":"folder-id","lightning":true}'
tools/bin/cloudcc detail dashboard <projectPath> <dashboardId>
tools/bin/cloudcc runtime dashboard <projectPath> '{"userId":"005...","roleId":"...","profileId":"...","folderId":"..."}'

tools/bin/cloudcc create dashboard <projectPath> @dashboard.json
tools/bin/cloudcc apply msapi <projectPath> <planId>
```

Read commands call `/metadata/v1/dashboards`, `/metadata/v1/dashboards/{id}`, and the read-only `/metadata/v1/dashboards/runtime-visible` diagnostic. They do not use the standard catalog.

## Aggregate Create Spec

```json
{
  "name": "销售管理-经营分析",
  "lightning": true,
  "folder": {
    "name": "销售管理",
    "folderType": "lightningdashboard",
    "viewType": "1",
    "purview": "1"
  },
  "components": [
    {
      "name": "本月订单额",
      "reportId": "report-id",
      "dashboardType": "number",
      "x": 0,
      "y": 0,
      "width": 4,
      "height": 3
    }
  ]
}
```

For Lightning dashboards, MetadataService normalizes a newly supplied folder to `foldertype=lightningdashboard`. Omitted private-folder settings default to owner-only `viewtype=1`, writable `purview=1`, and the current actor as owner. A dashboard supports at most 15 components. Component/filter IDs must be unique and component coordinates and dimensions must be valid.

`create` is an aggregate operation: the generated plan includes the optional folder, root, components, and filters. Omitting `components` remains a valid root-only dashboard create.

## Verification

After apply:

1. Use `detail dashboard` and check `relationCounts.tp_sys_dashboard_report`.
2. Use `runtime dashboard` with the target user context and check `visible=true` plus `visibilityReason`.
3. For browser UAT, query the actual folder. An empty `recentDashboard` response only means the user has not opened the dashboard yet.

MetadataService never writes recent-item, display preference, or snapshot runtime state as part of dashboard metadata creation.
