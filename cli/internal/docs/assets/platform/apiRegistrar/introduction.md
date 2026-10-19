# CloudCC 接口注册器

## 定位

接口注册器把一个服务端可访问的远程 HTTP 地址注册为平台运行时能力，供自定义类、触发器调用的类和定时类通过 `CCRemoteClient` 使用。管理入口为 `/settings/apiRegistrar/`。

它不是 OpenAPI：OpenAPI 用于外部系统调用 CloudCC；接口注册器用于 CloudCC 高代码调用外部系统。

## 两个标识

- `apiName`：软件包中的组件唯一标识，由平台创建时生成，全表唯一；用于配置识别和迁移，不是运行时代码参数。
- `apiCode`：业务调用唯一标识，全表唯一；创建时允许用户指定，用户值只能包含大小写英文字母和数字。未提供该字段时，由 MetadataService 生成 `API-yyyyMMdd-<128-bit Base62>` 格式的值；显式空值或纯空格不视为未提供。高代码通过 `new CCRemoteClient(userInfo, apiCode)` 调用。

不得把注册 URL、记录 ID 或 `apiName` 写进 `CCRemoteClient` 构造器代替 `apiCode`。

## 生命周期

1. 创建：提交名称、URL、可选描述和可选 `apiCode`；平台生成 ID 和 `apiName`，缺省时生成 `apiCode`，初始状态为 `DRAFT`。
2. 调试：服务端发起 HTTP 请求；当前 setup-svc 在 HTTP 状态码属于 `2xx` 时标记 `success` 并进入 `ACTIVE`，其他状态或网络失败标记为 `failed`/`INACTIVE`。
3. 修改：只修改 URL 或描述，状态回到 `DRAFT`，必须重新调试。
4. 停用：运行态可为 `INACTIVE`，高代码正常调用不得把停用接口视为可用。
5. 删除：逻辑删除配置并释放原 `apiCode`，因此该值可以用于新的注册项；既有调用日志保留。

调试次数、最后调试状态、耗时和时间由运行时维护；其中 `last_debug_time` 按表定义自动更新时间。元数据计划不得伪造这些字段或直接将状态改为 `ACTIVE`。

## 能力边界

- MetadataService `api-registrars`：创建、更新、逻辑删除、列表、详情和变更账本。
- setup-svc 运行态：调试、日志列表和日志详情；这些操作不是元数据 apply，也没有 MetadataService rollback 语义。
- 横纵版仅开放 MetadataService CRUD，不开放 `debug/logs/logDetail`；已注册接口仍可由 main-app 高代码运行时调用。
- 当前注册 URL 只要求 CloudCC 服务端网络可访问，不附加域名白名单或客户端可访问要求。
- 当前日志查看没有额外权限控制；方案不能虚构尚不存在的权限模型。

## 支持的请求格式

接口注册器支持 `GET`、`POST`，以及 `JSON`、`FORM`、`MULTIPART` 和 `XML`。XML 用于 SOAP 或其他
原始 XML HTTP 服务，具有以下固定边界：

- POST JSON 可以使用普通参数 Map，也可以用只含 `"__rawBody"` 的 Map 发送完整 JSON 原文；原文模式支持
  `[` 开头的根数组，例如 `["xxx"]`，并避免对象被重排或二次序列化。
- CLI 调试不直接传 `__rawBody`，而是把完整 JSON 放在字符串类型的 `requestBody` 中；CLI 会先校验 JSON，
  setup-svc 再转换为运行时原文参数。

- XML 只支持 `POST`，请求体按 UTF-8 原文发送，不解析 WSDL、XML Schema 或 SOAP Fault。
- SOAP 1.1 默认使用 `text/xml;charset=UTF-8`，目标服务要求时由调用方提供 `SOAPAction`。
- SOAP 1.2 应显式传入 `application/soap+xml;charset=UTF-8`，需要 action 时放在 Content-Type 参数中。
- XML 请求同样受平台连接超时、读取超时、请求体大小、配额和限流规则约束。
- 接口注册器只保存 URL。SOAP 用户名、密码、Token、证书等凭据必须由调用代码按请求传入，不能写入注册器描述或源码日志。

CLI 调试和高代码调用的完整示例见 `cloudcc doc platform/apiRegistrar devguide`。
