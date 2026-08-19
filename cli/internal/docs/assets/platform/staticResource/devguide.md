# 静态资源（Static Resource）操作指南

直接查看：`cloudcc doc platform/staticResource`

---

## 1. 常见操作

- 查询静态资源列表
- 查看静态资源详情
- 新建/编辑静态资源（上传）
- 删除静态资源
- 查询静态资源容量（上传前校验）

---

## 2. 常用 CLI 命令

### 1) 查看静态资源指导文档

命令：

```bash
cloudcc doc platform/staticResource
```

说明：输出静态资源完整操作文档，并自动附带本文件内容。

### 2) 查询静态资源列表

命令：

```bash
cloudcc get staticResource <projectPath> [encodedCondJson]
```

参数说明：

| 参数名称 | 类型 | 是否必须 | 默认值 | 备注 |
| :--- | :--- | :--- | :--- | :--- |
| projectPath | string | 非必须 | 当前目录 | 项目根目录 |
| encodedCondJson | string | 非必须 | — | `encodeURI(JSON.stringify(body))` 后的查询条件 |

请求体（decoded 后）常用字段：

| 参数名称 | 类型 | 是否必须 | 默认值 | 备注 |
| :--- | :--- | :--- | :--- | :--- |
| label | string | 非必须 | `""` | 可用于按标签/名称筛选 |

### 3) 查询静态资源详情

命令：

```bash
cloudcc detail staticResource <projectPath> <resourceId>
```

参数说明：

| 参数名称 | 类型 | 是否必须 | 默认值 | 备注 |
| :--- | :--- | :--- | :--- | :--- |
| projectPath | string | 非必须 | 当前目录 | 项目根目录 |
| resourceId | string | 必须 | — | 静态资源 id |

### 4) 新建/编辑静态资源（上传）

命令：

```bash
cloudcc create staticResource <projectPath> <encodedBodyJson>
```

参数说明：

| 参数名称 | 类型 | 是否必须 | 默认值 | 备注 |
| :--- | :--- | :--- | :--- | :--- |
| projectPath | string | 非必须 | 当前目录 | 项目根目录 |
| encodedBodyJson | string | 必须 | — | `encodeURI(JSON.stringify(body))` |

请求体（decoded 后）字段：

| 参数名称 | 类型 | 是否必须 | 默认值 | 备注 |
| :--- | :--- | :--- | :--- | :--- |
| filePath | string | 视场景 | — | 新建必填；编辑不换文件可不传 |
| label | string | 必须 | — | 标签/名称 |
| desc | string | 非必须 | `""` | 描述 |
| id | string | 仅编辑 | `""` | 编辑时传静态资源 id |
| name | string | 非必须 | 文件名 | 不传则从 `filePath` 推导 |
| type | string | 非必须 | 文件后缀 | 不传则从文件名推导 |

### 5) 删除静态资源

命令：

```bash
cloudcc delete staticResource <projectPath> <resourceId>
```

参数说明：

| 参数名称 | 类型 | 是否必须 | 默认值 | 备注 |
| :--- | :--- | :--- | :--- | :--- |
| projectPath | string | 非必须 | 当前目录 | 项目根目录 |
| resourceId | string | 必须 | — | 静态资源 id |

### 6) 统计静态资源容量（上传前校验）

命令：

```bash
cloudcc count staticResource <projectPath> [encodedCondJson]
```

参数说明：

| 参数名称 | 类型 | 是否必须 | 默认值 | 备注 |
| :--- | :--- | :--- | :--- | :--- |
| projectPath | string | 非必须 | 当前目录 | 项目根目录 |
| encodedCondJson | string | 非必须 | — | 当前实现通常不传，传值时同样为 `encodeURI(JSON.stringify(body))` |

---

## 3. 示例

### 按 label 查询列表

```bash
cloudcc get staticResource . '{"label":"logo"}'
```

### 查询容量统计

```bash
cloudcc count staticResource .
```

### 新建静态资源（上传文件）

```bash
cloudcc create staticResource . '{"filePath":"./assets/logo.png","label":"logo_png","desc":"cli upload"}'
```

### 编辑静态资源（不替换文件）

```bash
cloudcc create staticResource . '{"id":"69c4f469ec9eb4d5c8a1b76d","label":"logo_png_v2","desc":"rename only"}'
```

---

## 4. Checklist

- [ ] 已在正确项目目录执行命令（或显式传入 `projectPath`）
- [ ] 列表查询条件已按原始 JSON 或 `@file.json` 传入
- [ ] 删除前已确认资源未被页面/脚本/组件引用
- [ ] 上传前已先执行容量统计
