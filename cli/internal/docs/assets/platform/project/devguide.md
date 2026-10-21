# CloudCC Project 开发指南（Go 版技能）

## 1. 前置条件与总体说明

- 适用人群：CloudCC 前端组件、客户端脚本、后端类、触发器、定时器等二次开发工程师。
- Go 版技能不要求在线安装 `cloudcc-cli`，也不依赖全局 Node/npm。
- 工程根目录下必须提供 `cloudcc-cli.config.json`；Lightning 使用开发者密钥/安全标记，横纵版使用 main-app 登录配置。
- Go 版 CLI 优先执行技能内置 wrapper：`tools/bin/cloudcc`。

## 2. 验证 Go 版 CLI

在技能目录中执行：

```bash
tools/bin/cloudcc --version
tools/bin/cloudcc doc platform/config devguide
```

如果提示平台二进制缺失，先构建：

```bash
./tools/build.sh
```

构建需要本机安装 Go；普通使用不需要 Node/npm。

## 3. 项目配置

新项目推荐使用 `cloudcc create project <name|.>` 生成默认配置：

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

技能包初始化时先确认 `lightning|horizontal`，未指定时保持 Lightning。Lightning 再确认公有云或私有云：公有云使用技能根配置的默认地址 `https://dc52.apis.cloudcc.cn/metadata`；私有云需要把当前环境的 `metadataService.url` 改为本环境地址。项目级配置可在需要隔离项目环境时覆盖该配置。

横纵版必须显式初始化，不会改变默认 Lightning 模板：

```bash
cloudcc create project horizontal-demo --platform horizontal \
  --main-app-url https://tenant.example.com \
  --username user@example.com
```

生成的配置包含 `platformMode=horizontal`、`executionMode=auto`、`endpoints.mainAppUrl`、`auth.username/password/language` 和 `metadataService.url`。替换占位值后先运行 `cloudcc doctor platform`，再运行 `cloudcc doctor provider`。配置 MetadataService 后，横纵版低代码、`dataIndex` 和 `dataBulk` 可走 MSAPI；常规 OpenAPI CRUD 走 main-app。

Go 版优先支持 `cloudcc-cli.config.json`。历史 `cloudcc-cli.config.js` 不能由 Go 直接执行，建议迁移为 JSON。
`username/baseUrl/orgId/clientId/openSecretKey` 是兼容旧明文配置或 `CloudCCDev` 解析后的字段，不是新项目最小必需配置。

## 4. 常用命令

```bash
cloudcc doc platform/object introduction
cloudcc doc platform/object devguide
cloudcc get config .
cloudcc use config dev .
cloudcc query openapi . '<encodeURI(JSON.stringify(body))>'
```

在技能目录内请使用：

```bash
tools/bin/cloudcc doc platform/object introduction
```

## 5. 当前范围

- 已覆盖：P0-P3，包括 doc、config、openapi、常规 metadata 操作、classes/triggers/timer/script/html/staticResource 的基础本地文件与发布能力。
- 暂缓：P4，包括完整 plugin Vue 构建替代、JSP analyze/split 迁移、完整 MCP 工具注册。

## 6. 建议工作流

1. 先读取目标模块的 introduction 文档。
2. 再读取目标模块的 devguide 文档。
3. 确认项目配置存在并可解析。
4. 对写操作优先使用测试环境。
5. 对 P4 暂缓命令继续使用 Node 版技能或新建迁移任务。

## 用户级全局 CLI 安装

在完整技能目录执行，不要在业务项目目录运行相对的安装脚本路径。

macOS（zsh/bash）：

```bash
sh tools/install.sh
export PATH="$HOME/.local/bin:$PATH"
cloudcc --version
cloudcc doc list
```

Windows PowerShell：

```powershell
Set-ExecutionPolicy -Scope Process Bypass
& .\tools\install.ps1
cloudcc --version
cloudcc doc list
```

安装器自动选择 AMD64/ARM64：Mac 使用 `~/.local/bin` 和用户 shell 启动文件，Windows 使用 `%LOCALAPPDATA%\CloudCC\bin` 和用户 PATH。无需管理员权限。Mac 新终端自动生效，当前终端执行 export；Windows 安装会话立即生效，其他终端和应用重启后生效。

入口引用完整技能包，升级保留目录即可继续使用；移动目录后重跑安装。重复执行不重复添加 PATH，不覆盖非受管同名文件。Windows 技能路径不能包含 `%` 或换行。无需全局安装时，`tools/bin/cloudcc` 或 `tools\bin\cloudcc.cmd` 仍可自动选择随包架构。

全局安装只解决命令发现；业务操作仍读取命令指定项目或当前目录的 `cloudcc-cli.config.json`，不会自动选择租户，也不修改项目连接配置。
