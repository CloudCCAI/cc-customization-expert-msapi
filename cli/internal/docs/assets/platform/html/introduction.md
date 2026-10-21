# CloudCC HTML 组件介绍

`html` 用于管理平台中的 HTML 组件，以单 HTML 文件作为页面入口，适合承载自定义展示和轻量交互。JS、CSS 和图片可作为静态资源管理，再通过资源 URL 引用。

## 使用流程与边界

CLI 支持本地创建、查询、读取详情、发布和删除。通常先 `cloudcc create html <name> [projectPath]` 创建骨架，再编辑内容并使用 `cloudcc publish html <name> [projectPath]` 发布。

HTML 组件与 Lightning `pagecomponent`、`customPage` 以及横纵版 `visualPage` 是不同资源，不能混用目录或发布命令。横纵版当前不支持该 HTML 资源的远程操作。交互涉及服务端数据或复杂规则时，应调用受支持的平台接口或自定义类，不能在页面中嵌入凭据。

详细参数、资源引用和示例见 `cloudcc doc html devguide`。全部模块可通过 `cloudcc doc list` 查询。
