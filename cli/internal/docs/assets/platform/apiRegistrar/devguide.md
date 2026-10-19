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

以上运行态命令只适用于 Lightning。横纵版接口注册器仅支持 MetadataService CRUD；由于 main-app 没有
对外管理控制器，且调试日志的权威数据位于按租户路由的 Mongo，横纵版不支持 `debug/logs/logDetail`。

CLI 输出 `debug`、`logs` 和 `logDetail` 结果前会递归脱敏常见敏感字段和值，包括 `Authorization`、`Cookie`、`accessToken`、`token`、`secret`、`password`、`apiKey` 等字段，以及字符串中的 `Bearer ...`、`access_token=...`、`token=...` 等片段。脱敏只作用于 CLI 返回给调用方的显示数据，不改变 setup-svc 的实际执行和平台原始日志存储。

从技能 `2.2.92` 开始，脱敏还覆盖 SOAP/XML 中带或不带命名空间的敏感元素及敏感属性，例如
`<wsse:BinarySecurityToken>`、`<Password>`、`clientSecret="..."`。这仍然只是 CLI 显示保护：平台原始
运行日志可能包含远端请求和响应，不应把真实凭据写入可长期留存的调试文件，也不能把接口注册器当作凭据保险库。

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

调试请求使用 `apiCode`、`method`、`requestHeaders`、`requestBody` 和 `contentType`。平台支持 GET/POST，以及 JSON、FORM、MULTIPART、XML；上传文件内容必须是 `byte[]`，并可通过 `fileName` 指定文件名。`requestHeaders` 和 `requestBody` 都是 JSON 字符串字段，而不是嵌套对象。

## CLI 调试：JSON、FORM 与文件

建议把请求保存成文件并使用 `@文件名`，避免命令行转义和敏感值进入 shell 历史。JSON 示例：

```json
{
  "apiCode": "ERPOrderSync01",
  "method": "POST",
  "contentType": "JSON",
  "requestHeaders": "{\"Content-Type\":\"application/json\"}",
  "requestBody": "{\"orderNo\":\"SO-10001\"}"
}
```

```bash
cloudcc debug apiRegistrar <projectPath> @debug-json.json
```

POST JSON 的 `requestBody` 是字符串，可以放对象或根数组。CLI 在发网前验证字符串中的 JSON 语法，
但不会把它解析成 Map 后重新编码；setup-svc 会按 UTF-8 原文发送，因此数组、字段顺序、空白和数值
字面量不会被 CLI 改写。根数组示例：

```json
{
  "apiCode": "ERPBatchService01",
  "method": "POST",
  "contentType": "JSON",
  "requestHeaders": "{\"Content-Type\":\"application/json;charset=UTF-8\"}",
  "requestBody": "[\"xxx\",{\"orderNo\":\"SO-10001\"}]"
}
```

注意外层调试文件仍是 JSON，所以 `requestBody` 内部的双引号必须转义。不要把数组直接写成外层
`requestBody: ["xxx"]`；该字段必须是字符串。JSON 根数组原文只支持 POST；GET JSON 参数仍使用对象。

FORM 和 MULTIPART 仍按 JSON 字符串编码请求字段。MULTIPART 的文件内容使用 Base64 或 data URI，文件数据键需配套
`<数据键>Name` 文件名键。不要把大文件直接长期保存在项目配置或版本库中。

## CLI 调试：SOAP 1.1

XML 请求只支持 `POST`。把完整 SOAP Envelope 作为 `requestBody` 字符串，不要再包一层 `body` 对象，也不要
对 XML 进行 Base64 编码：

```json
{
  "apiCode": "ERPSoapService01",
  "method": "POST",
  "contentType": "XML",
  "requestHeaders": "{\"Content-Type\":\"text/xml;charset=UTF-8\",\"SOAPAction\":\"queryOrder\"}",
  "requestBody": "<soapenv:Envelope xmlns:soapenv=\"http://schemas.xmlsoap.org/soap/envelope/\" xmlns:ord=\"urn:example:order\"><soapenv:Header/><soapenv:Body><ord:queryOrder><ord:orderNo>SO-10001</ord:orderNo></ord:queryOrder></soapenv:Body></soapenv:Envelope>"
}
```

```bash
cloudcc debug apiRegistrar <projectPath> @debug-soap11.json
```

如果省略 Content-Type，运行时会为 XML 使用 `text/xml;charset=UTF-8`。是否需要 `SOAPAction` 由目标 SOAP 服务决定。

## CLI 调试：SOAP 1.2

SOAP 1.2 使用 `application/soap+xml`，通常不使用独立 `SOAPAction` Header；需要 action 时将其作为
Content-Type 参数：

```json
{
  "apiCode": "ERPSoapService01",
  "method": "POST",
  "contentType": "XML",
  "requestHeaders": "{\"Content-Type\":\"application/soap+xml;charset=UTF-8;action=urn:example:queryOrder\"}",
  "requestBody": "<env:Envelope xmlns:env=\"http://www.w3.org/2003/05/soap-envelope\" xmlns:ord=\"urn:example:order\"><env:Body><ord:queryOrder><ord:orderNo>SO-10001</ord:orderNo></ord:queryOrder></env:Body></env:Envelope>"
}
```

CLI 会在发起网络请求前拒绝 XML+GET、空 XML、缺失 `apiCode`、缺失 `method` 或缺失/未知 `contentType`。
CLI 不解析 WSDL、XSD 或 SOAP Fault；HTTP 2xx 只表示远端 HTTP 调用成功，业务代码仍需判断 SOAP Body/Fault。

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

普通 `Map` 会由运行时序列化成 JSON 对象。需要发送根数组或必须保持原始 JSON 文本时，传入只含
`"__rawBody"` 的 Map。当前离线 `cloudcc.jar` 不声明该常量，因此直接使用约定字符串键，不需要替换 JAR：

