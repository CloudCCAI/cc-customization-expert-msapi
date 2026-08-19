# CloudCC 自定义 HTML 组件开发指南

## 1. 模块定位

`html` 模块用于通过 CLI 管理**自定义 HTML 组件**（开发者控制台中的 HTML 组件），当前提供：

- 本地脚手架：`cloudcc create html ...`（在项目下生成 `html/<apiName>/index.html` 与 `config.json`）
- 发布到平台：`cloudcc publish html ...`（上传并自动创建挂载该 HTML 的自定义页面，与控制台行为一致）
- 查询列表：`cloudcc get html ...`
- 查询详情：`cloudcc detail html ...`
- 删除组件：`cloudcc delete html ...`
- 文档查看：`cloudcc doc platform/html introduction|devguide`

---

## 2. 开发前准备

- **仅 `create`**：只需可写本地目录，在项目根下生成 `html/<apiName>/`，**不依赖** `accessToken`。
- **`publish` / `get` / `detail` / `delete`**：需已完成 `cloudcc doc platform/project devguide` 的环境准备，项目配置中含 **`accessToken`**。
- **`publish`** 另需：当前环境在「自定义组件列表」中能解析到内置组件 **`cloudcc-iframe`**。若不存在，会在调用接口前失败。

若配置缺失，相关命令会报错，例如：

```text
Error: Configuration not found or accessToken is missing
```

---

## 3. 命令总览（以代码实现为准）

```bash
cloudcc create html <apiName> [projectPath]
cloudcc publish html <apiName> [projectPath]
cloudcc get html [projectPath] [encodedBodyJson]
cloudcc detail html [projectPath] <id>
cloudcc delete html [projectPath] <id>
cloudcc doc platform/html <introduction|devguide>
```

说明：

- `apiName`：与目录 `html/<apiName>/` 同名；`create` 中仅允许字母、数字、下划线、连字符。
- `projectPath`：项目根目录；多数命令中可省略，默认当前工作目录。
- `encodedBodyJson`（仅 `get` 可选）：`encodeURI(JSON.stringify(对象))` 后的字符串。

---

## 4. 本地创建（create）

在项目根下创建 **`html/<apiName>/`**，并生成 **`config.json`** 与 **`index.html`**（默认带 Tailwind + Alpine 与 `$CCDK` 判断的骨架）。

```bash
cloudcc create html <apiName> [projectPath]
```

参数说明：

- `apiName`：子目录名（必填），与后续 `publish`、配置文件中的 `apiName` 一致。
- `projectPath`：项目根路径，默认当前目录。

示例：

```bash
cloudcc create html MyApi
cloudcc create html MyApi /path/to/project
```

成功时，标准输出为 JSON：`path`（绝对路径）、`apiName`、`htmlLabel`（默认同 `apiName`，可在 `config.json` 中修改）。

若目录已存在，会报错。可在 `config.json` 中编辑 `htmlLabel`；发布时使用文件中的 `apiName`、`htmlLabel` 与 `index.html` 全文作为 `htmlContent`。

---

## 5. 发布（publish）

读取 **`html/<apiName>/config.json`** 与 **`index.html`**，调用保存接口并在成功后自动创建自定义页面（`cloudcc-iframe` 挂载访问路径）。

```bash
cloudcc publish html <apiName> [projectPath]
```

参数说明：

- `apiName`：本地子目录名，用于定位 `html/<apiName>/`。
- `projectPath`：项目根路径，默认当前目录。

示例：

```bash
cloudcc publish html MyApi
cloudcc publish html MyApi /path/to/project
```

### 5.1 发布过程说明（CLI 行为）

1. 读取项目配置  
2. 拉取自定义组件列表，查找 **`cloudcc-iframe`**  
3. 保存 HTML 组件（请求体来自 `config.json` + `index.html`）  
4. 自动创建自定义页面  

成功时，标准输出为**一行 JSON 对象**，包含：

- `htmlComponent`：HTML 组件保存结果  
- `customPage`：自动创建的自定义页面结果  

常见失败场景：

- 未先 `create`：缺少 `html/<apiName>/config.json` 或 `index.html`  
- 无法拉取组件列表、找不到 `cloudcc-iframe`：会在保存 HTML 之前失败  
- HTML 已保存，但自动创建自定义页面失败：错误信息中会带上已创建的 HTML 组件 **`id`**

---

## 6. 查询列表

```bash
cloudcc get html [projectPath] [encodedBodyJson]
```

参数说明：

- `projectPath`：项目路径，默认当前目录  
- `encodedBodyJson`：可选。不传时使用默认分页与空条件；传入时可自定义 `pageNo`、`pageSize`、`condition`（如按 `htmlLabel`、`apiName` 筛选）

默认请求体等价于：

```json
{
  "pageNo": 1,
  "pageSize": 50,
  "condition": { "htmlLabel": "", "apiName": "" }
}
```

示例：

```bash
cloudcc get html
cloudcc get html ./
cloudcc get html ./ '{"pageNo":1,"pageSize":50,"condition":{"htmlLabel":"helloa","apiName":"helloa"}}'
```

成功时，标准输出为**分页数据** JSON（含列表与总数等字段，以实际返回为准）。

---

## 7. 查询详情

```bash
cloudcc detail html [projectPath] <id>
```

参数说明：

- `projectPath`：项目路径，默认当前目录  
- `id`：HTML 组件 ID（必填）

