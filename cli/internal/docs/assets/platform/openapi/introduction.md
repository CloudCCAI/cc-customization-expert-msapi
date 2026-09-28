# OpenAPI：常规业务数据操作

`openapi` 是 CloudCC CLI 查询、新增、修改、删除和 upsert 常规业务数据的一级 Domain。
它既适用于客户、联系人、商机等标准 CRM 对象，也适用于租户中的自定义对象；调用时统一使用目标对象和字段的 API Name。

## 如何选择

| 用户需求 | 使用能力 |
|---|---|
| 查询、新增、修改、删除或 upsert 日常业务记录 | `openapi` |
| 大批量数据导入、初始化或离线批处理 | `dataBulk` |
| 查看、创建或优化数据库索引 | `dataIndex` |
| 配置对象、字段、页面、权限或流程等平台二开能力 | 对应元数据 Domain |

不要因为数据行数超过一条就改用 `dataBulk`。只要属于正常业务操作，就使用 `openapi`；
`dataBulk` 面向大批量数据导入，不适用于日常业务场景。

## 快速开始

```bash
# 查看 Domain 和完整文档
cloudcc domain openapi
cloudcc doc platform/openapi devguide

# 新增标准对象业务数据
cloudcc create openapi <projectPath> \
  '{"objectApiName":"Account","data":[{"name":"示例客户"}]}'

# 新增自定义对象业务数据；替换为目标租户真实对象和字段 API Name
cloudcc create openapi <projectPath> \
  '{"objectApiName":"<customObjectApiName>","data":[{"<fieldApiName>":"示例值"}]}'
```

OpenAPI 在 MSAPI、UIAPI 和 Universal 三种技能包中都可用，直接调用平台 `api-svc`，不经过 MetadataService。
执行前确认项目配置能够解析出 `apiSvc` 和 `accessToken`：

```bash
cloudcc get config <projectPath>
```

浏览器站点需要复制 OpenAPI SDK 时仍可使用 `cloudcc get openapi [targetDir] [outputFileName]`；
这是前端集成辅助能力，不是 CLI 常规业务数据 CRUD 的主入口。
