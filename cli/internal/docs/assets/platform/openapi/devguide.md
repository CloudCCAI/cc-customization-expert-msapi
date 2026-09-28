# OpenAPI 常规业务数据操作指南

## 能力定位

常规业务记录操作统一使用 `openapi`，包括标准 CRM 对象和自定义对象。CLI 直接调用平台
`api-svc` 的带权限接口，不走 MetadataService。

- 标准对象示例：客户、联系人、商机等，使用目标租户真实标准对象 API Name。
- 自定义对象：使用目标租户真实自定义对象 API Name，能力和命令与标准对象相同。
- `dataBulk` 只用于大批量数据导入、初始化和离线批处理，不用于日常业务操作。
- `dataIndex` 只用于数据库索引查看、创建和优化。
- 对象、字段、页面、权限和流程等平台二开配置使用对应元数据 Domain。

## 使用前检查

项目的 `cloudcc-cli.config.json` 当前环境必须能够解析出 `apiSvc` 和 `accessToken`：

```bash
cloudcc get config <projectPath>
cloudcc domain openapi
```

记录中的对象、字段和记录 ID 必须来自目标租户，不得猜测或自造。调用身份还必须具有对应对象、字段和记录权限。

## CLI 动作

```bash
cloudcc query     openapi <projectPath> <bodyJson|@file>
cloudcc pageQuery openapi <projectPath> <bodyJson|@file>
cloudcc create    openapi <projectPath> <bodyJson|@file>
cloudcc update    openapi <projectPath> <bodyJson|@file>
cloudcc delete    openapi <projectPath> <bodyJson|@file>
cloudcc upsert    openapi <projectPath> <bodyJson|@file>
```

`bodyJson` 可以是原始 JSON，也可以是 URI 编码后的 JSON；复杂内容推荐写入 JSON 文件并传 `@file`。

## 标准对象示例

```bash
# 查询客户
cloudcc query openapi . \
  '{"objectApiName":"Account","expressions":"","fields":"id,name"}'

# 分页查询客户
cloudcc pageQuery openapi . \
  '{"objectApiName":"Account","fields":"id,name","expressions":"","pageNUM":1,"pageSize":20}'

# 新增客户
cloudcc create openapi . \
  '{"objectApiName":"Account","data":[{"name":"示例客户"}]}'

# 修改客户；id 必须来自目标租户真实回读
cloudcc update openapi . \
  '{"objectApiName":"Account","data":[{"id":"<recordId>","name":"更新后的客户"}]}'

# 删除客户；id 必须来自目标租户真实回读
cloudcc delete openapi . \
  '{"objectApiName":"Account","data":[{"id":"<recordId>"}]}'

# upsert 的匹配字段和数据必须符合目标租户对象定义
cloudcc upsert openapi . @account-upsert.json
```

## 自定义对象示例

自定义对象与标准对象使用完全相同的命令，只需替换为目标租户真实对象和字段 API Name：

```json
{
  "objectApiName": "<customObjectApiName>",
  "data": [
    {
      "<fieldApiName>": "示例值"
    }
  ]
}
```

```bash
cloudcc create openapi . @custom-object-records.json
cloudcc query openapi . \
  '{"objectApiName":"<customObjectApiName>","expressions":"","fields":"id,<fieldApiName>"}'
```

## 输入和输出

- `create/update/delete/upsert` 接受 `data` 或 `Data`；单个对象和对象数组都会规范化后提交。
- 查询条件、分页字段和业务字段必须使用目标环境实际支持的 OpenAPI 请求结构。
- 成功时 stdout 输出平台原始 JSON，自动化调用必须继续检查其中的逐条业务结果。
- 失败时 CLI 返回非零退出码，并优先显示平台返回的 `returnInfo` 或 `message`。

## 浏览器 SDK

仅当独立站点需要在浏览器中调用 CloudCC OpenAPI 时复制 SDK：

```bash
cloudcc get openapi [targetDir] [outputFileName]
```

浏览器 SDK 需要独立处理登录和前端凭据保护；它不替代上面的 CLI 常规业务数据命令。