示例：

```bash
cloudcc detail html . 69d780d9e4b0870b069e8fd1
cloudcc detail html ./ 69d780d9e4b0870b069e8fd1
```

成功时，标准输出为**详情对象** JSON（含 `htmlContent`、`accessPath` 等）。

若缺少 `id`，会报错：

```text
HTML Component detail requires id
```

---

## 8. 删除 HTML 组件

```bash
cloudcc delete html [projectPath] <id>
```

参数说明：

- `projectPath`：项目路径，默认当前目录  
- `id`：HTML 组件 ID（必填）

示例：

```bash
cloudcc delete html . 69e599c9e4b0870b069e8fdc
```

成功时，标准输出为**完整响应** JSON（含 `returnInfo` 等）。

若缺少 `id`，会报错：

```text
HTML Component delete requires id
```

---

## 9. 文档命令

```bash
cloudcc doc platform/html introduction
cloudcc doc platform/html devguide
```

说明：

- 仅支持 `introduction` 与 `devguide`  
- 传入其他子命令会抛错  

---

## 10. 推荐操作流程

```bash
# 1) 本地生成目录与模板
cloudcc create html demo

# 2) 编辑 html/demo/index.html 与 html/demo/config.json（如需修改显示名等）

# 3) 发布到平台（需 accessToken 与 cloudcc-iframe）
cloudcc publish html demo

# 4) 查列表 / 按 id 查详情
cloudcc get html
cloudcc detail html . <id>

# 5) 不再需要时按 id 删除（仅平台侧组件；本地目录需自行维护）
cloudcc delete html . <id>
```

---

## 11. 注意事项

- **`publish`** 强依赖环境中能解析到 **`cloudcc-iframe`**，否则无法完成「保存 HTML + 自动建页」  
- **`get` / `detail` / `delete` / `publish`** 均依赖有效 `accessToken`  
- **`create`** 仅写本地文件，不产生线上资源  
- 删除前请确认无菜单或其他入口仍依赖该 HTML 或关联自定义页面  
- **`publish`** 成功输出为 `{ htmlComponent, customPage }` 两段结构，脚本解析时请勿仍按「单对象」处理  

---

## 12. 适用范围（页面开发）

- 本文用于在 CloudCC 相关项目中编写单个 HTML 入口页，适合 CloudCC 内嵌页面或独立打开的轻量页面场景。  
- 在项目根目录创建 `html` 文件夹，再建子文件夹，最后在子文件夹中创建 `index.html` 进行开发（也可用 `cloudcc create html <apiName>` 生成）。  
- 静态依赖（样式、脚本、字体等）优先使用线上绝对地址（`https://...`）引入；公有云推荐使用中国大陆可稳定访问的源（如 BootCDN）。  
- 私有云或内网环境可先下载依赖并上传为组织静态资源，再替换为静态资源 URL。  

---

## 13. 推荐前端依赖组合

推荐组合：**Tailwind CSS + Alpine.js + ECharts**。以下为各库在浏览器中直接引入的写法与顺序说明。

### 13.1 Tailwind CSS（浏览器端编译）

```html
<script src="https://cdn.bootcdn.net/ajax/libs/tailwindcss-browser/4.1.13/index.global.min.js"></script>
```

### 13.2 Alpine.js（交互层）

```html
<script defer src="https://cdn.bootcdn.net/ajax/libs/alpinejs/3.15.0/cdn.min.js"></script>
```

建议放在 `head`，并在同页组合时保持 **Tailwind 在前、Alpine 在后**。

### 13.3 ECharts（图表）

```html
<script src="https://cdn.bootcdn.net/ajax/libs/echarts/6.0.0/echarts.min.js"></script>
```

建议在 Tailwind 与 Alpine **之后**加载，并在**图表初始化脚本之前**引入。

---

## 14. CloudCC 运行环境能力（CCDK）

- 在 CloudCC 运行环境（如自定义页面、内嵌 HTML）中，可通过 `window.$CCDK` 使用平台能力。  
- 本地直接用浏览器打开 HTML 文件时通常不存在 `$CCDK`，请先做存在性判断，避免运行时报错。  
- 业务接口与平台能力细节以各模块文档为准：`cloudcc doc <module> introduction|devguide`。  

---

## 15. 最小可用模板

下面示例包含 Tailwind、Alpine 与 `$CCDK` 判断。若需图表，请在 `head` 中按 **第 13 节**顺序在 Alpine 之后增加 ECharts 的 `<script>`，再在页面底部编写初始化逻辑。

```html
<!doctype html>
<html lang="zh-CN">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>CloudCC HTML Page</title>
    <script src="https://cdn.bootcdn.net/ajax/libs/tailwindcss-browser/4.1.13/index.global.min.js"></script>
    <script defer src="https://cdn.bootcdn.net/ajax/libs/alpinejs/3.15.0/cdn.min.js"></script>
  </head>
  <body class="p-4">
    <div x-data="{ ready: true }">
      <h1 class="text-xl font-semibold">CloudCC HTML 页面</h1>
      <p class="mt-2 text-gray-600" x-show="ready">页面已初始化</p>
    </div>

    <script>
      const cc = window.$CCDK || null;
      if (!cc) {
        console.warn("$CCDK 不可用，当前可能是本地浏览器环境。");
      }
    </script>
  </body>
</html>
```
