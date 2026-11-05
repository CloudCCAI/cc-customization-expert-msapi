# CloudCC 应用程序 Domain 完整 CLI 指南

本指南面向 CLI 用户，覆盖 Application 的查询、主体字段、页签、简档、页脚工具、创建、更新、删除、计划核对、回读、回滚和 provider 边界。页脚工具是应用程序的一部分，不应脱离 Application 生命周期单独操作。

要求：页脚工具和本指南中的规范化回读需要 MetadataService `1.1.106+`。本文使用 `<projectPath>` 表示包含 `cloudcc-cli.config.json` 的 CLI 项目。

## 1. 命令总览

| 目的 | 命令 |
|---|---|
| 列表/过滤 | `cloudcc get application <projectPath> [filter]` |
| 唯一详情 | `cloudcc detail application <projectPath> <appId>` |
| 页脚工具候选 | `cloudcc candidates application <projectPath> footerTools` |
| 创建计划 | `cloudcc create application <projectPath> @spec.json` |
| 更新计划 | `cloudcc update application <projectPath> @spec.json` |
| 更新计划（ID 与 JSON 分开） | `cloudcc update application <projectPath> <appId> @patch.json` |
| 删除计划 | `cloudcc delete application <projectPath> <appId>` |
| 应用计划 | `cloudcc apply msapi <projectPath> <planId>` |
| 查看实际变更 | `cloudcc changes msapi <projectPath> <operationId>` |
| 回滚已应用操作 | `cloudcc rollback msapi <projectPath> <operationId>` |

JSON 可使用 `@file.json`、内联 JSON 或 CLI 支持的 URI 编码 JSON。推荐文件形式，便于审查和版本管理。MSAPI 写命令只返回 `planId`；先检查步骤、风险和警告，再执行 apply。

## 2. Provider 选择与兼容性

| provider | Application 能力 |
|---|---|
| MSAPI | 完整查询；应用、页签、简档、页脚工具的计划式写入；changes/rollback |
| Lightning UIAPI | 保留既有 `/api/newApp/*` 应用命令；不提供 `footerTools` |
| Horizontal UIAPI | 只使用已声明的适配器；不具备能力时失败关闭 |

Universal 版选择 UIAPI 后，不含 `footerTools` 的旧应用 JSON 仍按原适配器发送。create/update JSON 只要包含 `footerTools`，CLI 就会以 `application_footer_tools_require_msapi` 在联网前拒绝，避免静默丢字段。候选命令也要求 MSAPI。

不要把 UIAPI 的原始表单参数与本文的 MSAPI 友好 JSON 混用。旧客户端或旧服务端仍可继续处理不含新字段的请求；使用新页脚能力时必须先升级 MetadataService。

### 2.1 Lightning UIAPI 旧版位置参数命令

以下命令是已有 UIAPI 能力，继续保留；它们不是 MSAPI 友好 JSON 的简写：

```bash
cloudcc create application <projectPath> <appLabel> <appName> [tabIds]
cloudcc get application <projectPath> [encodedCondJson]
cloudcc update application <projectPath> <appId> <appLabel> <appName> [tabIds] [encodedOptionsJson]
cloudcc delete application <projectPath> <appId>
```

- `tabIds` 是真实页签 ID 的逗号分隔列表。创建时省略会使用 `acf000001`；显式列表不含该 ID 时，旧适配器会自动补入。
- create 会按旧流程读取角色列表并组装权限参数。
- `encodedCondJson` 和 `encodedOptionsJson` 使用 `encodeURI(JSON.stringify(...))` 结果；更新前旧适配器会读取应用详情并合并覆盖参数。
- 这些命令直接使用 Lightning `/api/newApp/getAppList`、`/api/newApp/save`、`/api/newApp/deleteApp` 适配器，不产生 MSAPI `planId`，也不能携带 `footerTools`。

