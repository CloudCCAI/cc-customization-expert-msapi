# CloudCC Script 文档（全量）

## 客户端脚本（Script）操作指南

直接查看：`cloudcc doc platform/script`

---

## 1. 常见操作

- 查询客户端脚本列表
- 查看客户端脚本详情
- 新建客户端脚本（生成本地目录与模板）
- 发布客户端脚本到环境（保存/更新线上脚本）
- 从环境拉取脚本内容到本地（覆盖本地脚本文件）
- 删除客户端脚本

---

## 2. 常用 CLI 命令

### 1) 查看客户端脚本文档

命令：

```bash
cloudcc doc platform/script
```

说明：输出客户端脚本文档，并自动附带本文件内容。

---

### 2) 查询客户端脚本列表

命令：

```bash
cloudcc get script <encodedCondJson> <projectPath>
```

参数说明：

| 参数名称 | 类型 | 是否必须 | 默认值 | 备注 |
| :--- | :--- | :--- | :--- | :--- |
| encodedCondJson | string | 视场景 | — | `encodeURI(JSON.stringify(body))` |
| projectPath | string | 非必须 | 当前目录 | 项目根目录 |

请求体（decoded 后）字段（与页面抓包一致）：

| 参数名称 | 类型 | 是否必须 | 默认值 | 备注 |
| :--- | :--- | :--- | :--- | :--- |
| pageSize | number | 非必须 | `50` | 每页条数 |
| pageNo | number | 非必须 | `1` | 页码（从 1 开始） |
| condition | object | 非必须 | `{}` | 查询条件 |
| condition.scriptName | string | 非必须 | `""` | 脚本名称筛选 |
| condition.pageLabel | string | 非必须 | `""` | 页面标签筛选 |
| condition.objName | string | 非必须 | `""` | 对象名称label筛选（例如你说“客户对象”，就传 `客户`；不要让 AI 自动映射成 `account`） |


> 注意：`objName` 直接按你要查询的对象名称来传，并按实际值原样传入。  
> 例如对象名称是 `客户`、`客户004`，就直接传 `客户`、`客户004`；**不要**按语义自动换成 `account`。  
> 如果返回为空，通常意味着目标环境里的对象名称不是你传的这个值（可以去对象管理里看一下该对象的名称/标识）。

---

### 3) 查看客户端脚本详情

命令（两种用法）：

```bash
# 1) 按本地脚本路径查看（不依赖线上）
cloudcc detail script <objName/scriptName>

# 2) 按线上脚本 id 查看
cloudcc detail script "" <scriptId> <projectPath>
```

说明：

- 传 `objName/scriptName`：读取本地 `script/<objName>/<scriptName>/config.json` 和脚本文件。
- 传 `scriptId`：调用线上接口查询并返回详情。

---

### 4) 新建客户端脚本（本地生成）

命令：

```bash
cloudcc create script <encodedBodyJson>
```

参数说明：

| 参数名称 | 类型 | 是否必须 | 默认值 | 备注 |
| :--- | :--- | :--- | :--- | :--- |
| encodedBodyJson | string | 必须 | — | `encodeURI(JSON.stringify(body))` |

请求体（decoded 后）常用字段（会写入 `config.json`）：

| 参数名称 | 类型 | 是否必须 | 默认值 | 备注 |
| :--- | :--- | :--- | :--- | :--- |
| objName | string | 必须 | — | 对象 API 名称（目录名） |
| scriptName | string | 必须 | — | 脚本名称（目录名 + 文件名） |
| pageId / eventType / event / fieldId / usageScenario | string | 非必须 | `""` | 作为脚本配置元数据 |

生成目录结构：

- `script/<objName>/<scriptName>/<scriptName>.js`
- `script/<objName>/<scriptName>/config.json`

---

### 5) 发布客户端脚本（保存到线上）

命令：

```bash
cloudcc publish script <objName/scriptName>
```

说明：

- 读取本地 `script/<objName>/<scriptName>/<scriptName>.js` 与 `config.json`
- 只会发布 `function main($CCDK, obj) { ... }` 的函数体内容（会校验 main 函数存在）
- 首次发布成功后，会把返回的 `id` 写回 `config.json`，用于后续更新/拉取/删除

---

### 6) 拉取客户端脚本（用线上内容覆盖本地）

命令：

```bash
cloudcc pull script <objName/scriptName>
```

说明：要求本地 `config.json` 内已存在 `id`（先发布一次）。

---

### 7) 拉取线上脚本到本地（按 id 拉取并生成目录）

命令：

```bash
cloudcc pullList script <scriptId> <projectPath>
```

说明：会在 `<projectPath>/script/<objName>/<scriptName>/` 下生成 `config.json` 与脚本文件。

---

