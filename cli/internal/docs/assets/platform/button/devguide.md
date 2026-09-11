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
| `event`       | 按钮类型：`lightning` / `lightning-script` / `lightning-url` / `URL` |
| `behavior`    | 打开方式：`self` 当前页 / `newWindow` 新窗口 |

---

## 3. 新建自定义按钮

按钮写入使用 MetadataService 的 plan/apply 流程。先准备一个 JSON 文件，再创建计划、确认计划内容，最后 apply。

示例 `button-url.json`：

```json
{
  "objId": "202646FC67ACF24D39sG",
  "label": "打开帮助",
  "name": "open_help",
  "category": "CustomButton",
  "btnType": "detailBtn",
  "event": "URL",
  "behavior": "newWindow",
  "url": "https://example.com/help"
}
```

执行命令：

```bash
cloudcc plan msapi <projectPath> button @button-url.json create
cloudcc apply msapi <projectPath> <planId> '{"async":true}'
cloudcc operation msapi <projectPath> <applyId>
cloudcc get button <projectPath> <prefix>
```

常用字段：

| 字段 | 必填 | 说明 |
|------|------|------|
| `objId` / `objectId` | 是 | 所属对象 ID。批量创建也可以在顶层统一写 `objectId`。 |
| `label` | 是 | 按钮显示名称。 |
| `name` | 建议 | 按钮 API 名称。省略时系统会按按钮 ID 生成默认 API 名称。 |
| `category` | 否 | 自定义按钮写 `CustomButton`，省略时默认按自定义按钮创建。 |
| `btnType` | 否 | `detailBtn` 详情页按钮；`listBtn` 列表按钮。默认 `detailBtn`。 |
| `event` | 是 | 按钮类型。可选 `lightning`、`lightning-script`、`lightning-url`、`URL`。兼容别名：`template` 会按 `lightning` 入库，`url` 会按 `URL` 入库。 |
| `behavior` | 否 | `lightning-url` 常用打开方式：`self` 当前页打开；`newWindow` 新窗口打开。 |
| `functionCode` | 按类型 | PC 端内容。`lightning` / `lightning-script` 填 PC 端脚本；`lightning-url` 填 PC 端自定义页面或页面地址；`URL` 按钮建议改用 `url`。 |
| `url` | URL 按钮优先填写 | `URL` 按钮的跳转地址。`url` 与兼容字段 `functionCode` 至少填写一个；两者都填写时必须相同。计划入库时会将有效地址同步写入 `url` 和 `functionCode`。 |
| `mobileurl` | 否 | `lightning-url` 的移动端地址。选择平台自定义页面时通常是 `__UNI__110007E/#/pages/index/index?pageApi=<pageApi>`；选择自定义小程序时填写实际小程序路径。 |
| `h5FunctionCode` | 否 | `lightning-script` 的移动端脚本。 |
| `menubar` | 否 | `lightning-url` 且 `behavior` 为 `newWindow` 时可用，通常填写 `show` 或 `hidden`。 |
| `remark` / `description` | 否 | 备注说明。 |

四种按钮类型和入库字段：

| `event` | 界面显示 | 适用场景 | 主要入库字段 |
|---------|----------|----------|--------------|
| `lightning` | `template` | 仅 PC 端脚本模板按钮 | `EVENT=lightning`，`FUNCTION_CODE=<functionCode>` |
| `lightning-script` | `lightning-script` | PC + 移动端脚本按钮 | `EVENT=lightning-script`，`FUNCTION_CODE=<functionCode>`，`H5FUNCTION_CODE=<h5FunctionCode>` |
| `lightning-url` | `lightning-url` | PC + 移动端页面/链接按钮 | `EVENT=lightning-url`，`FUNCTION_CODE=<functionCode>`，`MOBILEURL=<mobileurl>`，可带 `BEHAVIOR`、`MENUBAR` |
| `URL` | `url` | 普通 URL 跳转按钮 | `EVENT=URL`，`URL=<url>`，`FUNCTION_CODE=<url>` |

> `URL` 按钮优先使用 `url`。为兼容已有 JSON，也可只传 `functionCode`；计划会把它视为 URL 地址。两者都传且不一致时，计划会失败，避免入库后回显和运行行为不一致。

