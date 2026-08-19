# CloudCC 按钮与链接操作指南

---

## 1. 入口与列表页

进入路径：`对象管理 → 选择对象 → 按钮和链接` 标签页

列表页分为两个 Tab：
- **标准按钮和链接**：系统内置，不可删除
- **自定义按钮和链接**：开发者自定义

列表展示字段：标签 / 名称 / 显示类型

---

## 2. 查询按钮列表

```bash
cloudcc get button . <prefix>
```

- `<prefix>`：对象的 prefix，可从对象管理页或对象 API 名称前缀获取（如 `a75`）

返回结构：

```json
{
  "objname": "玫瑰花不带刺",
  "objid": "202646FC67ACF24D39sG",
  "standbutton": [
    {
      "id": "adc20262A1F4957UmT0l",
      "label": "提交待审批",
      "name": "Submit",
      "btnType": "detailBtn",
      "category": "StandardButton",
      "event": "URL",
      "behavior": "self",
      "url": null,
      "functionCode": null,
      "mobileurl": ""
    }
  ],
  "custbutton": []
}
```

| 字段          | 说明                                        |
|---------------|---------------------------------------------|
| `standbutton` | 标准按钮列表                                |
| `custbutton`  | 自定义按钮列表                              |
| `btnType`     | `detailBtn` 详情页按钮 / `listBtn` 列表按钮 |
| `category`    | `StandardButton` 标准 / `CustomButton` 自定义 |
| `event`       | `URL` 跳转 / `JavaScript` 执行脚本         |
| `behavior`    | 打开方式：`self` 当前页 / `newWindow` 新窗口 |

---

## 3. 新建自定义按钮

```bash
cloudcc create button . <objid> <label> [name] [btnType] [event]
```

| 参数        | 必填 | 说明                                                                        |
|-------------|------|-----------------------------------------------------------------------------|
| `objid`     | ✅   | 所属对象 ID                                                                 |
| `label`     | ✅   | 按钮标签（显示名称）                                                        |
| `name`      | ❌   | 按钮 API 名称，默认同 `label`                                               |
| `btnType`   | ❌   | 显示类型：`detailBtn`（详情页按钮，默认）/ `listBtn`（列表按钮）            |
| `event`     | ❌   | 按钮类型，见下表，默认 `template`                                           |

### 按钮类型（event）

| 取值              | 平台支持         | 说明                                                      |
|-------------------|------------------|-----------------------------------------------------------|
| `template`        | 仅 PC            | PC 端 JavaScript 代码，默认代码 `alert('hello world')`    |
| `lightning-script`| PC + 移动端      | PC 和移动端各需要 JS 代码，默认均为 `alert('hello world')` |
| `lightning-url`   | PC + 移动端      | PC 端填 URL，移动端自动选取自定义页面组件列表第一项       |
| `url`             | 仅 PC            | PC 端 URL 地址，默认 `www.cloudcc.com`                    |

### 示例

```bash
# 新建 template 类型按钮（默认）
cloudcc create button . 202646FC67ACF24D39sG "我的按钮"

# 指定名称和显示类型
cloudcc create button . 202646FC67ACF24D39sG "我的按钮" myBtn detailBtn

# lightning-script 类型，列表按钮
cloudcc create button . 202646FC67ACF24D39sG "脚本按钮" scriptBtn listBtn lightning-script

# lightning-url 类型（移动端自动取第一个自定义页面组件）
cloudcc create button . 202646FC67ACF24D39sG "跳转按钮" urlBtn detailBtn lightning-url

# url 类型
cloudcc create button . 202646FC67ACF24D39sG "外链按钮" linkBtn detailBtn url
```

> **注意：** 各类型代码/URL 默认值仅供快速创建使用，创建后请在页面上编辑替换为实际内容。

---

## 4. 删除自定义按钮

```bash
cloudcc delete button . <id>
```

| 参数  | 必填 | 说明               |
|-------|------|--------------------|
| `id`  | ✅   | 自定义按钮 ID      |

> **注意：** 仅支持删除自定义按钮，标准按钮不可删除。

### 示例

```bash
cloudcc delete button . adc202642BBED6BFlu6D
```

---

## 5. 查看文档

```bash
cloudcc doc platform/button introduction   # 能力与适用场景说明
cloudcc doc platform/button devguide        # 本操作指南
```

---
