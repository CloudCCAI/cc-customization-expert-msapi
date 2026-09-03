# CloudCC 接口注册器开发指南

从 CLI/技能 `2.2.33` 开始，接口注册器元数据 CRUD 要求目标 MetadataService 至少为 `1.1.41`；接口注册器调试、日志和高代码远程调用调整建议目标 setup-svc 至少为 `19.7.R8`。setup-svc 可能存在 `19.7.DEV1`、`19.7.B1`、`19.7.G1` 或客户分支版本，CLI 只在初始化/doctor/config 检查中提醒，不按版本字符串直接阻断；若目标环境实际缺少接口能力，运行态请求会返回 setup-svc 原始错误。

## 管理与 CLI

低代码配置写入走 MetadataService：

```bash
cloudcc create apiRegistrar <projectPath> @api-registrar.json
cloudcc update apiRegistrar <projectPath> @api-registrar.json
cloudcc delete apiRegistrar <projectPath> <id-apiCode-or-apiName>
cloudcc getList apiRegistrar <projectPath> [searchKeyword|@query.json]
cloudcc detail apiRegistrar <projectPath> <id-apiCode-or-apiName>
```

列表查询 JSON 支持 `searchKeyword`、`status`、`page`、`pageSize`；`status` 使用 `DRAFT`、`ACTIVE` 或 `INACTIVE`。

创建 spec 允许指定全表唯一的 `apiCode`：

```json
{
  "name": "ERP 订单同步",
  "apiUrl": "https://erp.example.com/api/orders",
  "apiDescription": "发送已审核订单",
  "apiCode": "ERPOrderSync01"
}
```

用户指定的 `apiCode` 只能包含 `A-Z`、`a-z` 和 `0-9`，首尾及中间都不能出现空格、连字符、下划线或其他特殊字符。只有省略 `apiCode` 字段时，MetadataService 才使用 `API-yyyyMMdd-` 前缀和 128-bit 随机数的 Base62 编码生成；显式空值或纯空格会被拒绝。数据库唯一键负责最终并发唯一性。`apiCode` 创建后不可修改；删除注册项会释放原值，之后可以复用。

调试和日志走 setup-svc 实时通道：

```bash
cloudcc debug apiRegistrar <projectPath> @debug.json
cloudcc logs apiRegistrar <projectPath> @query.json
cloudcc logDetail apiRegistrar <projectPath> @detail.json
```

CLI 输出 `debug`、`logs` 和 `logDetail` 结果前会递归脱敏常见敏感字段和值，包括 `Authorization`、`Cookie`、`accessToken`、`token`、`secret`、`password`、`apiKey` 等字段，以及字符串中的 `Bearer ...`、`access_token=...`、`token=...` 等片段。脱敏只作用于 CLI 返回给调用方的显示数据，不改变 setup-svc 的实际执行和平台原始日志存储。

真实端点均为 `POST`：

| 操作 | 端点 |
| --- | --- |
| 创建 | `/api/register/register` |
| 列表 | `/api/register/list` |
| 详情 | `/api/register/detail` |
| 更新 | `/api/register/update` |
| 删除 | `/api/register/delete` |
| 调试 | `/api/register/debug` |
| 日志 | `/api/register/logs` |
| 日志详情 | `/api/register/logDetail` |

调试请求使用 `apiCode`、`method`、`requestHeaders`、`requestBody` 和 `contentType`。平台支持 GET/POST，以及 JSON、FORM、MULTIPART；上传文件内容必须是 `byte[]`，并可通过 `fileName` 指定文件名。

## 高代码调用

```java
Map<String, Object> body = new HashMap<String, Object>();
Map<String, String> headers = new HashMap<String, String>();
body.put("orderNo", orderNo);

CCRemoteClient client = new CCRemoteClient(userInfo, "ERPOrderSync01");
CCRemoteResult result = client.execute(
    CCRemoteClient.HttpMethod.POST,
    body,
    headers,
    CCRemoteClient.ContentType.JSON
);

if (!result.isResult()) {
    throw new RuntimeException("Remote call failed: " + result.getCode() + " " + result.getMessage());
}
int status = result.getHttpStatus();
Object data = result.getData();
String traceId = result.getTraceId();
```

默认连接超时 3 秒、读取超时 30 秒。可用四参数构造器设置更短超时，但不能超过系统配置。具体 getter 名称应以目标租户 SDK 编译结果为准；若 SDK 只暴露字段或不同 getter，不得凭空编造方法，需按 validate 诊断调整。

## 设计规则

- 源码只保存 `apiCode`；URL 变更只改注册器，不改业务类。
- 生成源码前先确认注册项已调试成功且为 `ACTIVE`。`DRAFT` 或 `INACTIVE` 必须作为配置阻断，而不是靠运行时反复重试。
- 检查 `result`、`code`、`message`、`httpStatus`，业务需要时记录 `traceId`；不能仅因 execute 返回对象就判定业务成功。
- 重试只用于明确可重试且具备幂等键的请求；不要在触发器保存事务中无界重试。
- 触发器只做入口，远程调用封装在自定义类。非强一致集成优先在 commit 后、定时类或外部异步链路执行，避免外部超时阻塞记录保存。
- 定时同步必须分页、记录业务游标/幂等键，并区分 HTTP 成功、远端业务失败和本地落库失败。
- 业务代码可以按外部系统要求传递必要的鉴权 header/body/query 参数，但不得在自定义类、触发器、定时类的业务日志、异常消息或调试输出中主动打印完整 Token、Cookie、Secret、Password、API Key 等敏感值；需要排查时优先记录 `traceId`、业务单号、HTTP 状态和脱敏后的错误摘要。接口注册器只管理 URL 和运行态调用，不等同于凭据保险库。