### 8) 删除客户端脚本

命令（两种用法）：

```bash
# 1) 直接按 scriptId 删除
cloudcc delete script <scriptId> <projectPath>

# 2) 按 "objName/scriptName" 删除（会先查询解析出 id，再删除）
cloudcc delete script <objName/scriptName> <projectPath>
```

---

## 3. 示例

### 查询客户端脚本列表（默认条件）

```bash
cloudcc get script '{"pageSize":50,"pageNo":1,"condition":{"scriptName":"","pageLabel":"","objName":""}}' .
```

### 按脚本名筛选（例如：test）

```bash
cloudcc get script '{"pageSize":50,"pageNo":1,"condition":{"scriptName":"test","pageLabel":"","objName":""}}' .
```

### 按对象名称筛选（示例：`客户` / `客户004`）

```bash
# 对象名称为“客户”
cloudcc get script '{"pageSize":50,"pageNo":1,"condition":{"scriptName":"","pageLabel":"","objName":"客户"}}' .

# 对象名称为“客户004”
cloudcc get script '{"pageSize":50,"pageNo":1,"condition":{"scriptName":"","pageLabel":"","objName":"客户004"}}' .
```

错误示例（不要这样传）：

```bash
# 业务上说“客户”，就不要自作主张写成 account
cloudcc get script '{"pageSize":50,"pageNo":1,"condition":{"scriptName":"","pageLabel":"","objName":"account"}}' .
```

### 新建脚本（本地）

```bash
cloudcc create script '{"objName":"contact","scriptName":"hello_script"}'
```

### 发布脚本（上线）

```bash
cloudcc publish script contact/hello_script
```

### 拉取脚本（从线上覆盖本地）

```bash
cloudcc pull script contact/hello_script
```

### 删除脚本（按 id）

```bash
cloudcc delete script <scriptId> .
```

---

## 4. Checklist

- [ ] 已在正确项目目录执行命令（或显式传入 `projectPath`）
- [ ] `cloudcc get script` 的条件已按 `encodeURI(JSON.stringify(...))` 传入
- [ ] 发布前脚本文件包含 `function main($CCDK, obj) { ... }`
- [ ] 首次发布后确认 `config.json` 已写入 `id`
- [ ] 拉取/删除前确认目标 `id` 正确（避免误删）

---

## 5. 进阶示例

### 5.1 快速入门示例

#### 进入客户端脚本设置页面

1. 登录 CloudCC 系统
2. 点击右上角头像，选择「开发者平台」（仅管理员简档可见）
3. 左侧菜单：`扩展 -> 客户端脚本`

#### 新建并执行第一个脚本

在代码输入区输入：

```javascript
$CCDK.CCMessage.showMessage("hello cloudCC");
```

保存后进入对应页面（如详情页 `onLoad`），可看到提示信息。

### 5.2 基于 devid 控制页面元素

> 可通过浏览器开发者工具（F12）查看页面元素上的 `devid`，再在脚本中精确定位。

#### 禁用编辑页字段

```javascript
let style = document.createElement("style");
style.type = "text/css";
style.innerHTML = '[devid="ffe201100003855g6Ipz"]{pointer-events:none}';
document.getElementsByTagName("head").item(0).appendChild(style);
```

#### 隐藏详情页字段（不占位）

```javascript
let style = document.createElement("style");
style.type = "text/css";
style.innerHTML = '[devid="ffe20220523account01"]{display:none !important}';
document.getElementsByTagName("head").item(0).appendChild(style);
```

#### 隐藏详情页字段（保留占位）

```javascript
let style = document.createElement("style");
style.type = "text/css";
style.innerHTML = '[devid="ffe20220523account01"]{visibility:hidden !important}';
document.getElementsByTagName("head").item(0).appendChild(style);
```

#### 移动页面元素位置

```javascript
const moveNode = document.querySelector('[devid="aee2024EFBBA5EEfs1Ta"]');
const targetNode = document.querySelector('[devid="adf2024D33E03DA6xzBj"]');
const targetParentNode = targetNode && targetNode.parentNode;

if (moveNode && targetNode && targetParentNode) {
    targetParentNode.insertBefore(moveNode, targetNode);
}
```

### 5.3 事件选择建议

- 全局脚本：适合应用启动、销毁等全局行为
- 列表页脚本：适合列表加载后增强
- 新建/编辑页脚本：适合 `beforeSave` 校验与赋值
- 详情页脚本：适合详情渲染后字段展示控制

### 5.4 使用注意事项

- DOM 可能未渲染完成，建议结合 `setTimeout` 或 `MutationObserver`
- 优先使用 `$CCDK` 能力，不建议直接拼接接口调用
- 对频繁执行逻辑做好性能控制，避免页面卡顿
