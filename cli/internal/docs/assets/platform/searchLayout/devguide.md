# 搜索布局 CLI 用户指南

本文从能力发现、读取现状、候选项检查、六类布局 JSON、计划审阅和执行，一直覆盖到回读、变更审计、回滚与故障排查。使用技能配置搜索布局时，应先阅读本文，不要直接照抄 setup-web 的私有请求字段。

## 0. 能力边界

```text
cloudcc --version
cloudcc doctor provider <projectPath>
cloudcc domain searchLayout
cloudcc doc searchLayout devguide
```

- `searchLayout`、`search-layout`、`search-layouts` 是同一资源。
- `pagelayout` 管理记录详情页的 section、字段和相关列表；它不管理本文六类搜索布局。
- `lookupFilter` 管理关系字段候选记录的过滤条件；它不管理“查找对话框”的显示字段。
- 当前首版只覆盖六类布局，不包含业务机会专属销售漏斗。
- MSAPI 写入由 MetadataService 生成审计计划，再委托 setup-service 执行 SQL、MongoDB 和缓存副作用。
- `get/capabilities/candidates` 是只读命令；`update` 只生成计划，不会直接修改租户。

## 1. 完整命令

```text
cloudcc capabilities searchLayout <projectPath> <object>
cloudcc get searchLayout <projectPath> <object>
cloudcc candidates searchLayout <projectPath> <object> searchResult
cloudcc candidates searchLayout <projectPath> <object> lookupDialog
cloudcc candidates searchLayout <projectPath> <object> menu
cloudcc candidates searchLayout <projectPath> <object> listView
cloudcc candidates searchLayout <projectPath> <object> quickCreate
cloudcc candidates searchLayout <projectPath> <object> duplicateMerge
cloudcc update searchLayout <projectPath> <@search-layout.json>
```

MSAPI 完整执行链：

```text
cloudcc update searchLayout . @search-layout.json
cloudcc apply msapi <planId>
cloudcc operation msapi <operationId>
cloudcc changes msapi <operationId>
cloudcc get searchLayout . <object>
```

需要撤销时：

```text
cloudcc rollback-plan msapi <operationId>
cloudcc rollback msapi <operationId>
cloudcc get searchLayout . <object>
```

## 2. 第一步：检查对象支持哪些 section

```text
cloudcc capabilities searchLayout . <object>
```

典型返回：

```json
{
  "sections": {
    "searchResult": {"supported": true, "writable": true, "resultFieldLimit": 10},
    "lookupDialog": {"supported": true, "writable": true},
    "menu": {"supported": true, "writable": true},
    "listView": {"supported": true, "writable": true, "recentButtonMaxCount": 5},
    "quickCreate": {"supported": true, "writable": true},
    "duplicateMerge": {
      "supported": false,
      "writable": false,
      "reason": "duplicate_merge_not_supported_for_object"
    }
  }
}
```

如果 `duplicateMerge.supported=false`，不要提交该 section。CLI 会在生成计划前返回 `search_layout_section_unsupported`，不会把请求发到写接口。

## 3. 第二步：读取当前完整配置

```text
cloudcc get searchLayout . <object>
```

返回中的 `sections` 是规范化状态。已选字段和按钮会同时返回稳定 ID、API 名、标签；`required=true` 表示平台固定项。更新文件可以使用候选项中唯一的 ID 或 API 名，推荐在跨环境脚本中使用 API 名。

不要把 `get` 返回中的 `capabilities`、对象 ID或标签复制到更新文件。更新文件只需要 `object` 和要修改的 `sections`。

## 4. 第三步：逐 section 检查候选项

```text
cloudcc candidates searchLayout . <object> searchResult
```

候选项不是对象全部字段的简单列表。不同 section 有不同过滤规则，例如快速新建会排除公式、汇总、长文本、关系、多选关系、图片和文件字段；查重合并还会排除日期、日期时间等字段。始终以目标环境 candidates 返回为准。

候选项结构：

```json
{
  "section": "searchResult",
  "candidates": {
    "resultFields": [
      {"id": "...", "apiName": "name", "label": "客户名称", "required": true},
      {"id": "...", "apiName": "phone", "label": "电话"}
    ],
    "searchFields": [],
    "exactSearchFields": [],
    "customButtons": [],
    "resultFieldLimit": 10
  }
}
```

## 5. 更新语义

搜索布局使用 section 级补丁：

