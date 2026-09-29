# High-code 能力发现与执行指南

先使用 `cloudcc domain highcode` 查看子资源，再使用 `cloudcc domain <resource>` 获取准确命令：

```bash
cloudcc domain classes
cloudcc domain trigger
cloudcc domain timer
```

能力组的动作是子资源动作并集。不同资源的创建、校验、发布、查询和删除命令并不完全相同，
不得根据聚合 `actions` 猜测命令。以叶子资源返回的 `commands` 和对应 devguide 为准。

`classes` 发布执行本地编译、目标 setup-svc validate 和 save；触发器与定时类执行目标
validate 后 save；页面组件和自定义页面使用 devconsole 边界；sidecar 运行在平台外部。
