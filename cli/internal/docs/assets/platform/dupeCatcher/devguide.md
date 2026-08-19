# CloudCC 查重过滤器（Dupe Catcher）CLI 命令说明

> 对应 one-setup-web 的接口前缀：`{setupSvc}/api/duplication/*`

## 1. 功能范围

- 过滤器（Filter）：列表、详情、保存（新建/编辑）、删除
- 规则（Rule）：回显详情、保存（新建/编辑）、删除
- 辅助查询：可选对象、可选字段、条件面板字段列表

## 2. 命令总览

```bash
# 过滤器列表
cloudcc get dupeCatcher [projectPath]

# 过滤器详情（含规则列表）
cloudcc detail dupeCatcher <projectPath> <filterid>

# 保存过滤器（新建/编辑）
cloudcc create dupeCatcher <projectPath> filter <encodedBodyJson>

# 删除过滤器
cloudcc delete dupeCatcher <projectPath> filter <filterid>

# 规则详情（回显）
cloudcc detail dupeCatcher <projectPath> rule <ruleid>

# 保存规则（新建/编辑）
cloudcc create dupeCatcher <projectPath> rule <encodedBodyJson>

# 删除规则
cloudcc delete dupeCatcher <projectPath> rule <ruleid>

# 新建过滤器前：查询可选对象
cloudcc get dupeCatcher objects [projectPath]

# 新建规则前：查询可选字段
cloudcc get dupeCatcher <projectPath> fields <filterid>

# 条件面板字段列表（复用工作流接口）
cloudcc get dupeCatcher <projectPath> workflowFields <targetobjectid>
```

## 3. 参数说明

### 3.1 projectPath

- **可选**，项目根目录（默认当前目录）。

### 3.2 encodedBodyJson

- **必须**：`encodeURI(JSON.stringify(body))`

### 3.3 命令参数一览

| 命令 | 参数 |
| :--- | :--- |
| `cloudcc get dupeCatcher [projectPath]` | `projectPath` 可选 |
| `cloudcc detail dupeCatcher <projectPath> <filterid>` | `projectPath` 可选（默认当前目录），`filterid` 必填 |
| `cloudcc create dupeCatcher <projectPath> filter <encodedBodyJson>` | `projectPath` 可选（默认当前目录），`encodedBodyJson` 必填 |
| `cloudcc delete dupeCatcher <projectPath> filter <filterid>` | `projectPath` 可选（默认当前目录），`filterid` 必填 |
| `cloudcc detail dupeCatcher <projectPath> rule <ruleid>` | `projectPath` 可选（默认当前目录），`ruleid` 必填 |
| `cloudcc create dupeCatcher <projectPath> rule <encodedBodyJson>` | `projectPath` 可选（默认当前目录），`encodedBodyJson` 必填 |
| `cloudcc delete dupeCatcher <projectPath> rule <ruleid>` | `projectPath` 可选（默认当前目录），`ruleid` 必填 |
| `cloudcc get dupeCatcher objects [projectPath]` | `projectPath` 可选 |
| `cloudcc get dupeCatcher <projectPath> fields <filterid>` | `projectPath` 可选（默认当前目录），`filterid` 必填 |
| `cloudcc get dupeCatcher <projectPath> workflowFields <targetobjectid>` | `projectPath` 可选（默认当前目录），`targetobjectid` 必填 |

### 3.4 `create dupeCatcher filter` 的 body 字段

| 字段 | 类型 | 是否必须 | 说明 |
| :--- | :--- | :--- | :--- |
| `id` | string | 否 | 编辑时传；新建可不传 |
| `name` | string | 是 | 过滤器名称 |
| `objid` | string | 是 | 目标对象 id |
| `errormessage` | string | 否 | 错误提示文案 |
| `insertoperation` | string | 是 | 新增策略：`check` / `nothing` |
| `insertIsSave` | string/null | 条件必填 | `insertoperation=check` 时建议传 `"1"`/`"0"`，否则传 `null` |
| `insertIsTips` | string/null | 条件必填 | `insertoperation=check` 时建议传 `"1"`/`"0"`，否则传 `null` |
| `updateoperation` | string | 是 | 更新策略：`check` / `nothing` |
| `updateIsSave` | string/null | 条件必填 | `updateoperation=check` 时建议传 `"1"`/`"0"`，否则传 `null` |
| `updateIsTips` | string/null | 条件必填 | `updateoperation=check` 时建议传 `"1"`/`"0"`，否则传 `null` |
| `isprofile` | boolean | 否 | 是否简档 |
| `isactive` | boolean | 否 | 是否启用 |
| `conditionVals` | string | 是 | 条件面板产出的 JSON 字符串 |

### 3.5 `create dupeCatcher rule` 的 body 字段

| 字段 | 类型 | 是否必须 | 说明 |
| :--- | :--- | :--- | :--- |
| `id` | string | 否 | 编辑时传；新建可不传 |
| `filterid` | string | 是 | 所属过滤器 id |
| `fieldid` | string | 是 | 匹配字段 id |
| `matchoption` | string | 是 | `exact` / `contains` / `firstNletters` / `lastNletters` |
| `firstletters` | string | 条件必填 | `matchoption` 为 `firstNletters/lastNletters` 时传字母数 |
| `isblank` | number | 否 | 空值是否参与匹配，常用 `1/0` |

## 4. 示例

### 4.1 查询过滤器列表

```bash
cloudcc get dupeCatcher .
```

### 4.2 保存过滤器（新建）

> 关键字段（与页面一致）：`name`、`objid`、`insertoperation`、`updateoperation`、`conditionVals` 等。

```bash
cloudcc create dupeCatcher . filter '{"name":"测试过滤器","objid":"<objid>","errormessage":"","insertoperation":"check","insertIsSave":"1","insertIsTips":"1","updateoperation":"check","updateIsSave":"1","updateIsTips":"1","isprofile":false,"isactive":true,"conditionVals":"{}"}'
```

### 4.3 保存规则（新建）

```bash
cloudcc create dupeCatcher . rule '{"filterid":"<filterid>","fieldid":"<fieldid>","matchoption":"exact","firstletters":"","isblank":0}'
```

## 5. 注意事项

- `filterid` / `ruleid` / `objid` / `fieldid` 都来自平台返回数据，请先通过列表/详情接口获取。
- `conditionVals` 是页面条件面板产出的 **json 字符串**（不是对象），请按接口要求传入字符串。