```java
String rawJson = "[\"xxx\",{\"orderNo\":\"SO-10001\"}]";
Map<String, Object> body = new HashMap<String, Object>();
body.put("__rawBody", rawJson);

Map<String, String> headers = new HashMap<String, String>();
headers.put("Content-Type", "application/json;charset=UTF-8");

CCRemoteResult result = new CCRemoteClient(userInfo, "ERPBatchService01").execute(
    CCRemoteClient.HttpMethod.POST,
    body,
    headers,
    CCRemoteClient.ContentType.JSON
);
```

原文模式只支持 POST，`body` 必须且只能包含 `"__rawBody"`，值必须是非空且合法的 JSON 字符串。
自定义类、触发器和定时类运行时都遵守同一约定；推荐由自定义类封装调用，触发器和定时类只负责
准备业务数据并调用该类。已有触发器或定时类若直接调用 `CCRemoteClient`，也必须使用相同的单键 Map。

SOAP/XML 高代码示例：

```java
String envelope = "<soapenv:Envelope xmlns:soapenv=\"http://schemas.xmlsoap.org/soap/envelope/\">"
        + "<soapenv:Body><queryOrder><orderNo>SO-10001</orderNo></queryOrder></soapenv:Body>"
        + "</soapenv:Envelope>";

Map<String, Object> body = new HashMap<String, Object>();
body.put(CCRemoteClient.XML_BODY_PARAM, envelope);

Map<String, String> headers = new HashMap<String, String>();
headers.put("SOAPAction", "queryOrder");
// SOAP 1.2 可改为：
// headers.put("Content-Type", "application/soap+xml;charset=UTF-8;action=urn:example:queryOrder");

CCRemoteResult result = new CCRemoteClient(userInfo, "ERPSoapService01").execute(
    CCRemoteClient.HttpMethod.POST,
    body,
    headers,
    CCRemoteClient.ContentType.XML
);

if (!result.isResult()) {
    throw new RuntimeException("SOAP HTTP call failed: " + result.getCode() + " " + result.getMessage());
}
String responseXml = result.getData() == null ? "" : String.valueOf(result.getData());
```

XML 模式下 `body` 必须且只能包含 `CCRemoteClient.XML_BODY_PARAM`，对应值必须是非空字符串。不要把 XML
Envelope 当作 JSON Map 拆分，也不要手工序列化为带引号的 JSON 字符串。

默认连接超时 3 秒、读取超时 30 秒。可用四参数构造器设置更短超时，但不能超过系统配置。具体 getter 名称应以目标租户 SDK 编译结果为准；若 SDK 只暴露字段或不同 getter，不得凭空编造方法，需按 validate 诊断调整。

## 设计规则

- 源码只保存 `apiCode`；URL 变更只改注册器，不改业务类。
- 生成源码前先确认注册项已调试成功且为 `ACTIVE`。`DRAFT` 或 `INACTIVE` 必须作为配置阻断，而不是靠运行时反复重试。
- 检查 `result`、`code`、`message`、`httpStatus`，业务需要时记录 `traceId`；不能仅因 execute 返回对象就判定业务成功。
- 重试只用于明确可重试且具备幂等键的请求；不要在触发器保存事务中无界重试。
- 触发器只做入口，远程调用封装在自定义类。非强一致集成优先在 commit 后、定时类或外部异步链路执行，避免外部超时阻塞记录保存。
- 定时同步必须分页、记录业务游标/幂等键，并区分 HTTP 成功、远端业务失败和本地落库失败。
- 业务代码可以按外部系统要求传递必要的鉴权 header/body/query 参数，但不得在自定义类、触发器、定时类的业务日志、异常消息或调试输出中主动打印完整 Token、Cookie、Secret、Password、API Key 等敏感值；需要排查时优先记录 `traceId`、业务单号、HTTP 状态和脱敏后的错误摘要。接口注册器只管理 URL 和运行态调用，不等同于凭据保险库。

## 状态、结果与排障

- 创建或修改 URL/描述后状态是 `DRAFT`，必须重新调试。
- 当前运行时以 HTTP `2xx` 作为调试成功并切换 `ACTIVE` 的条件；`3xx/4xx/5xx`、超时和网络错误会失败。
- SOAP Fault 可能通过 HTTP 200 返回。接口注册器会将 HTTP 调试视为成功，调用代码仍必须检查返回 XML 中的 Fault。
- `XML requests only support POST`：把 `method` 改为 `POST`。
- `XML requestBody must be a nonblank raw XML string`：检查 JSON 中的 `requestBody` 是否为字符串，并正确转义双引号。
- `JSON requestBody must be a string`：把根数组作为字符串放入外层调试 JSON，不要直接传数组字段。
- `JSON requestBody must contain valid JSON`：修复缺失引号、括号不配对或尾随逗号；根数组使用标准 JSON，例如 `["xxx"]`。
- `JSON array or scalar requestBody only supports POST`：根数组原文改用 POST；GET 参数继续使用 JSON 对象。
- HTTP 415：检查 SOAP 版本与 Content-Type，SOAP 1.1 通常是 `text/xml`，SOAP 1.2 是 `application/soap+xml`。
- HTTP 500 且返回 SOAP Fault：HTTP 链路已到达目标服务，按 Fault Code/Reason 排查命名空间、action 和业务参数。
- 超过 payload 限制：按 UTF-8 字节数缩小 Envelope；不要依赖字符数估算中文报文大小。
- 调试成功但正式调用提示停用：确认使用的是同一租户和相同 `apiCode`，并检查注册项是否在调试后又被修改回 `DRAFT`。
