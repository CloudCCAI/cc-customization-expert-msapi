# CloudCC 菜单 CLI 命令说明

## 支持的命令

| 操作 | 说明 |
|------|------|
| `create menu object` | 创建自定义对象菜单 |
| `create menu page` | 创建自定义页面菜单 |
| `create menu script` | 创建自定义脚本菜单 |
| `create menu site` | 创建站点菜单 |
| `update menu page` | 更新自定义页面菜单 |
| `get menu` | 查询菜单列表 |
| `delete menu` | 删除菜单 |

## CLI 命令详解

### 创建自定义对象菜单

```bash
cloudcc create menu object <path> <objectId-or-objectApiName> <tabName> [tabStyle] [mobileimg] [cloudccservicetab]
```

**参数说明：**

| 参数 | 必填 | 说明 |
|------|------|------|
| `path` | 是 | 项目路径，`.` 表示当前目录 |
| `objectId-or-objectApiName` | 是 | 自定义对象真实 ID 或对象 API 名；推荐传对象 API 名 |
| `tabName` | 是 | 菜单显示名称 |
| `tabStyle` | 否 | PC 端图标（默认 `cloudtab145`）|
| `mobileimg` | 否 | 移动端图标（默认 `cloudcc01`）|
| `cloudccservicetab` | 否 | 服务图标（默认 `cloudccservicetab_1`）|

**对象引用规则：**

- `objectId` 可以使用，但必须来自 CloudCC/CLI 创建回执、扫描或 detail/readback，不能自行构造。
- 如果已知对象 API 名，优先传 `objectApiName`；MetadataService 会解析真实 `objectId` 和 `objectPrefix` 后再写选项卡。
- 如果同时传 `objectId`、`objectApiName`、`objectPrefix`，系统会校验它们是否指向同一个对象；不一致会拒绝计划。
- `objectPrefix` 是运行时属性，一般不需要调用方填写；仅在离线兼容场景下作为兜底。

**简档可见性规则：**

- `profileIds` / `profiles` 表示菜单可见简档，简档 ID 也必须来自 CLI 查询或回读，不能自行构造。
- 如果没有传简档集合，或集合为空，MetadataService 默认按当前租户 `tp_sys_profile` 的全部简档生成菜单可见性。
- 只传 `aaa000001` 通常只代表系统管理员简档可见，不代表其它简档可见。

**示例：**

```bash
# 推荐：使用对象 API 名
cloudcc create menu object . after_sales_ticket "售后工单"

# 兼容：使用 CLI 回读到的真实对象 ID
cloudcc create menu object . obj2026B20A71E7MYvUj "售后工单"
```

### 创建自定义页面菜单

```bash
cloudcc create menu page <path> <pageApi> <tabName> <pname> [tabStyle] [mobileimg] [cloudccservicetab] [mobileurl]
```

**参数说明：**

| 参数 | 必填 | 说明 |
|------|------|------|
| `path` | 是 | 项目路径 |
| `pageApi` | 是 | 自定义页面 API 名称（CLI 会追加 `#lightning`）|
| `tabName` | 是 | 菜单显示名称 |
| `pname` | 是 | 菜单内部名称（建议字母开头）|
| `tabStyle` / `mobileimg` / `cloudccservicetab` | 否 | 图标参数 |
| `mobileurl` | 否 | 移动端地址 |

**示例：**

```bash
cloudcc create menu page . contract-assistant "合同助手" contract_menu
```

### 创建自定义脚本菜单

```bash
cloudcc create menu script <path> <tabName> <pname> [functioncode] [tabStyle] [mobileimg] [cloudccservicetab]
```

**参数说明：**

| 参数 | 必填 | 说明 |
|------|------|------|
| `path` | 是 | 项目路径 |
| `tabName` | 是 | 菜单显示名称 |
| `pname` | 是 | 菜单内部名称 |
| `functioncode` | 否 | 脚本内容（默认示例脚本）|

**示例：**

```bash
cloudcc create menu script . "数据导入工具" data_import_menu "ccc.alert('Hello World');"
```

### 创建站点菜单

```bash
cloudcc create menu site <path> <siteId> <tabName> [tabStyle] [mobileimg] [cloudccservicetab]
```

**参数说明：**

| 参数 | 必填 | 说明 |
|------|------|------|
| `path` | 是 | 项目路径 |
| `siteId` | 是 | 站点 ID |
| `tabName` | 是 | 菜单显示名称 |

**示例：**

```bash
cloudcc create menu site . a0H9D000000XXXXUAI "合作伙伴门户"
```

### 查询菜单列表

```bash
cloudcc get menu <projectPath> [encodedCondJson]
```

**示例：**

```bash
# 查询所有菜单
cloudcc get menu .

# 带查询条件
cloudcc get menu . '%7B%22type%22%3A%22page%22%7D'
```

### 更新自定义页面菜单

```bash
cloudcc update menu page <path> <tabId> <pageApi> [encodedOptionsJson]
```

**参数说明：**

| 参数 | 必填 | 说明 |
|------|------|------|
| `path` | 是 | 项目路径 |
| `tabId` | 是 | 菜单 ID |
| `pageApi` | 是 | 目标页面 API（支持传 `xxx` 或 `xxx#lightning`） |
| `encodedOptionsJson` | 否 | 覆盖参数，需传 `encodeURI(JSON.stringify(obj))` |

`encodedOptionsJson` 可覆盖字段：
- `tabName`
- `pname`
- `tabStyle`
- `mobileimg`
- `cloudccservicetab`
- `mobileurl`
- `p3`
- `lightningPage`

更新行为说明（CLI）：

1. 根据 `tabId` 读取目标菜单并校验类型  
2. 将 `pageApi` 规范为 `lightningPage`（自动补 `#lightning`）  
3. 合并可编辑字段（如 `tabName`、`pname`、图标等）并执行保存  
4. 命令成功后返回更新成功提示

**示例：**

```bash
# 仅更新 pageApi
cloudcc update menu page . a0I9D000000XXXXUAI contract-assistant

# 更新 pageApi + 菜单名称
cloudcc update menu page . a0I9D000000XXXXUAI contract-assistant '%7B%22tabName%22%3A%22合同助手%22%7D'
```

### 删除菜单

```bash
cloudcc delete menu <projectPath> <tabId>
```

**示例：**

```bash
cloudcc delete menu . a0I9D000000XXXXUAI
```
