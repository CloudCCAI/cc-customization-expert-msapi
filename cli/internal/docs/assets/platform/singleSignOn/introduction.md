# CloudCC 单点登录介绍

`singleSignOn` 管理 CloudCC 作为服务提供商（SP）时使用的 SAML 单点登录配置，包括身份提供方标识、实体标识、登录/退出地址和证书。适用于连接企业身份系统、维护登录配置及更新证书。

## 支持能力

支持查询列表、读取详情、创建、更新和删除。MSAPI 使用 `single-sign-ons` Domain，写入先生成计划，审核后显式 apply，并可查询变更记录和请求回滚。

这与 `identityProvider` 模块所管理的另一侧身份提供商配置不同，不能混用参数。单点登录配置保存成功也不等于真实登录链路已验证；证书、身份字段映射和目标环境的登录行为仍需验证。

先运行 `cloudcc doc list` 查找文档，再使用 `cloudcc doc singleSignOn devguide` 查看命令、参数、证书更新规则和执行流程。
