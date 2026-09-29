# Metadata 能力组

`metadata` 表示 CloudCC 平台配置和模型控制面。它是能力组，不是一个可以直接执行
`cloudcc get metadata` 的具体资源。

使用以下命令发现具体资源：

```bash
cloudcc domain metadata
cloudcc domain objects
cloudcc domain fields
cloudcc domain pagelayout
```

`cloudcc domain metadata` 返回全部子资源摘要；`cloudcc domain <resource>` 返回该资源的别名、
provider 路由、动作、可执行命令形式和文档入口。目录是当前 CLI 发行包内置的离线能力声明，
不代表目标租户已经部署对应 MetadataService 版本，也不读取项目配置、凭据或网络。

MSAPI 路由由 MetadataService 承载，支持计划、显式执行和服务端回滚语义。UIAPI 路由通过
setup-svc adapter 执行，安全语义可能不同。Universal 包在未读取项目上下文时只声明
`provider-selected`，不会假定当前选择了哪条路由。