| 输入 | 含义 |
|---|---|
| 省略某个 section | 完整保留该 section |
| 提交 section，但省略某属性 | 保留该属性 |
| 提交非空数组 | 按数组顺序完整替换该有序集合 |
| 提交空数组 | 仅对平台接口明确支持清空的集合生效；不支持时计划前失败 |
| 固定字段未出现在数组中 | 服务按 candidates 规则自动补齐并在计划中展示 |
| 不支持的 section/属性 | 计划前失败，不静默忽略 |

因此只修改快速新建时，不要复制其他五个 section：

```json
{
  "object": "account",
  "sections": {
    "quickCreate": {
      "enabled": true,
      "fields": ["name", "phone"]
    }
  }
}
```

## 6. 搜索结果 searchResult

普通搜索：

```json
{
  "object": "account",
  "sections": {
    "searchResult": {
      "resultFields": ["name", "phone", "ownerid"],
      "searchMode": "normal",
      "searchFields": ["name", "phone"],
      "customButtons": []
    }
  }
}
```

精确搜索：

```json
{
  "object": "account",
  "sections": {
    "searchResult": {
      "resultFields": ["name", "phone"],
      "searchMode": "exact",
      "exactSearchFields": [
        {"field": "name", "control": true},
        {"field": "phone", "control": false}
      ]
    }
  }
}
```

规则：

- `resultFields` 最多 10 项，顺序就是搜索结果列顺序。
- `searchMode` 只能为 `normal` 或 `exact`。
- 普通模式使用 `searchFields`，精确模式使用 `exactSearchFields`；非当前模式的字段会被规范化为空。
- `exactSearchFields[].control` 必须是布尔值，不能使用字符串 `"true"`。
- `customButtons` 只接受该 section 返回的自定义按钮候选。
- `name` 等固定字段即使省略也会自动补齐；审阅 plan 时确认最终顺序。

## 7. 查找对话框 lookupDialog

```json
{
  "object": "account",
  "sections": {
    "lookupDialog": {
      "conditionFields": ["name", "phone"],
      "resultFields": ["name", "phone", "ownerid"]
    }
  }
}
```

- `conditionFields` 控制查找条件。
- `resultFields` 控制查找结果列，两者不是同一个集合。
- `name` 通常是查找结果固定首字段；服务会根据 candidates 自动补齐。
- 两个数组都有顺序。当前平台接口不能可靠清空 `conditionFields`，请省略它来保留原值；提交空数组会返回 `search_layout_collection_not_clearable`。`resultFields: []` 会保留平台固定字段并移除其他字段。

## 8. 菜单 menu

```json
{
  "object": "account",
  "sections": {
    "menu": {
      "desktopFields": ["name", "phone", "ownerid"],
      "mobileFields": ["name", "phone"],
      "mobileScreenFields": ["status"],
      "embeddedPages": [],
      "defaultMobileTab": "0002"
    }
  }
}
```

- `desktopFields`、`mobileFields`、`mobileScreenFields` 使用不同候选集合，不能混用。
- `embeddedPages` 使用页面候选，不是字段候选。
- `defaultMobileTab` 必须来自 candidates 的 `defaultMobileTabs`。
- 某些对象会返回 `options.showOpportunityStage`，只有 capabilities/state 已出现该属性时才可修改：

```json
{"options":{"showOpportunityStage":true}}
```

- `kanban.groupField` 只能使用菜单 candidates 中的 `groupFields`（分组类型字段）。
- `kanban.summaryField` 只能使用菜单 candidates 中的 `summaryFields`（数字/货币汇总字段）。省略会保留，空字符串会清空。

## 9. 列表视图 listView

```json
{
  "object": "account",
  "sections": {
    "listView": {
      "standardButtons": ["new", "edit"],
      "customButtons": ["batch_assign"],
      "recentButton": {
        "enabled": true,
        "maxCount": 3
      }
    }
  }
}
```

- 标准按钮与自定义按钮使用不同候选集合。
- `maxCount` 必须在 `1..recentButtonMaxCount` 范围内；上限来自目标环境，不能假定永远为 5。
- 只修改最近使用按钮时，可以省略两个按钮数组。

## 10. 快速新建 quickCreate

```json
{
  "object": "account",
  "sections": {
    "quickCreate": {
      "enabled": true,
      "fields": ["name", "phone"]
    }
  }
}
```

- `enabled` 与 `fields` 是独立属性；停用不等于删除字段配置。
- 只切换状态时省略 `fields`，会保留原字段。
- 当前平台接口不能可靠清空快速新建字段；显式 `fields: []` 会在计划前返回 `search_layout_collection_not_clearable`。只停用时请设置 `enabled:false` 并省略 `fields`。
- 候选受当前 Profile、版本和字段类型限制。