```bash
cloudcc create application . "销售工作台" sales_workbench "acf000001,<real-tab-id>"
cloudcc update application . <app-id> "销售工作台" sales_workbench "acf000001,<real-tab-id>" "%7B%22navigationStyle%22:%221%22%7D"
cloudcc get application . '%7B%22appType%22%3A%22app%22%7D'
```

若项目选择 MSAPI，请使用本指南命令总览中的 JSON 形式；若项目选择 UIAPI，上述位置参数与既有 JSON 表单请求保持兼容。两种 provider 的成功响应与回滚能力不同，不能把 UIAPI 直写当成已审计的 plan/apply 操作。

## 3. 查询与详情回读

```bash
cloudcc get application .
cloudcc get application . sales
cloudcc detail application . <app-id>
```

列表过滤是模糊匹配；详情必须使用真实应用 ID。列表的 `applications[]` 保留既有应用字段。详情重点检查：

- `application`：应用主体；
- `relations` / `relationCounts`：语言、页签链接、页签定义、简档关系和页脚关系证据；
- `tabResolved` / `missingTabIds`：链接到的页签是否都能回读；
- `footerTools`：按数字 `seq` 排序的友好结构；
- `footerToolsResolved` / `missingFooterToolComponentIds`：页脚表是否可读、历史组件能否识别。

`footerTools[]` 回读使用 `componentId/kind/name/label/pageApi/icon/panel/autoStart/openWith/seq`，不会把行 ID、应用 ID、内部属性列作为业务字段。若历史组件无法识别，仍会回读且 `kind="unknown"`，同时 `footerToolsResolved=false`；先确认组件来源再更新，不能猜测类型。

所有 `appId`、`tabId`、`profileId` 和 `componentId` 都应来自 CLI 或服务回读，不得自行构造。

## 4. 应用主体字段

| 字段 | 用途与规则 |
|---|---|
| `id` / `appId` | 更新、删除必需；创建可由服务生成 |
| `name`（兼容 `appName/appname`） | 应用内部名称 |
| `label`（兼容 `labelName/appLabel`） | 用户显示名；同时维护当前语言标签 |
| `description` | 说明 |
| `apiName` | 创建时可指定；更新时保留现有 API 名，不用于重命名 |
| `appType` | 应用类型；创建默认 `app` |
| `layoutType` | 布局类型 |
| `appStyle` | 应用样式 |
| `backgroundColor` / `bgcolor` | 背景色 |
| `hightnum` / `hightNum` | 既有高度参数；创建默认 `200` |
| `mobile` / `mobileImage` | 移动端可用性及图片 |
| `defaultLaunchTabId` / `defaultTabId` | 默认启动页签 ID |
| `navigationStyle` | 导航样式 |
| `toolAlignment` | 应用工具对齐方式；不是 `footerTools` 的别名 |
| `custom`、`managed`、`seq` | 自定义/托管标记与应用顺序 |

主体更新沿用既有 Application 编译语义，不是任意字段 PATCH：一旦请求包含主体字段，部分省略字段会使用既有默认值，`apiName` 则始终保留。因此稳妥做法是先 `detail`，以当前主体为基础构造更新 JSON。只操作页脚工具时不要顺带提交主体字段。

## 5. 页签 `tabs`

绑定已有页签最安全：

```json
{"tabs":["<tab-id-1>","<tab-id-2>"]}
```

也兼容逗号分隔的 `tabIds`。创建应用时可提交页签对象并随应用创建：

```json
{
  "tabs":[{
    "create":true,
    "id":"<optional-tab-id>",
    "name":"contract_workspace",
    "label":"合同工作台",
    "type":"page",
    "objectId":"<optional-object-id>",
    "url":"<optional-url>",
    "tabStyle":"cloudtab145",
    "mobileImage":"<optional-image>",
    "seq":0
  }]
}
```

