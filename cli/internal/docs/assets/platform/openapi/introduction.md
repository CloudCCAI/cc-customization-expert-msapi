# OpenAPI SDK 简介

`cloudcc-openapi-sdk.js` 是面向浏览器场景的 CloudCC OpenAPI 调用 SDK，支持：

- 使用 CRM 账号 + 密码登录；
- 自动换取访问令牌；
- 调用对象查询等 OpenAPI 接口。

同时，CLI 侧现在也支持直接读取项目配置中的 `apiSvc` 与 `accessToken`，通过带权限接口完成数据增删改查，不需要额外调用登录接口。

该 SDK 适用于独立站点（如自建前端页面），通过在 `index.html` 中引入脚本即可使用。

## 快速开始

1. 复制 SDK 到你的站点目录：

```bash
cloudcc get openapi <你的站点目录>
```

2. 在 `index.html` 中引入依赖与 SDK：

```html
<script src="https://cdnjs.cloudflare.com/ajax/libs/crypto-js/4.2.0/crypto-js.min.js"></script>
<script src="https://cdn.jsdelivr.net/npm/jsencrypt@3.3.2/bin/jsencrypt.min.js"></script>
<script src="./cloudcc-openapi-sdk.js"></script>
```

3. 初始化并登录：

```javascript
const api = new CloudCCOpenAPI({ ClientId, SecretKey, loginsvcurl });
await api.login("crm账号", "crm密码");
```

4. 调用接口读取对象数据：

```javascript
const res = await api.cquery({
  objectApiName: "Account",
  fields: "id,name",
  pageNo: 1,
  pageSize: 20
});
```
