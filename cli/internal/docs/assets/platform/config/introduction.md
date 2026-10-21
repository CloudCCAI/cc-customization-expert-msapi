# CloudCC 项目配置介绍

`config` 管理本地项目的 `cloudcc-cli.config.json`，用于选择当前环境并查看解析后的配置。适用于在开发、测试等环境之间切换，以及排查平台类型、认证和服务地址配置。

## 能力与边界

- `cloudcc use config <env> [projectPath]` 切换当前环境。
- `cloudcc get config [projectPath]` 查看当前解析配置。
- `cloudcc doctor platform [projectPath]` 检查平台配置；`cloudcc doctor provider [projectPath]` 检查实际执行通道。
- 平台缺省为 Lightning；横纵版需要显式设置 `platformMode=horizontal`。
- 配置模块不负责创建或修改 CRM 业务资源。环境切换会改变后续命令的目标，执行写操作前应复核目标。
- 技能根目录的 `config.json` 是发行元信息，不能作为项目连接配置使用。真实凭据不应提交到 Git 或分发包。

完整参数、初始化方式和示例见 `cloudcc doc config devguide`。
