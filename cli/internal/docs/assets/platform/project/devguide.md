# CloudCC Project 开发指南（Go 版技能）

## 1. 前置条件与总体说明

- 适用人群：CloudCC 前端组件、客户端脚本、后端类、触发器、定时器等二次开发工程师。
- Go 版技能不要求在线安装 `cloudcc-cli`，也不依赖全局 Node/npm。
- 工程根目录下必须提供 `cloudcc-cli.config.json`，用于配置开发者密钥、安全标记及当前环境 `use`。
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
    "safetyMark": "请设置一个安全标识",
    "CloudCCDev": "请设置开发者密钥"
  }
}
```

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
