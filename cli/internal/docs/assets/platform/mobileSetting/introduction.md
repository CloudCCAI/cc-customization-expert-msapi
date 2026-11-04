# 移动端设置 CLI

`mobileSetting` 把 setup-web 的七组移动端设置纳入 CloudCC CLI。它复用现有 setup-service 契约，但不要求、也不会修改 setup-web 或 setup-service 代码。

## 能力范围

| CLI 资源 | setup-web 路由 | 用途 |
|---|---|---|
| `parameters` | `/#/settings/mobile-setting-parameter` | 通用移动参数及活动图片水印 |
| `homepage` | `/#/settings/mobile-setting-homepage` | 主页组件的启用、停用、关闭状态 |
| `offlineCache` | `/#/settings/mobile-setting-offline` | 离线缓存开关、覆盖策略、缓存页签 |
| `recommendedModules` | `/#/settings/mobile-setting-standard-module` | 推荐功能模块 |
| `scanModules` | `/#/settings/mobile-setting-scan-module` | 扫一扫功能模块 |
| `homepageShortcuts` | `/#/settings/mobile-setting-homepage-shortcut` | 主页快捷功能及显示开关 |
| `firstPage` | `/#/settings/mobile-setting-first-page/mobile-setting-unified-menu` | 一级页面的四个入口 |

## 最短上手路径

```powershell
cloudcc capabilities mobileSetting .
cloudcc get mobileSetting . homepage
cloudcc candidates mobileSetting . homepage
cloudcc update mobileSetting . homepage '@homepage.json'
```

在 MSAPI 包中，`update` 只生成计划，不会立即写入。检查输出中的风险、警告和差异后，再运行：

```powershell
cloudcc apply msapi . <planId>
cloudcc changes msapi . <changeId>
```

在 Lightning UIAPI 包中，CLI 直接调用 setup-service，并在写入后回读核对。Horizontal UIAPI 没有经过验证的持久化契约，会以 `mobile_settings_horizontal_uiapi_unsupported` 明确失败。

完整字段、示例、风险与排错说明：

```powershell
cloudcc doc mobileSetting devguide
```
