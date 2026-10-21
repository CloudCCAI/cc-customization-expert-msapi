# CloudCC 单点登录开发与使用指南

## 1. 操作前准备

`singleSignOn` 对应 `single-sign-ons` Domain，用于 CloudCC SP 侧 SAML 配置。先运行 `cloudcc doctor provider <projectPath>` 检查目标环境。以下写入流程使用 MSAPI；UIAPI 根据已适配接口直接提交，不提供 MetadataService 的计划账本和回滚保证。

## 2. 查询与详情

```bash
cloudcc get singleSignOn <projectPath>
cloudcc get singleSignOn <projectPath> <id-or-selector>
cloudcc detail singleSignOn <projectPath> <id>
```

MSAPI 列表参数按 selector 处理。修改或删除前先查询真实 ID，不要生成 ID 来替代线上资源。UIAPI 列表过滤可使用目标 setup-service 接受的 JSON，例如 `{"start":0,"limit":30,"keyword":"企业"}`。

## 3. 创建

保存下面结构为 `sso.json`，将示例值替换为身份系统提供的真实配置；创建时省略 `id`。

```json
{
  "name": "企业 SAML 登录",
  "issuer": "https://identity.example.com/issuer",
  "entityId": "https://crm.example.com/saml",
  "loginUrl": "https://identity.example.com/login",
  "logoutUrl": "https://identity.example.com/logout",
  "enableLogout": false,
  "certBase64": "<已验证的证书内容>",
  "certName": "enterprise-saml.cer"
}
```

```bash
cloudcc create singleSignOn <projectPath> @sso.json
# 审阅返回的计划、警告和影响，再执行：
cloudcc apply msapi <projectPath> <planId>
cloudcc detail singleSignOn <projectPath> <实际返回的资源ID>
```

| 参数 | 说明 |
|---|---|
| `id` | 更新或删除时使用真实回读的配置 ID |
| `name` | 配置显示名称 |
| `issuer` / `idpissuer` | 身份提供方标识 |
| `entityId` / `entityid` | SAML 实体标识 |
| `loginUrl` / `loginurl` | 身份提供方登录地址 |
| `logoutUrl` / `logouturl` | 退出地址 |
| `enableLogout` / `enablelogout` | 单点退出开关，支持布尔值 |
| `certBase64` / `cert` / `certificate` | 已验证的证书内容 |
| `certName` / `certname` | 证书名称 |
| `certDetailInfo` | 已验证的证书详情；以目标环境证书校验结果为准 |
| `identityType` / `identitySource` / `customAttributeName` | 身份类型、身份来源和自定义属性映射，值取自目标平台支持的选项 |

请求绑定固定为 `HTTP-Redirect`。证书内容和详情不能使用示例占位值执行；MetadataService 计划不能替代真实 X.509 证书验证或登录验证。

## 4. 更新与证书维护

```bash
cloudcc update singleSignOn <projectPath> @sso-update.json
cloudcc apply msapi <projectPath> <planId>
cloudcc detail singleSignOn <projectPath> <id>
```

更新 JSON 必须包含真实 `id`。可更新名称、地址和身份映射；未提供新的证书内容时，更新保留已有证书及证书名称。更新证书应提交新的证书内容、名称及已验证详情，再检查实际登录及退出链路。

## 5. 删除、变更与回滚

```bash
cloudcc delete singleSignOn <projectPath> <id>
cloudcc apply msapi <projectPath> <planId>
cloudcc changes msapi <projectPath> <operationId>
cloudcc rollback msapi <projectPath> <operationId>
```

删除会影响对应 SAML 登录入口，先核对业务依赖。回滚以服务端返回的计划和可执行范围为准，执行后回读配置并验证登录行为；不能只凭配置记录恢复就判定登录成功。

## 6. 文档入口

```bash
cloudcc doc list
cloudcc doc singleSignOn introduction
cloudcc doc single-sign-ons devguide
```

完整路径 `cloudcc doc platform/singleSignOn devguide` 继续可用。
