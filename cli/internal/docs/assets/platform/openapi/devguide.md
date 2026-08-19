# OpenAPI SDK 开发指南

## 命令说明

### 复制 SDK 到站点目录

```bash
cloudcc get openapi [targetDir] [outputFileName]
```

- `targetDir`：可选，目标目录；默认当前目录。
- `outputFileName`：可选，输出文件名；默认 `cloudcc-openapi-sdk.js`。

示例：

```bash
# 复制到当前目录
cloudcc get openapi

# 复制到指定目录
cloudcc get openapi ./public/assets/js

# 复制到指定目录并重命名
cloudcc get openapi ./public/assets/js cc-openapi.js
```

### 直接调用带权限 OpenAPI CRUD

以下命令直接读取：

- `let config = await getPackageJson(projectPath)` 解析出的 `accessToken`
- `let config = await getPackageJson(projectPath)` 解析出的 `apiSvc`

不会调用登录接口。

```bash
cloudcc query openapi [projectPath] <encodedBodyJson>
cloudcc pageQuery openapi [projectPath] <encodedBodyJson>
cloudcc create openapi [projectPath] <encodedBodyJson>
cloudcc update openapi [projectPath] <encodedBodyJson>
cloudcc delete openapi [projectPath] <encodedBodyJson>
cloudcc upsert openapi [projectPath] <encodedBodyJson>
```

参数说明：

- `projectPath`：可选，项目路径；默认当前目录。
- `encodedBodyJson`：必填，使用 `encodeURI(JSON.stringify(body))` 编码后的 JSON 字符串。

服务映射：

- `query` -> `cqueryWithRoleRight`
- `pageQuery` -> `pageQueryWithRoleRight`
- `create` -> `insertWithRoleRight`
- `update` -> `updateWithRoleRight`
- `delete` -> `deleteWithRoleRight`
- `upsert` -> `upsertWithRoleRight`

示例：

```bash
cloudcc query openapi . '{"objectApiName":"Account","expressions":"","fields":"id,name"}'

cloudcc pageQuery openapi . '{"objectApiName":"Account","fields":"id,name","expressions":"","pageNUM":1,"pageSize":20}'

cloudcc create openapi . '{"objectApiName":"Account","data":[{"name":"demo account"}]}'

cloudcc update openapi . '{"objectApiName":"Account","data":[{"id":"001xxxxxxxxxxxx","name":"demo account 2"}]}'

cloudcc delete openapi . '{"objectApiName":"Account","data":[{"id":"001xxxxxxxxxxxx"}]}'
```

说明：

- `create` 会兼容传入 `data` 或 `Data`，最终按官方文档使用 `Data` 提交。
- `create/update/delete/upsert` 若传入单对象，会自动包装成数组字符串；传数组则原样序列化。
- 成功时 stdout 输出原始 OpenAPI 返回 JSON，便于上层脚本继续处理。

## 页面集成步骤

1. 在 `index.html` 按顺序引入依赖与 SDK：

```html
<script src="https://cdnjs.cloudflare.com/ajax/libs/crypto-js/4.2.0/crypto-js.min.js"></script>
<script src="https://cdn.jsdelivr.net/npm/jsencrypt@3.3.2/bin/jsencrypt.min.js"></script>
<script src="/assets/js/cloudcc-openapi-sdk.js"></script>
```

2. 创建实例并登录：

```javascript
const api = new CloudCCOpenAPI({
  ClientId: "你的ClientId",
  SecretKey: "你的SecretKey",
  loginsvcurl: "https://global.apis.cloudcc.cn/login"
});

await api.login("crm账号", "crm密码");
```

3. 调用对象接口（示例为查询）：

```javascript
const result = await api.cquery({
  objectApiName: "Account",
  fields: "id,name,createdOn",
  filter: "",
  pageNo: 1,
  pageSize: 20
});
```

## 注意事项

- 该 SDK 运行于浏览器环境，请确保页面可访问 `CryptoJS` 与 `JSEncrypt`。
- 不建议在公开站点暴露高权限账号；请按业务场景配置最小权限账号。
- 若你已有构建系统（Vite/Webpack），也可将 SDK 作为静态资源进行托管后再按 URL 引入。