`lightning-script` 按钮示例：

```json
{
  "objId": "202646FC67ACF24D39sG",
  "label": "校验合同",
  "name": "validate_contract",
  "category": "CustomButton",
  "btnType": "detailBtn",
  "event": "lightning-script",
  "behavior": "self",
  "functionCode": "alert('validate pc');",
  "h5FunctionCode": "alert('validate h5');"
}
```

`lightning-url` 按钮示例：

```json
{
  "objId": "202646FC67ACF24D39sG",
  "label": "打开移动页面",
  "name": "open_mobile_page",
  "category": "CustomButton",
  "btnType": "detailBtn",
  "event": "lightning-url",
  "behavior": "newWindow",
  "menubar": "show",
  "functionCode": "pc_page_api_or_url",
  "mobileurl": "__UNI__110007E/#/pages/index/index?pageApi=mobile_page_api"
}
```

`template` 显示类型对应的实际入库 event 是 `lightning`。CLI 可以直接写 `lightning`，也可以写别名 `template`，计划会按 `lightning` 入库：

```json
{
  "objId": "202646FC67ACF24D39sG",
  "label": "模板按钮",
  "name": "template_button",
  "category": "CustomButton",
  "btnType": "detailBtn",
  "event": "lightning",
  "functionCode": "alert('template');"
}
```

服务预约定位类按钮还可以填写 `scopeon`、`radius`、`baseaddress`、`uploadphoto`、`uploadfromalbum`、`restrictionType` 等字段。

---

### 3.1 批量创建自定义按钮

一次要在同一个对象下创建多个自定义按钮时，文件顶层写 `objectId`、`objectApiName` 或 `objectPrefix` 指定目标对象，并在 `buttons[]` 中写每个按钮。`buttons[]` 内每一项的字段与单个按钮创建一致。批量创建是对象级能力：同一个文件只能作用于一个对象，数组项不能覆盖到其它对象，也不能和其它 domain 混在同一次计划里提交。

示例 `buttons-batch.json`：

```json
{
  "objectId": "202646FC67ACF24D39sG",
  "onExisting": "createOnly",
  "buttons": [
    {
      "id": "btn_contract_submit",
      "label": "提交合同",
      "name": "submit_contract",
      "category": "CustomButton",
      "btnType": "detailBtn",
      "event": "lightning-script",
      "behavior": "self",
      "functionCode": "alert('submit');"
    },
    {
      "id": "btn_contract_open_help",
      "label": "查看指引",
      "name": "open_contract_help",
      "category": "CustomButton",
      "btnType": "listBtn",
      "event": "URL",
      "behavior": "newWindow",
      "url": "https://example.com/help"
    }
  ]
}
```

执行命令：

```bash
cloudcc plan msapi <projectPath> buttons @buttons-batch.json create
cloudcc apply msapi <projectPath> <planId> '{"async":true}'
cloudcc operation msapi <projectPath> <applyId>
cloudcc get button <projectPath> <prefix>
```

`button` / `buttons` 都可作为 `plan msapi` 的 domain 参数。批量计划会逐项检查同批重复、目标对象已有同 ID / API 名 / 名称 / 标签按钮、以及数组项是否声明了其它对象。批量创建只处理自定义按钮；`category` 为 `StandardButton` 的项会标记为 `FAILED_PRECHECK`，标准按钮请使用对应的更新或配置能力。批量项里的 `event` 和字段入库规则与单个按钮完全一致。

`onExisting` 支持：

| 策略 | 行为 |
|------|------|
| `createOnly` | 默认策略；目标已存在时该项标记为 `FAILED_PRECHECK`，其它无关项继续生成步骤。 |
| `skipExisting` | 目标已存在时跳过该项，plan metadata 记录为 `SKIPPED`。 |

调用方应读取 plan metadata 中的 `batchItemResults`、`batchExecutableCount`、`batchPrecheckFailedCount`。`batchItemResults[].status` 可能是 `PLANNED`、`SKIPPED` 或 `FAILED_PRECHECK`；预检失败项不会生成 SQL 步骤。如果整批都没有可执行项，`apply` 会失败，避免提交空计划。

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