## 11. 查重合并 duplicateMerge

先检查支持状态：

```text
cloudcc capabilities searchLayout . <object>
cloudcc candidates searchLayout . <object> duplicateMerge
```

更新示例：

```json
{
  "object": "account",
  "sections": {
    "duplicateMerge": {
      "enabled": true,
      "fields": ["name", "phone", "email"]
    }
  }
}
```

- 只有 `supported=true` 时才允许提交。
- `name` 是固定字段，省略时仍会自动补齐。
- `enabled=false` 只停用；省略 `fields` 会保留字段配置。
- `fields: []` 清空可配置字段，但固定字段仍会出现在规范化计划和回读中。

## 12. 一次修改多个 section

可以在一个计划中提交多个 section：

```json
{
  "object": "account",
  "sections": {
    "searchResult": {
      "resultFields": ["name", "phone"],
      "searchMode": "normal",
      "searchFields": ["name", "phone"]
    },
    "lookupDialog": {
      "conditionFields": ["name", "phone"],
      "resultFields": ["name", "phone"]
    },
    "quickCreate": {
      "enabled": true,
      "fields": ["name", "phone"]
    }
  }
}
```

计划会记录完整 before/desired fingerprint，但 `changedSections` 只列出本次提交的 section。Apply 前如果搜索布局已被其他管理员修改，会返回 `search_layout_plan_state_drifted`，需要重新读取并生成计划。

## 13. 审阅、执行和回读

```text
cloudcc update searchLayout . @search-layout.json
```

审阅计划时至少确认：

- `domain` 为 `search-layouts`。
- `executionBackend` 为 `SETUP_SERVICE`。
- `metadata.changedSections` 只包含预期 section。
- 固定字段补齐、字段顺序、按钮顺序和 enabled 状态符合预期。
- 未提交 section 的 desired state 与 before state 一致。

然后执行并回读：

```text
cloudcc apply msapi <planId>
cloudcc operation msapi <operationId>
cloudcc changes msapi <operationId>
cloudcc get searchLayout . <object>
```

只有 apply 成功且回读与 desired state 一致，才算完成。不要把 HTTP 200 或 plan 创建成功当成布局已经生效。

## 14. 回滚

每次成功执行都会记录 setup-service 恢复步骤：

```text
cloudcc rollback-plan msapi <operationId>
cloudcc rollback msapi <operationId>
cloudcc get searchLayout . <object>
```

回滚也会检查当前状态 fingerprint。如果执行后又被其他管理员修改，应先停止并人工审阅，不要跳过 drift 保护。

## 15. 常见错误

| 错误码 | 含义与处理 |
|---|---|
| `search_layout_section_unsupported` | 当前对象不支持该 section；重新运行 capabilities |
| `search_layout_selector_not_candidate` | 字段/按钮/页面不是该 section 候选；重新运行 candidates |
| `search_layout_no_changes` | 提交内容与当前规范化状态相同；无需生成或执行写计划 |
| `search_layout_selector_ambiguous` | API 名匹配多项；改用稳定 ID |
| `search_layout_selector_duplicate` | 数组中多个选择器解析为同一项；去重 |
| `search_layout_field_limit_exceeded` | 搜索结果超过 10 字段 |
| `search_layout_recent_button_invalid` | 最近使用按钮数量超出目标环境上限 |
| `search_layout_property_unsupported` | 对象不支持所提交的可选属性 |
| `search_layout_collection_not_clearable` | 当前平台接口不能可靠清空该集合；省略属性以保留现值 |
| `search_layout_plan_state_drifted` | 计划创建后状态发生变化；重新 get/plan |
| `search_layout_readback_mismatch` | 保存后回读与计划不一致；检查 changes 并准备回滚 |
| `cloudcc_setup_svc_not_configured` | MetadataService 未配置 setup-service 地址 |

## 16. 推荐操作清单

1. `cloudcc capabilities searchLayout ...` 确认支持的 section。
2. `cloudcc get searchLayout ...` 保存修改前状态。
3. 对每个要修改的 section 执行 `candidates`。
4. JSON 只提交需要修改的 section 和属性。
5. 生成计划，检查固定字段、顺序、changedSections 和 before/desired。
6. 显式 apply。
7. 检查 operation 与 changes。
8. 再次 get，核对六类布局的规范化回读。
9. 如需撤销，先审阅 rollback-plan，再执行 rollback 并再次回读。

```text
cloudcc doc searchLayout introduction
cloudcc doc searchLayout devguide
```
