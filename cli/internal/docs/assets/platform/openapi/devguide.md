# OpenAPI 常规业务数据操作指南

## 能力定位

常规业务记录操作统一使用 `openapi`，包括标准 CRM 对象和自定义对象。Lightning 默认直接调用平台
`api-svc` 的带权限接口，不走 MetadataService。显式 `platformMode=horizontal` 时，常规业务数据 CRUD
由 main-app 提供。

- 标准对象示例：客户、联系人、商机等，使用目标租户真实标准对象 API Name。
- 自定义对象：使用目标租户真实自定义对象 API Name，能力和命令与标准对象相同。
- 已有业务记录可以流式上传本地附件并自动完成文件绑定。
- 已有业务记录可以提交到目标租户为该对象配置的批准过程。
- `dataBulk` 只用于大批量数据导入、初始化和离线批处理，不用于日常业务操作。
- `dataIndex` 只用于数据库索引查看、创建和优化。
- 对象、字段、页面、权限和流程等平台二开配置使用对应元数据 Domain。

## 使用前检查

Lightning 项目的当前环境必须能够解析出 `apiSvc` 和 `accessToken`。横纵版项目必须显式配置
`platformMode=horizontal`、`endpoints.mainAppUrl` 和 `auth.username/password/language`：

```bash
cloudcc get config <projectPath>
cloudcc doctor platform <projectPath>
cloudcc domain openapi
```

横纵版支持 `query/pageQuery/create/update/delete/upsert openapi`。附件和审批仍然失败关闭；不要把
Lightning 的附件或审批调用方式套用到横纵版。

CLI 会自动维护横纵版认证状态，password 不写入项目缓存。query/pageQuery 在认证状态失效时会重新认证并重试一次；
写请求不会自动重放，返回结果不确定时必须先核对目标记录再决定是否重试。若目标环境要求额外认证步骤，CLI 会返回明确错误并停止请求。

记录中的对象、字段和记录 ID 必须来自目标租户，不得猜测或自造。调用身份还必须具有对应对象、字段和记录权限。

## CLI 动作

```bash
cloudcc query     openapi <projectPath> <bodyJson|@file>
cloudcc pageQuery openapi <projectPath> <bodyJson|@file>
cloudcc create    openapi <projectPath> <bodyJson|@file>
cloudcc update    openapi <projectPath> <bodyJson|@file>
cloudcc delete    openapi <projectPath> <bodyJson|@file>
cloudcc upsert    openapi <projectPath> <bodyJson|@file>
cloudcc uploadAttachment openapi <projectPath> <recordId> <filePath> [optionsJson|@file]
cloudcc submitApproval   openapi <projectPath> <bodyJson|@file>
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

## 上传业务记录附件

```bash
cloudcc uploadAttachment openapi . \
  <recordId> ./报价说明.pdf

cloudcc uploadAttachment openapi . \
  <recordId> ./source.bin \
  '{"fileName":"报价说明.pdf","sourceForm":"cloudcc-cli"}'
```

命令使用流式 multipart 请求调用 `/api/file/upload`，不会把完整文件读入内存。默认以本地文件名作为
`fileName`，也可在 options 中传 `fileName`、`groupid`、`libid`、`parentid`、`isFromEmail` 或
`sourceForm`。上传时把 `recordId` 传给平台做记录权限检查；上传成功后，CLI 用返回的 `name`、
`type`、`fileContentId`、`fileinfoid`、`filesize` 自动调用 `/api/file/bind`。

bind 会再次检查记录编辑权限和锁定状态。只有上传与绑定都成功，命令才返回成功。如果上传成功但
bind 失败，CLI 不自动删除文件，而是在错误中返回 `fileContentId` 和 `fileinfoid`，避免隐藏已发生的
平台状态变化。

## 提交业务记录审批

```bash
# 最小请求
cloudcc submitApproval openapi . \
  '{"relatedId":"<recordId>"}'

# 带意见和显式下一审批人
cloudcc submitApproval openapi . \
  '{"relatedId":"<recordId>","fprId":"<userId>","comments":"请审批","appPath":"detail"}'
```

`relatedId` 是详情页路由中的业务记录 ID，不是对象元数据 ID。可选字段只有 `fprId`、`comments` 和
`appPath`；未知字段会在本地失败，避免拼写错误被静默忽略。平台返回 `Manual` 表示流程要求手工选择
下一审批人，以目标租户真实用户 ID 填写 `fprId` 后重试。已经处于审批中、重复提交、缺少上级或没有
匹配流程等错误保持平台原始 `returnCode/returnInfo` 语义。

## 浏览器 SDK

仅当独立站点需要在浏览器中调用 CloudCC OpenAPI 时复制 SDK：

```bash
cloudcc get openapi [targetDir] [outputFileName]
```

浏览器 SDK 需要独立处理登录和前端凭据保护；它不替代上面的 CLI 常规业务数据命令。