对象形式还支持 `description/remark`、`visualPageId/vpageId`、`displayType`、`parentTab`、`functionCode`、`managed`、`cloudccServiceTab` 和 `apiName`。更新应用时对象只用于重新建立应用链接，不会把它当成页签创建命令；需要新页签时先使用 `menu` Domain 创建并回读 ID。

更新时非空 `tabs` 采用替换链接语义。省略 `tabs` 会保留现有链接；当前版本的空数组也不会清空链接，如需重构请提交目标完整非空集合。引用不存在的页签会报 `application_tab_reference_missing`。

## 6. 简档可见性与默认应用

| 字段 | 语义 |
|---|---|
| `visibleProfileIds` / `profileIds` / `isvisible` | 可见简档集合 |
| `visibleProfiles` | 字符串 ID 或 `{profileId/id}` 对象数组 |
| `allProfileIds` / `allProfiles` / `profileUniverse` | 本次参与计算的完整简档集合 |
| `defaultProfileIds` / `isdefault` | 新建关系时将应用设为默认的简档 |

创建时若可见集合和全集都省略，服务读取运行时全部简档并默认全部可见；只给可见集合时，该集合同时作为全集。更新已有关系时只更新现有简档行，不自动补建缺失关系。

兼容注意：不含 `footerTools` 的旧式主体更新保留既有简档替换行为；若省略可见集合，现有可见行可能被改为不可见。为了兼容旧版本，该行为没有被暗改。执行主体更新时应先从详情读取简档关系，并显式提交目标 `visibleProfileIds`。只提交 `footerTools` 时不会触碰简档。

## 7. 页脚工具 `footerTools`

先发现候选：

```bash
cloudcc candidates application . footerTools
```

候选始终包含内置群策，并可汇总已存在且能识别的租户配置。`complete=false` 是正常的：已启用呼叫中心适配器和未配置的自定义页面可能来自其他权威来源。未出现在候选中的组件只能使用从目标环境真实读取的描述符。

| 字段 | 规则 |
|---|---|
| `componentId`（别名 `component`） | 必填，同一数组不得重复 |
| `kind` / `componentKind` | `ccchat`、`callcenter`、`customPage` |
| `name` / `componentName` | 呼叫中心和自定义页面需使用来源可信的名称 |
| `label` | 显示名；省略时沿用现值或名称 |
| `pageApi` | 仅 `customPage` 使用且必填；切换为其他类型时旧值会清除 |
| `icon` | 仅 `{ "code":"...", "name":"..." }`；不支持文件、字节或上传 ID |
| `panel.width` | 整数 1..1920，默认 340 |
| `panel.height` | 整数 1..2560，默认 480 |
| `autoStart` | 布尔值，默认 `true` |
| `openWith` | `eject` 或 `rightSide`，默认 `eject` |

禁止提交 `id/appId/app_id/customcomponentid/attribute/iscustom/custom/tooltype/columns` 等内部字段。

集合语义：

| 操作 | 省略 `footerTools` | `footerTools: []` | 非空数组 |
|---|---|---|---|
| create | 不创建 | 不创建 | 按数组创建 |
| update | 完整保留 | 明确清空 | 按数组完整对账 |

完整对账会删除未提交的现有工具，按数组顺序设置 `seq`，保留未变化行 ID，并只生成真实差异步骤。完全相同的数组可以生成零变更计划。计划绑定页脚集合指纹；规划后若外部修改，apply 以 `application_footer_tools_drift` 失败，必须重新读取并生成计划。

## 8. 完整创建示例

