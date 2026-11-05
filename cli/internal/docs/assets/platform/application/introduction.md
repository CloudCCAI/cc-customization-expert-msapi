# CloudCC 应用（Application）

`application` 是完整的应用程序 Domain，不只是页脚工具入口。它统一管理应用主体、显示名称、导航页签、简档可见性、默认应用关系，以及应用拥有的 `footerTools`。页脚工具属于应用生命周期的一部分，不是独立一级 Domain。

## 适用场景

- 查询应用列表，按 ID、名称或 API 名定位应用；
- 创建或更新应用主体及本地化显示名称；
- 绑定已有菜单页签，或在创建应用时一并创建自定义页签；
- 配置哪些简档可见、哪些简档将其作为默认应用；
- 发现并配置群策、呼叫中心或自定义页面页脚工具；
- 生成计划、人工核对、应用、查看变更和回滚；
- 删除非默认应用及其应用自有关系。

## 能力与边界

MetadataService `1.1.106+` 提供完整的 MSAPI plan/apply 生命周期。所有写命令只生成计划，不会直接修改目标环境；必须另行执行 `cloudcc apply msapi`。详情回读会同时报告页签和页脚工具是否完整解析。

| 能力 | MSAPI | Lightning UIAPI | Horizontal UIAPI |
|---|---|---|---|
| 应用旧版查询/创建/更新/删除 | 支持 | 保留既有适配器 | 按平台适配器能力，未适配时失败关闭 |
| plan/apply/changes/rollback | 支持 | 不提供 MSAPI 计划 | 不提供 MSAPI 计划 |
| `footerTools` 候选和生命周期 | 支持 | 不支持；联网前失败关闭 | 不支持；失败关闭 |

兼容规则：更新省略 `footerTools` 会保留现有页脚工具；传 `[]` 明确清空；传非空数组按提交顺序完整对账。既有不含该字段的 UIAPI 请求不受影响。二进制或上传图标不在本能力范围内。

## 快速开始

```bash
cloudcc get application <projectPath> [filter]
cloudcc detail application <projectPath> <appId>
cloudcc candidates application <projectPath> footerTools
cloudcc create application <projectPath> @application.json
cloudcc apply msapi <projectPath> <planId>
cloudcc detail application <projectPath> <appId>
```

完整字段、集合语义、更新注意事项、删除保护和排障方法见：

```bash
cloudcc doc platform/application devguide
```
