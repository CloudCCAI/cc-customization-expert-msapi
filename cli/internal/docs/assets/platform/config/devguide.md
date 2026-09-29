# CloudCC 配置模块开发指南（Go 版）

## 1. 模块定位

`config` 模块用于管理本地项目配置文件 `cloudcc-cli.config.json` 的环境切换与查看。

当前支持：

- 切换当前环境：`cloudcc use config <env> [projectPath]`
- 查看当前解析配置：`cloudcc get config [projectPath]`
- 查看开发文档：`cloudcc doc platform/config devguide`

Go 版不执行历史 `cloudcc-cli.config.js`，请迁移为 JSON。

## 2. 开发前准备

执行命令前请确认：

- 已完成 `cloudcc doc platform/project devguide` 的初始化流程。
- 项目根目录存在 `cloudcc-cli.config.json`。
- `cloudcc-cli.config.json` 中包含当前环境配置，默认环境名为 `dev`。

## 3. 配置示例

新项目默认使用 `cloudcc create project <name|.>` 生成以下配置：

```json
{
  "use": "dev",
  "dev": {
    "executionMode": "msapi",
    "safetyMark": "请设置一个安全标识",
    "CloudCCDev": "请设置开发者密钥",
    "metadataService": {
      "url": "https://dc52.apis.cloudcc.cn/metadata"
    }
  }
}
```

技能初始化时先询问平台类型，未指定时保持 Lightning。Lightning 再询问环境类型：

- 选择公有云或直接回车时，自动写入默认 MetadataService 地址 `https://dc52.apis.cloudcc.cn/metadata`，无需再输入 URL。
- 选择私有云时，继续提示输入私有云 MetadataService 地址，并写入技能根配置当前环境的 `metadataService.url`。

`username/baseUrl/orgId/clientId/openSecretKey` 是兼容旧明文配置或 `CloudCCDev` 解析后的字段，不应作为新项目最小必需配置展示。

### 横纵版配置

未配置 `platformMode` 时严格保持现有 Lightning 行为。只有显式设置为 `horizontal` 才启用横纵版客户端：

```bash
cloudcc create project horizontal-demo --platform horizontal \
  --main-app-url https://tenant.example.com \
  --username user@example.com
```

```json
{
  "use": "dev",
  "dev": {
    "platformMode": "horizontal",
    "executionMode": "auto",
    "endpoints": {
      "mainAppUrl": "https://tenant.example.com"
    },
    "auth": {
      "username": "user@example.com",
      "password": "CLOUDCC_PASSWORD",
      "language": "zh"
    },
    "metadataService": {
      "url": "https://metadata.example.com"
    }
  }
}
```

- `platformMode` 的空值或缺省值等价于 `lightning`；未知非空值直接报错。
- `executionMode=auto` 在横纵版首期明确选择 UIAPI，不探测 Lightning MetadataService；显式 `msapi` 失败关闭。
- 横纵版首期只开放 `doctor platform` 和 `query openapi`；写操作失败关闭。
- password 不写入项目缓存；配置、日志和命令输出不得打印真实 password、token 或 Cookie。
- 只读 `query openapi` 在认证状态失效时会重新认证并重试一次；横纵版写操作不采用自动重试。

## 4. 命令总览

```bash
cloudcc use config <env> [projectPath]
cloudcc get config [projectPath]
cloudcc doctor platform [projectPath]
cloudcc doctor provider [projectPath]
cloudcc doc platform/config devguide
```

参数约定：

- `env`：目标环境名称。
- `projectPath`：项目路径，不传时默认当前目录。

## 5. 切换环境

```bash
cloudcc use config dev .
```

命令会读取 `<projectPath>/cloudcc-cli.config.json`，将顶层 `use` 字段更新为目标环境。

## 6. 查看配置

```bash
cloudcc get config .
```

Lightning 命令会解析当前环境配置，并在需要时从 `CloudCCDev` 补齐 `apiSvc`、`setupSvc`、`accessToken`、`secretKey`、`pluginToken` 等字段。横纵版不会执行这套 Lightning 解析，输出中的敏感认证字段会脱敏；附带兼容性报告会把仅适用于 Lightning 的检查标为 `not_applicable`，且不会探测 setup-svc。

## 7. MetadataService/MSAPI 地址

真实调用 MetadataService/MSAPI HTTP 接口前，CLI 会按以下顺序解析地址：

1. 环境变量 `CLOUDCC_METADATA_SERVICE_URL`。
2. 当前环境配置里的 `metadataService.url`、`metadataServiceUrl` 或 `metadata_service_url`。
3. 交互式运行时询问用户地址，并写回当前环境的 `metadataService.url`。

技能包根目录默认已经包含公有云 MetadataService 地址。非交互运行不会静默使用 localhost；如果没有配置地址，会直接报错并提示设置 `CLOUDCC_METADATA_SERVICE_URL` 或更新 `cloudcc-cli.config.json`。

推荐把地址与开发者配置放在同一个环境下：

```json
{
  "use": "dev",
  "dev": {
    "safetyMark": "请设置一个安全标识",
    "CloudCCDev": "请设置开发者密钥",
    "metadataService": {
      "url": "https://dc52.apis.cloudcc.cn/metadata"
    }
  }
}
```

## 8. 常见注意事项

- 执行路径错误会导致找不到 `cloudcc-cli.config.json`。
- `env` 必须是配置中可识别的环境名称。
- `CloudCCDev` 当前按 base64 JSON 解码；其他历史加密格式需要单独迁移。
- MetadataService 请求如果返回 `401 invalid_token`，CLI 会自动清理当前 `safetyMark` 对应的 `.cloudcc-cache.json` 缓存项，重新从项目配置解析 token 并重试一次；若刷新后的 token 仍被拒绝，CLI 会再次清理缓存，避免下次继续复用坏 token。
- 若只需要离线读取 `doc` 文档，不需要项目配置。