```json
{
  "name":"sales_workbench",
  "label":"销售工作台",
  "description":"销售人员统一入口",
  "apiName":"sales_workbench",
  "appType":"app",
  "appStyle":"1",
  "navigationStyle":"1",
  "mobile":true,
  "tabs":["<home-tab-id>","<opportunity-tab-id>"],
  "defaultLaunchTabId":"<home-tab-id>",
  "allProfileIds":["<sales-profile-id>","<admin-profile-id>"],
  "visibleProfileIds":["<sales-profile-id>","<admin-profile-id>"],
  "defaultProfileIds":["<sales-profile-id>"],
  "footerTools":[
    {
      "componentId":"cloudcc_im_ccchat",
      "kind":"ccchat",
      "label":"团队协作",
      "icon":{"code":"footer-ccchat","name":"ccchat"},
      "panel":{"width":340,"height":480},
      "autoStart":true,
      "openWith":"eject"
    },
    {
      "componentId":"<custom-component-id>",
      "kind":"customPage",
      "name":"知识助手",
      "label":"知识助手",
      "pageApi":"knowledge_assistant",
      "openWith":"rightSide"
    }
  ]
}
```

```bash
cloudcc create application . @application.json
cloudcc apply msapi . <planId>
cloudcc changes msapi . <operationId>
cloudcc detail application . <appId>
```

验收时至少确认应用主体、`tabResolved=true`、简档关系、`footerToolsResolved=true` 及工具顺序。

## 9. 安全更新示例

只改页脚工具，不触碰主体、页签或简档：

```json
{
  "id":"<app-id>",
  "footerTools":[
    {"componentId":"cloudcc_im_ccchat","kind":"ccchat","label":"协作"}
  ]
}
```

明确清空：

```json
{"id":"<app-id>","footerTools":[]}
```

主体更新应带上目标简档集合：

```json
{
  "id":"<app-id>",
  "name":"sales_workbench",
  "label":"销售工作台（新版）",
  "appType":"app",
  "hightnum":"200",
  "mobile":true,
  "visibleProfileIds":["<sales-profile-id>","<admin-profile-id>"]
}
```

也可把 ID 与 JSON 分开：

```bash
cloudcc update application . <app-id> @application-patch.json
```

## 10. 删除、变更和回滚

```bash
cloudcc delete application . <app-id>
cloudcc apply msapi . <planId>
cloudcc changes msapi . <operationId>
cloudcc rollback msapi . <operationId>
```

删除计划先处理页脚工具、语言、简档和页签链接，再删除应用主体；不会删除页签定义本身。若应用仍是任何简档的默认应用，`application_default_in_use` 会拒绝删除。删除计划也绑定页脚集合指纹，外部变化会使旧计划失效。Mongo 中的上传图标不在当前清理合同内，因此本版本禁止通过友好 API 创建该类图标。

回滚只针对成功应用且仍满足回滚条件的操作。回滚后再次执行 `detail`，不要只依赖命令退出码。

## 11. 常见错误与排障清单

| 错误码 | 处理 |
|---|---|
| `application_id_required` / `application_not_found` | 使用详情回读得到的真实应用 ID |
| `application_tab_reference_missing` | 先创建/查询菜单页签，再提交真实 ID |
| `application_default_in_use` | 先调整相关简档的默认应用 |
| `application_footer_tools_require_msapi` | 将项目 provider 切换到 MSAPI；请求尚未联网 |
| `application_footer_tool_table_unavailable` | 升级 MetadataService/目标元数据结构 |
| `application_footer_tools_array_required` | 使用数组；更新时要保留就省略字段 |
| `application_footer_tool_duplicate_component` | 去除重复 `componentId` |
| `application_footer_tool_kind_required` | 使用三种受支持 kind 之一 |
| `application_footer_tool_page_api_required` | 为 `customPage` 提供真实 `pageApi` |
| `application_footer_tool_panel_invalid` | `panel` 必须是 JSON 对象 |
| `application_footer_tool_panel_size_invalid` | 宽高必须为范围内整数 |
| `application_footer_tool_icon_upload_unsupported` | 改用内置 code/name 描述符 |
| `application_footer_tools_drift` | 重新 detail、重新规划并审查 |

上线检查清单：provider 与服务版本正确；候选来源可信；主体更新已显式携带目标简档；plan 步骤符合预期；apply 后 `changes` 可解释；详情中页签和页脚均已解析；必要时已实际验证 rollback。
