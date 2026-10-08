# High-code 能力组

`highcode` 是 CloudCC 可部署代码和页面扩展资源的能力组，包括 `classes`、`triggers`、
`timer`、`script`、`html`、`staticResource`、`pagecomponent`、`customPage`、`visualPage`、
`customComponent` 和 `sidecar`。

```bash
cloudcc domain highcode
cloudcc domain classes
cloudcc domain triggers
cloudcc domain pagecomponent
cloudcc domain visualPage
cloudcc domain customComponent
```

高代码写入沿用 setup-svc、devconsole 或外部运行时边界，不进入 MetadataService 元数据
plan/apply。具体资源详情返回可执行命令和对应 introduction/devguide 文档入口。

Lightning 的页面扩展是 `customPage` 与 `pagecomponent`。横纵版没有 devconsole，页面扩展使用
main-app `visualPage` 与横纵版独有 `customComponent`，两组资源不可互换。横纵版的类、触发器、
定时类、VisualPage、静态资源和 customComponent 使用 Struts Action；script/html 暂不支持横纵版。
