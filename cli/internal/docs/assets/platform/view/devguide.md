# CloudCC 视图开发指南

## 1. 当前支持命令

```bash
cloudcc get view <projectPath> <objId>
cloudcc update view fieldSetupAllViews <projectPath> <objId> <fieldIds(逗号或JSON数组)> [allViewId]
cloudcc doc platform/view introduction
cloudcc doc platform/view devguide
```

## 2. 参数说明

- `projectPath`：项目路径（用于读取配置）
- `objId`：对象 ID
- `fieldIds`：字段 ID 列表，支持：
  - 逗号分隔字符串（例如 `ffe2024D0B157CENv70t,ffe202593C0FF31GbnZM`）
  - JSON 数组（可先 `encodeURI`）
- `allViewId`：可选；不传时 CLI 会先执行 `cloudcc get view <projectPath> <objId>`，按 `label=全部` 自动解析视图 ID

## 3. 使用示例

```bash
# 不传 viewId：自动查 label=全部 的视图
cloudcc update view fieldSetupAllViews . 2024CE595708124vAsZS "ffe2024D0B157CENv70t,ffe202593C0FF31GbnZM,ffe202543798E16dFu7g"

# 显式传入 viewId：按指定视图更新
cloudcc update view fieldSetupAllViews . 2024CE595708124vAsZS "%5B%22ffe2024D0B157CENv70t%22%2C%22ffe202593C0FF31GbnZM%22%5D" aec2026CC9BCAA1YbIgV
```
