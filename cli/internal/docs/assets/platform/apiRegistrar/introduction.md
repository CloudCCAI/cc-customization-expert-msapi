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
2. 调试：服务端发起 HTTP 请求；当前 setup-svc 仅在 HTTP 状态码等于 `200` 时标记 `success` 并进入 `ACTIVE`，其他状态标记为 `failed`/`INACTIVE`。
3. 修改：只修改 URL 或描述，状态回到 `DRAFT`，必须重新调试。
4. 停用：运行态可为 `INACTIVE`，高代码正常调用不得把停用接口视为可用。
5. 删除：逻辑删除配置并释放原 `apiCode`，因此该值可以用于新的注册项；既有调用日志保留。

调试次数、最后调试状态、耗时和时间由运行时维护；其中 `last_debug_time` 按表定义自动更新时间。元数据计划不得伪造这些字段或直接将状态改为 `ACTIVE`。

## 能力边界

- MetadataService `api-registrars`：创建、更新、逻辑删除、列表、详情和变更账本。
- setup-svc 运行态：调试、日志列表和日志详情；这些操作不是元数据 apply，也没有 MetadataService rollback 语义。
- 当前注册 URL 只要求 CloudCC 服务端网络可访问，不附加域名白名单或客户端可访问要求。
- 当前日志查看没有额外权限控制；方案不能虚构尚不存在的权限模型。
