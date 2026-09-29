# Metadata 能力发现与执行指南

## 发现

```bash
cloudcc domains --category metadata
cloudcc domain metadata
cloudcc domain fields
cloudcc domain recordType
```

能力组的 `actions` 是子资源动作并集，`actionSemantics` 固定为
`union-of-child-resources`。这些动作不能直接拼成 `cloudcc <action> metadata`；应读取具体资源
返回的 `commands`。

## Provider 路由

- MSAPI：`metadata-service` / `msapi`。
- UIAPI：`setup-svc` / `uiapi` provider adapter。
- Universal：运行具体资源命令时根据项目 provider 配置选择；纯离线发现不解析项目。

## 实时能力

Domain Catalog 描述 CLI 发行包声明的能力。目标租户实际可用能力还受后端部署版本、权限、
feature flag 和租户元数据影响。需要实时验证时使用具体资源读命令、provider doctor 或
MetadataService capabilities 扫描，不能把 `cloudcc domains` 当作租户探测结果。

