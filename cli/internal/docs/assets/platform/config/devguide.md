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
    "safetyMark": "请设置一个安全标识",
    "CloudCCDev": "请设置开发者密钥"
  }
}
```

`username/baseUrl/orgId/clientId/openSecretKey` 是兼容旧明文配置或 `CloudCCDev` 解析后的字段，不应作为新项目最小必需配置展示。

## 4. 命令总览

```bash
cloudcc use config <env> [projectPath]
cloudcc get config [projectPath]
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

命令会解析当前环境配置，并在需要时从 `CloudCCDev` 补齐 `apiSvc`、`setupSvc`、`accessToken`、`secretKey`、`pluginToken` 等字段。

## 7. MetadataService/MSAPI 地址

真实调用 MetadataService/MSAPI HTTP 接口前，CLI 会按以下顺序解析地址：

1. 环境变量 `CLOUDCC_METADATA_SERVICE_URL`。
2. 当前环境配置里的 `metadataService.url`、`metadataServiceUrl` 或 `metadata_service_url`。
3. 交互式运行时询问用户地址，并写回当前环境的 `metadataService.url`。

非交互运行不会静默使用 localhost；如果没有配置地址，会直接报错并提示设置 `CLOUDCC_METADATA_SERVICE_URL` 或更新 `cloudcc-cli.config.json`。

推荐把地址与开发者配置放在同一个环境下：

```json
{
  "use": "dev",
  "dev": {
    "safetyMark": "请设置一个安全标识",
    "CloudCCDev": "请设置开发者密钥",
    "metadataService": {
      "url": "http://127.0.0.1:8087"
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
